-- 修改张三的卡片开卡日期为 2025-07-15
UPDATE cards 
SET start_date = '2025-07-15', recharge_at = '2025-07-15'
WHERE user_id = (SELECT id FROM users WHERE nickname = '张三')
LIMIT 1;

-- 获取张三卡片的ID并记录
SET @card_id = (SELECT id FROM cards WHERE user_id = (SELECT id FROM users WHERE nickname = '张三') LIMIT 1);
SET @merchant_id = (SELECT merchant_id FROM cards WHERE id = @card_id);

-- 创建 3 条使用记录
INSERT INTO usages (card_id, merchant_id, used_times, used_at, status) VALUES
(@card_id, @merchant_id, 1, '2025-07-20 10:30:00', 'success'),
(@card_id, @merchant_id, 1, '2025-08-10 14:00:00', 'success'),
(@card_id, @merchant_id, 1, '2025-09-15 16:30:00', 'success');

-- 更新卡片的使用次数统计
UPDATE cards 
SET used_times = 3, remain_times = total_times - 3, last_used_at = '2025-09-15 16:30:00'
WHERE id = @card_id;

-- 验证修改结果
SELECT '=== 张三的卡片信息 ===' AS '';
SELECT c.*, u.nickname FROM cards c 
JOIN users u ON c.user_id = u.id 
WHERE u.nickname = '张三';

SELECT '=== 张三的使用记录 ===' AS '';
SELECT * FROM usages 
WHERE card_id = @card_id
ORDER BY used_at DESC;
