package models

import "time"

type MerchantRoleStartPendingConfig struct {
	ID                         uint       `json:"id" gorm:"primaryKey"`
	MerchantID                 uint       `json:"merchant_id" gorm:"index"`
	ServiceRoleID              uint       `json:"service_role_id" gorm:"index"`
	StartPendingTimeoutSeconds int        `json:"start_pending_timeout_seconds" gorm:"default:300"`
	CreatedAt                  *time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                  *time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (MerchantRoleStartPendingConfig) TableName() string {
	return "merchant_role_start_pending_configs"
}

func (MerchantRoleStartPendingConfig) TableComment() string {
	return "商户岗位待开始服务超时配置表"
}
