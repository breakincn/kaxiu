package handlers

import (
	"kabao/models"
	"testing"
	"time"
)

func TestDecorateAppointmentDisplayMarksCrossDayUnfinished(t *testing.T) {
	loc := appointmentLocation()
	appointmentTime := time.Date(2026, 3, 19, 10, 45, 0, 0, loc)
	now := time.Date(2026, 3, 25, 9, 0, 0, 0, loc)
	sessionID := uint(12)
	arrivedAt := time.Date(2026, 3, 19, 10, 37, 0, 0, loc)

	appt := models.Appointment{
		Status:               "arrived",
		AppointmentTime:      &appointmentTime,
		ArrivedAt:            &arrivedAt,
		ServiceSessionID:     &sessionID,
		PredictedWaitMinutes: 46,
	}

	decorateAppointmentDisplay(&appt, now)

	if appt.DisplayWaitState != appointmentDisplayWaitStateCrossDayUnclosed {
		t.Fatalf("want display_wait_state=%q, got %q", appointmentDisplayWaitStateCrossDayUnclosed, appt.DisplayWaitState)
	}
	if appt.CurrentEstimatedWaitMinutes != 0 {
		t.Fatalf("want current_estimated_wait_minutes=0, got %d", appt.CurrentEstimatedWaitMinutes)
	}
	if appt.DisplayWaitMessage == "" {
		t.Fatalf("want non-empty display_wait_message")
	}
	if appt.DisruptionReason != "service_unclosed_cross_day" {
		t.Fatalf("want disruption_reason=service_unclosed_cross_day, got %q", appt.DisruptionReason)
	}
}

func TestDecorateAppointmentDisplayKeepsActiveWaiting(t *testing.T) {
	loc := appointmentLocation()
	appointmentTime := time.Date(2026, 3, 25, 10, 45, 0, 0, loc)
	now := time.Date(2026, 3, 25, 10, 50, 0, 0, loc)
	sessionID := uint(12)

	appt := models.Appointment{
		Status:               "arrived",
		AppointmentTime:      &appointmentTime,
		ServiceSessionID:     &sessionID,
		PredictedWaitMinutes: 18,
	}

	decorateAppointmentDisplay(&appt, now)

	if appt.DisplayWaitState != appointmentDisplayWaitStateActiveWaiting {
		t.Fatalf("want display_wait_state=%q, got %q", appointmentDisplayWaitStateActiveWaiting, appt.DisplayWaitState)
	}
	if appt.CurrentEstimatedWaitMinutes != 18 {
		t.Fatalf("want current_estimated_wait_minutes=18, got %d", appt.CurrentEstimatedWaitMinutes)
	}
}

func TestCanArriveForAppointmentUsesRemainingServiceThreshold(t *testing.T) {
	loc := appointmentLocation()
	merchant := &models.Merchant{
		AppointmentReserveBufferMinutes: 10,
		AppointmentGraceWindowMinutes:   15,
	}
	appointmentTime := time.Date(2026, 3, 29, 10, 30, 0, 0, loc)
	reservedEndAt := appointmentTime.Add(45 * time.Minute)
	appt := models.Appointment{
		Status:                       "confirmed",
		AppointmentTime:              &appointmentTime,
		ReservedEndAt:                &reservedEndAt,
		LateArrivalMinServiceMinutes: 22,
	}

	if !canArriveForAppointment(appt, merchant, time.Date(2026, 3, 29, 10, 53, 0, 0, loc)) {
		t.Fatalf("want appointment still arriveable at threshold")
	}
	if canArriveForAppointment(appt, merchant, time.Date(2026, 3, 29, 10, 54, 0, 0, loc)) {
		t.Fatalf("want appointment not arriveable after threshold")
	}
}
