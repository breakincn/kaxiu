-- 卡包项目：时间字段统一迁移脚本
-- 目标：将所有时间语义字段从 string/longtext 迁移到 DATE/DATETIME(3)
-- 执行前请备份数据库！
-- 在测试环境先执行一遍！

START TRANSACTION;

-- 0) 清理可能存在的临时列（防止重复执行导致的冲突）
SET @sql = IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
   WHERE TABLE_SCHEMA = 'kabao' AND TABLE_NAME = 'cards' AND COLUMN_NAME = 'recharge_at_new') > 0,
  'ALTER TABLE cards DROP COLUMN recharge_at_new, DROP COLUMN start_date_new, DROP COLUMN end_date_new, DROP COLUMN last_used_at_new',
  'SELECT "No temp columns to drop in cards"'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
   WHERE TABLE_SCHEMA = 'kabao' AND TABLE_NAME = 'usages' AND COLUMN_NAME = 'used_at_new') > 0,
  'ALTER TABLE usages DROP COLUMN used_at_new',
  'SELECT "No temp columns to drop in usages"'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
   WHERE TABLE_SCHEMA = 'kabao' AND TABLE_NAME = 'appointments' AND COLUMN_NAME = 'appointment_time_new') > 0,
  'ALTER TABLE appointments DROP COLUMN appointment_time_new',
  'SELECT "No temp columns to drop in appointments"'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
   WHERE TABLE_SCHEMA = 'kabao' AND TABLE_NAME = 'users' AND COLUMN_NAME = 'created_at_new') > 0,
  'ALTER TABLE users DROP COLUMN created_at_new',
  'SELECT "No temp columns to drop in users"'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
   WHERE TABLE_SCHEMA = 'kabao' AND TABLE_NAME = 'merchants' AND COLUMN_NAME = 'created_at_new') > 0,
  'ALTER TABLE merchants DROP COLUMN created_at_new',
  'SELECT "No temp columns to drop in merchants"'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
   WHERE TABLE_SCHEMA = 'kabao' AND TABLE_NAME = 'notices' AND COLUMN_NAME = 'created_at_new') > 0,
  'ALTER TABLE notices DROP COLUMN created_at_new',
  'SELECT "No temp columns to drop in notices"'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
   WHERE TABLE_SCHEMA = 'kabao' AND TABLE_NAME = 'verify_codes' AND COLUMN_NAME = 'created_at_new') > 0,
  'ALTER TABLE verify_codes DROP COLUMN created_at_new',
  'SELECT "No temp columns to drop in verify_codes"'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 1) cards 表迁移
-- 1.1 新增临时列
ALTER TABLE cards
  ADD COLUMN recharge_at_new DATE NULL,
  ADD COLUMN start_date_new  DATE NULL,
  ADD COLUMN end_date_new    DATE NULL,
  ADD COLUMN last_used_at_new DATETIME(3) NULL;

-- 1.2 转换存量数据
UPDATE cards
SET
  recharge_at_new = CASE
    WHEN NULLIF(TRIM(recharge_at), '') IS NULL THEN NULL
    WHEN recharge_at REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}$'
      THEN DATE(STR_TO_DATE(recharge_at, '%Y-%m-%d %H:%i:%s'))
    ELSE STR_TO_DATE(recharge_at, '%Y-%m-%d')
  END,
  start_date_new  = CASE
    WHEN NULLIF(TRIM(start_date), '') IS NULL THEN NULL
    WHEN start_date REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}$'
      THEN DATE(STR_TO_DATE(start_date, '%Y-%m-%d %H:%i:%s'))
    ELSE STR_TO_DATE(start_date, '%Y-%m-%d')
  END,
  end_date_new    = CASE
    WHEN NULLIF(TRIM(end_date), '') IS NULL THEN NULL
    WHEN end_date REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}$'
      THEN DATE(STR_TO_DATE(end_date, '%Y-%m-%d %H:%i:%s'))
    ELSE STR_TO_DATE(end_date, '%Y-%m-%d')
  END,
  last_used_at_new = CASE
    WHEN NULLIF(TRIM(last_used_at), '') IS NULL THEN NULL
    WHEN last_used_at REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
      THEN STR_TO_DATE(last_used_at, '%Y-%m-%d')
    ELSE STR_TO_DATE(last_used_at, '%Y-%m-%d %H:%i:%s')
  END;

-- 1.3 替换列
ALTER TABLE cards
  DROP COLUMN recharge_at,
  DROP COLUMN start_date,
  DROP COLUMN end_date,
  DROP COLUMN last_used_at;

ALTER TABLE cards
  CHANGE COLUMN recharge_at_new recharge_at DATE NULL COMMENT '充值时间/开卡时间',
  CHANGE COLUMN start_date_new  start_date  DATE NULL COMMENT '有效期开始日期',
  CHANGE COLUMN end_date_new    end_date    DATE NULL COMMENT '有效期结束日期',
  CHANGE COLUMN last_used_at_new last_used_at DATETIME(3) NULL COMMENT '最后使用时间';

-- 2) usages 表迁移
-- 2.1 新增临时列
ALTER TABLE usages
  ADD COLUMN used_at_new DATETIME(3) NULL;

-- 2.2 转换存量数据
UPDATE usages
SET used_at_new = STR_TO_DATE(NULLIF(TRIM(used_at), ''), '%Y-%m-%d %H:%i:%s');

-- 2.3 替换列
ALTER TABLE usages
  DROP COLUMN used_at;

ALTER TABLE usages
  CHANGE COLUMN used_at_new used_at DATETIME(3) NULL COMMENT '使用时间';

-- 3) appointments 表迁移
-- 3.1 新增临时列
ALTER TABLE appointments
  ADD COLUMN appointment_time_new DATETIME(3) NULL;

-- 3.2 转换存量数据
UPDATE appointments
SET appointment_time_new = STR_TO_DATE(NULLIF(TRIM(appointment_time), ''), '%Y-%m-%d %H:%i:%s');

-- 3.3 替换列
ALTER TABLE appointments
  DROP COLUMN appointment_time;

ALTER TABLE appointments
  CHANGE COLUMN appointment_time_new appointment_time DATETIME(3) NULL COMMENT '预约时间';

-- 4) 统一所有表的 created_at 为 DATETIME(3)
-- 4.1 users 表
ALTER TABLE users
  ADD COLUMN created_at_new DATETIME(3) NULL;

UPDATE users
SET created_at_new =
  CASE
    WHEN created_at IS NULL OR TRIM(created_at) = '' THEN NULL
    WHEN created_at REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
      THEN STR_TO_DATE(created_at, '%Y-%m-%d')
    ELSE STR_TO_DATE(created_at, '%Y-%m-%d %H:%i:%s')
  END;

ALTER TABLE users
  DROP COLUMN created_at;

ALTER TABLE users
  CHANGE COLUMN created_at_new created_at DATETIME(3) NULL COMMENT '创建时间';

-- 4.2 merchants 表
ALTER TABLE merchants
  ADD COLUMN created_at_new DATETIME(3) NULL;

UPDATE merchants
SET created_at_new =
  CASE
    WHEN created_at IS NULL OR TRIM(created_at) = '' THEN NULL
    WHEN created_at REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
      THEN STR_TO_DATE(created_at, '%Y-%m-%d')
    ELSE STR_TO_DATE(created_at, '%Y-%m-%d %H:%i:%s')
  END;

ALTER TABLE merchants
  DROP COLUMN created_at;

ALTER TABLE merchants
  CHANGE COLUMN created_at_new created_at DATETIME(3) NULL COMMENT '创建时间';

-- 4.3 notices 表
ALTER TABLE notices
  ADD COLUMN created_at_new DATETIME(3) NULL;

UPDATE notices
SET created_at_new =
  CASE
    WHEN created_at IS NULL OR TRIM(created_at) = '' THEN NULL
    WHEN created_at REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
      THEN STR_TO_DATE(created_at, '%Y-%m-%d')
    ELSE STR_TO_DATE(created_at, '%Y-%m-%d %H:%i:%s')
  END;

ALTER TABLE notices
  DROP COLUMN created_at;

ALTER TABLE notices
  CHANGE COLUMN created_at_new created_at DATETIME(3) NULL COMMENT '创建时间';

-- 4.4 verify_codes 表
ALTER TABLE verify_codes
  ADD COLUMN created_at_new DATETIME(3) NULL;

UPDATE verify_codes
SET created_at_new =
  CASE
    WHEN created_at IS NULL OR TRIM(created_at) = '' THEN NULL
    WHEN created_at REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
      THEN STR_TO_DATE(created_at, '%Y-%m-%d')
    ELSE STR_TO_DATE(created_at, '%Y-%m-%d %H:%i:%s')
  END;

ALTER TABLE verify_codes
  DROP COLUMN created_at;

ALTER TABLE verify_codes
  CHANGE COLUMN created_at_new created_at DATETIME(3) NULL COMMENT '创建时间';

-- 检查迁移结果
SELECT 
  'cards' as table_name,
  COUNT(*) as total_rows,
  COUNT(recharge_at) as recharge_at_not_null,
  COUNT(start_date) as start_date_not_null,
  COUNT(end_date) as end_date_not_null,
  COUNT(last_used_at) as last_used_at_not_null
FROM cards
UNION ALL
SELECT 
  'usages' as table_name,
  COUNT(*) as total_rows,
  COUNT(used_at) as used_at_not_null,
  0,0,0
FROM usages
UNION ALL
SELECT 
  'appointments' as table_name,
  COUNT(*) as total_rows,
  COUNT(appointment_time) as appointment_time_not_null,
  0,0,0
FROM appointments
UNION ALL
SELECT 
  'users' as table_name,
  COUNT(*) as total_rows,
  COUNT(created_at) as created_at_not_null,
  0,0,0
FROM users
UNION ALL
SELECT 
  'merchants' as table_name,
  COUNT(*) as total_rows,
  COUNT(created_at) as created_at_not_null,
  0,0,0
FROM merchants
UNION ALL
SELECT 
  'notices' as table_name,
  COUNT(*) as total_rows,
  COUNT(created_at) as created_at_not_null,
  0,0,0
FROM notices
UNION ALL
SELECT 
  'verify_codes' as table_name,
  COUNT(*) as total_rows,
  COUNT(created_at) as created_at_not_null,
  0,0,0
FROM verify_codes;

COMMIT;

-- 迁移完成！
-- 请重启后端服务让 GORM AutoMigrate 生效
