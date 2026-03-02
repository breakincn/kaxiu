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
	dsn := "file:scheduler_service_session_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Card{}, &models.Usage{}, &models.ServiceSession{}, &models.TechnicianAttendance{}); err != nil {
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

func TestRunOnce_FinishedNotStarveActiveSessions(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupSchedulerTestDB(t)
	config.DB = db

	now := time.Now()
	m := models.Merchant{
		Name:     "m-starve",
		Phone:    "18800000104",
		Password: "pwd",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	// 大量低ID finished 会话，模拟旧批次里会挤占 limit(200)
	finishedAt := now.Add(-10 * time.Minute)
	for i := 0; i < 220; i++ {
		s := models.ServiceSession{
			MerchantID:             m.ID,
			Status:                 "finished",
			FinishedAt:             &finishedAt,
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
