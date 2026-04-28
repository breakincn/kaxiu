package models

import "time"

// PromotionCardCampaign 推广卡活动
type PromotionCardCampaign struct {
	ID                   uint       `json:"id" gorm:"primaryKey;comment:活动ID"`
	MerchantID           uint       `json:"merchant_id" gorm:"index;comment:商户ID"`
	CardTemplateID       uint       `json:"card_template_id" gorm:"index;comment:售卡模板ID"`
	Title                string     `json:"title" gorm:"size:360;comment:推广标题"`
	RewardTotalTimes     int        `json:"reward_total_times" gorm:"comment:奖励次数"`
	RewardRechargeAmount int        `json:"reward_recharge_amount" gorm:"comment:奖励额度（单位：分）"`
	RewardCardQuantity   int        `json:"reward_card_quantity" gorm:"comment:奖励卡总库存"`
	RewardThreshold      int        `json:"reward_threshold" gorm:"comment:达标推广数量"`
	PromoPrice           int        `json:"promo_price" gorm:"comment:促销价格（单位：分）"`
	PromoQuantity        int        `json:"promo_quantity" gorm:"comment:促销资格总库存"`
	PromoEndsAt          *time.Time `json:"promo_ends_at" gorm:"type:datetime(3);comment:促销截止时间"`
	Slug                 string     `json:"slug" gorm:"size:80;uniqueIndex;comment:活动短链接标识"`
	Status               string     `json:"status" gorm:"size:20;default:'active';comment:活动状态（active/disabled）"`
	CreatedAt            *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt            *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	Merchant     Merchant     `json:"merchant" gorm:"foreignKey:MerchantID"`
	CardTemplate CardTemplate `json:"card_template" gorm:"foreignKey:CardTemplateID"`
}

func (PromotionCardCampaign) TableName() string {
	return "promotion_card_campaigns"
}

func (PromotionCardCampaign) TableComment() string {
	return "推广卡活动表"
}

// PromotionCardClaim 用户促销领取资格
type PromotionCardClaim struct {
	ID             uint       `json:"id" gorm:"primaryKey;comment:资格ID"`
	CampaignID     uint       `json:"campaign_id" gorm:"uniqueIndex:uidx_campaign_user_claim;index;comment:活动ID"`
	UserID         uint       `json:"user_id" gorm:"uniqueIndex:uidx_campaign_user_claim;index;comment:用户ID"`
	ReferrerUserID *uint      `json:"referrer_user_id" gorm:"index;comment:推广人用户ID"`
	PromotionCode  string     `json:"promotion_code" gorm:"size:20;comment:推广码"`
	Status         string     `json:"status" gorm:"size:20;default:'active';comment:资格状态（active/expired/paid/confirmed）"`
	ClaimedAt      *time.Time `json:"claimed_at" gorm:"type:datetime(3);comment:领取时间"`
	ExpiresAt      *time.Time `json:"expires_at" gorm:"type:datetime(3);comment:过期时间"`
	PaidAt         *time.Time `json:"paid_at" gorm:"type:datetime(3);comment:付款时间"`
	ExpiredAt      *time.Time `json:"expired_at" gorm:"type:datetime(3);comment:失效时间"`
	OrderID        *uint      `json:"order_id" gorm:"index;comment:关联订单ID"`
	CreatedAt      *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt      *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (PromotionCardClaim) TableName() string {
	return "promotion_card_claims"
}

func (PromotionCardClaim) TableComment() string {
	return "推广卡促销领取资格表"
}

// PromotionCardReferrer 推广人活动进度
type PromotionCardReferrer struct {
	ID                uint       `json:"id" gorm:"primaryKey;comment:记录ID"`
	CampaignID        uint       `json:"campaign_id" gorm:"uniqueIndex:uidx_campaign_user_referrer;index;comment:活动ID"`
	UserID            uint       `json:"user_id" gorm:"uniqueIndex:uidx_campaign_user_referrer;index;comment:推广人用户ID"`
	PromotionCode     string     `json:"promotion_code" gorm:"size:6;comment:6位推广码"`
	RegisterCount     int        `json:"register_count" gorm:"comment:注册累计数"`
	PaidCount         int        `json:"paid_count" gorm:"comment:付款累计数"`
	ProgressCount     int        `json:"progress_count" gorm:"comment:总进度"`
	RewardStatus      string     `json:"reward_status" gorm:"size:20;default:'';comment:奖励状态（/claimable/claimed/expired）"`
	RewardQualifiedAt *time.Time `json:"reward_qualified_at" gorm:"type:datetime(3);comment:达标时间"`
	RewardExpiresAt   *time.Time `json:"reward_expires_at" gorm:"type:datetime(3);comment:奖励过期时间"`
	RewardClaimedAt   *time.Time `json:"reward_claimed_at" gorm:"type:datetime(3);comment:奖励领取时间"`
	RewardExpiredAt   *time.Time `json:"reward_expired_at" gorm:"type:datetime(3);comment:奖励失效时间"`
	RewardCardID      *uint      `json:"reward_card_id" gorm:"index;comment:奖励卡ID"`
	CreatedAt         *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt         *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (PromotionCardReferrer) TableName() string {
	return "promotion_card_referrers"
}

func (PromotionCardReferrer) TableComment() string {
	return "推广卡推广人进度表"
}

// PromotionCardReferral 被邀请人贡献记录
type PromotionCardReferral struct {
	ID                  uint       `json:"id" gorm:"primaryKey;comment:记录ID"`
	CampaignID          uint       `json:"campaign_id" gorm:"uniqueIndex:uidx_campaign_referral;index;comment:活动ID"`
	ReferrerUserID      uint       `json:"referrer_user_id" gorm:"uniqueIndex:uidx_campaign_referral;index;comment:推广人用户ID"`
	InviteeUserID       uint       `json:"invitee_user_id" gorm:"uniqueIndex:uidx_campaign_referral;index;comment:被邀请人用户ID"`
	PromotionCode       string     `json:"promotion_code" gorm:"size:6;comment:推广码"`
	RegisteredCountedAt *time.Time `json:"registered_counted_at" gorm:"type:datetime(3);comment:注册计数时间"`
	PaidCountedAt       *time.Time `json:"paid_counted_at" gorm:"type:datetime(3);comment:付款计数时间"`
	CreatedAt           *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt           *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (PromotionCardReferral) TableName() string {
	return "promotion_card_referrals"
}

func (PromotionCardReferral) TableComment() string {
	return "推广卡邀请贡献表"
}
