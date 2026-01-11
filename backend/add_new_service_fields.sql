-- 添加新的服务字段到merchants表
-- 支持重复执行

-- 添加 support_project 字段（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'support_project') > 0,
    'SELECT "support_project column already exists";',
    'ALTER TABLE merchants ADD COLUMN support_project BOOLEAN DEFAULT FALSE COMMENT "是否开启项目服务（0-不开启，1-开启）";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加 support_order_complete 字段（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'support_order_complete') > 0,
    'SELECT "support_order_complete column already exists";',
    'ALTER TABLE merchants ADD COLUMN support_order_complete BOOLEAN DEFAULT FALSE COMMENT "是否开启结单功能（0-不开启，1-开启）";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加 support_hand_card 字段（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'support_hand_card') > 0,
    'SELECT "support_hand_card column already exists";',
    'ALTER TABLE merchants ADD COLUMN support_hand_card BOOLEAN DEFAULT FALSE COMMENT "是否开启发手牌功能（0-不开启，1-开启）";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加 support_room_number_card 字段（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'support_room_number_card') > 0,
    'SELECT "support_room_number_card column already exists";',
    'ALTER TABLE merchants ADD COLUMN support_room_number_card BOOLEAN DEFAULT FALSE COMMENT "是否开启发房间号牌功能（0-不开启，1-开启）";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加项目设置字段（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'projects') > 0,
    'SELECT "projects column already exists";',
    'ALTER TABLE merchants ADD COLUMN projects JSON COMMENT "项目设置（JSON格式）";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加手牌设置字段（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'hand_card_prefix') > 0,
    'SELECT "hand_card_prefix column already exists";',
    'ALTER TABLE merchants ADD COLUMN hand_card_prefix VARCHAR(20) DEFAULT "H" COMMENT "手牌前缀";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'hand_card_start_no') > 0,
    'SELECT "hand_card_start_no column already exists";',
    'ALTER TABLE merchants ADD COLUMN hand_card_start_no INT DEFAULT 1 COMMENT "手牌起始号码";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'hand_card_end_no') > 0,
    'SELECT "hand_card_end_no column already exists";',
    'ALTER TABLE merchants ADD COLUMN hand_card_end_no INT DEFAULT 100 COMMENT "手牌结束号码";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加房间号牌设置字段（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'room_number_card_prefix') > 0,
    'SELECT "room_number_card_prefix column already exists";',
    'ALTER TABLE merchants ADD COLUMN room_number_card_prefix VARCHAR(20) DEFAULT "R" COMMENT "房间号牌前缀";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'room_number_card_start_no') > 0,
    'SELECT "room_number_card_start_no column already exists";',
    'ALTER TABLE merchants ADD COLUMN room_number_card_start_no INT DEFAULT 1 COMMENT "房间号牌起始号码";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'merchants' 
     AND COLUMN_NAME = 'room_number_card_end_no') > 0,
    'SELECT "room_number_card_end_no column already exists";',
    'ALTER TABLE merchants ADD COLUMN room_number_card_end_no INT DEFAULT 50 COMMENT "房间号牌结束号码";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
