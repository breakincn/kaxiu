package handlers

import (
	"crypto/rand"
	"errors"
	"kabao/config"
	"kabao/models"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SendSMSCode(c *gin.Context) {
	var input struct {
		Phone   string `json:"phone" binding:"required"`
		Purpose string `json:"purpose" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供手机号和用途"})
		return
	}

	var last models.SMSCode
	if err := config.DB.
		Where("phone = ? AND purpose = ?", input.Phone, input.Purpose).
		Order("id DESC").
		First(&last).Error; err == nil {
		if last.CreatedAt != nil {
			if time.Since(*last.CreatedAt) < 60*time.Second {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "发送过于频繁，请稍后再试"})
				return
			}
		}
	}

	code, err := generateDigits(6)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证码生成失败"})
		return
	}

	expiresAt := time.Now().Add(5 * time.Minute).Unix()
	rec := models.SMSCode{
		Phone:     input.Phone,
		Purpose:   input.Purpose,
		Code:      code,
		ExpiresAt: expiresAt,
		Used:      false,
	}
	if err := config.DB.Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送失败"})
		return
	}

	resp := gin.H{
		"message": "发送成功",
		"data": gin.H{
			"expires_in": 300,
		},
	}
	if gin.Mode() != gin.ReleaseMode {
		resp["data"].(gin.H)["debug_code"] = code
	}
	c.JSON(http.StatusOK, resp)
}

func consumeSMSCode(db *gorm.DB, phone, purpose, code string) error {
	var rec models.SMSCode
	now := time.Now().Unix()
	err := db.
		Where("phone = ? AND purpose = ? AND code = ? AND used = ? AND expires_at > ?", phone, purpose, code, false, now).
		Order("id DESC").
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("验证码错误或已过期")
		}
		return err
	}

	usedAt := time.Now()
	if err := db.Model(&models.SMSCode{}).
		Where("id = ? AND used = ?", rec.ID, false).
		Updates(map[string]interface{}{"used": true, "used_at": &usedAt}).Error; err != nil {
		return err
	}
	return nil
}

func generateDigits(n int) (string, error) {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b[i] = byte('0' + num.Int64())
	}
	return string(b), nil
}
