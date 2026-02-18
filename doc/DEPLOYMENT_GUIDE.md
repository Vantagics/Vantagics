# VantageData 部署指南

## 概述

本文档描述如何部署 VantageData 的完整系统，包括桌面客户端、License 服务器和 Marketplace 服务器。

## 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│                      VantageData 客户端                          │
│                    (Windows/macOS/Linux)                        │
│                                                                 │
└────────────┬────────────────────────────────┬───────────────────┘
             │                                │
             │ 激活/认证                       │ 市场功能
             ▼                                ▼
┌─────────────────────────┐      ┌─────────────────────────┐
│                         │      │                         │
│   License Server        │◄────►│  Marketplace Server     │
│   (授权服务器)           │      │  (市场服务器)            │
│                         │      │                         │
│  - 序列号管理            │      │  - 分析包管理            │
│  - LLM 配置分发         │      │  - Credits 系统         │
│  - 搜索引擎配置          │      │  - 用户管理             │
│  - 市场认证             │      │  - 分类管理             │
│                         │      │                         │
└─────────────────────────┘      └─────────────────────────┘
```

## 服务器信息

### License Server

- **域名**：license.vantagics.com
- **IP**：107.172.86.131
- **管理端口**：8899
- **授权端口**：6699
- **部署目录**：/root/license_server

### Marketplace Server

- **域名**：market.vantagics.com
- **IP**：107.172.86.131
- **端口**：8088
- **部署目录**：/root/marketplace_server

## 环境准备

### 服务器要求

- **操作系统**：Linux（推荐 Ubuntu 20.04+）
- **内存**：至少 2GB
- **存储**：至少 10GB 可用空间
- **网络**：公网 IP 和域名

### 软件依赖

- Go 1.25.5+
- SQLite 3
- Nginx（可选，用于反向代理）

## License 服务器部署

### 1. 编译

```bash
cd tools/license_server
go build -o license_server .
```

### 2. 配置环境变量

创建 `.env` 文件：

```bash
LICENSE_MARKETPLACE_SECRET=your-secret-key-here
LICENSE_DB_PASSWORD=your-db-password
LICENSE_ADMIN_PASSWORD=your-admin-password
```

### 3. 上传到服务器

```bash
scp license_server root@license.vantagics.com:/root/license_server/
scp .env root@license.vantagics.com:/root/license_server/
```

### 4. 创建启动脚本

在服务器上创建 `/root/license_server/start.sh`：

```bash
#!/bin/bash
cd /root/license_server

# 加载环境变量
export $(cat .env | xargs)

# 停止旧进程
pkill -f license_server

# 启动新进程
nohup ./license_server > server.log 2>&1 &

echo "License Server started"
```

赋予执行权限：

```bash
chmod +x /root/license_server/start.sh
```

### 5. 启动服务

```bash
ssh root@license.vantagedata.chat "cd /root/license_server && ./start.sh"
```

### 6. 验证服务

```bash
# 检查健康状态
curl http://license.vantagics.com:6699/health

# 检查管理界面
curl http://license.vantagics.com:8899/
```

## Marketplace 服务器部署

### 1. 编译

```bash
cd tools/marketplace_server
go build -o marketplace_server .
```

### 2. 配置环境变量

创建 `.env` 文件：

```bash
MARKETPLACE_JWT_SECRET=your-secret-key-here
```

> ⚠️ **重要**：`MARKETPLACE_JWT_SECRET` 必须与 License 服务器的 `LICENSE_MARKETPLACE_SECRET` 保持一致！

### 3. 上传到服务器

```bash
scp marketplace_server root@market.vantagics.com:/root/marketplace_server/
scp .env root@market.vantagics.com:/root/marketplace_server/
```

### 4. 创建启动脚本

在服务器上创建 `/root/marketplace_server/start.sh`：

```bash
#!/bin/bash
cd /root/marketplace_server

# 加载环境变量
export $(cat .env | xargs)

# 停止旧进程
pkill -f marketplace_server

# 启动新进程
nohup ./marketplace_server > server.log 2>&1 &

echo "Marketplace Server started"
```

赋予执行权限：

```bash
chmod +x /root/marketplace_server/start.sh
```

### 5. 启动服务

```bash
ssh root@market.vantagedata.chat "cd /root/marketplace_server && ./start.sh"
```

### 6. 验证服务

```bash
# 检查服务状态
curl http://market.vantagics.com:8088/

# 检查 API
curl http://market.vantagics.com:8088/api/categories
```

## 快速部署脚本

使用统一部署脚本同时部署两个服务器：

```bash
cd tools
bash deploy_all.sh
```

该脚本会自动：
1. 编译 License 服务器和 Marketplace 服务器
2. 通过 SSH 上传到对应服务器
3. 自动重启服务
4. 验证服务状态

## Nginx 反向代理配置（可选）

### License 服务器

创建 `/etc/nginx/sites-available/license.vantagics.com`：

```nginx
server {
    listen 80;
    server_name license.vantagics.com;

    # 管理界面
    location / {
        proxy_pass http://localhost:8899;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 授权 API
    location /api/ {
        proxy_pass http://localhost:6699;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Marketplace 服务器

创建 `/etc/nginx/sites-available/market.vantagics.com`：

```nginx
server {
    listen 80;
    server_name market.vantagics.com;

    location / {
        proxy_pass http://localhost:8088;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 增加上传文件大小限制
    client_max_body_size 100M;
}
```

### 启用配置

```bash
# 创建符号链接
ln -s /etc/nginx/sites-available/license.vantagics.com /etc/nginx/sites-enabled/
ln -s /etc/nginx/sites-available/market.vantagics.com /etc/nginx/sites-enabled/

# 测试配置
nginx -t

# 重启 Nginx
systemctl restart nginx
```

## SSL/HTTPS 配置

### 使用 Let's Encrypt

```bash
# 安装 Certbot
apt-get update
apt-get install certbot python3-certbot-nginx

# 为 License 服务器申请证书
certbot --nginx -d license.vantagics.com

# 为 Marketplace 服务器申请证书
certbot --nginx -d market.vantagics.com

# 自动续期
certbot renew --dry-run
```

### 手动配置 SSL

如果使用自己的证书，修改 Nginx 配置：

```nginx
server {
    listen 443 ssl;
    server_name license.vantagics.com;

    ssl_certificate /path/to/certificate.crt;
    ssl_certificate_key /path/to/private.key;

    # ... 其他配置
}
```

## 客户端部署

### Windows

```bash
cd src
wails build -platform windows/amd64
```

生成的安装包位于 `src/build/bin/VantageData.exe`。

### macOS

```bash
cd src
wails build -platform darwin/universal
```

生成的应用位于 `src/build/bin/VantageData.app`。

### Linux

```bash
cd src
wails build -platform linux/amd64
```

生成的可执行文件位于 `src/build/bin/VantageData`。

## 监控和维护

### 查看日志

```bash
# License 服务器
ssh root@license.vantagedata.chat "tail -f /root/license_server/server.log"

# Marketplace 服务器
ssh root@market.vantagedata.chat "tail -f /root/marketplace_server/server.log"
```

### 重启服务

```bash
# License 服务器
ssh root@license.vantagedata.chat "cd /root/license_server && ./start.sh"

# Marketplace 服务器
ssh root@market.vantagedata.chat "cd /root/marketplace_server && ./start.sh"
```

### 备份数据库

```bash
# License 服务器
ssh root@license.vantagedata.chat "cp /root/license_server/license_server.db /root/backups/license_$(date +%Y%m%d).db"

# Marketplace 服务器
ssh root@market.vantagedata.chat "cp /root/marketplace_server/marketplace.db /root/backups/marketplace_$(date +%Y%m%d).db"
```

### 自动备份脚本

创建 `/root/backup.sh`：

```bash
#!/bin/bash
BACKUP_DIR="/root/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份 License 数据库
cp /root/license_server/license_server.db $BACKUP_DIR/license_$DATE.db

# 备份 Marketplace 数据库
cp /root/marketplace_server/marketplace.db $BACKUP_DIR/marketplace_$DATE.db

# 删除 30 天前的备份
find $BACKUP_DIR -name "*.db" -mtime +30 -delete

echo "Backup completed: $DATE"
```

添加到 crontab（每天凌晨 2 点执行）：

```bash
0 2 * * * /root/backup.sh >> /root/backup.log 2>&1
```

## 故障排查

### License 服务器无法访问

1. 检查服务是否运行：
   ```bash
   ps aux | grep license_server
   ```

2. 检查端口是否监听：
   ```bash
   netstat -tlnp | grep 6699
   netstat -tlnp | grep 8899
   ```

3. 检查防火墙规则：
   ```bash
   ufw status
   ufw allow 6699
   ufw allow 8899
   ```

4. 查看日志：
   ```bash
   tail -f /root/license_server/server.log
   ```

### Marketplace 服务器无法访问

1. 检查服务是否运行：
   ```bash
   ps aux | grep marketplace_server
   ```

2. 检查端口是否监听：
   ```bash
   netstat -tlnp | grep 8088
   ```

3. 检查防火墙规则：
   ```bash
   ufw allow 8088
   ```

4. 查看日志：
   ```bash
   tail -f /root/marketplace_server/server.log
   ```

### 市场认证失败

1. 检查共享密钥是否一致：
   ```bash
   # License 服务器
   ssh root@license.vantagedata.chat "grep LICENSE_MARKETPLACE_SECRET /root/license_server/.env"

   # Marketplace 服务器
   ssh root@market.vantagedata.chat "grep MARKETPLACE_JWT_SECRET /root/marketplace_server/.env"
   ```

2. 测试认证流程：
   ```bash
   # 1. 请求认证令牌
   curl -X POST http://license.vantagedata.chat:6699/api/marketplace-auth \
     -H "Content-Type: application/json" \
     -d '{"sn":"YOUR-SN","email":"user@example.com"}'

   # 2. 使用令牌登录市场
   curl -X POST http://market.vantagedata.chat:8088/api/auth/sn-login \
     -H "Content-Type: application/json" \
     -d '{"license_token":"TOKEN-FROM-STEP-1"}'
   ```

## 安全建议

1. **修改默认密码**：首次部署后立即修改管理员密码
2. **使用 HTTPS**：生产环境务必配置 SSL 证书
3. **限制网络访问**：
   - 管理端口（8899）只允许管理员 IP 访问
   - 授权端口（6699）和市场端口（8088）可对外开放
4. **定期备份**：每天自动备份数据库文件
5. **监控日志**：定期检查服务器日志，发现异常及时处理
6. **更新维护**：及时更新服务器程序，修复安全漏洞
7. **强密钥**：共享密钥使用强随机字符串（至少 32 字符）

## 性能优化

### 数据库优化

```bash
# 定期执行 VACUUM 优化数据库
sqlite3 /root/license_server/license_server.db "VACUUM;"
sqlite3 /root/marketplace_server/marketplace.db "VACUUM;"
```

### Nginx 优化

在 Nginx 配置中添加：

```nginx
# 启用 gzip 压缩
gzip on;
gzip_types text/plain text/css application/json application/javascript;

# 启用缓存
proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=my_cache:10m max_size=1g inactive=60m;
proxy_cache my_cache;
```

### 系统资源监控

```bash
# 安装监控工具
apt-get install htop iotop

# 查看系统资源
htop

# 查看磁盘 I/O
iotop
```

---

**文档版本**：1.0  
**最后更新**：2026-02-16
