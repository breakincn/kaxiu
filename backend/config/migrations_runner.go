package config

import (
	"fmt"
	"regexp"
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
			"ALTER TABLE merchants ADD COLUMN appointment_prediction_buffer_minutes INT NOT NULL DEFAULT 5 COMMENT '预约保护预测缓冲分钟数'",
			"ALTER TABLE appointments ADD COLUMN confirmed_at datetime(3) NULL DEFAULT NULL COMMENT '确认时间'",
			"ALTER TABLE appointments ADD COLUMN arrived_at datetime(3) NULL DEFAULT NULL COMMENT '到店核销时间'",
			"ALTER TABLE appointments ADD COLUMN completed_at datetime(3) NULL DEFAULT NULL COMMENT '完成时间'",
			"ALTER TABLE appointments ADD COLUMN no_show_at datetime(3) NULL DEFAULT NULL COMMENT '失约时间'",
			"ALTER TABLE appointments ADD COLUMN service_session_id bigint unsigned NULL DEFAULT NULL COMMENT '关联服务会话ID'",
			"ALTER TABLE appointments ADD COLUMN usage_id bigint unsigned NULL DEFAULT NULL COMMENT '关联核销记录ID'",
			"ALTER TABLE appointments ADD COLUMN predicted_delay_minutes INT NOT NULL DEFAULT 0 COMMENT '预约预计延迟分钟数'",
			"ALTER TABLE appointments ADD COLUMN resolution_note varchar(255) NOT NULL DEFAULT '' COMMENT '改签/补偿/人工处理备注'",
			"ALTER TABLE appointments ADD INDEX idx_appointments_service_session_id (service_session_id)",
			"ALTER TABLE appointments ADD INDEX idx_appointments_usage_id (usage_id)",
			"ALTER TABLE service_sessions ADD COLUMN occupies_next_appointment BOOLEAN NOT NULL DEFAULT 0 COMMENT '是否命中过预约保护冲突'",
			"ALTER TABLE service_sessions ADD COLUMN next_appointment_id bigint unsigned NULL DEFAULT NULL COMMENT '受保护的下一预约ID'",
			"ALTER TABLE service_sessions ADD COLUMN predicted_appointment_delay_minutes INT NOT NULL DEFAULT 0 COMMENT '预约到店后检测到的预计延迟分钟数'",
			"ALTER TABLE service_sessions ADD COLUMN predicted_ready_at datetime(3) NULL DEFAULT NULL COMMENT '预计可开始服务时间'",
			"ALTER TABLE service_sessions ADD INDEX idx_service_sessions_next_appointment_id (next_appointment_id)",
		},
	},
	{
		Version: "2026031701_cleanup",
		Name:    "cleanup_appointment_compatibility_columns",
		Statements: []string{
			"ALTER TABLE appointments ADD COLUMN IF NOT EXISTS predicted_delay_minutes INT NOT NULL DEFAULT 0 COMMENT '预约预计延迟分钟数'",
			"UPDATE appointments SET predicted_delay_minutes = predicted_wait_minutes WHERE predicted_delay_minutes = 0",
			"ALTER TABLE appointments DROP COLUMN IF EXISTS predicted_wait_minutes",
			"ALTER TABLE merchants DROP COLUMN IF EXISTS appointment_max_wait_minutes",
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
			"CREATE TABLE IF NOT EXISTS appointment_protection_blocks (\n  id bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',\n  merchant_id bigint unsigned NOT NULL COMMENT '商户ID',\n  appointment_id bigint unsigned NOT NULL COMMENT '被保护的预约ID',\n  technician_id bigint unsigned NOT NULL COMMENT '被保护预约绑定的客服ID',\n  walk_in_service_session_id bigint unsigned NULL DEFAULT NULL COMMENT '被拒绝分配的现场服务会话ID',\n  blocked_reason varchar(255) NOT NULL DEFAULT '' COMMENT '拒派原因说明',\n  predicted_reserved_wait_minutes INT NOT NULL DEFAULT 0 COMMENT '若继续派单将占用预约锁定时段的预计分钟数',\n  alternative_wait_minutes INT NOT NULL DEFAULT 0 COMMENT '改派其他客服的预计等待分钟数',\n  decision_mode varchar(50) NOT NULL DEFAULT '' COMMENT '拒派决策模式（strict_reservation_lock 等）',\n  blocked_at datetime(3) NULL DEFAULT NULL COMMENT '拒派发生时间',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_appointment_protection_blocks_once (appointment_id, technician_id, walk_in_service_session_id, decision_mode),\n  KEY idx_appointment_protection_blocks_merchant_id (merchant_id),\n  KEY idx_appointment_protection_blocks_appointment_id (appointment_id),\n  KEY idx_appointment_protection_blocks_technician_id (technician_id),\n  KEY idx_appointment_protection_blocks_walk_in_service_session_id (walk_in_service_session_id),\n  KEY idx_appointment_protection_blocks_blocked_at (blocked_at)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='预约保护拒派审计表'",
		},
	},
	{
		Version: "2026031802",
		Name:    "add_project_gap_and_appointment_slot_granularity",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN service_gap_minutes INT NOT NULL DEFAULT 3 COMMENT '服务间歇时间（分钟）'",
			"ALTER TABLE merchant_projects ADD COLUMN bookable_online BOOLEAN NOT NULL DEFAULT 1 COMMENT '是否参与公开预约'",
			"ALTER TABLE merchants ADD COLUMN appointment_slot_granularity_minutes INT NOT NULL DEFAULT 15 COMMENT '预约时段展示粒度分钟数'",
		},
	},
	{
		Version: "2026032301",
		Name:    "add_appointment_locking_publishings_and_protected_repair_slots",
		Statements: []string{
			"ALTER TABLE appointments ADD COLUMN booking_root_id bigint unsigned NULL DEFAULT NULL COMMENT '预约根ID（改签链路共用）'",
			"ALTER TABLE appointments ADD COLUMN reserved_start_at datetime(3) NULL DEFAULT NULL COMMENT '预约锁定开始时间'",
			"ALTER TABLE appointments ADD COLUMN reserved_end_at datetime(3) NULL DEFAULT NULL COMMENT '预约锁定结束时间'",
			"ALTER TABLE appointments ADD COLUMN occupied_end_at datetime(3) NULL DEFAULT NULL COMMENT '占用截止时间（含服务间隙）'",
			"ALTER TABLE appointments ADD COLUMN cancel_deadline_at datetime(3) NULL DEFAULT NULL COMMENT '用户取消截止时间'",
			"ALTER TABLE appointments ADD COLUMN booking_close_deadline_at datetime(3) NULL DEFAULT NULL COMMENT '新预约关闭时间'",
			"ALTER TABLE appointments ADD COLUMN late_arrival_min_service_minutes INT NOT NULL DEFAULT 0 COMMENT '迟到后最小可服务分钟数'",
			"ALTER TABLE appointments ADD INDEX idx_appointments_booking_root_id (booking_root_id)",
			"CREATE TABLE IF NOT EXISTS technician_schedule_publishings (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  merchant_id bigint unsigned NOT NULL,\n  technician_id bigint unsigned NULL DEFAULT NULL,\n  publish_date date NULL DEFAULT NULL,\n  start_at datetime(3) NULL DEFAULT NULL,\n  end_at datetime(3) NULL DEFAULT NULL,\n  status varchar(20) NOT NULL DEFAULT 'published' COMMENT '状态（published/leave/canceled）',\n  published_at datetime(3) NULL DEFAULT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_tsp_merchant_id (merchant_id),\n  KEY idx_tsp_technician_id (technician_id),\n  KEY idx_tsp_publish_date (publish_date),\n  KEY idx_tsp_status (status)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客服次日可预约排班发布表'",
			"CREATE TABLE IF NOT EXISTS protected_repair_slots (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  merchant_id bigint unsigned NOT NULL,\n  appointment_id bigint unsigned NOT NULL,\n  technician_id bigint unsigned NULL DEFAULT NULL,\n  publish_date date NULL DEFAULT NULL,\n  start_at datetime(3) NULL DEFAULT NULL,\n  end_at datetime(3) NULL DEFAULT NULL,\n  status varchar(20) NOT NULL DEFAULT 'reserved' COMMENT '状态（reserved/released/consumed/expired）',\n  source_type varchar(30) NOT NULL DEFAULT '',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_prs_merchant_id (merchant_id),\n  KEY idx_prs_appointment_id (appointment_id),\n  KEY idx_prs_technician_id (technician_id),\n  KEY idx_prs_publish_date (publish_date),\n  KEY idx_prs_status (status)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='受影响预约保护修复容量表'",
		},
	},
	{
		Version: "2026032302",
		Name:    "add_appointment_cancel_requests",
		Statements: []string{
			"CREATE TABLE IF NOT EXISTS appointment_cancel_requests (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  appointment_id bigint unsigned NOT NULL,\n  merchant_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  status varchar(30) NOT NULL DEFAULT 'pending_user' COMMENT '状态（pending_user/accepted/rejected/canceled）',\n  reason varchar(255) NOT NULL DEFAULT '' COMMENT '取消原因',\n  proposed_by_type varchar(20) NOT NULL DEFAULT '' COMMENT '申请发起方类型（merchant/staff）',\n  proposed_by_id bigint unsigned NULL DEFAULT NULL,\n  confirmed_by_type varchar(20) NOT NULL DEFAULT '' COMMENT '确认/拒绝方类型（user/merchant/staff）',\n  confirmed_by_id bigint unsigned NULL DEFAULT NULL,\n  confirmed_at datetime(3) NULL DEFAULT NULL,\n  objection_note varchar(255) NOT NULL DEFAULT '' COMMENT '用户抗辩说明',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_acr_appointment_id (appointment_id),\n  KEY idx_acr_merchant_id (merchant_id),\n  KEY idx_acr_user_id (user_id),\n  KEY idx_acr_status (status),\n  KEY idx_acr_proposed_by_id (proposed_by_id),\n  KEY idx_acr_confirmed_by_id (confirmed_by_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='预约取消申请表'",
		},
	},
	{
		Version: "2026032303",
		Name:    "add_appointment_settlements_and_p1_snapshot_fields",
		Statements: []string{
			"ALTER TABLE appointments ADD COLUMN appointment_settlement_id bigint unsigned NULL DEFAULT NULL COMMENT '关联预约结算ID'",
			"ALTER TABLE appointments ADD COLUMN settlement_status_snapshot varchar(30) NOT NULL DEFAULT '' COMMENT '预约侧结算状态快照'",
			"ALTER TABLE appointments ADD COLUMN actual_arrived_at datetime(3) NULL DEFAULT NULL COMMENT '实际到店时间'",
			"ALTER TABLE appointments ADD COLUMN actual_start_at datetime(3) NULL DEFAULT NULL COMMENT '实际开始服务时间'",
			"ALTER TABLE appointments ADD COLUMN merchant_breach_pending BOOLEAN NOT NULL DEFAULT 0 COMMENT '是否处于商户违约待判定风险标记'",
			"ALTER TABLE appointments ADD COLUMN breach_decision_at datetime(3) NULL DEFAULT NULL COMMENT '违约最终判定时间'",
			"ALTER TABLE appointments ADD COLUMN disruption_status varchar(30) NOT NULL DEFAULT '' COMMENT '异常闭环状态快照'",
			"ALTER TABLE appointments ADD COLUMN disruption_reason varchar(255) NOT NULL DEFAULT '' COMMENT '异常原因'",
			"ALTER TABLE appointments ADD COLUMN merchant_cancel_reason varchar(255) NOT NULL DEFAULT '' COMMENT '商户取消原因'",
			"ALTER TABLE appointments ADD COLUMN user_rebuttal_note varchar(255) NOT NULL DEFAULT '' COMMENT '用户抗辩说明'",
			"ALTER TABLE appointments ADD COLUMN liability_level varchar(30) NOT NULL DEFAULT '' COMMENT '责任级别快照'",
			"ALTER TABLE appointments ADD COLUMN salary_settlement_reference_status varchar(20) NOT NULL DEFAULT '' COMMENT '薪资对账参考状态'",
			"ALTER TABLE appointments ADD INDEX idx_appointments_appointment_settlement_id (appointment_settlement_id)",
			"CREATE TABLE IF NOT EXISTS appointment_settlements (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  appointment_id bigint unsigned NOT NULL,\n  booking_root_id bigint unsigned NULL DEFAULT NULL,\n  merchant_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  card_id bigint unsigned NOT NULL,\n  project_id bigint unsigned NULL DEFAULT NULL,\n  status varchar(30) NOT NULL DEFAULT 'pending' COMMENT '结算状态（pending/locked/deducted/refunded/transferred/offset）',\n  asset_mode varchar(20) NOT NULL DEFAULT '' COMMENT '资产处理模式（deduct/freeze）',\n  deducted_times INT NOT NULL DEFAULT 0 COMMENT '已扣减次数',\n  frozen_times INT NOT NULL DEFAULT 0 COMMENT '已冻结次数',\n  refunded_times INT NOT NULL DEFAULT 0 COMMENT '已退款或解冻次数',\n  transferred_to_settlement_id bigint unsigned NULL DEFAULT NULL,\n  settlement_status_snapshot varchar(30) NOT NULL DEFAULT '' COMMENT '当前结算状态快照',\n  merchant_breach_pending BOOLEAN NOT NULL DEFAULT 0 COMMENT '是否处于商户违约待判定状态',\n  breach_decision_at datetime(3) NULL DEFAULT NULL COMMENT '违约最终判定时间',\n  liability_level varchar(30) NOT NULL DEFAULT '' COMMENT '责任级别',\n  salary_settlement_reference_status varchar(20) NOT NULL DEFAULT '' COMMENT '薪资对账参考状态',\n  latest_reason varchar(255) NOT NULL DEFAULT '' COMMENT '最近一次结算原因',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_as_appointment_id (appointment_id),\n  KEY idx_as_booking_root_id (booking_root_id),\n  KEY idx_as_merchant_id (merchant_id),\n  KEY idx_as_user_id (user_id),\n  KEY idx_as_card_id (card_id),\n  KEY idx_as_project_id (project_id),\n  KEY idx_as_status (status),\n  KEY idx_as_transferred_to_settlement_id (transferred_to_settlement_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='预约结算表'",
		},
	},
	{
		Version: "2026032304",
		Name:    "add_force_majeure_relief_requests_and_compensation_links",
		Statements: []string{
			"ALTER TABLE appointment_compensations ADD COLUMN source_type varchar(30) NOT NULL DEFAULT '' COMMENT '来源类型（merchant_breach/merchant_failure_offset/delay_ledger/manual）'",
			"ALTER TABLE appointment_compensations ADD COLUMN source_id bigint unsigned NULL DEFAULT NULL COMMENT '来源ID'",
			"ALTER TABLE appointment_compensations ADD COLUMN force_majeure_relief_request_id bigint unsigned NULL DEFAULT NULL COMMENT '关联不可抗力撤销申请ID'",
			"ALTER TABLE appointment_compensations ADD INDEX idx_appointment_compensations_source_id (source_id)",
			"ALTER TABLE appointment_compensations ADD INDEX idx_appointment_compensations_force_majeure_relief_request_id (force_majeure_relief_request_id)",
			"CREATE TABLE IF NOT EXISTS force_majeure_relief_requests (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  appointment_id bigint unsigned NOT NULL,\n  merchant_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  proposed_by_type varchar(20) NOT NULL DEFAULT '' COMMENT '申请发起方类型（merchant/staff/user）',\n  proposed_by_id bigint unsigned NULL DEFAULT NULL,\n  reason varchar(255) NOT NULL DEFAULT '' COMMENT '不可抗力原因',\n  evidence_note varchar(255) NOT NULL DEFAULT '' COMMENT '补充举证说明',\n  status varchar(20) NOT NULL DEFAULT 'pending' COMMENT '状态（pending/accepted/rejected/canceled）',\n  confirmed_by_type varchar(20) NOT NULL DEFAULT '' COMMENT '确认/拒绝方类型（merchant/staff/user）',\n  confirmed_by_id bigint unsigned NULL DEFAULT NULL,\n  confirmed_at datetime(3) NULL DEFAULT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_fmrr_appointment_id (appointment_id),\n  KEY idx_fmrr_merchant_id (merchant_id),\n  KEY idx_fmrr_user_id (user_id),\n  KEY idx_fmrr_status (status),\n  KEY idx_fmrr_proposed_by_id (proposed_by_id),\n  KEY idx_fmrr_confirmed_by_id (confirmed_by_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='不可抗力撤销违约补偿申请表'",
		},
	},
	{
		Version: "2026032305",
		Name:    "add_technician_disruption_counters_and_ledgers",
		Statements: []string{
			"CREATE TABLE IF NOT EXISTS technician_monthly_disruption_counters (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  merchant_id bigint unsigned NOT NULL,\n  technician_id bigint unsigned NOT NULL,\n  month_key varchar(7) NOT NULL DEFAULT '' COMMENT '自然月标识（YYYY-MM）',\n  leave_disruption_count int NOT NULL DEFAULT 0 COMMENT '当月请假导致未履约次数',\n  first_exempt_used BOOLEAN NOT NULL DEFAULT 0 COMMENT '当月首次免责是否已使用',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uk_tmdc_merchant_technician_month (merchant_id, technician_id, month_key),\n  KEY idx_tmdc_merchant_id (merchant_id),\n  KEY idx_tmdc_technician_id (technician_id),\n  KEY idx_tmdc_month_key (month_key)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客服月度异常责任计数表'",
			"CREATE TABLE IF NOT EXISTS technician_disruption_ledgers (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  appointment_id bigint unsigned NOT NULL,\n  booking_root_id bigint unsigned NULL DEFAULT NULL,\n  merchant_id bigint unsigned NOT NULL,\n  technician_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  month_key varchar(7) NOT NULL DEFAULT '' COMMENT '自然月标识（YYYY-MM）',\n  disruption_reason varchar(30) NOT NULL DEFAULT '' COMMENT '异常原因',\n  liability_level varchar(30) NOT NULL DEFAULT '' COMMENT '责任级别（merchant_exempt/technician_chargeable）',\n  salary_settlement_reference_status varchar(20) NOT NULL DEFAULT 'open' COMMENT '薪资对账参考状态（open/reconciled）',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uk_tdl_appointment_id (appointment_id),\n  KEY idx_tdl_booking_root_id (booking_root_id),\n  KEY idx_tdl_merchant_id (merchant_id),\n  KEY idx_tdl_technician_id (technician_id),\n  KEY idx_tdl_month_key (month_key),\n  KEY idx_tdl_liability_level (liability_level)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客服异常责任账本'",
		},
	},
	{
		Version: "2026032306",
		Name:    "add_appointment_delay_ledgers_and_project_delay_fields",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN delay_tolerance_minutes INT NOT NULL DEFAULT 1 COMMENT '拖堂补偿容忍分钟数'",
			"ALTER TABLE merchant_projects ADD COLUMN delay_compensation_mode varchar(20) NOT NULL DEFAULT 'minutes_bucket' COMMENT '拖堂补偿模式（minutes_bucket/amount_bucket/fixed_unit）'",
			"ALTER TABLE merchant_projects ADD COLUMN delay_redeem_threshold_percent INT NOT NULL DEFAULT 100 COMMENT '拖堂补偿兑现阈值百分比'",
			"ALTER TABLE merchant_projects ADD COLUMN delay_fixed_unit_value INT NOT NULL DEFAULT 0 COMMENT '固定单位补偿值'",
			"CREATE TABLE IF NOT EXISTS appointment_delay_ledgers (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  appointment_id bigint unsigned NOT NULL,\n  booking_root_id bigint unsigned NULL DEFAULT NULL,\n  merchant_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  card_id bigint unsigned NOT NULL,\n  project_id bigint unsigned NULL DEFAULT NULL,\n  service_session_id bigint unsigned NULL DEFAULT NULL,\n  technician_id bigint unsigned NULL DEFAULT NULL,\n  scheduled_start_at datetime(3) NULL DEFAULT NULL,\n  actual_start_at datetime(3) NULL DEFAULT NULL,\n  delay_minutes int NOT NULL DEFAULT 0,\n  credited_minutes int NOT NULL DEFAULT 0,\n  service_duration_minutes int NOT NULL DEFAULT 0,\n  unit_compensation_value int NOT NULL DEFAULT 0,\n  delay_compensation_value int NOT NULL DEFAULT 0,\n  ledger_status varchar(20) NOT NULL DEFAULT 'recorded' COMMENT '账本状态（recorded/redeemed/ignored）',\n  redeem_status varchar(20) NOT NULL DEFAULT 'pending' COMMENT '兑现状态（pending/redeemed/skipped）',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uk_adl_appointment_id (appointment_id),\n  KEY idx_adl_booking_root_id (booking_root_id),\n  KEY idx_adl_merchant_id (merchant_id),\n  KEY idx_adl_user_id (user_id),\n  KEY idx_adl_card_id (card_id),\n  KEY idx_adl_project_id (project_id),\n  KEY idx_adl_service_session_id (service_session_id),\n  KEY idx_adl_technician_id (technician_id),\n  KEY idx_adl_ledger_status (ledger_status),\n  KEY idx_adl_redeem_status (redeem_status)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='预约拖堂补偿账本'",
		},
	},
	{
		Version: "2026032401",
		Name:    "add_appointment_reschedule_recommendation_toggle",
		Statements: []string{
			"ALTER TABLE merchants ADD COLUMN appointment_reschedule_recommendation_enabled BOOLEAN NOT NULL DEFAULT 0 COMMENT '是否启用系统优化性改签推荐'",
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
				if err := execMigrationStatement(tx, stmt); err != nil && !isIgnorableMigrationError(err) {
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

var (
	alterTableAddColumnIfNotExistsPattern = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+ADD\s+COLUMN\s+IF\s+NOT\s+EXISTS\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+(.+)$`)
	alterTableDropColumnIfExistsPattern   = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+DROP\s+COLUMN\s+IF\s+EXISTS\s+(` + "`?[A-Za-z0-9_]+`?" + `)(.*)$`)
)

func execMigrationStatement(tx *gorm.DB, stmt string) error {
	rewritten, skip := rewriteMigrationStatementForCompatibility(tx, stmt)
	if skip {
		return nil
	}
	return tx.Exec(rewritten).Error
}

func rewriteMigrationStatementForCompatibility(tx *gorm.DB, stmt string) (rewritten string, skip bool) {
	trimmed := strings.TrimSpace(stmt)
	if trimmed == "" {
		return "", true
	}

	if matches := alterTableAddColumnIfNotExistsPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		tableToken, columnToken, definition := matches[1], matches[2], matches[3]
		if hasColumnByTableName(tx, unquoteIdentifier(tableToken), unquoteIdentifier(columnToken)) {
			return "", true
		}
		return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableToken, columnToken, definition), false
	}

	if matches := alterTableDropColumnIfExistsPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		tableToken, columnToken, suffix := matches[1], matches[2], strings.TrimSpace(matches[3])
		if !hasColumnByTableName(tx, unquoteIdentifier(tableToken), unquoteIdentifier(columnToken)) {
			return "", true
		}
		rewritten = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableToken, columnToken)
		if suffix != "" {
			rewritten += " " + suffix
		}
		return rewritten, false
	}

	return stmt, false
}

func unquoteIdentifier(name string) string {
	return strings.Trim(name, "`")
}

func hasColumnByTableName(tx *gorm.DB, tableName, columnName string) bool {
	if tx == nil || tableName == "" || columnName == "" {
		return false
	}
	columnTypes, err := tx.Migrator().ColumnTypes(tableName)
	if err != nil {
		return false
	}
	for _, columnType := range columnTypes {
		if strings.EqualFold(columnType.Name(), columnName) {
			return true
		}
	}
	return false
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
