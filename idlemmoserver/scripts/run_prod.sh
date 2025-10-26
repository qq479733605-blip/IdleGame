#!/bin/bash

# 运行生产环境构建版本的脚本
# 适用于 Linux/macOS

echo "🚀 启动 IdleMMO 生产环境服务..."

# 检查二进制文件是否存在
if [ ! -f "bin/login-service" ]; then
    echo "❌ 未找到构建文件，请先运行: ./scripts/build_all.sh"
    exit 1
fi

# 检查NATS是否运行
if ! pgrep -f "nats-server" > /dev/null; then
    echo "⚠️  NATS Server 未运行，正在启动..."
    if command -v nats-server &> /dev/null; then
        nats-server -p 4222 &
        sleep 2
        echo "✅ NATS Server 已启动"
    else
        echo "❌ 未找到 nats-server，请先安装并启动 NATS Server"
        exit 1
    fi
fi

# 创建必要的目录
mkdir -p saves
mkdir -p logs

# 启动各个服务
echo "🔧 启动 Login Service..."
./bin/login-service > logs/login.log 2>&1 &
LOGIN_PID=$!

echo "🌐 启动 Gateway Service..."
./bin/gateway-service > logs/gateway.log 2>&1 &
GATEWAY_PID=$!

echo "🎮 启动 Game Service..."
./bin/game-service > logs/game.log 2>&1 &
GAME_PID=$!

echo "💾 启动 Persistence Service..."
./bin/persist-service > logs/persist.log 2>&1 &
PERSIST_PID=$!

echo $LOGIN_PID > .login.pid
echo $GATEWAY_PID > .gateway.pid
echo $GAME_PID > .game.pid
echo $PERSIST_PID > .persist.pid

echo "🎉 生产环境服务启动完成！"
echo "💡 按 Ctrl+C 停止所有服务"

trap 'echo ""; echo "🛑 正在停止所有服务..."; kill $LOGIN_PID $GATEWAY_PID $GAME_PID $PERSIST_PID 2>/dev/null; rm -f .*.pid; echo "✅ 所有服务已停止"; exit 0' INT

wait