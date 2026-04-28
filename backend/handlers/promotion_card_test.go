package handlers

import (
	"strings"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPromotionCardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Merchant{},
		&models.CardTemplate{},
		&models.PromotionCardCampaign{},
		&models.PromotionCardClaim{},
		&models.PromotionCardReferrer{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestMaybeQualifyPromotionRewardMarksClaimableWhenQuotaAvailable(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupPromotionCardTestDB(t)
	config.DB = db

	merchant := models.Merchant{Name: "测试商户", Phone: "13800000011", Password: "x", Type: "理发", SupportDirectSale: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	template := models.CardTemplate{
		MerchantID: merchant.ID,
		Name:       "测试卡",
		CardType:   "times",
		Price:      10000,
		TotalTimes: 10,
		IsActive:   true,
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatalf("create template failed: %v", err)
	}
	campaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		RewardTotalTimes:   2,
		RewardCardQuantity: 3,
		RewardThreshold:    2,
		Slug:               "promo-unit-test",
		Status:             "active",
	}
	if err := db.Create(&campaign).Error; err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:    campaign.ID,
		UserID:        100,
		PromotionCode: "111111",
		RewardStatus:  promotionCardRewardStatusClaimable,
	}).Error; err != nil {
		t.Fatalf("seed reserved reward failed: %v", err)
	}

	referrer := models.PromotionCardReferrer{
		CampaignID:    campaign.ID,
		UserID:        200,
		PromotionCode: "222222",
		ProgressCount: 2,
	}

	now := time.Now()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return maybeQualifyPromotionReward(tx, &referrer, &campaign, now)
	}); err != nil {
		t.Fatalf("maybeQualifyPromotionReward failed: %v", err)
	}

	if referrer.RewardStatus != promotionCardRewardStatusClaimable {
		t.Fatalf("want reward_status=%s, got %s", promotionCardRewardStatusClaimable, referrer.RewardStatus)
	}
	if referrer.RewardQualifiedAt == nil {
		t.Fatalf("want reward_qualified_at to be set")
	}
	if referrer.RewardExpiresAt == nil || !referrer.RewardExpiresAt.After(now) {
		t.Fatalf("want reward_expires_at after now, got %+v", referrer.RewardExpiresAt)
	}
}
