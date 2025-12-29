package handlers

import (
	"kabao/config"
	"kabao/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// MerchantRegister 商户注册
func MerchantRegister(c *gin.Context) {
	var input struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name" binding:"required"`
		Type     string `json:"type"`
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

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	merchant := models.Merchant{
		Phone:     input.Phone,
		Password:  string(hashedPassword),
		Name:      input.Name,
		Type:      input.Type,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := config.DB.Create(&merchant).Error; err != nil {
		log.Printf("创建商户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
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
