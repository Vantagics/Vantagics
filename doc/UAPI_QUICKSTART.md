# UAPI 快速开始指南

## 简介

UAPI (Universal API) 是一个通用数据访问层，可以将任意网页转换为标准化的、模式对齐的 JSON 数据。RapidBI 已集成 UAPI SDK，让您可以轻松访问结构化数据。

## 快速开始

### 1. 获取 API Token

1. 访问 [UAPI 官网](https://docs.uapi.nl/)
2. 注册账号
3. 在控制台创建 API Token
4. 复制您的 Token

### 2. 配置 RapidBI

#### 方法一：通过界面配置

1. 打开 RapidBI 应用
2. 点击右上角设置图标
3. 选择"UAPI 配置"
4. 启用 UAPI 搜索
5. 粘贴您的 API Token
6. 点击"测试连接"
7. 测试成功后点击"保存"

#### 方法二：手动编辑配置文件

编辑 `~/rapidbi/config.json`：

```json
{
  "uapiConfig": {
    "enabled": true,
    "apiToken": "your-api-token-here",
    "tested": true
  }
}
```

### 3. 使用 UAPI 搜索

配置完成后，Agent 会自动获得 `uapi_search` 工具。您可以在对话中请求 Agent 使用 UAPI 搜索：

**示例对话：**

```
用户: 使用 UAPI 搜索最新的人工智能趋势

Agent: 我将使用 UAPI 搜索工具为您查找...
[调用 uapi_search 工具]
根据搜索结果，当前人工智能的主要趋势包括...
```

## 支持的数据源

### General (通用)
适用于大多数网页内容的结构化提取。

**使用场景：**
- 新闻文章
- 博客内容
- 产品信息
- 公司数据

**示例：**
```json
{
  "query": "latest AI news",
  "source": "general",
  "max_results": 10
}
```

### Social (社交媒体)
访问社交平台的结构化数据。

**使用场景：**
- 用户信息
- 社交动态
- 趋势话题

**示例：**
```json
{
  "query": "trending topics",
  "source": "social",
  "max_results": 5
}
```

### Game (游戏)
游戏平台和游戏数据。

**使用场景：**
- 游戏统计
- 玩家数据
- 排行榜

**示例：**
```json
{
  "query": "top games 2026",
  "source": "game",
  "max_results": 10
}
```

### Image (图片)
图片和媒体内容搜索。

**使用场景：**
- 图片搜索
- 媒体资源
- 视觉内容

**示例：**
```json
{
  "query": "technology images",
  "source": "image",
  "max_results": 20
}
```

## 常见问题

### Q: UAPI 和普通网页搜索有什么区别？

A: UAPI 提供结构化、标准化的数据，而普通搜索返回原始 HTML。UAPI 的优势：
- 稳定的字段名称
- 清晰的数据类型
- ISO 8601 时间戳
- 无需处理 HTML 解析

### Q: UAPI 搜索速度如何？

A: UAPI 通常比传统网页抓取更快，因为：
- 优化的数据提取
- 内置缓存机制
- 无需浏览器渲染

### Q: 如何处理 API 配额限制？

A: 
1. 合理使用 `max_results` 参数
2. 缓存常用查询结果
3. 升级到更高的 API 计划
4. 监控 API 使用情况

### Q: 支持哪些语言？

A: UAPI 支持多语言查询，包括：
- 英语
- 中文
- 其他主要语言

### Q: 数据更新频率如何？

A: 取决于数据源：
- 实时数据：社交媒体、新闻
- 定期更新：统计数据、报告
- 静态数据：历史信息

## 最佳实践

### 1. 精确的查询

使用具体的关键词获得更好的结果：

❌ 不好：`"data"`
✅ 好：`"2026 AI market data trends"`

### 2. 合理的结果数量

根据需求设置 `max_results`：

- 快速预览：3-5 条
- 常规分析：10-20 条
- 深度研究：30-50 条

### 3. 选择合适的数据源

根据查询内容选择最合适的 `source`：

- 新闻、文章 → `general`
- 社交趋势 → `social`
- 游戏数据 → `game`
- 图片内容 → `image`

### 4. 错误处理

始终检查返回结果：

```go
result, err := uapiTool.InvokableRun(ctx, searchInput)
if err != nil {
    // 处理错误
    log.Printf("Search failed: %v", err)
    return
}

// 解析结果
var searchResults []UAPISearchResult
json.Unmarshal([]byte(result), &searchResults)
```

### 5. 性能优化

- 使用超时控制：`context.WithTimeout()`
- 并发请求时注意速率限制
- 缓存频繁查询的结果

## 示例代码

### 基础搜索

```go
package main

import (
    "context"
    "fmt"
    "rapidbi/agent"
)

func main() {
    // 创建工具
    tool, _ := agent.NewUAPISearchTool(nil, "your-token")
    
    // 执行搜索
    ctx := context.Background()
    input := `{"query": "AI trends", "max_results": 5}`
    result, _ := tool.InvokableRun(ctx, input)
    
    fmt.Println(result)
}
```

### 多源搜索

```go
sources := []string{"general", "social", "game"}
for _, source := range sources {
    input := fmt.Sprintf(`{
        "query": "technology",
        "source": "%s",
        "max_results": 3
    }`, source)
    
    result, err := tool.InvokableRun(ctx, input)
    if err != nil {
        fmt.Printf("Error in %s: %v\n", source, err)
        continue
    }
    
    fmt.Printf("Results from %s:\n%s\n\n", source, result)
}
```

## 下一步

- 阅读 [完整集成文档](./UAPI_INTEGRATION.md)
- 查看 [UAPI 官方文档](https://docs.uapi.nl/)
- 探索 [API 参考](https://docs.uapi.nl/api-reference)

## 支持

遇到问题？

1. 检查 [常见问题](#常见问题)
2. 查看 [UAPI 文档](https://docs.uapi.nl/)
3. 提交 Issue 到项目仓库

---

**提示**：UAPI 是一个强大的工具，但请合理使用 API 配额，避免不必要的请求。
