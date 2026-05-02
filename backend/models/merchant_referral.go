package models

import "time"

// MerchantReferralProfile 推广用户档案
type MerchantReferralProfile struct {
	ID                     uint       `json:"id" gorm:"primaryKey;comment:档案ID"`
	UserID                  uint       `json:"user_id" gorm:"uniqueIndex;comment:推广用户ID"`
	PromotionCode           string     `json:"promotion_code" gorm:"size:12;uniqueIndex;comment:推广码"`
	DefaultCommissionRateBP int        `json:"default_commission_rate_bp" gorm:"default:5000;comment:默认分成比例基点（10000=100%）"`
	CreatedAt               *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt               *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (MerchantReferralProfile) TableName() string {
	return "merchant_referral_profiles"
}

func (MerchantReferralProfile) TableComment() string {
	return "用户推广商户档案表"
}

// MerchantReferral 商户邀请关系
type MerchantReferral struct {
	ID                   uint       `json:"id" gorm:"primaryKey;comment:关系ID"`
	ReferrerUserID       uint       `json:"referrer_user_id" gorm:"index;comment:推广用户ID"`
	MerchantID           uint       `json:"merchant_id" gorm:"uniqueIndex;comment:商户ID"`
	PromotionCode        string     `json:"promotion_code" gorm:"size:12;index;comment:推广码"`
	RegisteredAt         *time.Time `json:"registered_at" gorm:"comment:商户注册时间"`
	FirstPaidAt          *time.Time `json:"first_paid_at" gorm:"comment:首次付费时间"`
	CommissionExpiresAt  *time.Time `json:"commission_expires_at" gorm:"comment:分成截止时间"`
	DefaultCommissionRateBP int     `json:"default_commission_rate_bp" gorm:"default:5000;comment:默认分成比例基点（10000=100%）"`
	CreatedAt            *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt            *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (MerchantReferral) TableName() string {
	return "merchant_referrals"
}

func (MerchantReferral) TableComment() string {
	return "用户推广商户邀请关系表"
}

// MerchantReferralPaymentLedger 商户付费分成台账
type MerchantReferralPaymentLedger struct {
	ID                     uint       `json:"id" gorm:"primaryKey;comment:台账ID"`
	MerchantReferralID     uint       `json:"merchant_referral_id" gorm:"index;comment:邀请关系ID"`
	ReferrerUserID         uint       `json:"referrer_user_id" gorm:"index;comment:推广用户ID"`
	MerchantID             uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	PaidAmount             int        `json:"paid_amount" gorm:"comment:商户实付金额（单位：分）"`
	CommissionRateBP       int        `json:"commission_rate_bp" gorm:"comment:分成比例基点（10000=100%）"`
	CommissionAmount       int        `json:"commission_amount" gorm:"comment:应分成金额（单位：分）"`
	PaidAt                 *time.Time `json:"paid_at" gorm:"comment:商户付款时间"`
	ClaimDeadline          *time.Time `json:"claim_deadline" gorm:"comment:申请提现截止时间"`
	IsWithinCommissionWindow bool     `json:"is_within_commission_window" gorm:"default:true;comment:是否命中分成窗口"`
	WithdrawalStatus       string     `json:"withdrawal_status" gorm:"size:20;default:'claimable';comment:提现状态（claimable/applied/approved/paid/expired/rejected）"`
	WithdrawalID           *uint      `json:"withdrawal_id" gorm:"index;comment:提现申请ID"`
	Note                   string     `json:"note" gorm:"size:255;default:'';comment:备注"`
	CreatedAt              *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt              *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (MerchantReferralPaymentLedger) TableName() string {
	return "merchant_referral_payment_ledgers"
}

func (MerchantReferralPaymentLedger) TableComment() string {
	return "用户推广商户付费分成台账表"
}

// MerchantReferralWithdrawal 提现申请
type MerchantReferralWithdrawal struct {
	ID            uint       `json:"id" gorm:"primaryKey;comment:提现申请ID"`
	ReferrerUserID uint      `json:"referrer_user_id" gorm:"index;comment:推广用户ID"`
	AppliedAmount int        `json:"applied_amount" gorm:"comment:申请金额（单位：分）"`
	PayeeName     string     `json:"payee_name" gorm:"size:80;default:'';comment:收款人"`
	PayeeAccount  string     `json:"payee_account" gorm:"size:120;default:'';comment:收款账号"`
	PayeeChannel  string     `json:"payee_channel" gorm:"size:30;default:'wechat';comment:收款渠道（wechat/alipay/bank）"`
	Status        string     `json:"status" gorm:"size:20;default:'pending';comment:审核状态（pending/approved/paid/rejected）"`
	ReviewedBy    string     `json:"reviewed_by" gorm:"size:60;default:'';comment:审核人"`
	ReviewNote    string     `json:"review_note" gorm:"size:255;default:'';comment:审核备注"`
	ReviewedAt    *time.Time `json:"reviewed_at" gorm:"comment:审核时间"`
	PaidAt        *time.Time `json:"paid_at" gorm:"comment:打款时间"`
	CreatedAt     *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt     *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (MerchantReferralWithdrawal) TableName() string {
	return "merchant_referral_withdrawals"
}

func (MerchantReferralWithdrawal) TableComment() string {
	return "用户推广商户分成提现申请表"
}

// MerchantReferralWithdrawalItem 提现申请明细
type MerchantReferralWithdrawalItem struct {
	ID                 uint       `json:"id" gorm:"primaryKey;comment:明细ID"`
	WithdrawalID       uint       `json:"withdrawal_id" gorm:"uniqueIndex:uidx_mrwi_withdrawal_ledger;index;comment:提现申请ID"`
	PaymentLedgerID    uint       `json:"payment_ledger_id" gorm:"uniqueIndex:uidx_mrwi_withdrawal_ledger;uniqueIndex:uidx_mrwi_ledger_once;index;comment:分成台账ID"`
	CommissionAmount   int        `json:"commission_amount" gorm:"comment:本明细金额（单位：分）"`
	CreatedAt          *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

func (MerchantReferralWithdrawalItem) TableName() string {
	return "merchant_referral_withdrawal_items"
}

func (MerchantReferralWithdrawalItem) TableComment() string {
	return "用户推广商户分成提现明细表"
}
