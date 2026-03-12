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
	"kabao/queue"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:handlers_queue_status_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Merchant{},
		&models.User{},
		&models.MerchantProject{},
		&models.ServiceRole{},
		&models.Technician{},
		&models.TechnicianAttendance{},
		&models.Usage{},
		&models.ServiceSession{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestTriggerNextCallingRejectsAutoMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupHandlerTestDB(t)
	m := models.Merchant{
		Name:         "m-auto",
		Phone:        "18800000001",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "auto",
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/queue/call-next", nil)
	c.Set("merchant_id", m.ID)

	TriggerNextCalling(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "当前非人工叫号模式") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestPromoteManualSingleCalledSessionDefaultsDelaySeconds(t *testing.T) {
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	config.DB = setupHandlerTestDB(t)
	m := models.Merchant{
		Name:                        "m-manual",
		Phone:                       "18800000002",
		Password:                    "pwd",
		SupportQueue:                true,
		QueueMode:                   "manual",
		SupportMultiCustomerService: false,
		StartDelaySeconds:           0,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	now := time.Now()
	s := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: 1001,
		Status:         "staff_selecting",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		promoteManualSingleCalledSession(tx, &m, s.InitialUsageID, now)
		return nil
	}); err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	var got struct {
		StartDelaySeconds int
		Status            string
		DiffSeconds       int
	}
	if err := config.DB.
		Table("service_sessions").
		Select("start_delay_seconds, status, CAST(strftime('%s', scheduled_start_at) - strftime('%s', start_confirmed_at) AS INTEGER) AS diff_seconds").
		Where("id = ?", s.ID).
		Scan(&got).Error; err != nil {
		t.Fatalf("query promoted session failed: %v", err)
	}

	if got.StartDelaySeconds != 60 {
		t.Fatalf("want start_delay_seconds=60, got %d", got.StartDelaySeconds)
	}
	if models.NormalizeSessionStatus(got.Status) != "delay_pending" {
		t.Fatalf("want status delay_pending, got %s", got.Status)
	}
	if got.DiffSeconds < 59 || got.DiffSeconds > 61 {
		t.Fatalf("want scheduled_start_at-start_confirmed_at around 60s, got %d", got.DiffSeconds)
	}
}

func TestShouldExcludeFromPendingByStatus(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{status: "finished", want: true},
		{status: "canceled", want: true},
		{status: "timeout_waiting", want: true},
		{status: "timeout_failed", want: true},
		{status: "staff_selecting", want: false},
	}
	for _, tc := range cases {
		if got := shouldExcludeFromPendingByStatus(tc.status); got != tc.want {
			t.Fatalf("status=%s want=%v got=%v", tc.status, tc.want, got)
		}
	}
}

func TestTriggerNextCallingManualModeResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	config.DB = setupHandlerTestDB(t)
	m := models.Merchant{
		Name:         "m-manual-2",
		Phone:        "18800000003",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	queue.Default = &noopQueueStore{}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/queue/call-next", nil)
	c.Set("merchant_id", m.ID)

	TriggerNextCalling(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("want data field, got %v", body)
	}
}

func TestGetQueuePendingListStaffOnlySeesOwnSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	config.DB = setupHandlerTestDB(t)
	now := time.Now()

	m := models.Merchant{
		Name:         "m-queue-pending",
		Phone:        "18800000004",
		Password:     "pwd",
		SupportQueue: true,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	user1 := models.User{Username: "u1", Nickname: "甲"}
	user2 := models.User{Username: "u2", Nickname: "乙"}
	if err := config.DB.Create(&user1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := config.DB.Create(&user2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}

	tech1 := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服1", Code: "0001", Account: "js0001", Password: "pwd", IsActive: true}
	tech2 := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服2", Code: "0002", Account: "js0002", Password: "pwd", IsActive: true}
	if err := config.DB.Create(&tech1).Error; err != nil {
		t.Fatalf("create tech1 failed: %v", err)
	}
	if err := config.DB.Create(&tech2).Error; err != nil {
		t.Fatalf("create tech2 failed: %v", err)
	}

	usage1 := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	usage2 := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := config.DB.Create(&usage1).Error; err != nil {
		t.Fatalf("create usage1 failed: %v", err)
	}
	if err := config.DB.Create(&usage2).Error; err != nil {
		t.Fatalf("create usage2 failed: %v", err)
	}

	sess1 := models.ServiceSession{
		MerchantID:                 m.ID,
		UserID:                     user1.ID,
		InitialUsageID:             usage1.ID,
		TechnicianID:               &tech1.ID,
		Status:                     "start_pending",
		StartPendingTimeoutSeconds: 180,
		CreatedAt:                  &now,
		UpdatedAt:                  &now,
	}
	sess2 := models.ServiceSession{
		MerchantID:                 m.ID,
		UserID:                     user2.ID,
		InitialUsageID:             usage2.ID,
		TechnicianID:               &tech2.ID,
		Status:                     "serving",
		StartPendingTimeoutSeconds: 180,
		CreatedAt:                  &now,
		UpdatedAt:                  &now,
	}
	if err := config.DB.Create(&sess1).Error; err != nil {
		t.Fatalf("create sess1 failed: %v", err)
	}
	if err := config.DB.Create(&sess2).Error; err != nil {
		t.Fatalf("create sess2 failed: %v", err)
	}

	queue.Default = &noopQueueStore{
		snapshot: queue.Snapshot{
			Tickets: []queue.Ticket{
				{ID: usage1.ID, No: 6},
				{ID: usage2.ID, No: 7},
			},
		},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/queue/pending-list", nil)
	c.Set("merchant_id", m.ID)
	c.Set("auth_type", "staff")
	c.Set("technician_id", tech1.ID)

	GetQueuePendingList(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Data []queuePendingItem `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("want 1 item, got %d body=%s", len(body.Data), rec.Body.String())
	}
	if body.Data[0].UsageID != usage1.ID {
		t.Fatalf("want only usage %d, got %+v", usage1.ID, body.Data)
	}
}

func TestGetQueueTimeoutWaitingListStaffOnlySeesOwnSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupHandlerTestDB(t)
	now := time.Now()

	m := models.Merchant{
		Name:         "m-queue-timeout",
		Phone:        "18800000005",
		Password:     "pwd",
		SupportQueue: true,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	user1 := models.User{Username: "u3", Nickname: "丙"}
	user2 := models.User{Username: "u4", Nickname: "丁"}
	if err := config.DB.Create(&user1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := config.DB.Create(&user2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}

	tech1 := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服1", Code: "0001", Account: "js0101", Password: "pwd", IsActive: true}
	tech2 := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服2", Code: "0002", Account: "js0102", Password: "pwd", IsActive: true}
	if err := config.DB.Create(&tech1).Error; err != nil {
		t.Fatalf("create tech1 failed: %v", err)
	}
	if err := config.DB.Create(&tech2).Error; err != nil {
		t.Fatalf("create tech2 failed: %v", err)
	}

	usage1 := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	usage2 := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := config.DB.Create(&usage1).Error; err != nil {
		t.Fatalf("create usage1 failed: %v", err)
	}
	if err := config.DB.Create(&usage2).Error; err != nil {
		t.Fatalf("create usage2 failed: %v", err)
	}

	sess1 := models.ServiceSession{
		MerchantID:       m.ID,
		UserID:           user1.ID,
		InitialUsageID:   usage1.ID,
		LastTechnicianID: &tech1.ID,
		Status:           "timeout_waiting",
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	sess2 := models.ServiceSession{
		MerchantID:       m.ID,
		UserID:           user2.ID,
		InitialUsageID:   usage2.ID,
		LastTechnicianID: &tech2.ID,
		Status:           "timeout_waiting",
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	if err := config.DB.Create(&sess1).Error; err != nil {
		t.Fatalf("create sess1 failed: %v", err)
	}
	if err := config.DB.Create(&sess2).Error; err != nil {
		t.Fatalf("create sess2 failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/queue/timeout-waiting-list", nil)
	c.Set("merchant_id", m.ID)
	c.Set("auth_type", "staff")
	c.Set("technician_id", tech1.ID)

	GetQueueTimeoutWaitingList(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Data []queueTimeoutWaitingItem `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("want 1 item, got %d body=%s", len(body.Data), rec.Body.String())
	}
	if body.Data[0].UsageID != usage1.ID {
		t.Fatalf("want only usage %d, got %+v", usage1.ID, body.Data)
	}
}

func TestReassignCurrentPendingSessionSupportsCustomerServiceMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupHandlerTestDB(t)
	now := time.Now()

	m := models.Merchant{
		Name:                       "m-cs-reassign",
		Phone:                      "18800000006",
		Password:                   "pwd",
		SupportCustomerServiceMode: true,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	oldTech := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服A", Code: "1001", Account: "js1001", Password: "pwd", IsActive: true}
	newTech := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服B", Code: "1002", Account: "js1002", Password: "pwd", IsActive: true}
	if err := config.DB.Create(&oldTech).Error; err != nil {
		t.Fatalf("create old tech failed: %v", err)
	}
	if err := config.DB.Create(&newTech).Error; err != nil {
		t.Fatalf("create new tech failed: %v", err)
	}

	oldAtt := models.TechnicianAttendance{MerchantID: m.ID, TechnicianID: oldTech.ID, CheckedInAt: &now, Status: "busy"}
	newAtt := models.TechnicianAttendance{MerchantID: m.ID, TechnicianID: newTech.ID, CheckedInAt: &now, Status: "idle"}
	if err := config.DB.Create(&oldAtt).Error; err != nil {
		t.Fatalf("create old attendance failed: %v", err)
	}
	if err := config.DB.Create(&newAtt).Error; err != nil {
		t.Fatalf("create new attendance failed: %v", err)
	}

	usage := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := config.DB.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	session := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: usage.ID,
		SessionMode:    models.SessionModeCustomerService,
		TechnicianID:   &oldTech.ID,
		Status:         "cs_start_pending",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("merchant_id", m.ID)
	c.Set("auth_type", "staff")
	c.Set("technician_id", oldTech.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(session.ID), 10)}}
	c.Request = httptest.NewRequest(http.MethodPost, "/queue/sessions/"+strconv.FormatUint(uint64(session.ID), 10)+"/reassign-current-pending", strings.NewReader(`{"reason":"下班前转交"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	ReassignCurrentPendingSession(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var gotSession models.ServiceSession
	if err := config.DB.First(&gotSession, session.ID).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotSession.TechnicianID == nil || *gotSession.TechnicianID != newTech.ID {
		t.Fatalf("want session reassigned to %d, got %+v", newTech.ID, gotSession.TechnicianID)
	}
	if gotSession.LastTechnicianID == nil || *gotSession.LastTechnicianID != oldTech.ID {
		t.Fatalf("want last technician %d, got %+v", oldTech.ID, gotSession.LastTechnicianID)
	}

	var gotOldAtt struct {
		Status string `gorm:"column:status"`
	}
	if err := config.DB.Model(&models.TechnicianAttendance{}).Select("status").First(&gotOldAtt, oldAtt.ID).Error; err != nil {
		t.Fatalf("reload old attendance failed: %v", err)
	}
	if gotOldAtt.Status != "idle" {
		t.Fatalf("want old attendance idle, got %s", gotOldAtt.Status)
	}

	var gotNewAtt struct {
		Status string `gorm:"column:status"`
	}
	if err := config.DB.Model(&models.TechnicianAttendance{}).Select("status").First(&gotNewAtt, newAtt.ID).Error; err != nil {
		t.Fatalf("reload new attendance failed: %v", err)
	}
	if gotNewAtt.Status != "busy" {
		t.Fatalf("want new attendance busy, got %s", gotNewAtt.Status)
	}
}

func TestReassignCurrentPendingSessionCustomerServiceRejectsNoTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupHandlerTestDB(t)
	now := time.Now()

	m := models.Merchant{
		Name:                       "m-cs-no-target",
		Phone:                      "18800000007",
		Password:                   "pwd",
		SupportCustomerServiceMode: true,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	tech := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服A", Code: "1101", Account: "js1101", Password: "pwd", IsActive: true}
	if err := config.DB.Create(&tech).Error; err != nil {
		t.Fatalf("create tech failed: %v", err)
	}

	att := models.TechnicianAttendance{MerchantID: m.ID, TechnicianID: tech.ID, CheckedInAt: &now, Status: "busy"}
	if err := config.DB.Create(&att).Error; err != nil {
		t.Fatalf("create attendance failed: %v", err)
	}

	usage := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := config.DB.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	session := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: usage.ID,
		SessionMode:    models.SessionModeCustomerService,
		TechnicianID:   &tech.ID,
		Status:         "cs_start_pending",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("merchant_id", m.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(session.ID), 10)}}
	c.Request = httptest.NewRequest(http.MethodPost, "/queue/sessions/"+strconv.FormatUint(uint64(session.ID), 10)+"/reassign-current-pending", strings.NewReader(`{"reason":"下班前转交"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	ReassignCurrentPendingSession(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "当前没有可接手的空闲客服") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}

	var gotSession models.ServiceSession
	if err := config.DB.First(&gotSession, session.ID).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotSession.TechnicianID == nil || *gotSession.TechnicianID != tech.ID {
		t.Fatalf("want session still on %d, got %+v", tech.ID, gotSession.TechnicianID)
	}
}

func TestReassignCurrentPendingSessionCustomerServiceStaffOnlyOwnSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupHandlerTestDB(t)
	now := time.Now()

	m := models.Merchant{
		Name:                       "m-cs-auth",
		Phone:                      "18800000008",
		Password:                   "pwd",
		SupportCustomerServiceMode: true,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	ownerTech := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服A", Code: "1201", Account: "js1201", Password: "pwd", IsActive: true}
	otherTech := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服B", Code: "1202", Account: "js1202", Password: "pwd", IsActive: true}
	targetTech := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服C", Code: "1203", Account: "js1203", Password: "pwd", IsActive: true}
	if err := config.DB.Create(&ownerTech).Error; err != nil {
		t.Fatalf("create owner tech failed: %v", err)
	}
	if err := config.DB.Create(&otherTech).Error; err != nil {
		t.Fatalf("create other tech failed: %v", err)
	}
	if err := config.DB.Create(&targetTech).Error; err != nil {
		t.Fatalf("create target tech failed: %v", err)
	}

	if err := config.DB.Create(&models.TechnicianAttendance{MerchantID: m.ID, TechnicianID: otherTech.ID, CheckedInAt: &now, Status: "busy"}).Error; err != nil {
		t.Fatalf("create other attendance failed: %v", err)
	}
	if err := config.DB.Create(&models.TechnicianAttendance{MerchantID: m.ID, TechnicianID: targetTech.ID, CheckedInAt: &now, Status: "idle"}).Error; err != nil {
		t.Fatalf("create target attendance failed: %v", err)
	}

	usage := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := config.DB.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	session := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: usage.ID,
		SessionMode:    models.SessionModeCustomerService,
		TechnicianID:   &otherTech.ID,
		Status:         "cs_start_pending",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("merchant_id", m.ID)
	c.Set("auth_type", "staff")
	c.Set("technician_id", ownerTech.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(session.ID), 10)}}
	c.Request = httptest.NewRequest(http.MethodPost, "/queue/sessions/"+strconv.FormatUint(uint64(session.ID), 10)+"/reassign-current-pending", strings.NewReader(`{"reason":"越权测试"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	ReassignCurrentPendingSession(c)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want status 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "仅可转交自己当前待上号单") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestReassignCurrentPendingSessionCustomerServiceOperationalCanReassignAny(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	config.DB = setupHandlerTestDB(t)
	now := time.Now()

	m := models.Merchant{
		Name:                       "m-cs-operational",
		Phone:                      "18800000009",
		Password:                   "pwd",
		SupportCustomerServiceMode: true,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	ownerTech := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服A", Code: "1301", Account: "js1301", Password: "pwd", IsActive: true}
	operTech := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "运营", Code: "1302", Account: "js1302", Password: "pwd", IsActive: true}
	targetTech := models.Technician{MerchantID: m.ID, ServiceRoleID: 1, Name: "客服B", Code: "1303", Account: "js1303", Password: "pwd", IsActive: true}
	if err := config.DB.Create(&ownerTech).Error; err != nil {
		t.Fatalf("create owner tech failed: %v", err)
	}
	if err := config.DB.Create(&operTech).Error; err != nil {
		t.Fatalf("create operational tech failed: %v", err)
	}
	if err := config.DB.Create(&targetTech).Error; err != nil {
		t.Fatalf("create target tech failed: %v", err)
	}

	if err := config.DB.Create(&models.TechnicianAttendance{MerchantID: m.ID, TechnicianID: ownerTech.ID, CheckedInAt: &now, Status: "busy"}).Error; err != nil {
		t.Fatalf("create owner attendance failed: %v", err)
	}
	if err := config.DB.Create(&models.TechnicianAttendance{MerchantID: m.ID, TechnicianID: targetTech.ID, CheckedInAt: &now, Status: "idle"}).Error; err != nil {
		t.Fatalf("create target attendance failed: %v", err)
	}

	usage := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := config.DB.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	session := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: usage.ID,
		SessionMode:    models.SessionModeCustomerService,
		TechnicianID:   &ownerTech.ID,
		Status:         "cs_start_pending",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("merchant_id", m.ID)
	c.Set("auth_type", "staff")
	c.Set("technician_id", operTech.ID)
	c.Set("service_role", models.ServiceRole{RoleType: "operational"})
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(session.ID), 10)}}
	c.Request = httptest.NewRequest(http.MethodPost, "/queue/sessions/"+strconv.FormatUint(uint64(session.ID), 10)+"/reassign-current-pending", strings.NewReader(`{"reason":"运营转交"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	ReassignCurrentPendingSession(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

type noopQueueStore struct {
	snapshot queue.Snapshot
}

func (n *noopQueueStore) Enqueue(merchantID uint, date string, qt queue.QueueType, id uint, startNo int, autoCallFirst bool, now time.Time) (queue.Ticket, bool) {
	return queue.Ticket{}, false
}
func (n *noopQueueStore) MarkDone(merchantID uint, date string, qt queue.QueueType, doneID uint, now time.Time) {
}
func (n *noopQueueStore) UnmarkDone(merchantID uint, date string, qt queue.QueueType, id uint) {}
func (n *noopQueueStore) GetNo(merchantID uint, date string, qt queue.QueueType, id uint) (int, bool) {
	return 0, false
}
func (n *noopQueueStore) CallNextUncalled(merchantID uint, date string, qt queue.QueueType, now time.Time) uint {
	return 0
}
func (n *noopQueueStore) Uncall(merchantID uint, date string, qt queue.QueueType, id uint) {}
func (n *noopQueueStore) Snapshot(merchantID uint, date string, qt queue.QueueType) queue.Snapshot {
	return n.snapshot
}
