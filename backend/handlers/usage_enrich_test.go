package handlers

import (
	"kabao/config"
	"kabao/models"
	"testing"
	"time"
)

func TestResolveSessionStartConfirmedAtBackfillsWhenServing(t *testing.T) {
	startedAt := time.Now().Add(-3 * time.Minute)

	got := resolveSessionStartConfirmedAt("serving", nil, &startedAt)
	if got == nil {
		t.Fatalf("want non-nil start_confirmed_at when serving and started_at exists")
	}
	if !got.Equal(startedAt) {
		t.Fatalf("want start_confirmed_at=started_at, got %v want %v", got, startedAt)
	}
}

func TestResolveSessionStartConfirmedAtKeepsNilOutsideServingStages(t *testing.T) {
	startedAt := time.Now().Add(-3 * time.Minute)

	got := resolveSessionStartConfirmedAt("start_pending", nil, &startedAt)
	if got != nil {
		t.Fatalf("want nil for non-serving stages, got %v", got)
	}
}

func TestShouldFallbackServiceTechnicianToLastSkipsStaffSelectingTimeout(t *testing.T) {
	got := shouldFallbackServiceTechnicianToLast("cs_staff_selecting", nil, nil, nil)
	if got {
		t.Fatalf("want historical technician hidden while waiting for reassignment")
	}
}

func TestShouldFallbackServiceTechnicianToLastKeepsStartedSessionHistory(t *testing.T) {
	startedAt := time.Now().Add(-3 * time.Minute)

	got := shouldFallbackServiceTechnicianToLast("cs_finished", nil, &startedAt, nil)
	if !got {
		t.Fatalf("want historical technician kept after service has started")
	}
}

func TestEnrichUsagesWithServiceSessionAddsAppointmentID(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionFactoryTestDB(t)

	now := time.Now()
	usage := models.Usage{MerchantID: 1, CardID: 1, UsedTimes: 1, UsedAt: &now, Status: "in_progress"}
	if err := config.DB.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	appointmentID := uint(92)
	session := models.ServiceSession{
		MerchantID:      1,
		UserID:          1,
		CardID:          1,
		InitialUsageID:  usage.ID,
		SourceType:      "appointment",
		SourceID:        &appointmentID,
		Status:          "serving",
		DurationMinutes: 30,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	usages := []models.Usage{usage}
	enrichUsagesWithServiceSession(&usages)

	if usages[0].AppointmentID == nil || *usages[0].AppointmentID != appointmentID {
		t.Fatalf("want appointment_id=%d, got %+v", appointmentID, usages[0].AppointmentID)
	}
}

func TestApplyUsageCardSnapshotFromCardPersistsPostVerifyState(t *testing.T) {
	card := models.Card{
		CardNo:      "0012",
		CardType:    "活力成长单车卡",
		TotalTimes:  32,
		UsedTimes:   23,
		RemainTimes: 9,
	}
	usage := models.Usage{}

	applyUsageCardSnapshotFromCard(&usage, card)

	if usage.CardNoSnapshot != "0012" || usage.CardTypeSnapshot != "活力成长单车卡" {
		t.Fatalf("unexpected card text snapshots: no=%q type=%q", usage.CardNoSnapshot, usage.CardTypeSnapshot)
	}
	if usage.CardTotalTimesSnapshot == nil || *usage.CardTotalTimesSnapshot != 32 {
		t.Fatalf("want total snapshot 32, got %+v", usage.CardTotalTimesSnapshot)
	}
	if usage.CardUsedTimesSnapshot == nil || *usage.CardUsedTimesSnapshot != 23 {
		t.Fatalf("want used snapshot 23, got %+v", usage.CardUsedTimesSnapshot)
	}
	if usage.CardRemainTimesSnapshot == nil || *usage.CardRemainTimesSnapshot != 9 {
		t.Fatalf("want remain snapshot 9, got %+v", usage.CardRemainTimesSnapshot)
	}
}

func TestEnrichUsagesWithCardSnapshotsBackfillsLegacyHistory(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionFactoryTestDB(t)

	now := time.Date(2026, 4, 16, 15, 40, 0, 0, time.Local)
	card := models.Card{MerchantID: 1, UserID: 1, CardNo: "0012", CardType: "活力成长单车卡", TotalTimes: 32, UsedTimes: 23, RemainTimes: 9}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	first := models.Usage{MerchantID: 1, CardID: card.ID, UsedTimes: 1, UsedAt: &now, Status: "success"}
	secondAt := now.Add(42 * time.Minute)
	second := models.Usage{MerchantID: 1, CardID: card.ID, UsedTimes: 1, UsedAt: &secondAt, Status: "success"}
	if err := config.DB.Create(&first).Error; err != nil {
		t.Fatalf("create first usage failed: %v", err)
	}
	if err := config.DB.Create(&second).Error; err != nil {
		t.Fatalf("create second usage failed: %v", err)
	}

	usages := []models.Usage{second, first}
	enrichUsagesWithCardSnapshots(usages)

	if usages[0].CardRemainTimesSnapshot == nil || *usages[0].CardRemainTimesSnapshot != 9 {
		t.Fatalf("want latest remain snapshot 9, got %+v", usages[0].CardRemainTimesSnapshot)
	}
	if usages[0].CardUsedTimesSnapshot == nil || *usages[0].CardUsedTimesSnapshot != 23 {
		t.Fatalf("want latest used snapshot 23, got %+v", usages[0].CardUsedTimesSnapshot)
	}
	if usages[1].CardRemainTimesSnapshot == nil || *usages[1].CardRemainTimesSnapshot != 10 {
		t.Fatalf("want older remain snapshot 10, got %+v", usages[1].CardRemainTimesSnapshot)
	}
	if usages[1].CardUsedTimesSnapshot == nil || *usages[1].CardUsedTimesSnapshot != 22 {
		t.Fatalf("want older used snapshot 22, got %+v", usages[1].CardUsedTimesSnapshot)
	}
}

func TestEnrichUsagesWithCardSnapshotsPrefersStoredSnapshots(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionFactoryTestDB(t)

	now := time.Date(2026, 4, 16, 15, 40, 0, 0, time.Local)
	card := models.Card{MerchantID: 1, UserID: 1, CardNo: "0012", CardType: "活力成长单车卡", TotalTimes: 32, UsedTimes: 19, RemainTimes: 13}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	total := 32
	firstUsed := 22
	firstRemain := 10
	first := models.Usage{
		MerchantID:              1,
		CardID:                  card.ID,
		UsedTimes:               1,
		UsedAt:                  &now,
		Status:                  "success",
		CardTotalTimesSnapshot:  &total,
		CardUsedTimesSnapshot:   &firstUsed,
		CardRemainTimesSnapshot: &firstRemain,
		CardNoSnapshot:          "0012",
		CardTypeSnapshot:        "活力成长单车卡",
	}
	secondAt := now.Add(42 * time.Minute)
	secondUsed := 23
	secondRemain := 9
	second := first
	second.UsedAt = &secondAt
	second.CardUsedTimesSnapshot = &secondUsed
	second.CardRemainTimesSnapshot = &secondRemain
	if err := config.DB.Create(&first).Error; err != nil {
		t.Fatalf("create first usage failed: %v", err)
	}
	if err := config.DB.Create(&second).Error; err != nil {
		t.Fatalf("create second usage failed: %v", err)
	}

	usages := []models.Usage{second, first}
	enrichUsagesWithCardSnapshots(usages)

	if usages[0].CardUsedTimesSnapshot == nil || *usages[0].CardUsedTimesSnapshot != 23 {
		t.Fatalf("want latest stored used snapshot 23, got %+v", usages[0].CardUsedTimesSnapshot)
	}
	if usages[0].CardRemainTimesSnapshot == nil || *usages[0].CardRemainTimesSnapshot != 9 {
		t.Fatalf("want latest stored remain snapshot 9, got %+v", usages[0].CardRemainTimesSnapshot)
	}
	if usages[1].CardUsedTimesSnapshot == nil || *usages[1].CardUsedTimesSnapshot != 22 {
		t.Fatalf("want older stored used snapshot 22, got %+v", usages[1].CardUsedTimesSnapshot)
	}
	if usages[1].CardRemainTimesSnapshot == nil || *usages[1].CardRemainTimesSnapshot != 10 {
		t.Fatalf("want older stored remain snapshot 10, got %+v", usages[1].CardRemainTimesSnapshot)
	}
}

func TestEnrichServiceSessionsWithCardSnapshotsUsesInitialUsage(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionFactoryTestDB(t)

	now := time.Date(2026, 4, 16, 15, 40, 0, 0, time.Local)
	card := models.Card{MerchantID: 1, UserID: 1, CardNo: "0012", CardType: "活力成长单车卡", TotalTimes: 32, UsedTimes: 23, RemainTimes: 9}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	first := models.Usage{MerchantID: 1, CardID: card.ID, UsedTimes: 1, UsedAt: &now, Status: "success"}
	secondAt := now.Add(42 * time.Minute)
	second := models.Usage{MerchantID: 1, CardID: card.ID, UsedTimes: 1, UsedAt: &secondAt, Status: "success"}
	if err := config.DB.Create(&first).Error; err != nil {
		t.Fatalf("create first usage failed: %v", err)
	}
	if err := config.DB.Create(&second).Error; err != nil {
		t.Fatalf("create second usage failed: %v", err)
	}

	sessions := []models.ServiceSession{
		{MerchantID: 1, UserID: 1, CardID: card.ID, InitialUsageID: second.ID, Card: &models.Card{ID: card.ID}, InitialUsage: &second},
		{MerchantID: 1, UserID: 1, CardID: card.ID, InitialUsageID: first.ID, Card: &models.Card{ID: card.ID}, InitialUsage: &first},
	}
	enrichServiceSessionsWithCardSnapshots(sessions)

	if sessions[0].Card.RemainTimes != 9 || sessions[0].Card.UsedTimes != 23 {
		t.Fatalf("want latest service card snapshot 23/9, got used=%d remain=%d", sessions[0].Card.UsedTimes, sessions[0].Card.RemainTimes)
	}
	if sessions[1].Card.RemainTimes != 10 || sessions[1].Card.UsedTimes != 22 {
		t.Fatalf("want older service card snapshot 22/10, got used=%d remain=%d", sessions[1].Card.UsedTimes, sessions[1].Card.RemainTimes)
	}
}
