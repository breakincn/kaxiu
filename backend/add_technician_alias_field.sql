-- 添加技师自定义称谓字段
-- 执行前请备份数据库！

ALTER TABLE merchants 
ADD COLUMN technician_alias VARCHAR(20) DEFAULT '技师' 
COMMENT '技师自定义称谓（如：小二、服务员等）';

-- 验证字段添加成功
DESCRIBE merchants;

SELECT '字段添加完成' AS message;
