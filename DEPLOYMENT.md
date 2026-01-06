# 卡包项目部署说明

## 架构概览

- **前端**：用户端（`kabao.app`） + 商户端（`kabao.shop`）独立部署
- **后端**：三个独立服务 + 单一 API 域名 `api.kabao.app`
  - `user-service`（8081）→ `/api/user/*`、`/api/platform/*`
  - `merchant-service`（8082）→ `/api/merchant/*`
  - `admin-service`（8083）→ `/api/admin/*`
- **数据库**：共享 MySQL，可选 schema 隔离
- **反向代理**：Nginx 按 API 前缀分发到不同后端服务

---

## 1) 前端独立部署

### 1.1 构建产物

```bash
# 用户端
cd frontend
npm run build:user
# 输出：dist-user/

# 商户端
npm run build:merchant
# 输出：dist-merchant/
```

### 1.2 部署到不同域名

- 用户端：`dist-user/` → `kabao.app`
- 商户端：`dist-merchant/` → `kabao.shop`

### 1.3 环境变量（可选覆盖）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `VITE_API_BASE_URL` | 开发 `/api`，生产 `https://api.kabao.app` | API 基础地址 |
| `VITE_APP_TARGET` | `user` 或 `merchant` | 控制路由集合与构建输出 |

---

## 2) 后端三服务部署

### 2.1 数据库配置

#### 方案 A：共享数据库（推荐）
```bash
# 三个服务共用同一 DB
export KABAO_DSN="root:password@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local"
```

#### 方案 B：Schema 隔离（可选）
```bash
# 用户端服务
export KABAO_DSN="root:password@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local"

# 商户端服务
export KABAO_DSN="root:password@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local"

# 平台端服务
export KABAO_DSN="root:password@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local"
```

> 如需 schema 隔离，请在 DSN 中指定不同 schema（如 `kabao_user_service` 等），并确保每个 schema 有相同表结构。

### 2.2 CORS 配置（环境变量覆盖）

默认仅允许 `https://kabao.app`、`https://kabao.shop`。

```bash
# 开发环境可临时放宽
export KABAO_CORS_ALLOW_ORIGINS="http://localhost:3000,http://localhost:3001"

# 生产环境（默认可省略）
export KABAO_CORS_ALLOW_ORIGINS="https://kabao.app,https://kabao.shop"
```

### 2.3 平台管理员 Token（平台后台鉴权）

```bash
export PLATFORM_ADMIN_TOKEN="your-secret-token"
```

### 2.4 启动命令

```bash
# 用户端服务（8081）
cd backend
go run cmd/user_service/main.go

# 商户端服务（8082）
go run cmd/merchant_service/main.go

# 平台端服务（8083）
go run cmd/admin_service/main.go
```

### 2.5 可选：单体后端（兼容旧部署）

```bash
# 原单体入口（8080，包含所有路由）
go run main.go
```

---

## 3) Nginx 反向代理配置

### 3.1 域名：`kabao.app`、`api.kabao.app`

```nginx
##
# kabao.app 站点 HTTPS
##
server {
    listen 443 ssl http2;
    server_name kabao.app www.kabao.app;

    ssl_certificate     /etc/nginx/ssl/kabao.app.crt;
    ssl_certificate_key /etc/nginx/ssl/kabao.app.key;

    root /opt/kabao/frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    add_header Strict-Transport-Security "max-age=63072000" always;
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
}

##
# api.kabao.app API 反向代理
##
server {
    listen 443 ssl;
    server_name api.kabao.app;

    # SSL 证书配置
    ssl_certificate     /etc/nginx/ssl/kabao.app.crt;
    ssl_certificate_key /etc/nginx/ssl/kabao.app.key;

    # 用户端 API
    location /api/user/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 平台公开接口
    location /api/platform/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 商户端 API
    location /api/merchant/ {
        proxy_pass http://127.0.0.1:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 平台后台 API
    location /api/admin/ {
        proxy_pass http://127.0.0.1:8083;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 静态资源（兼容旧路径）
    location /api/uploads/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 4) 开发环境快速启动

### 4.1 启动后端三服务（并行）

```bash
# 终端 1：用户端服务
cd backend && go run cmd/user_service/main.go

# 终端 2：商户端服务
cd backend && go run cmd/merchant_service/main.go

# 终端 3：平台端服务（如需）
cd backend && go run cmd/admin_service/main.go
```

### 4.2 启动前端（并行）

```bash
# 终端 4：用户端前端（3000）
cd frontend && npm run dev:user

# 终端 5：商户端前端（3001）
cd frontend && npm run dev:merchant
```

### 4.3 开发代理（Vite）

前端默认走 `/api` 代理到 `http://10.0.0.20:8080`（可在 `vite.config.js` 修改）。

如需代理到本地三服务，可临时调整 `vite.config.js` 的 proxy：

```js
proxy: {
  '/api/user': { target: 'http://localhost:8081', changeOrigin: true },
  '/api/merchant': { target: 'http://localhost:8082', changeOrigin: true },
  '/api/admin': { target: 'http://localhost:8083', changeOrigin: true },
  '/api/platform': { target: 'http://localhost:8081', changeOrigin: true },
  '/api/uploads': { target: 'http://localhost:8081', changeOrigin: true }
}
```

---

## 5) 平台后台：域名配置接口

### 5.1 接口地址

- `GET /api/admin/system/config`
- `PUT /api/admin/system/config`

### 5.2 请求头（鉴权）

```
X-Platform-Admin-Token: ${PLATFORM_ADMIN_TOKEN}
```

### 5.3 默认值

```json
{
  "user_app_domain": "kabao.app",
  "merchant_app_domain": "kabao.shop",
  "api_domain": "api.kabao.app"
}
```

---

## 6) 常见问题

### Q1：前端 API 404？
- 确认后端对应服务已启动（8081/8082/8083）
- 确认 Nginx 或 Vite 代理按前缀转发正确

### Q2：CORS 错误？
- 检查 `KABAO_CORS_ALLOW_ORIGINS` 是否包含前端域名
- 开发环境可临时设为 `"*"`（不推荐生产使用）

### Q3：数据库连接失败？
- 确认 `KABAO_DSN` 正确
- 确认 MySQL 版本兼容（建议 8.0+）
- 确认数据库/Schema 已存在

### Q4：平台后台 403？
- 检查 `PLATFORM_ADMIN_TOKEN` 环境变量
- 确认请求头 `X-Platform-Admin-Token` 正确

---

## 7) 生产部署建议

1. **容器化**：每个后端服务独立 Docker 镜像
2. **编排**：使用 Docker Compose / K8s
3. **监控**：分别监控 8081/8082/8083 健康状态
4. **日志**：统一收集三服务日志
5. **数据库**：建议共享数据库，按 schema 隔离（可选）
6. **SSL**：前端域名与 API 域名均启用 HTTPS
7. **限流**：在 Nginx 或 API Gateway 层加限流

---

## 8) 版本兼容

- **旧单体后端**：仍可通过 `go run main.go` 启动（8080），包含所有路由
- **前端**：新旧 API 路径兼容，但建议尽快迁移到新前缀
- **数据库**：表结构无破坏性变更，新增 `system_configs` 表

---

## 9) 下一步扩展

- **API Gateway**：如需更复杂的路由/聚合/限流，可引入网关层
- **Schema 隔离**：如需完全数据隔离，可按服务拆分 schema
- **多环境**：通过环境变量区分 dev/staging/prod 配置
- **CI/CD**：为每个服务独立构建与部署流水线

---

> 如需进一步支持（Dockerfile、K8s YAML、CI/CD 示例），请告知。
