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
	if err := db.AutoMigrate(&models.Merchant{}, &models.Usage{}, &models.ServiceSession{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	now := time.Now()
	merchant := models.Merchant{Name: "m", Phone: "18800000009", Password: "pwd", SupportQueue: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	techID := uint(11)
	verifierID := uint(12)
	roomID := uint(22)
	usage := models.Usage{MerchantID: merchant.ID, Status: "in_progress", TechnicianID: &verifierID}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:                  merchant.ID,
		InitialUsageID:              usage.ID,
		Status:                      "serving",
		LastTechnicianID:            &techID,
		ServiceTechnicianIDs:        models.MerchantProjectDefaultServiceTechnicianIDs{techID},
		StartConfirmedTechnicianIDs: models.MerchantProjectDefaultServiceTechnicianIDs{techID},
		RoomID:                      &roomID,
		StartConfirmedAt:            &now,
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
	if gotUsage.TechnicianID == nil || *gotUsage.TechnicianID != verifierID {
		t.Fatalf("finish must preserve usage verifier technician_id=%d, got %+v", verifierID, gotUsage.TechnicianID)
	}
	if len(stub.doneIDs) != 1 || stub.doneIDs[0] != usage.ID {
		t.Fatalf("want queue MarkDone on usage %d, got %+v", usage.ID, stub.doneIDs)
	}
}

func TestFinalizeUsageAndSessionCompletesUnstartedSessionAndMarksQueueDone(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_finish_finalize_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Usage{}, &models.ServiceSession{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	now := time.Now()
	merchant := models.Merchant{Name: "m2", Phone: "18800000019", Password: "pwd", SupportQueue: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	usage := models.Usage{MerchantID: merchant.ID, Status: "in_progress"}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:     merchant.ID,
		InitialUsageID: usage.ID,
		Status:         "qms_staff_selecting",
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	oldQueue := queue.Default
	defer func() { queue.Default = oldQueue }()
	stub := &stubQueueStore{}
	queue.Default = stub

	handled, err := FinalizeUsageAndSession(db, usage.ID, &merchant, now, FinishOptions{MarkQueueDone: true})
	if err != nil {
		t.Fatalf("FinalizeUsageAndSession failed: %v", err)
	}
	if !handled {
		t.Fatalf("want linked session handled")
	}

	var gotSession struct {
		Status string `gorm:"column:status"`
	}
	if err := db.Model(&models.ServiceSession{}).Select("status").First(&gotSession, session.ID).Error; err != nil {
		t.Fatalf("load session failed: %v", err)
	}
	if gotSession.Status != "qms_finished" {
		t.Fatalf("want qms_finished session, got %s", gotSession.Status)
	}

	var gotUsage struct {
		Status string `gorm:"column:status"`
	}
	if err := db.Model(&models.Usage{}).Select("status").First(&gotUsage, usage.ID).Error; err != nil {
		t.Fatalf("load usage failed: %v", err)
	}
	if gotUsage.Status != "success" {
		t.Fatalf("want success usage, got %s", gotUsage.Status)
	}
	if len(stub.doneIDs) != 1 || stub.doneIDs[0] != usage.ID {
		t.Fatalf("want queue MarkDone on usage %d, got %+v", usage.ID, stub.doneIDs)
	}
}

func TestFinalizeUsageAndSessionReturnsFalseWhenSessionMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_finalize_missing_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Usage{}, &models.ServiceSession{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	usage := models.Usage{MerchantID: 1, Status: "in_progress"}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}

	handled, err := FinalizeUsageAndSession(db, usage.ID, nil, time.Now(), FinishOptions{})
	if err != nil {
		t.Fatalf("FinalizeUsageAndSession failed: %v", err)
	}
	if handled {
		t.Fatalf("want missing session to be ignored")
	}
}

func TestFinishServiceSessionMarksLinkedAppointmentCompleted(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_finish_appointment_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Appointment{}, &models.AppointmentSettlement{}, &models.Usage{}, &models.ServiceSession{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}, &models.MerchantProject{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	now := time.Now()
	merchant := models.Merchant{Name: "m-appointment", Phone: "18800000029", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	appointmentTime := now.Add(-10 * time.Minute)
	appointment := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          1,
		CardID:          1,
		Status:          "arrived",
		AppointmentTime: &appointmentTime,
	}
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
	usage := models.Usage{MerchantID: merchant.ID, Status: "in_progress"}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:       merchant.ID,
		InitialUsageID:   usage.ID,
		Status:           "serving",
		SourceType:       "appointment",
		SourceID:         &appointment.ID,
		StartConfirmedAt: &now,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := FinishServiceSession(db, &session, &merchant, now, FinishOptions{}); err != nil {
		t.Fatalf("FinishServiceSession failed: %v", err)
	}

	var got struct {
		Status                          string `gorm:"column:status"`
		CompletedAtRaw                  string `gorm:"column:completed_at"`
		MerchantBreachPending           bool   `gorm:"column:merchant_breach_pending"`
		LiabilityLevel                  string `gorm:"column:liability_level"`
		SalarySettlementReferenceStatus string `gorm:"column:salary_settlement_reference_status"`
	}
	if err := db.Table("appointments").Select("status, completed_at, merchant_breach_pending, liability_level, salary_settlement_reference_status").Where("id = ?", appointment.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload appointment failed: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("want completed appointment, got %s", got.Status)
	}
	if got.CompletedAtRaw == "" {
		t.Fatalf("want completed_at filled")
	}
	if got.MerchantBreachPending || got.LiabilityLevel != "none" || got.SalarySettlementReferenceStatus != "normal" {
		t.Fatalf("want completed non-breach snapshot, got %+v", got)
	}
}

func TestFinishServiceSessionCreatesDelayLedgerWhenActualStartIsLate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_finish_delay_ledger_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Appointment{}, &models.AppointmentSettlement{}, &models.Usage{}, &models.ServiceSession{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}, &models.MerchantProject{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	now := time.Date(2026, 3, 23, 12, 20, 0, 0, time.UTC)
	merchant := models.Merchant{Name: "m-delay", Phone: "18800000039", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	project := models.MerchantProject{MerchantID: merchant.ID, Name: "肩颈", Duration: 60, DelayToleranceMinutes: 1, DelayCompensationMode: "minutes_bucket"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	reservedStart := now.Add(-20 * time.Minute)
	actualStart := reservedStart.Add(8 * time.Minute)
	appointment := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          1,
		CardID:          1,
		ProjectID:       &project.ID,
		Status:          "arrived",
		ReservedStartAt: &reservedStart,
		ActualStartAt:   &actualStart,
	}
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
	usage := models.Usage{MerchantID: merchant.ID, Status: "in_progress"}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:       merchant.ID,
		InitialUsageID:   usage.ID,
		Status:           "serving",
		SourceType:       "appointment",
		SourceID:         &appointment.ID,
		StartConfirmedAt: &actualStart,
		StartedAt:        &actualStart,
		DurationMinutes:  60,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := FinishServiceSession(db, &session, &merchant, now, FinishOptions{}); err != nil {
		t.Fatalf("FinishServiceSession failed: %v", err)
	}

	var ledger struct {
		DelayMinutes    int `gorm:"column:delay_minutes"`
		CreditedMinutes int `gorm:"column:credited_minutes"`
	}
	if err := db.Table("appointment_delay_ledgers").Select("delay_minutes, credited_minutes").Where("appointment_id = ?", appointment.ID).Scan(&ledger).Error; err != nil {
		t.Fatalf("load delay ledger failed: %v", err)
	}
	if ledger.DelayMinutes != 8 || ledger.CreditedMinutes != 8 {
		t.Fatalf("want 8 minute delay ledger, got %+v", ledger)
	}
}

func TestFinishServiceSessionSkipsDelayLedgerWithinTolerance(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_finish_delay_tolerance_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Appointment{}, &models.AppointmentSettlement{}, &models.Usage{}, &models.ServiceSession{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}, &models.MerchantProject{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	now := time.Date(2026, 3, 23, 12, 20, 0, 0, time.UTC)
	merchant := models.Merchant{Name: "m-delay-tolerance", Phone: "18800000040", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	project := models.MerchantProject{MerchantID: merchant.ID, Name: "肩颈", Duration: 60, DelayToleranceMinutes: 5, DelayCompensationMode: "minutes_bucket"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	reservedStart := now.Add(-20 * time.Minute)
	actualStart := reservedStart.Add(4 * time.Minute)
	appointment := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          1,
		CardID:          1,
		ProjectID:       &project.ID,
		Status:          "arrived",
		ReservedStartAt: &reservedStart,
		ActualStartAt:   &actualStart,
	}
	if err := db.Create(&appointment).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	settlement := models.AppointmentSettlement{AppointmentID: appointment.ID, MerchantID: merchant.ID, UserID: appointment.UserID, CardID: appointment.CardID, Status: "pending", SettlementStatusSnapshot: "pending"}
	if err := db.Create(&settlement).Error; err != nil {
		t.Fatalf("create settlement failed: %v", err)
	}
	if err := db.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Update("appointment_settlement_id", settlement.ID).Error; err != nil {
		t.Fatalf("bind settlement failed: %v", err)
	}
	usage := models.Usage{MerchantID: merchant.ID, Status: "in_progress"}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:       merchant.ID,
		InitialUsageID:   usage.ID,
		Status:           "serving",
		SourceType:       "appointment",
		SourceID:         &appointment.ID,
		StartConfirmedAt: &actualStart,
		StartedAt:        &actualStart,
		DurationMinutes:  60,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := FinishServiceSession(db, &session, &merchant, now, FinishOptions{}); err != nil {
		t.Fatalf("FinishServiceSession failed: %v", err)
	}

	var count int64
	if err := db.Model(&models.AppointmentDelayLedger{}).Where("appointment_id = ?", appointment.ID).Count(&count).Error; err != nil {
		t.Fatalf("count delay ledgers failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("want no delay ledger within tolerance, got %d", count)
	}
}

func TestFinishServiceSessionCreatesAmountBucketDelayLedger(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sessionflow_finish_delay_amount_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Appointment{}, &models.AppointmentSettlement{}, &models.Usage{}, &models.ServiceSession{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}, &models.MerchantProject{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	now := time.Date(2026, 3, 23, 13, 20, 0, 0, time.UTC)
	merchant := models.Merchant{Name: "m-delay-amount", Phone: "18800000041", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	project := models.MerchantProject{MerchantID: merchant.ID, Name: "面部", Duration: 80, DelayToleranceMinutes: 1, DelayCompensationMode: "amount_bucket", DelayFixedUnitValue: 100}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	reservedStart := now.Add(-20 * time.Minute)
	actualStart := reservedStart.Add(10 * time.Minute)
	appointment := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          1,
		CardID:          1,
		ProjectID:       &project.ID,
		Status:          "arrived",
		ReservedStartAt: &reservedStart,
		ActualStartAt:   &actualStart,
	}
	if err := db.Create(&appointment).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	settlement := models.AppointmentSettlement{AppointmentID: appointment.ID, MerchantID: merchant.ID, UserID: appointment.UserID, CardID: appointment.CardID, Status: "pending", SettlementStatusSnapshot: "pending"}
	if err := db.Create(&settlement).Error; err != nil {
		t.Fatalf("create settlement failed: %v", err)
	}
	if err := db.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Update("appointment_settlement_id", settlement.ID).Error; err != nil {
		t.Fatalf("bind settlement failed: %v", err)
	}
	usage := models.Usage{MerchantID: merchant.ID, Status: "in_progress"}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:       merchant.ID,
		InitialUsageID:   usage.ID,
		Status:           "serving",
		SourceType:       "appointment",
		SourceID:         &appointment.ID,
		StartConfirmedAt: &actualStart,
		StartedAt:        &actualStart,
		DurationMinutes:  80,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := FinishServiceSession(db, &session, &merchant, now, FinishOptions{}); err != nil {
		t.Fatalf("FinishServiceSession failed: %v", err)
	}

	var ledger struct {
		DelayMinutes           int `gorm:"column:delay_minutes"`
		CreditedMinutes        int `gorm:"column:credited_minutes"`
		DelayCompensationValue int `gorm:"column:delay_compensation_value"`
	}
	if err := db.Table("appointment_delay_ledgers").Select("delay_minutes, credited_minutes, delay_compensation_value").Where("appointment_id = ?", appointment.ID).Scan(&ledger).Error; err != nil {
		t.Fatalf("load delay ledger failed: %v", err)
	}
	if ledger.DelayMinutes != 10 || ledger.CreditedMinutes != 10 || ledger.DelayCompensationValue != 13 {
		t.Fatalf("want amount bucket ledger value ceil(100*10/80)=13, got %+v", ledger)
	}
}
