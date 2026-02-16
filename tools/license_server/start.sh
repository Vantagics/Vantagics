#!/bin/bash

# License Server Start Script
# Usage: ./start.sh
# Starts both auth server (port 6699) and management server (port 8899)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Set environment variables (override defaults for production)
export LICENSE_DB_PASSWORD="${LICENSE_DB_PASSWORD:-sunion123!}"
export LICENSE_ADMIN_PASSWORD="${LICENSE_ADMIN_PASSWORD:-sunion123}"
export LICENSE_MARKETPLACE_SECRET="${LICENSE_MARKETPLACE_SECRET:-marketplace-server-jwt-secret-key-2024}"

echo "Starting License Server..."
echo "  DB Password: ${LICENSE_DB_PASSWORD:0:4}..."
echo "  Marketplace Secret: ${LICENSE_MARKETPLACE_SECRET:0:8}..."

# Kill existing license_server processes
pkill -f 'license_server' 2>/dev/null || true
sleep 1

# Also kill by ports in case process name doesn't match
fuser -k 6699/tcp 2>/dev/null || true
fuser -k 8899/tcp 2>/dev/null || true
sleep 1

# Start server (it runs both auth:6699 and manage:8899 internally)
nohup ./license_server > server.log 2>&1 &

sleep 2

# Verify both ports are listening
AUTH_OK=false
MANAGE_OK=false

if curl -s -o /dev/null -w '%{http_code}' http://localhost:6699/health | grep -q "200"; then
    AUTH_OK=true
    echo "  Auth server (6699): OK"
else
    echo "  Auth server (6699): FAILED - check server.log"
fi

if curl -s -o /dev/null -w '%{http_code}' http://localhost:8899/ | grep -qE "200|302"; then
    MANAGE_OK=true
    echo "  Management server (8899): OK"
else
    echo "  Management server (8899): FAILED - check server.log"
fi

if $AUTH_OK && $MANAGE_OK; then
    echo "License Server started successfully."
else
    echo "Warning: Some services may not have started correctly."
    tail -20 server.log
fi
