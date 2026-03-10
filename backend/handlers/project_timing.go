package handlers

import (
	"kabao/models"

	"gorm.io/gorm"
)

func normalizeSessionStartDelaySeconds(v int) int {
	if v < 0 {
		return 60
	}
	return v
}

func resolveProjectServiceConfig(tx *gorm.DB, merchantID uint, projectID *uint, defaultDuration int, defaultDelaySeconds int) (int, int) {
	durationMinutes := defaultDuration
	delaySeconds := normalizeSessionStartDelaySeconds(defaultDelaySeconds)

	if tx == nil || merchantID == 0 || projectID == nil || *projectID == 0 {
		return durationMinutes, delaySeconds
	}

	var project models.MerchantProject
	if err := tx.Where("id = ? AND merchant_id = ?", *projectID, merchantID).First(&project).Error; err != nil {
		return durationMinutes, delaySeconds
	}

	if project.Duration > 0 {
		durationMinutes = project.Duration
	}
	delaySeconds = normalizeSessionStartDelaySeconds(project.StartDelaySeconds)
	return durationMinutes, delaySeconds
}
