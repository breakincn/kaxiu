package handlers

import (
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
	SourceType  string
	SourceID    *uint
	Appointment *models.Appointment
}

func createServiceSessionForUsage(tx *gorm.DB, merchant models.Merchant, card models.Card, verifyCode models.VerifyCode, usage models.Usage, now time.Time) (models.ServiceSession, string, bool, error) {
	session := models.ServiceSession{}
	if tx == nil {
		return session, "", false, nil
	}

	isQueueMode := !merchant.SupportCustomerServiceMode && merchant.SupportQueue && (merchant.QueueMode == "auto" || merchant.QueueMode == "manual")
	source, err := detectServiceSessionSource(tx, merchant, card, now)
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
	predictedAppointmentDelayMinutes := 0
	var predictedReadyAt *time.Time

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

	if merchant.SupportCustomerServiceMode && source.Appointment != nil && source.Appointment.TechnicianID != nil && *source.Appointment.TechnicianID > 0 {
		techID := *source.Appointment.TechnicianID
		occupiedMinutes := projectBookingOccupiedMinutes(durationMinutes, 3)
		if verifyCode.ProjectID != nil {
			_, gapMinutes := resolveProjectBookingConfig(tx, merchant.ID, verifyCode.ProjectID, durationMinutes, 3)
			occupiedMinutes = projectBookingOccupiedMinutes(durationMinutes, gapMinutes)
		}
		availability, err := evaluateWalkInTechnicianAvailability(tx, merchant, techID, now, occupiedMinutes)
		if err != nil {
			return models.ServiceSession{}, "", false, err
		}
		// 预约客户到店后，若原预约客服仍被前序服务占用，则直接进入预约优先等待状态，
		// 避免掉入普通现场流转并被后续现场单再次插队。
		if availability.State != appointmentAvailabilitySafe && availability.PredictedWaitMinutes > 0 {
			status = models.WithCSPrefix("appointment_waiting")
			nextStep = ""
		} else if merchant.SupportRoom {
			status = models.WithCSPrefix("room_selecting")
			nextStep = "room_select"
		} else {
			status = models.WithCSPrefix("start_pending")
			nextStep = ""
		}
		session.TechnicianID = &techID
		session.LastTechnicianID = &techID
		predictedAppointmentDelayMinutes = availability.PredictedWaitMinutes
	}

	session = models.ServiceSession{
		MerchantID:                       merchant.ID,
		UserID:                           card.UserID,
		CardID:                           card.ID,
		ProjectID:                        verifyCode.ProjectID,
		InitialUsageID:                   usage.ID,
		VerifyCode:                       verifyCode.Code,
		SessionMode:                      sessionMode,
		SourceType:                       source.SourceType,
		SourceID:                         source.SourceID,
		TechnicianID:                     session.TechnicianID,
		LastTechnicianID:                 session.LastTechnicianID,
		Status:                           status,
		RoomSelectDeadlineAt:             roomSelectDeadlineAt,
		StartConfirmedAt:                 startConfirmedAt,
		StartDelaySeconds:                delaySeconds,
		ScheduledStartAt:                 scheduledStartAt,
		DurationMinutes:                  durationMinutes,
		AutoFinishDelaySeconds:           autoFinishDelaySeconds,
		AutoIdleAfterSeconds:             180,
		PredictedReadyAt:                 predictedReadyAt,
		PredictedAppointmentDelayMinutes: predictedAppointmentDelayMinutes,
	}
	if source.Appointment != nil {
		if status == models.WithCSPrefix("appointment_waiting") {
			occupiedMinutes := projectBookingOccupiedMinutes(durationMinutes, 3)
			if verifyCode.ProjectID != nil {
				_, gapMinutes := resolveProjectBookingConfig(tx, merchant.ID, verifyCode.ProjectID, durationMinutes, 3)
				occupiedMinutes = projectBookingOccupiedMinutes(durationMinutes, gapMinutes)
			}
			readyAt := appointmentPredictedFinishAt(now, occupiedMinutes, &merchant)
			session.PredictedReadyAt = &readyAt
			session.PredictedAppointmentDelayMinutes = predictedAppointmentDelayMinutes
		}
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
	source, err := detectServiceSessionSource(config.DB, merchant, card, now)
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
