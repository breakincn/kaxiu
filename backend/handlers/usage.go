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
	config.DB.Preload("Merchant").Preload("Technician").Preload("Project").Where("card_id = ?", cardID).Order("used_at DESC").Find(&usages)
	enrichUsagesWithServiceSession(&usages)
	autoFixUsages(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}

func GetMerchantUsages(c *gin.Context) {
	merchantID := c.Param("id")
	var usages []models.Usage
	config.DB.Preload("Card").Preload("Card.User").Preload("Technician").Preload("Merchant").Preload("Project").Where("merchant_id = ?", merchantID).Order("used_at DESC").Find(&usages)
	enrichUsagesWithServiceSession(&usages)
	autoFixUsages(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}

func enrichUsagesWithServiceSession(usages *[]models.Usage) {
	if usages == nil || len(*usages) == 0 {
		return
	}

	ids := make([]uint, 0, len(*usages))
	for i := range *usages {
		u := (*usages)[i]
		if u.ID > 0 {
			ids = append(ids, u.ID)
		}
	}
	if len(ids) == 0 {
		return
	}

	type sessLite struct {
		ID           uint       `gorm:"column:id"`
		InitialUsageID uint      `gorm:"column:initial_usage_id"`
		Status       string     `gorm:"column:status"`
		PrecheckAt   *time.Time `gorm:"column:precheck_at"`
	}

	var sessions []sessLite
	if err := config.DB.
		Table("service_sessions").
		Select("id, initial_usage_id, status, precheck_at").
		Where("initial_usage_id IN ?", ids).
		Find(&sessions).Error; err != nil {
		return
	}

	byUsageID := make(map[uint]sessLite, len(sessions))
	for i := range sessions {
		s := sessions[i]
		if s.InitialUsageID == 0 {
			continue
		}
		byUsageID[s.InitialUsageID] = s
	}

	for i := range *usages {
		u := &(*usages)[i]
		if s, ok := byUsageID[u.ID]; ok {
			sid := s.ID
			u.ServiceSessionID = &sid
			u.ServiceSessionStatus = s.Status
			u.ServiceSessionPrecheckAt = s.PrecheckAt
		}
	}
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
				finishedAt := u.UsedAt.Add(time.Duration(15+5) * time.Minute)
				config.DB.Model(u).Updates(map[string]interface{}{
					"status":        "success",
					"technician_id": nil,
					"finished_at":   finishedAt,
				})
				u.Status = "success"
				u.TechnicianID = nil
				u.FinishedAt = &finishedAt
			}
		}
	}
}
