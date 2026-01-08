-- =============================================================================
-- role_permissions 和 service_roles 表注释检查和补充SQL脚本
-- 执行前请备份数据库，建议在测试环境先验证
-- =============================================================================

-- 1. service_roles 表注释（如果表注释不存在）
ALTER TABLE `service_roles` COMMENT = '平台客服类型表';

-- 2. role_permissions 表注释（添加表注释）
ALTER TABLE `role_permissions` COMMENT = '客服类型权限关联表';

-- =============================================================================
-- 验证当前字段注释是否完整
-- =============================================================================

-- 检查 service_roles 表字段注释
SELECT 
    TABLE_NAME,
    COLUMN_NAME,
    COLUMN_COMMENT,
    COLUMN_TYPE,
    IS_NULLABLE,
    COLUMN_DEFAULT
FROM 
    INFORMATION_SCHEMA.COLUMNS 
WHERE 
    TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME = 'service_roles'
ORDER BY 
    ORDINAL_POSITION;

-- 检查 role_permissions 表字段注释
SELECT 
    TABLE_NAME,
    COLUMN_NAME,
    COLUMN_COMMENT,
    COLUMN_TYPE,
    IS_NULLABLE,
    COLUMN_DEFAULT
FROM 
    INFORMATION_SCHEMA.COLUMNS 
WHERE 
    TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME = 'role_permissions'
ORDER BY 
    ORDINAL_POSITION;

-- =============================================================================
-- 字段注释完整性检查（如果需要重新添加）
-- =============================================================================

-- service_roles 表字段注释（确保完整性）
ALTER TABLE `service_roles` 
MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
MODIFY COLUMN `key` varchar(50) NOT NULL COMMENT '客服类型标识（如：technician、teacher）',
MODIFY COLUMN `name` varchar(50) NOT NULL COMMENT '客服类型名称（如：技师、老师）',
MODIFY COLUMN `description` varchar(255) DEFAULT '' COMMENT '客服类型描述',
MODIFY COLUMN `is_active` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用（0-禁用，1-启用）',
MODIFY COLUMN `allow_permission_adjust` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否允许权限调整（0-不允许，1-允许）',
MODIFY COLUMN `sort` int NOT NULL DEFAULT 0 COMMENT '排序顺序（越小越靠前）',
MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间';

-- role_permissions 表字段注释（确保完整性）
ALTER TABLE `role_permissions` 
MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
MODIFY COLUMN `service_role_id` int unsigned NOT NULL COMMENT '客服类型ID（外键关联service_roles表）',
MODIFY COLUMN `permission_id` int unsigned NOT NULL COMMENT '权限ID（外键关联permissions表）',
MODIFY COLUMN `allowed` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否允许权限（0-不允许，1-允许）',
MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间';

-- =============================================================================
-- 表注释验证
-- =============================================================================

-- 检查表注释
SELECT 
    TABLE_NAME AS '表名',
    TABLE_COMMENT AS '表注释'
FROM 
    INFORMATION_SCHEMA.TABLES 
WHERE 
    TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME IN ('service_roles', 'role_permissions')
ORDER BY 
    TABLE_NAME;

-- =============================================================================
-- 完整性报告
-- =============================================================================

-- 显示所有缺少注释的字段
SELECT 
    TABLE_NAME,
    COLUMN_NAME,
    COLUMN_TYPE,
    '缺少注释' AS '状态'
FROM 
    INFORMATION_SCHEMA.COLUMNS 
WHERE 
    TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME IN ('service_roles', 'role_permissions')
    AND (COLUMN_COMMENT IS NULL OR COLUMN_COMMENT = '')
ORDER BY 
    TABLE_NAME, ORDINAL_POSITION;

-- =============================================================================
-- 注意事项：
-- 1. 执行前请确保数据库名称正确
-- 2. 建议在业务低峰期执行
-- 3. 执行前请备份数据库
-- 4. 如果字段注释已存在，MODIFY语句会更新但不会报错
-- =============================================================================
