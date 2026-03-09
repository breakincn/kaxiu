package models

import "time"

type ServiceRole struct {
	ID                         uint       `json:"id" gorm:"primaryKey"`
	MerchantID                 *uint      `json:"merchant_id" gorm:"index"`
	RoleType                   string     `json:"role_type" gorm:"size:20;default:''"`
	Key                        string     `json:"key" gorm:"size:50;uniqueIndex"`
	Name                       string     `json:"name" gorm:"size:50;not null"`
	AccountPrefix              string     `json:"account_prefix" gorm:"size:5;default:''"`
	RequireAttendance          bool       `json:"require_attendance" gorm:"default:true"`
	StartPendingTimeoutSeconds int        `json:"start_pending_timeout_seconds" gorm:"default:300"`
	Description                string     `json:"description" gorm:"size:255"`
	IsActive                   bool       `json:"is_active" gorm:"default:true"`
	AllowPermissionAdjust      bool       `json:"allow_permission_adjust" gorm:"default:false"`
	Sort                       int        `json:"sort" gorm:"default:0"`
	CreatedAt                  *time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                  *time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ServiceRole) TableName() string {
	return "service_roles"
}

func (ServiceRole) TableComment() string {
	return "平台客服类型表"
}
