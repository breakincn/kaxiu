package handlers

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func tryAutoCallNextForIdleTechnicians(db *gorm.DB, merchantID uint, now time.Time) {
	if db == nil || merchantID == 0 {
		return
	}
	_ = db.Transaction(func(tx *gorm.DB) error {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var ids []uint
		tx.Model(&models.TechnicianAttendance{}).
			Distinct().
			Where("merchant_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL AND status = ?", merchantID, start, "idle").
			Pluck("technician_id", &ids)
		for _, tid := range ids {
			if tid == 0 {
				continue
			}
			tryAutoCallNextForTechnician(tx, merchantID, tid, now)
		}
		return nil
	})
}

func tryAutoCallNextForTechnician(tx *gorm.DB, merchantID uint, technicianID uint, now time.Time) {
	if merchantID == 0 || technicianID == 0 {
		return
	}
	if queue.Default == nil {
		return
	}

	var merchant models.Merchant
	if err := tx.First(&merchant, merchantID).Error; err != nil {
		return
	}
	if !merchant.SupportQueue || merchant.QueueMode != "auto" {
		return
	}
	if !merchant.SupportMultiCustomerService {
		return
	}
	if merchant.QueuePaused {
		return
	}
	// 检查技师是否暂停了叫号
	var tech models.Technician
	if err := tx.Where("id = ? AND merchant_id = ?", technicianID, merchantID).First(&tech).Error; err != nil {
		return
	}
	if tech.QueuePaused {
		return
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// 已存在活跃会话时，不重复分配下一号
	{
		var cnt int64
		err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND technician_id = ? AND status IN ? AND updated_at >= ?", merchantID, technicianID, models.ExpandStatusesWithKnownPrefixes([]string{"start_pending", "delay_pending", "serving", "auto_finishing"}), start).
			Count(&cnt).Error
		if err != nil {
			return
		}
		if cnt > 0 {
			return
		}
	}
	// 多窗口叫号：不依赖客服模式开关。并发上限由当天空闲技师数量天然控制。
	// 通过对考勤记录加行锁 + 将会话置为 start_pending(带 technician_id) 实现并发控制。
	// 分配成功后将技师置为 busy，避免并发路径重复分配。
	var att models.TechnicianAttendance
	attRes := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, technicianID, start).
		Order("id desc").
		Limit(1).
		Find(&att)
	if attRes.Error != nil || attRes.RowsAffected == 0 {
		return
	}
	if att.Status != "idle" {
		return
	}

	// 行锁后再次校验，避免“先查后改”竞态导致重复分配。
	{
		var cnt int64
		err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND technician_id = ? AND status IN ? AND updated_at >= ?", merchantID, technicianID, models.ExpandStatusesWithKnownPrefixes([]string{"start_pending", "delay_pending", "serving", "auto_finishing"}), start).
			Count(&cnt).Error
		if err != nil || cnt > 0 {
			return
		}
	}

	date := now.Format("2006-01-02")
	nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	if nextUsageID == 0 {
		return
	}

	q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND initial_usage_id = ? AND start_confirmed_at IS NULL AND technician_id IS NULL", merchantID, nextUsageID)
	q = q.Where("status IN ?", models.ExpandStatusesWithKnownPrefixes([]string{"staff_selecting", "room_locked", "timeout_waiting"}))

	var nextSession models.ServiceSession
	if err := q.Order("id desc").First(&nextSession).Error; err != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return
	}
	// 模式守卫：跳过跨模式会话（商户配置切换后的历史会话防护）
	if models.ValidateSessionModeForEntry(&nextSession, &merchant) != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return
	}

	updates := map[string]interface{}{
		"technician_id":                 technicianID,
		"status":                        models.ApplyStatusPrefix(nextSession.Status, "start_pending"),
		"staff_select_entered_at":       nil,
		"staff_select_cooldown_until":   nil,
		"start_pending_timeout_seconds": config.MerchantQueueWaitingStartSeconds(&merchant),
	}
	result := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND merchant_id = ? AND technician_id IS NULL AND start_confirmed_at IS NULL", nextSession.ID, merchantID).
		Updates(updates)
	if result.Error != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return
	}
	if result.RowsAffected == 0 {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return
	}

	if err := tx.Model(&models.TechnicianAttendance{}).
		Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, merchantID, technicianID, "idle").
		Updates(map[string]interface{}{"status": "busy"}).Error; err != nil {
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
		return
	}
}

func sameLocalDay(a, b time.Time) bool {
	aa := a.In(time.Local)
	bb := b.In(time.Local)
	ay, am, ad := aa.Date()
	by, bm, bd := bb.Date()
	return ay == by && am == bm && ad == bd
}

func TechnicianCheckIn(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	var techID uint
	if authType == "staff" {
		techIDAny, ok := c.Get("technician_id")
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可签到"})
			return
		}
		v, _ := techIDAny.(uint)
		techID = v
		if techID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可签到"})
			return
		}
	} else {
		var input struct {
			TechnicianID uint `json:"technician_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		techID = input.TechnicianID
	}

	now := time.Now()

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工作人员"})
		return
	}

	req, err := isRoleAttendanceRequired(config.DB, merchantID, tech.ServiceRoleID)
	if err == nil && !req {
		_ = ensureAttendanceForNoCheckinRole(config.DB, merchantID, techID, now)
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "该岗位无需签到"})
		return
	}

	var attendance models.TechnicianAttendance
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	err = config.DB.Where("merchant_id = ? AND technician_id = ? AND created_at >= ?", merchantID, techID, start).First(&attendance).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := config.DB.Transaction(func(tx *gorm.DB) error {
			attendance = models.TechnicianAttendance{MerchantID: merchantID, TechnicianID: techID, CheckedInAt: &now, CheckedOutAt: nil, Status: "idle", NextStatus: nil}
			if err := tx.Create(&attendance).Error; err != nil {
				return err
			}
			tryAutoCallNextForTechnician(tx, merchantID, techID, now)
			return tx.Preload("Technician").First(&attendance, attendance.ID).Error
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": attendance})
		return
	}

	// 当天已上班且未下班：不允许重复上班签到
	if attendance.CheckedInAt != nil && sameLocalDay(*attendance.CheckedInAt, now) && attendance.CheckedOutAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "今日已上班签到，请先下班签到"})
		return
	}

	updates := map[string]interface{}{"checked_in_at": now, "checked_out_at": nil, "status": "idle", "next_status": nil}
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TechnicianAttendance{}).Where("id = ?", attendance.ID).Updates(updates).Error; err != nil {
			return err
		}
		tryAutoCallNextForTechnician(tx, merchantID, techID, now)
		return tx.Preload("Technician").First(&attendance, attendance.ID).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": attendance})
}

func TechnicianCheckOut(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	var techID uint
	if authType == "staff" {
		techIDAny, ok := c.Get("technician_id")
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可签到"})
			return
		}
		v, _ := techIDAny.(uint)
		techID = v
		if techID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可签到"})
			return
		}
	} else {
		var input struct {
			TechnicianID uint `json:"technician_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		techID = input.TechnicianID
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 若岗位无需签到，自动补一条当天考勤记录，保证后续状态更新有记录可写
	{
		var tech models.Technician
		if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err == nil {
			if req, err := isRoleAttendanceRequired(config.DB, merchantID, tech.ServiceRoleID); err == nil && !req {
				_ = ensureAttendanceForNoCheckinRole(config.DB, merchantID, techID, now)
			}
		}
	}

	// 按岗位独立配置签到：若岗位无需签到，则无需下班签到
	{
		var tech models.Technician
		if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err == nil {
			if req, err := isRoleAttendanceRequired(config.DB, merchantID, tech.ServiceRoleID); err == nil && !req {
				_ = ensureAttendanceForNoCheckinRole(config.DB, merchantID, techID, now)
				c.JSON(http.StatusOK, gin.H{"ok": true, "message": "该岗位无需签到"})
				return
			}
		}
	}

	var attendance models.TechnicianAttendance
	if err := config.DB.Where("merchant_id = ? AND technician_id = ? AND created_at >= ?", merchantID, techID, start).
		Order("created_at DESC").
		First(&attendance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未签到"})
		return
	}
	if attendance.CheckedInAt == nil || !sameLocalDay(*attendance.CheckedInAt, now) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "今日未上班签到"})
		return
	}
	if attendance.CheckedOutAt != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "今日已下班签到"})
		return
	}

	updates := map[string]interface{}{"checked_out_at": now, "status": "rest"}
	if err := config.DB.Model(&models.TechnicianAttendance{}).Where("id = ?", attendance.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	config.DB.Preload("Technician").First(&attendance, attendance.ID)
	c.JSON(http.StatusOK, gin.H{"data": attendance})
}

func UpdateTechnicianServiceStatus(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 按岗位独立配置签到：不再依赖商户全局开关

	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	var techID uint
	var input struct {
		TechnicianID *uint  `json:"technician_id"`
		Status       string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if authType == "staff" {
		techIDAny, ok := c.Get("technician_id")
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可操作"})
			return
		}
		v, _ := techIDAny.(uint)
		techID = v
		if techID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可操作"})
			return
		}
	} else {
		if input.TechnicianID == nil || *input.TechnicianID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "technician_id 不能为空"})
			return
		}
		techID = *input.TechnicianID
	}

	allowed := map[string]bool{"idle": true, "paused": true}
	if !allowed[input.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的状态"})
		return
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 若岗位无需签到，自动补一条当天考勤记录，保证后续状态更新有记录可写
	{
		var tech models.Technician
		if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err == nil {
			if req, err := isRoleAttendanceRequired(config.DB, merchantID, tech.ServiceRoleID); err == nil && !req {
				_ = ensureAttendanceForNoCheckinRole(config.DB, merchantID, techID, now)
			}
		}
	}

	var out models.TechnicianAttendance
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var attendance models.TechnicianAttendance
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, techID, start).
			Order("id DESC").
			First(&attendance).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{}
		cur := attendance.Status
		// 4状态逻辑：
		// - idle: 允许手动切到 paused
		// - paused: 只允许手动切回 idle
		// - busy: 仅允许设置 next_status=paused（结单后自动暂停），不能直接改 status
		if cur == "idle" {
			if input.Status != "paused" {
				return apiErr{status: http.StatusBadRequest, msg: "空闲状态仅允许切换为暂停"}
			}
			updates["status"] = "paused"
			updates["next_status"] = nil
		} else if cur == "paused" {
			if input.Status != "idle" {
				return apiErr{status: http.StatusBadRequest, msg: "暂停状态仅允许切换为空闲"}
			}
			updates["status"] = "idle"
			updates["next_status"] = nil
		} else if cur == "busy" {
			if input.Status != "paused" {
				return apiErr{status: http.StatusBadRequest, msg: "服务中仅允许设置结单后进入暂停"}
			}
			ns := "paused"
			updates["next_status"] = &ns
		} else {
			return apiErr{status: http.StatusBadRequest, msg: "当前状态不可操作"}
		}

		if err := tx.Model(&models.TechnicianAttendance{}).Where("id = ?", attendance.ID).Updates(updates).Error; err != nil {
			return err
		}
		if cur == "paused" && input.Status == "idle" {
			tryAutoCallNextForTechnician(tx, merchantID, techID, now)
		}
		return tx.Preload("Technician").First(&out, attendance.ID).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "未签到"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

func ListAvailableTechnicians(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 按岗位独立配置签到：不再依赖商户全局开关

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var list []models.TechnicianAttendance
	config.DB.Preload("Technician").
		Where("merchant_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL AND status IN ('idle')", merchantID, start).
		Order("updated_at desc").
		Find(&list)

	c.JSON(http.StatusOK, gin.H{"data": list})
}

func GetCurrentTechnicianAttendance(c *gin.Context) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var technicianID uint
	if authType == "staff" {
		technicianIDAny, ok := c.Get("technician_id")
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可查看"})
			return
		}
		v, _ := technicianIDAny.(uint)
		technicianID = v
		if technicianID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可查看"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可查看"})
		return
	}

	var attendance models.TechnicianAttendance
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 按岗位独立配置签到：若岗位无需签到，确保当天存在考勤记录
	{
		var tech models.Technician
		if err := config.DB.Where("id = ? AND merchant_id = ?", technicianID, merchantID).First(&tech).Error; err == nil {
			if req, err := isRoleAttendanceRequired(config.DB, merchantID, tech.ServiceRoleID); err == nil && !req {
				_ = ensureAttendanceForNoCheckinRole(config.DB, merchantID, technicianID, now)
			}
		}
	}
	err := config.DB.Where("merchant_id = ? AND technician_id = ? AND created_at >= ?", merchantID, technicianID, start).
		Order("created_at DESC").
		First(&attendance).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": attendance})
}
