package models

import "time"

type ServiceRole struct {
	ID                    uint       `json:"id" gorm:"primaryKey"`
	Key                   string     `json:"key" gorm:"size:50;uniqueIndex;not null"`
	Name                  string     `json:"name" gorm:"size:50;not null"`
	Description           string     `json:"description" gorm:"size:255"`
	IsActive              bool       `json:"is_active" gorm:"default:true"`
	AllowPermissionAdjust bool       `json:"allow_permission_adjust" gorm:"default:false"`
	Sort                  int        `json:"sort" gorm:"default:0"`
	CreatedAt             *time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt             *time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ServiceRole) TableName() string {
	return "service_roles"
}

func (ServiceRole) TableComment() string {
	return "平台客服类型表"
}
