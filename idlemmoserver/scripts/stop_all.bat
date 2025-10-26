@echo off
REM 停止所有服务的脚本
REM 适用于 Windows

echo 🛑 停止 IdleMMO 模块化服务...

REM 停止所有相关的go进程
echo 🧹 清理服务进程...
taskkill /F /IM "go.exe" /FI "WINDOWTITLE eq *login*" 2>NUL
taskkill /F /IM "go.exe" /FI "WINDOWTITLE eq *gateway*" 2>NUL
taskkill /F /IM "go.exe" /FI "WINDOWTITLE eq *game*" 2>NUL
taskkill /F /IM "go.exe" /FI "WINDOWTITLE eq *persist*" 2>NUL

REM 更通用的方式：杀死包含特定路径的进程
for /f "tokens=2" %%i in ('tasklist /FI "IMAGENAME eq go.exe" /FO csv ^| findstr "login\\main.go"') do taskkill /F /PID %%i 2>NUL
for /f "tokens=2" %%i in ('tasklist /FI "IMAGENAME eq go.exe" /FO csv ^| findstr "gateway\\main.go"') do taskkill /F /PID %%i 2>NUL
for /f "tokens=2" %%i in ('tasklist /FI "IMAGENAME eq go.exe" /FO csv ^| findstr "game\\main.go"') do taskkill /F /PID %%i 2>NUL
for /f "tokens=2" %%i in ('tasklist /FI "IMAGENAME eq go.exe" /FO csv ^| findstr "persist\\main.go"') do taskkill /F /PID %%i 2>NUL

REM 清理PID文件
if exist .login.pid del .login.pid
if exist .gateway.pid del .gateway.pid
if exist .game.pid del .game.pid
if exist .persist.pid del .persist.pid

echo ✅ 所有服务已停止