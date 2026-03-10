package config

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type dbMigration struct {
	Version    string
	Name       string
	Statements []string
}

var defaultMigrations = []dbMigration{
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
			"CREATE TABLE IF NOT EXISTS merchant_role_start_pending_configs (\n  id int unsigned NOT NULL AUTO_INCREMENT,\n  merchant_id int unsigned NOT NULL,\n  service_role_id int unsigned NOT NULL,\n  start_pending_timeout_seconds int NOT NULL DEFAULT 300,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_m_role_start_pending (merchant_id, service_role_id),\n  KEY idx_m_role_start_pending_merchant (merchant_id),\n  KEY idx_m_role_start_pending_role (service_role_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户岗位待起单超时配置（覆盖service_roles.start_pending_timeout_seconds）'",
		},
	},
	{
		Version: "2026030902",
		Name:    "add_technician_original_password",
		Statements: []string{
			"ALTER TABLE technicians ADD COLUMN original_password varchar(100) NOT NULL DEFAULT '' COMMENT '最近一次系统生成或重置的原始密码（改密后清空）'",
		},
	},
	{
		Version: "2026031001",
		Name:    "add_project_start_delay_seconds",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN start_delay_seconds INT NOT NULL DEFAULT 60 COMMENT '服务开始延迟时间（秒）'",
		},
	},
}

func RunMigrations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	if err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version varchar(64) NOT NULL PRIMARY KEY,
  name varchar(255) NOT NULL DEFAULT '',
  applied_at datetime NOT NULL
)`).Error; err != nil {
		return err
	}

	for _, m := range defaultMigrations {
		var count int64
		if err := db.Table("schema_migrations").Where("version = ?", m.Version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, stmt := range m.Statements {
				if strings.TrimSpace(stmt) == "" {
					continue
				}
				if err := tx.Exec(stmt).Error; err != nil && !isIgnorableMigrationError(err) {
					return fmt.Errorf("migration %s failed on statement %q: %w", m.Version, stmt, err)
				}
			}
			return tx.Exec(
				"INSERT INTO schema_migrations(version, name, applied_at) VALUES (?, ?, ?)",
				m.Version, m.Name, time.Now(),
			).Error
		}); err != nil {
			return err
		}
	}

	return nil
}

func isIgnorableMigrationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	ignorable := []string{
		"duplicate column name",
		"already exists",
		"duplicate key name",
		"can't drop",
		"check that column/key exists",
	}
	for _, s := range ignorable {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
