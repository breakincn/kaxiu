package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSecurityFixTestDB(t *testing.T) *gorm.DB {
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
		&models.Appointment{},
		&models.Permission{},
		&models.ServiceRole{},
		&models.RolePermission{},
		&models.MerchantRolePermissionOverride{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func seedPermission(t *testing.T, db *gorm.DB, key string) models.Permission {
	t.Helper()
	p := models.Permission{Key: key, Name: key}
	if err := db.Create(&p).Error; err != nil {
		t.Fatalf("create permission %s failed: %v", key, err)
	}
	return p
}

func createStaffRoleWithPermission(t *testing.T, db *gorm.DB, merchantID uint, permissionKey string) models.ServiceRole {
	t.Helper()
	role := models.ServiceRole{MerchantID: &merchantID, Key: permissionKey + ".role", Name: permissionKey + ".role"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	perm := seedPermission(t, db, permissionKey)
	if err := db.Create(&models.RolePermission{ServiceRoleID: role.ID, PermissionID: perm.ID, Allowed: true}).Error; err != nil {
		t.Fatalf("create role permission failed: %v", err)
	}
	return role
}

func TestCreateAppointmentRejectsMismatchedBodyUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	tomorrow := time.Now().Add(24*time.Hour).Format("2006-01-02") + " 10:00:00"
	body, _ := json.Marshal(map[string]any{
		"card_id":          1,
		"merchant_id":      1,
		"user_id":          2,
		"project_id":       1,
		"appointment_time": tomorrow,
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/user/appointments", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uint(1))

	CreateAppointment(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetCardsReturnsOnlyCurrentUserCards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupSecurityFixTestDB(t)

	m := models.Merchant{Name: "m", Phone: "18800001111", Password: "pwd"}
	u1 := models.User{Username: "u1", Password: "pwd"}
	u2 := models.User{Username: "u2", Password: "pwd"}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	if err := config.DB.Create(&u1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := config.DB.Create(&u2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}
	if err := config.DB.Create(&models.Card{UserID: u1.ID, MerchantID: m.ID, CardNo: "00001", CardType: "A"}).Error; err != nil {
		t.Fatalf("create card1 failed: %v", err)
	}
	if err := config.DB.Create(&models.Card{UserID: u2.ID, MerchantID: m.ID, CardNo: "00002", CardType: "B"}).Error; err != nil {
		t.Fatalf("create card2 failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/cards", nil)
	c.Set("user_id", u1.ID)

	GetCards(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []models.Card `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("want 1 card, got %d", len(resp.Data))
	}
	if resp.Data[0].UserID != u1.ID {
		t.Fatalf("want card user_id=%d, got %d", u1.ID, resp.Data[0].UserID)
	}
}

func TestInternalQueueEndpointDisabledWithoutTokenConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := os.Getenv("KABAO_INTERNAL_TOKEN")
	defer os.Setenv("KABAO_INTERNAL_TOKEN", old)
	_ = os.Unsetenv("KABAO_INTERNAL_TOKEN")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/internal/merchants/1/queue/onsite", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	InternalGetOnsiteQueueSnapshot(c)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateTechnicianAliasRejectsStaffWithoutInfoPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupSecurityFixTestDB(t)

	m := models.Merchant{Name: "m", Phone: "18800001112", Password: "pwd", TechnicianAlias: "技师"}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"technician_alias": "顾问"})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/technician-alias", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth_type", "staff")
	c.Set("merchant_id", m.ID)
	c.Set("service_role_id", uint(999))

	UpdateTechnicianAlias(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMerchantSearchUsersOnlyReturnsMerchantLinkedUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupSecurityFixTestDB(t)

	m := models.Merchant{Name: "m", Phone: "18800001113", Password: "pwd"}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	role := createStaffRoleWithPermission(t, config.DB, m.ID, "merchant.card.issue")

	phone1 := "13900000001"
	phone2 := "13900000002"
	u1 := models.User{Username: "u1", Password: "pwd", Phone: &phone1, Nickname: "linked-card"}
	u2 := models.User{Username: "u2", Password: "pwd", Phone: &phone2, Nickname: "unlinked"}
	if err := config.DB.Create(&u1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := config.DB.Create(&u2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}
	if err := config.DB.Create(&models.Card{UserID: u1.ID, MerchantID: m.ID, CardNo: "00001", CardType: "A"}).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/users/search?phone=139", nil)
	c.Set("auth_type", "staff")
	c.Set("merchant_id", m.ID)
	c.Set("service_role_id", role.ID)

	MerchantSearchUsers(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []struct {
			ID       uint    `json:"id"`
			Phone    *string `json:"phone"`
			Nickname string  `json:"nickname"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("want 1 user, got %d body=%s", len(resp.Data), rec.Body.String())
	}
	if resp.Data[0].ID != u1.ID {
		t.Fatalf("want user_id=%d, got %d", u1.ID, resp.Data[0].ID)
	}
}

func TestGetMerchantsReturnsOnlyCurrentMerchant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupSecurityFixTestDB(t)

	m1 := models.Merchant{Name: "m1", Phone: "18800001114", Password: "pwd"}
	m2 := models.Merchant{Name: "m2", Phone: "18800001115", Password: "pwd"}
	if err := config.DB.Create(&m1).Error; err != nil {
		t.Fatalf("create merchant1 failed: %v", err)
	}
	if err := config.DB.Create(&m2).Error; err != nil {
		t.Fatalf("create merchant2 failed: %v", err)
	}
	role := createStaffRoleWithPermission(t, config.DB, m1.ID, "merchant.info.manage")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/merchants", nil)
	c.Set("auth_type", "staff")
	c.Set("merchant_id", m1.ID)
	c.Set("service_role_id", role.ID)

	GetMerchants(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []models.Merchant `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("want 1 merchant, got %d body=%s", len(resp.Data), rec.Body.String())
	}
	if resp.Data[0].ID != m1.ID {
		t.Fatalf("want merchant_id=%d, got %d", m1.ID, resp.Data[0].ID)
	}
}
