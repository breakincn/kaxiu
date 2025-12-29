# 用户登录功能测试说明

## 数据库更新

首先需要执行以下 SQL 更新数据库：

```bash
cd /Users/will/Projects/Go/kabao/backend
mysql -uroot -p123456 kabao < add_password_field.sql
```

或者直接在 MySQL 客户端执行：

```sql
-- 给 users 表添加 password 字段
ALTER TABLE `users` ADD COLUMN `password` VARCHAR(255) NOT NULL DEFAULT '' AFTER `phone`;

-- 为张三设置密码 (密码: 123456)
UPDATE `users` SET `password` = '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' WHERE `phone` = '13800138001';

-- 为李四设置密码 (密码: 123456)
UPDATE `users` SET `password` = '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' WHERE `phone` = '13800138002';
```

## 测试账号

### 张三
- 手机号: `13800138001`
- 密码: `123456`
- 拥有卡片: 洗剪吹10次卡（剩余10次）

### 李四
- 手机号: `13800138002`
- 密码: `123456`
- 拥有卡片: 烫染套餐5次卡（剩余3次，已使用2次）

## 启动项目

### 后端启动
```bash
cd /Users/will/Projects/Go/kabao/backend
go mod tidy  # 安装依赖（包含 bcrypt）
go run main.go
```

### 前端启动
```bash
cd /Users/will/Projects/Go/kabao/frontend
npm install
npm run dev
```

## 测试流程

### 1. 用户登录
1. 访问 `http://localhost:3000`，会自动跳转到登录页
2. 使用张三的账号登录：
   - 手机号: `13800138001`
   - 密码: `123456`
3. 登录成功后会跳转到卡片列表页
4. 可以看到张三的卡片数据

### 2. 切换用户
1. 点击右上角的设置图标
2. 点击"退出登录"
3. 使用李四的账号登录：
   - 手机号: `13800138002`
   - 密码: `123456`
4. 登录成功后可以看到李四的卡片数据

### 3. 数据隔离验证
- 张三登录后只能看到自己的卡片（洗剪吹10次卡）
- 李四登录后只能看到自己的卡片（烫染套餐5次卡）
- 两个用户的数据完全隔离

## API 接口

### 登录接口
```
POST /api/login
Content-Type: application/json

{
  "phone": "13800138001",
  "password": "123456"
}

响应：
{
  "data": {
    "token": "user_1_1735466234",
    "user_id": 1,
    "phone": "13800138001",
    "nickname": "张三"
  }
}
```

### 获取当前用户信息
```
GET /api/me
Authorization: Bearer user_1_1735466234

响应：
{
  "data": {
    "id": 1,
    "phone": "13800138001",
    "nickname": "张三",
    "created_at": "2024-12-29 10:00:00"
  }
}
```

### 获取用户卡片
```
GET /api/users/1/cards?status=active
Authorization: Bearer user_1_1735466234

响应：
{
  "data": [
    {
      "id": 1,
      "card_type": "洗剪吹10次卡",
      "remain_times": 10,
      ...
    }
  ]
}
```

## 注意事项

1. 所有需要认证的接口都必须在请求头中携带 `Authorization: Bearer {token}`
2. Token 格式为 `user_{用户ID}_{时间戳}`（简化版本，生产环境应使用 JWT）
3. 如果 token 无效或过期，会返回 401 状态码，前端会自动跳转到登录页
4. 密码使用 bcrypt 加密存储，不会明文保存
