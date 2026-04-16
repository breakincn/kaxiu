package handlers

import (
	"testing"
	"time"

	"kabao/config"
	"kabao/models"
	"kabao/queue"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLiveServiceStatusTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Merchant{},
		&models.ServiceRole{},
		&models.Technician{},
		&models.TechnicianAttendance{},
		&models.Room{},
		&models.MerchantProject{},
		&models.ServiceSession{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestBuildMerchantLiveServiceStatusCustomerServiceMode(t *testing.T) {
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	config.DB = setupLiveServiceStatusTestDB(t)
	queue.Default = nil

	now := time.Date(2026, 3, 26, 15, 0, 0, 0, time.Local)
	role := models.ServiceRole{Name: "专业客服", Key: "therapist_cs", RoleType: "professional"}
	if err := config.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}

	merchant := models.Merchant{
		Name:                       "客服门店",
		Phone:                      "18810000001",
		Password:                   "pwd",
		IsOpen:                     true,
		SupportCustomerService:     true,
		SupportCustomerServiceMode: true,
		SupportTechnicianCheckin:   true,
	}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	tech1 := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "客服1", Code: "cs001", Account: "cs001", Password: "pwd", IsActive: true}
	tech2 := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "客服2", Code: "cs002", Account: "cs002", Password: "pwd", IsActive: true}
	if err := config.DB.Create(&tech1).Error; err != nil {
		t.Fatalf("create tech1 failed: %v", err)
	}
	if err := config.DB.Create(&tech2).Error; err != nil {
		t.Fatalf("create tech2 failed: %v", err)
	}

	checkInAt := now.Add(-2 * time.Hour)
	servingStatus := "busy"
	idleStatus := "idle"
	if err := config.DB.Create(&models.TechnicianAttendance{
		MerchantID:   merchant.ID,
		TechnicianID: tech1.ID,
		CheckedInAt:  &checkInAt,
		Status:       servingStatus,
	}).Error; err != nil {
		t.Fatalf("create att1 failed: %v", err)
	}
	if err := config.DB.Create(&models.TechnicianAttendance{
		MerchantID:   merchant.ID,
		TechnicianID: tech2.ID,
		CheckedInAt:  &checkInAt,
		Status:       idleStatus,
	}).Error; err != nil {
		t.Fatalf("create att2 failed: %v", err)
	}

	startedAt := now.Add(-10 * time.Minute)
	scheduledFinishAt := now.Add(20 * time.Minute)
	createdAt1 := now.Add(-12 * time.Minute)
	if err := config.DB.Create(&models.ServiceSession{
		MerchantID:           merchant.ID,
		SessionMode:          models.SessionModeCustomerService,
		Status:               "serving",
		TechnicianID:         &tech1.ID,
		StartedAt:            &startedAt,
		ScheduledFinishAt:    &scheduledFinishAt,
		DurationMinutes:      30,
		AutoIdleAfterSeconds: 180,
		CreatedAt:            &createdAt1,
		UpdatedAt:            &createdAt1,
	}).Error; err != nil {
		t.Fatalf("create serving session failed: %v", err)
	}

	createdAt2 := now.Add(-3 * time.Minute)
	if err := config.DB.Create(&models.ServiceSession{
		MerchantID:      merchant.ID,
		SessionMode:     models.SessionModeCustomerService,
		Status:          "staff_selecting",
		DurationMinutes: 30,
		CreatedAt:       &createdAt2,
		UpdatedAt:       &createdAt2,
	}).Error; err != nil {
		t.Fatalf("create waiting session failed: %v", err)
	}

	got, err := buildMerchantLiveServiceStatus(&merchant, now)
	if err != nil {
		t.Fatalf("build live status failed: %v", err)
	}

	if !got.Enabled || got.Mode != models.SessionModeCustomerService {
		t.Fatalf("unexpected mode result: %+v", got)
	}
	if got.Counts.ArrivedCount != 2 || got.Counts.WaitingCount != 1 || got.Counts.ServingCount != 1 {
		t.Fatalf("unexpected counts: %+v", got.Counts)
	}
	if got.Counts.CheckedInStaffCount != 2 || got.Counts.ServiceableStaffCount != 2 || got.Counts.BusyStaffCount != 1 || got.Counts.IdleStaffCount != 1 {
		t.Fatalf("unexpected staff counts: %+v", got.Counts)
	}
	if got.Estimate.WaitMinutes == nil || *got.Estimate.WaitMinutes < 10 || *got.Estimate.WaitMinutes > 20 {
		t.Fatalf("unexpected wait estimate: %+v", got.Estimate)
	}
	if got.StatusLevel != "normal" && got.StatusLevel != "busy" {
		t.Fatalf("unexpected status level: %+v", got)
	}
}

func TestBuildMerchantLiveServiceStatusRequiresOpenAttendanceForIdleStaff(t *testing.T) {
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	config.DB = setupLiveServiceStatusTestDB(t)
	queue.Default = nil

	now := time.Date(2026, 3, 26, 15, 30, 0, 0, time.Local)
	role := models.ServiceRole{Name: "专业客服", Key: "hair_stylist", RoleType: "professional"}
	if err := config.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}

	merchant := models.Merchant{
		Name:                       "未签到门店",
		Phone:                      "18810000009",
		Password:                   "pwd",
		IsOpen:                     true,
		SupportCustomerService:     true,
		SupportCustomerServiceMode: true,
		SupportTechnicianCheckin:   false,
	}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	tech := models.Technician{MerchantID: merchant.ID, ServiceRoleID: role.ID, Name: "Tony老师", Code: "sty001", Account: "sty001", Password: "pwd", IsActive: true}
	if err := config.DB.Create(&tech).Error; err != nil {
		t.Fatalf("create tech failed: %v", err)
	}

	got, err := buildMerchantLiveServiceStatus(&merchant, now)
	if err != nil {
		t.Fatalf("build live status failed: %v", err)
	}

	if got.Counts.CheckedInStaffCount != 0 || got.Counts.ServiceableStaffCount != 0 || got.Counts.IdleStaffCount != 0 {
		t.Fatalf("unsigned technician must not be counted as idle staff: %+v", got.Counts)
	}
	if got.StatusLevel != "unavailable" {
		t.Fatalf("unexpected status without signed-in staff: %+v", got)
	}
}

func TestBuildMerchantLiveServiceStatusManualQueueSingle(t *testing.T) {
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	config.DB = setupLiveServiceStatusTestDB(t)

	now := time.Date(2026, 3, 26, 16, 0, 0, 0, time.Local)
	merchant := models.Merchant{
		Name:         "叫号门店",
		Phone:        "18810000002",
		Password:     "pwd",
		IsOpen:       true,
		SupportQueue: true,
		QueueMode:    "manual",
		QueuePrefix:  "A",
	}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	startedAt := now.Add(-10 * time.Minute)
	scheduledFinishAt := now.Add(10 * time.Minute)
	createdAt1 := now.Add(-10 * time.Minute)
	if err := config.DB.Create(&models.ServiceSession{
		MerchantID:        merchant.ID,
		SessionMode:       models.SessionModeQueueManualSingle,
		Status:            "serving",
		InitialUsageID:    2001,
		StartedAt:         &startedAt,
		ScheduledFinishAt: &scheduledFinishAt,
		DurationMinutes:   20,
		CreatedAt:         &createdAt1,
		UpdatedAt:         &createdAt1,
	}).Error; err != nil {
		t.Fatalf("create current queue session failed: %v", err)
	}

	createdAt2 := now.Add(-2 * time.Minute)
	if err := config.DB.Create(&models.ServiceSession{
		MerchantID:      merchant.ID,
		SessionMode:     models.SessionModeQueueManualSingle,
		Status:          "staff_selecting",
		InitialUsageID:  2002,
		DurationMinutes: 15,
		CreatedAt:       &createdAt2,
		UpdatedAt:       &createdAt2,
	}).Error; err != nil {
		t.Fatalf("create pending queue session failed: %v", err)
	}

	calledAt := now.Add(-1 * time.Minute)
	queue.Default = &noopQueueStore{
		snapshot: queue.Snapshot{
			Tickets: []queue.Ticket{
				{ID: 2001, No: 1, EnqueuedAt: createdAt1, CalledAt: &calledAt},
				{ID: 2002, No: 2, EnqueuedAt: createdAt2},
			},
			ByID: map[uint]queue.Ticket{
				2001: {ID: 2001, No: 1, EnqueuedAt: createdAt1, CalledAt: &calledAt},
				2002: {ID: 2002, No: 2, EnqueuedAt: createdAt2},
			},
			CurrentID:       2001,
			CurrentCalledAt: &calledAt,
			MaxCalledNo:     1,
		},
	}

	got, err := buildMerchantLiveServiceStatus(&merchant, now)
	if err != nil {
		t.Fatalf("build live status failed: %v", err)
	}

	if got.Mode != models.SessionModeQueueManualSingle || got.Queue == nil {
		t.Fatalf("unexpected queue result: %+v", got)
	}
	if got.Queue.CurrentCalledNo != 1 || got.Queue.WaitingQueueCount != 1 {
		t.Fatalf("unexpected queue metrics: %+v", got.Queue)
	}
	if got.Counts.WaitingCount != 1 || got.Counts.ServingCount != 1 || got.Counts.CalledCount != 0 {
		t.Fatalf("unexpected counts: %+v", got.Counts)
	}
	if got.Estimate.WaitMinutes == nil || *got.Estimate.WaitMinutes < 25 {
		t.Fatalf("unexpected wait estimate: %+v", got.Estimate)
	}
	if got.Confidence != "high" {
		t.Fatalf("unexpected confidence: %+v", got)
	}
}

func TestBuildMerchantLiveServiceStatusQueuePaused(t *testing.T) {
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	config.DB = setupLiveServiceStatusTestDB(t)
	queue.Default = &noopQueueStore{snapshot: queue.Snapshot{ByID: map[uint]queue.Ticket{}}}

	now := time.Date(2026, 3, 26, 17, 0, 0, 0, time.Local)
	merchant := models.Merchant{
		Name:         "暂停叫号门店",
		Phone:        "18810000003",
		Password:     "pwd",
		IsOpen:       true,
		SupportQueue: true,
		QueueMode:    "manual",
		QueuePaused:  true,
	}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	got, err := buildMerchantLiveServiceStatus(&merchant, now)
	if err != nil {
		t.Fatalf("build live status failed: %v", err)
	}

	if got.StatusLevel != "paused" || got.Estimate.WaitText != "暂停叫号中" || !got.Estimate.SuggestAppointment {
		t.Fatalf("unexpected paused result: %+v", got)
	}
}
