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
		{
			Version: "2026031002",
			Name:    "add_merchant_queue_timeout_waiting_seconds",
			Statements: []string{
				"ALTER TABLE merchants ADD COLUMN queue_timeout_waiting_seconds INT NOT NULL DEFAULT 900",
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

func TestRunMigrationsSupportsAlterTableCompatibilityGuards(t *testing.T) {
	oldMigrations := defaultMigrations
	defaultMigrations = []dbMigration{
		{
			Version: "2026031701_cleanup",
			Name:    "cleanup_appointment_compatibility_columns",
			Statements: []string{
				"ALTER TABLE appointments ADD COLUMN IF NOT EXISTS predicted_delay_minutes INT NOT NULL DEFAULT 0",
				"UPDATE appointments SET predicted_delay_minutes = predicted_wait_minutes WHERE predicted_delay_minutes = 0",
				"ALTER TABLE appointments DROP COLUMN IF EXISTS predicted_wait_minutes",
				"ALTER TABLE merchants DROP COLUMN IF EXISTS appointment_max_wait_minutes",
			},
		},
	}
	defer func() { defaultMigrations = oldMigrations }()

	dsn := "file:config_migrations_runner_compat_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE appointments (id integer primary key, predicted_wait_minutes integer not null default 3)").Error; err != nil {
		t.Fatalf("create appointments failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE merchants (id integer primary key)").Error; err != nil {
		t.Fatalf("create merchants failed: %v", err)
	}
	if err := db.Exec("INSERT INTO appointments (id, predicted_wait_minutes) VALUES (1, 7)").Error; err != nil {
		t.Fatalf("seed appointments failed: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	if !db.Migrator().HasColumn("appointments", "predicted_delay_minutes") {
		t.Fatalf("predicted_delay_minutes should exist")
	}
	if db.Migrator().HasColumn("appointments", "predicted_wait_minutes") {
		t.Fatalf("predicted_wait_minutes should be dropped")
	}

	var predictedDelay int
	if err := db.Raw("SELECT predicted_delay_minutes FROM appointments WHERE id = 1").Scan(&predictedDelay).Error; err != nil {
		t.Fatalf("query predicted_delay_minutes failed: %v", err)
	}
	if predictedDelay != 7 {
		t.Fatalf("want predicted_delay_minutes 7, got %d", predictedDelay)
	}
}

func TestAcquireMigrationLockSkipsNonMySQLDialectors(t *testing.T) {
	dsn := "file:config_migrations_runner_lock_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	release, err := acquireMigrationLock(db, 5)
	if err != nil {
		t.Fatalf("acquireMigrationLock should skip sqlite, got err: %v", err)
	}
	if release != nil {
		t.Fatalf("acquireMigrationLock should not return release func for sqlite")
	}
}

func TestRunMigrationsSkipsPlainAddColumnWhenColumnAlreadyExists(t *testing.T) {
	oldMigrations := defaultMigrations
	defaultMigrations = []dbMigration{
		{
			Version: "2026030701",
			Name:    "add_merchant_queue_waiting_start_seconds",
			Statements: []string{
				"ALTER TABLE merchants ADD COLUMN queue_waiting_start_seconds INT NOT NULL DEFAULT 180",
			},
		},
	}
	defer func() { defaultMigrations = oldMigrations }()

	dsn := "file:config_migrations_runner_plain_add_column_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE merchants (id integer primary key, queue_waiting_start_seconds integer not null default 180)").Error; err != nil {
		t.Fatalf("create merchants failed: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("run migrations failed: %v", err)
	}

	var count int64
	if err := db.Table("schema_migrations").Where("version = ?", "2026030701").Count(&count).Error; err != nil {
		t.Fatalf("count schema_migrations failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("want migration version to be recorded once, got %d", count)
	}
}

func TestHasAppliedMigrationDoesNotReusePreviousTableContext(t *testing.T) {
	dsn := "file:config_migrations_runner_has_applied_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE merchants (id integer primary key)").Error; err != nil {
		t.Fatalf("create merchants failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE schema_migrations (version varchar(64) primary key, name varchar(255), applied_at datetime)").Error; err != nil {
		t.Fatalf("create schema_migrations failed: %v", err)
	}
	if err := db.Exec("INSERT INTO schema_migrations(version, name, applied_at) VALUES ('2026030701', 'x', CURRENT_TIMESTAMP)").Error; err != nil {
		t.Fatalf("seed schema_migrations failed: %v", err)
	}

	tx := db.Table("merchants")
	if !hasAppliedMigration(tx, "2026030701") {
		t.Fatalf("expected hasAppliedMigration to query schema_migrations instead of previous table context")
	}
}

func TestHasColumnByTableNameDoesNotReusePreviousTableContext(t *testing.T) {
	dsn := "file:config_migrations_runner_has_column_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE merchants (id integer primary key, queue_waiting_start_seconds integer not null default 180)").Error; err != nil {
		t.Fatalf("create merchants failed: %v", err)
	}

	tx := db.Where("version = ?", "2026030701").Table("schema_migrations")
	if !hasColumnByTableName(tx, "merchants", "queue_waiting_start_seconds") {
		t.Fatalf("expected hasColumnByTableName to query merchants columns without leaking previous where clauses")
	}
}

func TestRunMigrationsSkipsDuplicateIndexStatements(t *testing.T) {
	oldMigrations := defaultMigrations
	defaultMigrations = []dbMigration{
		{
			Version: "2026031201",
			Name:    "add_service_session_source_fields",
			Statements: []string{
				"ALTER TABLE service_sessions ADD COLUMN source_id bigint unsigned NULL DEFAULT NULL",
				"ALTER TABLE service_sessions ADD INDEX idx_service_sessions_source_id (source_id)",
			},
		},
	}
	defer func() { defaultMigrations = oldMigrations }()

	dsn := "file:config_migrations_runner_duplicate_index_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE service_sessions (id integer primary key, source_id integer)").Error; err != nil {
		t.Fatalf("create service_sessions failed: %v", err)
	}
	if err := db.Exec("CREATE INDEX idx_service_sessions_source_id ON service_sessions(source_id)").Error; err != nil {
		t.Fatalf("create source_id index failed: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("run migrations failed: %v", err)
	}
}

func TestRunMigrationsSupportsServiceSessionTechnicianDropCompatibilityGuards(t *testing.T) {
	oldMigrations := defaultMigrations
	defaultMigrations = []dbMigration{
		{
			Version: "2026042002",
			Name:    "drop_service_session_technician_id",
			Statements: []string{
				"ALTER TABLE service_sessions DROP FOREIGN KEY IF EXISTS fk_service_sessions_technician",
				"ALTER TABLE service_sessions DROP INDEX IF EXISTS idx_service_sessions_technician_id",
				"ALTER TABLE service_sessions DROP COLUMN IF EXISTS technician_id",
			},
		},
	}
	defer func() { defaultMigrations = oldMigrations }()

	dsn := "file:config_migrations_runner_drop_technician_id_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE service_sessions (id integer primary key, technician_id integer, last_technician_id integer)").Error; err != nil {
		t.Fatalf("create service_sessions failed: %v", err)
	}
	if err := db.Exec("CREATE INDEX idx_service_sessions_technician_id ON service_sessions(technician_id)").Error; err != nil {
		t.Fatalf("create technician_id index failed: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	type sqliteColumnInfo struct {
		Name string `gorm:"column:name"`
	}
	var cols []sqliteColumnInfo
	if err := db.Raw("PRAGMA table_info(service_sessions)").Scan(&cols).Error; err != nil {
		t.Fatalf("query sqlite table_info failed: %v", err)
	}
	for _, col := range cols {
		if col.Name == "technician_id" {
			t.Fatalf("technician_id should be dropped")
		}
	}
	var indexCount int64
	if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", "idx_service_sessions_technician_id").Scan(&indexCount).Error; err != nil {
		t.Fatalf("query sqlite indexes failed: %v", err)
	}
	if indexCount > 0 {
		t.Fatalf("idx_service_sessions_technician_id should be dropped")
	}
}

func TestRunMigrationsSkipsPredictedDelayBackfillWhenLegacyColumnMissing(t *testing.T) {
	oldMigrations := defaultMigrations
	defaultMigrations = []dbMigration{
		{
			Version: "2026031701_cleanup",
			Name:    "cleanup_appointment_compatibility_columns",
			Statements: []string{
				"ALTER TABLE appointments ADD COLUMN IF NOT EXISTS predicted_delay_minutes INT NOT NULL DEFAULT 0",
				"UPDATE appointments SET predicted_delay_minutes = predicted_wait_minutes WHERE predicted_delay_minutes = 0",
				"ALTER TABLE appointments DROP COLUMN IF EXISTS predicted_wait_minutes",
			},
		},
	}
	defer func() { defaultMigrations = oldMigrations }()

	dsn := "file:config_migrations_runner_missing_legacy_column_test?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE appointments (id integer primary key, predicted_delay_minutes integer not null default 0)").Error; err != nil {
		t.Fatalf("create appointments failed: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("run migrations failed: %v", err)
	}
}
