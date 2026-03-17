package sessionflow

import (
	"kabao/models"
	"kabao/queue"
	"time"

	"gorm.io/gorm"
)

type FinishOptions struct {
	AllowedBaseStatuses []string
	MarkQueueDone       bool
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
	if s.TechnicianID != nil && *s.TechnicianID > 0 {
		uUpdates["technician_id"] = *s.TechnicianID
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
		TechnicianID        *uint  `gorm:"column:technician_id"`
		StartConfirmedAtRaw string `gorm:"column:start_confirmed_at"`
		FinishedAtRaw       string `gorm:"column:finished_at"`
	}
	query := tx.
		Table("service_sessions").
		Select("id", "merchant_id", "initial_usage_id", "status", "technician_id", "start_confirmed_at", "finished_at").
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
		ID:             row.ID,
		MerchantID:     row.MerchantID,
		InitialUsageID: row.InitialUsageID,
		Status:         row.Status,
		TechnicianID:   row.TechnicianID,
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
