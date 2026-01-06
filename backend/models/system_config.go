package models

import "time"

type SystemConfig struct {
	ID        uint       `json:"id" gorm:"primaryKey;comment:配置ID"`
	Key       string     `json:"key" gorm:"size:100;uniqueIndex;comment:配置项Key"`
	Value     string     `json:"value" gorm:"size:500;comment:配置项Value"`
	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (SystemConfig) TableName() string {
	return "system_configs"
}
