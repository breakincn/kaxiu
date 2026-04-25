package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMultiServiceBookingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Merchant{},
		&models.Card{},
		&models.MerchantProject{},
		&models.CardProject{},
		&models.MultiServiceBooking{},
		&models.MultiServicePenaltyLedger{},
		&models.VerifyCode{},
		&models.Usage{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func seedMultiServiceBookingFixture(t *testing.T, db *gorm.DB, serviceCapacity int, startOffset time.Duration) (models.Merchant, models.MerchantProject, []models.User, time.Time) {
	t.Helper()
	merchant := models.Merchant{Name: "multi-service-merchant", Phone: fmt.Sprintf("177%08d", time.Now().UnixNano()%100000000), Password: "pwd", IsOpen: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	slotStart := time.Now().In(config.ProjectServiceTimeLocation()).Truncate(time.Minute).Add(startOffset)
	weekday := int(slotStart.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	project := models.MerchantProject{
		MerchantID:       merchant.ID,
		Name:             "舞蹈团课",
		Duration:         60,
		ServiceCapacity:  serviceCapacity,
		Price:            88,
		IsActive:         true,
		ServiceTimeSlots: models.MerchantProjectServiceTimeSlots{{RecurrenceType: "weekly", Weekday: weekday, StartTime: slotStart.Format("15:04")}},
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	users := make([]models.User, 0, serviceCapacity+1)
	for i := 0; i < serviceCapacity+1; i++ {
		user := models.User{Username: fmt.Sprintf("multi_service_user_%d", i+1)}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		card := models.Card{
			UserID:      user.ID,
			MerchantID:  merchant.ID,
			CardNo:      fmt.Sprintf("MS%03d", i+1),
			CardType:    "舞蹈课",
			TotalTimes:  10,
			RemainTimes: 10,
		}
		if err := db.Create(&card).Error; err != nil {
			t.Fatalf("create card failed: %v", err)
		}
		if err := db.Create(&models.CardProject{CardID: card.ID, ProjectID: project.ID}).Error; err != nil {
			t.Fatalf("create card project failed: %v", err)
		}
		users = append(users, user)
	}

	return merchant, project, users, slotStart
}

func newMultiServiceBookingContext(method, path string, userID uint, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", userID)
	return c, rec
}

func TestCreateCardMultiServiceBookingRejectsWhenSlotFull(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupMultiServiceBookingTestDB(t)

	_, project, users, slotStart := seedMultiServiceBookingFixture(t, config.DB, 2, 2*time.Hour)

	for i := 0; i < 2; i++ {
		var card models.Card
		if err := config.DB.Where("user_id = ?", users[i].ID).First(&card).Error; err != nil {
			t.Fatalf("load card failed: %v", err)
		}
		body, _ := json.Marshal(gin.H{"project_id": project.ID, "slot_start_at": slotStart.Format("2006-01-02 15:04:05")})
		c, rec := newMultiServiceBookingContext(http.MethodPost, fmt.Sprintf("/user/cards/%d/multi-service-bookings", card.ID), users[i].ID, body)
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", card.ID)}}
		CreateCardMultiServiceBooking(c)
		if rec.Code != http.StatusOK {
			t.Fatalf("prefill want 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	}

	var extraCard models.Card
	if err := config.DB.Where("user_id = ?", users[2].ID).First(&extraCard).Error; err != nil {
		t.Fatalf("load extra card failed: %v", err)
	}
	body, _ := json.Marshal(gin.H{"project_id": project.ID, "slot_start_at": slotStart.Format("2006-01-02 15:04:05")})
	c, rec := newMultiServiceBookingContext(http.MethodPost, fmt.Sprintf("/user/cards/%d/multi-service-bookings", extraCard.ID), users[2].ID, body)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", extraCard.ID)}}
	CreateCardMultiServiceBooking(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "当前场次预约已满") {
		t.Fatalf("unexpected body=%s", rec.Body.String())
	}
}

func TestValidateMultiServiceVerifyEligibilityRejectsUnbookedUserWhenSlotFull(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupMultiServiceBookingTestDB(t)

	merchant, project, users, slotStart := seedMultiServiceBookingFixture(t, db, 2, 30*time.Minute)
	for i := 0; i < 2; i++ {
		var card models.Card
		if err := db.Where("user_id = ?", users[i].ID).First(&card).Error; err != nil {
			t.Fatalf("load card failed: %v", err)
		}
		booking := models.MultiServiceBooking{
			MerchantID:  merchant.ID,
			ProjectID:   project.ID,
			CardID:      card.ID,
			UserID:      users[i].ID,
			SlotStartAt: &slotStart,
			SlotEndAt:   ptrHandlerTime(slotStart.Add(57 * time.Minute)),
			Status:      multiServiceBookingStatusBooked,
			BookedAt:    ptrHandlerTime(time.Now()),
		}
		if err := db.Create(&booking).Error; err != nil {
			t.Fatalf("create booking failed: %v", err)
		}
	}

	var extraCard models.Card
	if err := db.Where("user_id = ?", users[2].ID).First(&extraCard).Error; err != nil {
		t.Fatalf("load extra card failed: %v", err)
	}
	_, _, err := validateMultiServiceVerifyEligibility(db, extraCard, project, slotStart.Add(-30*time.Minute))
	if err == nil || !strings.Contains(err.Error(), "当前课程预约已满") {
		t.Fatalf("unexpected err=%v", err)
	}
}

func ptrHandlerTime(v time.Time) *time.Time { return &v }
