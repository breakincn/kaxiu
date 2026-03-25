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

	appt := models.Appointment{
		Status:               "arrived",
		AppointmentTime:      &appointmentTime,
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
