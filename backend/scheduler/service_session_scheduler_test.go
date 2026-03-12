package scheduler

import (
	"testing"
	"time"

	"kabao/config"
	"kabao/models"
	"kabao/queue"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSchedulerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:scheduler_service_session_test_" + time.Now().Format("20060102150405_000000000") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Technician{}, &models.Card{}, &models.Usage{}, &models.ServiceSession{}, &models.TechnicianAttendance{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestFinalizeUsagesAfterQueueEnded_OnlyInProgressToSuccess(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupSchedulerTestDB(t)
	config.DB = db

	now := time.Now()
	endedAt := now.Add(-16 * time.Minute)
	m := models.Merchant{
		Name:         "m-ended",
		Phone:        "18800000101",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
		QueuePaused:  true,
		QueueEndedAt: &endedAt,
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	u1 := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	u2 := models.Usage{MerchantID: m.ID, Status: "failed"}
	u3 := models.Usage{MerchantID: m.ID, Status: "canceled"}
	if err := db.Create(&[]*models.Usage{&u1, &u2, &u3}).Error; err != nil {
		t.Fatalf("create usages failed: %v", err)
	}

	if err := finalizeUsagesAfterQueueEnded(db, now); err != nil {
		t.Fatalf("finalizeUsagesAfterQueueEnded failed: %v", err)
	}

	var got1, got2, got3 struct{ Status string }
	if err := db.Table("usages").Select("status").Where("id = ?", u1.ID).Scan(&got1).Error; err != nil {
		t.Fatalf("reload u1 failed: %v", err)
	}
	if err := db.Table("usages").Select("status").Where("id = ?", u2.ID).Scan(&got2).Error; err != nil {
		t.Fatalf("reload u2 failed: %v", err)
	}
	if err := db.Table("usages").Select("status").Where("id = ?", u3.ID).Scan(&got3).Error; err != nil {
		t.Fatalf("reload u3 failed: %v", err)
	}

	if got1.Status != "success" {
		t.Fatalf("want u1 success, got %s", got1.Status)
	}
	if got2.Status != "failed" {
		t.Fatalf("want u2 remain failed, got %s", got2.Status)
	}
	if got3.Status != "canceled" {
		t.Fatalf("want u3 remain canceled, got %s", got3.Status)
	}
}

func TestFinalizeUsagesAfterQueueEnded_FinishesLinkedPrefixedSession(t *testing.T) {
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	db := setupSchedulerTestDB(t)
	config.DB = db
	fq := &fakeQueueStore{}
	queue.Default = fq

	now := time.Now()
	endedAt := now.Add(-16 * time.Minute)
	m := models.Merchant{
		Name:         "m-ended-linked-session",
		Phone:        "18800000132",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
		QueuePaused:  true,
		QueueEndedAt: &endedAt,
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	u := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	startAt := now.Add(-30 * time.Minute)
	session := models.ServiceSession{
		MerchantID:       m.ID,
		InitialUsageID:   u.ID,
		Status:           "qms_delay_pending",
		StartConfirmedAt: &startAt,
		ScheduledStartAt: &startAt,
		CreatedAt:        &startAt,
		UpdatedAt:        &startAt,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := finalizeUsagesAfterQueueEnded(db, now); err != nil {
		t.Fatalf("finalizeUsagesAfterQueueEnded failed: %v", err)
	}

	var gotU struct{ Status string }
	if err := db.Table("usages").Select("status").Where("id = ?", u.ID).Scan(&gotU).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotU.Status != "success" {
		t.Fatalf("want usage success, got %s", gotU.Status)
	}

	var gotS struct{ Status string }
	if err := db.Table("service_sessions").Select("status").Where("id = ?", session.ID).Scan(&gotS).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotS.Status != "qms_finished" {
		t.Fatalf("want session qms_finished, got %s", gotS.Status)
	}
	if fq.markDoneCount == 0 {
		t.Fatalf("want MarkDone called")
	}
}

func TestFinalizeSession_WhenQueuePaused_DoesNotCallNext(t *testing.T) {
	oldQueue := queue.Default
	defer func() { queue.Default = oldQueue }()

	db := setupSchedulerTestDB(t)
	fq := &fakeQueueStore{}
	queue.Default = fq

	now := time.Now()
	m := models.Merchant{
		Name:                        "m-auto-paused",
		Phone:                       "18800000102",
		Password:                    "pwd",
		SupportQueue:                true,
		QueueMode:                   "auto",
		SupportMultiCustomerService: false,
		QueuePaused:                 true,
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	u := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	s := models.ServiceSession{
		MerchantID:       m.ID,
		InitialUsageID:   u.ID,
		Status:           "serving",
		StartConfirmedAt: &now,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return finalizeSession(tx, &s, now)
	}); err != nil {
		t.Fatalf("finalizeSession failed: %v", err)
	}

	if fq.markDoneCount == 0 {
		t.Fatalf("want MarkDone called at least once")
	}
	if fq.callNextCount != 0 {
		t.Fatalf("want CallNextUncalled not called when queue paused, got %d", fq.callNextCount)
	}
}

func TestMoveMultiQueueStartPendingToTimeoutWaitingPreservesLastTechnician(t *testing.T) {
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	db := setupSchedulerTestDB(t)
	config.DB = db
	queue.Default = &fakeQueueStore{}

	now := time.Now()
	m := models.Merchant{
		Name:                        "m-auto-multi-timeout",
		Phone:                       "18800000109",
		Password:                    "pwd",
		SupportQueue:                true,
		QueueMode:                   "auto",
		SupportMultiCustomerService: true,
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	tech := models.Technician{
		MerchantID:    m.ID,
		ServiceRoleID: 1,
		Name:          "大漂亮",
		Code:          "0001",
		Account:       "js0001",
		Password:      "pwd",
		IsActive:      true,
		WindowNo:      "A1",
	}
	if err := db.Create(&tech).Error; err != nil {
		t.Fatalf("create technician failed: %v", err)
	}

	u := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	s := models.ServiceSession{
		MerchantID:                 m.ID,
		InitialUsageID:             u.ID,
		TechnicianID:               &tech.ID,
		Status:                     "qm_start_pending",
		StartPendingTimeoutSeconds: 180,
		CreatedAt:                  &now,
		UpdatedAt:                  &now,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return moveMultiQueueStartPendingToTimeoutWaiting(tx, &s, &m, now)
	}); err != nil {
		t.Fatalf("moveMultiQueueStartPendingToTimeoutWaiting failed: %v", err)
	}

	var got struct {
		Status           string
		TechnicianID     *uint
		LastTechnicianID *uint
	}
	if err := db.Table("service_sessions").Select("status, technician_id, last_technician_id").Where("id = ?", s.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}

	if models.NormalizeSessionStatus(got.Status) != "timeout_waiting" {
		t.Fatalf("want timeout_waiting, got %s", got.Status)
	}
	if got.TechnicianID != nil {
		t.Fatalf("want technician_id cleared, got %v", *got.TechnicianID)
	}
	if got.LastTechnicianID == nil || *got.LastTechnicianID != tech.ID {
		t.Fatalf("want last_technician_id=%d, got %v", tech.ID, got.LastTechnicianID)
	}
}

func TestAdvanceOne_ManualStaffSelecting_CrossDayCancelAndRefund(t *testing.T) {
	oldQueue := queue.Default
	defer func() { queue.Default = oldQueue }()

	db := setupSchedulerTestDB(t)
	fq := &fakeQueueStore{}
	queue.Default = fq

	now := time.Now()
	// 手动叫号模式
	m := models.Merchant{
		Name:         "m-manual",
		Phone:        "18800000103",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	// 卡初始：remain=9 used=1（模拟核销已扣一次）
	c := models.Card{MerchantID: m.ID, UserID: 1, CardNo: "c1", CardType: "t", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	// usage in_progress，used_times=1
	u := models.Usage{MerchantID: m.ID, CardID: c.ID, UsedTimes: 1, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	// staff_selecting 且 updated_at 在昨日，触发跨天兜底
	baseAt := now.Add(-26 * time.Hour)
	s := models.ServiceSession{
		MerchantID:     m.ID,
		CardID:         c.ID,
		InitialUsageID: u.ID,
		Status:         "staff_selecting",
		CreatedAt:      &baseAt,
		UpdatedAt:      &baseAt,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	// gorm 可能会自动写入 created_at/updated_at，这里强制改回跨天时间，确保兜底触发
	if err := db.Model(&models.ServiceSession{}).Where("id = ?", s.ID).UpdateColumns(map[string]interface{}{
		"created_at": baseAt,
		"updated_at": baseAt,
	}).Error; err != nil {
		t.Fatalf("update session timestamps failed: %v", err)
	}

	if err := advanceOne(db, &s, now); err != nil {
		t.Fatalf("advanceOne failed: %v", err)
	}

	var gotS struct {
		Status string
	}
	if err := db.Table("service_sessions").Select("status").Where("id = ?", s.ID).Scan(&gotS).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotS.Status != "canceled" {
		t.Fatalf("want session canceled, got %s", gotS.Status)
	}

	var gotU struct{ Status string }
	if err := db.Table("usages").Select("status").Where("id = ?", u.ID).Scan(&gotU).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotU.Status != "failed" {
		t.Fatalf("want usage failed, got %s", gotU.Status)
	}

	var gotC struct {
		RemainTimes int
		UsedTimes   int
	}
	if err := db.Table("cards").Select("remain_times, used_times").Where("id = ?", c.ID).Scan(&gotC).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotC.RemainTimes != 10 {
		t.Fatalf("want card remain_times=10, got %d", gotC.RemainTimes)
	}
	if gotC.UsedTimes != 0 {
		t.Fatalf("want card used_times=0, got %d", gotC.UsedTimes)
	}
}

func TestSkipCurrentAndCallNext_NonAutoSingle_UsesSharedFailFlow(t *testing.T) {
	oldQueue := queue.Default
	defer func() { queue.Default = oldQueue }()

	db := setupSchedulerTestDB(t)
	fq := &fakeQueueStore{}
	queue.Default = fq

	now := time.Now()
	m := models.Merchant{
		Name:         "m-skip-shared-fail",
		Phone:        "18800000131",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "auto",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	c := models.Card{MerchantID: m.ID, UserID: 1, CardNo: "c-skip", CardType: "t", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	u := models.Usage{MerchantID: m.ID, CardID: c.ID, UsedTimes: 1, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	startAt := now.Add(-2 * time.Minute)
	s := models.ServiceSession{
		MerchantID:       m.ID,
		CardID:           c.ID,
		InitialUsageID:   u.ID,
		SessionMode:      models.SessionModeQueueManualSingle,
		Status:           "qms_delay_pending",
		StartConfirmedAt: &startAt,
		ScheduledStartAt: &startAt,
		CreatedAt:        &startAt,
		UpdatedAt:        &startAt,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return skipCurrentAndCallNext(tx, &s, &m, now)
	}); err != nil {
		t.Fatalf("skipCurrentAndCallNext failed: %v", err)
	}

	var gotS struct {
		Status           string
		StartConfirmedAt *time.Time
		ScheduledStartAt *time.Time
	}
	if err := db.Table("service_sessions").Select("status, start_confirmed_at, scheduled_start_at").Where("id = ?", s.ID).Scan(&gotS).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotS.Status != "qms_canceled" {
		t.Fatalf("want session qms_canceled, got %s", gotS.Status)
	}
	if gotS.StartConfirmedAt != nil || gotS.ScheduledStartAt != nil {
		t.Fatalf("want start timing fields cleared")
	}

	var gotU struct{ Status string }
	if err := db.Table("usages").Select("status").Where("id = ?", u.ID).Scan(&gotU).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotU.Status != "failed" {
		t.Fatalf("want usage failed, got %s", gotU.Status)
	}

	var gotC struct {
		RemainTimes int
		UsedTimes   int
	}
	if err := db.Table("cards").Select("remain_times, used_times").Where("id = ?", c.ID).Scan(&gotC).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotC.RemainTimes != 10 || gotC.UsedTimes != 0 {
		t.Fatalf("want refunded card remain=10 used=0, got remain=%d used=%d", gotC.RemainTimes, gotC.UsedTimes)
	}
	if fq.markDoneCount == 0 {
		t.Fatalf("want MarkDone called")
	}
}

func TestAdvanceOne_TimeoutFailed_DoesIdempotentCleanup(t *testing.T) {
	oldQueue := queue.Default
	defer func() { queue.Default = oldQueue }()

	db := setupSchedulerTestDB(t)
	fq := &fakeQueueStore{}
	queue.Default = fq

	now := time.Now()
	m := models.Merchant{
		Name:         "m-timeout-failed",
		Phone:        "18800000108",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	// 卡初始：remain=9 used=1（模拟核销已扣一次）
	c := models.Card{MerchantID: m.ID, UserID: 1, CardNo: "c-timeout", CardType: "t", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	// usage in_progress，used_times=1
	u := models.Usage{MerchantID: m.ID, CardID: c.ID, UsedTimes: 1, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	s := models.ServiceSession{
		MerchantID:     m.ID,
		CardID:         c.ID,
		InitialUsageID: u.ID,
		Status:         "timeout_failed",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := advanceOne(db, &s, now); err != nil {
		t.Fatalf("advanceOne failed: %v", err)
	}
	if fq.markDoneCount == 0 {
		t.Fatalf("want MarkDone called at least once")
	}

	var gotU struct {
		Status     string
		FinishedAt string `gorm:"column:finished_at"`
	}
	if err := db.Table("usages").Select("status, finished_at").Where("id = ?", u.ID).Scan(&gotU).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotU.Status != "failed" {
		t.Fatalf("want usage failed, got %s", gotU.Status)
	}
	if gotU.FinishedAt == "" {
		t.Fatalf("want usage finished_at set")
	}

	var gotC struct {
		RemainTimes int
		UsedTimes   int
	}
	if err := db.Table("cards").Select("remain_times, used_times").Where("id = ?", c.ID).Scan(&gotC).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotC.RemainTimes != 10 {
		t.Fatalf("want card remain_times=10, got %d", gotC.RemainTimes)
	}
	if gotC.UsedTimes != 0 {
		t.Fatalf("want card used_times=0, got %d", gotC.UsedTimes)
	}

	markDoneAfterFirst := fq.markDoneCount
	if err := advanceOne(db, &s, now.Add(5*time.Second)); err != nil {
		t.Fatalf("advanceOne second run failed: %v", err)
	}
	if fq.markDoneCount < markDoneAfterFirst {
		t.Fatalf("markDoneCount should not decrease")
	}
	var gotC2 struct {
		RemainTimes int
		UsedTimes   int
	}
	if err := db.Table("cards").Select("remain_times, used_times").Where("id = ?", c.ID).Scan(&gotC2).Error; err != nil {
		t.Fatalf("reload card second run failed: %v", err)
	}
	if gotC2.RemainTimes != 10 || gotC2.UsedTimes != 0 {
		t.Fatalf("card times should remain idempotent, got remain=%d used=%d", gotC2.RemainTimes, gotC2.UsedTimes)
	}
}

func TestRunOnce_FinishedNotStarveActiveSessions(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupSchedulerTestDB(t)
	config.DB = db

	m := models.Merchant{
		Name:     "m-starve",
		Phone:    "18800000104",
		Password: "pwd",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	// 大量低ID finished 会话，模拟旧批次里会挤占 limit(200)
	for i := 0; i < 220; i++ {
		s := models.ServiceSession{
			MerchantID:             m.ID,
			Status:                 "finished",
			AutoIdleAfterSeconds:   180,
			AutoFinishDelaySeconds: 60,
		}
		if err := db.Create(&s).Error; err != nil {
			t.Fatalf("create finished session failed at %d: %v", i, err)
		}
	}

	// 一个高ID活跃会话：非叫号模式下应被推进到 delay_pending
	active := models.ServiceSession{
		MerchantID:        m.ID,
		Status:            "staff_selecting",
		StartDelaySeconds: 60,
	}
	if err := db.Create(&active).Error; err != nil {
		t.Fatalf("create active session failed: %v", err)
	}

	if err := runOnce(db); err != nil {
		t.Fatalf("runOnce failed: %v", err)
	}

	var got struct{ Status string }
	if err := db.Table("service_sessions").Select("status").Where("id = ?", active.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload active session failed: %v", err)
	}
	if got.Status != "delay_pending" {
		t.Fatalf("want active session promoted to delay_pending, got %s", got.Status)
	}
}

func TestRunOnce_QueueEndedFinalizesLinkedSessionAndUsage(t *testing.T) {
	oldDB := config.DB
	oldQueue := queue.Default
	defer func() {
		config.DB = oldDB
		queue.Default = oldQueue
	}()

	db := setupSchedulerTestDB(t)
	config.DB = db
	fq := &fakeQueueStore{}
	queue.Default = fq

	now := time.Now()
	endedAt := now.Add(-16 * time.Minute)
	m := models.Merchant{
		Name:         "m-runonce-ended",
		Phone:        "18800000133",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
		QueuePaused:  true,
		QueueEndedAt: &endedAt,
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	u := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	startAt := now.Add(-20 * time.Minute)
	s := models.ServiceSession{
		MerchantID:       m.ID,
		InitialUsageID:   u.ID,
		Status:           "qms_serving",
		SessionMode:      models.SessionModeQueueManualSingle,
		StartConfirmedAt: &startAt,
		StartedAt:        &startAt,
		ScheduledFinishAt: &startAt,
		CreatedAt:        &startAt,
		UpdatedAt:        &startAt,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := runOnce(db); err != nil {
		t.Fatalf("runOnce failed: %v", err)
	}

	var gotU struct{ Status string }
	if err := db.Table("usages").Select("status").Where("id = ?", u.ID).Scan(&gotU).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotU.Status != "success" {
		t.Fatalf("want usage success, got %s", gotU.Status)
	}

	var gotS struct{ Status string }
	if err := db.Table("service_sessions").Select("status").Where("id = ?", s.ID).Scan(&gotS).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotS.Status != "qms_finished" {
		t.Fatalf("want session qms_finished, got %s", gotS.Status)
	}
	if fq.markDoneCount == 0 {
		t.Fatalf("want MarkDone called")
	}
}

func TestRunOnce_BackfillsLegacySessionMode(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupSchedulerTestDB(t)
	config.DB = db

	now := time.Now()
	m := models.Merchant{
		Name:         "m-legacy-mode",
		Phone:        "18800000134",
		Password:     "pwd",
		SupportQueue: true,
		QueueMode:    "manual",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	s := models.ServiceSession{
		MerchantID: m.ID,
		Status:     "qms_finished",
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if err := db.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Update("session_mode", "").Error; err != nil {
		t.Fatalf("clear session_mode failed: %v", err)
	}

	if err := runOnce(db); err != nil {
		t.Fatalf("runOnce failed: %v", err)
	}

	var got struct {
		SessionMode string
		Status      string
	}
	if err := db.Table("service_sessions").Select("session_mode, status").Where("id = ?", s.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if got.SessionMode != models.SessionModeQueueManualSingle {
		t.Fatalf("want session_mode=%s, got %s", models.SessionModeQueueManualSingle, got.SessionMode)
	}
	if got.Status != "qms_finished" {
		t.Fatalf("want status remain qms_finished, got %s", got.Status)
	}
}

func TestReleaseFinishedSessionTechnicians_ReleasesBusyAttendance(t *testing.T) {
	db := setupSchedulerTestDB(t)

	now := time.Now()
	m := models.Merchant{
		Name:     "m-release",
		Phone:    "18800000105",
		Password: "pwd",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	techID := uint(9001)
	checkedInAt := now.Add(-2 * time.Hour)
	att := models.TechnicianAttendance{
		MerchantID:   m.ID,
		TechnicianID: techID,
		CheckedInAt:  &checkedInAt,
		Status:       "busy",
	}
	if err := db.Create(&att).Error; err != nil {
		t.Fatalf("create attendance failed: %v", err)
	}

	finishedAt := now.Add(-5 * time.Minute)
	s := models.ServiceSession{
		MerchantID:           m.ID,
		Status:               "finished",
		TechnicianID:         &techID,
		FinishedAt:           &finishedAt,
		AutoIdleAfterSeconds: 1,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create finished session failed: %v", err)
	}

	if err := releaseFinishedSessionTechnicians(db, now); err != nil {
		t.Fatalf("releaseFinishedSessionTechnicians failed: %v", err)
	}

	var gotAtt struct{ Status string }
	if err := db.Table("technician_attendances").Select("status").Where("id = ?", att.ID).Scan(&gotAtt).Error; err != nil {
		t.Fatalf("reload attendance failed: %v", err)
	}
	if gotAtt.Status != "idle" {
		t.Fatalf("want attendance idle, got %s", gotAtt.Status)
	}
}

func TestBackfillServingStartConfirmedAt_FillsFromStartedAt(t *testing.T) {
	db := setupSchedulerTestDB(t)

	now := time.Now()
	m := models.Merchant{
		Name:     "m-serving-backfill",
		Phone:    "18800000106",
		Password: "pwd",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	u := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	// 注意：sqlite 在测试环境下会将时间列以 string 存储/读取，避免走 advanceOne 的 SELECT * 扫描。
	startedAt := now.Add(-40 * time.Minute)
	s := models.ServiceSession{
		MerchantID:     m.ID,
		InitialUsageID: u.ID,
		Status:         "serving",
		StartedAt:      &startedAt,
		CreatedAt:      &startedAt,
		UpdatedAt:      &startedAt,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if err := db.Model(&models.ServiceSession{}).Where("id = ?", s.ID).UpdateColumn("start_confirmed_at", nil).Error; err != nil {
		t.Fatalf("clear start_confirmed_at failed: %v", err)
	}

	if err := backfillServingStartConfirmedAt(db, now); err != nil {
		t.Fatalf("backfillServingStartConfirmedAt failed: %v", err)
	}

	var got struct{ StartConfirmedAt string }
	if err := db.Table("service_sessions").Select("start_confirmed_at").Where("id = ?", s.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if got.StartConfirmedAt == "" {
		t.Fatalf("want start_confirmed_at filled")
	}
}

func TestFinalizeSession_FromServing_MarksFinishedAndUsageSuccess(t *testing.T) {
	db := setupSchedulerTestDB(t)

	now := time.Now()
	m := models.Merchant{
		Name:     "m-serving-finalize",
		Phone:    "18800000107",
		Password: "pwd",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	u := models.Usage{MerchantID: m.ID, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	startConfirmedAt := now.Add(-40 * time.Minute)
	scheduledFinishAt := now.Add(-10 * time.Minute)
	finishedAt := now
	s := models.ServiceSession{
		MerchantID:        m.ID,
		InitialUsageID:    u.ID,
		Status:            "serving",
		StartConfirmedAt:  &startConfirmedAt,
		ScheduledFinishAt: &scheduledFinishAt,
		FinishedAt:        &finishedAt,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return finalizeSession(tx, &s, now)
	}); err != nil {
		t.Fatalf("finalizeSession failed: %v", err)
	}

	var gotS struct{ Status string }
	if err := db.Table("service_sessions").Select("status").Where("id = ?", s.ID).Scan(&gotS).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotS.Status != "finished" {
		t.Fatalf("want session finished, got %s", gotS.Status)
	}

	var gotU struct{ Status string }
	if err := db.Table("usages").Select("status").Where("id = ?", u.ID).Scan(&gotU).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotU.Status != "success" {
		t.Fatalf("want usage success, got %s", gotU.Status)
	}
}

type fakeQueueStore struct {
	callNextCount int
	markDoneCount int
}

func (f *fakeQueueStore) Enqueue(merchantID uint, date string, qt queue.QueueType, id uint, startNo int, autoCallFirst bool, now time.Time) (queue.Ticket, bool) {
	return queue.Ticket{}, false
}
func (f *fakeQueueStore) MarkDone(merchantID uint, date string, qt queue.QueueType, doneID uint, now time.Time) {
	f.markDoneCount++
}
func (f *fakeQueueStore) UnmarkDone(merchantID uint, date string, qt queue.QueueType, id uint) {}
func (f *fakeQueueStore) GetNo(merchantID uint, date string, qt queue.QueueType, id uint) (int, bool) {
	return 0, false
}
func (f *fakeQueueStore) CallNextUncalled(merchantID uint, date string, qt queue.QueueType, now time.Time) uint {
	f.callNextCount++
	return 0
}
func (f *fakeQueueStore) Uncall(merchantID uint, date string, qt queue.QueueType, id uint) {}
func (f *fakeQueueStore) Snapshot(merchantID uint, date string, qt queue.QueueType) queue.Snapshot {
	return queue.Snapshot{}
}

type timeoutWaitingEmptySnapshotQueueStore struct {
	markDoneCount int
	noByID        map[uint]int
}

func (f *timeoutWaitingEmptySnapshotQueueStore) Enqueue(merchantID uint, date string, qt queue.QueueType, id uint, startNo int, autoCallFirst bool, now time.Time) (queue.Ticket, bool) {
	return queue.Ticket{}, false
}
func (f *timeoutWaitingEmptySnapshotQueueStore) MarkDone(merchantID uint, date string, qt queue.QueueType, doneID uint, now time.Time) {
	f.markDoneCount++
}
func (f *timeoutWaitingEmptySnapshotQueueStore) UnmarkDone(merchantID uint, date string, qt queue.QueueType, id uint) {
}
func (f *timeoutWaitingEmptySnapshotQueueStore) GetNo(merchantID uint, date string, qt queue.QueueType, id uint) (int, bool) {
	if f.noByID == nil {
		return 0, false
	}
	no, ok := f.noByID[id]
	return no, ok
}
func (f *timeoutWaitingEmptySnapshotQueueStore) CallNextUncalled(merchantID uint, date string, qt queue.QueueType, now time.Time) uint {
	return 0
}
func (f *timeoutWaitingEmptySnapshotQueueStore) Uncall(merchantID uint, date string, qt queue.QueueType, id uint) {
}
func (f *timeoutWaitingEmptySnapshotQueueStore) Snapshot(merchantID uint, date string, qt queue.QueueType) queue.Snapshot {
	return queue.Snapshot{}
}

func TestAdvanceOne_AutoMultiTimeoutWaiting_ExpiresEvenWhenSnapshotEmpty(t *testing.T) {
	oldQueue := queue.Default
	defer func() { queue.Default = oldQueue }()

	db := setupSchedulerTestDB(t)
	fq := &timeoutWaitingEmptySnapshotQueueStore{noByID: map[uint]int{}}
	queue.Default = fq

	now := time.Now()
	m := models.Merchant{
		Name:                        "m-auto-multi-timeout",
		Phone:                       "18800000999",
		Password:                    "pwd",
		SupportQueue:                true,
		QueueMode:                   "auto",
		SupportMultiCustomerService: true,
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	c := models.Card{MerchantID: m.ID, UserID: 1, CardNo: "c-auto-multi", CardType: "t", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	u := models.Usage{MerchantID: m.ID, CardID: c.ID, UsedTimes: 1, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	timeoutAt := now.Add(-16 * time.Minute)
	s := models.ServiceSession{
		MerchantID:     m.ID,
		CardID:         c.ID,
		InitialUsageID: u.ID,
		Status:         "timeout_waiting",
		CreatedAt:      &timeoutAt,
		UpdatedAt:      &timeoutAt,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	fq.noByID[u.ID] = 12

	if err := advanceOne(db, &s, now); err != nil {
		t.Fatalf("advanceOne failed: %v", err)
	}

	var gotS struct {
		Status     string
		FinishedAt string `gorm:"column:finished_at"`
	}
	if err := db.Table("service_sessions").Select("status, finished_at").Where("id = ?", s.ID).Scan(&gotS).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotS.Status != "timeout_failed" {
		t.Fatalf("want session timeout_failed, got %s", gotS.Status)
	}
	if gotS.FinishedAt == "" {
		t.Fatalf("want session finished_at set")
	}

	var gotU struct {
		Status string
	}
	if err := db.Table("usages").Select("status").Where("id = ?", u.ID).Scan(&gotU).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotU.Status != "failed" {
		t.Fatalf("want usage failed, got %s", gotU.Status)
	}

	var gotC struct {
		RemainTimes int
		UsedTimes   int
	}
	if err := db.Table("cards").Select("remain_times, used_times").Where("id = ?", c.ID).Scan(&gotC).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotC.RemainTimes != 10 || gotC.UsedTimes != 0 {
		t.Fatalf("want refunded card remain=10 used=0, got remain=%d used=%d", gotC.RemainTimes, gotC.UsedTimes)
	}
	if fq.markDoneCount == 0 {
		t.Fatalf("want MarkDone called when timeout_waiting expires")
	}
}

func TestAdvanceOne_HistoricalAutoMultiTimeoutWaiting_StillExpiresAfterQueueDisabled(t *testing.T) {
	oldQueue := queue.Default
	defer func() { queue.Default = oldQueue }()

	db := setupSchedulerTestDB(t)
	queue.Default = nil

	now := time.Now()
	m := models.Merchant{
		Name:                        "m-history-auto-multi-timeout",
		Phone:                       "18800001000",
		Password:                    "pwd",
		SupportQueue:                false,
		QueueMode:                   "auto",
		SupportMultiCustomerService: true,
		SupportCustomerService:      true,
		SupportCustomerServiceMode:  true,
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	c := models.Card{MerchantID: m.ID, UserID: 1, CardNo: "c-history-auto-multi", CardType: "t", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	u := models.Usage{MerchantID: m.ID, CardID: c.ID, UsedTimes: 1, Status: "in_progress"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	timeoutAt := now.Add(-16 * time.Minute)
	s := models.ServiceSession{
		MerchantID:     m.ID,
		CardID:         c.ID,
		InitialUsageID: u.ID,
		SessionMode:    models.SessionModeQueueAutoMulti,
		Status:         "qm_timeout_waiting",
		CreatedAt:      &timeoutAt,
		UpdatedAt:      &timeoutAt,
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := advanceOne(db, &s, now); err != nil {
		t.Fatalf("advanceOne failed: %v", err)
	}

	var gotS struct {
		Status     string
		FinishedAt string `gorm:"column:finished_at"`
	}
	if err := db.Table("service_sessions").Select("status, finished_at").Where("id = ?", s.ID).Scan(&gotS).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotS.Status != "qm_timeout_failed" {
		t.Fatalf("want session qm_timeout_failed, got %s", gotS.Status)
	}
	if gotS.FinishedAt == "" {
		t.Fatalf("want session finished_at set")
	}

	var gotU struct {
		Status string
	}
	if err := db.Table("usages").Select("status").Where("id = ?", u.ID).Scan(&gotU).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotU.Status != "failed" {
		t.Fatalf("want usage failed, got %s", gotU.Status)
	}

	var gotC struct {
		RemainTimes int
		UsedTimes   int
	}
	if err := db.Table("cards").Select("remain_times, used_times").Where("id = ?", c.ID).Scan(&gotC).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotC.RemainTimes != 10 || gotC.UsedTimes != 0 {
		t.Fatalf("want refunded card remain=10 used=0, got remain=%d used=%d", gotC.RemainTimes, gotC.UsedTimes)
	}
}
