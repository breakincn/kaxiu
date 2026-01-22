package handlers

import (
	"errors"
	"kabao/config"
	"kabao/middleware"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BindUsageHandCard 绑定本次核销记录的手牌号（可跳过：hand_card_no 为空则保持未分配）
func BindUsageHandCard(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	usageID := strings.TrimSpace(c.Param("id"))
	if usageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usage_id 不能为空"})
		return
	}

	var input struct {
		HandCardNo string `json:"hand_card_no"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	handCardNo := strings.TrimSpace(input.HandCardNo)
	if handCardNo == "" {
		// 跳过：保持未分配
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var merchant models.Merchant
		if err := tx.First(&merchant, merchantID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "商户不存在"}
			}
			return err
		}
		if !merchant.SupportHandCard {
			return apiErr{status: http.StatusBadRequest, msg: "商户未开启手牌功能"}
		}

		var usage models.Usage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", usageID, merchantID).First(&usage).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "核销记录不存在"}
			}
			return err
		}

		// 已归还的不允许再绑定/修改
		if usage.HandCardReturnedAt != nil {
			return apiErr{status: http.StatusBadRequest, msg: "该手牌已归还，无法修改"}
		}

		// 已分配但未归还：不允许更换为其他号码（避免绕过“手牌号不重复”与追溯）
		if usage.HandCardNo != nil && strings.TrimSpace(*usage.HandCardNo) != "" {
			if *usage.HandCardNo != handCardNo {
				return apiErr{status: http.StatusBadRequest, msg: "该核销记录已绑定手牌，无法更换"}
			}
			return nil
		}

		now := time.Now()
		updates := map[string]interface{}{
			"hand_card_no":          handCardNo,
			"hand_card_assigned_at": &now,
		}
		if err := tx.Model(&models.Usage{}).Where("id = ? AND merchant_id = ? AND hand_card_no IS NULL", usage.ID, merchantID).Updates(updates).Error; err != nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func fmtInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}

// QueryHandCardForReturn 输入手牌号查询待归还的核销记录
func QueryHandCardForReturn(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	handCardNo := strings.TrimSpace(c.Query("hand_card_no"))
	if handCardNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hand_card_no 不能为空"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportHandCard {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启手牌功能"})
		return
	}

	var usage models.Usage
	err := config.DB.
		Preload("Card").Preload("Card.User").Preload("Project").Preload("Merchant").
		Where("merchant_id = ? AND hand_card_no = ? AND hand_card_assigned_at IS NOT NULL AND hand_card_returned_at IS NULL", merchantID, handCardNo).
		Order("id desc").
		First(&usage).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到待归还记录"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	usages := []models.Usage{usage}
	enrichUsagesWithServiceSession(&usages)
	usage = usages[0]

	c.JSON(http.StatusOK, gin.H{"data": usage})
}

// ReturnHandCard 确认归还手牌
func ReturnHandCard(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		HandCardNo string `json:"hand_card_no" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	handCardNo := strings.TrimSpace(input.HandCardNo)
	if handCardNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hand_card_no 不能为空"})
		return
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var merchant models.Merchant
		if err := tx.First(&merchant, merchantID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "商户不存在"}
			}
			return err
		}
		if !merchant.SupportHandCard {
			return apiErr{status: http.StatusBadRequest, msg: "商户未开启手牌功能"}
		}

		var usage models.Usage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND hand_card_no = ? AND hand_card_assigned_at IS NOT NULL AND hand_card_returned_at IS NULL", merchantID, handCardNo).
			Order("id desc").
			First(&usage).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "未找到待归还记录"}
			}
			return err
		}

		now := time.Now()
		if err := tx.Model(&models.Usage{}).
			Where("id = ? AND merchant_id = ? AND hand_card_returned_at IS NULL", usage.ID, merchantID).
			Update("hand_card_returned_at", &now).Error; err != nil {
			return err
		}

		// 若该卡所有“已分配”手牌均已归还，则自动解锁
		var cnt int64
		if err := tx.Model(&models.Usage{}).
			Where("card_id = ? AND merchant_id = ? AND hand_card_assigned_at IS NOT NULL AND hand_card_returned_at IS NULL", usage.CardID, merchantID).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			updates := map[string]interface{}{
				"locked":          false,
				"locked_reason":   "",
				"locked_at":       nil,
				"locked_by":       nil,
				"unlocked_at":     &now,
				"unlocked_by":     nil,
				"unlocked_reason": "手牌已全部归还",
			}
			if err := tx.Model(&models.Card{}).Where("id = ? AND merchant_id = ?", usage.CardID, merchantID).Updates(updates).Error; err != nil {
				return err
			}
		} else {
			// 仍有未归还手牌：更新锁卡原因，避免继续显示已归还的手牌号
			var nos []string
			if err := tx.Model(&models.Usage{}).
				Select("hand_card_no").
				Where("card_id = ? AND merchant_id = ? AND hand_card_assigned_at IS NOT NULL AND hand_card_returned_at IS NULL", usage.CardID, merchantID).
				Where("hand_card_no IS NOT NULL AND hand_card_no <> ''").
				Order("hand_card_assigned_at asc").
				Pluck("hand_card_no", &nos).Error; err != nil {
				return err
			}
			uniq := make([]string, 0, len(nos))
			seen := make(map[string]struct{}, len(nos))
			for _, n := range nos {
				v := strings.TrimSpace(n)
				if v == "" {
					continue
				}
				if _, ok := seen[v]; ok {
					continue
				}
				seen[v] = struct{}{}
				uniq = append(uniq, v)
			}
			reason := "你有" + fmtInt64(cnt) + "个未归还手牌"
			if len(uniq) > 0 {
				reason = reason + "：" + strings.Join(uniq, ",")
			}
			if err := tx.Model(&models.Card{}).
				Where("id = ? AND merchant_id = ? AND locked = ?", usage.CardID, merchantID, true).
				Update("locked_reason", reason).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// UnlockCardByMerchant 强制解锁卡片（需 merchant.card.unlock 权限）
func UnlockCardByMerchant(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	okPerm, err := middleware.HasPermission(c, "merchant.card.unlock")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "权限检查失败"})
		return
	}
	if !okPerm {
		c.JSON(http.StatusForbidden, gin.H{"error": "无解锁权限"})
		return
	}

	cardID := strings.TrimSpace(c.Param("id"))
	if cardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "card_id 不能为空"})
		return
	}

	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "手动解锁"
	}

	// 解锁人
	operatorIDAny, _ := c.Get("technician_id")
	operatorID, _ := operatorIDAny.(uint)

	now := time.Now()
	updates := map[string]interface{}{
		"locked":          false,
		"locked_reason":   "",
		"locked_at":       nil,
		"locked_by":       nil,
		"unlocked_at":     &now,
		"unlocked_reason": reason,
	}
	if operatorID > 0 {
		updates["unlocked_by"] = &operatorID
	} else {
		updates["unlocked_by"] = nil
	}

	if err := config.DB.Model(&models.Card{}).
		Where("id = ? AND merchant_id = ?", cardID, merchantID).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
