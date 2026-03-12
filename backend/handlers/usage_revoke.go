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

func UserRevokeUsage(c *gin.Context) {
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

	usageID := c.Param("id")
	var remainTimes int

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		var u models.Usage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Merchant").Where("id = ?", usageID).First(&u).Error; err != nil {
			return err
		}
		if u.Status != "in_progress" {
			return apiErr{status: http.StatusBadRequest, msg: "当前状态不可撤销"}
		}
		if u.UsedAt == nil {
			return apiErr{status: http.StatusBadRequest, msg: "核销记录异常"}
		}
		if !u.Merchant.SupportCustomerServiceMode {
			return apiErr{status: http.StatusBadRequest, msg: "该商户无需撤销"}
		}
		dl := revokeDeadlineAt(u.UsedAt)
		if dl == nil || !now.Before(*dl) {
			return apiErr{status: http.StatusBadRequest, msg: "已超过可撤销时间"}
		}

		var card models.Card
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, u.CardID).Error; err != nil {
			return err
		}
		if card.UserID != userID {
			return apiErr{status: http.StatusForbidden, msg: "无权操作"}
		}

		var s models.ServiceSession
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("initial_usage_id = ?", u.ID).
			Order("id desc").
			First(&s).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusBadRequest, msg: "找不到服务会话"}
			}
			return err
		}

		if s.StartTimeoutCount < 2 {
			startTerm := resolveStartTerm(&u.Merchant)
			return apiErr{status: http.StatusBadRequest, msg: startTerm + "超时未达到2次，不可撤销"}
		}
		if !isServiceSessionInRevokeablePhase(&u.Merchant, &s) {
			return apiErr{status: http.StatusBadRequest, msg: "当前阶段不可撤销"}
		}

		// 返还次数
		used := u.UsedTimes
		if used <= 0 {
			used = 1
		}
		if err := tx.Model(&models.Card{}).Where("id = ?", card.ID).Updates(map[string]interface{}{
			"remain_times": gorm.Expr("remain_times + ?", used),
			"used_times":   gorm.Expr("CASE WHEN used_times >= ? THEN used_times - ? ELSE 0 END", used, used),
		}).Error; err != nil {
			return err
		}
		if err := tx.First(&card, card.ID).Error; err != nil {
			return err
		}
		remainTimes = card.RemainTimes

		// 标记 usage 为失败（撤销）
		finishedAt := now
		if err := tx.Model(&models.Usage{}).Where("id = ? AND status = ?", u.ID, "in_progress").Updates(map[string]interface{}{
			"status":        "failed",
			"finished_at":   &finishedAt,
			"technician_id": nil,
		}).Error; err != nil {
			return err
		}

		// 释放技师状态
		if s.TechnicianID != nil && *s.TechnicianID > 0 {
			_ = tx.Model(&models.TechnicianAttendance{}).
				Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *s.TechnicianID, "busy").
				Updates(map[string]interface{}{"status": "idle"}).Error
		}

		// 取消并释放会话资源
		updates := map[string]interface{}{
			"status":                  models.ApplyStatusPrefix(s.Status, "canceled"),
			"technician_id":           nil,
			"room_id":                 nil,
			"room_locked_at":          nil,
			"room_select_deadline_at": nil,
		}
		return tx.Model(&models.ServiceSession{}).
			Where("id = ? AND status NOT IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"finished", "canceled"})).
			Updates(updates).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"remain_times": remainTimes}})
}
