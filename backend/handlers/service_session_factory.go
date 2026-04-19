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
	startPendingTimeoutSeconds := 0

	durationMinutes, delaySeconds := resolveProjectServiceConfig(tx, merchant.ID, verifyCode.ProjectID, 50, 60)
	var defaultServiceTechnicianID uint
	var defaultServiceTechnicianIDs []uint
	if merchant.SupportCustomerServiceMode && source.Appointment == nil {
		var err error
		defaultServiceTechnicianIDs, err = resolveProjectDefaultServiceTechnicianIDs(tx, merchant.ID, verifyCode.ProjectID)
		if err != nil {
			return models.ServiceSession{}, "", false, err
		}
		if len(defaultServiceTechnicianIDs) > 0 {
			defaultServiceTechnicianID = defaultServiceTechnicianIDs[0]
		}
	}
	if isQueueMode {
		durationMinutes, delaySeconds = resolveProjectServiceConfig(tx, merchant.ID, verifyCode.ProjectID, 15, merchant.StartDelaySeconds)
		status = "staff_selecting"
		nextStep = ""
		shouldEnqueueOnsite = true
		autoFinishDelaySeconds = 300
	} else if merchant.SupportRoom {
		status = "room_selecting"
		dl := now.Add(time.Duration(config.GetProjectRoomSelectTimeoutSeconds(tx, merchant.ID, verifyCode.ProjectID)) * time.Second)
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
		delayMinutes, readyAt, err := estimateAppointmentArrivalDelay(tx, merchant, techID, now)
		if err != nil {
			return models.ServiceSession{}, "", false, err
		}
		if merchant.SupportRoom {
			status = models.WithCSPrefix("room_selecting")
			nextStep = "room_select"
		} else {
			status = models.WithCSPrefix("start_pending")
			nextStep = ""
			startPendingTimeoutSeconds = config.ResolveServiceSessionStartPendingTimeoutSeconds(tx, merchant.ID, techID, verifyCode.ProjectID)
		}
		session.TechnicianID = &techID
		session.LastTechnicianID = &techID
		predictedAppointmentDelayMinutes = delayMinutes
		predictedReadyAt = readyAt
	} else if merchant.SupportCustomerServiceMode && defaultServiceTechnicianID > 0 {
		techID := defaultServiceTechnicianID
		if merchant.SupportRoom {
			status = models.WithCSPrefix("room_selecting")
			nextStep = "room_select"
		} else {
			status = models.WithCSPrefix("start_pending")
			nextStep = ""
			startPendingTimeoutSeconds = config.ResolveServiceSessionStartPendingTimeoutSeconds(tx, merchant.ID, techID, verifyCode.ProjectID)
		}
		session.TechnicianID = &techID
		session.LastTechnicianID = &techID
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
		ServiceTechnicianIDs:             models.MerchantProjectDefaultServiceTechnicianIDs(defaultServiceTechnicianIDs),
		Status:                           status,
		RoomSelectDeadlineAt:             roomSelectDeadlineAt,
		StartConfirmedAt:                 startConfirmedAt,
		StartDelaySeconds:                delaySeconds,
		ScheduledStartAt:                 scheduledStartAt,
		DurationMinutes:                  durationMinutes,
		AutoFinishDelaySeconds:           autoFinishDelaySeconds,
		AutoIdleAfterSeconds:             180,
		StartPendingTimeoutSeconds:       startPendingTimeoutSeconds,
		PredictedReadyAt:                 predictedReadyAt,
		PredictedAppointmentDelayMinutes: predictedAppointmentDelayMinutes,
	}
	if err := tx.Create(&session).Error; err != nil {
		return models.ServiceSession{}, "", false, err
	}
	return session, nextStep, shouldEnqueueOnsite, nil
}

func resolveProjectDefaultServiceTechnicianID(tx *gorm.DB, merchantID uint, projectID *uint) (uint, error) {
	ids, err := resolveProjectDefaultServiceTechnicianIDs(tx, merchantID, projectID)
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	return ids[0], nil
}

func resolveProjectDefaultServiceTechnicianIDs(tx *gorm.DB, merchantID uint, projectID *uint) ([]uint, error) {
	if tx == nil || merchantID == 0 || projectID == nil || *projectID == 0 {
		return nil, nil
	}
	project, err := config.ResolveMerchantProject(tx, merchantID, projectID)
	if err != nil || project == nil {
		return nil, err
	}
	if project.ServiceCapacity <= 1 || len(project.DefaultServiceTechnicianIDs) == 0 {
		return nil, nil
	}

	ids := make([]uint, 0, len(project.DefaultServiceTechnicianIDs))
	seen := make(map[uint]struct{}, len(project.DefaultServiceTechnicianIDs))
	for _, id := range project.DefaultServiceTechnicianIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	var techs []models.Technician
	if err := tx.
		Joins("JOIN service_roles sr ON sr.id = technicians.service_role_id").
		Where("technicians.merchant_id = ? AND technicians.id IN ? AND technicians.is_active = ? AND sr.role_type = ?", merchantID, ids, true, "professional").
		Find(&techs).Error; err != nil {
		return nil, err
	}
	valid := make(map[uint]struct{}, len(techs))
	for _, tech := range techs {
		valid[tech.ID] = struct{}{}
	}
	resolved := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := valid[id]; ok {
			resolved = append(resolved, id)
		}
	}
	return resolved, nil
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
