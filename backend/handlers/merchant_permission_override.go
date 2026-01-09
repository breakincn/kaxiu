package handlers

import (
	"kabao/config"
	"kabao/middleware"
	"kabao/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetMyPermissions(c *gin.Context) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType == "merchant" {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"permission_keys": []string{"*"}}})
		return
	}
	if authType != "staff" {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"permission_keys": []string{}}})
		return
	}

	var perms []models.Permission
	config.DB.Order("sort asc, id asc").Find(&perms)

	keys := make([]string, 0, len(perms))
	for _, p := range perms {
		if strings.TrimSpace(p.Key) == "" {
			continue
		}
		ok, err := middleware.HasPermission(c, p.Key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "权限读取失败"})
			return
		}
		if ok {
			keys = append(keys, p.Key)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"permission_keys": keys}})
}

func GetMerchantRolePermissionOverrides(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "technician" {
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

	var role models.ServiceRole
	if err := config.DB.Where("`key` = ?", roleKey).First(&role).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var perms []models.Permission
	config.DB.Order("sort asc, id asc").Find(&perms)

	type item struct {
		Permission       models.Permission `json:"permission"`
		DefaultAllowed   bool              `json:"default_allowed"`
		OverrideAllowed  *bool             `json:"override_allowed"`
		EffectiveAllowed bool              `json:"effective_allowed"`
	}

	items := make([]item, 0, len(perms))
	for _, p := range perms {
		defaultAllowed := false
		var rp models.RolePermission
		if err := config.DB.Where("service_role_id = ? AND permission_id = ? AND allowed = ?", role.ID, p.ID, true).First(&rp).Error; err == nil {
			defaultAllowed = true
		}

		var o models.MerchantRolePermissionOverride
		err := config.DB.Where("merchant_id = ? AND service_role_id = ? AND permission_id = ?", merchantID, role.ID, p.ID).First(&o).Error
		if err == nil {
			v := o.Allowed
			eff := v
			items = append(items, item{Permission: p, DefaultAllowed: defaultAllowed, OverrideAllowed: &v, EffectiveAllowed: eff})
			continue
		}
		if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取失败"})
			return
		}

		items = append(items, item{Permission: p, DefaultAllowed: defaultAllowed, OverrideAllowed: nil, EffectiveAllowed: defaultAllowed})
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"role": role, "items": items}})
}

func SetMerchantRolePermissionOverrides(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "technician" {
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

	var role models.ServiceRole
	if err := config.DB.Where("`key` = ?", roleKey).First(&role).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	if !role.AllowPermissionAdjust {
		c.JSON(http.StatusForbidden, gin.H{"error": "该角色不允许微调权限"})
		return
	}

	var input struct {
		Items []struct {
			PermissionKey string `json:"permission_key"`
			Allowed       bool   `json:"allowed"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, it := range input.Items {
		key := strings.TrimSpace(it.PermissionKey)
		if key == "" {
			continue
		}
		var perm models.Permission
		if err := config.DB.Where("`key` = ?", key).First(&perm).Error; err != nil {
			continue
		}

		var existing models.MerchantRolePermissionOverride
		err := config.DB.Where("merchant_id = ? AND service_role_id = ? AND permission_id = ?", merchantID, role.ID, perm.ID).First(&existing).Error
		if err == nil {
			config.DB.Model(&models.MerchantRolePermissionOverride{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{"allowed": it.Allowed})
			continue
		}
		if err != gorm.ErrRecordNotFound {
			continue
		}
		config.DB.Create(&models.MerchantRolePermissionOverride{MerchantID: merchantID, ServiceRoleID: role.ID, PermissionID: perm.ID, Allowed: it.Allowed})
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
