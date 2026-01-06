#!/bin/bash

# =============================================================================
# 卡包系统 - 前端启动脚本
# =============================================================================
# 使用说明：
# 1. 开发环境：./start-frontend.sh dev
# 2. 生产环境预览：./start-frontend.sh preview
# 3. 停止所有前端：./start-frontend.sh stop
# =============================================================================

set -e

# 获取运行模式参数
MODE=${1:-dev}

echo "前端服务管理 (模式: $MODE)..."

# 创建日志目录
mkdir -p logs

# 停止现有前端服务
stop_frontend() {
    echo "停止现有前端服务..."
    pkill -f "vite.*VITE_APP_TARGET" || true
    pkill -f "vite.*--port" || true
    sleep 2
    echo "前端服务已停止"
}

# 开发环境启动
start_dev() {
    echo "启动开发环境前端服务..."
    
    # 停止现有服务
    stop_frontend
    
    # 启动用户端前端（默认端口 3000）
    cd /opt/kabao/frontend
    echo "启动用户端前端 (端口: 3000)..."
    nohup npm run dev:user > ../logs/user-frontend.log 2>&1 &
    echo "用户端前端已启动 (PID: $!)"
    
    # 启动商户端前端（端口 3001）
    echo "启动商户端前端 (端口: 3001)..."
    nohup npm run dev:merchant -- --port 3001 > ../logs/merchant-frontend.log 2>&1 &
    echo "商户端前端已启动 (PID: $!)"
    
    # 启动平台端前端（端口 3002）
    echo "启动平台端前端 (端口: 3002)..."
    nohup npm run dev:admin -- --port 3002 > ../logs/admin-frontend.log 2>&1 &
    echo "平台端前端已启动 (PID: $!)"
    
    cd ..
    
    echo ""
    echo "开发环境前端服务已全部启动！"
    echo "访问地址："
    echo "  - 用户端: http://localhost:3000/"
    echo "  - 商户端: http://localhost:3001/"
    echo "  - 平台端: http://localhost:3002/"
    echo ""
    echo "查看日志："
    echo "  - 用户端: tail -f logs/user-frontend.log"
    echo "  - 商户端: tail -f logs/merchant-frontend.log"
    echo "  - 平台端: tail -f logs/admin-frontend.log"
}

# 生产环境预览
start_preview() {
    echo "启动生产环境预览..."
    
    # 停止现有服务
    stop_frontend
    
    # 检查构建文件
    if [ ! -d "/opt/kabao/frontend/dist-user" ]; then
        echo "错误：用户端构建文件不存在，请先运行 npm run build:user"
        exit 1
    fi
    
    if [ ! -d "/opt/kabao/frontend/dist-merchant" ]; then
        echo "错误：商户端构建文件不存在，请先运行 npm run build:merchant"
        exit 1
    fi
    
    if [ ! -d "/opt/kabao/frontend/dist-admin" ]; then
        echo "错误：平台端构建文件不存在，请先运行 npm run build:admin"
        exit 1
    fi
    
    # 启动用户端预览
    cd /opt/kabao/frontend
    echo "启动用户端预览 (端口: 4173)..."
    nohup npm run preview:user > ../logs/user-preview.log 2>&1 &
    echo "用户端预览已启动 (PID: $!)"
    
    # 启动商户端预览
    echo "启动商户端预览 (端口: 4174)..."
    nohup npm run preview:merchant > ../logs/merchant-preview.log 2>&1 &
    echo "商户端预览已启动 (PID: $!)"
    
    # 启动平台端预览
    echo "启动平台端预览 (端口: 4175)..."
    nohup npm run preview:admin > ../logs/admin-preview.log 2>&1 &
    echo "平台端预览已启动 (PID: $!)"
    
    cd ..
    
    echo ""
    echo "生产环境预览服务已全部启动！"
    echo "访问地址："
    echo "  - 用户端: http://localhost:4173/"
    echo "  - 商户端: http://localhost:4174/"
    echo "  - 平台端: http://localhost:4175/"
    echo ""
    echo "查看日志："
    echo "  - 用户端: tail -f logs/user-preview.log"
    echo "  - 商户端: tail -f logs/merchant-preview.log"
    echo "  - 平台端: tail -f logs/admin-preview.log"
}

# 检查服务状态
check_status() {
    echo "检查前端服务状态..."
    echo ""
    
    # 检查开发环境
    echo "开发环境："
    if pgrep -f "vite.*VITE_APP_TARGET=user" > /dev/null; then
        echo "  ✅ 用户端前端运行中 (端口: 3000)"
    else
        echo "  ❌ 用户端前端未运行"
    fi
    
    if pgrep -f "vite.*VITE_APP_TARGET=merchant" > /dev/null; then
        echo "  ✅ 商户端前端运行中 (端口: 3001)"
    else
        echo "  ❌ 商户端前端未运行"
    fi
    
    if pgrep -f "vite.*VITE_APP_TARGET=admin" > /dev/null; then
        echo "  ✅ 平台端前端运行中 (端口: 3002)"
    else
        echo "  ❌ 平台端前端未运行"
    fi
    
    echo ""
    
    # 检查预览环境
    echo "预览环境："
    if pgrep -f "vite.*preview.*user" > /dev/null; then
        echo "  ✅ 用户端预览运行中 (端口: 4173)"
    else
        echo "  ❌ 用户端预览未运行"
    fi
    
    if pgrep -f "vite.*preview.*merchant" > /dev/null; then
        echo "  ✅ 商户端预览运行中 (端口: 4174)"
    else
        echo "  ❌ 商户端预览未运行"
    fi
    
    if pgrep -f "vite.*preview.*admin" > /dev/null; then
        echo "  ✅ 平台端预览运行中 (端口: 4175)"
    else
        echo "  ❌ 平台端预览未运行"
    fi
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
        stop_frontend
        ;;
    "status")
        check_status
        ;;
    *)
        echo "使用方法："
        echo "  $0 dev      # 启动开发环境"
        echo "  $0 preview  # 启动生产环境预览"
        echo "  $0 stop     # 停止所有前端服务"
        echo "  $0 status   # 查看服务状态"
        exit 1
        ;;
esac
