package handlers

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// MerchantRegister 商户注册
func MerchantRegister(c *gin.Context) {
	var input struct {
		Phone      string `json:"phone" binding:"required"`
		Password   string `json:"password" binding:"required,min=6"`
		Name       string `json:"name" binding:"required"`
		Type       string `json:"type"`
		Code       string `json:"code" binding:"required"`
		InviteCode string `json:"invite_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查手机号是否已注册
	var existingMerchant models.Merchant
	if err := config.DB.Where("phone = ?", input.Phone).First(&existingMerchant).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该手机号已注册"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := consumeSMSCode(tx, input.Phone, "merchant_register", input.Code); err != nil {
			return err
		}

		var invite models.InviteCode
		if err := tx.Where("code = ? AND used = ?", input.InviteCode, false).First(&invite).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("邀请码无效或已使用")
			}
			return err
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		merchant = models.Merchant{
			Phone:     input.Phone,
			Password:  string(hashedPassword),
			Name:      input.Name,
			Type:      input.Type,
			CreatedAt: func() *time.Time { t := time.Now(); return &t }(),
		}
		if err := tx.Create(&merchant).Error; err != nil {
			return err
		}

		usedAt := time.Now()
		updates := map[string]interface{}{
			"used":                true,
			"used_at":             &usedAt,
			"used_by_merchant_id": merchant.ID,
		}
		res := tx.Model(&models.InviteCode{}).
			Where("code = ? AND used = ?", input.InviteCode, false).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("邀请码无效或已使用")
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("商户注册成功: ID=%d, 手机号=%s, 名称=%s", merchant.ID, merchant.Phone, merchant.Name)

	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
		"data": gin.H{
			"id":    merchant.ID,
			"phone": merchant.Phone,
			"name":  merchant.Name,
		},
	})
}

// MerchantLogin 商户登录
func MerchantLogin(c *gin.Context) {
	var input struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var merchant models.Merchant
	if err := config.DB.Where("phone = ?", input.Phone).First(&merchant).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "手机号或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(merchant.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "手机号或密码错误"})
		return
	}

	// 生成 JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"merchant_id": merchant.ID,
		"phone":       merchant.Phone,
		"type":        "merchant",
		"exp":         time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	tokenString, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		log.Printf("生成token失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败"})
		return
	}

	log.Printf("商户登录成功: ID=%d, 手机号=%s", merchant.ID, merchant.Phone)

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"merchant": gin.H{
			"id":    merchant.ID,
			"phone": merchant.Phone,
			"name":  merchant.Name,
			"type":  merchant.Type,
		},
	})
}

func GetMerchants(c *gin.Context) {
	var merchants []models.Merchant
	config.DB.Find(&merchants)
	c.JSON(http.StatusOK, gin.H{"data": merchants})
}

func GetMerchant(c *gin.Context) {
	id := c.Param("id")
	var merchant models.Merchant
	if err := config.DB.First(&merchant, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func UpdateCurrentMerchantServices(c *gin.Context) {
	merchantIDAny, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	var input struct {
		SupportAppointment *bool   `json:"support_appointment"`
		SupportQueue       *bool   `json:"support_queue"`
		QueuePrefix        *string `json:"queue_prefix"`
		QueueStartNo       *int    `json:"queue_start_no"`
		SupportDirectSale  *bool   `json:"support_direct_sale"`
		AvgServiceMinutes  *int    `json:"avg_service_minutes"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.SupportAppointment != nil {
		updates["support_appointment"] = *input.SupportAppointment
	}
	if input.SupportQueue != nil {
		updates["support_queue"] = *input.SupportQueue
	}
	if input.QueuePrefix != nil {
		updates["queue_prefix"] = strings.TrimSpace(*input.QueuePrefix)
	}
	if input.QueueStartNo != nil {
		if *input.QueueStartNo < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "叫号起始号码必须大于等于1"})
			return
		}
		updates["queue_start_no"] = *input.QueueStartNo
	}
	if input.SupportDirectSale != nil {
		updates["support_direct_sale"] = *input.SupportDirectSale
	}
	if input.AvgServiceMinutes != nil {
		if *input.AvgServiceMinutes < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "平均服务时长必须大于等于1"})
			return
		}
		updates["avg_service_minutes"] = *input.AvgServiceMinutes
	}

	if len(updates) == 0 {
		config.DB.First(&merchant, merchantID)
		c.JSON(http.StatusOK, gin.H{"data": merchant})
		return
	}

	if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	config.DB.First(&merchant, merchantID)
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func CreateMerchant(c *gin.Context) {
	var merchant models.Merchant
	if err := c.ShouldBindJSON(&merchant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.DB.Create(&merchant)
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func UpdateMerchant(c *gin.Context) {
	id := c.Param("id")
	var merchant models.Merchant
	if err := config.DB.First(&merchant, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	var input struct {
		Name               string `json:"name"`
		Type               string `json:"type"`
		SupportAppointment *bool  `json:"support_appointment"`
		AvgServiceMinutes  *int   `json:"avg_service_minutes"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Type != "" {
		updates["type"] = input.Type
	}
	if input.SupportAppointment != nil {
		updates["support_appointment"] = *input.SupportAppointment
	}
	if input.AvgServiceMinutes != nil {
		updates["avg_service_minutes"] = *input.AvgServiceMinutes
	}

	config.DB.Model(&merchant).Updates(updates)
	config.DB.First(&merchant, id)
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func GetCurrentUserMerchant(c *gin.Context) {
	merchantID, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func BindMerchantPhone(c *gin.Context) {
	merchantIDAny, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		Phone    string `json:"phone" binding:"required"`
		Code     string `json:"code" binding:"required"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Phone = strings.TrimSpace(input.Phone)
	input.Code = strings.TrimSpace(input.Code)
	input.Password = strings.TrimSpace(input.Password)
	if input.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供手机号"})
		return
	}
	if input.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入验证码"})
		return
	}

	merchantID, _ := merchantIDAny.(uint)
	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 如果商户已有手机号（换绑场景），需要验证密码
	if merchant.Phone != "" && merchant.Phone != input.Phone {
		if input.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "换绑手机号需要输入商户密码"})
			return
		}

		// 验证密码
		if err := bcrypt.CompareHashAndPassword([]byte(merchant.Password), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
			return
		}
	}

	// 检查新手机号是否已被其他商户使用
	var existingByPhone models.Merchant
	if err := config.DB.Where("phone = ? AND id != ?", input.Phone, merchantID).First(&existingByPhone).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该手机号已被其他商户绑定"})
		return
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := consumeSMSCode(tx, input.Phone, "merchant_bind_phone", input.Code); err != nil {
			return err
		}
		return tx.Model(&models.Merchant{}).Where("id = ?", merchantID).Update("phone", input.Phone).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var updated models.Merchant
	if err := config.DB.First(&updated, merchantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// UpdateMerchantInfo 更新商户信息（营业时间和地址）
func UpdateMerchantInfo(c *gin.Context) {
	merchantIDAny, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	var input struct {
		MorningStart   *string `json:"morning_start"`
		MorningEnd     *string `json:"morning_end"`
		AfternoonStart *string `json:"afternoon_start"`
		AfternoonEnd   *string `json:"afternoon_end"`
		EveningStart   *string `json:"evening_start"`
		EveningEnd     *string `json:"evening_end"`
		AllDayStart    *string `json:"all_day_start"`
		AllDayEnd      *string `json:"all_day_end"`
		Province       *string `json:"province"`
		City           *string `json:"city"`
		District       *string `json:"district"`
		Address        *string `json:"address"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.MorningStart != nil {
		updates["morning_start"] = strings.TrimSpace(*input.MorningStart)
	}
	if input.MorningEnd != nil {
		updates["morning_end"] = strings.TrimSpace(*input.MorningEnd)
	}
	if input.AfternoonStart != nil {
		updates["afternoon_start"] = strings.TrimSpace(*input.AfternoonStart)
	}
	if input.AfternoonEnd != nil {
		updates["afternoon_end"] = strings.TrimSpace(*input.AfternoonEnd)
	}
	if input.EveningStart != nil {
		updates["evening_start"] = strings.TrimSpace(*input.EveningStart)
	}
	if input.EveningEnd != nil {
		updates["evening_end"] = strings.TrimSpace(*input.EveningEnd)
	}
	if input.AllDayStart != nil {
		updates["all_day_start"] = strings.TrimSpace(*input.AllDayStart)
	}
	if input.AllDayEnd != nil {
		updates["all_day_end"] = strings.TrimSpace(*input.AllDayEnd)
	}
	if input.Province != nil {
		updates["province"] = strings.TrimSpace(*input.Province)
	}
	if input.City != nil {
		updates["city"] = strings.TrimSpace(*input.City)
	}
	if input.District != nil {
		updates["district"] = strings.TrimSpace(*input.District)
	}
	if input.Address != nil {
		updates["address"] = strings.TrimSpace(*input.Address)
	}

	if len(updates) == 0 {
		config.DB.First(&merchant, merchantID)
		c.JSON(http.StatusOK, gin.H{"data": merchant})
		return
	}

	if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	config.DB.First(&merchant, merchantID)
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}
