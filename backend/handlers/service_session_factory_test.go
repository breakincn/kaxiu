package handlers

import (
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

func setupServiceSessionFactoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsnName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dsnName+"?mode=memory&cache=shared&_loc=auto&parseTime=true"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.ServiceRole{}, &models.Technician{}, &models.TechnicianAttendance{}, &models.Room{}, &models.Card{}, &models.VerifyCode{}, &models.Usage{}, &models.ServiceSession{}, &models.Appointment{}, &models.MerchantProject{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestChooseServiceSessionRoomSkipsStaffSelectWithProjectDefaultTechnicians(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionFactoryTestDB(t)

	now := time.Now()
	merchant := models.Merchant{Name: "m", Phone: "18800000004", Password: "pwd", SupportCustomerServiceMode: true, SupportRoom: true}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	role := models.ServiceRole{RoleType: "professional", Key: "teacher", Name: "专业客服", IsActive: true}
	if err := config.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	firstTech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "朱古丽", Code: "0001", Account: "jsls0001", IsActive: true}
	secondTech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "671", Code: "0002", Account: "bls0001", IsActive: true}
	if err := config.DB.Create(&firstTech).Error; err != nil {
		t.Fatalf("create first technician failed: %v", err)
	}
	if err := config.DB.Create(&secondTech).Error; err != nil {
		t.Fatalf("create second technician failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:                  merchant.ID,
		Name:                        "爵士舞课",
		Duration:                    60,
		ServiceCapacity:             1,
		DefaultServiceTechnicianIDs: models.MerchantProjectDefaultServiceTechnicianIDs{firstTech.ID, secondTech.ID},
		ServiceTimeSlots:            models.MerchantProjectServiceTimeSlots{},
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	room := models.Room{MerchantID: merchant.ID, Name: "A101", IsActive: true}
	if err := config.DB.Create(&room).Error; err != nil {
		t.Fatalf("create room failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:                 merchant.ID,
		ProjectID:                  &project.ID,
		SessionMode:                models.SessionModeCustomerService,
		Status:                     "cs_room_selecting",
		ServiceTechnicianIDs:       models.MerchantProjectDefaultServiceTechnicianIDs{firstTech.ID, secondTech.ID},
		CreatedAt:                  &now,
		UpdatedAt:                  &now,
		InitialUsageID:             1,
		StartPendingTimeoutSeconds: 0,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("merchant_id", merchant.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(session.ID), 10)}}
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/service-sessions/"+strconv.FormatUint(uint64(session.ID), 10)+"/room", strings.NewReader(`{"room_id":`+strconv.FormatUint(uint64(room.ID), 10)+`}`))
	c.Request.Header.Set("Content-Type", "application/json")

	ChooseServiceSessionRoom(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var got models.ServiceSession
	if err := config.DB.Select("id", "room_id", "technician_id", "last_technician_id", "service_technician_ids", "status").First(&got, session.ID).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if got.Status != "cs_start_pending" {
		t.Fatalf("want cs_start_pending, got %s", got.Status)
	}
	if got.TechnicianID == nil || *got.TechnicianID != firstTech.ID {
		t.Fatalf("want first technician %d for compatibility field, got %+v", firstTech.ID, got.TechnicianID)
	}
	if len(got.ServiceTechnicianIDs) != 2 || got.ServiceTechnicianIDs[0] != firstTech.ID || got.ServiceTechnicianIDs[1] != secondTech.ID {
		t.Fatalf("want bound service technicians [%d %d], got %+v", firstTech.ID, secondTech.ID, got.ServiceTechnicianIDs)
	}
}

func TestServiceSessionStartScanAllowsAnyProjectDefaultProfessionalTechnician(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionFactoryTestDB(t)

	now := time.Now()
	merchant := models.Merchant{
		Name:                       "m",
		Phone:                      "18800000003",
		Password:                   "pwd",
		SupportCustomerServiceMode: true,
	}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	role := models.ServiceRole{RoleType: "professional", Key: "teacher", Name: "专业客服", IsActive: true}
	if err := config.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	firstTech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "朱古丽", Code: "0001", Account: "jsls0001", IsActive: true}
	secondTech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "671", Code: "0002", Account: "bls0001", IsActive: true}
	if err := config.DB.Create(&firstTech).Error; err != nil {
		t.Fatalf("create first technician failed: %v", err)
	}
	if err := config.DB.Create(&secondTech).Error; err != nil {
		t.Fatalf("create second technician failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:                  merchant.ID,
		Name:                        "爵士舞课",
		Duration:                    60,
		ServiceCapacity:             1,
		DefaultServiceTechnicianIDs: models.MerchantProjectDefaultServiceTechnicianIDs{firstTech.ID, secondTech.ID},
		ServiceTimeSlots:            models.MerchantProjectServiceTimeSlots{},
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	usage := models.Usage{MerchantID: merchant.ID, ProjectID: &project.ID, Status: "in_progress"}
	if err := config.DB.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	if err := config.DB.Create(&models.TechnicianAttendance{
		MerchantID:   merchant.ID,
		TechnicianID: secondTech.ID,
		CheckedInAt:  &now,
		Status:       "idle",
	}).Error; err != nil {
		t.Fatalf("create attendance failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:     merchant.ID,
		ProjectID:      &project.ID,
		InitialUsageID: usage.ID,
		SessionMode:    models.SessionModeCustomerService,
		TechnicianID:   &firstTech.ID,
		Status:         "cs_start_pending",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("merchant_id", merchant.ID)
	c.Set("auth_type", "staff")
	c.Set("technician_id", secondTech.ID)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/verify", strings.NewReader(`{"code":"SS:"}`))

	if handled := handleServiceSessionStartScan(c, "SS:"+strconv.FormatUint(uint64(session.ID), 10)); !handled {
		t.Fatalf("want start scan handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var gotSession models.ServiceSession
	if err := config.DB.Select("id", "technician_id", "last_technician_id", "service_technician_ids", "status").First(&gotSession, session.ID).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotSession.TechnicianID == nil || *gotSession.TechnicianID != secondTech.ID {
		t.Fatalf("want scanner technician %d, got %+v", secondTech.ID, gotSession.TechnicianID)
	}
	if gotSession.LastTechnicianID == nil || *gotSession.LastTechnicianID != secondTech.ID {
		t.Fatalf("want last technician %d, got %+v", secondTech.ID, gotSession.LastTechnicianID)
	}
	if gotSession.Status != "cs_delay_pending" {
		t.Fatalf("want cs_delay_pending, got %s", gotSession.Status)
	}
	if len(gotSession.ServiceTechnicianIDs) != 2 || gotSession.ServiceTechnicianIDs[0] != firstTech.ID || gotSession.ServiceTechnicianIDs[1] != secondTech.ID {
		t.Fatalf("want bound service technicians [%d %d], got %+v", firstTech.ID, secondTech.ID, gotSession.ServiceTechnicianIDs)
	}
	var gotUsage models.Usage
	if err := config.DB.First(&gotUsage, usage.ID).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotUsage.TechnicianID == nil || *gotUsage.TechnicianID != secondTech.ID {
		t.Fatalf("want usage technician %d, got %+v", secondTech.ID, gotUsage.TechnicianID)
	}
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
	secondTech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "老师B", Code: "0002", Account: "pro0002", IsActive: true}
	if err := config.DB.Create(&secondTech).Error; err != nil {
		t.Fatalf("create second technician failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:                  merchant.ID,
		Name:                        "团课",
		Duration:                    60,
		ServiceCapacity:             1,
		DefaultServiceTechnicianIDs: models.MerchantProjectDefaultServiceTechnicianIDs{tech.ID, secondTech.ID},
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
	if len(session.ServiceTechnicianIDs) != 2 || session.ServiceTechnicianIDs[0] != tech.ID || session.ServiceTechnicianIDs[1] != secondTech.ID {
		t.Fatalf("want bound service technicians [%d %d], got %+v", tech.ID, secondTech.ID, session.ServiceTechnicianIDs)
	}
	if session.Status != "cs_start_pending" {
		t.Fatalf("want cs_start_pending, got %s", session.Status)
	}
	if session.StartPendingTimeoutSeconds != 240 {
		t.Fatalf("want project start timeout 240, got %d", session.StartPendingTimeoutSeconds)
	}
}

func TestPerformVerifyCommitUsesProjectServiceTimeForDefaultTechnicians(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionFactoryTestDB(t)

	loc := config.ProjectServiceTimeLocation()
	now := time.Now().In(loc)
	slotStart := now.Add(30 * time.Minute).Truncate(time.Minute)
	weekday := int(slotStart.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	merchant := models.Merchant{Name: "m-service-time", Phone: "18800000006", Password: "pwd", SupportCustomerServiceMode: true}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	role := models.ServiceRole{RoleType: "professional", Key: "teacher", Name: "专业客服", IsActive: true}
	if err := config.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	firstTech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "朱古丽", Code: "0001", Account: "jsls0001", IsActive: true}
	secondTech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "671", Code: "0002", Account: "bls0001", IsActive: true}
	if err := config.DB.Create(&firstTech).Error; err != nil {
		t.Fatalf("create first technician failed: %v", err)
	}
	if err := config.DB.Create(&secondTech).Error; err != nil {
		t.Fatalf("create second technician failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:                  merchant.ID,
		Name:                        "爵士舞课",
		Duration:                    60,
		ServiceCapacity:             1,
		DefaultServiceTechnicianIDs: models.MerchantProjectDefaultServiceTechnicianIDs{firstTech.ID, secondTech.ID},
		StartPendingTimeoutSeconds:  300,
		ServiceTimeSlots: models.MerchantProjectServiceTimeSlots{{
			RecurrenceType: "weekly",
			Weekday:        weekday,
			StartTime:      slotStart.Format("15:04"),
		}},
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	card := models.Card{UserID: 9, MerchantID: merchant.ID, CardNo: "00003", CardType: "times", TotalTimes: 10, RemainTimes: 10, RechargeAt: &now, LastUsedAt: &now}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	verifyCode := models.VerifyCode{CardID: card.ID, Code: "VERIFY-SERVICE-TIME", ExpireAt: now.Add(10 * time.Minute).Unix(), ProjectID: &project.ID}
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

	var session struct {
		Status                     string
		ScheduledStartAt           string `gorm:"column:scheduled_start_at"`
		StartPendingTimeoutSeconds int    `gorm:"column:start_pending_timeout_seconds"`
		ServiceTechnicianIDs       models.MerchantProjectDefaultServiceTechnicianIDs
	}
	if err := config.DB.Model(&models.ServiceSession{}).
		Select("status", "scheduled_start_at", "start_pending_timeout_seconds", "service_technician_ids").
		Where("id = ?", result.SessionID).
		First(&session).Error; err != nil {
		t.Fatalf("load service session failed: %v", err)
	}
	if session.Status != "cs_start_pending" {
		t.Fatalf("want cs_start_pending, got %s", session.Status)
	}
	gotStart, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(session.ScheduledStartAt), loc)
	if err != nil {
		gotStart, err = time.ParseInLocation("2006-01-02 15:04:05-07:00", strings.TrimSpace(session.ScheduledStartAt), loc)
	}
	if err != nil || !gotStart.Equal(slotStart) {
		t.Fatalf("want scheduled start %s, got %q err=%v", slotStart, session.ScheduledStartAt, err)
	}
	if session.StartPendingTimeoutSeconds != 0 {
		t.Fatalf("want no start pending timeout for project service time, got %d", session.StartPendingTimeoutSeconds)
	}
	if len(session.ServiceTechnicianIDs) != 2 {
		t.Fatalf("want both service technicians, got %+v", session.ServiceTechnicianIDs)
	}
}
