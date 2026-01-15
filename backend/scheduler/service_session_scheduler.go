package scheduler

import (
	"kabao/config"
	"kabao/models"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	schedulerTickInterval = 3 * time.Second
	schedulerBatchLimit   = 200
	staffSelectingTimeout = 5 * time.Minute
	// 房间会话超时时间, 房间会话30分钟内没选技师、没开始服务则超时,自动取消房间锁定
	sessionAbandonTimeout = 30 * time.Minute
)

func cancelAndReleaseSession(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	updates := map[string]interface{}{
		"status":                  "canceled",
		"room_id":                 nil,
		"room_locked_at":          nil,
		"room_select_deadline_at": nil,
	}
	return tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ('room_selecting','room_locked','staff_selecting')", s.ID).
		Updates(updates).Error
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
		Where("status IN ('room_selecting','room_locked','staff_selecting','delay_pending','serving','auto_finishing','finished')").
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
		if s.TechnicianID == nil && s.StartedAt == nil && s.CreatedAt != nil {
			if now.Sub(*s.CreatedAt) >= sessionAbandonTimeout {
				return cancelAndReleaseSession(tx, &s, now)
			}
		}

		switch s.Status {
		case "room_locked", "staff_selecting":
			// 选人超时释放房间（仅限：有房间、已锁定房间、未选择工作人员）
			if s.RoomID == nil || s.TechnicianID != nil || s.RoomLockedAt == nil {
				return nil
			}
			var merchant models.Merchant
			if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
				return err
			}
			if !merchant.SupportRoom {
				return nil
			}
			deadline := s.RoomLockedAt.Add(staffSelectingTimeout)
			if now.Before(deadline) {
				return nil
			}
			dl := now.Add(90 * time.Second)
			updates := map[string]interface{}{
				"room_id":                 nil,
				"room_locked_at":          nil,
				"status":                  "room_selecting",
				"room_select_deadline_at": dl,
			}
			return tx.Model(&models.ServiceSession{}).Where("id = ? AND status IN ('room_locked','staff_selecting')", s.ID).Updates(updates).Error
		case "room_selecting":
			if s.RoomSelectDeadlineAt != nil && now.After(*s.RoomSelectDeadlineAt) {
				if s.TechnicianID == nil && s.StartedAt == nil && s.CreatedAt != nil && now.Sub(*s.CreatedAt) >= sessionAbandonTimeout {
					return cancelAndReleaseSession(tx, &s, now)
				}
				return autoAssignRoom(tx, &s, now)
			}
			return nil
		case "delay_pending":
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
				return tx.Model(&models.ServiceSession{}).Where("id = ? AND status = ?", s.ID, "delay_pending").Updates(updates).Error
			}
			return nil
		case "serving":
			if s.ScheduledFinishAt != nil && now.After(*s.ScheduledFinishAt) {
				finishAt := s.ScheduledFinishAt.Add(time.Duration(s.AutoFinishDelaySeconds) * time.Second)
				updates := map[string]interface{}{
					"status":      "auto_finishing",
					"finished_at": finishAt,
				}
				return tx.Model(&models.ServiceSession{}).Where("id = ? AND status = ?", s.ID, "serving").Updates(updates).Error
			}
			return nil
		case "auto_finishing":
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
			Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','precheck_pending','delay_pending','serving','auto_finishing')", s.MerchantID, r.ID).
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
	updates := map[string]interface{}{
		"status": "finished",
	}
	if s.FinishedAt == nil {
		updates["finished_at"] = now
	}
	if err := tx.Model(&models.ServiceSession{}).Where("id = ? AND status = ?", s.ID, "auto_finishing").Updates(updates).Error; err != nil {
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

	return tx.Model(&models.Usage{}).
		Where("id = ? AND status = ?", s.InitialUsageID, "in_progress").
		Updates(uUpdates).Error
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
	return tx.Model(&models.TechnicianAttendance{}).
		Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
		Updates(map[string]interface{}{"status": "idle"}).Error
}
