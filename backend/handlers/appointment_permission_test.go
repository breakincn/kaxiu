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

func setupAppointmentPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Merchant{},
		&models.User{},
		&models.MerchantProject{},
		&models.Permission{},
		&models.SystemConfig{},
		&models.ServiceRole{},
		&models.RolePermission{},
		&models.MerchantRolePermissionOverride{},
		&models.Technician{},
		&models.Appointment{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func seedAppointmentPermissionFixture(t *testing.T, db *gorm.DB) (models.Merchant, models.User, models.Technician, models.Technician, models.ServiceRole, models.ServiceRole) {
	t.Helper()

	merchant := models.Merchant{Name: "m", Phone: "18800009999", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	phone := "13900009999"
	user := models.User{Username: "u1", Phone: &phone, Nickname: "u"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	viewRole := models.ServiceRole{MerchantID: &merchant.ID, Key: "m_view", Name: "view role", RoleType: "professional", IsActive: true}
	otherRole := models.ServiceRole{MerchantID: &merchant.ID, Key: "m_other", Name: "other role", RoleType: "professional", IsActive: true}
	if err := db.Create(&viewRole).Error; err != nil {
		t.Fatalf("create view role failed: %v", err)
	}
	if err := db.Create(&otherRole).Error; err != nil {
		t.Fatalf("create other role failed: %v", err)
	}

	tech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: viewRole.ID, Name: "A", Code: "A1", Account: "js0001", Password: "x", IsActive: true}
	otherTech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: otherRole.ID, Name: "B", Code: "B1", Account: "js0002", Password: "x", IsActive: true}
	if err := db.Create(&tech).Error; err != nil {
		t.Fatalf("create tech failed: %v", err)
	}
	if err := db.Create(&otherTech).Error; err != nil {
		t.Fatalf("create other tech failed: %v", err)
	}

	return merchant, user, tech, otherTech, viewRole, otherRole
}

func seedAppointmentPermission(t *testing.T, db *gorm.DB, roleID uint, key string) {
	t.Helper()

	perm := models.Permission{Key: key, Name: key}
	if err := db.Where(models.Permission{Key: key}).FirstOrCreate(&perm).Error; err != nil {
		t.Fatalf("create permission failed: %v", err)
	}

	rp := models.RolePermission{ServiceRoleID: roleID, PermissionID: perm.ID, Allowed: true}
	if err := db.Where(models.RolePermission{ServiceRoleID: roleID, PermissionID: perm.ID}).FirstOrCreate(&rp).Error; err != nil {
		t.Fatalf("create role permission failed: %v", err)
	}
}

func createAppointmentRecord(t *testing.T, db *gorm.DB, merchantID, userID uint, technicianID *uint, status string) models.Appointment {
	t.Helper()
	appt := models.Appointment{
		MerchantID:   merchantID,
		UserID:       userID,
		TechnicianID: technicianID,
		Status:       status,
	}
	if err := db.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	return appt
}

func newStaffContext(method, path string, merchantID, technicianID, roleID uint) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, nil)
	c.Set("auth_type", "staff")
	c.Set("merchant_id", merchantID)
	c.Set("technician_id", technicianID)
	c.Set("service_role_id", roleID)
	return c, rec
}

func TestGetMerchantAppointmentsRejectsStaffWithoutAppointmentPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAppointmentPermissionTestDB(t)

	merchant, user, tech, _, role, _ := seedAppointmentPermissionFixture(t, config.DB)
	createAppointmentRecord(t, config.DB, merchant.ID, user.ID, &tech.ID, "pending")

	c, rec := newStaffContext(http.MethodGet, "/merchant/merchants/"+strconv.Itoa(int(merchant.ID))+"/appointments", merchant.ID, tech.ID, role.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(merchant.ID))}}

	GetMerchantAppointments(c)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetMerchantAppointmentsWithViewPermissionOnlyReturnsOwnAppointments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAppointmentPermissionTestDB(t)

	merchant, user, tech, otherTech, role, _ := seedAppointmentPermissionFixture(t, config.DB)
	own := createAppointmentRecord(t, config.DB, merchant.ID, user.ID, &tech.ID, "pending")
	createAppointmentRecord(t, config.DB, merchant.ID, user.ID, &otherTech.ID, "pending")
	seedAppointmentPermission(t, config.DB, role.ID, "merchant.appointment.view")

	c, rec := newStaffContext(http.MethodGet, "/merchant/merchants/"+strconv.Itoa(int(merchant.ID))+"/appointments", merchant.ID, tech.ID, role.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(merchant.ID))}}

	GetMerchantAppointments(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []models.Appointment `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != own.ID {
		t.Fatalf("want only own appointment %d, got body=%s", own.ID, rec.Body.String())
	}
}

func TestConfirmAppointmentRequiresViewOrManagePermissionForStaff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAppointmentPermissionTestDB(t)

	merchant, user, tech, _, role, _ := seedAppointmentPermissionFixture(t, config.DB)
	appt := createAppointmentRecord(t, config.DB, merchant.ID, user.ID, &tech.ID, "pending")

	c, rec := newStaffContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/confirm", merchant.ID, tech.ID, role.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}

	ConfirmAppointment(c)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestConfirmAppointmentWithViewPermissionOnlyAllowsOwnAppointment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAppointmentPermissionTestDB(t)

	merchant, user, tech, otherTech, role, _ := seedAppointmentPermissionFixture(t, config.DB)
	seedAppointmentPermission(t, config.DB, role.ID, "merchant.appointment.view")

	own := createAppointmentRecord(t, config.DB, merchant.ID, user.ID, &tech.ID, "pending")
	other := createAppointmentRecord(t, config.DB, merchant.ID, user.ID, &otherTech.ID, "pending")

	c, rec := newStaffContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(own.ID))+"/confirm", merchant.ID, tech.ID, role.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(own.ID))}}
	ConfirmAppointment(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 after passing permission gate, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "预约时间为空") {
		t.Fatalf("want missing appointment time after permission passes, got body=%s", rec.Body.String())
	}

	c, rec = newStaffContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(other.ID))+"/confirm", merchant.ID, tech.ID, role.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(other.ID))}}
	ConfirmAppointment(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 for other appointment, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCancelAppointmentWithViewPermissionOnlyAllowsOwnAppointment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAppointmentPermissionTestDB(t)

	merchant, user, tech, otherTech, role, _ := seedAppointmentPermissionFixture(t, config.DB)
	seedAppointmentPermission(t, config.DB, role.ID, "merchant.appointment.view")

	own := createAppointmentRecord(t, config.DB, merchant.ID, user.ID, &tech.ID, "pending")
	other := createAppointmentRecord(t, config.DB, merchant.ID, user.ID, &otherTech.ID, "pending")

	c, rec := newStaffContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(own.ID))+"/cancel", merchant.ID, tech.ID, role.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(own.ID))}}
	CancelAppointment(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 for own appointment cancel, got %d body=%s", rec.Code, rec.Body.String())
	}

	c, rec = newStaffContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(other.ID))+"/cancel", merchant.ID, tech.ID, role.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(other.ID))}}
	CancelAppointment(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 for other appointment cancel, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetAvailableTimeSlotsAllowsUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAppointmentPermissionTestDB(t)

	merchant := models.Merchant{
		Name:               "m",
		Phone:              "18800009998",
		Password:           "pwd",
		SupportAppointment: true,
		AllDayStart:        "10:00",
		AllDayEnd:          "00:00",
	}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	phone := "13900009998"
	user := models.User{Username: "u2", Phone: &phone, Nickname: "u2"}
	if err := config.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID: merchant.ID,
		Name:       "塑形私教课",
		Duration:   45,
		IsActive:   true,
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location failed: %v", err)
	}
	date := time.Now().In(loc).Add(24 * time.Hour).Format("2006-01-02")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/merchants/"+strconv.Itoa(int(merchant.ID))+"/available-slots?date="+date+"&project_id="+strconv.Itoa(int(project.ID)), nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(merchant.ID))}}
	c.Set("auth_type", "user")
	c.Set("user_id", user.ID)

	GetAvailableTimeSlots(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			TimeSlots []struct {
				Time string `json:"time"`
			} `json:"time_slots"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v body=%s", err, rec.Body.String())
	}
	if len(resp.Data.TimeSlots) == 0 {
		t.Fatalf("want non-empty time slots, got body=%s", rec.Body.String())
	}
}
