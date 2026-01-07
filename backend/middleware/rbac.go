package middleware

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireAnyPermission(permissionKeys ...string) gin.HandlerFunc {
	keys := make([]string, 0, len(permissionKeys))
	for _, k := range permissionKeys {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}

	return func(c *gin.Context) {
		if len(keys) == 0 {
			c.Next()
			return
		}
		for _, k := range keys {
			ok, err := HasPermission(c, k)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
				c.Abort()
				return
			}
			if ok {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		c.Abort()
	}
}

func RequirePermission(permissionKey string) gin.HandlerFunc {
	key := strings.TrimSpace(permissionKey)
	return func(c *gin.Context) {
		if key == "" {
			c.Next()
			return
		}
		ok, err := HasPermission(c, key)
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

func HasPermission(c *gin.Context, permissionKey string) (bool, error) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType == "merchant" {
		return true, nil
	}
	if authType != "staff" {
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

	serviceRoleIDAny, ok := c.Get("service_role_id")
	if !ok {
		return false, nil
	}
	serviceRoleID, ok := serviceRoleIDAny.(uint)
	if !ok || serviceRoleID == 0 {
		return false, nil
	}

	// 兼容技师逻辑（如果技师信息存在，优先使用）
	if techAny, ok := c.Get("technician"); ok {
		if tech, ok := techAny.(models.Technician); ok {
			serviceRoleID = tech.ServiceRoleID
		}
	}

	// 查找权限
	var perm models.Permission
	if err := config.DB.Where("`key` = ?", permissionKey).First(&perm).Error; err != nil {
		return false, nil
	}

	// 优先检查商户级别的权限覆盖
	var override models.MerchantRolePermissionOverride
	err := config.DB.Where("merchant_id = ? AND service_role_id = ? AND permission_id = ?", merchantID, serviceRoleID, perm.ID).First(&override).Error
	if err == nil {
		return override.Allowed, nil
	}

	// 检查全局角色权限
	var rolePerm models.RolePermission
	err = config.DB.Where("service_role_id = ? AND permission_id = ? AND allowed = ?", serviceRoleID, perm.ID, true).First(&rolePerm).Error
	if err == nil {
		return true, nil
	}

	return false, nil
}
