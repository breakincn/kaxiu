package handlers

import (
	"testing"
	"time"

	"kabao/config"
	"kabao/models"
)

func TestRepairCrossDayUnfinishedAppointmentsDryRunAndExecute(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	loc := appointmentLocation()
	now := time.Date(2026, 3, 25, 9, 0, 0, 0, loc)
	appointmentTime := time.Date(2026, 3, 19, 10, 45, 0, 0, loc)
	arrivedAt := time.Date(2026, 3, 19, 10, 37, 29, 0, loc)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "R002", CardType: "次卡", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
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
		MerchantID:       merchant.ID,
		UserID:           user.ID,
		CardID:           card.ID,
		ProjectID:        &projectID,
		InitialUsageID:   usage.ID,
		SessionMode:      models.SessionModeCustomerService,
		SourceType:       "appointment",
		SourceID:         &appt.ID,
		TechnicianID:     &tech.ID,
		LastTechnicianID: &tech.ID,
		Status:           "cs_appointment_waiting",
		CreatedAt:        &arrivedAt,
		UpdatedAt:        &arrivedAt,
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("service_session_id", session.ID).Error; err != nil {
		t.Fatalf("bind session failed: %v", err)
	}

	dryResults, err := RepairCrossDayUnfinishedAppointments(config.DB, now, 10, true, "dry-run")
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if len(dryResults) != 1 || dryResults[0].Status != "dry_run" {
		t.Fatalf("want one dry-run result, got %+v", dryResults)
	}

	runResults, err := RepairCrossDayUnfinishedAppointments(config.DB, now, 10, false, "execute")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if len(runResults) != 1 || runResults[0].Status != "closed" {
		t.Fatalf("want one closed result, got %+v", runResults)
	}

	var got struct {
		Status string
	}
	if err := config.DB.Table("appointments").Select("status").Where("id = ?", appt.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload appointment failed: %v", err)
	}
	if got.Status != "failed" {
		t.Fatalf("want appointment failed, got %s", got.Status)
	}
}
