package models

import "time"

type ServiceSessionExtendRequest struct {
	ID                     uint       `json:"id" gorm:"primaryKey;comment:服务加钟申请ID"`
	MerchantID             uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	UserID                 uint       `json:"user_id" gorm:"index;comment:用户ID"`
	CardID                 uint       `json:"card_id" gorm:"index;comment:卡片ID"`
	ServiceSessionID       uint       `json:"service_session_id" gorm:"index;comment:服务会话ID"`
	InitialUsageID         uint       `json:"initial_usage_id" gorm:"index;comment:首次核销使用记录ID"`
	ProjectID              uint       `json:"project_id" gorm:"index;comment:申请加钟项目ID"`
	Minutes                int        `json:"minutes" gorm:"not null;comment:申请加钟时长分钟"`
	BeforeRemainingSeconds int        `json:"before_remaining_seconds" gorm:"not null;default:0;comment:确认加钟前服务剩余秒数"`
	AfterRemainingSeconds  int        `json:"after_remaining_seconds" gorm:"not null;default:0;comment:确认加钟后服务剩余秒数"`
	Status                 string     `json:"status" gorm:"size:20;not null;default:'pending';index;comment:状态（pending/approved/rejected/canceled）"`
	RejectReason           string     `json:"reject_reason" gorm:"size:255;not null;default:'';comment:拒绝原因"`
	HandledByType          string     `json:"handled_by_type" gorm:"size:20;not null;default:'';comment:处理人类型"`
	HandledByID            *uint      `json:"handled_by_id" gorm:"index;comment:处理人ID"`
	HandledAt              *time.Time `json:"handled_at" gorm:"type:datetime(3);comment:处理时间"`
	CreatedAt              *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt              *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	Project        *MerchantProject `json:"project" gorm:"foreignKey:ProjectID"`
	ServiceSession *ServiceSession  `json:"service_session,omitempty" gorm:"foreignKey:ServiceSessionID"`
}

func (ServiceSessionExtendRequest) TableName() string {
	return "service_session_extend_requests"
}

func (ServiceSessionExtendRequest) TableComment() string {
	return "服务加钟申请表"
}
