package models

import "time"

type RolePermission struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	ServiceRoleID uint       `json:"service_role_id" gorm:"index;uniqueIndex:uidx_role_perm"`
	PermissionID  uint       `json:"permission_id" gorm:"index;uniqueIndex:uidx_role_perm"`
	Allowed       bool       `json:"allowed" gorm:"default:false"`
	CreatedAt     *time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     *time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	ServiceRole ServiceRole `json:"service_role" gorm:"foreignKey:ServiceRoleID"`
	Permission  Permission  `json:"permission" gorm:"foreignKey:PermissionID"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
