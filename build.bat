@echo off
REM RapidBI Build Script for Windows
REM This script helps to build the RapidBI application using Wails.

setlocal EnableDelayedExpansion

set APP_NAME=RapidBI
set SRC_DIR=src
set BUILD_DIR=%SRC_DIR%\build\bin

REM Parse command line arguments
set COMMAND=%~1
if "%COMMAND%"=="" set COMMAND=build

REM Display help
if /i "%COMMAND%"=="help" goto :show_help
if /i "%COMMAND%"=="-h" goto :show_help
if /i "%COMMAND%"=="--help" goto :show_help

REM Execute command
if /i "%COMMAND%"=="clean" goto :clean
if /i "%COMMAND%"=="install-deps" goto :install_deps
if /i "%COMMAND%"=="build" goto :build
if /i "%COMMAND%"=="debug" goto :build_debug
if /i "%COMMAND%"=="quick" goto :quick_build

echo Unknown command: %COMMAND%
goto :show_help

:show_help
echo Usage: build.bat [command]
echo.
echo Commands:
echo   build         Build the application (default) - Full Wails build
echo   quick         Quick build (backend only) - Fast iteration for Go code
echo   debug         Build the application with debug symbols
echo   clean         Remove build artifacts
echo   install-deps  Install Go and NPM dependencies
echo   help          Show this help message
echo.
echo Example:
echo   build.bat           (Full build with Wails)
echo   build.bat quick     (Quick backend-only build)
echo   build.bat debug
echo   build.bat clean
exit /b 0

:clean
echo Cleaning build artifacts...
if exist "%BUILD_DIR%" rmdir /s /q "%BUILD_DIR%"
if exist "%SRC_DIR%\frontend\dist" rmdir /s /q "%SRC_DIR%\frontend\dist"
if exist "rapidbi.exe" del /q "rapidbi.exe"
echo Done.
exit /b 0

:install_deps
echo Checking dependencies...
where go >nul 2>nul
if errorlevel 1 (
    echo Error: Go is not installed. Please install Go from https://golang.org/
    pause
    exit /b 1
)

where npm >nul 2>nul
if errorlevel 1 (
    echo Error: NPM is not installed. Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

where wails >nul 2>nul
if errorlevel 1 (
    echo Wails CLI not found. Installing latest Wails v2...
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    if errorlevel 1 (
        echo Failed to install Wails. Please install manually.
        pause
        exit /b 1
    )
)

echo Installing Go dependencies...
cd /d "%SRC_DIR%"
call go mod download
if errorlevel 1 (
    echo Failed to download Go dependencies.
    pause
    exit /b 1
)

echo Installing NPM dependencies...
cd /d frontend
call npm install
if errorlevel 1 (
    echo Failed to install NPM dependencies.
    pause
    exit /b 1
)

echo Dependencies installed successfully.
exit /b 0

:build
echo Checking dependencies...
where go >nul 2>nul
if errorlevel 1 (
    echo Error: Go is not installed. Please install Go from https://golang.org/
    pause
    exit /b 1
)

where npm >nul 2>nul
if errorlevel 1 (
    echo Error: NPM is not installed. Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

where wails >nul 2>nul
if errorlevel 1 (
    echo Error: Wails CLI is not installed.
    echo Please run: build.bat install-deps
    pause
    exit /b 1
)

echo Starting build for %APP_NAME%...
cd /d "%SRC_DIR%"
call wails build -clean
if errorlevel 1 (
    echo Build failed!
    pause
    exit /b 1
)

echo.
echo %APP_NAME% build finished successfully!
echo Output directory: %BUILD_DIR%
exit /b 0

:build_debug
echo Checking dependencies...
where go >nul 2>nul
if errorlevel 1 (
    echo Error: Go is not installed. Please install Go from https://golang.org/
    pause
    exit /b 1
)

where npm >nul 2>nul
if errorlevel 1 (
    echo Error: NPM is not installed. Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

where wails >nul 2>nul
if errorlevel 1 (
    echo Error: Wails CLI is not installed.
    echo Please run: build.bat install-deps
    pause
    exit /b 1
)

echo Starting debug build for %APP_NAME%...
cd /d "%SRC_DIR%"
call wails build -debug
if errorlevel 1 (
    echo Build failed!
    pause
    exit /b 1
)

echo.
echo %APP_NAME% debug build finished successfully!
echo Output directory: %BUILD_DIR%
exit /b 0

:quick_build
echo Performing quick build (backend only)...
echo Note: This builds only the Go backend without frontend changes.
echo.

where go >nul 2>nul
if errorlevel 1 (
    echo Error: Go is not installed. Please install Go from https://golang.org/
    pause
    exit /b 1
)

echo Building %APP_NAME% backend...
cd /d "%SRC_DIR%"
go build -o ..\rapidbi.exe
if errorlevel 1 (
    echo Quick build failed!
    pause
    exit /b 1
)

echo.
echo %APP_NAME% quick build finished successfully!
echo Output: rapidbi.exe (in root directory)
echo.
echo TIP: Use 'build.bat build' for a full build including frontend changes.
exit /b 0
