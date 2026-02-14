#!/bin/bash

# ==========================================
# VantageData - 一键部署所有服务
# License Server + Marketplace Server
# 优化版：整个目录打包上传，减少SSH连接次数
# ==========================================

SERVER_IP="107.172.86.131"
USER="root"
SSH_OPTS="-o StrictHostKeyChecking=no"

echo "=========================================="
echo "VantageData 一键部署（优化版 - SSH密钥认证）"
echo "=========================================="
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# ==========================================
# 1. 打包 License Server 完整目录
# ==========================================
echo "[1/5] Packaging License Server..."
cd "$SCRIPT_DIR"
tar -czf /tmp/license_server.tar.gz -C license_server . 2>/dev/null
if [ $? -ne 0 ]; then
    echo "     [ERROR] Failed to create package"
    exit 1
fi
ls -lh /tmp/license_server.tar.gz | awk '{print "     Package: "$5}'
echo ""

# ==========================================
# 2. 打包 Marketplace Server 完整目录
# ==========================================
echo "[2/5] Packaging Marketplace Server..."
cd "$SCRIPT_DIR"
tar -czf /tmp/marketplace_server.tar.gz -C marketplace_server . 2>/dev/null
if [ $? -ne 0 ]; then
    echo "     [ERROR] Failed to create package"
    exit 1
fi
ls -lh /tmp/marketplace_server.tar.gz | awk '{print "     Package: "$5}'
echo ""

# ==========================================
# 3. 一次性上传两个服务包
# ==========================================
echo "[3/5] Uploading packages to server..."
scp $SSH_OPTS /tmp/license_server.tar.gz /tmp/marketplace_server.tar.gz $USER@$SERVER_IP:/tmp/ 2>/dev/null
if [ $? -ne 0 ]; then
    echo "     [ERROR] Upload failed"
    exit 1
fi
echo "     Both packages uploaded successfully"
echo ""

# ==========================================
# 4. 在服务器上一次性完成所有部署操作
# ==========================================
echo "[4/5] Building and deploying on remote server..."
ssh $SSH_OPTS $USER@$SERVER_IP 'bash -s' << 'ENDSSH'
set -e

echo "  → Deploying License Server..."

# 解压并构建 License Server
cd /root
rm -rf license_server_new
mkdir -p license_server_new
cd license_server_new
tar -xzf /tmp/license_server.tar.gz

# 编译
go mod tidy 2>/dev/null || true
CGO_ENABLED=1 go build -o license_server .

# 停止老进程并替换
cd /root
pkill -f 'license_server' || true
sleep 2
rm -rf license_server_old
mv license_server license_server_old 2>/dev/null || true
mv license_server_new license_server

# 启动服务
cd license_server
chmod +x start.sh
./start.sh
sleep 3

# 健康检查
if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
    echo "  ✓ License Server started successfully"
else
    echo "  ✗ License Server health check failed"
fi

echo ""
echo "  → Deploying Marketplace Server..."

# 解压并构建 Marketplace Server
cd /root
rm -rf marketplace_server_new
mkdir -p marketplace_server_new
cd marketplace_server_new
tar -xzf /tmp/marketplace_server.tar.gz

# 编译
go mod tidy 2>/dev/null || true
CGO_ENABLED=0 go build -o marketplace_server .

# 停止老进程并替换
cd /root
pkill -f 'marketplace_server' || true
sleep 2
rm -rf marketplace_server_old
mv marketplace_server marketplace_server_old 2>/dev/null || true
mv marketplace_server_new marketplace_server

# 启动服务
cd marketplace_server
chmod +x start.sh
./start.sh
sleep 3

# 健康检查
if curl -s http://localhost:8088/ > /dev/null 2>&1; then
    echo "  ✓ Marketplace Server started successfully"
else
    echo "  ✗ Marketplace Server health check failed"
fi

echo ""
echo "  → Verifying services..."
pgrep -f 'license_server' > /dev/null && echo "  ✓ License Server running" || echo "  ✗ License Server NOT running"
pgrep -f 'marketplace_server' > /dev/null && echo "  ✓ Marketplace Server running" || echo "  ✗ Marketplace Server NOT running"

ENDSSH

if [ $? -ne 0 ]; then
    echo "     [ERROR] Deployment failed, check server logs"
    exit 1
fi
echo ""

# ==========================================
# 5. 清理临时文件
# ==========================================
echo "[5/5] Cleanup..."
rm -f /tmp/license_server.tar.gz
rm -f /tmp/marketplace_server.tar.gz
echo "     Temporary files removed"
echo ""

# ==========================================
echo "=========================================="
echo "部署完成！"
echo "=========================================="
echo ""
echo "License Server:     http://license.vantagedata.chat:8080"
echo "Marketplace Server: http://market.vantagedata.chat:8088"
echo ""
echo "管理面板:"
echo "  License:     http://license.vantagedata.chat:8080/admin/"
echo "  Marketplace: http://market.vantagedata.chat:8088/admin/"
echo ""
echo "查看日志:"
echo "  ssh root@license.vantagedata.chat \"tail -f /root/license_server/server.log\""
echo "  ssh root@market.vantagedata.chat \"tail -f /root/marketplace_server/server.log\""
echo ""
