package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		&models.Technician{},
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
