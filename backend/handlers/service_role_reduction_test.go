package handlers

import (
	"bytes"
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

func setupServiceRoleReductionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Merchant{},
		&models.Permission{},
		&models.ServiceRole{},
		&models.RolePermission{},
		&models.SystemConfig{},
		&models.Technician{},
		&models.MerchantRoleAttendanceConfig{},
		&models.MerchantRoleStartPendingConfig{},
		&models.MerchantRolePermissionOverride{},
		&models.TechnicianAttendance{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func mustCreateServiceRole(t *testing.T, db *gorm.DB, role models.ServiceRole) models.ServiceRole {
	t.Helper()
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	return role
}

func mustCreatePermission(t *testing.T, db *gorm.DB, key string) models.Permission {
	t.Helper()
	perm := models.Permission{Key: key, Name: key}
	if err := db.Create(&perm).Error; err != nil {
		t.Fatalf("create permission failed: %v", err)
	}
	return perm
}

func TestAdminListServiceRolesReturnsOnlyFixedRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceRoleReductionTestDB(t)

	mustCreateServiceRole(t, config.DB, models.ServiceRole{Key: "store_manager", Name: "店长", RoleType: "operational", IsActive: true})
	mustCreateServiceRole(t, config.DB, models.ServiceRole{Key: "front_desk", Name: "前台", RoleType: "operational", IsActive: true})
	mustCreateServiceRole(t, config.DB, models.ServiceRole{Key: "teacher", Name: "助教", RoleType: "professional", IsActive: true})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/service-roles", nil)

	AdminListServiceRoles(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []models.ServiceRole `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("want 2 fixed roles, got %d body=%s", len(resp.Data), rec.Body.String())
	}
	for _, role := range resp.Data {
		if !config.IsFixedServiceRoleKey(role.Key) {
			t.Fatalf("unexpected role returned: %s", role.Key)
		}
	}
}

func TestAdminServiceRoleMutationsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name    string
		handler gin.HandlerFunc
		method  string
		path    string
		body    string
	}{
		{name: "create", handler: AdminCreateServiceRole, method: http.MethodPost, path: "/admin/service-roles", body: `{"key":"teacher","name":"助教"}`},
		{name: "update", handler: AdminUpdateServiceRole, method: http.MethodPut, path: "/admin/service-roles/1", body: `{"name":"新前台"}`},
		{name: "delete", handler: AdminDeleteServiceRole, method: http.MethodDelete, path: "/admin/service-roles/1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			if tc.name != "create" {
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			}
			tc.handler(c)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("want 403, got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateMerchantProfessionalRoleUsesStableKeyAndRejectsReservedName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceRoleReductionTestDB(t)

	m := models.Merchant{Name: "m", Phone: "18800001200", Password: "pwd"}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	for _, key := range []string{"merchant.card.verify", "merchant.card.finish", "merchant.card.sell"} {
		mustCreatePermission(t, config.DB, key)
	}
	if err := config.DB.Create(&models.SystemConfig{Key: "professional_base_permission_keys", Value: "merchant.card.verify,merchant.card.finish,merchant.card.sell"}).Error; err != nil {
		t.Fatalf("create system config failed: %v", err)
	}

	body := `{"name":"助教","account_prefix":"zj"}`
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/professional-roles", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", m.ID)

	CreateMerchantProfessionalRole(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.ServiceRole `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Data.Key != "m1_zj" {
		t.Fatalf("want stable role key m1_zj, got %s", resp.Data.Key)
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/professional-roles", bytes.NewBufferString(`{"name":"店长","account_prefix":"dz"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", m.ID)

	CreateMerchantProfessionalRole(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetMerchantProfessionalRolesExcludesPlatformRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceRoleReductionTestDB(t)

	m := models.Merchant{Name: "m", Phone: "18800001201", Password: "pwd"}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	mid := m.ID
	mustCreateServiceRole(t, config.DB, models.ServiceRole{Key: "teacher", Name: "助教", RoleType: "professional", IsActive: true})
	mustCreateServiceRole(t, config.DB, models.ServiceRole{MerchantID: &mid, Key: "m1_zj", Name: "助教", AccountPrefix: "zj", RoleType: "professional", IsActive: true})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/professional-roles", nil)
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", m.ID)

	GetMerchantProfessionalRoles(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []models.ServiceRole `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Key != "m1_zj" {
		t.Fatalf("want only merchant role, got body=%s", rec.Body.String())
	}
}

func TestCreateMerchantTechnicianRejectsPlatformProfessionalRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceRoleReductionTestDB(t)

	m := models.Merchant{Name: "m", Phone: "18800001202", Password: "pwd"}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	mustCreateServiceRole(t, config.DB, models.ServiceRole{Key: "teacher", Name: "助教", AccountPrefix: "zj", RoleType: "professional", IsActive: true})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/technicians", bytes.NewBufferString(`{"name":"艾薇","role":"teacher"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", m.ID)

	CreateMerchantTechnician(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}
