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

func setupServiceSessionExtendRequestTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Merchant{},
		&models.Card{},
		&models.MerchantProject{},
		&models.CardProject{},
		&models.Usage{},
		&models.ServiceSession{},
		&models.ServiceSessionExtendRequest{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func newExtendRequestUserContext(method, path string, userID uint, body string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", userID)
	return c, rec
}

func newExtendRequestMerchantContext(method, path string, merchantID uint, body string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", merchantID)
	c.Set("auth_type", "merchant")
	return c, rec
}

func seedExtendRequestFixture(t *testing.T) (models.Merchant, models.User, models.Card, models.MerchantProject, models.MerchantProject, models.ServiceSession) {
	t.Helper()
	now := time.Now().Truncate(time.Second)
	merchant := models.Merchant{Name: "加钟门店", Phone: "18800001111", Password: "pwd"}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	userPhone := "17700001111"
	user := models.User{Phone: &userPhone}
	if err := config.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	card := models.Card{
		UserID:      user.ID,
		MerchantID:  merchant.ID,
		CardNo:      "00012",
		CardType:    "活力次卡",
		TotalTimes:  10,
		RemainTimes: 7,
		UsedTimes:   3,
	}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	ownedProject := models.MerchantProject{MerchantID: merchant.ID, Name: "有氧健身操", Duration: 30, IsActive: true}
	otherProject := models.MerchantProject{MerchantID: merchant.ID, Name: "普拉提", Duration: 45, IsActive: true}
	if err := config.DB.Create(&ownedProject).Error; err != nil {
		t.Fatalf("create owned project failed: %v", err)
	}
	if err := config.DB.Create(&otherProject).Error; err != nil {
		t.Fatalf("create other project failed: %v", err)
	}
	if err := config.DB.Create(&models.CardProject{CardID: card.ID, ProjectID: ownedProject.ID}).Error; err != nil {
		t.Fatalf("create card project failed: %v", err)
	}
	usage := models.Usage{
		CardID:     card.ID,
		MerchantID: merchant.ID,
		ProjectID:  &ownedProject.ID,
		UsedTimes:  1,
		UsedAt:     &now,
		Status:     "in_progress",
	}
	if err := config.DB.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card.ID,
		ProjectID:       &ownedProject.ID,
		InitialUsageID:  usage.ID,
		Status:          "serving",
		DurationMinutes: 30,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create service session failed: %v", err)
	}
	return merchant, user, card, ownedProject, otherProject, session
}

func TestCreateServiceSessionExtendRequestRequiresCardProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionExtendRequestTestDB(t)

	_, user, _, _, otherProject, session := seedExtendRequestFixture(t)

	sessionID := strconv.Itoa(int(session.ID))
	body := `{"project_id":` + strconv.Itoa(int(otherProject.ID)) + `}`
	c, rec := newExtendRequestUserContext(http.MethodPost, "/user/service-sessions/"+sessionID+"/extend-requests", user.ID, body)
	c.Params = gin.Params{{Key: "id", Value: sessionID}}

	CreateServiceSessionExtendRequest(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "该项目不属于当前卡片") {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestApproveServiceSessionExtendRequestExtendsServingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionExtendRequestTestDB(t)

	merchant, user, card, ownedProject, _, session := seedExtendRequestFixture(t)

	sessionID := strconv.Itoa(int(session.ID))
	createBody := `{"project_id":` + strconv.Itoa(int(ownedProject.ID)) + `}`
	createCtx, createRec := newExtendRequestUserContext(http.MethodPost, "/user/service-sessions/"+sessionID+"/extend-requests", user.ID, createBody)
	createCtx.Params = gin.Params{{Key: "id", Value: sessionID}}
	CreateServiceSessionExtendRequest(createCtx)
	if createRec.Code != http.StatusOK {
		t.Fatalf("want create 200, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	var createResp struct {
		Data models.ServiceSessionExtendRequest `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response failed: %v", err)
	}
	if createResp.Data.Status != "pending" || createResp.Data.Minutes != ownedProject.Duration {
		t.Fatalf("unexpected create response: %+v", createResp.Data)
	}

	requestID := strconv.Itoa(int(createResp.Data.ID))
	approveCtx, approveRec := newExtendRequestMerchantContext(http.MethodPost, "/merchant/service-session-extend-requests/"+requestID+"/approve", merchant.ID, `{}`)
	approveCtx.Params = gin.Params{{Key: "id", Value: requestID}}
	ApproveServiceSessionExtendRequest(approveCtx)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("want approve 200, got %d body=%s", approveRec.Code, approveRec.Body.String())
	}

	var gotSession models.ServiceSession
	if err := config.DB.First(&gotSession, session.ID).Error; err != nil {
		t.Fatalf("load service session failed: %v", err)
	}
	if gotSession.DurationMinutes != session.DurationMinutes+ownedProject.Duration {
		t.Fatalf("want duration %d, got %d", session.DurationMinutes+ownedProject.Duration, gotSession.DurationMinutes)
	}
	wantAutoFinishDelaySeconds := session.AutoFinishDelaySeconds + ownedProject.Duration*60
	if gotSession.AutoFinishDelaySeconds != wantAutoFinishDelaySeconds {
		t.Fatalf("want auto finish delay %d, got %d", wantAutoFinishDelaySeconds, gotSession.AutoFinishDelaySeconds)
	}
	var gotCard struct {
		RemainTimes int
		UsedTimes   int
	}
	if err := config.DB.Table("cards").Select("remain_times, used_times").Where("id = ?", card.ID).Scan(&gotCard).Error; err != nil {
		t.Fatalf("load card failed: %v", err)
	}
	if gotCard.RemainTimes != card.RemainTimes-1 || gotCard.UsedTimes != card.UsedTimes+1 {
		t.Fatalf("want card deducted to remain=%d used=%d, got remain=%d used=%d", card.RemainTimes-1, card.UsedTimes+1, gotCard.RemainTimes, gotCard.UsedTimes)
	}
	var gotReq struct {
		Status                 string
		HandledAt              *string
		BeforeRemainingSeconds int
		AfterRemainingSeconds  int
	}
	if err := config.DB.Table("service_session_extend_requests").Select("status, handled_at, before_remaining_seconds, after_remaining_seconds").Where("id = ?", createResp.Data.ID).Scan(&gotReq).Error; err != nil {
		t.Fatalf("load extend request failed: %v", err)
	}
	if gotReq.Status != "approved" || gotReq.HandledAt == nil {
		t.Fatalf("request not approved: %+v", gotReq)
	}
	if gotReq.BeforeRemainingSeconds != 0 || gotReq.AfterRemainingSeconds != ownedProject.Duration*60 {
		t.Fatalf("unexpected remaining snapshots: %+v", gotReq)
	}
}

func TestApproveServiceSessionExtendRequestDeductsBalanceCardAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionExtendRequestTestDB(t)

	now := time.Now().Truncate(time.Second)
	merchant := models.Merchant{Name: "额度门店", Phone: "18800002222", Password: "pwd"}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	userPhone := "17700002222"
	user := models.User{Phone: &userPhone}
	if err := config.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	card := models.Card{
		UserID:         user.ID,
		MerchantID:     merchant.ID,
		CardNo:         "B001",
		CardType:       "储值卡",
		RechargeAmount: 100,
	}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	project := models.MerchantProject{MerchantID: merchant.ID, Name: "精油护理", Duration: 30, Price: 35, IsActive: true}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if err := config.DB.Create(&models.CardProject{CardID: card.ID, ProjectID: project.ID}).Error; err != nil {
		t.Fatalf("create card project failed: %v", err)
	}
	usage := models.Usage{
		CardID:     card.ID,
		MerchantID: merchant.ID,
		ProjectID:  &project.ID,
		UsedTimes:  1,
		UsedAt:     &now,
		Status:     "in_progress",
	}
	if err := config.DB.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card.ID,
		ProjectID:       &project.ID,
		InitialUsageID:  usage.ID,
		Status:          "serving",
		DurationMinutes: 30,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create service session failed: %v", err)
	}

	sessionID := strconv.Itoa(int(session.ID))
	createBody := `{"project_id":` + strconv.Itoa(int(project.ID)) + `}`
	createCtx, createRec := newExtendRequestUserContext(http.MethodPost, "/user/service-sessions/"+sessionID+"/extend-requests", user.ID, createBody)
	createCtx.Params = gin.Params{{Key: "id", Value: sessionID}}
	CreateServiceSessionExtendRequest(createCtx)
	if createRec.Code != http.StatusOK {
		t.Fatalf("want create 200, got %d body=%s", createRec.Code, createRec.Body.String())
	}
	var createResp struct {
		Data models.ServiceSessionExtendRequest `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response failed: %v", err)
	}

	requestID := strconv.Itoa(int(createResp.Data.ID))
	approveCtx, approveRec := newExtendRequestMerchantContext(http.MethodPost, "/merchant/service-session-extend-requests/"+requestID+"/approve", merchant.ID, `{}`)
	approveCtx.Params = gin.Params{{Key: "id", Value: requestID}}
	ApproveServiceSessionExtendRequest(approveCtx)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("want approve 200, got %d body=%s", approveRec.Code, approveRec.Body.String())
	}

	var gotCard struct {
		RechargeAmount int
	}
	if err := config.DB.Table("cards").Select("recharge_amount").Where("id = ?", card.ID).Scan(&gotCard).Error; err != nil {
		t.Fatalf("load card failed: %v", err)
	}
	if gotCard.RechargeAmount != 65 {
		t.Fatalf("want recharge amount 65, got %d", gotCard.RechargeAmount)
	}
	var gotSession models.ServiceSession
	if err := config.DB.First(&gotSession, session.ID).Error; err != nil {
		t.Fatalf("load service session failed: %v", err)
	}
	if gotSession.DurationMinutes != 60 {
		t.Fatalf("want duration 60, got %d", gotSession.DurationMinutes)
	}
}
