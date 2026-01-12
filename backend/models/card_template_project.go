package models

import "time"

type CardTemplateProject struct {
	ID             uint       `json:"id" gorm:"primaryKey;comment:关联ID"`
	CardTemplateID uint       `json:"card_template_id" gorm:"not null;uniqueIndex:uk_card_template_project,priority:1;index;comment:卡片模板ID"`
	ProjectID      uint       `json:"project_id" gorm:"not null;uniqueIndex:uk_card_template_project,priority:2;index;comment:项目ID"`
	CreatedAt      *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	Project MerchantProject `json:"project" gorm:"foreignKey:ProjectID"`
}

func (CardTemplateProject) TableName() string {
	return "card_template_projects"
}

func (CardTemplateProject) TableComment() string {
	return "卡片模板项目关联表"
}
