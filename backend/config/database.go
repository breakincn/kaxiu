package config

import (
	"encoding/json"
	"kabao/models"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := os.Getenv("KABAO_DSN")
	if dsn == "" {
		dbConfig := GetDatabaseConfig()
		dsn = dbConfig.GetDSN()
	}
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 先删除旧的外键约束，仅在约束存在时执行，避免启动时打印误报。
	dropForeignKeyIfExists("cards", "fk_cards_user")
	dropForeignKeyIfExists("cards", "fk_cards_merchant")
	dropForeignKeyIfExists("usages", "fk_usages_card")
	dropForeignKeyIfExists("usages", "fk_usages_merchant")
	dropForeignKeyIfExists("notices", "fk_notices_merchant")
	// DB.Exec("ALTER TABLE `appointments` DROP FOREIGN KEY `fk_appointments_user`")
	// DB.Exec("ALTER TABLE `appointments` DROP FOREIGN KEY `fk_appointments_merchant`")

	// 自动迁移
	// 临时跳过迁移，优先启动HTTPS服务
	log.Println("数据库连接成功，跳过迁移（HTTPS模式）")
	/*
		err = DB.AutoMigrate(
			&models.User{},
			&models.Merchant{},
			&models.Technician{},
			&models.TechnicianAttendance{},
			// &models.ServiceRole{},  // 跳过，避免索引冲突
			&models.Permission{},
			&models.RolePermission{},
			&models.MerchantRolePermissionOverride{},
			&models.SystemConfig{},
			&models.Card{},
			&models.CardProject{},
			&models.Usage{},
			&models.Room{},
			&models.ServiceSession{},
			&models.ServiceSessionExtendRequest{},
			&models.Notice{},
			&models.Appointment{},
			&models.MerchantProject{},
			&models.CardTemplateProject{},
			// &models.Shop{},
			&models.DirectPurchase{},
			&models.TechnicianAttendance{},
		)
		if err != nil {
			log.Fatal("数据库迁移失败:", err)
		}
		log.Println("数据库迁移完成")
	*/

	if !IsTruthyEnv("KABAO_ENABLE_LEGACY_INLINE_DDL") {
		if err := RunMigrations(DB); err != nil {
			log.Fatal("数据库迁移失败:", err)
		}
		migrateLegacyMerchantProjects()
		if err := EnsureAllMerchantsDefaultProjects(DB); err != nil {
			log.Fatal("补齐默认项目失败:", err)
		}
		log.Println("数据库初始化成功")

		// 初始化商户注册邀请码（幂等）
		initInviteCodes()
		initServiceRoles()
		cleanupServiceRoles()
		initPermissions()
		initProfessionalBasePermissionConfig()
		initRolePermissions()
		initTestData()
		return
	}

	migrateLegacyMerchantProjects()
	if err := EnsureAllMerchantsDefaultProjects(DB); err != nil {
		log.Fatal("补齐默认项目失败:", err)
	}

	// 兼容历史数据：为旧用户补充默认 username，避免新增唯一索引导致异常
	DB.Exec("UPDATE users SET username = CONCAT('u', id) WHERE username IS NULL OR username = ''")

	// 技师账号唯一性调整：从 account 全局唯一改为 (merchant_id, account) 商户内唯一
	// 尝试删除旧的 account 唯一索引（不同环境下索引名可能不同，忽略错误即可）
	DB.Exec("ALTER TABLE `technicians` DROP INDEX `idx_technicians_account`")
	DB.Exec("ALTER TABLE `technicians` DROP INDEX `account`")
	DB.Exec("ALTER TABLE `technicians` ADD COLUMN `original_password` varchar(100) NOT NULL DEFAULT '' COMMENT '最近一次系统生成或重置的原始密码（改密后清空）'")

	// 修正 technicians 表唯一索引：确保每个 (商户, 角色) 的编号独立自增
	// 1. 清理可能存在的错误索引名
	DB.Exec("ALTER TABLE `technicians` DROP INDEX `uidx_merchant_code`")
	DB.Exec("ALTER TABLE `technicians` DROP INDEX `uidx_merchant_role_code`")
	// 2. 清理冲突数据：同一商户同一角色下重复的 code 只保留最早的一条
	DB.Exec("DELETE t1 FROM technicians t1 INNER JOIN technicians t2 WHERE t1.id > t2.id AND t1.merchant_id = t2.merchant_id AND t1.service_role_id = t2.service_role_id AND t1.code = t2.code")
	// 3. 创建正确的唯一索引 (merchant_id, service_role_id, code)
	DB.Exec("ALTER TABLE `technicians` ADD UNIQUE INDEX `uidx_merchant_role_code` (`merchant_id`, `service_role_id`, `code`) COMMENT '同一商户同一角色下编号唯一'")

	// service_roles: 账号前缀（最多5个字母，用于生成工作人员账号）
	DB.Exec("ALTER TABLE `service_roles` ADD COLUMN `account_prefix` varchar(5) NOT NULL DEFAULT '' COMMENT '账号前缀（最多5个英文字母）'")
	// service_roles: 支持商户自定义专业岗位
	DB.Exec("ALTER TABLE `service_roles` ADD COLUMN `merchant_id` int unsigned NULL DEFAULT NULL COMMENT '所属商户ID（NULL 表示平台默认）'")
	DB.Exec("ALTER TABLE `service_roles` ADD COLUMN `role_type` varchar(20) NOT NULL DEFAULT '' COMMENT '角色类型：operational/professional'")
	// service_roles: 是否需要签到（按岗位配置）
	DB.Exec("ALTER TABLE `service_roles` ADD COLUMN `require_attendance` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否需要签到（0-不需要，1-需要）'")
	// service_roles: 岗位默认待开始服务超时秒数
	DB.Exec("ALTER TABLE `service_roles` ADD COLUMN `start_pending_timeout_seconds` int NOT NULL DEFAULT 300 COMMENT '岗位默认待开始服务超时秒数'")

	// merchant_role_attendance_configs: 商户维度覆盖岗位签到配置
	DB.Exec("CREATE TABLE IF NOT EXISTS `merchant_role_attendance_configs` (\n  `id` int unsigned NOT NULL AUTO_INCREMENT,\n  `merchant_id` int unsigned NOT NULL,\n  `service_role_id` int unsigned NOT NULL,\n  `require_attendance` tinyint(1) NOT NULL DEFAULT 1,\n  `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (`id`),\n  UNIQUE KEY `uidx_m_role_att` (`merchant_id`,`service_role_id`),\n  KEY `idx_m_role_att_merchant` (`merchant_id`),\n  KEY `idx_m_role_att_role` (`service_role_id`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户岗位签到配置（覆盖service_roles.require_attendance）'")
	// merchant_role_start_pending_configs: 商户维度覆盖岗位待开始服务超时秒数
	DB.Exec("CREATE TABLE IF NOT EXISTS `merchant_role_start_pending_configs` (\n  `id` int unsigned NOT NULL AUTO_INCREMENT,\n  `merchant_id` int unsigned NOT NULL,\n  `service_role_id` int unsigned NOT NULL,\n  `start_pending_timeout_seconds` int NOT NULL DEFAULT 300,\n  `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),\n  `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),\n  PRIMARY KEY (`id`),\n  UNIQUE KEY `uidx_m_role_start_pending` (`merchant_id`,`service_role_id`),\n  KEY `idx_m_role_start_pending_merchant` (`merchant_id`),\n  KEY `idx_m_role_start_pending_role` (`service_role_id`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户岗位待开始服务超时配置（覆盖service_roles.start_pending_timeout_seconds）'")

	// 添加表注释
	DB.Exec("ALTER TABLE `users` COMMENT = '用户表'")
	DB.Exec("ALTER TABLE `merchants` COMMENT = '商户表'")
	DB.Exec("ALTER TABLE `technicians` COMMENT = '商户技师账号表'")
	DB.Exec("ALTER TABLE `service_roles` COMMENT = '平台客服类型表'")
	DB.Exec("ALTER TABLE `permissions` COMMENT = '权限枚举表'")
	DB.Exec("ALTER TABLE `role_permissions` COMMENT = '平台角色默认权限表'")
	DB.Exec("ALTER TABLE `merchant_role_permission_overrides` COMMENT = '商户角色权限微调表'")
	DB.Exec("ALTER TABLE `cards` COMMENT = '用户会员卡表'")
	DB.Exec("ALTER TABLE `usages` COMMENT = '卡片使用记录表'")
	DB.Exec("ALTER TABLE `rooms` COMMENT = '商户房间配置表'")
	DB.Exec("ALTER TABLE `technician_attendances` COMMENT = '工作人员签到与可服务状态表'")
	DB.Exec("ALTER TABLE `service_sessions` COMMENT = '服务会话表'")
	DB.Exec("ALTER TABLE `notices` COMMENT = '商户通知表'")
	DB.Exec("ALTER TABLE `appointments` COMMENT = '用户预约表'")
	DB.Exec("ALTER TABLE `verify_codes` COMMENT = '核销码表'")
	DB.Exec("ALTER TABLE `sms_codes` COMMENT = '短信验证码表'")
	DB.Exec("ALTER TABLE `invite_codes` COMMENT = '邀请码表'")
	// Shop 模块表注释
	DB.Exec("ALTER TABLE `payment_configs` COMMENT = '商户收款配置表'")
	DB.Exec("ALTER TABLE `card_templates` COMMENT = '卡片售卖模板表'")
	DB.Exec("ALTER TABLE `direct_purchases` COMMENT = '直购订单记录表'")
	DB.Exec("ALTER TABLE `merchant_shop_slugs` COMMENT = '商户店铺短链接表'")

	// 添加字段注释
	addFieldComments()

	// service_sessions: 选客服无空闲时冷却截止时间
	DB.Exec("ALTER TABLE `service_sessions` ADD COLUMN `staff_select_cooldown_until` datetime(3) NULL COMMENT '选客服无空闲时冷却截止时间'")
	// service_sessions: 用户进入选择客服页时间（以拉取可选客服列表为准）
	DB.Exec("ALTER TABLE `service_sessions` ADD COLUMN `staff_select_entered_at` datetime(3) NULL COMMENT '用户进入选择客服页时间（以拉取可选客服列表为准）'")
	// service_sessions: 待开始服务超时秒数（0表示使用系统默认）
	DB.Exec("ALTER TABLE `service_sessions` ADD COLUMN `start_pending_timeout_seconds` int NOT NULL DEFAULT 0 COMMENT '待开始服务超时秒数（0表示使用系统默认）'")
	// service_sessions: 最后一次叫号/分配的工作人员ID（用于过号等待等保留展示）
	DB.Exec("ALTER TABLE `service_sessions` ADD COLUMN `last_technician_id` bigint unsigned NULL DEFAULT NULL COMMENT '最后一次叫号/分配的工作人员ID（用于过号等待等保留展示）'")
	DB.Exec("ALTER TABLE `service_sessions` ADD INDEX `idx_service_sessions_last_technician_id` (`last_technician_id`)")
	// service_sessions: 服务单绑定的多个服务人员
	DB.Exec("ALTER TABLE `service_sessions` ADD COLUMN `service_technician_ids` JSON NULL COMMENT '服务单绑定的服务人员ID列表（专业客服）'")
	DB.Exec("UPDATE `service_sessions` SET `service_technician_ids` = CASE WHEN `technician_id` IS NULL THEN JSON_ARRAY() ELSE JSON_ARRAY(`technician_id`) END WHERE `service_technician_ids` IS NULL")
	DB.Exec("ALTER TABLE `service_sessions` MODIFY COLUMN `service_technician_ids` JSON NOT NULL COMMENT '服务单绑定的服务人员ID列表（专业客服）'")
	// service_sessions: 会话模式（用于客服/叫号模式隔离）
	DB.Exec("ALTER TABLE `service_sessions` ADD COLUMN `session_mode` varchar(20) NOT NULL DEFAULT '' COMMENT '会话模式'")
	// service_sessions: 会话来源（到店/预约）
	DB.Exec("ALTER TABLE `service_sessions` ADD COLUMN `source_type` varchar(20) NOT NULL DEFAULT 'walk_in' COMMENT '会话来源（walk_in/appointment）'")
	DB.Exec("ALTER TABLE `service_sessions` ADD COLUMN `source_id` bigint unsigned NULL DEFAULT NULL COMMENT '来源业务ID（如appointment_id）'")
	DB.Exec("ALTER TABLE `service_sessions` ADD INDEX `idx_service_sessions_source_id` (`source_id`)")

	// appointments: 关联卡片ID（用于同商户不同卡预约隔离）
	DB.Exec("ALTER TABLE `appointments` ADD COLUMN `card_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '卡片ID（外键关联cards表）'")
	DB.Exec("ALTER TABLE `appointments` ADD INDEX `idx_appointments_card_id` (`card_id`)")
	// 回填历史预约：优先按预约时间匹配“预约日前最近的一张卡”，避免同商户多卡时挂错卡
	DB.Exec("UPDATE appointments a SET a.card_id = (SELECT c.id FROM cards c WHERE c.user_id = a.user_id AND c.merchant_id = a.merchant_id AND c.recharge_at IS NOT NULL AND a.appointment_time IS NOT NULL AND c.recharge_at <= DATE(a.appointment_time) ORDER BY c.recharge_at DESC, c.id DESC LIMIT 1) WHERE a.card_id = 0")
	// 仍未回填的（如 appointment_time 为空），回退到最早的一张卡
	DB.Exec("UPDATE appointments a SET a.card_id = (SELECT c.id FROM cards c WHERE c.user_id = a.user_id AND c.merchant_id = a.merchant_id ORDER BY c.id ASC LIMIT 1) WHERE a.card_id = 0")

	// 添加 support_order_complete 字段到 merchants 表
	DB.Exec("ALTER TABLE `merchants` ADD COLUMN `support_order_complete` BOOLEAN DEFAULT FALSE COMMENT '是否开启服务结束功能（0-不开启，1-开启）'")
	// merchants: 是否开启多个客服（多窗口叫号）
	DB.Exec("ALTER TABLE `merchants` ADD COLUMN `support_multi_customer_service` BOOLEAN DEFAULT FALSE COMMENT '是否开启多个客服（多窗口叫号）'")
	// merchants: 等待上号时间（秒）
	DB.Exec("ALTER TABLE `merchants` ADD COLUMN `queue_waiting_start_seconds` INT NOT NULL DEFAULT 180 COMMENT '等待上号时间（秒）'")
	// merchants: 超时过号等待时间（秒）
	DB.Exec("ALTER TABLE `merchants` ADD COLUMN `queue_timeout_waiting_seconds` INT NOT NULL DEFAULT 900 COMMENT '超时过号等待时间（秒）'")
	// merchants: 核销后开始服务延迟秒数（未开启客服但开启服务结束功能时使用）
	DB.Exec("ALTER TABLE `merchants` ADD COLUMN `start_delay_seconds` int NOT NULL DEFAULT 60 COMMENT '核销后开始服务延迟秒数（未开启客服但开启服务结束功能时使用）'")
	// merchant_projects: 项目级服务开始延迟时间（秒）
	DB.Exec("ALTER TABLE `merchant_projects` ADD COLUMN `start_delay_seconds` int NOT NULL DEFAULT 60 COMMENT '服务开始延迟时间（秒）'")
	// merchant_projects: 项目级自动分配房间延迟时间（秒）
	DB.Exec("ALTER TABLE `merchant_projects` ADD COLUMN `room_select_timeout_seconds` int NOT NULL DEFAULT 90 COMMENT '自动分配房间延迟时间（秒）'")
	// merchant_projects: 项目级待开始服务倒计时（秒）
	DB.Exec("ALTER TABLE `merchant_projects` ADD COLUMN `start_pending_timeout_seconds` int NOT NULL DEFAULT 300 COMMENT '待开始服务倒计时秒数'")
	// merchant_projects: 项目级服务人数与固定服务时间
	DB.Exec("ALTER TABLE `merchant_projects` ADD COLUMN `service_capacity` int NOT NULL DEFAULT 1 COMMENT '服务人数'")
	DB.Exec("ALTER TABLE `merchant_projects` ADD COLUMN `show_participants` BOOLEAN NOT NULL DEFAULT TRUE COMMENT '用户端是否展示参与服务用户'")
	DB.Exec("ALTER TABLE `merchant_projects` ADD COLUMN `service_time_slots` JSON NULL COMMENT '服务时间槽'")
	DB.Exec("UPDATE `merchant_projects` SET `service_time_slots` = JSON_ARRAY() WHERE `service_time_slots` IS NULL")
	DB.Exec("ALTER TABLE `merchant_projects` MODIFY COLUMN `service_time_slots` JSON NOT NULL COMMENT '服务时间槽'")
	DB.Exec("ALTER TABLE `merchant_projects` ADD COLUMN `default_service_technician_ids` JSON NULL COMMENT '多人项目默认服务人员ID列表（专业客服）'")
	DB.Exec("UPDATE `merchant_projects` SET `default_service_technician_ids` = JSON_ARRAY() WHERE `default_service_technician_ids` IS NULL")
	DB.Exec("ALTER TABLE `merchant_projects` MODIFY COLUMN `default_service_technician_ids` JSON NOT NULL COMMENT '多人项目默认服务人员ID列表（专业客服）'")
	// merchant_projects: 项目级自动分配客服延迟时间（分钟）
	DB.Exec("ALTER TABLE `merchant_projects` ADD COLUMN `auto_assign_technician_delay_minutes` int NOT NULL DEFAULT 5 COMMENT '自动分配客服延迟时间（分钟）'")

	// cards: 锁卡状态（手牌未归还等）
	DB.Exec("ALTER TABLE `cards` ADD COLUMN `locked` BOOLEAN NOT NULL DEFAULT FALSE COMMENT '卡片是否锁定（手牌未归还等）'")
	DB.Exec("ALTER TABLE `cards` ADD COLUMN `locked_reason` varchar(255) NOT NULL DEFAULT '' COMMENT '锁卡原因'")
	DB.Exec("ALTER TABLE `cards` ADD COLUMN `locked_at` datetime(3) NULL COMMENT '锁卡时间'")
	DB.Exec("ALTER TABLE `cards` ADD COLUMN `locked_by` bigint unsigned NULL COMMENT '锁卡操作人（0/NULL 表示系统）'")
	DB.Exec("ALTER TABLE `cards` ADD COLUMN `unlocked_at` datetime(3) NULL COMMENT '解锁时间'")
	DB.Exec("ALTER TABLE `cards` ADD COLUMN `unlocked_by` bigint unsigned NULL COMMENT '解锁操作人（0/NULL 表示系统）'")
	DB.Exec("ALTER TABLE `cards` ADD COLUMN `unlocked_reason` varchar(255) NOT NULL DEFAULT '' COMMENT '解锁原因'")
	DB.Exec("ALTER TABLE `cards` ADD INDEX `idx_cards_locked` (`locked`)")

	// usages: 手牌绑定与归还
	DB.Exec("ALTER TABLE `usages` ADD COLUMN `hand_card_no` varchar(20) NULL DEFAULT NULL COMMENT '手牌号（商户核销后输入绑定）'")
	// 兼容已存在字段：将空字符串改为 NULL（未分配）
	DB.Exec("UPDATE `usages` SET `hand_card_no` = NULL WHERE `hand_card_no` = ''")
	DB.Exec("ALTER TABLE `usages` MODIFY COLUMN `hand_card_no` varchar(20) NULL DEFAULT NULL COMMENT '手牌号（商户核销后输入绑定）'")
	DB.Exec("ALTER TABLE `usages` ADD COLUMN `hand_card_assigned_at` datetime(3) NULL COMMENT '手牌分配时间'")
	DB.Exec("ALTER TABLE `usages` ADD COLUMN `hand_card_returned_at` datetime(3) NULL COMMENT '手牌归还时间'")
	// 生成列：仅在“已分配且未归还”时为 1，否则为 NULL（用于实现部分唯一约束）
	DB.Exec("ALTER TABLE `usages` ADD COLUMN `hand_card_active` tinyint GENERATED ALWAYS AS (IF(hand_card_no IS NOT NULL AND hand_card_returned_at IS NULL, 1, NULL)) STORED COMMENT '手牌占用标记（1-占用；NULL-不占用）'")
	DB.Exec("ALTER TABLE `usages` DROP INDEX `idx_usages_hand_card_no`")
	DB.Exec("ALTER TABLE `usages` ADD INDEX `idx_usages_hand_card_no` (`hand_card_no`)")
	// 同一商户同一手牌号：在“未归还”(hand_card_returned_at 为 NULL)期间不允许重复；归还后允许复用
	DB.Exec("ALTER TABLE `usages` DROP INDEX `uidx_usages_merchant_hand_card_no`")
	DB.Exec("ALTER TABLE `usages` DROP INDEX `uidx_usages_merchant_hand_card_active`")
	DB.Exec("ALTER TABLE `usages` ADD UNIQUE INDEX `uidx_usages_merchant_hand_card_active` (`merchant_id`, `hand_card_no`, `hand_card_active`)")
	DB.Exec("ALTER TABLE `usages` ADD INDEX `idx_usages_hand_card_returned_at` (`hand_card_returned_at`)")

	log.Println("数据库初始化成功")

	// 初始化商户注册邀请码（幂等）
	initInviteCodes()
	initServiceRoles()
	cleanupServiceRoles()
	initPermissions()
	initProfessionalBasePermissionConfig()
	initRolePermissions()

	// 初始化测试数据
	initTestData()
}

func migrateLegacyMerchantProjects() {
	if !DB.Migrator().HasColumn(&models.Merchant{}, "projects") {
		return
	}

	type row struct {
		ID       uint
		Projects string
	}
	var rows []row
	DB.Raw("SELECT id, projects FROM merchants WHERE projects IS NOT NULL AND projects <> '' AND projects <> '[]'").Scan(&rows)
	if len(rows) == 0 {
		return
	}

	type legacyProject struct {
		Name     string `json:"name"`
		Duration int    `json:"duration"`
	}

	for _, r := range rows {
		var cnt int64
		DB.Model(&models.MerchantProject{}).Where("merchant_id = ?", r.ID).Count(&cnt)
		if cnt > 0 {
			continue
		}
		var arr []legacyProject
		if err := json.Unmarshal([]byte(r.Projects), &arr); err != nil {
			continue
		}
		for i, p := range arr {
			name := p.Name
			if name == "" || p.Duration <= 0 {
				continue
			}
			mp := models.MerchantProject{
				MerchantID: r.ID,
				Name:       name,
				Duration:   p.Duration,
				Price:      0,
				IsActive:   true,
				SortOrder:  i,
			}
			DB.Create(&mp)
		}
	}
}

func dropForeignKeyIfExists(table, constraint string) {
	if DB == nil || !DB.Migrator().HasConstraint(table, constraint) {
		return
	}
	if err := DB.Migrator().DropConstraint(table, constraint); err != nil {
		log.Printf("删除外键失败 %s.%s: %v", table, constraint, err)
	}
}

func initServiceRoles() {
	defaults := []models.ServiceRole{
		{Key: "store_manager", Name: "店长", AccountPrefix: "sm", RoleType: "operational", Description: "运营客服-店长", IsActive: true, AllowPermissionAdjust: true, Sort: 5},
		{Key: "front_desk", Name: "前台", AccountPrefix: "fd", RoleType: "operational", Description: "运营客服-前台", IsActive: true, AllowPermissionAdjust: true, Sort: 6},
	}

	for _, r := range defaults {
		var existing models.ServiceRole
		if err := DB.Where("`key` = ?", r.Key).First(&existing).Error; err == nil {
			updates := map[string]interface{}{}
			if existing.Name == "" {
				updates["name"] = r.Name
			}
			if strings.TrimSpace(existing.RoleType) == "" && strings.TrimSpace(r.RoleType) != "" {
				updates["role_type"] = r.RoleType
			}
			if existing.AccountPrefix == "" && r.AccountPrefix != "" {
				updates["account_prefix"] = r.AccountPrefix
			}
			if !existing.AllowPermissionAdjust && r.AllowPermissionAdjust {
				updates["allow_permission_adjust"] = true
			}
			if existing.Description == "" {
				updates["description"] = r.Description
			}
			if existing.Sort == 0 {
				updates["sort"] = r.Sort
			}
			// 历史默认专业岗位：更新为非激活，避免出现在岗位下拉
			if r.IsActive == false && existing.IsActive == true {
				updates["is_active"] = false
			}
			if len(updates) > 0 {
				DB.Model(&models.ServiceRole{}).Where("id = ?", existing.ID).Updates(updates)
			}
			continue
		}
		DB.Create(&r)
	}
}

var merchantRoleKeyWithSuffixPattern = regexp.MustCompile(`^m(\d+)_([a-z]{1,5})_\d+$`)

func cleanupServiceRoles() {
	normalizeMerchantServiceRoleKeys()
	purgeLegacyPlatformServiceRoles()
}

func normalizeMerchantServiceRoleKeys() {
	var roles []models.ServiceRole
	DB.Where("merchant_id IS NOT NULL").Order("id asc").Find(&roles)
	if len(roles) == 0 {
		return
	}

	targets := make(map[uint]string, len(roles))
	used := map[string]uint{}
	conflicts := map[string][]uint{}
	for _, role := range roles {
		if role.MerchantID == nil {
			continue
		}
		prefix := strings.ToLower(strings.TrimSpace(role.AccountPrefix))
		if prefix == "" {
			log.Fatalf("商户岗位缺少账号前缀，无法规范化 key，role_id=%d", role.ID)
		}
		targetKey := BuildMerchantServiceRoleKey(*role.MerchantID, prefix)
		if ownerID, ok := used[targetKey]; ok && ownerID != role.ID {
			conflicts[targetKey] = appendUniqueUint(conflicts[targetKey], ownerID, role.ID)
			continue
		}
		used[targetKey] = role.ID
		targets[role.ID] = targetKey
	}
	if len(conflicts) > 0 {
		for targetKey, roleIDs := range conflicts {
			log.Printf("WARN: 商户岗位 key 冲突，已跳过规范化，需人工清理 target_key=%s conflict_role_ids=%v", targetKey, roleIDs)
		}
	}

	for _, role := range roles {
		targetKey, ok := targets[role.ID]
		if !ok || strings.TrimSpace(role.Key) == targetKey {
			continue
		}
		currentKey := strings.TrimSpace(role.Key)
		if currentKey == "" || merchantRoleKeyWithSuffixPattern.MatchString(currentKey) || strings.HasPrefix(currentKey, "m") {
			if err := DB.Model(&models.ServiceRole{}).Where("id = ?", role.ID).Update("key", targetKey).Error; err != nil {
				log.Fatalf("商户岗位 key 规范化失败，role_id=%d target_key=%s err=%v", role.ID, targetKey, err)
			}
		}
	}
}

func appendUniqueUint(items []uint, values ...uint) []uint {
	seen := make(map[uint]struct{}, len(items))
	for _, item := range items {
		seen[item] = struct{}{}
	}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

func purgeLegacyPlatformServiceRoles() {
	var roles []models.ServiceRole
	DB.Where("merchant_id IS NULL AND `key` NOT IN ?", FixedServiceRoleKeys()).Find(&roles)
	if len(roles) == 0 {
		return
	}

	roleIDs := make([]uint, 0, len(roles))
	for _, role := range roles {
		roleIDs = append(roleIDs, role.ID)
	}

	if err := DB.Transaction(func(tx *gorm.DB) error {
		var technicianIDs []uint
		if err := tx.Model(&models.Technician{}).Where("service_role_id IN ?", roleIDs).Pluck("id", &technicianIDs).Error; err != nil {
			return err
		}
		if len(technicianIDs) > 0 {
			if err := tx.Where("technician_id IN ?", technicianIDs).Delete(&models.TechnicianAttendance{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("service_role_id IN ?", roleIDs).Delete(&models.Technician{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_role_id IN ?", roleIDs).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_role_id IN ?", roleIDs).Delete(&models.MerchantRolePermissionOverride{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_role_id IN ?", roleIDs).Delete(&models.MerchantRoleAttendanceConfig{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_role_id IN ?", roleIDs).Delete(&models.MerchantRoleStartPendingConfig{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", roleIDs).Delete(&models.ServiceRole{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Fatalf("清理历史平台客服角色失败: %v", err)
	}
}

func initInviteCodes() {
	defaultCodes := []string{
		"KABAO-20260101-001",
		"KABAO-20260101-002",
		"KABAO-20260101-003",
		"KABAO-20260101-004",
		"KABAO-20260101-005",
	}

	for _, code := range defaultCodes {
		var existing models.InviteCode
		if err := DB.Where("code = ?", code).First(&existing).Error; err == nil {
			continue
		}
		DB.Create(&models.InviteCode{Code: code, Used: false})
	}
}

func initTestData() {
	// 检查是否已有数据
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	// 创建测试用户
	p1 := "13800138001"
	p2 := "13800138002"
	p3 := "13800138003"
	users := []models.User{
		{Username: "u1", Phone: &p1, Nickname: "朱迪亚"},
		{Username: "u2", Phone: &p2, Nickname: "u1"},
		{Username: "u3", Phone: &p3, Nickname: "u2"},
	}
	DB.Create(&users)

	// 创建测试商户
	merchants := []models.Merchant{
		{Name: "快剪理发店", Type: "理发", SupportAppointment: true},
		{Name: "顺风洗车", Type: "洗车", SupportAppointment: false},
	}
	DB.Create(&merchants)

	// 创建测试卡片
	parseDate := func(s string) *time.Time {
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			return nil
		}
		return &t
	}
	parseDateTime := func(s string) *time.Time {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
		if err != nil {
			return nil
		}
		return &t
	}
	cards := []models.Card{
		{
			UserID:         1,
			MerchantID:     1,
			CardNo:         "C1",
			CardType:       "10次专业剪发卡",
			TotalTimes:     10,
			RemainTimes:    8,
			UsedTimes:      2,
			RechargeAmount: 200,
			RechargeAt:     parseDate("2023-10-01"),
			StartDate:      parseDate("2023-10-01"),
			EndDate:        parseDate("2030-01-01"),
		},
		{
			UserID:         1,
			MerchantID:     2,
			CardNo:         "C2",
			CardType:       "精洗美容次卡",
			TotalTimes:     10,
			RemainTimes:    4,
			UsedTimes:      6,
			RechargeAmount: 300,
			RechargeAt:     parseDate("2023-06-01"),
			StartDate:      parseDate("2023-06-01"),
			EndDate:        parseDate("2030-06-01"),
		},
		{
			UserID:         1,
			MerchantID:     1,
			CardNo:         "C3",
			CardType:       "10次洗头卡",
			TotalTimes:     10,
			RemainTimes:    7,
			UsedTimes:      3,
			RechargeAmount: 200,
			RechargeAt:     parseDate("2023-10-01"),
			LastUsedAt:     parseDateTime("2023-11-20 22:30:00"),
			StartDate:      parseDate("2023-10-01"),
			EndDate:        parseDate("2024-10-01"),
		},
	}
	DB.Create(&cards)

	// 创建测试使用记录
	usages := []models.Usage{
		{CardID: 3, MerchantID: 1, UsedTimes: 1, UsedAt: parseDateTime("2023-11-20 22:30:00"), Status: "success"},
		{CardID: 3, MerchantID: 1, UsedTimes: 1, UsedAt: parseDateTime("2023-11-15 17:30:00"), Status: "success"},
	}
	DB.Create(&usages)

	// 创建测试通知
	notices := []models.Notice{
		{MerchantID: 1, Title: "元旦休息通知", Content: "本店将于1月1日至1月3日放假休息，敬请谅解..."},
		{MerchantID: 1, Title: "会员升级活动", Content: "新老客户充值享8折优惠，仅限本周。"},
	}
	DB.Create(&notices)

	// 创建测试预约
	appointments := []models.Appointment{
		{MerchantID: 1, UserID: 2, AppointmentTime: parseDateTime("2024-01-05 18:00:00"), Status: "pending"},
		{MerchantID: 1, UserID: 3, AppointmentTime: parseDateTime("2024-01-05 17:00:00"), Status: "confirmed"},
		{MerchantID: 1, UserID: 1, AppointmentTime: parseDateTime("2024-01-05 18:00:00"), Status: "pending"},
	}
	DB.Create(&appointments)

	log.Println("测试数据初始化完成")
}

func initPermissions() {
	defaults := []models.Permission{
		// 商户管理 (10-19)
		{Key: "merchant.info.manage", Name: "商户信息设置", Group: "商户管理", Description: "更新商户基本信息", Sort: 10},
		{Key: "merchant.service.manage", Name: "商户服务设置", Group: "商户管理", Description: "更新商户服务能力开关", Sort: 11},
		{Key: "merchant.business_status.manage", Name: "营业状态管理", Group: "商户管理", Description: "操作营业中/打烊", Sort: 12},

		// 通知管理 (20-29)
		{Key: "merchant.notice.manage", Name: "通知管理", Group: "通知管理", Description: "创建、删除、置顶/取消置顶通知", Sort: 20},

		// 售卡管理 (30-39)
		{Key: "merchant.direct_sale.manage", Name: "售卡管理", Group: "售卡管理", Description: "管理直购售卡配置、模板、订单等", Sort: 30},

		// 卡片管理 (40-59)
		{Key: "merchant.card.issue", Name: "发卡/开卡", Group: "卡片管理", Description: "创建卡片、发卡", Sort: 40},
		{Key: "merchant.card.verify", Name: "核销", Group: "卡片管理", Description: "核销会员卡", Sort: 41},
		{Key: "merchant.card.verify_finish", Name: "核销即结束服务", Group: "卡片管理", Description: "核销后自动结束服务（不再需要二次扫码结束服务）", Sort: 42},
		{Key: "merchant.card.finish", Name: "结束服务", Group: "卡片管理", Description: "技师扫码结束服务，将进行中核销置为完成", Sort: 43},
		{Key: "merchant.card.sell", Name: "售卡", Group: "卡片管理", Description: "技师售卡：查询售卡模板、生成售卡二维码", Sort: 44},
		{Key: "merchant.card.unlock", Name: "解锁卡片", Group: "卡片管理", Description: "解锁被锁定的卡片（手牌未归还等）", Sort: 45},

		// 客服管理 (60-69)
		{Key: "merchant.cs.manage", Name: "客服管理", Group: "客服管理", Description: "新增/编辑/禁用/删除客服账号", Sort: 60},
		{Key: "merchant.table.view", Name: "查看看板", Group: "看板", Description: "查看看板页面与相关数据", Sort: 61},
		{Key: "merchant.room.view", Name: "查看房间", Group: "看板", Description: "查看房间使用/空闲情况", Sort: 62},
		{Key: "merchant.staff.view", Name: "查看客服", Group: "看板", Description: "查看专业客服签到/空闲/服务状态", Sort: 63},

		// 预约管理 (70-89)
		{Key: "merchant.appointment.view", Name: "预约查看", Group: "预约管理", Description: "可被用户预约、查看和处理预约自己的预约", Sort: 70},
		{Key: "merchant.appointment.manage", Name: "预约管理", Group: "预约管理", Description: "确认/完成整个店铺的预约", Sort: 71},

		// 叫号管理 (80-89)
		{Key: "merchant.queue.calling", Name: "叫号", Group: "叫号管理", Description: "启动/暂停商户或自己的叫号服务", Sort: 80},

		// 权限管理 (90-99)
		{Key: "merchant.permission.adjust", Name: "权限微调", Group: "权限管理", Description: "调整角色权限覆盖", Sort: 90},
	}

	for _, p := range defaults {
		var existing models.Permission
		if err := DB.Where("`key` = ?", p.Key).First(&existing).Error; err == nil {
			updates := map[string]interface{}{}
			if existing.Name == "" {
				updates["name"] = p.Name
			}
			if existing.Group == "" {
				updates["group"] = p.Group
			}
			if existing.Description == "" {
				updates["description"] = p.Description
			}
			// 强制更新 sort 值以匹配最新定义
			if existing.Sort != p.Sort {
				updates["sort"] = p.Sort
			}
			if len(updates) > 0 {
				DB.Model(&models.Permission{}).Where("id = ?", existing.ID).Updates(updates)
			}
			continue
		}
		DB.Create(&p)
	}
}

func initProfessionalBasePermissionConfig() {
	const configKey = "professional_base_permission_keys"
	const defaultValue = "merchant.card.finish,merchant.card.sell,merchant.card.verify"

	var sc models.SystemConfig
	if err := DB.Where("`key` = ?", configKey).First(&sc).Error; err == nil {
		return
	}
	DB.Create(&models.SystemConfig{Key: configKey, Value: defaultValue})
}

func initRolePermissions() {
	ensure := func(roleKey string, permKey string) {
		var role models.ServiceRole
		if err := DB.Where("`key` = ?", roleKey).First(&role).Error; err != nil {
			return
		}
		var perm models.Permission
		if err := DB.Where("`key` = ?", permKey).First(&perm).Error; err != nil {
			return
		}
		var existing models.RolePermission
		if err := DB.Where("service_role_id = ? AND permission_id = ?", role.ID, perm.ID).First(&existing).Error; err == nil {
			return
		}
		DB.Create(&models.RolePermission{ServiceRoleID: role.ID, PermissionID: perm.ID, Allowed: true})
	}

	// 店长：除收款配置/店铺短链（merchant.direct_sale.manage）与商户地址信息设置（merchant.info.manage）之外，默认全开
	var perms []models.Permission
	DB.Find(&perms)
	for _, p := range perms {
		if strings.TrimSpace(p.Key) == "" {
			continue
		}
		if p.Key == "merchant.direct_sale.manage" || p.Key == "merchant.info.manage" {
			continue
		}
		ensure("store_manager", p.Key)
	}

	// 前台：营业状态、通知、核销、售卡管理、售卡、发卡、预约管理、叫号
	frontDeskPerms := []string{
		"merchant.business_status.manage",
		"merchant.notice.manage",
		"merchant.card.verify",
		"merchant.direct_sale.manage",
		"merchant.card.sell",
		"merchant.card.issue",
		"merchant.appointment.manage",
		"merchant.room.view",
		"merchant.staff.view",
		"merchant.queue.calling",
	}
	for _, pk := range frontDeskPerms {
		ensure("front_desk", pk)
	}
}

func addFieldComments() {
	// service_roles 表字段注释
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `key` varchar(50) NOT NULL COMMENT '客服类型标识（如：technician、teacher）'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `name` varchar(50) NOT NULL COMMENT '客服类型名称（如：技师、老师）'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `account_prefix` varchar(5) NOT NULL DEFAULT '' COMMENT '账号前缀（最多5个英文字母）'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `require_attendance` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否需要签到（0-不需要，1-需要）'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `description` varchar(255) DEFAULT '' COMMENT '客服类型描述'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `is_active` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用（0-禁用，1-启用）'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `allow_permission_adjust` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否允许权限调整（0-不允许，1-允许）'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `sort` int NOT NULL DEFAULT 0 COMMENT '排序顺序（越小越靠前）'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间'")
	DB.Exec("ALTER TABLE `service_roles` MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间'")

	// permissions 表字段注释
	DB.Exec("ALTER TABLE `permissions` MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID'")
	DB.Exec("ALTER TABLE `permissions` MODIFY COLUMN `key` varchar(80) NOT NULL COMMENT '权限标识（如：merchant.card.verify）'")
	DB.Exec("ALTER TABLE `permissions` MODIFY COLUMN `name` varchar(80) NOT NULL COMMENT '权限名称（如：核销）'")
	DB.Exec("ALTER TABLE `permissions` MODIFY COLUMN `group` varchar(80) DEFAULT '' COMMENT '权限分组（如：卡片、商户）'")
	DB.Exec("ALTER TABLE `permissions` MODIFY COLUMN `description` varchar(255) DEFAULT '' COMMENT '权限描述'")
	DB.Exec("ALTER TABLE `permissions` MODIFY COLUMN `sort` int NOT NULL DEFAULT 0 COMMENT '排序顺序（越小越靠前）'")
	DB.Exec("ALTER TABLE `permissions` MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间'")
	DB.Exec("ALTER TABLE `permissions` MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间'")

	// role_permissions 表字段注释
	DB.Exec("ALTER TABLE `role_permissions` MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID'")
	DB.Exec("ALTER TABLE `role_permissions` MODIFY COLUMN `service_role_id` int unsigned NOT NULL COMMENT '客服类型ID（外键关联service_roles表）'")
	DB.Exec("ALTER TABLE `role_permissions` MODIFY COLUMN `permission_id` int unsigned NOT NULL COMMENT '权限ID（外键关联permissions表）'")
	DB.Exec("ALTER TABLE `role_permissions` MODIFY COLUMN `allowed` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否允许权限（0-不允许，1-允许）'")
	DB.Exec("ALTER TABLE `role_permissions` MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间'")
	DB.Exec("ALTER TABLE `role_permissions` MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '更新时间'")

	// merchant_role_permission_overrides 表字段注释
	DB.Exec("ALTER TABLE `merchant_role_permission_overrides` MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID'")
	DB.Exec("ALTER TABLE `merchant_role_permission_overrides` MODIFY COLUMN `merchant_id` int unsigned NOT NULL COMMENT '商户ID（外键关联merchants表）'")
	DB.Exec("ALTER TABLE `merchant_role_permission_overrides` MODIFY COLUMN `service_role_id` int unsigned NOT NULL COMMENT '客服类型ID（外键关联service_roles表）'")
	DB.Exec("ALTER TABLE `merchant_role_permission_overrides` MODIFY COLUMN `permission_id` int unsigned NOT NULL COMMENT '权限ID（外键关联permissions表）'")
	DB.Exec("ALTER TABLE `merchant_role_permission_overrides` MODIFY COLUMN `allowed` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否允许权限（0-不允许，1-允许）'")
	DB.Exec("ALTER TABLE `merchant_role_permission_overrides` MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间'")
	DB.Exec("ALTER TABLE `merchant_role_permission_overrides` MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '更新时间'")
}
