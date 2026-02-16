#!/bin/bash

# ==========================================
# VantageData - 一键部署所有服务
# License Server + Marketplace Server
#
# 用法:
#   ./deploy_all.sh              部署全部
#   ./deploy_all.sh license      仅部署 License Server
#   ./deploy_all.sh market       仅部署 Marketplace Server
#   ./deploy_all.sh nginx        仅部署 Nginx 配置
# ==========================================

set -e

SERVER_IP="107.172.86.131"
USER="root"
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

TARGET="${1:-all}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=========================================="
echo "VantageData 一键部署"
echo "Target: $TARGET"
echo "=========================================="
echo ""

# ==========================================
# License Server 部署
# ==========================================
deploy_license() {
    echo "[License] Packaging source files..."
    cd "$SCRIPT_DIR/license_server"
    tar -czf /tmp/license_server.tar.gz main.go go.mod go.sum templates cmd start.sh
    ls -lh /tmp/license_server.tar.gz | awk '{print "         Package: "$5}'

    echo "[License] Uploading to server..."
    scp $SSH_OPTS /tmp/license_server.tar.gz $USER@$SERVER_IP:/tmp/

    echo "[License] Building and deploying on remote server..."
    ssh $SSH_OPTS $USER@$SERVER_IP 'bash -s' << 'ENDSSH'
set -e
cd /root
rm -rf license_server_new
mkdir -p license_server_new
cd license_server_new
tar -xzf /tmp/license_server.tar.gz

echo "  Building..."
go mod tidy 2>/dev/null || true
CGO_ENABLED=0 go build -o license_server .
CGO_ENABLED=0 go build -o reset_password ./cmd/reset_password/

echo "  Stopping old server..."
pkill -f '/root/license_server/license_server' 2>/dev/null || true
sleep 2

echo "  Swapping directories..."
cd /root
rm -rf license_server_old
if [ -d license_server ]; then
    cp license_server/license_server.db license_server_new/ 2>/dev/null || true
    cp license_server/license_server.db-shm license_server_new/ 2>/dev/null || true
    cp license_server/license_server.db-wal license_server_new/ 2>/dev/null || true
    mv license_server license_server_old
fi
mv license_server_new license_server

echo "  Starting server..."
cd license_server
sed -i 's/\r$//' start.sh
chmod +x start.sh
./start.sh
ENDSSH

    echo "[License] Health check..."
    ssh $SSH_OPTS $USER@$SERVER_IP "sleep 2 && curl -sf http://localhost:6699/health > /dev/null && echo '  Auth (6699): OK' || echo '  Auth (6699): FAILED'"
    ssh $SSH_OPTS $USER@$SERVER_IP "curl -sf -o /dev/null http://localhost:8899/ && echo '  Admin (8899): OK' || echo '  Admin (8899): FAILED'"
    echo ""

    rm -f /tmp/license_server.tar.gz
}

# ==========================================
# Marketplace Server 部署
# ==========================================
deploy_market() {
    echo "[Market] Packaging source files..."
    cd "$SCRIPT_DIR/marketplace_server"
    tar -czf /tmp/marketplace_server.tar.gz main.go go.mod go.sum templates start.sh
    ls -lh /tmp/marketplace_server.tar.gz | awk '{print "         Package: "$5}'

    echo "[Market] Uploading to server..."
    scp $SSH_OPTS /tmp/marketplace_server.tar.gz $USER@$SERVER_IP:/tmp/

    echo "[Market] Building and deploying on remote server..."
    ssh $SSH_OPTS $USER@$SERVER_IP 'bash -s' << 'ENDSSH'
set -e
cd /root
rm -rf marketplace_server_new
mkdir -p marketplace_server_new
cd marketplace_server_new
tar -xzf /tmp/marketplace_server.tar.gz

echo "  Building..."
go mod tidy 2>/dev/null || true
CGO_ENABLED=0 go build -o marketplace_server .

echo "  Stopping old server..."
pkill -f '/root/marketplace_server/marketplace_server' 2>/dev/null || true
sleep 2

echo "  Swapping directories..."
cd /root
rm -rf marketplace_server_old
if [ -d marketplace_server ]; then
    cp marketplace_server/marketplace.db marketplace_server_new/ 2>/dev/null || true
    cp marketplace_server/marketplace.db-shm marketplace_server_new/ 2>/dev/null || true
    cp marketplace_server/marketplace.db-wal marketplace_server_new/ 2>/dev/null || true
    mv marketplace_server marketplace_server_old
fi
mv marketplace_server_new marketplace_server

echo "  Starting server..."
cd marketplace_server
sed -i 's/\r$//' start.sh
chmod +x start.sh
./start.sh
ENDSSH

    echo "[Market] Health check..."
    ssh $SSH_OPTS $USER@$SERVER_IP "sleep 2 && curl -sf http://localhost:8088/ > /dev/null && echo '  Marketplace (8088): OK' || echo '  Marketplace (8088): FAILED'"
    echo ""

    rm -f /tmp/marketplace_server.tar.gz
}

# ==========================================
# Nginx 配置部署
# ==========================================
deploy_nginx() {
    echo "[Nginx] Uploading configuration..."
    scp $SSH_OPTS "$SCRIPT_DIR/../deploy/nginx/vantagedata.chat.conf" $USER@$SERVER_IP:/etc/nginx/conf.d/

    echo "[Nginx] Testing and reloading..."
    ssh $SSH_OPTS $USER@$SERVER_IP "nginx -t && nginx -s reload && echo '  Nginx: OK' || echo '  Nginx: FAILED'"
    echo ""
}

# ==========================================
# 验证所有服务状态
# ==========================================
verify() {
    echo "=========================================="
    echo "服务状态验证"
    echo "=========================================="
    ssh $SSH_OPTS $USER@$SERVER_IP 'bash -s' << 'ENDSSH'
echo "进程检查:"
pgrep -f '/root/license_server/license_server' > /dev/null && echo "  [OK] License Server running" || echo "  [X] License Server NOT running"
pgrep -f '/root/marketplace_server/marketplace_server' > /dev/null && echo "  [OK] Marketplace Server running" || echo "  [X] Marketplace Server NOT running"
echo ""
echo "端口检查:"
ss -tlnp | grep -E ':(6699|8899|8088) ' || echo "  No matching ports found"
ENDSSH
    echo ""
    echo "=========================================="
    echo "部署完成！"
    echo "=========================================="
    echo ""
    echo "License Server:"
    echo "  Auth API:  https://license.vantagedata.chat/  (port 6699)"
    echo "  Admin:     https://license.vantagedata.chat/admin/  (port 8899)"
    echo ""
    echo "Marketplace Server:"
    echo "  Service:   https://market.vantagedata.chat/  (port 8088)"
    echo "  Admin:     https://market.vantagedata.chat/admin/"
    echo ""
    echo "查看日志:"
    echo "  ssh root@$SERVER_IP \"tail -f /root/license_server/server.log\""
    echo "  ssh root@$SERVER_IP \"tail -f /root/marketplace_server/server.log\""
}

# ==========================================
# 执行
# ==========================================
case "$TARGET" in
    all)
        deploy_license
        deploy_market
        deploy_nginx
        verify
        ;;
    license)
        deploy_license
        verify
        ;;
    market)
        deploy_market
        verify
        ;;
    nginx)
        deploy_nginx
        verify
        ;;
    *)
        echo "[ERROR] Unknown target: $TARGET"
        echo "Usage: $0 [all|license|market|nginx]"
        exit 1
        ;;
esac
