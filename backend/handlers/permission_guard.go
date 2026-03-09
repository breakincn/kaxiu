package handlers

import (
	"kabao/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func hasMerchantPermissionInHandler(c *gin.Context, permissionKey string) (bool, error) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType == "merchant" {
		return true, nil
	}

	ok, err := middleware.HasPermission(c, permissionKey)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func requireMerchantPermissionInHandler(c *gin.Context, permissionKey string) bool {
	ok, err := hasMerchantPermissionInHandler(c, permissionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
		return false
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return false
	}
	return true
}

func requireAnyMerchantPermissionInHandler(c *gin.Context, permissionKeys ...string) bool {
	for _, permissionKey := range permissionKeys {
		ok, err := hasMerchantPermissionInHandler(c, permissionKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
			return false
		}
		if ok {
			return true
		}
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
	return false
}
