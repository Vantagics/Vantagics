@echo off
setlocal enabledelayedexpansion

REM ==========================================
REM VantageData - 一键部署所有服务
REM License Server + Marketplace Server
REM
REM 用法:
REM   deploy_all.cmd              部署全部
REM   deploy_all.cmd license      仅部署 License Server
REM   deploy_all.cmd market       仅部署 Marketplace Server
REM   deploy_all.cmd nginx        仅部署 Nginx 配置
REM ==========================================

set SERVER_IP=107.172.86.131
set USER=root
set PASS=sunion123
set SSH_OPTS=-o StrictHostKeyChecking=no -o UserKnownHostsFile=NUL

REM 解析参数
set TARGET=%~1
if "%TARGET%"=="" set TARGET=all

echo ==========================================
echo VantageData 一键部署
echo Target: %TARGET%
echo ==========================================
echo.
echo 远程服务器: %USER%@%SERVER_IP%
echo 远程安装目录:
echo   License Server:     /root/license_server
echo   Marketplace Server: /root/marketplace_server
echo   Nginx Config:       /etc/nginx/conf.d/
echo.

REM 检查sshpass是否可用
where sshpass >nul 2>&1
if errorlevel 1 (
    echo [WARN] sshpass not found, will prompt for password.
    echo        Password: %PASS%
    echo.
    set USE_SSHPASS=0
) else (
    set USE_SSHPASS=1
)

if /i "%TARGET%"=="license" goto :deploy_license
if /i "%TARGET%"=="market" goto :deploy_market
if /i "%TARGET%"=="nginx" goto :deploy_nginx
if /i "%TARGET%"=="all" goto :deploy_all
echo [ERROR] Unknown target: %TARGET%
echo Usage: deploy_all.cmd [all^|license^|market^|nginx]
exit /b 1

:deploy_all
call :deploy_license
if errorlevel 1 exit /b 1
call :deploy_market
if errorlevel 1 exit /b 1
call :deploy_nginx
if errorlevel 1 exit /b 1
goto :verify

REM ==========================================
REM License Server 部署
REM ==========================================
:deploy_license
echo [License] Packaging source files...
cd /d "%~dp0license_server"

REM 打包源码（包含 cmd/reset_password 子目录）
tar -czf "%TEMP%\license_server.tar.gz" main.go go.mod go.sum templates cmd start.sh 2>NUL
if errorlevel 1 (
    echo [ERROR] Failed to create license_server package
    exit /b 1
)
for %%A in ("%TEMP%\license_server.tar.gz") do echo         Package: %%~zA bytes

echo [License] Uploading to server...
call :ssh_scp "%TEMP%\license_server.tar.gz" "/tmp/"
if errorlevel 1 (
    echo [ERROR] Upload failed
    exit /b 1
)

echo [License] Uploading deploy script...
call :ssh_scp "deploy_remote.sh" "/tmp/deploy_license.sh"

echo [License] Building and deploying on remote server...
call :ssh_exec "sed -i 's/\r$//' /tmp/deploy_license.sh && bash /tmp/deploy_license.sh"
if errorlevel 1 (
    echo [WARN] License Server deployment may have issues
) else (
    echo [OK] License Server deployed
)

echo [License] Health check...
call :ssh_exec "sleep 2 && curl -sf http://localhost:6699/health > /dev/null && echo '  Auth (6699): OK' || echo '  Auth (6699): FAILED' && curl -sf -o /dev/null -w '' http://localhost:8899/ && echo '  Admin (8899): OK' || echo '  Admin (8899): FAILED'"
echo.

del /f /q "%TEMP%\license_server.tar.gz" 2>NUL
if /i "%TARGET%"=="license" goto :verify
exit /b 0

REM ==========================================
REM Marketplace Server 部署
REM ==========================================
:deploy_market
echo [Market] Packaging source files...
cd /d "%~dp0marketplace_server"

tar -czf "%TEMP%\marketplace_server.tar.gz" main.go go.mod go.sum templates start.sh 2>NUL
if errorlevel 1 (
    echo [ERROR] Failed to create marketplace_server package
    exit /b 1
)
for %%A in ("%TEMP%\marketplace_server.tar.gz") do echo         Package: %%~zA bytes

echo [Market] Uploading to server...
call :ssh_scp "%TEMP%\marketplace_server.tar.gz" "/tmp/"
if errorlevel 1 (
    echo [ERROR] Upload failed
    exit /b 1
)

echo [Market] Uploading deploy script...
call :ssh_scp "deploy_remote.sh" "/tmp/deploy_market.sh"

echo [Market] Building and deploying on remote server...
call :ssh_exec "sed -i 's/\r$//' /tmp/deploy_market.sh && bash /tmp/deploy_market.sh"
if errorlevel 1 (
    echo [WARN] Marketplace Server deployment may have issues
) else (
    echo [OK] Marketplace Server deployed
)

echo [Market] Health check...
call :ssh_exec "sleep 2 && curl -sf http://localhost:8088/ > /dev/null && echo '  Marketplace (8088): OK' || echo '  Marketplace (8088): FAILED'"
echo.

del /f /q "%TEMP%\marketplace_server.tar.gz" 2>NUL
if /i "%TARGET%"=="market" goto :verify
exit /b 0

REM ==========================================
REM Nginx 配置部署
REM ==========================================
:deploy_nginx
echo [Nginx] Uploading configuration...
cd /d "%~dp0..\deploy\nginx"
call :ssh_scp "vantagedata.chat.conf" "/etc/nginx/conf.d/"
if errorlevel 1 (
    echo [ERROR] Nginx config upload failed
    exit /b 1
)

echo [Nginx] Testing and reloading...
call :ssh_exec "nginx -t && nginx -s reload && echo '  Nginx: OK' || echo '  Nginx: FAILED'"
echo.
if /i "%TARGET%"=="nginx" goto :verify
exit /b 0

REM ==========================================
REM 验证所有服务状态
REM ==========================================
:verify
echo ==========================================
echo 服务状态验证
echo ==========================================
call :ssh_exec "echo '进程检查:' && (pgrep -f '/root/license_server/license_server' > /dev/null && echo '  [OK] License Server running' || echo '  [X] License Server NOT running') && (pgrep -f '/root/marketplace_server/marketplace_server' > /dev/null && echo '  [OK] Marketplace Server running' || echo '  [X] Marketplace Server NOT running') && echo '' && echo '端口检查:' && (ss -tlnp | grep -E ':(6699|8899|8088) ' || echo '  No matching ports found') && echo '' && echo '文件检查:' && echo '  /root/license_server/:' && ls -lh /root/license_server/license_server /root/license_server/start.sh 2>/dev/null && echo '  /root/marketplace_server/:' && ls -lh /root/marketplace_server/marketplace_server /root/marketplace_server/start.sh 2>/dev/null"
echo.
echo ==========================================
echo 部署完成！
echo ==========================================
echo.
echo License Server:
echo   Auth API:  https://license.vantagedata.chat/  (port 6699)
echo   Admin:     https://license.vantagedata.chat/admin/  (port 8899)
echo.
echo Marketplace Server:
echo   Service:   https://market.vantagedata.chat/  (port 8088)
echo   Admin:     https://market.vantagedata.chat/admin/
echo.
echo 查看日志:
echo   ssh root@%SERVER_IP% "tail -f /root/license_server/server.log"
echo   ssh root@%SERVER_IP% "tail -f /root/marketplace_server/server.log"
echo.
goto :eof

REM ==========================================
REM SSH 辅助函数
REM ==========================================
:ssh_exec
if "%USE_SSHPASS%"=="1" (
    sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER_IP% %1 2>NUL
) else (
    ssh %SSH_OPTS% %USER%@%SERVER_IP% %1
)
exit /b %errorlevel%

:ssh_scp
if "%USE_SSHPASS%"=="1" (
    sshpass -p "%PASS%" scp %SSH_OPTS% %1 %USER%@%SERVER_IP%:%2 2>NUL
) else (
    scp %SSH_OPTS% %1 %USER%@%SERVER_IP%:%2
)
exit /b %errorlevel%
