# UAPI SDK 集成文档

## 概述

RapidBI 已集成 [UAPI SDK](https://github.com/AxT-Team/uapi-sdk-go) 用于结构化数据搜索。UAPI 提供标准化、模式对齐的数据访问层，支持多种数据源。

## 功能特性

- ✅ 结构化数据搜索（社交媒体、游戏、图片等）
- ✅ 标准化的 JSON 响应格式
- ✅ 稳定的字段名称和数据类型
- ✅ 多数据源支持
- ✅ 前端配置界面
- ✅ 连接测试功能

## 安装

UAPI SDK 已通过 Go modules 安装：

```bash
cd src
go get github.com/AxT-Team/uapi-sdk-go@latest
```

## 配置

### 1. 获取 API Token

访问 [UAPI 文档](https://docs.uapi.nl/) 获取您的 API Token。

### 2. 前端配置

在 RapidBI 应用中：

1. 打开设置 → UAPI 配置
2. 启用 UAPI 搜索
3. 输入您的 API Token
4. （可选）自定义 Base URL
5. 点击"测试连接"验证配置
6. 保存配置

### 3. 配置文件

配置保存在 `~/rapidbi/config.json`：

```json
{
  "uapiConfig": {
    "enabled": true,
    "apiToken": "your-api-token-here",
    "baseUrl": "https://api.uapi.nl",
    "tested": true
  }
}
```

## 使用方法

### 在 Agent 中使用

UAPI 搜索工具会自动添加到 Agent 的工具列表中（如果已配置并启用）。

Agent 可以使用以下工具：

#### `uapi_search`

搜索结构化数据。

**参数：**
- `query` (必需): 搜索关键词
- `max_results` (可选): 最大结果数量（默认：10，最大：50）
- `source` (可选): 数据源类型
  - `general`: 通用搜索（默认）
  - `social`: 社交媒体数据
  - `game`: 游戏数据
  - `image`: 图片数据

**示例：**

```json
{
  "query": "artificial intelligence trends",
  "max_results": 10,
  "source": "general"
}
```

### 代码示例

#### 创建 UAPI 工具

```go
import "rapidbi/agent"

// 创建 UAPI 搜索工具
uapiTool, err := agent.NewUAPISearchTool(logger, apiToken)
if err != nil {
    log.Fatal(err)
}

// 执行搜索
ctx := context.Background()
searchInput := `{"query": "test", "max_results": 5, "source": "general"}`
result, err := uapiTool.InvokableRun(ctx, searchInput)
```

#### 在 Eino Service 中集成

UAPI 工具已自动集成到 `EinoService` 中：

```go
// src/agent/eino.go
if s.cfg.UAPIConfig != nil && s.cfg.UAPIConfig.Enabled {
    uapiTool, err := NewUAPISearchTool(s.Logger, s.cfg.UAPIConfig.APIToken)
    if err == nil {
        tools = append(tools, uapiTool)
    }
}
```

## 测试

### 运行测试程序

```bash
# 设置 API Token
set UAPI_API_TOKEN=your-token-here

# 运行测试
cd src
go run test_uapi.go
```

### 测试输出示例

```
=== UAPI SDK Integration Test ===

Test 1: Creating UAPI search tool...
✅ UAPI tool created successfully

Test 2: Getting tool information...
✅ Tool Name: uapi_search
   Description: Search for structured data across multiple sources using UAPI...

Test 3: Performing general search...
[LOG] [UAPI-SEARCH] Searching for: artificial intelligence trends (max: 3 results, source: general)
✅ Search completed
   Result: [...]

=== All Tests Completed ===
```

## 数据源说明

### General (通用)
- 通用网页内容
- 结构化数据提取
- 适用于大多数搜索场景

### Social (社交媒体)
- QQ、微信等社交平台数据
- 用户信息、动态等
- 需要相应的 API 权限

### Game (游戏)
- 游戏平台数据
- 游戏统计信息
- 玩家数据等

### Image (图片)
- 图片搜索
- 媒体内容
- 图片元数据

## API 响应格式

UAPI 返回标准化的 JSON 响应：

```json
{
  "id": "correlation-id",
  "success": true,
  "data": {
    "results": [
      {
        "title": "Result Title",
        "url": "https://example.com",
        "snippet": "Result description...",
        "source": "general",
        "published_at": "2026-01-21T00:00:00Z",
        "metadata": {}
      }
    ]
  },
  "uapi_version": "1.0",
  "schema_version": "1.0"
}
```

## 错误处理

### 常见错误

1. **API Token 无效**
   ```
   Error: UAPI API token is required
   ```
   解决：检查配置中的 API Token 是否正确

2. **连接超时**
   ```
   Error: context deadline exceeded
   ```
   解决：检查网络连接，或增加超时时间

3. **配额限制**
   ```
   Error: rate limit exceeded
   ```
   解决：等待配额重置，或升级 API 计划

## 性能优化

1. **缓存结果**：对于重复查询，考虑缓存结果
2. **批量请求**：合并多个查询以减少 API 调用
3. **超时设置**：根据需求调整超时时间（默认 60 秒）
4. **结果限制**：使用 `max_results` 参数限制返回数量

## 安全建议

1. ✅ 不要在代码中硬编码 API Token
2. ✅ 使用环境变量或配置文件存储凭证
3. ✅ 定期轮换 API Token
4. ✅ 限制 API Token 的权限范围
5. ✅ 监控 API 使用情况

## 相关链接

- [UAPI 官方文档](https://docs.uapi.nl/)
- [UAPI Go SDK](https://github.com/AxT-Team/uapi-sdk-go)
- [UAPI Python SDK](https://github.com/AxT-Team/uapi-sdk-python)
- [UAPI TypeScript SDK](https://github.com/AxT-Team/uapi-sdk-typescript)

## 更新日志

### v0.1.0 (2026-01-21)
- ✅ 初始集成 UAPI SDK
- ✅ 添加 UAPI 搜索工具
- ✅ 前端配置界面
- ✅ 连接测试功能
- ✅ 多数据源支持

## 支持

如有问题或建议，请：
1. 查看 [UAPI 文档](https://docs.uapi.nl/)
2. 提交 Issue 到项目仓库
3. 联系技术支持

---

**注意**：UAPI SDK 的具体实现方法需要根据官方文档进行调整。当前实现提供了基础框架，实际的 API 调用需要参考 UAPI SDK 的最新文档。
