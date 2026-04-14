package handlers

import (
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
