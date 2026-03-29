package scheduler

import (
	"strings"
	"testing"
	"time"

	"kabao/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAppointmentSchedulerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Card{}, &models.Appointment{}, &models.AppointmentSettlement{}, &models.AppointmentCompensation{}, &models.AppointmentCancelRequest{}, &models.AppointmentRescheduleRequest{}, &models.Technician{}, &models.MerchantProject{}, &models.ProtectedRepairSlot{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestRunAppointmentNoShowOnceMarksConfirmedAppointmentAsNoShow(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)

	now := time.Now()
	merchant := models.Merchant{
		Name:                            "appointment-no-show",
		Phone:                           "18800000201",
		Password:                        "pwd",
		SupportAppointment:              true,
		AppointmentGraceWindowMinutes:   15,
		AppointmentReserveBufferMinutes: 10,
	}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	appointmentTime := now.Add(-20 * time.Minute)
	appointment := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          1,
		CardID:          1,
		Status:          "confirmed",
		AppointmentTime: &appointmentTime,
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

	if err := runAppointmentNoShowOnce(db, now); err != nil {
		t.Fatalf("runAppointmentNoShowOnce failed: %v", err)
	}

	var got struct {
		Status                          string `gorm:"column:status"`
		NoShowAtRaw                     string `gorm:"column:no_show_at"`
		LiabilityLevel                  string `gorm:"column:liability_level"`
		SalarySettlementReferenceStatus string `gorm:"column:salary_settlement_reference_status"`
	}
	if err := db.Table("appointments").Select("status, no_show_at, liability_level, salary_settlement_reference_status").Where("id = ?", appointment.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload appointment failed: %v", err)
	}
	if got.Status != "no_show" {
		t.Fatalf("want no_show, got %s", got.Status)
	}
	if got.NoShowAtRaw == "" {
		t.Fatalf("want no_show_at filled")
	}
	if got.LiabilityLevel != "user" || got.SalarySettlementReferenceStatus != "no_pay" {
		t.Fatalf("want user no_show liability snapshot, got %+v", got)
	}
}

func TestRunAppointmentNoShowOnceMarksUnassignedCustomerServiceAppointmentAsInconsistent(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)

	now := time.Now()
	merchant := models.Merchant{
		Name:                            "appointment-no-show-unassigned",
		Phone:                           "18800000209",
		Password:                        "pwd",
		SupportAppointment:              true,
		SupportCustomerServiceMode:      true,
		AppointmentGraceWindowMinutes:   15,
		AppointmentReserveBufferMinutes: 10,
	}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	appointmentTime := now.Add(-20 * time.Minute)
	appointment := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          1,
		CardID:          1,
		Status:          "confirmed",
		AppointmentTime: &appointmentTime,
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

	if err := runAppointmentNoShowOnce(db, now); err != nil {
		t.Fatalf("runAppointmentNoShowOnce failed: %v", err)
	}

	var got struct {
		Status                          string `gorm:"column:status"`
		NoShowAtRaw                     string `gorm:"column:no_show_at"`
		DisruptionReason                string `gorm:"column:disruption_reason"`
		LiabilityLevel                  string `gorm:"column:liability_level"`
		SalarySettlementReferenceStatus string `gorm:"column:salary_settlement_reference_status"`
		ResolutionNote                  string `gorm:"column:resolution_note"`
	}
	if err := db.Table("appointments").Select("status, no_show_at, disruption_reason, liability_level, salary_settlement_reference_status, resolution_note").Where("id = ?", appointment.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload appointment failed: %v", err)
	}
	if got.Status != "no_show" {
		t.Fatalf("want no_show, got %s", got.Status)
	}
	if got.NoShowAtRaw == "" {
		t.Fatalf("want no_show_at filled")
	}
	if got.DisruptionReason != "appointment_state_inconsistent" {
		t.Fatalf("want disruption_reason=appointment_state_inconsistent, got %s", got.DisruptionReason)
	}
	if got.LiabilityLevel != "pending_merchant" || got.SalarySettlementReferenceStatus != "pending" {
		t.Fatalf("want pending_merchant/pending snapshot, got %+v", got)
	}
	if !strings.Contains(got.ResolutionNote, "未分配客服") {
		t.Fatalf("want resolution note mention unassigned technician, got %q", got.ResolutionNote)
	}
}

func TestRunAppointmentNoShowOnceWaitsForLateArrivalServiceThreshold(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)

	now := time.Date(2026, 3, 29, 10, 50, 0, 0, time.UTC)
	merchant := models.Merchant{
		Name:                            "appointment-no-show-threshold",
		Phone:                           "18800000219",
		Password:                        "pwd",
		SupportAppointment:              true,
		AppointmentGraceWindowMinutes:   15,
		AppointmentReserveBufferMinutes: 10,
	}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	appointmentTime := time.Date(2026, 3, 29, 10, 30, 0, 0, time.UTC)
	reservedEndAt := appointmentTime.Add(45 * time.Minute)
	appointment := models.Appointment{
		MerchantID:                      merchant.ID,
		UserID:                          1,
		CardID:                          1,
		Status:                          "confirmed",
		AppointmentTime:                 &appointmentTime,
		ReservedEndAt:                   &reservedEndAt,
		LateArrivalMinServiceMinutes:    22,
		SettlementStatusSnapshot:        "pending",
		SalarySettlementReferenceStatus: "pending",
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

	if err := runAppointmentNoShowOnce(db, now); err != nil {
		t.Fatalf("runAppointmentNoShowOnce failed: %v", err)
	}

	var status string
	if err := db.Table("appointments").Select("status").Where("id = ?", appointment.ID).Scan(&status).Error; err != nil {
		t.Fatalf("reload appointment failed: %v", err)
	}
	if status != "confirmed" {
		t.Fatalf("want confirmed before late-arrival threshold, got %s", status)
	}

	if err := runAppointmentNoShowOnce(db, time.Date(2026, 3, 29, 10, 54, 0, 0, time.UTC)); err != nil {
		t.Fatalf("runAppointmentNoShowOnce after threshold failed: %v", err)
	}
	if err := db.Table("appointments").Select("status").Where("id = ?", appointment.ID).Scan(&status).Error; err != nil {
		t.Fatalf("reload appointment after threshold failed: %v", err)
	}
	if status != "no_show" {
		t.Fatalf("want no_show after late-arrival threshold, got %s", status)
	}
}

func TestRunAppointmentAssignOnceBackfillsConfirmedAppointmentWithoutTechnician(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)

	now := time.Now()
	merchant := models.Merchant{
		Name:                            "appointment-backfill",
		Phone:                           "18800000202",
		Password:                        "pwd",
		SupportAppointment:              true,
		SupportCustomerServiceMode:      true,
		AppointmentGraceWindowMinutes:   15,
		AppointmentReserveBufferMinutes: 10,
	}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	roleMerchantID := merchant.ID
	role := models.ServiceRole{Name: "专业客服", Key: "professional", RoleType: "professional", MerchantID: &roleMerchantID}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	tech := models.Technician{MerchantID: merchant.ID, Name: "客服A", Account: "js0001", IsActive: true, ServiceRoleID: role.ID}
	if err := db.Create(&tech).Error; err != nil {
		t.Fatalf("create technician failed: %v", err)
	}

	appointmentTime := now.Add(2 * time.Hour).Truncate(time.Second)
	appointment := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          1,
		CardID:          1,
		Status:          "confirmed",
		AppointmentTime: &appointmentTime,
	}
	if err := db.Create(&appointment).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	if err := runAppointmentAssignOnce(db); err != nil {
		t.Fatalf("runAppointmentAssignOnce failed: %v", err)
	}

	var got struct {
		Status       string `gorm:"column:status"`
		TechnicianID *uint  `gorm:"column:technician_id"`
	}
	if err := db.Table("appointments").Select("status, technician_id").Where("id = ?", appointment.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload appointment failed: %v", err)
	}
	if got.Status != "confirmed" {
		t.Fatalf("want confirmed, got %s", got.Status)
	}
	if got.TechnicianID == nil || *got.TechnicianID != tech.ID {
		t.Fatalf("want technician %d assigned, got %+v", tech.ID, got.TechnicianID)
	}
}

func TestRunMerchantBreachCompensationSettlesArrivedAppointmentToMerchantBreach(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)
	now := time.Date(2026, 3, 23, 18, 0, 0, 0, time.UTC)

	merchant := models.Merchant{Name: "appointment-breach", Phone: "18800000203", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	card := models.Card{MerchantID: merchant.ID, CardNo: "B001", CardType: "times", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	arrivedAt := now.Add(-15 * time.Minute)
	decisionAt := now.Add(-1 * time.Minute)
	appt := models.Appointment{
		MerchantID:            merchant.ID,
		UserID:                1,
		CardID:                card.ID,
		Status:                "arrived",
		ActualArrivedAt:       &arrivedAt,
		MerchantBreachPending: true,
		BreachDecisionAt:      &decisionAt,
		LiabilityLevel:        "pending_merchant",
	}
	if err := db.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	settlement := models.AppointmentSettlement{AppointmentID: appt.ID, MerchantID: merchant.ID, UserID: appt.UserID, CardID: appt.CardID, Status: "pending", SettlementStatusSnapshot: "pending", MerchantBreachPending: true}
	if err := db.Create(&settlement).Error; err != nil {
		t.Fatalf("create settlement failed: %v", err)
	}
	if err := db.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("appointment_settlement_id", settlement.ID).Error; err != nil {
		t.Fatalf("bind settlement failed: %v", err)
	}

	if err := runMerchantBreachCompensation(db, now); err != nil {
		t.Fatalf("runMerchantBreachCompensation failed: %v", err)
	}

	var got struct {
		MerchantBreachPending bool   `gorm:"column:merchant_breach_pending"`
		DisruptionReason      string `gorm:"column:disruption_reason"`
		LiabilityLevel        string `gorm:"column:liability_level"`
	}
	if err := db.Table("appointments").Select("merchant_breach_pending, disruption_reason, liability_level").Where("id = ?", appt.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload appointment failed: %v", err)
	}
	if got.MerchantBreachPending || got.DisruptionReason != "merchant_breach" || got.LiabilityLevel != "merchant" {
		t.Fatalf("want merchant_breach settled, got %+v", got)
	}
	var gotCard models.Card
	if err := db.First(&gotCard, card.ID).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotCard.TotalTimes != 12 || gotCard.RemainTimes != 11 {
		t.Fatalf("want breach compensation +2 times, got total=%d remain=%d", gotCard.TotalTimes, gotCard.RemainTimes)
	}
	var count int64
	if err := db.Model(&models.AppointmentCompensation{}).Where("appointment_id = ? AND source_type = ?", appt.ID, "merchant_breach").Count(&count).Error; err != nil {
		t.Fatalf("count compensation failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("want 1 merchant_breach compensation, got %d", count)
	}
}

func TestRunMerchantBreachCompensationSettlesFormalMitigationToFailureOffset(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)
	now := time.Date(2026, 3, 23, 18, 30, 0, 0, time.UTC)

	merchant := models.Merchant{Name: "appointment-offset", Phone: "18800000204", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	card := models.Card{MerchantID: merchant.ID, CardNo: "B002", CardType: "times", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	decisionAt := now.Add(-1 * time.Minute)
	appt := models.Appointment{
		MerchantID:            merchant.ID,
		UserID:                1,
		CardID:                card.ID,
		Status:                "confirmed",
		MerchantBreachPending: true,
		BreachDecisionAt:      &decisionAt,
		LiabilityLevel:        "pending_merchant",
	}
	if err := db.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	settlement := models.AppointmentSettlement{AppointmentID: appt.ID, MerchantID: merchant.ID, UserID: appt.UserID, CardID: appt.CardID, Status: "pending", SettlementStatusSnapshot: "pending", MerchantBreachPending: true}
	if err := db.Create(&settlement).Error; err != nil {
		t.Fatalf("create settlement failed: %v", err)
	}
	if err := db.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("appointment_settlement_id", settlement.ID).Error; err != nil {
		t.Fatalf("bind settlement failed: %v", err)
	}
	req := models.AppointmentCancelRequest{
		AppointmentID:  appt.ID,
		MerchantID:     merchant.ID,
		UserID:         appt.UserID,
		Status:         "pending_user",
		Reason:         "门店无法履约",
		ProposedByType: "merchant",
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create mitigation request failed: %v", err)
	}

	if err := runMerchantBreachCompensation(db, now); err != nil {
		t.Fatalf("runMerchantBreachCompensation failed: %v", err)
	}

	var got struct {
		MerchantBreachPending bool   `gorm:"column:merchant_breach_pending"`
		DisruptionReason      string `gorm:"column:disruption_reason"`
	}
	if err := db.Table("appointments").Select("merchant_breach_pending, disruption_reason").Where("id = ?", appt.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload appointment failed: %v", err)
	}
	if got.MerchantBreachPending || got.DisruptionReason != "merchant_failure_offset" {
		t.Fatalf("want merchant_failure_offset settled, got %+v", got)
	}
	var gotCard models.Card
	if err := db.First(&gotCard, card.ID).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotCard.TotalTimes != 11 || gotCard.RemainTimes != 10 {
		t.Fatalf("want offset compensation +1 time, got total=%d remain=%d", gotCard.TotalTimes, gotCard.RemainTimes)
	}
}

func TestRunAppointmentDelayLedgerSettlementRedeemsMinutesBucket(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)
	now := time.Date(2026, 3, 23, 19, 0, 0, 0, time.UTC)

	merchant := models.Merchant{Name: "appointment-delay-redeem", Phone: "18800000205", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:                  merchant.ID,
		Name:                        "肩颈",
		Duration:                    60,
		DelayCompensationMode:       "minutes_bucket",
		DelayRedeemThresholdPercent: 100,
		DelayToleranceMinutes:       1,
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	card := models.Card{MerchantID: merchant.ID, CardNo: "DL01", CardType: "times", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	appt1 := models.Appointment{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, Status: "completed"}
	appt2 := models.Appointment{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, Status: "completed"}
	if err := db.Create(&appt1).Error; err != nil {
		t.Fatalf("create appt1 failed: %v", err)
	}
	if err := db.Create(&appt2).Error; err != nil {
		t.Fatalf("create appt2 failed: %v", err)
	}
	ledger1 := models.AppointmentDelayLedger{AppointmentID: appt1.ID, MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, DelayMinutes: 25, CreditedMinutes: 25, LedgerStatus: "recorded", RedeemStatus: "pending"}
	ledger2 := models.AppointmentDelayLedger{AppointmentID: appt2.ID, MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, DelayMinutes: 35, CreditedMinutes: 35, LedgerStatus: "recorded", RedeemStatus: "pending"}
	if err := db.Create(&ledger1).Error; err != nil {
		t.Fatalf("create ledger1 failed: %v", err)
	}
	if err := db.Create(&ledger2).Error; err != nil {
		t.Fatalf("create ledger2 failed: %v", err)
	}

	if err := runAppointmentDelayLedgerSettlement(db, now); err != nil {
		t.Fatalf("runAppointmentDelayLedgerSettlement failed: %v", err)
	}

	var gotCard models.Card
	if err := db.First(&gotCard, card.ID).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotCard.TotalTimes != 11 || gotCard.RemainTimes != 10 {
		t.Fatalf("want redeemed +1 time, got total=%d remain=%d", gotCard.TotalTimes, gotCard.RemainTimes)
	}
	var comp struct {
		SourceType string `gorm:"column:source_type"`
		Value      int    `gorm:"column:value"`
	}
	if err := db.Table("appointment_compensations").Select("source_type, value").Where("appointment_id = ?", appt2.ID).Take(&comp).Error; err != nil {
		t.Fatalf("load compensation failed: %v", err)
	}
	if comp.SourceType != "delay_bucket_redeem" || comp.Value != 1 {
		t.Fatalf("want delay bucket redeem compensation, got %+v", comp)
	}
	var ledgerStatuses []struct {
		RedeemStatus string `gorm:"column:redeem_status"`
		LedgerStatus string `gorm:"column:ledger_status"`
	}
	if err := db.Table("appointment_delay_ledgers").Select("redeem_status, ledger_status").Order("id asc").Scan(&ledgerStatuses).Error; err != nil {
		t.Fatalf("load ledger statuses failed: %v", err)
	}
	if len(ledgerStatuses) != 2 || ledgerStatuses[0].RedeemStatus != "redeemed" || ledgerStatuses[1].RedeemStatus != "redeemed" {
		t.Fatalf("want ledgers redeemed, got %+v", ledgerStatuses)
	}
}

func TestRunAppointmentDelayLedgerSettlementKeepsFixedUnitRemainderPending(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)
	now := time.Date(2026, 3, 23, 19, 10, 0, 0, time.UTC)

	merchant := models.Merchant{Name: "appointment-delay-fixed", Phone: "18800000206", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:                  merchant.ID,
		Name:                        "面部",
		Duration:                    60,
		DelayCompensationMode:       "fixed_unit",
		DelayRedeemThresholdPercent: 50,
		DelayToleranceMinutes:       1,
		DelayFixedUnitValue:         2,
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	card := models.Card{MerchantID: merchant.ID, CardNo: "DL02", CardType: "times", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	appt1 := models.Appointment{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, Status: "completed"}
	appt2 := models.Appointment{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, Status: "completed"}
	if err := db.Create(&appt1).Error; err != nil {
		t.Fatalf("create appt1 failed: %v", err)
	}
	if err := db.Create(&appt2).Error; err != nil {
		t.Fatalf("create appt2 failed: %v", err)
	}
	ledger1 := models.AppointmentDelayLedger{AppointmentID: appt1.ID, MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, DelayMinutes: 20, CreditedMinutes: 20, LedgerStatus: "recorded", RedeemStatus: "pending"}
	ledger2 := models.AppointmentDelayLedger{AppointmentID: appt2.ID, MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, DelayMinutes: 15, CreditedMinutes: 15, LedgerStatus: "recorded", RedeemStatus: "pending"}
	if err := db.Create(&ledger1).Error; err != nil {
		t.Fatalf("create ledger1 failed: %v", err)
	}
	if err := db.Create(&ledger2).Error; err != nil {
		t.Fatalf("create ledger2 failed: %v", err)
	}

	if err := runAppointmentDelayLedgerSettlement(db, now); err != nil {
		t.Fatalf("runAppointmentDelayLedgerSettlement failed: %v", err)
	}

	var gotCard models.Card
	if err := db.First(&gotCard, card.ID).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotCard.TotalTimes != 12 || gotCard.RemainTimes != 11 {
		t.Fatalf("want fixed unit redeem +2 times, got total=%d remain=%d", gotCard.TotalTimes, gotCard.RemainTimes)
	}
	var comp struct {
		SourceType string `gorm:"column:source_type"`
		Value      int    `gorm:"column:value"`
	}
	if err := db.Table("appointment_compensations").Select("source_type, value").Where("appointment_id = ?", appt2.ID).Take(&comp).Error; err != nil {
		t.Fatalf("load compensation failed: %v", err)
	}
	if comp.SourceType != "delay_bucket_redeem" || comp.Value != 2 {
		t.Fatalf("want fixed unit compensation value 2, got %+v", comp)
	}
	var ledgers []struct {
		ID              uint   `gorm:"column:id"`
		CreditedMinutes int    `gorm:"column:credited_minutes"`
		RedeemStatus    string `gorm:"column:redeem_status"`
		LedgerStatus    string `gorm:"column:ledger_status"`
	}
	if err := db.Table("appointment_delay_ledgers").Select("id, credited_minutes, redeem_status, ledger_status").Order("id asc").Scan(&ledgers).Error; err != nil {
		t.Fatalf("load ledgers failed: %v", err)
	}
	if len(ledgers) != 2 {
		t.Fatalf("want 2 ledgers, got %+v", ledgers)
	}
	if ledgers[0].RedeemStatus != "redeemed" || ledgers[0].LedgerStatus != "redeemed" {
		t.Fatalf("want first ledger fully redeemed, got %+v", ledgers[0])
	}
	if ledgers[1].RedeemStatus != "pending" || ledgers[1].LedgerStatus != "recorded" || ledgers[1].CreditedMinutes != 5 {
		t.Fatalf("want second ledger keep 5-minute remainder pending, got %+v", ledgers[1])
	}
}

func TestRunAppointmentDelayLedgerSettlementRedeemsAmountBucketToBalanceCard(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)
	now := time.Date(2026, 3, 23, 19, 20, 0, 0, time.UTC)

	merchant := models.Merchant{Name: "appointment-delay-amount", Phone: "18800000207", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:                  merchant.ID,
		Name:                        "面部",
		Duration:                    80,
		DelayCompensationMode:       "amount_bucket",
		DelayRedeemThresholdPercent: 100,
		DelayToleranceMinutes:       1,
		DelayFixedUnitValue:         100,
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	card := models.Card{MerchantID: merchant.ID, CardNo: "DL03", CardType: "充值卡", RechargeAmount: 1000}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	appt1 := models.Appointment{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, Status: "completed"}
	appt2 := models.Appointment{MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, Status: "completed"}
	if err := db.Create(&appt1).Error; err != nil {
		t.Fatalf("create appt1 failed: %v", err)
	}
	if err := db.Create(&appt2).Error; err != nil {
		t.Fatalf("create appt2 failed: %v", err)
	}
	ledger1 := models.AppointmentDelayLedger{AppointmentID: appt1.ID, MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, DelayMinutes: 10, CreditedMinutes: 10, DelayCompensationValue: 40, LedgerStatus: "recorded", RedeemStatus: "pending"}
	ledger2 := models.AppointmentDelayLedger{AppointmentID: appt2.ID, MerchantID: merchant.ID, UserID: 1, CardID: card.ID, ProjectID: &project.ID, DelayMinutes: 15, CreditedMinutes: 15, DelayCompensationValue: 70, LedgerStatus: "recorded", RedeemStatus: "pending"}
	if err := db.Create(&ledger1).Error; err != nil {
		t.Fatalf("create ledger1 failed: %v", err)
	}
	if err := db.Create(&ledger2).Error; err != nil {
		t.Fatalf("create ledger2 failed: %v", err)
	}

	if err := runAppointmentDelayLedgerSettlement(db, now); err != nil {
		t.Fatalf("runAppointmentDelayLedgerSettlement failed: %v", err)
	}

	var gotCard models.Card
	if err := db.First(&gotCard, card.ID).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotCard.RechargeAmount != 1100 {
		t.Fatalf("want balance card recharge_amount +100, got %d", gotCard.RechargeAmount)
	}
	var comp struct {
		Type       string `gorm:"column:type"`
		SourceType string `gorm:"column:source_type"`
		Value      int    `gorm:"column:value"`
	}
	if err := db.Table("appointment_compensations").Select("type, source_type, value").Where("appointment_id = ?", appt2.ID).Take(&comp).Error; err != nil {
		t.Fatalf("load compensation failed: %v", err)
	}
	if comp.Type != "manual_adjustment" || comp.SourceType != "delay_bucket_redeem" || comp.Value != 100 {
		t.Fatalf("want balance compensation 100, got %+v", comp)
	}
	var ledgers []struct {
		DelayCompensationValue int    `gorm:"column:delay_compensation_value"`
		RedeemStatus           string `gorm:"column:redeem_status"`
		LedgerStatus           string `gorm:"column:ledger_status"`
	}
	if err := db.Table("appointment_delay_ledgers").Select("delay_compensation_value, redeem_status, ledger_status").Order("id asc").Scan(&ledgers).Error; err != nil {
		t.Fatalf("load ledgers failed: %v", err)
	}
	if len(ledgers) != 2 {
		t.Fatalf("want 2 ledgers, got %+v", ledgers)
	}
	if ledgers[0].RedeemStatus != "redeemed" || ledgers[0].LedgerStatus != "redeemed" {
		t.Fatalf("want first amount ledger redeemed, got %+v", ledgers[0])
	}
	if ledgers[1].RedeemStatus != "pending" || ledgers[1].LedgerStatus != "recorded" || ledgers[1].DelayCompensationValue != 30 {
		t.Fatalf("want second amount ledger keep 30 remainder pending, got %+v", ledgers[1])
	}
}
