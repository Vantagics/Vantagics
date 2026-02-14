#!/bin/bash

# License Server Start Script
# Usage: ./start.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Marketplace JWT secret - must match marketplace_server's MARKETPLACE_JWT_SECRET
export LICENSE_MARKETPLACE_SECRET="${LICENSE_MARKETPLACE_SECRET:-marketplace-server-jwt-secret-key-2024}"

# Database password (optional, defaults to built-in value)
export LICENSE_DB_PASSWORD="${LICENSE_DB_PASSWORD:-vantagedata2024}"

# Admin password (optional, defaults to built-in value)
export LICENSE_ADMIN_PASSWORD="${LICENSE_ADMIN_PASSWORD:-admin123}"

echo "Starting License Server..."
echo "  Marketplace Secret: ${LICENSE_MARKETPLACE_SECRET:0:8}..."

# Kill existing process on port 8080
fuser -k 8080/tcp 2>/dev/null || true
sleep 1

# Start server
nohup ./license_server -port 8080 -db ./license.db -templates ./templates > server.log 2>&1 &

sleep 2

# Verify
if curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/api/health | grep -q "200"; then
    echo "License Server started successfully on port 8080"
else
    echo "Warning: Server may not have started correctly. Check server.log"
fi
