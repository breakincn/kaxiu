package config

import "gorm.io/gorm"

const DefaultProjectRoomSelectTimeoutSeconds = 90

func GetProjectRoomSelectTimeoutSeconds(tx *gorm.DB, projectID *uint) int {
	if tx == nil || projectID == nil || *projectID == 0 {
		return DefaultProjectRoomSelectTimeoutSeconds
	}
	var project struct {
		RoomSelectTimeoutSeconds int `gorm:"column:room_select_timeout_seconds"`
	}
	if err := tx.Table("merchant_projects").
		Select("room_select_timeout_seconds").
		Where("id = ?", *projectID).
		First(&project).Error; err != nil {
		return DefaultProjectRoomSelectTimeoutSeconds
	}
	if project.RoomSelectTimeoutSeconds < 1 {
		return DefaultProjectRoomSelectTimeoutSeconds
	}
	if project.RoomSelectTimeoutSeconds > 3600 {
		return 3600
	}
	return project.RoomSelectTimeoutSeconds
}
