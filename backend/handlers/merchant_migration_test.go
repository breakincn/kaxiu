package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMerchantHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.ServiceSession{}, &models.TechnicianAttendance{}, &models.ServiceSessionExtendRequest{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestUpdateCurrentMerchantServices_DoesNotMigrateOnHandCardToggle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                       "m",
		Phone:                      "18800001001",
		Password:                   "pwd",
		SupportCustomerService:     true,
		SupportCustomerServiceMode: true,
		SupportHandCard:            false,
		SupportRoom:                false,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	s := models.ServiceSession{
		MerchantID:       m.ID,
		InitialUsageID:   100,
		SessionMode:      models.SessionModeCustomerService,
		Status:           "cs_start_pending",
		StartConfirmedAt: nil,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_hand_card": true,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got models.ServiceSession
	if err := config.DB.First(&got, s.ID).Error; err != nil {
		t.Fatalf("load session failed: %v", err)
	}
	if models.NormalizeSessionStatus(got.Status) != "start_pending" {
		t.Fatalf("status should stay start_pending, got %s", got.Status)
	}
	// sqlite 下 datetime(3) 字段可能以 string 形式返回，避免直接断言 time
	var row struct {
		StartConfirmedAtRaw string `gorm:"column:start_confirmed_at"`
	}
	if err := config.DB.Table("service_sessions").Select("start_confirmed_at").Where("id = ?", s.ID).Scan(&row).Error; err != nil {
		t.Fatalf("scan start_confirmed_at failed: %v", err)
	}
	if row.StartConfirmedAtRaw != "" {
		t.Fatalf("start_confirmed_at should stay empty, got %s", row.StartConfirmedAtRaw)
	}
}

func TestUpdateCurrentMerchantServices_StoresAppointmentSchedulingMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                       "m",
		Phone:                      "18800001014",
		Password:                   "pwd",
		SupportAppointment:         true,
		SupportCustomerService:     true,
		SupportCustomerServiceMode: true,
		AppointmentSchedulingMode:  appointmentSchedulingModeGrouped,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"appointment_scheduling_mode": appointmentSchedulingModeMixedTimeline,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got models.Merchant
	if err := config.DB.First(&got, m.ID).Error; err != nil {
		t.Fatalf("load merchant failed: %v", err)
	}
	if got.AppointmentSchedulingMode != appointmentSchedulingModeMixedTimeline {
		t.Fatalf("want mixed timeline, got %s", got.AppointmentSchedulingMode)
	}
}

func TestUpdateCurrentMerchantServices_RejectsMixedTimelineWithoutCustomerServiceMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                       "m",
		Phone:                      "18800001015",
		Password:                   "pwd",
		SupportAppointment:         true,
		SupportCustomerService:     true,
		SupportCustomerServiceMode: false,
		AppointmentSchedulingMode:  appointmentSchedulingModeGrouped,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"appointment_scheduling_mode": appointmentSchedulingModeMixedTimeline,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCurrentMerchantServices_MigratesOnlyOnCustomerServiceModeDisable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                       "m",
		Phone:                      "18800001002",
		Password:                   "pwd",
		SupportCustomerService:     true,
		SupportCustomerServiceMode: true,
		SupportRoom:                false,
		StartDelaySeconds:          60,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	s := models.ServiceSession{
		MerchantID:       m.ID,
		InitialUsageID:   101,
		SessionMode:      models.SessionModeCustomerService,
		Status:           "cs_start_pending",
		StartConfirmedAt: nil,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_customer_service_mode": false,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var row struct {
		Status              string `gorm:"column:status"`
		StartConfirmedAtRaw string `gorm:"column:start_confirmed_at"`
	}
	if err := config.DB.Table("service_sessions").
		Select("status, start_confirmed_at").
		Where("id = ?", s.ID).
		Scan(&row).Error; err != nil {
		t.Fatalf("load session failed: %v", err)
	}
	if models.NormalizeSessionStatus(row.Status) != "delay_pending" {
		t.Fatalf("want migrated status delay_pending, got %s", row.Status)
	}
	if row.StartConfirmedAtRaw == "" {
		t.Fatalf("start_confirmed_at should be set")
	}
}

func TestUpdateCurrentMerchantServices_DoesNotDowngradeQueueModeWhenSupportOrderCompleteUnchanged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                 "m",
		Phone:                "18800001003",
		Password:             "pwd",
		SupportQueue:         true,
		QueueMode:            "auto",
		SupportOrderComplete: false,
		SupportHandCard:      false,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_hand_card":      true,
		"support_order_complete": false,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got models.Merchant
	if err := config.DB.First(&got, m.ID).Error; err != nil {
		t.Fatalf("load merchant failed: %v", err)
	}
	if got.QueueMode != "auto" {
		t.Fatalf("queue_mode should stay auto, got %s", got.QueueMode)
	}
}

func TestUpdateCurrentMerchantServices_DowngradesQueueModeWhenSupportOrderCompleteTurnsOff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                 "m",
		Phone:                "18800001004",
		Password:             "pwd",
		SupportQueue:         true,
		QueueMode:            "auto",
		SupportOrderComplete: true,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_order_complete": false,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got models.Merchant
	if err := config.DB.First(&got, m.ID).Error; err != nil {
		t.Fatalf("load merchant failed: %v", err)
	}
	if got.QueueMode != "manual" {
		t.Fatalf("queue_mode should become manual, got %s", got.QueueMode)
	}
}

func TestUpdateCurrentMerchantServices_RejectsQueueAndCustomerServiceModeConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                       "m",
		Phone:                      "18800001005",
		Password:                   "pwd",
		SupportQueue:               false,
		SupportCustomerService:     true,
		SupportCustomerServiceMode: false,
		QueueMode:                  "manual",
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_queue":                 true,
		"support_customer_service_mode": true,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCurrentMerchantServices_RejectsDisablingQueueWithActiveQueueSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:         "m",
		Phone:        "18800001006",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	s := models.ServiceSession{
		MerchantID:       m.ID,
		InitialUsageID:   201,
		SessionMode:      models.SessionModeQueueManualSingle,
		Status:           "qms_serving",
		StartConfirmedAt: &now,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_queue": false,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCurrentMerchantServices_RejectsSwitchingToCustomerServiceModeWithActiveQueueSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                       "m",
		Phone:                      "18800001007",
		Password:                   "pwd",
		SupportQueue:               true,
		QueueMode:                  "manual",
		SupportCustomerService:     true,
		SupportCustomerServiceMode: false,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	s := models.ServiceSession{
		MerchantID:       m.ID,
		InitialUsageID:   202,
		SessionMode:      models.SessionModeQueueManualSingle,
		Status:           "qms_delay_pending",
		StartConfirmedAt: &now,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_queue":                 false,
		"support_customer_service_mode": true,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCurrentMerchantServices_RejectsDisablingQueueWithRoomSelectingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:         "m",
		Phone:        "18800001008",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	s := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: 203,
		SessionMode:    models.SessionModeQueueManualSingle,
		Status:         "qms_room_selecting",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_queue": false,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCurrentMerchantServices_RejectsDisablingQueueWithTimeoutWaitingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:         "m",
		Phone:        "18800001009",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	s := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: 204,
		SessionMode:    models.SessionModeQueueManualSingle,
		Status:         "qms_timeout_waiting",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_queue": false,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCurrentMerchantServices_RejectsSwitchingToCustomerServiceModeWithRoomLockedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                       "m",
		Phone:                      "18800001010",
		Password:                   "pwd",
		SupportQueue:               true,
		QueueMode:                  "manual",
		SupportCustomerService:     true,
		SupportCustomerServiceMode: false,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	roomID := uint(7)
	s := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: 205,
		SessionMode:    models.SessionModeQueueManualSingle,
		Status:         "qms_room_locked",
		RoomID:         &roomID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_queue":                 false,
		"support_customer_service_mode": true,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCurrentMerchantServices_RejectsSwitchingToCustomerServiceModeWithStartPendingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                       "m",
		Phone:                      "18800001011",
		Password:                   "pwd",
		SupportQueue:               true,
		QueueMode:                  "manual",
		SupportCustomerService:     true,
		SupportCustomerServiceMode: false,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	techID := uint(9)
	s := models.ServiceSession{
		MerchantID:       m.ID,
		InitialUsageID:   206,
		SessionMode:      models.SessionModeQueueManualSingle,
		Status:           "qms_start_pending",
		TechnicianID:     &techID,
		StartConfirmedAt: nil,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_queue":                 false,
		"support_customer_service_mode": true,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCurrentMerchantServices_RejectsSwitchingToCustomerServiceModeWithActiveMultiQueueSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                        "m",
		Phone:                       "18800001012",
		Password:                    "pwd",
		SupportQueue:                true,
		QueueMode:                   "auto",
		SupportMultiCustomerService: true,
		SupportCustomerService:      true,
		SupportCustomerServiceMode:  false,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	techID := uint(10)
	s := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: 207,
		SessionMode:    models.SessionModeQueueAutoMulti,
		Status:         "qm_start_pending",
		TechnicianID:   &techID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_queue":                 false,
		"support_customer_service_mode": true,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCurrentMerchantServices_AllowsSwitchingToCustomerServiceModeWhenAllSessionsTerminal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMerchantHandlerTestDB(t)

	m := models.Merchant{
		Name:                       "m",
		Phone:                      "18800001013",
		Password:                   "pwd",
		SupportQueue:               true,
		QueueMode:                  "manual",
		SupportCustomerService:     true,
		SupportCustomerServiceMode: false,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	finished := models.ServiceSession{
		MerchantID:       m.ID,
		InitialUsageID:   208,
		SessionMode:      models.SessionModeQueueManualSingle,
		Status:           "qms_finished",
		StartConfirmedAt: &now,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	timeoutFailed := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: 209,
		SessionMode:    models.SessionModeQueueManualSingle,
		Status:         "qms_timeout_failed",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&finished).Error; err != nil {
		t.Fatalf("create finished session failed: %v", err)
	}
	if err := config.DB.Create(&timeoutFailed).Error; err != nil {
		t.Fatalf("create timeout_failed session failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"support_queue":                 false,
		"support_customer_service_mode": true,
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/services", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", m.ID)

	UpdateCurrentMerchantServices(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got models.Merchant
	if err := config.DB.First(&got, m.ID).Error; err != nil {
		t.Fatalf("load merchant failed: %v", err)
	}
	if got.SupportQueue {
		t.Fatalf("support_queue should be false after switch")
	}
	if !got.SupportCustomerServiceMode {
		t.Fatalf("support_customer_service_mode should be true after switch")
	}
}
