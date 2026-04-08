package models

import "time"

type TechnicianAppointmentProject struct {
	ID           uint       `json:"id" gorm:"primaryKey;comment:主键ID"`
	MerchantID   uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	TechnicianID uint       `json:"technician_id" gorm:"index;uniqueIndex:uidx_tap_once;comment:客服ID"`
	ProjectID    uint       `json:"project_id" gorm:"index;uniqueIndex:uidx_tap_once;comment:项目ID"`
	CreatedAt    *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (TechnicianAppointmentProject) TableName() string {
	return "technician_appointment_projects"
}

func (TechnicianAppointmentProject) TableComment() string {
	return "客服可预约项目绑定表"
}
