package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
)

func TestGetMerchantSchedulerHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAppointmentPermissionTestDB(t)

	now := time.Now()
	payload := `{"scheduler":"service_session","service":"merchant_service","pid":1001,"hostname":"devbox","tick_at":"` + now.Format(time.RFC3339Nano) + `"}`
	if err := config.DB.Create(&models.SystemConfig{Key: "scheduler_heartbeat_service_session", Value: payload}).Error; err != nil {
		t.Fatalf("seed system config failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/system/scheduler-health", nil)

	GetMerchantSchedulerHealth(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			OverallStatus string `json:"overall_status"`
			Items         []struct {
				Key    string `json:"key"`
				Status string `json:"status"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Data.OverallStatus == "" {
		t.Fatalf("want overall status")
	}
	if len(resp.Data.Items) != 3 {
		t.Fatalf("want 3 scheduler items, got %d", len(resp.Data.Items))
	}
}

func TestGetAdminSchedulerHealthWithQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAppointmentPermissionTestDB(t)

	staleTick := time.Now().Add(-10 * time.Minute)
	payload := `{"scheduler":"appointment","service":"admin_service","pid":1002,"hostname":"devbox","tick_at":"` + staleTick.Format(time.RFC3339Nano) + `"}`
	if err := config.DB.Create(&models.SystemConfig{Key: "scheduler_heartbeat_appointment", Value: payload}).Error; err != nil {
		t.Fatalf("seed system config failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/system/scheduler-health?stale_minutes=1", nil)

	GetAdminSchedulerHealth(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			OverallStatus string `json:"overall_status"`
			Items         []struct {
				Key    string `json:"key"`
				Status string `json:"status"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Data.OverallStatus != "degraded" {
		t.Fatalf("want degraded overall status, got %s", resp.Data.OverallStatus)
	}
}
