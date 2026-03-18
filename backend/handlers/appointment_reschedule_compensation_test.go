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
	if err := config.DB.AutoMigrate(&models.Card{}, &models.AppointmentCompensation{}, &models.AppointmentRescheduleRequest{}, &models.AppointmentProtectionBlock{}); err != nil {
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
		MerchantID:                    merchant.ID,
		AppointmentID:                 appt.ID,
		TechnicianID:                  tech.ID,
		BlockedReason:                 "为保护预约，拒绝将现场客户派给该客服",
		PredictedReservedWaitMinutes:  20,
		AlternativeWaitMinutes:        10,
		DecisionMode:                  "alternative_preferred",
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
