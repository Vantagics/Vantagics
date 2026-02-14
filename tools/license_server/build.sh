#!/bin/bash

# VantageData License Server Build Script
# Builds for macOS locally and Linux on remote server

set -e

BUILD_DIR="./build"
mkdir -p "$BUILD_DIR"

# Remote server config
SERVER="license.vantagedata.chat"
USER="root"
PASS="sunion123"
REMOTE_DIR="/root/license_server"

# SSH options to handle host key changes
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

echo "=========================================="
echo "VantageData License Server Build"
echo "=========================================="

# Remove old host key if exists (in case server was reinstalled)
ssh-keygen -R "$SERVER" 2>/dev/null || true

# Build Windows version locally (skip macOS on Windows)
echo ""
echo "[1/3] Building for Windows..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "$BUILD_DIR/license_server.exe" . || echo "      [WARN] Windows build skipped"

# Build Linux version on remote server
echo ""
echo "[2/3] Building for Linux on $SERVER..."

echo "      Creating remote directory..."
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "mkdir -p $REMOTE_DIR"

echo "      Uploading source files..."
sshpass -p "$PASS" scp $SSH_OPTS main.go go.mod go.sum "$USER@$SERVER:$REMOTE_DIR/"
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "mkdir -p $REMOTE_DIR/templates"
sshpass -p "$PASS" scp $SSH_OPTS templates/*.go "$USER@$SERVER:$REMOTE_DIR/templates/"

echo "      Compiling on server..."
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "cd $REMOTE_DIR && go mod tidy && CGO_ENABLED=1 go build -o license_server ."

echo "      Done: $REMOTE_DIR/license_server"

echo ""
echo "[3/3] Deploying start script and restarting service..."

echo "      Uploading start script..."
sshpass -p "$PASS" scp $SSH_OPTS start.sh "$USER@$SERVER:$REMOTE_DIR/"
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "chmod +x $REMOTE_DIR/start.sh"

echo "      Stopping existing server..."
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "pkill -f 'license_server' || true"
sleep 2

echo "      Starting server..."
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "$REMOTE_DIR/start.sh"

echo "      Checking server status..."
sleep 3
sshpass -p "$PASS" ssh $SSH_OPTS "$USER@$SERVER" "curl -s http://localhost:8080/api/health && echo 'Server started successfully' || (echo 'ERROR: Server failed to start' && tail -20 $REMOTE_DIR/server.log)"

echo ""
echo "=========================================="
echo "Build & Deploy Complete"
echo "=========================================="
echo ""
echo "Windows: $BUILD_DIR/license_server.exe"
echo "Linux:   $USER@$SERVER:$REMOTE_DIR/license_server"
echo ""
echo "Service: http://$SERVER:8080"
echo "Admin:   http://$SERVER:8080/admin/"
echo "API:     http://$SERVER:8080/api/health"
