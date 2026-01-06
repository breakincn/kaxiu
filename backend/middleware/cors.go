package middleware

import (
	"os"
	"strings"

	"github.com/gin-contrib/cors"
)

func CORSMiddleware() cors.Config {
	originsEnv := strings.TrimSpace(os.Getenv("KABAO_CORS_ALLOW_ORIGINS"))
	origins := []string{"https://kabao.app", "https://kabao.shop"}
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
