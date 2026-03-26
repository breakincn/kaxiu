package handlers

import (
	"database/sql"
	"fmt"
	"kabao/models"
	"math"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type appointmentAvailabilityState string

const (
	appointmentAvailabilitySafe        appointmentAvailabilityState = "safe"
	appointmentAvailabilityUnavailable appointmentAvailabilityState = "unavailable"
)

const (
	appointmentDisplayWaitStateNone             = ""
	appointmentDisplayWaitStateRiskPending      = "risk_pending"
	appointmentDisplayWaitStateActiveWaiting    = "active_waiting"
	appointmentDisplayWaitStateCrossDayUnclosed = "cross_day_unfinished"
	appointmentDisplayWaitStateArrivedBound     = "arrived_bound"
)

type appointmentAvailability struct {
	State                  appointmentAvailabilityState `json:"availability_state"`
	PredictedWaitMinutes   int                          `json:"predicted_delay_minutes"`
	Reason                 string                       `json:"availability_reason"`
	NextAppointmentID      *uint                        `json:"next_appointment_id,omitempty"`
	AlternativeWaitMinutes int                          `json:"alternative_wait_minutes,omitempty"`
	DecisionMode           string                       `json:"decision_mode,omitempty"`
}

type appointmentRescheduleRuleMode string

const (
	appointmentRescheduleForbidden       appointmentRescheduleRuleMode = "forbidden"
	appointmentRescheduleTomorrowOnly    appointmentRescheduleRuleMode = "tomorrow_only"
	appointmentRescheduleTodayOrTomorrow appointmentRescheduleRuleMode = "today_or_tomorrow"
)

type appointmentRescheduleEligibility struct {
	Allowed            bool                          `json:"allowed"`
	Reason             string                        `json:"reason"`
	RuleMode           appointmentRescheduleRuleMode `json:"rule_mode"`
	AllowedDates       []string                      `json:"allowed_dates"`
	DefaultDate        string                        `json:"default_date"`
	MinutesUntilAnchor int                           `json:"minutes_until_anchor"`
	ProtectionConsumed bool                          `json:"protection_consumed"`
}

func normalizeAppointmentStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "finished":
		// 兼容历史数据：历史 finished 在新语义下视为 completed，只读路径统一归一化。
		return "completed"
	default:
		return strings.TrimSpace(status)
	}
}

func sameCalendarDay(left, right time.Time) bool {
	return left.Year() == right.Year() && left.Month() == right.Month() && left.Day() == right.Day()
}

func appointmentIsCrossDayUnfinished(appt *models.Appointment, now time.Time) bool {
	if appt == nil {
		return false
	}
	if normalizeAppointmentStatus(appt.Status) != "arrived" || appt.ActualStartAt != nil || appt.AppointmentTime == nil {
		return false
	}
	loc := appointmentLocation()
	return !sameCalendarDay(appt.AppointmentTime.In(loc), now.In(loc))
}

func appointmentHasArrivalEvidence(appt *models.Appointment) bool {
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

func decorateAppointmentDisplay(appt *models.Appointment, now time.Time) {
	if appt == nil {
		return
	}
	appt.DisplayWaitState = appointmentDisplayWaitStateNone
	appt.DisplayWaitMessage = ""
	appt.CurrentEstimatedWaitMinutes = 0

	status := normalizeAppointmentStatus(appt.Status)
	if status == "confirmed" && appt.PredictedWaitMinutes > 0 {
		appt.DisplayWaitState = appointmentDisplayWaitStateRiskPending
		appt.DisplayWaitMessage = fmt.Sprintf("该预约属于风险可约，若前序服务压单，预计到店等待 %d 分钟", appt.PredictedWaitMinutes)
		appt.CurrentEstimatedWaitMinutes = appt.PredictedWaitMinutes
		return
	}
	if status != "arrived" {
		return
	}

	if appointmentIsCrossDayUnfinished(appt, now) {
		appt.DisplayWaitState = appointmentDisplayWaitStateCrossDayUnclosed
		if appointmentHasArrivalEvidence(appt) {
			if strings.TrimSpace(appt.LiabilityLevel) == "" {
				appt.LiabilityLevel = "merchant"
			}
			if strings.TrimSpace(appt.DisruptionReason) == "" {
				appt.DisruptionReason = "service_unclosed_cross_day"
			}
			appt.DisplayWaitMessage = "该预约已过原预约日，客户已有到店记录，但服务未开始且系统未自动结案，当前按商户履约异常处理"
			return
		}
		if strings.TrimSpace(appt.LiabilityLevel) == "" {
			appt.LiabilityLevel = "pending_merchant"
		}
		if strings.TrimSpace(appt.DisruptionReason) == "" {
			appt.DisruptionReason = "appointment_state_inconsistent"
		}
		appt.DisplayWaitMessage = "该预约已过原预约日，但缺少有效到店或开单记录，当前状态与履约事实不一致，请核对现场记录后异常结案"
		return
	}

	if appt.PredictedWaitMinutes > 0 {
		appt.DisplayWaitState = appointmentDisplayWaitStateActiveWaiting
		appt.DisplayWaitMessage = fmt.Sprintf("原预约客服暂未释放，当前预计等待 %d 分钟，系统已记录本次预约延迟并按履约异常持续跟进", appt.PredictedWaitMinutes)
		appt.CurrentEstimatedWaitMinutes = appt.PredictedWaitMinutes
		return
	}

	if appt.ServiceSessionID != nil && *appt.ServiceSessionID > 0 {
		appt.DisplayWaitState = appointmentDisplayWaitStateArrivedBound
		appt.DisplayWaitMessage = "客户已到店，服务会话已绑定到本次预约"
	}
}

func merchantAppointmentReserveBufferMinutes(m *models.Merchant) int {
	if m != nil && m.AppointmentReserveBufferMinutes > 0 {
		return m.AppointmentReserveBufferMinutes
	}
	return 10
}

func merchantAppointmentGraceWindowMinutes(m *models.Merchant) int {
	if m != nil && m.AppointmentGraceWindowMinutes > 0 {
		return m.AppointmentGraceWindowMinutes
	}
	return 15
}

func merchantAppointmentPredictionBufferMinutes(m *models.Merchant) int {
	if m != nil && m.AppointmentPredictionBufferMinute > 0 {
		return m.AppointmentPredictionBufferMinute
	}
	return 5
}

func merchantAppointmentSlotGranularityMinutes(m *models.Merchant) int {
	if m != nil && m.AppointmentSlotGranularityMinutes > 0 {
		return normalizeAppointmentSlotGranularityMinutes(m.AppointmentSlotGranularityMinutes)
	}
	return 15
}

func merchantAppointmentRescheduleSameOrNextDayThresholdMinutes(m *models.Merchant) int {
	if m != nil && m.AppointmentRescheduleSameOrNextDayThresholdMinutes > 0 {
		return m.AppointmentRescheduleSameOrNextDayThresholdMinutes
	}
	return 180
}

func merchantAppointmentRescheduleNextDayOnlyThresholdMinutes(m *models.Merchant) int {
	if m != nil && m.AppointmentRescheduleNextDayOnlyThresholdMinutes > 0 {
		return m.AppointmentRescheduleNextDayOnlyThresholdMinutes
	}
	return 90
}

func appointmentProtectedStatuses() []string {
	return []string{"pending", "confirmed", "arrived", "finished", "completed"}
}

func appointmentHardFinishAt(start time.Time, occupiedMinutes int) time.Time {
	if occupiedMinutes <= 0 {
		occupiedMinutes = projectBookingOccupiedMinutes(30, 3)
	}
	return start.Add(time.Duration(occupiedMinutes) * time.Minute)
}

func appointmentPredictedFinishAt(start time.Time, occupiedMinutes int, merchant *models.Merchant) time.Time {
	// 预测完成时间只用于现场等待判断与风险提示，不再参与未来预约硬阻塞。
	return appointmentHardFinishAt(start, occupiedMinutes).Add(time.Duration(merchantAppointmentPredictionBufferMinutes(merchant)) * time.Minute)
}

func appointmentDetectableWindow(appt models.Appointment, merchant *models.Merchant) (time.Time, time.Time, bool) {
	if appt.AppointmentTime == nil {
		return time.Time{}, time.Time{}, false
	}
	start := appt.AppointmentTime.Add(-time.Duration(merchantAppointmentReserveBufferMinutes(merchant)) * time.Minute)
	end := appt.AppointmentTime.Add(time.Duration(merchantAppointmentGraceWindowMinutes(merchant)) * time.Minute)
	return start, end, true
}

func canArriveForAppointment(appt models.Appointment, merchant *models.Merchant, now time.Time) bool {
	start, end, ok := appointmentDetectableWindow(appt, merchant)
	if !ok {
		return false
	}
	status := normalizeAppointmentStatus(appt.Status)
	if status != "confirmed" && status != "arrived" {
		return false
	}
	return !now.Before(start) && !now.After(end)
}

func appointmentActiveConflictStatuses() []string {
	return models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"})
}

func getProjectDurationForAppointment(tx *gorm.DB, merchantID uint, projectID *uint) int {
	if projectID == nil || *projectID == 0 || tx == nil {
		return 30
	}
	var p models.MerchantProject
	if err := tx.Where("id = ? AND merchant_id = ?", *projectID, merchantID).First(&p).Error; err != nil {
		return 30
	}
	if p.Duration <= 0 {
		return 30
	}
	return p.Duration
}

func getProjectOccupiedMinutesForAppointment(tx *gorm.DB, merchantID uint, projectID *uint) int {
	durationMinutes, gapMinutes := resolveProjectBookingConfig(tx, merchantID, projectID, 30, 3)
	return projectBookingOccupiedMinutes(durationMinutes, gapMinutes)
}

func appointmentSelectColumns() string {
	return "id, card_id, merchant_id, user_id, booking_root_id, project_id, technician_id, appointment_time, reserved_start_at, reserved_end_at, occupied_end_at, cancel_deadline_at, booking_close_deadline_at, late_arrival_min_service_minutes, appointment_settlement_id, settlement_status_snapshot, status, confirmed_at, arrived_at, actual_arrived_at, actual_start_at, completed_at, no_show_at, service_session_id, usage_id, predicted_delay_minutes, merchant_breach_pending, breach_decision_at, disruption_status, disruption_reason, merchant_cancel_reason, user_rebuttal_note, liability_level, salary_settlement_reference_status, resolution_note, closed_reason, closed_by_type, closed_by_id, reschedule_reason, replaced_by_appointment_id, replaces_appointment_id, canceled_at, failed_at, failed_reason, created_at"
}

func technicianSchedulePublishingSelectColumns() string {
	return "id, merchant_id, technician_id, publish_date, start_at, end_at, status, published_at, created_at"
}

func protectedRepairSlotSelectColumns() string {
	return "id, merchant_id, appointment_id, technician_id, publish_date, start_at, end_at, status, source_type, created_at"
}

func scanAppointmentRow(rows *sql.Rows) (models.Appointment, error) {
	var (
		appt                       models.Appointment
		bookingRootIDRaw           interface{}
		projectIDRaw               interface{}
		technicianIDRaw            interface{}
		appointmentTimeRaw         interface{}
		reservedStartAtRaw         interface{}
		reservedEndAtRaw           interface{}
		occupiedEndAtRaw           interface{}
		cancelDeadlineAtRaw        interface{}
		bookingCloseDeadlineAtRaw  interface{}
		appointmentSettlementIDRaw interface{}
		confirmedAtRaw             interface{}
		arrivedAtRaw               interface{}
		actualArrivedAtRaw         interface{}
		actualStartAtRaw           interface{}
		completedAtRaw             interface{}
		noShowAtRaw                interface{}
		serviceSessionIDRaw        interface{}
		usageIDRaw                 interface{}
		breachDecisionAtRaw        interface{}
		closedByIDRaw              interface{}
		replacedByAppointmentIDRaw interface{}
		replacesAppointmentIDRaw   interface{}
		canceledAtRaw              interface{}
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
		&reservedStartAtRaw,
		&reservedEndAtRaw,
		&occupiedEndAtRaw,
		&cancelDeadlineAtRaw,
		&bookingCloseDeadlineAtRaw,
		&appt.LateArrivalMinServiceMinutes,
		&appointmentSettlementIDRaw,
		&appt.SettlementStatusSnapshot,
		&appt.Status,
		&confirmedAtRaw,
		&arrivedAtRaw,
		&actualArrivedAtRaw,
		&actualStartAtRaw,
		&completedAtRaw,
		&noShowAtRaw,
		&serviceSessionIDRaw,
		&usageIDRaw,
		&appt.PredictedWaitMinutes,
		&appt.MerchantBreachPending,
		&breachDecisionAtRaw,
		&appt.DisruptionStatus,
		&appt.DisruptionReason,
		&appt.MerchantCancelReason,
		&appt.UserRebuttalNote,
		&appt.LiabilityLevel,
		&appt.SalarySettlementReferenceStatus,
		&appt.ResolutionNote,
		&appt.ClosedReason,
		&appt.ClosedByType,
		&closedByIDRaw,
		&appt.RescheduleReason,
		&replacedByAppointmentIDRaw,
		&replacesAppointmentIDRaw,
		&canceledAtRaw,
		&failedAtRaw,
		&appt.FailedReason,
		&createdAtRaw,
	); err != nil {
		return appt, err
	}
	if v, ok := gormValueToUint(bookingRootIDRaw); ok {
		appt.BookingRootID = &v
	}
	if v, ok := gormValueToUint(projectIDRaw); ok {
		appt.ProjectID = &v
	}
	if v, ok := gormValueToUint(technicianIDRaw); ok {
		appt.TechnicianID = &v
	}
	if v, ok := parseDBTimeValue(appointmentTimeRaw); ok {
		appt.AppointmentTime = &v
	}
	if v, ok := parseDBTimeValue(reservedStartAtRaw); ok {
		appt.ReservedStartAt = &v
	}
	if v, ok := parseDBTimeValue(reservedEndAtRaw); ok {
		appt.ReservedEndAt = &v
	}
	if v, ok := parseDBTimeValue(occupiedEndAtRaw); ok {
		appt.OccupiedEndAt = &v
	}
	if v, ok := parseDBTimeValue(cancelDeadlineAtRaw); ok {
		appt.CancelDeadlineAt = &v
	}
	if v, ok := parseDBTimeValue(bookingCloseDeadlineAtRaw); ok {
		appt.BookingCloseDeadlineAt = &v
	}
	if v, ok := gormValueToUint(appointmentSettlementIDRaw); ok {
		appt.AppointmentSettlementID = &v
	}
	if v, ok := parseDBTimeValue(confirmedAtRaw); ok {
		appt.ConfirmedAt = &v
	}
	if v, ok := parseDBTimeValue(arrivedAtRaw); ok {
		appt.ArrivedAt = &v
	}
	if v, ok := parseDBTimeValue(actualArrivedAtRaw); ok {
		appt.ActualArrivedAt = &v
	}
	if v, ok := parseDBTimeValue(actualStartAtRaw); ok {
		appt.ActualStartAt = &v
	}
	if v, ok := parseDBTimeValue(completedAtRaw); ok {
		appt.CompletedAt = &v
	}
	if v, ok := parseDBTimeValue(noShowAtRaw); ok {
		appt.NoShowAt = &v
	}
	if v, ok := gormValueToUint(serviceSessionIDRaw); ok {
		appt.ServiceSessionID = &v
	}
	if v, ok := gormValueToUint(usageIDRaw); ok {
		appt.UsageID = &v
	}
	if v, ok := parseDBTimeValue(breachDecisionAtRaw); ok {
		appt.BreachDecisionAt = &v
	}
	if v, ok := gormValueToUint(closedByIDRaw); ok {
		appt.ClosedByID = &v
	}
	if v, ok := gormValueToUint(replacedByAppointmentIDRaw); ok {
		appt.ReplacedByAppointmentID = &v
	}
	if v, ok := gormValueToUint(replacesAppointmentIDRaw); ok {
		appt.ReplacesAppointmentID = &v
	}
	if v, ok := parseDBTimeValue(canceledAtRaw); ok {
		appt.CanceledAt = &v
	}
	if v, ok := parseDBTimeValue(failedAtRaw); ok {
		appt.FailedAt = &v
	}
	if v, ok := parseDBTimeValue(createdAtRaw); ok {
		appt.CreatedAt = &v
	}
	return appt, nil
}

func scanTechnicianSchedulePublishingRow(rows *sql.Rows) (models.TechnicianSchedulePublishing, error) {
	var (
		row            models.TechnicianSchedulePublishing
		technicianRaw  interface{}
		publishDateRaw interface{}
		startAtRaw     interface{}
		endAtRaw       interface{}
		publishedAtRaw interface{}
		createdAtRaw   interface{}
	)
	if err := rows.Scan(&row.ID, &row.MerchantID, &technicianRaw, &publishDateRaw, &startAtRaw, &endAtRaw, &row.Status, &publishedAtRaw, &createdAtRaw); err != nil {
		return row, err
	}
	if v, ok := gormValueToUint(technicianRaw); ok {
		row.TechnicianID = &v
	}
	if v, ok := parseDBTimeValue(publishDateRaw); ok {
		row.PublishDate = &v
	}
	if v, ok := parseDBTimeValue(startAtRaw); ok {
		row.StartAt = &v
	}
	if v, ok := parseDBTimeValue(endAtRaw); ok {
		row.EndAt = &v
	}
	if v, ok := parseDBTimeValue(publishedAtRaw); ok {
		row.PublishedAt = &v
	}
	if v, ok := parseDBTimeValue(createdAtRaw); ok {
		row.CreatedAt = &v
	}
	return row, nil
}

func scanProtectedRepairSlotRow(rows *sql.Rows) (models.ProtectedRepairSlot, error) {
	var (
		row            models.ProtectedRepairSlot
		technicianRaw  interface{}
		publishDateRaw interface{}
		startAtRaw     interface{}
		endAtRaw       interface{}
		createdAtRaw   interface{}
	)
	if err := rows.Scan(&row.ID, &row.MerchantID, &row.AppointmentID, &technicianRaw, &publishDateRaw, &startAtRaw, &endAtRaw, &row.Status, &row.SourceType, &createdAtRaw); err != nil {
		return row, err
	}
	if v, ok := gormValueToUint(technicianRaw); ok {
		row.TechnicianID = &v
	}
	if v, ok := parseDBTimeValue(publishDateRaw); ok {
		row.PublishDate = &v
	}
	if v, ok := parseDBTimeValue(startAtRaw); ok {
		row.StartAt = &v
	}
	if v, ok := parseDBTimeValue(endAtRaw); ok {
		row.EndAt = &v
	}
	if v, ok := parseDBTimeValue(createdAtRaw); ok {
		row.CreatedAt = &v
	}
	return row, nil
}

func loadProtectedAppointmentsForTechnician(tx *gorm.DB, merchantID uint, technicianID uint, start, end time.Time, excludeAppointmentID uint) ([]models.Appointment, error) {
	if tx == nil || merchantID == 0 || technicianID == 0 {
		return nil, nil
	}
	q := tx.Table("appointments").
		Select(appointmentSelectColumns()).
		Where("merchant_id = ? AND technician_id = ? AND appointment_time IS NOT NULL AND appointment_time >= ? AND appointment_time <= ?",
			merchantID, technicianID, start, end).
		Where("status IN ?", appointmentProtectedStatuses()).
		Order("appointment_time asc")
	if excludeAppointmentID > 0 {
		q = q.Where("id <> ?", excludeAppointmentID)
	}
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]models.Appointment, 0)
	for rows.Next() {
		appt, err := scanAppointmentRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, appt)
	}
	return list, nil
}

func evaluateBookingTechnicianAvailability(tx *gorm.DB, merchant models.Merchant, technicianID uint, appointmentStart time.Time, projectOccupiedMinutes int, excludeAppointmentID uint) (appointmentAvailability, error) {
	out := appointmentAvailability{State: appointmentAvailabilitySafe}
	if technicianID == 0 {
		return out, nil
	}

	windowStart := appointmentStart.Add(-6 * time.Hour)
	windowEnd := appointmentStart.Add(6 * time.Hour)
	existing, err := loadProtectedAppointmentsForTechnician(tx, merchant.ID, technicianID, windowStart, windowEnd, excludeAppointmentID)
	if err != nil {
		return out, err
	}

	newFinish := appointmentHardFinishAt(appointmentStart, projectOccupiedMinutes)
	for _, appt := range existing {
		if appt.AppointmentTime == nil {
			continue
		}
		existingStart := *appt.AppointmentTime
		existingFinish := appointmentHardFinishAt(existingStart, getAppointmentOccupiedMinutes(merchant.ID, appt))
		if appointmentStart.Before(existingFinish) && existingStart.Before(newFinish) {
			reason := "该客服该时段已被其他预约占用"
			if !existingStart.Before(appointmentStart) {
				reason = "当前预约会与后续预约重叠"
			}
			return appointmentAvailability{
				State:             appointmentAvailabilityUnavailable,
				Reason:            reason,
				NextAppointmentID: &appt.ID,
			}, nil
		}
	}

	return out, nil
}

func earliestAlternativeTechnicianWaitMinutes(tx *gorm.DB, merchant models.Merchant, excludeTechnicianID uint, now time.Time) (int, bool, error) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var techs []models.TechnicianAttendance
	if err := tx.Preload("Technician").Preload("Technician.ServiceRole").
		Joins("JOIN technicians t ON t.id = technician_attendances.technician_id").
		Joins("JOIN service_roles sr ON sr.id = t.service_role_id").
		Where("technician_attendances.merchant_id = ? AND technician_attendances.checked_in_at >= ? AND technician_attendances.checked_out_at IS NULL", merchant.ID, start).
		Where("technician_attendances.technician_id <> ?", excludeTechnicianID).
		Where("t.is_active = ?", true).
		Where("sr.role_type = ? AND sr.`key` NOT IN ('store_manager','front_desk')", "professional").
		Find(&techs).Error; err != nil {
		return 0, false, err
	}

	found := false
	best := 0
	for _, att := range techs {
		waitMinutes := 0
		if att.Status == "idle" {
			if !found || waitMinutes < best {
				found = true
				best = waitMinutes
			}
			continue
		}
		sess, err := loadEarliestTechnicianConflictSession(tx, merchant.ID, att.TechnicianID, appointmentActiveConflictStatuses())
		if err != nil {
			return 0, false, err
		}
		if sess == nil {
			continue
		}
		var readyAt time.Time
		switch {
		case sess.PredictedReadyAt != nil:
			readyAt = *sess.PredictedReadyAt
		case sess.ScheduledFinishAt != nil:
			readyAt = sess.ScheduledFinishAt.Add(time.Duration(merchantAppointmentPredictionBufferMinutes(&merchant)) * time.Minute)
		default:
			continue
		}
		waitMinutes = int(math.Ceil(readyAt.Sub(now).Minutes()))
		if waitMinutes < 0 {
			waitMinutes = 0
		}
		if !found || waitMinutes < best {
			found = true
			best = waitMinutes
		}
	}
	return best, found, nil
}

func evaluateWalkInTechnicianAvailability(tx *gorm.DB, merchant models.Merchant, technicianID uint, walkInStart time.Time, projectOccupiedMinutes int) (appointmentAvailability, error) {
	out := appointmentAvailability{State: appointmentAvailabilitySafe}
	if tx == nil || technicianID == 0 {
		return out, nil
	}

	list, err := loadProtectedAppointmentsForTechnician(tx, merchant.ID, technicianID, walkInStart.Add(-1*time.Hour), walkInStart.Add(12*time.Hour), 0)
	if err != nil {
		return out, err
	}
	var next *models.Appointment
	for i := range list {
		if list[i].AppointmentTime != nil && !list[i].AppointmentTime.Before(walkInStart) {
			next = &list[i]
			break
		}
	}
	if next == nil || next.AppointmentTime == nil {
		return out, nil
	}

	predictedFinish := appointmentPredictedFinishAt(walkInStart, projectOccupiedMinutes, &merchant)
	delayReserved := int(math.Ceil(predictedFinish.Sub(*next.AppointmentTime).Minutes()))
	if delayReserved < 0 {
		delayReserved = 0
	}
	out.NextAppointmentID = &next.ID
	out.PredictedWaitMinutes = delayReserved
	if delayReserved > 0 {
		out.State = appointmentAvailabilityUnavailable
		out.Reason = fmt.Sprintf("该现场单会占用 %s 的预约时段，当前不可分配", next.AppointmentTime.Format("15:04"))
		out.DecisionMode = "strict_reservation_lock"
		altWait, hasAlt, err := earliestAlternativeTechnicianWaitMinutes(tx, merchant, technicianID, walkInStart)
		if err != nil {
			return out, err
		}
		if hasAlt {
			out.AlternativeWaitMinutes = altWait
		}
	}
	return out, nil
}

func estimateAppointmentArrivalDelay(tx *gorm.DB, merchant models.Merchant, technicianID uint, now time.Time) (int, *time.Time, error) {
	if tx == nil || technicianID == 0 {
		return 0, nil, nil
	}

	blocker, err := loadEarliestTechnicianConflictSession(
		tx,
		merchant.ID,
		technicianID,
		models.ExpandStatusesWithKnownPrefixes([]string{"start_pending", "delay_pending", "serving", "auto_finishing"}),
	)
	if err != nil {
		return 0, nil, err
	}
	if blocker == nil {
		return 0, nil, nil
	}

	var readyAt time.Time
	switch {
	case blocker.PredictedReadyAt != nil:
		readyAt = *blocker.PredictedReadyAt
	case blocker.ScheduledFinishAt != nil:
		readyAt = blocker.ScheduledFinishAt.Add(time.Duration(merchantAppointmentPredictionBufferMinutes(&merchant)) * time.Minute)
	default:
		return 0, nil, nil
	}
	delayMinutes := int(math.Ceil(readyAt.Sub(now).Minutes()))
	if delayMinutes <= 0 {
		return 0, nil, nil
	}
	return delayMinutes, &readyAt, nil
}

func loadEarliestTechnicianConflictSession(tx *gorm.DB, merchantID, technicianID uint, statuses []string) (*models.ServiceSession, error) {
	if tx == nil || merchantID == 0 || technicianID == 0 || len(statuses) == 0 {
		return nil, nil
	}
	rows, err := tx.Table("service_sessions").
		Select("id, predicted_ready_at, scheduled_finish_at").
		Where("merchant_id = ? AND technician_id = ? AND status IN ?", merchantID, technicianID, statuses).
		Order("COALESCE(predicted_ready_at, scheduled_finish_at, updated_at) asc, id asc").
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
		session              models.ServiceSession
		predictedReadyAtRaw  interface{}
		scheduledFinishAtRaw interface{}
	)
	if err := rows.Scan(&session.ID, &predictedReadyAtRaw, &scheduledFinishAtRaw); err != nil {
		return nil, err
	}
	if v, ok := parseDBTimeValue(predictedReadyAtRaw); ok {
		session.PredictedReadyAt = &v
	}
	if v, ok := parseDBTimeValue(scheduledFinishAtRaw); ok {
		session.ScheduledFinishAt = &v
	}
	return &session, nil
}

func hasAppointmentProtectionBlock(tx *gorm.DB, appointmentID uint) (bool, error) {
	if tx == nil || appointmentID == 0 {
		return false, nil
	}
	var count int64
	if err := tx.Model(&models.AppointmentProtectionBlock{}).Where("appointment_id = ?", appointmentID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func recordAppointmentProtectionBlock(tx *gorm.DB, merchantID uint, appointmentID uint, technicianID uint, walkInServiceSessionID *uint, availability appointmentAvailability, blockedAt time.Time) error {
	if tx == nil || appointmentID == 0 || technicianID == 0 || availability.NextAppointmentID == nil || *availability.NextAppointmentID == 0 {
		return nil
	}
	if availability.State != appointmentAvailabilityUnavailable {
		return nil
	}
	row := models.AppointmentProtectionBlock{
		MerchantID:                   merchantID,
		AppointmentID:                appointmentID,
		TechnicianID:                 technicianID,
		WalkInServiceSessionID:       walkInServiceSessionID,
		BlockedReason:                availability.Reason,
		PredictedReservedWaitMinutes: availability.PredictedWaitMinutes,
		AlternativeWaitMinutes:       availability.AlternativeWaitMinutes,
		DecisionMode:                 strings.TrimSpace(availability.DecisionMode),
		BlockedAt:                    &blockedAt,
	}
	if row.DecisionMode == "" {
		row.DecisionMode = "strict_reservation_lock"
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func computeAppointmentRescheduleEligibility(tx *gorm.DB, appt models.Appointment, merchant models.Merchant, now time.Time) (appointmentRescheduleEligibility, error) {
	out := appointmentRescheduleEligibility{
		Allowed:      false,
		RuleMode:     appointmentRescheduleForbidden,
		AllowedDates: []string{},
	}
	if !appointmentAllowsReschedule(appt.Status) {
		out.Reason = "当前预约状态不可改签"
		return out, nil
	}
	if appt.AppointmentTime == nil {
		out.Reason = "预约时间不存在"
		return out, nil
	}

	loc := now.Location()
	appointmentTime := appt.AppointmentTime.In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	appointmentDateStart := time.Date(appointmentTime.Year(), appointmentTime.Month(), appointmentTime.Day(), 0, 0, 0, 0, loc)
	todayDate := todayStart.Format("2006-01-02")
	tomorrowDate := todayStart.Add(24 * time.Hour).Format("2006-01-02")
	yesterdayStart := todayStart.Add(-24 * time.Hour)

	switch {
	case appointmentDateStart.Equal(todayStart):
		out.Allowed = true
		out.RuleMode = appointmentRescheduleTomorrowOnly
		out.AllowedDates = []string{tomorrowDate}
		out.DefaultDate = tomorrowDate
		return out, nil
	case appointmentDateStart.Equal(yesterdayStart):
		todayAnchorTime := time.Date(now.Year(), now.Month(), now.Day(), appointmentTime.Hour(), appointmentTime.Minute(), appointmentTime.Second(), appointmentTime.Nanosecond(), loc)
		out.MinutesUntilAnchor = int(math.Ceil(todayAnchorTime.Sub(now).Minutes()))
		protected, err := hasAppointmentProtectionBlock(tx, appt.ID)
		if err != nil {
			return out, err
		}
		out.ProtectionConsumed = protected
		if protected {
			out.Reason = "该预约已占用门店预约保护资源，当前不可改签"
			return out, nil
		}
		if out.MinutesUntilAnchor <= merchantAppointmentRescheduleNextDayOnlyThresholdMinutes(&merchant) {
			out.Reason = fmt.Sprintf("距原预约时间不足 %d 分钟，当前不可改签", merchantAppointmentRescheduleNextDayOnlyThresholdMinutes(&merchant))
			return out, nil
		}
		if out.MinutesUntilAnchor > merchantAppointmentRescheduleSameOrNextDayThresholdMinutes(&merchant) {
			out.Allowed = true
			out.RuleMode = appointmentRescheduleTodayOrTomorrow
			out.AllowedDates = []string{todayDate, tomorrowDate}
			out.DefaultDate = todayDate
			return out, nil
		}
		out.Allowed = true
		out.RuleMode = appointmentRescheduleTomorrowOnly
		out.AllowedDates = []string{tomorrowDate}
		out.DefaultDate = tomorrowDate
		return out, nil
	case appointmentDateStart.Before(yesterdayStart):
		out.Reason = "更早历史预约不支持改签"
		return out, nil
	default:
		out.Allowed = true
		out.RuleMode = appointmentRescheduleTomorrowOnly
		out.AllowedDates = []string{tomorrowDate}
		out.DefaultDate = tomorrowDate
		return out, nil
	}
}

func detectServiceSessionSource(tx *gorm.DB, merchant models.Merchant, card models.Card, now time.Time) (serviceSessionSource, error) {
	source := serviceSessionSource{SourceType: serviceSessionSourceWalkIn}
	if tx == nil || merchant.ID == 0 || card.ID == 0 {
		return source, nil
	}

	rows, err := tx.Table("appointments").
		Select("id, technician_id, appointment_time, status").
		Where("card_id = ? AND merchant_id = ? AND status IN ? AND appointment_time IS NOT NULL", card.ID, merchant.ID, []string{"confirmed", "arrived"}).
		Order("appointment_time asc").
		Limit(1).
		Rows()
	if err != nil {
		return source, err
	}
	defer rows.Close()
	if !rows.Next() {
		return source, nil
	}

	var apptID uint
	var technicianID *uint
	var rawAppointmentTime any
	var status string
	if err := rows.Scan(&apptID, &technicianID, &rawAppointmentTime, &status); err != nil {
		return source, err
	}
	appointmentTime, ok := parseDBTimeValue(rawAppointmentTime)
	if !ok {
		return source, fmt.Errorf("unsupported appointment_time value: %T", rawAppointmentTime)
	}

	appt := models.Appointment{
		ID:              apptID,
		TechnicianID:    technicianID,
		AppointmentTime: &appointmentTime,
		Status:          status,
	}
	start, end, ok := appointmentDetectableWindow(appt, &merchant)
	if !ok || now.Before(start) || now.After(end) {
		return source, nil
	}
	source.SourceType = serviceSessionSourceAppointment
	source.SourceID = &appt.ID
	source.Appointment = &appt
	return source, nil
}

func parseDBTimeValue(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x, true
	case *time.Time:
		if x != nil {
			return *x, true
		}
	case string:
		return parseDBTimeString(x)
	case []byte:
		return parseDBTimeString(string(x))
	}
	return time.Time{}, false
}

func parseDBTimeString(v string) (time.Time, bool) {
	layouts := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func loadAppointmentByID(tx *gorm.DB, appointmentID uint) (*models.Appointment, error) {
	if tx == nil || appointmentID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("appointments").
		Select(appointmentSelectColumns()).
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
	appt, err := scanAppointmentRow(rows)
	if err != nil {
		return nil, err
	}
	return &appt, nil
}

func gormValueToUint(v any) (uint, bool) {
	switch x := v.(type) {
	case uint:
		return x, true
	case int64:
		if x > 0 {
			return uint(x), true
		}
	case int:
		if x > 0 {
			return uint(x), true
		}
	case int32:
		if x > 0 {
			return uint(x), true
		}
	case uint64:
		if x > 0 {
			return uint(x), true
		}
	case []byte:
		if parsed, err := strconv.ParseUint(string(x), 10, 64); err == nil && parsed > 0 {
			return uint(parsed), true
		}
	case string:
		if parsed, err := strconv.ParseUint(x, 10, 64); err == nil && parsed > 0 {
			return uint(parsed), true
		}
	}
	return 0, false
}
