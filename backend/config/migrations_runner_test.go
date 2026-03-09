package config

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunMigrationsIsIdempotent(t *testing.T) {
	oldMigrations := defaultMigrations
	defaultMigrations = []dbMigration{
		{
			Version: "2026030501",
			Name:    "add_technician_password_need_reset",
			Statements: []string{
				"ALTER TABLE technicians ADD COLUMN password_need_reset BOOLEAN NOT NULL DEFAULT 0",
			},
		},
		{
			Version: "2026030701",
			Name:    "add_merchant_queue_waiting_start_seconds",
			Statements: []string{
				"ALTER TABLE merchants ADD COLUMN queue_waiting_start_seconds INT NOT NULL DEFAULT 180",
			},
		},
		{
			Version: "2026030901",
			Name:    "add_role_start_pending_timeout_settings",
			Statements: []string{
				"ALTER TABLE service_roles ADD COLUMN start_pending_timeout_seconds INT NOT NULL DEFAULT 300",
				"CREATE TABLE IF NOT EXISTS merchant_role_start_pending_configs (id integer primary key, merchant_id integer not null, service_role_id integer not null, start_pending_timeout_seconds integer not null default 300)",
			},
		},
	}
	defer func() { defaultMigrations = oldMigrations }()

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
	if err := db.Exec("CREATE TABLE service_roles (id integer primary key)").Error; err != nil {
		t.Fatalf("create service_roles failed: %v", err)
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
