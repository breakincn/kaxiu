-- 为 merchants 表添加营业状态字段
ALTER TABLE merchants
ADD COLUMN is_open TINYINT(1) DEFAULT 1 COMMENT '营业状态（0-打烊，1-营业中）' AFTER address;
