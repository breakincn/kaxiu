# 卡包 kabao.me

小商户通用卡片 & 预约排队管理平台

## 项目结构

```
kabao/
├── backend/          # Go 后端
│   ├── config/       # 配置（数据库）
│   ├── handlers/     # API 处理器
│   ├── models/       # 数据模型
│   ├── routes/       # 路由
│   └── main.go       # 入口
└── frontend/         # Vue3 前端
    ├── src/
    │   ├── api/      # API 调用
    │   ├── router/   # 路由
    │   └── views/    # 页面
    └── ...
```

## 技术栈

### 后端
- Go + Gin + GORM + MySQL

### 前端
- Vue 3 + Vue Router + TailwindCSS + Axios

## 快速开始

### 1. 数据库准备

```sql
CREATE DATABASE kabao CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

根据需要修改 `backend/config/database.go` 中的数据库连接配置：
```go
dsn := "root:123456@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local"
```

### 2. 启动后端

```bash
cd backend
go mod tidy
go run main.go
```

后端服务运行于 http://localhost:8080

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端服务运行于 http://localhost:3000

## 功能模块

### 用户端
- **卡片包** - 查看所有卡片，支持"进行中/已失效"筛选
- **卡片详情** - 查看卡片详细信息、生成核销码、查看预约排队、商户通知、使用记录

### 商户端
- **排队管理** - 确认/取消预约、完成服务
- **快捷核销** - 输入核销码进行核销
- **通知管理** - 发布商户通知

## API 接口

### 用户相关
- `GET /api/users` - 获取用户列表
- `GET /api/users/:id` - 获取用户详情
- `POST /api/users` - 创建用户

### 商户相关
- `GET /api/merchants` - 获取商户列表
- `GET /api/merchants/:id` - 获取商户详情
- `PUT /api/merchants/:id` - 更新商户信息
- `GET /api/merchants/:id/queue` - 获取排队状态

### 卡片相关
- `GET /api/cards` - 获取所有卡片
- `GET /api/cards/:id` - 获取卡片详情
- `GET /api/users/:id/cards` - 获取用户卡片（支持 status 参数筛选）
- `POST /api/cards` - 创建卡片
- `POST /api/cards/:id/verify-code` - 生成核销码
- `POST /api/verify` - 核销卡片

### 预约相关
- `GET /api/merchants/:id/appointments` - 获取商户预约列表
- `GET /api/cards/:id/appointment` - 获取卡片关联预约
- `POST /api/appointments` - 创建预约
- `PUT /api/appointments/:id/confirm` - 确认预约
- `PUT /api/appointments/:id/finish` - 完成服务
- `PUT /api/appointments/:id/cancel` - 取消预约

### 通知相关
- `GET /api/merchants/:id/notices` - 获取商户通知
- `POST /api/notices` - 发布通知

## 测试数据

启动后端后会自动初始化测试数据：
- 用户：张三、u1、u2
- 商户：快剪理发店、顺风洗车
- 卡片：多张测试卡片
- 预约、通知、使用记录等

## 页面访问

- 用户端卡片包：http://localhost:3000/user/cards
- 卡片详情：http://localhost:3000/user/cards/:id
- 商户端管理：http://localhost:3000/merchant
