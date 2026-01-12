package models

import "time"

type CardProject struct {
	ID        uint       `json:"id" gorm:"primaryKey;comment:关联ID"`
	CardID    uint       `json:"card_id" gorm:"not null;uniqueIndex:uk_card_project,priority:1;index;comment:卡片ID"`
	ProjectID uint       `json:"project_id" gorm:"not null;uniqueIndex:uk_card_project,priority:2;index;comment:项目ID"`
	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	Project MerchantProject `json:"project" gorm:"foreignKey:ProjectID"`
}

func (CardProject) TableName() string {
	return "card_projects"
}

func (CardProject) TableComment() string {
	return "卡片项目关联表"
}
