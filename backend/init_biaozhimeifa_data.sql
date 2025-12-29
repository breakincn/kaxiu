-- 标致美发店完整数据初始化脚本
-- 数据库: kabao

USE kabao;

-- 1. 创建商户：标致美发店
INSERT INTO merchants (name, type, support_appointment, avg_service_minutes, created_at) 
VALUES ('标致美发店', '理发', 1, 40, NOW());

-- 获取刚创建的商户ID（假设是第一个商户，ID为1，如果已有数据会自增）
SET @merchant_id = LAST_INSERT_ID();

-- 2. 创建测试用户（用于开卡）
INSERT INTO users (phone, nickname, created_at) VALUES 
('13800138001', '张三', NOW()),
('13800138002', '李四', NOW()),
('13800138003', '王五', NOW());

-- 获取用户ID
SET @user1_id = (SELECT id FROM users WHERE phone = '13800138001');
SET @user2_id = (SELECT id FROM users WHERE phone = '13800138002');
SET @user3_id = (SELECT id FROM users WHERE phone = '13800138003');

-- 3. 为用户创建卡片
-- 张三的卡片 - 洗剪吹10次卡
INSERT INTO cards (user_id, merchant_id, card_no, card_type, total_times, remain_times, used_times, 
                   recharge_amount, recharge_at, start_date, end_date, created_at)
VALUES (@user1_id, @merchant_id, 'BZ202412290001', '洗剪吹10次卡', 10, 10, 0, 
        298, NOW(), CURDATE(), DATE_ADD(CURDATE(), INTERVAL 365 DAY), NOW());

-- 李四的卡片 - 烫染套餐5次卡（已使用2次）
INSERT INTO cards (user_id, merchant_id, card_no, card_type, total_times, remain_times, used_times, 
                   recharge_amount, recharge_at, last_used_at, start_date, end_date, created_at)
VALUES (@user2_id, @merchant_id, 'BZ202412290002', '烫染套餐5次卡', 5, 3, 2, 
        988, DATE_SUB(NOW(), INTERVAL 30 DAY), DATE_SUB(NOW(), INTERVAL 7 DAY), 
        DATE_SUB(CURDATE(), INTERVAL 30 DAY), DATE_ADD(CURDATE(), INTERVAL 335 DAY), 
        DATE_SUB(NOW(), INTERVAL 30 DAY));

-- 王五的卡片 - 洗护20次卡（已使用5次）
INSERT INTO cards (user_id, merchant_id, card_no, card_type, total_times, remain_times, used_times, 
                   recharge_amount, recharge_at, last_used_at, start_date, end_date, created_at)
VALUES (@user3_id, @merchant_id, 'BZ202412290003', '洗护20次卡', 20, 15, 5, 
        568, DATE_SUB(NOW(), INTERVAL 60 DAY), DATE_SUB(NOW(), INTERVAL 3 DAY),
        DATE_SUB(CURDATE(), INTERVAL 60 DAY), DATE_ADD(CURDATE(), INTERVAL 305 DAY), 
        DATE_SUB(NOW(), INTERVAL 60 DAY));

-- 获取卡片ID
SET @card2_id = (SELECT id FROM cards WHERE card_no = 'BZ202412290002');
SET @card3_id = (SELECT id FROM cards WHERE card_no = 'BZ202412290003');

-- 4. 创建使用记录
-- 李四的使用记录（2次）
INSERT INTO usages (card_id, merchant_id, used_times, used_at, status, created_at) VALUES
(@card2_id, @merchant_id, 1, DATE_SUB(NOW(), INTERVAL 15 DAY), 'success', DATE_SUB(NOW(), INTERVAL 15 DAY)),
(@card2_id, @merchant_id, 1, DATE_SUB(NOW(), INTERVAL 7 DAY), 'success', DATE_SUB(NOW(), INTERVAL 7 DAY));

-- 王五的使用记录（5次）
INSERT INTO usages (card_id, merchant_id, used_times, used_at, status, created_at) VALUES
(@card3_id, @merchant_id, 1, DATE_SUB(NOW(), INTERVAL 50 DAY), 'success', DATE_SUB(NOW(), INTERVAL 50 DAY)),
(@card3_id, @merchant_id, 1, DATE_SUB(NOW(), INTERVAL 35 DAY), 'success', DATE_SUB(NOW(), INTERVAL 35 DAY)),
(@card3_id, @merchant_id, 1, DATE_SUB(NOW(), INTERVAL 20 DAY), 'success', DATE_SUB(NOW(), INTERVAL 20 DAY)),
(@card3_id, @merchant_id, 1, DATE_SUB(NOW(), INTERVAL 10 DAY), 'success', DATE_SUB(NOW(), INTERVAL 10 DAY)),
(@card3_id, @merchant_id, 1, DATE_SUB(NOW(), INTERVAL 3 DAY), 'success', DATE_SUB(NOW(), INTERVAL 3 DAY));

-- 5. 创建商户通知
INSERT INTO notices (merchant_id, title, content, created_at) VALUES
(@merchant_id, '新年优惠活动', '即日起至春节前，充值任意套餐卡赠送护发精油一瓶！', NOW()),
(@merchant_id, '营业时间调整', '本店自12月30日起营业时间调整为：周一至周日 09:00-21:00', DATE_SUB(NOW(), INTERVAL 2 DAY)),
(@merchant_id, '技师推荐', '新晋高级发型师小王已到店，擅长日韩风格造型，欢迎预约体验！', DATE_SUB(NOW(), INTERVAL 7 DAY));

-- 6. 创建预约记录
-- 今日预约
INSERT INTO appointments (merchant_id, user_id, appointment_time, status, created_at) VALUES
(@merchant_id, @user1_id, DATE_FORMAT(DATE_ADD(NOW(), INTERVAL 2 HOUR), '%Y-%m-%d %H:00:00'), 'confirmed', NOW()),
(@merchant_id, @user2_id, DATE_FORMAT(DATE_ADD(NOW(), INTERVAL 3 HOUR), '%Y-%m-%d %H:00:00'), 'confirmed', NOW());

-- 明日预约
INSERT INTO appointments (merchant_id, user_id, appointment_time, status, created_at) VALUES
(@merchant_id, @user3_id, DATE_FORMAT(DATE_ADD(NOW(), INTERVAL 1 DAY), '%Y-%m-%d 14:00:00'), 'pending', NOW());

-- 查询结果验证
SELECT '=== 商户信息 ===' AS '';
SELECT * FROM merchants WHERE id = @merchant_id;

SELECT '=== 用户信息 ===' AS '';
SELECT * FROM users WHERE phone LIKE '13800138%';

SELECT '=== 卡片信息 ===' AS '';
SELECT c.*, u.nickname, u.phone FROM cards c 
JOIN users u ON c.user_id = u.id 
WHERE c.merchant_id = @merchant_id;

SELECT '=== 使用记录 ===' AS '';
SELECT u.*, c.card_no FROM usages u
JOIN cards c ON u.card_id = c.id
WHERE u.merchant_id = @merchant_id;

SELECT '=== 商户通知 ===' AS '';
SELECT * FROM notices WHERE merchant_id = @merchant_id;

SELECT '=== 预约记录 ===' AS '';
SELECT a.*, u.nickname FROM appointments a
JOIN users u ON a.user_id = u.id
WHERE a.merchant_id = @merchant_id;

SELECT '=== 数据初始化完成 ===' AS '';
