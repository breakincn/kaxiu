package handlers

import (
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupServiceSessionUserRoomTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:handlers_service_session_user_room_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Merchant{},
		&models.ServiceRole{},
		&models.Technician{},
		&models.MerchantRoleStartPendingConfig{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestBuildServiceSessionRoomSelectionUpdatesKeepsAppointmentTechnicianAndMovesToStartPending(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupServiceSessionUserRoomTestDB(t)

	merchant := models.Merchant{
		Name:                       "m",
		Phone:                      "18800000002",
		Password:                   "pwd",
		SupportCustomerService:     true,
		SupportCustomerServiceMode: true,
		SupportRoom:                true,
	}
	if err := config.DB.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	role := models.ServiceRole{Name: "专业客服", Key: "professional", RoleType: "professional"}
	if err := config.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	tech := models.Technician{MerchantID: merchant.ID, Name: "大漂亮", Account: "js0001", ServiceRoleID: role.ID, IsActive: true}
	if err := config.DB.Create(&tech).Error; err != nil {
		t.Fatalf("create technician failed: %v", err)
	}

	session := models.ServiceSession{
		MerchantID:   merchant.ID,
		UserID:       7,
		CardID:       9,
		Status:       "cs_room_selecting",
		TechnicianID: &tech.ID,
	}
	lockedAt := time.Now()
	const roomID uint = 11

	updates := buildServiceSessionRoomSelectionUpdates(config.DB, merchant, session, roomID, lockedAt)

	status, ok := updates["status"].(string)
	if !ok {
		t.Fatalf("want string status, got %+v", updates["status"])
	}
	if got := models.NormalizeSessionStatus(status); got != "start_pending" {
		t.Fatalf("want start_pending, got %+v", updates["status"])
	}
	if got, ok := updates["room_id"].(uint); !ok || got != roomID {
		t.Fatalf("want room_id=%d, got %+v", roomID, updates["room_id"])
	}
	if got, ok := updates["start_pending_timeout_seconds"].(int); !ok || got <= 0 {
		t.Fatalf("want positive start_pending_timeout_seconds, got %+v", updates["start_pending_timeout_seconds"])
	}
	if got, ok := updates["staff_select_entered_at"]; !ok || got != nil {
		t.Fatalf("want staff_select_entered_at cleared, got %+v", updates["staff_select_entered_at"])
	}
	if got, ok := updates["staff_select_cooldown_until"]; !ok || got != nil {
		t.Fatalf("want staff_select_cooldown_until cleared, got %+v", updates["staff_select_cooldown_until"])
	}
}
