package handlers

import (
	"kaxiu/config"
	"kaxiu/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetMerchantNotices(c *gin.Context) {
	merchantID := c.Param("id")
	limit := c.DefaultQuery("limit", "10")

	var notices []models.Notice
	config.DB.Where("merchant_id = ?", merchantID).Order("created_at DESC").Limit(10).Find(&notices)

	_ = limit
	c.JSON(http.StatusOK, gin.H{"data": notices})
}

func CreateNotice(c *gin.Context) {
	var input struct {
		MerchantID uint   `json:"merchant_id" binding:"required"`
		Title      string `json:"title" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notice := models.Notice{
		MerchantID: input.MerchantID,
		Title:      input.Title,
		Content:    input.Content,
		CreatedAt:  time.Now().Format("2006-01-02"),
	}
	config.DB.Create(&notice)
	c.JSON(http.StatusOK, gin.H{"data": notice})
}
