package scheduler

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	schedulerTickInterval  = 3 * time.Second
	schedulerBatchLimit    = 200
	staffSelectingTimeout  = 5 * time.Minute
	// 房间会话超时时间, 房间会话60分钟内没选技师、没开始服务则超时,自动取消房间锁定
	sessionAbandonTimeout = 60 * time.Minute
)

func getStartPendingTimeoutForSession(s *models.ServiceSession) time.Duration {
	if s == nil {
		return config.StartPendingTimeout()
	}
	if s.StartPendingTimeoutSeconds > 0 {
		return time.Duration(s.StartPendingTimeoutSeconds) * time.Second
	}
	return config.StartPendingTimeout()
}

func finishAndReleaseSession(tx *gorm.DB, s *models.ServiceSession, finishedAt time.Time) error {
	updates := map[string]interface{}{
		"status":                  "finished",
		"finished_at":             finishedAt,
		"room_id":                 nil,
		"room_locked_at":          nil,
		"room_select_deadline_at": nil,
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status = ?", s.ID, "start_pending").
		Updates(updates).Error; err != nil {
		return err
	}

	if s.TechnicianID == nil || *s.TechnicianID == 0 {
		return nil
	}
	return tx.Model(&models.TechnicianAttendance{}).
		Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
		Updates(map[string]interface{}{"status": "idle"}).Error
}

func cancelAndReleaseSession(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	updates := map[string]interface{}{
		"status":                  "canceled",
		"room_id":                 nil,
		"room_locked_at":          nil,
		"room_select_deadline_at": nil,
	}
	
	// 释放技师状态
	if s.TechnicianID != nil && *s.TechnicianID > 0 {
		if err := tx.Model(&models.TechnicianAttendance{}).
			Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
			Updates(map[string]interface{}{"status": "idle"}).Error; err != nil {
			return err
		}
	}
	
	return tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ('room_selecting','room_locked','staff_selecting')", s.ID).
		Updates(updates).Error
}

func autoAssignTechnicianIfPossible(tx *gorm.DB, s *models.ServiceSession, now time.Time) (bool, error) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	activeSessionStatuses := []string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"}

	type candLite struct {
		ID           uint `gorm:"column:id"`
		TechnicianID uint `gorm:"column:technician_id"`
	}

	var cand candLite
	err := tx.
		Model(&models.TechnicianAttendance{}).
		Select("technician_attendances.id, technician_attendances.technician_id").
		Joins("JOIN technicians t ON t.id = technician_attendances.technician_id").
		Joins("JOIN service_roles sr ON sr.id = t.service_role_id").
		Where("technician_attendances.merchant_id = ? AND technician_attendances.checked_in_at >= ? AND technician_attendances.checked_out_at IS NULL AND technician_attendances.status IN ('idle')", s.MerchantID, start).
		Where("NOT EXISTS (SELECT 1 FROM service_sessions ss WHERE ss.merchant_id = ? AND ss.technician_id = technician_attendances.technician_id AND ss.status IN ?)", s.MerchantID, activeSessionStatuses).
		Where("t.is_active = ?", true).
		Where("sr.role_type = ? AND sr.`key` NOT IN ('store_manager','front_desk')", "professional").
		Order("technician_attendances.updated_at asc").
		Limit(1).
		First(&cand).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	var att models.TechnicianAttendance
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL AND status = ?", cand.ID, s.MerchantID, cand.TechnicianID, start, "idle").
		First(&att).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	if err := tx.Model(&models.TechnicianAttendance{}).Where("id = ?", att.ID).Update("status", "busy").Error; err != nil {
		return false, err
	}

	updates := map[string]interface{}{
		"technician_id":               cand.TechnicianID,
		"status":                      "start_pending",
		"staff_select_cooldown_until": nil,
		"staff_select_entered_at":     nil,
		"start_pending_timeout_seconds": int(config.StartPendingTimeout().Seconds()),
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND technician_id IS NULL AND status IN ('room_locked','staff_selecting')", s.ID).
		Updates(updates).Error; err != nil {
		return false, err
	}
	return true, nil
}

func StartServiceSessionScheduler() {
	go func() {
		ticker := time.NewTicker(schedulerTickInterval)
		defer ticker.Stop()

		for range ticker.C {
			if config.DB == nil {
				continue
			}
			if err := runOnce(config.DB); err != nil {
				log.Printf("service session scheduler error: %v", err)
			}
		}
	}()
}

func runOnce(db *gorm.DB) error {
	now := time.Now()

	var sessions []models.ServiceSession
	err := db.
		Where("status IN ('room_selecting','room_locked','staff_selecting','start_pending','delay_pending','serving','auto_finishing','finished')").
		Order("id asc").
		Limit(schedulerBatchLimit).
		Find(&sessions).Error
	if err != nil {
		return err
	}

	for i := range sessions {
		s := sessions[i]
		if err := advanceOne(db, &s, now); err != nil {
			log.Printf("advance session %d error: %v", s.ID, err)
		}
	}
	return nil
}

func advanceOne(db *gorm.DB, session *models.ServiceSession, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&s, session.ID).Error; err != nil {
			return err
		}
		baseAt := s.UpdatedAt
		if baseAt == nil {
			baseAt = s.CreatedAt
		}
		if s.TechnicianID == nil && s.StartedAt == nil && baseAt != nil {
			if now.Sub(*baseAt) >= sessionAbandonTimeout {
				return cancelAndReleaseSession(tx, &s, now)
			}
		}

		switch s.Status {
		case "room_locked", "staff_selecting":
			// 选客服超时：
			// - 5分钟内：等待用户选择
			// - 5分钟后：自动分配空闲最久的客服
			// - 若无空闲客服：进入3分钟冷却，冷却后用户可再次选择
			if s.RoomID == nil || s.TechnicianID != nil || s.RoomLockedAt == nil {
				return nil
			}
			if s.StaffSelectCooldownUntil != nil && now.Before(*s.StaffSelectCooldownUntil) {
				return nil
			}
			var merchant models.Merchant
			if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
				return err
			}
			// 若商户已关闭客服模式：降级为非客服流程（不再选客服/不自动分配客服），进入延迟起单
			if !merchant.SupportCustomerServiceMode {
				delaySeconds := merchant.StartDelaySeconds
				if delaySeconds <= 0 {
					delaySeconds = 60
				}
				startAt := now.Add(time.Duration(delaySeconds) * time.Second)
				updates := map[string]interface{}{
					"status":                    "delay_pending",
					"technician_id":             nil,
					"staff_select_cooldown_until": nil,
					"staff_select_entered_at":     nil,
					"start_confirmed_at":        &now,
					"scheduled_start_at":        &startAt,
					"room_select_deadline_at":   nil,
				}
				return tx.Model(&models.ServiceSession{}).
					Where("id = ? AND status IN ('room_locked','staff_selecting') AND start_confirmed_at IS NULL", s.ID).
					Updates(updates).Error
			}
			// 必须在用户进入选择客服页后才允许开始5分钟自动分配计时
			if s.StaffSelectEnteredAt == nil {
				return nil
			}
			if !merchant.SupportRoom {
				return nil
			}
			deadline := s.StaffSelectEnteredAt.Add(staffSelectingTimeout)
			if now.Before(deadline) {
				return nil
			}
			ok, err := autoAssignTechnicianIfPossible(tx, &s, now)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
			dl := now.Add(3 * time.Minute)
			updates := map[string]interface{}{
				"status":                      "staff_selecting",
				"staff_select_cooldown_until": &dl,
			}
			return tx.Model(&models.ServiceSession{}).
				Where("id = ? AND technician_id IS NULL AND status IN ('room_locked','staff_selecting')", s.ID).
				Updates(updates).Error
		case "room_selecting":
			if s.RoomSelectDeadlineAt != nil && now.After(*s.RoomSelectDeadlineAt) {
				baseAt := s.UpdatedAt
				if baseAt == nil {
					baseAt = s.CreatedAt
				}
				if s.TechnicianID == nil && s.StartedAt == nil && baseAt != nil && now.Sub(*baseAt) >= sessionAbandonTimeout {
					return cancelAndReleaseSession(tx, &s, now)
				}
				return autoAssignRoom(tx, &s, now)
			}
			return nil
		case "start_pending":
			// 若商户已关闭客服模式：降级为非客服流程，进入延迟起单（避免卡在待起单/待上钟）
			{
				var merchant models.Merchant
				if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
					return err
				}
				if !merchant.SupportCustomerServiceMode {
					delaySeconds := merchant.StartDelaySeconds
					if delaySeconds <= 0 {
						delaySeconds = 60
					}
					startAt := now.Add(time.Duration(delaySeconds) * time.Second)
					updates := map[string]interface{}{
						"status":              "delay_pending",
						"technician_id":       nil,
						"start_confirmed_at":  &now,
						"scheduled_start_at":  &startAt,
						"staff_select_entered_at": nil,
						"staff_select_cooldown_until": nil,
						"start_pending_timeout_seconds": 0,
					}
					// 释放技师占用（如果有）
					if s.TechnicianID != nil && *s.TechnicianID > 0 {
						_ = tx.Model(&models.TechnicianAttendance{}).
							Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
							Updates(map[string]interface{}{"status": "idle"}).Error
					}
					return tx.Model(&models.ServiceSession{}).
						Where("id = ? AND status = ? AND start_confirmed_at IS NULL", s.ID, "start_pending").
						Updates(updates).Error
				}
			}
			// 待起单超时：不再支持“手动结单”。超时后仅允许用户重新选择工作人员并重新起单。
			if s.StartConfirmedAt != nil {
				return nil
			}
			if s.UpdatedAt == nil {
				return nil
			}
			startDeadline := s.UpdatedAt.Add(getStartPendingTimeoutForSession(&s))
			if now.Before(startDeadline) {
				return nil
			}

			updates := map[string]interface{}{
				"status":                "staff_selecting",
				"technician_id":         nil,
				"staff_select_entered_at": nil,
				"start_pending_timeout_seconds": 0,
				"start_timeout_count":   gorm.Expr("start_timeout_count + ?", 1),
				"start_timeout_last_at": now,
			}
			if err := tx.Model(&models.ServiceSession{}).
				Where("id = ? AND status = ? AND start_confirmed_at IS NULL", s.ID, "start_pending").
				Updates(updates).Error; err != nil {
				return err
			}
			if s.InitialUsageID > 0 && queue.Default != nil {
				date := now.Format("2006-01-02")
				queue.Default.Uncall(s.MerchantID, date, queue.QueueTypeOnsite, s.InitialUsageID)
			}

			// 释放技师状态
			if s.TechnicianID != nil && *s.TechnicianID > 0 {
				if err := tx.Model(&models.TechnicianAttendance{}).
					Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
					Updates(map[string]interface{}{"status": "idle"}).Error; err != nil {
					return err
				}
			}
			return nil
		case "delay_pending":
			if s.StartConfirmedAt == nil {
				return nil
			}
			if s.ScheduledStartAt != nil && !now.Before(*s.ScheduledStartAt) {
				updates := map[string]interface{}{
					"status":     "serving",
					"started_at": now,
				}
				if s.ScheduledFinishAt == nil {
					if s.DurationMinutes > 0 {
						finishAt := now.Add(time.Duration(s.DurationMinutes) * time.Minute)
						updates["scheduled_finish_at"] = finishAt
					}
				}
				return tx.Model(&models.ServiceSession{}).
					Where("id = ? AND status = ? AND start_confirmed_at IS NOT NULL", s.ID, "delay_pending").
					Updates(updates).Error
			}
			return nil
		case "serving":
			if s.StartConfirmedAt == nil {
				return nil
			}
			if s.ScheduledFinishAt != nil && now.After(*s.ScheduledFinishAt) {
				finishAt := s.ScheduledFinishAt.Add(time.Duration(s.AutoFinishDelaySeconds) * time.Second)
				updates := map[string]interface{}{
					"status":      "auto_finishing",
					"finished_at": finishAt,
				}
				return tx.Model(&models.ServiceSession{}).
					Where("id = ? AND status = ? AND start_confirmed_at IS NOT NULL", s.ID, "serving").
					Updates(updates).Error
			}
			return nil
		case "auto_finishing":
			if s.StartConfirmedAt == nil {
				return nil
			}
			if s.FinishedAt != nil && !now.Before(*s.FinishedAt) {
				return finalizeSession(tx, &s, now)
			}
			return nil
		case "finished":
			return releaseTechnicianIfNeeded(tx, &s, now)
		default:
			return nil
		}
	})
}

func autoAssignRoom(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	var merchant models.Merchant
	if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
		return err
	}
	if !merchant.SupportRoom {
		updates := map[string]interface{}{
			"status": "staff_selecting",
		}
		return tx.Model(&models.ServiceSession{}).Where("id = ? AND status = ?", s.ID, "room_selecting").Updates(updates).Error
	}

	var rooms []models.Room
	if err := tx.Where("merchant_id = ? AND is_active = ?", s.MerchantID, true).Order("id asc").Find(&rooms).Error; err != nil {
		return err
	}
	for i := range rooms {
		r := rooms[i]
		var cnt int64
		if err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','start_pending','delay_pending','serving','auto_finishing')", s.MerchantID, r.ID).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			lockedAt := now
			updates := map[string]interface{}{
				"room_id":        r.ID,
				"room_locked_at": lockedAt,
				"status":         "room_locked",
			}
			return tx.Model(&models.ServiceSession{}).Where("id = ? AND status = ?", s.ID, "room_selecting").Updates(updates).Error
		}
	}
	return nil
}

func finalizeSession(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	if s.StartConfirmedAt == nil {
		return nil
	}
	updates := map[string]interface{}{
		"status": "finished",
	}
	if s.FinishedAt == nil {
		updates["finished_at"] = now
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status = ? AND start_confirmed_at IS NOT NULL", s.ID, "auto_finishing").
		Updates(updates).Error; err != nil {
		return err
	}

	if s.InitialUsageID == 0 {
		return nil
	}

	finishedAt := now
	if s.FinishedAt != nil {
		finishedAt = *s.FinishedAt
	}

	uUpdates := map[string]interface{}{
		"status":      "success",
		"finished_at": finishedAt,
	}
	if s.TechnicianID != nil && *s.TechnicianID > 0 {
		uUpdates["technician_id"] = *s.TechnicianID
	}
	if s.RoomID != nil && *s.RoomID > 0 {
		uUpdates["room_id"] = *s.RoomID
	}

	if err := tx.Model(&models.Usage{}).
		Where("id = ? AND status = ?", s.InitialUsageID, "in_progress").
		Updates(uUpdates).Error; err != nil {
		return err
	}

	// 自动叫号：仅现场叫号队列生效；预约队列走预约流程
	var merchant models.Merchant
	if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
		return nil
	}
	if merchant.SupportQueue && merchant.QueueMode == "auto" {
		date := now.Format("2006-01-02")
		queue.Default.MarkDone(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID, now)
		if !merchant.SupportMultiCustomerService {
			queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
		}
	}
	return nil
}

func releaseTechnicianIfNeeded(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	if s.TechnicianID == nil || *s.TechnicianID == 0 {
		return nil
	}
	if s.FinishedAt == nil {
		return nil
	}
	releaseAt := s.FinishedAt.Add(time.Duration(s.AutoIdleAfterSeconds) * time.Second)
	if now.Before(releaseAt) {
		return nil
	}
	var att models.TechnicianAttendance
	res := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
		Limit(1).
		Find(&att)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}

	updates := map[string]interface{}{
		"next_status": nil,
	}
	newAttStatus := "idle"
	if att.NextStatus != nil && *att.NextStatus == "paused" {
		updates["status"] = "paused"
		newAttStatus = "paused"
	} else {
		updates["status"] = "idle"
	}
	if err := tx.Model(&models.TechnicianAttendance{}).Where("id = ?", att.ID).Updates(updates).Error; err != nil {
		return err
	}
	if newAttStatus == "idle" {
		if err := autoCallNextForTechnician(tx, s.MerchantID, *s.TechnicianID, now); err != nil {
			log.Printf("auto call next failed: merchant=%d tech=%d err=%v", s.MerchantID, *s.TechnicianID, err)
		}
	}
	return nil
}

func autoCallNextForTechnician(tx *gorm.DB, merchantID uint, technicianID uint, now time.Time) error {
	if merchantID == 0 || technicianID == 0 {
		return nil
	}
	if queue.Default == nil {
		return nil
	}

	var merchant models.Merchant
	if err := tx.First(&merchant, merchantID).Error; err != nil {
		return nil
	}
	if !merchant.SupportQueue || merchant.QueueMode != "auto" {
		return nil
	}
	if !merchant.SupportCustomerServiceMode {
		return nil
	}
	if !merchant.SupportMultiCustomerService {
		return nil
	}

	date := now.Format("2006-01-02")
	nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	if nextUsageID == 0 {
		return nil
	}

	var nextSession models.ServiceSession
	q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND initial_usage_id = ? AND start_confirmed_at IS NULL AND technician_id IS NULL", merchantID, nextUsageID)
	if merchant.SupportRoom {
		q = q.Where("status IN ('room_locked','staff_selecting') AND room_id IS NOT NULL")
	} else {
		q = q.Where("status IN ('staff_selecting','room_locked')")
	}
	if err := q.Order("id desc").First(&nextSession).Error; err != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return nil
	}

	updates := map[string]interface{}{
		"technician_id": technicianID,
		"status":        "start_pending",
		"staff_select_entered_at": nil,
		"staff_select_cooldown_until": nil,
		"start_pending_timeout_seconds": int(config.StartPendingTimeout().Seconds()),
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND merchant_id = ? AND technician_id IS NULL AND start_confirmed_at IS NULL", nextSession.ID, merchantID).
		Updates(updates).Error; err != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return nil
	}
	return nil
}
