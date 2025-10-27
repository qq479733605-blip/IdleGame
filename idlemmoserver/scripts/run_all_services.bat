@echo off
echo 🚀 启动 IdleMMO 模块化服务...

REM 检查NATS是否运行
echo 🔍 检查NATS服务...
tasklist /FI "IMAGENAME eq nats-server.exe" 2>NUL | find /I /N "nats-server.exe">NUL
if %ERRORLEVEL% NEQ 0 (
    echo ⚠️  NATS Server 未运行，正在启动...
    where nats-server.exe >nul 2>nul
    if %ERRORLEVEL% NEQ 0 (
        echo ❌ 未找到 nats-server.exe，请先安装并启动 NATS Server
        echo    安装: go install github.com/nats-io/nats-server/v2@latest
        pause
        exit /b 1
    )
    start "NATS Server" nats-server.exe -p 4222
    timeout /t 3 /nobreak >nul
    echo ✅ NATS Server 已启动
) else (
    echo ✅ NATS Server 已运行
)

REM 创建必要的目录
if not exist "saves" mkdir saves
if not exist "logs" mkdir logs

echo 🔧 启动 Auth Service...
start "Auth Service" go run auth/main.go
set /a AUTH_PID=!ERRORLEVEL!
echo Auth Service PID: %AUTH_PID%

echo 🌐 启动 Gateway Service...
start "Gateway Service" go run gateway/main.go
set /a GATEWAY_PID=!ERRORLEVEL!
echo Gateway Service PID: %GATEWAY_PID%

echo 🎮 启动 Game Service...
start "Game Service" go run game/main.go
set /a GAME_PID=!ERRORLEVEL!
echo Game Service PID: %GAME_PID%

echo 💾 启动 Persistence Service...
start "Persistence Service" go run persist/main.go
set /a PERSIST_PID=!ERRORLEVEL!
echo Persistence Service PID: %PERSIST_PID%

echo ⏳ 等待服务启动...
timeout /t 5 /nobreak >nul

echo ""
echo "🎉 所有服务启动成功！"
echo ""
echo "📊 服务端口:"
echo "   - Gateway Service: http://localhost:8002"
echo "   - WebSocket: ws://localhost:8002/ws"
echo "   - Auth Service: :8001"
echo "   - Game Service: :8003"
echo "   - Persistence Service: :8004"
echo ""
echo "📋 日志文件:"
echo "   - Auth: logs/auth.log"
echo "   - Gateway: logs/gateway.log"
echo "   - Game: logs/game.log"
echo "   - Persistence: logs/persist.log"
echo ""
echo "💡 关闭此窗口将停止所有服务"

pause