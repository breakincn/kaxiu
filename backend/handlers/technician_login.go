package handlers

import (
	"kabao/config"
	"kabao/models"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// TechnicianLogin 技师通过店铺路径登录
func TechnicianLogin(c *gin.Context) {
	slug := strings.ToLower(strings.TrimSpace(c.Param("slug")))
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "店铺路径不能为空"})
		return
	}

	var input struct {
		Account  string `json:"account" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account := strings.TrimSpace(input.Account)
	password := strings.TrimSpace(input.Password)

	var shopSlug models.MerchantShopSlug
	if err := config.DB.Where("slug = ?", slug).First(&shopSlug).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "店铺不存在"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, shopSlug.MerchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	var tech models.Technician
	if err := config.DB.Where("merchant_id = ? AND account = ?", merchant.ID, account).First(&tech).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号或密码错误"})
		return
	}

	if !tech.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已禁用"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(tech.Password), []byte(password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号或密码错误"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"merchant_id":   merchant.ID,
		"technician_id": tech.ID,
		"account":       tech.Account,
		"type":          "technician",
		"exp":           time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	tokenString, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		log.Printf("生成token失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败"})
		return
	}

	log.Printf("技师登录成功(店铺路径): ID=%d, 账号=%s, merchant_id=%d, slug=%s", tech.ID, tech.Account, merchant.ID, slug)

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"merchant": gin.H{
			"id":    merchant.ID,
			"phone": merchant.Phone,
			"name":  merchant.Name,
			"type":  merchant.Type,
		},
		"technician": gin.H{
			"id":      tech.ID,
			"name":    tech.Name,
			"code":    tech.Code,
			"account": tech.Account,
		},
	})
}
