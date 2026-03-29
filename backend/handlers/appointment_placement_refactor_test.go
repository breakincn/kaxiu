package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
)

func appointmentFixtureTime(hour, minute int) time.Time {
	loc := appointmentLocation()
	base := time.Now().In(loc).Add(24 * time.Hour)
	return time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, loc)
}

func seedAppointmentPlacementFixture(t *testing.T, customerServiceMode bool) (models.Merchant, models.User, models.Technician, models.MerchantProject, models.Card, models.Card) {
	t.Helper()
	merchant, user, tech, otherTech, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	merchant.SupportCustomerServiceMode = customerServiceMode
	merchant.AppointmentPredictionBufferMinute = 10
	if err := config.DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Updates(map[string]interface{}{
		"support_customer_service_mode":         customerServiceMode,
		"appointment_prediction_buffer_minutes": 10,
	}).Error; err != nil {
		t.Fatalf("update merchant failed: %v", err)
	}
	if err := config.DB.Model(&models.Technician{}).Where("id = ?", otherTech.ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("disable secondary technician failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:        merchant.ID,
		Name:              "塑形私教课",
		Duration:          45,
		BookableOnline:    true,
		ServiceGapMinutes: 3,
		StartDelaySeconds: 60,
		IsActive:          true,
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	card1 := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "C1001", CardType: "次卡", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	card2 := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "C1002", CardType: "次卡", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := config.DB.Create(&card1).Error; err != nil {
		t.Fatalf("create card1 failed: %v", err)
	}
	if err := config.DB.Create(&card2).Error; err != nil {
		t.Fatalf("create card2 failed: %v", err)
	}
	return merchant, user, tech, project, card1, card2
}

func mustLoadAppointmentForTest(t *testing.T, appointmentID uint) models.Appointment {
	t.Helper()
	appt, err := loadAppointmentByID(config.DB, appointmentID)
	if err != nil {
		t.Fatalf("load appointment failed: %v", err)
	}
	if appt == nil {
		t.Fatalf("appointment %d not found", appointmentID)
	}
	return *appt
}

func TestCreateAppointmentAutoAssignsTechnicianAndRejectsOverlap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)
	withAppointmentCurrentTime(t, time.Date(time.Now().In(appointmentLocation()).Year(), time.Now().In(appointmentLocation()).Month(), time.Now().In(appointmentLocation()).Day(), 10, 30, 0, 0, appointmentLocation()))

	merchant, user, tech, project, card1, card2 := seedAppointmentPlacementFixture(t, true)
	seedNextDayPublishedScheduleForMerchant(t, merchant, tech.ID)
	slot := appointmentFixtureTime(10, 0)

	body, _ := json.Marshal(gin.H{
		"card_id":          card1.ID,
		"merchant_id":      merchant.ID,
		"user_id":          user.ID,
		"project_id":       project.ID,
		"appointment_time": slot.Format("2006-01-02 15:04:05"),
		"technician_id":    nil,
	})
	c, rec := newUserJSONContext(http.MethodPost, "/user/appointments", user.ID, body)
	CreateAppointment(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var appointmentID uint
	if err := config.DB.Table("appointments").Select("id").Order("id desc").Limit(1).Scan(&appointmentID).Error; err != nil {
		t.Fatalf("load created appointment id failed: %v", err)
	}
	created := mustLoadAppointmentForTest(t, appointmentID)
	if created.TechnicianID == nil || *created.TechnicianID != tech.ID {
		t.Fatalf("want technician auto assigned to %d, got %+v", tech.ID, created.TechnicianID)
	}
	if created.AppointmentSettlementID == nil || *created.AppointmentSettlementID == 0 {
		t.Fatalf("want appointment settlement created, got %+v", created.AppointmentSettlementID)
	}
	if created.SettlementStatusSnapshot != "pending" {
		t.Fatalf("want settlement_status_snapshot pending, got %s", created.SettlementStatusSnapshot)
	}
	var settlement models.AppointmentSettlement
	if err := config.DB.First(&settlement, *created.AppointmentSettlementID).Error; err != nil {
		t.Fatalf("load appointment settlement failed: %v", err)
	}
	if settlement.AppointmentID != created.ID {
		t.Fatalf("want settlement appointment_id %d, got %d", created.ID, settlement.AppointmentID)
	}
	if settlement.Status != "pending" {
		t.Fatalf("want settlement status pending, got %s", settlement.Status)
	}

	body, _ = json.Marshal(gin.H{
		"card_id":          card2.ID,
		"merchant_id":      merchant.ID,
		"user_id":          user.ID,
		"project_id":       project.ID,
		"appointment_time": slot.Format("2006-01-02 15:04:05"),
	})
	c, rec = newUserJSONContext(http.MethodPost, "/user/appointments", user.ID, body)
	CreateAppointment(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for overlapping auto-assigned appointment, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestEvaluateBookingTechnicianAvailabilityUsesStrictHardNoOverlap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	start := appointmentFixtureTime(10, 0)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &start,
		Status:          "pending",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	availability, err := evaluateBookingTechnicianAvailability(config.DB, merchant, tech.ID, start.Add(30*time.Minute), projectBookingOccupiedMinutes(project.Duration, project.ServiceGapMinutes), 0)
	if err != nil {
		t.Fatalf("evaluate availability failed: %v", err)
	}
	if availability.State != appointmentAvailabilityUnavailable {
		t.Fatalf("want unavailable, got %+v", availability)
	}
}

func TestEvaluateWalkInTechnicianAvailabilityRejectsReservationConflictImmediately(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	appointmentTime := appointmentFixtureTime(11, 0)
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

	walkInStart := appointmentTime.Add(-30 * time.Minute)
	availability, err := evaluateWalkInTechnicianAvailability(config.DB, merchant, tech.ID, walkInStart, projectBookingOccupiedMinutes(project.Duration, project.ServiceGapMinutes))
	if err != nil {
		t.Fatalf("evaluate walk-in availability failed: %v", err)
	}
	if availability.State != appointmentAvailabilityUnavailable {
		t.Fatalf("want unavailable, got %+v", availability)
	}
	if availability.DecisionMode != "strict_reservation_lock" {
		t.Fatalf("want strict_reservation_lock, got %+v", availability)
	}
	if availability.NextAppointmentID == nil || *availability.NextAppointmentID != appt.ID {
		t.Fatalf("want next appointment %d, got %+v", appt.ID, availability.NextAppointmentID)
	}
	if availability.PredictedWaitMinutes <= 0 {
		t.Fatalf("want positive reserved delay, got %+v", availability)
	}
}

func TestAvailableSlotsAndCreateRejectNonBookableProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)
	withAppointmentCurrentTime(t, time.Date(time.Now().In(appointmentLocation()).Year(), time.Now().In(appointmentLocation()).Month(), time.Now().In(appointmentLocation()).Day(), 10, 30, 0, 0, appointmentLocation()))

	merchant, user, _, _, card1, _ := seedAppointmentPlacementFixture(t, false)
	seedNextDayPublishedScheduleForMerchant(t, merchant)
	project := models.MerchantProject{
		MerchantID:        merchant.ID,
		Name:              "线下附加项",
		Duration:          30,
		BookableOnline:    false,
		ServiceGapMinutes: 3,
		StartDelaySeconds: 60,
		IsActive:          true,
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create offline project failed: %v", err)
	}
	if err := config.DB.Model(&models.MerchantProject{}).Where("id = ?", project.ID).Update("bookable_online", false).Error; err != nil {
		t.Fatalf("update offline project failed: %v", err)
	}
	date := appointmentFixtureTime(10, 0).Format("2006-01-02")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/merchants/"+strconv.Itoa(int(merchant.ID))+"/available-slots?date="+date+"&project_id="+strconv.Itoa(int(project.ID)), nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(merchant.ID))}}
	c.Set("auth_type", "user")
	c.Set("user_id", user.ID)
	GetAvailableTimeSlots(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for non-bookable project slots, got %d body=%s", rec.Code, rec.Body.String())
	}

	body, _ := json.Marshal(gin.H{
		"card_id":          card1.ID,
		"merchant_id":      merchant.ID,
		"user_id":          user.ID,
		"project_id":       project.ID,
		"appointment_time": appointmentFixtureTime(11, 0).Format("2006-01-02 15:04:05"),
	})
	userCtx, userRec := newUserJSONContext(http.MethodPost, "/user/appointments", user.ID, body)
	CreateAppointment(userCtx)
	if userRec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for non-bookable create, got %d body=%s", userRec.Code, userRec.Body.String())
	}
}

func TestAvailableSlotsOrderedChronologicallyForUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)
	withAppointmentCurrentTime(t, time.Date(time.Now().In(appointmentLocation()).Year(), time.Now().In(appointmentLocation()).Month(), time.Now().In(appointmentLocation()).Day(), 10, 30, 0, 0, appointmentLocation()))

	merchant, user, _, _, card1, _ := seedAppointmentPlacementFixture(t, false)
	if err := config.DB.Model(&models.MerchantProject{}).Where("merchant_id = ?", merchant.ID).Update("bookable_online", false).Error; err != nil {
		t.Fatalf("disable default bookable projects failed: %v", err)
	}
	merchant.AllDayStart = "10:00"
	merchant.AllDayEnd = "12:00"
	if err := config.DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Updates(map[string]interface{}{
		"all_day_start": "10:00",
		"all_day_end":   "12:00",
	}).Error; err != nil {
		t.Fatalf("update merchant hours failed: %v", err)
	}
	project := models.MerchantProject{
		MerchantID:        merchant.ID,
		Name:              "半小时项目",
		Duration:          30,
		BookableOnline:    true,
		ServiceGapMinutes: 0,
		StartDelaySeconds: 60,
		IsActive:          true,
	}
	if err := config.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if err := config.DB.Model(&models.MerchantProject{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
		"duration":            30,
		"service_gap_minutes": 0,
		"bookable_online":     true,
	}).Error; err != nil {
		t.Fatalf("normalize project booking config failed: %v", err)
	}
	seedNextDayPublishedScheduleForMerchant(t, merchant)
	blockStart := appointmentFixtureTime(11, 0)
	blocker := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		AppointmentTime: &blockStart,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&blocker).Error; err != nil {
		t.Fatalf("create blocker appointment failed: %v", err)
	}

	date := appointmentFixtureTime(10, 0).Format("2006-01-02")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/merchants/"+strconv.Itoa(int(merchant.ID))+"/available-slots?date="+date+"&project_id="+strconv.Itoa(int(project.ID)), nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(merchant.ID))}}
	c.Set("auth_type", "user")
	c.Set("user_id", user.ID)
	GetAvailableTimeSlots(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			TimeSlots []struct {
				Time string `json:"time"`
			} `json:"time_slots"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v body=%s", err, rec.Body.String())
	}
	if len(resp.Data.TimeSlots) < 3 {
		t.Fatalf("want at least 3 ranked slots, got body=%s", rec.Body.String())
	}
	if got := resp.Data.TimeSlots[0].Time; got != appointmentFixtureTime(10, 0).Format("2006-01-02 15:04:05") {
		t.Fatalf("want earliest slot 10:00 first, got %s body=%s", got, rec.Body.String())
	}
	found1130 := false
	for _, slot := range resp.Data.TimeSlots {
		if slot.Time == appointmentFixtureTime(11, 30).Format("2006-01-02 15:04:05") {
			found1130 = true
			break
		}
	}
	if !found1130 {
		t.Fatalf("want 11:30 slot still available, got body=%s", rec.Body.String())
	}
}

func TestGetAvailableTimeSlotsFiltersUnpublishedTechnicians(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)
	withAppointmentCurrentTime(t, time.Date(time.Now().In(appointmentLocation()).Year(), time.Now().In(appointmentLocation()).Month(), time.Now().In(appointmentLocation()).Day(), 10, 30, 0, 0, appointmentLocation()))

	merchant, user, tech, project, _, _ := seedAppointmentPlacementFixture(t, true)
	unpublishedTech := models.Technician{
		MerchantID:    merchant.ID,
		ServiceRoleID: tech.ServiceRoleID,
		Name:          "小美",
		Code:          "X1",
		Account:       "js0099",
		Password:      "x",
		IsActive:      true,
	}
	if err := config.DB.Create(&unpublishedTech).Error; err != nil {
		t.Fatalf("create unpublished technician failed: %v", err)
	}
	seedNextDayPublishedScheduleForMerchant(t, merchant, tech.ID)

	date := appointmentFixtureTime(10, 0).Format("2006-01-02")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/merchant/merchants/"+strconv.Itoa(int(merchant.ID))+"/available-slots?date="+date+"&project_id="+strconv.Itoa(int(project.ID)), nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(merchant.ID))}}
	c.Set("auth_type", "user")
	c.Set("user_id", user.ID)
	GetAvailableTimeSlots(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Technicians []struct {
				ID uint `json:"id"`
			} `json:"technicians"`
			TimeSlots []struct {
				TechnicianIDs []uint `json:"technician_ids"`
			} `json:"time_slots"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v body=%s", err, rec.Body.String())
	}
	if len(resp.Data.Technicians) != 1 || resp.Data.Technicians[0].ID != tech.ID {
		t.Fatalf("want only published technician %d, got %+v", tech.ID, resp.Data.Technicians)
	}
	for _, slot := range resp.Data.TimeSlots {
		for _, technicianID := range slot.TechnicianIDs {
			if technicianID == unpublishedTech.ID {
				t.Fatalf("unpublished technician %d should not appear in slot candidates: %+v", unpublishedTech.ID, resp.Data.TimeSlots)
			}
		}
	}
}

func TestConfirmAppointmentAssignsTechnicianForLegacyPendingRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	seedNextDayPublishedScheduleForMerchant(t, merchant, tech.ID)
	start := appointmentFixtureTime(13, 0)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		AppointmentTime: &start,
		Status:          "pending",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create legacy pending appointment failed: %v", err)
	}

	c, rec := newMerchantJSONContext(http.MethodPut, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/confirm", merchant.ID, nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	ConfirmAppointment(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 on confirm, got %d body=%s", rec.Code, rec.Body.String())
	}

	got := mustLoadAppointmentForTest(t, appt.ID)
	if got.TechnicianID == nil || *got.TechnicianID != tech.ID {
		t.Fatalf("want technician %d assigned on confirm, got %+v", tech.ID, got.TechnicianID)
	}
	if got.Status != "confirmed" {
		t.Fatalf("want confirmed, got %s", got.Status)
	}
}

func TestCreateRescheduleRequestPersistsAssignedTechnicianWhenUnspecified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)
	withAppointmentCurrentTime(t, time.Date(time.Now().In(appointmentLocation()).Year(), time.Now().In(appointmentLocation()).Month(), time.Now().In(appointmentLocation()).Day(), 10, 30, 0, 0, appointmentLocation()))

	merchant, user, tech, project, card1, _ := seedAppointmentPlacementFixture(t, true)
	seedNextDayPublishedScheduleForMerchant(t, merchant, tech.ID)
	oldTime := appointmentFixtureTime(10, 0)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card1.ID,
		ProjectID:       &project.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &oldTime,
		Status:          "confirmed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	body, _ := json.Marshal(gin.H{
		"appointment_time": appointmentFixtureTime(15, 0).Format("2006-01-02 15:04:05"),
		"reason":           "改到下午",
	})
	c, rec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-requests", user.ID, body)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CreateAppointmentRescheduleRequest(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var proposalID uint
	if err := config.DB.Table("appointment_reschedule_requests").Select("id").Order("id desc").Limit(1).Scan(&proposalID).Error; err != nil {
		t.Fatalf("load proposal id failed: %v", err)
	}
	proposal, err := loadAppointmentRescheduleRequestByID(config.DB, proposalID)
	if err != nil {
		t.Fatalf("load proposal failed: %v", err)
	}
	if proposal == nil {
		t.Fatalf("proposal %d not found", proposalID)
	}
	if proposal.NewTechnicianID == nil || *proposal.NewTechnicianID != tech.ID {
		t.Fatalf("want proposal technician %d, got %+v", tech.ID, proposal.NewTechnicianID)
	}
}
