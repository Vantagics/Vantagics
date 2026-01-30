# AppData Manager

命令行工具，用于管理加密存储的商店开发者凭证。

## 工作流程

1. 开发者使用此工具配置商店凭证
2. 凭证加密存储在当前目录的 `appdata.dat`
3. 编译时，`build.sh` 自动将 `appdata.dat` 嵌入到应用程序中
4. 应用程序运行时从嵌入的数据中读取凭证（只读）

## 功能

- 使用 AES-256-GCM 加密存储敏感凭证
- 支持多个电商平台（Shopify、WooCommerce、Magento 等）
- 凭证以 Base64 编码存储
- 加密密钥：`vantagedata`

## 编译

```bash
cd tools/appdata_manager
go build -o appdata_manager .
```

## 使用方法

```bash
# 列出所有凭证
./appdata_manager list

# 添加新凭证
./appdata_manager add shopify

# 编辑现有凭证
./appdata_manager edit shopify

# 删除凭证
./appdata_manager delete shopify

# 导出凭证（未加密 JSON，用于备份）
./appdata_manager export backup.json

# 导入凭证
./appdata_manager import backup.json
```

## 支持的平台

- shopify
- woocommerce
- magento
- bigcommerce
- squarespace
- wix

## 数据存储位置

凭证存储在当前目录：`./appdata.dat`

编译时会自动复制到 `src/agent/appdata.dat` 并嵌入程序。

## 安全说明

- 凭证使用 AES-256-GCM 加密
- 密钥通过 SHA-256 从 "vantagedata" 派生
- 导出的 JSON 文件是未加密的，请妥善保管
- 嵌入程序后，凭证无法在运行时修改
