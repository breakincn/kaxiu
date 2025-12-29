package models

type User struct {
	ID        uint   `json:"id" gorm:"primaryKey;comment:用户ID"`
	Phone     string `json:"phone" gorm:"size:20;uniqueIndex;comment:手机号"`
	Password  string `json:"-" gorm:"size:255;comment:登录密码（bcrypt加密）"`
	Nickname  string `json:"nickname" gorm:"size:50;comment:用户昵称"`
	CreatedAt string `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (User) TableName() string {
	return "users"
}

func (User) TableComment() string {
	return "用户表"
}

type Merchant struct {
	ID                 uint   `json:"id" gorm:"primaryKey;comment:商户ID"`
	Name               string `json:"name" gorm:"size:100;comment:商户名称"`
	Type               string `json:"type" gorm:"size:50;comment:商户类型（如：理发、美容等）"`
	SupportAppointment bool   `json:"support_appointment" gorm:"default:false;comment:是否支持预约（0-不支持，1-支持）"`
	AvgServiceMinutes  int    `json:"avg_service_minutes" gorm:"default:30;comment:平均服务时长（分钟）"`
	CreatedAt          string `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (Merchant) TableName() string {
	return "merchants"
}

func (Merchant) TableComment() string {
	return "商户表"
}

type Card struct {
	ID             uint   `json:"id" gorm:"primaryKey;comment:卡片ID"`
	UserID         uint   `json:"user_id" gorm:"index;comment:用户ID（外键关联users表）"`
	MerchantID     uint   `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	CardNo         string `json:"card_no" gorm:"size:50;comment:卡号"`
	CardType       string `json:"card_type" gorm:"size:100;comment:卡片类型（如：洗剪吹10次卡）"`
	TotalTimes     int    `json:"total_times" gorm:"comment:总次数"`
	RemainTimes    int    `json:"remain_times" gorm:"comment:剩余次数"`
	UsedTimes      int    `json:"used_times" gorm:"comment:已使用次数"`
	RechargeAmount int    `json:"recharge_amount" gorm:"comment:充值金额（单位：元）"`
	RechargeAt     string `json:"recharge_at" gorm:"comment:充值时间/开卡时间"`
	LastUsedAt     string `json:"last_used_at" gorm:"comment:最后使用时间"`
	StartDate      string `json:"start_date" gorm:"comment:有效期开始日期"`
	EndDate        string `json:"end_date" gorm:"comment:有效期结束日期"`
	CreatedAt      string `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	User     User     `json:"user" gorm:"foreignKey:UserID"`
	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (Card) TableName() string {
	return "cards"
}

func (Card) TableComment() string {
	return "用户会员卡表"
}

type Usage struct {
	ID         uint   `json:"id" gorm:"primaryKey;comment:记录ID"`
	CardID     uint   `json:"card_id" gorm:"index;comment:卡片ID（外键关联cards表）"`
	MerchantID uint   `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	UsedTimes  int    `json:"used_times" gorm:"comment:本次核销次数"`
	UsedAt     string `json:"used_at" gorm:"comment:使用时间"`
	Status     string `json:"status" gorm:"size:20;default:success;comment:状态（success-成功，failed-失败）"`
	CreatedAt  string `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	Card     Card     `json:"card" gorm:"foreignKey:CardID"`
	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (Usage) TableName() string {
	return "usages"
}

func (Usage) TableComment() string {
	return "卡片使用记录表"
}

type Notice struct {
	ID         uint   `json:"id" gorm:"primaryKey;comment:通知ID"`
	MerchantID uint   `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	Title      string `json:"title" gorm:"size:200;comment:通知标题"`
	Content    string `json:"content" gorm:"type:text;comment:通知内容"`
	IsPinned   bool   `json:"is_pinned" gorm:"default:false;comment:是否置顶（0-否，1-是）"`
	CreatedAt  string `json:"created_at" gorm:"comment:创建时间"`

	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (Notice) TableName() string {
	return "notices"
}

func (Notice) TableComment() string {
	return "商户通知表"
}

type Appointment struct {
	ID              uint   `json:"id" gorm:"primaryKey;comment:预约ID"`
	MerchantID      uint   `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	UserID          uint   `json:"user_id" gorm:"index;comment:用户ID（外键关联users表）"`
	AppointmentTime string `json:"appointment_time" gorm:"comment:预约时间"`
	Status          string `json:"status" gorm:"size:20;default:pending;comment:预约状态（pending-待确认，confirmed-已确认/排队中，finished-已完成，canceled-已取消）"`
	CreatedAt       string `json:"created_at" gorm:"comment:创建时间"`

	User     User     `json:"user" gorm:"foreignKey:UserID"`
	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (Appointment) TableName() string {
	return "appointments"
}

func (Appointment) TableComment() string {
	return "用户预约排队表"
}

type VerifyCode struct {
	ID        uint   `json:"id" gorm:"primaryKey;comment:核销码ID"`
	CardID    uint   `json:"card_id" gorm:"index;comment:卡片ID（外键关联cards表）"`
	Code      string `json:"code" gorm:"size:50;uniqueIndex;comment:核销码"`
	ExpireAt  int64  `json:"expire_at" gorm:"comment:过期时间（Unix时间戳）"`
	Used      bool   `json:"used" gorm:"default:false;comment:是否已使用（0-未使用，1-已使用）"`
	CreatedAt string `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (VerifyCode) TableName() string {
	return "verify_codes"
}

func (VerifyCode) TableComment() string {
	return "核销码表"
}
