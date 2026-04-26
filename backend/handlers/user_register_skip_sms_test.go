package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserRegisterSkipSMSTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.SMSCode{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestUserRegisterCanSkipSMSVerificationDuringOpenPhase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	oldSecret := os.Getenv("KABAO_USER_JWT_SECRET")
	defer func() { config.DB = oldDB }()
	defer func() { _ = os.Setenv("KABAO_USER_JWT_SECRET", oldSecret) }()

	config.DB = setupUserRegisterSkipSMSTestDB(t)
	_ = os.Setenv("KABAO_USER_JWT_SECRET", "test-user-jwt-secret")

	body, _ := json.Marshal(map[string]any{
		"username": "dev_user_001",
		"password": "123456",
		"phone":    "13800138004",
		"code":     "",
		"nickname": "詹玲珑",
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	UserRegister(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var user models.User
	if err := config.DB.Where("username = ?", "dev_user_001").First(&user).Error; err != nil {
		t.Fatalf("user should be created: %v", err)
	}
	if user.Phone == nil || *user.Phone != "13800138004" {
		t.Fatalf("user phone should be saved, got %+v", user.Phone)
	}
}
