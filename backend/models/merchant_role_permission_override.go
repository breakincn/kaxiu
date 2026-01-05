package models

import "time"

type MerchantRolePermissionOverride struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	MerchantID    uint       `json:"merchant_id" gorm:"index;uniqueIndex:uidx_m_role_perm"`
	ServiceRoleID uint       `json:"service_role_id" gorm:"index;uniqueIndex:uidx_m_role_perm"`
	PermissionID  uint       `json:"permission_id" gorm:"index;uniqueIndex:uidx_m_role_perm"`
	Allowed       bool       `json:"allowed" gorm:"default:false"`
	CreatedAt     *time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     *time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	ServiceRole ServiceRole `json:"service_role" gorm:"foreignKey:ServiceRoleID"`
	Permission  Permission  `json:"permission" gorm:"foreignKey:PermissionID"`
}

func (MerchantRolePermissionOverride) TableName() string {
	return "merchant_role_permission_overrides"
}
