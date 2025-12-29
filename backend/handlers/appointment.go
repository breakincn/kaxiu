package handlers

import (
	"kaxiu/config"
	"kaxiu/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetMerchantAppointments(c *gin.Context) {
	merchantID := c.Param("id")
	status := c.Query("status")

	var appointments []models.Appointment
	query := config.DB.Preload("User").Where("merchant_id = ?", merchantID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Order("appointment_time ASC").Find(&appointments)
	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

func GetUserAppointments(c *gin.Context) {
	userID := c.Param("id")
	var appointments []models.Appointment
	config.DB.Preload("Merchant").Where("user_id = ?", userID).Order("appointment_time DESC").Find(&appointments)
	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

func GetCardAppointment(c *gin.Context) {
	cardID := c.Param("id")

	var card models.Card
	if err := config.DB.First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}

	var appointment models.Appointment
	err := config.DB.Preload("Merchant").
		Where("user_id = ? AND merchant_id = ? AND status IN ('pending', 'confirmed')", card.UserID, card.MerchantID).
		Order("appointment_time ASC").
		First(&appointment).Error

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}

	// 计算排队信息
	var queueBefore int64
	config.DB.Model(&models.Appointment{}).
		Where("merchant_id = ? AND status IN ('pending', 'confirmed') AND appointment_time < ?",
			card.MerchantID, appointment.AppointmentTime).
		Count(&queueBefore)

	var merchant models.Merchant
	config.DB.First(&merchant, card.MerchantID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"appointment":       appointment,
			"queue_before":      queueBefore,
			"estimated_minutes": int(queueBefore) * merchant.AvgServiceMinutes,
		},
	})
}

func CreateAppointment(c *gin.Context) {
	var input struct {
		MerchantID      uint   `json:"merchant_id" binding:"required"`
		UserID          uint   `json:"user_id" binding:"required"`
		AppointmentTime string `json:"appointment_time" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查商户是否支持预约
	var merchant models.Merchant
	if err := config.DB.First(&merchant, input.MerchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	if !merchant.SupportAppointment {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该商户不支持预约"})
		return
	}

	appointment := models.Appointment{
		MerchantID:      input.MerchantID,
		UserID:          input.UserID,
		AppointmentTime: input.AppointmentTime,
		Status:          "pending",
	}
	config.DB.Create(&appointment)
	config.DB.Preload("User").Preload("Merchant").First(&appointment, appointment.ID)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func ConfirmAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	if appointment.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能确认待处理的预约"})
		return
	}

	config.DB.Model(&appointment).Update("status", "confirmed")
	config.DB.Preload("User").Preload("Merchant").First(&appointment, id)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func FinishAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	if appointment.Status != "confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能完成已确认的预约"})
		return
	}

	config.DB.Model(&appointment).Update("status", "finished")
	config.DB.Preload("User").Preload("Merchant").First(&appointment, id)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func CancelAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	config.DB.Model(&appointment).Update("status", "canceled")
	config.DB.Preload("User").Preload("Merchant").First(&appointment, id)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func GetQueueStatus(c *gin.Context) {
	merchantID := c.Param("id")

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 待处理预约数
	var pendingCount int64
	config.DB.Model(&models.Appointment{}).
		Where("merchant_id = ? AND status = 'pending'", merchantID).
		Count(&pendingCount)

	// 今日核销数
	today := time.Now().Format("2006-01-02")
	var todayVerifyCount int64
	config.DB.Model(&models.Usage{}).
		Where("merchant_id = ? AND DATE(used_at) = ?", merchantID, today).
		Count(&todayVerifyCount)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"pending_appointments": pendingCount,
			"today_verify_count":   todayVerifyCount,
			"avg_service_minutes":  merchant.AvgServiceMinutes,
		},
	})
}
