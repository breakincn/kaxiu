package handlers

import (
	"bytes"
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

func setupAppointmentLifecycleTestDB(t *testing.T) {
	t.Helper()
	config.DB = setupAppointmentPermissionTestDB(t)
	if err := config.DB.AutoMigrate(&models.Card{}, &models.VerifyCode{}, &models.Usage{}, &models.ServiceSession{}, &models.TechnicianAttendance{}, &models.AppointmentCompensation{}, &models.AppointmentSettlement{}, &models.AppointmentRescheduleRequest{}, &models.AppointmentProtectionBlock{}, &models.ForceMajeureReliefRequest{}, &models.TechnicianMonthlyDisruptionCounter{}, &models.TechnicianDisruptionLedger{}, &models.AppointmentDelayLedger{}); err != nil {
		t.Fatalf("migrate extra tables failed: %v", err)
	}
}

func newMerchantJSONContext(method, path string, merchantID uint, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth_type", "merchant")
	c.Set("merchant_id", merchantID)
	return c, rec
}

func newUserJSONContext(method, path string, userID uint, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth_type", "user")
	c.Set("user_id", userID)
	return c, rec
}

func TestFilterAppointmentRescheduleSlotsByComparison(t *testing.T) {
	slots := []appointmentTimeSlot{
		{Time: "2026-03-25 10:00:00", PlacementScoreValue: 100},
		{Time: "2026-03-25 10:30:00", PlacementScoreValue: 120},
		{Time: "2026-03-25 11:00:00", PlacementScoreValue: 140},
	}

	stageA := filterAppointmentRescheduleSlotsByComparison(slots, appointmentRescheduleEligibility{
		Allowed:  true,
		RuleMode: appointmentRescheduleTodayOrTomorrow,
	}, 120)
	if len(stageA) != 2 {
		t.Fatalf("stage A should keep better and not_worse slots, got %d", len(stageA))
	}
	if stageA[0].ComparisonKind != "better" || stageA[0].ComparisonLabel != "更优于当前" {
		t.Fatalf("stage A first slot comparison mismatch: %+v", stageA[0])
	}
	if stageA[1].ComparisonKind != "not_worse" || stageA[1].ComparisonLabel != "不劣于当前" {
		t.Fatalf("stage A second slot comparison mismatch: %+v", stageA[1])
	}

	stageBC := filterAppointmentRescheduleSlotsByComparison(slots, appointmentRescheduleEligibility{
		Allowed:  true,
		RuleMode: appointmentRescheduleTomorrowOnly,
	}, 120)
	if len(stageBC) != 1 {
		t.Fatalf("stage B/C should keep only better slots, got %d", len(stageBC))
	}
	if stageBC[0].ComparisonKind != "better" || stageBC[0].Time != "2026-03-25 10:00:00" {
		t.Fatalf("stage B/C slot mismatch: %+v", stageBC[0])
	}
}

func TestMerchantRescheduleRequestNeedsUserAcceptance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	seedNextDayPublishedScheduleForMerchant(t, merchant, tech.ID)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "A001", CardType: "次卡", TotalTimes: 10, RemainTimes: 8, UsedTimes: 2}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	loc := appointmentLocation()
	base := appointmentCurrentTime().In(loc).Add(24 * time.Hour)
	oldTime := time.Date(base.Year(), base.Month(), base.Day(), 11, 0, 0, 0, loc)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &oldTime,
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

	newTime := oldTime.Add(2 * time.Hour).Format("2006-01-02 15:04:05")
	body, _ := json.Marshal(gin.H{
		"appointment_time": newTime,
		"technician_id":    tech.ID,
		"reason":           "商户建议改到下午",
	})
	c, rec := newMerchantJSONContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule", merchant.ID, body)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}

	RescheduleAppointment(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var pendingReq struct {
		ID     uint
		Status string
	}
	if err := config.DB.Table("appointment_reschedule_requests").
		Select("id, status").
		Where("appointment_id = ?", appt.ID).
		Take(&pendingReq).Error; err != nil {
		t.Fatalf("load pending request failed: %v", err)
	}
	if pendingReq.Status != "pending_user" {
		t.Fatalf("want pending_user request, got %+v", pendingReq)
	}

	uc, urec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-requests/"+strconv.Itoa(int(pendingReq.ID))+"/accept", user.ID, nil)
	uc.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(int(appt.ID))},
		{Key: "request_id", Value: strconv.Itoa(int(pendingReq.ID))},
	}
	AcceptAppointmentRescheduleRequest(uc)
	if urec.Code != http.StatusOK {
		t.Fatalf("want 200 when user accepts, got %d body=%s", urec.Code, urec.Body.String())
	}

	var oldGot struct {
		Status                  string
		ClosedReason            string
		ReplacedByAppointmentID *uint
	}
	if err := config.DB.Table("appointments").Select("status, closed_reason, replaced_by_appointment_id").Where("id = ?", appt.ID).Take(&oldGot).Error; err != nil {
		t.Fatalf("reload old appointment failed: %v", err)
	}
	if oldGot.Status != "canceled" || oldGot.ClosedReason != "rescheduled" || oldGot.ReplacedByAppointmentID == nil {
		t.Fatalf("want old appointment canceled by reschedule, got %+v", oldGot)
	}
	var oldSettlement models.AppointmentSettlement
	if err := config.DB.First(&oldSettlement, *appt.AppointmentSettlementID).Error; err != nil {
		t.Fatalf("load old settlement failed: %v", err)
	}
	if oldSettlement.Status != "transferred" || oldSettlement.TransferredToSettlementID == nil {
		t.Fatalf("want old settlement transferred, got %+v", oldSettlement)
	}
	var newAppointment models.Appointment
	newAppointmentPtr, err := loadAppointmentByID(config.DB, *oldGot.ReplacedByAppointmentID)
	if err != nil {
		t.Fatalf("load new appointment failed: %v", err)
	}
	if newAppointmentPtr == nil {
		t.Fatalf("new appointment %d not found", *oldGot.ReplacedByAppointmentID)
	}
	newAppointment = *newAppointmentPtr
	if newAppointment.AppointmentSettlementID == nil || *newAppointment.AppointmentSettlementID == 0 {
		t.Fatalf("want new appointment settlement created, got %+v", newAppointment.AppointmentSettlementID)
	}
	if newAppointment.SettlementStatusSnapshot != "pending" {
		t.Fatalf("want new appointment settlement snapshot pending, got %s", newAppointment.SettlementStatusSnapshot)
	}
}

func TestUserRescheduleRequestNeedsMerchantAcceptance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)
	loc := appointmentLocation()
	withAppointmentCurrentTime(t, time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), time.Now().In(loc).Day(), 10, 30, 0, 0, loc))

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	seedNextDayPublishedScheduleForMerchant(t, merchant, tech.ID)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "A003", CardType: "次卡", TotalTimes: 10, RemainTimes: 8, UsedTimes: 2}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	base := appointmentCurrentTime().In(loc).Add(24 * time.Hour)
	oldTime := time.Date(base.Year(), base.Month(), base.Day(), 11, 0, 0, 0, loc)
	appt := models.Appointment{MerchantID: merchant.ID, UserID: user.ID, CardID: card.ID, TechnicianID: &tech.ID, AppointmentTime: &oldTime, Status: "confirmed"}
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

	newTime := oldTime.Add(3 * time.Hour).Format("2006-01-02 15:04:05")
	body, _ := json.Marshal(gin.H{
		"appointment_time": newTime,
		"technician_id":    tech.ID,
		"reason":           "用户下午更方便",
	})
	uc, urec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-requests", user.ID, body)
	uc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CreateAppointmentRescheduleRequest(uc)
	if urec.Code != http.StatusOK {
		t.Fatalf("want 200 when user creates request, got %d body=%s", urec.Code, urec.Body.String())
	}

	var pendingReq struct {
		ID     uint
		Status string
	}
	if err := config.DB.Table("appointment_reschedule_requests").Select("id, status").Where("appointment_id = ?", appt.ID).Take(&pendingReq).Error; err != nil {
		t.Fatalf("load pending request failed: %v", err)
	}
	if pendingReq.Status != "pending_merchant" {
		t.Fatalf("want pending_merchant request, got %+v", pendingReq)
	}

	mc, mrec := newMerchantJSONContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-requests/"+strconv.Itoa(int(pendingReq.ID))+"/accept", merchant.ID, nil)
	mc.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(int(appt.ID))},
		{Key: "request_id", Value: strconv.Itoa(int(pendingReq.ID))},
	}
	AcceptAppointmentRescheduleRequest(mc)
	if mrec.Code != http.StatusOK {
		t.Fatalf("want 200 when merchant accepts, got %d body=%s", mrec.Code, mrec.Body.String())
	}

	var newGot struct {
		Status           string
		RescheduleReason string
	}
	if err := config.DB.Table("appointments").Select("status, reschedule_reason").Where("replaces_appointment_id = ?", appt.ID).Take(&newGot).Error; err != nil {
		t.Fatalf("load new appointment failed: %v", err)
	}
	if newGot.Status != "confirmed" {
		t.Fatalf("want new appointment confirmed, got %s", newGot.Status)
	}
	if newGot.RescheduleReason != "用户下午更方便" {
		t.Fatalf("want reschedule reason saved, got %s", newGot.RescheduleReason)
	}
	var oldSettlement models.AppointmentSettlement
	if err := config.DB.First(&oldSettlement, *appt.AppointmentSettlementID).Error; err != nil {
		t.Fatalf("load old settlement failed: %v", err)
	}
	if oldSettlement.Status != "transferred" || oldSettlement.TransferredToSettlementID == nil {
		t.Fatalf("want old settlement transferred, got %+v", oldSettlement)
	}
}

func TestChooseServiceSessionTechnicianKeepsMerchantBreachPendingForAppointmentWaiting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, otherTech, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	if err := config.DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Updates(map[string]interface{}{
		"support_customer_service_mode": true,
		"support_room":                  false,
	}).Error; err != nil {
		t.Fatalf("update merchant mode failed: %v", err)
	}
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "W001", CardType: "次卡", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	now := time.Now().Truncate(time.Second)
	appt := models.Appointment{
		MerchantID:            merchant.ID,
		UserID:                user.ID,
		CardID:                card.ID,
		TechnicianID:          &tech.ID,
		Status:                "arrived",
		MerchantBreachPending: true,
		LiabilityLevel:        "pending_merchant",
		ActualArrivedAt:       &now,
		ArrivedAt:             &now,
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	settlement := models.AppointmentSettlement{
		AppointmentID:            appt.ID,
		MerchantID:               merchant.ID,
		UserID:                   user.ID,
		CardID:                   card.ID,
		Status:                   "pending",
		SettlementStatusSnapshot: "pending",
		MerchantBreachPending:    true,
		LiabilityLevel:           "pending_merchant",
	}
	if err := config.DB.Create(&settlement).Error; err != nil {
		t.Fatalf("create settlement failed: %v", err)
	}
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appt.ID).Updates(map[string]interface{}{
		"appointment_settlement_id":  settlement.ID,
		"settlement_status_snapshot": "pending",
	}).Error; err != nil {
		t.Fatalf("bind settlement failed: %v", err)
	}
	session := models.ServiceSession{
		MerchantID: merchant.ID,
		UserID:     user.ID,
		CardID:     card.ID,
		SourceType: serviceSessionSourceAppointment,
		SourceID:   &appt.ID,
		Status:     "cs_staff_selecting",
	}
	if err := config.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	body, _ := json.Marshal(gin.H{"technician_id": otherTech.ID})
	c, rec := newMerchantJSONContext(http.MethodPost, "/merchant/service-sessions/"+strconv.Itoa(int(session.ID))+"/technician", merchant.ID, body)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(session.ID))}}

	ChooseServiceSessionTechnician(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var gotSession struct {
		Status                     string `gorm:"column:status"`
		TechnicianID               *uint  `gorm:"column:technician_id"`
		StartPendingTimeoutSeconds int    `gorm:"column:start_pending_timeout_seconds"`
	}
	if err := config.DB.Table("service_sessions").Select("status, technician_id, start_pending_timeout_seconds").Where("id = ?", session.ID).Take(&gotSession).Error; err != nil {
		t.Fatalf("load session failed: %v", err)
	}
	if gotSession.Status != "cs_start_pending" {
		t.Fatalf("want cs_start_pending, got %+v", gotSession)
	}
	if gotSession.TechnicianID == nil || *gotSession.TechnicianID != otherTech.ID {
		t.Fatalf("want technician=%d, got %+v", otherTech.ID, gotSession.TechnicianID)
	}
	if gotSession.StartPendingTimeoutSeconds <= 0 {
		t.Fatalf("want positive start pending timeout, got %d", gotSession.StartPendingTimeoutSeconds)
	}
	var gotAppointment struct {
		MerchantBreachPending bool   `gorm:"column:merchant_breach_pending"`
		LiabilityLevel        string `gorm:"column:liability_level"`
	}
	if err := config.DB.Table("appointments").Select("merchant_breach_pending, liability_level").Where("id = ?", appt.ID).Take(&gotAppointment).Error; err != nil {
		t.Fatalf("load appointment failed: %v", err)
	}
	if !gotAppointment.MerchantBreachPending || gotAppointment.LiabilityLevel != "pending_merchant" {
		t.Fatalf("want pending merchant breach retained, got %+v", gotAppointment)
	}
}

func TestCanceledRescheduleRequestCannotBeAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)
	withAppointmentCurrentTime(t, time.Date(time.Now().In(appointmentLocation()).Year(), time.Now().In(appointmentLocation()).Month(), time.Now().In(appointmentLocation()).Day(), 10, 30, 0, 0, appointmentLocation()))

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	seedNextDayPublishedScheduleForMerchant(t, merchant, tech.ID)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "A004", CardType: "次卡", TotalTimes: 10, RemainTimes: 8, UsedTimes: 2}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	oldTime := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	appt := models.Appointment{MerchantID: merchant.ID, UserID: user.ID, CardID: card.ID, TechnicianID: &tech.ID, AppointmentTime: &oldTime, Status: "confirmed"}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	newTime := oldTime.Add(90 * time.Minute).Format("2006-01-02 15:04:05")
	body, _ := json.Marshal(gin.H{
		"appointment_time": newTime,
		"technician_id":    tech.ID,
		"reason":           "用户想提前",
	})
	uc, urec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-requests", user.ID, body)
	uc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CreateAppointmentRescheduleRequest(uc)
	if urec.Code != http.StatusOK {
		t.Fatalf("want 200 when user creates request, got %d body=%s", urec.Code, urec.Body.String())
	}

	var pendingReq struct {
		ID     uint
		Status string
	}
	if err := config.DB.Table("appointment_reschedule_requests").Select("id, status").Where("appointment_id = ?", appt.ID).Take(&pendingReq).Error; err != nil {
		t.Fatalf("load pending request failed: %v", err)
	}

	cancelCtx, cancelRec := newUserJSONContext(http.MethodPost, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-requests/"+strconv.Itoa(int(pendingReq.ID))+"/cancel", user.ID, nil)
	cancelCtx.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(int(appt.ID))},
		{Key: "request_id", Value: strconv.Itoa(int(pendingReq.ID))},
	}
	CancelAppointmentRescheduleRequest(cancelCtx)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("want 200 when user cancels request, got %d body=%s", cancelRec.Code, cancelRec.Body.String())
	}

	acceptCtx, acceptRec := newMerchantJSONContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/reschedule-requests/"+strconv.Itoa(int(pendingReq.ID))+"/accept", merchant.ID, nil)
	acceptCtx.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(int(appt.ID))},
		{Key: "request_id", Value: strconv.Itoa(int(pendingReq.ID))},
	}
	AcceptAppointmentRescheduleRequest(acceptCtx)
	if acceptRec.Code == http.StatusOK {
		t.Fatalf("want non-200 when merchant accepts canceled request, got body=%s", acceptRec.Body.String())
	}

	var reqGot struct {
		Status string
	}
	if err := config.DB.Table("appointment_reschedule_requests").Select("status").Where("id = ?", pendingReq.ID).Take(&reqGot).Error; err != nil {
		t.Fatalf("reload request failed: %v", err)
	}
	if reqGot.Status != "canceled" {
		t.Fatalf("want request canceled, got %+v", reqGot)
	}

	var count int64
	if err := config.DB.Table("appointments").Where("replaces_appointment_id = ?", appt.ID).Count(&count).Error; err != nil {
		t.Fatalf("count replacement appointments failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("want no replacement appointment created after cancel, got %d", count)
	}
}

func TestCreateAppointmentCompensationAddsCardTimes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "A002", CardType: "次卡", TotalTimes: 10, RemainTimes: 8, UsedTimes: 2}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	apptTime := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &apptTime,
		Status:          "failed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	body, _ := json.Marshal(gin.H{
		"type":   "extra_times",
		"value":  2,
		"reason": "门店爽约补偿",
		"remark": "赠送2次",
	})
	c, rec := newMerchantJSONContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/compensations", merchant.ID, body)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}

	CreateAppointmentCompensation(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var gotCard models.Card
	if err := config.DB.First(&gotCard, card.ID).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotCard.TotalTimes != 12 || gotCard.RemainTimes != 10 {
		t.Fatalf("want total=12 remain=10, got total=%d remain=%d", gotCard.TotalTimes, gotCard.RemainTimes)
	}

	var comp struct {
		Status    string
		AppliedAt string
	}
	if err := config.DB.Table("appointment_compensations").
		Select("status, applied_at").
		Where("appointment_id = ?", appt.ID).
		Take(&comp).Error; err != nil {
		t.Fatalf("reload compensation failed: %v", err)
	}
	if comp.Status != "applied" {
		t.Fatalf("want applied compensation, got %s", comp.Status)
	}
	if comp.AppliedAt == "" {
		t.Fatalf("want applied_at filled, got empty")
	}
}

func TestGetAppointmentSettlementForMerchantAndUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "S001", CardType: "次卡", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	apptTime := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	appt := models.Appointment{MerchantID: merchant.ID, UserID: user.ID, CardID: card.ID, TechnicianID: &tech.ID, AppointmentTime: &apptTime, Status: "confirmed"}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("booking_root_id", appt.ID).Error; err != nil {
		t.Fatalf("set booking_root_id failed: %v", err)
	}
	appt.BookingRootID = &appt.ID
	settlement, err := initializeAppointmentSettlement(config.DB, &appt, "test_seed")
	if err != nil {
		t.Fatalf("initialize settlement failed: %v", err)
	}

	mc, mrec := newMerchantJSONContext(http.MethodGet, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/settlement", merchant.ID, nil)
	mc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	GetMerchantAppointmentSettlement(mc)
	if mrec.Code != http.StatusOK {
		t.Fatalf("want 200 from merchant settlement query, got %d body=%s", mrec.Code, mrec.Body.String())
	}

	uc, urec := newUserJSONContext(http.MethodGet, "/user/appointments/"+strconv.Itoa(int(appt.ID))+"/settlement", user.ID, nil)
	uc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	GetUserAppointmentSettlement(uc)
	if urec.Code != http.StatusOK {
		t.Fatalf("want 200 from user settlement query, got %d body=%s", urec.Code, urec.Body.String())
	}

	var merchantResp struct {
		Data models.AppointmentSettlement `json:"data"`
	}
	if err := json.Unmarshal(mrec.Body.Bytes(), &merchantResp); err != nil {
		t.Fatalf("decode merchant response failed: %v", err)
	}
	if merchantResp.Data.ID != settlement.ID || merchantResp.Data.AppointmentID != appt.ID {
		t.Fatalf("want merchant settlement %d, got %+v", settlement.ID, merchantResp.Data)
	}

	var userResp struct {
		Data models.AppointmentSettlement `json:"data"`
	}
	if err := json.Unmarshal(urec.Body.Bytes(), &userResp); err != nil {
		t.Fatalf("decode user response failed: %v", err)
	}
	if userResp.Data.ID != settlement.ID || userResp.Data.AppointmentID != appt.ID {
		t.Fatalf("want user settlement %d, got %+v", settlement.ID, userResp.Data)
	}
}

func TestDelayLedgerQueriesAndCompensationSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "D001", CardType: "次卡", TotalTimes: 10, RemainTimes: 9, UsedTimes: 1}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	apptTime := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	appt := models.Appointment{MerchantID: merchant.ID, UserID: user.ID, CardID: card.ID, TechnicianID: &tech.ID, AppointmentTime: &apptTime, Status: "completed"}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	ledger := models.AppointmentDelayLedger{
		AppointmentID:   appt.ID,
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card.ID,
		TechnicianID:    &tech.ID,
		DelayMinutes:    8,
		CreditedMinutes: 8,
		LedgerStatus:    "recorded",
		RedeemStatus:    "pending",
	}
	if err := config.DB.Create(&ledger).Error; err != nil {
		t.Fatalf("create ledger failed: %v", err)
	}
	comp := models.AppointmentCompensation{
		AppointmentID: appt.ID,
		MerchantID:    merchant.ID,
		UserID:        user.ID,
		CardID:        card.ID,
		Type:          "extra_times",
		Value:         1,
		Status:        "applied",
		Reason:        "delay_bucket_redeem",
		SourceType:    "delay_bucket_redeem",
	}
	if err := config.DB.Create(&comp).Error; err != nil {
		t.Fatalf("create compensation failed: %v", err)
	}

	mc, mrec := newMerchantJSONContext(http.MethodGet, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/delay-ledgers", merchant.ID, nil)
	mc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	ListMerchantAppointmentDelayLedgers(mc)
	if mrec.Code != http.StatusOK {
		t.Fatalf("want 200 from merchant delay ledgers, got %d body=%s", mrec.Code, mrec.Body.String())
	}

	uc, urec := newUserJSONContext(http.MethodGet, "/user/cards/"+strconv.Itoa(int(card.ID))+"/delay-ledgers", user.ID, nil)
	uc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(card.ID))}}
	ListUserCardDelayLedgers(uc)
	if urec.Code != http.StatusOK {
		t.Fatalf("want 200 from user delay ledgers, got %d body=%s", urec.Code, urec.Body.String())
	}

	sc, srec := newMerchantJSONContext(http.MethodGet, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/compensation-summary", merchant.ID, nil)
	sc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	GetMerchantAppointmentCompensationSummary(sc)
	if srec.Code != http.StatusOK {
		t.Fatalf("want 200 from compensation summary, got %d body=%s", srec.Code, srec.Body.String())
	}

	var summary struct {
		Data struct {
			TotalDelayMinutes    int                              `json:"total_delay_minutes"`
			TotalCreditedMinutes int                              `json:"total_credited_minutes"`
			DelayLedgers         []models.AppointmentDelayLedger  `json:"delay_ledgers"`
			Compensations        []models.AppointmentCompensation `json:"compensations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(srec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary failed: %v", err)
	}
	if summary.Data.TotalDelayMinutes != 8 || summary.Data.TotalCreditedMinutes != 8 {
		t.Fatalf("want summarized delay minutes, got %+v", summary.Data)
	}
	if len(summary.Data.DelayLedgers) != 1 || len(summary.Data.Compensations) != 1 {
		t.Fatalf("want ledger and compensation listed, got %+v", summary.Data)
	}
}

func TestForceMajeureReliefAcceptanceCancelsMerchantBreachCompensation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "F001", CardType: "次卡", TotalTimes: 12, RemainTimes: 10, UsedTimes: 2}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	apptTime := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &apptTime,
		Status:          "failed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	comp := models.AppointmentCompensation{
		AppointmentID: appt.ID,
		MerchantID:    merchant.ID,
		UserID:        user.ID,
		CardID:        card.ID,
		Type:          "extra_times",
		Value:         2,
		Status:        "applied",
		Reason:        "商户违约补偿",
		SourceType:    "merchant_breach",
	}
	if err := config.DB.Create(&comp).Error; err != nil {
		t.Fatalf("create compensation failed: %v", err)
	}

	body, _ := json.Marshal(gin.H{"reason": "停电停水", "evidence_note": "双方确认门店设施故障"})
	mc, mrec := newMerchantJSONContext(http.MethodPost, "/merchant/appointments/"+strconv.Itoa(int(appt.ID))+"/force-majeure-relief", merchant.ID, body)
	mc.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(appt.ID))}}
	CreateForceMajeureReliefRequest(mc)
	if mrec.Code != http.StatusOK {
		t.Fatalf("want 200 when create relief request, got %d body=%s", mrec.Code, mrec.Body.String())
	}

	var request models.ForceMajeureReliefRequest
	if err := config.DB.Where("appointment_id = ?", appt.ID).First(&request).Error; err != nil {
		t.Fatalf("load relief request failed: %v", err)
	}
	if request.Status != "pending" {
		t.Fatalf("want pending relief request, got %s", request.Status)
	}

	uc, urec := newUserJSONContext(http.MethodPost, "/user/appointments/force-majeure-relief/"+strconv.Itoa(int(request.ID))+"/accept", user.ID, nil)
	uc.Params = gin.Params{{Key: "request_id", Value: strconv.Itoa(int(request.ID))}}
	AcceptForceMajeureReliefRequest(uc)
	if urec.Code != http.StatusOK {
		t.Fatalf("want 200 when user accepts, got %d body=%s", urec.Code, urec.Body.String())
	}

	gotRequest, err := loadForceMajeureReliefRequestByID(config.DB, request.ID)
	if err != nil {
		t.Fatalf("reload relief request failed: %v", err)
	}
	if gotRequest == nil || gotRequest.Status != "accepted" {
		t.Fatalf("want accepted relief request, got %+v", gotRequest)
	}

	var gotComp struct {
		Status                      string
		ForceMajeureReliefRequestID *uint `gorm:"column:force_majeure_relief_request_id"`
	}
	if err := config.DB.Table("appointment_compensations").Select("status, force_majeure_relief_request_id").Where("id = ?", comp.ID).Take(&gotComp).Error; err != nil {
		t.Fatalf("reload compensation failed: %v", err)
	}
	if gotComp.Status != "canceled" {
		t.Fatalf("want compensation canceled, got %s", gotComp.Status)
	}
	if gotComp.ForceMajeureReliefRequestID == nil || *gotComp.ForceMajeureReliefRequestID != request.ID {
		t.Fatalf("want compensation linked to relief request, got %+v", gotComp.ForceMajeureReliefRequestID)
	}

	var gotCard models.Card
	if err := config.DB.First(&gotCard, card.ID).Error; err != nil {
		t.Fatalf("reload card failed: %v", err)
	}
	if gotCard.TotalTimes != 10 || gotCard.RemainTimes != 8 {
		t.Fatalf("want card compensation rolled back to total=10 remain=8, got total=%d remain=%d", gotCard.TotalTimes, gotCard.RemainTimes)
	}
}

func TestForceMajeureReliefRejectKeepsCompensationApplied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "F002", CardType: "次卡", TotalTimes: 11, RemainTimes: 9, UsedTimes: 2}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	apptTime := time.Now().Add(3 * time.Hour).Truncate(time.Second)
	appt := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          user.ID,
		CardID:          card.ID,
		TechnicianID:    &tech.ID,
		AppointmentTime: &apptTime,
		Status:          "failed",
	}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	comp := models.AppointmentCompensation{
		AppointmentID: appt.ID,
		MerchantID:    merchant.ID,
		UserID:        user.ID,
		CardID:        card.ID,
		Type:          "discount_note",
		Value:         0,
		Status:        "applied",
		Reason:        "商户违约补偿",
		SourceType:    "merchant_breach",
	}
	if err := config.DB.Create(&comp).Error; err != nil {
		t.Fatalf("create compensation failed: %v", err)
	}
	request := models.ForceMajeureReliefRequest{
		AppointmentID:  appt.ID,
		MerchantID:     merchant.ID,
		UserID:         user.ID,
		ProposedByType: "merchant",
		ProposedByID:   &merchant.ID,
		Reason:         "公共设施故障",
		Status:         "pending",
	}
	if err := config.DB.Create(&request).Error; err != nil {
		t.Fatalf("create relief request failed: %v", err)
	}

	uc, urec := newUserJSONContext(http.MethodPost, "/user/appointments/force-majeure-relief/"+strconv.Itoa(int(request.ID))+"/reject", user.ID, nil)
	uc.Params = gin.Params{{Key: "request_id", Value: strconv.Itoa(int(request.ID))}}
	RejectForceMajeureReliefRequest(uc)
	if urec.Code != http.StatusOK {
		t.Fatalf("want 200 when user rejects, got %d body=%s", urec.Code, urec.Body.String())
	}

	gotRequest, err := loadForceMajeureReliefRequestByID(config.DB, request.ID)
	if err != nil {
		t.Fatalf("reload request failed: %v", err)
	}
	if gotRequest == nil || gotRequest.Status != "rejected" {
		t.Fatalf("want rejected request, got %+v", gotRequest)
	}

	var gotComp struct {
		Status                      string
		ForceMajeureReliefRequestID *uint `gorm:"column:force_majeure_relief_request_id"`
	}
	if err := config.DB.Table("appointment_compensations").Select("status, force_majeure_relief_request_id").Where("id = ?", comp.ID).Take(&gotComp).Error; err != nil {
		t.Fatalf("reload compensation failed: %v", err)
	}
	if gotComp.Status != "applied" {
		t.Fatalf("want compensation kept applied, got %s", gotComp.Status)
	}
	if gotComp.ForceMajeureReliefRequestID != nil {
		t.Fatalf("want compensation not linked on reject, got %+v", gotComp.ForceMajeureReliefRequestID)
	}
}

func TestComputeAppointmentRescheduleEligibilityForTodayAppointment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "R001", CardType: "次卡", TotalTimes: 10, RemainTimes: 10}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	loc := appointmentLocation()
	now := time.Date(2026, 3, 18, 9, 0, 0, 0, loc)
	apptTime := time.Date(2026, 3, 18, 17, 0, 0, 0, loc)
	appt := models.Appointment{MerchantID: merchant.ID, UserID: user.ID, CardID: card.ID, TechnicianID: &tech.ID, AppointmentTime: &apptTime, Status: "confirmed"}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	eligibility, err := computeAppointmentRescheduleEligibility(config.DB, appt, merchant, now)
	if err != nil {
		t.Fatalf("compute eligibility failed: %v", err)
	}
	if !eligibility.Allowed || eligibility.RuleMode != appointmentRescheduleTomorrowOnly {
		t.Fatalf("want tomorrow_only allowed, got %+v", eligibility)
	}
	if len(eligibility.AllowedDates) != 1 || eligibility.AllowedDates[0] != "2026-03-19" {
		t.Fatalf("want only tomorrow allowed, got %+v", eligibility.AllowedDates)
	}
}

func TestComputeAppointmentRescheduleEligibilityForYesterdayAppointmentWindows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	merchant.AppointmentRescheduleSameOrNextDayThresholdMinutes = 180
	merchant.AppointmentRescheduleNextDayOnlyThresholdMinutes = 90
	if err := config.DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Updates(map[string]interface{}{
		"appointment_reschedule_same_or_next_day_threshold_minutes": 180,
		"appointment_reschedule_next_day_only_threshold_minutes":    90,
	}).Error; err != nil {
		t.Fatalf("update merchant thresholds failed: %v", err)
	}
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "R002", CardType: "次卡", TotalTimes: 10, RemainTimes: 10}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	loc := appointmentLocation()
	apptTime := time.Date(2026, 3, 17, 17, 0, 0, 0, loc)
	appt := models.Appointment{MerchantID: merchant.ID, UserID: user.ID, CardID: card.ID, TechnicianID: &tech.ID, AppointmentTime: &apptTime, Status: "confirmed"}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	cases := []struct {
		name        string
		now         time.Time
		wantAllowed bool
		wantMode    appointmentRescheduleRuleMode
		wantDates   []string
	}{
		{
			name:        "more than same or next threshold",
			now:         time.Date(2026, 3, 18, 9, 0, 0, 0, loc),
			wantAllowed: true,
			wantMode:    appointmentRescheduleTodayOrTomorrow,
			wantDates:   []string{"2026-03-18", "2026-03-19"},
		},
		{
			name:        "between thresholds",
			now:         time.Date(2026, 3, 18, 15, 0, 0, 0, loc),
			wantAllowed: true,
			wantMode:    appointmentRescheduleTomorrowOnly,
			wantDates:   []string{"2026-03-19"},
		},
		{
			name:        "inside forbidden threshold",
			now:         time.Date(2026, 3, 18, 15, 31, 0, 0, loc),
			wantAllowed: false,
			wantMode:    appointmentRescheduleForbidden,
			wantDates:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eligibility, err := computeAppointmentRescheduleEligibility(config.DB, appt, merchant, tc.now)
			if err != nil {
				t.Fatalf("compute eligibility failed: %v", err)
			}
			if eligibility.Allowed != tc.wantAllowed || eligibility.RuleMode != tc.wantMode {
				t.Fatalf("want allowed=%v mode=%s, got %+v", tc.wantAllowed, tc.wantMode, eligibility)
			}
			if len(eligibility.AllowedDates) != len(tc.wantDates) {
				t.Fatalf("want dates=%v, got=%v", tc.wantDates, eligibility.AllowedDates)
			}
			for i := range tc.wantDates {
				if eligibility.AllowedDates[i] != tc.wantDates[i] {
					t.Fatalf("want dates=%v, got=%v", tc.wantDates, eligibility.AllowedDates)
				}
			}
		})
	}
}

func TestComputeAppointmentRescheduleEligibilityBlockedAfterProtectionConsumed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "R003", CardType: "次卡", TotalTimes: 10, RemainTimes: 10}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}
	loc := appointmentLocation()
	apptTime := time.Date(2026, 3, 17, 17, 0, 0, 0, loc)
	appt := models.Appointment{MerchantID: merchant.ID, UserID: user.ID, CardID: card.ID, TechnicianID: &tech.ID, AppointmentTime: &apptTime, Status: "confirmed"}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}
	block := models.AppointmentProtectionBlock{
		MerchantID:                   merchant.ID,
		AppointmentID:                appt.ID,
		TechnicianID:                 tech.ID,
		BlockedReason:                "为保护预约，拒绝将现场客户派给该客服",
		PredictedReservedWaitMinutes: 20,
		AlternativeWaitMinutes:       10,
		DecisionMode:                 "strict_reservation_lock",
	}
	blockedAt := time.Date(2026, 3, 18, 10, 0, 0, 0, loc)
	block.BlockedAt = &blockedAt
	if err := config.DB.Create(&block).Error; err != nil {
		t.Fatalf("create protection block failed: %v", err)
	}

	eligibility, err := computeAppointmentRescheduleEligibility(config.DB, appt, merchant, time.Date(2026, 3, 18, 11, 0, 0, 0, loc))
	if err != nil {
		t.Fatalf("compute eligibility failed: %v", err)
	}
	if eligibility.Allowed {
		t.Fatalf("want forbidden after protection consumed, got %+v", eligibility)
	}
	if !eligibility.ProtectionConsumed {
		t.Fatalf("want protection_consumed=true, got %+v", eligibility)
	}
}
