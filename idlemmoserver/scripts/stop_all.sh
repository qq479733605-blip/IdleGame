#!/bin/bash

# 停止所有服务的脚本
# 适用于 Linux/macOS

echo "🛑 停止 IdleMMO 模块化服务..."

# 从PID文件读取并停止服务
if [ -f .login.pid ]; then
    LOGIN_PID=$(cat .login.pid)
    if kill -0 $LOGIN_PID 2>/dev/null; then
        kill $LOGIN_PID
        echo "✅ Login Service 已停止"
    fi
    rm -f .login.pid
fi

if [ -f .gateway.pid ]; then
    GATEWAY_PID=$(cat .gateway.pid)
    if kill -0 $GATEWAY_PID 2>/dev/null; then
        kill $GATEWAY_PID
        echo "✅ Gateway Service 已停止"
    fi
    rm -f .gateway.pid
fi

if [ -f .game.pid ]; then
    GAME_PID=$(cat .game.pid)
    if kill -0 $GAME_PID 2>/dev/null; then
        kill $GAME_PID
        echo "✅ Game Service 已停止"
    fi
    rm -f .game.pid
fi

if [ -f .persist.pid ]; then
    PERSIST_PID=$(cat .persist.pid)
    if kill -0 $PERSIST_PID 2>/dev/null; then
        kill $PERSIST_PID
        echo "✅ Persistence Service 已停止"
    fi
    rm -f .persist.pid
fi

# 强制杀死可能残留的进程
echo "🧹 清理残留进程..."
pkill -f "go run login/main.go" 2>/dev/null
pkill -f "go run gateway/main.go" 2>/dev/null
pkill -f "go run game/main.go" 2>/dev/null
pkill -f "go run persist/main.go" 2>/dev/null

echo "✅ 所有服务已停止"