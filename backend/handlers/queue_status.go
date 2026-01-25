package handlers

import (
	"kabao/config"
	"kabao/middleware"
	"kabao/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetQueueCallingStatus 获取叫号状态
// GET /queue/calling-status
func GetQueueCallingStatus(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 获取账号类型
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	var technicianQueuePaused *bool
	if authType == "staff" {
		techIDAny, _ := c.Get("technician_id")
		techID, _ := techIDAny.(uint)
		if techID > 0 {
			var tech models.Technician
			if err := config.DB.First(&tech, techID).Error; err == nil {
				technicianQueuePaused = &tech.QueuePaused
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"queue_paused":            merchant.QueuePaused,
			"queue_mode":              merchant.QueueMode,
			"support_queue":           merchant.SupportQueue,
			"technician_queue_paused": technicianQueuePaused,
		},
	})
}

// UpdateQueueCallingStatus 更新商户叫号全局状态（开始/暂停）
// PUT /queue/calling-status
func UpdateQueueCallingStatus(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		QueuePaused *bool `json:"queue_paused"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.QueuePaused == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue_paused 不能为空"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	if err := config.DB.Model(&merchant).Update("queue_paused", *input.QueuePaused).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"queue_paused": *input.QueuePaused,
		},
	})
}

// UpdateTechnicianQueuePaused 更新技师自己的叫号状态（开始/暂停）
// PUT /technician/queue-paused
func UpdateTechnicianQueuePaused(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 必须是技师账号
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员账号可操作"})
		return
	}

	techIDAny, _ := c.Get("technician_id")
	techID, _ := techIDAny.(uint)
	if techID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "获取工作人员信息失败"})
		return
	}

	var input struct {
		QueuePaused *bool `json:"queue_paused"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.QueuePaused == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue_paused 不能为空"})
		return
	}

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作人员不存在"})
		return
	}

	if err := config.DB.Model(&tech).Update("queue_paused", *input.QueuePaused).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"queue_paused": *input.QueuePaused,
		},
	})
}

// GetTechnicianQueuePaused 获取技师自己的叫号状态
// GET /technician/queue-paused
func GetTechnicianQueuePaused(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 必须是技师账号
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员账号可操作"})
		return
	}

	techIDAny, _ := c.Get("technician_id")
	techID, _ := techIDAny.(uint)
	if techID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "获取工作人员信息失败"})
		return
	}

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作人员不存在"})
		return
	}

	// 同时获取商户的全局叫号状态
	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"queue_paused":          tech.QueuePaused,
			"merchant_queue_paused": merchant.QueuePaused,
		},
	})
}

// 未使用的变量占位
var _ = middleware.RequirePermission
