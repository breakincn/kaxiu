package handlers

import (
	"kabao/config"
	"kabao/models"
	"testing"
	"time"
)

func TestComputeStartPendingRemainingSecondsForSession(t *testing.T) {
	now := time.Date(2026, 3, 6, 10, 0, 0, 0, time.Local)

	t.Run("non_start_pending_returns_zero", func(t *testing.T) {
		s := &models.ServiceSession{Status: "serving"}
		if got := computeStartPendingRemainingSecondsForSession(s, now); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("confirmed_returns_zero", func(t *testing.T) {
		confirmed := now.Add(-5 * time.Second)
		s := &models.ServiceSession{
			Status:           "start_pending",
			StartConfirmedAt: &confirmed,
			UpdatedAt:        &now,
		}
		if got := computeStartPendingRemainingSecondsForSession(s, now); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("updated_at_with_custom_timeout", func(t *testing.T) {
		updated := now.Add(-30 * time.Second)
		s := &models.ServiceSession{
			Status:                     "start_pending",
			UpdatedAt:                  &updated,
			StartPendingTimeoutSeconds: 120,
		}
		if got := computeStartPendingRemainingSecondsForSession(s, now); got != 90 {
			t.Fatalf("expected 90, got %d", got)
		}
	})

	t.Run("fallback_to_created_at", func(t *testing.T) {
		created := now.Add(-10 * time.Second)
		s := &models.ServiceSession{
			Status:                     "start_pending",
			CreatedAt:                  &created,
			StartPendingTimeoutSeconds: 60,
		}
		if got := computeStartPendingRemainingSecondsForSession(s, now); got != 50 {
			t.Fatalf("expected 50, got %d", got)
		}
	})

	t.Run("scheduled_start_at_takes_priority", func(t *testing.T) {
		updated := now.Add(-30 * time.Second)
		scheduled := now.Add(37*time.Minute + 15*time.Second)
		s := &models.ServiceSession{
			Status:                     "start_pending",
			UpdatedAt:                  &updated,
			ScheduledStartAt:           &scheduled,
			StartPendingTimeoutSeconds: 120,
		}
		if got := computeStartPendingRemainingSecondsForSession(s, now); got != 2235 {
			t.Fatalf("expected 2235, got %d", got)
		}
	})

	t.Run("elapsed_scheduled_start_at_returns_zero", func(t *testing.T) {
		scheduled := now.Add(-time.Second)
		s := &models.ServiceSession{
			Status:           "start_pending",
			ScheduledStartAt: &scheduled,
		}
		if got := computeStartPendingRemainingSecondsForSession(s, now); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("timeout_elapsed_returns_zero", func(t *testing.T) {
		updated := now.Add(-121 * time.Second)
		s := &models.ServiceSession{
			Status:                     "start_pending",
			UpdatedAt:                  &updated,
			StartPendingTimeoutSeconds: 120,
		}
		if got := computeStartPendingRemainingSecondsForSession(s, now); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("fallback_to_default_timeout", func(t *testing.T) {
		updated := now.Add(-10 * time.Second)
		s := &models.ServiceSession{
			Status:    "start_pending",
			UpdatedAt: &updated,
		}
		expect := int(config.StartPendingTimeout().Seconds()) - 10
		if expect < 0 {
			expect = 0
		}
		if got := computeStartPendingRemainingSecondsForSession(s, now); got != expect {
			t.Fatalf("expected %d, got %d", expect, got)
		}
	})
}
