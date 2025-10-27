#!/bin/bash

echo "Starting all services with new Auth architecture..."

cd "$(dirname "$0")/.."

# 检查NATS是否运行
echo "Checking if NATS is running..."
if ! pgrep -f "nats-server" > /dev/null; then
    echo "Starting NATS server..."
    nats-server &
    sleep 2
fi

# 启动Auth Service (替换Login Service)
echo "Starting Auth Service..."
cd auth
go run main.go &
AUTH_PID=$!
cd ..

# 启动Persist Service
echo "Starting Persist Service..."
cd persist
go run main.go &
PERSIST_PID=$!
cd ..

# 启动Game Service
echo "Starting Game Service..."
cd game
go run main.go &
GAME_PID=$!
cd ..

# 启动Gateway Service
echo "Starting Gateway Service..."
cd gateway
go run main.go &
GATEWAY_PID=$!
cd ..

echo "All services started!"
echo "Auth Service PID: $AUTH_PID"
echo "Persist Service PID: $PERSIST_PID"
echo "Game Service PID: $GAME_PID"
echo "Gateway Service PID: $GATEWAY_PID"
echo ""
echo "Gateway is running on port 8002"
echo "Frontend should connect to: http://localhost:8002"
echo ""

# 等待用户输入来停止服务
read -p "Press Enter to stop all services..."

# 停止所有服务
echo "Stopping all services..."
kill $AUTH_PID 2>/dev/null
kill $PERSIST_PID 2>/dev/null
kill $GAME_PID 2>/dev/null
kill $GATEWAY_PID 2>/dev/null

echo "All services stopped."