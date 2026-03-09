package config

import (
	"strings"
	"testing"

	"kabao/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupServiceRoleCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.ServiceRole{},
		&models.Technician{},
		&models.TechnicianAttendance{},
		&models.RolePermission{},
		&models.MerchantRolePermissionOverride{},
		&models.MerchantRoleAttendanceConfig{},
		&models.MerchantRoleStartPendingConfig{},
	); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestCleanupServiceRolesNormalizesMerchantKeysAndPurgesLegacyPlatformRoles(t *testing.T) {
	oldDB := DB
	defer func() { DB = oldDB }()
	DB = setupServiceRoleCleanupTestDB(t)

	merchantID := uint(2)
	merchantRole := models.ServiceRole{MerchantID: &merchantID, Key: "m2_zj_1768364633", Name: "助教", AccountPrefix: "zj", RoleType: "professional", IsActive: true}
	legacyPlatformRole := models.ServiceRole{Key: "teacher", Name: "助教", AccountPrefix: "zj", RoleType: "professional", IsActive: true}
	fixedRole := models.ServiceRole{Key: "store_manager", Name: "店长", AccountPrefix: "sm", RoleType: "operational", IsActive: true}
	if err := DB.Create(&merchantRole).Error; err != nil {
		t.Fatalf("create merchant role failed: %v", err)
	}
	if err := DB.Create(&legacyPlatformRole).Error; err != nil {
		t.Fatalf("create legacy role failed: %v", err)
	}
	if err := DB.Create(&fixedRole).Error; err != nil {
		t.Fatalf("create fixed role failed: %v", err)
	}
	if err := DB.Create(&models.Technician{MerchantID: merchantID, ServiceRoleID: legacyPlatformRole.ID, Name: "legacy", Code: "0001", Account: "zj0001", Password: "pwd", IsActive: true}).Error; err != nil {
		t.Fatalf("create technician failed: %v", err)
	}

	cleanupServiceRoles()

	var normalized models.ServiceRole
	if err := DB.First(&normalized, merchantRole.ID).Error; err != nil {
		t.Fatalf("load normalized role failed: %v", err)
	}
	if normalized.Key != "m2_zj" {
		t.Fatalf("want normalized key m2_zj, got %s", normalized.Key)
	}

	if err := DB.First(&models.ServiceRole{}, legacyPlatformRole.ID).Error; err == nil {
		t.Fatalf("legacy platform role should be removed")
	}
	if err := DB.First(&models.ServiceRole{}, fixedRole.ID).Error; err != nil {
		t.Fatalf("fixed role should remain: %v", err)
	}

	var techCount int64
	if err := DB.Model(&models.Technician{}).Where("service_role_id = ?", legacyPlatformRole.ID).Count(&techCount).Error; err != nil {
		t.Fatalf("count technicians failed: %v", err)
	}
	if techCount != 0 {
		t.Fatalf("legacy technicians should be removed, got %d", techCount)
	}
}
