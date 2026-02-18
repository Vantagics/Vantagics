#!/bin/bash

# VantageData Marketplace Server - Build & Deploy
# Target: market.vantagics.com:8088

set -e

BUILD_DIR="./build"
mkdir -p "$BUILD_DIR"

# Remote server config
SERVER="market.vantagics.com"
USER="root"
PASS="sunion123"
REMOTE_DIR="/root/marketplace_server"
PORT=8088

# SSH options
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

echo "=========================================="
echo "VantageData Marketplace Server Deploy"
echo "Target: $SERVER:$PORT"
echo "=========================================="

# Step 1: Build Windows version locally
echo ""
echo "[1/4] Building for Windows..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "$BUILD_DIR/marketplace_server.exe" .
echo "      Done: $BUILD_DIR/marketplace_server.exe"

# Step 2: Create remote directory and upload source
echo ""
echo "[2/4] Uploading source to $SERVER..."

sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "mkdir -p $REMOTE_DIR/templates" 2>/dev/null || \
    ssh $SSH_OPTS "$USER@$SERVER" "mkdir -p $REMOTE_DIR/templates"

echo "      Uploading Go source files..."
sshpass -p "$PASS" scp $SSH_OPTS main.go go.mod go.sum "$USER@$SERVER:$REMOTE_DIR/" 2>/dev/null || \
    scp $SSH_OPTS main.go go.mod go.sum "$USER@$SERVER:$REMOTE_DIR/"

echo "      Uploading templates..."
sshpass -p "$PASS" scp $SSH_OPTS templates/*.go "$USER@$SERVER:$REMOTE_DIR/templates/" 2>/dev/null || \
    scp $SSH_OPTS templates/*.go "$USER@$SERVER:$REMOTE_DIR/templates/"

# Step 3: Build on remote server
echo ""
echo "[3/4] Compiling on $SERVER..."
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "cd $REMOTE_DIR && go mod tidy && CGO_ENABLED=0 go build -o marketplace_server ." 2>/dev/null || \
    ssh $SSH_OPTS "$USER@$SERVER" "cd $REMOTE_DIR && go mod tidy && CGO_ENABLED=0 go build -o marketplace_server ."
echo "      Done: $REMOTE_DIR/marketplace_server"

# Step 4: Restart service
echo ""
echo "[4/4] Deploying and restarting service..."

echo "      Stopping existing server..."
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "pkill -f 'marketplace_server' || true" 2>/dev/null || \
    ssh $SSH_OPTS "$USER@$SERVER" "pkill -f 'marketplace_server' || true"

echo "      Uploading start script..."
sshpass -p "$PASS" scp $SSH_OPTS start.sh "$USER@$SERVER:$REMOTE_DIR/" 2>/dev/null || \
    scp $SSH_OPTS start.sh "$USER@$SERVER:$REMOTE_DIR/"

echo "      Starting new server..."
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "cd $REMOTE_DIR && chmod +x start.sh && ./start.sh" 2>/dev/null || \
    ssh $SSH_OPTS "$USER@$SERVER" "cd $REMOTE_DIR && chmod +x start.sh && ./start.sh"

echo "      Checking server status..."
sleep 3
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "curl -s http://localhost:$PORT/ | head -5 && echo 'Server started successfully' || (echo 'ERROR: Server failed to start' && tail -20 $REMOTE_DIR/server.log)" 2>/dev/null || \
    ssh $SSH_OPTS "$USER@$SERVER" "curl -s http://localhost:$PORT/ | head -5 && echo 'Server started successfully' || (echo 'ERROR: Server failed to start' && tail -20 $REMOTE_DIR/server.log)"

echo ""
echo "=========================================="
echo "Deploy Complete"
echo "=========================================="
echo ""
echo "Windows: $BUILD_DIR/marketplace_server.exe"
echo "Linux:   $USER@$SERVER:$REMOTE_DIR/marketplace_server"
echo ""
echo "Service: http://$SERVER:$PORT"
echo "Admin:   http://$SERVER:$PORT/admin/"
echo "API:     http://$SERVER:$PORT/api/packs"
echo ""
