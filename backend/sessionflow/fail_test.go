package sessionflow

import (
	"testing"
	"time"

	"kabao/models"
	"kabao/queue"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFailServiceSessionAndRefundMarksUsageFailedAndRefundsCard(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_fail_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Card{}, &models.Usage{}, &models.ServiceSession{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	merchant := models.Merchant{Name: "m", Phone: "18800000010", Password: "pwd", SupportQueue: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	card := models.Card{MerchantID: merchant.ID, CardNo: "00001", CardType: "times", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	usage := models.Usage{MerchantID: merchant.ID, CardID: card.ID, UsedTimes: 1, Status: "in_progress"}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{MerchantID: merchant.ID, InitialUsageID: usage.ID, Status: "timeout_waiting"}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	oldQueue := queue.Default
	defer func() { queue.Default = oldQueue }()
	stub := &stubQueueStore{}
	queue.Default = stub

	if err := FailServiceSessionAndRefund(db, &session, &merchant, time.Now(), FailOptions{
		AllowedBaseStatuses: []string{"timeout_waiting"},
		TargetStatus:        "timeout_failed",
		MarkQueueDone:       true,
	}); err != nil {
		t.Fatalf("FailServiceSessionAndRefund failed: %v", err)
	}

	var gotUsage struct {
		Status string `gorm:"column:status"`
	}
	if err := db.Model(&models.Usage{}).Select("status").First(&gotUsage, usage.ID).Error; err != nil {
		t.Fatalf("load usage failed: %v", err)
	}
	if gotUsage.Status != "failed" {
		t.Fatalf("want failed usage, got %s", gotUsage.Status)
	}

	var gotCard struct {
		RemainTimes int `gorm:"column:remain_times"`
		UsedTimes   int `gorm:"column:used_times"`
	}
	if err := db.Model(&models.Card{}).Select("remain_times, used_times").First(&gotCard, card.ID).Error; err != nil {
		t.Fatalf("load card failed: %v", err)
	}
	if gotCard.RemainTimes != 10 || gotCard.UsedTimes != 0 {
		t.Fatalf("want refunded card remain=10 used=0, got remain=%d used=%d", gotCard.RemainTimes, gotCard.UsedTimes)
	}
	if len(stub.doneIDs) != 1 || stub.doneIDs[0] != usage.ID {
		t.Fatalf("want queue MarkDone on usage %d, got %+v", usage.ID, stub.doneIDs)
	}
}

func TestCompleteUnstartedServiceSessionReturnsNilWhenSessionMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_complete_missing_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Usage{}, &models.ServiceSession{}, &models.TechnicianAttendance{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	usage := models.Usage{MerchantID: 1, Status: "in_progress"}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	handled, err := CompleteUnstartedServiceSession(db, usage.ID, 1, time.Now())
	if err != nil {
		t.Fatalf("CompleteUnstartedServiceSession failed: %v", err)
	}
	if handled {
		t.Fatalf("want missing session to be ignored")
	}
}
