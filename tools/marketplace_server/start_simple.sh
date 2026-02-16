#!/bin/bash
export MARKETPLACE_JWT_SECRET=marketplace-server-jwt-secret-key-2024
fuser -k 8088/tcp 2>/dev/null || true
sleep 1
nohup /root/marketplace_server/marketplace_server -port 8088 -db /root/marketplace_server/marketplace.db > /root/marketplace_server/server.log 2>&1 &
sleep 2
curl -s -o /dev/null -w "%{http_code}" http://localhost:8088/
