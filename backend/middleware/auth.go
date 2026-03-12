package middleware

import (
	"kabao/config"
	"kabao/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var legacyUserTokenHitCount atomic.Uint64

// AuthMiddleware 验证用户/商户/员工登录状态
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("Authorization"))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}
		token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}

		// 兼容旧版用户 token: user_{userID}_{timestamp}
		if strings.HasPrefix(token, "user_") {
			if !config.AllowLegacyUserToken() {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "旧版token已禁用"})
				c.Abort()
				return
			}
			if !parseLegacyUserToken(c, token) {
				c.Abort()
				return
			}
			c.Next()
			return
		}

		if parseUserJWT(c, token) {
			c.Next()
			return
		}
		if parseMerchantOrStaffJWT(c, token) {
			c.Next()
			return
		}
		if c.IsAborted() {
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
		c.Abort()
	}
}

func parseJWTWithSecret(raw string, secret string) (jwt.MapClaims, bool) {
	if strings.TrimSpace(secret) == "" {
		return nil, false
	}
	parsed, err := jwt.ParseWithClaims(raw, jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, false
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, false
	}
	if !validateTokenExp(claims) {
		return nil, false
	}
	return claims, true
}

func validateTokenExp(claims jwt.MapClaims) bool {
	expAny, ok := claims["exp"]
	if !ok {
		return false
	}
	var exp int64
	switch v := expAny.(type) {
	case float64:
		exp = int64(v)
	case int64:
		exp = v
	case int:
		exp = int64(v)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return false
		}
		exp = parsed
	default:
		return false
	}
	return exp > time.Now().Unix()
}

func parseLegacyUserToken(c *gin.Context, token string) bool {
	parts := strings.Split(token, "_")
	if len(parts) < 3 || parts[0] != "user" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
		return false
	}
	userID, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
		return false
	}
	var user models.User
	if err := config.DB.First(&user, uint(userID)).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return false
	}
	hitCount := legacyUserTokenHitCount.Add(1)
	c.Header("X-Auth-Legacy-Token", "deprecated")
	log.Printf("WARN: legacy user token accepted: user_id=%d path=%s ua=%q hit_count=%d", userID, c.Request.URL.Path, c.Request.UserAgent(), hitCount)
	c.Set("auth_type", "user")
	c.Set("user_id", uint(userID))
	c.Set("user", user)
	return true
}

func parseUserJWT(c *gin.Context, token string) bool {
	claims, ok := parseJWTWithSecret(token, config.UserJWTSecret())
	if !ok {
		return false
	}
	typeVal, _ := claims["type"].(string)
	if typeVal != "user" {
		return false
	}
	uidAny, ok := claims["user_id"]
	if !ok {
		return false
	}
	var userID uint
	switch v := uidAny.(type) {
	case float64:
		userID = uint(v)
	case int64:
		userID = uint(v)
	case int:
		userID = uint(v)
	case string:
		x, err := strconv.ParseUint(strings.TrimSpace(v), 10, 32)
		if err != nil {
			return false
		}
		userID = uint(x)
	default:
		return false
	}
	if userID == 0 {
		return false
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		return false
	}

	c.Set("auth_type", "user")
	c.Set("user_id", userID)
	c.Set("user", user)
	return true
}

func parseMerchantOrStaffJWT(c *gin.Context, token string) bool {
	claims, ok := parseJWTWithSecret(token, config.JWTSecret())
	if !ok {
		return false
	}

	typeVal, _ := claims["type"].(string)
	if typeVal != "merchant" && typeVal != "staff" {
		return false
	}

	merchantIDFloat, ok := claims["merchant_id"].(float64)
	if !ok || merchantIDFloat <= 0 {
		return false
	}
	merchantID := uint(merchantIDFloat)
	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		return false
	}

	c.Set("auth_type", typeVal)
	c.Set("merchant_id", merchantID)
	c.Set("merchant", merchant)

	if typeVal != "staff" {
		return true
	}

	staffIDFloat, ok := claims["staff_id"].(float64)
	if !ok || staffIDFloat <= 0 {
		return false
	}
	staffID := uint(staffIDFloat)

	serviceRoleIDFloat, ok := claims["service_role_id"].(float64)
	if !ok || serviceRoleIDFloat <= 0 {
		return false
	}
	serviceRoleID := uint(serviceRoleIDFloat)

	var staffRole models.ServiceRole
	if err := config.DB.First(&staffRole, serviceRoleID).Error; err != nil {
		return false
	}
	if !config.IsMerchantUsableRole(&staffRole, merchantID) {
		return false
	}

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", staffID, merchantID).First(&tech).Error; err != nil {
		return false
	}
	if tech.ServiceRoleID != staffRole.ID || !tech.IsActive {
		return false
	}

	if tech.PasswordNeedReset {
		if !(c.Request.Method == http.MethodPost && strings.HasSuffix(c.Request.URL.Path, "/technician/password/reset")) {
			c.JSON(http.StatusForbidden, gin.H{"error": "请先修改初始密码"})
			c.Abort()
			return false
		}
	}

	c.Set("staff_id", staffID)
	c.Set("service_role_id", serviceRoleID)
	c.Set("service_role", staffRole)
	c.Set("technician_id", staffID)
	c.Set("technician", tech)

	return true
}
