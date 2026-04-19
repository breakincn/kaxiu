package handlers

import (
	"net/http/httptest"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupServiceSessionFactoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:handlers_service_session_factory_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.ServiceRole{}, &models.Technician{}, &models.Card{}, &models.VerifyCode{}, &models.Usage{}, &models.ServiceSession{}, &models.Appointment{}, &models.MerchantProject{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestPerformVerifyCommitCreatesAppointmentSourceSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionFactoryTestDB(t)

	now := time.Now()
	merchant := models.Merchant{
		Name:         "m",
		Phone:        "18800000001",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "auto",
	}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	card := models.Card{
		UserID:         7,
		MerchantID:     merchant.ID,
		CardNo:         "00001",
		CardType:       "times",
		TotalTimes:     10,
		RemainTimes:    10,
		RechargeAt:     &now,
		LastUsedAt:     &now,
		RechargeAmount: 0,
	}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	verifyCode := models.VerifyCode{
		CardID:   card.ID,
		Code:     "VERIFY-APPOINTMENT",
		ExpireAt: now.Add(10 * time.Minute).Unix(),
	}
	if err := config.DB.Create(&verifyCode).Error; err != nil {
		t.Fatalf("create verify code failed: %v", err)
	}
	apptTime := now.Add(-5 * time.Minute)
	appointment := models.Appointment{
		CardID:          card.ID,
		MerchantID:      merchant.ID,
		UserID:          card.UserID,
		AppointmentTime: &apptTime,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appointment).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("auth_type", "merchant")

	result := verifyCommitResult{}
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = performVerifyCommit(tx, ctx, merchant, verifyCode, card, "")
		return err
	}); err != nil {
		t.Fatalf("performVerifyCommit failed: %v", err)
	}

	var session models.ServiceSession
	if err := config.DB.First(&session, result.SessionID).Error; err != nil {
		t.Fatalf("load service session failed: %v", err)
	}
	if session.SourceType != serviceSessionSourceAppointment {
		t.Fatalf("want source_type=%s, got %s", serviceSessionSourceAppointment, session.SourceType)
	}
	if session.SourceID == nil || *session.SourceID != appointment.ID {
		t.Fatalf("want source_id=%d, got %+v", appointment.ID, session.SourceID)
	}
}

func TestPerformVerifyCommitBindsProjectDefaultProfessionalTechnician(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionFactoryTestDB(t)

	now := time.Now()
	merchant := models.Merchant{
		Name:                       "m",
		Phone:                      "18800000002",
		Password:                   "pwd",
		SupportCustomerServiceMode: true,
	}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	role := models.ServiceRole{RoleType: "professional", Key: "teacher", Name: "专业客服", IsActive: true, StartPendingTimeoutSeconds: 180}
	if err := config.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	tech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "老师A", Code: "0001", Account: "pro0001", IsActive: true}
	if err := config.DB.Create(&tech).Error; err != nil {
		t.Fatalf("create technician failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:                  merchant.ID,
		Name:                        "团课",
		Duration:                    60,
		ServiceCapacity:             15,
		DefaultServiceTechnicianIDs: models.MerchantProjectDefaultServiceTechnicianIDs{tech.ID},
		StartPendingTimeoutSeconds:  240,
		ServiceTimeSlots:            models.MerchantProjectServiceTimeSlots{},
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	card := models.Card{
		UserID:         8,
		MerchantID:     merchant.ID,
		CardNo:         "00002",
		CardType:       "times",
		TotalTimes:     10,
		RemainTimes:    10,
		RechargeAt:     &now,
		LastUsedAt:     &now,
		RechargeAmount: 0,
	}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	verifyCode := models.VerifyCode{
		CardID:    card.ID,
		Code:      "VERIFY-GROUP",
		ExpireAt:  now.Add(10 * time.Minute).Unix(),
		ProjectID: &project.ID,
	}
	if err := config.DB.Create(&verifyCode).Error; err != nil {
		t.Fatalf("create verify code failed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("auth_type", "merchant")

	result := verifyCommitResult{}
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = performVerifyCommit(tx, ctx, merchant, verifyCode, card, "")
		return err
	}); err != nil {
		t.Fatalf("performVerifyCommit failed: %v", err)
	}

	if result.NextStep != "" || result.BoundTechnicianID != tech.ID {
		t.Fatalf("want bound technician without next step, got next=%q bound=%d", result.NextStep, result.BoundTechnicianID)
	}
	var session models.ServiceSession
	if err := config.DB.First(&session, result.SessionID).Error; err != nil {
		t.Fatalf("load service session failed: %v", err)
	}
	if session.TechnicianID == nil || *session.TechnicianID != tech.ID {
		t.Fatalf("want technician %d, got %+v", tech.ID, session.TechnicianID)
	}
	if session.Status != "cs_start_pending" {
		t.Fatalf("want cs_start_pending, got %s", session.Status)
	}
	if session.StartPendingTimeoutSeconds != 240 {
		t.Fatalf("want project start timeout 240, got %d", session.StartPendingTimeoutSeconds)
	}
}
