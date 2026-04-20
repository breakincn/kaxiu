package handlers

import (
	"testing"
	"time"

	"kabao/config"
	"kabao/models"
)

func TestNormalizeAppointmentForReadAutoClosesCrossDayUnfinished(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	loc := appointmentLocation()
	now := time.Date(2026, 3, 25, 9, 0, 0, 0, loc)
	appointmentTime := time.Date(2026, 3, 19, 10, 45, 0, 0, loc)
	arrivedAt := time.Date(2026, 3, 19, 10, 37, 29, 0, loc)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "R001", CardType: "次卡", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	projectID := uint(1)
	usage := models.Usage{CardID: card.ID, MerchantID: merchant.ID, ProjectID: &projectID, UsedTimes: 1, Status: "in_progress"}
	if err := config.DB.Create(&usage).Error; err != nil {
		t.Fatalf("create usage failed: %v", err)
	}
	appt := models.Appointment{
		MerchantID:           merchant.ID,
		UserID:               user.ID,
		CardID:               card.ID,
		TechnicianID:         &tech.ID,
		ProjectID:            &projectID,
		Status:               "arrived",
		AppointmentTime:      &appointmentTime,
		ArrivedAt:            &arrivedAt,
		UsageID:              &usage.ID,
		PredictedWaitMinutes: 46,
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID:                 merchant.ID,
		UserID:                     user.ID,
		CardID:                     card.ID,
		ProjectID:                  &projectID,
		InitialUsageID:             usage.ID,
		SessionMode:                models.SessionModeCustomerService,
		SourceType:                 "appointment",
		SourceID:                   &appt.ID,
		LastTechnicianID:           &tech.ID,
		ServiceTechnicianIDs:       models.MerchantProjectDefaultServiceTechnicianIDs{tech.ID},
		Status:                     "cs_start_pending",
		CreatedAt:                  &arrivedAt,
		UpdatedAt:                  &arrivedAt,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("service_session_id", session.ID).Error; err != nil {
		t.Fatalf("bind session failed: %v", err)
	}
	appt.ServiceSessionID = &session.ID

	if err := normalizeAppointmentForRead(&appt, now); err != nil {
		t.Fatalf("normalizeAppointmentForRead failed: %v", err)
	}

	if appt.Status != "failed" || appt.FailedReason != "service_unclosed_cross_day" {
		t.Fatalf("want failed cross-day appointment, got status=%s reason=%s", appt.Status, appt.FailedReason)
	}
	if appt.ActualArrivedAt == nil {
		t.Fatalf("want actual_arrived_at backfilled")
	}
	if appt.AppointmentSettlementID == nil || *appt.AppointmentSettlementID == 0 {
		t.Fatalf("want settlement created, got %+v", appt.AppointmentSettlementID)
	}

	var gotUsage struct {
		Status string
	}
	if err := config.DB.Table("usages").Select("status").Where("id = ?", usage.ID).Scan(&gotUsage).Error; err != nil {
		t.Fatalf("reload usage failed: %v", err)
	}
	if gotUsage.Status != "failed" {
		t.Fatalf("want usage failed, got %s", gotUsage.Status)
	}

	var gotSession struct {
		Status string
	}
	if err := config.DB.Table("service_sessions").Select("status").Where("id = ?", session.ID).Scan(&gotSession).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if gotSession.Status != "cs_canceled" {
		t.Fatalf("want session canceled, got %s", gotSession.Status)
	}

	var gotCard struct {
		RemainTimes int64
		UsedTimes   int64
	}
	if err := config.DB.Table("cards").Select("remain_times", "used_times").Where("id = ?", card.ID).Scan(&gotCard).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotCard.RemainTimes != 10 || gotCard.UsedTimes != 0 {
		t.Fatalf("want card counters refunded, got remain=%d used=%d", gotCard.RemainTimes, gotCard.UsedTimes)
	}
}
