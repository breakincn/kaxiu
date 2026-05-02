package handlers

import (
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
		&models.PromotionCardReferral{},
		&models.DirectPurchase{},
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

func TestListMyPromotionRewardCardsHidesInactiveCampaignAndKeepsRecentlyExpiredRewardFor24Hours(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupPromotionCardTestDB(t)
	config.DB = db

	user := models.User{Username: "promo-user-2", Password: "pwd", Nickname: "推广用户2"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	merchant := models.Merchant{Name: "测试门店2", Phone: "13800000023", Password: "x", Type: "理发", SupportDirectSale: true}
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

	inactiveCampaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "已结束活动",
		RewardTotalTimes:   3,
		RewardCardQuantity: 10,
		RewardThreshold:    15,
		Slug:               "promo-inactive-hide",
		Status:             "disabled",
	}
	if err := db.Create(&inactiveCampaign).Error; err != nil {
		t.Fatalf("create inactive campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:    inactiveCampaign.ID,
		UserID:        user.ID,
		PromotionCode: "111111",
		ProgressCount: 0,
	}).Error; err != nil {
		t.Fatalf("create inactive referrer failed: %v", err)
	}

	expiredPromoAt := time.Now().Add(-2 * time.Hour)
	expiredPromoCampaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "促销已过期活动",
		RewardTotalTimes:   3,
		RewardCardQuantity: 10,
		RewardThreshold:    15,
		PromoPrice:         100,
		PromoQuantity:      10,
		PromoEndsAt:        &expiredPromoAt,
		Slug:               "promo-ended-hide",
		Status:             "active",
	}
	if err := db.Create(&expiredPromoCampaign).Error; err != nil {
		t.Fatalf("create expired promo campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:    expiredPromoCampaign.ID,
		UserID:        user.ID,
		PromotionCode: "121212",
		ProgressCount: 0,
	}).Error; err != nil {
		t.Fatalf("create expired promo referrer failed: %v", err)
	}

	recentExpiredAt := time.Now().Add(-23 * time.Hour)
	recentExpiredCampaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "近期失效奖励",
		RewardTotalTimes:   3,
		RewardCardQuantity: 10,
		RewardThreshold:    15,
		Slug:               "promo-recent-expired",
		Status:             "active",
	}
	if err := db.Create(&recentExpiredCampaign).Error; err != nil {
		t.Fatalf("create recent expired campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:      recentExpiredCampaign.ID,
		UserID:          user.ID,
		PromotionCode:   "222222",
		ProgressCount:   15,
		RewardStatus:    promotionCardRewardStatusExpired,
		RewardExpiredAt: &recentExpiredAt,
	}).Error; err != nil {
		t.Fatalf("create recent expired referrer failed: %v", err)
	}

	recentExpiredButPromoEndedAt := time.Now().Add(-30 * time.Minute)
	recentExpiredButPromoEndedCampaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "奖励近期失效但活动已结束",
		RewardTotalTimes:   3,
		RewardCardQuantity: 10,
		RewardThreshold:    15,
		PromoPrice:         100,
		PromoQuantity:      10,
		PromoEndsAt:        &recentExpiredButPromoEndedAt,
		Slug:               "promo-ended-recent-expired-hide",
		Status:             "active",
	}
	if err := db.Create(&recentExpiredButPromoEndedCampaign).Error; err != nil {
		t.Fatalf("create recent expired but promo ended campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:      recentExpiredButPromoEndedCampaign.ID,
		UserID:          user.ID,
		PromotionCode:   "232323",
		ProgressCount:   15,
		RewardStatus:    promotionCardRewardStatusExpired,
		RewardExpiredAt: &recentExpiredAt,
	}).Error; err != nil {
		t.Fatalf("create recent expired but promo ended referrer failed: %v", err)
	}

	staleExpiredAt := time.Now().Add(-25 * time.Hour)
	staleExpiredCampaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "过期超24小时奖励",
		RewardTotalTimes:   3,
		RewardCardQuantity: 10,
		RewardThreshold:    15,
		Slug:               "promo-stale-expired",
		Status:             "active",
	}
	if err := db.Create(&staleExpiredCampaign).Error; err != nil {
		t.Fatalf("create stale expired campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:      staleExpiredCampaign.ID,
		UserID:          user.ID,
		PromotionCode:   "333333",
		ProgressCount:   15,
		RewardStatus:    promotionCardRewardStatusExpired,
		RewardExpiredAt: &staleExpiredAt,
	}).Error; err != nil {
		t.Fatalf("create stale expired referrer failed: %v", err)
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
		t.Fatalf("want only recent expired reward item kept, got %d body=%s", len(resp.Data), rec.Body.String())
	}
	if resp.Data[0].CampaignID != recentExpiredCampaign.ID {
		t.Fatalf("want campaign_id=%d kept, got %+v", recentExpiredCampaign.ID, resp.Data[0])
	}
	if resp.Data[0].RewardStatus != promotionCardRewardStatusExpired {
		t.Fatalf("want reward_status expired, got %+v", resp.Data[0])
	}
}

func TestListMerchantPromotionCampaignsReturnsProgressCount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupPromotionCardTestDB(t)
	config.DB = db

	merchant := models.Merchant{Name: "测试商户", Phone: "13800000029", Password: "x", Type: "理发", SupportDirectSale: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	template := models.CardTemplate{
		MerchantID: merchant.ID,
		Name:       "测试课时卡",
		CardType:   "lesson",
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
		RewardTotalTimes:   5,
		RewardCardQuantity: 20,
		RewardThreshold:    10,
		Slug:               "merchant-progress-campaign",
		Status:             "active",
	}
	if err := db.Create(&campaign).Error; err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:    campaign.ID,
		UserID:        101,
		PromotionCode: "111111",
		ProgressCount: 1,
	}).Error; err != nil {
		t.Fatalf("create first referrer failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:    campaign.ID,
		UserID:        102,
		PromotionCode: "222222",
		ProgressCount: 2,
	}).Error; err != nil {
		t.Fatalf("create second referrer failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/promotion-campaigns?template_id=1", nil)
	c.Set("merchant_id", merchant.ID)

	ListMerchantPromotionCampaigns(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []merchantPromotionCampaignListItem `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("want 1 campaign item, got %d body=%s", len(resp.Data), rec.Body.String())
	}
	if resp.Data[0].ProgressCount != 3 {
		t.Fatalf("want progress_count=3, got %+v", resp.Data[0])
	}
}

func TestDirectPurchaseStorePendingStatusFitsColumnAndSupportsLegacyValue(t *testing.T) {
	if len(directPurchaseStatusStoreWait) > 20 {
		t.Fatalf("store pending status too long for direct_purchases.status: %s (%d)", directPurchaseStatusStoreWait, len(directPurchaseStatusStoreWait))
	}
	if directPurchaseStatusStoreWait != "store_pending_wait" {
		t.Fatalf("unexpected store pending status constant: %s", directPurchaseStatusStoreWait)
	}
}

func TestCanDeletePromotionCampaign(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupPromotionCardTestDB(t)
	config.DB = db

	merchant := models.Merchant{Name: "测试商户", Phone: "13800000033", Password: "x", Type: "理发", SupportDirectSale: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	template := models.CardTemplate{MerchantID: merchant.ID, Name: "测试卡", CardType: "times", Price: 10000, TotalTimes: 10, IsActive: true}
	if err := db.Create(&template).Error; err != nil {
		t.Fatalf("create template failed: %v", err)
	}

	campaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "可删除活动",
		RewardTotalTimes:   1,
		RewardCardQuantity: 5,
		RewardThreshold:    2,
		Slug:               "promo-delete-ok",
		Status:             "active",
	}
	if err := db.Create(&campaign).Error; err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}

	canDelete, err := canDeletePromotionCampaign(db, campaign.ID)
	if err != nil {
		t.Fatalf("canDeletePromotionCampaign failed: %v", err)
	}
	if !canDelete {
		t.Fatalf("want campaign deletable")
	}

	progressCampaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "有进度活动",
		RewardTotalTimes:   1,
		RewardCardQuantity: 5,
		RewardThreshold:    2,
		Slug:               "promo-delete-progress",
		Status:             "active",
	}
	if err := db.Create(&progressCampaign).Error; err != nil {
		t.Fatalf("create progress campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardReferrer{
		CampaignID:    progressCampaign.ID,
		UserID:        101,
		PromotionCode: "111111",
		RegisterCount: 1,
		ProgressCount: 1,
	}).Error; err != nil {
		t.Fatalf("create referrer failed: %v", err)
	}

	canDelete, err = canDeletePromotionCampaign(db, progressCampaign.ID)
	if err != nil {
		t.Fatalf("canDeletePromotionCampaign(progress) failed: %v", err)
	}
	if canDelete {
		t.Fatalf("want campaign with referral progress not deletable")
	}

	claimCampaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "有领取活动",
		RewardTotalTimes:   1,
		RewardCardQuantity: 5,
		RewardThreshold:    2,
		Slug:               "promo-delete-claim",
		Status:             "active",
	}
	if err := db.Create(&claimCampaign).Error; err != nil {
		t.Fatalf("create claim campaign failed: %v", err)
	}
	if err := db.Create(&models.PromotionCardClaim{
		CampaignID: claimCampaign.ID,
		UserID:     102,
		Status:     promotionCardClaimStatusActive,
	}).Error; err != nil {
		t.Fatalf("create claim failed: %v", err)
	}

	canDelete, err = canDeletePromotionCampaign(db, claimCampaign.ID)
	if err != nil {
		t.Fatalf("canDeletePromotionCampaign(claim) failed: %v", err)
	}
	if canDelete {
		t.Fatalf("want campaign with active claim not deletable")
	}
}

func TestUpdateMerchantPromotionCampaignPreservesRewardValueAndThreshold(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupPromotionCardTestDB(t)
	config.DB = db

	merchant := models.Merchant{Name: "测试商户", Phone: "13800000035", Password: "x", Type: "理发", SupportDirectSale: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	template := models.CardTemplate{MerchantID: merchant.ID, Name: "测试卡", CardType: "times", Price: 10000, TotalTimes: 10, IsActive: true}
	if err := db.Create(&template).Error; err != nil {
		t.Fatalf("create template failed: %v", err)
	}
	campaign := models.PromotionCardCampaign{
		MerchantID:         merchant.ID,
		CardTemplateID:     template.ID,
		Title:              "原始活动",
		RewardTotalTimes:   5,
		RewardCardQuantity: 8,
		RewardThreshold:    15,
		Slug:               "promo-update-locked-fields",
		Status:             "active",
	}
	if err := db.Create(&campaign).Error; err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}

	body := `{"card_template_id":` + strconv.Itoa(int(template.ID)) + `,"title":"更新标题","reward_total_times":99,"reward_card_quantity":12,"reward_threshold":88,"promo_price":0,"promo_quantity":0,"promo_ends_at":"","status":"active"}`
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/promotion-campaigns/"+strconv.Itoa(int(campaign.ID)), strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(campaign.ID))}}
	c.Set("merchant_id", merchant.ID)

	UpdateMerchantPromotionCampaign(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var updated models.PromotionCardCampaign
	if err := db.First(&updated, campaign.ID).Error; err != nil {
		t.Fatalf("reload campaign failed: %v", err)
	}
	if updated.RewardTotalTimes != 5 {
		t.Fatalf("want reward_total_times kept as 5, got %d", updated.RewardTotalTimes)
	}
	if updated.RewardThreshold != 15 {
		t.Fatalf("want reward_threshold kept as 15, got %d", updated.RewardThreshold)
	}
	if updated.RewardCardQuantity != 12 {
		t.Fatalf("want reward_card_quantity updated to 12, got %d", updated.RewardCardQuantity)
	}
	if updated.Title != "更新标题" {
		t.Fatalf("want title updated, got %s", updated.Title)
	}
}
