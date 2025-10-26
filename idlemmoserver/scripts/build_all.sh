#!/bin/bash

# 构建所有服务的脚本
# 适用于 Linux/macOS

echo "🔨 构建 IdleMMO 模块化服务..."

# 创建构建目录
mkdir -p bin

echo "📦 构建 Common 模块..."
cd common
go mod tidy
cd ..

echo "🔧 构建 Login Service..."
cd login
go mod tidy
go build -o ../bin/login-service main.go
cd ..

echo "🌐 构建 Gateway Service..."
cd gateway
go mod tidy
go build -o ../bin/gateway-service main.go
cd ..

echo "🎮 构建 Game Service..."
cd game
go mod tidy
go build -o ../bin/game-service main.go
cd ..

echo "💾 构建 Persistence Service..."
cd persist
go mod tidy
go build -o ../bin/persist-service main.go
cd ..

echo ""
echo "✅ 构建完成！"
echo ""
echo "📁 可执行文件位置:"
echo "   - bin/login-service"
echo "   - bin/gateway-service"
echo "   - bin/game-service"
echo "   - bin/persist-service"
echo ""
echo "🚀 运行: ./scripts/run_prod.sh"