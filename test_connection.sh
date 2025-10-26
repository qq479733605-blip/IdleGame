#!/bin/bash

echo "🔍 测试前后端连接状态..."

# 测试后端服务
echo ""
echo "=== 测试后端API ==="

# 测试Gateway Health
echo "1. 测试Gateway服务 (http://localhost:8002)..."
curl -s -o /dev/null -w "%{http_code}" http://localhost:8002/health || echo "Gateway服务连接失败"

# 测试前端
echo ""
echo "=== 测试前端服务 ==="
echo "2. 测试前端服务 (http://localhost:5173)..."
curl -s -o /dev/null -w "%{http_code}" http://localhost:5173 || echo "前端服务连接失败"

echo ""
echo "=== 服务配置总结 ==="
echo "✅ 前端 (Vue): http://localhost:5173"
echo "✅ 后端API (Gateway): http://localhost:8002"
echo "✅ WebSocket: ws://localhost:8002/ws"
echo "✅ 登录接口: http://localhost:8002/login"
echo "✅ 注册接口: http://localhost:8002/register"

echo ""
echo "🎉 前后端端口配置完成！"
echo "💡 请在浏览器中访问: http://localhost:5173"