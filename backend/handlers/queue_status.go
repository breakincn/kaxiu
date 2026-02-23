package handlers

import (
	"kabao/config"
	"kabao/middleware"
	"kabao/models"
	"kabao/queue"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func isMerchantInBusinessHours(m *models.Merchant, now time.Time) bool {
	if m == nil {
		return true
	}

	// IsOpen 优先
	if !m.IsOpen {
		return false
	}

	parseHM := func(s string) (hour int, min int, ok bool) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, 0, false
		}
		t, err := time.Parse("15:04", s)
		if err != nil {
			return 0, 0, false
		}
		return t.Hour(), t.Minute(), true
	}
	inRange := func(start, end string) bool {
		sh, sm, ok1 := parseHM(start)
		eh, em, ok2 := parseHM(end)
		if !ok1 || !ok2 {
			return false
		}
		curMin := now.Hour()*60 + now.Minute()
		startMin := sh*60 + sm
		endMin := eh*60 + em
		// 允许跨天营业（如 22:00-02:00）
		if startMin <= endMin {
			return curMin >= startMin && curMin <= endMin
		}
		return curMin >= startMin || curMin <= endMin
	}

	allDayStart := strings.TrimSpace(m.AllDayStart)
	allDayEnd := strings.TrimSpace(m.AllDayEnd)
	if allDayStart != "" && allDayEnd != "" {
		return inRange(allDayStart, allDayEnd)
	}

	if inRange(m.MorningStart, m.MorningEnd) {
		return true
	}
	if inRange(m.AfternoonStart, m.AfternoonEnd) {
		return true
	}
	if inRange(m.EveningStart, m.EveningEnd) {
		return true
	}

	// 未配置时间段：默认认为营业中（以 IsOpen 为准）
	return true
}

func finalizeSessionManual(tx *gorm.DB, merchant *models.Merchant, s *models.ServiceSession, now time.Time) error {
	if tx == nil || merchant == nil || s == nil {
		return nil
	}
	if s.StartConfirmedAt == nil {
		return nil
	}
	updates := map[string]interface{}{
		"status": models.ApplyStatusPrefix(s.Status, "finished"),
	}
	if s.FinishedAt == nil {
		updates["finished_at"] = now
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ? AND start_confirmed_at IS NOT NULL", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"serving", "auto_finishing"})).
		Updates(updates).Error; err != nil {
		return err
	}
	if err := tx.First(s, s.ID).Error; err != nil {
		return err
	}

	if s.InitialUsageID == 0 {
		return nil
	}
	finishedAt := now
	if s.FinishedAt != nil {
		finishedAt = *s.FinishedAt
	}
	uUpdates := map[string]interface{}{
		"status":      "success",
		"finished_at": finishedAt,
	}
	if s.TechnicianID != nil && *s.TechnicianID > 0 {
		uUpdates["technician_id"] = *s.TechnicianID
	}
	if s.RoomID != nil && *s.RoomID > 0 {
		uUpdates["room_id"] = *s.RoomID
	}
	if err := tx.Model(&models.Usage{}).
		Where("id = ? AND status = ?", s.InitialUsageID, "in_progress").
		Updates(uUpdates).Error; err != nil {
		return err
	}
	if merchant.SupportQueue && queue.Default != nil {
		date := now.Format("2006-01-02")
		queue.Default.MarkDone(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID, now)
	}
	return nil
}

func assignNextSessionToTechnicianManual(tx *gorm.DB, merchant *models.Merchant, technicianID uint, now time.Time) (nextUsageID uint, assigned bool, err error) {
	if tx == nil || merchant == nil || merchant.ID == 0 || technicianID == 0 {
		return 0, false, nil
	}
	if queue.Default == nil {
		return 0, false, nil
	}
	date := now.Format("2006-01-02")

	// 策略1：优先从队列取下一个未叫号
	nextUsageID = queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	if nextUsageID > 0 {
		// 尝试找到对应的可分配 session
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND initial_usage_id = ? AND start_confirmed_at IS NULL AND technician_id IS NULL", merchant.ID, nextUsageID)
		q = q.Where("status IN ?", models.ExpandStatusWithKnownPrefixes("staff_selecting"))

		var nextSession models.ServiceSession
		if err := q.Order("id desc").First(&nextSession).Error; err == nil {
			// 找到了，分配给当前技师
			updates := map[string]interface{}{
				"technician_id":                 technicianID,
				"last_technician_id":            technicianID,
				"status":                        models.ApplyStatusPrefix(nextSession.Status, "start_pending"),
				"staff_select_entered_at":       nil,
				"staff_select_cooldown_until":   nil,
				"start_pending_timeout_seconds": int(config.StartPendingTimeout().Seconds()),
			}
			if err := tx.Model(&models.ServiceSession{}).
				Where("id = ? AND merchant_id = ? AND technician_id IS NULL AND start_confirmed_at IS NULL", nextSession.ID, merchant.ID).
				Updates(updates).Error; err != nil {
				queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
				return 0, false, err
			}
			return nextUsageID, true, nil
		} else if err != gorm.ErrRecordNotFound {
			// 数据库错误
			queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
			return 0, false, err
		}
		// 队列返回的号找不到对应的可分配 session（可能已被分配），回退队列状态
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
	}

	// 策略2：队列无可叫号或队列返回的号已被分配，直接从数据库查找任意一个待分配的 session
	var anySession models.ServiceSession
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND technician_id IS NULL AND start_confirmed_at IS NULL", merchant.ID).
		Where("status IN ?", models.ExpandStatusWithKnownPrefixes("staff_selecting")).
		Order("id asc").
		First(&anySession).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, false, nil
		}
		return 0, false, err
	}

	// 找到了一个待分配的 session，分配给当前技师
	updates := map[string]interface{}{
		"technician_id":                 technicianID,
		"last_technician_id":            technicianID,
		"status":                        models.ApplyStatusPrefix(anySession.Status, "start_pending"),
		"staff_select_entered_at":       nil,
		"staff_select_cooldown_until":   nil,
		"start_pending_timeout_seconds": int(config.StartPendingTimeout().Seconds()),
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND merchant_id = ? AND technician_id IS NULL AND start_confirmed_at IS NULL", anySession.ID, merchant.ID).
		Updates(updates).Error; err != nil {
		return 0, false, err
	}

	// 如果这个 session 有对应的 usage_id，尝试在队列中标记为已叫（如果还没叫的话）
	if anySession.InitialUsageID > 0 {
		_ = queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	}

	return anySession.InitialUsageID, true, nil
}

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

	now := time.Now()
	isInBusinessHours := isMerchantInBusinessHours(&merchant, now)

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
			"queue_ended_at":          merchant.QueueEndedAt,
			"queue_mode":              merchant.QueueMode,
			"support_queue":           merchant.SupportQueue,
			"is_in_business_hours":    isInBusinessHours,
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

	now := time.Now()
	// 恢复叫号：清空结束叫号时间，避免结束叫号后的自动收尾误触发
	if !*input.QueuePaused {
		updates := map[string]interface{}{
			"queue_paused":   false,
			"queue_ended_at": nil,
		}
		if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"queue_paused":   false,
				"queue_ended_at": nil,
			},
		})
		return
	}
	// 打烊时：暂停叫号升级为结束叫号，并记录 queue_ended_at
	if *input.QueuePaused && !isMerchantInBusinessHours(&merchant, now) {
		updates := map[string]interface{}{
			"queue_paused":   true,
			"queue_ended_at": now,
		}
		if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"queue_paused":   true,
				"queue_ended_at": now,
			},
		})
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

	// 若已打烊：技师点击“暂停叫号”时升级为“结束叫号”，写入 merchants.queue_ended_at
	if *input.QueuePaused {
		var merchant models.Merchant
		if err := config.DB.First(&merchant, merchantID).Error; err == nil {
			now := time.Now()
			if !isMerchantInBusinessHours(&merchant, now) {
				_ = config.DB.Model(&merchant).Updates(map[string]interface{}{
					"queue_paused":   true,
					"queue_ended_at": now,
				}).Error
			}
		}
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

func promoteManualSingleCalledSession(tx *gorm.DB, merchant *models.Merchant, usageID uint, now time.Time) {
	if tx == nil || merchant == nil || merchant.ID == 0 || usageID == 0 {
		return
	}
	if merchant.QueueMode != "manual" || merchant.SupportMultiCustomerService {
		return
	}

	date := now.Format("2006-01-02")
	var s models.ServiceSession
	q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND initial_usage_id = ?", merchant.ID, usageID).
		Where("status IN ?", models.ExpandStatusWithKnownPrefixes("staff_selecting")).
		Order("id desc")
	if err := q.First(&s).Error; err != nil {
		if queue.Default != nil {
			queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, usageID)
		}
		return
	}

	startAt := now.Add(time.Duration(merchant.StartDelaySeconds) * time.Second)
	updates := map[string]interface{}{
		"start_confirmed_at":  now,
		"scheduled_start_at":  startAt,
		"status":              models.ApplyStatusPrefix(s.Status, "delay_pending"),
		"start_delay_seconds": merchant.StartDelaySeconds,
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND start_confirmed_at IS NULL AND status IN ?", s.ID, models.ExpandStatusWithKnownPrefixes("staff_selecting")).
		Updates(updates).Error; err != nil {
		if queue.Default != nil {
			queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, usageID)
		}
		return
	}
}

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

	// 检查商户是否暂停叫号，若暂停则先恢复
	// 恢复叫号时也需要清空 queue_ended_at：否则调度器会把 queue_ended_at 最近 15 分钟内的商户批量收尾，导致“待叫号”刷新后变“完成”
	if merchant.QueuePaused || merchant.QueueEndedAt != nil {
		updates := map[string]interface{}{
			"queue_paused":   false,
			"queue_ended_at": nil,
		}
		if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
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
	if nextUsageID > 0 {
		_ = config.DB.Transaction(func(tx *gorm.DB) error {
			promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
			return nil
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"next_usage_id": nextUsageID,
			"queue_paused":  false,
		},
	})
}

// TriggerContinueCalling 手动叫号：技师点击“继续叫号”= 完成当前服务 + 推进下一号
// POST /queue/continue-call
func TriggerContinueCalling(c *gin.Context) {
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportQueue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启叫号功能"})
		return
	}
	if strings.TrimSpace(merchant.QueueMode) != "manual" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前非人工叫号模式"})
		return
	}
	if merchant.QueuePaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "叫号已暂停"})
		return
	}
	if queue.Default == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "叫号服务未初始化"})
		return
	}

	// 检查技师是否暂停了自己的叫号
	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作人员不存在"})
		return
	}
	if tech.QueuePaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "您的叫号已暂停"})
		return
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	out := gin.H{
		"finished_session_id": uint(0),
		"next_usage_id":       uint(0),
		"assigned":            false,
		"reason":              "",
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// 锁定该技师当前进行中的会话
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND status IN ? AND start_confirmed_at IS NOT NULL", merchantID, techID, models.ExpandStatusesWithKnownPrefixes([]string{"serving", "auto_finishing"})).
			Order("id desc").
			First(&s).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// 若当前技师已有待上号(start_pending)的会话：
				// - 倒计时未结束：提示等待（前端置灰继续叫号按钮）
				// - 倒计时结束：允许跳过该号并继续叫下一个（不退核销，允许后续扫码回补）
				var pending models.ServiceSession
				err2 := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("merchant_id = ? AND technician_id = ? AND status IN ? AND start_confirmed_at IS NULL", merchantID, techID, models.ExpandStatusWithKnownPrefixes("start_pending")).
					Order("id desc").
					First(&pending).Error
				if err2 == nil {
					baseAt := pending.UpdatedAt
					if baseAt == nil {
						baseAt = pending.CreatedAt
					}
					if baseAt == nil {
						out["reason"] = "当前已有待上号用户，请先扫码上号"
						out["pending_session_id"] = pending.ID
						return nil
					}
					timeout := config.StartPendingTimeout()
					if pending.StartPendingTimeoutSeconds > 0 {
						timeout = time.Duration(pending.StartPendingTimeoutSeconds) * time.Second
					}
					deadline := baseAt.Add(timeout)
					remain := int(deadline.Sub(now).Seconds())
					if remain > 0 {
						out["reason"] = "当前已有待上号用户，请等待倒计时结束后再继续叫号"
						out["pending_session_id"] = pending.ID
						out["need_wait"] = true
						out["pending_remaining_seconds"] = remain
						out["pending_timeout_seconds"] = int(timeout.Seconds())
						return nil
					}

					// 倒计时已到：跳过该号（不退核销），释放窗口，继续分配下一号
					skipUpdates := map[string]interface{}{
						"status":                        models.ApplyStatusPrefix(pending.Status, "timeout_waiting"),
						"last_technician_id":            techID,
						"technician_id":                 nil,
						"staff_select_entered_at":       nil,
						"staff_select_cooldown_until":   nil,
						"start_pending_timeout_seconds": 0,
						"start_timeout_count":           gorm.Expr("start_timeout_count + ?", 1),
						"start_timeout_last_at":         now,
					}
					if err3 := tx.Model(&models.ServiceSession{}).
						Where("id = ? AND merchant_id = ? AND status IN ? AND start_confirmed_at IS NULL", pending.ID, merchantID, models.ExpandStatusWithKnownPrefixes("start_pending")).
						Updates(skipUpdates).Error; err3 != nil {
						return err3
					}
					if pending.InitialUsageID > 0 && queue.Default != nil {
						queue.Default.MarkDone(merchant.ID, date, queue.QueueTypeOnsite, pending.InitialUsageID, now)
					}
					out["skipped_session_id"] = pending.ID
				} else if err2 != gorm.ErrRecordNotFound {
					return err2
				}

				// 没有进行中服务：允许“继续叫号”作为“推进下一号/分配下一位”的触发
				if !merchant.SupportMultiCustomerService {
					nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
					if nextUsageID > 0 {
						promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
					}
					out["next_usage_id"] = nextUsageID
					out["assigned"] = false
					if nextUsageID == 0 {
						out["reason"] = "当前无进行中的服务，且暂无可叫号的用户"
					}
					return nil
				}

				nextUsageID, assigned, err := assignNextSessionToTechnicianManual(tx, &merchant, techID, now)
				if err != nil {
					return err
				}
				out["next_usage_id"] = nextUsageID
				out["assigned"] = assigned
				if nextUsageID == 0 {
					out["reason"] = "当前无进行中的服务，且暂无可分配的用户"
				}
				return nil
			}
			return err
		}
		out["finished_session_id"] = s.ID

		// 检查服务是否达到项目设定的服务时长
		if s.StartedAt != nil && s.DurationMinutes > 0 {
			servedMinutes := int(now.Sub(*s.StartedAt).Minutes())
			if servedMinutes < s.DurationMinutes {
				// 未达到服务时长，返回二次确认信息，不执行结束操作
				out["need_confirm"] = true
				out["served_minutes"] = servedMinutes
				out["required_minutes"] = s.DurationMinutes
				out["remaining_minutes"] = s.DurationMinutes - servedMinutes
				return nil
			}
		}

		// 调试日志：检查为什么没有触发二次确认
		if s.StartedAt == nil {
			log.Printf("[DEBUG] ServiceSession %d: StartedAt is nil, skipping duration check", s.ID)
		} else if s.DurationMinutes <= 0 {
			log.Printf("[DEBUG] ServiceSession %d: DurationMinutes is %d, skipping duration check", s.ID, s.DurationMinutes)
		} else {
			servedMinutes := int(now.Sub(*s.StartedAt).Minutes())
			log.Printf("[DEBUG] ServiceSession %d: served %d minutes, required %d minutes, no confirmation needed", s.ID, servedMinutes, s.DurationMinutes)
		}

		if err := finalizeSessionManual(tx, &merchant, &s, now); err != nil {
			return err
		}

		if !merchant.SupportMultiCustomerService {
			nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
			if nextUsageID > 0 {
				promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
			}
			out["next_usage_id"] = nextUsageID
			out["assigned"] = false
			return nil
		}

		nextUsageID, assigned, err := assignNextSessionToTechnicianManual(tx, &merchant, techID, now)
		if err != nil {
			return err
		}
		out["next_usage_id"] = nextUsageID
		out["assigned"] = assigned
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

// TriggerContinueCallingForce 手动叫号：技师点击“继续叫号”强制结束 = 强制完成当前服务 + 推进下一号
// POST /queue/continue-call-force
func TriggerContinueCallingForce(c *gin.Context) {
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportQueue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启叫号功能"})
		return
	}
	if strings.TrimSpace(merchant.QueueMode) != "manual" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前非人工叫号模式"})
		return
	}
	if merchant.QueuePaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "叫号已暂停"})
		return
	}
	if queue.Default == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "叫号服务未初始化"})
		return
	}

	// 检查技师是否暂停了自己的叫号
	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作人员不存在"})
		return
	}
	if tech.QueuePaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "您的叫号已暂停"})
		return
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	out := gin.H{
		"finished_session_id": uint(0),
		"next_usage_id":       uint(0),
		"assigned":            false,
		"reason":              "",
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// 锁定该技师当前进行中的会话
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND status IN ? AND start_confirmed_at IS NOT NULL", merchantID, techID, models.ExpandStatusesWithKnownPrefixes([]string{"serving", "auto_finishing"})).
			Order("id desc").
			First(&s).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				out["reason"] = "当前无进行中的服务"
				return nil
			}
			return err
		}
		out["finished_session_id"] = s.ID

		// 强制结束，不检查服务时长
		if err := finalizeSessionManual(tx, &merchant, &s, now); err != nil {
			return err
		}

		if !merchant.SupportMultiCustomerService {
			nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
			if nextUsageID > 0 {
				promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
			}
			out["next_usage_id"] = nextUsageID
			out["assigned"] = false
			return nil
		}

		nextUsageID, assigned, err := assignNextSessionToTechnicianManual(tx, &merchant, techID, now)
		if err != nil {
			return err
		}
		out["next_usage_id"] = nextUsageID
		out["assigned"] = assigned
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
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
	// 防御性处理：若历史上写入了 queue_ended_at 但又恢复运营（queue_paused=false），这里清理掉，避免调度器 finalizeUsagesAfterQueueEnded 误收尾
	if merchant.QueueEndedAt != nil {
		_ = config.DB.Model(&merchant).Update("queue_ended_at", nil).Error
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
	if nextUsageID > 0 {
		_ = config.DB.Transaction(func(tx *gorm.DB) error {
			promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
			return nil
		})
	}

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
