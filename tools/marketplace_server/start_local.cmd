@echo off
REM Marketplace Server Local Start Script for Windows
REM Usage: start_local.cmd

cd /d "%~dp0"

REM JWT secret - must match license_server's LICENSE_MARKETPLACE_SECRET
set MARKETPLACE_JWT_SECRET=marketplace-server-jwt-secret-key-2024

echo ==========================================
echo Starting Marketplace Server (Local)
echo Port: 8088
echo ==========================================

REM Kill existing process on port 8088
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8088" ^| findstr "LISTENING"') do (
    echo Stopping existing process on port 8088...
    taskkill /F /PID %%a 2>nul
)

REM Start server
echo Starting server...
start /B build\marketplace_server.exe -port 8088 -db marketplace_local.db > server_local.log 2>&1

timeout /t 3 /nobreak >nul

REM Verify
curl -s -o nul -w "%%{http_code}" http://localhost:8088/ | findstr "200" >nul
if %ERRORLEVEL% EQU 0 (
    echo [SUCCESS] Marketplace Server started on http://localhost:8088
) else (
    echo [WARNING] Server may not have started. Check server_local.log
)

echo.
