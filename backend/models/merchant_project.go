package models

import "time"

type MerchantProject struct {
	ID                uint       `json:"id" gorm:"primaryKey;comment:项目ID"`
	MerchantID        uint       `json:"merchant_id" gorm:"index;index:idx_merchant_active,priority:1;comment:商户ID"`
	Name              string     `json:"name" gorm:"size:100;not null;comment:项目名称"`
	Duration          int        `json:"duration" gorm:"not null;comment:服务时长（分钟）"`
	BookableOnline    bool       `json:"bookable_online" gorm:"not null;default:true;comment:是否参与公开预约"`
	ServiceGapMinutes int        `json:"service_gap_minutes" gorm:"not null;default:3;comment:服务间歇时间（分钟）"`
	StartDelaySeconds int        `json:"start_delay_seconds" gorm:"not null;default:60;comment:服务开始延迟时间（秒）"`
	Price             float64    `json:"price" gorm:"type:decimal(10,2);default:0.00;comment:项目价格（元）"`
	Description       string     `json:"description" gorm:"type:text;comment:项目描述"`
	IsActive          bool       `json:"is_active" gorm:"default:true;index:idx_merchant_active,priority:2;comment:是否启用"`
	SortOrder         int        `json:"sort_order" gorm:"default:0;index;comment:排序顺序"`
	CreatedAt         *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt         *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (MerchantProject) TableName() string {
	return "merchant_projects"
}

func (MerchantProject) TableComment() string {
	return "商户项目表"
}
