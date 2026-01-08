-- 安全的权限键和名称迁移脚本
-- 执行前请备份数据库！

-- 1. 先检查并处理重复键问题
-- 如果新键已存在，先删除或重命名它

-- 处理 merchant.worker.manage
DELETE FROM role_permissions WHERE permission_id = (
    SELECT id FROM permissions WHERE `key` = 'merchant.worker.manage'
);

DELETE FROM merchant_role_permission_overrides WHERE permission_id = (
    SELECT id FROM permissions WHERE `key` = 'merchant.worker.manage'
);

DELETE FROM permissions WHERE `key` = 'merchant.worker.manage';

-- 处理 merchant.info.manage
DELETE FROM role_permissions WHERE permission_id = (
    SELECT id FROM permissions WHERE `key` = 'merchant.info.manage'
);

DELETE FROM merchant_role_permission_overrides WHERE permission_id = (
    SELECT id FROM permissions WHERE `key` = 'merchant.info.manage'
);

DELETE FROM permissions WHERE `key` = 'merchant.info.manage';

-- 处理 merchant.service.manage
DELETE FROM role_permissions WHERE permission_id = (
    SELECT id FROM permissions WHERE `key` = 'merchant.service.manage'
);

DELETE FROM merchant_role_permission_overrides WHERE permission_id = (
    SELECT id FROM permissions WHERE `key` = 'merchant.service.manage'
);

DELETE FROM permissions WHERE `key` = 'merchant.service.manage';

-- 处理 merchant.business_status.manage
DELETE FROM role_permissions WHERE permission_id = (
    SELECT id FROM permissions WHERE `key` = 'merchant.business_status.manage'
);

DELETE FROM merchant_role_permission_overrides WHERE permission_id = (
    SELECT id FROM permissions WHERE `key` = 'merchant.business_status.manage'
);

DELETE FROM permissions WHERE `key` = 'merchant.business_status.manage';

-- 2. 现在安全地更新权限键和名称
UPDATE permissions SET 
    `key` = 'merchant.worker.manage',
    name = '客服管理'
WHERE `key` = 'merchant.tech.manage';

UPDATE permissions SET 
    `key` = 'merchant.info.manage',
    name = '商户信息设置'
WHERE `key` = 'merchant.merchant.update';

UPDATE permissions SET 
    `key` = 'merchant.service.manage',
    name = '商户服务设置'
WHERE `key` = 'merchant.service.update';

UPDATE permissions SET 
    `key` = 'merchant.business_status.manage',
    name = '营业状态管理'
WHERE `key` = 'merchant.business_status.update';

-- 3. 验证更新结果
SELECT '权限迁移完成' AS message;

SELECT 
    `key`,
    name,
    `group`,
    description,
    sort
FROM permissions 
WHERE `key` IN (
    'merchant.worker.manage',
    'merchant.info.manage', 
    'merchant.service.manage',
    'merchant.business_status.manage'
)
ORDER BY sort;
