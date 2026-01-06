package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetCardUsages(c *gin.Context) {
	cardID := c.Param("id")
	var usages []models.Usage
	config.DB.Preload("Merchant").Preload("Technician").Where("card_id = ?", cardID).Order("used_at DESC").Find(&usages)
	autoFixUsages(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}

func GetMerchantUsages(c *gin.Context) {
	merchantID := c.Param("id")
	var usages []models.Usage
	config.DB.Preload("Card").Preload("Card.User").Preload("Technician").Where("merchant_id = ?", merchantID).Order("used_at DESC").Find(&usages)
	autoFixUsages(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}

// autoFixUsages 对超过12小时的使用记录自动置为完成并清空技师ID
func autoFixUsages(usages *[]models.Usage) {
	now := time.Now()
	for i := range *usages {
		u := &(*usages)[i]
		if u.UsedAt == nil || u.Status == "failed" {
			continue
		}
		// 超过12小时，自动置为完成并清空技师ID
		if now.Sub(*u.UsedAt) > 12*time.Hour {
			if u.Status != "success" || u.TechnicianID != nil {
				// 计算结单时间 = 核销时间 + 商户服务时间 + 5分钟
				var merchant models.Merchant
				if err := config.DB.First(&merchant, u.MerchantID).Error; err == nil {
					serviceMinutes := merchant.AvgServiceMinutes
					if serviceMinutes <= 0 {
						serviceMinutes = 15 // 默认15分钟
					}
					finishedAt := u.UsedAt.Add(time.Duration(serviceMinutes+5) * time.Minute)
					config.DB.Model(u).Updates(map[string]interface{}{
						"status":        "success",
						"technician_id": nil,
						"finished_at":   finishedAt,
					})
					u.Status = "success"
					u.TechnicianID = nil
					u.FinishedAt = &finishedAt
				} else {
					// 如果商户信息获取失败，使用当前时间作为兜底
					config.DB.Model(u).Updates(map[string]interface{}{
						"status":        "success",
						"technician_id": nil,
						"finished_at":   now,
					})
					u.Status = "success"
					u.TechnicianID = nil
					u.FinishedAt = &now
				}
			}
		}
	}
}
