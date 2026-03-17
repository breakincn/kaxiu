package scheduler

import (
	"testing"
	"time"

	"kabao/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAppointmentSchedulerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:appointment_scheduler_test_" + time.Now().Format("20060102150405_000000000") + "?mode=memory&cache=shared&_loc=auto&parseTime=true"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.Appointment{}, &models.Technician{}, &models.MerchantProject{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestRunAppointmentNoShowOnceMarksConfirmedAppointmentAsNoShow(t *testing.T) {
	db := setupAppointmentSchedulerTestDB(t)

	now := time.Now()
	merchant := models.Merchant{
		Name:                            "appointment-no-show",
		Phone:                           "18800000201",
		Password:                        "pwd",
		SupportAppointment:              true,
		AppointmentGraceWindowMinutes:   15,
		AppointmentReserveBufferMinutes: 10,
	}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}

	appointmentTime := now.Add(-20 * time.Minute)
	appointment := models.Appointment{
		MerchantID:      merchant.ID,
		UserID:          1,
		CardID:          1,
		Status:          "confirmed",
		AppointmentTime: &appointmentTime,
	}
	if err := db.Create(&appointment).Error; err != nil {
		t.Fatalf("create appointment failed: %v", err)
	}

	if err := runAppointmentNoShowOnce(db, now); err != nil {
		t.Fatalf("runAppointmentNoShowOnce failed: %v", err)
	}

	var got struct {
		Status      string `gorm:"column:status"`
		NoShowAtRaw string `gorm:"column:no_show_at"`
	}
	if err := db.Table("appointments").Select("status, no_show_at").Where("id = ?", appointment.ID).Scan(&got).Error; err != nil {
		t.Fatalf("reload appointment failed: %v", err)
	}
	if got.Status != "no_show" {
		t.Fatalf("want no_show, got %s", got.Status)
	}
	if got.NoShowAtRaw == "" {
		t.Fatalf("want no_show_at filled")
	}
}
