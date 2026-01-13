# 分析结果与请求绑定架构

## 概述

系统已经实现了将分析结果与用户请求绑定的完整功能。每个用户的分析请求都会被分配唯一 ID，分析结果会与该请求关联，用户点击请求后可以在仪表盘上重新查看结果。

## 架构设计

### 1. 唯一 ID 分配

每个用户消息（分析请求）在创建时自动获得唯一 ID：

```go
// src/chat_service.go
type ChatMessage struct {
    ID        string     `json:"id"`           // 唯一消息 ID
    Role      string     `json:"role"`         // "user" 或 "assistant"
    Content   string     `json:"content"`      // 用户请求文本
    Timestamp int64      `json:"timestamp"`    // 时间戳
    ChartData *ChartData `json:"chart_data,omitempty"` // 绑定的图表数据
}
```

**ID 生成：**
- 使用纳秒时间戳：`time.Now().UnixNano()`
- 全局唯一，确保不会冲突

### 2. 分析结果类型

系统支持多种分析结果类型：

```go
type ChartData struct {
    Type string `json:"type"` // "echarts" | "image" | "table" | "csv"
    Data string `json:"data"` // JSON字符串或base64/data URL
}
```

**支持的类型：**

| 类型 | 说明 | 数据格式 |
|------|------|---------|
| `echarts` | ECharts 交互式图表 | ECharts option JSON |
| `image` | 静态图片 | base64 data URL |
| `table` | 表格数据 | JSON数组 `[{col1: val1, ...}, ...]` |
| `csv` | CSV下载文件 | CSV data URL |

### 3. 存储结构

**会话目录结构：**
```
DataCacheDir/
└── sessions/
    ├── <thread_id_1>/              # 会话目录（唯一ID）
    │   ├── history.json            # 包含所有消息和绑定的图表数据
    │   ├── chat.log                # 详细日志（如果启用）
    │   └── files/                  # 生成的文件（CSV, Python图片等）
    │       ├── analysis_result.csv
    │       └── chart_20250112.png
    ├── <thread_id_2>/
    │   ├── history.json
    │   └── files/
    └── ...
```

**history.json 示例：**
```json
{
  "id": "1705123456789000000",
  "title": "数据分析会话",
  "data_source_id": "ds001",
  "created_at": 1705123456,
  "messages": [
    {
      "id": "msg001",
      "role": "user",
      "content": "分析销售趋势并生成图表",
      "timestamp": 1705123456,
      "chart_data": {
        "type": "echarts",
        "data": "{\"title\":{\"text\":\"销售趋势\"},\"xAxis\":{\"type\":\"category\",\"data\":[\"Jan\",\"Feb\",\"Mar\"]},\"yAxis\":{\"type\":\"value\"},\"series\":[{\"data\":[120,200,150],\"type\":\"line\"}]}"
      }
    },
    {
      "id": "msg002",
      "role": "assistant",
      "content": "根据数据分析，销售趋势呈上升态势...",
      "timestamp": 1705123460
    }
  ],
  "files": [
    {
      "name": "sales_chart.png",
      "path": "files/sales_chart.png",
      "type": "image/png",
      "size": 45231,
      "created_at": 1705123458
    }
  ]
}
```

## 数据流程

### 生成分析结果时

```
1. 用户发送分析请求
   ↓
2. 创建 ChatMessage (自动分配唯一ID)
   ↓
3. LLM 生成响应（包含图表代码块）
   ↓
4. 后端解析响应，检测图表类型
   ↓
5. 提取图表数据，创建 ChartData 对象
   ↓
6. 调用 attachChartToUserMessage()
   将 ChartData 附加到用户消息
   ↓
7. 保存到 history.json
   ↓
8. 发出 dashboard-update 事件
   更新前端实时显示
```

**代码实现（src/app.go）：**

```go
// 检测 ECharts 图表
reECharts := regexp.MustCompile("(?s)```[ \\t]*json:echarts\\s*({.*?})\\s*```")
matchECharts := reECharts.FindStringSubmatch(resp)
if len(matchECharts) > 1 {
    chartData = &ChartData{Type: "echarts", Data: matchECharts[1]}
    runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
        "sessionId": threadID,
        "type":      "echarts",
        "data":      matchECharts[1],
    })
}

// 附加到用户消息
if chartData != nil && threadID != "" {
    a.attachChartToUserMessage(threadID, chartData)
}
```

### 点击用户消息查看结果时

```
1. 用户点击有图表的消息
   ↓
2. ChatSidebar 发出 'user-message-clicked' 事件
   携带: { messageId, content, chartData }
   ↓
3. App.tsx 接收事件
   ↓
4. 设置状态:
   - setSelectedUserRequest(content)
   - setActiveChart(chartData)
   ↓
5. Dashboard 组件接收新 props
   ↓
6. 渲染:
   - 显示用户请求文本
   - 显示对应的图表
   - 如果无图表，显示友好提示
```

**代码实现：**

**ChatSidebar.tsx:**
```typescript
const handleUserMessageClick = (message: main.ChatMessage) => {
    EventsEmit('user-message-clicked', {
        messageId: message.id,
        content: message.content,
        chartData: message.chart_data  // 绑定的图表数据
    });
};
```

**App.tsx:**
```typescript
EventsOn("user-message-clicked", (payload: any) => {
    setSelectedUserRequest(payload.content);
    if (payload.chartData) {
        setActiveChart({
            type: payload.chartData.type,
            data: payload.chartData.data
        });
    } else {
        setActiveChart(null);  // 无图表，显示默认视图
    }
});
```

**Dashboard.tsx:**
```typescript
// 显示用户请求
{userRequestText && (
    <div className="bg-blue-50 border border-blue-100 rounded-lg p-3">
        <p className="text-xs font-semibold">Analysis Request</p>
        <p className="text-sm">{userRequestText}</p>

        {!activeChart && (
            <div className="bg-amber-50 text-amber-800">
                ⚠️ This analysis request has no visualization results yet.
            </div>
        )}
    </div>
)}

// 显示图表
{activeChart && (
    <section>
        <h2>Latest Analysis</h2>
        {renderChart()}  {/* 根据 type 渲染不同类型的图表 */}
    </section>
)}
```

## 交互体验

### 视觉指示

**有图表的用户消息：**
- 消息下方显示图标：📊 "Has visualization - Click to view"
- 鼠标悬停时：
  - 背景色变深 (hover:bg-blue-700)
  - 阴影增强 (hover:shadow-lg)
  - 轻微放大 (hover:scale-[1.02])
  - 鼠标变为指针 (cursor: pointer)
  - 显示 tooltip："Click to view analysis results on dashboard"

**无图表的用户消息：**
- 无特殊标记
- 不可点击

### 仪表盘显示

**情况 1：点击有图表的消息**
```
┌─────────────────────────────────────┐
│ 智能仪表盘                            │
│ Welcome back                         │
├─────────────────────────────────────┤
│ 📊 Analysis Request:                 │
│ "分析销售趋势并生成图表"               │
├─────────────────────────────────────┤
│ Latest Analysis                      │
│ [ECharts 交互式图表显示]               │
├─────────────────────────────────────┤
│ Key Metrics                          │
│ [数据源信息]                          │
├─────────────────────────────────────┤
│ Automated Insights                   │
│ [自动化洞察建议]                       │
└─────────────────────────────────────┘
```

**情况 2：点击无图表的消息**
```
┌─────────────────────────────────────┐
│ 智能仪表盘                            │
│ Welcome back                         │
├─────────────────────────────────────┤
│ 📊 Analysis Request:                 │
│ "这是什么数据库？"                     │
│ ⚠️ 此分析请求暂无可视化结果。           │
├─────────────────────────────────────┤
│ Key Metrics                          │
│ [数据源信息]                          │
├─────────────────────────────────────┤
│ Automated Insights                   │
│ [自动化洞察建议]                       │
└─────────────────────────────────────┘
```

## 图表类型优先级

当一个响应包含多种图表时，按以下优先级绑定（只绑定第一个检测到的）：

1. **ECharts** - 交互式图表（最高优先级）
   - 格式：` ```json:echarts\n{...}\n``` `
   - 支持缩放、tooltip、图例交互

2. **Image** - 静态图片
   - 格式：`![Chart](data:image/png;base64,...)`
   - Python matplotlib 生成的图表

3. **Table** - 表格数据
   - 格式：` ```json:table\n[{...}, ...]\n``` `
   - SQL 查询结果

4. **CSV** - CSV下载链接
   - 格式：`[Download](data:text/csv;base64,...)`

## 数据持久化

### 自动保存

- **时机：** 每次生成分析结果后立即保存
- **位置：** `DataCacheDir/sessions/<thread_id>/history.json`
- **格式：** JSON（包含所有消息和绑定的图表数据）

### 自动加载

- **时机：** 应用启动 / 切换会话
- **方法：** `LoadThreads()` 从 `history.json` 反序列化
- **结果：** 所有历史消息和图表数据完整恢复

## 性能考虑

### 数据大小

- **ECharts JSON：** 通常 < 10KB
- **Base64 图片：** 可能 50-500KB
- **表格数据：** 取决于行数，建议限制在 1000 行以内

### 优化措施

1. **SQL 结果截断：** 最多返回 1000 行
2. **工具输出截断：** 最多保留 3000 字符用于上下文
3. **图片压缩：** Python matplotlib 使用适当 DPI
4. **定期清理：** 建议定期删除旧会话

## 测试场景

### 场景 1：创建带图表的分析

```
操作：发送 "展示各类别的销售额"
预期：
  1. ✅ 生成 ECharts 柱状图
  2. ✅ 图表显示在仪表盘
  3. ✅ 用户消息显示 "Has visualization" 标记
  4. ✅ chart_data 保存到 history.json
```

### 场景 2：点击查看历史图表

```
操作：点击之前有图表的用户消息
预期：
  1. ✅ 仪表盘显示用户请求文本
  2. ✅ 仪表盘显示对应图表
  3. ✅ 无 "no visualization" 警告
```

### 场景 3：无图表请求

```
操作：发送 "数据库有多少张表？" → 点击该消息
预期：
  1. ✅ 显示用户请求文本
  2. ✅ 显示 "无可视化结果" 提示
  3. ✅ 显示默认数据源信息
```

### 场景 4：应用重启后恢复

```
操作：
  1. 创建分析生成图表
  2. 关闭应用
  3. 重新打开应用
  4. 点击之前的用户消息
预期：
  1. ✅ 图表数据从 history.json 加载
  2. ✅ 点击后正确显示图表
```

## 调试信息

### 后端日志

启用 `detailedLog` 后，会在 `sessions/<thread_id>/chat.log` 中记录：

```
[2025-01-12 10:30:45] USER REQUEST
分析销售趋势并生成图表

[2025-01-12 10:30:50] LLM RESPONSE
根据数据分析...
```json:echarts
{"title":{"text":"销售趋势"}...}
```

[2025-01-12 10:30:50] [CHART] Attached chart (type=echarts) to user message: msg001
```

### 前端控制台

添加了详细的调试日志：

```
[DEBUG] User message clicked: {messageId: "msg001", content: "...", chartData: {...}}
[DEBUG] Has chartData: true
[DEBUG] Chart type: echarts
[DEBUG] Chart data preview: {"title":{"text":"销售趋势"}...
[DEBUG] Active chart set: echarts

[Dashboard] userRequestText: 分析销售趋势并生成图表
[Dashboard] activeChart: echarts
```

## API 参考

### 后端

**ChartData 结构：**
```go
type ChartData struct {
    Type string `json:"type"` // "echarts" | "image" | "table" | "csv"
    Data string `json:"data"` // JSON字符串或base64/data URL
}
```

**ChatMessage 结构：**
```go
type ChatMessage struct {
    ID        string     `json:"id"`
    Role      string     `json:"role"`
    Content   string     `json:"content"`
    Timestamp int64      `json:"timestamp"`
    ChartData *ChartData `json:"chart_data,omitempty"`
}
```

**attachChartToUserMessage 方法：**
```go
func (a *App) attachChartToUserMessage(threadID string, chartData *ChartData)
```

### 前端

**事件：**
```typescript
// 发出事件（ChatSidebar）
EventsEmit('user-message-clicked', {
    messageId: string,
    content: string,
    chartData: ChartData | null
});

// 监听事件（App.tsx）
EventsOn("user-message-clicked", (payload: {
    messageId: string,
    content: string,
    chartData?: { type: string, data: string }
}) => { ... });
```

**组件属性：**
```typescript
// MessageBubble
interface MessageBubbleProps {
    onClick?: () => void;
    hasChart?: boolean;
}

// Dashboard
interface DashboardProps {
    data: main.DashboardData | null;
    activeChart?: { type: 'echarts' | 'image' | 'table' | 'csv', data: any } | null;
    userRequestText?: string | null;
}
```

## 总结

✅ **已实现的功能：**

1. **唯一 ID 分配** - 每个用户请求自动获得唯一纳秒时间戳 ID
2. **结果绑定** - ChartData 直接存储在 ChatMessage 中
3. **持久化存储** - 保存到 `sessions/<thread_id>/history.json`
4. **目录结构** - 每个会话有独立目录，包含 history.json 和 files/
5. **类型标注** - 正确标注为 echarts/image/table/csv
6. **点击查看** - 用户点击消息后在仪表盘显示对应结果
7. **视觉指示** - 有图表的消息显示标记和 hover 效果
8. **友好提示** - 无图表时显示友好提示信息

✅ **用户请求满足情况：**

| 需求 | 状态 | 说明 |
|------|------|------|
| 分配唯一ID给请求 | ✅ 已实现 | 使用纳秒时间戳 |
| 以ID为目录名保存结果 | ✅ 已实现 | `sessions/<thread_id>/` |
| 点击请求显示绑定结果 | ✅ 已实现 | user-message-clicked 事件 |
| 结果正确标注类型 | ✅ 已实现 | ChartData.Type |
| 方便再次显示 | ✅ 已实现 | 从 history.json 加载 |
| 鼠标悬停提示 | ✅ 已实现 | cursor-pointer + tooltip |

系统已经完整实现了分析结果与用户请求的绑定功能！
