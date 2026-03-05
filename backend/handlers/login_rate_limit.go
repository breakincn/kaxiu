package handlers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type loginLimitState struct {
	failCount    int
	windowStart  time.Time
	blockedUntil time.Time
}

type memoryLoginLimiter struct {
	mu      sync.Mutex
	storage map[string]loginLimitState
}

var defaultLoginLimiter = &memoryLoginLimiter{
	storage: map[string]loginLimitState{},
}

const (
	loginWindow      = 10 * time.Minute
	loginMaxFailures = 5
	loginBlockTTL    = 15 * time.Minute
)

func loginLimitKey(c *gin.Context, realm, account string) string {
	ip := strings.TrimSpace(c.ClientIP())
	realm = strings.TrimSpace(strings.ToLower(realm))
	account = strings.TrimSpace(strings.ToLower(account))
	return realm + "|" + ip + "|" + account
}

func (l *memoryLoginLimiter) check(key string, now time.Time) (bool, int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	st, ok := l.storage[key]
	if !ok {
		return false, 0
	}
	if st.blockedUntil.After(now) {
		return true, int64(st.blockedUntil.Sub(now).Seconds())
	}
	if st.windowStart.IsZero() || now.Sub(st.windowStart) > loginWindow {
		delete(l.storage, key)
	}
	return false, 0
}

func (l *memoryLoginLimiter) onFailure(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	st := l.storage[key]
	if st.windowStart.IsZero() || now.Sub(st.windowStart) > loginWindow {
		st.windowStart = now
		st.failCount = 0
		st.blockedUntil = time.Time{}
	}
	st.failCount++
	if st.failCount >= loginMaxFailures {
		st.blockedUntil = now.Add(loginBlockTTL)
		st.failCount = 0
		st.windowStart = now
	}
	l.storage[key] = st
}

func (l *memoryLoginLimiter) onSuccess(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.storage, key)
}

func enforceLoginRateLimit(c *gin.Context, realm, account string) (string, bool) {
	key := loginLimitKey(c, realm, account)
	blocked, retrySec := defaultLoginLimiter.check(key, time.Now())
	if blocked {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "登录尝试过于频繁，请稍后再试",
			"retry_after": retrySec,
		})
		return key, false
	}
	return key, true
}

func recordLoginFailure(key string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	defaultLoginLimiter.onFailure(key, time.Now())
}

func recordLoginSuccess(key string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	defaultLoginLimiter.onSuccess(key)
}
