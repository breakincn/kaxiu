-- 重构：预结单 -> 起单（字段与状态命名同步）
-- 开发阶段可直接执行；存量环境也尽量保持幂等

-- 1) service_sessions: precheck_at -> start_confirmed_at
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'service_sessions'
       AND COLUMN_NAME = 'start_confirmed_at') > 0,
    'SELECT "start_confirmed_at column already exists";',
    (SELECT IF(
        (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
         WHERE TABLE_SCHEMA = DATABASE()
           AND TABLE_NAME = 'service_sessions'
           AND COLUMN_NAME = 'precheck_at') > 0,
        'ALTER TABLE service_sessions CHANGE COLUMN precheck_at start_confirmed_at TIMESTAMP NULL COMMENT "起单确认时间";',
        'ALTER TABLE service_sessions ADD COLUMN start_confirmed_at TIMESTAMP NULL COMMENT "起单确认时间";'
    ))
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2) service_sessions: delay_seconds -> start_delay_seconds
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'service_sessions'
       AND COLUMN_NAME = 'start_delay_seconds') > 0,
    'SELECT "start_delay_seconds column already exists";',
    (SELECT IF(
        (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
         WHERE TABLE_SCHEMA = DATABASE()
           AND TABLE_NAME = 'service_sessions'
           AND COLUMN_NAME = 'delay_seconds') > 0,
        'ALTER TABLE service_sessions CHANGE COLUMN delay_seconds start_delay_seconds INT NOT NULL DEFAULT 60 COMMENT "起单可延迟秒数";',
        'ALTER TABLE service_sessions ADD COLUMN start_delay_seconds INT NOT NULL DEFAULT 60 COMMENT "起单可延迟秒数";'
    ))
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 3) service_sessions.status: precheck_pending -> start_pending
-- 注意：不同环境 ENUM 定义可能不同，这里尽量做数据层替换，ENUM 本身由后续结构迁移/建表时保证
UPDATE service_sessions SET status = 'start_pending' WHERE status = 'precheck_pending';

-- 4) 兼容：若存在旧字段名但新字段不存在，确保不丢数据
-- （已通过 CHANGE COLUMN 处理；这里不额外处理）
