-- 新增：商户自定义术语（起单/结单）
-- start_term / finish_term 为空时，前端默认展示“起单/结单”

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'merchants'
       AND COLUMN_NAME = 'start_term') > 0,
    'SELECT "start_term column already exists";',
    'ALTER TABLE merchants ADD COLUMN start_term varchar(20) NOT NULL DEFAULT "" COMMENT "起单显示名词（可为空）";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'merchants'
       AND COLUMN_NAME = 'finish_term') > 0,
    'SELECT "finish_term column already exists";',
    'ALTER TABLE merchants ADD COLUMN finish_term varchar(20) NOT NULL DEFAULT "" COMMENT "结单显示名词（可为空）";'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
