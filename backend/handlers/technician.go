package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func GetMerchantTechnicians(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "technician" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var list []models.Technician
	config.DB.Where("merchant_id = ?", merchantID).Order("id desc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func CreateMerchantTechnician(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "technician" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		Name string `json:"name" binding:"required"`
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入技师姓名"})
		return
	}
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入技师编号"})
		return
	}

	account := "js" + code
	defaultPassword := code + "12345"

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	tech := models.Technician{
		MerchantID: merchantID,
		Name:       name,
		Code:       code,
		Account:    account,
		Password:   string(hashedPassword),
	}
	if err := config.DB.Create(&tech).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":               tech.ID,
			"merchant_id":      tech.MerchantID,
			"name":             tech.Name,
			"code":             tech.Code,
			"account":          tech.Account,
			"default_password": defaultPassword,
		},
	})
}

func GetTechniciansByMerchantID(c *gin.Context) {
	merchantID := c.Param("id")
	if strings.TrimSpace(merchantID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户ID不能为空"})
		return
	}

	var list []models.Technician
	config.DB.Where("merchant_id = ?", merchantID).Order("id desc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}
