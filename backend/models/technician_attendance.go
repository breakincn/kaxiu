package models

import "time"

type TechnicianAttendance struct {
	ID           uint       `json:"id" gorm:"primaryKey;comment:签到记录ID"`
	MerchantID   uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	TechnicianID uint       `json:"technician_id" gorm:"index;comment:工作人员ID"`
	CheckedInAt  *time.Time `json:"checked_in_at" gorm:"type:datetime(3);comment:签到时间"`
	CheckedOutAt *time.Time `json:"checked_out_at" gorm:"type:datetime(3);comment:下班时间"`
	Status       string     `json:"status" gorm:"size:20;default:idle;comment:状态（idle-空闲，busy-服务中，paused-暂停，rest-下班）"`
	CreatedAt    *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt    *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	Technician Technician `json:"technician" gorm:"foreignKey:TechnicianID"`
}

func (TechnicianAttendance) TableName() string {
	return "technician_attendances"
}

func (TechnicianAttendance) TableComment() string {
	return "工作人员签到与可服务状态表"
}
