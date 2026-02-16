#!/bin/bash

echo "=== License Server Deploy ==="
echo "Target dir: /root/license_server"
echo "Current date: $(date)"

cd /root
rm -rf license_server_new
mkdir -p license_server_new
cd license_server_new
tar -xzf /tmp/license_server.tar.gz
echo "  Extracted files:"
ls -la

echo "  Building..."
go mod tidy 2>/dev/null || true
CGO_ENABLED=0 go build -o license_server . 2>&1
if [ $? -ne 0 ]; then
    echo "  [ERROR] Failed to build license_server"
    exit 1
fi
CGO_ENABLED=0 go build -o reset_password ./cmd/reset_password/ 2>&1
if [ $? -ne 0 ]; then
    echo "  [ERROR] Failed to build reset_password"
    exit 1
fi

echo "  Build complete. Binary info:"
ls -la license_server reset_password

echo "  Stopping old server..."
pkill -f '/root/license_server/license_server' 2>/dev/null || true
sleep 2

echo "  Swapping directories..."
cd /root
rm -rf license_server_old
if [ -d license_server ]; then
    echo "  Copying DB files from old dir..."
    cp license_server/license_server.db license_server_new/ 2>/dev/null || true
    cp license_server/license_server.db-shm license_server_new/ 2>/dev/null || true
    cp license_server/license_server.db-wal license_server_new/ 2>/dev/null || true
    mv license_server license_server_old
    echo "  Old dir moved to license_server_old"
fi
mv license_server_new license_server
echo "  New dir is now /root/license_server"

echo "  Starting server..."
cd /root/license_server
sed -i 's/\r$//' start.sh
chmod +x start.sh
./start.sh

echo ""
echo "  Verifying install dir:"
ls -la /root/license_server/license_server /root/license_server/reset_password /root/license_server/start.sh
echo "=== License Server Deploy Done ==="
