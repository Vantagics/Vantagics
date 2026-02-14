#!/bin/bash

# Quick Deploy - Package and upload as tar.gz (faster than individual files)

set -e

PASS="sunion123"
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

echo "=========================================="
echo "Quick Deploy - Both Servers"
echo "=========================================="

# Package license_server
echo ""
echo "[1/4] Packaging license_server..."
cd D:/workprj/VantageData/tools/license_server
tar -czf /tmp/license_server.tar.gz main.go go.mod go.sum templates/ start.sh
echo "      Package created: $(du -h /tmp/license_server.tar.gz | cut -f1)"

# Upload and deploy license_server
echo ""
echo "[2/4] Deploying license_server..."
sshpass -p "$PASS" scp $SSH_OPTS /tmp/license_server.tar.gz root@107.172.86.131:/tmp/
sshpass -p "$PASS" ssh $SSH_OPTS root@107.172.86.131 'bash -s' << 'LICENSE_DEPLOY'
set -e
cd /root
rm -rf license_server_new
mkdir -p license_server_new
cd license_server_new
tar -xzf /tmp/license_server.tar.gz
echo "Building..."
go mod tidy
CGO_ENABLED=1 go build -o license_server .
echo "Stopping old server..."
pkill -f "license_server -port" || true
sleep 2
echo "Moving to production..."
cd /root
rm -rf license_server_old
mv license_server license_server_old 2>/dev/null || true
mv license_server_new license_server
cd license_server
chmod +x start.sh
echo "Starting server..."
./start.sh
sleep 3
curl -s http://localhost:8080/api/health && echo "✓ License Server running" || echo "✗ Failed to start"
LICENSE_DEPLOY

# Package marketplace_server
echo ""
echo "[3/4] Packaging marketplace_server..."
cd D:/workprj/VantageData/tools/marketplace_server
tar -czf /tmp/marketplace_server.tar.gz main.go go.mod go.sum templates/ start.sh
echo "      Package created: $(du -h /tmp/marketplace_server.tar.gz | cut -f1)"

# Upload and deploy marketplace_server
echo ""
echo "[4/4] Deploying marketplace_server..."
sshpass -p "$PASS" scp $SSH_OPTS /tmp/marketplace_server.tar.gz root@107.172.86.131:/tmp/
sshpass -p "$PASS" ssh $SSH_OPTS root@107.172.86.131 'bash -s' << 'MARKETPLACE_DEPLOY'
set -e
cd /root
rm -rf marketplace_server_new
mkdir -p marketplace_server_new
cd marketplace_server_new
tar -xzf /tmp/marketplace_server.tar.gz
echo "Building..."
go mod tidy
CGO_ENABLED=0 go build -o marketplace_server .
echo "Stopping old server..."
pkill -f "marketplace_server -port" || true
sleep 2
echo "Moving to production..."
cd /root
rm -rf marketplace_server_old
mv marketplace_server marketplace_server_old 2>/dev/null || true
mv marketplace_server_new marketplace_server
cd marketplace_server
chmod +x start.sh
echo "Starting server..."
./start.sh
sleep 3
curl -s http://localhost:8088/ | head -5 && echo "✓ Marketplace Server running" || echo "✗ Failed to start"
MARKETPLACE_DEPLOY

echo ""
echo "=========================================="
echo "Deployment Complete!"
echo "=========================================="
echo ""
echo "License Server:     http://license.vantagedata.chat:8080"
echo "Marketplace Server: http://market.vantagedata.chat:8088"
echo ""
