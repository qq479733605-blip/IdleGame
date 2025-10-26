#!/bin/bash

# 运行所有服务的启动脚本
# 适用于 Linux/macOS

echo "🚀 启动 IdleMMO 模块化服务..."

# 检查NATS是否运行
if ! pgrep -f "nats-server" > /dev/null; then
    echo "⚠️  NATS Server 未运行，正在启动..."
    # 尝试启动NATS，如果失败则提示用户手动启动
    if command -v nats-server &> /dev/null; then
        nats-server -p 4222 &
        sleep 2
        echo "✅ NATS Server 已启动"
    else
        echo "❌ 未找到 nats-server，请先安装并启动 NATS Server"
        echo "   安装: brew install nats-server 或 download from https://github.com/nats-io/nats-server"
        exit 1
    fi
else
    echo "✅ NATS Server 已运行"
fi

# 创建必要的目录
mkdir -p saves
mkdir -p logs

# 启动各个服务
echo "🔧 启动 Login Service..."
go run login/main.go > logs/login.log 2>&1 &
LOGIN_PID=$!
echo "Login Service PID: $LOGIN_PID"

echo "🌐 启动 Gateway Service..."
go run gateway/main.go > logs/gateway.log 2>&1 &
GATEWAY_PID=$!
echo "Gateway Service PID: $GATEWAY_PID"

echo "🎮 启动 Game Service..."
go run game/main.go > logs/game.log 2>&1 &
GAME_PID=$!
echo "Game Service PID: $GAME_PID"

echo "💾 启动 Persistence Service..."
go run persist/main.go > logs/persist.log 2>&1 &
PERSIST_PID=$!
echo "Persistence Service PID: $PERSIST_PID"

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 5

# 检查服务状态
check_service() {
    local pid=$1
    local name=$2
    if kill -0 $pid 2>/dev/null; then
        echo "✅ $name 运行正常"
        return 0
    else
        echo "❌ $name 启动失败"
        return 1
    fi
}

all_running=true
check_service $LOGIN_PID "Login Service" || all_running=false
check_service $GATEWAY_PID "Gateway Service" || all_running=false
check_service $GAME_PID "Game Service" || all_running=false
check_service $PERSIST_PID "Persistence Service" || all_running=false

if $all_running; then
    echo ""
    echo "🎉 所有服务启动成功！"
    echo ""
    echo "📊 服务端口:"
    echo "   - Gateway Service: http://localhost:8002"
    echo "   - WebSocket: ws://localhost:8002/ws"
    echo "   - Login Service: :8001"
    echo "   - Game Service: :8003"
    echo "   - Persistence Service: :8004"
    echo ""
    echo "📋 日志文件:"
    echo "   - Login: logs/login.log"
    echo "   - Gateway: logs/gateway.log"
    echo "   - Game: logs/game.log"
    echo "   - Persistence: logs/persist.log"
    echo ""
    echo "💡 按 Ctrl+C 停止所有服务"

    # 保存PID到文件，方便停止
    echo $LOGIN_PID > .login.pid
    echo $GATEWAY_PID > .gateway.pid
    echo $GAME_PID > .game.pid
    echo $PERSIST_PID > .persist.pid

    # 等待中断信号
    trap 'echo ""; echo "🛑 正在停止所有服务..."; kill $LOGIN_PID $GATEWAY_PID $GAME_PID $PERSIST_PID 2>/dev/null; rm -f .*.pid; echo "✅ 所有服务已停止"; exit 0' INT

    # 保持脚本运行
    wait
else
    echo ""
    echo "❌ 部分服务启动失败，请检查日志文件"
    kill $LOGIN_PID $GATEWAY_PID $GAME_PID $PERSIST_PID 2>/dev/null
    exit 1
fi