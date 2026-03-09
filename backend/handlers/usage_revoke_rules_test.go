package handlers

import (
	"kabao/models"
	"testing"
	"time"
)

func TestCanRevokeUsageWithSession(t *testing.T) {
	now := time.Date(2026, 3, 9, 10, 0, 0, 0, time.Local)
	usedAt := now.Add(-30 * time.Minute)

	newUsage := func() models.Usage {
		return models.Usage{
			Status: "in_progress",
			UsedAt: &usedAt,
			Merchant: models.Merchant{
				SupportCustomerServiceMode: true,
			},
		}
	}

	t.Run("staff_selecting_after_two_timeouts_can_revoke", func(t *testing.T) {
		u := newUsage()
		s := models.ServiceSession{
			Status:            "cs_staff_selecting",
			StartTimeoutCount: 2,
		}
		if !canRevokeUsageWithSession(&u, &s, now) {
			t.Fatalf("expected revokeable in staff selecting")
		}
	})

	t.Run("room_locked_after_two_timeouts_can_revoke", func(t *testing.T) {
		u := newUsage()
		s := models.ServiceSession{
			Status:            "cs_room_locked",
			StartTimeoutCount: 2,
		}
		if !canRevokeUsageWithSession(&u, &s, now) {
			t.Fatalf("expected revokeable in room locked")
		}
	})

	t.Run("canceled_after_two_timeouts_can_revoke", func(t *testing.T) {
		u := newUsage()
		s := models.ServiceSession{
			Status:            "cs_canceled",
			StartTimeoutCount: 2,
		}
		if !canRevokeUsageWithSession(&u, &s, now) {
			t.Fatalf("expected revokeable in canceled")
		}
	})

	t.Run("start_pending_cannot_revoke", func(t *testing.T) {
		u := newUsage()
		s := models.ServiceSession{
			Status:            "cs_start_pending",
			StartTimeoutCount: 2,
		}
		if canRevokeUsageWithSession(&u, &s, now) {
			t.Fatalf("expected not revokeable in start pending")
		}
	})

	t.Run("serving_cannot_revoke", func(t *testing.T) {
		u := newUsage()
		s := models.ServiceSession{
			Status:            "cs_serving",
			StartTimeoutCount: 2,
		}
		if canRevokeUsageWithSession(&u, &s, now) {
			t.Fatalf("expected not revokeable in serving")
		}
	})

	t.Run("start_confirmed_cannot_revoke", func(t *testing.T) {
		u := newUsage()
		startConfirmedAt := now.Add(-1 * time.Minute)
		s := models.ServiceSession{
			Status:            "cs_staff_selecting",
			StartTimeoutCount: 2,
			StartConfirmedAt:  &startConfirmedAt,
		}
		if canRevokeUsageWithSession(&u, &s, now) {
			t.Fatalf("expected not revokeable after start confirmed")
		}
	})

	t.Run("less_than_two_timeouts_cannot_revoke", func(t *testing.T) {
		u := newUsage()
		s := models.ServiceSession{
			Status:            "cs_staff_selecting",
			StartTimeoutCount: 1,
		}
		if canRevokeUsageWithSession(&u, &s, now) {
			t.Fatalf("expected not revokeable before two timeouts")
		}
	})

	t.Run("expired_usage_cannot_revoke", func(t *testing.T) {
		oldUsedAt := now.Add(-13 * time.Hour)
		u := newUsage()
		u.UsedAt = &oldUsedAt
		s := models.ServiceSession{
			Status:            "cs_staff_selecting",
			StartTimeoutCount: 2,
		}
		if canRevokeUsageWithSession(&u, &s, now) {
			t.Fatalf("expected not revokeable after deadline")
		}
	})
}
