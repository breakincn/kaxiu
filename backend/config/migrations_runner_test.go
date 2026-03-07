package config

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunMigrationsIsIdempotent(t *testing.T) {
	dsn := "file:config_migrations_runner_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE technicians (id integer primary key)").Error; err != nil {
		t.Fatalf("create technicians failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE merchants (id integer primary key)").Error; err != nil {
		t.Fatalf("create merchants failed: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	var count int64
	if err := db.Table("schema_migrations").Count(&count).Error; err != nil {
		t.Fatalf("count schema_migrations failed: %v", err)
	}
	if count != int64(len(defaultMigrations)) {
		t.Fatalf("want %d migrations, got %d", len(defaultMigrations), count)
	}
}
