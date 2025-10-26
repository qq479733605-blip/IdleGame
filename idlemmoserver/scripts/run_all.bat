@echo off
REM 运行所有服务的启动脚本
REM 适用于 Windows

echo 🚀 启动 IdleMMO 模块化服务...

REM 检查NATS是否运行
tasklist /FI "IMAGENAME eq nats-server.exe" 2>NUL | find /I /N "nats-server.exe">NUL
if %ERRORLEVEL% neq 0 (
    echo ⚠️  NATS Server 未运行，正在启动...
    REM 尝试启动NATS，如果失败则提示用户手动启动
    where nats-server >nul 2>&1
    if %ERRORLEVEL% equ 0 (
        start /B nats-server -p 4222
        timeout /t 2 /nobreak >nul
        echo ✅ NATS Server 已启动
    ) else (
        echo ❌ 未找到 nats-server，请先安装并启动 NATS Server
        echo    安装: 下载 from https://github.com/nats-io/nats-server
        pause
        exit /b 1
    )
) else (
    echo ✅ NATS Server 已运行
)

REM 创建必要的目录
if not exist saves mkdir saves
if not exist logs mkdir logs

REM 启动各个服务
echo 🔧 启动 Login Service...
start /B cmd /c "go run login/main.go > logs\login.log 2>&1"
echo Login Service 已启动

echo 🌐 启动 Gateway Service...
start /B cmd /c "go run gateway/main.go > logs\gateway.log 2>&1"
echo Gateway Service 已启动

echo 🎮 启动 Game Service...
start /B cmd /c "go run game/main.go > logs\game.log 2>&1"
echo Game Service 已启动

echo 💾 启动 Persistence Service...
start /B cmd /c "go run persist/main.go > logs\persist.log 2>&1"
echo Persistence Service 已启动

REM 等待服务启动
echo ⏳ 等待服务启动...
timeout /t 5 /nobreak >nul

echo.
echo 🎉 所有服务启动完成！
echo.
echo 📊 服务端口:
echo    - Gateway Service: http://localhost:8002
echo    - WebSocket: ws://localhost:8002/ws
echo    - Login Service: :8001
echo    - Game Service: :8003
echo    - Persistence Service: :8004
echo.
echo 📋 日志文件:
echo    - Login: logs\login.log
echo    - Gateway: logs\gateway.log
echo    - Game: logs\game.log
echo    - Persistence: logs\persist.log
echo.
echo 💡 按 Ctrl+C 停止所有服务
echo.

REM 等待用户中断
pause