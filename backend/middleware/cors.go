package middleware

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
)

func CORSMiddleware() cors.Config {
	originsEnv := strings.TrimSpace(os.Getenv("KABAO_CORS_ALLOW_ORIGINS"))
	fmt.Println("DEBUG KABAO_CORS_ALLOW_ORIGINS =", originsEnv) // 打印环境变量
	var explicitOrigins []string
	if originsEnv != "" {
		parts := strings.Split(originsEnv, ",")
		explicitOrigins = make([]string, 0, len(parts))
		for _, p := range parts {
			v := strings.TrimSpace(p)
			if v != "" {
				explicitOrigins = append(explicitOrigins, v)
			}
		}
	}

	return cors.Config{
		AllowOriginFunc:  buildAllowOriginFunc(explicitOrigins),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Platform-Admin-Token"},
		AllowCredentials: true,
		AllowWildcard:    true,
	}
}

func buildAllowOriginFunc(explicitOrigins []string) func(string) bool {
	allowed := map[string]struct{}{
		"https://kabao.app":     {},
		"https://kabao.shop":    {},
		"http://localhost:3000": {},
		"http://127.0.0.1:3000": {},
		"https://localhost:3000": {},
		"https://127.0.0.1:3000": {},
	}
	for _, origin := range explicitOrigins {
		allowed[origin] = struct{}{}
	}

	return func(origin string) bool {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			return false
		}
		if _, ok := allowed[origin]; ok {
			return true
		}

		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return false
		}

		host := u.Hostname()
		portValue := u.Port()
		if host == "" || portValue == "" {
			return false
		}
		port, err := strconv.Atoi(portValue)
		if err != nil {
			return false
		}
		if port != 3000 && port != 3001 && port != 3002 && port != 5173 {
			return false
		}

		if host == "localhost" || host == "127.0.0.1" {
			return true
		}

		ip := net.ParseIP(host)
		if ip == nil {
			return false
		}

		ip4 := ip.To4()
		if ip4 == nil {
			return false
		}

		if ip4[0] == 10 {
			return true
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		return false
	}
}
