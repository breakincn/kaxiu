package models

import "time"

type MultiServiceBooking struct {
	ID            uint       `json:"id" gorm:"primaryKey;comment:多人项目课程预约ID"`
	MerchantID    uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	ProjectID     uint       `json:"project_id" gorm:"index;comment:项目ID"`
	CardID        uint       `json:"card_id" gorm:"index;comment:卡片ID"`
	UserID        uint       `json:"user_id" gorm:"index;comment:用户ID"`
	SlotStartAt   *time.Time `json:"slot_start_at" gorm:"type:datetime(3);index;comment:场次开始时间"`
	SlotEndAt     *time.Time `json:"slot_end_at" gorm:"type:datetime(3);comment:场次结束时间"`
	Status        string     `json:"status" gorm:"size:20;not null;default:'booked';comment:状态（booked/canceled/attended/no_show）"`
	BookedAt      *time.Time `json:"booked_at" gorm:"type:datetime(3);comment:预约时间"`
	CanceledAt    *time.Time `json:"canceled_at" gorm:"type:datetime(3);comment:取消时间"`
	CancelReason  string     `json:"cancel_reason" gorm:"size:255;default:'';comment:取消原因"`
	CancelPenalty bool       `json:"cancel_penalty" gorm:"not null;default:false;comment:取消是否计作失约"`
	AttendedAt    *time.Time `json:"attended_at" gorm:"type:datetime(3);comment:到店/核销时间"`
	NoShowAt      *time.Time `json:"no_show_at" gorm:"type:datetime(3);comment:失约时间"`
	UsageID       *uint      `json:"usage_id" gorm:"index;comment:关联使用记录ID"`
	CreatedAt     *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt     *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (MultiServiceBooking) TableName() string {
	return "multi_service_bookings"
}

func (MultiServiceBooking) TableComment() string {
	return "多人项目课程预约表"
}

type MultiServicePenaltyLedger struct {
	ID                 uint       `json:"id" gorm:"primaryKey;comment:多人项目失约惩罚账本ID"`
	MerchantID         uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	ProjectID          uint       `json:"project_id" gorm:"index;comment:项目ID"`
	CardID             uint       `json:"card_id" gorm:"index;comment:卡片ID"`
	UserID             uint       `json:"user_id" gorm:"index;comment:用户ID"`
	BookingID          *uint      `json:"booking_id" gorm:"index;comment:关联课程预约ID"`
	PenaltyType        string     `json:"penalty_type" gorm:"size:30;not null;default:'';comment:惩罚类型（late_cancel/no_show/over_limit）"`
	CountsTowardNoShow bool       `json:"counts_toward_no_show" gorm:"not null;default:false;comment:是否计入半年失约累计"`
	ChargedTimes       int        `json:"charged_times" gorm:"not null;default:0;comment:扣减次数"`
	ChargedAmount      int        `json:"charged_amount" gorm:"not null;default:0;comment:扣减额度"`
	UsageID            *uint      `json:"usage_id" gorm:"index;comment:关联使用记录ID"`
	Remark             string     `json:"remark" gorm:"size:255;default:'';comment:备注"`
	PenaltyAt          *time.Time `json:"penalty_at" gorm:"type:datetime(3);index;comment:惩罚时间"`
	CreatedAt          *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (MultiServicePenaltyLedger) TableName() string {
	return "multi_service_penalty_ledgers"
}

func (MultiServicePenaltyLedger) TableComment() string {
	return "多人项目课程失约惩罚账本"
}
