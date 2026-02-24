# Snowflake 和 BigQuery 数据源集成指南

## 概述

Vantagics 现已支持从 Snowflake 和 BigQuery 导入数据，为欧美市场用户提供企业级数据仓库集成能力。

## Snowflake 集成

### 功能特性

- ✅ 支持 Snowflake 账户连接
- ✅ 自动导入数据库表到本地 SQLite
- ✅ 支持指定 Warehouse、Database、Schema 和 Role
- ✅ 完整的表结构和数据同步

### 配置要求

1. **账户标识符（Account Identifier）**
   - 格式：`account_name.region`
   - 示例：`xy12345.us-east-1`
   - 获取方式：从 Snowflake 控制台 URL 中提取

2. **用户名和密码**
   - 使用您的 Snowflake 登录凭证

3. **可选配置**
   - **Warehouse**: 计算资源仓库名称（如 `COMPUTE_WH`）
   - **Database**: 要导入的数据库名称
   - **Schema**: 要导入的模式名称（如 `PUBLIC`）
   - **Role**: 使用的角色（如 `ACCOUNTADMIN`）

### 使用步骤

1. 在"添加数据源"对话框中选择 **Snowflake**
2. 填写必需的连接信息：
   - 账户标识符
   - 用户名
   - 密码
3. （可选）填写 Warehouse、Database、Schema、Role
4. 点击"导入"按钮
5. 系统将自动：
   - 连接到 Snowflake
   - 列出所有可访问的表
   - 将数据导入到本地 SQLite 数据库
   - 创建数据源记录

### 技术实现

- 使用 `github.com/snowflakedb/gosnowflake` 驱动
- DSN 格式：`user:password@account/database/schema?warehouse=wh&role=role`
- 数据通过 `copyTable` 方法同步到本地 SQLite

### 注意事项

- 首次导入可能需要较长时间，取决于数据量
- 建议指定具体的 Database 和 Schema 以减少导入时间
- 导入后的数据存储在本地，可离线分析

## BigQuery 集成

### 功能特性

- ✅ 支持 Google Cloud BigQuery 连接
- ✅ 使用服务账户 JSON 密钥认证
- ✅ 支持指定项目和数据集
- ⚠️ 需要额外的 Go 依赖包

### 配置要求

1. **项目 ID（Project ID）**
   - 您的 Google Cloud 项目 ID
   - 示例：`my-gcp-project`

2. **服务账户凭证（Service Account JSON）**
   - 完整的 JSON 密钥文件内容
   - 需要包含 BigQuery 读取权限

3. **可选配置**
   - **Dataset ID**: 特定数据集 ID（留空导入所有数据集）

### 获取服务账户密钥

1. 进入 [Google Cloud Console](https://console.cloud.google.com)
2. 导航到 **IAM & Admin** → **Service Accounts**
3. 创建新的服务账户或选择现有账户
4. 授予 **BigQuery Data Viewer** 和 **BigQuery Job User** 角色
5. 创建并下载 JSON 密钥文件
6. 将 JSON 文件内容复制到配置界面

### 使用步骤

1. 在"添加数据源"对话框中选择 **BigQuery**
2. 填写项目 ID
3. （可选）填写数据集 ID
4. 粘贴服务账户 JSON 密钥内容
5. 点击"导入"按钮

### 技术实现

- 需要安装：`go get cloud.google.com/go/bigquery`
- 使用 Google Cloud BigQuery Go 客户端库
- 支持标准 SQL 查询

### 当前状态

⚠️ **开发中**: BigQuery 集成的完整实现需要额外的依赖包。当前版本会提示用户安装所需的 Go 包。

完整实现将包括：
1. 解析服务账户 JSON 凭证
2. 创建 BigQuery 客户端
3. 列出指定数据集中的表（或所有数据集）
4. 查询每个表并复制数据到本地 SQLite
5. 创建并返回数据源记录

## 前端界面

### Snowflake 配置界面

```typescript
- 账户标识符输入框（必填）
- 用户名输入框（必填）
- 密码输入框（必填）
- Warehouse 输入框（可选）
- Database 输入框（可选）
- Schema 输入框（可选）
- Role 输入框（可选）
```

### BigQuery 配置界面

```typescript
- 项目 ID 输入框（必填）
- 数据集 ID 输入框（可选）
- 服务账户 JSON 文本区域（必填，多行）
```

## 国际化支持

已添加完整的中英文翻译：

### 英文
- `snowflake_setup_guide`: "❄️ Snowflake Connection"
- `snowflake_desc`: "Connect to your Snowflake data warehouse..."
- `bigquery_setup_guide`: "📊 BigQuery Connection"
- `bigquery_step1-4`: 设置步骤说明

### 中文
- `snowflake_setup_guide`: "❄️ Snowflake 连接"
- `snowflake_desc`: "连接到您的 Snowflake 数据仓库..."
- `bigquery_setup_guide`: "📊 BigQuery 连接"
- `bigquery_step1-4`: 设置步骤说明

## 数据源类型

在 `DataSourceConfig` 中新增字段：

```go
// Snowflake configuration
SnowflakeAccount   string `json:"snowflake_account,omitempty"`
SnowflakeUser      string `json:"snowflake_user,omitempty"`
SnowflakePassword  string `json:"snowflake_password,omitempty"`
SnowflakeWarehouse string `json:"snowflake_warehouse,omitempty"`
SnowflakeDatabase  string `json:"snowflake_database,omitempty"`
SnowflakeSchema    string `json:"snowflake_schema,omitempty"`
SnowflakeRole      string `json:"snowflake_role,omitempty"`

// BigQuery configuration
BigQueryProjectID   string `json:"bigquery_project_id,omitempty"`
BigQueryDatasetID   string `json:"bigquery_dataset_id,omitempty"`
BigQueryCredentials string `json:"bigquery_credentials,omitempty"`
```

## 依赖包

### 已添加
- `github.com/snowflakedb/gosnowflake v1.7.2`

### 待添加（BigQuery）
- `cloud.google.com/go/bigquery`

## 测试建议

### Snowflake 测试
1. 准备测试账户和凭证
2. 创建测试数据库和表
3. 验证连接和数据导入
4. 测试各种配置组合（有/无 Warehouse、Database 等）

### BigQuery 测试
1. 创建 GCP 测试项目
2. 设置服务账户和权限
3. 准备测试数据集和表
4. 验证 JSON 密钥解析
5. 测试数据导入流程

## 后续优化

1. **增量同步**: 支持定期刷新数据
2. **查询下推**: 直接在 Snowflake/BigQuery 执行查询
3. **性能优化**: 并行导入多个表
4. **错误处理**: 更详细的错误信息和重试机制
5. **连接池**: 复用数据库连接
6. **BigQuery 完整实现**: 完成 BigQuery 的数据导入逻辑

## 安全建议

1. **凭证加密**: 考虑加密存储敏感凭证
2. **权限最小化**: 使用只读权限的账户
3. **连接超时**: 设置合理的超时时间
4. **审计日志**: 记录所有数据访问操作

## 支持和反馈

如遇到问题或有改进建议，请通过以下方式反馈：
- GitHub Issues
- 技术支持邮箱
- 用户社区论坛
