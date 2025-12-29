package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetCards(c *gin.Context) {
	var cards []models.Card
	config.DB.Preload("User").Preload("Merchant").Find(&cards)
	c.JSON(http.StatusOK, gin.H{"data": cards})
}

func GetCard(c *gin.Context) {
	id := c.Param("id")
	var card models.Card
	if err := config.DB.Preload("User").Preload("Merchant").First(&card, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": card})
}

func GetUserCards(c *gin.Context) {
	userID := c.Param("id")
	status := c.Query("status")

	var cards []models.Card
	query := config.DB.Preload("Merchant").Where("user_id = ?", userID)

	now := time.Now().Format("2006-01-02")
	if status == "active" {
		query = query.Where("end_date >= ? AND remain_times > 0", now)
	} else if status == "expired" {
		query = query.Where("end_date < ? OR remain_times = 0", now)
	}

	query.Find(&cards)
	c.JSON(http.StatusOK, gin.H{"data": cards})
}

func GetMerchantCards(c *gin.Context) {
	merchantID := c.Param("id")
	var cards []models.Card
	config.DB.Preload("User").Where("merchant_id = ?", merchantID).Find(&cards)
	c.JSON(http.StatusOK, gin.H{"data": cards})
}

func CreateCard(c *gin.Context) {
	var input struct {
		UserID         uint   `json:"user_id" binding:"required"`
		MerchantID     uint   `json:"merchant_id" binding:"required"`
		CardNo         string `json:"card_no"`
		CardType       string `json:"card_type" binding:"required"`
		TotalTimes     int    `json:"total_times" binding:"required"`
		RechargeAmount int    `json:"recharge_amount"`
		StartDate      string `json:"start_date"`
		EndDate        string `json:"end_date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().Format("2006-01-02")
	card := models.Card{
		UserID:         input.UserID,
		MerchantID:     input.MerchantID,
		CardNo:         input.CardNo,
		CardType:       input.CardType,
		TotalTimes:     input.TotalTimes,
		RemainTimes:    input.TotalTimes,
		UsedTimes:      0,
		RechargeAmount: input.RechargeAmount,
		RechargeAt:     now,
		StartDate:      input.StartDate,
		EndDate:        input.EndDate,
	}

	if card.StartDate == "" {
		card.StartDate = now
	}

	config.DB.Create(&card)
	config.DB.Preload("User").Preload("Merchant").First(&card, card.ID)
	c.JSON(http.StatusOK, gin.H{"data": card})
}

func UpdateCard(c *gin.Context) {
	id := c.Param("id")
	var card models.Card
	if err := config.DB.First(&card, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}

	var input struct {
		TotalTimes     *int   `json:"total_times"`
		RemainTimes    *int   `json:"remain_times"`
		RechargeAmount *int   `json:"recharge_amount"`
		EndDate        string `json:"end_date"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.TotalTimes != nil {
		updates["total_times"] = *input.TotalTimes
	}
	if input.RemainTimes != nil {
		updates["remain_times"] = *input.RemainTimes
	}
	if input.RechargeAmount != nil {
		updates["recharge_amount"] = *input.RechargeAmount
		updates["recharge_at"] = time.Now().Format("2006-01-02")
	}
	if input.EndDate != "" {
		updates["end_date"] = input.EndDate
	}

	config.DB.Model(&card).Updates(updates)
	config.DB.Preload("User").Preload("Merchant").First(&card, id)
	c.JSON(http.StatusOK, gin.H{"data": card})
}

func GenerateVerifyCode(c *gin.Context) {
	cardID := c.Param("id")
	var card models.Card
	if err := config.DB.First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}

	// 检查卡片是否有效
	now := time.Now()
	endDate, _ := time.Parse("2006-01-02", card.EndDate)
	if now.After(endDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片已过期"})
		return
	}

	if card.RemainTimes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "剩余次数不足"})
		return
	}

	// 生成核销码（5分钟有效）
	code := uuid.New().String()[:8]
	expireAt := now.Add(5 * time.Minute).Unix()

	verifyCode := models.VerifyCode{
		CardID:   card.ID,
		Code:     code,
		ExpireAt: expireAt,
		Used:     false,
	}
	config.DB.Create(&verifyCode)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"code":      code,
			"expire_at": expireAt,
			"card_id":   card.ID,
		},
	})
}

func VerifyCard(c *gin.Context) {
	var input struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var verifyCode models.VerifyCode
	if err := config.DB.Where("code = ?", input.Code).First(&verifyCode).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "核销码不存在"})
		return
	}

	if verifyCode.Used {
		c.JSON(http.StatusBadRequest, gin.H{"error": "核销码已使用"})
		return
	}

	if time.Now().Unix() > verifyCode.ExpireAt {
		c.JSON(http.StatusBadRequest, gin.H{"error": "核销码已过期"})
		return
	}

	var card models.Card
	config.DB.Preload("Merchant").First(&card, verifyCode.CardID)

	if card.RemainTimes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "剩余次数不足"})
		return
	}

	// 扣减次数
	now := time.Now().Format("2006-01-02 15:04:05")
	config.DB.Model(&card).Updates(map[string]interface{}{
		"remain_times": card.RemainTimes - 1,
		"used_times":   card.UsedTimes + 1,
		"last_used_at": now,
	})

	// 标记核销码已使用
	config.DB.Model(&verifyCode).Update("used", true)

	// 创建使用记录
	usage := models.Usage{
		CardID:     card.ID,
		MerchantID: card.MerchantID,
		UsedTimes:  1,
		UsedAt:     now,
		Status:     "success",
	}
	config.DB.Create(&usage)

	c.JSON(http.StatusOK, gin.H{
		"message": "核销成功",
		"data": gin.H{
			"card_id":      card.ID,
			"remain_times": card.RemainTimes - 1,
			"used_at":      now,
		},
	})
}

func GetTodayVerify(c *gin.Context) {
	merchantID := c.Param("id")
	today := time.Now().Format("2006-01-02")

	var count int64
	config.DB.Model(&models.Usage{}).
		Where("merchant_id = ? AND DATE(used_at) = ?", merchantID, today).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"count": count}})
}
