package scheduler

import (
	"kabao/models"
	"strings"

	"gorm.io/gorm"
)

func backfillLegacyServiceSessionModes(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	type sessionLite struct {
		ID          uint   `gorm:"column:id"`
		MerchantID  uint   `gorm:"column:merchant_id"`
		Status      string `gorm:"column:status"`
		SessionMode string `gorm:"column:session_mode"`
	}

	var rows []sessionLite
	if err := db.Table("service_sessions").
		Select("id", "merchant_id", "status", "session_mode").
		Where("session_mode = '' OR session_mode IS NULL").
		Order("id asc").
		Limit(schedulerBatchLimit).
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	merchantCache := map[uint]models.Merchant{}
	for i := range rows {
		row := rows[i]
		if row.ID == 0 {
			continue
		}

		targetMode := models.InferSessionModeFromStatus(row.Status)
		if targetMode == "" {
			merchant, ok := merchantCache[row.MerchantID]
			if !ok {
				if err := db.Select(
					"id",
					"support_customer_service_mode",
					"support_queue",
					"queue_mode",
					"support_multi_customer_service",
					"support_order_complete",
				).First(&merchant, row.MerchantID).Error; err != nil {
					continue
				}
				merchantCache[row.MerchantID] = merchant
			}
			targetMode = models.ResolveSessionMode(&merchant)
		}
		if strings.TrimSpace(targetMode) == "" {
			continue
		}

		_ = db.Model(&models.ServiceSession{}).
			Where("id = ? AND (session_mode = '' OR session_mode IS NULL)", row.ID).
			Update("session_mode", targetMode).Error
	}
	return nil
}
