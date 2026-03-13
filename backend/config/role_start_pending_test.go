package config

import (
	"kabao/models"
	"testing"
)

func TestEffectiveRoleStartPendingTimeoutSeconds(t *testing.T) {
	t.Run("override_wins", func(t *testing.T) {
		role := &models.ServiceRole{StartPendingTimeoutSeconds: 300}
		override := &models.MerchantRoleStartPendingConfig{StartPendingTimeoutSeconds: 180}
		if got := EffectiveRoleStartPendingTimeoutSeconds(role, override); got != 180 {
			t.Fatalf("expected 180, got %d", got)
		}
	})

	t.Run("role_default_used_when_no_override", func(t *testing.T) {
		role := &models.ServiceRole{StartPendingTimeoutSeconds: 420}
		if got := EffectiveRoleStartPendingTimeoutSeconds(role, nil); got != 420 {
			t.Fatalf("expected 420, got %d", got)
		}
	})

	t.Run("fallback_default_is_five_minutes", func(t *testing.T) {
		if got := EffectiveRoleStartPendingTimeoutSeconds(nil, nil); got != 300 {
			t.Fatalf("expected 300, got %d", got)
		}
	})
}

func TestRoleStartPendingLabelForMerchant(t *testing.T) {
	t.Run("queue_mode_uses_calling_term", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportQueue:               true,
			SupportCustomerServiceMode: false,
			QueueMode:                  "auto",
			StartTerm:                  "上钟",
		}
		if got := RoleStartPendingLabelForMerchant(merchant); got != "待叫号超时秒数" {
			t.Fatalf("expected 待叫号超时秒数, got %s", got)
		}
	})

	t.Run("customer_service_mode_uses_custom_term", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportQueue:               false,
			SupportCustomerServiceMode: true,
			StartTerm:                  "上钟",
		}
		if got := RoleStartPendingLabelForMerchant(merchant); got != "待上钟超时秒数" {
			t.Fatalf("expected 待上钟超时秒数, got %s", got)
		}
	})

	t.Run("customer_service_mode_falls_back_to_default", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportCustomerServiceMode: true,
		}
		if got := RoleStartPendingLabelForMerchant(merchant); got != "待起单超时秒数" {
			t.Fatalf("expected 待起单超时秒数, got %s", got)
		}
	})
}
