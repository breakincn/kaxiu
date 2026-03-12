package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthMiddlewareTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:middleware_auth_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if err := db.Create(&models.User{Username: "u1", Password: "x"}).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return db
}

func TestAuthMiddlewareRejectsLegacyUserTokenByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAuthMiddlewareTestDB(t)

	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/user/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/user/me", nil)
	req.Header.Set("Authorization", "Bearer user_1_123456")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body == "" || !strings.Contains(body, "旧版token已禁用") {
		t.Fatalf("want legacy token disabled error, got %s", body)
	}
}

func TestAuthMiddlewareAllowsLegacyUserTokenWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("KABAO_ALLOW_LEGACY_USER_TOKEN", "true")

	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAuthMiddlewareTestDB(t)

	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/user/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/user/me", nil)
	req.Header.Set("Authorization", "Bearer user_1_123456")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Auth-Legacy-Token"); got != "deprecated" {
		t.Fatalf("want deprecated header, got %q", got)
	}
}
