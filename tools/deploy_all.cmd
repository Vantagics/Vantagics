@echo off
setlocal enabledelayedexpansion

REM ==========================================
REM Vantagics deployment script
REM Targets: all | license | market | nginx
REM ==========================================

set SERVER_IP=107.172.86.131
set USER=root
set PASS=%~2
set TARGET=%~1
if "%TARGET%"=="" set TARGET=all

set SSH_OPTS=-o StrictHostKeyChecking=no -o UserKnownHostsFile=NUL
set SSH_KEY=
set PREPARED_SSH_KEY=
set PUTTY_PLINK=
set PUTTY_PSCP=
set PUTTY_HOSTKEY=SHA256:yoyEXbuT2kezyG9Y8cJDZplBMZgaPAN7+sureAkVRVE
set USE_PUTTY=0
set USE_SSHPASS=0

if defined VANTAGICS_SSH_KEY set SSH_KEY=%VANTAGICS_SSH_KEY%
if not defined SSH_KEY if exist "%USERPROFILE%\.ssh\id_rsa" set SSH_KEY=%USERPROFILE%\.ssh\id_rsa%
if not defined SSH_KEY if exist "%USERPROFILE%\.ssh\vantagics_rsa" set SSH_KEY=%USERPROFILE%\.ssh\vantagics_rsa%
if defined SSH_KEY call :prepare_ssh_key "%SSH_KEY%"
if defined PREPARED_SSH_KEY (
    set SSH_KEY=%PREPARED_SSH_KEY%
    set SSH_OPTS=%SSH_OPTS% -i "%SSH_KEY%"
)

if exist "C:\Program Files\PuTTY\plink.exe" set PUTTY_PLINK=C:\Program Files\PuTTY\plink.exe
if exist "C:\Program Files\PuTTY\pscp.exe" set PUTTY_PSCP=C:\Program Files\PuTTY\pscp.exe

echo ==========================================
echo Vantagics deployment
echo Target: %TARGET%
echo ==========================================
echo Remote server: %USER%@%SERVER_IP%
if defined SSH_KEY (
    echo SSH key: %SSH_KEY%
) else (
    echo SSH key: [not configured]
)
echo.

if defined SSH_KEY (
    echo [INFO] Using SSH key authentication.
) else (
    if defined PUTTY_PLINK if defined PUTTY_PSCP (
        if not defined PASS set /p PASS=SSH Password: 
        set USE_PUTTY=1
        echo [INFO] Using PuTTY password authentication.
    ) else (
        where sshpass >nul 2>&1
        if errorlevel 1 (
            echo [WARN] No SSH key, PuTTY, or sshpass available.
            echo [WARN] Native ssh/scp will prompt multiple times.
        ) else (
            if not defined PASS set /p PASS=SSH Password: 
            set USE_SSHPASS=1
            echo [INFO] Using password authentication via sshpass.
        )
    )
)
echo.

if /i "%TARGET%"=="license" goto :deploy_license
if /i "%TARGET%"=="market" goto :deploy_market
if /i "%TARGET%"=="nginx" goto :deploy_nginx
if /i "%TARGET%"=="all" goto :deploy_all
echo [ERROR] Unknown target: %TARGET%
echo Usage: deploy_all.cmd [all^|license^|market^|nginx] [password]
exit /b 1

:deploy_all
call :deploy_license
if errorlevel 1 goto :cleanup_fail
call :deploy_market
if errorlevel 1 goto :cleanup_fail
call :deploy_nginx
if errorlevel 1 goto :cleanup_fail
goto :verify

:deploy_license
echo [License] Packaging source files...
cd /d "%~dp0license_server"
tar -czf "%TEMP%\license_server.tar.gz" main.go go.mod go.sum templates cmd start.sh 2>NUL
if errorlevel 1 (
    echo [ERROR] Failed to create license_server package
    exit /b 1
)
for %%A in ("%TEMP%\license_server.tar.gz") do echo         Package: %%~zA bytes

echo [License] Uploading package...
call :ssh_scp "%TEMP%\license_server.tar.gz" "/tmp/"
if errorlevel 1 exit /b 1

echo [License] Uploading deploy script...
call :ssh_scp "deploy_remote.sh" "/tmp/deploy_license.sh"
if errorlevel 1 exit /b 1

echo [License] Running remote deploy...
call :ssh_exec "sed -i 's/\r$//' /tmp/deploy_license.sh && bash /tmp/deploy_license.sh"
if errorlevel 1 (
    echo [WARN] License deployment may have issues
) else (
    echo [OK] License deployed
)

echo [License] Health check...
call :ssh_exec "sleep 2 && curl -sf http://localhost:6699/health > /dev/null && echo '  Auth (6699): OK' || echo '  Auth (6699): FAILED' && curl -sf -o /dev/null -w '' http://localhost:8899/ && echo '  Admin (8899): OK' || echo '  Admin (8899): FAILED'"
echo.
del /f /q "%TEMP%\license_server.tar.gz" 2>NUL
if /i "%TARGET%"=="license" goto :verify
exit /b 0

:deploy_market
echo [Market] Packaging source files...
cd /d "%~dp0marketplace_server"
tar -czf "%TEMP%\marketplace_server.tar.gz" --exclude="*_test.go" --exclude="build" *.go go.mod go.sum templates i18n start.sh logo.png 2>NUL
if errorlevel 1 (
    echo [ERROR] Failed to create marketplace_server package
    exit /b 1
)
for %%A in ("%TEMP%\marketplace_server.tar.gz") do echo         Package: %%~zA bytes

echo [Market] Uploading package...
call :ssh_scp "%TEMP%\marketplace_server.tar.gz" "/tmp/"
if errorlevel 1 exit /b 1

echo [Market] Uploading deploy script...
call :ssh_scp "deploy_remote.sh" "/tmp/deploy_market.sh"
if errorlevel 1 exit /b 1

echo [Market] Running remote deploy...
call :ssh_exec "sed -i 's/\r$//' /tmp/deploy_market.sh && bash /tmp/deploy_market.sh"
if errorlevel 1 (
    echo [WARN] Marketplace deployment may have issues
) else (
    echo [OK] Marketplace deployed
)

echo [Market] Health check...
call :ssh_exec "sleep 2 && curl -sf http://localhost:8088/ > /dev/null && echo '  Marketplace (8088): OK' || echo '  Marketplace (8088): FAILED'"
echo.
del /f /q "%TEMP%\marketplace_server.tar.gz" 2>NUL
if /i "%TARGET%"=="market" goto :verify
exit /b 0

:deploy_nginx
echo [Nginx] Uploading configuration...
cd /d "%~dp0..\deploy\nginx"
call :ssh_scp "vantagics.com.conf" "/etc/nginx/conf.d/"
if errorlevel 1 exit /b 1

echo [Nginx] Testing and reloading...
call :ssh_exec "nginx -t && nginx -s reload && echo '  Nginx: OK' || echo '  Nginx: FAILED'"
echo.
if /i "%TARGET%"=="nginx" goto :verify
exit /b 0

:verify
echo ==========================================
echo Service verification
echo ==========================================
call :ssh_exec "echo 'Process check:' && (pgrep -f '/root/license_server/license_server' > /dev/null && echo '  [OK] License Server running' || echo '  [X] License Server NOT running') && (pgrep -f '/root/marketplace_server/marketplace_server' > /dev/null && echo '  [OK] Marketplace Server running' || echo '  [X] Marketplace Server NOT running') && echo '' && echo 'Port check:' && (ss -tlnp | grep -E ':(6699|8899|8088) ' || echo '  No matching ports found') && echo '' && echo 'File check:' && echo '  /root/license_server/:' && ls -lh /root/license_server/license_server /root/license_server/start.sh 2>/dev/null && echo '  /root/marketplace_server/:' && ls -lh /root/marketplace_server/marketplace_server /root/marketplace_server/start.sh 2>/dev/null"
echo.
echo Deployment complete.
echo License: https://license.vantagics.com/
echo Marketplace: https://market.vantagics.com/
goto :cleanup_ok

:prepare_ssh_key
set PREPARED_SSH_KEY=
set "_SOURCE_SSH_KEY=%~1"
if not exist "%_SOURCE_SSH_KEY%" exit /b 0
set "_TARGET_SSH_KEY=%TEMP%\vantagics_deploy_key_%USERNAME%"
copy /y "%_SOURCE_SSH_KEY%" "%_TARGET_SSH_KEY%" >nul
if errorlevel 1 exit /b 0
icacls "%_TARGET_SSH_KEY%" /inheritance:r >nul
icacls "%_TARGET_SSH_KEY%" /grant:r "%USERNAME%:R" >nul
icacls "%_TARGET_SSH_KEY%" /remove:g "Everyone" "Users" "Authenticated Users" >nul 2>&1
set PREPARED_SSH_KEY=%_TARGET_SSH_KEY%
exit /b 0

:ssh_exec
if "%USE_PUTTY%"=="1" (
    "%PUTTY_PLINK%" -batch -hostkey "%PUTTY_HOSTKEY%" -pw "%PASS%" -no-antispoof %USER%@%SERVER_IP% %1
) else if "%USE_SSHPASS%"=="1" (
    sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER_IP% %1 2>NUL
) else (
    ssh %SSH_OPTS% %USER%@%SERVER_IP% %1
)
exit /b %errorlevel%

:ssh_scp
if "%USE_PUTTY%"=="1" (
    "%PUTTY_PSCP%" -batch -hostkey "%PUTTY_HOSTKEY%" -pw "%PASS%" %1 %USER%@%SERVER_IP%:%2
) else if "%USE_SSHPASS%"=="1" (
    sshpass -p "%PASS%" scp %SSH_OPTS% %1 %USER%@%SERVER_IP%:%2 2>NUL
) else (
    scp %SSH_OPTS% %1 %USER%@%SERVER_IP%:%2
)
exit /b %errorlevel%

:cleanup_ok
if defined PREPARED_SSH_KEY del /f /q "%PREPARED_SSH_KEY%" 2>NUL
goto :eof

:cleanup_fail
if defined PREPARED_SSH_KEY del /f /q "%PREPARED_SSH_KEY%" 2>NUL
exit /b 1
