package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
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

func nextMerchantCardNo(tx *gorm.DB, merchantID uint) (string, error) {
	var last string
	err := tx.Raw(
		"SELECT card_no FROM cards WHERE merchant_id = ? AND card_no REGEXP '^[0-9]{5}$' ORDER BY card_no DESC LIMIT 1 FOR UPDATE",
		merchantID,
	).Scan(&last).Error
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(last) == "" {
		return fmt.Sprintf("%05d", 1), nil
	}
	seq, err := strconv.Atoi(last)
	if err != nil {
		// 如果历史数据不符合预期，退化到1
		return fmt.Sprintf("%05d", 1), nil
	}
	seq++
	if seq < 1 {
		seq = 1
	}
	return fmt.Sprintf("%05d", seq), nil
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

func GetNextMerchantCardNo(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var next string
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		v, err := nextMerchantCardNo(tx, merchantID)
		if err != nil {
			return err
		}
		next = v
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"card_no": next}})
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

	phone := c.Query("phone")
	nickname := c.Query("nickname")
	cardNo := c.Query("card_no")
	cardType := c.Query("card_type")
	userIDStr := strings.TrimSpace(c.Query("user_id"))
	userCode := strings.TrimSpace(c.Query("user_code"))

	var cards []models.Card
	query := config.DB.
		Model(&models.Card{}).
		Joins("LEFT JOIN users ON users.id = cards.user_id").
		Where("cards.merchant_id = ?", merchantID)

	if userIDStr != "" {
		uid, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil || uid == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id 参数错误"})
			return
		}
		query = query.Where("cards.user_id = ?", uint(uid))
	} else if userCode != "" {
		uid, err := parseUserCodeToUserID(userCode)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		query = query.Where("cards.user_id = ?", uid)
	}

	if phone != "" {
		query = query.Where("users.phone LIKE ?", "%"+phone+"%")
	}
	if nickname != "" {
		query = query.Where("users.nickname LIKE ?", "%"+nickname+"%")
	}
	if cardNo != "" {
		query = query.Where("cards.card_no LIKE ?", "%"+cardNo+"%")
	}
	if cardType != "" {
		query = query.Where("cards.card_type LIKE ?", "%"+cardType+"%")
	}

	query.
		Select("cards.*").
		Preload("User").
		Order("cards.id desc").
		Find(&cards)
	c.JSON(http.StatusOK, gin.H{"data": cards})
}

func parseUserCodeToUserID(code string) (uint, error) {
	v := strings.TrimSpace(code)
	if v == "" {
		return 0, errors.New("用户码无效")
	}
	if !strings.HasPrefix(v, "kabao-user:") {
		return 0, errors.New("用户码格式不正确")
	}
	parts := strings.Split(v, ":")
	if len(parts) != 4 {
		return 0, errors.New("用户码格式不正确")
	}
	uidStr := strings.TrimSpace(parts[1])
	expStr := strings.TrimSpace(parts[2])
	sig := strings.TrimSpace(parts[3])
	uid64, err := strconv.ParseUint(uidStr, 10, 32)
	if err != nil || uid64 == 0 {
		return 0, errors.New("用户码无效")
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || exp <= 0 {
		return 0, errors.New("用户码无效")
	}
	if time.Now().Unix() > exp {
		return 0, errors.New("用户码已过期，请用户刷新后重试")
	}

	msg := uidStr + ":" + expStr
	mac := hmac.New(sha256.New, []byte("your-secret-key"))
	mac.Write([]byte(msg))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(sig))) {
		return 0, errors.New("用户码无效")
	}
	return uint(uid64), nil
}

func GetMerchantCard(c *gin.Context) {
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

	cardID := c.Param("id")
	var card models.Card
	if err := config.DB.Preload("User").Preload("Merchant").First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}

	if card.MerchantID != merchantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看此卡"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": card})
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
		TotalTimes     int    `json:"total_times"`
		RechargeAmount int    `json:"recharge_amount"`
		StartDate      string `json:"start_date"`
		EndDate        string `json:"end_date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.TotalTimes <= 0 && input.RechargeAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "总次数与充值金额不能同时为空"})
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
	var card models.Card
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		cardNo := strings.TrimSpace(input.CardNo)
		if cardNo == "" {
			v, err := nextMerchantCardNo(tx, merchantID)
			if err != nil {
				return err
			}
			cardNo = v
		}

		remain := 0
		if input.TotalTimes > 0 {
			remain = input.TotalTimes
		}

		card = models.Card{
			UserID:         input.UserID,
			MerchantID:     merchantID,
			CardNo:         cardNo,
			CardType:       input.CardType,
			TotalTimes:     input.TotalTimes,
			RemainTimes:    remain,
			UsedTimes:      0,
			RechargeAmount: input.RechargeAmount,
			RechargeAt:     dateOnlyPtr(now),
			StartDate:      startDate,
			EndDate:        endDate,
		}
		if card.StartDate == nil {
			card.StartDate = dateOnlyPtr(now)
		}

		return tx.Create(&card).Error
	}); err != nil {
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

	// 用户端仅允许生成自己的卡片核销码
	if userIDAny, ok := c.Get("user_id"); ok {
		if userID, ok2 := userIDAny.(uint); ok2 && userID > 0 {
			if card.UserID != userID {
				c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此卡"})
				return
			}
		}
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

	// 优先复用未过期且未使用的核销码（12小时有效）
	var existing models.VerifyCode
	if err := config.DB.
		Where("card_id = ? AND used = ? AND expire_at > ?", card.ID, false, now.Unix()).
		Order("id desc").
		First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"code":      existing.Code,
				"expire_at": existing.ExpireAt,
				"card_id":   card.ID,
			},
		})
		return
	}

	code := uuid.New().String()[:8]
	expireAt := now.Add(12 * time.Hour).Unix()

	verifyCode := models.VerifyCode{CardID: card.ID, Code: code, ExpireAt: expireAt, Used: false}
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
	var merchant models.Merchant
	var usedAt time.Time
	var remainTimes int
	usageStatus := "success"

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		if err := tx.First(&merchant, merchantID).Error; err != nil {
			return apiErr{status: http.StatusNotFound, msg: "商户不存在"}
		}
		if merchant.SupportCustomerService {
			usageStatus = "in_progress"
		}

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
			Updates(map[string]interface{}{"used": true, "used_at": now})
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
			CardID:             card.ID,
			MerchantID:         card.MerchantID,
			UsedTimes:          1,
			UsedAt:             &usedAt,
			VerifyCode:         verifyCode.Code,
			VerifyCodeExpireAt: verifyCode.ExpireAt,
			Status:             usageStatus,
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

func FinishVerifyCard(c *gin.Context) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "technician" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅技师账号可结单"})
		return
	}

	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	techIDAny, ok := c.Get("technician_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	techID, ok := techIDAny.(uint)
	if !ok || techID == 0 {
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

	var usage models.Usage
	var verifyCode models.VerifyCode
	var finishedAt time.Time

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", input.Code).First(&verifyCode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "核销码不存在"}
			}
			return err
		}
		if verifyCode.ExpireAt > 0 && now.Unix() > verifyCode.ExpireAt {
			return apiErr{status: http.StatusBadRequest, msg: "核销码已过期"}
		}
		if !verifyCode.Used {
			return apiErr{status: http.StatusBadRequest, msg: "该核销码尚未核销"}
		}

		// 找到对应使用记录（同卡同码，取最新）
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND card_id = ? AND verify_code = ?", merchantID, verifyCode.CardID, verifyCode.Code).
			Order("id desc").
			First(&usage).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "找不到核销记录"}
			}
			return err
		}

		if usage.Status == "success" {
			return apiErr{status: http.StatusBadRequest, msg: "该记录已结单"}
		}
		if usage.Status != "in_progress" {
			return apiErr{status: http.StatusBadRequest, msg: "该记录不可结单"}
		}

		finishedAt = now
		return tx.Model(&models.Usage{}).Where("id = ?", usage.ID).Updates(map[string]interface{}{
			"technician_id": techID,
			"finished_at":   finishedAt,
			"status":        "success",
		}).Error
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

	config.DB.Preload("Technician").First(&usage, usage.ID)
	c.JSON(http.StatusOK, gin.H{
		"message": "结单成功",
		"data": gin.H{
			"usage_id":      usage.ID,
			"card_id":       usage.CardID,
			"finished_at":   finishedAt.Format("2006-01-02 15:04:05"),
			"technician_id": usage.TechnicianID,
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
