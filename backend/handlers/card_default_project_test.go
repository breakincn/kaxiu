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

func setupCardDefaultProjectTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Merchant{}, &models.Card{}, &models.MerchantProject{}, &models.CardProject{}, &models.ServiceSession{}, &models.VerifyCode{}, &models.MultiServiceBooking{}, &models.MultiServicePenaltyLedger{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestGetCardFallsBackToDefaultProjectConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupCardDefaultProjectTestDB(t)
	user := models.User{Username: "u1", Nickname: "用户1"}
	if err := config.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	merchant := models.Merchant{Name: "商户A", Phone: "18800001111", Password: "pwd", QueueWaitingStartSeconds: 90}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:                 merchant.ID,
		Name:                       "默认项目",
		Duration:                   50,
		StartPendingTimeoutSeconds: 300,
		IsDefault:                  true,
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	card := models.Card{UserID: user.ID, MerchantID: merchant.ID, CardNo: "C1", CardType: "次卡", TotalTimes: 10, RemainTimes: 10}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/cards/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("user_id", user.ID)

	GetCard(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.Card `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Data.StartPendingTimeoutSeconds != 300 {
		t.Fatalf("expected card start pending timeout 300, got %d", resp.Data.StartPendingTimeoutSeconds)
	}
	if len(resp.Data.Projects) != 1 || resp.Data.Projects[0].ID != project.ID {
		t.Fatalf("expected default project fallback, got %+v", resp.Data.Projects)
	}
}

func TestGetCardMultiServiceOverviewListsAllProjectCardUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupCardDefaultProjectTestDB(t)
	users := []models.User{
		{Username: "u1", Nickname: "西门吹雪"},
		{Username: "u2", Nickname: "南宫百合"},
		{Username: "u3", Nickname: "未核销用户"},
	}
	for i := range users {
		if err := config.DB.Create(&users[i]).Error; err != nil {
			t.Fatalf("create user %d failed: %v", i, err)
		}
	}
	merchant := models.Merchant{Name: "壹舞团", Phone: "18800002222", Password: "pwd", QueueWaitingStartSeconds: 90}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	now := time.Now().In(config.ProjectServiceTimeLocation())
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	slotStart := now.Add(-30 * time.Minute)
	project := models.MerchantProject{
		MerchantID:      merchant.ID,
		Name:            "爵士舞课",
		Duration:        60,
		ServiceCapacity: 15,
		IsActive:        true,
		ServiceTimeSlots: models.MerchantProjectServiceTimeSlots{{
			RecurrenceType: "weekly",
			Weekday:        weekday,
			StartTime:      slotStart.Format("15:04"),
		}},
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	cards := make([]models.Card, len(users))
	for i := range users {
		cards[i] = models.Card{UserID: users[i].ID, MerchantID: merchant.ID, CardNo: "C" + users[i].Username, CardType: "成人爵士60课时", TotalTimes: 60, RemainTimes: 60}
		if err := config.DB.Create(&cards[i]).Error; err != nil {
			t.Fatalf("create card %d failed: %v", i, err)
		}
		if err := config.DB.Create(&models.CardProject{CardID: cards[i].ID, ProjectID: project.ID}).Error; err != nil {
			t.Fatalf("create card project %d failed: %v", i, err)
		}
	}
	for i := 0; i < 2; i++ {
		session := models.ServiceSession{
			MerchantID:     merchant.ID,
			UserID:         users[i].ID,
			CardID:         cards[i].ID,
			ProjectID:      &project.ID,
			InitialUsageID: uint(i + 1),
			Status:         "cs_start_pending",
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}
		if err := config.DB.Create(&session).Error; err != nil {
			t.Fatalf("create service session %d failed: %v", i, err)
		}
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/cards/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("user_id", users[0].ID)

	GetCard(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.Card `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Data.Projects) != 1 || resp.Data.Projects[0].MultiServiceOverview == nil {
		t.Fatalf("expected project overview, got %+v", resp.Data.Projects)
	}
	overview := resp.Data.Projects[0].MultiServiceOverview
	if len(overview.Participants) != 3 {
		t.Fatalf("expected all 3 project card users, got %+v", overview.Participants)
	}
	if overview.UsedCount != 2 || overview.RemainingCount != 13 {
		t.Fatalf("unexpected counts used=%d remaining=%d", overview.UsedCount, overview.RemainingCount)
	}
	checked := map[string]bool{}
	for _, participant := range overview.Participants {
		checked[participant.Nickname] = participant.CheckedIn
	}
	if !checked["西门吹雪"] || !checked["南宫百合"] {
		t.Fatalf("expected verified users checked in, got %+v", checked)
	}
	if checked["未核销用户"] {
		t.Fatalf("expected unverified project card user not checked in, got %+v", checked)
	}
}

func TestGetCardMultiServiceOverviewIncludesDefaultProjectCardsWithoutBindings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupCardDefaultProjectTestDB(t)
	users := []models.User{
		{Username: "u10", Nickname: "西门吹雪"},
		{Username: "u11", Nickname: "南宫百合"},
	}
	for i := range users {
		if err := config.DB.Create(&users[i]).Error; err != nil {
			t.Fatalf("create user %d failed: %v", i, err)
		}
	}
	merchant := models.Merchant{Name: "壹舞团", Phone: "18800003333", Password: "pwd", QueueWaitingStartSeconds: 90}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	now := time.Now().In(config.ProjectServiceTimeLocation())
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	project := models.MerchantProject{
		MerchantID:      merchant.ID,
		Name:            "爵士舞课",
		Duration:        60,
		ServiceCapacity: 15,
		IsActive:        true,
		IsDefault:       true,
		ServiceTimeSlots: models.MerchantProjectServiceTimeSlots{{
			RecurrenceType: "weekly",
			Weekday:        weekday,
			StartTime:      now.Add(-30 * time.Minute).Format("15:04"),
		}},
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	firstCard := models.Card{UserID: users[0].ID, MerchantID: merchant.ID, CardNo: "CD1", CardType: "成人爵士60课时", TotalTimes: 60, RemainTimes: 60}
	secondCard := models.Card{UserID: users[1].ID, MerchantID: merchant.ID, CardNo: "CD2", CardType: "成人爵士60课时", TotalTimes: 60, RemainTimes: 60}
	if err := config.DB.Create(&firstCard).Error; err != nil {
		t.Fatalf("create first card failed: %v", err)
	}
	if err := config.DB.Create(&secondCard).Error; err != nil {
		t.Fatalf("create second card failed: %v", err)
	}
	if err := config.DB.Create(&models.CardProject{CardID: secondCard.ID, ProjectID: project.ID}).Error; err != nil {
		t.Fatalf("create second card project failed: %v", err)
	}

	session := models.ServiceSession{
		MerchantID:     merchant.ID,
		UserID:         users[1].ID,
		CardID:         secondCard.ID,
		ProjectID:      &project.ID,
		InitialUsageID: 1,
		Status:         "cs_start_pending",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create service session failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/cards/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("user_id", users[0].ID)

	GetCard(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.Card `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	overview := resp.Data.Projects[0].MultiServiceOverview
	if overview == nil {
		t.Fatalf("expected project overview")
	}
	if len(overview.Participants) != 2 {
		t.Fatalf("expected default-project fallback to include both card users, got %+v", overview.Participants)
	}
	checked := map[string]bool{}
	for _, participant := range overview.Participants {
		checked[participant.Nickname] = participant.CheckedIn
	}
	if checked["西门吹雪"] {
		t.Fatalf("expected unverified default-project card user to remain unchecked, got %+v", checked)
	}
	if !checked["南宫百合"] {
		t.Fatalf("expected bound card user to be checked in, got %+v", checked)
	}
}
