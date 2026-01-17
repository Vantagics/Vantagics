@echo off
REM RapidBI Build Script

set "SRC_DIR=src"
set "DIST_DIR=dist"
set "BUILD_DIR=src\build\bin"
set "OUTPUT_NAME=rapidbi"

REM Parse command line arguments
set "COMMAND=%~1"
if "%COMMAND%"=="" set "COMMAND=build"

if /i "%COMMAND%"=="clean" goto :clean
if /i "%COMMAND%"=="build" goto :build_all
if /i "%COMMAND%"=="windows" goto :build_windows
if /i "%COMMAND%"=="macos" goto :build_macos

:build_all
if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"
call :build_windows
if errorlevel 1 exit /b 1
call :build_macos
if errorlevel 1 exit /b 1
exit /b 0

:build_windows
echo [Windows] Building...
cd /d "%SRC_DIR%"
set CGO_ENABLED=1
call wails build -clean -platform windows/amd64 -nsis
if errorlevel 1 (
    echo Error: Windows build failed!
    pause
    exit /b 1
)
cd /d ..
if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"
move /y "%BUILD_DIR%\%OUTPUT_NAME%.exe" "%DIST_DIR%\" >nul 2>nul
exit /b 0

:build_macos
echo [macOS] Starting...
echo DEBUG: 1
set "ZIG_EXE=zig"
echo DEBUG: 2
go install github.com/randall77/makefat@latest
if errorlevel 1 (
    echo Error: Failed to install makefat!
    pause
    exit /b 1
)
echo DEBUG: 3

cd /d "%SRC_DIR%"
set CGO_ENABLED=1
set "CC=%CD%\zcc.bat"
set "CXX=%CD%\zxx.bat"
set "ZIG_EXE=%ZIG_EXE%"
set GOOS=darwin
set GOARCH=arm64
echo DEBUG: 4
go build -o ..\%DIST_DIR%\rapidbi_arm64 -ldflags="-s -w" .
if errorlevel 1 (
    echo Error: macOS arm64 build failed!
    pause
    exit /b 1
)

set GOARCH=amd64
echo DEBUG: 5
go build -o ..\%DIST_DIR%\rapidbi_amd64 -ldflags="-s -w" .
if errorlevel 1 (
    echo Error: macOS amd64 build failed!
    pause
    exit /b 1
)

echo [macOS] Creating Universal...
cd /d ..\%DIST_DIR%
makefat rapidbi_universal rapidbi_arm64 rapidbi_amd64
if errorlevel 1 (
    echo Error: Failed to create universal binary!
    pause
    exit /b 1
)

echo [macOS] Bundling...
set "APP_BUNDLE=RapidBI.app"
if exist "%APP_BUNDLE%" rmdir /s /q "%APP_BUNDLE%"
mkdir "%APP_BUNDLE%\Contents\MacOS"
mkdir "%APP_BUNDLE%\Contents\Resources"
move /y rapidbi_universal "%APP_BUNDLE%\Contents\MacOS\%OUTPUT_NAME%" >nul
copy /y "..\src\build\Info.plist" "%APP_BUNDLE%\Contents\Info.plist" >nul
copy /y "..\src\build\appicon.png" "%APP_BUNDLE%\Contents\Resources\iconfile.png" >nul
del /q rapidbi_arm64 rapidbi_amd64

echo [macOS] Zipping App Bundle...
powershell -Command "Compress-Archive -Path '%APP_BUNDLE%' -DestinationPath 'RapidBI_macOS_Universal.zip' -Force"
if errorlevel 1 (
    echo Error: Failed to create zip archive!
    pause
    exit /b 1
)

cd /d ..
exit /b 0

:clean
if exist "%DIST_DIR%" rmdir /s /q "%DIST_DIR%"
exit /b 0