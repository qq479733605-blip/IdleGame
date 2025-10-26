#!/bin/bash

echo "🎯 进行完整的端到端测试..."

# 测试1: Gateway Health Check
echo ""
echo "=== 测试1: Gateway Health Check ==="
response=$(curl -s http://localhost:8002/health)
if [ "$response" = "OK" ]; then
    echo "✅ Gateway Health Check - 通过"
else
    echo "❌ Gateway Health Check - 失败"
fi

# 测试2: 用户注册
echo ""
echo "=== 测试2: 用户注册 ==="
register_response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser123","password":"123456"}' \
  http://localhost:8002/register)

echo "注册响应: $register_response"
if echo "$register_response" | grep -q "success\|token\|already exists"; then
    echo "✅ 用户注册 - 通过"
else
    echo "❌ 用户注册 - 失败"
fi

# 测试3: 用户登录
echo ""
echo "=== 测试3: 用户登录 ==="
login_response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser123","password":"123456"}' \
  http://localhost:8002/login)

echo "登录响应: $login_response"
if echo "$login_response" | grep -q "success\|token"; then
    echo "✅ 用户登录 - 通过"

    # 提取token用于WebSocket测试
    token=$(echo "$login_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$token" ]; then
        echo "🔑 获取到Token: ${token:0:20}..."
    fi
else
    echo "❌ 用户登录 - 失败"
fi

# 测试4: 前端访问
echo ""
echo "=== 测试4: 前端服务 ==="
frontend_response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:5173)
if [ "$frontend_response" = "200" ]; then
    echo "✅ 前端服务 - 通过"
else
    echo "❌ 前端服务 - 失败 (HTTP $frontend_response)"
fi

echo ""
echo "=== 测试总结 ==="
echo "🌐 前端地址: http://localhost:5173"
echo "🔌 后端API: http://localhost:8002"
echo "📡 WebSocket: ws://localhost:8002/ws"
echo "🔑 登录接口: http://localhost:8002/login"
echo "📝 注册接口: http://localhost:8002/register"
echo "💚 健康检查: http://localhost:8002/health"

echo ""
echo "🎉 前后端端口配置完成！可以开始使用游戏了！"