#!/bin/bash

# =============================================================================
# 卡包系统 - 用户端前端部署脚本
# =============================================================================
# 使用说明：
# 1. 开发环境部署（不构建）：./deploy-user-frontend.sh dev
# 2. 开发环境部署并构建：./deploy-user-frontend.sh dev build
# 3. 生产环境预览（不构建）：./deploy-user-frontend.sh preview
# 4. 生产环境预览并构建：./deploy-user-frontend.sh preview build
# 5. 停止服务：./deploy-user-frontend.sh stop
# =============================================================================

set -e

# 获取运行模式参数
MODE=${1:-dev}
BUILD_FRONTEND=${2:-""}

echo "用户端前端部署 (模式: $MODE, 构建前端: ${BUILD_FRONTEND:-"不构建"})..."

# 创建日志目录
mkdir -p logs

# 停止现有用户端前端服务
stop_user_frontend() {
    echo "停止现有用户端前端服务..."
    pkill -f "VITE_APP_TARGET=user" || true
    pkill -f "vite.*preview.*user" || true
    sleep 2
    echo "用户端前端服务已停止"
}

# 构建前端
build_frontend() {
    echo "构建用户端前端..."
    if [ ! -d "frontend" ]; then
        echo "❌ 未找到frontend目录"
        exit 1
    fi
    
    cd frontend
    npm run build:user
    if [ $? -ne 0 ]; then
        echo "❌ 用户端构建失败"
        cd ..
        exit 1
    fi
    echo "✅ 用户端构建完成"
    cd ..
}

# 开发环境启动
start_dev() {
    echo "启动开发环境用户端前端..."
    
    # 停止现有服务
    stop_user_frontend
    
    # 检查并构建前端
    if [ "$BUILD_FRONTEND" = "build" ]; then
        build_frontend
    else
        # 检查构建文件是否存在
        if [ ! -d "frontend/dist-user" ]; then
            echo "用户端构建文件不存在，自动构建..."
            build_frontend
        fi
    fi
    
    cd frontend
    
    # 启动用户端前端（端口 3000）
    echo "启动用户端前端 (端口: 3000)..."
    nohup npm run dev:user > ../logs/user-frontend.log 2>&1 &
    USER_FRONTEND_PID=$!
    echo "用户端前端已启动 (PID: $USER_FRONTEND_PID)"
    
    cd ..
    
    # 等待服务启动
    sleep 3
    
    # 检查服务状态
    if ps -p $USER_FRONTEND_PID > /dev/null; then
        echo "✅ 用户端前端运行正常 (PID: $USER_FRONTEND_PID, 端口: 3000)"
    else
        echo "❌ 用户端前端启动失败"
        echo "错误日志："
        tail -n 20 logs/user-frontend.log
        exit 1
    fi
    
    echo ""
    echo "用户端前端部署完成！"
    echo "访问地址：http://localhost:3000/"
    echo "查看日志：tail -f logs/user-frontend.log"
}

# 生产环境预览
start_preview() {
    echo "启动生产环境用户端预览..."
    
    # 停止现有服务
    stop_user_frontend
    
    # 检查并构建前端
    if [ "$BUILD_FRONTEND" = "build" ]; then
        build_frontend
    else
        # 检查构建文件是否存在
        if [ ! -d "frontend/dist-user" ]; then
            echo "错误：用户端构建文件不存在，请先运行 ./deploy-user-frontend.sh preview build"
            exit 1
        fi
    fi
    
    cd frontend
    
    # 启动用户端预览
    echo "启动用户端预览 (端口: 4173)..."
    nohup npm run preview:user > ../logs/user-preview.log 2>&1 &
    USER_PREVIEW_PID=$!
    echo "用户端预览已启动 (PID: $USER_PREVIEW_PID)"
    
    cd ..
    
    # 等待服务启动
    sleep 3
    
    # 检查服务状态
    if ps -p $USER_PREVIEW_PID > /dev/null; then
        echo "✅ 用户端预览运行正常 (PID: $USER_PREVIEW_PID, 端口: 4173)"
    else
        echo "❌ 用户端预览启动失败"
        echo "错误日志："
        tail -n 20 logs/user-preview.log
        exit 1
    fi
    
    echo ""
    echo "用户端预览部署完成！"
    echo "访问地址：http://localhost:4173/"
    echo "查看日志：tail -f logs/user-preview.log"
}

# 主逻辑
case $MODE in
    "dev")
        start_dev
        ;;
    "preview")
        start_preview
        ;;
    "stop")
        stop_user_frontend
        ;;
    *)
        echo "使用方法："
        echo "  $0 dev           # 启动开发环境"
        echo "  $0 dev build     # 启动开发环境并构建前端"
        echo "  $0 preview       # 启动生产环境预览"
        echo "  $0 preview build # 启动生产环境预览并构建前端"
        echo "  $0 stop          # 停止用户端前端服务"
        exit 1
        ;;
esac
