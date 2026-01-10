package models

import "time"

type ServiceSession struct {
	ID             uint   `json:"id" gorm:"primaryKey;comment:服务会话ID"`
	MerchantID     uint   `json:"merchant_id" gorm:"index;comment:商户ID"`
	UserID         uint   `json:"user_id" gorm:"index;comment:用户ID"`
	CardID         uint   `json:"card_id" gorm:"index;comment:卡片ID"`
	InitialUsageID uint   `json:"initial_usage_id" gorm:"index;comment:首次核销usage_id"`
	VerifyCode     string `json:"verify_code" gorm:"size:50;index;default:'';comment:首次核销码"`
	RoomID         *uint  `json:"room_id" gorm:"index;comment:房间ID"`
	TechnicianID   *uint  `json:"technician_id" gorm:"index;comment:工作人员ID"`

	Status string `json:"status" gorm:"size:30;default:created;comment:状态（created-已创建，room_selecting-选房中，room_locked-房间已锁定，staff_selecting-选人中，precheck_pending-待预结单，delay_pending-延迟中，serving-服务中，auto_finishing-待自动结单，finished-已完成，canceled-已取消）"`

	RoomSelectDeadlineAt *time.Time `json:"room_select_deadline_at" gorm:"type:datetime(3);comment:选房截止时间"`
	RoomLockedAt         *time.Time `json:"room_locked_at" gorm:"type:datetime(3);comment:房间锁定时间"`

	PrecheckAt             *time.Time `json:"precheck_at" gorm:"type:datetime(3);comment:预结单时间"`
	DelaySeconds           int        `json:"delay_seconds" gorm:"default:60;comment:延迟开计时秒数"`
	ScheduledStartAt       *time.Time `json:"scheduled_start_at" gorm:"type:datetime(3);comment:计划开始计时时间"`
	StartedAt              *time.Time `json:"started_at" gorm:"type:datetime(3);comment:开始计时时间"`
	DurationMinutes        int        `json:"duration_minutes" gorm:"default:0;comment:服务时长分钟（含加钟累计）"`
	ScheduledFinishAt      *time.Time `json:"scheduled_finish_at" gorm:"type:datetime(3);comment:计划结束时间"`
	FinishedAt             *time.Time `json:"finished_at" gorm:"type:datetime(3);comment:结单完成时间"`
	AutoFinishDelaySeconds int        `json:"auto_finish_delay_seconds" gorm:"default:60;comment:到点后自动结单延迟秒数"`

	AutoIdleAfterSeconds int `json:"auto_idle_after_seconds" gorm:"default:180;comment:结单后自动回空闲秒数"`

	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	Room       *Room       `json:"room" gorm:"foreignKey:RoomID"`
	Technician *Technician `json:"technician" gorm:"foreignKey:TechnicianID"`
}

func (ServiceSession) TableName() string {
	return "service_sessions"
}

func (ServiceSession) TableComment() string {
	return "服务会话表"
}
