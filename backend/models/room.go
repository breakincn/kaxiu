package models

import "time"

type Room struct {
	ID         uint       `json:"id" gorm:"primaryKey;comment:房间ID"`
	MerchantID uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	Name       string     `json:"name" gorm:"size:50;comment:房间编号/名称"`
	IsActive   bool       `json:"is_active" gorm:"default:true;comment:是否启用"`
	CreatedAt  *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt  *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (Room) TableName() string {
	return "rooms"
}

func (Room) TableComment() string {
	return "商户房间配置表"
}
