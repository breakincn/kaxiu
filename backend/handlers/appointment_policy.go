package handlers

import (
	"errors"
	"fmt"
	"kabao/models"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	appointmentCleanupBufferMinutes = 3
	appointmentReservedWeight       = 2.5
	appointmentWalkInWeight         = 1.0
)

type appointmentAvailabilityState string

const (
	appointmentAvailabilitySafe        appointmentAvailabilityState = "safe"
	appointmentAvailabilityConditional appointmentAvailabilityState = "conditional"
	appointmentAvailabilityUnavailable appointmentAvailabilityState = "unavailable"
)

type appointmentAvailability struct {
	State                appointmentAvailabilityState `json:"availability_state"`
	PredictedWaitMinutes int                          `json:"predicted_wait_minutes"`
	Reason               string                       `json:"availability_reason"`
	NextAppointmentID    *uint                        `json:"next_appointment_id,omitempty"`
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

func merchantAppointmentMaxWaitMinutes(m *models.Merchant) int {
	if m != nil && m.AppointmentMaxWaitMinutes > 0 {
		return m.AppointmentMaxWaitMinutes
	}
	return 15
}

func merchantAppointmentPredictionBufferMinutes(m *models.Merchant) int {
	if m != nil && m.AppointmentPredictionBufferMinute > 0 {
		return m.AppointmentPredictionBufferMinute
	}
	return 5
}

func appointmentProtectedStatuses() []string {
	return []string{"confirmed", "arrived", "finished", "completed"}
}

func appointmentPredictedFinishAt(start time.Time, serviceMinutes int, merchant *models.Merchant) time.Time {
	if serviceMinutes <= 0 {
		serviceMinutes = 30
	}
	// 预测公式对前后台保持一致：项目标准时长 + 固定收尾缓冲 + 商户可配置的风险缓冲。
	totalMinutes := serviceMinutes + appointmentCleanupBufferMinutes + merchantAppointmentPredictionBufferMinutes(merchant)
	return start.Add(time.Duration(totalMinutes) * time.Minute)
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
	// appointment_waiting 必须视为活跃占用，避免预约优先等待中的会话被新的普通单再次插入。
	return models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting", "appointment_waiting", "start_pending", "delay_pending", "serving", "auto_finishing"})
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

func loadProtectedAppointmentsForTechnician(tx *gorm.DB, merchantID uint, technicianID uint, start, end time.Time, excludeAppointmentID uint) ([]models.Appointment, error) {
	if tx == nil || merchantID == 0 || technicianID == 0 {
		return nil, nil
	}
	var list []models.Appointment
	q := tx.Where("merchant_id = ? AND technician_id = ? AND appointment_time IS NOT NULL AND appointment_time >= ? AND appointment_time <= ?",
		merchantID, technicianID, start, end).
		Where("status IN ?", appointmentProtectedStatuses()).
		Order("appointment_time asc")
	if excludeAppointmentID > 0 {
		q = q.Where("id <> ?", excludeAppointmentID)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func evaluateBookingTechnicianAvailability(tx *gorm.DB, merchant models.Merchant, technicianID uint, appointmentStart time.Time, projectDuration int, excludeAppointmentID uint) (appointmentAvailability, error) {
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

	newFinish := appointmentPredictedFinishAt(appointmentStart, projectDuration, &merchant)
	maxWait := merchantAppointmentMaxWaitMinutes(&merchant)
	for _, appt := range existing {
		if appt.AppointmentTime == nil {
			continue
		}
		existingStart := *appt.AppointmentTime
		existingFinish := appointmentPredictedFinishAt(existingStart, getAppointmentServiceMinutes(merchant.ID, appt), &merchant)

		if existingStart.Before(appointmentStart) {
			waitMinutes := int(math.Ceil(existingFinish.Sub(appointmentStart).Minutes()))
			if waitMinutes > maxWait {
				return appointmentAvailability{
					State:                appointmentAvailabilityUnavailable,
					PredictedWaitMinutes: waitMinutes,
					Reason:               fmt.Sprintf("该客服前一笔预约预计会让当前预约等待 %d 分钟", waitMinutes),
					NextAppointmentID:    &appt.ID,
				}, nil
			}
			if waitMinutes > out.PredictedWaitMinutes {
				out.State = appointmentAvailabilityConditional
				out.PredictedWaitMinutes = waitMinutes
				out.Reason = fmt.Sprintf("该客服前一笔预约可能导致等待 %d 分钟", waitMinutes)
				out.NextAppointmentID = &appt.ID
			}
			continue
		}

		waitMinutes := int(math.Ceil(newFinish.Sub(existingStart).Minutes()))
		if waitMinutes > maxWait {
			return appointmentAvailability{
				State:                appointmentAvailabilityUnavailable,
				PredictedWaitMinutes: waitMinutes,
				Reason:               fmt.Sprintf("当前预约会挤占后续预约 %d 分钟", waitMinutes),
				NextAppointmentID:    &appt.ID,
			}, nil
		}
		if waitMinutes > out.PredictedWaitMinutes {
			out.State = appointmentAvailabilityConditional
			out.PredictedWaitMinutes = waitMinutes
			out.Reason = fmt.Sprintf("当前预约会轻微挤占后续预约，预计等待 %d 分钟", waitMinutes)
			out.NextAppointmentID = &appt.ID
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
		var sess models.ServiceSession
		err := tx.Where("merchant_id = ? AND technician_id = ? AND status IN ?", merchant.ID, att.TechnicianID, appointmentActiveConflictStatuses()).
			Order("COALESCE(predicted_ready_at, scheduled_finish_at, updated_at) asc").
			First(&sess).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return 0, false, err
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

func evaluateWalkInTechnicianAvailability(tx *gorm.DB, merchant models.Merchant, technicianID uint, walkInStart time.Time, projectDuration int) (appointmentAvailability, error) {
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

	predictedFinish := appointmentPredictedFinishAt(walkInStart, projectDuration, &merchant)
	delayReserved := int(math.Ceil(predictedFinish.Sub(*next.AppointmentTime).Minutes()))
	if delayReserved < 0 {
		delayReserved = 0
	}
	out.NextAppointmentID = &next.ID
	out.PredictedWaitMinutes = delayReserved
	if delayReserved > merchantAppointmentMaxWaitMinutes(&merchant) {
		out.State = appointmentAvailabilityUnavailable
		out.Reason = fmt.Sprintf("该现场单会让后续预约等待 %d 分钟，超过上限", delayReserved)
		return out, nil
	}

	altWait, hasAlt, err := earliestAlternativeTechnicianWaitMinutes(tx, merchant, technicianID, walkInStart)
	if err != nil {
		return out, err
	}
	if hasAlt && float64(altWait)*appointmentWalkInWeight < float64(delayReserved)*appointmentReservedWeight {
		out.State = appointmentAvailabilityUnavailable
		out.Reason = fmt.Sprintf("等待其他客服约 %d 分钟比压后预约更优", altWait)
		return out, nil
	}
	if delayReserved > 0 {
		out.State = appointmentAvailabilityConditional
		out.Reason = fmt.Sprintf("将影响 %s 的预约，预计等待 %d 分钟", next.AppointmentTime.Format("15:04"), delayReserved)
	}
	return out, nil
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
	var appt models.Appointment
	if err := tx.Where("id = ?", appointmentID).First(&appt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &appt, nil
}
