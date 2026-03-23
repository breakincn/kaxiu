package appointmentliability

import (
	"strconv"
	"strings"
	"time"

	"kabao/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	liabilityMerchant               = "merchant"
	liabilityMerchantExempt         = "merchant_exempt"
	liabilityTechnicianChargeable   = "technician_chargeable"
	salarySettlementStatusOpen      = "open"
	disruptionReasonTechnicianLeave = "technician_leave"
	protectedRepairSlotSourceLeave  = "leave"
)

func SyncLeaveDisruptionLedger(tx *gorm.DB, appointmentID uint, finalizedAt time.Time) error {
	if tx == nil || appointmentID == 0 {
		return nil
	}
	appt, err := loadAppointment(tx, appointmentID)
	if err != nil || appt == nil {
		return err
	}
	level := strings.TrimSpace(appt.LiabilityLevel)
	if level != liabilityMerchant {
		return nil
	}
	if appt.TechnicianID == nil || *appt.TechnicianID == 0 {
		return nil
	}
	affected, err := appointmentAffectedByLeave(tx, appointmentID)
	if err != nil || !affected {
		return err
	}
	existing, err := loadDisruptionLedgerByAppointment(tx, appointmentID)
	if err != nil {
		return err
	}
	if existing != nil {
		return applyInternalLiabilitySnapshot(tx, appt.ID, existing.LiabilityLevel, existing.SalarySettlementReferenceStatus, disruptionReasonTechnicianLeave)
	}

	monthKey := liabilityMonthKey(appt, finalizedAt)
	counter, err := lockMonthlyCounter(tx, appt.MerchantID, *appt.TechnicianID, monthKey)
	if err != nil {
		return err
	}

	internalLevel := liabilityMerchantExempt
	firstExemptUsed := true
	if counter.FirstExemptUsed || counter.LeaveDisruptionCount > 0 {
		internalLevel = liabilityTechnicianChargeable
		firstExemptUsed = counter.FirstExemptUsed
	}
	counter.LeaveDisruptionCount++
	if internalLevel == liabilityMerchantExempt {
		firstExemptUsed = true
	}
	counter.FirstExemptUsed = firstExemptUsed

	if counter.ID == 0 {
		if err := tx.Create(counter).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Model(&models.TechnicianMonthlyDisruptionCounter{}).Where("id = ?", counter.ID).Updates(map[string]interface{}{
			"leave_disruption_count": counter.LeaveDisruptionCount,
			"first_exempt_used":      counter.FirstExemptUsed,
		}).Error; err != nil {
			return err
		}
	}

	ledger := models.TechnicianDisruptionLedger{
		AppointmentID:                   appt.ID,
		BookingRootID:                   appt.BookingRootID,
		MerchantID:                      appt.MerchantID,
		TechnicianID:                    *appt.TechnicianID,
		UserID:                          appt.UserID,
		MonthKey:                        monthKey,
		DisruptionReason:                disruptionReasonTechnicianLeave,
		LiabilityLevel:                  internalLevel,
		SalarySettlementReferenceStatus: salarySettlementStatusOpen,
	}
	if err := tx.Create(&ledger).Error; err != nil {
		return err
	}
	return applyInternalLiabilitySnapshot(tx, appt.ID, internalLevel, salarySettlementStatusOpen, disruptionReasonTechnicianLeave)
}

func liabilityMonthKey(appt *models.Appointment, finalizedAt time.Time) string {
	if appt != nil && appt.AppointmentTime != nil && !appt.AppointmentTime.IsZero() {
		return appt.AppointmentTime.Format("2006-01")
	}
	return finalizedAt.Format("2006-01")
}

func appointmentAffectedByLeave(tx *gorm.DB, appointmentID uint) (bool, error) {
	if tx == nil || appointmentID == 0 {
		return false, nil
	}
	var count int64
	err := tx.Model(&models.ProtectedRepairSlot{}).
		Where("appointment_id = ? AND source_type = ?", appointmentID, protectedRepairSlotSourceLeave).
		Count(&count).Error
	return count > 0, err
}

func loadAppointment(tx *gorm.DB, appointmentID uint) (*models.Appointment, error) {
	if tx == nil || appointmentID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("appointments").
		Select("id, merchant_id, user_id, booking_root_id, technician_id, appointment_time, liability_level").
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
		bookingRootIDRaw   interface{}
		technicianIDRaw    interface{}
		appointmentTimeRaw interface{}
	)
	if err := rows.Scan(&appt.ID, &appt.MerchantID, &appt.UserID, &bookingRootIDRaw, &technicianIDRaw, &appointmentTimeRaw, &appt.LiabilityLevel); err != nil {
		return nil, err
	}
	if v, ok := rawUint(bookingRootIDRaw); ok {
		appt.BookingRootID = &v
	}
	if v, ok := rawUint(technicianIDRaw); ok {
		appt.TechnicianID = &v
	}
	if v, ok := rawTime(appointmentTimeRaw); ok {
		appt.AppointmentTime = &v
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

func lockMonthlyCounter(tx *gorm.DB, merchantID, technicianID uint, monthKey string) (*models.TechnicianMonthlyDisruptionCounter, error) {
	var counter models.TechnicianMonthlyDisruptionCounter
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND technician_id = ? AND month_key = ?", merchantID, technicianID, monthKey).
		First(&counter).Error
	if err == nil {
		return &counter, nil
	}
	if err == gorm.ErrRecordNotFound {
		return &models.TechnicianMonthlyDisruptionCounter{
			MerchantID:   merchantID,
			TechnicianID: technicianID,
			MonthKey:     monthKey,
		}, nil
	}
	return nil, err
}

func loadDisruptionLedgerByAppointment(tx *gorm.DB, appointmentID uint) (*models.TechnicianDisruptionLedger, error) {
	if tx == nil || appointmentID == 0 {
		return nil, nil
	}
	var ledger models.TechnicianDisruptionLedger
	if err := tx.Where("appointment_id = ?", appointmentID).First(&ledger).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ledger, nil
}

func applyInternalLiabilitySnapshot(tx *gorm.DB, appointmentID uint, liabilityLevel, salaryStatus, reason string) error {
	if tx == nil || appointmentID == 0 {
		return nil
	}
	if err := tx.Model(&models.Appointment{}).Where("id = ?", appointmentID).Updates(map[string]interface{}{
		"liability_level":                    liabilityLevel,
		"salary_settlement_reference_status": salaryStatus,
		"disruption_reason":                  reason,
	}).Error; err != nil {
		return err
	}
	var settlementID uint
	if err := tx.Model(&models.Appointment{}).Where("id = ?", appointmentID).Select("appointment_settlement_id").Scan(&settlementID).Error; err != nil {
		return err
	}
	if settlementID == 0 {
		return nil
	}
	return tx.Model(&models.AppointmentSettlement{}).Where("id = ?", settlementID).Updates(map[string]interface{}{
		"liability_level":                    liabilityLevel,
		"salary_settlement_reference_status": salaryStatus,
	}).Error
}
