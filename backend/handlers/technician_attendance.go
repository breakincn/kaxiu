package handlers

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportTechnicianCheckin {
		c.JSON(http.StatusForbidden, gin.H{"error": "该商户未启用工作人员签到"})
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

	var attendance models.TechnicianAttendance
	err := config.DB.Where("merchant_id = ? AND technician_id = ?", merchantID, techID).First(&attendance).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		attendance = models.TechnicianAttendance{MerchantID: merchantID, TechnicianID: techID, CheckedInAt: &now, CheckedOutAt: nil, Status: "idle", NextStatus: nil}
		if err := config.DB.Create(&attendance).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		config.DB.Preload("Technician").First(&attendance, attendance.ID)
		c.JSON(http.StatusOK, gin.H{"data": attendance})
		return
	}

	// 当天已上班且未下班：不允许重复上班签到
	if attendance.CheckedInAt != nil && sameLocalDay(*attendance.CheckedInAt, now) && attendance.CheckedOutAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "今日已上班签到，请先下班签到"})
		return
	}

	updates := map[string]interface{}{"checked_in_at": now, "checked_out_at": nil, "status": "idle", "next_status": nil}
	if err := config.DB.Model(&models.TechnicianAttendance{}).Where("id = ?", attendance.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	config.DB.Preload("Technician").First(&attendance, attendance.ID)
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportTechnicianCheckin {
		c.JSON(http.StatusForbidden, gin.H{"error": "该商户未启用工作人员签到"})
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

	var attendance models.TechnicianAttendance
	if err := config.DB.Where("merchant_id = ? AND technician_id = ?", merchantID, techID).First(&attendance).Error; err != nil {
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportTechnicianCheckin {
		c.JSON(http.StatusForbidden, gin.H{"error": "该商户未启用工作人员签到"})
		return
	}

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

	var out models.TechnicianAttendance
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var attendance models.TechnicianAttendance
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ?", merchantID, techID).
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportTechnicianCheckin {
		c.JSON(http.StatusForbidden, gin.H{"error": "该商户未启用工作人员签到"})
		return
	}

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
	err := config.DB.Where("merchant_id = ? AND technician_id = ?", merchantID, technicianID).
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
