package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
