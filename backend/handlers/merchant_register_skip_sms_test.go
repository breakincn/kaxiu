package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMerchantRegisterSkipSMSTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.InviteCode{}, &models.SMSCode{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if err := db.Create(&models.InviteCode{Code: "KABAO-TEST-001", Used: false}).Error; err != nil {
		t.Fatalf("seed invite code failed: %v", err)
	}
	return db
}

func TestMerchantRegisterCanSkipSMSVerificationDuringOpenPhase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupMerchantRegisterSkipSMSTestDB(t)

	body, _ := json.Marshal(map[string]any{
		"phone":       "18800009999",
		"password":    "123456",
		"name":        "开放期商户",
		"type":        "理发",
		"code":        "",
		"invite_code": "KABAO-TEST-001",
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/register", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	MerchantRegister(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var merchant models.Merchant
	if err := config.DB.Where("phone = ?", "18800009999").First(&merchant).Error; err != nil {
		t.Fatalf("merchant should be created: %v", err)
	}

	var invite struct {
		Used             bool  `gorm:"column:used"`
		UsedByMerchantID *uint `gorm:"column:used_by_merchant_id"`
	}
	if err := config.DB.Table("invite_codes").Select("used, used_by_merchant_id").Where("code = ?", "KABAO-TEST-001").Scan(&invite).Error; err != nil {
		t.Fatalf("invite code should exist: %v", err)
	}
	if !invite.Used {
		t.Fatalf("invite code should be marked used")
	}
	if invite.UsedByMerchantID == nil || *invite.UsedByMerchantID == 0 {
		t.Fatalf("invite code should reference created merchant")
	}
}
