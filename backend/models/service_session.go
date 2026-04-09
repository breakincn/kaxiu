package models

import "time"

type ServiceSession struct {
	ID             uint   `json:"id" gorm:"primaryKey;comment:服务会话ID"`
	MerchantID     uint   `json:"merchant_id" gorm:"index;comment:商户ID"`
	UserID         uint   `json:"user_id" gorm:"index;comment:用户ID"`
	CardID         uint   `json:"card_id" gorm:"index;comment:卡片ID"`
	ProjectID      *uint  `json:"project_id" gorm:"index;comment:项目ID（merchant_projects表主键，可为空）"`
	InitialUsageID uint   `json:"initial_usage_id" gorm:"index;comment:首次核销usage_id"`
	VerifyCode     string `json:"verify_code" gorm:"size:50;index;default:'';comment:首次核销码"`
	SessionMode    string `json:"session_mode" gorm:"column:session_mode;size:20;default:'';comment:会话模式"`
	SourceType     string `json:"source_type" gorm:"column:source_type;size:20;default:'walk_in';comment:会话来源（walk_in/appointment）"`
	SourceID       *uint  `json:"source_id" gorm:"column:source_id;index;comment:来源业务ID（如appointment_id）"`
	// 预约保护审计字段保留给现场拒派和历史兼容会话使用。
	OccupiesNextAppointment          bool       `json:"occupies_next_appointment" gorm:"column:occupies_next_appointment;default:false;comment:是否命中过预约保护冲突"`
	NextAppointmentID                *uint      `json:"next_appointment_id" gorm:"column:next_appointment_id;index;comment:受保护的下一预约ID"`
	PredictedAppointmentDelayMinutes int        `json:"predicted_appointment_delay_minutes" gorm:"column:predicted_appointment_delay_minutes;default:0;comment:预约到店后检测到的预计延迟分钟数"`
	PredictedReadyAt                 *time.Time `json:"predicted_ready_at" gorm:"column:predicted_ready_at;type:datetime(3);comment:预计可开始服务时间（用于延迟提示和兼容等待会话）"`
	RoomID                           *uint      `json:"room_id" gorm:"index;comment:房间ID"`
	TechnicianID                     *uint      `json:"technician_id" gorm:"index;comment:工作人员ID"`
	LastTechnicianID                 *uint      `json:"last_technician_id" gorm:"column:last_technician_id;index;comment:最后一次队列分配的工作人员ID（用于过号等待等保留展示）"`
	StartTimeoutCount                int        `json:"start_timeout_count" gorm:"column:start_timeout_count;default:0"`
	StartTimeoutLastAt               *time.Time `json:"start_timeout_last_at" gorm:"column:start_timeout_last_at;type:datetime"`
	StaffSelectCooldownUntil         *time.Time `json:"staff_select_cooldown_until" gorm:"column:staff_select_cooldown_until;type:datetime(3);comment:选客服无空闲时冷却截止时间"`
	StaffSelectEnteredAt             *time.Time `json:"staff_select_entered_at" gorm:"column:staff_select_entered_at;type:datetime(3);comment:用户进入选择客服页时间（以拉取可选客服列表为准）"`
	StartPendingTimeoutSeconds       int        `json:"start_pending_timeout_seconds" gorm:"column:start_pending_timeout_seconds;default:0;comment:待开始服务超时秒数（0表示使用系统默认）"`
	StartPendingRemainingSeconds     int        `json:"start_pending_remaining_seconds" gorm:"-"`

	Status string `json:"status" gorm:"size:30;default:created;comment:状态（created-已创建，room_selecting-选房中，room_locked-房间已锁定，staff_selecting-选人中，start_pending-待开始服务/待上号，delay_pending-延迟中，serving-服务中，auto_finishing-待自动结束，finished-已完成，canceled-已取消）"`

	RoomSelectDeadlineAt *time.Time `json:"room_select_deadline_at" gorm:"type:datetime(3);comment:选房截止时间"`
	RoomLockedAt         *time.Time `json:"room_locked_at" gorm:"type:datetime(3);comment:房间锁定时间"`

	StartConfirmedAt       *time.Time `json:"start_confirmed_at" gorm:"column:start_confirmed_at;type:datetime(3);comment:开始服务确认时间"`
	StartDelaySeconds      int        `json:"start_delay_seconds" gorm:"column:start_delay_seconds;default:60;comment:开始服务可延迟秒数"`
	ScheduledStartAt       *time.Time `json:"scheduled_start_at" gorm:"type:datetime(3);comment:计划开始计时时间"`
	StartedAt              *time.Time `json:"started_at" gorm:"type:datetime(3);comment:开始计时时间"`
	DurationMinutes        int        `json:"duration_minutes" gorm:"default:0;comment:服务时长分钟（含加钟累计）"`
	ScheduledFinishAt      *time.Time `json:"scheduled_finish_at" gorm:"type:datetime(3);comment:计划结束时间"`
	FinishedAt             *time.Time `json:"finished_at" gorm:"type:datetime(3);comment:服务结束完成时间"`
	AutoFinishDelaySeconds int        `json:"auto_finish_delay_seconds" gorm:"default:60;comment:到点后自动结束延迟秒数"`

	AutoIdleAfterSeconds int `json:"auto_idle_after_seconds" gorm:"default:180;comment:服务结束后自动回空闲秒数"`

	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	Room           *Room            `json:"room" gorm:"foreignKey:RoomID"`
	Technician     *Technician      `json:"technician" gorm:"foreignKey:TechnicianID"`
	LastTechnician *Technician      `json:"last_technician" gorm:"foreignKey:LastTechnicianID"`
	Project        *MerchantProject `json:"project" gorm:"foreignKey:ProjectID"`
	Card           *Card            `json:"card" gorm:"foreignKey:CardID"`
	InitialUsage   *Usage           `json:"initial_usage" gorm:"foreignKey:InitialUsageID"`
	Appointment    *Appointment     `json:"appointment" gorm:"foreignKey:SourceID"`
	Merchant       *Merchant        `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (ServiceSession) TableName() string {
	return "service_sessions"
}

func (ServiceSession) TableComment() string {
	return "服务会话表"
}
