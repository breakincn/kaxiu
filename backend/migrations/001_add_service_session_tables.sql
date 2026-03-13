-- 新增服务会话相关表与字段迁移脚本
-- 适用于存量环境升级，请在业务低峰期执行

-- 1) 商户表增加房间功能开关（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'support_room') > 0,
    'SELECT "support_room column already exists";',
    'ALTER TABLE merchants ADD COLUMN support_room BOOLEAN DEFAULT FALSE COMMENT "是否启用房间功能";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 1.1) 商户表增加工作人员签到开关（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'support_technician_checkin') > 0,
    'SELECT "support_technician_checkin column already exists";',
    'ALTER TABLE merchants ADD COLUMN support_technician_checkin BOOLEAN DEFAULT FALSE COMMENT "是否启用工作人员签到";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2) 创建房间表
CREATE TABLE IF NOT EXISTS rooms (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    merchant_id BIGINT UNSIGNED NOT NULL COMMENT '商户ID',
    name VARCHAR(100) NOT NULL COMMENT '房间名称/编号',
    enabled BOOLEAN NOT NULL DEFAULT TRUE COMMENT '是否启用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_merchant (merchant_id),
    UNIQUE KEY uk_merchant_name (merchant_id, name)
) COMMENT='房间配置表';

-- 3) 创建工作人员签到状态表
CREATE TABLE IF NOT EXISTS technician_attendances (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    merchant_id BIGINT UNSIGNED NOT NULL COMMENT '商户ID',
    technician_id BIGINT UNSIGNED NOT NULL COMMENT '技师ID',
    checked_in_at TIMESTAMP NOT NULL COMMENT '签到时间',
    status ENUM('available','idle','busy') NOT NULL DEFAULT 'available' COMMENT '服务状态',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_merchant_technician (merchant_id, technician_id),
    INDEX idx_checked_in_at (checked_in_at),
    INDEX idx_status (status)
) COMMENT='工作人员签到状态表';

-- 4) 创建服务会话表
CREATE TABLE IF NOT EXISTS service_sessions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    merchant_id BIGINT UNSIGNED NOT NULL COMMENT '商户ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    card_id BIGINT UNSIGNED NOT NULL COMMENT '卡片ID',
    initial_usage_id BIGINT UNSIGNED NOT NULL COMMENT '核销产生的使用记录ID',
    verify_code VARCHAR(32) NOT NULL COMMENT '核销码（用于关联）',
    status ENUM('room_selecting','staff_selecting','room_locked','start_pending','delay_pending','serving','auto_finishing','finished','canceled') NOT NULL DEFAULT 'room_selecting' COMMENT '会话状态',
    room_id BIGINT UNSIGNED NULL COMMENT '分配的房间ID',
    technician_id BIGINT UNSIGNED NULL COMMENT '分配的技师ID',
    room_select_deadline_at TIMESTAMP NULL COMMENT '选房截止时间（核销后90秒内）',
    start_confirmed_at TIMESTAMP NULL COMMENT '开始服务确认时间',
    start_delay_seconds INT NOT NULL DEFAULT 60 COMMENT '开始服务可延迟秒数',
    scheduled_start_at TIMESTAMP NULL COMMENT '计划开始计时时间',
    duration_minutes INT NOT NULL DEFAULT 50 COMMENT '服务总时长（分钟）',
    auto_finish_delay_seconds INT NOT NULL DEFAULT 60 COMMENT '自动结束延迟秒数（服务结束后延迟自动结束）',
    auto_idle_after_seconds INT NOT NULL DEFAULT 180 COMMENT '服务结束后技师置为空闲的延迟秒数',
    started_at TIMESTAMP NULL COMMENT '服务开始时间',
    scheduled_finish_at TIMESTAMP NULL COMMENT '预计结束时间（用于调度）',
    finished_at TIMESTAMP NULL COMMENT '实际结束时间',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_merchant (merchant_id),
    INDEX idx_user (user_id),
    INDEX idx_card (card_id),
    INDEX idx_status (status),
    INDEX idx_technician (technician_id),
    INDEX idx_room (room_id),
    INDEX idx_room_select_deadline (room_select_deadline_at),
    INDEX idx_scheduled_finish (scheduled_finish_at),
    INDEX idx_finished_at (finished_at)
) COMMENT='服务会话表';

-- 5) 为存量 technicians 表增加索引（若不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'technicians' 
     AND INDEX_NAME = 'idx_merchant') > 0,
    'SELECT "idx_merchant index already exists on technicians";',
    'ALTER TABLE technicians ADD INDEX idx_merchant (merchant_id);'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 6) 为存量 usages 表增加索引（若不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'usages' 
     AND INDEX_NAME = 'idx_card_id') > 0,
    'SELECT "idx_card_id index already exists on usages";',
    'ALTER TABLE usages ADD INDEX idx_card_id (card_id);'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'usages' 
     AND INDEX_NAME = 'idx_merchant_id') > 0,
    'SELECT "idx_merchant_id index already exists on usages";',
    'ALTER TABLE usages ADD INDEX idx_merchant_id (merchant_id);'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'usages' 
     AND INDEX_NAME = 'idx_used_at') > 0,
    'SELECT "idx_used_at index already exists on usages";',
    'ALTER TABLE usages ADD INDEX idx_used_at (used_at);'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 7) 为存量 cards 表增加索引（若不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'cards' 
     AND INDEX_NAME = 'idx_user_id') > 0,
    'SELECT "idx_user_id index already exists on cards";',
    'ALTER TABLE cards ADD INDEX idx_user_id (user_id);'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'cards' 
     AND INDEX_NAME = 'idx_merchant_id') > 0,
    'SELECT "idx_merchant_id index already exists on cards";',
    'ALTER TABLE cards ADD INDEX idx_merchant_id (merchant_id);'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 8) 为存量 merchants 表增加索引（若不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND INDEX_NAME = 'idx_avg_service_minutes') > 0,
    'SELECT "idx_avg_service_minutes index already exists on merchants";',
    'ALTER TABLE merchants ADD INDEX idx_avg_service_minutes (avg_service_minutes);'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 9) 可选：初始化示例房间（仅演示，生产环境请自行配置）
-- INSERT IGNORE INTO rooms (merchant_id, name, enabled) 
-- SELECT id, CONCAT('房间', (ROW_NUMBER() OVER (PARTITION BY id ORDER BY id))), TRUE 
-- FROM merchants WHERE support_room = TRUE LIMIT 10;

-- 10) 可选：为存量商户开启房间功能（谨慎操作，请按业务需求决定）
-- UPDATE merchants SET support_room = TRUE WHERE id IN (1,2,3);

-- 执行后请检查：
-- SELECT COUNT(*) AS rooms FROM rooms;
-- SELECT COUNT(*) AS attendances FROM technician_attendances;
-- SELECT COUNT(*) AS sessions FROM service_sessions;
-- SHOW INDEX FROM rooms;
-- SHOW INDEX FROM technician_attendances;
-- SHOW INDEX FROM service_sessions;
