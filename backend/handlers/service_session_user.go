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

func UserGetServiceSession(c *gin.Context) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	userID, _ := userIDAny.(uint)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	id := c.Param("id")
	var s models.ServiceSession
	if err := config.DB.Preload("Room").Preload("Technician").Preload("Technician.ServiceRole").Where("id = ? AND user_id = ?", id, userID).First(&s).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s})
}

func UserGetVerifyCodeStatus(c *gin.Context) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	userID, _ := userIDAny.(uint)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code 不能为空"})
		return
	}

	var vc models.VerifyCode
	if err := config.DB.Where("code = ?", code).First(&vc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "核销码不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var card models.Card
	if err := config.DB.First(&card, vc.CardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	if card.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此核销码"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, card.MerchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	resp := gin.H{
		"code":                  vc.Code,
		"card_id":               vc.CardID,
		"expire_at":             vc.ExpireAt,
		"used":                  vc.Used,
		"used_at":               vc.UsedAt,
		"merchant_support_room": merchant.SupportRoom,
	}

	if vc.Used {
		var s models.ServiceSession
		err := config.DB.
			Where("verify_code = ? AND user_id = ? AND card_id = ?", vc.Code, userID, card.ID).
			Order("id desc").
			First(&s).Error
		if err == nil {
			nextStep := ""
			if s.Status == "room_selecting" {
				nextStep = "room_select"
			} else if s.Status == "staff_selecting" || s.Status == "room_locked" {
				nextStep = "staff_select"
			}
			resp["session_id"] = s.ID
			resp["session_status"] = s.Status
			resp["next_step"] = nextStep
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func UserListAvailableRooms(c *gin.Context) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	userID, _ := userIDAny.(uint)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	sid := c.Param("id")
	var s models.ServiceSession
	if err := config.DB.Where("id = ? AND user_id = ?", sid, userID).First(&s).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, s.MerchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportRoom {
		c.JSON(http.StatusOK, gin.H{"data": []models.Room{}})
		return
	}

	var rooms []models.Room
	config.DB.Where("merchant_id = ? AND is_active = ?", s.MerchantID, true).Order("id asc").Find(&rooms)

	available := make([]models.Room, 0, len(rooms))
	for i := range rooms {
		r := rooms[i]
		var cnt int64
		config.DB.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','start_pending','delay_pending','serving','auto_finishing')", s.MerchantID, r.ID).
			Count(&cnt)
		if cnt == 0 {
			available = append(available, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": available})
}

func UserChooseServiceSessionRoom(c *gin.Context) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	userID, _ := userIDAny.(uint)
	if userID == 0 {
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
	autoAdjusted := false
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", sid, userID).First(&s).Error; err != nil {
			return err
		}

		var merchant models.Merchant
		if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
			return err
		}
		if !merchant.SupportRoom {
			return apiErr{status: http.StatusBadRequest, msg: "商户未开启房间功能"}
		}
		if s.Status != "room_selecting" {
			return apiErr{status: http.StatusBadRequest, msg: "当前状态不可选房"}
		}

		var room models.Room
		if err := tx.Where("id = ? AND merchant_id = ? AND is_active = ?", input.RoomID, s.MerchantID, true).First(&room).Error; err != nil {
			return apiErr{status: http.StatusBadRequest, msg: "房间不可用"}
		}

		var cnt int64
		if err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','start_pending','delay_pending','serving','auto_finishing')", s.MerchantID, room.ID).
			Count(&cnt).Error; err != nil {
			return err
		}
		chosenRoomID := room.ID
		if cnt > 0 {
			autoAdjusted = true
			// 自动分配其它可用房间
			var rooms []models.Room
			if err := tx.Where("merchant_id = ? AND is_active = ?", s.MerchantID, true).Order("id asc").Find(&rooms).Error; err != nil {
				return err
			}
			found := false
			for i := range rooms {
				r := rooms[i]
				if r.ID == chosenRoomID {
					continue
				}
				var c2 int64
				if err := tx.Model(&models.ServiceSession{}).
					Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','start_pending','delay_pending','serving','auto_finishing')", s.MerchantID, r.ID).
					Count(&c2).Error; err != nil {
					return err
				}
				if c2 == 0 {
					chosenRoomID = r.ID
					found = true
					break
				}
			}
			if !found {
				return apiErr{status: http.StatusBadRequest, msg: "无可用房间"}
			}
		}

		lockedAt := now
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
			"room_id":        chosenRoomID,
			"room_locked_at": lockedAt,
			"status":         "staff_selecting",
		}).Error; err != nil {
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

	resp := gin.H{"data": out}
	if autoAdjusted {
		resp["room_auto_adjusted"] = true
		resp["message"] = "房间已占用，系统已自动调整"
	}
	c.JSON(http.StatusOK, resp)
}

func UserListAvailableTechnicians(c *gin.Context) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	userID, _ := userIDAny.(uint)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	sid := c.Param("id")
	var s models.ServiceSession
	if err := config.DB.Where("id = ? AND user_id = ?", sid, userID).First(&s).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 记录用户进入选择客服页的时间：以“拉取可选客服列表”为准，仅首次写入
	now := time.Now()
	if s.Status == "staff_selecting" || s.Status == "room_locked" {
		if s.StaffSelectEnteredAt == nil {
			config.DB.Model(&models.ServiceSession{}).
				Where("id = ? AND user_id = ? AND staff_select_entered_at IS NULL AND status IN ('staff_selecting','room_locked')", s.ID, userID).
				Update("staff_select_entered_at", now)
			s.StaffSelectEnteredAt = &now
		}
	}
	if s.StaffSelectCooldownUntil != nil && now.Before(*s.StaffSelectCooldownUntil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前没有空闲客服，3分钟后可再次选择客服"})
		return
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	activeSessionStatuses := []string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"}
	var list []models.TechnicianAttendance
	config.DB.
		Model(&models.TechnicianAttendance{}).
		Joins("JOIN technicians t ON t.id = technician_attendances.technician_id").
		Joins("JOIN service_roles sr ON sr.id = t.service_role_id").
		Preload("Technician").
		Preload("Technician.ServiceRole").
		Where("technician_attendances.merchant_id = ? AND technician_attendances.checked_in_at >= ? AND technician_attendances.checked_out_at IS NULL AND technician_attendances.status IN ('idle')", s.MerchantID, start).
		Where("NOT EXISTS (SELECT 1 FROM service_sessions ss WHERE ss.merchant_id = ? AND ss.technician_id = technician_attendances.technician_id AND ss.status IN ?)", s.MerchantID, activeSessionStatuses).
		Where("t.is_active = ?", true).
		Where("sr.role_type = ? AND sr.`key` NOT IN ('store_manager','front_desk')", "professional").
		Order("technician_attendances.updated_at asc").
		Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func UserChooseServiceSessionTechnician(c *gin.Context) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	userID, _ := userIDAny.(uint)
	if userID == 0 {
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
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var out models.ServiceSession

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", sid, userID).First(&s).Error; err != nil {
			return err
		}
		if s.Status != "staff_selecting" && s.Status != "room_locked" {
			return apiErr{status: http.StatusBadRequest, msg: "当前状态不可选工作人员"}
		}
		if s.StaffSelectCooldownUntil != nil && now.Before(*s.StaffSelectCooldownUntil) {
			return apiErr{status: http.StatusBadRequest, msg: "当前没有空闲客服，3分钟后可再次选择客服"}
		}

		var merchant models.Merchant
		if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
			return err
		}
		if merchant.SupportRoom && s.RoomID == nil {
			return apiErr{status: http.StatusBadRequest, msg: "请先选择房间"}
		}

		// 避免对带 JOIN 的查询直接加 FOR UPDATE，MySQL/InnoDB 可能扩大锁范围并导致锁等待。
		// 先无锁选出候选记录，再按主键加锁二次校验。
		var candidate models.TechnicianAttendance
		if err := tx.
			Model(&models.TechnicianAttendance{}).
			Joins("JOIN technicians t ON t.id = technician_attendances.technician_id").
			Joins("JOIN service_roles sr ON sr.id = t.service_role_id").
			Where("technician_attendances.merchant_id = ? AND technician_attendances.technician_id = ? AND technician_attendances.checked_in_at >= ? AND technician_attendances.checked_out_at IS NULL AND technician_attendances.status IN ('idle')", s.MerchantID, input.TechnicianID, start).
			Where("NOT EXISTS (SELECT 1 FROM service_sessions ss WHERE ss.merchant_id = ? AND ss.technician_id = technician_attendances.technician_id AND ss.status IN ?)", s.MerchantID, []string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"}).
			Where("t.is_active = ?", true).
			Where("sr.role_type = ? AND sr.`key` NOT IN ('store_manager','front_desk')", "professional").
			First(&candidate).Error; err != nil {
			return apiErr{status: http.StatusBadRequest, msg: "工作人员不可选"}
		}

		var att models.TechnicianAttendance
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL AND status = ?", candidate.ID, s.MerchantID, input.TechnicianID, start, "idle").
			First(&att).Error; err != nil {
			return apiErr{status: http.StatusBadRequest, msg: "工作人员不可选"}
		}

		if err := tx.Model(&models.TechnicianAttendance{}).Where("id = ?", att.ID).Update("status", "busy").Error; err != nil {
			return err
		}

		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
			"technician_id": input.TechnicianID,
			"status":        "start_pending",
			"staff_select_entered_at": nil,
			"staff_select_cooldown_until": nil,
			"start_pending_timeout_seconds": int(config.StartPendingTimeout().Seconds()),
		}).Error; err != nil {
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

func ExtendServiceSessionUser(c *gin.Context) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	userID, _ := userIDAny.(uint)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	id := c.Param("id")
	var input struct {
		Minutes uint `json:"minutes" binding:"required,min=5,max=180"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var out models.ServiceSession
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", id, userID).First(&s).Error; err != nil {
			return err
		}

		if s.Status != "serving" {
			return apiErr{status: http.StatusBadRequest, msg: "仅服务中可加钟"}
		}

		// 延长服务时长与自动结单延迟
		updates := map[string]interface{}{
			"duration_minutes":          gorm.Expr("duration_minutes + ?", input.Minutes),
			"auto_finish_delay_seconds": gorm.Expr("auto_finish_delay_seconds + ?", input.Minutes*60),
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
