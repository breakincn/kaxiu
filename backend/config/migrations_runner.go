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
			"CREATE TABLE IF NOT EXISTS merchant_role_start_pending_configs (\n  id int unsigned NOT NULL AUTO_INCREMENT,\n  merchant_id int unsigned NOT NULL,\n  service_role_id int unsigned NOT NULL,\n  start_pending_timeout_seconds int NOT NULL DEFAULT 300,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_m_role_start_pending (merchant_id, service_role_id),\n  KEY idx_m_role_start_pending_merchant (merchant_id),\n  KEY idx_m_role_start_pending_role (service_role_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户岗位待开始服务超时配置（覆盖service_roles.start_pending_timeout_seconds）'",
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
	{
		Version: "2026031002",
		Name:    "add_merchant_queue_timeout_waiting_seconds",
		Statements: []string{
			"ALTER TABLE merchants ADD COLUMN queue_timeout_waiting_seconds INT NOT NULL DEFAULT 900 COMMENT '超时过号等待时间（秒）'",
		},
	},
	{
		Version: "2026031201",
		Name:    "add_service_session_source_fields",
		Statements: []string{
			"ALTER TABLE service_sessions ADD COLUMN source_type varchar(20) NOT NULL DEFAULT 'walk_in' COMMENT '会话来源（walk_in/appointment）'",
			"ALTER TABLE service_sessions ADD COLUMN source_id bigint unsigned NULL DEFAULT NULL COMMENT '来源业务ID（如appointment_id）'",
			"ALTER TABLE service_sessions ADD INDEX idx_service_sessions_source_id (source_id)",
		},
	},
	{
		Version: "2026031701",
		Name:    "add_appointment_closure_fields",
		Statements: []string{
			"ALTER TABLE merchants ADD COLUMN appointment_reserve_buffer_minutes INT NOT NULL DEFAULT 10 COMMENT '预约前保留缓冲分钟数'",
			"ALTER TABLE merchants ADD COLUMN appointment_grace_window_minutes INT NOT NULL DEFAULT 15 COMMENT '预约后到店宽限分钟数'",
			"ALTER TABLE merchants ADD COLUMN appointment_max_wait_minutes INT NOT NULL DEFAULT 15 COMMENT '预约客户最大可承诺等待分钟数'",
			"ALTER TABLE merchants ADD COLUMN appointment_prediction_buffer_minutes INT NOT NULL DEFAULT 5 COMMENT '预约保护预测缓冲分钟数'",
			"ALTER TABLE appointments ADD COLUMN confirmed_at datetime(3) NULL DEFAULT NULL COMMENT '确认时间'",
			"ALTER TABLE appointments ADD COLUMN arrived_at datetime(3) NULL DEFAULT NULL COMMENT '到店核销时间'",
			"ALTER TABLE appointments ADD COLUMN completed_at datetime(3) NULL DEFAULT NULL COMMENT '完成时间'",
			"ALTER TABLE appointments ADD COLUMN no_show_at datetime(3) NULL DEFAULT NULL COMMENT '失约时间'",
			"ALTER TABLE appointments ADD COLUMN service_session_id bigint unsigned NULL DEFAULT NULL COMMENT '关联服务会话ID'",
			"ALTER TABLE appointments ADD COLUMN usage_id bigint unsigned NULL DEFAULT NULL COMMENT '关联核销记录ID'",
			"ALTER TABLE appointments ADD COLUMN predicted_wait_minutes INT NOT NULL DEFAULT 0 COMMENT '预约预计等待分钟数'",
			"ALTER TABLE appointments ADD COLUMN resolution_note varchar(255) NOT NULL DEFAULT '' COMMENT '改签/补偿/人工处理备注'",
			"ALTER TABLE appointments ADD INDEX idx_appointments_service_session_id (service_session_id)",
			"ALTER TABLE appointments ADD INDEX idx_appointments_usage_id (usage_id)",
			"ALTER TABLE service_sessions ADD COLUMN occupies_next_appointment BOOLEAN NOT NULL DEFAULT 0 COMMENT '是否占用了下一预约的预计时段'",
			"ALTER TABLE service_sessions ADD COLUMN next_appointment_id bigint unsigned NULL DEFAULT NULL COMMENT '受影响的下一预约ID'",
			"ALTER TABLE service_sessions ADD COLUMN predicted_appointment_delay_minutes INT NOT NULL DEFAULT 0 COMMENT '压预约派单预计会造成的预约等待分钟数'",
			"ALTER TABLE service_sessions ADD COLUMN predicted_ready_at datetime(3) NULL DEFAULT NULL COMMENT '预计可开始服务时间'",
			"ALTER TABLE service_sessions ADD INDEX idx_service_sessions_next_appointment_id (next_appointment_id)",
		},
	},
	{
		Version: "2026031702",
		Name:    "add_appointment_reschedule_and_compensation",
		Statements: []string{
			"ALTER TABLE appointments ADD COLUMN closed_reason varchar(50) NOT NULL DEFAULT '' COMMENT '关闭原因（canceled/rescheduled/no_show/completed等）'",
			"ALTER TABLE appointments ADD COLUMN closed_by_type varchar(20) NOT NULL DEFAULT '' COMMENT '关闭操作人类型（merchant/staff/system/user）'",
			"ALTER TABLE appointments ADD COLUMN closed_by_id bigint unsigned NULL DEFAULT NULL COMMENT '关闭操作人ID'",
			"ALTER TABLE appointments ADD COLUMN reschedule_reason varchar(255) NOT NULL DEFAULT '' COMMENT '改签原因'",
			"ALTER TABLE appointments ADD COLUMN replaced_by_appointment_id bigint unsigned NULL DEFAULT NULL COMMENT '本预约被哪条新预约替代'",
			"ALTER TABLE appointments ADD COLUMN replaces_appointment_id bigint unsigned NULL DEFAULT NULL COMMENT '本预约替代了哪条旧预约'",
			"ALTER TABLE appointments ADD INDEX idx_appointments_closed_by_id (closed_by_id)",
			"ALTER TABLE appointments ADD INDEX idx_appointments_replaced_by_appointment_id (replaced_by_appointment_id)",
			"ALTER TABLE appointments ADD INDEX idx_appointments_replaces_appointment_id (replaces_appointment_id)",
			"CREATE TABLE IF NOT EXISTS appointment_compensations (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  appointment_id bigint unsigned NOT NULL,\n  merchant_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  card_id bigint unsigned NOT NULL,\n  service_session_id bigint unsigned NULL DEFAULT NULL,\n  type varchar(30) NOT NULL COMMENT '补偿类型（extra_times/extend_minutes/discount_note/other_note）',\n  value int NOT NULL DEFAULT 0,\n  status varchar(20) NOT NULL DEFAULT 'pending' COMMENT '状态（pending/applied/canceled）',\n  reason varchar(255) NOT NULL DEFAULT '',\n  remark varchar(255) NOT NULL DEFAULT '',\n  created_by_type varchar(20) NOT NULL DEFAULT '',\n  created_by_id bigint unsigned NULL DEFAULT NULL,\n  applied_at datetime(3) NULL DEFAULT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_appointment_compensations_appointment_id (appointment_id),\n  KEY idx_appointment_compensations_merchant_id (merchant_id),\n  KEY idx_appointment_compensations_user_id (user_id),\n  KEY idx_appointment_compensations_card_id (card_id),\n  KEY idx_appointment_compensations_service_session_id (service_session_id),\n  KEY idx_appointment_compensations_created_by_id (created_by_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='预约补偿记录表'",
		},
	},
	{
		Version: "2026031703",
		Name:    "add_appointment_reschedule_requests",
		Statements: []string{
			"CREATE TABLE IF NOT EXISTS appointment_reschedule_requests (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  appointment_id bigint unsigned NOT NULL,\n  merchant_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  current_project_id bigint unsigned NULL DEFAULT NULL,\n  current_technician_id bigint unsigned NULL DEFAULT NULL,\n  current_appointment_time datetime(3) NULL DEFAULT NULL,\n  new_project_id bigint unsigned NULL DEFAULT NULL,\n  new_technician_id bigint unsigned NULL DEFAULT NULL,\n  new_appointment_time datetime(3) NULL DEFAULT NULL,\n  status varchar(30) NOT NULL DEFAULT 'pending_user' COMMENT '状态（pending_user/pending_merchant/accepted/rejected/canceled）',\n  reason varchar(255) NOT NULL DEFAULT '',\n  proposed_by_type varchar(20) NOT NULL DEFAULT '',\n  proposed_by_id bigint unsigned NULL DEFAULT NULL,\n  confirmed_by_type varchar(20) NOT NULL DEFAULT '',\n  confirmed_by_id bigint unsigned NULL DEFAULT NULL,\n  confirmed_at datetime(3) NULL DEFAULT NULL,\n  result_appointment_id bigint unsigned NULL DEFAULT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_appointment_reschedule_requests_appointment_id (appointment_id),\n  KEY idx_appointment_reschedule_requests_merchant_id (merchant_id),\n  KEY idx_appointment_reschedule_requests_user_id (user_id),\n  KEY idx_appointment_reschedule_requests_status (status),\n  KEY idx_appointment_reschedule_requests_proposed_by_id (proposed_by_id),\n  KEY idx_appointment_reschedule_requests_confirmed_by_id (confirmed_by_id),\n  KEY idx_appointment_reschedule_requests_result_appointment_id (result_appointment_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='预约改签提议表'",
		},
	},
	{
		Version: "2026031801",
		Name:    "add_appointment_reschedule_windows_and_protection_blocks",
		Statements: []string{
			"ALTER TABLE merchants ADD COLUMN appointment_reschedule_same_or_next_day_threshold_minutes INT NOT NULL DEFAULT 180 COMMENT '昨天预约可改签到今天或明天的剩余分钟阈值'",
			"ALTER TABLE merchants ADD COLUMN appointment_reschedule_next_day_only_threshold_minutes INT NOT NULL DEFAULT 90 COMMENT '昨天预约仅可改签到明天的剩余分钟阈值'",
			"CREATE TABLE IF NOT EXISTS appointment_protection_blocks (\n  id bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',\n  merchant_id bigint unsigned NOT NULL COMMENT '商户ID',\n  appointment_id bigint unsigned NOT NULL COMMENT '被保护的预约ID',\n  technician_id bigint unsigned NOT NULL COMMENT '被保护预约绑定的客服ID',\n  walk_in_service_session_id bigint unsigned NULL DEFAULT NULL COMMENT '被拒绝分配的现场服务会话ID',\n  blocked_reason varchar(255) NOT NULL DEFAULT '' COMMENT '拒派原因说明',\n  predicted_reserved_wait_minutes INT NOT NULL DEFAULT 0 COMMENT '若继续派单将导致预约等待的预计分钟数',\n  alternative_wait_minutes INT NOT NULL DEFAULT 0 COMMENT '改派其他客服的预计等待分钟数',\n  decision_mode varchar(50) NOT NULL DEFAULT '' COMMENT '拒派决策模式（max_wait_protection/alternative_preferred）',\n  blocked_at datetime(3) NULL DEFAULT NULL COMMENT '拒派发生时间',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_appointment_protection_blocks_once (appointment_id, technician_id, walk_in_service_session_id, decision_mode),\n  KEY idx_appointment_protection_blocks_merchant_id (merchant_id),\n  KEY idx_appointment_protection_blocks_appointment_id (appointment_id),\n  KEY idx_appointment_protection_blocks_technician_id (technician_id),\n  KEY idx_appointment_protection_blocks_walk_in_service_session_id (walk_in_service_session_id),\n  KEY idx_appointment_protection_blocks_blocked_at (blocked_at)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='预约保护拒派审计表'",
		},
	},
	{
		Version: "2026031802",
		Name:    "add_project_gap_and_appointment_slot_granularity",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN service_gap_minutes INT NOT NULL DEFAULT 3 COMMENT '服务间歇时间（分钟）'",
			"ALTER TABLE merchants ADD COLUMN appointment_slot_granularity_minutes INT NOT NULL DEFAULT 15 COMMENT '预约时段展示粒度分钟数'",
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
