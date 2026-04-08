package config

import (
	"kabao/models"

	"gorm.io/gorm"
)

const DefaultProjectRoomSelectTimeoutSeconds = 90

func GetProjectRoomSelectTimeoutSeconds(tx *gorm.DB, merchantID uint, projectID *uint) int {
	if tx == nil || merchantID == 0 {
		return DefaultProjectRoomSelectTimeoutSeconds
	}
	project, err := ResolveMerchantProject(tx, merchantID, projectID)
	if err != nil || project == nil {
		return DefaultProjectRoomSelectTimeoutSeconds
	}
	return NormalizeProjectRoomSelectTimeoutSeconds(project.RoomSelectTimeoutSeconds)
}

func NormalizeProjectRoomSelectTimeoutSeconds(v int) int {
	if v < 1 {
		return DefaultProjectRoomSelectTimeoutSeconds
	}
	if v > 3600 {
		return 3600
	}
	return v
}

func ResolveProjectConfigForSession(tx *gorm.DB, merchantID uint, projectID *uint) (*models.MerchantProject, error) {
	return ResolveMerchantProject(tx, merchantID, projectID)
}
