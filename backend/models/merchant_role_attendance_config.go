package models

import "time"

type MerchantRoleAttendanceConfig struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	MerchantID    uint       `json:"merchant_id" gorm:"index;uniqueIndex:uidx_m_role_att"`
	ServiceRoleID uint       `json:"service_role_id" gorm:"index;uniqueIndex:uidx_m_role_att"`
	RequireAttendance bool    `json:"require_attendance" gorm:"column:require_attendance;default:true"`
	CreatedAt     *time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     *time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (MerchantRoleAttendanceConfig) TableName() string {
	return "merchant_role_attendance_configs"
}
