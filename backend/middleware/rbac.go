package middleware

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RequirePermission(permissionKey string) gin.HandlerFunc {
	key := strings.TrimSpace(permissionKey)
	return func(c *gin.Context) {
		if key == "" {
			c.Next()
			return
		}
		ok, err := hasPermission(c, key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
			c.Abort()
			return
		}
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func hasPermission(c *gin.Context, permissionKey string) (bool, error) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType == "merchant" {
		return true, nil
	}
	if authType != "technician" {
		return false, nil
	}

	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		return false, nil
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		return false, nil
	}

	techAny, ok := c.Get("technician")
	if !ok {
		return false, nil
	}
	tech, ok := techAny.(models.Technician)
	if !ok {
		return false, nil
	}
	if tech.ServiceRoleID == 0 {
		return false, nil
	}

	var perm models.Permission
	if err := config.DB.Where("`key` = ?", permissionKey).First(&perm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	// override 优先
	var override models.MerchantRolePermissionOverride
	err := config.DB.Where("merchant_id = ? AND service_role_id = ? AND permission_id = ?", merchantID, tech.ServiceRoleID, perm.ID).First(&override).Error
	if err == nil {
		return override.Allowed, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}

	var rp models.RolePermission
	err = config.DB.Where("service_role_id = ? AND permission_id = ? AND allowed = ?", tech.ServiceRoleID, perm.ID, true).First(&rp).Error
	if err == nil {
		return true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	return false, err
}
