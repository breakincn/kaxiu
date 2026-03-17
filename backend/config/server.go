package config

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func ConfigureTrustedProxies(r *gin.Engine) {
	if r == nil {
		return
	}

	trusted := trustedProxiesFromEnv()
	if len(trusted) == 0 {
		if err := r.SetTrustedProxies(nil); err != nil {
			log.Printf("WARN: 设置 Gin TrustedProxies 失败: %v", err)
		}
		return
	}
	if err := r.SetTrustedProxies(trusted); err != nil {
		log.Printf("WARN: 设置 Gin TrustedProxies 失败: %v", err)
	}
}

func NewTLSServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:     addr,
		Handler:  handler,
		ErrorLog: log.New(tlsHandshakeLogFilter{next: os.Stderr}, "", log.LstdFlags),
	}
}

func trustedProxiesFromEnv() []string {
	raw := EnvString("KABAO_TRUSTED_PROXIES")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type tlsHandshakeLogFilter struct {
	next io.Writer
}

func (w tlsHandshakeLogFilter) Write(p []byte) (int, error) {
	line := string(p)
	if strings.Contains(line, "http: TLS handshake error") && strings.Contains(line, "remote error: tls: unknown certificate") {
		return len(p), nil
	}
	return w.next.Write(p)
}
