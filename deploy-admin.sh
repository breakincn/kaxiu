#!/bin/bash

# =============================================================================
# 卡包系统 - Admin 服务部署脚本（优化版）
# =============================================================================
# 特点：
# - 前端构建可选，且不会导致“假卡死”
# - Node.js 内存显式配置，避免 Vite rendering chunks 卡住
# - 前端构建失败不会中断后端部署
# - 更适合线上 / 云服务器使用
# =============================================================================

set -e

# -----------------------------------------------------------------------------
# 参数解析
# -----------------------------------------------------------------------------
MODE=${1:-prod}                 # prod | dev
BUILD_FRONTEND=${2:-""}         # build | 空

echo "🚀 开始部署 Admin 服务"
echo "  - 模式: $MODE"
echo "  - 是否构建前端: ${BUILD_FRONTEND:-不构建}"
echo ""

# -----------------------------------------------------------------------------
# 基础目录
# -----------------------------------------------------------------------------
ROOT_DIR=$(pwd)
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR/backend"

mkdir -p bin logs

# -----------------------------------------------------------------------------
# 可选：构建前端（不会影响后端部署）
# -----------------------------------------------------------------------------
if [ "$BUILD_FRONTEND" = "build" ]; then
    echo "🎨 开始构建 Admin 前端..."

    if [ -d "$FRONTEND_DIR" ]; then
        cd "$FRONTEND_DIR"

        # 🚨 关键：防止 Vite / Rollup 内存不足
        export NODE_OPTIONS=--max-old-space-size=4096

        echo "NODE_OPTIONS=$NODE_OPTIONS"
        echo "执行: npm run build:admin"
        echo "（production build 可能需要 1~5 分钟，请耐心等待）"
        echo ""

        # 不让前端失败影响后端部署
        if npm run build:admin; then
            echo "✅ 前端构建完成"
        else
            echo "⚠️ 前端构建失败（已忽略），请检查前端日志"
        fi

        cd "$ROOT_DIR"
        echo ""
    else
        echo "⚠️ 未找到 frontend 目录，跳过前端构建"
        echo ""
    fi
fi

# -----------------------------------------------------------------------------
# 环境变量配置
# -----------------------------------------------------------------------------
if [ "$MODE" = "dev" ]; then
    export KABAO_DSN="${KABAO_DSN:-kabao:kabao123456@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local}"
    export KABAO_CORS_ALLOW_ORIGINS="${KABAO_CORS_ALLOW_ORIGINS:-http://localhost:3000,http://localhost:5173,https://kabao.app,https://kabao.shop}"
    export PLATFORM_ADMIN_TOKEN="${PLATFORM_ADMIN_TOKEN:-KabaoAdmin2026!}"
    export GIN_MODE=debug
    export LOG_LEVEL=debug
else
    export KABAO_DSN="${KABAO_DSN:-kabao:kabao123456@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local}"
    export KABAO_CORS_ALLOW_ORIGINS="${KABAO_CORS_ALLOW_ORIGINS:-https://kabao.app,https://kabao.shop}"
    export PLATFORM_ADMIN_TOKEN="${PLATFORM_ADMIN_TOKEN:-KabaoAdmin2026!}"
    export GIN_MODE=release
    export LOG_LEVEL=info
fi

echo "🔧 运行环境："
echo "  GIN_MODE=$GIN_MODE"
echo "  LOG_LEVEL=$LOG_LEVEL"
echo ""

# -----------------------------------------------------------------------------
# 停止旧进程（只杀 admin）
# -----------------------------------------------------------------------------
echo "🛑 停止旧的 Admin 服务..."
pkill -f "admin-service" || true
sleep 2

# -----------------------------------------------------------------------------
# 编译后端
# -----------------------------------------------------------------------------
echo "🧱 编译 Admin 后端服务..."

cd "$BACKEND_DIR"

if [ ! -f "go.mod" ]; then
    echo "初始化 Go Module..."
    go mod init kabao
fi

echo "下载依赖..."
go mod tidy

echo "编译二进制..."
go build -o "$ROOT_DIR/bin/admin-service" ./cmd/admin_service/main.go

cd "$ROOT_DIR"
echo "✅ 后端编译完成"
echo ""

# -----------------------------------------------------------------------------
# 启动服务
# -----------------------------------------------------------------------------
echo "🚀 启动 Admin 服务..."

nohup ./bin/admin-service > logs/admin-service.log 2>&1 &
ADMIN_PID=$!

sleep 3

if ps -p "$ADMIN_PID" > /dev/null; then
    echo "✅ Admin 服务启动成功"
    echo "  PID: $ADMIN_PID"
    echo "  Port: 8083"
else
    echo "❌ Admin 服务启动失败"
    tail -n 30 logs/admin-service.log
    exit 1
fi

echo "$ADMIN_PID" > logs/admin-service.pid

# -----------------------------------------------------------------------------
# 结果提示
# -----------------------------------------------------------------------------
echo ""
echo "🎉 Admin 服务部署完成"
echo "--------------------------------------"
echo "访问地址:"
echo "  后端: http://localhost:8083"
echo "  前端: https://kabao.shop/platform-admin/login"
echo ""
echo "常用命令:"
echo "  查看日志: tail -f logs/admin-service.log"
echo "  停止服务: pkill -f admin-service"
echo "  重启服务: ./deploy-admin.sh"
echo ""
