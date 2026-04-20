package handlers

import (
	"kabao/config"
	"kabao/models"
	"time"
)

func computeStartPendingRemainingSecondsForSession(s *models.ServiceSession, now time.Time) int {
	if s == nil {
		return 0
	}
	if models.NormalizeSessionStatus(s.Status) != "start_pending" {
		return 0
	}
	if s.StartConfirmedAt != nil {
		return 0
	}
	if s.ScheduledStartAt != nil {
		remain := int(s.ScheduledStartAt.Sub(now).Seconds())
		if remain <= 0 {
			return 0
		}
		return remain
	}
	baseAt := s.UpdatedAt
	if baseAt == nil {
		baseAt = s.CreatedAt
	}
	if baseAt == nil {
		return 0
	}

	timeout := config.StartPendingTimeout()
	if s.StartPendingTimeoutSeconds > 0 {
		timeout = time.Duration(s.StartPendingTimeoutSeconds) * time.Second
	}
	if timeout <= 0 {
		return 0
	}

	remain := int(baseAt.Add(timeout).Sub(now).Seconds())
	if remain <= 0 {
		return 0
	}
	return remain
}
