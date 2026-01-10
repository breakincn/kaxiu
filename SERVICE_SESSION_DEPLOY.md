# 服务会话功能部署与回归检查指南

## 1. 部署前准备

### 1.1 数据库迁移
```bash
# 在业务低峰期执行迁移脚本
mysql -u root -p your_db < backend/migrations/001_add_service_session_tables.sql
```

### 1.2 配置项说明
- `Merchant.support_room`：商户是否启用房间功能（默认 false）
- `Merchant.avg_service_minutes`：默认服务时长（分钟，默认 50）
- 新增字段在模型中已有默认值，无需额外配置

### 1.3 编译与启动
```bash
# 后端
cd backend && go build -o bin/user_service ./cmd/user_service && ./bin/user_service &
cd backend && go build -o bin/merchant_service ./cmd/merchant_service && ./bin/merchant_service &

# 前端
cd frontend && npm run build
# 将 dist-user 部署到用户端域名，dist-merchant 部署到商户端域名
```

## 2. 关键路径自测检查点

### 2.1 商户端：房间管理
- [ ] 登录商户端 → Dashboard → 服务 Tab → 房间管理入口 → 可新增/编辑/删除房间
- [ ] 房间名称唯一性校验（同商户下不可重复）
- [ ] 房间启用/停用状态生效

### 2.2 商户端：技师签到与状态切换
- [ ] 技师账号登录（js0001 / 000112345）→ Dashboard → 服务 Tab → 签到成功
- [ ] 签到后可在“可服务/忙碌”间切换状态
- [ ] 商户登录后看不到签到按钮（仅技师可见）

### 2.3 商户端：核销生成用户继续办理二维码
- [ ] 商户扫码核销成功 → 页面显示“用户继续办理”二维码
- [ ] 二维码内容格式：`https://kabao.app/user/service-sessions/{id}?next_step=room_select`（或 `staff_select`）
- [ ] 复制链接可正常访问用户端会话页

### 2.4 用户端：会话页选房/选人
- [ ] 未登录访问 `/user/service-sessions/:id` → 跳登录页并带回 redirect 参数
- [ ] 登录成功后自动跳回会话页
- [ ] 若商户开启房间功能，显示选房列表，选房成功后进入选人
- [ ] 若商户未开启房间功能，直接进入选人
- [ ] 选人成功后显示预结单码 `SS:{id}`

### 2.5 技师端：预结单扫码与状态展示
- [ ] 技师扫码 `SS:{id}` → 扫码页提示“预结单成功！会话#xxx 已开始计时”
- [ ] 技师回到 Dashboard → 服务 Tab → “我的服务中”分组出现该会话（高亮）
- [ ] 其他会话归入“其他会话”分组

### 2.6 加钟功能
- [ ] 用户端：会话页在服务中状态显示“加钟”模块，输入分钟数（5~180）提交成功
- [ ] 技师端：在“我的服务中”会话右侧显示“加钟”按钮（仅服务中），弹窗输入分钟数提交成功
- [ ] 加钟后会话时长延长，预计结束时间更新

### 2.7 自动状态流转（后台调度器）
- [ ] 核销后 90 秒未选房 → 自动分配可用房间（若冲突自动换房）
- [ ] 预结单扫码后进入 `delay_pending`，延迟秒数结束后自动进入 `serving`
- [ ] 服务时长到达后进入 `auto_finishing`，延迟 `auto_finish_delay_seconds` 后自动 `finished`
- [ ] 自动 `finished` 后技师状态在 `auto_idle_after_seconds` 后置为 `idle`

### 2.8 异常与边界
- [ ] 选房时若房间被占用 → 自动分配其他可用房间并提示
- [ ] 选人时技师不可选（非 available/idle）→ 提示“工作人员不可选”
- [ ] 非服务中状态加钟 → 提示“仅服务中可加钟”
- [ ] 用户尝试操作他人会话 → 权限拦截

## 3. 回归检查脚本（可选）

### 3.1 快速检查 SQL
```sql
-- 检查新表是否创建
SHOW TABLES LIKE 'rooms';
SHOW TABLES LIKE 'technician_attendances';
SHOW TABLES LIKE 'service_sessions';

-- 检查索引
SHOW INDEX FROM rooms;
SHOW INDEX FROM technician_attendances;
SHOW INDEX FROM service_sessions;

-- 检查数据示例
SELECT id, support_room FROM merchants LIMIT 3;
SELECT * FROM rooms LIMIT 3;
SELECT * FROM technician_attendances WHERE status = 'available' LIMIT 3;
SELECT id, status, room_id, technician_id FROM service_sessions ORDER BY id DESC LIMIT 5;
```

### 3.2 API 健康检查
```bash
# 商户端
curl -H "Authorization: Bearer $MERCHANT_TOKEN" https://api.kabao.app/merchant/rooms
curl -H "Authorization: Bearer $MERCHANT_TOKEN" https://api.kabao.app/merchant/technician/checkin -X POST -d '{}'
curl -H "Authorization: Bearer $MERCHANT_TOKEN" https://api.kabao.app/merchant/service-sessions

# 用户端
curl -H "Authorization: Bearer $USER_TOKEN" https://api.kabao.app/user/service-sessions/1
```

## 4. 常见问题

### 4.1 技师登录
- 账号规则：js + 编号（如 js0001）
- 默认密码：编号 + 12345（如 000112345）
- 登录入口复用商户端登录页

### 4.2 二维码生成
- 生产环境域名：https://kabao.app
- 开发环境域名：http://localhost:3000
- 通过环境变量 `VITE_APP_TARGET` 区分构建目标

### 4.3 调度器
- 目前采用后台 goroutine 轮询（每 10 秒一次）
- 后续可切换为定时任务或延迟队列

---

完成以上检查点后，整个服务会话重构流程即可在存量环境正常运行。
