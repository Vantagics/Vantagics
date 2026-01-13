# 消息与图表绑定功能

## 概述

实现了将分析结果与用户请求绑定的功能。当用户点击聊天历史中的某个分析请求时，左侧智能仪表盘会显示该请求对应的可视化结果。

## 核心功能

### 1. 图表数据绑定

分析响应中生成的图表（ECharts、图片、表格、CSV）会自动关联到触发该分析的用户消息上。

### 2. 交互式消息查看

- 点击用户消息 → 仪表盘显示对应的图表结果
- 仪表盘标题下方显示用户的分析请求文本
- 如果请求没有图表结果，显示提示信息

### 3. 视觉指示

有图表结果的用户消息会显示特殊标记：
- "Has visualization - Click to view" 提示
- 可点击样式（hover 效果）

## 技术实现

### 后端实现

#### 1. ChatMessage 结构扩展

```go
// ChartData represents chart/visualization data associated with a message
type ChartData struct {
    Type string `json:"type"` // "echarts", "image", "table", "csv"
    Data string `json:"data"` // JSON string or base64/data URL
}

// ChatMessage represents a single message in a chat thread
type ChatMessage struct {
    ID        string     `json:"id"`
    Role      string     `json:"role"`
    Content   string     `json:"content"`
    Timestamp int64      `json:"timestamp"`
    ChartData *ChartData `json:"chart_data,omitempty"` // NEW
}
```

#### 2. SendMessage 方法增强

在 `app.go` 的 `SendMessage` 方法中：

```go
// 检测响应中的图表数据
var chartData *ChartData

// 优先级：ECharts > Image > Table > CSV
reECharts := regexp.MustCompile("(?s)```[ \\t]*json:echarts\\s*({.*?})\\s*```")
matchECharts := reECharts.FindStringSubmatch(resp)
if len(matchECharts) > 1 {
    chartData = &ChartData{Type: "echarts", Data: matchECharts[1]}
}

// ... 其他图表类型检测

// 将图表数据附加到用户消息
if chartData != nil && threadID != "" {
    a.attachChartToUserMessage(threadID, chartData)
}
```

#### 3. attachChartToUserMessage 辅助方法

```go
func (a *App) attachChartToUserMessage(threadID string, chartData *ChartData) {
    threads, _ := a.chatService.LoadThreads()

    // 查找目标线程
    var targetThread *ChatThread
    for i := range threads {
        if threads[i].ID == threadID {
            targetThread = &threads[i]
            break
        }
    }

    // 找到最后一条用户消息并附加图表数据
    for i := len(targetThread.Messages) - 1; i >= 0; i-- {
        if targetThread.Messages[i].Role == "user" {
            targetThread.Messages[i].ChartData = chartData
            break
        }
    }

    // 保存更新的线程
    a.chatService.SaveThreads([]ChatThread{*targetThread})
}
```

### 前端实现

#### 1. MessageBubble 组件增强

```typescript
interface MessageBubbleProps {
    role: 'user' | 'assistant';
    content: string;
    onClick?: () => void;
    hasChart?: boolean;  // NEW
}

// 渲染带有视觉指示的可点击消息
{isUser && hasChart && (
    <div className="mb-2 flex items-center gap-2 text-xs opacity-70">
        <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4z..."/>
        </svg>
        <span>Has visualization - Click to view</span>
    </div>
)}
```

#### 2. ChatSidebar 事件处理

```typescript
const handleUserMessageClick = (message: main.ChatMessage) => {
    EventsEmit('user-message-clicked', {
        messageId: message.id,
        content: message.content,
        chartData: message.chart_data
    });
};

// 在 MessageBubble 上添加点击处理
<MessageBubble
    role={msg.role as 'user' | 'assistant'}
    content={msg.content}
    onClick={msg.role === 'user' ? () => handleUserMessageClick(msg) : undefined}
    hasChart={msg.role === 'user' && !!msg.chart_data}
/>
```

#### 3. App.tsx 状态管理

```typescript
const [selectedUserRequest, setSelectedUserRequest] = useState<string | null>(null);

// 监听用户消息点击事件
const unsubscribeUserMessageClick = EventsOn("user-message-clicked", (payload: any) => {
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

#### 4. Dashboard 显示用户请求

```typescript
interface DashboardProps {
    data: main.DashboardData | null;
    activeChart?: { type: 'echarts' | 'image' | 'table' | 'csv', data: any } | null;
    userRequestText?: string | null;  // NEW
}

// 在仪表盘标题下方显示用户请求
{userRequestText && (
    <div className="mt-4 p-3 bg-blue-50 border border-blue-100 rounded-lg">
        <div className="flex items-start gap-2">
            <BarChart3 className="w-4 h-4 text-blue-600" />
            <div>
                <p className="text-xs font-semibold text-blue-900 uppercase">
                    Analysis Request
                </p>
                <p className="text-sm text-blue-800">{userRequestText}</p>
            </div>
        </div>
        {!activeChart && (
            <div className="mt-2 p-2 bg-amber-50 border border-amber-200 rounded text-xs">
                <span>⚠️ This analysis request has no visualization results yet.</span>
            </div>
        )}
    </div>
)}
```

## 用户体验流程

### 场景 1：查看有图表的分析请求

```
用户点击聊天历史中的消息："分析销售趋势"
    ↓
前端发出 "user-message-clicked" 事件
    ↓
App 更新 selectedUserRequest 和 activeChart
    ↓
仪表盘显示：
    ┌─────────────────────────────────┐
    │ 智能仪表盘                        │
    ├─────────────────────────────────┤
    │ 📊 Analysis Request:             │
    │ "分析销售趋势"                     │
    ├─────────────────────────────────┤
    │ [销售趋势图表]                     │
    └─────────────────────────────────┘
```

### 场景 2：查看无图表的分析请求

```
用户点击聊天历史中的消息："这是什么数据库？"
    ↓
前端发出 "user-message-clicked" 事件（chartData = null）
    ↓
App 更新 selectedUserRequest，清空 activeChart
    ↓
仪表盘显示：
    ┌─────────────────────────────────┐
    │ 智能仪表盘                        │
    ├─────────────────────────────────┤
    │ 📊 Analysis Request:             │
    │ "这是什么数据库？"                 │
    │ ⚠️ 此分析请求暂无可视化结果。        │
    ├─────────────────────────────────┤
    │ [显示数据源信息和指标]               │
    └─────────────────────────────────┘
```

### 场景 3：默认视图（无选中消息）

```
用户未点击任何消息
    ↓
仪表盘显示：
    ┌─────────────────────────────────┐
    │ 智能仪表盘                        │
    │ Welcome back                     │
    ├─────────────────────────────────┤
    │ [最新的分析图表（如有）]            │
    │ [关键指标]                        │
    │ [自动化洞察]                      │
    └─────────────────────────────────┘
```

## 数据持久化

图表数据存储在 `history.json` 中：

```json
{
  "id": "1705123456789000000",
  "title": "数据分析会话",
  "messages": [
    {
      "id": "msg001",
      "role": "user",
      "content": "分析销售趋势",
      "timestamp": 1705123456,
      "chart_data": {
        "type": "echarts",
        "data": "{\"xAxis\":{\"type\":\"category\",\"data\":[...]},\"yAxis\":{...}}"
      }
    },
    {
      "id": "msg002",
      "role": "assistant",
      "content": "根据数据分析，销售趋势呈上升态势...",
      "timestamp": 1705123460
    }
  ]
}
```

## 图表类型优先级

当一个响应包含多种图表时，按以下优先级绑定：

1. **ECharts** - 交互式图表（最高优先级）
2. **Image** - 静态图片（PNG/JPG）
3. **Table** - 表格数据
4. **CSV** - CSV 下载链接

只有第一个检测到的图表会被绑定到用户消息。

## API 变更

### 后端

新增数据结构：
- `ChartData` - 图表数据结构
- `ChatMessage.ChartData` - 消息关联的图表数据

新增方法：
- `attachChartToUserMessage(threadID, chartData)` - 将图表附加到用户消息

### 前端

新增事件：
- `user-message-clicked` - 用户点击消息时发出

新增组件属性：
- `MessageBubble.onClick` - 点击处理器
- `MessageBubble.hasChart` - 是否有关联图表
- `Dashboard.userRequestText` - 用户请求文本

## 优势

### 1. 上下文关联
- 图表与请求直接关联，清晰明了
- 可以快速回顾历史分析结果

### 2. 持久化存储
- 图表数据保存在 history.json
- 即使重启应用，图表关联依然存在

### 3. 直观反馈
- 有图表的消息有视觉标记
- 无图表时显示友好提示

### 4. 灵活交互
- 点击任意用户消息查看对应结果
- 支持在不同分析结果间快速切换

## 注意事项

### 1. 性能考虑

- 图表数据（特别是 base64 图片）可能较大
- history.json 文件大小会随图表数量增长
- 建议定期清理旧会话

### 2. 兼容性

- 与现有的 session-based 图表系统并存
- `dashboard-update` 事件仍然正常工作
- 新旧会话数据可以无缝共存

### 3. 未来改进

可能的增强方向：
1. **多图表支持** - 一个请求可以关联多个图表
2. **图表注释** - 为图表添加说明或标注
3. **图表比较** - 并排显示多个分析结果
4. **导出功能** - 导出请求及其图表为报告

## 相关文件

### 后端
- `src/chat_service.go` - ChatMessage 和 ChartData 结构定义
- `src/app.go` - SendMessage 和 attachChartToUserMessage 实现

### 前端
- `src/frontend/src/components/MessageBubble.tsx` - 消息气泡显示和点击处理
- `src/frontend/src/components/ChatSidebar.tsx` - 消息点击事件发送
- `src/frontend/src/App.tsx` - 事件监听和状态管理
- `src/frontend/src/components/Dashboard.tsx` - 用户请求和图表显示

## 测试场景

### 场景 1：创建带图表的分析
1. 发送分析请求："展示各类别的销售额"
2. 等待响应生成图表
3. 验证用户消息显示 "Has visualization" 标记

### 场景 2：点击查看图表
1. 点击有图表的用户消息
2. 验证仪表盘显示用户请求文本
3. 验证仪表盘显示对应图表

### 场景 3：无图表请求
1. 发送简单问题："数据库有多少张表？"
2. 点击该消息
3. 验证显示 "无可视化结果" 提示
4. 验证显示默认数据源信息

### 场景 4：切换不同请求
1. 点击第一个有图表的消息
2. 验证显示第一个图表
3. 点击第二个有图表的消息
4. 验证显示切换到第二个图表

## 总结

这个功能实现了将分析结果与用户请求的直接绑定，使用户能够：

- ✅ 快速查看每个请求的对应结果
- ✅ 在历史分析间自由切换
- ✅ 清晰了解哪些请求有可视化结果
- ✅ 获得友好的无结果提示

通过持久化存储和直观的交互设计，显著提升了数据分析的工作效率和用户体验。

---

**实现完成！用户现在可以点击任何分析请求，立即在仪表盘上查看对应的可视化结果！** 🎊
