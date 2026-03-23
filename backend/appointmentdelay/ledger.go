package appointmentdelay

import (
	"math"
	"strconv"
	"strings"
	"time"

	"kabao/models"

	"gorm.io/gorm"
)

func RecordLedgerIfNeeded(tx *gorm.DB, appointmentID uint, sessionID *uint, fallbackActualStartAt time.Time) error {
	if tx == nil || appointmentID == 0 {
		return nil
	}
	appt, err := loadAppointmentDelayTarget(tx, appointmentID)
	if err != nil || appt == nil {
		return err
	}
	if appt.Status != "completed" {
		return nil
	}
	if appt.ReservedStartAt == nil {
		return nil
	}
	actualStartAt := appt.ActualStartAt
	if actualStartAt == nil && !fallbackActualStartAt.IsZero() {
		actualStartAt = &fallbackActualStartAt
	}
	if actualStartAt == nil || !actualStartAt.After(*appt.ReservedStartAt) {
		return nil
	}
	delayMinutes := int(math.Ceil(actualStartAt.Sub(*appt.ReservedStartAt).Minutes()))
	if delayMinutes <= 1 {
		return nil
	}

	var count int64
	if err := tx.Model(&models.AppointmentDelayLedger{}).Where("appointment_id = ?", appointmentID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	project, _ := loadProjectDelayConfig(tx, appt.MerchantID, appt.ProjectID)
	tolerance := 1
	unitValue := 0
	if project != nil {
		if project.DelayToleranceMinutes > 0 {
			tolerance = project.DelayToleranceMinutes
		}
		unitValue = project.DelayFixedUnitValue
	}
	if delayMinutes <= tolerance {
		return nil
	}
	creditedMinutes := delayMinutes
	actualStart := *actualStartAt
	ledger := models.AppointmentDelayLedger{
		AppointmentID:          appt.ID,
		BookingRootID:          appt.BookingRootID,
		MerchantID:             appt.MerchantID,
		UserID:                 appt.UserID,
		CardID:                 appt.CardID,
		ProjectID:              appt.ProjectID,
		ServiceSessionID:       sessionID,
		TechnicianID:           appt.TechnicianID,
		ScheduledStartAt:       appt.ReservedStartAt,
		ActualStartAt:          &actualStart,
		DelayMinutes:           delayMinutes,
		CreditedMinutes:        creditedMinutes,
		ServiceDurationMinutes: appt.ServiceDurationMinutes,
		UnitCompensationValue:  unitValue,
		LedgerStatus:           "recorded",
		RedeemStatus:           "pending",
	}
	if unitValue > 0 && appt.ServiceDurationMinutes > 0 && strings.TrimSpace(project.DelayCompensationMode) == "amount_bucket" {
		ledger.DelayCompensationValue = int(math.Ceil(float64(unitValue*creditedMinutes) / float64(appt.ServiceDurationMinutes)))
	}
	return tx.Create(&ledger).Error
}

type appointmentDelayTarget struct {
	ID                     uint
	MerchantID             uint
	UserID                 uint
	CardID                 uint
	BookingRootID          *uint
	ProjectID              *uint
	TechnicianID           *uint
	Status                 string
	ReservedStartAt        *time.Time
	ActualStartAt          *time.Time
	ServiceDurationMinutes int
}

func loadAppointmentDelayTarget(tx *gorm.DB, appointmentID uint) (*appointmentDelayTarget, error) {
	if tx == nil || appointmentID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("appointments").
		Select("id, merchant_id, user_id, card_id, booking_root_id, project_id, technician_id, status, reserved_start_at, actual_start_at").
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
		appt               appointmentDelayTarget
		bookingRootIDRaw   interface{}
		projectIDRaw       interface{}
		technicianIDRaw    interface{}
		reservedStartAtRaw interface{}
		actualStartAtRaw   interface{}
	)
	if err := rows.Scan(&appt.ID, &appt.MerchantID, &appt.UserID, &appt.CardID, &bookingRootIDRaw, &projectIDRaw, &technicianIDRaw, &appt.Status, &reservedStartAtRaw, &actualStartAtRaw); err != nil {
		return nil, err
	}
	if v, ok := rawUint(bookingRootIDRaw); ok {
		appt.BookingRootID = &v
	}
	if v, ok := rawUint(projectIDRaw); ok {
		appt.ProjectID = &v
	}
	if v, ok := rawUint(technicianIDRaw); ok {
		appt.TechnicianID = &v
	}
	if v, ok := rawTime(reservedStartAtRaw); ok {
		appt.ReservedStartAt = &v
	}
	if v, ok := rawTime(actualStartAtRaw); ok {
		appt.ActualStartAt = &v
	}
	if appt.ProjectID != nil && *appt.ProjectID > 0 {
		var p models.MerchantProject
		if err := tx.Select("duration").Where("id = ? AND merchant_id = ?", *appt.ProjectID, appt.MerchantID).First(&p).Error; err == nil {
			appt.ServiceDurationMinutes = p.Duration
		}
	}
	return &appt, nil
}

func rawUint(v interface{}) (uint, bool) {
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

func rawTime(v interface{}) (time.Time, bool) {
	switch vv := v.(type) {
	case time.Time:
		return vv, true
	case *time.Time:
		if vv != nil {
			return *vv, true
		}
	case []byte:
		if tm, ok := parseTimeString(string(vv)); ok {
			return tm, true
		}
	case string:
		if tm, ok := parseTimeString(vv); ok {
			return tm, true
		}
	}
	return time.Time{}, false
}

func parseTimeString(s string) (time.Time, bool) {
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
		if tm, err := time.Parse(layout, value); err == nil {
			return tm, true
		}
	}
	return time.Time{}, false
}

func loadProjectDelayConfig(tx *gorm.DB, merchantID uint, projectID *uint) (*models.MerchantProject, error) {
	if tx == nil || projectID == nil || *projectID == 0 {
		return nil, nil
	}
	var project models.MerchantProject
	if err := tx.Select("id", "merchant_id", "duration", "delay_tolerance_minutes", "delay_compensation_mode", "delay_redeem_threshold_percent", "delay_fixed_unit_value").
		Where("id = ? AND merchant_id = ?", *projectID, merchantID).
		First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &project, nil
}
