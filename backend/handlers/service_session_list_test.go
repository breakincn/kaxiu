package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
)

func TestListServiceSessionsFiltersByDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	merchant := models.Merchant{Name: "测试门店", Phone: "13800000000", Password: "secret"}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	loc := appointmentLocation()
	today := time.Date(2026, 4, 9, 10, 0, 0, 0, loc)
	yesterday := today.Add(-24 * time.Hour)

	sessions := []models.ServiceSession{
		{MerchantID: merchant.ID, UserID: 1, CardID: 1, Status: "serving"},
		{MerchantID: merchant.ID, UserID: 2, CardID: 2, Status: "finished"},
	}
	if err := config.DB.Create(&sessions).Error; err != nil {
		t.Fatalf("create sessions failed: %v", err)
	}
	if err := config.DB.Model(&models.ServiceSession{}).Where("id = ?", sessions[0].ID).Updates(map[string]interface{}{"created_at": today, "updated_at": today}).Error; err != nil {
		t.Fatalf("update today session time failed: %v", err)
	}
	if err := config.DB.Model(&models.ServiceSession{}).Where("id = ?", sessions[1].ID).Updates(map[string]interface{}{"created_at": yesterday, "updated_at": yesterday}).Error; err != nil {
		t.Fatalf("update yesterday session time failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodGet, "/merchant/service-sessions?date=2026-04-09", merchant.ID)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/service-sessions?date=2026-04-09", nil)
	ListServiceSessions(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []models.ServiceSession `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("want 1 session, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != sessions[0].ID {
		t.Fatalf("want today session id %d, got %d", sessions[0].ID, resp.Data[0].ID)
	}
}

func TestListServiceSessionsReturnsInitialUsageCardSnapshots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)
	if err := config.DB.AutoMigrate(&models.Card{}, &models.Usage{}); err != nil {
		t.Fatalf("migrate card usage failed: %v", err)
	}
	config.DB.Exec("DELETE FROM service_sessions")
	config.DB.Exec("DELETE FROM usages")
	config.DB.Exec("DELETE FROM cards")

	merchant := models.Merchant{Name: "测试门店", Phone: "13800000002", Password: "secret"}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	card := models.Card{MerchantID: merchant.ID, UserID: 1, CardNo: "0012", CardType: "活力成长单车卡", TotalTimes: 32, UsedTimes: 19, RemainTimes: 13}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	loc := appointmentLocation()
	firstAt := time.Date(2026, 4, 16, 15, 40, 0, 0, loc)
	secondAt := time.Date(2026, 4, 16, 16, 22, 0, 0, loc)
	total := 32
	firstUsed := 22
	firstRemain := 10
	secondUsed := 23
	secondRemain := 9
	firstUsage := models.Usage{
		MerchantID:              merchant.ID,
		CardID:                  card.ID,
		UsedTimes:               1,
		Status:                  "success",
		CreatedAt:               &firstAt,
		CardNoSnapshot:          "0012",
		CardTypeSnapshot:        "活力成长单车卡",
		CardTotalTimesSnapshot:  &total,
		CardUsedTimesSnapshot:   &firstUsed,
		CardRemainTimesSnapshot: &firstRemain,
	}
	secondUsage := models.Usage{
		MerchantID:              merchant.ID,
		CardID:                  card.ID,
		UsedTimes:               1,
		Status:                  "success",
		CreatedAt:               &secondAt,
		CardNoSnapshot:          "0012",
		CardTypeSnapshot:        "活力成长单车卡",
		CardTotalTimesSnapshot:  &total,
		CardUsedTimesSnapshot:   &secondUsed,
		CardRemainTimesSnapshot: &secondRemain,
	}
	if err := config.DB.Create(&firstUsage).Error; err != nil {
		t.Fatalf("create first usage failed: %v", err)
	}
	if err := config.DB.Create(&secondUsage).Error; err != nil {
		t.Fatalf("create second usage failed: %v", err)
	}
	if err := config.DB.Model(&models.Usage{}).Where("id = ?", firstUsage.ID).Update("created_at", firstAt).Error; err != nil {
		t.Fatalf("update first usage time failed: %v", err)
	}
	if err := config.DB.Model(&models.Usage{}).Where("id = ?", secondUsage.ID).Update("created_at", secondAt).Error; err != nil {
		t.Fatalf("update second usage time failed: %v", err)
	}

	sessions := []models.ServiceSession{
		{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, InitialUsageID: firstUsage.ID, Status: "finished", CreatedAt: &firstAt},
		{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, InitialUsageID: secondUsage.ID, Status: "finished", CreatedAt: &secondAt},
	}
	if err := config.DB.Create(&sessions).Error; err != nil {
		t.Fatalf("create sessions failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodGet, "/merchant/service-sessions?date=2026-04-16", merchant.ID)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/service-sessions?date=2026-04-16", nil)
	ListServiceSessions(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []models.ServiceSession `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	byUsageID := make(map[uint]models.ServiceSession, len(resp.Data))
	for _, session := range resp.Data {
		byUsageID[session.InitialUsageID] = session
	}

	firstSession, ok := byUsageID[firstUsage.ID]
	if !ok {
		t.Fatalf("missing first usage session")
	}
	secondSession, ok := byUsageID[secondUsage.ID]
	if !ok {
		t.Fatalf("missing second usage session")
	}
	if firstSession.Card == nil || firstSession.Card.UsedTimes != 22 || firstSession.Card.RemainTimes != 10 {
		t.Fatalf("want first session card snapshot 22/10, got %+v", firstSession.Card)
	}
	if firstSession.InitialUsage == nil || firstSession.InitialUsage.CardUsedTimesSnapshot == nil || *firstSession.InitialUsage.CardUsedTimesSnapshot != 22 {
		t.Fatalf("want first usage snapshot used 22, got %+v", firstSession.InitialUsage)
	}
	if secondSession.Card == nil || secondSession.Card.UsedTimes != 23 || secondSession.Card.RemainTimes != 9 {
		t.Fatalf("want second session card snapshot 23/9, got %+v", secondSession.Card)
	}
	if secondSession.InitialUsage == nil || secondSession.InitialUsage.CardRemainTimesSnapshot == nil || *secondSession.InitialUsage.CardRemainTimesSnapshot != 9 {
		t.Fatalf("want second usage snapshot remain 9, got %+v", secondSession.InitialUsage)
	}
}

func TestListServiceSessionsSelfOnlyForStaff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	merchant := models.Merchant{Name: "测试门店", Phone: "13800000001", Password: "secret"}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	role := models.ServiceRole{Key: "technician", Name: "技师", RoleType: "professional"}
	if err := config.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}

	techA := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "客服A", Code: "cs0001", Account: "cs0001", Password: "secret"}
	techB := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "客服B", Code: "cs0002", Account: "cs0002", Password: "secret"}
	if err := config.DB.Create(&techA).Error; err != nil {
		t.Fatalf("create techA failed: %v", err)
	}
	if err := config.DB.Create(&techB).Error; err != nil {
		t.Fatalf("create techB failed: %v", err)
	}

	sessions := []models.ServiceSession{
		{MerchantID: merchant.ID, UserID: 1, CardID: 1, TechnicianID: &techA.ID, Status: "serving"},
		{MerchantID: merchant.ID, UserID: 2, CardID: 2, LastTechnicianID: &techA.ID, Status: "canceled"},
		{MerchantID: merchant.ID, UserID: 3, CardID: 3, TechnicianID: &techB.ID, Status: "finished"},
	}
	if err := config.DB.Create(&sessions).Error; err != nil {
		t.Fatalf("create sessions failed: %v", err)
	}

	c, rec := newStaffContext(http.MethodGet, "/merchant/service-sessions?self_only=1", merchant.ID, techA.ID, role.ID)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/service-sessions?self_only=1", nil)
	ListServiceSessions(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []models.ServiceSession `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("want 2 sessions, got %d", len(resp.Data))
	}
	for _, session := range resp.Data {
		if session.ID == sessions[2].ID {
			t.Fatalf("unexpected session %d in self_only response", session.ID)
		}
	}
}
