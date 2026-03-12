package sessionflow

import (
	"testing"
	"time"

	"kabao/models"
	"kabao/queue"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubQueueStore struct {
	doneIDs []uint
}

func (s *stubQueueStore) Enqueue(uint, string, queue.QueueType, uint, int, bool, time.Time) (queue.Ticket, bool) {
	return queue.Ticket{}, false
}
func (s *stubQueueStore) MarkDone(_ uint, _ string, _ queue.QueueType, doneID uint, _ time.Time) {
	s.doneIDs = append(s.doneIDs, doneID)
}
func (s *stubQueueStore) UnmarkDone(uint, string, queue.QueueType, uint)                 {}
func (s *stubQueueStore) GetNo(uint, string, queue.QueueType, uint) (int, bool)          { return 0, false }
func (s *stubQueueStore) CallNextUncalled(uint, string, queue.QueueType, time.Time) uint { return 0 }
func (s *stubQueueStore) Uncall(uint, string, queue.QueueType, uint)                     {}
func (s *stubQueueStore) Snapshot(uint, string, queue.QueueType) queue.Snapshot {
	return queue.Snapshot{}
}

func TestFinishServiceSessionUpdatesUsageAndMarksQueueDone(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_finish_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Usage{}, &models.ServiceSession{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	now := time.Now()
	merchant := models.Merchant{Name: "m", Phone: "18800000009", Password: "pwd", SupportQueue: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	techID := uint(11)
	roomID := uint(22)
	usage := models.Usage{MerchantID: merchant.ID, Status: "in_progress"}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:       merchant.ID,
		InitialUsageID:   usage.ID,
		Status:           "serving",
		TechnicianID:     &techID,
		RoomID:           &roomID,
		StartConfirmedAt: &now,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	oldQueue := queue.Default
	defer func() { queue.Default = oldQueue }()
	stub := &stubQueueStore{}
	queue.Default = stub

	if err := FinishServiceSession(db, &session, &merchant, now, FinishOptions{MarkQueueDone: true}); err != nil {
		t.Fatalf("FinishServiceSession failed: %v", err)
	}

	var gotSession struct {
		Status string `gorm:"column:status"`
	}
	if err := db.Model(&models.ServiceSession{}).Select("status").First(&gotSession, session.ID).Error; err != nil {
		t.Fatalf("load session failed: %v", err)
	}
	if gotSession.Status != "finished" {
		t.Fatalf("want finished session, got %s", gotSession.Status)
	}

	var gotUsage struct {
		Status       string `gorm:"column:status"`
		TechnicianID *uint  `gorm:"column:technician_id"`
	}
	if err := db.Model(&models.Usage{}).Select("status, technician_id").First(&gotUsage, usage.ID).Error; err != nil {
		t.Fatalf("load usage failed: %v", err)
	}
	if gotUsage.Status != "success" {
		t.Fatalf("want success usage, got %s", gotUsage.Status)
	}
	if gotUsage.TechnicianID == nil || *gotUsage.TechnicianID != techID {
		t.Fatalf("want technician_id=%d, got %+v", techID, gotUsage.TechnicianID)
	}
	if len(stub.doneIDs) != 1 || stub.doneIDs[0] != usage.ID {
		t.Fatalf("want queue MarkDone on usage %d, got %+v", usage.ID, stub.doneIDs)
	}
}
