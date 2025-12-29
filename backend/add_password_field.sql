-- 给 users 表添加 password 字段
ALTER TABLE `users` ADD COLUMN `password` VARCHAR(255) NOT NULL DEFAULT '' AFTER `phone`;

-- 为张三设置密码 (密码: 123456)
-- bcrypt 加密后的密码
UPDATE `users` SET `password` = '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' WHERE `phone` = '13800138001';

-- 为李四设置密码 (密码: 123456)
UPDATE `users` SET `password` = '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' WHERE `phone` = '13800138002';
