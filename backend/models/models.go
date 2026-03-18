package models

import "time"

type User struct {
	ID        uint       `json:"id" gorm:"primaryKey;comment:用户ID"`
	Username  string     `json:"username" gorm:"size:50;uniqueIndex;comment:用户名"`
	Phone     *string    `json:"phone" gorm:"size:20;uniqueIndex;comment:手机号"`
	Password  string     `json:"-" gorm:"size:255;comment:登录密码（bcrypt加密）"`
	Nickname  string     `json:"nickname" gorm:"size:50;comment:用户昵称"`
	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (User) TableName() string {
	return "users"
}

func (User) TableComment() string {
	return "用户表"
}

type Merchant struct {
	ID                          uint       `json:"id" gorm:"primaryKey;comment:商户ID"`
	Name                        string     `json:"name" gorm:"size:100;comment:商户名称"`
	Phone                       string     `json:"phone" gorm:"size:20;uniqueIndex;comment:手机号"`
	Password                    string     `json:"-" gorm:"size:255;comment:登录密码（bcrypt加密）"`
	Type                        string     `json:"type" gorm:"size:50;comment:商户类型（如：理发、美容等）"`
	SupportAppointment          bool       `json:"support_appointment" gorm:"default:false;comment:是否支持预约（0-不支持，1-支持）"`
	SupportQueue                bool       `json:"support_queue" gorm:"default:false;comment:是否支持叫号/排队（0-不支持，1-支持）"`
	SupportProject              bool       `json:"support_project" gorm:"default:false;comment:是否开启项目服务（0-不开启，1-开启）"`
	SupportRoom                 bool       `json:"support_room" gorm:"default:false;comment:是否启用房间/教室功能（0-不启用，1-启用）"`
	SupportTechnicianCheckin    bool       `json:"support_technician_checkin" gorm:"default:false;comment:是否启用工作人员签到（0-不启用，1-启用）"`
	SupportHandCard             bool       `json:"support_hand_card" gorm:"default:false;comment:是否开启发手牌功能（0-不开启，1-开启）"`
	QueuePrefix                 string     `json:"queue_prefix" gorm:"size:20;default:'';comment:叫号前缀（如A、B）"`
	QueueStartNo                int        `json:"queue_start_no" gorm:"default:1;comment:叫号起始号码"`
	QueueMode                   string     `json:"queue_mode" gorm:"size:20;default:'auto';comment:叫号模式（auto-自动叫号，manual-人工叫号）"`
	QueueWindowTerm             string     `json:"queue_window_term" gorm:"size:20;default:'窗口';comment:叫号窗口自定义名词（默认：窗口，可自定义如：台号、工位等）"`
	QueueWaitingStartSeconds    int        `json:"queue_waiting_start_seconds" gorm:"default:180;comment:等待上号时间（秒）"`
	QueueTimeoutWaitingSeconds  int        `json:"queue_timeout_waiting_seconds" gorm:"default:900;comment:超时过号等待时间（秒）"`
	QueuePaused                 bool       `json:"queue_paused" gorm:"default:false;comment:叫号是否全局暂停（0-正常，1-暂停）"`
	QueueEndedAt                *time.Time `json:"queue_ended_at" gorm:"type:datetime(3);comment:结束叫号时间（用于打烊后自动收尾）"`
	SupportDirectSale           bool       `json:"support_direct_sale" gorm:"default:false;comment:是否支持直购售卡（0-不支持，1-支持）"`
	SupportCustomerService      bool       `json:"support_customer_service" gorm:"default:false;comment:是否开启客服账号设置功能（0-不开启，1-开启）"`
	SupportCustomerServiceMode  bool       `json:"support_customer_service_mode" gorm:"default:false;comment:是否开启客服选择模式（0-不开启，1-开启，需先开启SupportCustomerService）"`
	SupportMultiCustomerService bool       `json:"support_multi_customer_service" gorm:"default:false;comment:是否开启多个客服（多窗口叫号）"`
	SupportOrderComplete        bool       `json:"support_order_complete" gorm:"default:false;comment:是否开启服务结束功能（0-不开启，1-开启）"`
	StartDelaySeconds           int        `json:"start_delay_seconds" gorm:"default:60;comment:核销后开始服务延迟秒数（未开启客服但开启服务结束功能时使用）"`
	// 预约保护参数：confirmed 预约会占用未来产能，客服模式现场派单/预约选人都复用这些阈值。
	AppointmentReserveBufferMinutes                    int    `json:"appointment_reserve_buffer_minutes" gorm:"default:10;comment:预约前保留缓冲分钟数"`
	AppointmentGraceWindowMinutes                      int    `json:"appointment_grace_window_minutes" gorm:"default:15;comment:预约后到店宽限分钟数"`
	AppointmentMaxWaitMinutes                          int    `json:"appointment_max_wait_minutes" gorm:"default:15;comment:预约客户最大可承诺等待分钟数"`
	AppointmentPredictionBufferMinute                  int    `json:"appointment_prediction_buffer_minutes" gorm:"column:appointment_prediction_buffer_minutes;default:5;comment:预约保护预测缓冲分钟数"`
	AppointmentSlotGranularityMinutes                  int    `json:"appointment_slot_granularity_minutes" gorm:"default:15;comment:预约时段展示粒度分钟数"`
	AppointmentRescheduleSameOrNextDayThresholdMinutes int    `json:"appointment_reschedule_same_or_next_day_threshold_minutes" gorm:"default:180;comment:昨天预约可改签到今天或明天的剩余分钟阈值"`
	AppointmentRescheduleNextDayOnlyThresholdMinutes   int    `json:"appointment_reschedule_next_day_only_threshold_minutes" gorm:"default:90;comment:昨天预约仅可改签到明天的剩余分钟阈值"`
	TechnicianAlias                                    string `json:"technician_alias" gorm:"size:20;default:'技师';comment:技师自定义称谓（如：小二、服务员等）"`
	StartTerm                                          string `json:"start_term" gorm:"size:20;default:'';comment:开始服务显示名词（可为空）"`
	FinishTerm                                         string `json:"finish_term" gorm:"size:20;default:'';comment:结束服务显示名词（可为空）"`
	// 手牌设置
	HandCardPrefix  string `json:"hand_card_prefix" gorm:"size:20;default:'H';comment:手牌前缀"`
	HandCardStartNo int    `json:"hand_card_start_no" gorm:"default:1;comment:手牌起始号码"`
	HandCardEndNo   int    `json:"hand_card_end_no" gorm:"default:100;comment:手牌结束号码"`
	// 房间号牌设置
	RoomNumberCardPrefix  string `json:"room_number_card_prefix" gorm:"size:20;default:'R';comment:房间号牌前缀"`
	RoomNumberCardStartNo int    `json:"room_number_card_start_no" gorm:"default:1;comment:房间号牌起始号码"`
	RoomNumberCardEndNo   int    `json:"room_number_card_end_no" gorm:"default:50;comment:房间号牌结束号码"`
	// 营业时间
	MorningStart   string `json:"morning_start" gorm:"size:10;default:'';comment:上午营业开始时间（格式：HH:MM）"`
	MorningEnd     string `json:"morning_end" gorm:"size:10;default:'';comment:上午营业结束时间（格式：HH:MM）"`
	AfternoonStart string `json:"afternoon_start" gorm:"size:10;default:'';comment:下午营业开始时间（格式：HH:MM）"`
	AfternoonEnd   string `json:"afternoon_end" gorm:"size:10;default:'';comment:下午营业结束时间（格式：HH:MM）"`
	EveningStart   string `json:"evening_start" gorm:"size:10;default:'';comment:晚上营业开始时间（格式：HH:MM）"`
	EveningEnd     string `json:"evening_end" gorm:"size:10;default:'';comment:晚上营业结束时间（格式：HH:MM）"`
	AllDayStart    string `json:"all_day_start" gorm:"size:10;default:'';comment:全天营业开始时间（格式：HH:MM）"`
	AllDayEnd      string `json:"all_day_end" gorm:"size:10;default:'';comment:全天营业结束时间（格式：HH:MM）"`
	// 地址信息
	Province string `json:"province" gorm:"size:50;default:'';comment:省份"`
	City     string `json:"city" gorm:"size:50;default:'';comment:城市"`
	District string `json:"district" gorm:"size:50;default:'';comment:区县"`
	Address  string `json:"address" gorm:"size:200;default:'';comment:详细地址（街道门牌号）"`
	// 营业状态
	IsOpen    bool       `json:"is_open" gorm:"default:true;comment:营业状态（0-打烊，1-营业中）"`
	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (Merchant) TableName() string {
	return "merchants"
}

func (Merchant) TableComment() string {
	return "商户表"
}

type Card struct {
	ID             uint       `json:"id" gorm:"primaryKey;comment:卡片ID"`
	UserID         uint       `json:"user_id" gorm:"index;comment:用户ID（外键关联users表）"`
	MerchantID     uint       `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	CardNo         string     `json:"card_no" gorm:"size:50;comment:卡号"`
	CardType       string     `json:"card_type" gorm:"size:100;comment:卡片类型（如：洗剪吹10次卡）"`
	TotalTimes     int        `json:"total_times" gorm:"comment:总次数"`
	RemainTimes    int        `json:"remain_times" gorm:"comment:剩余次数"`
	UsedTimes      int        `json:"used_times" gorm:"comment:已使用次数"`
	RechargeAmount int        `json:"recharge_amount" gorm:"comment:充值金额（单位：元）"`
	RechargeAt     *time.Time `json:"recharge_at" gorm:"type:date;comment:充值时间/开卡时间"`
	LastUsedAt     *time.Time `json:"last_used_at" gorm:"type:datetime(3);comment:最后使用时间"`
	StartDate      *time.Time `json:"start_date" gorm:"type:date;comment:有效期开始日期"`
	EndDate        *time.Time `json:"end_date" gorm:"type:date;comment:有效期结束日期"`

	Locked       bool       `json:"locked" gorm:"default:false;comment:卡片是否锁定（手牌未归还等）"`
	LockedReason string     `json:"locked_reason" gorm:"size:255;default:'';comment:锁卡原因"`
	LockedAt     *time.Time `json:"locked_at" gorm:"type:datetime(3);comment:锁卡时间"`
	LockedBy     *uint      `json:"locked_by" gorm:"comment:锁卡操作人（0/NULL 表示系统）"`

	UnlockedAt     *time.Time `json:"unlocked_at" gorm:"type:datetime(3);comment:解锁时间"`
	UnlockedBy     *uint      `json:"unlocked_by" gorm:"comment:解锁操作人（0/NULL 表示系统）"`
	UnlockedReason string     `json:"unlocked_reason" gorm:"size:255;default:'';comment:解锁原因"`
	CreatedAt      *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	StartPendingTimeoutSeconds int64 `json:"start_pending_timeout_seconds" gorm:"-"`
	StartScanTimeoutSeconds    int64 `json:"start_scan_timeout_seconds" gorm:"-"`

	User     User              `json:"user" gorm:"foreignKey:UserID"`
	Merchant Merchant          `json:"merchant" gorm:"foreignKey:MerchantID"`
	Projects []MerchantProject `json:"projects" gorm:"-"`
}

func (Card) TableName() string {
	return "cards"
}

func (Card) TableComment() string {
	return "用户会员卡表"
}

type Usage struct {
	ID                 uint       `json:"id" gorm:"primaryKey;comment:记录ID"`
	CardID             uint       `json:"card_id" gorm:"index;comment:卡片ID（外键关联cards表）"`
	MerchantID         uint       `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	ProjectID          *uint      `json:"project_id" gorm:"index;comment:项目ID（merchant_projects表主键，可为空）"`
	UsedTimes          int        `json:"used_times" gorm:"comment:本次核销次数"`
	UsedAt             *time.Time `json:"used_at" gorm:"type:datetime(3);comment:使用时间"`
	VerifyCode         string     `json:"verify_code" gorm:"size:50;index;default:'';comment:核销码（用于服务结束/追溯）"`
	VerifyCodeExpireAt int64      `json:"verify_code_expire_at" gorm:"index;comment:核销码过期时间（Unix时间戳）"`
	TechnicianID       *uint      `json:"technician_id" gorm:"index;comment:服务人员技师ID（technicians表主键，可为空）"`
	FinishedAt         *time.Time `json:"finished_at" gorm:"type:datetime(3);comment:服务完成/结束时间"`
	Status             string     `json:"status" gorm:"size:20;default:success;comment:状态（in_progress-进行中，success-完成，failed-失败）"`

	QueueNo       int        `json:"queue_no" gorm:"-"`
	QueueCalledAt *time.Time `json:"queue_called_at" gorm:"-"`
	QueueKind     string     `json:"queue_kind" gorm:"-"`

	HandCardNo         *string    `json:"hand_card_no" gorm:"size:20;index;comment:手牌号（商户核销后输入绑定）"`
	HandCardAssignedAt *time.Time `json:"hand_card_assigned_at" gorm:"type:datetime(3);comment:手牌分配时间"`
	HandCardReturnedAt *time.Time `json:"hand_card_returned_at" gorm:"type:datetime(3);comment:手牌归还时间"`
	CreatedAt          *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	ServiceSessionID                           *uint       `json:"service_session_id" gorm:"-"`
	ServiceSessionStatus                       string      `json:"service_session_status" gorm:"-"`
	ServiceSessionStartConfirmedAt             *time.Time  `json:"service_session_start_confirmed_at" gorm:"-"`
	ServiceSessionScheduledStartAt             *time.Time  `json:"service_session_scheduled_start_at" gorm:"-"`
	ServiceSessionStartedAt                    *time.Time  `json:"service_session_started_at" gorm:"-"`
	ServiceSessionScheduledFinishAt            *time.Time  `json:"service_session_scheduled_finish_at" gorm:"-"`
	ServiceSessionFinishedAt                   *time.Time  `json:"service_session_finished_at" gorm:"-"`
	ServiceSessionDurationMinutes              int         `json:"service_session_duration_minutes" gorm:"-"`
	ServiceSessionUpdatedAt                    *time.Time  `json:"service_session_updated_at" gorm:"-"`
	ServiceSessionStartPendingTimeoutSeconds   int         `json:"service_session_start_pending_timeout_seconds" gorm:"-"`
	ServiceSessionStartPendingRemainingSeconds int         `json:"service_session_start_pending_remaining_seconds" gorm:"-"`
	RoomSelectDeadlineAt                       *time.Time  `json:"room_select_deadline_at" gorm:"-"`
	RoomLockedAt                               *time.Time  `json:"room_locked_at" gorm:"-"`
	StaffSelectCooldownUntil                   *time.Time  `json:"staff_select_cooldown_until" gorm:"-"`
	StaffSelectEnteredAt                       *time.Time  `json:"staff_select_entered_at" gorm:"-"`
	ServiceRoom                                *Room       `json:"service_room" gorm:"-"`
	ServiceTechnician                          *Technician `json:"service_technician" gorm:"-"`
	ServiceTechnicianAvailable                 bool        `json:"service_technician_available" gorm:"-"`
	ServiceTechnicianUnavailableReason         string      `json:"service_technician_unavailable_reason" gorm:"-"`
	StartTimeoutCount                          int         `json:"start_timeout_count" gorm:"-"`
	RevokeDeadlineAt                           *time.Time  `json:"revoke_deadline_at" gorm:"-"`
	CanRevoke                                  bool        `json:"can_revoke" gorm:"-"`

	Card       Card             `json:"card" gorm:"foreignKey:CardID"`
	Merchant   Merchant         `json:"merchant" gorm:"foreignKey:MerchantID"`
	Project    *MerchantProject `json:"project" gorm:"foreignKey:ProjectID"`
	Technician *Technician      `json:"technician" gorm:"foreignKey:TechnicianID"`
}

func (Usage) TableName() string {
	return "usages"
}

func (Usage) TableComment() string {
	return "卡片使用记录表"
}

type Notice struct {
	ID         uint       `json:"id" gorm:"primaryKey;comment:通知ID"`
	MerchantID uint       `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	Title      string     `json:"title" gorm:"size:200;comment:通知标题"`
	Content    string     `json:"content" gorm:"type:text;comment:通知内容"`
	IsPinned   bool       `json:"is_pinned" gorm:"default:false;comment:是否置顶（0-否，1-是）"`
	CreatedAt  *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (Notice) TableName() string {
	return "notices"
}

func (Notice) TableComment() string {
	return "商户通知表"
}

type Appointment struct {
	ID                      uint       `json:"id" gorm:"primaryKey;comment:预约ID"`
	CardID                  uint       `json:"card_id" gorm:"index;comment:卡片ID（外键关联cards表）"`
	MerchantID              uint       `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	UserID                  uint       `json:"user_id" gorm:"index;comment:用户ID（外键关联users表）"`
	ProjectID               *uint      `json:"project_id" gorm:"index;comment:项目ID（merchant_projects表主键，可为空）"`
	TechnicianID            *uint      `json:"technician_id" gorm:"index;comment:技师ID（technicians表主键，可为空）"`
	AppointmentTime         *time.Time `json:"appointment_time" gorm:"type:datetime(3);comment:预约时间"`
	Status                  string     `json:"status" gorm:"size:20;default:pending;comment:预约状态（pending-待确认，confirmed-已确认，arrived-已到店，completed-已完成，canceled-已取消，failed-失败，no_show-失约；历史finished按completed兼容读取）"`
	ConfirmedAt             *time.Time `json:"confirmed_at" gorm:"type:datetime(3);comment:确认时间"`
	ArrivedAt               *time.Time `json:"arrived_at" gorm:"type:datetime(3);comment:到店核销时间"`
	CompletedAt             *time.Time `json:"completed_at" gorm:"type:datetime(3);comment:完成时间"`
	NoShowAt                *time.Time `json:"no_show_at" gorm:"type:datetime(3);comment:失约时间"`
	ServiceSessionID        *uint      `json:"service_session_id" gorm:"index;comment:关联服务会话ID"`
	UsageID                 *uint      `json:"usage_id" gorm:"index;comment:关联核销记录ID"`
	PredictedWaitMinutes    int        `json:"predicted_wait_minutes" gorm:"default:0;comment:预约预计等待分钟数（风险预约/到店等待时回写）"`
	ResolutionNote          string     `json:"resolution_note" gorm:"size:255;default:'';comment:改签/补偿/人工处理备注"`
	ClosedReason            string     `json:"closed_reason" gorm:"size:50;default:'';comment:关闭原因（canceled/rescheduled/no_show/completed等）"`
	ClosedByType            string     `json:"closed_by_type" gorm:"size:20;default:'';comment:关闭操作人类型（merchant/staff/system/user）"`
	ClosedByID              *uint      `json:"closed_by_id" gorm:"index;comment:关闭操作人ID"`
	RescheduleReason        string     `json:"reschedule_reason" gorm:"size:255;default:'';comment:改签原因"`
	ReplacedByAppointmentID *uint      `json:"replaced_by_appointment_id" gorm:"index;comment:本预约被哪条新预约替代"`
	ReplacesAppointmentID   *uint      `json:"replaces_appointment_id" gorm:"index;comment:本预约替代了哪条旧预约"`
	CanceledAt              *time.Time `json:"canceled_at" gorm:"type:datetime(3);comment:取消时间"`
	FailedAt                *time.Time `json:"failed_at" gorm:"type:datetime(3);comment:失败时间（自动分配失败时填充）"`
	FailedReason            string     `json:"failed_reason" gorm:"size:255;default:'';comment:失败原因（自动分配失败时说明）"`
	CreatedAt               *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	User               User                           `json:"user" gorm:"foreignKey:UserID"`
	Card               Card                           `json:"card" gorm:"foreignKey:CardID"`
	Merchant           Merchant                       `json:"merchant" gorm:"foreignKey:MerchantID"`
	Project            *MerchantProject               `json:"project" gorm:"foreignKey:ProjectID"`
	Technician         *Technician                    `json:"technician" gorm:"foreignKey:TechnicianID"`
	Compensations      []AppointmentCompensation      `json:"compensations" gorm:"foreignKey:AppointmentID"`
	RescheduleRequests []AppointmentRescheduleRequest `json:"reschedule_requests" gorm:"foreignKey:AppointmentID"`
}

func (Appointment) TableName() string {
	return "appointments"
}

func (Appointment) TableComment() string {
	return "用户预约排队表"
}

type AppointmentCompensation struct {
	ID               uint       `json:"id" gorm:"primaryKey;comment:预约补偿ID"`
	AppointmentID    uint       `json:"appointment_id" gorm:"index;comment:关联预约ID"`
	MerchantID       uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	UserID           uint       `json:"user_id" gorm:"index;comment:用户ID"`
	CardID           uint       `json:"card_id" gorm:"index;comment:卡片ID"`
	ServiceSessionID *uint      `json:"service_session_id" gorm:"index;comment:关联服务会话ID（补时使用）"`
	Type             string     `json:"type" gorm:"size:30;comment:补偿类型（extra_times/extend_minutes/discount_note/other_note）"`
	Value            int        `json:"value" gorm:"default:0;comment:补偿数值（次数/分钟/折扣描述值）"`
	Status           string     `json:"status" gorm:"size:20;default:pending;comment:状态（pending/applied/canceled）"`
	Reason           string     `json:"reason" gorm:"size:255;default:'';comment:补偿原因"`
	Remark           string     `json:"remark" gorm:"size:255;default:'';comment:补偿备注"`
	CreatedByType    string     `json:"created_by_type" gorm:"size:20;default:'';comment:创建人类型（merchant/staff/system）"`
	CreatedByID      *uint      `json:"created_by_id" gorm:"index;comment:创建人ID"`
	AppliedAt        *time.Time `json:"applied_at" gorm:"type:datetime(3);comment:补偿执行时间"`
	CreatedAt        *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	Appointment Appointment `json:"appointment" gorm:"foreignKey:AppointmentID"`
}

func (AppointmentCompensation) TableName() string {
	return "appointment_compensations"
}

type AppointmentProtectionBlock struct {
	ID                           uint       `json:"id" gorm:"primaryKey;comment:主键ID"`
	MerchantID                   uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	AppointmentID                uint       `json:"appointment_id" gorm:"index;comment:被保护的预约ID"`
	TechnicianID                 uint       `json:"technician_id" gorm:"index;comment:被保护预约绑定的客服ID"`
	WalkInServiceSessionID       *uint      `json:"walk_in_service_session_id" gorm:"index;comment:被拒绝分配的现场服务会话ID"`
	BlockedReason                string     `json:"blocked_reason" gorm:"size:255;default:'';comment:拒派原因说明"`
	PredictedReservedWaitMinutes int        `json:"predicted_reserved_wait_minutes" gorm:"default:0;comment:若继续派单将导致预约等待的预计分钟数"`
	AlternativeWaitMinutes       int        `json:"alternative_wait_minutes" gorm:"default:0;comment:改派其他客服的预计等待分钟数"`
	DecisionMode                 string     `json:"decision_mode" gorm:"size:50;default:'';comment:拒派决策模式（max_wait_protection/alternative_preferred）"`
	BlockedAt                    *time.Time `json:"blocked_at" gorm:"type:datetime(3);comment:拒派发生时间"`
	CreatedAt                    *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (AppointmentProtectionBlock) TableName() string {
	return "appointment_protection_blocks"
}

func (AppointmentCompensation) TableComment() string {
	return "预约补偿记录表"
}

type AppointmentRescheduleRequest struct {
	ID                     uint       `json:"id" gorm:"primaryKey;comment:预约改签提议ID"`
	AppointmentID          uint       `json:"appointment_id" gorm:"index;comment:关联预约ID"`
	MerchantID             uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	UserID                 uint       `json:"user_id" gorm:"index;comment:用户ID"`
	CurrentProjectID       *uint      `json:"current_project_id" gorm:"index;comment:原项目ID"`
	CurrentTechnicianID    *uint      `json:"current_technician_id" gorm:"index;comment:原客服ID"`
	CurrentAppointmentTime *time.Time `json:"current_appointment_time" gorm:"column:current_appointment_time;type:datetime(3);comment:原预约时间"`
	NewProjectID           *uint      `json:"new_project_id" gorm:"index;comment:新项目ID"`
	NewTechnicianID        *uint      `json:"new_technician_id" gorm:"index;comment:新客服ID"`
	NewAppointmentTime     *time.Time `json:"new_appointment_time" gorm:"type:datetime(3);comment:提议的新预约时间"`
	Status                 string     `json:"status" gorm:"size:30;default:pending_user;comment:状态（pending_user/pending_merchant/accepted/rejected/canceled）"`
	Reason                 string     `json:"reason" gorm:"size:255;default:'';comment:改签原因"`
	ProposedByType         string     `json:"proposed_by_type" gorm:"size:20;default:'';comment:提议发起方类型（merchant/staff/user）"`
	ProposedByID           *uint      `json:"proposed_by_id" gorm:"index;comment:提议发起方ID"`
	ConfirmedByType        string     `json:"confirmed_by_type" gorm:"size:20;default:'';comment:确认/拒绝方类型（merchant/staff/user）"`
	ConfirmedByID          *uint      `json:"confirmed_by_id" gorm:"index;comment:确认/拒绝方ID"`
	ConfirmedAt            *time.Time `json:"confirmed_at" gorm:"type:datetime(3);comment:确认/拒绝时间"`
	ResultAppointmentID    *uint      `json:"result_appointment_id" gorm:"index;comment:接受后生成的新预约ID"`
	CreatedAt              *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	CurrentProject    *MerchantProject `json:"current_project,omitempty" gorm:"foreignKey:CurrentProjectID"`
	NewProject        *MerchantProject `json:"new_project,omitempty" gorm:"foreignKey:NewProjectID"`
	CurrentTechnician *Technician      `json:"current_technician,omitempty" gorm:"foreignKey:CurrentTechnicianID"`
	NewTechnician     *Technician      `json:"new_technician,omitempty" gorm:"foreignKey:NewTechnicianID"`
}

func (AppointmentRescheduleRequest) TableName() string {
	return "appointment_reschedule_requests"
}

func (AppointmentRescheduleRequest) TableComment() string {
	return "预约改签提议表"
}

type VerifyCode struct {
	ID        uint       `json:"id" gorm:"primaryKey;comment:核销码ID"`
	CardID    uint       `json:"card_id" gorm:"index;comment:卡片ID（外键关联cards表）"`
	ProjectID *uint      `json:"project_id" gorm:"index;comment:项目ID（merchant_projects表主键，可为空）"`
	Code      string     `json:"code" gorm:"size:50;uniqueIndex;comment:核销码"`
	ExpireAt  int64      `json:"expire_at" gorm:"comment:过期时间（Unix时间戳）"`
	Used      bool       `json:"used" gorm:"default:false;comment:是否已使用（0-未使用，1-已使用）"`
	UsedAt    *time.Time `json:"used_at" gorm:"type:datetime(3);comment:使用时间"`
	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (VerifyCode) TableName() string {
	return "verify_codes"
}

func (VerifyCode) TableComment() string {
	return "核销码表"
}

type SMSCode struct {
	ID        uint       `json:"id" gorm:"primaryKey;comment:短信验证码ID"`
	Phone     string     `json:"phone" gorm:"size:20;index;comment:手机号"`
	Purpose   string     `json:"purpose" gorm:"size:50;index;comment:用途"`
	Code      string     `json:"-" gorm:"size:10;comment:验证码"`
	ExpiresAt int64      `json:"expires_at" gorm:"index;comment:过期时间（Unix时间戳）"`
	Used      bool       `json:"used" gorm:"default:false;comment:是否已使用（0-未使用，1-已使用）"`
	UsedAt    *time.Time `json:"used_at" gorm:"type:datetime(3);comment:使用时间"`
	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (SMSCode) TableName() string {
	return "sms_codes"
}

type InviteCode struct {
	ID               uint       `json:"id" gorm:"primaryKey;comment:邀请码ID"`
	Code             string     `json:"code" gorm:"size:50;uniqueIndex;comment:邀请码"`
	Used             bool       `json:"used" gorm:"default:false;comment:是否已使用（0-未使用，1-已使用）"`
	UsedByMerchantID *uint      `json:"used_by_merchant_id" gorm:"index;comment:使用该邀请码注册的商户ID"`
	UsedAt           *time.Time `json:"used_at" gorm:"type:datetime(3);comment:使用时间"`
	CreatedAt        *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (InviteCode) TableName() string {
	return "invite_codes"
}
