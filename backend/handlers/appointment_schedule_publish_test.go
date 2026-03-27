package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
)

func newMerchantContext(method, path string, merchantID uint) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, nil)
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchantID)
	return c, rec
}

func TestPublishNextDayScheduleCreatesPublishings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, _, tech, otherTech, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	if err := config.DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Update("support_customer_service_mode", true).Error; err != nil {
		t.Fatalf("enable customer service mode failed: %v", err)
	}
	merchant.SupportCustomerServiceMode = true
	if err := config.DB.Model(&models.Technician{}).Where("id IN ?", []uint{tech.ID, otherTech.ID}).Update("is_active", true).Error; err != nil {
		t.Fatalf("activate technicians failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodPost, "/merchant/schedules/publish-next-day", merchant.ID)
	PublishNextDaySchedule(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	loc := appointmentLocation()
	nextDate := time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), time.Now().In(loc).Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	var count int64
	if err := config.DB.Table("technician_schedule_publishings").Where("merchant_id = ? AND publish_date >= ? AND publish_date < ? AND status = ?", merchant.ID, nextDate, nextDate.Add(24*time.Hour), "published").Count(&count).Error; err != nil {
		t.Fatalf("count publishings failed: %v", err)
	}
	if count == 0 {
		t.Fatalf("want publishings created, got 0")
	}

	queryCtx, queryRec := newMerchantContext(http.MethodGet, "/merchant/schedules/publishings?date="+nextDate.Format("2006-01-02"), merchant.ID)
	queryCtx.Request = httptest.NewRequest(http.MethodGet, "/merchant/schedules/publishings?date="+nextDate.Format("2006-01-02"), nil)
	ListSchedulePublishings(queryCtx)
	if queryRec.Code != http.StatusOK {
		t.Fatalf("want 200 from schedule publishings, got %d body=%s", queryRec.Code, queryRec.Body.String())
	}
}

func TestMarkScheduleLeaveCreatesAffectedAppointmentsAndProtectedRepairSlots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	nextDay := time.Now().In(loc).Add(24 * time.Hour)
	publishDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, loc)
	startAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 10, 0, 0, 0, loc)
	endAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 13, 0, 0, 0, loc)
	publishing := models.TechnicianSchedulePublishing{
		MerchantID:   merchant.ID,
		TechnicianID: &tech.ID,
		PublishDate:  &publishDate,
		StartAt:      &startAt,
		EndAt:        &endAt,
		Status:       "published",
	}
	if err := config.DB.Create(&publishing).Error; err != nil {
		t.Fatalf("create publishing failed: %v", err)
	}

	appointmentTime := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 10, 30, 0, 0, loc)
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("booking_root_id", appt.ID).Error; err != nil {
		t.Fatalf("set booking_root_id failed: %v", err)
	}
	appt.BookingRootID = &appt.ID
	if _, err := initializeAppointmentSettlement(config.DB, &appt, "test_seed"); err != nil {
		t.Fatalf("initialize settlement failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodPost, "/merchant/schedules/"+strconv.Itoa(int(publishing.ID))+"/leave", merchant.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(publishing.ID))}}
	MarkScheduleLeave(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	refreshed, err := loadSchedulePublishingByID(config.DB, merchant.ID, publishing.ID)
	if err != nil {
		t.Fatalf("reload publishing failed: %v", err)
	}
	if refreshed == nil {
		t.Fatalf("publishing %d not found", publishing.ID)
	}
	if refreshed.Status != "leave" {
		t.Fatalf("want leave status, got %s", refreshed.Status)
	}

	slots, err := loadProtectedRepairSlotsByPublishDate(config.DB, merchant.ID, &publishDate)
	if err != nil {
		t.Fatalf("query repair slots failed: %v", err)
	}
	filtered := make([]models.ProtectedRepairSlot, 0, len(slots))
	for _, slot := range slots {
		if slot.AppointmentID == appt.ID {
			filtered = append(filtered, slot)
		}
	}
	if len(filtered) != 1 {
		t.Fatalf("want 1 protected repair slot, got %d", len(slots))
	}
	if filtered[0].Status != "reserved" {
		t.Fatalf("want reserved repair slot, got %s", filtered[0].Status)
	}

	queryCtx, queryRec := newMerchantContext(http.MethodGet, "/merchant/schedules/affected-appointments?schedule_id="+strconv.Itoa(int(publishing.ID)), merchant.ID)
	queryCtx.Request = httptest.NewRequest(http.MethodGet, "/merchant/schedules/affected-appointments?schedule_id="+strconv.Itoa(int(publishing.ID)), nil)
	GetScheduleAffectedAppointments(queryCtx)
	if queryRec.Code != http.StatusOK {
		t.Fatalf("want 200 from affected appointments, got %d body=%s", queryRec.Code, queryRec.Body.String())
	}

	var resp struct {
		Data struct {
			AffectedAppointments       []models.Appointment         `json:"affected_appointments"`
			ProtectedRepairSlots       []models.ProtectedRepairSlot `json:"protected_repair_slots"`
			AffectedAppointmentRepairs []struct {
				Appointment struct {
					ID uint `json:"id"`
				} `json:"appointment"`
				AffectedByLeave         bool   `json:"affected_by_leave"`
				HasHighQualityCandidate bool   `json:"has_high_quality_candidate"`
				Decision                string `json:"decision"`
				Reason                  string `json:"reason"`
				CandidateTime           string `json:"candidate_time"`
				Recommendations         []struct {
					Time string `json:"time"`
				} `json:"recommendations"`
			} `json:"affected_appointment_repairs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(queryRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode affected response failed: %v", err)
	}
	if len(resp.Data.AffectedAppointments) != 1 {
		t.Fatalf("want 1 affected appointment, got %d", len(resp.Data.AffectedAppointments))
	}
	if len(resp.Data.ProtectedRepairSlots) == 0 {
		t.Fatalf("want protected repair slots in response")
	}
	if len(resp.Data.AffectedAppointmentRepairs) != 1 {
		t.Fatalf("want 1 repair inspection, got %d", len(resp.Data.AffectedAppointmentRepairs))
	}
	if !resp.Data.AffectedAppointmentRepairs[0].AffectedByLeave {
		t.Fatalf("want affected_by_leave=true")
	}
	if resp.Data.AffectedAppointmentRepairs[0].Decision == "" || resp.Data.AffectedAppointmentRepairs[0].Reason == "" {
		t.Fatalf("want structured repair decision in response")
	}
}

func TestListSchedulePublishingsIncludesUnpublishedTechniciansBeforePublish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, _, tech, otherTech, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	if err := config.DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Update("support_customer_service_mode", true).Error; err != nil {
		t.Fatalf("enable customer service mode failed: %v", err)
	}
	merchant.SupportCustomerServiceMode = true
	if err := config.DB.Model(&models.Technician{}).Where("id IN ?", []uint{tech.ID, otherTech.ID}).Update("is_active", true).Error; err != nil {
		t.Fatalf("activate technicians failed: %v", err)
	}

	loc := appointmentLocation()
	targetDate := time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), time.Now().In(loc).Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	path := "/merchant/schedules/publishings?date=" + targetDate.Format("2006-01-02")
	c, rec := newMerchantContext(http.MethodGet, path, merchant.ID)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)

	ListSchedulePublishings(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Publishings []models.TechnicianSchedulePublishing `json:"publishings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Data.Publishings) != 2 {
		t.Fatalf("want 2 preview rows, got %d body=%s", len(resp.Data.Publishings), rec.Body.String())
	}
	for _, row := range resp.Data.Publishings {
		if row.Status != "unpublished" {
			t.Fatalf("want unpublished preview row, got %s", row.Status)
		}
		if row.Technician == nil || row.Technician.Account == "" || row.Technician.Name == "" {
			t.Fatalf("want technician identity in preview row, got %+v", row.Technician)
		}
	}
}

func TestMarkScheduleLeaveByTechnicianBeforePublishAndPublishSkipsLeaveRows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, _, tech, otherTech, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	if err := config.DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Update("support_customer_service_mode", true).Error; err != nil {
		t.Fatalf("enable customer service mode failed: %v", err)
	}
	merchant.SupportCustomerServiceMode = true
	if err := config.DB.Model(&models.Technician{}).Where("id IN ?", []uint{tech.ID, otherTech.ID}).Update("is_active", true).Error; err != nil {
		t.Fatalf("activate technicians failed: %v", err)
	}

	loc := appointmentLocation()
	targetDate := time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), time.Now().In(loc).Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	body, _ := json.Marshal(gin.H{
		"date":          targetDate.Format("2006-01-02"),
		"technician_id": tech.ID,
	})
	c, rec := newMerchantContext(http.MethodPost, "/merchant/schedules/leave", merchant.ID)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/schedules/leave", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	MarkScheduleLeaveByTechnician(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	c, rec = newMerchantContext(http.MethodPost, "/merchant/schedules/publish-next-day", merchant.ID)
	PublishNextDaySchedule(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 on publish, got %d body=%s", rec.Code, rec.Body.String())
	}

	rows, err := loadSchedulePublishingsByDate(config.DB, merchant.ID, targetDate)
	if err != nil {
		t.Fatalf("load rows failed: %v", err)
	}
	var leaveCount int
	var publishedCount int
	for _, row := range rows {
		if row.TechnicianID == nil {
			continue
		}
		if *row.TechnicianID == tech.ID && row.Status == "leave" {
			leaveCount++
		}
		if *row.TechnicianID == tech.ID && row.Status == "published" {
			t.Fatalf("leave technician should not be published")
		}
		if *row.TechnicianID == otherTech.ID && row.Status == "published" {
			publishedCount++
		}
	}
	if leaveCount == 0 {
		t.Fatalf("want leave row for technician %d", tech.ID)
	}
	if publishedCount == 0 {
		t.Fatalf("want published row for technician %d", otherTech.ID)
	}
}

func TestWithdrawNextDayScheduleConvertsPublishedRowsToCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, _, tech, otherTech, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	if err := config.DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Update("support_customer_service_mode", true).Error; err != nil {
		t.Fatalf("enable customer service mode failed: %v", err)
	}
	merchant.SupportCustomerServiceMode = true
	if err := config.DB.Model(&models.Technician{}).Where("id IN ?", []uint{tech.ID, otherTech.ID}).Update("is_active", true).Error; err != nil {
		t.Fatalf("activate technicians failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodPost, "/merchant/schedules/publish-next-day", merchant.ID)
	PublishNextDaySchedule(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	loc := appointmentLocation()
	targetDate := time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), time.Now().In(loc).Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	path := "/merchant/schedules/withdraw-next-day?date=" + targetDate.Format("2006-01-02")
	c, rec = newMerchantContext(http.MethodPost, path, merchant.ID)
	c.Request = httptest.NewRequest(http.MethodPost, path, nil)
	WithdrawNextDaySchedule(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("withdraw want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	rows, err := loadSchedulePublishingsByDate(config.DB, merchant.ID, targetDate)
	if err != nil {
		t.Fatalf("load rows failed: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("want rows after withdraw")
	}
	for _, row := range rows {
		if row.Status != "canceled" {
			t.Fatalf("want canceled rows after withdraw, got %s", row.Status)
		}
	}
}

func TestUnmarkScheduleLeaveByTechnicianRestoresPreviousState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, _, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	if err := config.DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Update("support_customer_service_mode", true).Error; err != nil {
		t.Fatalf("enable customer service mode failed: %v", err)
	}
	merchant.SupportCustomerServiceMode = true
	if err := config.DB.Model(&models.Technician{}).Where("id = ?", tech.ID).Update("is_active", true).Error; err != nil {
		t.Fatalf("activate technician failed: %v", err)
	}

	loc := appointmentLocation()
	targetDate := time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), time.Now().In(loc).Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)

	body, _ := json.Marshal(gin.H{
		"date":          targetDate.Format("2006-01-02"),
		"technician_id": tech.ID,
	})
	c, rec := newMerchantContext(http.MethodPost, "/merchant/schedules/leave", merchant.ID)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/schedules/leave", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	MarkScheduleLeaveByTechnician(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("mark leave want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	c, rec = newMerchantContext(http.MethodPost, "/merchant/schedules/unleave", merchant.ID)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/schedules/unleave", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	UnmarkScheduleLeaveByTechnician(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("unleave want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	rows, err := buildSchedulePublishingRowsForDate(config.DB, merchant, targetDate)
	if err != nil {
		t.Fatalf("build rows failed: %v", err)
	}
	found := false
	for _, row := range rows {
		if row.TechnicianID != nil && *row.TechnicianID == tech.ID {
			found = true
			if row.Status != "unpublished" {
				t.Fatalf("want unpublished after unleave, got %s", row.Status)
			}
		}
	}
	if !found {
		t.Fatalf("want technician row after unleave")
	}
}

func TestScheduleEndpointsRequireAppointmentManagePermissionForStaff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAppointmentPermissionTestDB(t)

	merchant, _, tech, _, role, _ := seedAppointmentPermissionFixture(t, config.DB)
	seedAppointmentPermission(t, config.DB, role.ID, "merchant.appointment.view")

	loc := appointmentLocation()
	nextDay := time.Now().In(loc).Add(24 * time.Hour)
	publishDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, loc)
	startAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 10, 0, 0, 0, loc)
	endAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 13, 0, 0, 0, loc)
	publishing := models.TechnicianSchedulePublishing{
		MerchantID:   merchant.ID,
		TechnicianID: &tech.ID,
		PublishDate:  &publishDate,
		StartAt:      &startAt,
		EndAt:        &endAt,
		Status:       "published",
	}
	if err := config.DB.Create(&publishing).Error; err != nil {
		t.Fatalf("create publishing failed: %v", err)
	}

	cases := []struct {
		name string
		run  func() *httptest.ResponseRecorder
	}{
		{
			name: "list publishings",
			run: func() *httptest.ResponseRecorder {
				path := "/merchant/schedules/publishings?date=" + publishDate.Format("2006-01-02")
				c, rec := newStaffContext(http.MethodGet, path, merchant.ID, tech.ID, role.ID)
				c.Request = httptest.NewRequest(http.MethodGet, path, nil)
				ListSchedulePublishings(c)
				return rec
			},
		},
		{
			name: "publish next day",
			run: func() *httptest.ResponseRecorder {
				c, rec := newStaffContext(http.MethodPost, "/merchant/schedules/publish-next-day", merchant.ID, tech.ID, role.ID)
				PublishNextDaySchedule(c)
				return rec
			},
		},
		{
			name: "mark leave",
			run: func() *httptest.ResponseRecorder {
				path := "/merchant/schedules/" + strconv.Itoa(int(publishing.ID)) + "/leave"
				c, rec := newStaffContext(http.MethodPost, path, merchant.ID, tech.ID, role.ID)
				c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(publishing.ID))}}
				MarkScheduleLeave(c)
				return rec
			},
		},
		{
			name: "affected appointments",
			run: func() *httptest.ResponseRecorder {
				path := "/merchant/schedules/affected-appointments?schedule_id=" + strconv.Itoa(int(publishing.ID))
				c, rec := newStaffContext(http.MethodGet, path, merchant.ID, tech.ID, role.ID)
				c.Request = httptest.NewRequest(http.MethodGet, path, nil)
				GetScheduleAffectedAppointments(c)
				return rec
			},
		},
	}

	for _, tc := range cases {
		rec := tc.run()
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s: want 403, got %d body=%s", tc.name, rec.Code, rec.Body.String())
		}
	}
}

func TestCreateAppointmentRescheduleRequestRejectsUserWhenAffectedByLeave(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	nextDay := time.Now().In(loc).Add(24 * time.Hour)
	appointmentTime := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 11, 0, 0, 0, loc)
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	publishDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, loc)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	repairSlot := models.ProtectedRepairSlot{
		MerchantID:    merchant.ID,
		AppointmentID: appt.ID,
		TechnicianID:  &tech.ID,
		PublishDate:   &publishDate,
		StartAt:       &appointmentTime,
		EndAt:         &reservedEnd,
		Status:        "reserved",
		SourceType:    "leave",
	}
	if err := config.DB.Create(&repairSlot).Error; err != nil {
		t.Fatalf("create repair slot failed: %v", err)
	}

	body, _ := json.Marshal(gin.H{
		"appointment_time": appointmentTime.Format("2006-01-02 15:04:05"),
		"technician_id":    tech.ID,
		"reason":           "用户想改时间",
	})
	c, rec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-requests", user.ID, body)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CreateAppointmentRescheduleRequest(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body == "" || !containsText(body, "需由商户发起保护性改签") {
		t.Fatalf("want protective reschedule rejection, got body=%s", body)
	}
}

func TestGetTechnicianMonthlyDisruptionsReturnsCounterAndLedgerSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	apptTime := time.Date(2026, 3, 24, 10, 30, 0, 0, appointmentLocation())
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &apptTime,
		Status:          "failed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	if err := config.DB.Create(&models.TechnicianMonthlyDisruptionCounter{
		MerchantID:           merchant.ID,
		TechnicianID:         tech.ID,
		MonthKey:             "2026-03",
		LeaveDisruptionCount: 2,
		FirstExemptUsed:      true,
	}).Error; err != nil {
		t.Fatalf("create counter failed: %v", err)
	}
	if err := config.DB.Create(&models.TechnicianDisruptionLedger{
		AppointmentID:                   appt.ID,
		MerchantID:                      merchant.ID,
		TechnicianID:                    tech.ID,
		UserID:                          user.ID,
		MonthKey:                        "2026-03",
		DisruptionReason:                "technician_leave",
		LiabilityLevel:                  "technician_chargeable",
		SalarySettlementReferenceStatus: "open",
	}).Error; err != nil {
		t.Fatalf("create ledger failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodGet, "/merchant/technicians/"+strconv.Itoa(int(tech.ID))+"/monthly-disruptions?month=2026-03", merchant.ID)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/technicians/"+strconv.Itoa(int(tech.ID))+"/monthly-disruptions?month=2026-03", nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(tech.ID))}}
	GetTechnicianMonthlyDisruptions(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Month                     string                              `json:"month"`
			LeaveDisruptionCount      int                                 `json:"leave_disruption_count"`
			FirstExemptUsed           bool                                `json:"first_exempt_used"`
			MerchantExemptCount       int                                 `json:"merchant_exempt_count"`
			TechnicianChargeableCount int                                 `json:"technician_chargeable_count"`
			Items                     []models.TechnicianDisruptionLedger `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Data.Month != "2026-03" || resp.Data.LeaveDisruptionCount != 2 || !resp.Data.FirstExemptUsed {
		t.Fatalf("want month counter returned, got %+v", resp.Data)
	}
	if resp.Data.TechnicianChargeableCount != 1 || resp.Data.MerchantExemptCount != 0 {
		t.Fatalf("want liability summary returned, got %+v", resp.Data)
	}
	if len(resp.Data.Items) != 1 || resp.Data.Items[0].AppointmentID != appt.ID {
		t.Fatalf("want ledger items returned, got %+v", resp.Data.Items)
	}
}

func TestCancelAppointmentRejectsMerchantWhenAffectedByLeaveAndHasRepairCandidate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	nextDay := time.Now().In(loc).Add(24 * time.Hour)
	publishDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, loc)
	startAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 10, 0, 0, 0, loc)
	endAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 13, 0, 0, 0, loc)
	publishing := models.TechnicianSchedulePublishing{
		MerchantID:   merchant.ID,
		TechnicianID: &tech.ID,
		PublishDate:  &publishDate,
		StartAt:      &startAt,
		EndAt:        &endAt,
		Status:       "published",
	}
	if err := config.DB.Create(&publishing).Error; err != nil {
		t.Fatalf("create publishing failed: %v", err)
	}
	appointmentTime := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 10, 30, 0, 0, loc)
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	repairSlot := models.ProtectedRepairSlot{
		MerchantID:    merchant.ID,
		AppointmentID: appt.ID,
		TechnicianID:  &tech.ID,
		PublishDate:   &publishDate,
		StartAt:       &appointmentTime,
		EndAt:         &reservedEnd,
		Status:        "reserved",
		SourceType:    "leave",
	}
	if err := config.DB.Create(&repairSlot).Error; err != nil {
		t.Fatalf("create repair slot failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel", merchant.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CancelAppointment(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body == "" || !containsText(body, "高质量修复方案") {
		t.Fatalf("want high quality repair rejection, got body=%s", body)
	}
}

func TestGetAppointmentRescheduleEligibilityRejectsUserWhenAffectedByLeave(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	nextDay := time.Now().In(loc).Add(24 * time.Hour)
	publishDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, loc)
	appointmentTime := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 11, 0, 0, 0, loc)
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	repairSlot := models.ProtectedRepairSlot{
		MerchantID:    merchant.ID,
		AppointmentID: appt.ID,
		TechnicianID:  &tech.ID,
		PublishDate:   &publishDate,
		StartAt:       &appointmentTime,
		EndAt:         &reservedEnd,
		Status:        "reserved",
		SourceType:    "leave",
	}
	if err := config.DB.Create(&repairSlot).Error; err != nil {
		t.Fatalf("create repair slot failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-eligibility", nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	c.Set("auth_type", "user")
	c.Set("user_id", user.ID)
	GetAppointmentRescheduleEligibility(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !containsText(body, "需由商户发起保护性改签") {
		t.Fatalf("want protective eligibility message, got body=%s", body)
	}
}

func TestGetAppointmentRescheduleSlotsAllowsMerchantRepairCandidateWhenAffectedByLeave(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	nextDay := time.Now().In(loc).Add(24 * time.Hour)
	publishDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, loc)
	startAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 10, 0, 0, 0, loc)
	endAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 13, 0, 0, 0, loc)
	publishing := models.TechnicianSchedulePublishing{
		MerchantID:   merchant.ID,
		TechnicianID: &tech.ID,
		PublishDate:  &publishDate,
		StartAt:      &startAt,
		EndAt:        &endAt,
		Status:       "published",
	}
	if err := config.DB.Create(&publishing).Error; err != nil {
		t.Fatalf("create publishing failed: %v", err)
	}
	appointmentTime := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 10, 30, 0, 0, loc)
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	repairSlot := models.ProtectedRepairSlot{
		MerchantID:    merchant.ID,
		AppointmentID: appt.ID,
		TechnicianID:  &tech.ID,
		PublishDate:   &publishDate,
		StartAt:       &appointmentTime,
		EndAt:         &reservedEnd,
		Status:        "reserved",
		SourceType:    "leave",
	}
	if err := config.DB.Create(&repairSlot).Error; err != nil {
		t.Fatalf("create repair slot failed: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-slots?date="+publishDate.Format("2006-01-02"), nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchant.ID)
	GetAppointmentRescheduleSlots(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !containsText(body, "time_slots") {
		t.Fatalf("want time slots payload, got body=%s", body)
	}
}

func TestProtectiveRescheduleFlowCreatesReplacementAppointment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	nextDay := time.Now().In(loc).Add(24 * time.Hour)
	publishDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, loc)
	startAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 10, 0, 0, 0, loc)
	endAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 13, 0, 0, 0, loc)
	publishing := models.TechnicianSchedulePublishing{
		MerchantID:   merchant.ID,
		TechnicianID: &tech.ID,
		PublishDate:  &publishDate,
		StartAt:      &startAt,
		EndAt:        &endAt,
		Status:       "published",
	}
	if err := config.DB.Create(&publishing).Error; err != nil {
		t.Fatalf("create publishing failed: %v", err)
	}
	originalTime := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 10, 0, 0, 0, loc)
	originalReservedEnd := originalTime.Add(time.Duration(project.Duration) * time.Minute)
	originalOccupiedEnd := originalReservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	original := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &originalTime,
		ReservedStartAt: &originalTime,
		ReservedEndAt:   &originalReservedEnd,
		OccupiedEndAt:   &originalOccupiedEnd,
		BookingRootID:   nil,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&original).Error; err != nil {
		t.Fatalf("create original appointment failed: %v", err)
	}
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", original.ID).Update("booking_root_id", original.ID).Error; err != nil {
		t.Fatalf("set booking_root_id failed: %v", err)
	}
	original.BookingRootID = &original.ID
	repairSlot := models.ProtectedRepairSlot{
		MerchantID:    merchant.ID,
		AppointmentID: original.ID,
		TechnicianID:  &tech.ID,
		PublishDate:   &publishDate,
		StartAt:       &originalTime,
		EndAt:         &originalReservedEnd,
		Status:        "reserved",
		SourceType:    "leave",
	}
	if err := config.DB.Create(&repairSlot).Error; err != nil {
		t.Fatalf("create repair slot failed: %v", err)
	}

	newTime := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 11, 15, 0, 0, loc)
	body, _ := json.Marshal(gin.H{
		"appointment_time": newTime.Format("2006-01-02 15:04:05"),
		"technician_id":    tech.ID,
		"reason":           "客服请假，提供保护性改签",
	})
	merchantCtx, merchantRec := newMerchantContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(original.ID))+"/reschedule-requests", merchant.ID)
	merchantCtx.Request = httptest.NewRequest(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(original.ID))+"/reschedule-requests", bytes.NewReader(body))
	merchantCtx.Request.Header.Set("Content-Type", "application/json")
	merchantCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(original.ID))}}
	CreateAppointmentRescheduleRequest(merchantCtx)
	if merchantRec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", merchantRec.Code, merchantRec.Body.String())
	}

	var proposalID uint
	if err := config.DB.Table("appointment_reschedule_requests").Select("id").Order("id desc").Limit(1).Scan(&proposalID).Error; err != nil {
		t.Fatalf("load proposal id failed: %v", err)
	}
	proposalPtr, err := loadAppointmentRescheduleRequestByID(config.DB, proposalID)
	if err != nil || proposalPtr == nil {
		t.Fatalf("load proposal failed: %v", err)
	}
	proposal := *proposalPtr
	if proposal.Status != "pending_user" {
		t.Fatalf("want pending_user proposal, got %s", proposal.Status)
	}

	userCtx, userRec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(original.ID))+"/reschedule-requests/"+strconv.Itoa(int(proposal.ID))+"/accept", user.ID, nil)
	userCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(original.ID))}, {Key: "request_id", Value: strconv.Itoa(int(proposal.ID))}}
	AcceptAppointmentRescheduleRequest(userCtx)
	if userRec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", userRec.Code, userRec.Body.String())
	}

	updatedOriginal := mustLoadAppointmentForTest(t, original.ID)
	if updatedOriginal.Status != "canceled" {
		t.Fatalf("want original canceled, got %s", updatedOriginal.Status)
	}
	if updatedOriginal.ReplacedByAppointmentID == nil || *updatedOriginal.ReplacedByAppointmentID == 0 {
		t.Fatalf("want original replaced_by_appointment_id set")
	}
	replacement := mustLoadAppointmentForTest(t, *updatedOriginal.ReplacedByAppointmentID)
	if replacement.BookingRootID == nil || *replacement.BookingRootID != original.ID {
		t.Fatalf("want replacement booking_root_id=%d, got %+v", original.ID, replacement.BookingRootID)
	}
	if replacement.ReservedStartAt == nil || !replacement.ReservedStartAt.Equal(newTime) {
		t.Fatalf("want replacement reserved_start_at=%s, got %+v", newTime.Format(time.RFC3339), replacement.ReservedStartAt)
	}
	if replacement.ReservedEndAt == nil || replacement.OccupiedEndAt == nil {
		t.Fatalf("want replacement scheduling snapshot populated")
	}
	if replacement.Status != "confirmed" {
		t.Fatalf("want replacement confirmed, got %s", replacement.Status)
	}
}

func TestCancelAppointmentRejectsMerchantWithinFiveHours(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	soon := time.Now().In(loc).Add(4 * time.Hour)
	appointmentTime := time.Date(soon.Year(), soon.Month(), soon.Day(), soon.Hour(), 0, 0, 0, loc)
	if !appointmentTime.After(time.Now().In(loc)) {
		appointmentTime = appointmentTime.Add(time.Hour)
	}
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel", merchant.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CancelAppointment(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !containsText(body, "取消申请") {
		t.Fatalf("want cancel-request gating message, got body=%s", body)
	}
}

func TestCancelAppointmentStoresMerchantReasonAndRefundSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	later := time.Now().In(loc).Add(8 * time.Hour)
	appointmentTime := time.Date(later.Year(), later.Month(), later.Day(), later.Hour(), 0, 0, 0, loc)
	if !appointmentTime.After(time.Now().In(loc).Add(5 * time.Hour)) {
		appointmentTime = appointmentTime.Add(2 * time.Hour)
	}
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	if _, err := initializeAppointmentSettlement(config.DB, &appt, "test_seed"); err != nil {
		t.Fatalf("initialize settlement failed: %v", err)
	}

	body, _ := json.Marshal(gin.H{"reason": "门店临时停业整备"})
	c, rec := newMerchantContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel", merchant.ID)
	c.Request = httptest.NewRequest(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CancelAppointment(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	updated := mustLoadAppointmentForTest(t, appt.ID)
	if updated.Status != "canceled" {
		t.Fatalf("want canceled, got %s", updated.Status)
	}
	if updated.MerchantCancelReason != "门店临时停业整备" {
		t.Fatalf("want merchant cancel reason saved, got %s", updated.MerchantCancelReason)
	}
	if updated.SettlementStatusSnapshot != "refunded" {
		t.Fatalf("want settlement snapshot refunded, got %s", updated.SettlementStatusSnapshot)
	}
	var settlement models.AppointmentSettlement
	if err := config.DB.First(&settlement, *appt.AppointmentSettlementID).Error; err != nil {
		t.Fatalf("load settlement failed: %v", err)
	}
	if settlement.Status != "refunded" || settlement.LatestReason != "merchant_direct_cancel" {
		t.Fatalf("want refunded settlement with merchant_direct_cancel, got status=%s latest_reason=%s", settlement.Status, settlement.LatestReason)
	}
}

func TestUpdateAppointmentUserRebuttalPersistsNoteAfterMerchantCancel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	later := time.Now().In(loc).Add(8 * time.Hour)
	appointmentTime := time.Date(later.Year(), later.Month(), later.Day(), later.Hour(), 0, 0, 0, loc)
	if !appointmentTime.After(time.Now().In(loc).Add(5 * time.Hour)) {
		appointmentTime = appointmentTime.Add(2 * time.Hour)
	}
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	cancelBody, _ := json.Marshal(gin.H{"reason": "设备故障无法履约"})
	mc, mrec := newMerchantContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel", merchant.ID)
	mc.Request = httptest.NewRequest(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel", bytes.NewReader(cancelBody))
	mc.Request.Header.Set("Content-Type", "application/json")
	mc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CancelAppointment(mc)
	if mrec.Code != http.StatusOK {
		t.Fatalf("want 200 when merchant cancels, got %d body=%s", mrec.Code, mrec.Body.String())
	}

	rebuttalBody, _ := json.Marshal(gin.H{"user_rebuttal_note": "我已按时到店并接受等待"})
	uc, urec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/rebuttal", user.ID, rebuttalBody)
	uc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	UpdateAppointmentUserRebuttal(uc)
	if urec.Code != http.StatusOK {
		t.Fatalf("want 200 when rebuttal saved, got %d body=%s", urec.Code, urec.Body.String())
	}

	updated := mustLoadAppointmentForTest(t, appt.ID)
	if updated.UserRebuttalNote != "我已按时到店并接受等待" {
		t.Fatalf("want rebuttal saved, got %s", updated.UserRebuttalNote)
	}
	if !containsText(updated.ResolutionNote, "用户抗辩") {
		t.Fatalf("want resolution note appended with rebuttal, got %s", updated.ResolutionNote)
	}
}

func TestAppointmentCancelRequestFlowCancelsAppointmentAfterUserAccepts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	soon := time.Now().In(loc).Add(4 * time.Hour)
	appointmentTime := time.Date(soon.Year(), soon.Month(), soon.Day(), soon.Hour(), 0, 0, 0, loc)
	if !appointmentTime.After(time.Now().In(loc)) {
		appointmentTime = appointmentTime.Add(time.Hour)
	}
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("booking_root_id", appt.ID).Error; err != nil {
		t.Fatalf("set booking_root_id failed: %v", err)
	}
	appt.BookingRootID = &appt.ID
	if _, err := initializeAppointmentSettlement(config.DB, &appt, "test_seed"); err != nil {
		t.Fatalf("initialize settlement failed: %v", err)
	}

	body, _ := json.Marshal(gin.H{"reason": "门店临时无法履约"})
	merchantCtx, merchantRec := newMerchantContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel-requests", merchant.ID)
	merchantCtx.Request = httptest.NewRequest(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel-requests", bytes.NewReader(body))
	merchantCtx.Request.Header.Set("Content-Type", "application/json")
	merchantCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CreateAppointmentCancelRequest(merchantCtx)
	if merchantRec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", merchantRec.Code, merchantRec.Body.String())
	}

	var requestID uint
	if err := config.DB.Table("appointment_cancel_requests").Select("id").Order("id desc").Limit(1).Scan(&requestID).Error; err != nil {
		t.Fatalf("load cancel request id failed: %v", err)
	}
	requestPtr, err := loadAppointmentCancelRequestByID(config.DB, requestID)
	if err != nil || requestPtr == nil {
		t.Fatalf("load cancel request failed: %v", err)
	}
	if requestPtr.Status != "pending_user" {
		t.Fatalf("want pending_user cancel request, got %s", requestPtr.Status)
	}

	userCtx, userRec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel-requests/"+strconv.Itoa(int(requestID))+"/accept", user.ID, nil)
	userCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}, {Key: "request_id", Value: strconv.Itoa(int(requestID))}}
	AcceptAppointmentCancelRequest(userCtx)
	if userRec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", userRec.Code, userRec.Body.String())
	}

	updated := mustLoadAppointmentForTest(t, appt.ID)
	if updated.Status != "canceled" {
		t.Fatalf("want appointment canceled, got %s", updated.Status)
	}
	reloadedReq, err := loadAppointmentCancelRequestByID(config.DB, requestID)
	if err != nil || reloadedReq == nil {
		t.Fatalf("reload cancel request failed: %v", err)
	}
	if reloadedReq.Status != "accepted" {
		t.Fatalf("want cancel request accepted, got %s", reloadedReq.Status)
	}
	var settlement models.AppointmentSettlement
	if err := config.DB.First(&settlement, *appt.AppointmentSettlementID).Error; err != nil {
		t.Fatalf("load settlement failed: %v", err)
	}
	if settlement.Status != "refunded" || updated.SettlementStatusSnapshot != "refunded" {
		t.Fatalf("want refunded settlement snapshot, got settlement=%s appointment=%s", settlement.Status, updated.SettlementStatusSnapshot)
	}
}

func TestAppointmentCancelRequestFlowKeepsAppointmentWhenUserRejects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	soon := time.Now().In(loc).Add(4 * time.Hour)
	appointmentTime := time.Date(soon.Year(), soon.Month(), soon.Day(), soon.Hour(), 0, 0, 0, loc)
	if !appointmentTime.After(time.Now().In(loc)) {
		appointmentTime = appointmentTime.Add(time.Hour)
	}
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	body, _ := json.Marshal(gin.H{"reason": "门店临时无法履约"})
	merchantCtx, merchantRec := newMerchantContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel-requests", merchant.ID)
	merchantCtx.Request = httptest.NewRequest(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel-requests", bytes.NewReader(body))
	merchantCtx.Request.Header.Set("Content-Type", "application/json")
	merchantCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CreateAppointmentCancelRequest(merchantCtx)
	if merchantRec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", merchantRec.Code, merchantRec.Body.String())
	}

	var requestID uint
	if err := config.DB.Table("appointment_cancel_requests").Select("id").Order("id desc").Limit(1).Scan(&requestID).Error; err != nil {
		t.Fatalf("load cancel request id failed: %v", err)
	}

	rejectBody, _ := json.Marshal(gin.H{"objection_note": "我仍然按时到店"})
	userCtx, userRec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/cancel-requests/"+strconv.Itoa(int(requestID))+"/reject", user.ID, rejectBody)
	userCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}, {Key: "request_id", Value: strconv.Itoa(int(requestID))}}
	RejectAppointmentCancelRequest(userCtx)
	if userRec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", userRec.Code, userRec.Body.String())
	}

	updated := mustLoadAppointmentForTest(t, appt.ID)
	if updated.Status != "confirmed" {
		t.Fatalf("want appointment stay confirmed, got %s", updated.Status)
	}
	reloadedReq, err := loadAppointmentCancelRequestByID(config.DB, requestID)
	if err != nil || reloadedReq == nil {
		t.Fatalf("reload cancel request failed: %v", err)
	}
	if reloadedReq.Status != "rejected" {
		t.Fatalf("want cancel request rejected, got %s", reloadedReq.Status)
	}
	if reloadedReq.ObjectionNote != "我仍然按时到店" {
		t.Fatalf("want objection note persisted, got %s", reloadedReq.ObjectionNote)
	}
}

func TestCheckInAppointmentCreatesSessionWithoutDeductingCardTimes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	startAt := time.Now().In(loc).Add(5 * time.Minute)
	appointmentTime := time.Date(startAt.Year(), startAt.Month(), startAt.Day(), startAt.Hour(), startAt.Minute(), 0, 0, loc)
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("booking_root_id", appt.ID).Error; err != nil {
		t.Fatalf("set booking_root_id failed: %v", err)
	}
	appt.BookingRootID = &appt.ID
	if _, err := initializeAppointmentSettlement(config.DB, &appt, "test_seed"); err != nil {
		t.Fatalf("initialize settlement failed: %v", err)
	}

	beforeCard := models.Card{}
	if err := config.DB.First(&beforeCard, card1.ID).Error; err != nil {
		t.Fatalf("load card before checkin failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/check-in", merchant.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CheckInAppointment(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	updated := mustLoadAppointmentForTest(t, appt.ID)
	if updated.Status != "arrived" {
		t.Fatalf("want appointment arrived, got %s", updated.Status)
	}
	if updated.ActualArrivedAt == nil || updated.ServiceSessionID == nil || updated.UsageID == nil {
		t.Fatalf("want actual_arrived_at/service_session_id/usage_id filled, got %+v", updated)
	}
	afterCard := models.Card{}
	if err := config.DB.First(&afterCard, card1.ID).Error; err != nil {
		t.Fatalf("load card after checkin failed: %v", err)
	}
	if afterCard.RemainTimes != beforeCard.RemainTimes || afterCard.UsedTimes != beforeCard.UsedTimes {
		t.Fatalf("want card times unchanged on appointment checkin, before=%d/%d after=%d/%d", beforeCard.RemainTimes, beforeCard.UsedTimes, afterCard.RemainTimes, afterCard.UsedTimes)
	}
	var usage struct {
		Status string `gorm:"column:status"`
	}
	if err := config.DB.Table("usages").Select("status").Where("id = ?", *updated.UsageID).Take(&usage).Error; err != nil {
		t.Fatalf("load usage failed: %v", err)
	}
	if usage.Status != "in_progress" {
		t.Fatalf("want usage in_progress, got %s", usage.Status)
	}
	var session struct {
		Status     string `gorm:"column:status"`
		SourceType string `gorm:"column:source_type"`
		SourceID   *uint  `gorm:"column:source_id"`
	}
	if err := config.DB.Table("service_sessions").Select("status, source_type, source_id").Where("id = ?", *updated.ServiceSessionID).Take(&session).Error; err != nil {
		t.Fatalf("load service session failed: %v", err)
	}
	if session.SourceType != serviceSessionSourceAppointment || session.SourceID == nil || *session.SourceID != appt.ID {
		t.Fatalf("want appointment source session, got source_type=%s source_id=%+v", session.SourceType, session.SourceID)
	}
	if models.NormalizeSessionStatus(session.Status) != "start_pending" {
		t.Fatalf("want appointment check-in enter execution path, got %s", session.Status)
	}
}

func TestCheckInAppointmentKeepsExecutionPathAndMarksDelayPendingWhenTechnicianBusy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, card2 := seedAppointmentPlacementFixture(t, true)
	loc := appointmentLocation()
	now := time.Now().In(loc)
	appointmentTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, loc).Add(5 * time.Minute)
	reservedEnd := appointmentTime.Add(time.Duration(project.Duration) * time.Minute)
	occupiedEnd := reservedEnd.Add(time.Duration(project.ServiceGapMinutes) * time.Minute)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &appointmentTime,
		ReservedStartAt: &appointmentTime,
		ReservedEndAt:   &reservedEnd,
		OccupiedEndAt:   &occupiedEnd,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("booking_root_id", appt.ID).Error; err != nil {
		t.Fatalf("set booking_root_id failed: %v", err)
	}
	appt.BookingRootID = &appt.ID
	if _, err := initializeAppointmentSettlement(config.DB, &appt, "test_seed"); err != nil {
		t.Fatalf("initialize settlement failed: %v", err)
	}

	blockerUsedAt := now.Add(-10 * time.Minute)
	blockerUsage := models.Usage{
		CardID:             card2.ID,
		MerchantID:         merchant.ID,
		ProjectID:          &project.ID,
		UsedTimes:          1,
		UsedAt:             &blockerUsedAt,
		VerifyCode:         "BLOCKER-CHECKIN",
		VerifyCodeExpireAt: now.Add(30 * time.Minute).Unix(),
		Status:             "in_progress",
	}
	if err := config.DB.Create(&blockerUsage).Error; err != nil {
		t.Fatalf("create blocker usage failed: %v", err)
	}
	predictedReadyAt := now.Add(22 * time.Minute)
	blockerSession := models.ServiceSession{
		MerchantID:                       merchant.ID,
		UserID:                           user.ID,
		CardID:                           card2.ID,
		ProjectID:                        &project.ID,
		InitialUsageID:                   blockerUsage.ID,
		VerifyCode:                       blockerUsage.VerifyCode,
		SessionMode:                      models.ResolveSessionMode(&merchant),
		SourceType:                       serviceSessionSourceWalkIn,
		Status:                           models.WithCSPrefix("serving"),
		TechnicianID:                     &tech.ID,
		LastTechnicianID:                 &tech.ID,
		DurationMinutes:                  project.Duration,
		PredictedReadyAt:                 &predictedReadyAt,
		ScheduledFinishAt:                &predictedReadyAt,
		PredictedAppointmentDelayMinutes: 0,
	}
	if err := config.DB.Create(&blockerSession).Error; err != nil {
		t.Fatalf("create blocker session failed: %v", err)
	}

	c, rec := newMerchantContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/check-in", merchant.ID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CheckInAppointment(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			PredictedDelayMinutes int    `json:"predicted_delay_minutes"`
			SessionWaitState      string `json:"session_wait_state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode check-in response failed: %v", err)
	}
	if resp.Data.PredictedDelayMinutes <= 0 {
		t.Fatalf("want positive predicted delay in response, got %+v", resp.Data)
	}
	if resp.Data.SessionWaitState != "start_pending" {
		t.Fatalf("want execution-path session_wait_state, got %+v", resp.Data)
	}

	updated := mustLoadAppointmentForTest(t, appt.ID)
	if updated.Status != "arrived" {
		t.Fatalf("want appointment arrived, got %s", updated.Status)
	}
	if updated.PredictedWaitMinutes <= 0 {
		t.Fatalf("want appointment predicted delay persisted, got %+v", updated)
	}
	if !updated.MerchantBreachPending {
		t.Fatalf("want merchant_breach_pending=true, got %+v", updated)
	}
	if updated.LiabilityLevel != "pending_merchant" {
		t.Fatalf("want pending_merchant liability, got %+v", updated)
	}

	var session struct {
		Status                           string `gorm:"column:status"`
		PredictedAppointmentDelayMinutes int    `gorm:"column:predicted_appointment_delay_minutes"`
	}
	if err := config.DB.Table("service_sessions").
		Select("status, predicted_appointment_delay_minutes").
		Where("id = ?", *updated.ServiceSessionID).
		Take(&session).Error; err != nil {
		t.Fatalf("load created service session failed: %v", err)
	}
	var predictedReadyAtText string
	if err := config.DB.Table("service_sessions").
		Select("predicted_ready_at").
		Where("id = ?", *updated.ServiceSessionID).
		Scan(&predictedReadyAtText).Error; err != nil {
		t.Fatalf("load predicted_ready_at failed: %v", err)
	}
	if models.NormalizeSessionStatus(session.Status) != "start_pending" {
		t.Fatalf("want start_pending on the new appointment main path, got %+v", session)
	}
	if session.PredictedAppointmentDelayMinutes <= 0 || strings.TrimSpace(predictedReadyAtText) == "" {
		t.Fatalf("want predicted delay persisted on service session, got %+v predicted_ready_at=%q", session, predictedReadyAtText)
	}

	var settlement struct {
		Status                   string `gorm:"column:status"`
		LatestReason             string `gorm:"column:latest_reason"`
		LiabilityLevel           string `gorm:"column:liability_level"`
		MerchantBreachPending    bool   `gorm:"column:merchant_breach_pending"`
		SettlementStatusSnapshot string `gorm:"column:settlement_status_snapshot"`
	}
	if err := config.DB.Table("appointment_settlements").
		Select("status, latest_reason, liability_level, merchant_breach_pending, settlement_status_snapshot").
		Where("id = ?", *updated.AppointmentSettlementID).
		Take(&settlement).Error; err != nil {
		t.Fatalf("load appointment settlement failed: %v", err)
	}
	if settlement.Status != "pending" || settlement.LatestReason != "merchant_delay_pending" {
		t.Fatalf("want settlement pending merchant_delay_pending, got %+v", settlement)
	}
	if !settlement.MerchantBreachPending || settlement.LiabilityLevel != "pending_merchant" {
		t.Fatalf("want pending merchant liability on settlement, got %+v", settlement)
	}
}

func containsText(body, want string) bool {
	return strings.Contains(body, want)
}
