package models

import "time"

type Permission struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Key         string     `json:"key" gorm:"size:80;uniqueIndex;not null"`
	Name        string     `json:"name" gorm:"size:80;not null"`
	Group       string     `json:"group" gorm:"size:80"`
	Description string     `json:"description" gorm:"size:255"`
	Sort        int        `json:"sort" gorm:"default:0"`
	CreatedAt   *time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   *time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Permission) TableName() string {
	return "permissions"
}
