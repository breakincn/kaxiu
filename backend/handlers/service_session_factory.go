package handlers

import (
	"fmt"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"log"
	"os"
	"time"

	"gorm.io/gorm"
)

const (
	serviceSessionSourceWalkIn      = "walk_in"
	serviceSessionSourceAppointment = "appointment"
)

type serviceSessionSource struct {
	SourceType string
	SourceID   *uint
}

func detectServiceSessionSource(tx *gorm.DB, merchantID uint, card models.Card, now time.Time) (serviceSessionSource, error) {
	source := serviceSessionSource{SourceType: serviceSessionSourceWalkIn}
	if tx == nil || merchantID == 0 || card.ID == 0 {
		return source, nil
	}

	rows, err := tx.
		Table("appointments").
		Select("id, appointment_time").
		Where("card_id = ? AND merchant_id = ? AND status = 'confirmed' AND appointment_time IS NOT NULL", card.ID, merchantID).
		Order("appointment_time asc").
		Limit(1).
		Rows()
	if err != nil {
		return source, err
	}
	defer rows.Close()
	if !rows.Next() {
		return source, nil
	}

	var apptID uint
	var rawAppointmentTime any
	if err := rows.Scan(&apptID, &rawAppointmentTime); err != nil {
		return source, err
	}
	at, ok := parseDBTimeValue(rawAppointmentTime)
	if !ok {
		return source, fmt.Errorf("unsupported appointment_time value: %T", rawAppointmentTime)
	}
	if at.Format("2006-01-02") != now.Format("2006-01-02") {
		return source, nil
	}
	diff := now.Sub(at)
	if diff < 0 {
		diff = -diff
	}
	if diff > 30*time.Minute {
		return source, nil
	}

	source.SourceType = serviceSessionSourceAppointment
	source.SourceID = &apptID
	return source, nil
}

func parseDBTimeValue(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x, true
	case *time.Time:
		if x != nil {
			return *x, true
		}
	case string:
		return parseDBTimeString(x)
	case []byte:
		return parseDBTimeString(string(x))
	}
	return time.Time{}, false
}

func parseDBTimeString(v string) (time.Time, bool) {
	layouts := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func createServiceSessionForUsage(tx *gorm.DB, merchant models.Merchant, card models.Card, verifyCode models.VerifyCode, usage models.Usage, now time.Time) (models.ServiceSession, string, bool, error) {
	session := models.ServiceSession{}
	if tx == nil {
		return session, "", false, nil
	}

	isQueueMode := !merchant.SupportCustomerServiceMode && merchant.SupportQueue && (merchant.QueueMode == "auto" || merchant.QueueMode == "manual")
	source, err := detectServiceSessionSource(tx, merchant.ID, card, now)
	if err != nil {
		return session, "", false, err
	}

	nextStep := "staff_select"
	shouldEnqueueOnsite := true
	status := "staff_selecting"
	var roomSelectDeadlineAt *time.Time
	var startConfirmedAt *time.Time
	var scheduledStartAt *time.Time
	autoFinishDelaySeconds := 60

	durationMinutes, delaySeconds := resolveProjectServiceConfig(tx, merchant.ID, verifyCode.ProjectID, 50, 60)
	if isQueueMode {
		durationMinutes, delaySeconds = resolveProjectServiceConfig(tx, merchant.ID, verifyCode.ProjectID, 15, merchant.StartDelaySeconds)
		status = "staff_selecting"
		nextStep = ""
		shouldEnqueueOnsite = true
		autoFinishDelaySeconds = 300
	} else if merchant.SupportRoom {
		status = "room_selecting"
		dl := now.Add(90 * time.Second)
		roomSelectDeadlineAt = &dl
		nextStep = "room_select"
	} else {
		nextStep = "staff_select"
	}

	sessionMode := models.ResolveSessionMode(&merchant)
	if isQueueMode {
		status = models.WithModePrefix(sessionMode, status)
	} else if merchant.SupportCustomerServiceMode {
		status = models.WithCSPrefix(status)
	}

	session = models.ServiceSession{
		MerchantID:             merchant.ID,
		UserID:                 card.UserID,
		CardID:                 card.ID,
		ProjectID:              verifyCode.ProjectID,
		InitialUsageID:         usage.ID,
		VerifyCode:             verifyCode.Code,
		SessionMode:            sessionMode,
		SourceType:             source.SourceType,
		SourceID:               source.SourceID,
		Status:                 status,
		RoomSelectDeadlineAt:   roomSelectDeadlineAt,
		StartConfirmedAt:       startConfirmedAt,
		StartDelaySeconds:      delaySeconds,
		ScheduledStartAt:       scheduledStartAt,
		DurationMinutes:        durationMinutes,
		AutoFinishDelaySeconds: autoFinishDelaySeconds,
		AutoIdleAfterSeconds:   180,
	}
	if err := tx.Create(&session).Error; err != nil {
		return models.ServiceSession{}, "", false, err
	}
	return session, nextStep, shouldEnqueueOnsite, nil
}

func enqueueVerifyUsageIfNeeded(merchant models.Merchant, card models.Card, usageID uint, shouldEnqueueOnsite bool) {
	if !merchant.SupportQueue || !shouldEnqueueOnsite || usageID == 0 {
		return
	}

	now := time.Now()
	source, err := detectServiceSessionSource(config.DB, merchant.ID, card, now)
	if err != nil {
		log.Printf("WARN: detect service session source for enqueue failed: merchant=%d card=%d usage=%d err=%v", merchant.ID, card.ID, usageID, err)
	}
	if err == nil && source.SourceType == serviceSessionSourceAppointment {
		return
	}

	date := now.Format("2006-01-02")
	autoCallFirst := false
	if merchant.QueueMode == "auto" && !merchant.SupportMultiCustomerService {
		autoCallFirst = true
	}
	tk, created := queue.Default.Enqueue(merchant.ID, date, queue.QueueTypeOnsite, usageID, merchant.QueueStartNo, autoCallFirst, now)
	if os.Getenv("KABAO_QUEUE_DEBUG") == "1" {
		log.Printf("[queue-debug] verify enqueue onsite: merchant=%d date=%s usage_id=%d created=%v queue_no=%d called_at=%v\n", merchant.ID, date, usageID, created, tk.No, tk.CalledAt)
	}
	if merchant.QueueMode == "auto" && merchant.SupportMultiCustomerService {
		tryAutoCallNextForIdleTechnicians(config.DB, merchant.ID, now)
	}
}
