@echo off
REM VantageData Build Script (Optimized for Multi-core)

REM Proxy settings (SOCKS5)
set "HTTP_PROXY=socks5://127.0.0.1:10808"
set "HTTPS_PROXY=socks5://127.0.0.1:10808"

REM Ensure GOPATH\bin is in PATH for wails, makefat, etc.
for /f "delims=" %%i in ('go env GOPATH') do set "GOPATH_DIR=%%i"
set "PATH=%~dp0bin;%GOPATH_DIR%\bin;%PATH%"

REM Add NSIS to PATH (using short path name)
set "PATH=C:\PROGRA~2\NSIS;C:\PROGRA~1\NSIS;%PATH%"

REM Enable Go parallel compilation (use all CPU cores)
REM GOMAXPROCS controls runtime, but go build uses all cores by default
REM Set build cache to speed up incremental builds
for /f "delims=" %%i in ('go env GOCACHE') do set "GOCACHE=%%i"
if not exist "%GOCACHE%" mkdir "%GOCACHE%"

set "SRC_DIR=src"
set "DIST_DIR=dist"
set "BUILD_DIR=src\build\bin"
set "OUTPUT_NAME=vantagedata"

REM Parse command line arguments
set "COMMAND=%~1"
if "%COMMAND%"=="" set "COMMAND=build"
set "SKIP_NSIS="
set "SKIP_FRONTEND="
set "SKIP_TOOLS="

REM Parse optional flags
:parse_args
shift
if "%~1"=="" goto :done_args
if /i "%~1"=="--skip-nsis" set "SKIP_NSIS=1"
if /i "%~1"=="--skip-frontend" set "SKIP_FRONTEND=1"
if /i "%~1"=="--skip-tools" set "SKIP_TOOLS=1"
if /i "%~1"=="--fast" (
    set "SKIP_NSIS=1"
)
goto :parse_args
:done_args

if /i "%COMMAND%"=="clean" goto :clean
if /i "%COMMAND%"=="build" goto :build_all
if /i "%COMMAND%"=="windows" goto :build_windows
if /i "%COMMAND%"=="tools" goto :build_tools
if /i "%COMMAND%"=="fast" (
    set "SKIP_NSIS=1"
    goto :build_all
)

:build_all
if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"
call :build_windows
if errorlevel 1 exit /b 1
if not defined SKIP_TOOLS (
    call :build_tools
    if errorlevel 1 exit /b 1
)
exit /b 0

:build_windows
echo [Windows] Building (Dynamic Link - External DLL)...
set "DUCKDB_LIB_DIR=%~dp0libduckDB\windows"
cd /d "%SRC_DIR%"
set CGO_ENABLED=1
set "CC=%~dp0bin\zcc.bat"
set "CXX=%~dp0bin\zxx.bat"

set "CGO_LDFLAGS=-L%DUCKDB_LIB_DIR% -lduckdb -lws2_32 -lbcrypt -lcrypt32 -lole32 -luser32 -lshell32 -ladvapi32 -lrstrtmgr -lpsapi -lstdc++ -Wl,--subsystem,windows"
set "CGO_CFLAGS=-I%DUCKDB_LIB_DIR%"
set "CGO_LDFLAGS_ALLOW=.*"

REM Add DLL directory to PATH so wails can run the binary to generate bindings
set "PATH=%DUCKDB_LIB_DIR%;%PATH%"

REM Build wails command with options
set WAILS_CMD=wails build -platform windows/amd64

REM Skip -clean to leverage incremental builds (much faster)
REM Use -clean only when you need a full rebuild

REM Skip NSIS installer generation for faster dev builds
if not defined SKIP_NSIS (
    set "WAILS_CMD=%WAILS_CMD% -nsis"
)

REM Skip frontend rebuild if unchanged (saves ~20-30s)
if defined SKIP_FRONTEND (
    set "WAILS_CMD=%WAILS_CMD% -s"
)

REM Enable verbose output to see parallel compilation
if defined VERBOSE (
    set "WAILS_CMD=%WAILS_CMD% -v 2"
)

REM -ldflags "-H windowsgui" ensures no console window is shown at runtime
REM -s -w strips debug info and DWARF symbols to reduce binary size
REM Append ldflags LAST to avoid quoting issues in CMD
set WAILS_CMD=%WAILS_CMD% -ldflags "-H windowsgui -s -w"

call %WAILS_CMD%
if errorlevel 1 (
    echo Error: Windows build failed!
    exit /b 1
)
cd /d ..
if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"
if exist "%BUILD_DIR%\%OUTPUT_NAME%.exe" (
    copy /y "%BUILD_DIR%\%OUTPUT_NAME%.exe" "%DIST_DIR%\" >nul
    copy /y "libduckDB\windows\duckdb.dll" "%DIST_DIR%\" >nul
    echo Windows build and duckdb.dll copied to %DIST_DIR%\
) else (
    echo Warning: %BUILD_DIR%\%OUTPUT_NAME%.exe not found
)
REM Copy NSIS installer
if not defined SKIP_NSIS (
    if exist "%BUILD_DIR%\VantageData-amd64-installer.exe" (
        copy /y "%BUILD_DIR%\VantageData-amd64-installer.exe" "%DIST_DIR%\" >nul
        echo NSIS installer copied to %DIST_DIR%\VantageData-amd64-installer.exe
    ) else (
        echo Warning: NSIS installer not found
    )
)
exit /b 0

:build_tools
echo.
echo [Tools] Building standalone tools (parallel)...
set "TOOLS_OUTPUT_DIR=%DIST_DIR%\tools"
if not exist "%TOOLS_OUTPUT_DIR%" mkdir "%TOOLS_OUTPUT_DIR%"

REM Create temporary batch files for parallel builds
echo @echo off > "%TEMP%\build_appdata.bat"
echo cd /d "%~dp0tools\appdata_manager" >> "%TEMP%\build_appdata.bat"
echo set GOOS=windows>> "%TEMP%\build_appdata.bat"
echo set GOARCH=amd64>> "%TEMP%\build_appdata.bat"
echo set CGO_ENABLED=0>> "%TEMP%\build_appdata.bat"
echo go build -ldflags="-s -w" -o "..\..\%DIST_DIR%\tools\appdata_manager.exe" . >> "%TEMP%\build_appdata.bat"
echo if errorlevel 1 (echo [ERROR] appdata_manager build failed! ^& exit /b 1) else (echo [OK] appdata_manager built successfully.) >> "%TEMP%\build_appdata.bat"

echo @echo off > "%TEMP%\build_license.bat"
echo cd /d "%~dp0tools\license_server" >> "%TEMP%\build_license.bat"
echo set GOOS=windows>> "%TEMP%\build_license.bat"
echo set GOARCH=amd64>> "%TEMP%\build_license.bat"
echo set CGO_ENABLED=1>> "%TEMP%\build_license.bat"
echo go build -ldflags="-s -w" -o "..\..\%DIST_DIR%\tools\license_server.exe" . >> "%TEMP%\build_license.bat"
echo if errorlevel 1 (echo [ERROR] license_server build failed! ^& exit /b 1) else (echo [OK] license_server built successfully.) >> "%TEMP%\build_license.bat"

REM Clean up signal files
del /q "%TEMP%\build_appdata.done" 2>nul
del /q "%TEMP%\build_license.done" 2>nul

REM Append signal file creation to each build script
echo echo done ^> "%TEMP%\build_appdata.done" >> "%TEMP%\build_appdata.bat"
echo echo done ^> "%TEMP%\build_license.done" >> "%TEMP%\build_license.bat"

REM Launch parallel builds (no pause - let them exit naturally)
echo   Starting parallel builds...
start /MIN "" cmd /c "call "%TEMP%\build_appdata.bat""
start /MIN "" cmd /c "call "%TEMP%\build_license.bat""

REM Wait for both builds to complete via signal files
echo   Waiting for builds to complete...
:wait_tools
timeout /t 1 /nobreak >nul 2>&1
if not exist "%TEMP%\build_appdata.done" goto :wait_tools
if not exist "%TEMP%\build_license.done" goto :wait_tools

REM Check if builds succeeded
if not exist "%DIST_DIR%\tools\appdata_manager.exe" (
    echo   Error: appdata_manager.exe not found!
    exit /b 1
)
if not exist "%DIST_DIR%\tools\license_server.exe" (
    echo   Error: license_server.exe not found!
    exit /b 1
)

REM Cleanup temp files
del /q "%TEMP%\build_appdata.bat" 2>nul
del /q "%TEMP%\build_license.bat" 2>nul
del /q "%TEMP%\build_appdata.done" 2>nul
del /q "%TEMP%\build_license.done" 2>nul

echo.
echo   All tools built successfully!
echo   Tools directory: %DIST_DIR%\tools
exit /b 0

:clean
if exist "%DIST_DIR%" rmdir /s /q "%DIST_DIR%"
if exist "tools\appdata_manager\appdata_manager.exe" del /q "tools\appdata_manager\appdata_manager.exe"
if exist "tools\appdata_manager\appdata_manager" del /q "tools\appdata_manager\appdata_manager"
if exist "tools\license_server\build\license_server.exe" del /q "tools\license_server\build\license_server.exe"
if exist "tools\license_server\build\license_server_macos" del /q "tools\license_server\build\license_server_macos"
exit /b 0
