package config

import (
	"kabao/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

func ProjectServiceTimeLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}
	return loc
}

func MerchantProjectServiceTimeSlotMatchesDate(slot models.MerchantProjectServiceTimeSlot, date time.Time) bool {
	if strings.TrimSpace(slot.RecurrenceType) == "monthly" {
		return slot.MonthDay == date.Day()
	}

	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return slot.Weekday == weekday
}

func NextMerchantProjectServiceStart(project models.MerchantProject, now time.Time) (*time.Time, bool) {
	if len(project.ServiceTimeSlots) == 0 {
		return nil, false
	}

	loc := ProjectServiceTimeLocation()
	localNow := now.In(loc)
	var nearest *time.Time
	for _, slot := range project.ServiceTimeSlots {
		startClock, err := time.ParseInLocation("15:04", strings.TrimSpace(slot.StartTime), loc)
		if err != nil {
			continue
		}
		for offset := 0; offset <= 370; offset++ {
			candidateDate := localNow.AddDate(0, 0, offset)
			if !MerchantProjectServiceTimeSlotMatchesDate(slot, candidateDate) {
				continue
			}
			startAt := time.Date(candidateDate.Year(), candidateDate.Month(), candidateDate.Day(), startClock.Hour(), startClock.Minute(), 0, 0, loc)
			if startAt.Before(localNow) {
				continue
			}
			if nearest == nil || startAt.Before(*nearest) {
				v := startAt
				nearest = &v
			}
			break
		}
	}
	if nearest == nil {
		return nil, false
	}
	return nearest, true
}

func ResolveProjectNextServiceStart(tx *gorm.DB, merchantID uint, projectID *uint, now time.Time) (*time.Time, bool, error) {
	if tx == nil || merchantID == 0 || projectID == nil || *projectID == 0 {
		return nil, false, nil
	}
	project, err := ResolveMerchantProject(tx, merchantID, projectID)
	if err != nil || project == nil {
		return nil, false, err
	}
	startAt, ok := NextMerchantProjectServiceStart(*project, now)
	return startAt, ok, nil
}
