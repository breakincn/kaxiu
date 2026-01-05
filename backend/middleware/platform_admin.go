package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func PlatformAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := strings.TrimSpace(os.Getenv("PLATFORM_ADMIN_TOKEN"))
		if secret == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "平台管理员未配置"})
			c.Abort()
			return
		}

		got := strings.TrimSpace(c.GetHeader("X-Platform-Admin-Token"))
		if got == "" || got != secret {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
			c.Abort()
			return
		}

		c.Next()
	}
}
