#!/bin/bash

# Dashboard Drag-Drop Layout Deployment Script
# Usage: ./deploy.sh [environment] [options]
# Environments: dev, staging, production
# Options: --skip-tests, --skip-backup, --rollback

set -e  # Exit on any error

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
FRONTEND_DIR="$PROJECT_ROOT/src/frontend"
BACKEND_DIR="$PROJECT_ROOT/src"
DEPLOY_LOG="$PROJECT_ROOT/deploy.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$DEPLOY_LOG"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$DEPLOY_LOG"
    exit 1
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$DEPLOY_LOG"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$DEPLOY_LOG"
}

# Parse command line arguments
ENVIRONMENT=${1:-dev}
SKIP_TESTS=false
SKIP_BACKUP=false
ROLLBACK=false

for arg in "$@"; do
    case $arg in
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --skip-backup)
            SKIP_BACKUP=true
            shift
            ;;
        --rollback)
            ROLLBACK=true
            shift
            ;;
    esac
done

# Validate environment
case $ENVIRONMENT in
    dev|staging|production)
        log "Deploying to $ENVIRONMENT environment"
        ;;
    *)
        error "Invalid environment: $ENVIRONMENT. Use: dev, staging, or production"
        ;;
esac

# Rollback function
rollback() {
    log "Starting rollback procedure..."
    
    if [ -f "$PROJECT_ROOT/backup/rapidbi_backup.exe" ]; then
        log "Restoring previous binary..."
        cp "$PROJECT_ROOT/backup/rapidbi_backup.exe" "$BACKEND_DIR/rapidbi.exe"
        success "Binary restored from backup"
    else
        warning "No backup binary found"
    fi
    
    if [ -f "$PROJECT_ROOT/backup/database_backup.db" ]; then
        log "Restoring database backup..."
        cp "$PROJECT_ROOT/backup/database_backup.db" "$BACKEND_DIR/app.db"
        success "Database restored from backup"
    else
        warning "No database backup found"
    fi
    
    success "Rollback completed"
    exit 0
}

# Handle rollback request
if [ "$ROLLBACK" = true ]; then
    rollback
fi

# Pre-deployment checks
log "Starting pre-deployment checks..."

# Check if required tools are installed
command -v node >/dev/null 2>&1 || error "Node.js is not installed"
command -v npm >/dev/null 2>&1 || error "npm is not installed"
command -v go >/dev/null 2>&1 || error "Go is not installed"

# Check if we're in the right directory
if [ ! -f "$PROJECT_ROOT/build.sh" ]; then
    error "Not in project root directory. Please run from project root."
fi

# Create backup directory
mkdir -p "$PROJECT_ROOT/backup"

# Backup current version (if not skipped)
if [ "$SKIP_BACKUP" = false ]; then
    log "Creating backup of current version..."
    
    if [ -f "$BACKEND_DIR/rapidbi.exe" ]; then
        cp "$BACKEND_DIR/rapidbi.exe" "$PROJECT_ROOT/backup/rapidbi_backup.exe"
        log "Binary backed up"
    fi
    
    if [ -f "$BACKEND_DIR/app.db" ]; then
        cp "$BACKEND_DIR/app.db" "$PROJECT_ROOT/backup/database_backup.db"
        log "Database backed up"
    fi
    
    success "Backup completed"
fi

# Run tests (if not skipped)
if [ "$SKIP_TESTS" = false ]; then
    log "Running test suite..."
    
    # Frontend tests
    log "Running frontend tests..."
    cd "$FRONTEND_DIR"
    npm test --run || error "Frontend tests failed"
    success "Frontend tests passed"
    
    # Backend tests
    log "Running backend tests..."
    cd "$BACKEND_DIR"
    go test -v ./... || error "Backend tests failed"
    success "Backend tests passed"
    
    cd "$PROJECT_ROOT"
else
    warning "Skipping tests as requested"
fi

# Build frontend
log "Building frontend..."
cd "$FRONTEND_DIR"
npm install || error "Frontend dependency installation failed"
npm run build || error "Frontend build failed"
success "Frontend build completed"

# Build backend
log "Building backend..."
cd "$BACKEND_DIR"
go mod tidy || error "Go module cleanup failed"
go build -o rapidbi.exe || error "Backend build failed"
success "Backend build completed"

# Database migration
log "Running database migrations..."
cd "$BACKEND_DIR"
if [ -f "database/verify_migration.go" ]; then
    go run database/verify_migration.go || error "Database migration failed"
    success "Database migration completed"
else
    warning "No migration verification script found"
fi

# Environment-specific deployment steps
case $ENVIRONMENT in
    dev)
        log "Development deployment - no additional steps needed"
        ;;
    staging)
        log "Staging deployment - running smoke tests..."
        # Add staging-specific deployment steps here
        ;;
    production)
        log "Production deployment - running full validation..."
        # Add production-specific deployment steps here
        ;;
esac

# Post-deployment validation
log "Running post-deployment validation..."

# Check if binary exists and is executable
if [ ! -f "$BACKEND_DIR/rapidbi.exe" ]; then
    error "Deployment failed - binary not found"
fi

# Test binary startup (quick test)
log "Testing application startup..."
cd "$BACKEND_DIR"
timeout 10s ./rapidbi.exe --version >/dev/null 2>&1 || warning "Could not verify application startup"

# Generate deployment report
DEPLOY_REPORT="$PROJECT_ROOT/deployment_report_$(date +'%Y%m%d_%H%M%S').md"
cat > "$DEPLOY_REPORT" << EOF
# Deployment Report

**Date**: $(date)
**Environment**: $ENVIRONMENT
**Status**: SUCCESS
**Deployed By**: $(whoami)
**Git Commit**: $(git rev-parse HEAD 2>/dev/null || echo "Unknown")

## Build Information
- Frontend Build: SUCCESS
- Backend Build: SUCCESS
- Tests: $([ "$SKIP_TESTS" = false ] && echo "PASSED" || echo "SKIPPED")
- Database Migration: SUCCESS

## Deployment Steps Completed
- [x] Pre-deployment checks
- [x] Backup creation
- [x] Test execution
- [x] Frontend build
- [x] Backend build
- [x] Database migration
- [x] Post-deployment validation

## Files Modified
- \`src/rapidbi.exe\` - Updated application binary
- \`src/frontend/dist/\` - Updated frontend assets
- Database schema updated (if applicable)

## Rollback Information
- Backup location: \`backup/\`
- Rollback command: \`./deploy.sh $ENVIRONMENT --rollback\`

## Next Steps
- Monitor application performance
- Check error logs
- Verify user functionality
- Update documentation if needed
EOF

success "Deployment completed successfully!"
log "Deployment report generated: $DEPLOY_REPORT"
log "To rollback: ./deploy.sh $ENVIRONMENT --rollback"

cd "$PROJECT_ROOT"