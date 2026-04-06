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

func normalizeProjectServiceGapMinutes(v int) int {
	if v < 0 {
		return 3
	}
	return v
}

func normalizeAppointmentSlotGranularityMinutes(v int) int {
	if v < 5 || v > 60 || v%5 != 0 {
		return 5
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

func resolveProjectBookingConfig(tx *gorm.DB, merchantID uint, projectID *uint, defaultDuration int, defaultGapMinutes int) (int, int) {
	durationMinutes := defaultDuration
	gapMinutes := normalizeProjectServiceGapMinutes(defaultGapMinutes)

	if tx == nil || merchantID == 0 || projectID == nil || *projectID == 0 {
		return durationMinutes, gapMinutes
	}

	var project models.MerchantProject
	if err := tx.Where("id = ? AND merchant_id = ?", *projectID, merchantID).First(&project).Error; err != nil {
		return durationMinutes, gapMinutes
	}

	if project.Duration > 0 {
		durationMinutes = project.Duration
	}
	gapMinutes = normalizeProjectServiceGapMinutes(project.ServiceGapMinutes)
	return durationMinutes, gapMinutes
}

func projectBookingOccupiedMinutes(durationMinutes int, gapMinutes int) int {
	if durationMinutes <= 0 {
		durationMinutes = 30
	}
	if gapMinutes < 0 {
		gapMinutes = 3
	}
	return durationMinutes + gapMinutes
}
