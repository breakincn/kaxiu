-- =============================================================================
-- 卡包系统数据库字段注释添加SQL脚本
-- 执行前请备份数据库，建议在测试环境先验证
-- =============================================================================

-- 1. service_roles 表字段注释
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

-- 2. permissions 表字段注释
ALTER TABLE `permissions` 
MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
MODIFY COLUMN `key` varchar(80) NOT NULL COMMENT '权限标识（如：merchant.card.verify）',
MODIFY COLUMN `name` varchar(80) NOT NULL COMMENT '权限名称（如：核销）',
MODIFY COLUMN `group` varchar(80) DEFAULT '' COMMENT '权限分组（如：卡片、商户）',
MODIFY COLUMN `description` varchar(255) DEFAULT '' COMMENT '权限描述',
MODIFY COLUMN `sort` int NOT NULL DEFAULT 0 COMMENT '排序顺序（越小越靠前）',
MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间';

-- 3. role_permissions 表字段注释
ALTER TABLE `role_permissions` 
MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
MODIFY COLUMN `service_role_id` int unsigned NOT NULL COMMENT '客服类型ID（外键关联service_roles表）',
MODIFY COLUMN `permission_id` int unsigned NOT NULL COMMENT '权限ID（外键关联permissions表）',
MODIFY COLUMN `allowed` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否允许权限（0-不允许，1-允许）',
MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间';

-- 4. merchant_role_permission_overrides 表字段注释
ALTER TABLE `merchant_role_permission_overrides` 
MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
MODIFY COLUMN `merchant_id` int unsigned NOT NULL COMMENT '商户ID（外键关联merchants表）',
MODIFY COLUMN `service_role_id` int unsigned NOT NULL COMMENT '客服类型ID（外键关联service_roles表）',
MODIFY COLUMN `permission_id` int unsigned NOT NULL COMMENT '权限ID（外键关联permissions表）',
MODIFY COLUMN `allowed` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否允许权限（0-不允许，1-允许）',
MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间';

-- 5. system_configs 表字段注释（如果表存在）
ALTER TABLE `system_configs` 
MODIFY COLUMN `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
MODIFY COLUMN `key` varchar(100) NOT NULL COMMENT '配置项键名',
MODIFY COLUMN `value` varchar(500) DEFAULT '' COMMENT '配置项值',
MODIFY COLUMN `created_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
MODIFY COLUMN `updated_at` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间';

-- ============================================================================= 
-- 执行验证查询（可选）
-- =============================================================================

-- 检查字段注释是否添加成功
SELECT 
    TABLE_NAME,
    COLUMN_NAME,
    COLUMN_COMMENT
FROM 
    INFORMATION_SCHEMA.COLUMNS 
WHERE 
    TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME IN ('service_roles', 'permissions', 'role_permissions', 'merchant_role_permission_overrides', 'system_configs')
    AND COLUMN_COMMENT != ''
ORDER BY 
    TABLE_NAME, ORDINAL_POSITION;

-- =============================================================================
-- 注意事项：
-- 1. 执行前请确保数据库名称正确
-- 2. 建议在业务低峰期执行
-- 3. 执行前请备份数据库
-- 4. 如果某些表不存在，对应的ALTER语句会报错，可以忽略
-- =============================================================================
