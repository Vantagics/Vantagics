@echo off
REM License Server Windows Build Script
REM Requires: Go, GCC (MinGW-w64) for CGO (go-sqlcipher)
REM Install MinGW-w64: https://www.mingw-w64.org/ or via MSYS2

setlocal

echo ==========================================
echo VantageData License Server - Windows Build
echo ==========================================

REM Check for GCC (required by go-sqlcipher)
where gcc >nul 2>nul
if errorlevel 1 (
    echo Error: GCC not found. go-sqlcipher requires CGO.
    echo Please install MinGW-w64 or MSYS2 and add gcc to PATH.
    echo   MSYS2: https://www.msys2.org/
    echo   Then: pacman -S mingw-w64-x86_64-gcc
    exit /b 1
)

REM Build from the license_server directory
pushd "%~dp0.."

echo.
echo Building license_server.exe ...
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
go build -o build\license_server.exe -ldflags="-s -w" .
if errorlevel 1 (
    echo Error: Build failed!
    popd
    exit /b 1
)

popd

echo.
echo ==========================================
echo Build Complete
echo ==========================================
echo Output: tools\license_server\build\license_server.exe
