package handlers

import (
	"encoding/json"
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

func setupPromotionCardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
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

func TestListMyPromotionRewardCardsReturnsActiveProgressItem(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupPromotionCardTestDB(t)
	config.DB = db

	user := models.User{Username: "promo-user", Password: "pwd", Nickname: "推广用户"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	merchant := models.Merchant{Name: "测试门店", Phone: "13800000022", Password: "x", Type: "理发", SupportDirectSale: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	template := models.CardTemplate{
		MerchantID: merchant.ID,
		Name:       "成人爵士60课时",
		CardType:   "lesson",
		Price:      280000,
		TotalTimes: 60,
		IsActive:   true,
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatalf("create template failed: %v", err)
	}

	activeCampaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "楚韵江南新店开张大促",
		RewardTotalTimes:   3,
		RewardCardQuantity: 10,
		RewardThreshold:    15,
		Slug:               "promo-active-test",
		Status:             "active",
	}
	if err := db.Create(&activeCampaign).Error; err != nil {
		t.Fatalf("create active campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:    activeCampaign.ID,
		UserID:        user.ID,
		PromotionCode: "123456",
		RegisterCount: 4,
		PaidCount:     3,
		ProgressCount: 7,
	}).Error; err != nil {
		t.Fatalf("create active referrer failed: %v", err)
	}

	claimedCampaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "已领取奖励活动",
		RewardTotalTimes:   1,
		RewardCardQuantity: 10,
		RewardThreshold:    2,
		Slug:               "promo-claimed-test",
		Status:             "active",
	}
	if err := db.Create(&claimedCampaign).Error; err != nil {
		t.Fatalf("create claimed campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:    claimedCampaign.ID,
		UserID:        user.ID,
		PromotionCode: "654321",
		ProgressCount: 2,
		RewardStatus:  promotionCardRewardStatusClaimed,
	}).Error; err != nil {
		t.Fatalf("create claimed referrer failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/promotion-reward-cards?status=active", nil)
	c.Set("user_id", user.ID)

	ListMyPromotionRewardCards(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []promotionRewardListItem `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("want 1 active promotion item, got %d body=%s", len(resp.Data), rec.Body.String())
	}
	item := resp.Data[0]
	if item.CampaignID != activeCampaign.ID {
		t.Fatalf("want campaign_id=%d, got %d", activeCampaign.ID, item.CampaignID)
	}
	if item.ProgressCount != 7 || item.RewardThreshold != 15 {
		t.Fatalf("unexpected progress payload: %+v", item)
	}
	if item.MerchantName != merchant.Name || item.CardName != template.Name {
		t.Fatalf("unexpected card info payload: %+v", item)
	}
	if item.ListItemType != "promotion_reward" {
		t.Fatalf("want list_item_type=promotion_reward, got %s", item.ListItemType)
	}
}
