@echo off
REM License Server Local Start Script for Windows
REM Usage: start_local.cmd

cd /d "%~dp0"

REM Marketplace JWT secret - must match marketplace_server's MARKETPLACE_JWT_SECRET
set LICENSE_MARKETPLACE_SECRET=marketplace-server-jwt-secret-key-2024
set LICENSE_DB_PASSWORD=vantagedata2024
set LICENSE_ADMIN_PASSWORD=admin123

echo ==========================================
echo Starting License Server (Local)
echo Port: 8080
echo ==========================================

REM Kill existing process on port 8080
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8080" ^| findstr "LISTENING"') do (
    echo Stopping existing process on port 8080...
    taskkill /F /PID %%a 2>nul
)

REM Start server
echo Starting server...
start /B build\license_server.exe -port 8080 -db license_local.db -templates templates > server_local.log 2>&1

timeout /t 3 /nobreak >nul

REM Verify
curl -s -o nul -w "%%{http_code}" http://localhost:8080/api/health | findstr "200" >nul
if %ERRORLEVEL% EQU 0 (
    echo [SUCCESS] License Server started on http://localhost:8080
    echo Admin: http://localhost:8080/admin/
    echo API: http://localhost:8080/api/health
) else (
    echo [WARNING] Server may not have started. Check server_local.log
)

echo.
