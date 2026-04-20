package sessionflow

import (
	"kabao/appointmentdelay"
	"kabao/appointmentliability"
	"kabao/models"
	"kabao/queue"
	"time"

	"gorm.io/gorm"
)

type FinishOptions struct {
	AllowedBaseStatuses []string
	MarkQueueDone       bool
}

func sessionflowServiceSessionPrimaryTechnicianIDs(s *models.ServiceSession) []uint {
	if s == nil {
		return nil
	}
	if len(s.StartConfirmedTechnicianIDs) > 0 {
		return []uint(s.StartConfirmedTechnicianIDs)
	}
	if len(s.ServiceTechnicianIDs) > 0 {
		return []uint(s.ServiceTechnicianIDs)
	}
	if s.LastTechnicianID != nil && *s.LastTechnicianID > 0 {
		return []uint{*s.LastTechnicianID}
	}
	return nil
}

func syncAppointmentOutcome(tx *gorm.DB, appointmentID uint, appointmentUpdates map[string]interface{}, settlementUpdates map[string]interface{}) error {
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

func FinishServiceSession(tx *gorm.DB, s *models.ServiceSession, merchant *models.Merchant, now time.Time, opts FinishOptions) error {
	if tx == nil || s == nil {
		return nil
	}
	if s.StartConfirmedAt == nil {
		return nil
	}

	allowed := opts.AllowedBaseStatuses
	if len(allowed) == 0 {
		allowed = []string{"serving", "auto_finishing"}
	}

	updates := map[string]interface{}{
		"status": models.ApplyStatusPrefix(s.Status, "finished"),
	}
	if s.FinishedAt == nil {
		updates["finished_at"] = now
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ? AND start_confirmed_at IS NOT NULL", s.ID, models.ExpandStatusesWithKnownPrefixes(allowed)).
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
	if ids := sessionflowServiceSessionPrimaryTechnicianIDs(s); len(ids) > 0 {
		uUpdates["technician_id"] = ids[0]
	}
	if err := tx.Model(&models.Usage{}).
		Where("id = ? AND status = ?", s.InitialUsageID, "in_progress").
		Updates(uUpdates).Error; err != nil {
		return err
	}

	if opts.MarkQueueDone && merchant != nil && merchant.SupportQueue && queue.Default != nil {
		date := now.Format("2006-01-02")
		queue.Default.MarkDone(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID, now)
	}
	if s.SourceType == "appointment" && s.SourceID != nil {
		completedAt := finishedAt
		actualStartAt := completedAt
		if s.StartedAt != nil {
			actualStartAt = *s.StartedAt
		}
		if err := syncAppointmentOutcome(tx, *s.SourceID, map[string]interface{}{
			"status":                             "completed",
			"completed_at":                       &completedAt,
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &completedAt,
			"disruption_status":                  "closed",
			"disruption_reason":                  "service_completed",
			"liability_level":                    "none",
			"salary_settlement_reference_status": "normal",
			"actual_start_at":                    actualStartAt,
		}, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &completedAt,
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

func FinalizeUsageAndSession(tx *gorm.DB, usageID uint, merchant *models.Merchant, finishedAt time.Time, opts FinishOptions) (bool, error) {
	if tx == nil || usageID == 0 {
		return false, nil
	}

	var row struct {
		ID                  uint   `gorm:"column:id"`
		MerchantID          uint   `gorm:"column:merchant_id"`
		InitialUsageID      uint   `gorm:"column:initial_usage_id"`
		Status              string `gorm:"column:status"`
		LastTechnicianID    *uint  `gorm:"column:last_technician_id"`
		ServiceTechnicianIDs models.MerchantProjectDefaultServiceTechnicianIDs `gorm:"column:service_technician_ids"`
		StartConfirmedTechnicianIDs models.MerchantProjectDefaultServiceTechnicianIDs `gorm:"column:start_confirmed_technician_ids"`
		SourceType          string `gorm:"column:source_type"`
		SourceID            *uint  `gorm:"column:source_id"`
		StartConfirmedAtRaw string `gorm:"column:start_confirmed_at"`
		FinishedAtRaw       string `gorm:"column:finished_at"`
	}
	query := tx.
		Table("service_sessions").
		Select("id", "merchant_id", "initial_usage_id", "status", "last_technician_id", "service_technician_ids", "start_confirmed_technician_ids", "source_type", "source_id", "start_confirmed_at", "finished_at").
		Where("initial_usage_id = ?", usageID).
		Order("id desc").
		Limit(1).
		Find(&row)
	if query.Error != nil {
		return false, query.Error
	}
	if query.RowsAffected == 0 {
		return false, nil
	}

	if row.StartConfirmedAtRaw == "" {
		handled, err := CompleteUnstartedServiceSession(tx, usageID, row.MerchantID, finishedAt)
		if err != nil {
			return true, err
		}
		if handled {
			if opts.MarkQueueDone && merchant != nil && merchant.SupportQueue && queue.Default != nil {
				date := finishedAt.Format("2006-01-02")
				queue.Default.MarkDone(merchant.ID, date, queue.QueueTypeOnsite, usageID, finishedAt)
			}
			return true, nil
		}
		return false, nil
	}

	s := models.ServiceSession{
		ID:                         row.ID,
		MerchantID:                 row.MerchantID,
		InitialUsageID:             row.InitialUsageID,
		Status:                     row.Status,
		LastTechnicianID:           row.LastTechnicianID,
		ServiceTechnicianIDs:       row.ServiceTechnicianIDs,
		StartConfirmedTechnicianIDs: row.StartConfirmedTechnicianIDs,
		SourceType:                 row.SourceType,
		SourceID:                   row.SourceID,
	}
	s.StartConfirmedAt = &finishedAt

	finishOpts := opts
	if len(finishOpts.AllowedBaseStatuses) == 0 {
		finishOpts.AllowedBaseStatuses = []string{"start_pending", "delay_pending", "serving", "auto_finishing"}
	}
	if err := FinishServiceSession(tx, &s, merchant, finishedAt, finishOpts); err != nil {
		return true, err
	}
	return true, nil
}
