@echo off
setlocal enabledelayedexpansion

REM Vantagics Marketplace Server - One-Click Build & Deploy
REM Target: market.vantagics.com:8088

set BUILD_DIR=build
if not exist "%BUILD_DIR%" mkdir "%BUILD_DIR%"

REM Remote server config
set SERVER=market.vantagics.com
set USER=root
set PASS=sunion123
set REMOTE_DIR=/root/marketplace_server
set PORT=8088

REM SSH options
set SSH_OPTS=-o StrictHostKeyChecking=no -o UserKnownHostsFile=NUL

echo ==========================================
echo Vantagics Marketplace Server Deploy
echo Target: %SERVER%:%PORT%
echo ==========================================

REM Step 1: Build Windows version locally
echo.
echo [1/4] Building for Windows...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -o "%BUILD_DIR%\marketplace_server.exe" .
if errorlevel 1 (
    echo      Build failed!
    exit /b 1
)
echo      Done: %BUILD_DIR%\marketplace_server.exe

REM Step 2: Create remote directory and upload source
echo.
echo [2/4] Uploading source to %SERVER%...

sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "mkdir -p %REMOTE_DIR%/templates" 2>NUL
if errorlevel 1 (
    echo      [WARN] sshpass not found, trying ssh directly...
    ssh %SSH_OPTS% %USER%@%SERVER% "mkdir -p %REMOTE_DIR%/templates"
)

echo      Uploading Go source files...
sshpass -p "%PASS%" scp %SSH_OPTS% main.go go.mod go.sum %USER%@%SERVER%:%REMOTE_DIR%/ 2>NUL
if errorlevel 1 (
    scp %SSH_OPTS% main.go go.mod go.sum %USER%@%SERVER%:%REMOTE_DIR%/
)

echo      Uploading templates...
sshpass -p "%PASS%" scp %SSH_OPTS% templates\*.go %USER%@%SERVER%:%REMOTE_DIR%/templates/ 2>NUL
if errorlevel 1 (
    scp %SSH_OPTS% templates\*.go %USER%@%SERVER%:%REMOTE_DIR%/templates/
)

REM Step 3: Build on remote server
echo.
echo [3/4] Compiling on %SERVER%...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "cd %REMOTE_DIR% && go mod tidy && CGO_ENABLED=0 go build -o marketplace_server ." 2>NUL
if errorlevel 1 (
    ssh %SSH_OPTS% %USER%@%SERVER% "cd %REMOTE_DIR% && go mod tidy && CGO_ENABLED=0 go build -o marketplace_server ."
)
echo      Done: %REMOTE_DIR%/marketplace_server

REM Step 4: Restart service
echo.
echo [4/4] Deploying and restarting service...

echo      Stopping existing server...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "pkill -f 'marketplace_server' || true" 2>NUL
if errorlevel 1 (
    ssh %SSH_OPTS% %USER%@%SERVER% "pkill -f 'marketplace_server' || true"
)

echo      Uploading start script...
sshpass -p "%PASS%" scp %SSH_OPTS% start.sh %USER%@%SERVER%:%REMOTE_DIR%/ 2>NUL
if errorlevel 1 (
    scp %SSH_OPTS% start.sh %USER%@%SERVER%:%REMOTE_DIR%/
)

echo      Starting new server...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "cd %REMOTE_DIR% && sed -i 's/\r$//' start.sh && chmod +x start.sh && ./start.sh" 2>NUL
if errorlevel 1 (
    ssh %SSH_OPTS% %USER%@%SERVER% "cd %REMOTE_DIR% && sed -i 's/\r$//' start.sh && chmod +x start.sh && ./start.sh"
)

echo      Checking server status...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "sleep 2 && pgrep -f 'marketplace_server' > /dev/null && echo 'Server started successfully on port %PORT%' || (echo 'ERROR: Server failed to start' && tail -20 %REMOTE_DIR%/server.log)" 2>NUL
if errorlevel 1 (
    ssh %SSH_OPTS% %USER%@%SERVER% "sleep 2 && pgrep -f 'marketplace_server' > /dev/null && echo 'Server started successfully on port %PORT%' || (echo 'ERROR: Server failed to start' && tail -20 %REMOTE_DIR%/server.log)"
)

echo.
echo ==========================================
echo Deploy Complete
echo ==========================================
echo.
echo Windows: %BUILD_DIR%\marketplace_server.exe
echo Linux:   %USER%@%SERVER%:%REMOTE_DIR%/marketplace_server
echo.
echo Service: http://%SERVER%:%PORT%
echo Admin:   http://%SERVER%:%PORT%/admin/
echo API:     http://%SERVER%:%PORT%/api/packs
echo.

endlocal
