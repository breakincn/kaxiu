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
