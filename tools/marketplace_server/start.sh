#!/bin/bash

# Marketplace Server Start Script
# Usage: ./start.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# JWT secret for marketplace - must match license_server's LICENSE_MARKETPLACE_SECRET
export MARKETPLACE_JWT_SECRET="${MARKETPLACE_JWT_SECRET:-marketplace-server-jwt-secret-key-2024}"

echo "Starting Marketplace Server..."
echo "  JWT Secret: ${MARKETPLACE_JWT_SECRET:0:8}..."

# Kill existing process on port 8088
fuser -k 8088/tcp 2>/dev/null || true
sleep 1

# Start server
nohup ./marketplace_server -port 8088 -db ./marketplace.db > server.log 2>&1 &

sleep 2

# Verify
if curl -s -o /dev/null -w '%{http_code}' http://localhost:8088/ | grep -q "200"; then
    echo "Marketplace Server started successfully on port 8088"
else
    echo "Warning: Server may not have started correctly. Check server.log"
fi
