package scheduler

import (
	"kabao/appointmentliability"
	"kabao/config"
	"kabao/models"
	"kabao/sessionflow"
	"log"
	"math"
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

func syncAppointmentLiabilitySnapshot(tx *gorm.DB, appointmentID uint, appointmentUpdates map[string]interface{}, settlementUpdates map[string]interface{}) error {
	if tx == nil || appointmentID == 0 {
		return nil
	}
	if len(appointmentUpdates) > 0 {
		if err := tx.Model(&models.Appointment{}).Where("id = ?", appointmentID).Updates(appointmentUpdates).Error; err != nil {
			return err
		}
	}
	if len(settlementUpdates) == 0 {
		return nil
	}
	var settlementID uint
	if err := tx.Model(&models.Appointment{}).Where("id = ?", appointmentID).Select("appointment_settlement_id").Scan(&settlementID).Error; err != nil {
		return err
	}
	if settlementID == 0 {
		return appointmentliability.SyncLeaveDisruptionLedger(tx, appointmentID, time.Now())
	}
	if err := tx.Model(&models.AppointmentSettlement{}).Where("id = ?", settlementID).Updates(settlementUpdates).Error; err != nil {
		return err
	}
	return appointmentliability.SyncLeaveDisruptionLedger(tx, appointmentID, time.Now())
}

func loadSchedulerAppointmentByID(tx *gorm.DB, appointmentID uint) (*models.Appointment, error) {
	if tx == nil || appointmentID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("appointments").
		Select("id, card_id, merchant_id, user_id, booking_root_id, project_id, technician_id, appointment_time, reserved_end_at, late_arrival_min_service_minutes, status, predicted_delay_minutes, arrived_at, actual_arrived_at, actual_start_at, usage_id, service_session_id, appointment_settlement_id, settlement_status_snapshot, resolution_note, failed_at, failed_reason, created_at").
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
		appt                       models.Appointment
		bookingRootIDRaw           interface{}
		projectIDRaw               interface{}
		technicianIDRaw            interface{}
		appointmentTimeRaw         interface{}
		reservedEndAtRaw           interface{}
		arrivedAtRaw               interface{}
		actualArrivedAtRaw         interface{}
		actualStartAtRaw           interface{}
		usageIDRaw                 interface{}
		serviceSessionIDRaw        interface{}
		appointmentSettlementIDRaw interface{}
		failedAtRaw                interface{}
		createdAtRaw               interface{}
	)
	if err := rows.Scan(
		&appt.ID,
		&appt.CardID,
		&appt.MerchantID,
		&appt.UserID,
		&bookingRootIDRaw,
		&projectIDRaw,
		&technicianIDRaw,
		&appointmentTimeRaw,
		&reservedEndAtRaw,
		&appt.LateArrivalMinServiceMinutes,
		&appt.Status,
		&appt.PredictedWaitMinutes,
		&arrivedAtRaw,
		&actualArrivedAtRaw,
		&actualStartAtRaw,
		&usageIDRaw,
		&serviceSessionIDRaw,
		&appointmentSettlementIDRaw,
		&appt.SettlementStatusSnapshot,
		&appt.ResolutionNote,
		&failedAtRaw,
		&appt.FailedReason,
		&createdAtRaw,
	); err != nil {
		return nil, err
	}
	if v, ok := schedulerValueToUint(bookingRootIDRaw); ok {
		appt.BookingRootID = &v
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
	if v, ok := parseSchedulerDBTimeValue(reservedEndAtRaw); ok {
		appt.ReservedEndAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(arrivedAtRaw); ok {
		appt.ArrivedAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(actualArrivedAtRaw); ok {
		appt.ActualArrivedAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(actualStartAtRaw); ok {
		appt.ActualStartAt = v
	}
	if v, ok := schedulerValueToUint(usageIDRaw); ok {
		appt.UsageID = &v
	}
	if v, ok := schedulerValueToUint(serviceSessionIDRaw); ok {
		appt.ServiceSessionID = &v
	}
	if v, ok := schedulerValueToUint(appointmentSettlementIDRaw); ok {
		appt.AppointmentSettlementID = &v
	}
	if v, ok := parseSchedulerDBTimeValue(failedAtRaw); ok {
		appt.FailedAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(createdAtRaw); ok {
		appt.CreatedAt = v
	}
	return &appt, nil
}

func appointmentNoShowDeadline(appt *models.Appointment, merchant models.Merchant) (time.Time, bool) {
	if appt == nil || appt.AppointmentTime == nil {
		return time.Time{}, false
	}
	if appt.ReservedEndAt != nil {
		minServiceMinutes := appt.LateArrivalMinServiceMinutes
		if minServiceMinutes <= 0 && appt.ReservedEndAt.After(*appt.AppointmentTime) {
			minServiceMinutes = int(appt.ReservedEndAt.Sub(*appt.AppointmentTime) / time.Minute / 2)
		}
		return appt.ReservedEndAt.Add(-time.Duration(minServiceMinutes) * time.Minute), true
	}
	graceMinutes := merchant.AppointmentGraceWindowMinutes
	if graceMinutes <= 0 {
		graceMinutes = 15
	}
	return appt.AppointmentTime.Add(time.Duration(graceMinutes) * time.Minute), true
}

func sameSchedulerCalendarDay(left, right time.Time) bool {
	return left.Year() == right.Year() && left.Month() == right.Month() && left.Day() == right.Day()
}

func schedulerAppointmentHasArrivalEvidence(appt *models.Appointment) bool {
	if appt == nil {
		return false
	}
	if appt.ActualArrivedAt != nil || appt.ArrivedAt != nil {
		return true
	}
	if appt.UsageID != nil && *appt.UsageID > 0 {
		return true
	}
	if appt.ServiceSessionID != nil && *appt.ServiceSessionID > 0 {
		return true
	}
	return false
}

func schedulerAppointmentIsCrossDayUnstarted(appt *models.Appointment, now time.Time) bool {
	if appt == nil || appt.AppointmentTime == nil {
		return false
	}
	if strings.TrimSpace(appt.Status) != "arrived" || appt.ActualStartAt != nil {
		return false
	}
	return !sameSchedulerCalendarDay(appt.AppointmentTime.In(time.Local), now.In(time.Local))
}

func appendSchedulerResolutionNote(left, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	return left + "\n" + right
}

func ensureSchedulerAppointmentSettlement(tx *gorm.DB, appt *models.Appointment, latestReason string) error {
	if tx == nil || appt == nil || appt.ID == 0 {
		return nil
	}
	if appt.AppointmentSettlementID != nil && *appt.AppointmentSettlementID > 0 {
		return nil
	}
	settlement := models.AppointmentSettlement{
		AppointmentID:            appt.ID,
		BookingRootID:            appt.BookingRootID,
		MerchantID:               appt.MerchantID,
		UserID:                   appt.UserID,
		CardID:                   appt.CardID,
		ProjectID:                appt.ProjectID,
		Status:                   "pending",
		AssetMode:                "deduct",
		SettlementStatusSnapshot: "pending",
		LatestReason:             strings.TrimSpace(latestReason),
	}
	if err := tx.Create(&settlement).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Appointment{}).Where("id = ?", appt.ID).Updates(map[string]interface{}{
		"appointment_settlement_id":  settlement.ID,
		"settlement_status_snapshot": settlement.SettlementStatusSnapshot,
	}).Error; err != nil {
		return err
	}
	appt.AppointmentSettlementID = &settlement.ID
	appt.SettlementStatusSnapshot = settlement.SettlementStatusSnapshot
	return nil
}

func closeCrossDayUnfinishedAppointment(tx *gorm.DB, s *models.ServiceSession, appt *models.Appointment, merchant *models.Merchant, now time.Time) error {
	if tx == nil || s == nil || appt == nil || merchant == nil {
		return nil
	}
	if err := ensureSchedulerAppointmentSettlement(tx, appt, "service_unclosed_cross_day"); err != nil {
		return err
	}

	usageID := s.InitialUsageID
	if usageID == 0 && appt.UsageID != nil {
		usageID = *appt.UsageID
	}
	if usageID > 0 {
		if err := sessionflow.FailUsageAndRefund(tx, usageID, merchant, now, false); err != nil {
			return err
		}
	}

	var actualArrivedAt *time.Time
	if appt.ActualArrivedAt != nil {
		tm := *appt.ActualArrivedAt
		actualArrivedAt = &tm
	} else if appt.ArrivedAt != nil {
		tm := *appt.ArrivedAt
		actualArrivedAt = &tm
	}

	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"room_selecting", "room_locked", "staff_selecting", "start_pending", "delay_pending"})).
		Updates(map[string]interface{}{
			"status":                              models.ApplyStatusPrefix(s.Status, "canceled"),
			"finished_at":                         &now,
			"technician_id":                       nil,
			"room_id":                             nil,
			"room_locked_at":                      nil,
			"room_select_deadline_at":             nil,
			"start_confirmed_at":                  nil,
			"scheduled_start_at":                  nil,
			"started_at":                          nil,
			"scheduled_finish_at":                 nil,
			"predicted_ready_at":                  nil,
			"predicted_appointment_delay_minutes": 0,
		}).Error; err != nil {
		return err
	}

	resolutionNote := appendSchedulerResolutionNote(appt.ResolutionNote, "系统自动结案：跨日未开始服务")
	appointmentUpdates := map[string]interface{}{
		"status":                             "failed",
		"failed_at":                          &now,
		"failed_reason":                      "service_unclosed_cross_day",
		"merchant_breach_pending":            false,
		"breach_decision_at":                 &now,
		"disruption_status":                  "closed",
		"disruption_reason":                  "service_unclosed_cross_day",
		"liability_level":                    "merchant",
		"salary_settlement_reference_status": "refund",
		"closed_reason":                      "auto_exception_closed",
		"closed_by_type":                     "system",
		"settlement_status_snapshot":         "pending",
		"resolution_note":                    resolutionNote,
	}
	if actualArrivedAt != nil {
		appointmentUpdates["actual_arrived_at"] = actualArrivedAt
	}
	settlementUpdates := map[string]interface{}{
		"status":                             "pending",
		"settlement_status_snapshot":         "pending",
		"merchant_breach_pending":            false,
		"breach_decision_at":                 &now,
		"liability_level":                    "merchant",
		"salary_settlement_reference_status": "refund",
		"latest_reason":                      "service_unclosed_cross_day",
	}
	return syncAppointmentLiabilitySnapshot(tx, appt.ID, appointmentUpdates, settlementUpdates)
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
		recordSchedulerTick(schedulerNameAppointment, time.Now())

		for range ticker.C {
			if config.DB == nil {
				continue
			}
			recordSchedulerTick(schedulerNameAppointment, time.Now())
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
	if err := runAppointmentNoShowOnce(db, now); err != nil {
		return err
	}
	if err := runMerchantBreachCompensation(db, now); err != nil {
		return err
	}
	if err := runAppointmentDelayLedgerSettlement(db, now); err != nil {
		return err
	}
	return runMultiServiceBookingSettlementOnce(db, now)
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
		appointment, err := loadSchedulerAppointmentByID(db, check.AppointmentID)
		if err != nil {
			log.Printf("load appointment %d before no_show failed: %v", check.AppointmentID, err)
			continue
		}
		if appointment == nil {
			continue
		}
		deadline, ok := appointmentNoShowDeadline(appointment, merchant)
		if !ok || now.Before(deadline) {
			continue
		}
		updates := map[string]interface{}{
			"status":     "no_show",
			"no_show_at": &now,
		}
		appointmentUpdates := map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &now,
			"disruption_status":                  "closed",
			"disruption_reason":                  "user_no_show",
			"liability_level":                    "user",
			"salary_settlement_reference_status": "no_pay",
		}
		settlementUpdates := map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &now,
			"liability_level":                    "user",
			"salary_settlement_reference_status": "no_pay",
			"latest_reason":                      "user_no_show",
		}
		if merchant.SupportCustomerServiceMode && appointment.TechnicianID == nil && !schedulerAppointmentHasArrivalEvidence(appointment) {
			updates["resolution_note"] = "系统巡检异常标记：客服模式预约超时失约前仍未分配客服"
			appointmentUpdates["disruption_reason"] = "appointment_state_inconsistent"
			appointmentUpdates["liability_level"] = "pending_merchant"
			appointmentUpdates["salary_settlement_reference_status"] = "pending"
			settlementUpdates["liability_level"] = "pending_merchant"
			settlementUpdates["salary_settlement_reference_status"] = "pending"
			settlementUpdates["latest_reason"] = "appointment_state_inconsistent"
		}
		if err := db.Model(&models.Appointment{}).
			Where("id = ? AND status = ?", check.AppointmentID, "confirmed").
			Updates(updates).Error; err != nil {
			log.Printf("mark appointment %d no_show error: %v", check.AppointmentID, err)
			continue
		}
		if err := syncAppointmentLiabilitySnapshot(db, check.AppointmentID, appointmentUpdates, settlementUpdates); err != nil {
			log.Printf("sync appointment %d no_show liability error: %v", check.AppointmentID, err)
		}
	}
	return nil
}

func runMerchantBreachCompensation(db *gorm.DB, now time.Time) error {
	rows, err := db.Table("appointments").
		Select("id").
		Where("merchant_breach_pending = ? AND breach_decision_at IS NOT NULL AND breach_decision_at <= ?", true, now).
		Order("breach_decision_at asc, id asc").
		Limit(appointmentSchedulerBatchLimit).
		Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	var appointmentIDs []uint
	for rows.Next() {
		var appointmentID uint
		if err := rows.Scan(&appointmentID); err != nil {
			return err
		}
		appointmentIDs = append(appointmentIDs, appointmentID)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, appointmentID := range appointmentIDs {
		if err := settleMerchantBreachAppointment(db, appointmentID, now); err != nil {
			log.Printf("settle merchant breach appointment %d error: %v", appointmentID, err)
		}
	}
	return nil
}

func settleMerchantBreachAppointment(db *gorm.DB, appointmentID uint, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		appt, err := loadSchedulerAppointmentCompensationTarget(tx, appointmentID)
		if err != nil {
			return err
		}
		if appt == nil || !appt.MerchantBreachPending {
			return nil
		}
		if appt.BreachDecisionAt == nil || appt.BreachDecisionAt.After(now) {
			return nil
		}

		resultReason := ""
		compensationType := ""
		compensationValue := 0
		liabilityLevel := "none"
		salaryStatus := "normal"

		switch {
		case appt.ActualArrivedAt != nil:
			resultReason = "merchant_breach"
			compensationType = "merchant_breach"
			compensationValue = 2
			liabilityLevel = "merchant"
			salaryStatus = "refund"
		default:
			formalAction, err := hasMerchantFormalMitigationRecord(tx, appt.ID)
			if err != nil {
				return err
			}
			if formalAction {
				resultReason = "merchant_failure_offset"
				compensationType = "merchant_failure_offset"
				compensationValue = 1
				liabilityLevel = "merchant"
				salaryStatus = "refund"
			}
		}

		if compensationType != "" && compensationValue > 0 {
			if err := ensureAppointmentCompensation(tx, *appt, compensationType, compensationValue, now); err != nil {
				return err
			}
		}
		if resultReason == "" {
			resultReason = "risk_released"
		}
		return syncAppointmentLiabilitySnapshot(tx, appt.ID, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &now,
			"disruption_status":                  "closed",
			"disruption_reason":                  resultReason,
			"liability_level":                    liabilityLevel,
			"salary_settlement_reference_status": salaryStatus,
		}, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &now,
			"liability_level":                    liabilityLevel,
			"salary_settlement_reference_status": salaryStatus,
			"latest_reason":                      resultReason,
		})
	})
}

func ensureAppointmentCompensation(tx *gorm.DB, appt models.Appointment, sourceType string, value int, now time.Time) error {
	if tx == nil || appt.ID == 0 || value <= 0 {
		return nil
	}
	var count int64
	if err := tx.Model(&models.AppointmentCompensation{}).
		Where("appointment_id = ? AND source_type = ? AND status IN ?", appt.ID, sourceType, []string{"pending", "applied"}).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if err := tx.Model(&models.Card{}).Where("id = ?", appt.CardID).Updates(map[string]interface{}{
		"total_times":  gorm.Expr("total_times + ?", value),
		"remain_times": gorm.Expr("remain_times + ?", value),
	}).Error; err != nil {
		return err
	}
	comp := models.AppointmentCompensation{
		AppointmentID: appt.ID,
		MerchantID:    appt.MerchantID,
		UserID:        appt.UserID,
		CardID:        appt.CardID,
		Type:          "extra_times",
		Value:         value,
		Status:        "applied",
		Reason:        sourceType,
		Remark:        "系统自动补偿",
		SourceType:    sourceType,
		CreatedByType: "system",
		AppliedAt:     &now,
	}
	return tx.Create(&comp).Error
}

func hasMerchantFormalMitigationRecord(tx *gorm.DB, appointmentID uint) (bool, error) {
	if tx == nil || appointmentID == 0 {
		return false, nil
	}
	var count int64
	if err := tx.Table("appointment_cancel_requests").
		Where("appointment_id = ? AND proposed_by_type IN ? AND status IN ?", appointmentID, []string{"merchant", "staff"}, []string{"pending_user", "rejected", "accepted"}).
		Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	if err := tx.Table("appointment_reschedule_requests").
		Where("appointment_id = ? AND proposed_by_type IN ? AND status IN ?", appointmentID, []string{"merchant", "staff"}, []string{"pending_user", "rejected", "accepted"}).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func loadSchedulerAppointmentCompensationTarget(tx *gorm.DB, appointmentID uint) (*models.Appointment, error) {
	if tx == nil || appointmentID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("appointments").
		Select("id, merchant_id, user_id, card_id, status, merchant_breach_pending, breach_decision_at, actual_arrived_at").
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
		breachDecisionRaw  interface{}
		actualArrivedAtRaw interface{}
	)
	if err := rows.Scan(&appt.ID, &appt.MerchantID, &appt.UserID, &appt.CardID, &appt.Status, &appt.MerchantBreachPending, &breachDecisionRaw, &actualArrivedAtRaw); err != nil {
		return nil, err
	}
	if v, ok := parseSchedulerDBTimeValue(breachDecisionRaw); ok {
		appt.BreachDecisionAt = v
	}
	if v, ok := parseSchedulerDBTimeValue(actualArrivedAtRaw); ok {
		appt.ActualArrivedAt = v
	}
	return &appt, nil
}

func runAppointmentDelayLedgerSettlement(db *gorm.DB, now time.Time) error {
	rows, err := db.Table("appointment_delay_ledgers").
		Select("merchant_id, card_id, project_id").
		Where("ledger_status = ? AND redeem_status = ?", "recorded", "pending").
		Group("merchant_id, card_id, project_id").
		Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	type bucketKey struct {
		MerchantID uint
		CardID     uint
		ProjectID  uint
	}
	var keys []bucketKey
	for rows.Next() {
		var merchantID, cardID, projectID uint
		if err := rows.Scan(&merchantID, &cardID, &projectID); err != nil {
			return err
		}
		keys = append(keys, bucketKey{MerchantID: merchantID, CardID: cardID, ProjectID: projectID})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, key := range keys {
		if err := settleDelayLedgerBucket(db, key.MerchantID, key.CardID, key.ProjectID, now); err != nil {
			log.Printf("settle delay ledger bucket merchant=%d card=%d project=%d error: %v", key.MerchantID, key.CardID, key.ProjectID, err)
		}
	}
	return nil
}

func settleDelayLedgerBucket(db *gorm.DB, merchantID, cardID, projectID uint, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var project models.MerchantProject
		if err := tx.Where("id = ? AND merchant_id = ?", projectID, merchantID).First(&project).Error; err != nil {
			return err
		}
		mode := strings.TrimSpace(project.DelayCompensationMode)
		if mode == "" {
			mode = "minutes_bucket"
		}
		var card models.Card
		if err := tx.Where("id = ? AND merchant_id = ?", cardID, merchantID).First(&card).Error; err != nil {
			return err
		}
		threshold := project.Duration
		if threshold <= 0 {
			threshold = 1
		}
		if project.DelayRedeemThresholdPercent > 0 {
			threshold = int(math.Ceil(float64(threshold*project.DelayRedeemThresholdPercent) / 100.0))
		}
		if threshold <= 0 {
			threshold = 1
		}

		var ledgers []models.AppointmentDelayLedger
		if err := tx.Where("merchant_id = ? AND card_id = ? AND project_id = ? AND ledger_status = ? AND redeem_status = ?", merchantID, cardID, projectID, "recorded", "pending").Order("id ASC").Find(&ledgers).Error; err != nil {
			return err
		}
		if len(ledgers) == 0 {
			return nil
		}

		total := 0
		for _, ledger := range ledgers {
			switch mode {
			case "amount_bucket":
				total += ledger.DelayCompensationValue
			default:
				total += ledger.CreditedMinutes
			}
		}
		if total < threshold {
			return nil
		}

		redeemUnits := total / threshold
		if redeemUnits <= 0 {
			return nil
		}
		redeemValue := redeemUnits
		switch mode {
		case "amount_bucket", "fixed_unit":
			if project.DelayFixedUnitValue > 0 {
				redeemValue = redeemUnits * project.DelayFixedUnitValue
			}
		}
		lastLedger := ledgers[len(ledgers)-1]
		var count int64
		if err := tx.Model(&models.AppointmentCompensation{}).
			Where("appointment_id = ? AND source_type = ? AND source_id = ? AND status IN ?", lastLedger.AppointmentID, "delay_bucket_redeem", lastLedger.ID, []string{"pending", "applied"}).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			compType := "extra_times"
			compReason := "delay_bucket_redeem"
			cardUpdates := map[string]interface{}{}
			switch normalizeCardType(card.CardType) {
			case "balance":
				compType = "manual_adjustment"
				cardUpdates["recharge_amount"] = gorm.Expr("recharge_amount + ?", redeemValue)
			default:
				cardUpdates["total_times"] = gorm.Expr("total_times + ?", redeemValue)
				cardUpdates["remain_times"] = gorm.Expr("remain_times + ?", redeemValue)
			}
			if err := tx.Model(&models.Card{}).Where("id = ?", cardID).Updates(cardUpdates).Error; err != nil {
				return err
			}
			comp := models.AppointmentCompensation{
				AppointmentID: lastLedger.AppointmentID,
				MerchantID:    merchantID,
				UserID:        lastLedger.UserID,
				CardID:        cardID,
				Type:          compType,
				Value:         redeemValue,
				Status:        "applied",
				Reason:        compReason,
				Remark:        "拖堂补偿自动兑现",
				SourceType:    "delay_bucket_redeem",
				SourceID:      &lastLedger.ID,
				CreatedByType: "system",
				AppliedAt:     &now,
			}
			if err := tx.Create(&comp).Error; err != nil {
				return err
			}
		}

		remaining := redeemUnits * threshold
		for _, ledger := range ledgers {
			bucketValue := ledger.CreditedMinutes
			bucketField := "credited_minutes"
			if mode == "amount_bucket" {
				bucketValue = ledger.DelayCompensationValue
				bucketField = "delay_compensation_value"
			}
			consume := 0
			if remaining > 0 {
				consume = minInt(remaining, bucketValue)
			}
			updates := map[string]interface{}{}
			switch {
			case consume <= 0:
			case consume >= bucketValue:
				updates["redeem_status"] = "redeemed"
				updates["ledger_status"] = "redeemed"
			default:
				updates[bucketField] = bucketValue - consume
				updates["redeem_status"] = "pending"
				updates["ledger_status"] = "recorded"
			}
			remaining -= consume
			if len(updates) == 0 {
				continue
			}
			if err := tx.Model(&models.AppointmentDelayLedger{}).Where("id = ?", ledger.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func normalizeCardType(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(s, "balance"), strings.Contains(raw, "充值"), strings.Contains(raw, "额度"):
		return "balance"
	case strings.Contains(s, "lesson"), strings.Contains(raw, "课时"):
		return "lesson"
	default:
		return "times"
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
				"technician_id":           bestTechID,
				"predicted_delay_minutes": 0,
				"failed_at":               nil,
				"failed_reason":           "",
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
