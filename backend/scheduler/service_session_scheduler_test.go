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
	if err := db.AutoMigrate(&models.Merchant{}, &models.Usage{}, &models.ServiceSession{}); err != nil {
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
