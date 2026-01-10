package handlers

import (
	"errors"
	"fmt"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func handleServiceSessionPrecheckScan(c *gin.Context, raw string) bool {
	code := strings.TrimSpace(raw)
	if !strings.HasPrefix(code, "SS:") {
		return false
	}
	sidStr := strings.TrimSpace(strings.TrimPrefix(code, "SS:"))
	sid64, err := strconv.ParseUint(sidStr, 10, 64)
	if err != nil || sid64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话码"})
		return true
	}

	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return true
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return true
	}

	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	var techID *uint
	if authType == "staff" {
		techIDAny, ok := c.Get("technician_id")
		if ok {
			if v, ok := techIDAny.(uint); ok && v > 0 {
				techID = &v
			}
		}
		if techID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可预结单"})
			return true
		}
	}

	now := time.Now()
	var out models.ServiceSession
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", uint(sid64), merchantID).First(&s).Error; err != nil {
			return err
		}

		var merchant models.Merchant
		if err := tx.First(&merchant, merchantID).Error; err != nil {
			return err
		}

		if merchant.SupportRoom && s.RoomID == nil {
			if err := assignRoomIfPossible(tx, &s, now); err != nil {
				return err
			}
			if s.RoomID == nil {
				return apiErr{status: http.StatusBadRequest, msg: "无可用房间"}
			}
		}

		if techID != nil {
			s.TechnicianID = techID
		}

		if s.TechnicianID == nil {
			return apiErr{status: http.StatusBadRequest, msg: "未选择工作人员"}
		}

		if s.Status == "finished" || s.Status == "canceled" {
			return apiErr{status: http.StatusBadRequest, msg: "会话状态不可预结单"}
		}

		startAt := now.Add(time.Duration(s.DelaySeconds) * time.Second)
		updates := map[string]interface{}{
			"precheck_at":        now,
			"scheduled_start_at": startAt,
			"status":             "delay_pending",
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.TechnicianAttendance{}).
			Where("merchant_id = ? AND technician_id = ?", merchantID, *s.TechnicianID).
			Updates(map[string]interface{}{"status": "busy"}).Error; err != nil {
			return err
		}

		if err := tx.Preload("Room").Preload("Technician").First(&out, s.ID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return true
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return true
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return true
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"action":     "precheck",
		"session_id": out.ID,
		"session":    out,
	}})
	return true
}

func assignRoomIfPossible(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	var rooms []models.Room
	if err := tx.Where("merchant_id = ? AND is_active = ?", s.MerchantID, true).Order("id asc").Find(&rooms).Error; err != nil {
		return err
	}
	for i := range rooms {
		r := rooms[i]
		var cnt int64
		if err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','precheck_pending','delay_pending','serving','auto_finishing')", s.MerchantID, r.ID).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			lockedAt := now
			s.RoomID = &r.ID
			s.RoomLockedAt = &lockedAt
			return tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
				"room_id":        r.ID,
				"room_locked_at": lockedAt,
				"status":         "room_locked",
			}).Error
		}
	}
	return nil
}

func GetServiceSession(c *gin.Context) {
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

	id := c.Param("id")
	var s models.ServiceSession
	if err := config.DB.Preload("Room").Preload("Technician").Where("id = ? AND merchant_id = ?", id, merchantID).First(&s).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s})
}

func ListServiceSessions(c *gin.Context) {
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

	status := c.Query("status")
	q := config.DB.Preload("Room").Preload("Technician").Where("merchant_id = ?", merchantID)
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var list []models.ServiceSession
	q.Order("id desc").Limit(200).Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func ChooseServiceSessionRoom(c *gin.Context) {
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

	var input struct {
		RoomID uint `json:"room_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sid := c.Param("id")
	now := time.Now()

	var out models.ServiceSession
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", sid, merchantID).First(&s).Error; err != nil {
			return err
		}
		if s.Status != "room_selecting" {
			return apiErr{status: http.StatusBadRequest, msg: "当前状态不可选房"}
		}

		var merchant models.Merchant
		if err := tx.First(&merchant, merchantID).Error; err != nil {
			return err
		}
		if !merchant.SupportRoom {
			return apiErr{status: http.StatusBadRequest, msg: "商户未开启房间功能"}
		}

		var room models.Room
		if err := tx.Where("id = ? AND merchant_id = ? AND is_active = ?", input.RoomID, merchantID, true).First(&room).Error; err != nil {
			return apiErr{status: http.StatusBadRequest, msg: "房间不可用"}
		}

		var cnt int64
		if err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','precheck_pending','delay_pending','serving','auto_finishing')", merchantID, room.ID).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			return apiErr{status: http.StatusBadRequest, msg: "房间已占用"}
		}

		lockedAt := now
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
			"room_id":        room.ID,
			"room_locked_at": lockedAt,
			"status":         "room_locked",
		}).Error; err != nil {
			return err
		}
		if err := tx.Preload("Room").Preload("Technician").First(&out, s.ID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

func ExtendServiceSessionDuration(c *gin.Context) {
	// 技师或商户都可操作
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

	var input struct {
		Minutes uint `json:"minutes" binding:"required,min=5,max=180"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sid := c.Param("id")
	var out models.ServiceSession
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", sid, merchantID).First(&s).Error; err != nil {
			return err
		}

		if s.Status != "serving" {
			return apiErr{status: http.StatusBadRequest, msg: "仅服务中可延长"}
		}

		// 延长服务时长与自动结单延迟
		updates := map[string]interface{}{
			"duration_minutes":          gorm.Expr("duration_minutes + ?", input.Minutes),
			"auto_finish_delay_seconds": gorm.Expr("auto_finish_delay_seconds + ?", input.Minutes*60),
		}
		if s.ScheduledFinishAt != nil {
			updates["scheduled_finish_at"] = s.ScheduledFinishAt.Add(time.Duration(input.Minutes) * time.Minute)
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}

		return tx.Preload("Room").Preload("Technician").First(&out, s.ID).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

func ChooseServiceSessionTechnician(c *gin.Context) {
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

	var input struct {
		TechnicianID uint `json:"technician_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sid := c.Param("id")
	cutoff := time.Now().Add(-15 * time.Minute)
	var out models.ServiceSession

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", sid, merchantID).First(&s).Error; err != nil {
			return err
		}
		if s.Status != "staff_selecting" && s.Status != "room_locked" {
			return apiErr{status: http.StatusBadRequest, msg: "当前状态不可选工作人员"}
		}

		var att models.TechnicianAttendance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND status IN ('available','idle')", merchantID, input.TechnicianID, cutoff).
			First(&att).Error; err != nil {
			return apiErr{status: http.StatusBadRequest, msg: "工作人员不可选"}
		}

		if err := tx.Model(&models.TechnicianAttendance{}).Where("id = ?", att.ID).Update("status", "busy").Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"technician_id": input.TechnicianID,
			"status":        "precheck_pending",
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Preload("Room").Preload("Technician").First(&out, s.ID).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": out, "precheck_code": "SS:" + strconv.FormatUint(uint64(out.ID), 10)})
}

func ExtendServiceSession(c *gin.Context) {
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

	var input struct {
		AddMinutes *int `json:"add_minutes"`
	}
	_ = c.ShouldBindJSON(&input)

	sid := c.Param("id")
	now := time.Now()
	var remainTimes int
	var out models.ServiceSession

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", sid, merchantID).First(&s).Error; err != nil {
			return err
		}
		if s.Status == "finished" || s.Status == "canceled" {
			return apiErr{status: http.StatusBadRequest, msg: "会话已结束"}
		}

		var merchant models.Merchant
		if err := tx.First(&merchant, merchantID).Error; err != nil {
			return err
		}
		addMinutes := merchant.AvgServiceMinutes
		if input.AddMinutes != nil && *input.AddMinutes > 0 {
			addMinutes = *input.AddMinutes
		}
		if addMinutes <= 0 {
			addMinutes = 50
		}

		var card models.Card
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, s.CardID).Error; err != nil {
			return err
		}
		if card.RemainTimes <= 0 {
			return apiErr{status: http.StatusBadRequest, msg: "剩余次数不足"}
		}
		if err := tx.Model(&models.Card{}).Where("id = ? AND remain_times > 0", card.ID).Updates(map[string]interface{}{
			"remain_times": gorm.Expr("remain_times - ?", 1),
			"used_times":   gorm.Expr("used_times + ?", 1),
			"last_used_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.First(&card, card.ID).Error; err != nil {
			return err
		}
		remainTimes = card.RemainTimes

		extCode := fmt.Sprintf("EXT-%d-%d", s.ID, now.Unix())
		u := models.Usage{
			CardID:     card.ID,
			MerchantID: card.MerchantID,
			UsedTimes:  1,
			UsedAt:     &now,
			VerifyCode: extCode,
			Status:     "success",
			FinishedAt: &now,
		}
		if err := tx.Create(&u).Error; err != nil {
			return err
		}

		newDuration := s.DurationMinutes + addMinutes
		updates := map[string]interface{}{"duration_minutes": newDuration}
		if s.ScheduledFinishAt != nil {
			updates["scheduled_finish_at"] = s.ScheduledFinishAt.Add(time.Duration(addMinutes) * time.Minute)
		} else if s.StartedAt != nil {
			updates["scheduled_finish_at"] = s.StartedAt.Add(time.Duration(newDuration) * time.Minute)
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Preload("Room").Preload("Technician").First(&out, s.ID).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": out, "remain_times": remainTimes})
}
