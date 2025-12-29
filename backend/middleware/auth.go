package middleware

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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
		if strings.HasPrefix(token, "Bearer ") {
			token = token[7:]
		}
		
		// 从 token 中解析 user_id（简化版本）
		// 格式: user_{userID}_{timestamp}
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
		
		// 验证用户是否存在
		var user models.User
		if err := config.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}
		
		// 将用户ID存入上下文
		c.Set("user_id", uint(userID))
		c.Set("user", user)
		c.Next()
	}
}
