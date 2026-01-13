package middleware

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
)

func CORSMiddleware() cors.Config {
	originsEnv := strings.TrimSpace(os.Getenv("KABAO_CORS_ALLOW_ORIGINS"))
	origins := []string{
		"https://kabao.app",
		"https://kabao.shop",
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://10.0.0.20:3000",
		"http://10.0.0.20:5173", // Vite 默认端口
		// 支持局域网HTTP访问
		"http://10.0.0.*:3000",
		"http://10.0.0.*:5173",
		"http://192.168.*:3000",
		"http://192.168.*:5173",
		// 支持HTTPS前端访问HTTP后端（混合内容）
		"https://10.0.0.20:3000",
		"https://10.0.0.20:3001",
		"https://10.0.0.20:3002",
		"https://10.0.0.*:3000",
		"https://10.0.0.*:3001",
		"https://10.0.0.*:3002",
		"https://10.0.0.*:5173",
		"https://192.168.*:3000",
		"https://192.168.*:3001",
		"https://192.168.*:3002",
		"https://192.168.*:5173",
	}
	fmt.Println("DEBUG KABAO_CORS_ALLOW_ORIGINS =", originsEnv) // 打印环境变量
	if originsEnv != "" {
		parts := strings.Split(originsEnv, ",")
		list := make([]string, 0, len(parts))
		for _, p := range parts {
			v := strings.TrimSpace(p)
			if v != "" {
				list = append(list, v)
			}
		}
		if len(list) > 0 {
			origins = list
		}
	}

	return cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Platform-Admin-Token"},
		AllowCredentials: true,
	}
}
