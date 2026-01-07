#!/bin/bash

# =============================================================================
# 卡包系统 - 用户端服务单独部署脚本
# =============================================================================
# 使用说明：
# 1. 默认为生产环境部署用户端服务(不构建前端):
# ./deploy-user.sh

# 2. 指定生产环境部署用户端服务(不构建前端)：
# ./deploy-user.sh prod
# 3. 指定生产环境部署用户端服务并构建前端:
# ./deploy-user.sh prod build

# 4. 开发环境部署(不构建前端)：
# ./deploy-user.sh dev
# 5. 开发环境部署并构建前端：
# ./deploy-user.sh dev build
# 
# 功能说明：
# - 只部署用户端服务（端口8081）
# - 不影响其他已运行的服务（merchant-service:8082, admin-service:8083）
# - 可选择是否重新构建前端文件
# - 适合单独重启或修复用户端服务时使用
# =============================================================================

set -e

# 获取部署模式参数，默认为prod
# 第二个参数决定是否构建前端：build 或不传
MODE=${1:-prod}
BUILD_FRONTEND=${2:-""}

echo "开始部署用户端服务 (模式: $MODE, 前端构建: ${BUILD_FRONTEND:-"不构建"})..."

# 创建必要目录
mkdir -p bin logs

# 可选：构建前端
if [ "$BUILD_FRONTEND" = "build" ]; then
    echo "构建前端文件..."
    if [ -d "frontend" ]; then
        cd frontend
        echo "执行 npm run build:user..."
        npm run build:user
        cd ..
        echo "✅ 前端构建完成"
    else
        echo "❌ 未找到frontend目录，跳过前端构建"
    fi
fi

# 设置环境变量
if [ "$MODE" = "dev" ]; then
    # 开发环境配置
    export KABAO_DSN="${KABAO_DSN:-kabao:kabao123456@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local}"
    export KABAO_CORS_ALLOW_ORIGINS="${KABAO_CORS_ALLOW_ORIGINS:-http://localhost:3000,http://localhost:3001,http://localhost:3002,http://localhost:5173,http://localhost:5174,https://kabao.app,https://kabao.shop}"
    export PLATFORM_ADMIN_TOKEN="${PLATFORM_ADMIN_TOKEN:-KabaoAdmin2026!}"
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
echo "  LOG_LEVEL=${LOG_LEVEL}"

# 只停止user服务
echo "停止现有用户端服务..."
pkill -f "user-service" || true
sleep 2

# 进入backend目录
cd backend

# 检查是否已初始化Go模块
if [ ! -f "go.mod" ]; then
    echo "创建 go.mod 文件..."
    go mod init kabao
fi

# 下载依赖
echo "下载 Go 依赖..."
go mod tidy

# 编译user服务
echo "编译用户端服务..."
go build -o ../bin/user-service ./cmd/user_service/main.go

cd ..

# 启动user服务
echo "启动用户端服务..."
nohup ./bin/user-service > logs/user-service.log 2>&1 &
echo "用户端服务已启动 (PID: $!)"
USER_PID=$!

# 等待服务启动
echo "等待服务启动..."
sleep 3

# 检查服务状态
echo "检查用户端服务状态..."
if ps -p $USER_PID > /dev/null; then
    echo "✅ 用户端服务运行正常 (PID: $USER_PID, 端口: 8081)"
else
    echo "❌ 用户端服务启动失败"
    echo "错误日志："
    tail -n 20 logs/user-service.log
    exit 1
fi

# 保存PID到文件
echo $USER_PID > logs/user-service.pid

echo ""
echo "用户端服务部署完成！"
echo ""
echo "部署信息："
echo "  - 服务名称：User Service"
echo "  - 监听端口：8081"
echo "  - 进程ID：$USER_PID"
echo "  - 部署模式：$MODE"
echo "  - 前端构建：${BUILD_FRONTEND:-"未构建"}"
echo "  - 日志文件：logs/user-service.log"
echo ""
echo "常用命令："
echo "  - 查看实时日志：tail -f logs/user-service.log"
echo "  - 停止服务：pkill -f user-service"
echo "  - 重启服务：./deploy-user.sh"
echo "  - 重启并构建前端：./deploy-user.sh prod build"
echo ""
echo "测试命令："
echo "  - 直接测试：curl http://localhost:8081/health"
echo "  - 代理测试：curl https://api.kabao.app/health"
echo ""
echo "前端访问："
echo "  - 用户端地址：https://kabao.shop"
echo "  - 用户注册/登录页面"
echo ""

# 可选：自动测试服务是否正常
echo "是否自动测试服务？(y/n)"
read -r response
if [[ "$response" =~ ^[Yy]$ ]]; then
    echo "测试用户端服务..."
    TEST_RESULT=$(curl -s http://localhost:8081/health)
    if [[ "$TEST_RESULT" == *"error"* ]] || [[ -z "$TEST_RESULT" ]]; then
        echo "❌ 服务测试失败：$TEST_RESULT"
        echo "请检查日志：tail -n 20 logs/user-service.log"
    else
        echo "✅ 服务测试成功：$TEST_RESULT"
    fi
fi
