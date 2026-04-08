package config

import (
	"testing"

	"kabao/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDefaultProjectTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Merchant{}, &models.MerchantProject{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestResolveMerchantProjectFallsBackToDefault(t *testing.T) {
	db := setupDefaultProjectTestDB(t)
	merchant := models.Merchant{Name: "m1", Phone: "18800002222", Password: "pwd"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	project := models.MerchantProject{MerchantID: merchant.ID, Name: "默认", Duration: 45, IsDefault: true}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	got, err := ResolveMerchantProject(db, merchant.ID, nil)
	if err != nil {
		t.Fatalf("resolve project failed: %v", err)
	}
	if got == nil || got.ID != project.ID {
		t.Fatalf("expected default project %d, got %+v", project.ID, got)
	}
}
