@echo off
REM 构建所有服务的脚本
REM 适用于 Windows

echo 🔨 构建 IdleMMO 模块化服务...

REM 创建构建目录
if not exist bin mkdir bin

echo 📦 构建 Common 模块...
cd common
go mod tidy
cd ..

echo 🔧 构建 Login Service...
cd login
go mod tidy
go build -o ../bin/login-service.exe main.go
cd ..

echo 🌐 构建 Gateway Service...
cd gateway
go mod tidy
go build -o ../bin/gateway-service.exe main.go
cd ..

echo 🎮 构建 Game Service...
cd game
go mod tidy
go build -o ../bin/game-service.exe main.go
cd ..

echo 💾 构建 Persistence Service...
cd persist
go mod tidy
go build -o ../bin/persist-service.exe main.go
cd ..

echo.
echo ✅ 构建完成！
echo.
echo 📁 可执行文件位置:
echo    - bin\login-service.exe
echo    - bin\gateway-service.exe
echo    - bin\game-service.exe
echo    - bin\persist-service.exe
echo.
echo 🚀 运行: scripts\run_prod.bat