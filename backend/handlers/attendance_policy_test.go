package handlers

import (
	"strings"
	"testing"
	"time"

	"kabao/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAttendancePolicyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.ServiceRole{},
		&models.MerchantRoleAttendanceConfig{},
		&models.Technician{},
		&models.TechnicianAttendance{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestIsRoleAttendanceRequiredFallsBackToRoleDefault(t *testing.T) {
	db := setupAttendancePolicyTestDB(t)

	role := models.ServiceRole{Name: "助教", Key: "teacher"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	if err := db.Model(&models.ServiceRole{}).Where("id = ?", role.ID).Update("require_attendance", false).Error; err != nil {
		t.Fatalf("update role require_attendance failed: %v", err)
	}

	required, err := isRoleAttendanceRequired(db, 2, role.ID)
	if err != nil {
		t.Fatalf("isRoleAttendanceRequired failed: %v", err)
	}
	if required {
		t.Fatalf("want fallback role default false, got true")
	}
}

func TestEnsureAttendanceForNoCheckinRoleCreatesIdleAttendanceWhenMissing(t *testing.T) {
	db := setupAttendancePolicyTestDB(t)

	now := time.Now()
	if err := ensureAttendanceForNoCheckinRole(db, 2, 3, now); err != nil {
		t.Fatalf("ensureAttendanceForNoCheckinRole failed: %v", err)
	}

	var got struct {
		Status       string     `gorm:"column:status"`
		CheckedOutAt *time.Time `gorm:"column:checked_out_at"`
	}
	if err := db.Table("technician_attendances").Select("status, checked_out_at").Where("merchant_id = ? AND technician_id = ?", 2, 3).First(&got).Error; err != nil {
		t.Fatalf("load attendance failed: %v", err)
	}
	if got.Status != "idle" {
		t.Fatalf("want idle attendance, got %s", got.Status)
	}
	if got.CheckedOutAt != nil {
		t.Fatalf("want open attendance, got checked_out_at=%v", got.CheckedOutAt)
	}
}
