package sessionflow

import (
	"kabao/models"
	"kabao/queue"
	"time"

	"gorm.io/gorm"
)

type FailOptions struct {
	AllowedBaseStatuses []string
	TargetStatus        string
	MarkQueueDone       bool
	SessionUpdates      map[string]interface{}
}

func FailServiceSessionAndRefund(tx *gorm.DB, s *models.ServiceSession, merchant *models.Merchant, now time.Time, opts FailOptions) error {
	if tx == nil || s == nil {
		return nil
	}
	if s.InitialUsageID == 0 {
		return nil
	}

	allowed := opts.AllowedBaseStatuses
	if len(allowed) == 0 {
		allowed = []string{"timeout_waiting", "start_pending", "delay_pending"}
	}
	targetStatus := opts.TargetStatus
	if targetStatus == "" {
		targetStatus = "timeout_failed"
	}

	updates := map[string]interface{}{
		"status":      models.ApplyStatusPrefix(s.Status, targetStatus),
		"finished_at": now,
	}
	for key, value := range opts.SessionUpdates {
		updates[key] = value
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes(allowed)).
		Updates(updates).Error; err != nil {
		return err
	}
	if s.SourceType == "appointment" && s.SourceID != nil {
		if err := syncAppointmentOutcome(tx, *s.SourceID, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &now,
			"disruption_status":                  "closed",
			"disruption_reason":                  "merchant_timeout_failed",
			"liability_level":                    "merchant",
			"salary_settlement_reference_status": "refund",
		}, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &now,
			"liability_level":                    "merchant",
			"salary_settlement_reference_status": "refund",
			"latest_reason":                      "merchant_timeout_failed",
		}); err != nil {
			return err
		}
	}

	return FailUsageAndRefund(tx, s.InitialUsageID, merchant, now, opts.MarkQueueDone)
}

func FailUsageAndRefund(tx *gorm.DB, usageID uint, merchant *models.Merchant, now time.Time, markQueueDone bool) error {
	if tx == nil || usageID == 0 {
		return nil
	}
	if markQueueDone && merchant != nil && merchant.SupportQueue && queue.Default != nil {
		date := now.Format("2006-01-02")
		queue.Default.MarkDone(merchant.ID, date, queue.QueueTypeOnsite, usageID, now)
	}
	var usage models.Usage
	query := tx.Select("id", "card_id", "used_times", "status").Limit(1).Find(&usage, usageID)
	if query.Error != nil {
		return query.Error
	}
	if query.RowsAffected == 0 {
		return nil
	}
	if usage.Status != "in_progress" {
		return nil
	}
	if err := tx.Model(&models.Usage{}).
		Where("id = ? AND status = ?", usageID, "in_progress").
		Updates(map[string]interface{}{
			"status":      "failed",
			"finished_at": now,
		}).Error; err != nil {
		return err
	}
	if usage.CardID > 0 && usage.UsedTimes > 0 {
		if err := tx.Model(&models.Card{}).
			Where("id = ?", usage.CardID).
			Updates(map[string]interface{}{
				"remain_times": gorm.Expr("remain_times + ?", usage.UsedTimes),
				"used_times":   gorm.Expr("used_times - ?", usage.UsedTimes),
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func CompleteUnstartedServiceSession(tx *gorm.DB, usageID uint, merchantID uint, finishedAt time.Time) (bool, error) {
	if tx == nil || usageID == 0 {
		return false, nil
	}
	var s struct {
		ID               uint       `gorm:"column:id"`
		Status           string     `gorm:"column:status"`
		TechnicianID     *uint      `gorm:"column:technician_id"`
		SourceType       string     `gorm:"column:source_type"`
		SourceID         *uint      `gorm:"column:source_id"`
		StartConfirmedAt *time.Time `gorm:"column:start_confirmed_at"`
	}
	query := tx.Table("service_sessions").
		Select("id,status,technician_id,source_type,source_id,start_confirmed_at").
		Where("initial_usage_id = ?", usageID).
		Order("id desc").
		Limit(1).
		Find(&s)
	if query.Error != nil {
		return false, query.Error
	}
	if query.RowsAffected == 0 {
		return false, nil
	}
	if s.StartConfirmedAt != nil {
		return false, nil
	}
	if s.TechnicianID != nil && *s.TechnicianID > 0 {
		_ = tx.Model(&models.TechnicianAttendance{}).
			Where("merchant_id = ? AND technician_id = ? AND status = ?", merchantID, *s.TechnicianID, "busy").
			Updates(map[string]interface{}{"status": "idle"}).Error
	}
	if err := tx.Table("service_sessions").
		Where("id = ? AND status NOT IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"finished", "canceled"})).
		Updates(map[string]interface{}{
			"status":                  models.ApplyStatusPrefix(s.Status, "finished"),
			"finished_at":             finishedAt,
			"technician_id":           nil,
			"room_id":                 nil,
			"room_locked_at":          nil,
			"room_select_deadline_at": nil,
		}).Error; err != nil {
		return false, err
	}
	if err := tx.Model(&models.Usage{}).
		Where("id = ?", usageID).
		Updates(map[string]interface{}{
			"status":        "success",
			"technician_id": nil,
			"finished_at":   finishedAt,
		}).Error; err != nil {
		return false, err
	}
	if s.SourceType == "appointment" && s.SourceID != nil {
		if err := syncAppointmentOutcome(tx, *s.SourceID, map[string]interface{}{
			"status":                             "completed",
			"completed_at":                       &finishedAt,
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &finishedAt,
			"disruption_status":                  "closed",
			"disruption_reason":                  "service_completed",
			"liability_level":                    "none",
			"salary_settlement_reference_status": "normal",
		}, map[string]interface{}{
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &finishedAt,
			"liability_level":                    "none",
			"salary_settlement_reference_status": "normal",
			"latest_reason":                      "service_completed",
		}); err != nil {
			return false, err
		}
	}
	return true, nil
}
