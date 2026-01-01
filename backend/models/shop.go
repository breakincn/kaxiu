package models

import "time"

// PaymentConfig 商户收款配置
// 存储商户的支付宝/微信收款信息，卡包不参与收款
type PaymentConfig struct {
	ID           uint   `json:"id" gorm:"primaryKey;comment:配置ID"`
	MerchantID   uint   `json:"merchant_id" gorm:"uniqueIndex;comment:商户ID（外键关联merchants表）"`
	AlipayQRCode string `json:"alipay_qr_code" gorm:"size:500;comment:支付宝收款码图片URL"`
	WechatQRCode string `json:"wechat_qr_code" gorm:"size:500;comment:微信收款码图片URL"`
	DefaultMethod string `json:"default_method" gorm:"size:20;comment:默认收款方式（alipay/wechat）"`
	CreatedAt    *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt    *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (PaymentConfig) TableName() string {
	return "payment_configs"
}

func (PaymentConfig) TableComment() string {
	return "商户收款配置表"
}

// CardTemplate 卡片售卖模板
// 商户配置的在售卡片信息
type CardTemplate struct {
	ID                 uint   `json:"id" gorm:"primaryKey;comment:模板ID"`
	MerchantID         uint   `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	Name               string `json:"name" gorm:"size:100;comment:卡片名称（如：洗车10次卡）"`
	CardType           string `json:"card_type" gorm:"size:50;comment:卡片类型（times-次数卡，balance-充值卡，lesson-课时卡）"`
	Price              int    `json:"price" gorm:"comment:售价（单位：分）"`
	TotalTimes         int    `json:"total_times" gorm:"comment:总次数（次数卡/课时卡适用）"`
	RechargeAmount     int    `json:"recharge_amount" gorm:"comment:充值金额（单位：分，充值卡适用）"`
	ValidDays          int    `json:"valid_days" gorm:"comment:有效期天数（0表示永久有效）"`
	SupportAppointment bool   `json:"support_appointment" gorm:"default:false;comment:是否支持预约"`
	Description        string `json:"description" gorm:"size:500;comment:卡片描述"`
	SortOrder          int    `json:"sort_order" gorm:"default:0;comment:排序顺序（越小越靠前）"`
	IsActive           bool   `json:"is_active" gorm:"default:true;comment:是否在售（0-下架，1-在售）"`
	CreatedAt          *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt          *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (CardTemplate) TableName() string {
	return "card_templates"
}

func (CardTemplate) TableComment() string {
	return "卡片售卖模板表"
}

// DirectPurchase 直购订单记录
// 用户通过扫码购买卡片的订单记录
type DirectPurchase struct {
	ID             uint   `json:"id" gorm:"primaryKey;comment:订单ID"`
	OrderNo        string `json:"order_no" gorm:"size:50;uniqueIndex;comment:订单号"`
	MerchantID     uint   `json:"merchant_id" gorm:"index;comment:商户ID（外键关联merchants表）"`
	UserID         uint   `json:"user_id" gorm:"index;comment:用户ID（外键关联users表）"`
	CardTemplateID uint   `json:"card_template_id" gorm:"index;comment:卡片模板ID（外键关联card_templates表）"`
	CardID         *uint  `json:"card_id" gorm:"index;comment:生成的卡片ID（外键关联cards表，确认后填充）"`
	Price          int    `json:"price" gorm:"comment:购买价格（单位：分）"`
	PaymentMethod  string `json:"payment_method" gorm:"size:20;comment:支付方式（alipay-支付宝，wechat-微信）"`
	Status         string `json:"status" gorm:"size:20;default:pending;comment:订单状态（pending-待支付，confirmed-已确认，canceled-已取消）"`
	ConfirmedAt    *time.Time `json:"confirmed_at" gorm:"type:datetime(3);comment:用户确认付款时间"`
	CreatedAt      *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	Merchant     Merchant      `json:"merchant" gorm:"foreignKey:MerchantID"`
	User         User          `json:"user" gorm:"foreignKey:UserID"`
	CardTemplate *CardTemplate `json:"card_template" gorm:"foreignKey:CardTemplateID"`
	Card         *Card         `json:"card" gorm:"foreignKey:CardID"`
}

func (DirectPurchase) TableName() string {
	return "direct_purchases"
}

func (DirectPurchase) TableComment() string {
	return "直购订单记录表"
}

// MerchantShopSlug 商户店铺短链接
// 用于生成友好的二维码地址如 kabao.me/shop/mitao
type MerchantShopSlug struct {
	ID         uint   `json:"id" gorm:"primaryKey;comment:ID"`
	MerchantID uint   `json:"merchant_id" gorm:"uniqueIndex;comment:商户ID（外键关联merchants表）"`
	Slug       string `json:"slug" gorm:"size:50;uniqueIndex;comment:店铺短链接标识"`
	CreatedAt  *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (MerchantShopSlug) TableName() string {
	return "merchant_shop_slugs"
}

func (MerchantShopSlug) TableComment() string {
	return "商户店铺短链接表"
}
