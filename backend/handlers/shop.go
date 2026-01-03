package handlers

import (
	"fmt"
	"io"
	"kabao/config"
	"kabao/models"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func requireDirectSaleEnabledByMerchantID(c *gin.Context, merchantID uint) bool {
	var m models.Merchant
	if err := config.DB.First(&m, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return false
	}
	if !m.SupportDirectSale {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启直购售卡服务"})
		return false
	}
	return true
}

// ==================== 商户收款配置 ====================

// GetPaymentConfig 获取商户收款配置
func GetPaymentConfig(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	if !requireDirectSaleEnabledByMerchantID(c, merchantID) {
		return
	}

	var paymentCfg models.PaymentConfig
	if err := config.DB.Where("merchant_id = ?", merchantID).First(&paymentCfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": paymentCfg})
}

// SavePaymentConfig 保存商户收款配置
func SavePaymentConfig(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	if !requireDirectSaleEnabledByMerchantID(c, merchantID) {
		return
	}

	var input struct {
		AlipayQRCode  string `json:"alipay_qr_code"`
		WechatQRCode  string `json:"wechat_qr_code"`
		DefaultMethod string `json:"default_method"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 至少需要配置一种收款方式
	if input.AlipayQRCode == "" && input.WechatQRCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少配置一种收款方式"})
		return
	}

	// 校验默认方式
	defaultMethod := strings.TrimSpace(input.DefaultMethod)
	if defaultMethod != "" && defaultMethod != "alipay" && defaultMethod != "wechat" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的默认收款方式"})
		return
	}
	if defaultMethod == "alipay" && input.AlipayQRCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置支付宝收款码，无法设为默认"})
		return
	}
	if defaultMethod == "wechat" && input.WechatQRCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置微信收款码，无法设为默认"})
		return
	}

	// 未指定默认方式时自动推导
	if defaultMethod == "" {
		if input.AlipayQRCode != "" {
			defaultMethod = "alipay"
		} else {
			defaultMethod = "wechat"
		}
	}

	var paymentConfig models.PaymentConfig
	err := config.DB.Where("merchant_id = ?", merchantID).First(&paymentConfig).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		paymentConfig = models.PaymentConfig{
			MerchantID:    merchantID,
			AlipayQRCode:  input.AlipayQRCode,
			WechatQRCode:  input.WechatQRCode,
			DefaultMethod: defaultMethod,
		}
		if err := config.DB.Create(&paymentConfig).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
			return
		}
	} else if err == nil {
		// 更新配置
		updates := map[string]interface{}{
			"alipay_qr_code": input.AlipayQRCode,
			"wechat_qr_code": input.WechatQRCode,
			"default_method": defaultMethod,
		}
		if err := config.DB.Model(&paymentConfig).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功", "data": paymentConfig})
}

// ==================== 卡片模板管理 ====================

// GetCardTemplates 获取商户的卡片模板列表
func GetCardTemplates(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var templates []models.CardTemplate
	config.DB.Where("merchant_id = ?", merchantID).Order("sort_order asc, id desc").Find(&templates)
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// CreateCardTemplate 创建卡片模板
func CreateCardTemplate(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		Name               string `json:"name" binding:"required"`
		CardType           string `json:"card_type" binding:"required"`
		Price              int    `json:"price" binding:"required,min=1"`
		TotalTimes         int    `json:"total_times"`
		RechargeAmount     int    `json:"recharge_amount"`
		ValidDays          int    `json:"valid_days"`
		SupportAppointment bool   `json:"support_appointment"`
		Description        string `json:"description"`
		SortOrder          int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证卡片类型
	if input.CardType != "times" && input.CardType != "balance" && input.CardType != "lesson" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的卡片类型"})
		return
	}

	// 次数卡和课时卡必须有次数
	if (input.CardType == "times" || input.CardType == "lesson") && input.TotalTimes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "次数卡/课时卡必须设置总次数"})
		return
	}

	// 充值卡必须有充值金额
	if input.CardType == "balance" && input.RechargeAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "充值卡必须设置充值金额"})
		return
	}

	template := models.CardTemplate{
		MerchantID:         merchantID,
		Name:               input.Name,
		CardType:           input.CardType,
		Price:              input.Price,
		TotalTimes:         input.TotalTimes,
		RechargeAmount:     input.RechargeAmount,
		ValidDays:          input.ValidDays,
		SupportAppointment: input.SupportAppointment,
		Description:        input.Description,
		SortOrder:          input.SortOrder,
		IsActive:           true,
	}

	if err := config.DB.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// UpdateCardTemplate 更新卡片模板
func UpdateCardTemplate(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var template models.CardTemplate
	if err := config.DB.Where("id = ? AND merchant_id = ?", id, merchantID).First(&template).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	var input struct {
		Name               *string `json:"name"`
		CardType           *string `json:"card_type"`
		Price              *int    `json:"price"`
		TotalTimes         *int    `json:"total_times"`
		RechargeAmount     *int    `json:"recharge_amount"`
		ValidDays          *int    `json:"valid_days"`
		SupportAppointment *bool   `json:"support_appointment"`
		Description        *string `json:"description"`
		SortOrder          *int    `json:"sort_order"`
		IsActive           *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.CardType != nil {
		updates["card_type"] = *input.CardType
	}
	if input.Price != nil {
		updates["price"] = *input.Price
	}
	if input.TotalTimes != nil {
		updates["total_times"] = *input.TotalTimes
	}
	if input.RechargeAmount != nil {
		updates["recharge_amount"] = *input.RechargeAmount
	}
	if input.ValidDays != nil {
		updates["valid_days"] = *input.ValidDays
	}
	if input.SupportAppointment != nil {
		updates["support_appointment"] = *input.SupportAppointment
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.SortOrder != nil {
		updates["sort_order"] = *input.SortOrder
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}

	if err := config.DB.Model(&template).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	config.DB.First(&template, template.ID)
	c.JSON(http.StatusOK, gin.H{"data": template})
}

// DeleteCardTemplate 删除卡片模板
func DeleteCardTemplate(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	id := c.Param("id")
	result := config.DB.Where("id = ? AND merchant_id = ?", id, merchantID).Delete(&models.CardTemplate{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ==================== 店铺短链接管理 ====================

// GetShopSlug 获取商户店铺短链接
func GetShopSlug(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	if !requireDirectSaleEnabledByMerchantID(c, merchantID) {
		return
	}

	var slug models.MerchantShopSlug
	if err := config.DB.Where("merchant_id = ?", merchantID).First(&slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": slug})
}

// SaveShopSlug 保存商户店铺短链接
func SaveShopSlug(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	if !requireDirectSaleEnabledByMerchantID(c, merchantID) {
		return
	}

	var input struct {
		Slug string `json:"slug" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证slug格式（只允许字母、数字、下划线、连字符）
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if len(slug) < 2 || len(slug) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "短链接长度需为2-30个字符"})
		return
	}
	matched, _ := regexp.MatchString(`^[a-z0-9_-]+$`, slug)
	if !matched {
		c.JSON(http.StatusBadRequest, gin.H{"error": "短链接只能包含字母、数字、下划线和连字符"})
		return
	}

	// 检查是否已被使用
	var existing models.MerchantShopSlug
	if err := config.DB.Where("slug = ? AND merchant_id != ?", slug, merchantID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该短链接已被使用"})
		return
	}

	var shopSlug models.MerchantShopSlug
	err := config.DB.Where("merchant_id = ?", merchantID).First(&shopSlug).Error

	if err == gorm.ErrRecordNotFound {
		shopSlug = models.MerchantShopSlug{
			MerchantID: merchantID,
			Slug:       slug,
		}
		if err := config.DB.Create(&shopSlug).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
			return
		}
	} else if err == nil {
		if err := config.DB.Model(&shopSlug).Update("slug", slug).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功", "data": shopSlug})
}

// ==================== 公开接口（无需登录） ====================

// GetShopInfo 获取商户店铺信息（公开接口，用于扫码后展示）
func GetShopInfo(c *gin.Context) {
	slug := c.Param("slug")

	// 通过slug查找商户
	var shopSlug models.MerchantShopSlug
	if err := config.DB.Where("slug = ?", slug).First(&shopSlug).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "店铺不存在"})
		return
	}

	// 获取商户信息
	var merchant models.Merchant
	if err := config.DB.First(&merchant, shopSlug.MerchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 获取收款配置
	var paymentConfig models.PaymentConfig
	config.DB.Where("merchant_id = ?", merchant.ID).First(&paymentConfig)

	// 获取在售卡片模板（仅当商户开启直购售卡服务）
	var templates []models.CardTemplate
	if merchant.SupportDirectSale {
		config.DB.Where("merchant_id = ? AND is_active = ?", merchant.ID, true).
			Order("sort_order asc, id desc").Find(&templates)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"merchant": gin.H{
				"id":   merchant.ID,
				"name": merchant.Name,
				"type": merchant.Type,
			},
			"payment_config": gin.H{
				"has_alipay":     paymentConfig.AlipayQRCode != "",
				"has_wechat":     paymentConfig.WechatQRCode != "",
				"alipay_qr_code": paymentConfig.AlipayQRCode,
				"wechat_qr_code": paymentConfig.WechatQRCode,
				"default_method": paymentConfig.DefaultMethod,
			},
			"card_templates": templates,
		},
	})
}

// GetShopInfoByID 通过商户ID获取店铺信息（公开接口）
func GetShopInfoByID(c *gin.Context) {
	merchantID := c.Param("id")

	// 获取商户信息
	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 获取收款配置
	var paymentConfig models.PaymentConfig
	config.DB.Where("merchant_id = ?", merchant.ID).First(&paymentConfig)

	// 获取在售卡片模板（仅当商户开启直购售卡服务）
	var templates []models.CardTemplate
	if merchant.SupportDirectSale {
		config.DB.Where("merchant_id = ? AND is_active = ?", merchant.ID, true).
			Order("sort_order asc, id desc").Find(&templates)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"merchant": gin.H{
				"id":   merchant.ID,
				"name": merchant.Name,
				"type": merchant.Type,
			},
			"payment_config": gin.H{
				"has_alipay":     paymentConfig.AlipayQRCode != "",
				"has_wechat":     paymentConfig.WechatQRCode != "",
				"alipay_qr_code": paymentConfig.AlipayQRCode,
				"wechat_qr_code": paymentConfig.WechatQRCode,
				"default_method": paymentConfig.DefaultMethod,
			},
			"card_templates": templates,
		},
	})
}

// ==================== 直购流程 ====================

// CreateDirectPurchase 创建直购订单（仅返回收款信息，不落库）
func CreateDirectPurchase(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var input struct {
		CardTemplateID uint   `json:"card_template_id" binding:"required"`
		PaymentMethod  string `json:"payment_method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.PaymentMethod != "alipay" && input.PaymentMethod != "wechat" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的支付方式"})
		return
	}

	// 获取卡片模板
	var template models.CardTemplate
	if err := config.DB.First(&template, input.CardTemplateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	if !requireDirectSaleEnabledByMerchantID(c, template.MerchantID) {
		return
	}

	if !template.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该卡片已下架"})
		return
	}

	// 获取商户收款配置
	var paymentConfig models.PaymentConfig
	if err := config.DB.Where("merchant_id = ?", template.MerchantID).First(&paymentConfig).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未配置收款信息"})
		return
	}

	// 验证商户是否配置了对应的支付方式
	var paymentURL string
	if input.PaymentMethod == "alipay" {
		if paymentConfig.AlipayQRCode != "" {
			paymentURL = paymentConfig.AlipayQRCode
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "商户未配置支付宝收款"})
			return
		}
	} else {
		if paymentConfig.WechatQRCode != "" {
			paymentURL = paymentConfig.WechatQRCode
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "商户未配置微信收款"})
			return
		}
	}

	// 生成订单号（不落库，仅用于后续用户确认时落库）
	_ = userID
	orderNo := fmt.Sprintf("DP%s%s", time.Now().Format("20060102150405"), uuid.New().String()[:6])

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"order_no":         orderNo,
			"card_template_id": template.ID,
			"price":            template.Price,
			"payment_url":      paymentURL,
			"payment_method":   input.PaymentMethod,
		},
	})
}

// ConfirmDirectPurchase 确认直购订单（用户确认已付款，首次确认才落库）
func ConfirmDirectPurchase(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	orderNo := c.Param("order_no")

	var input struct {
		CardTemplateID uint   `json:"card_template_id" binding:"required"`
		PaymentMethod  string `json:"payment_method" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		if err == io.EOF {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数为空"})
			return
		}
		if strings.Contains(err.Error(), "EOF") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数为空"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.PaymentMethod != "alipay" && input.PaymentMethod != "wechat" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的支付方式"})
		return
	}

	var purchase models.DirectPurchase
	err := config.DB.Where("order_no = ? AND user_id = ?", orderNo, userID).First(&purchase).Error
	if err == nil {
		if purchase.Status == "paid" {
			c.JSON(http.StatusOK, gin.H{
				"message": "已提交付款，等待商户确认",
				"data":    purchase,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单状态无效"})
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 首次确认：创建订单记录并标记为已付款
	var template models.CardTemplate
	if err := config.DB.First(&template, input.CardTemplateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	if !requireDirectSaleEnabledByMerchantID(c, template.MerchantID) {
		return
	}
	if !template.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该卡片已下架"})
		return
	}

	now := time.Now()
	purchase = models.DirectPurchase{
		OrderNo:        orderNo,
		MerchantID:     template.MerchantID,
		UserID:         userID,
		CardTemplateID: template.ID,
		Price:          template.Price,
		PaymentMethod:  input.PaymentMethod,
		Status:         "paid",
		PaidAt:         &now,
	}
	if err := config.DB.Create(&purchase).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已提交付款，等待商户确认",
		"data":    purchase,
	})
}

// MerchantConfirmDirectPurchase 商户确认直购订单（商户确认后开卡）
func MerchantConfirmDirectPurchase(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	if !requireDirectSaleEnabledByMerchantID(c, merchantID) {
		return
	}

	orderNo := c.Param("order_no")

	var purchase models.DirectPurchase
	if err := config.DB.Where("order_no = ? AND merchant_id = ?", orderNo, merchantID).First(&purchase).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}

	if purchase.Status == "confirmed" {
		config.DB.Preload("Card").Preload("CardTemplate").Preload("User").First(&purchase, purchase.ID)
		c.JSON(http.StatusOK, gin.H{
			"message": "订单已确认",
			"data":    purchase,
		})
		return
	}

	if purchase.Status != "paid" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单状态无效"})
		return
	}

	// 获取卡片模板
	var template models.CardTemplate
	if err := config.DB.First(&template, purchase.CardTemplateID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "卡片模板不存在"})
		return
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		confirmedAt := now

		var endDate *time.Time
		if template.ValidDays > 0 {
			end := now.AddDate(0, 0, template.ValidDays)
			endDate = &end
		} else {
			end := now.AddDate(100, 0, 0)
			endDate = &end
		}

		cardNo := uuid.New().String()[:8]
		card := models.Card{
			UserID:         purchase.UserID,
			MerchantID:     purchase.MerchantID,
			CardNo:         cardNo,
			CardType:       template.Name,
			TotalTimes:     template.TotalTimes,
			RemainTimes:    template.TotalTimes,
			UsedTimes:      0,
			RechargeAmount: template.RechargeAmount / 100,
			RechargeAt:     &now,
			StartDate:      &now,
			EndDate:        endDate,
		}

		if err := tx.Create(&card).Error; err != nil {
			return err
		}

		if err := tx.Model(&purchase).Updates(map[string]interface{}{
			"status":       "confirmed",
			"confirmed_at": &confirmedAt,
			"card_id":      card.ID,
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开卡失败"})
		return
	}

	config.DB.Preload("Card").Preload("CardTemplate").Preload("User").First(&purchase, purchase.ID)
	c.JSON(http.StatusOK, gin.H{
		"message": "开卡成功",
		"data":    purchase,
	})
}

// GetDirectPurchases 获取用户的直购订单列表
func GetDirectPurchases(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var purchases []models.DirectPurchase
	config.DB.Preload("Merchant").Preload("CardTemplate").Preload("Card").
		Where("user_id = ?", userID).
		Order("CASE WHEN status = 'paid' THEN 0 ELSE 1 END, CASE WHEN status = 'paid' THEN created_at END ASC, created_at DESC").
		Find(&purchases)

	c.JSON(http.StatusOK, gin.H{"data": purchases})
}

// GetMerchantDirectPurchases 获取商户的直购订单列表
func GetMerchantDirectPurchases(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	if !requireDirectSaleEnabledByMerchantID(c, merchantID) {
		return
	}

	var purchases []models.DirectPurchase
	config.DB.Preload("User").Preload("CardTemplate").Preload("Card").
		Where("merchant_id = ?", merchantID).
		Order("CASE WHEN status = 'paid' THEN 0 ELSE 1 END, CASE WHEN status = 'paid' THEN created_at END ASC, created_at DESC").
		Find(&purchases)

	c.JSON(http.StatusOK, gin.H{"data": purchases})
}

// ==================== 辅助函数 ====================

func getMerchantID(c *gin.Context) (uint, bool) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return 0, false
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return 0, false
	}
	return merchantID, true
}

func getUserID(c *gin.Context) (uint, bool) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "请先登录"})
		return 0, false
	}
	userID, ok := userIDAny.(uint)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return 0, false
	}
	return userID, true
}
