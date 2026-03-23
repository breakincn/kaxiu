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
	if err := db.AutoMigrate(&models.Merchant{}, &models.Card{}, &models.Appointment{}, &models.AppointmentSettlement{}, &models.Usage{}, &models.ServiceSession{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}); err != nil {
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
	appointment := models.Appointment{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, Status: "arrived", MerchantBreachPending: true}
	if err := db.Create(&appointment).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	settlement := models.AppointmentSettlement{AppointmentID: appointment.ID, MerchantID: merchant.ID, UserID: appointment.UserID, CardID: appointment.CardID, Status: "pending", SettlementStatusSnapshot: "pending", MerchantBreachPending: true}
	if err := db.Create(&settlement).Error; err != nil {
		t.Fatalf("create settlement failed: %v", err)
	}
	if err := db.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Updates(map[string]interface{}{"appointment_settlement_id": settlement.ID, "merchant_breach_pending": true}).Error; err != nil {
		t.Fatalf("bind settlement failed: %v", err)
	}
	session := models.ServiceSession{MerchantID: merchant.ID, InitialUsageID: usage.ID, Status: "timeout_waiting", SourceType: "appointment", SourceID: &appointment.ID}
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
	var gotAppointment struct {
		MerchantBreachPending           bool   `gorm:"column:merchant_breach_pending"`
		LiabilityLevel                  string `gorm:"column:liability_level"`
		SalarySettlementReferenceStatus string `gorm:"column:salary_settlement_reference_status"`
	}
	if err := db.Table("appointments").Select("merchant_breach_pending, liability_level, salary_settlement_reference_status").Where("id = ?", appointment.ID).Scan(&gotAppointment).Error; err != nil {
		t.Fatalf("load appointment failed: %v", err)
	}
	if gotAppointment.MerchantBreachPending || gotAppointment.LiabilityLevel != "merchant" || gotAppointment.SalarySettlementReferenceStatus != "refund" {
		t.Fatalf("want merchant refund liability snapshot, got %+v", gotAppointment)
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

func TestFailServiceSessionAndRefundSettlesLeaveDisruptionLiabilityCounter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_fail_leave_liability_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Card{}, &models.Appointment{}, &models.AppointmentSettlement{}, &models.Usage{}, &models.ServiceSession{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	merchant := models.Merchant{Name: "m-leave", Phone: "18800000011", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	tech := uint(88)

	createLeaveFailedAppointment := func(cardNo string) (models.Card, models.Appointment, models.Usage, models.ServiceSession) {
		card := models.Card{MerchantID: merchant.ID, CardNo: cardNo, CardType: "times", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
		if err := db.Create(&card).Error; err != nil {
			t.Fatalf("create card failed: %v", err)
		}
		usage := models.Usage{MerchantID: merchant.ID, CardID: card.ID, UsedTimes: 1, Status: "in_progress"}
		if err := db.Create(&usage).Error; err != nil {
			t.Fatalf("create usage failed: %v", err)
		}
		appointmentTime := now.Add(2 * time.Hour)
		appointment := models.Appointment{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, TechnicianID: &tech, Status: "arrived", AppointmentTime: &appointmentTime, MerchantBreachPending: true}
		if err := db.Create(&appointment).Error; err != nil {
			t.Fatalf("create appointment failed: %v", err)
		}
		settlement := models.AppointmentSettlement{AppointmentID: appointment.ID, MerchantID: merchant.ID, UserID: appointment.UserID, CardID: appointment.CardID, Status: "pending", SettlementStatusSnapshot: "pending", MerchantBreachPending: true}
		if err := db.Create(&settlement).Error; err != nil {
			t.Fatalf("create settlement failed: %v", err)
		}
		if err := db.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Updates(map[string]interface{}{"appointment_settlement_id": settlement.ID, "merchant_breach_pending": true}).Error; err != nil {
			t.Fatalf("bind settlement failed: %v", err)
		}
		repairSlot := models.ProtectedRepairSlot{MerchantID: merchant.ID, AppointmentID: appointment.ID, TechnicianID: &tech, Status: "reserved", SourceType: "leave"}
		if err := db.Create(&repairSlot).Error; err != nil {
			t.Fatalf("create repair slot failed: %v", err)
		}
		session := models.ServiceSession{MerchantID: merchant.ID, InitialUsageID: usage.ID, Status: "timeout_waiting", SourceType: "appointment", SourceID: &appointment.ID}
		if err := db.Create(&session).Error; err != nil {
			t.Fatalf("create session failed: %v", err)
		}
		return card, appointment, usage, session
	}

	_, appointment1, _, session1 := createLeaveFailedAppointment("L001")
	if err := FailServiceSessionAndRefund(db, &session1, &merchant, now, FailOptions{
		AllowedBaseStatuses: []string{"timeout_waiting"},
		TargetStatus:        "timeout_failed",
	}); err != nil {
		t.Fatalf("first FailServiceSessionAndRefund failed: %v", err)
	}
	var got1 struct {
		LiabilityLevel                  string `gorm:"column:liability_level"`
		SalarySettlementReferenceStatus string `gorm:"column:salary_settlement_reference_status"`
	}
	if err := db.Table("appointments").Select("liability_level, salary_settlement_reference_status").Where("id = ?", appointment1.ID).Scan(&got1).Error; err != nil {
		t.Fatalf("load first appointment failed: %v", err)
	}
	if got1.LiabilityLevel != "merchant_exempt" || got1.SalarySettlementReferenceStatus != "open" {
		t.Fatalf("want first leave disruption settled as merchant_exempt/open, got %+v", got1)
	}

	_, appointment2, _, session2 := createLeaveFailedAppointment("L002")
	if err := FailServiceSessionAndRefund(db, &session2, &merchant, now.Add(10*time.Minute), FailOptions{
		AllowedBaseStatuses: []string{"timeout_waiting"},
		TargetStatus:        "timeout_failed",
	}); err != nil {
		t.Fatalf("second FailServiceSessionAndRefund failed: %v", err)
	}
	var got2 struct {
		LiabilityLevel string `gorm:"column:liability_level"`
	}
	if err := db.Table("appointments").Select("liability_level").Where("id = ?", appointment2.ID).Scan(&got2).Error; err != nil {
		t.Fatalf("load second appointment failed: %v", err)
	}
	if got2.LiabilityLevel != "technician_chargeable" {
		t.Fatalf("want second leave disruption settled as technician_chargeable, got %+v", got2)
	}

	var counter models.TechnicianMonthlyDisruptionCounter
	if err := db.Where("merchant_id = ? AND technician_id = ? AND month_key = ?", merchant.ID, tech, "2026-03").First(&counter).Error; err != nil {
		t.Fatalf("load counter failed: %v", err)
	}
	if counter.LeaveDisruptionCount != 2 || !counter.FirstExemptUsed {
		t.Fatalf("want count=2 first_exempt_used=true, got %+v", counter)
	}

	var ledgerCount int64
	if err := db.Model(&models.TechnicianDisruptionLedger{}).Where("merchant_id = ? AND technician_id = ?", merchant.ID, tech).Count(&ledgerCount).Error; err != nil {
		t.Fatalf("count ledgers failed: %v", err)
	}
	if ledgerCount != 2 {
		t.Fatalf("want 2 disruption ledgers, got %d", ledgerCount)
	}
}
