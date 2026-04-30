package handlers

import (
	"crypto/rand"
	"fmt"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	promotionCardClaimStatusActive    = "active"
	promotionCardClaimStatusExpired   = "expired"
	promotionCardClaimStatusPaid      = "paid"
	promotionCardClaimStatusConfirmed = "confirmed"

	promotionCardRewardStatusClaimable = "claimable"
	promotionCardRewardStatusClaimed   = "claimed"
	promotionCardRewardStatusExpired   = "expired"

	directPurchaseSourcePromotion = "promotion_campaign"
	directPurchaseStatusStoreWait = "store_pending_wait"
)

type promotionCampaignDetail struct {
	ID                   uint        `json:"id"`
	MerchantID           uint        `json:"merchant_id"`
	CardTemplateID       uint        `json:"card_template_id"`
	Title                string      `json:"title"`
	RewardTotalTimes     int         `json:"reward_total_times"`
	RewardRechargeAmount int         `json:"reward_recharge_amount"`
	RewardCardQuantity   int         `json:"reward_card_quantity"`
	RewardThreshold      int         `json:"reward_threshold"`
	PromoPrice           int         `json:"promo_price"`
	PromoQuantity        int         `json:"promo_quantity"`
	PromoEndsAt          *time.Time  `json:"promo_ends_at"`
	Slug                 string      `json:"slug"`
	Status               string      `json:"status"`
	SharePath            string      `json:"share_path"`
	PromoActive          bool        `json:"promo_active"`
	PromoRemaining       int         `json:"promo_remaining"`
	RewardRemaining      int         `json:"reward_remaining"`
	CanOriginalPurchase  bool        `json:"can_original_purchase"`
	CanClaimPromo        bool        `json:"can_claim_promo"`
	CanShare             bool        `json:"can_share"`
	Merchant             interface{} `json:"merchant"`
	PaymentConfig        interface{} `json:"payment_config"`
	CardTemplate         interface{} `json:"card_template"`
	MyClaim              interface{} `json:"my_claim"`
	MyReferral           interface{} `json:"my_referral"`
	CurrentRefCode       string      `json:"current_ref_code"`
}

type promotionRewardListItem struct {
	ListItemType         string     `json:"list_item_type"`
	ReferrerID           uint       `json:"referrer_id"`
	CampaignID           uint       `json:"campaign_id"`
	MerchantID           uint       `json:"merchant_id"`
	MerchantName         string     `json:"merchant_name"`
	CardName             string     `json:"card_name"`
	CardTemplateType     string     `json:"card_template_type"`
	Title                string     `json:"title"`
	Slug                 string     `json:"slug"`
	CampaignStatus       string     `json:"campaign_status"`
	RewardThreshold      int        `json:"reward_threshold"`
	RewardTotalTimes     int        `json:"reward_total_times"`
	RewardRechargeAmount int        `json:"reward_recharge_amount"`
	RegisterCount        int        `json:"register_count"`
	PaidCount            int        `json:"paid_count"`
	ProgressCount        int        `json:"progress_count"`
	RewardStatus         string     `json:"reward_status"`
	RewardQualifiedAt    *time.Time `json:"reward_qualified_at"`
	RewardExpiresAt      *time.Time `json:"reward_expires_at"`
	RewardClaimedAt      *time.Time `json:"reward_claimed_at"`
	UpdatedAt            *time.Time `json:"updated_at"`
}

type merchantPromotionCampaignListItem struct {
	models.PromotionCardCampaign
	ProgressCount int  `json:"progress_count"`
	CanDelete     bool `json:"can_delete"`
}

func ListMerchantPromotionCampaigns(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	query := config.DB.Where("merchant_id = ?", merchantID)
	if templateID, _ := strconv.ParseUint(strings.TrimSpace(c.Query("template_id")), 10, 64); templateID > 0 {
		query = query.Where("card_template_id = ?", uint(templateID))
	}

	var campaigns []models.PromotionCardCampaign
	if err := query.Order("id desc").Find(&campaigns).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询活动失败"})
		return
	}

	progressByCampaign := map[uint]int{}
	if len(campaigns) > 0 {
		ids := make([]uint, 0, len(campaigns))
		for _, campaign := range campaigns {
			ids = append(ids, campaign.ID)
		}
		var rows []struct {
			CampaignID    uint `json:"campaign_id"`
			ProgressCount int  `json:"progress_count"`
		}
		if err := config.DB.Model(&models.PromotionCardReferrer{}).
			Select("campaign_id, COALESCE(SUM(progress_count), 0) AS progress_count").
			Where("campaign_id IN ?", ids).
			Group("campaign_id").
			Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询活动失败"})
			return
		}
		for _, row := range rows {
			progressByCampaign[row.CampaignID] = row.ProgressCount
		}
	}

	items := make([]merchantPromotionCampaignListItem, 0, len(campaigns))
	for _, campaign := range campaigns {
		canDelete, err := canDeletePromotionCampaign(config.DB, campaign.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询活动失败"})
			return
		}
		items = append(items, merchantPromotionCampaignListItem{
			PromotionCardCampaign: campaign,
			ProgressCount:         progressByCampaign[campaign.ID],
			CanDelete:             canDelete,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

func GetMerchantPromotionCampaign(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var campaign models.PromotionCardCampaign
	if err := config.DB.Where("id = ? AND merchant_id = ?", c.Param("id"), merchantID).First(&campaign).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "活动不存在"})
		return
	}

	detail, err := buildPromotionCampaignDetail(config.DB, &campaign, 0, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取活动失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": detail})
}

func CreateMerchantPromotionCampaign(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		CardTemplateID       uint   `json:"card_template_id" binding:"required"`
		Title                string `json:"title"`
		RewardTotalTimes     int    `json:"reward_total_times"`
		RewardRechargeAmount int    `json:"reward_recharge_amount"`
		RewardCardQuantity   int    `json:"reward_card_quantity" binding:"required"`
		RewardThreshold      int    `json:"reward_threshold" binding:"required"`
		PromoPrice           int    `json:"promo_price"`
		PromoQuantity        int    `json:"promo_quantity"`
		PromoEndsAt          string `json:"promo_ends_at"`
		Status               string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	campaign, err := savePromotionCampaign(0, merchantID, input.CardTemplateID, input.Title, input.RewardTotalTimes, input.RewardRechargeAmount, input.RewardCardQuantity, input.RewardThreshold, input.PromoPrice, input.PromoQuantity, input.PromoEndsAt, input.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := buildPromotionCampaignDetail(config.DB, campaign, 0, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取活动失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": detail})
}

func UpdateMerchantPromotionCampaign(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var existing models.PromotionCardCampaign
	if err := config.DB.Where("id = ? AND merchant_id = ?", c.Param("id"), merchantID).First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "活动不存在"})
		return
	}

	var input struct {
		CardTemplateID       uint   `json:"card_template_id" binding:"required"`
		Title                string `json:"title"`
		RewardTotalTimes     int    `json:"reward_total_times"`
		RewardRechargeAmount int    `json:"reward_recharge_amount"`
		RewardCardQuantity   int    `json:"reward_card_quantity" binding:"required"`
		RewardThreshold      int    `json:"reward_threshold" binding:"required"`
		PromoPrice           int    `json:"promo_price"`
		PromoQuantity        int    `json:"promo_quantity"`
		PromoEndsAt          string `json:"promo_ends_at"`
		Status               string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	campaign, err := savePromotionCampaign(existing.ID, merchantID, input.CardTemplateID, input.Title, input.RewardTotalTimes, input.RewardRechargeAmount, input.RewardCardQuantity, input.RewardThreshold, input.PromoPrice, input.PromoQuantity, input.PromoEndsAt, input.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := buildPromotionCampaignDetail(config.DB, campaign, 0, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取活动失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": detail})
}

func DeleteMerchantPromotionCampaign(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	campaignID, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || campaignID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "活动不存在"})
		return
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		var campaign models.PromotionCardCampaign
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND merchant_id = ?", uint(campaignID), merchantID).
			First(&campaign).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apiErr{status: http.StatusNotFound, msg: "活动不存在"}
			}
			return err
		}

		canDelete, err := canDeletePromotionCampaign(tx, campaign.ID)
		if err != nil {
			return err
		}
		if !canDelete {
			return apiErr{status: http.StatusBadRequest, msg: "当前活动已有领取、购买或转发进度，不能删除"}
		}

		if err := tx.Where("campaign_id = ?", campaign.ID).Delete(&models.PromotionCardReferral{}).Error; err != nil {
			return err
		}
		if err := tx.Where("campaign_id = ?", campaign.ID).Delete(&models.PromotionCardReferrer{}).Error; err != nil {
			return err
		}
		if err := tx.Where("campaign_id = ?", campaign.ID).Delete(&models.PromotionCardClaim{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", campaign.ID).Delete(&models.PromotionCardCampaign{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		renderPromotionErr(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func GetPromotionCampaignBySlug(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "活动标识不能为空"})
		return
	}

	var campaign models.PromotionCardCampaign
	if err := config.DB.Where("slug = ?", slug).First(&campaign).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "活动不存在"})
		return
	}

	userID := optionalUserIDFromAuthHeader(c)
	refCode := strings.TrimSpace(c.Param("refCode"))
	detail, err := buildPromotionCampaignDetail(config.DB, &campaign, userID, refCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取活动失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": detail})
}

func ListMyPromotionRewardCards(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	items, err := listPromotionRewardCards(config.DB, userID, strings.TrimSpace(c.Query("status")), time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取转发领卡进度失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func ClaimPromotionCampaign(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	slug := strings.TrimSpace(c.Param("slug"))
	refCode := strings.TrimSpace(c.Param("refCode"))
	now := time.Now()

	var result models.PromotionCardClaim
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		campaign, err := loadCampaignBySlugForUpdate(tx, slug)
		if err != nil {
			return err
		}
		if err := refreshPromotionCampaignState(tx, campaign.ID, now); err != nil {
			return err
		}
		if err := validateClaimableCampaign(tx, campaign, userID, now); err != nil {
			return err
		}

		referrerUserID, normalizedRefCode, err := resolveCampaignReferrer(tx, campaign.ID, userID, refCode)
		if err != nil {
			return err
		}

		expiresAt := now.Add(24 * time.Hour)
		row := models.PromotionCardClaim{
			CampaignID:     campaign.ID,
			UserID:         userID,
			ReferrerUserID: referrerUserID,
			PromotionCode:  normalizedRefCode,
			Status:         promotionCardClaimStatusActive,
			ClaimedAt:      &now,
			ExpiresAt:      &expiresAt,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		result = row
		return nil
	}); err != nil {
		renderPromotionErr(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func GetPromotionCampaignRefLink(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	slug := strings.TrimSpace(c.Param("slug"))
	now := time.Now()

	var campaign *models.PromotionCardCampaign
	var referrer models.PromotionCardReferrer
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		campaign, err = loadCampaignBySlugForUpdate(tx, slug)
		if err != nil {
			return err
		}
		if err := refreshPromotionCampaignState(tx, campaign.ID, now); err != nil {
			return err
		}
		referrer, err = getOrCreatePromotionReferrer(tx, campaign.ID, userID, now)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		renderPromotionErr(c, err)
		return
	}

	sharePath := fmt.Sprintf("/promo/%s/%s", campaign.Slug, referrer.PromotionCode)
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"campaign_id":    campaign.ID,
			"promotion_code": referrer.PromotionCode,
			"share_path":     sharePath,
		},
	})
}

func ClaimPromotionCampaignReward(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	slug := strings.TrimSpace(c.Param("slug"))
	now := time.Now()

	var outCard models.Card
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		campaign, err := loadCampaignBySlugForUpdate(tx, slug)
		if err != nil {
			return err
		}
		if err := refreshPromotionCampaignState(tx, campaign.ID, now); err != nil {
			return err
		}

		var referrer models.PromotionCardReferrer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("campaign_id = ? AND user_id = ?", campaign.ID, userID).
			First(&referrer).Error; err != nil {
			return apiErr{status: http.StatusBadRequest, msg: "暂无可领取奖励"}
		}
		if referrer.RewardStatus != promotionCardRewardStatusClaimable {
			return apiErr{status: http.StatusBadRequest, msg: "暂无可领取奖励"}
		}
		if referrer.RewardExpiresAt == nil || !referrer.RewardExpiresAt.After(now) {
			referrer.RewardStatus = promotionCardRewardStatusExpired
			referrer.RewardExpiredAt = &now
			referrer.RewardExpiresAt = &now
			if err := tx.Save(&referrer).Error; err != nil {
				return err
			}
			return apiErr{status: http.StatusBadRequest, msg: "奖励已过期"}
		}

		var template models.CardTemplate
		if err := tx.Where("id = ? AND merchant_id = ?", campaign.CardTemplateID, campaign.MerchantID).First(&template).Error; err != nil {
			return apiErr{status: http.StatusBadRequest, msg: "卡片模板不存在"}
		}

		card, err := createPromotionRewardCard(tx, &template, campaign, userID, now)
		if err != nil {
			return err
		}
		referrer.RewardStatus = promotionCardRewardStatusClaimed
		referrer.RewardClaimedAt = &now
		referrer.RewardCardID = &card.ID
		if err := tx.Save(&referrer).Error; err != nil {
			return err
		}
		outCard = card
		return nil
	}); err != nil {
		renderPromotionErr(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": outCard})
}

func savePromotionCampaign(id uint, merchantID uint, templateID uint, title string, rewardTotalTimes int, rewardRechargeAmount int, rewardCardQuantity int, rewardThreshold int, promoPrice int, promoQuantity int, promoEndsAtRaw string, status string) (*models.PromotionCardCampaign, error) {
	title = strings.TrimSpace(title)
	if len([]rune(title)) > 120 {
		return nil, fmt.Errorf("推广标题最多120个中文字")
	}
	if rewardCardQuantity <= 0 {
		return nil, fmt.Errorf("发卡数量必须大于0")
	}
	if rewardThreshold <= 0 {
		return nil, fmt.Errorf("推广数量必须大于0")
	}
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "disabled" {
		return nil, fmt.Errorf("活动状态无效")
	}

	var promoEndsAt *time.Time
	if strings.TrimSpace(promoEndsAtRaw) != "" {
		t, err := time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(promoEndsAtRaw), time.Local)
		if err != nil {
			t2, err2 := time.ParseInLocation("2006-01-02", strings.TrimSpace(promoEndsAtRaw), time.Local)
			if err2 != nil {
				return nil, fmt.Errorf("促销截止日期格式错误")
			}
			end := t2.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			promoEndsAt = &end
		} else {
			promoEndsAt = &t
		}
	}

	var campaign models.PromotionCardCampaign
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var template models.CardTemplate
		if err := tx.Where("id = ? AND merchant_id = ?", templateID, merchantID).First(&template).Error; err != nil {
			return fmt.Errorf("售卡模板不存在")
		}
		if !template.IsActive {
			return fmt.Errorf("售卡模板已下架")
		}

		if template.CardType == "balance" {
			if rewardRechargeAmount <= 0 {
				return fmt.Errorf("额度卡奖励额度必须大于0")
			}
			rewardTotalTimes = 0
		} else {
			if rewardTotalTimes <= 0 {
				return fmt.Errorf("次数/课时奖励次数必须大于0")
			}
			rewardRechargeAmount = 0
		}

		if promoPrice < 0 {
			return fmt.Errorf("促销价格不能小于0")
		}
		if promoPrice > 0 {
			if promoQuantity <= 0 {
				return fmt.Errorf("促销发卡数量必须大于0")
			}
			if promoEndsAt == nil {
				return fmt.Errorf("请填写促销截止日期")
			}
		} else {
			promoQuantity = 0
			promoEndsAt = nil
		}

		if id > 0 {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ? AND merchant_id = ?", id, merchantID).
				First(&campaign).Error; err != nil {
				return fmt.Errorf("活动不存在")
			}
			campaign.Slug = normalizePromotionCampaignSlug(campaign.Slug, merchantID, templateID)
		} else {
			campaign = models.PromotionCardCampaign{
				MerchantID:     merchantID,
				CardTemplateID: templateID,
				Status:         status,
			}
			slug, err := newPromotionCampaignSlug(tx, merchantID, templateID)
			if err != nil {
				return err
			}
			campaign.Slug = slug
		}

		campaign.CardTemplateID = templateID
		campaign.Title = title
		campaign.RewardTotalTimes = rewardTotalTimes
		campaign.RewardRechargeAmount = rewardRechargeAmount
		campaign.RewardCardQuantity = rewardCardQuantity
		campaign.RewardThreshold = rewardThreshold
		campaign.PromoPrice = promoPrice
		campaign.PromoQuantity = promoQuantity
		campaign.PromoEndsAt = promoEndsAt
		campaign.Status = status

		if id > 0 {
			if err := tx.Save(&campaign).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Create(&campaign).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

func buildPromotionCampaignDetail(db *gorm.DB, campaign *models.PromotionCardCampaign, userID uint, refCode string) (*promotionCampaignDetail, error) {
	if campaign == nil {
		return nil, fmt.Errorf("campaign is nil")
	}
	now := time.Now()
	if err := refreshPromotionCampaignState(db, campaign.ID, now); err != nil {
		return nil, err
	}
	if err := db.Where("id = ?", campaign.ID).First(campaign).Error; err != nil {
		return nil, err
	}

	var template models.CardTemplate
	if err := db.Where("id = ?", campaign.CardTemplateID).First(&template).Error; err != nil {
		return nil, err
	}
	fillCardTemplateProjectIDs(db, &template)

	var merchant models.Merchant
	if err := db.Where("id = ?", campaign.MerchantID).First(&merchant).Error; err != nil {
		return nil, err
	}

	var paymentConfig models.PaymentConfig
	db.Where("merchant_id = ?", campaign.MerchantID).First(&paymentConfig)

	promoActive, promoRemaining, rewardRemaining, err := getPromotionCampaignRemaining(db, campaign.ID, now)
	if err != nil {
		return nil, err
	}
	out := &promotionCampaignDetail{
		ID:                   campaign.ID,
		MerchantID:           campaign.MerchantID,
		CardTemplateID:       campaign.CardTemplateID,
		Title:                campaign.Title,
		RewardTotalTimes:     campaign.RewardTotalTimes,
		RewardRechargeAmount: campaign.RewardRechargeAmount,
		RewardCardQuantity:   campaign.RewardCardQuantity,
		RewardThreshold:      campaign.RewardThreshold,
		PromoPrice:           campaign.PromoPrice,
		PromoQuantity:        campaign.PromoQuantity,
		PromoEndsAt:          campaign.PromoEndsAt,
		Slug:                 campaign.Slug,
		Status:               campaign.Status,
		SharePath:            fmt.Sprintf("/promo/%s", campaign.Slug),
		PromoActive:          promoActive,
		PromoRemaining:       promoRemaining,
		RewardRemaining:      rewardRemaining,
		CanOriginalPurchase:  true,
		CanClaimPromo:        promoActive && promoRemaining > 0,
		CanShare:             rewardRemaining > 0,
		Merchant: gin.H{
			"id":   merchant.ID,
			"name": merchant.Name,
			"type": merchant.Type,
		},
		PaymentConfig: gin.H{
			"has_alipay":     paymentConfig.AlipayQRCode != "",
			"has_wechat":     paymentConfig.WechatQRCode != "",
			"alipay_qr_code": paymentConfig.AlipayQRCode,
			"wechat_qr_code": paymentConfig.WechatQRCode,
			"default_method": paymentConfig.DefaultMethod,
		},
		CardTemplate: gin.H{
			"id":                  template.ID,
			"name":                template.Name,
			"card_type":           template.CardType,
			"price":               template.Price,
			"total_times":         template.TotalTimes,
			"recharge_amount":     template.RechargeAmount,
			"valid_days":          template.ValidDays,
			"description":         template.Description,
			"project_ids":         template.ProjectIDs,
			"support_appointment": template.SupportAppointment,
		},
		CurrentRefCode: refCode,
	}

	if userID > 0 {
		var claim models.PromotionCardClaim
		if err := db.Where("campaign_id = ? AND user_id = ?", campaign.ID, userID).First(&claim).Error; err == nil {
			out.MyClaim = claim
			if claim.Status == promotionCardClaimStatusActive {
				out.CanClaimPromo = false
			}
		}
		var referrer models.PromotionCardReferrer
		if err := db.Where("campaign_id = ? AND user_id = ?", campaign.ID, userID).First(&referrer).Error; err == nil {
			out.MyReferral = referrer
		}
	}
	return out, nil
}

func loadCampaignBySlugForUpdate(tx *gorm.DB, slug string) (*models.PromotionCardCampaign, error) {
	var campaign models.PromotionCardCampaign
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("slug = ?", slug).First(&campaign).Error; err != nil {
		return nil, apiErr{status: http.StatusNotFound, msg: "活动不存在"}
	}
	if campaign.Status != "active" {
		return nil, apiErr{status: http.StatusBadRequest, msg: "活动已停用"}
	}
	return &campaign, nil
}

func validateClaimableCampaign(tx *gorm.DB, campaign *models.PromotionCardCampaign, userID uint, now time.Time) error {
	if campaign.PromoPrice <= 0 || campaign.PromoQuantity <= 0 || campaign.PromoEndsAt == nil {
		return apiErr{status: http.StatusBadRequest, msg: "当前活动未开启促销领取"}
	}
	if !campaign.PromoEndsAt.After(now) {
		return apiErr{status: http.StatusBadRequest, msg: "促销已结束"}
	}

	var existing models.PromotionCardClaim
	if err := tx.Where("campaign_id = ? AND user_id = ?", campaign.ID, userID).First(&existing).Error; err == nil {
		switch existing.Status {
		case promotionCardClaimStatusActive:
			return apiErr{status: http.StatusBadRequest, msg: "您已领取活动，请在24小时内完成付款"}
		case promotionCardClaimStatusPaid, promotionCardClaimStatusConfirmed:
			return apiErr{status: http.StatusBadRequest, msg: "您已参与过该活动"}
		default:
			return apiErr{status: http.StatusBadRequest, msg: "该活动每位用户只能领取一次"}
		}
	}

	promoActive, promoRemaining, _, err := getPromotionCampaignRemaining(tx, campaign.ID, now)
	if err != nil {
		return err
	}
	if !promoActive || promoRemaining <= 0 {
		return apiErr{status: http.StatusBadRequest, msg: "促销资格已领完"}
	}
	return nil
}

func getPromotionCampaignRemaining(db *gorm.DB, campaignID uint, now time.Time) (bool, int, int, error) {
	var campaign models.PromotionCardCampaign
	if err := db.Where("id = ?", campaignID).First(&campaign).Error; err != nil {
		return false, 0, 0, err
	}

	var activeClaims int64
	if err := db.Model(&models.PromotionCardClaim{}).
		Where("campaign_id = ? AND status IN ?", campaignID, []string{promotionCardClaimStatusActive, promotionCardClaimStatusPaid, promotionCardClaimStatusConfirmed}).
		Count(&activeClaims).Error; err != nil {
		return false, 0, 0, err
	}

	var rewardReserved int64
	if err := db.Model(&models.PromotionCardReferrer{}).
		Where("campaign_id = ? AND reward_status IN ?", campaignID, []string{promotionCardRewardStatusClaimable, promotionCardRewardStatusClaimed}).
		Count(&rewardReserved).Error; err != nil {
		return false, 0, 0, err
	}

	promoActive := campaign.Status == "active" && campaign.PromoPrice > 0 && campaign.PromoQuantity > 0 && campaign.PromoEndsAt != nil && campaign.PromoEndsAt.After(now)
	promoRemaining := campaign.PromoQuantity - int(activeClaims)
	if promoRemaining < 0 {
		promoRemaining = 0
	}
	rewardRemaining := campaign.RewardCardQuantity - int(rewardReserved)
	if rewardRemaining < 0 {
		rewardRemaining = 0
	}
	return promoActive, promoRemaining, rewardRemaining, nil
}

func canDeletePromotionCampaign(db *gorm.DB, campaignID uint) (bool, error) {
	var claimCount int64
	if err := db.Model(&models.PromotionCardClaim{}).
		Where("campaign_id = ? AND status IN ?", campaignID, []string{
			promotionCardClaimStatusActive,
			promotionCardClaimStatusPaid,
			promotionCardClaimStatusConfirmed,
		}).
		Count(&claimCount).Error; err != nil {
		return false, err
	}
	if claimCount > 0 {
		return false, nil
	}

	var purchaseCount int64
	if err := db.Model(&models.DirectPurchase{}).
		Where("promotion_campaign_id = ? AND status IN ?", campaignID, []string{
			"paid",
			"confirmed",
			directPurchaseStatusStoreWait,
		}).
		Count(&purchaseCount).Error; err != nil {
		return false, err
	}
	if purchaseCount > 0 {
		return false, nil
	}

	var progressCount int64
	if err := db.Model(&models.PromotionCardReferrer{}).
		Where("campaign_id = ?", campaignID).
		Where("register_count > 0 OR paid_count > 0 OR progress_count > 0").
		Count(&progressCount).Error; err != nil {
		return false, err
	}
	if progressCount > 0 {
		return false, nil
	}

	return true, nil
}

func resolveCampaignReferrer(tx *gorm.DB, campaignID uint, userID uint, refCode string) (*uint, string, error) {
	refCode = strings.TrimSpace(refCode)
	if refCode == "" {
		return nil, "", nil
	}

	var referrer models.PromotionCardReferrer
	if err := tx.Where("campaign_id = ? AND promotion_code = ?", campaignID, refCode).First(&referrer).Error; err != nil {
		return nil, "", apiErr{status: http.StatusBadRequest, msg: "推广码无效"}
	}
	if referrer.UserID == userID {
		return nil, "", nil
	}
	return &referrer.UserID, refCode, nil
}

func getOrCreatePromotionReferrer(tx *gorm.DB, campaignID uint, userID uint, now time.Time) (models.PromotionCardReferrer, error) {
	var referrer models.PromotionCardReferrer
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("campaign_id = ? AND user_id = ?", campaignID, userID).
		First(&referrer).Error; err == nil {
		return referrer, nil
	}

	code, err := newPromotionCode(tx, campaignID)
	if err != nil {
		return referrer, err
	}
	referrer = models.PromotionCardReferrer{
		CampaignID:    campaignID,
		UserID:        userID,
		PromotionCode: code,
	}
	if err := tx.Create(&referrer).Error; err != nil {
		return referrer, err
	}
	return referrer, nil
}

func createPromotionRewardCard(tx *gorm.DB, template *models.CardTemplate, campaign *models.PromotionCardCampaign, userID uint, now time.Time) (models.Card, error) {
	cardNo, err := nextMerchantCardNo(tx, campaign.MerchantID)
	if err != nil {
		return models.Card{}, err
	}

	startDate := now
	var endDate *time.Time
	if template.ValidDays > 0 {
		end := startDate.AddDate(0, 0, template.ValidDays)
		endDate = &end
	} else {
		end := startDate.AddDate(100, 0, 0)
		endDate = &end
	}

	card := models.Card{
		UserID:     userID,
		MerchantID: campaign.MerchantID,
		CardNo:     cardNo,
		CardType:   template.Name,
		RechargeAt: &startDate,
		StartDate:  &startDate,
		EndDate:    endDate,
	}
	if template.CardType == "balance" {
		card.TotalTimes = 0
		card.RemainTimes = 0
		card.RechargeAmount = campaign.RewardRechargeAmount / 100
	} else {
		card.TotalTimes = campaign.RewardTotalTimes
		card.RemainTimes = campaign.RewardTotalTimes
		card.RechargeAmount = 0
	}

	if err := tx.Create(&card).Error; err != nil {
		return models.Card{}, err
	}

	var projectIDs []uint
	tx.Model(&models.CardTemplateProject{}).
		Where("card_template_id = ?", template.ID).
		Order("id asc").
		Pluck("project_id", &projectIDs)
	for _, pid := range projectIDs {
		if pid == 0 {
			continue
		}
		if err := tx.Create(&models.CardProject{CardID: card.ID, ProjectID: pid}).Error; err != nil {
			return models.Card{}, err
		}
	}
	return card, nil
}

func listPromotionRewardCards(db *gorm.DB, userID uint, status string, now time.Time) ([]promotionRewardListItem, error) {
	if db == nil || userID == 0 {
		return []promotionRewardListItem{}, nil
	}

	var campaignIDs []uint
	if err := db.Model(&models.PromotionCardReferrer{}).
		Where("user_id = ?", userID).
		Distinct().
		Pluck("campaign_id", &campaignIDs).Error; err != nil {
		return nil, err
	}
	if len(campaignIDs) == 0 {
		return []promotionRewardListItem{}, nil
	}
	for _, campaignID := range campaignIDs {
		if campaignID == 0 {
			continue
		}
		if err := refreshPromotionCampaignState(db, campaignID, now); err != nil {
			return nil, err
		}
	}

	items := make([]promotionRewardListItem, 0)
	query := db.Table("promotion_card_referrers").
		Select(strings.Join([]string{
			"'promotion_reward' AS list_item_type",
			"promotion_card_referrers.id AS referrer_id",
			"promotion_card_referrers.campaign_id",
			"promotion_card_campaigns.merchant_id",
			"merchants.name AS merchant_name",
			"card_templates.name AS card_name",
			"card_templates.card_type AS card_template_type",
			"promotion_card_campaigns.title",
			"promotion_card_campaigns.slug",
			"promotion_card_campaigns.status AS campaign_status",
			"promotion_card_campaigns.reward_threshold",
			"promotion_card_campaigns.reward_total_times",
			"promotion_card_campaigns.reward_recharge_amount",
			"promotion_card_referrers.register_count",
			"promotion_card_referrers.paid_count",
			"promotion_card_referrers.progress_count",
			"promotion_card_referrers.reward_status",
			"promotion_card_referrers.reward_qualified_at",
			"promotion_card_referrers.reward_expires_at",
			"promotion_card_referrers.reward_claimed_at",
			"promotion_card_referrers.updated_at",
		}, ", ")).
		Joins("JOIN promotion_card_campaigns ON promotion_card_campaigns.id = promotion_card_referrers.campaign_id").
		Joins("JOIN merchants ON merchants.id = promotion_card_campaigns.merchant_id").
		Joins("JOIN card_templates ON card_templates.id = promotion_card_campaigns.card_template_id").
		Where("promotion_card_referrers.user_id = ?", userID).
		Where("COALESCE(promotion_card_referrers.reward_status, '') <> ?", promotionCardRewardStatusClaimed)

	switch status {
	case "expired":
		query = query.Where("(promotion_card_referrers.reward_status = ? OR promotion_card_campaigns.status <> ?)", promotionCardRewardStatusExpired, "active")
	default:
		query = query.Where("promotion_card_campaigns.status = ?", "active")
		query = query.Where("COALESCE(promotion_card_referrers.reward_status, '') IN ?", []string{"", promotionCardRewardStatusClaimable})
	}

	if err := query.
		Order("CASE WHEN promotion_card_referrers.reward_status = 'claimable' THEN 0 ELSE 1 END").
		Order("promotion_card_referrers.updated_at DESC").
		Scan(&items).Error; err != nil {
		return nil, err
	}
	if items == nil {
		return []promotionRewardListItem{}, nil
	}
	return items, nil
}

func refreshPromotionCampaignState(db *gorm.DB, campaignID uint, now time.Time) error {
	if db == nil || campaignID == 0 {
		return nil
	}

	if err := db.Model(&models.PromotionCardClaim{}).
		Where("campaign_id = ? AND status = ? AND expires_at IS NOT NULL AND expires_at <= ?", campaignID, promotionCardClaimStatusActive, now).
		Updates(map[string]interface{}{
			"status":     promotionCardClaimStatusExpired,
			"expired_at": now,
		}).Error; err != nil {
		return err
	}

	if err := db.Model(&models.PromotionCardReferrer{}).
		Where("campaign_id = ? AND reward_status = ? AND reward_expires_at IS NOT NULL AND reward_expires_at <= ?", campaignID, promotionCardRewardStatusClaimable, now).
		Updates(map[string]interface{}{
			"reward_status":     promotionCardRewardStatusExpired,
			"reward_expired_at": now,
		}).Error; err != nil {
		return err
	}
	return nil
}

func maybeCountPromotionRegistration(tx *gorm.DB, campaignSlug string, refCode string, inviteeUserID uint, now time.Time) error {
	campaignSlug = strings.TrimSpace(campaignSlug)
	refCode = strings.TrimSpace(refCode)
	if campaignSlug == "" || refCode == "" || inviteeUserID == 0 {
		return nil
	}

	campaign, err := loadCampaignBySlugForUpdate(tx, campaignSlug)
	if err != nil {
		return nil
	}
	if err := refreshPromotionCampaignState(tx, campaign.ID, now); err != nil {
		return err
	}

	var referrer models.PromotionCardReferrer
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("campaign_id = ? AND promotion_code = ?", campaign.ID, refCode).
		First(&referrer).Error; err != nil {
		return nil
	}
	if referrer.UserID == inviteeUserID {
		return nil
	}

	var referral models.PromotionCardReferral
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("campaign_id = ? AND referrer_user_id = ? AND invitee_user_id = ?", campaign.ID, referrer.UserID, inviteeUserID).
		First(&referral).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		referral = models.PromotionCardReferral{
			CampaignID:     campaign.ID,
			ReferrerUserID: referrer.UserID,
			InviteeUserID:  inviteeUserID,
			PromotionCode:  refCode,
		}
		if err := tx.Create(&referral).Error; err != nil {
			return err
		}
	}
	if referral.RegisteredCountedAt != nil {
		return nil
	}

	referral.RegisteredCountedAt = &now
	if err := tx.Save(&referral).Error; err != nil {
		return err
	}
	referrer.RegisterCount++
	referrer.ProgressCount++
	if err := maybeQualifyPromotionReward(tx, &referrer, campaign, now); err != nil {
		return err
	}
	return tx.Save(&referrer).Error
}

func maybeCountPromotionPayment(tx *gorm.DB, purchase *models.DirectPurchase, now time.Time) error {
	if purchase == nil || purchase.PromotionCampaignID == nil || purchase.ReferrerUserID == nil || purchase.UserID == 0 {
		return nil
	}

	if err := refreshPromotionCampaignState(tx, *purchase.PromotionCampaignID, now); err != nil {
		return err
	}

	var campaign models.PromotionCardCampaign
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", *purchase.PromotionCampaignID).
		First(&campaign).Error; err != nil {
		return err
	}

	var referrer models.PromotionCardReferrer
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("campaign_id = ? AND user_id = ?", campaign.ID, *purchase.ReferrerUserID).
		First(&referrer).Error; err != nil {
		return nil
	}
	if referrer.UserID == purchase.UserID {
		return nil
	}

	var referral models.PromotionCardReferral
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("campaign_id = ? AND referrer_user_id = ? AND invitee_user_id = ?", campaign.ID, referrer.UserID, purchase.UserID).
		First(&referral).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		referral = models.PromotionCardReferral{
			CampaignID:     campaign.ID,
			ReferrerUserID: referrer.UserID,
			InviteeUserID:  purchase.UserID,
			PromotionCode:  purchase.PromotionCode,
		}
		if err := tx.Create(&referral).Error; err != nil {
			return err
		}
	}
	if referral.PaidCountedAt != nil {
		return nil
	}

	referral.PaidCountedAt = &now
	if err := tx.Save(&referral).Error; err != nil {
		return err
	}
	referrer.PaidCount++
	referrer.ProgressCount++
	if err := maybeQualifyPromotionReward(tx, &referrer, &campaign, now); err != nil {
		return err
	}
	return tx.Save(&referrer).Error
}

func maybeQualifyPromotionReward(tx *gorm.DB, referrer *models.PromotionCardReferrer, campaign *models.PromotionCardCampaign, now time.Time) error {
	if referrer == nil || campaign == nil {
		return nil
	}
	if referrer.RewardStatus != "" {
		return nil
	}
	if referrer.ProgressCount < campaign.RewardThreshold {
		return nil
	}

	_, _, rewardRemaining, err := getPromotionCampaignRemaining(tx, campaign.ID, now)
	if err != nil {
		return err
	}
	if rewardRemaining <= 0 {
		return nil
	}

	expiresAt := now.Add(72 * time.Hour)
	referrer.RewardStatus = promotionCardRewardStatusClaimable
	referrer.RewardQualifiedAt = &now
	referrer.RewardExpiresAt = &expiresAt
	return nil
}

func newPromotionCampaignSlug(tx *gorm.DB, merchantID uint, templateID uint) (string, error) {
	for i := 0; i < 8; i++ {
		code, err := randomDigits(6)
		if err != nil {
			return "", err
		}
		slug := fmt.Sprintf("%d-%d-%s", merchantID, templateID, code)
		var count int64
		if err := tx.Model(&models.PromotionCardCampaign{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return slug, nil
		}
	}
	return "", fmt.Errorf("生成活动链接失败")
}

func normalizePromotionCampaignSlug(slug string, merchantID uint, templateID uint) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return slug
	}
	prefix := fmt.Sprintf("promo-%d-%d-", merchantID, templateID)
	if strings.HasPrefix(slug, prefix) {
		return strings.TrimPrefix(slug, "promo-")
	}
	return slug
}

func newPromotionCode(tx *gorm.DB, campaignID uint) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := randomDigits(6)
		if err != nil {
			return "", err
		}
		var count int64
		if err := tx.Model(&models.PromotionCardReferrer{}).
			Where("campaign_id = ? AND promotion_code = ?", campaignID, code).
			Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
	}
	return "", fmt.Errorf("生成推广码失败")
}

func randomDigits(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid length")
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i := range buf {
		out[i] = byte('0' + (buf[i] % 10))
	}
	return string(out), nil
}

func renderPromotionErr(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if ae, ok := err.(apiErr); ok {
		c.JSON(ae.status, gin.H{"error": ae.msg})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func optionalUserIDFromAuthHeader(c *gin.Context) uint {
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	if raw == "" {
		return 0
	}
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	if raw == "" {
		return 0
	}
	claims, ok := parsePromotionUserJWT(raw)
	if !ok {
		return 0
	}
	uidAny, ok := claims["user_id"]
	if !ok {
		return 0
	}
	switch v := uidAny.(type) {
	case float64:
		return uint(v)
	case int:
		return uint(v)
	case int64:
		return uint(v)
	case string:
		n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0
		}
		return uint(n)
	default:
		return 0
	}
}

func parsePromotionUserJWT(token string) (jwt.MapClaims, bool) {
	secret := strings.TrimSpace(config.UserJWTSecret())
	if secret == "" {
		return nil, false
	}
	parsed, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, false
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, false
	}
	typeVal, _ := claims["type"].(string)
	if typeVal != "user" {
		return nil, false
	}
	expAny, ok := claims["exp"]
	if !ok {
		return nil, false
	}
	var exp int64
	switch v := expAny.(type) {
	case float64:
		exp = int64(v)
	case int:
		exp = int64(v)
	case int64:
		exp = v
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return nil, false
		}
		exp = n
	default:
		return nil, false
	}
	if exp <= time.Now().Unix() {
		return nil, false
	}
	return claims, true
}
