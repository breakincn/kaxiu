package handlers

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type apiErr struct {
	status int
	msg    string
}

func (e apiErr) Error() string { return e.msg }

func parseDatePtr(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", v, time.Local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func dateOnlyPtr(t time.Time) *time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return &d
}

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

	now := time.Now()
	if status == "active" {
		query = query.Where("end_date >= ? AND remain_times > 0", dateOnlyPtr(now))
	} else if status == "expired" {
		query = query.Where("end_date < ? OR remain_times = 0", dateOnlyPtr(now))
	}

	query.Find(&cards)
	c.JSON(http.StatusOK, gin.H{"data": cards})
}

func GetMerchantCards(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可查看"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	var cards []models.Card
	config.DB.Preload("User").Where("merchant_id = ?", merchantID).Order("id desc").Find(&cards)
	c.JSON(http.StatusOK, gin.H{"data": cards})
}

func CreateCard(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可发卡"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		UserID         uint   `json:"user_id" binding:"required"`
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

	if input.TotalTimes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "总次数必须大于0"})
		return
	}

	var user models.User
	if err := config.DB.First(&user, input.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户不存在"})
		return
	}

	startDate, err := parseDatePtr(input.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "开始日期格式错误"})
		return
	}
	endDate, err := parseDatePtr(input.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期格式错误"})
		return
	}
	if endDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期不能为空"})
		return
	}

	now := time.Now()
	cardNo := input.CardNo
	if cardNo == "" {
		cardNo = uuid.New().String()[:8]
	}
	card := models.Card{
		UserID:         input.UserID,
		MerchantID:     merchantID,
		CardNo:         cardNo,
		CardType:       input.CardType,
		TotalTimes:     input.TotalTimes,
		RemainTimes:    input.TotalTimes,
		UsedTimes:      0,
		RechargeAmount: input.RechargeAmount,
		RechargeAt:     dateOnlyPtr(now),
		StartDate:      startDate,
		EndDate:        endDate,
	}
	if card.StartDate == nil {
		card.StartDate = dateOnlyPtr(now)
	}

	if err := config.DB.Create(&card).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
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
		updates["recharge_at"] = dateOnlyPtr(time.Now())
	}
	if input.EndDate != "" {
		endDate, err := parseDatePtr(input.EndDate)
		if err != nil || endDate == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期格式错误"})
			return
		}
		updates["end_date"] = endDate
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
	if card.EndDate != nil && now.After(*card.EndDate) {
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
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可核销"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var verifyCode models.VerifyCode
	var card models.Card
	var usedAt time.Time
	var remainTimes int

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", input.Code).First(&verifyCode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "核销码不存在"}
			}
			return err
		}

		if verifyCode.Used {
			return apiErr{status: http.StatusBadRequest, msg: "核销码已使用"}
		}

		if now.Unix() > verifyCode.ExpireAt {
			return apiErr{status: http.StatusBadRequest, msg: "核销码已过期"}
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, verifyCode.CardID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "卡片不存在"}
			}
			return err
		}

		if card.MerchantID != merchantID {
			return apiErr{status: http.StatusForbidden, msg: "无权核销此卡"}
		}

		if card.EndDate != nil && now.After(*card.EndDate) {
			return apiErr{status: http.StatusBadRequest, msg: "卡片已过期"}
		}
		if card.RemainTimes <= 0 {
			return apiErr{status: http.StatusBadRequest, msg: "剩余次数不足"}
		}

		res := tx.Model(&models.VerifyCode{}).
			Where("id = ? AND used = ?", verifyCode.ID, false).
			Update("used", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apiErr{status: http.StatusBadRequest, msg: "核销码已使用"}
		}

		usedAt = now
		res = tx.Model(&models.Card{}).
			Where("id = ? AND remain_times > 0", card.ID).
			Updates(map[string]interface{}{
				"remain_times": gorm.Expr("remain_times - ?", 1),
				"used_times":   gorm.Expr("used_times + ?", 1),
				"last_used_at": usedAt,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apiErr{status: http.StatusBadRequest, msg: "剩余次数不足"}
		}

		if err := tx.First(&card, card.ID).Error; err != nil {
			return err
		}
		remainTimes = card.RemainTimes

		usage := models.Usage{
			CardID:     card.ID,
			MerchantID: card.MerchantID,
			UsedTimes:  1,
			UsedAt:     &usedAt,
			Status:     "success",
		}
		if err := tx.Create(&usage).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "核销成功",
		"data": gin.H{
			"card_id":      card.ID,
			"remain_times": remainTimes,
			"used_at":      usedAt.Format("2006-01-02 15:04:05"),
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
