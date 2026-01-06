#!/bin/bash

# =============================================================================
# 卡包系统 - 停止服务脚本
# =============================================================================

echo "停止卡包系统服务..."

# 从 PID 文件读取并停止服务
if [ -f logs/user-service.pid ]; then
    PID=$(cat logs/user-service.pid)
    if ps -p $PID > /dev/null; then
        kill $PID
        echo "用户端服务已停止 (PID: $PID)"
    else
        echo "用户端服务未运行"
    fi
    rm -f logs/user-service.pid
else
    echo "用户端服务 PID 文件不存在"
fi

if [ -f logs/merchant-service.pid ]; then
    PID=$(cat logs/merchant-service.pid)
    if ps -p $PID > /dev/null; then
        kill $PID
        echo "商户端服务已停止 (PID: $PID)"
    else
        echo "商户端服务未运行"
    fi
    rm -f logs/merchant-service.pid
else
    echo "商户端服务 PID 文件不存在"
fi

if [ -f logs/admin-service.pid ]; then
    PID=$(cat logs/admin-service.pid)
    if ps -p $PID > /dev/null; then
        kill $PID
        echo "平台端服务已停止 (PID: $PID)"
    else
        echo "平台端服务未运行"
    fi
    rm -f logs/admin-service.pid
else
    echo "平台端服务 PID 文件不存在"
fi

# 强制停止残留进程
echo "检查残留进程..."
pkill -f "user-service" || echo "无用户端服务残留"
pkill -f "merchant-service" || echo "无商户端服务残留"
pkill -f "admin-service" || echo "无平台端服务残留"

echo "所有服务已停止！"
