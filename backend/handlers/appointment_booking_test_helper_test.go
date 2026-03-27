package handlers

import (
	"testing"
	"time"

	"kabao/config"
	"kabao/models"
)

func withAppointmentCurrentTime(t *testing.T, now time.Time) {
	t.Helper()
	old := appointmentCurrentTime
	appointmentCurrentTime = func() time.Time {
		return now.In(appointmentLocation())
	}
	t.Cleanup(func() {
		appointmentCurrentTime = old
	})
}

func seedNextDayPublishedScheduleForMerchant(t *testing.T, merchant models.Merchant, technicianIDs ...uint) {
	t.Helper()

	loc := appointmentLocation()
	now := appointmentCurrentTime().In(loc)
	targetDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	intervals, ok := getMerchantBusinessIntervalsForDate(merchant, targetDate)
	if !ok || len(intervals) == 0 {
		t.Fatalf("merchant %d missing business intervals for %s", merchant.ID, targetDate.Format("2006-01-02"))
	}

	publishedAt := now
	publishDate := targetDate
	rows := make([]models.TechnicianSchedulePublishing, 0, len(intervals)*max(1, len(technicianIDs)))
	if len(technicianIDs) == 0 {
		for _, interval := range intervals {
			startAt := interval.Start
			endAt := interval.End
			rows = append(rows, models.TechnicianSchedulePublishing{
				MerchantID:  merchant.ID,
				PublishDate: &publishDate,
				StartAt:     &startAt,
				EndAt:       &endAt,
				Status:      "published",
				PublishedAt: &publishedAt,
			})
		}
	} else {
		for _, technicianID := range technicianIDs {
			techID := technicianID
			for _, interval := range intervals {
				startAt := interval.Start
				endAt := interval.End
				rows = append(rows, models.TechnicianSchedulePublishing{
					MerchantID:   merchant.ID,
					TechnicianID: &techID,
					PublishDate:  &publishDate,
					StartAt:      &startAt,
					EndAt:        &endAt,
					Status:       "published",
					PublishedAt:  &publishedAt,
				})
			}
		}
	}

	if err := config.DB.Create(&rows).Error; err != nil {
		t.Fatalf("seed next-day published schedule failed: %v", err)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
