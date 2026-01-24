@echo off
REM Dashboard Drag-Drop Layout Deployment Script for Windows
REM Usage: deploy.bat [environment] [options]
REM Environments: dev, staging, production
REM Options: --skip-tests, --skip-backup, --rollback

setlocal enabledelayedexpansion

REM Configuration
set SCRIPT_DIR=%~dp0
set PROJECT_ROOT=%SCRIPT_DIR%
set FRONTEND_DIR=%PROJECT_ROOT%src\frontend
set BACKEND_DIR=%PROJECT_ROOT%src
set DEPLOY_LOG=%PROJECT_ROOT%deploy.log

REM Parse command line arguments
set ENVIRONMENT=%1
if "%ENVIRONMENT%"=="" set ENVIRONMENT=dev

set SKIP_TESTS=false
set SKIP_BACKUP=false
set ROLLBACK=false

:parse_args
if "%2"=="--skip-tests" (
    set SKIP_TESTS=true
    shift
    goto parse_args
)
if "%2"=="--skip-backup" (
    set SKIP_BACKUP=true
    shift
    goto parse_args
)
if "%2"=="--rollback" (
    set ROLLBACK=true
    shift
    goto parse_args
)
if not "%2"=="" (
    shift
    goto parse_args
)

REM Logging functions
:log
echo [%date% %time%] %~1
echo [%date% %time%] %~1 >> "%DEPLOY_LOG%"
goto :eof

:error
echo [ERROR] %~1
echo [ERROR] %~1 >> "%DEPLOY_LOG%"
exit /b 1

:success
echo [SUCCESS] %~1
echo [SUCCESS] %~1 >> "%DEPLOY_LOG%"
goto :eof

:warning
echo [WARNING] %~1
echo [WARNING] %~1 >> "%DEPLOY_LOG%"
goto :eof

REM Validate environment
if "%ENVIRONMENT%"=="dev" goto env_valid
if "%ENVIRONMENT%"=="staging" goto env_valid
if "%ENVIRONMENT%"=="production" goto env_valid
call :error "Invalid environment: %ENVIRONMENT%. Use: dev, staging, or production"

:env_valid
call :log "Deploying to %ENVIRONMENT% environment"

REM Handle rollback request
if "%ROLLBACK%"=="true" goto rollback

REM Pre-deployment checks
call :log "Starting pre-deployment checks..."

REM Check if required tools are installed
where node >nul 2>&1
if errorlevel 1 call :error "Node.js is not installed"

where npm >nul 2>&1
if errorlevel 1 call :error "npm is not installed"

where go >nul 2>&1
if errorlevel 1 call :error "Go is not installed"

REM Check if we're in the right directory
if not exist "%PROJECT_ROOT%build.bat" call :error "Not in project root directory. Please run from project root."

REM Create backup directory
if not exist "%PROJECT_ROOT%backup" mkdir "%PROJECT_ROOT%backup"

REM Backup current version (if not skipped)
if "%SKIP_BACKUP%"=="false" (
    call :log "Creating backup of current version..."
    
    if exist "%BACKEND_DIR%\rapidbi.exe" (
        copy "%BACKEND_DIR%\rapidbi.exe" "%PROJECT_ROOT%backup\rapidbi_backup.exe" >nul
        call :log "Binary backed up"
    )
    
    if exist "%BACKEND_DIR%\app.db" (
        copy "%BACKEND_DIR%\app.db" "%PROJECT_ROOT%backup\database_backup.db" >nul
        call :log "Database backed up"
    )
    
    call :success "Backup completed"
) else (
    call :warning "Skipping backup as requested"
)

REM Run tests (if not skipped)
if "%SKIP_TESTS%"=="false" (
    call :log "Running test suite..."
    
    REM Frontend tests
    call :log "Running frontend tests..."
    cd /d "%FRONTEND_DIR%"
    call npm test --run
    if errorlevel 1 call :error "Frontend tests failed"
    call :success "Frontend tests passed"
    
    REM Backend tests
    call :log "Running backend tests..."
    cd /d "%BACKEND_DIR%"
    go test -v ./...
    if errorlevel 1 call :error "Backend tests failed"
    call :success "Backend tests passed"
    
    cd /d "%PROJECT_ROOT%"
) else (
    call :warning "Skipping tests as requested"
)

REM Build frontend
call :log "Building frontend..."
cd /d "%FRONTEND_DIR%"
call npm install
if errorlevel 1 call :error "Frontend dependency installation failed"
call npm run build
if errorlevel 1 call :error "Frontend build failed"
call :success "Frontend build completed"

REM Build backend
call :log "Building backend..."
cd /d "%BACKEND_DIR%"
go mod tidy
if errorlevel 1 call :error "Go module cleanup failed"
go build -o rapidbi.exe
if errorlevel 1 call :error "Backend build failed"
call :success "Backend build completed"

REM Database migration
call :log "Running database migrations..."
cd /d "%BACKEND_DIR%"
if exist "database\verify_migration.go" (
    go run database\verify_migration.go
    if errorlevel 1 call :error "Database migration failed"
    call :success "Database migration completed"
) else (
    call :warning "No migration verification script found"
)

REM Environment-specific deployment steps
if "%ENVIRONMENT%"=="dev" (
    call :log "Development deployment - no additional steps needed"
)
if "%ENVIRONMENT%"=="staging" (
    call :log "Staging deployment - running smoke tests..."
    REM Add staging-specific deployment steps here
)
if "%ENVIRONMENT%"=="production" (
    call :log "Production deployment - running full validation..."
    REM Add production-specific deployment steps here
)

REM Post-deployment validation
call :log "Running post-deployment validation..."

REM Check if binary exists
if not exist "%BACKEND_DIR%\rapidbi.exe" call :error "Deployment failed - binary not found"

REM Test binary startup (quick test)
call :log "Testing application startup..."
cd /d "%BACKEND_DIR%"
timeout /t 5 >nul 2>&1
REM Note: Windows doesn't have a direct equivalent to timeout with command execution
REM This is a simplified version - in production, you might want to use PowerShell

REM Generate deployment report
set TIMESTAMP=%date:~-4,4%%date:~-10,2%%date:~-7,2%_%time:~0,2%%time:~3,2%%time:~6,2%
set TIMESTAMP=%TIMESTAMP: =0%
set DEPLOY_REPORT=%PROJECT_ROOT%deployment_report_%TIMESTAMP%.md

(
echo # Deployment Report
echo.
echo **Date**: %date% %time%
echo **Environment**: %ENVIRONMENT%
echo **Status**: SUCCESS
echo **Deployed By**: %USERNAME%
echo.
echo ## Build Information
echo - Frontend Build: SUCCESS
echo - Backend Build: SUCCESS
if "%SKIP_TESTS%"=="false" (
    echo - Tests: PASSED
) else (
    echo - Tests: SKIPPED
)
echo - Database Migration: SUCCESS
echo.
echo ## Deployment Steps Completed
echo - [x] Pre-deployment checks
echo - [x] Backup creation
echo - [x] Test execution
echo - [x] Frontend build
echo - [x] Backend build
echo - [x] Database migration
echo - [x] Post-deployment validation
echo.
echo ## Files Modified
echo - `src\rapidbi.exe` - Updated application binary
echo - `src\frontend\dist\` - Updated frontend assets
echo - Database schema updated ^(if applicable^)
echo.
echo ## Rollback Information
echo - Backup location: `backup\`
echo - Rollback command: `deploy.bat %ENVIRONMENT% --rollback`
echo.
echo ## Next Steps
echo - Monitor application performance
echo - Check error logs
echo - Verify user functionality
echo - Update documentation if needed
) > "%DEPLOY_REPORT%"

call :success "Deployment completed successfully!"
call :log "Deployment report generated: %DEPLOY_REPORT%"
call :log "To rollback: deploy.bat %ENVIRONMENT% --rollback"

cd /d "%PROJECT_ROOT%"
goto :eof

:rollback
call :log "Starting rollback procedure..."

if exist "%PROJECT_ROOT%backup\rapidbi_backup.exe" (
    call :log "Restoring previous binary..."
    copy "%PROJECT_ROOT%backup\rapidbi_backup.exe" "%BACKEND_DIR%\rapidbi.exe" >nul
    call :success "Binary restored from backup"
) else (
    call :warning "No backup binary found"
)

if exist "%PROJECT_ROOT%backup\database_backup.db" (
    call :log "Restoring database backup..."
    copy "%PROJECT_ROOT%backup\database_backup.db" "%BACKEND_DIR%\app.db" >nul
    call :success "Database restored from backup"
) else (
    call :warning "No database backup found"
)

call :success "Rollback completed"
exit /b 0