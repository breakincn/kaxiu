package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetCardUsages(c *gin.Context) {
	cardID := c.Param("id")
	var usages []models.Usage
	config.DB.Preload("Merchant").Preload("Technician").Where("card_id = ?", cardID).Order("used_at DESC").Find(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}

func GetMerchantUsages(c *gin.Context) {
	merchantID := c.Param("id")
	var usages []models.Usage
	config.DB.Preload("Card").Preload("Card.User").Preload("Technician").Where("merchant_id = ?", merchantID).Order("used_at DESC").Find(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}
