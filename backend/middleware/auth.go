package middleware

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware 验证用户登录状态
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		// 移除 "Bearer " 前缀
		token = strings.TrimPrefix(token, "Bearer ")

		// 兼容旧版用户 token: user_{userID}_{timestamp}
		if strings.HasPrefix(token, "user_") {
			parts := strings.Split(token, "_")
			if len(parts) < 3 || parts[0] != "user" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
				c.Abort()
				return
			}

			userID, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
				c.Abort()
				return
			}

			var user models.User
			if err := config.DB.First(&user, userID).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
				c.Abort()
				return
			}

			c.Set("user_id", uint(userID))
			c.Set("user", user)
			c.Next()
			return
		}

		// 商户端使用 JWT
		parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte("your-secret-key"), nil
		})
		if err != nil || parsedToken == nil || !parsedToken.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}

		typeVal, _ := claims["type"].(string)
		if typeVal != "merchant" && typeVal != "staff" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}

		merchantIDFloat, ok := claims["merchant_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}

		merchantID := uint(merchantIDFloat)
		var merchant models.Merchant
		if err := config.DB.First(&merchant, merchantID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "商户不存在"})
			c.Abort()
			return
		}

		c.Set("auth_type", typeVal)
		c.Set("merchant_id", merchantID)
		c.Set("merchant", merchant)

		if typeVal == "staff" {
			staffIDFloat, ok := claims["staff_id"].(float64)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
				c.Abort()
				return
			}
			staffID := uint(staffIDFloat)
			serviceRoleIDFloat, ok := claims["service_role_id"].(float64)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
				c.Abort()
				return
			}
			serviceRoleID := uint(serviceRoleIDFloat)

			// 根据角色类型加载对应的员工信息
			var staffRole models.ServiceRole
			if err := config.DB.First(&staffRole, serviceRoleID).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "员工角色不存在"})
				c.Abort()
				return
			}

			c.Set("staff_id", staffID)
			c.Set("service_role_id", serviceRoleID)
			c.Set("service_role", staffRole)

			// 如果是技师，还需要设置技师信息（兼容现有逻辑）
			if staffRole.Key == "technician" {
				var tech models.Technician
				if err := config.DB.Where("id = ? AND merchant_id = ?", staffID, merchantID).First(&tech).Error; err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "技师不存在"})
					c.Abort()
					return
				}
				c.Set("technician_id", staffID)
				c.Set("technician", tech)
			}
		}

		c.Next()
	}
}
