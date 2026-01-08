-- 权限迁移脚本
-- 1. 合并通知权限：将 merchant.notice.create 和 merchant.notice.delete 合并到 merchant.notice.manage
-- 2. 添加新的预约权限：merchant.appointment.view

-- 查找相关权限ID
SET @notice_manage_id = (SELECT id FROM permissions WHERE `key` = 'merchant.notice.manage');
SET @notice_create_id = (SELECT id FROM permissions WHERE `key` = 'merchant.notice.create');
SET @notice_delete_id = (SELECT id FROM permissions WHERE `key` = 'merchant.notice.delete');
SET @appointment_manage_id = (SELECT id FROM permissions WHERE `key` = 'merchant.appointment.manage');
SET @appointment_view_id = (SELECT id FROM permissions WHERE `key` = 'merchant.appointment.view');

-- 1. 迁移角色权限：如果角色有 create 或 delete 权限，则给予 manage 权限
INSERT INTO role_permissions (service_role_id, permission_id, allowed, created_at, updated_at)
SELECT DISTINCT service_role_id, @notice_manage_id, TRUE, NOW(), NOW()
FROM role_permissions
WHERE permission_id IN (@notice_create_id, @notice_delete_id) 
  AND allowed = TRUE
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp2 
    WHERE rp2.service_role_id = role_permissions.service_role_id 
      AND rp2.permission_id = @notice_manage_id
  );

-- 2. 迁移商户级权限覆盖：如果有 create 或 delete 权限，则给予 manage 权限
INSERT INTO merchant_role_permission_overrides (merchant_id, service_role_id, permission_id, allowed, created_at, updated_at)
SELECT DISTINCT merchant_id, service_role_id, @notice_manage_id, TRUE, NOW(), NOW()
FROM merchant_role_permission_overrides
WHERE permission_id IN (@notice_create_id, @notice_delete_id)
  AND allowed = TRUE
  AND NOT EXISTS (
    SELECT 1 FROM merchant_role_permission_overrides mrpo2
    WHERE mrpo2.merchant_id = merchant_role_permission_overrides.merchant_id
      AND mrpo2.service_role_id = merchant_role_permission_overrides.service_role_id
      AND mrpo2.permission_id = @notice_manage_id
  );

-- 3. 删除旧的通知权限记录
DELETE FROM role_permissions WHERE permission_id IN (@notice_create_id, @notice_delete_id);
DELETE FROM merchant_role_permission_overrides WHERE permission_id IN (@notice_create_id, @notice_delete_id);

-- 4. 删除旧的权限定义
DELETE FROM permissions WHERE `key` IN ('merchant.notice.create', 'merchant.notice.delete');

-- 5. 迁移预约权限：将现有的 appointment.manage 权限复制为 appointment.view
-- 这样有管理预约权限的角色也会获得预约权限
INSERT INTO role_permissions (service_role_id, permission_id, allowed, created_at, updated_at)
SELECT service_role_id, @appointment_view_id, allowed, NOW(), NOW()
FROM role_permissions
WHERE permission_id = @appointment_manage_id
  AND @appointment_view_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp2
    WHERE rp2.service_role_id = role_permissions.service_role_id
      AND rp2.permission_id = @appointment_view_id
  );

-- 6. 迁移商户级预约权限覆盖
INSERT INTO merchant_role_permission_overrides (merchant_id, service_role_id, permission_id, allowed, created_at, updated_at)
SELECT merchant_id, service_role_id, @appointment_view_id, allowed, NOW(), NOW()
FROM merchant_role_permission_overrides
WHERE permission_id = @appointment_manage_id
  AND @appointment_view_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM merchant_role_permission_overrides mrpo2
    WHERE mrpo2.merchant_id = merchant_role_permission_overrides.merchant_id
      AND mrpo2.service_role_id = merchant_role_permission_overrides.service_role_id
      AND mrpo2.permission_id = @appointment_view_id
  );

-- 完成！
SELECT '权限迁移完成' AS message;
