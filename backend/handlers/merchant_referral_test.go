package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMerchantReferralTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Merchant{},
		&models.InviteCode{},
		&models.MerchantReferralProfile{},
		&models.MerchantReferral{},
		&models.MerchantReferralPaymentLedger{},
		&models.MerchantReferralWithdrawal{},
		&models.MerchantReferralWithdrawalItem{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestMerchantRegisterBindsReferralCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantReferralTestDB(t)

	referrer := models.User{Username: "promoter", Nickname: "推广达人"}
	if err := config.DB.Create(&referrer).Error; err != nil {
		t.Fatalf("seed user failed: %v", err)
	}
	profile := models.MerchantReferralProfile{
		UserID:                  referrer.ID,
		PromotionCode:           "PROMO888",
		DefaultCommissionRateBP: merchantReferralDefaultCommissionRateBP,
	}
	if err := config.DB.Create(&profile).Error; err != nil {
		t.Fatalf("seed profile failed: %v", err)
	}
	if err := config.DB.Create(&models.InviteCode{Code: "KABAO-TEST-001", Used: false}).Error; err != nil {
		t.Fatalf("seed invite code failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"phone":         "18800006666",
		"password":      "123456",
		"name":          "被推广商户",
		"type":          "美容",
		"invite_code":   "KABAO-TEST-001",
		"referral_code": "PROMO888",
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/register", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	MerchantRegister(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var merchant models.Merchant
	if err := config.DB.Where("phone = ?", "18800006666").First(&merchant).Error; err != nil {
		t.Fatalf("merchant should exist: %v", err)
	}
	var referral models.MerchantReferral
	if err := config.DB.Where("merchant_id = ?", merchant.ID).First(&referral).Error; err != nil {
		t.Fatalf("referral should be created: %v", err)
	}
	if referral.ReferrerUserID != referrer.ID {
		t.Fatalf("want referrer_user_id=%d, got %d", referrer.ID, referral.ReferrerUserID)
	}
	if referral.PromotionCode != "PROMO888" {
		t.Fatalf("want promotion code PROMO888, got %s", referral.PromotionCode)
	}
}

func TestMerchantReferralPaymentLedgerAndWithdrawalFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantReferralTestDB(t)

	referrer := models.User{Username: "promoter2", Nickname: "推广二号"}
	if err := config.DB.Create(&referrer).Error; err != nil {
		t.Fatalf("seed user failed: %v", err)
	}
	merchant := models.Merchant{Name: "分成商户", Phone: "18800007777", Password: "x"}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("seed merchant failed: %v", err)
	}
	referral := models.MerchantReferral{
		ReferrerUserID:          referrer.ID,
		MerchantID:              merchant.ID,
		PromotionCode:           "PROMO999",
		RegisteredAt:            ptrTime(time.Now().Add(-48 * time.Hour)),
		DefaultCommissionRateBP: merchantReferralDefaultCommissionRateBP,
	}
	if err := config.DB.Create(&referral).Error; err != nil {
		t.Fatalf("seed referral failed: %v", err)
	}

	paidAt := time.Now().Add(-24 * time.Hour).UTC()
	ledgerBody, _ := json.Marshal(map[string]any{
		"merchant_id": merchant.ID,
		"paid_amount": 10000,
		"paid_at":     paidAt.Format(time.RFC3339),
		"note":        "首笔商户付费",
	})
	ledgerRec := httptest.NewRecorder()
	ledgerCtx, _ := gin.CreateTestContext(ledgerRec)
	ledgerCtx.Request = httptest.NewRequest(http.MethodPost, "/platform/merchant-referral-payments", bytes.NewReader(ledgerBody))
	ledgerCtx.Request.Header.Set("Content-Type", "application/json")

	CreateMerchantReferralPaymentLedger(ledgerCtx)

	if ledgerRec.Code != http.StatusOK {
		t.Fatalf("create ledger want 200, got %d body=%s", ledgerRec.Code, ledgerRec.Body.String())
	}

	var ledger models.MerchantReferralPaymentLedger
	if err := config.DB.Where("merchant_id = ?", merchant.ID).First(&ledger).Error; err != nil {
		t.Fatalf("ledger should exist: %v", err)
	}
	if ledger.CommissionAmount != 5000 {
		t.Fatalf("want commission 5000, got %d", ledger.CommissionAmount)
	}
	if ledger.WithdrawalStatus != withdrawalStatusClaimable {
		t.Fatalf("want ledger claimable, got %s", ledger.WithdrawalStatus)
	}

	withdrawBody, _ := json.Marshal(map[string]any{
		"payment_ledger_ids": []uint{ledger.ID},
		"payee_name":         "张三",
		"payee_account":      "wx_zhangsan",
		"payee_channel":      "wechat",
	})
	withdrawRec := httptest.NewRecorder()
	withdrawCtx, _ := gin.CreateTestContext(withdrawRec)
	withdrawCtx.Set("user_id", referrer.ID)
	withdrawCtx.Request = httptest.NewRequest(http.MethodPost, "/user/referral-commission/withdrawals", bytes.NewReader(withdrawBody))
	withdrawCtx.Request.Header.Set("Content-Type", "application/json")

	CreateReferralCommissionWithdrawal(withdrawCtx)

	if withdrawRec.Code != http.StatusOK {
		t.Fatalf("create withdrawal want 200, got %d body=%s", withdrawRec.Code, withdrawRec.Body.String())
	}

	var withdrawal models.MerchantReferralWithdrawal
	if err := config.DB.Where("referrer_user_id = ?", referrer.ID).First(&withdrawal).Error; err != nil {
		t.Fatalf("withdrawal should exist: %v", err)
	}
	if withdrawal.AppliedAmount != 5000 {
		t.Fatalf("want applied amount 5000, got %d", withdrawal.AppliedAmount)
	}

	reviewBody, _ := json.Marshal(map[string]any{
		"status":      "paid",
		"reviewed_by": "platform-admin",
		"review_note": "已人工打款",
	})
	reviewRec := httptest.NewRecorder()
	reviewCtx, _ := gin.CreateTestContext(reviewRec)
	reviewCtx.Params = gin.Params{{Key: "id", Value: strconvFormatUint(withdrawal.ID)}}
	reviewCtx.Request = httptest.NewRequest(http.MethodPost, "/admin/merchant-referral-withdrawals/"+strconvFormatUint(withdrawal.ID)+"/review", bytes.NewReader(reviewBody))
	reviewCtx.Request.Header.Set("Content-Type", "application/json")

	ReviewMerchantReferralWithdrawal(reviewCtx)

	if reviewRec.Code != http.StatusOK {
		t.Fatalf("review withdrawal want 200, got %d body=%s", reviewRec.Code, reviewRec.Body.String())
	}

	if err := config.DB.First(&ledger, ledger.ID).Error; err != nil {
		t.Fatalf("reload ledger failed: %v", err)
	}
	if ledger.WithdrawalStatus != withdrawalStatusPaid {
		t.Fatalf("want ledger paid, got %s", ledger.WithdrawalStatus)
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

func strconvFormatUint(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
