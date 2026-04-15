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
