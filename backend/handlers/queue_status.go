package handlers

import (
	"kabao/config"
	"kabao/middleware"
	"kabao/models"
	"kabao/queue"
	"log"
	"net/http"
	"os"
	"time"

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

// EnqueueOnsiteUsages 补偿：将指定 usage_id 补入现场叫号队列（当日）
// POST /queue/enqueue-onsite
// 仅用于历史数据未入队的补偿。
func EnqueueOnsiteUsages(c *gin.Context) {
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
		UsageIDs  []uint `json:"usage_ids"`
		CallNext  bool   `json:"call_next"`
		StartNo   *int   `json:"start_no"`
		QueueType string `json:"queue_type"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(input.UsageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usage_ids 不能为空"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportQueue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启叫号功能"})
		return
	}
	if merchant.QueueMode != "auto" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅自动叫号模式支持补入队列"})
		return
	}
	if queue.Default == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "叫号服务未初始化"})
		return
	}

	qt := queue.QueueTypeOnsite
	if input.QueueType == string(queue.QueueTypeAppointment) {
		qt = queue.QueueTypeAppointment
	}

	now := time.Now()
	date := now.Format("2006-01-02")
	startNo := merchant.QueueStartNo
	if input.StartNo != nil && *input.StartNo > 0 {
		startNo = *input.StartNo
	}
	autoCallFirst := false

	createdMap := make(map[uint]bool, len(input.UsageIDs))
	for _, uid := range input.UsageIDs {
		if uid == 0 {
			continue
		}
		// 校验 usage 属于本商户且仍在进行中，并存在对应 service_session
		var u models.Usage
		if err := config.DB.Where("id = ? AND merchant_id = ?", uid, merchantID).First(&u).Error; err != nil {
			createdMap[uid] = false
			continue
		}
		if u.Status != "in_progress" {
			createdMap[uid] = false
			continue
		}
		var ss models.ServiceSession
		if err := config.DB.Where("merchant_id = ? AND initial_usage_id = ?", merchantID, uid).Order("id desc").First(&ss).Error; err != nil {
			createdMap[uid] = false
			continue
		}

		tk, created := queue.Default.Enqueue(merchantID, date, qt, uid, startNo, autoCallFirst, now)
		createdMap[uid] = created
		if os.Getenv("KABAO_QUEUE_DEBUG") == "1" {
			log.Printf("[queue-debug] compensate enqueue onsite: merchant=%d date=%s usage_id=%d created=%v queue_no=%d called_at=%v\n", merchantID, date, uid, created, tk.No, tk.CalledAt)
		}
		_ = tk
	}

	var nextUsageID uint
	if input.CallNext {
		nextUsageID = queue.Default.CallNextUncalled(merchantID, date, qt, now)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"date":          date,
			"queue_type":    string(qt),
			"created_map":   createdMap,
			"next_usage_id": nextUsageID,
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

// TriggerNextCalling 触发下一个叫号（手动叫号模式下使用）
// POST /queue/call-next
// 运营客服点击"开始叫号"时：恢复商户叫号 + 触发下一个
// 专业客服点击"开始叫号"时：恢复商户叫号 + 触发下一个
// 扫码结单时：判断是否有权限，若有则触发下一个
func TriggerNextCalling(c *gin.Context) {
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

	// 检查是否开启了叫号
	if !merchant.SupportQueue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启叫号功能"})
		return
	}

	// 检查是否是手动叫号模式
	if merchant.QueueMode != "manual" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前为自动叫号模式，无需手动触发"})
		return
	}

	// 检查商户是否暂停叫号，若暂停则先恢复
	if merchant.QueuePaused {
		if err := config.DB.Model(&merchant).Update("queue_paused", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复叫号失败"})
			return
		}
	}

	// 触发下一个叫号
	if queue.Default == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "叫号服务未初始化"})
		return
	}

	now := time.Now()
	date := now.Format("2006-01-02")
	nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"next_usage_id": nextUsageID,
			"queue_paused":  false,
		},
	})
}

// TriggerNextCallingOnFinish 扫码结单时触发下一个叫号（手动叫号模式下使用）
// POST /queue/finish-call-next
// 仅当：
// 1. 商户开启了叫号
// 2. 商户是手动叫号模式
// 3. 商户未暂停叫号
// 4. 操作人员有叫号权限
// 5. 操作人员未暂停自己的叫号（技师账号）
// 才触发下一个叫号
func TriggerNextCallingOnFinish(c *gin.Context) {
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

	// 检查是否开启了叫号
	if !merchant.SupportQueue {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "商户未开启叫号"}})
		return
	}

	// 检查是否是手动叫号模式
	if merchant.QueueMode != "manual" {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "自动叫号模式"}})
		return
	}

	// 检查商户是否暂停叫号
	if merchant.QueuePaused {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "商户叫号已暂停"}})
		return
	}

	// 获取账号类型
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	// 检查叫号权限（商户老板号默认有权限）
	if authType != "merchant" {
		okCalling, err := middleware.HasPermission(c, "merchant.queue.calling")
		if err != nil || !okCalling {
			c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "无叫号权限"}})
			return
		}
	}

	// 检查技师是否暂停了自己的叫号
	if authType == "staff" {
		techIDAny, _ := c.Get("technician_id")
		techID, _ := techIDAny.(uint)
		if techID > 0 {
			var tech models.Technician
			if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err == nil {
				if tech.QueuePaused {
					c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "您的叫号已暂停"}})
					return
				}
			}
		}
	}

	// 触发下一个叫号
	if queue.Default == nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "叫号服务未初始化"}})
		return
	}

	now := time.Now()
	date := now.Format("2006-01-02")
	nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"triggered":     true,
			"next_usage_id": nextUsageID,
		},
	})
}

// TriggerAutoAssign 多客服叫号模式手动触发自动分配（运营兜底接口）
// POST /queue/trigger-auto-assign
// 用于极端情况：有空闲技师但队列卡在待叫号时，运营/商户手动触发重新分配
func TriggerAutoAssign(c *gin.Context) {
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

	// 仅在多客服叫号模式下可用
	if !merchant.SupportQueue || merchant.QueueMode != "auto" || !merchant.SupportMultiCustomerService {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前模式不支持手动分配"})
		return
	}

	// 权限检查：商户老板号默认有权限；技师账号需要“merchant.queue.calling”权限
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "merchant" {
		okAssign, err := middleware.HasPermission(c, "merchant.queue.calling")
		if err != nil || !okAssign {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权限操作"})
			return
		}
	}

	now := time.Now()
	// 调用技师签到模块的批量分配函数
	tryAutoCallNextForIdleTechnicians(config.DB, merchantID, now)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"triggered": true,
			"message":   "已尝试为所有空闲技师分配下一号",
		},
	})
}
