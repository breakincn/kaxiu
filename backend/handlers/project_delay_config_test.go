package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProjectHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.MerchantProject{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func newProjectMerchantJSONContext(method, path string, merchantID uint, body string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", merchantID)
	return c, rec
}

func seedProjectMerchant(t *testing.T, db *gorm.DB) models.Merchant {
	t.Helper()
	merchant := models.Merchant{Name: "project-merchant", Phone: "18800012345", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	return merchant
}

func TestCreateMerchantProjectSupportsDelayCompensationConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupProjectHandlerTestDB(t)
	merchant := seedProjectMerchant(t, config.DB)

	body := `{"name":"肩颈调理","duration":60,"room_select_timeout_seconds":120,"start_pending_timeout_seconds":240,"delay_tolerance_minutes":2,"delay_compensation_mode":"fixed_unit","delay_redeem_threshold_percent":50,"delay_fixed_unit_value":3}`
	c, rec := newProjectMerchantJSONContext(http.MethodPost, "/merchant/projects", merchant.ID, body)
	CreateMerchantProject(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.MerchantProject `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Data.RoomSelectTimeoutSeconds != 120 || resp.Data.StartPendingTimeoutSeconds != 240 || resp.Data.DelayToleranceMinutes != 2 || resp.Data.DelayCompensationMode != "fixed_unit" || resp.Data.DelayRedeemThresholdPercent != 50 || resp.Data.DelayFixedUnitValue != 3 {
		t.Fatalf("unexpected delay config in response: %+v", resp.Data)
	}

	var got models.MerchantProject
	if err := config.DB.First(&got, resp.Data.ID).Error; err != nil {
		t.Fatalf("load project failed: %v", err)
	}
	if got.RoomSelectTimeoutSeconds != 120 || got.StartPendingTimeoutSeconds != 240 || got.DelayToleranceMinutes != 2 || got.DelayCompensationMode != "fixed_unit" || got.DelayRedeemThresholdPercent != 50 || got.DelayFixedUnitValue != 3 {
		t.Fatalf("unexpected delay config in db: %+v", got)
	}
}

func TestUpdateMerchantProjectSupportsDelayCompensationConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupProjectHandlerTestDB(t)
	merchant := seedProjectMerchant(t, config.DB)
	project := models.MerchantProject{
		MerchantID:                  merchant.ID,
		Name:                        "肩颈调理",
		Duration:                    60,
		RoomSelectTimeoutSeconds:    90,
		StartPendingTimeoutSeconds:  300,
		DelayToleranceMinutes:       1,
		DelayCompensationMode:       "minutes_bucket",
		DelayRedeemThresholdPercent: 100,
		DelayFixedUnitValue:         0,
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	body := `{"room_select_timeout_seconds":150,"start_pending_timeout_seconds":180,"delay_tolerance_minutes":5,"delay_compensation_mode":"amount_bucket","delay_redeem_threshold_percent":120,"delay_fixed_unit_value":8}`
	projectID := strconv.Itoa(int(project.ID))
	c, rec := newProjectMerchantJSONContext(http.MethodPut, "/merchant/projects/"+projectID, merchant.ID, body)
	c.Params = gin.Params{{Key: "id", Value: projectID}}
	UpdateMerchantProject(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got models.MerchantProject
	if err := config.DB.First(&got, project.ID).Error; err != nil {
		t.Fatalf("load project failed: %v", err)
	}
	if got.RoomSelectTimeoutSeconds != 150 || got.StartPendingTimeoutSeconds != 180 || got.DelayToleranceMinutes != 5 || got.DelayCompensationMode != "amount_bucket" || got.DelayRedeemThresholdPercent != 120 || got.DelayFixedUnitValue != 8 {
		t.Fatalf("unexpected updated delay config: %+v", got)
	}
}

func TestCreateMerchantProjectAssignsDefaultAndSwitchesDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupProjectHandlerTestDB(t)
	merchant := seedProjectMerchant(t, config.DB)

	c1, rec1 := newProjectMerchantJSONContext(http.MethodPost, "/merchant/projects", merchant.ID, `{"name":"项目A","duration":45}`)
	CreateMerchantProject(c1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec1.Code, rec1.Body.String())
	}

	var first models.MerchantProject
	if err := config.DB.Where("merchant_id = ? AND name = ?", merchant.ID, "项目A").First(&first).Error; err != nil {
		t.Fatalf("load first project failed: %v", err)
	}
	if !first.IsDefault {
		t.Fatalf("first project should be default: %+v", first)
	}

	c2, rec2 := newProjectMerchantJSONContext(http.MethodPost, "/merchant/projects", merchant.ID, `{"name":"项目B","duration":60,"is_default":true}`)
	CreateMerchantProject(c2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}

	var projects []models.MerchantProject
	if err := config.DB.Where("merchant_id = ?", merchant.ID).Order("id asc").Find(&projects).Error; err != nil {
		t.Fatalf("load projects failed: %v", err)
	}
	defaultCount := 0
	secondDefault := false
	for _, project := range projects {
		if project.IsDefault {
			defaultCount++
			if project.Name == "项目B" {
				secondDefault = true
			}
		}
	}
	if defaultCount != 1 || !secondDefault {
		t.Fatalf("unexpected default projects: %+v", projects)
	}
}

func TestDeleteMerchantProjectRejectsLastProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupProjectHandlerTestDB(t)
	merchant := seedProjectMerchant(t, config.DB)
	project := models.MerchantProject{MerchantID: merchant.ID, Name: "唯一项目", Duration: 45, IsDefault: true}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	projectID := strconv.Itoa(int(project.ID))
	c, rec := newProjectMerchantJSONContext(http.MethodDelete, "/merchant/projects/"+projectID, merchant.ID, "")
	c.Params = gin.Params{{Key: "id", Value: projectID}}
	DeleteMerchantProject(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
