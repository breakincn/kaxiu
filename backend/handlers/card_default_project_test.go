package handlers

import (
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

func setupCardDefaultProjectTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Merchant{}, &models.Card{}, &models.MerchantProject{}, &models.CardProject{}); err != nil {
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
