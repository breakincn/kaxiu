package handlers

import (
	"kabao/models"
	"time"
)

func revokeDeadlineAt(usedAt *time.Time) *time.Time {
	if usedAt == nil {
		return nil
	}
	dl := usedAt.Add(12 * time.Hour)
	return &dl
}

func isServiceSessionInRevokeablePhase(merchant *models.Merchant, s *models.ServiceSession) bool {
	if merchant == nil || s == nil || !merchant.SupportCustomerServiceMode {
		return false
	}
	if s.StartConfirmedAt != nil || s.StartedAt != nil {
		return false
	}

	switch models.NormalizeSessionStatus(s.Status) {
	case "start_pending", "delay_pending", "serving", "auto_finishing", "finished":
		return false
	default:
		return true
	}
}

func canRevokeUsageWithSession(u *models.Usage, s *models.ServiceSession, now time.Time) bool {
	if u == nil || s == nil {
		return false
	}
	if u.Status != "in_progress" || u.UsedAt == nil {
		return false
	}
	if !u.Merchant.SupportCustomerServiceMode {
		return false
	}
	dl := revokeDeadlineAt(u.UsedAt)
	if dl == nil || !now.Before(*dl) {
		return false
	}
	if s.StartTimeoutCount < 2 {
		return false
	}
	return isServiceSessionInRevokeablePhase(&u.Merchant, s)
}
