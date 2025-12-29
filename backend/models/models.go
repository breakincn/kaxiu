package models

type User struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	Phone     string `json:"phone" gorm:"size:20;uniqueIndex"`
	Password  string `json:"-" gorm:"size:255"`
	Nickname  string `json:"nickname" gorm:"size:50"`
	CreatedAt string `json:"created_at" gorm:"autoCreateTime"`
}

type Merchant struct {
	ID                 uint   `json:"id" gorm:"primaryKey"`
	Name               string `json:"name" gorm:"size:100"`
	Type               string `json:"type" gorm:"size:50"`
	SupportAppointment bool   `json:"support_appointment" gorm:"default:false"`
	AvgServiceMinutes  int    `json:"avg_service_minutes" gorm:"default:30"`
	CreatedAt          string `json:"created_at" gorm:"autoCreateTime"`
}

type Card struct {
	ID             uint   `json:"id" gorm:"primaryKey"`
	UserID         uint   `json:"user_id" gorm:"index"`
	MerchantID     uint   `json:"merchant_id" gorm:"index"`
	CardNo         string `json:"card_no" gorm:"size:50"`
	CardType       string `json:"card_type" gorm:"size:100"`
	TotalTimes     int    `json:"total_times"`
	RemainTimes    int    `json:"remain_times"`
	UsedTimes      int    `json:"used_times"`
	RechargeAmount int    `json:"recharge_amount"`
	RechargeAt     string `json:"recharge_at"`
	LastUsedAt     string `json:"last_used_at"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	CreatedAt      string `json:"created_at" gorm:"autoCreateTime"`

	User     User     `json:"user" gorm:"foreignKey:UserID"`
	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

type Usage struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	CardID     uint   `json:"card_id" gorm:"index"`
	MerchantID uint   `json:"merchant_id" gorm:"index"`
	UsedTimes  int    `json:"used_times"`
	UsedAt     string `json:"used_at"`
	Status     string `json:"status" gorm:"size:20;default:success"`
	CreatedAt  string `json:"created_at" gorm:"autoCreateTime"`

	Card     Card     `json:"card" gorm:"foreignKey:CardID"`
	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

type Notice struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	MerchantID uint   `json:"merchant_id" gorm:"index"`
	Title      string `json:"title" gorm:"size:200"`
	Content    string `json:"content" gorm:"type:text"`
	IsPinned   bool   `json:"is_pinned" gorm:"default:false"`
	CreatedAt  string `json:"created_at"`

	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

type Appointment struct {
	ID              uint   `json:"id" gorm:"primaryKey"`
	MerchantID      uint   `json:"merchant_id" gorm:"index"`
	UserID          uint   `json:"user_id" gorm:"index"`
	AppointmentTime string `json:"appointment_time"`
	Status          string `json:"status" gorm:"size:20;default:pending"`
	CreatedAt       string `json:"created_at" gorm:"autoCreateTime"`

	User     User     `json:"user" gorm:"foreignKey:UserID"`
	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

type VerifyCode struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	CardID    uint   `json:"card_id" gorm:"index"`
	Code      string `json:"code" gorm:"size:50;uniqueIndex"`
	ExpireAt  int64  `json:"expire_at"`
	Used      bool   `json:"used" gorm:"default:false"`
	CreatedAt string `json:"created_at" gorm:"autoCreateTime"`
}
