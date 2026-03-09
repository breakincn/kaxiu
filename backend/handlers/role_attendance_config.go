package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type roleAttendanceItem struct {
	ServiceRoleID       uint   `json:"service_role_id"`
	ServiceRoleKey      string `json:"service_role_key"`
	ServiceRoleName     string `json:"service_role_name"`
	RoleType            string `json:"role_type"`
	RequireAttendance   bool   `json:"require_attendance"`
	DefaultRequire      bool   `json:"default_require_attendance"`
	HasMerchantOverride bool   `json:"has_merchant_override"`
}

// GetMerchantRoleAttendanceConfigs 返回商户各岗位是否需要签到的配置（含默认值+商户覆盖）
func GetMerchantRoleAttendanceConfigs(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var roles []models.ServiceRole
	config.DB.
		Where("is_active = ? AND ((merchant_id IS NULL AND `key` IN ?) OR merchant_id = ?)", true, config.FixedServiceRoleKeys(), merchantID).
		Order("role_type asc, sort asc, id asc").
		Find(&roles)

	var overrides []models.MerchantRoleAttendanceConfig
	config.DB.Where("merchant_id = ?", merchantID).Find(&overrides)
	overrideByRole := map[uint]models.MerchantRoleAttendanceConfig{}
	for _, o := range overrides {
		overrideByRole[o.ServiceRoleID] = o
	}

	out := make([]roleAttendanceItem, 0, len(roles))
	for _, r := range roles {
		req := r.RequireAttendance
		has := false
		if o, ok := overrideByRole[r.ID]; ok {
			req = o.RequireAttendance
			has = true
		}
		out = append(out, roleAttendanceItem{
			ServiceRoleID:       r.ID,
			ServiceRoleKey:      r.Key,
			ServiceRoleName:     r.Name,
			RoleType:            strings.TrimSpace(r.RoleType),
			RequireAttendance:   req,
			DefaultRequire:      r.RequireAttendance,
			HasMerchantOverride: has,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

// SetMerchantRoleAttendanceConfig 设置某岗位是否需要签到（商户覆盖）
func SetMerchantRoleAttendanceConfig(c *gin.Context) {
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

	var input struct {
		RequireAttendance *bool `json:"require_attendance" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role, err := config.FindMerchantUsableRole(config.DB, merchantID, roleKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色不存在"})
		return
	}

	require := *input.RequireAttendance
	// upsert 覆盖配置
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.MerchantRoleAttendanceConfig
		err := tx.Where("merchant_id = ? AND service_role_id = ?", merchantID, role.ID).First(&existing).Error
		if err == nil {
			return tx.Model(&models.MerchantRoleAttendanceConfig{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{"require_attendance": require}).Error
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return tx.Create(&models.MerchantRoleAttendanceConfig{MerchantID: merchantID, ServiceRoleID: role.ID, RequireAttendance: require}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
