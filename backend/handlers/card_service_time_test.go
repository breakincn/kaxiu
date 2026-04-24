package handlers

import (
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

func setupCardServiceTimeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Merchant{}, &models.Card{}, &models.MerchantProject{}, &models.CardProject{}, &models.VerifyCode{}, &models.Usage{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func seedCardServiceTimeFixture(t *testing.T, db *gorm.DB, slotStart time.Time) (models.User, models.Card, models.MerchantProject) {
	t.Helper()
	user := models.User{Username: "service_time_user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	merchant := models.Merchant{Name: "service_time_merchant", Phone: fmt.Sprintf("188%08d", time.Now().UnixNano()%100000000), Password: "pwd", IsOpen: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	card := models.Card{UserID: user.ID, MerchantID: merchant.ID, CardNo: "00001", CardType: "爵士舞课", TotalTimes: 10, RemainTimes: 10}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	weekday := int(slotStart.In(appointmentLocation()).Weekday())
	if weekday == 0 {
		weekday = 7
	}
	project := models.MerchantProject{
		MerchantID: merchant.ID,
		Name:       "爵士舞课",
		Duration:   60,
		IsActive:   true,
		ServiceTimeSlots: models.MerchantProjectServiceTimeSlots{{
			RecurrenceType: "weekly",
			Weekday:        weekday,
			StartTime:      slotStart.In(appointmentLocation()).Format("15:04"),
		}},
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if err := db.Create(&models.CardProject{CardID: card.ID, ProjectID: project.ID}).Error; err != nil {
		t.Fatalf("create card project failed: %v", err)
	}
	return user, card, project
}

func newUserVerifyCodeContext(method, path string, userID uint, body string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", userID)
	return c, rec
}

func TestGenerateVerifyCodeRejectsOutsideProjectServiceTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupCardServiceTimeTestDB(t)
	now := time.Now().In(appointmentLocation())
	user, card, project := seedCardServiceTimeFixture(t, config.DB, now.Add(2*time.Hour))

	body := fmt.Sprintf(`{"project_id":%d}`, project.ID)
	c, rec := newUserVerifyCodeContext(http.MethodPost, "/user/cards/1/verify-code", user.ID, body)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", card.ID)}}
	GenerateVerifyCode(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "当前不在卡片项目服务时间") {
		t.Fatalf("unexpected body=%s", rec.Body.String())
	}
}

func TestGenerateVerifyCodeAllowsWithinProjectServiceTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupCardServiceTimeTestDB(t)
	now := time.Now().In(appointmentLocation())
	user, card, project := seedCardServiceTimeFixture(t, config.DB, now.Add(30*time.Minute))

	body := fmt.Sprintf(`{"project_id":%d}`, project.ID)
	c, rec := newUserVerifyCodeContext(http.MethodPost, "/user/cards/1/verify-code", user.ID, body)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", card.ID)}}
	GenerateVerifyCode(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if strings.TrimSpace(payload.Data.Code) == "" {
		t.Fatalf("want verify code, body=%s", rec.Body.String())
	}
}

func TestGenerateVerifyCodeRejectsDuplicateMultiServiceCodeInSameWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupCardServiceTimeTestDB(t)
	now := time.Now().In(appointmentLocation())
	user, card, project := seedCardServiceTimeFixture(t, config.DB, now.Add(30*time.Minute))
	project.ServiceCapacity = 15
	if err := config.DB.Model(&models.MerchantProject{}).Where("id = ?", project.ID).Update("service_capacity", 15).Error; err != nil {
		t.Fatalf("update project service capacity failed: %v", err)
	}
	verifyCode := models.VerifyCode{
		CardID:    card.ID,
		ProjectID: &project.ID,
		Code:      "DUPLICATE01",
		ExpireAt:  now.Add(5 * time.Minute).Unix(),
		Used:      false,
		CreatedAt: &now,
	}
	if err := config.DB.Create(&verifyCode).Error; err != nil {
		t.Fatalf("create verify code failed: %v", err)
	}

	body := fmt.Sprintf(`{"project_id":%d}`, project.ID)
	c, rec := newUserVerifyCodeContext(http.MethodPost, "/user/cards/1/verify-code", user.ID, body)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", card.ID)}}
	GenerateVerifyCode(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "当前服务时间段已生成核销码") {
		t.Fatalf("unexpected body=%s", rec.Body.String())
	}
}

func TestGenerateVerifyCodeRejectsUsedMultiServiceCodeInSameWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupCardServiceTimeTestDB(t)
	now := time.Now().In(appointmentLocation())
	user, card, project := seedCardServiceTimeFixture(t, config.DB, now.Add(30*time.Minute))
	project.ServiceCapacity = 15
	if err := config.DB.Model(&models.MerchantProject{}).Where("id = ?", project.ID).Update("service_capacity", 15).Error; err != nil {
		t.Fatalf("update project service capacity failed: %v", err)
	}
	usedAt := now.Add(-2 * time.Minute)
	verifyCode := models.VerifyCode{
		CardID:    card.ID,
		ProjectID: &project.ID,
		Code:      "USEDVC01",
		ExpireAt:  now.Add(5 * time.Minute).Unix(),
		Used:      true,
		UsedAt:    &usedAt,
		CreatedAt: &now,
	}
	if err := config.DB.Create(&verifyCode).Error; err != nil {
		t.Fatalf("create verify code failed: %v", err)
	}

	body := fmt.Sprintf(`{"project_id":%d}`, project.ID)
	c, rec := newUserVerifyCodeContext(http.MethodPost, "/user/cards/1/verify-code", user.ID, body)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", card.ID)}}
	GenerateVerifyCode(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "当前服务时间段已核销") {
		t.Fatalf("unexpected body=%s", rec.Body.String())
	}
}
