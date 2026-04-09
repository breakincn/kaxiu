package scheduler

import (
	"errors"
	"fmt"
	"kabao/appointmentdelay"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"kabao/sessionflow"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	schedulerTickInterval        = 3 * time.Second
	schedulerBatchLimit          = 200
	defaultStaffSelectingTimeout = 5 * time.Minute
	// 房间会话超时时间, 房间会话60分钟内没选技师、没开始服务则超时,自动取消房间锁定
	sessionAbandonTimeout = 60 * time.Minute
)

func resolveStaffSelectingTimeout(tx *gorm.DB, s *models.ServiceSession) time.Duration {
	if s == nil {
		return defaultStaffSelectingTimeout
	}
	project, err := config.ResolveMerchantProject(tx, s.MerchantID, s.ProjectID)
	if err != nil || project == nil {
		return defaultStaffSelectingTimeout
	}
	if project.AutoAssignTechnicianDelayMinutes <= 0 {
		return 0
	}
	return time.Duration(project.AutoAssignTechnicianDelayMinutes) * time.Minute
}

func resolveQueueSessionModeForTimeoutWaiting(s *models.ServiceSession, merchant *models.Merchant) string {
	if s == nil {
		return ""
	}
	if models.IsQueueMode(s.SessionMode) {
		return s.SessionMode
	}

	status := strings.TrimSpace(s.Status)
	switch {
	case strings.HasPrefix(status, models.SessionModeQueueAutoSingle+"_"):
		return models.SessionModeQueueAutoSingle
	case strings.HasPrefix(status, models.SessionModeQueueAutoMulti+"_"):
		return models.SessionModeQueueAutoMulti
	case strings.HasPrefix(status, models.SessionModeQueueManualSingle+"_"):
		return models.SessionModeQueueManualSingle
	case strings.HasPrefix(status, models.SessionModeQueueManualMulti+"_"):
		return models.SessionModeQueueManualMulti
	}

	if merchant != nil && merchant.SupportQueue {
		mode := models.ResolveSessionMode(merchant)
		if models.IsQueueMode(mode) {
			return mode
		}
	}
	return ""
}

func queueDebugEnabledFor(merchantID uint, sessionID uint, usageID uint) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("KABAO_QUEUE_DEBUG")))
	if v == "" || v == "0" || v == "false" || v == "off" {
		return false
	}
	if v := strings.TrimSpace(os.Getenv("KABAO_QUEUE_DEBUG_MERCHANT")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			if merchantID != uint(id) {
				return false
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("KABAO_QUEUE_DEBUG_SESSION")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			if sessionID != uint(id) {
				return false
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("KABAO_QUEUE_DEBUG_USAGE")); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			if usageID != uint(id) {
				return false
			}
		}
	}
	return true
}

func shouldAutoCallNextInAutoSingleQueue(merchant *models.Merchant) bool {
	if merchant == nil {
		return false
	}
	return merchant.QueueMode == "auto" && !merchant.SupportMultiCustomerService && !merchant.QueuePaused
}

func isCrossDay(baseAt *time.Time, now time.Time) bool {
	if baseAt == nil {
		return false
	}
	baseDate := time.Date(baseAt.Year(), baseAt.Month(), baseAt.Day(), 0, 0, 0, 0, baseAt.Location())
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return nowDate.After(baseDate)
}

func getStartPendingTimeoutForSession(s *models.ServiceSession) time.Duration {
	if s == nil {
		return config.StartPendingTimeout()
	}
	if s.StartPendingTimeoutSeconds > 0 {
		return time.Duration(s.StartPendingTimeoutSeconds) * time.Second
	}
	return config.StartPendingTimeout()
}

func loadServiceSessionByID(tx *gorm.DB, sessionID uint) (*models.ServiceSession, error) {
	if tx == nil || sessionID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("service_sessions").
		Select("id, merchant_id, user_id, card_id, project_id, initial_usage_id, verify_code, session_mode, source_type, source_id, occupies_next_appointment, next_appointment_id, predicted_appointment_delay_minutes, predicted_ready_at, room_id, technician_id, last_technician_id, start_timeout_count, start_timeout_last_at, staff_select_cooldown_until, staff_select_entered_at, start_pending_timeout_seconds, status, room_select_deadline_at, room_locked_at, start_confirmed_at, start_delay_seconds, scheduled_start_at, started_at, duration_minutes, scheduled_finish_at, finished_at, auto_finish_delay_seconds, auto_idle_after_seconds, created_at, updated_at").
		Where("id = ?", sessionID).
		Limit(1).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	var (
		session                     models.ServiceSession
		projectIDRaw                interface{}
		sourceIDRaw                 interface{}
		nextAppointmentIDRaw        interface{}
		predictedReadyAtRaw         interface{}
		roomIDRaw                   interface{}
		technicianIDRaw             interface{}
		lastTechnicianIDRaw         interface{}
		startTimeoutLastAtRaw       interface{}
		staffSelectCooldownUntilRaw interface{}
		staffSelectEnteredAtRaw     interface{}
		roomSelectDeadlineAtRaw     interface{}
		roomLockedAtRaw             interface{}
		startConfirmedAtRaw         interface{}
		scheduledStartAtRaw         interface{}
		startedAtRaw                interface{}
		scheduledFinishAtRaw        interface{}
		finishedAtRaw               interface{}
		createdAtRaw                interface{}
		updatedAtRaw                interface{}
	)
	if err := rows.Scan(
		&session.ID,
		&session.MerchantID,
		&session.UserID,
		&session.CardID,
		&projectIDRaw,
		&session.InitialUsageID,
		&session.VerifyCode,
		&session.SessionMode,
		&session.SourceType,
		&sourceIDRaw,
		&session.OccupiesNextAppointment,
		&nextAppointmentIDRaw,
		&session.PredictedAppointmentDelayMinutes,
		&predictedReadyAtRaw,
		&roomIDRaw,
		&technicianIDRaw,
		&lastTechnicianIDRaw,
		&session.StartTimeoutCount,
		&startTimeoutLastAtRaw,
		&staffSelectCooldownUntilRaw,
		&staffSelectEnteredAtRaw,
		&session.StartPendingTimeoutSeconds,
		&session.Status,
		&roomSelectDeadlineAtRaw,
		&roomLockedAtRaw,
		&startConfirmedAtRaw,
		&session.StartDelaySeconds,
		&scheduledStartAtRaw,
		&startedAtRaw,
		&session.DurationMinutes,
		&scheduledFinishAtRaw,
		&finishedAtRaw,
		&session.AutoFinishDelaySeconds,
		&session.AutoIdleAfterSeconds,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return nil, err
	}
	if v, ok := schedulerValueToUint(projectIDRaw); ok {
		session.ProjectID = &v
	}
	if v, ok := schedulerValueToUint(sourceIDRaw); ok {
		session.SourceID = &v
	}
	if v, ok := schedulerValueToUint(nextAppointmentIDRaw); ok {
		session.NextAppointmentID = &v
	}
	if v, ok := parseSchedulerDBTimeValue(predictedReadyAtRaw); ok {
		session.PredictedReadyAt = v
	}
	if v, ok := schedulerValueToUint(roomIDRaw); ok {
		session.RoomID = &v
	}
	if v, ok := schedulerValueToUint(technicianIDRaw); ok {
		session.TechnicianID = &v
	}
	if v, ok := schedulerValueToUint(lastTechnicianIDRaw); ok {
		session.LastTechnicianID = &v
	}
	if v, ok := parseSchedulerDBTimeValue(startTimeoutLastAtRaw); ok {
		session.StartTimeoutLastAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(staffSelectCooldownUntilRaw); ok {
		session.StaffSelectCooldownUntil = v
	}
	if v, ok := parseSchedulerDBTimeValue(staffSelectEnteredAtRaw); ok {
		session.StaffSelectEnteredAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(roomSelectDeadlineAtRaw); ok {
		session.RoomSelectDeadlineAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(roomLockedAtRaw); ok {
		session.RoomLockedAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(startConfirmedAtRaw); ok {
		session.StartConfirmedAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(scheduledStartAtRaw); ok {
		session.ScheduledStartAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(startedAtRaw); ok {
		session.StartedAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(scheduledFinishAtRaw); ok {
		session.ScheduledFinishAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(finishedAtRaw); ok {
		session.FinishedAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(createdAtRaw); ok {
		session.CreatedAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(updatedAtRaw); ok {
		session.UpdatedAt = v
	}
	return &session, nil
}

func finalizeOverdueManualServingSessions(db *gorm.DB, now time.Time) error {
	if db == nil {
		return nil
	}
	deadline := now.Add(-4 * time.Hour)
	var sessions []models.ServiceSession
	if err := db.
		Where("status IN ? AND start_confirmed_at IS NOT NULL AND started_at IS NOT NULL AND started_at <= ?", models.ExpandStatusWithKnownPrefixes("serving"), deadline).
		Order("id asc").
		Limit(schedulerBatchLimit).
		Find(&sessions).Error; err != nil {
		return err
	}
	if len(sessions) == 0 {
		return nil
	}
	for i := range sessions {
		s := sessions[i]
		if s.MerchantID == 0 {
			continue
		}
		_ = db.Transaction(func(tx *gorm.DB) error {
			var locked models.ServiceSession
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", s.ID).First(&locked).Error; err != nil {
				return nil
			}
			if models.NormalizeSessionStatus(locked.Status) != "serving" || locked.StartConfirmedAt == nil || locked.StartedAt == nil {
				return nil
			}
			if locked.StartedAt.After(deadline) {
				return nil
			}
			var merchant models.Merchant
			if err := tx.First(&merchant, locked.MerchantID).Error; err != nil {
				return nil
			}
			if !merchant.SupportQueue || merchant.QueueMode != "manual" {
				return nil
			}
			return finalizeSession(tx, &locked, now)
		})
	}
	return nil
}

func moveMultiQueueStartPendingToTimeoutWaiting(tx *gorm.DB, s *models.ServiceSession, merchant *models.Merchant, now time.Time) error {
	if tx == nil || s == nil || merchant == nil {
		return nil
	}
	if s.InitialUsageID == 0 {
		return nil
	}
	if !merchant.SupportQueue || merchant.QueueMode != "auto" || !merchant.SupportMultiCustomerService {
		return nil
	}

	techID := uint(0)
	if s.TechnicianID != nil {
		techID = *s.TechnicianID
	}

	updates := map[string]interface{}{
		"status":                        models.ApplyStatusPrefix(s.Status, "timeout_waiting"),
		"technician_id":                 nil,
		"start_confirmed_at":            nil,
		"scheduled_start_at":            nil,
		"staff_select_entered_at":       nil,
		"staff_select_cooldown_until":   nil,
		"start_pending_timeout_seconds": 0,
		"start_timeout_count":           gorm.Expr("start_timeout_count + ?", 1),
		"start_timeout_last_at":         now,
	}
	if techID > 0 {
		updates["last_technician_id"] = techID
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ? AND start_confirmed_at IS NULL", s.ID, models.ExpandStatusWithKnownPrefixes("start_pending")).
		Updates(updates).Error; err != nil {
		return err
	}

	if queue.Default != nil {
		date := now.Format("2006-01-02")
		queue.Default.MarkDone(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID, now)
	}

	if techID > 0 {
		if err := tx.Model(&models.TechnicianAttendance{}).
			Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, techID, "busy").
			Updates(map[string]interface{}{"status": "idle"}).Error; err != nil {
			return err
		}
		_ = autoCallNextForTechnician(tx, s.MerchantID, techID, now)
	}

	return nil
}

func releaseFinishedSessionTechnicians(db *gorm.DB, now time.Time) error {
	if db == nil {
		return nil
	}

	type finishedLite struct {
		ID                   uint
		MerchantID           uint
		TechnicianID         *uint
		AutoIdleAfterSeconds int
		Status               string
		FinishedAtRaw        string `gorm:"column:finished_at"`
	}

	var sessions []finishedLite
	if err := db.Table("service_sessions").
		Select("id", "merchant_id", "technician_id", "auto_idle_after_seconds", "status", "finished_at").
		Where("status IN ? AND technician_id IS NOT NULL AND finished_at IS NOT NULL", models.ExpandStatusWithKnownPrefixes("finished")).
		Order("id asc").
		Limit(schedulerBatchLimit).
		Find(&sessions).Error; err != nil {
		return err
	}

	for i := range sessions {
		s := sessions[i]
		if strings.TrimSpace(s.FinishedAtRaw) == "" {
			continue
		}
		finishedAt, err := parseFinishedAtFromDBString(s.FinishedAtRaw, now.Location())
		if err != nil {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			var current struct{ Status string }
			if err := tx.Model(&models.ServiceSession{}).Select("status").Where("id = ?", s.ID).Scan(&current).Error; err != nil {
				return err
			}
			if models.NormalizeSessionStatus(current.Status) != "finished" {
				return nil
			}
			copy := models.ServiceSession{
				ID:                   s.ID,
				MerchantID:           s.MerchantID,
				TechnicianID:         s.TechnicianID,
				AutoIdleAfterSeconds: s.AutoIdleAfterSeconds,
				FinishedAt:           &finishedAt,
			}
			return releaseTechnicianIfNeeded(tx, &copy, now)
		}); err != nil {
			log.Printf("release technician for finished session %d error: %v", s.ID, err)
		}
	}

	return nil
}

func parseFinishedAtFromDBString(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty finished_at")
	}
	if loc == nil {
		loc = time.Local
	}
	layouts := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized finished_at format: %s", raw)
}

func finalizeUsagesAfterQueueEnded(db *gorm.DB, now time.Time) error {
	if db == nil {
		return nil
	}
	deadline := now.Add(-15 * time.Minute)
	var merchants []models.Merchant
	if err := db.
		Select("id", "support_queue").
		Where("support_queue = ? AND queue_mode = ? AND queue_paused = ? AND queue_ended_at IS NOT NULL AND queue_ended_at <= ?", true, "manual", true, deadline).
		Order("queue_ended_at asc").
		Limit(50).
		Find(&merchants).Error; err != nil {
		return err
	}
	if len(merchants) == 0 {
		return nil
	}
	for i := range merchants {
		m := merchants[i]
		if m.ID == 0 {
			continue
		}

		_ = db.Transaction(func(tx *gorm.DB) error {
			// 仅收尾进行中的 usage，避免覆盖 failed/canceled 等终态
			var usages []models.Usage
			if err := tx.Model(&models.Usage{}).
				Select("id", "status").
				Where("merchant_id = ? AND status = ?", m.ID, "in_progress").
				Order("id asc").
				Limit(500).
				Find(&usages).Error; err != nil {
				return nil
			}
			if len(usages) == 0 {
				return nil
			}
			for i := range usages {
				u := usages[i]
				handled, err := sessionflow.FinalizeUsageAndSession(tx, u.ID, &m, now, sessionflow.FinishOptions{
					MarkQueueDone: true,
				})
				if err != nil {
					return nil
				}
				if handled {
					continue
				}
				if err := tx.Model(&models.Usage{}).
					Where("id = ? AND merchant_id = ? AND status = ?", u.ID, m.ID, "in_progress").
					Updates(map[string]interface{}{
						"status":      "success",
						"finished_at": now,
					}).Error; err != nil {
					return nil
				}
			}
			return nil
		})
	}
	return nil
}

func autoFinalizeStaleUsages(db *gorm.DB, now time.Time) error {
	if db == nil {
		return nil
	}
	cutoff := now.Add(-12 * time.Hour)
	var usages []models.Usage
	if err := db.
		Preload("Merchant").
		Preload("Project").
		Where("used_at IS NOT NULL AND used_at <= ? AND status <> ?", cutoff, "failed").
		Order("id asc").
		Limit(schedulerBatchLimit).
		Find(&usages).Error; err != nil {
		return err
	}
	for i := range usages {
		u := usages[i]
		if u.UsedAt == nil {
			continue
		}
		if u.Merchant.SupportCustomerServiceMode {
			finishedAt := u.UsedAt.Add(12 * time.Hour)
			if err := db.Transaction(func(tx *gorm.DB) error {
				handled, err := sessionflow.CompleteUnstartedServiceSession(tx, u.ID, u.MerchantID, finishedAt)
				if err != nil {
					return err
				}
				if handled {
					return nil
				}
				return tx.Model(&models.Usage{}).
					Where("id = ?", u.ID).
					Updates(map[string]interface{}{
						"status":        "success",
						"technician_id": nil,
						"finished_at":   finishedAt,
					}).Error
			}); err != nil {
				log.Printf("auto finalize stale usage failed: usage=%d err=%v", u.ID, err)
			}
			continue
		}

		if u.Status == "success" && u.TechnicianID == nil {
			continue
		}
		durationMinutes := 15
		if u.Project != nil && u.Project.Duration > 0 {
			durationMinutes = u.Project.Duration
		}
		finishedAt := u.UsedAt.Add(time.Duration(durationMinutes+5) * time.Minute)
		if err := db.Transaction(func(tx *gorm.DB) error {
			handled, err := sessionflow.FinalizeUsageAndSession(tx, u.ID, &u.Merchant, finishedAt, sessionflow.FinishOptions{
				MarkQueueDone: u.Merchant.SupportQueue,
			})
			if err != nil {
				return err
			}
			if handled {
				return nil
			}
			return tx.Model(&models.Usage{}).
				Where("id = ?", u.ID).
				Updates(map[string]interface{}{
					"status":        "success",
					"technician_id": nil,
					"finished_at":   finishedAt,
				}).Error
		}); err != nil {
			log.Printf("auto finalize stale usage failed: usage=%d err=%v", u.ID, err)
		}
	}
	return nil
}

func finishAndReleaseSession(tx *gorm.DB, s *models.ServiceSession, finishedAt time.Time) error {
	updates := map[string]interface{}{
		"status":                  models.ApplyStatusPrefix(s.Status, "finished"),
		"finished_at":             finishedAt,
		"room_id":                 nil,
		"room_locked_at":          nil,
		"room_select_deadline_at": nil,
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ?", s.ID, models.ExpandStatusWithKnownPrefixes("start_pending")).
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
	// 释放技师状态
	if s.TechnicianID != nil && *s.TechnicianID > 0 {
		if err := tx.Model(&models.TechnicianAttendance{}).
			Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
			Updates(map[string]interface{}{"status": "idle"}).Error; err != nil {
			return err
		}
	}

	var merchant models.Merchant
	if err := tx.Select("id", "support_queue").First(&merchant, s.MerchantID).Error; err != nil {
		return err
	}
	return sessionflow.FailServiceSessionAndRefund(tx, s, &merchant, now, sessionflow.FailOptions{
		AllowedBaseStatuses: []string{"room_selecting", "room_locked", "staff_selecting"},
		TargetStatus:        "canceled",
		SessionUpdates: map[string]interface{}{
			"room_id":                 nil,
			"room_locked_at":          nil,
			"room_select_deadline_at": nil,
		},
	})
}

func autoAssignTechnicianIfPossible(tx *gorm.DB, s *models.ServiceSession, now time.Time) (bool, error) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	activeSessionStatuses := models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"})

	type candLite struct {
		ID            uint `gorm:"column:id"`
		TechnicianID  uint `gorm:"column:technician_id"`
		ServiceRoleID uint `gorm:"column:service_role_id"`
	}

	var cand candLite
	err := tx.
		Model(&models.TechnicianAttendance{}).
		Select("technician_attendances.id, technician_attendances.technician_id, t.service_role_id").
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

	timeoutSeconds := config.ResolveServiceSessionStartPendingTimeoutSecondsByRole(tx, s.MerchantID, cand.ServiceRoleID, s.ProjectID)

	updates := map[string]interface{}{
		"technician_id":                 cand.TechnicianID,
		"last_technician_id":            cand.TechnicianID,
		"status":                        models.ApplyStatusPrefix(s.Status, "start_pending"),
		"staff_select_cooldown_until":   nil,
		"staff_select_entered_at":       nil,
		"start_pending_timeout_seconds": timeoutSeconds,
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND technician_id IS NULL AND status IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting"})).
		Updates(updates).Error; err != nil {
		return false, err
	}
	return true, nil
}

func autoCallNextForMultiQueueIfPossible(tx *gorm.DB, merchant *models.Merchant, now time.Time) (bool, error) {
	if tx == nil || merchant == nil {
		return false, nil
	}
	if merchant.ID == 0 {
		return false, nil
	}
	if queue.Default == nil {
		return false, nil
	}
	if !merchant.SupportQueue || merchant.QueueMode != "auto" {
		return false, nil
	}
	if !merchant.SupportMultiCustomerService {
		return false, nil
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	activeSessionStatuses := models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"})

	type candLite struct {
		ID           uint `gorm:"column:id"`
		TechnicianID uint `gorm:"column:technician_id"`
	}

	var cand candLite
	err := tx.
		Model(&models.TechnicianAttendance{}).
		Select("technician_attendances.id, technician_attendances.technician_id").
		Joins("JOIN technicians t ON t.id = technician_attendances.technician_id").
		Joins("LEFT JOIN service_roles sr ON sr.id = t.service_role_id").
		Where("technician_attendances.merchant_id = ? AND technician_attendances.checked_in_at >= ? AND technician_attendances.checked_out_at IS NULL AND technician_attendances.status IN ('idle')", merchant.ID, start).
		Where("NOT EXISTS (SELECT 1 FROM service_sessions ss WHERE ss.merchant_id = ? AND ss.technician_id = technician_attendances.technician_id AND ss.status IN ? AND ss.updated_at >= ?)", merchant.ID, activeSessionStatuses, start).
		Where("t.is_active = ?", true).
		// 自动多客服叫号：按“可服务且在岗”选人，避免历史岗位 role_type 数据不一致导致空闲技师被误排除。
		Where("(sr.id IS NULL OR sr.`key` NOT IN ('store_manager','front_desk'))").
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
		Where("id = ? AND merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL AND status = ?", cand.ID, merchant.ID, cand.TechnicianID, start, "idle").
		First(&att).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	date := now.Format("2006-01-02")
	nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	if nextUsageID == 0 {
		return false, nil
	}

	q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND initial_usage_id = ? AND start_confirmed_at IS NULL AND technician_id IS NULL", merchant.ID, nextUsageID)
	q = q.Where("status IN ?", models.ExpandStatusesWithKnownPrefixes([]string{"staff_selecting", "room_locked", "timeout_waiting"}))

	var nextSession models.ServiceSession
	if err := q.Order("id desc").First(&nextSession).Error; err != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	updates := map[string]interface{}{
		"technician_id":                 cand.TechnicianID,
		"status":                        models.ApplyStatusPrefix(nextSession.Status, "start_pending"),
		"staff_select_entered_at":       nil,
		"staff_select_cooldown_until":   nil,
		"start_pending_timeout_seconds": config.MerchantQueueWaitingStartSeconds(merchant),
	}
	result := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND merchant_id = ? AND technician_id IS NULL AND start_confirmed_at IS NULL", nextSession.ID, merchant.ID).
		Updates(updates)
	if result.Error != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return false, nil
	}

	if err := tx.Model(&models.TechnicianAttendance{}).
		Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, merchant.ID, cand.TechnicianID, "idle").
		Updates(map[string]interface{}{"status": "busy"}).Error; err != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return false, err
	}

	return true, nil
}

func StartServiceSessionScheduler() {
	if queueDebugEnabledFor(0, 0, 0) {
		log.Printf("[queue-debug] service session scheduler started, tick=%s\n", schedulerTickInterval)
	}
	go func() {
		ticker := time.NewTicker(schedulerTickInterval)
		defer ticker.Stop()
		recordSchedulerTick(schedulerNameServiceSession, time.Now())

		for range ticker.C {
			if config.DB == nil {
				continue
			}
			recordSchedulerTick(schedulerNameServiceSession, time.Now())
			if err := runOnce(config.DB); err != nil {
				log.Printf("service session scheduler error: %v", err)
			}
		}
	}()
}

func runOnce(db *gorm.DB) error {
	now := time.Now()
	// 手动叫号自动收尾：
	// 1) serving 超过4小时未人工结单，自动置为完成
	// 2) 打烊结束叫号(queue_ended_at)后15分钟内，将未完成核销批量置为完成
	if err := finalizeOverdueManualServingSessions(db, now); err != nil {
		log.Printf("finalize overdue manual serving sessions error: %v", err)
	}
	if err := finalizeUsagesAfterQueueEnded(db, now); err != nil {
		log.Printf("finalize usages after queue ended error: %v", err)
	}
	if err := autoFinalizeStaleUsages(db, now); err != nil {
		log.Printf("auto finalize stale usages error: %v", err)
	}
	if err := backfillLegacyServiceSessionModes(db); err != nil {
		log.Printf("backfill legacy service session modes error: %v", err)
	}
	if err := backfillServingStartConfirmedAt(db, now); err != nil {
		log.Printf("backfill serving start_confirmed_at error: %v", err)
	}
	if err := releaseFinishedSessionTechnicians(db, now); err != nil {
		log.Printf("release finished session technicians error: %v", err)
	}

	type schedulerSessionLite struct {
		ID             uint   `gorm:"column:id"`
		MerchantID     uint   `gorm:"column:merchant_id"`
		InitialUsageID uint   `gorm:"column:initial_usage_id"`
		Status         string `gorm:"column:status"`
	}
	var sessions []schedulerSessionLite
	err := db.
		Table("service_sessions").
		Select("id", "merchant_id", "initial_usage_id", "status").
		Where("status IN ?", models.ExpandStatusesWithKnownPrefixes([]string{"room_selecting", "room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing", "timeout_waiting", "timeout_failed"})).
		Order("id asc").
		Limit(schedulerBatchLimit).
		Find(&sessions).Error
	if err != nil {
		return err
	}
	if queueDebugEnabledFor(0, 0, 0) {
		tw := make([]string, 0)
		for i := range sessions {
			s := sessions[i]
			if strings.Contains(s.Status, "timeout_waiting") {
				tw = append(tw,
					func() string {
						return "id=" +
							func() string { return strconv.FormatUint(uint64(s.ID), 10) }() +
							" status=" + s.Status +
							" usage=" + func() string { return strconv.FormatUint(uint64(s.InitialUsageID), 10) }()
					}(),
				)
			}
		}
		log.Printf("[queue-debug] scheduler runOnce: sessions=%d timeout_waiting=%d [%s]\n", len(sessions), len(tw), strings.Join(tw, ", "))
	}

	for i := range sessions {
		s := sessions[i]
		if queueDebugEnabledFor(s.MerchantID, s.ID, s.InitialUsageID) && strings.Contains(s.Status, "timeout_waiting") {
			log.Printf("[queue-debug] scheduler advance timeout_waiting: session=%d status=%s usage=%d\n", s.ID, s.Status, s.InitialUsageID)
		}
		if err := advanceOne(db, &models.ServiceSession{ID: s.ID}, now); err != nil {
			log.Printf("advance session %d error: %v", s.ID, err)
		}
	}
	return nil
}

func failStartPendingAndAssignNext(tx *gorm.DB, s *models.ServiceSession, merchant *models.Merchant, now time.Time) error {
	if tx == nil || s == nil || merchant == nil {
		return nil
	}
	techID := uint(0)
	if s.TechnicianID != nil {
		techID = *s.TechnicianID
	}

	updates := map[string]interface{}{
		"status":                        models.ApplyStatusPrefix(s.Status, "canceled"),
		"finished_at":                   now,
		"technician_id":                 nil,
		"start_confirmed_at":            nil,
		"scheduled_start_at":            nil,
		"staff_select_entered_at":       nil,
		"staff_select_cooldown_until":   nil,
		"start_pending_timeout_seconds": 0,
		"start_timeout_count":           gorm.Expr("start_timeout_count + ?", 1),
		"start_timeout_last_at":         now,
	}
	if techID > 0 {
		updates["last_technician_id"] = techID
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ? AND start_confirmed_at IS NULL", s.ID, models.ExpandStatusWithKnownPrefixes("start_pending")).
		Updates(updates).Error; err != nil {
		return err
	}

	if s.InitialUsageID > 0 {
		if err := sessionflow.FailUsageAndRefund(tx, s.InitialUsageID, merchant, now, true); err != nil {
			return err
		}
	}

	if techID > 0 {
		_ = tx.Model(&models.TechnicianAttendance{}).
			Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, techID, "busy").
			Updates(map[string]interface{}{"status": "idle"}).Error
		_ = autoCallNextForTechnician(tx, s.MerchantID, techID, now)
	}

	return nil
}

func advanceOne(db *gorm.DB, session *models.ServiceSession, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		sPtr, err := loadServiceSessionByID(tx.Clauses(clause.Locking{Strength: "UPDATE"}), session.ID)
		if err != nil {
			return err
		}
		if sPtr == nil {
			return gorm.ErrRecordNotFound
		}
		s := *sPtr
		if queueDebugEnabledFor(s.MerchantID, s.ID, s.InitialUsageID) && strings.Contains(s.Status, "timeout_waiting") {
			log.Printf("[queue-debug] advanceOne reload: session=%d status=%s baseStatus=%s merchant=%d usage=%d\n",
				s.ID, s.Status, models.NormalizeSessionStatus(s.Status), s.MerchantID, s.InitialUsageID)
		}

		// 模式一致性守卫：若 session_mode 与 status 前缀矛盾则跳过，
		// 避免跨模式错误推进（如商户切换模式后残留的异常会话）。
		// 必须放在任何可能写库推进状态的逻辑之前（例如 advanceQueueAutoSingleSerialIfPossible）。
		if !models.IsModeConsistentWithStatus(&s) {
			log.Printf("[mode-guard] advanceOne skip: session=%d status=%s mode=%s (mode/status prefix mismatch)\n",
				s.ID, s.Status, s.SessionMode)
			return nil
		}

		// 单队列串行启动（自动叫号 + 未开启多个客服）：
		// 核销后会话可能被创建出来但 start_confirmed_at 为空（表示仍在排队等待）。
		// 只有当它成为队列头(CurrentID)且已叫号(CurrentCalledAt!=nil)，并且当前没有其他进行中的会话时，
		// 才允许写入 start_confirmed_at/scheduled_start_at 进入 delay_pending，随后再推进到 serving。
		if err := advanceQueueAutoSingleSerialIfPossible(tx, &s, now); err != nil {
			return err
		}
		baseAt := s.UpdatedAt
		if baseAt == nil {
			baseAt = s.CreatedAt
		}
		if s.TechnicianID == nil && s.StartedAt == nil && baseAt != nil {
			// timeout_waiting/timeout_failed 需要走专门的过期判断，不应被 60min 兜底取消逻辑抢跑
			st := models.NormalizeSessionStatus(s.Status)
			if st != "timeout_waiting" && st != "timeout_failed" {
				if now.Sub(*baseAt) >= sessionAbandonTimeout {
					var merchant models.Merchant
					if err := tx.First(&merchant, s.MerchantID).Error; err == nil {
						if merchant.SupportQueue && merchant.QueueMode == "manual" {
							// 手动叫号模式不走 60min 兜底取消（等待人工），但允许后续状态机做跨天/超时兜底回滚
						} else {
							return cancelAndReleaseSession(tx, &s, now)
						}
					}
				}
			}
		}

		baseStatus := models.NormalizeSessionStatus(s.Status)
		switch baseStatus {
		case "room_locked", "staff_selecting":
			return handleRoomLockedOrStaffSelecting(tx, &s, now)
		case "room_selecting":
			return handleRoomSelecting(tx, &s, now)
		case "start_pending":
			return handleStartPending(tx, &s, now)
		case "delay_pending":
			return handleDelayPending(tx, &s, now)
		case "serving":
			return handleServing(tx, &s, now)
		case "auto_finishing":
			return handleAutoFinishing(tx, &s, now)
		case "timeout_waiting":
			return handleTimeoutWaiting(tx, &s, now)
		case "timeout_failed":
			return handleTimeoutFailed(tx, &s, now)
		case "finished":
			return releaseTechnicianIfNeeded(tx, &s, now)
		default:
			return nil
		}
	})
}

func handleTimeoutFailed(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	if tx == nil || s == nil {
		return nil
	}
	if s.InitialUsageID == 0 {
		return nil
	}

	finishedAt := now
	if s.FinishedAt != nil {
		finishedAt = *s.FinishedAt
	}

	var merchant models.Merchant
	if err := tx.Select("id", "support_queue").First(&merchant, s.MerchantID).Error; err == nil {
		if err := sessionflow.FailUsageAndRefund(tx, s.InitialUsageID, &merchant, finishedAt, merchant.SupportQueue); err != nil {
			return err
		}
	}
	if s.SourceType == "appointment" && s.SourceID != nil {
		if err := syncAppointmentLiabilitySnapshot(tx, *s.SourceID, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &finishedAt,
			"disruption_status":                  "closed",
			"disruption_reason":                  "merchant_timeout_failed",
			"liability_level":                    "merchant",
			"salary_settlement_reference_status": "refund",
		}, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &finishedAt,
			"liability_level":                    "merchant",
			"salary_settlement_reference_status": "refund",
			"latest_reason":                      "merchant_timeout_failed",
		}); err != nil {
			return err
		}
	}

	return nil
}

func advanceQueueAutoSingleSerialIfPossible(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	if tx == nil || s == nil {
		return nil
	}
	if s.StartConfirmedAt != nil {
		return nil
	}
	var merchant models.Merchant
	if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
		return nil
	}
	if !merchant.SupportQueue || merchant.QueueMode != "auto" || merchant.SupportMultiCustomerService {
		return nil
	}
	if queue.Default == nil || s.InitialUsageID == 0 {
		return nil
	}
	date := now.Format("2006-01-02")
	snap := queue.Default.Snapshot(merchant.ID, date, queue.QueueTypeOnsite)
	if snap.CurrentID == 0 || snap.CurrentCalledAt == nil {
		var activeCnt int64
		if err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND status IN ?", s.MerchantID, models.ExpandStatusesWithKnownPrefixes([]string{"start_pending", "delay_pending", "serving", "auto_finishing"})).
			Count(&activeCnt).Error; err == nil {
			if activeCnt == 0 {
				if merchant.QueuePaused {
					return nil
				}
				queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
				snap = queue.Default.Snapshot(merchant.ID, date, queue.QueueTypeOnsite)
			}
		}
	}
	if snap.CurrentID > 0 && snap.CurrentID == s.InitialUsageID && snap.CurrentCalledAt != nil {
		var activeCnt int64
		if err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND id <> ? AND status IN ?", s.MerchantID, s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"start_pending", "delay_pending", "serving", "auto_finishing"})).
			Count(&activeCnt).Error; err == nil {
			if activeCnt == 0 {
				delaySeconds := merchant.StartDelaySeconds
				if delaySeconds <= 0 {
					delaySeconds = 60
				}
				startAt := now.Add(time.Duration(delaySeconds) * time.Second)
				updates := map[string]interface{}{
					"status":              models.ApplyStatusPrefix(s.Status, "delay_pending"),
					"start_confirmed_at":  &now,
					"scheduled_start_at":  &startAt,
					"start_delay_seconds": delaySeconds,
				}
				if err := tx.Model(&models.ServiceSession{}).
					Where("id = ? AND start_confirmed_at IS NULL AND status IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"staff_selecting", "room_locked", "delay_pending"})).
					Updates(updates).Error; err == nil {
					s.StartConfirmedAt = &now
					s.ScheduledStartAt = &startAt
					s.Status = models.ApplyStatusPrefix(s.Status, "delay_pending")
				}
			}
		}
	}
	return nil
}

func handleRoomLockedOrStaffSelecting(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	var merchant models.Merchant
	if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
		return err
	}
	if merchant.SupportQueue && merchant.QueueMode == "manual" {
		abandonTimeout := 12 * time.Hour
		baseAt := s.UpdatedAt
		if baseAt == nil {
			baseAt = s.CreatedAt
		}
		if baseAt != nil {
			if isCrossDay(baseAt, now) || now.Sub(*baseAt) >= abandonTimeout {
				return cancelAndReleaseSession(tx, s, now)
			}
		}
		return nil
	}
	if !merchant.SupportCustomerServiceMode && merchant.SupportQueue && merchant.QueueMode == "auto" && !merchant.SupportMultiCustomerService {
		return nil
	}
	if merchant.SupportQueue && merchant.QueueMode == "auto" && merchant.SupportMultiCustomerService {
		if ok, err := autoCallNextForMultiQueueIfPossible(tx, &merchant, now); err != nil {
			return err
		} else if ok {
			return nil
		}
		return nil
	}
	if !merchant.SupportCustomerServiceMode {
		delaySeconds := merchant.StartDelaySeconds
		if delaySeconds <= 0 {
			delaySeconds = 60
		}
		startAt := now.Add(time.Duration(delaySeconds) * time.Second)
		updates := map[string]interface{}{
			"status":                      models.ApplyStatusPrefix(s.Status, "delay_pending"),
			"technician_id":               nil,
			"staff_select_cooldown_until": nil,
			"staff_select_entered_at":     nil,
			"start_confirmed_at":          &now,
			"scheduled_start_at":          &startAt,
			"room_select_deadline_at":     nil,
		}
		return tx.Model(&models.ServiceSession{}).
			Where("id = ? AND status IN ? AND start_confirmed_at IS NULL", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting"})).
			Updates(updates).Error
	}
	if s.RoomID == nil || s.TechnicianID != nil || s.RoomLockedAt == nil {
		return nil
	}
	if s.StaffSelectCooldownUntil != nil && now.Before(*s.StaffSelectCooldownUntil) {
		return nil
	}
	if s.StaffSelectEnteredAt == nil {
		return nil
	}
	if !merchant.SupportCustomerServiceMode || !merchant.SupportRoom {
		return nil
	}
	deadline := s.StaffSelectEnteredAt.Add(resolveStaffSelectingTimeout(tx, s))
	if now.Before(deadline) {
		return nil
	}
	ok, err := autoAssignTechnicianIfPossible(tx, s, now)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	dl := now.Add(3 * time.Minute)
	updates := map[string]interface{}{
		"status":                      models.ApplyStatusPrefix(s.Status, "staff_selecting"),
		"staff_select_cooldown_until": &dl,
	}
	return tx.Model(&models.ServiceSession{}).
		Where("id = ? AND technician_id IS NULL AND status IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting"})).
		Updates(updates).Error
}

func handleRoomSelecting(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	if s.RoomSelectDeadlineAt != nil && now.After(*s.RoomSelectDeadlineAt) {
		baseAt := s.UpdatedAt
		if baseAt == nil {
			baseAt = s.CreatedAt
		}
		if s.TechnicianID == nil && s.StartedAt == nil && baseAt != nil && now.Sub(*baseAt) >= sessionAbandonTimeout {
			var merchant models.Merchant
			if err := tx.First(&merchant, s.MerchantID).Error; err == nil {
				if merchant.SupportQueue && merchant.QueueMode == "manual" {
					return nil
				}
			}
			return cancelAndReleaseSession(tx, s, now)
		}
		return autoAssignRoom(tx, s, now)
	}
	return nil
}

func handleStartPending(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	{
		var merchant models.Merchant
		if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
			return err
		}
		skipDegradeToDelayPending := false
		if !merchant.SupportCustomerServiceMode && merchant.SupportQueue {
			skipDegradeToDelayPending = true
		}
		if merchant.SupportQueue && merchant.SupportMultiCustomerService {
			skipDegradeToDelayPending = true
		}
		if !skipDegradeToDelayPending && !merchant.SupportCustomerServiceMode {
			delaySeconds := merchant.StartDelaySeconds
			if delaySeconds <= 0 {
				delaySeconds = 60
			}
			startAt := now.Add(time.Duration(delaySeconds) * time.Second)
			updates := map[string]interface{}{
				"status":                        models.ApplyStatusPrefix(s.Status, "delay_pending"),
				"technician_id":                 nil,
				"start_confirmed_at":            &now,
				"scheduled_start_at":            &startAt,
				"staff_select_entered_at":       nil,
				"staff_select_cooldown_until":   nil,
				"start_pending_timeout_seconds": 0,
			}
			if s.TechnicianID != nil && *s.TechnicianID > 0 {
				_ = tx.Model(&models.TechnicianAttendance{}).
					Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
					Updates(map[string]interface{}{"status": "idle"}).Error
			}
			return tx.Model(&models.ServiceSession{}).
				Where("id = ? AND status IN ? AND start_confirmed_at IS NULL", s.ID, models.ExpandStatusWithKnownPrefixes("start_pending")).
				Updates(updates).Error
		}
	}
	if s.StartConfirmedAt != nil {
		return nil
	}
	if s.UpdatedAt == nil {
		return nil
	}
	startDeadline := s.UpdatedAt.Add(getStartPendingTimeoutForSession(s))
	if now.Before(startDeadline) {
		return nil
	}
	{
		var merchant models.Merchant
		if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
			return err
		}
		if !merchant.SupportCustomerServiceMode && merchant.SupportQueue && merchant.QueueMode == "manual" {
			abandonTimeout := 12 * time.Hour
			baseAt := s.UpdatedAt
			if baseAt == nil {
				baseAt = s.CreatedAt
			}
			isCrossDay2 := false
			if baseAt != nil {
				baseDate := time.Date(baseAt.Year(), baseAt.Month(), baseAt.Day(), 0, 0, 0, 0, baseAt.Location())
				nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				if nowDate.After(baseDate) {
					isCrossDay2 = true
				}
			}
			if baseAt != nil && (now.Sub(*baseAt) >= abandonTimeout || isCrossDay2) {
				return failStartPendingAndAssignNext(tx, s, &merchant, now)
			}
			return nil
		}
	}
	{
		var merchant models.Merchant
		if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
			return err
		}
		if merchant.SupportQueue && merchant.QueueMode == "auto" && merchant.SupportMultiCustomerService {
			return moveMultiQueueStartPendingToTimeoutWaiting(tx, s, &merchant, now)
		}
	}
	updates := map[string]interface{}{
		"status":                        models.ApplyStatusPrefix(s.Status, "staff_selecting"),
		"technician_id":                 nil,
		"staff_select_entered_at":       nil,
		"start_pending_timeout_seconds": 0,
		"start_timeout_count":           gorm.Expr("start_timeout_count + ?", 1),
		"start_timeout_last_at":         now,
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ? AND start_confirmed_at IS NULL", s.ID, models.ExpandStatusWithKnownPrefixes("start_pending")).
		Updates(updates).Error; err != nil {
		return err
	}
	if s.InitialUsageID > 0 && queue.Default != nil {
		date := now.Format("2006-01-02")
		queue.Default.Uncall(s.MerchantID, date, queue.QueueTypeOnsite, s.InitialUsageID)
	}
	if s.TechnicianID != nil && *s.TechnicianID > 0 {
		if err := tx.Model(&models.TechnicianAttendance{}).
			Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
			Updates(map[string]interface{}{"status": "idle"}).Error; err != nil {
			return err
		}
	}
	return nil
}

func handleDelayPending(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	var merchant models.Merchant
	if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
		return err
	}
	if merchant.SupportQueue && merchant.QueueMode == "auto" && merchant.SupportMultiCustomerService {
		if s.TechnicianID == nil || *s.TechnicianID == 0 {
			if queue.Default != nil && s.InitialUsageID > 0 {
				date := now.Format("2006-01-02")
				queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID)
			}
			updates := map[string]interface{}{
				"status":                      models.ApplyStatusPrefix(s.Status, "staff_selecting"),
				"start_confirmed_at":          nil,
				"scheduled_start_at":          nil,
				"technician_id":               nil,
				"staff_select_entered_at":     nil,
				"staff_select_cooldown_until": nil,
			}
			return tx.Model(&models.ServiceSession{}).
				Where("id = ? AND status IN ?", s.ID, models.ExpandStatusWithKnownPrefixes("delay_pending")).
				Updates(updates).Error
		}
		return nil
	}
	if !merchant.SupportCustomerServiceMode && merchant.SupportQueue && merchant.QueueMode == "auto" {
		if s.ScheduledStartAt != nil {
			timeoutAt := s.ScheduledStartAt.Add(config.StartScanTimeout())
			if now.After(timeoutAt) {
				return skipCurrentAndCallNext(tx, s, &merchant, now)
			}
		}
		return nil
	}
	if s.ScheduledStartAt != nil && !now.Before(*s.ScheduledStartAt) {
		updates := map[string]interface{}{
			"status":     models.ApplyStatusPrefix(s.Status, "serving"),
			"started_at": now,
		}
		if s.ScheduledFinishAt == nil {
			if s.DurationMinutes > 0 {
				finishAt := now.Add(time.Duration(s.DurationMinutes) * time.Minute)
				updates["scheduled_finish_at"] = finishAt
			}
		}
		if merchant.SupportQueue && merchant.QueueMode == "manual" && !merchant.SupportMultiCustomerService {
			if s.TechnicianID == nil || *s.TechnicianID == 0 {
				start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				var att models.TechnicianAttendance
				attRes := tx.
					Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("merchant_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL AND status = ?", s.MerchantID, start, "idle").
					Order("updated_at asc").
					Limit(1).
					Find(&att)
				if attRes.Error == nil && attRes.RowsAffected > 0 {
					updates["technician_id"] = att.TechnicianID
					updates["last_technician_id"] = att.TechnicianID
					_ = tx.Model(&models.TechnicianAttendance{}).
						Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, s.MerchantID, att.TechnicianID, "idle").
						Updates(map[string]interface{}{"status": "busy"}).Error
				}
			}
		}
		if err := tx.Model(&models.ServiceSession{}).
			Where("id = ? AND status IN ? AND start_confirmed_at IS NOT NULL", s.ID, models.ExpandStatusWithKnownPrefixes("delay_pending")).
			Updates(updates).Error; err != nil {
			return err
		}
		if s.SourceType == "appointment" && s.SourceID != nil {
			return tx.Model(&models.Appointment{}).Where("id = ? AND actual_start_at IS NULL", *s.SourceID).Update("actual_start_at", now).Error
		}
		return nil
	}
	return nil
}

func handleServing(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	if s.StartConfirmedAt == nil {
		if s.StartedAt != nil {
			if err := tx.Model(&models.ServiceSession{}).
				Where("id = ? AND status IN ? AND start_confirmed_at IS NULL", s.ID, models.ExpandStatusWithKnownPrefixes("serving")).
				Update("start_confirmed_at", *s.StartedAt).Error; err != nil {
				return err
			}
			s.StartConfirmedAt = s.StartedAt
		} else {
			return nil
		}
	}
	if s.ScheduledFinishAt != nil && now.After(*s.ScheduledFinishAt) {
		var merchant models.Merchant
		if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
			return err
		}
		if merchant.SupportQueue && merchant.QueueMode == "auto" && merchant.SupportMultiCustomerService {
			return finalizeSession(tx, s, now)
		}
		if !merchant.SupportCustomerServiceMode {
			return finalizeSession(tx, s, now)
		}
		finishAt := s.ScheduledFinishAt.Add(time.Duration(s.AutoFinishDelaySeconds) * time.Second)
		updates := map[string]interface{}{
			"status":      models.ApplyStatusPrefix(s.Status, "auto_finishing"),
			"finished_at": finishAt,
		}
		return tx.Model(&models.ServiceSession{}).
			Where("id = ? AND status IN ? AND start_confirmed_at IS NOT NULL", s.ID, models.ExpandStatusWithKnownPrefixes("serving")).
			Updates(updates).Error
	}
	return nil
}

func handleAutoFinishing(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	if s.StartConfirmedAt == nil {
		return nil
	}
	if s.FinishedAt != nil && !now.Before(*s.FinishedAt) {
		return finalizeSession(tx, s, now)
	}
	return nil
}

func handleTimeoutWaiting(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	if s.InitialUsageID == 0 {
		if queueDebugEnabledFor(s.MerchantID, s.ID, s.InitialUsageID) {
			log.Printf("[queue-debug] timeout_waiting skip: merchant=%d session=%d usage=%d reason=%s\n",
				s.MerchantID, s.ID, s.InitialUsageID, "empty_initial_usage")
		}
		return nil
	}
	var merchant models.Merchant
	if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
		if queueDebugEnabledFor(s.MerchantID, s.ID, s.InitialUsageID) {
			log.Printf("[queue-debug] timeout_waiting merchant load failed: merchant=%d session=%d usage=%d err=%v\n",
				s.MerchantID, s.ID, s.InitialUsageID, err)
		}
		return nil
	}
	if queueDebugEnabledFor(merchant.ID, s.ID, s.InitialUsageID) {
		log.Printf("[queue-debug] timeout_waiting merchant loaded: merchant=%d session=%d usage=%d supportQueue=%v queueMode=%s supportMCS=%v\n",
			merchant.ID, s.ID, s.InitialUsageID, merchant.SupportQueue, merchant.QueueMode, merchant.SupportMultiCustomerService)
	}
	mode := resolveQueueSessionModeForTimeoutWaiting(s, &merchant)
	if mode == "" {
		if queueDebugEnabledFor(merchant.ID, s.ID, s.InitialUsageID) {
			log.Printf("[queue-debug] timeout_waiting skip: merchant=%d session=%d usage=%d reason=%s\n",
				merchant.ID, s.ID, s.InitialUsageID, "non_queue_session_mode")
		}
		return nil
	}

	baseAt := s.StartTimeoutLastAt
	if baseAt == nil {
		baseAt = s.UpdatedAt
	}
	if baseAt == nil {
		baseAt = s.CreatedAt
	}

	timeoutWaitingSeconds := config.MerchantQueueTimeoutWaitingSeconds(&merchant)
	timeoutWaitingDuration := time.Duration(timeoutWaitingSeconds) * time.Second

	if mode == models.SessionModeQueueAutoMulti {
		if baseAt != nil && now.Sub(*baseAt) > timeoutWaitingDuration {
			return failTimeoutWaitingAndRefund(tx, s, &merchant, now)
		}
		return nil
	}

	if queue.Default == nil {
		if baseAt != nil && isCrossDay(baseAt, now) {
			return failTimeoutWaitingAndRefund(tx, s, &merchant, now)
		}
		if queueDebugEnabledFor(merchant.ID, s.ID, s.InitialUsageID) {
			log.Printf("[queue-debug] timeout_waiting skip: merchant=%d session=%d usage=%d reason=%s\n",
				merchant.ID, s.ID, s.InitialUsageID, "queue_default_nil")
		}
		return nil
	}
	date := now.Format("2006-01-02")
	snap := queue.Default.Snapshot(merchant.ID, date, queue.QueueTypeOnsite)
	minNo := 0
	if len(snap.Tickets) > 0 {
		minNo = snap.Tickets[0].No
	}
	maxCalledNo := snap.MaxCalledNo
	myNo, ok := queue.Default.GetNo(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID)
	if queueDebugEnabledFor(merchant.ID, s.ID, s.InitialUsageID) {
		log.Printf("[queue-debug] timeout_waiting check: merchant=%d session=%d usage=%d mode=%s status=%s tickets=%d minNo=%d maxCalledNo=%d getNoOk=%v myNo=%d lastAt=%v now=%v\n",
			merchant.ID, s.ID, s.InitialUsageID, merchant.QueueMode, s.Status, len(snap.Tickets), minNo, maxCalledNo, ok, myNo, s.StartTimeoutLastAt, now)
	}
	if !ok || myNo <= 0 || minNo <= 0 {
		if !ok {
			if isCrossDay(baseAt, now) {
				if queueDebugEnabledFor(merchant.ID, s.ID, s.InitialUsageID) {
					log.Printf("[queue-debug] timeout_waiting get_no_failed cross-day -> timeout_failed: merchant=%d session=%d usage=%d baseAt=%v now=%v\n",
						merchant.ID, s.ID, s.InitialUsageID, baseAt, now)
				}
				return failTimeoutWaitingAndRefund(tx, s, &merchant, now)
			}
		}
		if queueDebugEnabledFor(merchant.ID, s.ID, s.InitialUsageID) {
			log.Printf("[queue-debug] timeout_waiting skip: merchant=%d session=%d usage=%d reason=%s\n",
				merchant.ID, s.ID, s.InitialUsageID,
				func() string {
					if !ok {
						return "get_no_failed"
					}
					if myNo <= 0 {
						return "invalid_my_no"
					}
					return "min_no_empty_snapshot"
				}(),
			)
		}
		return nil
	}
	if mode == models.SessionModeQueueAutoSingle {
		currentNo := minNo
		cnt := s.StartTimeoutCount
		if models.QsTimeoutWaitingExpired(currentNo, myNo, cnt) {
			return failTimeoutWaitingAndRefund(tx, s, &merchant, now)
		}
		return nil
	}
	if mode == models.SessionModeQueueManualSingle || mode == models.SessionModeQueueManualMulti {
		currentNo := maxCalledNo
		if currentNo <= 0 {
			currentNo = minNo
		}
		endNo := myNo + 3
		exceedNoWindow := currentNo >= endNo+1
		exceedTimeWindow := false
		if baseAt != nil {
			exceedTimeWindow = now.Sub(*baseAt) > timeoutWaitingDuration
		}
		if queueDebugEnabledFor(merchant.ID, s.ID, s.InitialUsageID) {
			log.Printf("[queue-debug] timeout_waiting manual: merchant=%d session=%d usage=%d myNo=%d currentNo=%d endNo=%d exceedNo=%v exceedTime=%v baseAt=%v\n",
				merchant.ID, s.ID, s.InitialUsageID, myNo, currentNo, endNo, exceedNoWindow, exceedTimeWindow, baseAt)
		}
		if exceedNoWindow && exceedTimeWindow {
			if queueDebugEnabledFor(merchant.ID, s.ID, s.InitialUsageID) {
				log.Printf("[queue-debug] timeout_waiting -> timeout_failed: merchant=%d session=%d usage=%d\n", merchant.ID, s.ID, s.InitialUsageID)
			}
			return failTimeoutWaitingAndRefund(tx, s, &merchant, now)
		}
		return nil
	}
	return nil
}

func backfillServingStartConfirmedAt(db *gorm.DB, now time.Time) error {
	if db == nil {
		return nil
	}

	// 仅处理明显异常的数据：serving 且 started_at 有值，但 start_confirmed_at 为空。
	// 这类数据会被 advanceOne(serving) 的前置条件永久跳过，导致服务看板长期残留“服务中”。
	type lite struct {
		ID        uint
		StartedAt string `gorm:"column:started_at"`
		Status    string
	}
	var rows []lite
	if err := db.Table("service_sessions").
		Select("id", "status", "started_at").
		Where("status IN ? AND start_confirmed_at IS NULL AND started_at IS NOT NULL", models.ExpandStatusWithKnownPrefixes("serving")).
		Order("id asc").
		Limit(schedulerBatchLimit).
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	for i := range rows {
		r := rows[i]
		if strings.TrimSpace(r.StartedAt) == "" {
			continue
		}
		startedAt, err := parseFinishedAtFromDBString(r.StartedAt, now.Location())
		if err != nil {
			continue
		}
		_ = db.Model(&models.ServiceSession{}).
			Where("id = ? AND status IN ? AND start_confirmed_at IS NULL", r.ID, models.ExpandStatusWithKnownPrefixes("serving")).
			Update("start_confirmed_at", startedAt).Error
	}

	return nil
}

func failTimeoutWaitingAndRefund(tx *gorm.DB, s *models.ServiceSession, merchant *models.Merchant, now time.Time) error {
	return sessionflow.FailServiceSessionAndRefund(tx, s, merchant, now, sessionflow.FailOptions{
		AllowedBaseStatuses: []string{"timeout_waiting"},
		TargetStatus:        "timeout_failed",
		MarkQueueDone:       true,
	})
}

func autoAssignRoom(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	var merchant models.Merchant
	if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
		return err
	}
	if !merchant.SupportCustomerServiceMode || !merchant.SupportRoom {
		updates := map[string]interface{}{
			"status": models.ApplyStatusPrefix(s.Status, "staff_selecting"),
		}
		return tx.Model(&models.ServiceSession{}).Where("id = ? AND status IN ?", s.ID, models.ExpandStatusWithKnownPrefixes("room_selecting")).Updates(updates).Error
	}

	var rooms []models.Room
	if err := tx.Where("merchant_id = ? AND is_active = ?", s.MerchantID, true).Order("id asc").Find(&rooms).Error; err != nil {
		return err
	}
	for i := range rooms {
		r := rooms[i]
		var cnt int64
		if err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND room_id = ? AND status IN ?", s.MerchantID, r.ID, models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"})).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			lockedAt := now
			updates := map[string]interface{}{
				"room_id":        r.ID,
				"room_locked_at": lockedAt,
				"status":         models.ApplyStatusPrefix(s.Status, "room_locked"),
			}
			return tx.Model(&models.ServiceSession{}).
				Where("id = ? AND status IN ?", s.ID, models.ExpandStatusWithKnownPrefixes("room_selecting")).
				Updates(updates).Error
		}
	}
	return nil
}

func finalizeSession(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	var merchant models.Merchant
	if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
		return nil
	}
	if err := sessionflow.FinishServiceSession(tx, s, &merchant, now, sessionflow.FinishOptions{
		AllowedBaseStatuses: []string{"serving", "auto_finishing"},
		MarkQueueDone:       merchant.SupportQueue,
	}); err != nil {
		return err
	}
	if merchant.SupportQueue && merchant.QueueMode == "auto" {
		date := now.Format("2006-01-02")
		if shouldAutoCallNextInAutoSingleQueue(&merchant) {
			queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
		}
		// 多客服模式下，会在 releaseTechnicianIfNeeded/autoCallNextForTechnician 中触发
	}
	// 手动叫号模式：不自动触发，需要客服点击“开始叫号”或者扫码结单时手动触发
	// 手动叫号的触发在 queue_status.go 的 TriggerNextCalling 接口中实现
	if s.SourceType == "appointment" && s.SourceID != nil {
		actualStartAt := now
		if s.StartedAt != nil {
			actualStartAt = *s.StartedAt
		}
		if err := syncAppointmentLiabilitySnapshot(tx, *s.SourceID, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &now,
			"disruption_status":                  "closed",
			"disruption_reason":                  "service_completed",
			"liability_level":                    "none",
			"salary_settlement_reference_status": "normal",
			"actual_start_at":                    actualStartAt,
		}, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &now,
			"liability_level":                    "none",
			"salary_settlement_reference_status": "normal",
			"latest_reason":                      "service_completed",
		}); err != nil {
			return err
		}
		if err := appointmentdelay.RecordLedgerIfNeeded(tx, *s.SourceID, &s.ID, actualStartAt); err != nil {
			return err
		}
	}

	return nil
}

// skipCurrentAndCallNext 叫号模式下超时未扫码上号，跳过当前叫号并自动叫下一个
func skipCurrentAndCallNext(tx *gorm.DB, s *models.ServiceSession, merchant *models.Merchant, now time.Time) error {
	if tx == nil || s == nil || merchant == nil {
		return nil
	}

	// qs_ 单窗口：超时进入 timeout_waiting（允许插队窗口），不立即失败/退卡。
	// NormalizeLegacySessionMode 统一处理历史空 session_mode 的回退逻辑。
	isQueueAutoSingleMode := models.NormalizeLegacySessionMode(s, merchant) == models.SessionModeQueueAutoSingle
	if isQueueAutoSingleMode {
		updates := map[string]interface{}{
			"status":                models.ApplyStatusPrefix(s.Status, "timeout_waiting"),
			"start_confirmed_at":    nil,
			"scheduled_start_at":    nil,
			"started_at":            nil,
			"start_timeout_count":   gorm.Expr("start_timeout_count + ?", 1),
			"start_timeout_last_at": now,
		}
		if s.TechnicianID != nil && *s.TechnicianID > 0 {
			updates["last_technician_id"] = *s.TechnicianID
		}
		if err := tx.Model(&models.ServiceSession{}).
			Where("id = ? AND status IN ?", s.ID, models.ExpandStatusWithKnownPrefixes("delay_pending")).
			Updates(updates).Error; err != nil {
			return err
		}

		if queue.Default != nil && s.InitialUsageID > 0 {
			date := now.Format("2006-01-02")
			queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID)
			queue.Default.MarkDone(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID, now)
			if shouldAutoCallNextInAutoSingleQueue(merchant) {
				queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
			}
		}
		return nil
	}

	// 其它叫号模式：保持原逻辑，直接失败并退卡
	if err := sessionflow.FailServiceSessionAndRefund(tx, s, merchant, now, sessionflow.FailOptions{
		AllowedBaseStatuses: []string{"delay_pending"},
		TargetStatus:        "canceled",
		MarkQueueDone:       merchant.SupportQueue,
		SessionUpdates: map[string]interface{}{
			"start_confirmed_at": nil,
			"scheduled_start_at": nil,
		},
	}); err != nil {
		return err
	}

	// 自动叫下一个号
	if merchant.SupportQueue && s.InitialUsageID > 0 {
		date := now.Format("2006-01-02")
		if shouldAutoCallNextInAutoSingleQueue(merchant) {
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
		Select("id", "status", "next_status").
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
	// 仅自动叫号模式才自动分配
	if !merchant.SupportQueue || merchant.QueueMode != "auto" {
		return nil
	}
	if !merchant.SupportMultiCustomerService {
		return nil
	}
	// 商户全局暂停叫号时，不自动分配
	if merchant.QueuePaused {
		return nil
	}

	// 检查技师是否暂停了叫号
	var tech models.Technician
	if err := tx.Where("id = ? AND merchant_id = ?", technicianID, merchantID).First(&tech).Error; err != nil {
		return nil
	}
	if tech.QueuePaused {
		return nil
	}
	// 多窗口叫号：通过对考勤记录加行锁 + 将会话置为 start_pending(带 technician_id) 实现并发控制。
	// 注意：不要在这里把技师置为 busy，busy 仍由扫码起单时完成（见 handleQueueModeStartScan）。
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var att models.TechnicianAttendance
	attRes := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, technicianID, start).
		Order("id desc").
		Limit(1).
		Find(&att)
	if attRes.Error != nil || attRes.RowsAffected == 0 {
		return nil
	}
	if att.Status != "idle" {
		return nil
	}

	{
		var cnt int64
		err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND technician_id = ? AND status IN ? AND updated_at >= ?", merchantID, technicianID, models.ExpandStatusesWithKnownPrefixes([]string{"start_pending", "delay_pending", "serving", "auto_finishing"}), start).
			Count(&cnt).Error
		if err != nil || cnt > 0 {
			return nil
		}
	}

	date := now.Format("2006-01-02")
	nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	if nextUsageID == 0 {
		return nil
	}

	var nextSession models.ServiceSession
	q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND initial_usage_id = ? AND start_confirmed_at IS NULL AND technician_id IS NULL", merchantID, nextUsageID)
	q = q.Where("status IN ?", models.ExpandStatusesWithKnownPrefixes([]string{"staff_selecting", "room_locked", "timeout_waiting"}))
	if err := q.Order("id desc").First(&nextSession).Error; err != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return nil
	}

	updates := map[string]interface{}{
		"technician_id":                 technicianID,
		"status":                        models.ApplyStatusPrefix(nextSession.Status, "start_pending"),
		"staff_select_entered_at":       nil,
		"staff_select_cooldown_until":   nil,
		"start_pending_timeout_seconds": config.MerchantQueueWaitingStartSeconds(&merchant),
	}
	result := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND merchant_id = ? AND technician_id IS NULL AND start_confirmed_at IS NULL", nextSession.ID, merchantID).
		Updates(updates)
	if result.Error != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return nil
	}
	if result.RowsAffected == 0 {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return nil
	}

	// 叫号多客服模式：分配技师后，将技师状态从 idle 改为 busy，避免重复分配
	if err := tx.Model(&models.TechnicianAttendance{}).
		Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, merchantID, technicianID, "idle").
		Updates(map[string]interface{}{"status": "busy"}).Error; err != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return err
	}

	return nil
}
