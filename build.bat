@echo off
REM VantageData Build Script

REM Proxy settings (SOCKS5)
set "HTTP_PROXY=socks5://127.0.0.1:10808"
set "HTTPS_PROXY=socks5://127.0.0.1:10808"

REM Ensure GOPATH\bin is in PATH for wails, makefat, etc.
for /f "delims=" %%i in ('go env GOPATH') do set "GOPATH_DIR=%%i"
set "PATH=%~dp0bin;%GOPATH_DIR%\bin;%PATH%"

REM Add NSIS to PATH (using short path name)
set "PATH=C:\PROGRA~2\NSIS;C:\PROGRA~1\NSIS;%PATH%"

set "SRC_DIR=src"
set "DIST_DIR=dist"
set "BUILD_DIR=src\build\bin"
set "OUTPUT_NAME=vantagedata"

REM Parse command line arguments
set "COMMAND=%~1"
if "%COMMAND%"=="" set "COMMAND=build"

if /i "%COMMAND%"=="clean" goto :clean
if /i "%COMMAND%"=="build" goto :build_all
if /i "%COMMAND%"=="windows" goto :build_windows

:build_all
if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"
call :build_windows
if errorlevel 1 exit /b 1
call :build_tools
if errorlevel 1 exit /b 1
exit /b 0

:build_windows
echo [Windows] Building (Dynamic Link - External DLL)...
set "DUCKDB_LIB_DIR=%~dp0libduckDB\windows"
cd /d "%SRC_DIR%"
set CGO_ENABLED=1
set "CC=%~dp0bin\zcc.bat"
set "CXX=%~dp0bin\zxx.bat"

set "CGO_LDFLAGS=-L%DUCKDB_LIB_DIR% -lduckdb -lws2_32 -lbcrypt -lcrypt32 -lole32 -luser32 -lshell32 -ladvapi32 -lrstrtmgr -lpsapi -lstdc++"
set "CGO_CFLAGS=-I%DUCKDB_LIB_DIR%"
set "CGO_LDFLAGS_ALLOW=.*"

REM Add DLL directory to PATH so wails can run the binary to generate bindings
set "PATH=%DUCKDB_LIB_DIR%;%PATH%"

call wails build -clean -platform windows/amd64 -ldflags="-H windowsgui" -nsis
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
if exist "%BUILD_DIR%\VantageData-amd64-installer.exe" (
    copy /y "%BUILD_DIR%\VantageData-amd64-installer.exe" "%DIST_DIR%\" >nul
    echo NSIS installer copied to %DIST_DIR%\VantageData-amd64-installer.exe
) else (
    echo Warning: NSIS installer not found
)
exit /b 0

:build_tools
echo.
echo [Tools] Building standalone tools...
set "TOOLS_OUTPUT_DIR=%DIST_DIR%\tools"
if not exist "%TOOLS_OUTPUT_DIR%" mkdir "%TOOLS_OUTPUT_DIR%"

REM Build appdata_manager for Windows
echo   Building appdata_manager (Windows)...
cd /d "tools\appdata_manager"
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -o "..\..\%DIST_DIR%\tools\appdata_manager.exe" .
if errorlevel 1 (
    echo Error: appdata_manager Windows build failed!
    cd /d ..\..
    exit /b 1
)

cd /d ..\..
echo   appdata_manager built successfully.

REM Build license_server for Windows
echo   Building license_server (Windows)...
pushd "tools\license_server"
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1
go build -o "../../%DIST_DIR%/tools/license_server.exe" -ldflags="-s -w" .
if errorlevel 1 (
    echo Error: license_server Windows build failed!
    popd
    exit /b 1
)
popd
echo   license_server built successfully.

echo.
echo Tools directory: %DIST_DIR%\tools
exit /b 0

:clean
if exist "%DIST_DIR%" rmdir /s /q "%DIST_DIR%"
if exist "tools\appdata_manager\appdata_manager.exe" del /q "tools\appdata_manager\appdata_manager.exe"
if exist "tools\appdata_manager\appdata_manager" del /q "tools\appdata_manager\appdata_manager"
if exist "tools\license_server\build\license_server.exe" del /q "tools\license_server\build\license_server.exe"
if exist "tools\license_server\build\license_server_macos" del /q "tools\license_server\build\license_server_macos"
exit /b 0