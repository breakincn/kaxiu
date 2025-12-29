-- 恢复 kabao 数据库各表字段注释
USE kabao;

-- =============================================
-- 1. users 表（用户表）
-- =============================================
ALTER TABLE `users` COMMENT = '用户表';
ALTER TABLE `users` MODIFY COLUMN `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID';
ALTER TABLE `users` MODIFY COLUMN `phone` VARCHAR(20) NOT NULL COMMENT '手机号';
ALTER TABLE `users` MODIFY COLUMN `password` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '登录密码（bcrypt加密）';
ALTER TABLE `users` MODIFY COLUMN `nickname` VARCHAR(50) DEFAULT NULL COMMENT '用户昵称';
ALTER TABLE `users` MODIFY COLUMN `created_at` VARCHAR(255) DEFAULT NULL COMMENT '创建时间';

-- =============================================
-- 2. merchants 表（商户表）
-- =============================================
ALTER TABLE `merchants` COMMENT = '商户表';
ALTER TABLE `merchants` MODIFY COLUMN `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '商户ID';
ALTER TABLE `merchants` MODIFY COLUMN `name` VARCHAR(100) DEFAULT NULL COMMENT '商户名称';
ALTER TABLE `merchants` MODIFY COLUMN `type` VARCHAR(50) DEFAULT NULL COMMENT '商户类型（如：理发、美容等）';
ALTER TABLE `merchants` MODIFY COLUMN `support_appointment` TINYINT(1) DEFAULT 0 COMMENT '是否支持预约（0-不支持，1-支持）';
ALTER TABLE `merchants` MODIFY COLUMN `avg_service_minutes` INT DEFAULT 30 COMMENT '平均服务时长（分钟）';
ALTER TABLE `merchants` MODIFY COLUMN `created_at` VARCHAR(255) DEFAULT NULL COMMENT '创建时间';

-- =============================================
-- 3. cards 表（卡片表/会员卡表）
-- =============================================
ALTER TABLE `cards` COMMENT = '用户会员卡表';
ALTER TABLE `cards` MODIFY COLUMN `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '卡片ID';
ALTER TABLE `cards` MODIFY COLUMN `user_id` INT UNSIGNED DEFAULT NULL COMMENT '用户ID（外键关联users表）';
ALTER TABLE `cards` MODIFY COLUMN `merchant_id` INT UNSIGNED DEFAULT NULL COMMENT '商户ID（外键关联merchants表）';
ALTER TABLE `cards` MODIFY COLUMN `card_no` VARCHAR(50) DEFAULT NULL COMMENT '卡号';
ALTER TABLE `cards` MODIFY COLUMN `card_type` VARCHAR(100) DEFAULT NULL COMMENT '卡片类型（如：洗剪吹10次卡）';
ALTER TABLE `cards` MODIFY COLUMN `total_times` INT DEFAULT NULL COMMENT '总次数';
ALTER TABLE `cards` MODIFY COLUMN `remain_times` INT DEFAULT NULL COMMENT '剩余次数';
ALTER TABLE `cards` MODIFY COLUMN `used_times` INT DEFAULT NULL COMMENT '已使用次数';
ALTER TABLE `cards` MODIFY COLUMN `recharge_amount` INT DEFAULT NULL COMMENT '充值金额（单位：元）';
ALTER TABLE `cards` MODIFY COLUMN `recharge_at` VARCHAR(255) DEFAULT NULL COMMENT '充值时间/开卡时间';
ALTER TABLE `cards` MODIFY COLUMN `last_used_at` VARCHAR(255) DEFAULT NULL COMMENT '最后使用时间';
ALTER TABLE `cards` MODIFY COLUMN `start_date` VARCHAR(255) DEFAULT NULL COMMENT '有效期开始日期';
ALTER TABLE `cards` MODIFY COLUMN `end_date` VARCHAR(255) DEFAULT NULL COMMENT '有效期结束日期';
ALTER TABLE `cards` MODIFY COLUMN `created_at` VARCHAR(255) DEFAULT NULL COMMENT '创建时间';

-- =============================================
-- 4. usages 表（使用记录表）
-- =============================================
ALTER TABLE `usages` COMMENT = '卡片使用记录表';
ALTER TABLE `usages` MODIFY COLUMN `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '记录ID';
ALTER TABLE `usages` MODIFY COLUMN `card_id` INT UNSIGNED DEFAULT NULL COMMENT '卡片ID（外键关联cards表）';
ALTER TABLE `usages` MODIFY COLUMN `merchant_id` INT UNSIGNED DEFAULT NULL COMMENT '商户ID（外键关联merchants表）';
ALTER TABLE `usages` MODIFY COLUMN `used_times` INT DEFAULT NULL COMMENT '本次核销次数';
ALTER TABLE `usages` MODIFY COLUMN `used_at` VARCHAR(255) DEFAULT NULL COMMENT '使用时间';
ALTER TABLE `usages` MODIFY COLUMN `status` VARCHAR(20) DEFAULT 'success' COMMENT '状态（success-成功，failed-失败）';
ALTER TABLE `usages` MODIFY COLUMN `created_at` VARCHAR(255) DEFAULT NULL COMMENT '创建时间';

-- =============================================
-- 5. notices 表（通知表）
-- =============================================
ALTER TABLE `notices` COMMENT = '商户通知表';
ALTER TABLE `notices` MODIFY COLUMN `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '通知ID';
ALTER TABLE `notices` MODIFY COLUMN `merchant_id` INT UNSIGNED DEFAULT NULL COMMENT '商户ID（外键关联merchants表）';
ALTER TABLE `notices` MODIFY COLUMN `title` VARCHAR(200) DEFAULT NULL COMMENT '通知标题';
ALTER TABLE `notices` MODIFY COLUMN `content` TEXT DEFAULT NULL COMMENT '通知内容';
ALTER TABLE `notices` MODIFY COLUMN `is_pinned` TINYINT(1) DEFAULT 0 COMMENT '是否置顶（0-否，1-是）';
ALTER TABLE `notices` MODIFY COLUMN `created_at` VARCHAR(255) DEFAULT NULL COMMENT '创建时间';

-- =============================================
-- 6. appointments 表（预约表）⭐ 用户预约排队核心表
-- =============================================
ALTER TABLE `appointments` COMMENT = '用户预约排队表';
ALTER TABLE `appointments` MODIFY COLUMN `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '预约ID';
ALTER TABLE `appointments` MODIFY COLUMN `merchant_id` INT UNSIGNED DEFAULT NULL COMMENT '商户ID（外键关联merchants表）';
ALTER TABLE `appointments` MODIFY COLUMN `user_id` INT UNSIGNED DEFAULT NULL COMMENT '用户ID（外键关联users表）';
ALTER TABLE `appointments` MODIFY COLUMN `appointment_time` VARCHAR(255) DEFAULT NULL COMMENT '预约时间';
ALTER TABLE `appointments` MODIFY COLUMN `status` VARCHAR(20) DEFAULT 'pending' COMMENT '预约状态（pending-待确认，confirmed-已确认/排队中，finished-已完成，canceled-已取消）';
ALTER TABLE `appointments` MODIFY COLUMN `created_at` VARCHAR(255) DEFAULT NULL COMMENT '创建时间';

-- =============================================
-- 7. verify_codes 表（核销码表）
-- =============================================
ALTER TABLE `verify_codes` COMMENT = '核销码表';
ALTER TABLE `verify_codes` MODIFY COLUMN `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '核销码ID';
ALTER TABLE `verify_codes` MODIFY COLUMN `card_id` INT UNSIGNED DEFAULT NULL COMMENT '卡片ID（外键关联cards表）';
ALTER TABLE `verify_codes` MODIFY COLUMN `code` VARCHAR(50) DEFAULT NULL COMMENT '核销码';
ALTER TABLE `verify_codes` MODIFY COLUMN `expire_at` BIGINT DEFAULT NULL COMMENT '过期时间（Unix时间戳）';
ALTER TABLE `verify_codes` MODIFY COLUMN `used` TINYINT(1) DEFAULT 0 COMMENT '是否已使用（0-未使用，1-已使用）';
ALTER TABLE `verify_codes` MODIFY COLUMN `created_at` VARCHAR(255) DEFAULT NULL COMMENT '创建时间';

-- =============================================
-- 8. 查看表结构和注释
-- =============================================
SELECT 
    TABLE_NAME AS '表名',
    TABLE_COMMENT AS '表注释'
FROM 
    information_schema.TABLES 
WHERE 
    TABLE_SCHEMA = 'kabao'
ORDER BY 
    TABLE_NAME;
