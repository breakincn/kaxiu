package scheduler

import (
	"testing"
	"time"

	"kabao/models"
)

func TestGetSchedulerHealthReportMarksHealthyAndStale(t *testing.T) {
	db := setupSchedulerTestDB(t)

	now := time.Date(2026, 3, 25, 10, 0, 0, 0, time.Local)
	healthyTick := now.Add(-30 * time.Second).Format(time.RFC3339Nano)
	staleTick := now.Add(-10 * time.Minute).Format(time.RFC3339Nano)

	payloadHealthy := `{"scheduler":"service_session","service":"merchant_service","pid":1001,"hostname":"devbox","tick_at":"` + healthyTick + `"}`
	payloadStale := `{"scheduler":"appointment","service":"merchant_service","pid":1001,"hostname":"devbox","tick_at":"` + staleTick + `"}`
	if err := db.Create(&models.SystemConfig{Key: schedulerHeartbeatKey(schedulerNameServiceSession), Value: payloadHealthy}).Error; err != nil {
		t.Fatalf("seed healthy heartbeat failed: %v", err)
	}
	if err := db.Create(&models.SystemConfig{Key: schedulerHeartbeatKey(schedulerNameAppointment), Value: payloadStale}).Error; err != nil {
		t.Fatalf("seed stale heartbeat failed: %v", err)
	}

	report, err := GetSchedulerHealthReport(db, now, 3)
	if err != nil {
		t.Fatalf("GetSchedulerHealthReport failed: %v", err)
	}
	if report.OverallStatus != "degraded" {
		t.Fatalf("want degraded overall status, got %s", report.OverallStatus)
	}
	if len(report.Items) != 3 {
		t.Fatalf("want 3 scheduler items, got %d", len(report.Items))
	}

	statusByKey := map[string]string{}
	for _, item := range report.Items {
		statusByKey[item.Key] = item.Status
	}
	if statusByKey[schedulerNameServiceSession] != "healthy" {
		t.Fatalf("want service_session healthy, got %s", statusByKey[schedulerNameServiceSession])
	}
	if statusByKey[schedulerNameAppointment] != "stale" {
		t.Fatalf("want appointment stale, got %s", statusByKey[schedulerNameAppointment])
	}
	if statusByKey[schedulerNameHandCard] != "missing" {
		t.Fatalf("want hand_card missing, got %s", statusByKey[schedulerNameHandCard])
	}
}
