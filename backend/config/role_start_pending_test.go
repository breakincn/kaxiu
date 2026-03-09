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
