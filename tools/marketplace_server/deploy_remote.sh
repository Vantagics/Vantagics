#!/bin/bash
set -e

echo "=== Marketplace Server Deploy ==="
echo "Target dir: /root/marketplace_server"

cd /root
rm -rf marketplace_server_new
mkdir -p marketplace_server_new
cd marketplace_server_new
tar -xzf /tmp/marketplace_server.tar.gz

echo "  Building..."
go mod tidy 2>/dev/null || true
CGO_ENABLED=0 go build -o marketplace_server .

echo "  Build complete. Binary info:"
ls -la marketplace_server
file marketplace_server

echo "  Stopping old server..."
pkill -f '/root/marketplace_server/marketplace_server' 2>/dev/null || true
sleep 2

echo "  Swapping directories..."
cd /root
rm -rf marketplace_server_old
if [ -d marketplace_server ]; then
    echo "  Copying DB files from old dir..."
    cp marketplace_server/marketplace.db marketplace_server_new/ 2>/dev/null || true
    cp marketplace_server/marketplace.db-shm marketplace_server_new/ 2>/dev/null || true
    cp marketplace_server/marketplace.db-wal marketplace_server_new/ 2>/dev/null || true
    mv marketplace_server marketplace_server_old
    echo "  Old dir moved to marketplace_server_old"
fi
mv marketplace_server_new marketplace_server

echo "  Starting server..."
cd marketplace_server
sed -i 's/\r$//' start.sh
chmod +x start.sh
./start.sh

echo "  Verifying install dir:"
ls -la /root/marketplace_server/marketplace_server /root/marketplace_server/start.sh
echo "=== Marketplace Server Deploy Done ==="
