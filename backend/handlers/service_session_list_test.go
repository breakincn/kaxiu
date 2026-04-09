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
