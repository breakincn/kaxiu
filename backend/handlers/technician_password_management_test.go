package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTechnicianPasswordTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Merchant{},
		&models.ServiceRole{},
		&models.Technician{},
		&models.MerchantProject{},
		&models.TechnicianAppointmentProject{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func createTechnicianPasswordTestRole(t *testing.T, db *gorm.DB, merchantID uint) models.ServiceRole {
	t.Helper()
	role := models.ServiceRole{
		MerchantID:    &merchantID,
		Key:           "m1_js",
		Name:          "技师",
		AccountPrefix: "js",
		RoleType:      "professional",
		IsActive:      true,
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	return role
}

func TestCreateMerchantTechnicianStoresOriginalPasswordAndListFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupTechnicianPasswordTestDB(t)

	merchant := models.Merchant{Name: "m", Phone: "18800002201", Password: "pwd"}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	createTechnicianPasswordTestRole(t, config.DB, merchant.ID)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/technicians", bytes.NewBufferString(`{"name":"大漂亮","role":"m1_js"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchant.ID)

	CreateMerchantTechnician(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var createResp struct {
		Data struct {
			ID              uint   `json:"id"`
			DefaultPassword string `json:"default_password"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create response failed: %v", err)
	}
	if createResp.Data.ID == 0 || strings.TrimSpace(createResp.Data.DefaultPassword) == "" {
		t.Fatalf("unexpected create response: %s", rec.Body.String())
	}
	if len(createResp.Data.DefaultPassword) != 8 {
		t.Fatalf("want 8-digit default password, got %q", createResp.Data.DefaultPassword)
	}
	for _, ch := range createResp.Data.DefaultPassword {
		if ch < '0' || ch > '9' {
			t.Fatalf("want numeric-only default password, got %q", createResp.Data.DefaultPassword)
		}
	}

	var tech models.Technician
	if err := config.DB.First(&tech, createResp.Data.ID).Error; err != nil {
		t.Fatalf("load technician failed: %v", err)
	}
	if tech.OriginalPassword != createResp.Data.DefaultPassword {
		t.Fatalf("want original password stored, got %q", tech.OriginalPassword)
	}
	if !tech.PasswordNeedReset {
		t.Fatalf("want password_need_reset=true")
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/technicians", nil)
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchant.ID)

	GetMerchantTechnicians(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Data []struct {
			ID                      uint `json:"id"`
			CanViewOriginalPassword bool `json:"can_view_original_password"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response failed: %v", err)
	}
	if len(listResp.Data) != 1 || !listResp.Data[0].CanViewOriginalPassword {
		t.Fatalf("want can_view_original_password=true, got body=%s", rec.Body.String())
	}
}

func TestResetMerchantTechnicianPasswordAndSelfChangeClearsOriginalPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupTechnicianPasswordTestDB(t)

	merchant := models.Merchant{Name: "m", Phone: "18800002202", Password: "pwd"}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	role := createTechnicianPasswordTestRole(t, config.DB, merchant.ID)
	oldHashed, err := bcrypt.GenerateFromPassword([]byte("Initpass88"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	tech := models.Technician{
		MerchantID:        merchant.ID,
		ServiceRoleID:     role.ID,
		Name:              "大漂亮",
		Code:              "0001",
		Account:           "js0001",
		Password:          string(oldHashed),
		PasswordNeedReset: true,
		OriginalPassword:  "Initpass88",
		IsActive:          true,
	}
	if err := config.DB.Create(&tech).Error; err != nil {
		t.Fatalf("create technician failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/technicians/1/reset-password", bytes.NewBufferString(`{"new_password":"Resetpass88"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchant.ID)

	ResetMerchantTechnicianPassword(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	if err := config.DB.First(&tech, tech.ID).Error; err != nil {
		t.Fatalf("reload technician failed: %v", err)
	}
	if tech.OriginalPassword != "Resetpass88" {
		t.Fatalf("want reset original password stored, got %q", tech.OriginalPassword)
	}
	if !tech.PasswordNeedReset {
		t.Fatalf("want password_need_reset=true after merchant reset")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(tech.Password), []byte("Resetpass88")); err != nil {
		t.Fatalf("stored password mismatch: %v", err)
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/technician/password/reset", bytes.NewBufferString(`{"old_password":"Resetpass88","new_password":"Changed999"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth_type", "staff")
	c.Set("merchant_id", merchant.ID)
	c.Set("technician_id", tech.ID)

	ResetTechnicianPassword(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	if err := config.DB.First(&tech, tech.ID).Error; err != nil {
		t.Fatalf("reload technician failed: %v", err)
	}
	if tech.OriginalPassword != "" {
		t.Fatalf("want original password cleared after self change, got %q", tech.OriginalPassword)
	}
	if tech.PasswordNeedReset {
		t.Fatalf("want password_need_reset=false after self change")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(tech.Password), []byte("Changed999")); err != nil {
		t.Fatalf("stored password mismatch after self change: %v", err)
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/technicians/1/original-password", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchant.ID)

	GetMerchantTechnicianOriginalPassword(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 after self change, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMerchantTechnicianAppointmentProjectsCanBeConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupTechnicianPasswordTestDB(t)

	merchant := models.Merchant{Name: "m", Phone: "18800002203", Password: "pwd", SupportAppointment: true}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	role := createTechnicianPasswordTestRole(t, config.DB, merchant.ID)
	tech := models.Technician{
		MerchantID:    merchant.ID,
		ServiceRoleID: role.ID,
		Name:          "小美",
		Code:          "0001",
		Account:       "js0001",
		Password:      "pwd",
		IsActive:      true,
	}
	if err := config.DB.Create(&tech).Error; err != nil {
		t.Fatalf("create technician failed: %v", err)
	}
	projectA := models.MerchantProject{MerchantID: merchant.ID, Name: "塑形", Duration: 45, BookableOnline: true, IsActive: true}
	projectB := models.MerchantProject{MerchantID: merchant.ID, Name: "拉伸", Duration: 60, BookableOnline: true, IsActive: true}
	if err := config.DB.Create(&projectA).Error; err != nil {
		t.Fatalf("create projectA failed: %v", err)
	}
	if err := config.DB.Create(&projectB).Error; err != nil {
		t.Fatalf("create projectB failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/technicians/1/appointment-projects", nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(tech.ID))}}
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchant.ID)
	GetMerchantTechnicianAppointmentProjects(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var getResp struct {
		Data struct {
			BindingMode string `json:"binding_mode"`
			ProjectIDs  []uint `json:"project_ids"`
			Projects    []struct {
				ID uint `json:"id"`
			} `json:"projects"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("unmarshal get response failed: %v", err)
	}
	if getResp.Data.BindingMode != "merchant_default" {
		t.Fatalf("want merchant_default, got %s", getResp.Data.BindingMode)
	}
	if len(getResp.Data.ProjectIDs) != 0 {
		t.Fatalf("want empty custom project ids, got %+v", getResp.Data.ProjectIDs)
	}
	if len(getResp.Data.Projects) != 2 {
		t.Fatalf("want 2 available projects, got %d", len(getResp.Data.Projects))
	}

	body, _ := json.Marshal(map[string]any{
		"project_ids": []uint{projectB.ID, projectA.ID, projectA.ID},
	})
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/technicians/1/appointment-projects", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(tech.ID))}}
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchant.ID)
	SetMerchantTechnicianAppointmentProjects(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var rows []models.TechnicianAppointmentProject
	if err := config.DB.Where("merchant_id = ? AND technician_id = ?", merchant.ID, tech.ID).Order("project_id asc").Find(&rows).Error; err != nil {
		t.Fatalf("load binding rows failed: %v", err)
	}
	if len(rows) != 2 || rows[0].ProjectID != projectA.ID || rows[1].ProjectID != projectB.ID {
		t.Fatalf("unexpected binding rows: %+v", rows)
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/technicians", nil)
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchant.ID)
	GetMerchantTechnicians(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Data []struct {
			ID                            uint   `json:"id"`
			AppointmentProjectIDs         []uint `json:"appointment_project_ids"`
			AppointmentProjectBindingMode string `json:"appointment_project_binding_mode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response failed: %v", err)
	}
	if len(listResp.Data) != 1 {
		t.Fatalf("want 1 technician, got %d", len(listResp.Data))
	}
	if listResp.Data[0].AppointmentProjectBindingMode != "custom" {
		t.Fatalf("want custom binding mode, got %s", listResp.Data[0].AppointmentProjectBindingMode)
	}
	if len(listResp.Data[0].AppointmentProjectIDs) != 2 {
		t.Fatalf("want 2 bound project ids, got %+v", listResp.Data[0].AppointmentProjectIDs)
	}
}
