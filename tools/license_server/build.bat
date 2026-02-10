@echo off
setlocal enabledelayedexpansion

REM VantageData License Server Build Script (Windows)
REM Builds for Windows locally and Linux on remote server

set BUILD_DIR=build
if not exist "%BUILD_DIR%" mkdir "%BUILD_DIR%"

REM Remote server config
set SERVER=license.vantagedata.chat
set USER=root
set PASS=sunion123
set REMOTE_DIR=/root/license_server

REM SSH options to handle host key changes
set SSH_OPTS=-o StrictHostKeyChecking=no -o UserKnownHostsFile=NUL

echo ==========================================
echo VantageData License Server Build (Windows)
echo ==========================================

REM Build Windows version locally
echo.
echo [1/2] Building for Windows...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -o "%BUILD_DIR%\license_server.exe" .
if errorlevel 1 (
    echo      Build failed!
    exit /b 1
)
echo      Done: %BUILD_DIR%\license_server.exe

echo      Building reset_password tool...
go build -o "%BUILD_DIR%\reset_password.exe" ./cmd/reset_password/
if errorlevel 1 (
    echo      reset_password build failed!
    exit /b 1
)
echo      Done: %BUILD_DIR%\reset_password.exe

REM Build Linux version on remote server
echo.
echo [2/2] Building for Linux on %SERVER%...

echo      Creating remote directory...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "mkdir -p %REMOTE_DIR%"
if errorlevel 1 (
    echo      [WARN] sshpass not found, trying ssh directly...
    echo      Please enter password when prompted: %PASS%
    ssh %SSH_OPTS% %USER%@%SERVER% "mkdir -p %REMOTE_DIR%"
)

echo      Uploading source files...
sshpass -p "%PASS%" scp %SSH_OPTS% main.go go.mod go.sum %USER%@%SERVER%:%REMOTE_DIR%/ 2>NUL
if errorlevel 1 (
    scp %SSH_OPTS% main.go go.mod go.sum %USER%@%SERVER%:%REMOTE_DIR%/
)

sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "mkdir -p %REMOTE_DIR%/templates" 2>NUL
if errorlevel 1 (
    ssh %SSH_OPTS% %USER%@%SERVER% "mkdir -p %REMOTE_DIR%/templates"
)

sshpass -p "%PASS%" scp %SSH_OPTS% templates\*.go %USER%@%SERVER%:%REMOTE_DIR%/templates/ 2>NUL
if errorlevel 1 (
    scp %SSH_OPTS% templates\*.go %USER%@%SERVER%:%REMOTE_DIR%/templates/
)

sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "mkdir -p %REMOTE_DIR%/cmd/reset_password" 2>NUL
if errorlevel 1 (
    ssh %SSH_OPTS% %USER%@%SERVER% "mkdir -p %REMOTE_DIR%/cmd/reset_password"
)

sshpass -p "%PASS%" scp %SSH_OPTS% cmd\reset_password\main.go %USER%@%SERVER%:%REMOTE_DIR%/cmd/reset_password/ 2>NUL
if errorlevel 1 (
    scp %SSH_OPTS% cmd\reset_password\main.go %USER%@%SERVER%:%REMOTE_DIR%/cmd/reset_password/
)

echo      Compiling on server...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "cd %REMOTE_DIR% && go mod tidy && CGO_ENABLED=1 go build -o license_server . && CGO_ENABLED=1 go build -o reset_password ./cmd/reset_password/" 2>NUL
if errorlevel 1 (
    ssh %SSH_OPTS% %USER%@%SERVER% "cd %REMOTE_DIR% && go mod tidy && CGO_ENABLED=1 go build -o license_server . && CGO_ENABLED=1 go build -o reset_password ./cmd/reset_password/"
)

echo      Done: %REMOTE_DIR%/license_server

echo.
echo [3/3] Restarting server...
sshpass -p "%PASS%" ssh %SSH_OPTS% %USER%@%SERVER% "/root/runsrv.sh" 2>NUL
if errorlevel 1 (
    ssh %SSH_OPTS% %USER%@%SERVER% "/root/runsrv.sh"
)
echo      Server restarted

echo.
echo ==========================================
echo Build ^& Deploy Complete
echo ==========================================
echo.
echo Windows: %BUILD_DIR%\license_server.exe
echo Linux:   %USER%@%SERVER%:%REMOTE_DIR%/license_server
echo.
echo Server is running on %SERVER%

endlocal
