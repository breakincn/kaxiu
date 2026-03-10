package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetMerchantRoleStartPendingSetting(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	roleKey := strings.TrimSpace(c.Param("roleKey"))
	if roleKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "roleKey不能为空"})
		return
	}

	role, err := config.FindMerchantUsableRole(config.DB, merchantID, roleKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.Select("id", "start_term").First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取失败"})
		return
	}

	overrideCfg, err := config.GetMerchantRoleStartPendingConfig(config.DB, merchantID, role.ID)
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"role":                                  role,
		"start_pending_timeout_seconds":         config.EffectiveRoleStartPendingTimeoutSeconds(role, overrideCfg),
		"default_start_pending_timeout_seconds": config.NormalizeRoleStartPendingTimeoutSeconds(role.StartPendingTimeoutSeconds),
		"has_start_pending_timeout_override":    overrideCfg != nil,
		"start_pending_timeout_label":           config.RoleStartPendingLabelForMerchant(&merchant),
	}})
}

func SetMerchantRoleStartPendingSetting(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	roleKey := strings.TrimSpace(c.Param("roleKey"))
	if roleKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "roleKey不能为空"})
		return
	}

	role, err := config.FindMerchantUsableRole(config.DB, merchantID, roleKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.Select("id", "start_term").First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取失败"})
		return
	}

	var input struct {
		StartPendingTimeoutSeconds *int `json:"start_pending_timeout_seconds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if *input.StartPendingTimeoutSeconds < 1 || *input.StartPendingTimeoutSeconds > 3600 {
		c.JSON(http.StatusBadRequest, gin.H{"error": config.RoleStartPendingLabelForMerchant(&merchant) + "范围应为 1-3600 秒"})
		return
	}

	seconds := config.NormalizeRoleStartPendingTimeoutSeconds(*input.StartPendingTimeoutSeconds)
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.MerchantRoleStartPendingConfig
		err := tx.Where("merchant_id = ? AND service_role_id = ?", merchantID, role.ID).First(&existing).Error
		if err == nil {
			return tx.Model(&models.MerchantRoleStartPendingConfig{}).
				Where("id = ?", existing.ID).
				Update("start_pending_timeout_seconds", seconds).Error
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return tx.Create(&models.MerchantRoleStartPendingConfig{
			MerchantID:                 merchantID,
			ServiceRoleID:              role.ID,
			StartPendingTimeoutSeconds: seconds,
		}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
