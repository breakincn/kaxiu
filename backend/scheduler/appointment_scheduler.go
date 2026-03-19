package scheduler

import (
	"kabao/config"
	"kabao/models"
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	appointmentSchedulerTickInterval = 15 * time.Second
	appointmentAssignDelay           = 10 * time.Minute
	appointmentSchedulerBatchLimit   = 200
)

func merchantAppointmentPredictionBufferMinutes(m models.Merchant) int {
	if m.AppointmentPredictionBufferMinute > 0 {
		return m.AppointmentPredictionBufferMinute
	}
	return 5
}

func merchantAppointmentGraceWindowMinutes(m models.Merchant) int {
	if m.AppointmentGraceWindowMinutes > 0 {
		return m.AppointmentGraceWindowMinutes
	}
	return 15
}

func parseSchedulerDBTimeString(s string) (time.Time, bool) {
	value := strings.TrimSpace(s)
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func parseSchedulerDBTimeValue(raw interface{}) (*time.Time, bool) {
	switch v := raw.(type) {
	case nil:
		return nil, false
	case time.Time:
		tm := v
		return &tm, true
	case *time.Time:
		if v == nil {
			return nil, false
		}
		tm := *v
		return &tm, true
	case []byte:
		if parsed, ok := parseSchedulerDBTimeString(string(v)); ok {
			return &parsed, true
		}
	case string:
		if parsed, ok := parseSchedulerDBTimeString(v); ok {
			return &parsed, true
		}
	}
	return nil, false
}

func schedulerValueToUint(v interface{}) (uint, bool) {
	switch vv := v.(type) {
	case uint:
		return vv, true
	case uint8:
		return uint(vv), true
	case uint16:
		return uint(vv), true
	case uint32:
		return uint(vv), true
	case uint64:
		return uint(vv), true
	case int:
		if vv >= 0 {
			return uint(vv), true
		}
	case int64:
		if vv >= 0 {
			return uint(vv), true
		}
	case []byte:
		n, err := strconv.ParseUint(strings.TrimSpace(string(vv)), 10, 64)
		if err == nil {
			return uint(n), true
		}
	case string:
		n, err := strconv.ParseUint(strings.TrimSpace(vv), 10, 64)
		if err == nil {
			return uint(n), true
		}
	}
	return 0, false
}

func loadSchedulerAppointmentByID(tx *gorm.DB, appointmentID uint) (*models.Appointment, error) {
	if tx == nil || appointmentID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("appointments").
		Select("id, card_id, merchant_id, user_id, project_id, technician_id, appointment_time, status, predicted_wait_minutes, failed_at, failed_reason, created_at").
		Where("id = ?", appointmentID).
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
		appt               models.Appointment
		projectIDRaw       interface{}
		technicianIDRaw    interface{}
		appointmentTimeRaw interface{}
		failedAtRaw        interface{}
		createdAtRaw       interface{}
	)
	if err := rows.Scan(
		&appt.ID,
		&appt.CardID,
		&appt.MerchantID,
		&appt.UserID,
		&projectIDRaw,
		&technicianIDRaw,
		&appointmentTimeRaw,
		&appt.Status,
		&appt.PredictedWaitMinutes,
		&failedAtRaw,
		&appt.FailedReason,
		&createdAtRaw,
	); err != nil {
		return nil, err
	}
	if v, ok := schedulerValueToUint(projectIDRaw); ok {
		appt.ProjectID = &v
	}
	if v, ok := schedulerValueToUint(technicianIDRaw); ok {
		appt.TechnicianID = &v
	}
	if v, ok := parseSchedulerDBTimeValue(appointmentTimeRaw); ok {
		appt.AppointmentTime = v
	}
	if v, ok := parseSchedulerDBTimeValue(failedAtRaw); ok {
		appt.FailedAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(createdAtRaw); ok {
		appt.CreatedAt = v
	}
	return &appt, nil
}

func hardReservedAppointmentFinishAt(start time.Time, durationMinutes int) time.Time {
	if durationMinutes <= 0 {
		durationMinutes = 33
	}
	return start.Add(time.Duration(durationMinutes) * time.Minute)
}

func evaluateSchedulerBookingAvailability(tx *gorm.DB, merchant models.Merchant, technicianID uint, start time.Time, durationMinutes int, excludeAppointmentID uint) (string, int, error) {
	var appointments []models.Appointment
	if err := tx.Where("merchant_id = ? AND technician_id = ? AND status IN ? AND appointment_time IS NOT NULL",
		merchant.ID, technicianID, []string{"pending", "confirmed", "arrived", "finished", "completed"}).
		Where("id <> ?", excludeAppointmentID).
		Order("appointment_time asc").
		Find(&appointments).Error; err != nil {
		return "unavailable", 0, err
	}

	newFinish := hardReservedAppointmentFinishAt(start, durationMinutes)
	for _, appt := range appointments {
		if appt.AppointmentTime == nil {
			continue
		}
		existingStart := *appt.AppointmentTime
		existingFinish := hardReservedAppointmentFinishAt(existingStart, getSchedulerAppointmentDuration(tx, merchant.ID, appt.ProjectID))
		if start.Before(existingFinish) && existingStart.Before(newFinish) {
			return "unavailable", 0, nil
		}
	}
	return "safe", 0, nil
}

func getSchedulerAppointmentDuration(tx *gorm.DB, merchantID uint, projectID *uint) int {
	if projectID == nil || *projectID == 0 {
		return 33
	}
	var p models.MerchantProject
	if err := tx.Where("id = ? AND merchant_id = ?", *projectID, merchantID).First(&p).Error; err != nil {
		return 33
	}
	duration := p.Duration
	if duration <= 0 {
		duration = 30
	}
	gap := p.ServiceGapMinutes
	if gap < 0 {
		gap = 3
	}
	return duration + gap
}

func StartAppointmentScheduler() {
	go func() {
		ticker := time.NewTicker(appointmentSchedulerTickInterval)
		defer ticker.Stop()

		for range ticker.C {
			if config.DB == nil {
				continue
			}
			if err := runAppointmentAssignOnce(config.DB); err != nil {
				log.Printf("appointment scheduler error: %v", err)
			}
		}
	}()
}

func runAppointmentAssignOnce(db *gorm.DB) error {
	now := time.Now()
	cutoff := now.Add(-appointmentAssignDelay)

	var pendingIDs []uint
	err := db.Model(&models.Appointment{}).
		Where("technician_id IS NULL AND status = ? AND appointment_time IS NOT NULL AND created_at IS NOT NULL AND created_at <= ?", "pending", cutoff).
		Pluck("id", &pendingIDs).Error
	if err != nil {
		return err
	}

	// 已确认但仍未分配客服的预约不能只在创建后尝试一次。
	// 真实门店里，客服空闲状态会不断变化，所以 confirmed + technician_id IS NULL
	// 必须持续回补自动分配，直到成功分配、到店核销或超过宽限进入 no_show。
	var confirmedIDs []uint
	err = db.Model(&models.Appointment{}).
		Where("technician_id IS NULL AND status = ? AND appointment_time IS NOT NULL AND appointment_time >= ?", "confirmed", now.Add(-24*time.Hour)).
		Order("created_at asc").
		Limit(appointmentSchedulerBatchLimit).
		Pluck("id", &confirmedIDs).Error
	if err != nil {
		return err
	}

	for _, appointmentID := range pendingIDs {
		if err := tryAssignOneAppointment(db, appointmentID, now); err != nil {
			log.Printf("assign appointment %d error: %v", appointmentID, err)
		}
	}
	for _, appointmentID := range confirmedIDs {
		if err := tryAssignOneAppointment(db, appointmentID, now); err != nil {
			log.Printf("assign appointment %d error: %v", appointmentID, err)
		}
	}
	return runAppointmentNoShowOnce(db, now)
}

func runAppointmentNoShowOnce(db *gorm.DB, now time.Time) error {
	rows, err := db.Model(&models.Appointment{}).
		Select("id, merchant_id, appointment_time").
		Where("status = ? AND appointment_time IS NOT NULL", "confirmed").
		Order("appointment_time asc").
		Limit(appointmentSchedulerBatchLimit).
		Rows()
	if err != nil {
		return err
	}
	type appointmentDeadlineCheck struct {
		AppointmentID   uint
		MerchantID      uint
		AppointmentTime time.Time
	}
	checks := make([]appointmentDeadlineCheck, 0, appointmentSchedulerBatchLimit)

	for rows.Next() {
		var appointmentID uint
		var merchantID uint
		var appointmentTimeRaw interface{}
		if err := rows.Scan(&appointmentID, &merchantID, &appointmentTimeRaw); err != nil {
			return err
		}
		appointmentTime, ok := parseSchedulerDBTimeValue(appointmentTimeRaw)
		if !ok || appointmentTime == nil {
			continue
		}
		checks = append(checks, appointmentDeadlineCheck{
			AppointmentID:   appointmentID,
			MerchantID:      merchantID,
			AppointmentTime: *appointmentTime,
		})
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, check := range checks {
		var merchant models.Merchant
		if err := db.First(&merchant, check.MerchantID).Error; err != nil {
			log.Printf("load merchant for appointment %d error: %v", check.AppointmentID, err)
			continue
		}
		deadline := check.AppointmentTime.Add(time.Duration(merchantAppointmentGraceWindowMinutes(merchant)) * time.Minute)
		if now.Before(deadline) {
			continue
		}
		if err := db.Model(&models.Appointment{}).
			Where("id = ? AND status = ?", check.AppointmentID, "confirmed").
			Updates(map[string]interface{}{
				"status":     "no_show",
				"no_show_at": &now,
			}).Error; err != nil {
			log.Printf("mark appointment %d no_show error: %v", check.AppointmentID, err)
		}
	}
	return nil
}

func tryAssignOneAppointment(db *gorm.DB, appointmentID uint, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		aPtr, err := loadSchedulerAppointmentByID(tx, appointmentID)
		if err != nil {
			return err
		}
		if aPtr == nil {
			return gorm.ErrRecordNotFound
		}
		a := *aPtr
		if a.Status != "pending" && a.Status != "confirmed" {
			return nil
		}
		if a.TechnicianID != nil {
			return nil
		}
		if a.AppointmentTime == nil {
			return nil
		}

		var m models.Merchant
		if err := tx.First(&m, a.MerchantID).Error; err != nil {
			return err
		}
		// 未开启客服则不做自动分配
		if !m.SupportCustomerServiceMode {
			return nil
		}

		// 查找候选专业客服（排除运营客服；兼容历史 role_type 为空）
		var techs []models.Technician
		err = tx.
			Model(&models.Technician{}).
			Joins("JOIN service_roles sr ON sr.id = technicians.service_role_id").
			Where("technicians.merchant_id = ? AND technicians.is_active = ? AND (sr.role_type IS NULL OR sr.role_type = '' OR sr.role_type <> ?)", a.MerchantID, true, "operational").
			Order("technicians.id asc").
			Find(&techs).Error
		if err != nil {
			return err
		}
		if len(techs) == 0 {
			if a.Status == "confirmed" {
				return nil
			}
			failAt := now
			return tx.Model(&models.Appointment{}).
				Where("id = ? AND technician_id IS NULL AND status = ?", a.ID, "pending").
				Updates(map[string]interface{}{"status": "failed", "failed_at": &failAt, "failed_reason": "该时间段无可分配客服"}).Error
		}

		projectDurationCache := map[uint]int{}
		getOccupiedMinutes := func(ap models.Appointment) int {
			if ap.ProjectID == nil || *ap.ProjectID == 0 {
				return 33
			}
			pid := *ap.ProjectID
			if v, ok := projectDurationCache[pid]; ok {
				if v > 0 {
					return v
				}
				return 33
			}
			var p models.MerchantProject
			err := tx.Where("id = ? AND merchant_id = ?", pid, ap.MerchantID).First(&p).Error
			if err != nil {
				projectDurationCache[pid] = 0
				return 33
			}
			if p.Duration <= 0 {
				projectDurationCache[pid] = 0
				return 33
			}
			gap := p.ServiceGapMinutes
			if gap < 0 {
				gap = 3
			}
			projectDurationCache[pid] = p.Duration + gap
			return projectDurationCache[pid]
		}

		start := *a.AppointmentTime
		occupiedMinutes := getOccupiedMinutes(a)
		end := start.Add(time.Duration(occupiedMinutes) * time.Minute)

		overlaps := func(s1, e1, s2, e2 time.Time) bool {
			return s1.Before(e2) && s2.Before(e1)
		}

		bestTechID := uint(0)

		// 这里只回补历史遗留的空客服预约，规则与主链保持一致：未来预约严格 no-overlap。
		for _, t := range techs {
			var existing []models.Appointment
			err := tx.
				Where("merchant_id = ? AND technician_id = ? AND status IN ('pending','confirmed') AND appointment_time IS NOT NULL", a.MerchantID, t.ID).
				Order("appointment_time asc").
				Find(&existing).Error
			if err != nil {
				return err
			}

			conflict := false
			for _, e := range existing {
				if e.AppointmentTime == nil {
					continue
				}
				es := *e.AppointmentTime
				eMin := getOccupiedMinutes(e)
				ee := es.Add(time.Duration(eMin) * time.Minute)
				if overlaps(start, end, es, ee) {
					conflict = true
					break
				}
			}
			if conflict {
				continue
			}
			state, _, err := evaluateSchedulerBookingAvailability(tx, m, t.ID, start, occupiedMinutes, a.ID)
			if err != nil {
				return err
			}
			if state == "unavailable" {
				continue
			}
			if bestTechID == 0 || t.ID < bestTechID {
				bestTechID = t.ID
			}
		}

		if bestTechID > 0 {
			updates := map[string]interface{}{
				"technician_id":          bestTechID,
				"predicted_wait_minutes": 0,
				"failed_at":              nil,
				"failed_reason":          "",
			}
			if err := tx.Model(&models.Appointment{}).
				Where("id = ? AND technician_id IS NULL AND status = ?", a.ID, a.Status).
				Updates(updates).Error; err != nil {
				return err
			}
			return nil
		}

		// pending 仍沿用旧语义：创建 10 分钟后若还完全无客服可分配，则标记失败。
		// confirmed 不直接失败，因为它已经是商户确认过的预约，需要持续等待后续客服空闲时回补分配。
		if a.Status == "confirmed" {
			return nil
		}

		// 全部冲突，标记失败
		failAt := now
		if err := tx.Model(&models.Appointment{}).
			Where("id = ? AND technician_id IS NULL AND status = ?", a.ID, "pending").
			Updates(map[string]interface{}{"status": "failed", "failed_at": &failAt, "failed_reason": "该时间段无可分配客服"}).Error; err != nil {
			return err
		}
		return nil
	})
}
