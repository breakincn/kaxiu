package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetMerchantNotices(c *gin.Context) {
	merchantIDParam := strings.TrimSpace(c.Param("id"))
	merchantID64, err := strconv.ParseUint(merchantIDParam, 10, 32)
	if err != nil || merchantID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商户ID"})
		return
	}
	merchantID := uint(merchantID64)

	// 商户端请求必须满足租户边界；用户端允许读取商户通知用于展示。
	if _, hasMerchant := c.Get("merchant_id"); hasMerchant {
		if _, ok := ensureMerchantScope(c, "id"); !ok {
			return
		}
	} else if _, hasUser := c.Get("user_id"); !hasUser {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	limit := c.DefaultQuery("limit", "10")

	var notices []models.Notice
	// 置顶的通知优先显示，然后按创建时间倒序
	config.DB.Where("merchant_id = ?", merchantID).
		Order("is_pinned DESC, created_at DESC").
		Limit(3).
		Find(&notices)

	_ = limit
	c.JSON(http.StatusOK, gin.H{"data": notices})
}

func CreateNotice(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		MerchantID *uint  `json:"merchant_id"`
		Title      string `json:"title" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.MerchantID != nil && *input.MerchantID != merchantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "merchant_id 与登录态不一致"})
		return
	}

	// 检查该商户的通知数量
	var count int64
	config.DB.Model(&models.Notice{}).Where("merchant_id = ?", merchantID).Count(&count)
	if count >= 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "最多只能发布3条通知，请先删除一条后再发布"})
		return
	}

	notice := models.Notice{
		MerchantID: merchantID,
		Title:      input.Title,
		Content:    input.Content,
		IsPinned:   false,
		CreatedAt:  func() *time.Time { t := time.Now(); return &t }(),
	}
	config.DB.Create(&notice)
	c.JSON(http.StatusOK, gin.H{"data": notice})
}

// 删除通知
func DeleteNotice(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	noticeID := c.Param("id")

	var notice models.Notice
	if err := config.DB.Where("id = ? AND merchant_id = ?", noticeID, merchantID).First(&notice).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "通知不存在"})
		return
	}

	config.DB.Delete(&notice)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// 置顶/取消置顶通知
func TogglePinNotice(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	noticeID := c.Param("id")

	var notice models.Notice
	if err := config.DB.Where("id = ? AND merchant_id = ?", noticeID, merchantID).First(&notice).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "通知不存在"})
		return
	}

	// 如果要置顶，先取消该商户其他通知的置顶
	if !notice.IsPinned {
		config.DB.Model(&models.Notice{}).Where("merchant_id = ? AND id != ?", notice.MerchantID, noticeID).
			Update("is_pinned", false)
	}

	// 切换置顶状态
	notice.IsPinned = !notice.IsPinned
	config.DB.Save(&notice)

	c.JSON(http.StatusOK, gin.H{"data": notice})
}
