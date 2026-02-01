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

echo "=========================================="
echo "VantageData License Server Build"
echo "=========================================="

# Build macOS version locally
echo ""
echo "[1/2] Building for macOS..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o "$BUILD_DIR/license_server_macos" .
echo "      Done: $BUILD_DIR/license_server_macos"

# Build Linux version on remote server
echo ""
echo "[2/2] Building for Linux on $SERVER..."

echo "      Creating remote directory..."
sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no "$USER@$SERVER" "mkdir -p $REMOTE_DIR"

echo "      Uploading source files..."
sshpass -p "$PASS" scp -o StrictHostKeyChecking=no main.go templates.go go.mod go.sum "$USER@$SERVER:$REMOTE_DIR/"

echo "      Compiling on server..."
sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no "$USER@$SERVER" "cd $REMOTE_DIR && go mod tidy && CGO_ENABLED=1 go build -o license_server ."

echo "      Done: $REMOTE_DIR/license_server"

echo ""
echo "=========================================="
echo "Build Complete"
echo "=========================================="
echo ""
echo "macOS: $BUILD_DIR/license_server_macos"
echo "Linux: $USER@$SERVER:$REMOTE_DIR/license_server"
echo ""
echo "To run on server:"
echo "  ssh $USER@$SERVER '$REMOTE_DIR/license_server'"
