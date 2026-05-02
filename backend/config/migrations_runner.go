package config

import (
	"database/sql"
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
		Version: "2026050201",
		Name:    "add_merchant_referral_commission_system",
		Statements: []string{
			"CREATE TABLE IF NOT EXISTS merchant_referral_profiles (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  user_id bigint unsigned NOT NULL,\n  promotion_code varchar(12) NOT NULL COMMENT '推广码',\n  default_commission_rate_bp int NOT NULL DEFAULT 5000 COMMENT '默认分成比例基点（10000=100%）',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_mrp_user_id (user_id),\n  UNIQUE KEY uidx_mrp_promotion_code (promotion_code)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户推广商户档案表'",
			"CREATE TABLE IF NOT EXISTS merchant_referrals (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  referrer_user_id bigint unsigned NOT NULL,\n  merchant_id bigint unsigned NOT NULL,\n  promotion_code varchar(12) NOT NULL COMMENT '推广码',\n  registered_at datetime(3) NULL DEFAULT NULL COMMENT '商户注册时间',\n  first_paid_at datetime(3) NULL DEFAULT NULL COMMENT '首次付费时间',\n  commission_expires_at datetime(3) NULL DEFAULT NULL COMMENT '分成截止时间',\n  default_commission_rate_bp int NOT NULL DEFAULT 5000 COMMENT '默认分成比例基点（10000=100%）',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_mr_merchant_id (merchant_id),\n  KEY idx_mr_referrer_user_id (referrer_user_id),\n  KEY idx_mr_promotion_code (promotion_code)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户推广商户邀请关系表'",
			"CREATE TABLE IF NOT EXISTS merchant_referral_withdrawals (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  referrer_user_id bigint unsigned NOT NULL,\n  applied_amount int NOT NULL DEFAULT 0 COMMENT '申请金额（单位：分）',\n  payee_name varchar(80) NOT NULL DEFAULT '' COMMENT '收款人',\n  payee_account varchar(120) NOT NULL DEFAULT '' COMMENT '收款账号',\n  payee_channel varchar(30) NOT NULL DEFAULT 'wechat' COMMENT '收款渠道（wechat/alipay/bank）',\n  status varchar(20) NOT NULL DEFAULT 'pending' COMMENT '审核状态（pending/approved/paid/rejected）',\n  reviewed_by varchar(60) NOT NULL DEFAULT '' COMMENT '审核人',\n  review_note varchar(255) NOT NULL DEFAULT '' COMMENT '审核备注',\n  reviewed_at datetime(3) NULL DEFAULT NULL COMMENT '审核时间',\n  paid_at datetime(3) NULL DEFAULT NULL COMMENT '打款时间',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_mrw_referrer_user_id (referrer_user_id),\n  KEY idx_mrw_status (status)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户推广商户分成提现申请表'",
			"CREATE TABLE IF NOT EXISTS merchant_referral_payment_ledgers (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  merchant_referral_id bigint unsigned NOT NULL,\n  referrer_user_id bigint unsigned NOT NULL,\n  merchant_id bigint unsigned NOT NULL,\n  paid_amount int NOT NULL DEFAULT 0 COMMENT '商户实付金额（单位：分）',\n  commission_rate_bp int NOT NULL DEFAULT 5000 COMMENT '分成比例基点（10000=100%）',\n  commission_amount int NOT NULL DEFAULT 0 COMMENT '应分成金额（单位：分）',\n  paid_at datetime(3) NULL DEFAULT NULL COMMENT '商户付款时间',\n  claim_deadline datetime(3) NULL DEFAULT NULL COMMENT '申请提现截止时间',\n  is_within_commission_window tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否命中分成窗口',\n  withdrawal_status varchar(20) NOT NULL DEFAULT 'claimable' COMMENT '提现状态（claimable/applied/approved/paid/expired/rejected）',\n  withdrawal_id bigint unsigned NULL DEFAULT NULL COMMENT '提现申请ID',\n  note varchar(255) NOT NULL DEFAULT '' COMMENT '备注',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_mrpl_merchant_referral_id (merchant_referral_id),\n  KEY idx_mrpl_referrer_user_id (referrer_user_id),\n  KEY idx_mrpl_merchant_id (merchant_id),\n  KEY idx_mrpl_withdrawal_id (withdrawal_id),\n  KEY idx_mrpl_withdrawal_status (withdrawal_status),\n  KEY idx_mrpl_paid_at (paid_at)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户推广商户付费分成台账表'",
			"CREATE TABLE IF NOT EXISTS merchant_referral_withdrawal_items (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  withdrawal_id bigint unsigned NOT NULL,\n  payment_ledger_id bigint unsigned NOT NULL,\n  commission_amount int NOT NULL DEFAULT 0 COMMENT '本明细金额（单位：分）',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_mrwi_ledger_once (payment_ledger_id),\n  UNIQUE KEY uidx_mrwi_withdrawal_ledger (withdrawal_id, payment_ledger_id),\n  KEY idx_mrwi_withdrawal_id (withdrawal_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户推广商户分成提现明细表'",
		},
	},
	{
		Version: "2026042801",
		Name:    "add_promotion_card_campaign_system",
		Statements: []string{
			"CREATE TABLE IF NOT EXISTS promotion_card_campaigns (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  merchant_id bigint unsigned NOT NULL,\n  card_template_id bigint unsigned NOT NULL,\n  title varchar(360) NOT NULL DEFAULT '' COMMENT '推广标题',\n  reward_total_times int NOT NULL DEFAULT 0 COMMENT '奖励次数',\n  reward_recharge_amount int NOT NULL DEFAULT 0 COMMENT '奖励额度（单位：分）',\n  reward_card_quantity int NOT NULL DEFAULT 0 COMMENT '奖励卡总库存',\n  reward_threshold int NOT NULL DEFAULT 0 COMMENT '达标推广数量',\n  promo_price int NOT NULL DEFAULT 0 COMMENT '促销价格（单位：分）',\n  promo_quantity int NOT NULL DEFAULT 0 COMMENT '促销资格总库存',\n  promo_ends_at datetime(3) NULL DEFAULT NULL COMMENT '促销截止时间',\n  slug varchar(80) NOT NULL COMMENT '活动短链接标识',\n  status varchar(20) NOT NULL DEFAULT 'active' COMMENT '活动状态（active/disabled）',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_promotion_card_campaigns_slug (slug),\n  KEY idx_promotion_card_campaigns_merchant_id (merchant_id),\n  KEY idx_promotion_card_campaigns_card_template_id (card_template_id),\n  KEY idx_promotion_card_campaigns_status (status)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='推广卡活动表'",
			"CREATE TABLE IF NOT EXISTS promotion_card_claims (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  campaign_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  referrer_user_id bigint unsigned NULL DEFAULT NULL,\n  promotion_code varchar(20) NOT NULL DEFAULT '' COMMENT '推广码',\n  status varchar(20) NOT NULL DEFAULT 'active' COMMENT '资格状态（active/expired/paid/confirmed）',\n  claimed_at datetime(3) NULL DEFAULT NULL,\n  expires_at datetime(3) NULL DEFAULT NULL,\n  paid_at datetime(3) NULL DEFAULT NULL,\n  expired_at datetime(3) NULL DEFAULT NULL,\n  order_id bigint unsigned NULL DEFAULT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_campaign_user_claim (campaign_id, user_id),\n  KEY idx_promotion_card_claims_referrer_user_id (referrer_user_id),\n  KEY idx_promotion_card_claims_order_id (order_id),\n  KEY idx_promotion_card_claims_status (status)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='推广卡促销领取资格表'",
			"CREATE TABLE IF NOT EXISTS promotion_card_referrers (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  campaign_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  promotion_code varchar(6) NOT NULL COMMENT '6位推广码',\n  register_count int NOT NULL DEFAULT 0 COMMENT '注册累计数',\n  paid_count int NOT NULL DEFAULT 0 COMMENT '付款累计数',\n  progress_count int NOT NULL DEFAULT 0 COMMENT '总进度',\n  reward_status varchar(20) NOT NULL DEFAULT '' COMMENT '奖励状态（/claimable/claimed/expired）',\n  reward_qualified_at datetime(3) NULL DEFAULT NULL,\n  reward_expires_at datetime(3) NULL DEFAULT NULL,\n  reward_claimed_at datetime(3) NULL DEFAULT NULL,\n  reward_expired_at datetime(3) NULL DEFAULT NULL,\n  reward_card_id bigint unsigned NULL DEFAULT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_campaign_user_referrer (campaign_id, user_id),\n  UNIQUE KEY uidx_campaign_promotion_code (campaign_id, promotion_code),\n  KEY idx_promotion_card_referrers_reward_card_id (reward_card_id),\n  KEY idx_promotion_card_referrers_reward_status (reward_status)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='推广卡推广人进度表'",
			"CREATE TABLE IF NOT EXISTS promotion_card_referrals (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  campaign_id bigint unsigned NOT NULL,\n  referrer_user_id bigint unsigned NOT NULL,\n  invitee_user_id bigint unsigned NOT NULL,\n  promotion_code varchar(6) NOT NULL COMMENT '推广码',\n  registered_counted_at datetime(3) NULL DEFAULT NULL,\n  paid_counted_at datetime(3) NULL DEFAULT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_campaign_referral (campaign_id, referrer_user_id, invitee_user_id),\n  KEY idx_promotion_card_referrals_promotion_code (promotion_code)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='推广卡邀请贡献表'",
			"ALTER TABLE direct_purchases ADD COLUMN source_type varchar(40) NOT NULL DEFAULT '' COMMENT '订单来源类型'",
			"ALTER TABLE direct_purchases ADD COLUMN source_id bigint unsigned NULL DEFAULT NULL COMMENT '来源业务ID'",
			"ALTER TABLE direct_purchases ADD COLUMN promotion_campaign_id bigint unsigned NULL DEFAULT NULL COMMENT '推广活动ID'",
			"ALTER TABLE direct_purchases ADD COLUMN promotion_claim_id bigint unsigned NULL DEFAULT NULL COMMENT '促销领取资格ID'",
			"ALTER TABLE direct_purchases ADD COLUMN referrer_user_id bigint unsigned NULL DEFAULT NULL COMMENT '推广人用户ID'",
			"ALTER TABLE direct_purchases ADD COLUMN promotion_code varchar(20) NOT NULL DEFAULT '' COMMENT '推广码'",
			"ALTER TABLE direct_purchases ADD COLUMN original_price int NOT NULL DEFAULT 0 COMMENT '原价（单位：分）'",
			"ALTER TABLE direct_purchases ADD COLUMN applied_price int NOT NULL DEFAULT 0 COMMENT '成交价（单位：分）'",
			"ALTER TABLE direct_purchases ADD INDEX idx_direct_purchases_source_id (source_id)",
			"ALTER TABLE direct_purchases ADD INDEX idx_direct_purchases_promotion_campaign_id (promotion_campaign_id)",
			"ALTER TABLE direct_purchases ADD INDEX idx_direct_purchases_promotion_claim_id (promotion_claim_id)",
			"ALTER TABLE direct_purchases ADD INDEX idx_direct_purchases_referrer_user_id (referrer_user_id)",
		},
	},
	{
		Version: "2026042501",
		Name:    "add_multi_service_booking_system",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN multi_service_booking_cancel_deadline_minutes_before_start INT NOT NULL DEFAULT 60 COMMENT '多人项目课程预约取消截止时间（距开课前分钟数）'",
			"ALTER TABLE usages ADD COLUMN source_type varchar(40) NOT NULL DEFAULT '' COMMENT '使用记录来源类型'",
			"ALTER TABLE usages ADD COLUMN source_note varchar(255) NOT NULL DEFAULT '' COMMENT '使用记录来源备注'",
			"CREATE TABLE IF NOT EXISTS multi_service_bookings (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  merchant_id bigint unsigned NOT NULL,\n  project_id bigint unsigned NOT NULL,\n  card_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  slot_start_at datetime(3) NULL DEFAULT NULL,\n  slot_end_at datetime(3) NULL DEFAULT NULL,\n  status varchar(20) NOT NULL DEFAULT 'booked' COMMENT '状态（booked/canceled/attended/no_show）',\n  booked_at datetime(3) NULL DEFAULT NULL,\n  canceled_at datetime(3) NULL DEFAULT NULL,\n  cancel_reason varchar(255) NOT NULL DEFAULT '',\n  cancel_penalty BOOLEAN NOT NULL DEFAULT 0,\n  attended_at datetime(3) NULL DEFAULT NULL,\n  no_show_at datetime(3) NULL DEFAULT NULL,\n  usage_id bigint unsigned NULL DEFAULT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uk_msb_slot_user (project_id, slot_start_at, user_id),\n  KEY idx_msb_merchant_id (merchant_id),\n  KEY idx_msb_project_id (project_id),\n  KEY idx_msb_card_id (card_id),\n  KEY idx_msb_user_id (user_id),\n  KEY idx_msb_slot_start_at (slot_start_at),\n  KEY idx_msb_status (status),\n  KEY idx_msb_usage_id (usage_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='多人项目课程预约表'",
			"CREATE TABLE IF NOT EXISTS multi_service_penalty_ledgers (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  merchant_id bigint unsigned NOT NULL,\n  project_id bigint unsigned NOT NULL,\n  card_id bigint unsigned NOT NULL,\n  user_id bigint unsigned NOT NULL,\n  booking_id bigint unsigned NULL DEFAULT NULL,\n  penalty_type varchar(30) NOT NULL DEFAULT '' COMMENT '惩罚类型（late_cancel/no_show/over_limit）',\n  counts_toward_no_show BOOLEAN NOT NULL DEFAULT 0 COMMENT '是否计入半年失约累计',\n  charged_times int NOT NULL DEFAULT 0 COMMENT '扣减次数',\n  charged_amount int NOT NULL DEFAULT 0 COMMENT '扣减额度',\n  usage_id bigint unsigned NULL DEFAULT NULL,\n  remark varchar(255) NOT NULL DEFAULT '' COMMENT '备注',\n  penalty_at datetime(3) NULL DEFAULT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  KEY idx_mspl_merchant_id (merchant_id),\n  KEY idx_mspl_project_id (project_id),\n  KEY idx_mspl_card_id (card_id),\n  KEY idx_mspl_user_id (user_id),\n  KEY idx_mspl_booking_id (booking_id),\n  KEY idx_mspl_penalty_type (penalty_type),\n  KEY idx_mspl_usage_id (usage_id),\n  KEY idx_mspl_penalty_at (penalty_at)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='多人项目课程失约惩罚账本'",
		},
	},
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
		Version: "2026040601",
		Name:    "change_appointment_slot_granularity_default_to_5",
		Statements: []string{
			"ALTER TABLE merchants MODIFY COLUMN appointment_slot_granularity_minutes INT NOT NULL DEFAULT 5 COMMENT '预约时段展示粒度分钟数'",
		},
	},
	{
		Version: "2026040602",
		Name:    "add_appointment_scheduling_mode_and_technician_project_bindings",
		Statements: []string{
			"ALTER TABLE merchants ADD COLUMN appointment_scheduling_mode varchar(40) NOT NULL DEFAULT 'technician_grouped' COMMENT '预约排布模式（technician_grouped/technician_mixed_timeline）'",
			"CREATE TABLE IF NOT EXISTS technician_appointment_projects (\n  id bigint unsigned NOT NULL AUTO_INCREMENT,\n  merchant_id bigint unsigned NOT NULL,\n  technician_id bigint unsigned NOT NULL,\n  project_id bigint unsigned NOT NULL,\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (id),\n  UNIQUE KEY uidx_tap_once (technician_id, project_id),\n  KEY idx_tap_merchant_id (merchant_id),\n  KEY idx_tap_technician_id (technician_id),\n  KEY idx_tap_project_id (project_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客服可预约项目绑定表'",
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
	{
		Version: "2026032701",
		Name:    "ensure_merchant_projects_bookable_online",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN IF NOT EXISTS bookable_online BOOLEAN NOT NULL DEFAULT 1 COMMENT '是否参与公开预约'",
		},
	},
	{
		Version: "2026040701",
		Name:    "add_project_auto_assign_technician_delay_minutes",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN auto_assign_technician_delay_minutes INT NOT NULL DEFAULT 5 COMMENT '自动分配客服延迟时间（分钟）'",
		},
	},
	{
		Version: "2026040702",
		Name:    "add_project_start_pending_timeout_seconds",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN start_pending_timeout_seconds INT NOT NULL DEFAULT 300 COMMENT '待开始服务倒计时秒数'",
		},
	},
	{
		Version: "2026040703",
		Name:    "add_project_room_select_timeout_seconds",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN room_select_timeout_seconds INT NOT NULL DEFAULT 90 COMMENT '自动分配房间延迟时间（秒）'",
		},
	},
	{
		Version: "2026040801",
		Name:    "add_project_is_default",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT 0 COMMENT '是否为商户默认项目'",
			"ALTER TABLE merchant_projects ADD INDEX idx_merchant_default (merchant_id, is_default)",
		},
	},
	{
		Version: "2026040901",
		Name:    "add_merchant_appointment_reschedule_deadline_minutes_before_start",
		Statements: []string{
			"ALTER TABLE merchants ADD COLUMN appointment_reschedule_deadline_minutes_before_start INT NOT NULL DEFAULT 60 COMMENT '预约改签截止时间（距服务开始前分钟数）'",
		},
	},
	{
		Version: "2026041601",
		Name:    "create_service_session_extend_requests",
		Statements: []string{
			"CREATE TABLE IF NOT EXISTS service_session_extend_requests (\n  id bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '服务加钟申请ID',\n  merchant_id bigint unsigned NOT NULL COMMENT '商户ID',\n  user_id bigint unsigned NOT NULL COMMENT '用户ID',\n  card_id bigint unsigned NOT NULL COMMENT '卡片ID',\n  service_session_id bigint unsigned NOT NULL COMMENT '服务会话ID',\n  initial_usage_id bigint unsigned NOT NULL COMMENT '首次核销使用记录ID',\n  project_id bigint unsigned NOT NULL COMMENT '申请加钟项目ID',\n  minutes int NOT NULL COMMENT '申请加钟时长分钟',\n  before_remaining_seconds int NOT NULL DEFAULT 0 COMMENT '确认加钟前服务剩余秒数',\n  after_remaining_seconds int NOT NULL DEFAULT 0 COMMENT '确认加钟后服务剩余秒数',\n  status varchar(20) NOT NULL DEFAULT 'pending' COMMENT '状态（pending/approved/rejected/canceled）',\n  reject_reason varchar(255) NOT NULL DEFAULT '' COMMENT '拒绝原因',\n  handled_by_type varchar(20) NOT NULL DEFAULT '' COMMENT '处理人类型',\n  handled_by_id bigint unsigned NULL DEFAULT NULL COMMENT '处理人ID',\n  handled_at datetime(3) NULL DEFAULT NULL COMMENT '处理时间',\n  created_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',\n  updated_at datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',\n  PRIMARY KEY (id),\n  KEY idx_sser_merchant_id (merchant_id),\n  KEY idx_sser_user_id (user_id),\n  KEY idx_sser_card_id (card_id),\n  KEY idx_sser_service_session_id (service_session_id),\n  KEY idx_sser_initial_usage_id (initial_usage_id),\n  KEY idx_sser_project_id (project_id),\n  KEY idx_sser_status (status),\n  KEY idx_sser_handled_by_id (handled_by_id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务加钟申请表'",
		},
	},
	{
		Version: "2026041602",
		Name:    "add_service_session_extend_request_remaining_snapshots",
		Statements: []string{
			"ALTER TABLE service_session_extend_requests ADD COLUMN before_remaining_seconds INT NOT NULL DEFAULT 0 COMMENT '确认加钟前服务剩余秒数'",
			"ALTER TABLE service_session_extend_requests ADD COLUMN after_remaining_seconds INT NOT NULL DEFAULT 0 COMMENT '确认加钟后服务剩余秒数'",
		},
	},
	{
		Version: "2026041603",
		Name:    "repair_service_session_auto_finish_delay_after_extension",
		Statements: []string{
			"UPDATE service_sessions s JOIN merchants m ON m.id = s.merchant_id SET s.auto_finish_delay_seconds = CASE WHEN m.support_customer_service_mode = 0 AND m.support_queue = 1 AND m.queue_mode IN ('auto','manual') THEN 300 ELSE 60 END WHERE s.auto_finish_delay_seconds > CASE WHEN m.support_customer_service_mode = 0 AND m.support_queue = 1 AND m.queue_mode IN ('auto','manual') THEN 300 ELSE 60 END",
			"UPDATE service_sessions s JOIN merchants m ON m.id = s.merchant_id SET s.finished_at = DATE_ADD(s.scheduled_finish_at, INTERVAL CASE WHEN m.support_customer_service_mode = 0 AND m.support_queue = 1 AND m.queue_mode IN ('auto','manual') THEN 300 ELSE 60 END SECOND) WHERE s.status IN ('auto_finishing','cs_auto_finishing','qs_auto_finishing','qm_auto_finishing','qms_auto_finishing','qmm_auto_finishing') AND s.scheduled_finish_at IS NOT NULL AND s.finished_at IS NOT NULL AND s.finished_at > DATE_ADD(s.scheduled_finish_at, INTERVAL CASE WHEN m.support_customer_service_mode = 0 AND m.support_queue = 1 AND m.queue_mode IN ('auto','manual') THEN 300 ELSE 60 END SECOND)",
		},
	},
	{
		Version: "2026041604",
		Name:    "add_usage_card_snapshots",
		Statements: []string{
			"ALTER TABLE usages ADD COLUMN card_no_snapshot varchar(50) NOT NULL DEFAULT '' COMMENT '核销时卡号快照'",
			"ALTER TABLE usages ADD COLUMN card_type_snapshot varchar(100) NOT NULL DEFAULT '' COMMENT '核销时卡片类型快照'",
			"ALTER TABLE usages ADD COLUMN card_total_times_snapshot INT NULL DEFAULT NULL COMMENT '核销后卡片总次数快照'",
			"ALTER TABLE usages ADD COLUMN card_used_times_snapshot INT NULL DEFAULT NULL COMMENT '核销后卡片已用次数快照'",
			"ALTER TABLE usages ADD COLUMN card_remain_times_snapshot INT NULL DEFAULT NULL COMMENT '核销后卡片剩余次数快照'",
			"UPDATE usages u JOIN cards c ON c.id = u.card_id SET u.card_no_snapshot = c.card_no, u.card_type_snapshot = c.card_type WHERE u.card_no_snapshot = '' OR u.card_type_snapshot = ''",
			"UPDATE usages u JOIN cards c ON c.id = u.card_id LEFT JOIN (SELECT older.id AS usage_id, COALESCE(SUM(CASE WHEN later.used_times > 0 THEN later.used_times ELSE 0 END), 0) AS later_used FROM usages older LEFT JOIN usages later ON later.card_id = older.card_id AND later.status <> 'failed' AND later.id <> older.id AND ((older.used_at IS NOT NULL AND later.used_at IS NOT NULL AND later.used_at > older.used_at) OR (older.used_at IS NOT NULL AND later.used_at = older.used_at AND later.id > older.id) OR (older.used_at IS NULL AND later.id > older.id)) WHERE older.status <> 'failed' GROUP BY older.id) x ON x.usage_id = u.id SET u.card_total_times_snapshot = c.total_times, u.card_used_times_snapshot = GREATEST(c.used_times - COALESCE(x.later_used, 0), 0), u.card_remain_times_snapshot = c.remain_times + COALESCE(x.later_used, 0) WHERE u.status <> 'failed' AND (u.card_total_times_snapshot IS NULL OR u.card_used_times_snapshot IS NULL OR u.card_remain_times_snapshot IS NULL)",
		},
	},
	{
		Version: "2026041801",
		Name:    "add_merchant_show_province",
		Statements: []string{
			"ALTER TABLE merchants ADD COLUMN show_province BOOLEAN NOT NULL DEFAULT 0 COMMENT '用户端地址是否展示省份'",
		},
	},
	{
		Version: "2026041802",
		Name:    "add_project_service_capacity_and_time_slots",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN service_capacity INT NOT NULL DEFAULT 1 COMMENT '服务人数'",
			"ALTER TABLE merchant_projects ADD COLUMN service_time_slots JSON NULL COMMENT '服务时间槽'",
			"UPDATE merchant_projects SET service_time_slots = JSON_ARRAY() WHERE service_time_slots IS NULL",
			"ALTER TABLE merchant_projects MODIFY COLUMN service_time_slots JSON NOT NULL COMMENT '服务时间槽'",
		},
	},
	{
		Version: "2026041803",
		Name:    "add_project_show_participants",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN show_participants BOOLEAN NOT NULL DEFAULT 1 COMMENT '用户端是否展示参与服务用户'",
		},
	},
	{
		Version: "2026041901",
		Name:    "add_project_default_service_technician_ids",
		Statements: []string{
			"ALTER TABLE merchant_projects ADD COLUMN default_service_technician_ids JSON NULL COMMENT '多人项目默认服务人员ID列表（专业客服）'",
			"UPDATE merchant_projects SET default_service_technician_ids = JSON_ARRAY() WHERE default_service_technician_ids IS NULL",
			"ALTER TABLE merchant_projects MODIFY COLUMN default_service_technician_ids JSON NOT NULL COMMENT '多人项目默认服务人员ID列表（专业客服）'",
		},
	},
	{
		Version: "2026041902",
		Name:    "add_service_session_technician_ids",
		Statements: []string{
			"ALTER TABLE service_sessions ADD COLUMN service_technician_ids JSON NULL COMMENT '服务单绑定的服务人员ID列表（专业客服）'",
			"UPDATE service_sessions SET service_technician_ids = CASE WHEN technician_id IS NULL THEN JSON_ARRAY() ELSE JSON_ARRAY(technician_id) END WHERE service_technician_ids IS NULL",
			"ALTER TABLE service_sessions MODIFY COLUMN service_technician_ids JSON NOT NULL COMMENT '服务单绑定的服务人员ID列表（专业客服）'",
		},
	},
	{
		Version: "2026042001",
		Name:    "add_service_session_start_confirmed_technician_ids",
		Statements: []string{
			"ALTER TABLE service_sessions ADD COLUMN start_confirmed_technician_ids JSON NULL COMMENT '已扫码确认待开始服务的服务人员ID列表'",
			"UPDATE service_sessions SET start_confirmed_technician_ids = CASE WHEN start_confirmed_at IS NOT NULL AND technician_id IS NOT NULL THEN JSON_ARRAY(technician_id) ELSE JSON_ARRAY() END WHERE start_confirmed_technician_ids IS NULL",
			"ALTER TABLE service_sessions MODIFY COLUMN start_confirmed_technician_ids JSON NOT NULL COMMENT '已扫码确认待开始服务的服务人员ID列表'",
		},
	},
	{
		Version: "2026042002",
		Name:    "drop_service_session_technician_id",
		Statements: []string{
			"ALTER TABLE service_sessions DROP FOREIGN KEY IF EXISTS fk_service_sessions_technician",
			"ALTER TABLE service_sessions DROP INDEX IF EXISTS idx_service_sessions_technician_id",
			"ALTER TABLE service_sessions DROP INDEX IF EXISTS technician_id",
			"ALTER TABLE service_sessions DROP COLUMN IF EXISTS technician_id",
		},
	},
}

func RunMigrations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	if db.Dialector.Name() != "mysql" {
		return runMigrationsOnDB(db)
	}

	return db.Connection(func(conn *gorm.DB) error {
		releaseLock, err := acquireMigrationLock(conn, 30*time.Second)
		if err != nil {
			return err
		}
		if releaseLock != nil {
			defer func() {
				_ = releaseLock()
			}()
		}
		return runMigrationsOnDB(conn)
	})
}

func runMigrationsOnDB(db *gorm.DB) error {
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
			if err := insertSchemaMigrationRecord(tx, m.Version, m.Name, time.Now()); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func acquireMigrationLock(db *gorm.DB, timeout time.Duration) (func() error, error) {
	if db == nil || db.Dialector.Name() != "mysql" {
		return nil, nil
	}

	lockTimeoutSeconds := int(timeout / time.Second)
	if lockTimeoutSeconds <= 0 {
		lockTimeoutSeconds = 1
	}

	var result struct {
		Acquired sql.NullInt64 `gorm:"column:acquired"`
	}
	if err := db.Raw("SELECT GET_LOCK(?, ?) AS acquired", "kabao:schema_migrations", lockTimeoutSeconds).Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("acquire migration lock failed: %w", err)
	}
	if !result.Acquired.Valid || result.Acquired.Int64 != 1 {
		return nil, fmt.Errorf("acquire migration lock timed out after %ds", lockTimeoutSeconds)
	}

	return func() error {
		var release struct {
			Released sql.NullInt64 `gorm:"column:released"`
		}
		if err := db.Raw("SELECT RELEASE_LOCK(?) AS released", "kabao:schema_migrations").Scan(&release).Error; err != nil {
			return fmt.Errorf("release migration lock failed: %w", err)
		}
		if release.Released.Valid && release.Released.Int64 == 0 {
			return fmt.Errorf("migration lock is not held by current connection")
		}
		return nil
	}, nil
}

var (
	alterTableAddColumnIfNotExistsPattern   = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+ADD\s+COLUMN\s+IF\s+NOT\s+EXISTS\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+(.+)$`)
	alterTableAddColumnPattern              = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+ADD\s+COLUMN\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+(.+)$`)
	alterTableAddIndexPattern               = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+ADD\s+(?:UNIQUE\s+)?INDEX\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s*\(.+\)$`)
	alterTableDropIndexIfExistsPattern      = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+DROP\s+INDEX\s+IF\s+EXISTS\s+(` + "`?[A-Za-z0-9_]+`?" + `)$`)
	alterTableDropForeignKeyIfExistsPattern = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+DROP\s+FOREIGN\s+KEY\s+IF\s+EXISTS\s+(` + "`?[A-Za-z0-9_]+`?" + `)$`)
	alterTableDropColumnIfExistsPattern     = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(` + "`?[A-Za-z0-9_]+`?" + `)\s+DROP\s+COLUMN\s+IF\s+EXISTS\s+(` + "`?[A-Za-z0-9_]+`?" + `)(.*)$`)
	appointmentsPredictedDelayBackfillSQL   = regexp.MustCompile(`(?i)^UPDATE\s+appointments\s+SET\s+predicted_delay_minutes\s*=\s*predicted_wait_minutes\s+WHERE\s+predicted_delay_minutes\s*=\s*0$`)
	serviceSessionAutoFinishRepairSQL       = regexp.MustCompile(`(?i)^UPDATE\s+service_sessions\s+s\s+JOIN\s+merchants\s+m\s+ON\s+m\.id\s*=\s*s\.merchant_id\s+SET\s+`)
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
	if strings.Contains(trimmed, "UPDATE service_sessions SET service_technician_ids = CASE WHEN technician_id IS NULL") && !hasColumnByTableName(tx, "service_sessions", "technician_id") {
		return "UPDATE service_sessions SET service_technician_ids = JSON_ARRAY() WHERE service_technician_ids IS NULL", false
	}
	if strings.Contains(trimmed, "UPDATE service_sessions SET start_confirmed_technician_ids = CASE WHEN start_confirmed_at IS NOT NULL AND technician_id IS NOT NULL") && !hasColumnByTableName(tx, "service_sessions", "technician_id") {
		return "UPDATE service_sessions SET start_confirmed_technician_ids = JSON_ARRAY() WHERE start_confirmed_technician_ids IS NULL", false
	}

	if matches := alterTableAddColumnIfNotExistsPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		tableToken, columnToken, definition := matches[1], matches[2], matches[3]
		if hasColumnByTableName(tx, unquoteIdentifier(tableToken), unquoteIdentifier(columnToken)) {
			return "", true
		}
		return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableToken, columnToken, definition), false
	}

	if matches := alterTableAddColumnPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		tableToken, columnToken, definition := matches[1], matches[2], matches[3]
		if hasColumnByTableName(tx, unquoteIdentifier(tableToken), unquoteIdentifier(columnToken)) {
			return "", true
		}
		return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableToken, columnToken, definition), false
	}

	if matches := alterTableAddIndexPattern.FindStringSubmatch(trimmed); len(matches) == 3 {
		tableToken, indexToken := matches[1], matches[2]
		if hasIndexByTableName(tx, unquoteIdentifier(tableToken), unquoteIdentifier(indexToken)) {
			return "", true
		}
		return stmt, false
	}

	if matches := alterTableDropIndexIfExistsPattern.FindStringSubmatch(trimmed); len(matches) == 3 {
		tableToken, indexToken := matches[1], matches[2]
		if !hasIndexByTableName(tx, unquoteIdentifier(tableToken), unquoteIdentifier(indexToken)) {
			return "", true
		}
		if tx.Dialector != nil && tx.Dialector.Name() == "sqlite" {
			return fmt.Sprintf("DROP INDEX %s", indexToken), false
		}
		return fmt.Sprintf("ALTER TABLE %s DROP INDEX %s", tableToken, indexToken), false
	}

	if matches := alterTableDropForeignKeyIfExistsPattern.FindStringSubmatch(trimmed); len(matches) == 3 {
		tableToken, constraintToken := matches[1], matches[2]
		if tx.Dialector == nil || tx.Dialector.Name() != "mysql" {
			return "", true
		}
		if !hasConstraintByTableName(tx, unquoteIdentifier(tableToken), unquoteIdentifier(constraintToken)) {
			return "", true
		}
		return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", tableToken, constraintToken), false
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

	if appointmentsPredictedDelayBackfillSQL.MatchString(trimmed) {
		if !hasColumnByTableName(tx, "appointments", "predicted_wait_minutes") {
			return "", true
		}
		if !hasColumnByTableName(tx, "appointments", "predicted_delay_minutes") {
			return "", true
		}
		return stmt, false
	}

	if serviceSessionAutoFinishRepairSQL.MatchString(trimmed) && tx.Dialector.Name() != "mysql" {
		return "", true
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
	cleanDB := tx.Session(&gorm.Session{NewDB: true})
	columnTypes, err := cleanDB.Migrator().ColumnTypes(tableName)
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

func hasIndexByTableName(tx *gorm.DB, tableName, indexName string) bool {
	if tx == nil || tableName == "" || indexName == "" {
		return false
	}
	cleanDB := tx.Session(&gorm.Session{NewDB: true})
	return cleanDB.Migrator().HasIndex(tableName, indexName)
}

func hasConstraintByTableName(tx *gorm.DB, tableName, constraintName string) bool {
	if tx == nil || tableName == "" || constraintName == "" {
		return false
	}
	cleanDB := tx.Session(&gorm.Session{NewDB: true})
	return cleanDB.Migrator().HasConstraint(tableName, constraintName)
}

func hasAppliedMigration(tx *gorm.DB, version string) bool {
	if tx == nil || strings.TrimSpace(version) == "" {
		return false
	}
	var count int64
	query := tx.Session(&gorm.Session{NewDB: true}).Table("schema_migrations")
	if err := query.Where("version = ?", version).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func insertSchemaMigrationRecord(tx *gorm.DB, version, name string, appliedAt time.Time) error {
	if tx == nil {
		return fmt.Errorf("tx is nil")
	}

	if tx.Dialector.Name() == "mysql" {
		return tx.Exec(
			"INSERT IGNORE INTO schema_migrations(version, name, applied_at) VALUES (?, ?, ?)",
			version, name, appliedAt,
		).Error
	}

	if err := tx.Exec(
		"INSERT INTO schema_migrations(version, name, applied_at) VALUES (?, ?, ?)",
		version, name, appliedAt,
	).Error; err != nil {
		if hasAppliedMigration(tx, version) {
			return nil
		}
		return err
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
