package handlers

import (
	"bytes"
	"encoding/json"
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

func setupHandCardVerifyAgeGuardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:handlers_hand_card_verify_age_guard_test_%d?mode=memory&cache=shared&_loc=auto", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Card{}, &models.VerifyCode{}, &models.Usage{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func seedAgeGuardFixture(t *testing.T, db *gorm.DB, supportHandCard bool) (models.Merchant, models.Card, models.VerifyCode) {
	t.Helper()
	merchant := models.Merchant{
		Name:            "m",
		Phone:           fmt.Sprintf("188%08d", time.Now().UnixNano()%100000000),
		Password:        "pwd",
		SupportHandCard: supportHandCard,
	}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	card := models.Card{
		MerchantID:  merchant.ID,
		CardNo:      "00001",
		CardType:    "次卡",
		TotalTimes:  10,
		RemainTimes: 10,
		UsedTimes:   0,
	}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	verifyCode := models.VerifyCode{
		CardID:   card.ID,
		Code:     fmt.Sprintf("VERIFY-%d", time.Now().UnixNano()),
		ExpireAt: time.Now().Add(10 * time.Minute).Unix(),
		Used:     false,
	}
	if err := db.Create(&verifyCode).Error; err != nil {
		t.Fatalf("create verify code failed: %v", err)
	}

	return merchant, card, verifyCode
}

func createUnreturnedHandCardUsage(t *testing.T, db *gorm.DB, merchantID, cardID uint, handCardNo string, assignedAt time.Time) {
	t.Helper()
	no := handCardNo
	usedAt := assignedAt
	u := models.Usage{
		CardID:             cardID,
		MerchantID:         merchantID,
		UsedTimes:          1,
		UsedAt:             &usedAt,
		VerifyCode:         "OLD-CODE",
		VerifyCodeExpireAt: time.Now().Add(10 * time.Minute).Unix(),
		Status:             "success",
		HandCardNo:         &no,
		HandCardAssignedAt: &assignedAt,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create unreturned usage failed: %v", err)
	}
}

func assertNoVerifySideEffects(t *testing.T, db *gorm.DB, verifyCodeID, cardID uint, usageCountBefore int64) {
	t.Helper()

	var vc models.VerifyCode
	if err := db.First(&vc, verifyCodeID).Error; err != nil {
		t.Fatalf("load verify code failed: %v", err)
	}
	if vc.Used {
		t.Fatalf("verify code should remain unused")
	}

	var card models.Card
	if err := db.First(&card, cardID).Error; err != nil {
		t.Fatalf("load card failed: %v", err)
	}
	if card.RemainTimes != 10 || card.UsedTimes != 0 {
		t.Fatalf("card times should remain unchanged, remain=%d used=%d", card.RemainTimes, card.UsedTimes)
	}

	var usageCountAfter int64
	if err := db.Model(&models.Usage{}).Count(&usageCountAfter).Error; err != nil {
		t.Fatalf("count usages failed: %v", err)
	}
	if usageCountAfter != usageCountBefore {
		t.Fatalf("usage count should stay %d, got %d", usageCountBefore, usageCountAfter)
	}
}

func TestVerifyCardRejectsWhenOldUnreturnedHandCardExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupHandCardVerifyAgeGuardTestDB(t)

	merchant, card, verifyCode := seedAgeGuardFixture(t, config.DB, true)
	now := time.Now()
	createUnreturnedHandCardUsage(t, config.DB, merchant.ID, card.ID, "021", now.Add(-(8*time.Minute + 6*time.Second)))
	createUnreturnedHandCardUsage(t, config.DB, merchant.ID, card.ID, "022", now.Add(-(8*time.Minute + 2*time.Second)))

	var usageCountBefore int64
	if err := config.DB.Model(&models.Usage{}).Count(&usageCountBefore).Error; err != nil {
		t.Fatalf("count usages failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"code": verifyCode.Code})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/verify", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", merchant.ID)

	VerifyCard(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "该商户已启用手牌，请使用新的扫码核销流程") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	assertNoVerifySideEffects(t, config.DB, verifyCode.ID, card.ID, usageCountBefore)
}

func TestScanVerifyCardRejectsWhenOldUnreturnedHandCardExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupHandCardVerifyAgeGuardTestDB(t)

	merchant, card, verifyCode := seedAgeGuardFixture(t, config.DB, true)
	createUnreturnedHandCardUsage(t, config.DB, merchant.ID, card.ID, "021", time.Now().Add(-(8*time.Minute + 7*time.Second)))

	var usageCountBefore int64
	if err := config.DB.Model(&models.Usage{}).Count(&usageCountBefore).Error; err != nil {
		t.Fatalf("count usages failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"code": verifyCode.Code})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/verify/scan", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", merchant.ID)
	c.Set("auth_type", "merchant")

	ScanVerifyCard(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "该商户已启用手牌，请升级到新的扫码核销流程") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	assertNoVerifySideEffects(t, config.DB, verifyCode.ID, card.ID, usageCountBefore)
}

func TestVerifyHandCardUnreturnedAgeGuardNotInterceptRecentUsage(t *testing.T) {
	db := setupHandCardVerifyAgeGuardTestDB(t)
	merchant, card, _ := seedAgeGuardFixture(t, db, true)
	now := time.Now()
	createUnreturnedHandCardUsage(t, db, merchant.ID, card.ID, "021", now.Add(-(8*time.Minute + 4*time.Second)))

	if err := verifyHandCardUnreturnedAgeGuard(db, merchant.ID, true, now); err != nil {
		t.Fatalf("recent unreturned hand card should not be intercepted, got %v", err)
	}
}

func TestVerifyHandCardUnreturnedAgeGuardBypassWhenSupportDisabled(t *testing.T) {
	db := setupHandCardVerifyAgeGuardTestDB(t)
	merchant, card, _ := seedAgeGuardFixture(t, db, false)
	now := time.Now()
	createUnreturnedHandCardUsage(t, db, merchant.ID, card.ID, "021", now.Add(-(8*time.Minute + 8*time.Second)))

	if err := verifyHandCardUnreturnedAgeGuard(db, merchant.ID, false, now); err != nil {
		t.Fatalf("guard should bypass when support_hand_card disabled, got %v", err)
	}
}

func TestVerifyHandCardUnreturnedAgeGuardInterceptsAcrossCardsWithinMerchant(t *testing.T) {
	db := setupHandCardVerifyAgeGuardTestDB(t)
	merchant, cardA, _ := seedAgeGuardFixture(t, db, true)
	now := time.Now()
	createUnreturnedHandCardUsage(t, db, merchant.ID, cardA.ID, "021", now.Add(-(8*time.Minute + 8*time.Second)))

	cardB := models.Card{
		MerchantID:  merchant.ID,
		CardNo:      "00002",
		CardType:    "次卡",
		TotalTimes:  10,
		RemainTimes: 10,
		UsedTimes:   0,
	}
	if err := db.Create(&cardB).Error; err != nil {
		t.Fatalf("create second card failed: %v", err)
	}

	err := verifyHandCardUnreturnedAgeGuard(db, merchant.ID, true, now)
	if err == nil {
		t.Fatalf("want intercept error for merchant-level guard")
	}
	if !strings.Contains(err.Error(), "你尚有未归还的手牌021，请先归还手牌再核销") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyHandCardUnreturnedAgeGuardInterceptsLegacyUsageWithoutAssignedAt(t *testing.T) {
	db := setupHandCardVerifyAgeGuardTestDB(t)
	merchant, card, _ := seedAgeGuardFixture(t, db, true)
	now := time.Now()

	no := "021"
	oldUsedAt := now.Add(-(8*time.Minute + 9*time.Second))
	u := models.Usage{
		CardID:             card.ID,
		MerchantID:         merchant.ID,
		UsedTimes:          1,
		UsedAt:             &oldUsedAt,
		VerifyCode:         "LEGACY-CODE",
		VerifyCodeExpireAt: now.Add(10 * time.Minute).Unix(),
		Status:             "success",
		HandCardNo:         &no,
		HandCardAssignedAt: nil,
		HandCardReturnedAt: nil,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create legacy usage failed: %v", err)
	}

	err := verifyHandCardUnreturnedAgeGuard(db, merchant.ID, true, now)
	if err == nil {
		t.Fatalf("want intercept error for legacy usage without assigned_at")
	}
	if !strings.Contains(err.Error(), "你尚有未归还的手牌021，请先归还手牌再核销") {
		t.Fatalf("unexpected error: %v", err)
	}
}
