package handlers

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
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
	if err := config.DB.Preload("Room").Preload("Technician").Where("id = ? AND user_id = ?", id, userID).First(&s).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s})
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
			Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','precheck_pending','delay_pending','serving','auto_finishing')", s.MerchantID, r.ID).
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
			Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','precheck_pending','delay_pending','serving','auto_finishing')", s.MerchantID, room.ID).
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
					Where("merchant_id = ? AND room_id = ? AND status IN ('room_locked','staff_selecting','precheck_pending','delay_pending','serving','auto_finishing')", s.MerchantID, r.ID).
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

	cutoff := time.Now().Add(-15 * time.Minute)
	var list []models.TechnicianAttendance
	config.DB.Preload("Technician").
		Where("merchant_id = ? AND checked_in_at >= ? AND status IN ('available','idle')", s.MerchantID, cutoff).
		Order("updated_at desc").
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
	cutoff := time.Now().Add(-15 * time.Minute)
	var out models.ServiceSession

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", sid, userID).First(&s).Error; err != nil {
			return err
		}
		if s.Status != "staff_selecting" && s.Status != "room_locked" {
			return apiErr{status: http.StatusBadRequest, msg: "当前状态不可选工作人员"}
		}

		var merchant models.Merchant
		if err := tx.First(&merchant, s.MerchantID).Error; err != nil {
			return err
		}
		if merchant.SupportRoom && s.RoomID == nil {
			return apiErr{status: http.StatusBadRequest, msg: "请先选择房间"}
		}

		var att models.TechnicianAttendance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND status IN ('available','idle')", s.MerchantID, input.TechnicianID, cutoff).
			First(&att).Error; err != nil {
			return apiErr{status: http.StatusBadRequest, msg: "工作人员不可选"}
		}

		if err := tx.Model(&models.TechnicianAttendance{}).Where("id = ?", att.ID).Update("status", "busy").Error; err != nil {
			return err
		}

		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
			"technician_id": input.TechnicianID,
			"status":        "precheck_pending",
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

	c.JSON(http.StatusOK, gin.H{"data": out, "precheck_code": "SS:" + strconv.FormatUint(uint64(out.ID), 10)})
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
