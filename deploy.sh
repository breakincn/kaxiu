#!/bin/bash

# =============================================================================
# 卡包系统 - 生产环境部署脚本
# =============================================================================
# 使用说明：
# 1. 开发环境：./deploy.sh dev
# 2. 生产环境：./deploy.sh prod
# =============================================================================

set -e

# 获取部署模式参数
MODE=${1:-prod}

echo "开始部署卡包系统 (模式: $MODE)..."

# 创建必要目录
mkdir -p bin logs

# 设置环境变量
if [ "$MODE" = "dev" ]; then
    # 开发环境配置
    export KABAO_DSN="${KABAO_DSN:-kabao:kabao123456@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local}"
    export KABAO_CORS_ALLOW_ORIGINS="${KABAO_CORS_ALLOW_ORIGINS:-http://localhost:3000,http://localhost:5173,http://localhost:5174,https://kabao.app,https://kabao.shop}"
    export PLATFORM_ADMIN_TOKEN="${PLATFORM_ADMIN_TOKEN:-dev-token-123}"
    export GIN_MODE=debug
    export LOG_LEVEL=debug
else
    # 生产环境配置
    export KABAO_DSN="${KABAO_DSN:-kabao:kabao123456@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local}"
    export KABAO_CORS_ALLOW_ORIGINS="${KABAO_CORS_ALLOW_ORIGINS:-https://kabao.app,https://kabao.shop}"
    export PLATFORM_ADMIN_TOKEN="${PLATFORM_ADMIN_TOKEN:-KabaoAdmin2026!}"
    export GIN_MODE=release
    export LOG_LEVEL=info
fi

echo "环境变量："
echo "  KABAO_DSN=${KABAO_DSN}"
echo "  KABAO_CORS_ALLOW_ORIGINS=${KABAO_CORS_ALLOW_ORIGINS}"
echo "  PLATFORM_ADMIN_TOKEN=${PLATFORM_ADMIN_TOKEN}"
echo "  GIN_MODE=${GIN_MODE}"

# 停止现有服务
echo "停止现有服务..."
pkill -f "user-service" || true
pkill -f "merchant-service" || true
pkill -f "admin-service" || true
sleep 2

# 初始化 Go 模块和依赖
echo "初始化 Go 模块..."
cd backend

# 检查是否已初始化
if [ ! -f "go.mod" ]; then
    echo "创建 go.mod 文件..."
    go mod init kabao
fi

# 下载依赖
echo "下载 Go 依赖..."
go mod tidy

# 编译服务
echo "编译服务..."
go build -o ../bin/user-service ./cmd/user_service/main.go
go build -o ../bin/merchant-service ./cmd/merchant_service/main.go
go build -o ../bin/admin-service ./cmd/admin_service/main.go

cd ..

# 启动服务
echo "启动服务..."
nohup ./bin/user-service > logs/user-service.log 2>&1 &
echo "用户端服务已启动 (PID: $!)"
USER_PID=$!

nohup ./bin/merchant-service > logs/merchant-service.log 2>&1 &
echo "商户端服务已启动 (PID: $!)"
MERCHANT_PID=$!

nohup ./bin/admin-service > logs/admin-service.log 2>&1 &
echo "平台端服务已启动 (PID: $!)"
ADMIN_PID=$!

# 等待服务启动
echo "等待服务启动..."
sleep 3

# 检查服务状态
echo "检查服务状态..."
if ps -p $USER_PID > /dev/null; then
    echo "✅ 用户端服务运行正常 (PID: $USER_PID, 端口: 8081)"
else
    echo "❌ 用户端服务启动失败"
    tail -n 20 logs/user-service.log
fi

if ps -p $MERCHANT_PID > /dev/null; then
    echo "✅ 商户端服务运行正常 (PID: $MERCHANT_PID, 端口: 8082)"
else
    echo "❌ 商户端服务启动失败"
    tail -n 20 logs/merchant-service.log
fi

if ps -p $ADMIN_PID > /dev/null; then
    echo "✅ 平台端服务运行正常 (PID: $ADMIN_PID, 端口: 8083)"
else
    echo "❌ 平台端服务启动失败"
    tail -n 20 logs/admin-service.log
fi

# 保存 PID 到文件
echo $USER_PID > logs/user-service.pid
echo $MERCHANT_PID > logs/merchant-service.pid
echo $ADMIN_PID > logs/admin-service.pid

echo ""
echo "部署完成！"
echo "日志文件位置："
echo "  - 用户端：logs/user-service.log"
echo "  - 商户端：logs/merchant-service.log"
echo "  - 平台端：logs/admin-service.log"
echo ""
echo "查看实时日志："
echo "  tail -f logs/user-service.log"
echo "  tail -f logs/merchant-service.log"
echo "  tail -f logs/admin-service.log"
echo ""
echo "停止服务："
echo "  ./stop.sh"
