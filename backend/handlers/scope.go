package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func ensureMerchantScope(c *gin.Context, param string) (uint, bool) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return 0, false
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return 0, false
	}

	raw := strings.TrimSpace(c.Param(param))
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商户ID"})
		return 0, false
	}
	routeID64, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || routeID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商户ID"})
		return 0, false
	}
	routeID := uint(routeID64)
	if routeID != merchantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "越权访问商户资源"})
		return 0, false
	}
	return merchantID, true
}

func merchantIDFromRouteParam(c *gin.Context, param string) (uint, bool) {
	raw := strings.TrimSpace(c.Param(param))
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商户ID"})
		return 0, false
	}
	routeID64, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || routeID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商户ID"})
		return 0, false
	}
	return uint(routeID64), true
}

func mustUserID(c *gin.Context) (uint, bool) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return 0, false
	}
	userID, ok := userIDAny.(uint)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return 0, false
	}
	return userID, true
}
