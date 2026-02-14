@echo off
setlocal enabledelayedexpansion

REM ==========================================
REM VantageData - 一键部署所有服务
REM License Server + Marketplace Server
REM ==========================================

set SERVER_IP=107.172.86.131
set USER=root
set PASS=sunion123
set SSH_OPTS=-o StrictHostKeyChecking=no -o UserKnownHostsFile=NUL

echo ==========================================
echo VantageData 一键部署
echo ==========================================
echo.

REM 检查sshpass是否可用
where sshpass >nul 2>&1
if errorlevel 1 (
    echo [ERROR] sshpass not found! Please install sshpass.
    echo.
    echo Download: https://github.com/PowerShell/Win32-OpenSSH/releases
    pause
    exit /b 1
)

REM ==========================================
REM 1. 打包 License Server
REM ==========================================
echo [1/6] Packaging License Server...
cd /d "%~dp0license_server"
tar -czf "%TEMP%\license_server.tar.gz" main.go go.mod go.sum templates start.sh 2>NUL
if errorlevel 1 (
    echo      [ERROR] Failed to create package
    pause
    exit /b 1
)
for %%A in ("%TEMP%\license_server.tar.gz") do echo      Package: %%~zA bytes
echo.

REM ==========================================
REM 2. 上传并部署 License Server
REM ==========================================
echo [2/6] Deploying License Server...
sshpass -p "%PASS%" scp %SSH_OPTS% "%TEMP%\license_server.tar.gz" %USER%@%SERVER_IP%:/tmp/ 2>NUL
if errorlevel 1 (
    echo      [ERROR] Upload failed
    pause
    exit /b 1
)
echo      Uploaded successfully

echo      Building and starting on remote server...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER_IP% "cd /root && rm -rf license_server_new && mkdir -p license_server_new && cd license_server_new && tar -xzf /tmp/license_server.tar.gz && go mod tidy && CGO_ENABLED=1 go build -o license_server . && cd /root && pkill -f 'license_server' || true && sleep 2 && rm -rf license_server_old && mv license_server license_server_old 2>/dev/null || true && mv license_server_new license_server && cd license_server && chmod +x start.sh && ./start.sh && sleep 3 && curl -s http://localhost:8080/api/health && echo 'License Server started successfully' || echo 'WARNING: Health check failed'" 2>NUL

if errorlevel 1 (
    echo      [WARN] Deployment may have issues, check logs
) else (
    echo      [OK] License Server deployed
)
echo.

REM ==========================================
REM 3. 打包 Marketplace Server
REM ==========================================
echo [3/6] Packaging Marketplace Server...
cd /d "%~dp0marketplace_server"
tar -czf "%TEMP%\marketplace_server.tar.gz" main.go go.mod go.sum templates start.sh 2>NUL
if errorlevel 1 (
    echo      [ERROR] Failed to create package
    pause
    exit /b 1
)
for %%A in ("%TEMP%\marketplace_server.tar.gz") do echo      Package: %%~zA bytes
echo.

REM ==========================================
REM 4. 上传并部署 Marketplace Server
REM ==========================================
echo [4/6] Deploying Marketplace Server...
sshpass -p "%PASS%" scp %SSH_OPTS% "%TEMP%\marketplace_server.tar.gz" %USER%@%SERVER_IP%:/tmp/ 2>NUL
if errorlevel 1 (
    echo      [ERROR] Upload failed
    pause
    exit /b 1
)
echo      Uploaded successfully

echo      Building and starting on remote server...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER_IP% "cd /root && rm -rf marketplace_server_new && mkdir -p marketplace_server_new && cd marketplace_server_new && tar -xzf /tmp/marketplace_server.tar.gz && go mod tidy && CGO_ENABLED=0 go build -o marketplace_server . && cd /root && pkill -f 'marketplace_server' || true && sleep 2 && rm -rf marketplace_server_old && mv marketplace_server marketplace_server_old 2>/dev/null || true && mv marketplace_server_new marketplace_server && cd marketplace_server && chmod +x start.sh && ./start.sh && sleep 3 && curl -s http://localhost:8088/ | head -5 && echo 'Marketplace Server started successfully' || echo 'WARNING: Health check failed'" 2>NUL

if errorlevel 1 (
    echo      [WARN] Deployment may have issues, check logs
) else (
    echo      [OK] Marketplace Server deployed
)
echo.

REM ==========================================
REM 5. 验证服务状态
REM ==========================================
echo [5/6] Verifying services...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER_IP% "pgrep -f 'license_server' > /dev/null && echo '  [OK] License Server running' || echo '  [X] License Server NOT running'" 2>NUL
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER_IP% "pgrep -f 'marketplace_server' > /dev/null && echo '  [OK] Marketplace Server running' || echo '  [X] Marketplace Server NOT running'" 2>NUL
echo.

REM ==========================================
REM 6. 清理临时文件
REM ==========================================
echo [6/6] Cleanup...
del /f /q "%TEMP%\license_server.tar.gz" 2>NUL
del /f /q "%TEMP%\marketplace_server.tar.gz" 2>NUL
echo      Temporary files removed
echo.

REM ==========================================
echo ==========================================
echo 部署完成！
echo ==========================================
echo.
echo License Server:     http://license.vantagedata.chat:8080
echo Marketplace Server: http://market.vantagedata.chat:8088
echo.
echo 管理面板:
echo   License:     http://license.vantagedata.chat:8080/admin/
echo   Marketplace: http://market.vantagedata.chat:8088/admin/
echo.
echo 查看日志:
echo   ssh root@license.vantagedata.chat "tail -f /root/license_server/server.log"
echo   ssh root@market.vantagedata.chat "tail -f /root/marketplace_server/server.log"
echo.

pause
endlocal
