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
	if err := config.DB.AutoMigrate(&models.Card{}, &models.AppointmentCompensation{}, &models.AppointmentRescheduleRequest{}); err != nil {
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

func TestMerchantRescheduleRequestNeedsUserAcceptance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "A001", CardType: "次卡", TotalTimes: 10, RemainTimes: 8, UsedTimes: 2}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	oldTime := time.Now().Add(24 * time.Hour).Truncate(time.Second)
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
}

func TestUserRescheduleRequestNeedsMerchantAcceptance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	setupAppointmentLifecycleTestDB(t)

	merchant, user, tech, _, _, _ := seedAppointmentPermissionFixture(t, config.DB)
	card := models.Card{MerchantID: merchant.ID, UserID: user.ID, CardNo: "A003", CardType: "次卡", TotalTimes: 10, RemainTimes: 8, UsedTimes: 2}
	if err := config.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card failed: %v", err)
	}

	oldTime := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	appt := models.Appointment{MerchantID: merchant.ID, UserID: user.ID, CardID: card.ID, TechnicianID: &tech.ID, AppointmentTime: &oldTime, Status: "confirmed"}
	if err := config.DB.Create(&appt).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
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
