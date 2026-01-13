# 图表数据与分析请求绑定机制详解

## 完整数据流程

### 1️⃣ 数据结构定义

**src/chat_service.go (第15-27行):**
```go
// ChartData 存储图表数据
type ChartData struct {
	Type string `json:"type"` // "echarts", "image", "table", "csv"
	Data string `json:"data"` // JSON string or base64/data URL
}

// ChatMessage 用户消息结构
type ChatMessage struct {
	ID        string     `json:"id"`
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Timestamp int64      `json:"timestamp"`
	ChartData *ChartData `json:"chart_data,omitempty"` // ✅ 关联的图表数据
}
```

**关键点：**
- `ChartData` 是一个指针字段，可以为 nil（无图表）
- 使用 `omitempty` 标签，无图表时不会序列化该字段
- 序列化到 JSON 时保留完整结构

---

### 2️⃣ 图表生成与绑定（生成时）

**src/app.go (第728-809行):**

```go
// SendChatMessage 函数中的图表检测与绑定流程

func (a *App) SendChatMessage(threadID, message string) error {
    // ... 调用 LLM 生成响应 ...

    var chartData *ChartData

    // 🔍 1. 检测 ECharts 图表（优先级最高）
    reECharts := regexp.MustCompile(`(?s)` + "```" + `[ \t]*json:echarts\s*({.*?})\s*` + "```")
    matchECharts := reECharts.FindStringSubmatch(resp)
    if len(matchECharts) > 1 {
        chartData = &ChartData{Type: "echarts", Data: matchECharts[1]}
        // 发送实时更新到前端
        runtime.EventsEmit(a.ctx, "dashboard-update", ...)
    }

    // 🔍 2. 检测 Base64 图片
    if chartData == nil {
        reImage := regexp.MustCompile(`!\[.*?\]\((data:image\/.*?;base64,.*?)\)`)
        matchImage := reImage.FindStringSubmatch(resp)
        if len(matchImage) > 1 {
            chartData = &ChartData{Type: "image", Data: matchImage[1]}
            runtime.EventsEmit(a.ctx, "dashboard-update", ...)
        }
    }

    // 🔍 3. 检测表格数据
    if chartData == nil {
        reTable := regexp.MustCompile(`(?s)` + "```" + `[ \t]*json:table\s*(\[.*?\])\s*` + "```")
        matchTable := reTable.FindStringSubmatch(resp)
        if len(matchTable) > 1 {
            var tableData []map[string]interface{}
            json.Unmarshal([]byte(matchTable[1]), &tableData)
            tableDataJSON, _ := json.Marshal(tableData)
            chartData = &ChartData{Type: "table", Data: string(tableDataJSON)}
            runtime.EventsEmit(a.ctx, "dashboard-update", ...)
        }
    }

    // 🔍 4. 检测 CSV 下载链接
    if chartData == nil {
        reCSV := regexp.MustCompile(`\[.*?\]\((data:text/csv;base64,[A-Za-z0-9+/=]+)\)`)
        matchCSV := reCSV.FindStringSubmatch(resp)
        if len(matchCSV) > 1 {
            chartData = &ChartData{Type: "csv", Data: matchCSV[1]}
            runtime.EventsEmit(a.ctx, "dashboard-update", ...)
        }
    }

    // ✅ 关键：附加图表数据到用户消息
    if chartData != nil && threadID != "" {
        a.attachChartToUserMessage(threadID, chartData)
    }
}
```

**优先级顺序：**
1. ECharts 交互式图表（最优先）
2. 静态图片（matplotlib 等）
3. 表格数据
4. CSV 文件

---

### 3️⃣ 绑定实现（核心逻辑）

**src/app.go (第838-877行):**

```go
// attachChartToUserMessage 将图表数据附加到最后一条用户消息
func (a *App) attachChartToUserMessage(threadID string, chartData *ChartData) {
    if a.chatService == nil {
        return
    }

    // 🔄 1. 加载所有会话
    threads, err := a.chatService.LoadThreads()
    if err != nil {
        a.Log(fmt.Sprintf("attachChartToUserMessage: Failed to load threads: %v", err))
        return
    }

    // 🔍 2. 找到目标会话
    var targetThread *ChatThread
    for i := range threads {
        if threads[i].ID == threadID {
            targetThread = &threads[i]
            break
        }
    }

    if targetThread == nil {
        a.Log(fmt.Sprintf("attachChartToUserMessage: Thread %s not found", threadID))
        return
    }

    // 🎯 3. 找到最后一条用户消息并附加图表数据
    for i := len(targetThread.Messages) - 1; i >= 0; i-- {
        if targetThread.Messages[i].Role == "user" {
            targetThread.Messages[i].ChartData = chartData // ✅ 绑定图表
            a.Log(fmt.Sprintf("[CHART] Attached chart (type=%s) to user message: %s",
                chartData.Type, targetThread.Messages[i].ID))
            break
        }
    }

    // 💾 4. 保存更新后的会话（包含图表数据）
    if err := a.chatService.SaveThreads([]ChatThread{*targetThread}); err != nil {
        a.Log(fmt.Sprintf("attachChartToUserMessage: Failed to save thread: %v", err))
    }
}
```

**逻辑：**
1. 倒序遍历消息列表
2. 找到最后一条用户消息（最近的分析请求）
3. 附加 `ChartData` 对象
4. 调用 `SaveThreads` 持久化到磁盘

---

### 4️⃣ 持久化存储

**src/chat_service.go (第156-170行):**

```go
// saveThreadInternal 保存单个会话
func (s *ChatService) saveThreadInternal(t ChatThread) error {
    path := s.getThreadPath(t.ID)  // DataCacheDir/sessions/<threadID>/history.json
    dir := filepath.Dir(path)

    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    // ✅ 序列化整个 ChatThread，包括所有 Messages 和 ChartData
    data, err := json.MarshalIndent(t, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)  // 写入 history.json
}
```

**保存位置：**
```
DataCacheDir/
└── sessions/
    └── <thread_id>/
        ├── history.json  ← 包含完整的消息和图表数据
        └── files/
            └── chart.png
```

**history.json 示例：**
```json
{
  "id": "1736688000000000000",
  "title": "价格弹性分析",
  "data_source_id": "ds001",
  "created_at": 1736688000,
  "messages": [
    {
      "id": "msg001",
      "role": "user",
      "content": "价格弹性分析：分析价格变化对销量的影响",
      "timestamp": 1736688000,
      "chart_data": {                    // ✅ 图表数据已绑定
        "type": "echarts",
        "data": "{\"title\":{\"text\":\"价格弹性曲线\"},\"xAxis\":{...}}"
      }
    },
    {
      "id": "msg002",
      "role": "assistant",
      "content": "根据分析，价格弹性系数为 -1.8...",
      "timestamp": 1736688010
    }
  ]
}
```

---

### 5️⃣ 加载与显示（查看时）

**前端流程：**

**① 消息渲染（ChatSidebar.tsx 第617-622行）:**
```typescript
<MessageBubble
    role={msg.role as 'user' | 'assistant'}
    content={msg.content}
    onClick={msg.role === 'user' ? () => handleUserMessageClick(msg) : undefined}
    hasChart={msg.role === 'user' && !!msg.chart_data}  // ✅ 检查是否有图表
/>
```

**② 点击事件处理（ChatSidebar.tsx 第444-451行）:**
```typescript
const handleUserMessageClick = (message: main.ChatMessage) => {
    // ✅ 发送事件到 App.tsx，携带图表数据
    EventsEmit('user-message-clicked', {
        messageId: message.id,
        content: message.content,
        chartData: message.chart_data  // ✅ 传递图表数据
    });
};
```

**③ 仪表盘更新（App.tsx 第156-168行）:**
```typescript
EventsOn("user-message-clicked", (payload: any) => {
    setSelectedUserRequest(payload.content);

    if (payload.chartData) {
        // ✅ 设置活动图表，触发 Dashboard 重新渲染
        setActiveChart({
            type: payload.chartData.type,
            data: payload.chartData.data
        });
    } else {
        setActiveChart(null);  // 无图表，显示提示
    }
});
```

**④ 仪表盘渲染（Dashboard.tsx 第38-156行）:**
```typescript
const renderChart = () => {
    if (!activeChart) return null;

    if (activeChart.type === 'echarts') {
        // ✅ 渲染 ECharts 图表
        const options = JSON.parse(activeChart.data);
        return <Chart options={options} height="400px" />;
    }

    if (activeChart.type === 'image') {
        // ✅ 渲染静态图片
        return <img src={activeChart.data} alt="Analysis Chart" />;
    }

    if (activeChart.type === 'table') {
        // ✅ 渲染表格
        const tableData = activeChart.data as any[];
        return <table>...</table>;
    }

    // ... CSV 等
};
```

---

## 完整数据流程图

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. 用户发送分析请求 "价格弹性分析"                                  │
└────────────────────────┬────────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. LLM 生成响应（包含 ECharts JSON）                               │
│    ```json:echarts                                               │
│    {"title": {"text": "价格弹性曲线"}, ...}                        │
│    ```                                                           │
└────────────────────────┬────────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. app.go 检测图表类型（正则匹配）                                 │
│    ✅ 发现 ECharts → chartData = {type: "echarts", data: "..."}  │
└────────────────────────┬────────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. attachChartToUserMessage(threadID, chartData)                │
│    ├─ 加载会话 threads                                            │
│    ├─ 找到最后一条用户消息                                         │
│    ├─ msg.ChartData = chartData  ✅ 绑定图表                      │
│    └─ SaveThreads() → 写入 history.json                          │
└────────────────────────┬────────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. 磁盘存储（持久化）                                              │
│    sessions/<thread_id>/history.json                            │
│    {                                                             │
│      "messages": [                                               │
│        {                                                         │
│          "id": "msg001",                                         │
│          "role": "user",                                         │
│          "content": "价格弹性分析",                                │
│          "chart_data": {          ✅ 已保存                       │
│            "type": "echarts",                                    │
│            "data": "{...}"                                       │
│          }                                                       │
│        }                                                         │
│      ]                                                           │
│    }                                                             │
└────────────────────────┬────────────────────────────────────────┘
                         ▼
                    [应用重启]
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 6. 加载会话（LoadThreads）                                         │
│    读取 history.json → 反序列化为 ChatMessage[]                    │
│    ✅ chart_data 字段自动恢复                                      │
└────────────────────────┬────────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 7. 前端渲染（ChatSidebar）                                         │
│    <MessageBubble                                                │
│      hasChart={!!msg.chart_data}  ✅ 显示图表图标                  │
│      onClick={() => handleUserMessageClick(msg)}                │
│    />                                                            │
└────────────────────────┬────────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 8. 用户点击消息                                                    │
│    EventsEmit('user-message-clicked', {                         │
│      messageId: msg.id,                                          │
│      content: msg.content,                                       │
│      chartData: msg.chart_data  ✅ 传递图表数据                    │
│    })                                                            │
└────────────────────────┬────────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 9. App.tsx 接收事件                                                │
│    setActiveChart(payload.chartData)  ✅ 更新状态                  │
└────────────────────────┬────────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 10. Dashboard 渲染图表                                             │
│     renderChart() → 根据 type 渲染 ECharts/Image/Table            │
│     ✅ 用户看到之前的分析结果                                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## 测试验证

### 如何验证绑定是否工作？

**1. 查看日志：**
```
[CHART] Attached chart (type=echarts) to user message: msg001
```

**2. 检查 history.json：**
```bash
cat DataCacheDir/sessions/<thread_id>/history.json
```

查找：
```json
{
  "role": "user",
  "content": "价格弹性分析...",
  "chart_data": {  ← 应该存在
    "type": "echarts",
    "data": "..."
  }
}
```

**3. 前端测试：**
- 生成一个图表分析
- 刷新页面或重启应用
- 点击该用户消息
- 检查仪表盘是否显示图表（而不是"无可视化结果"）

**4. 浏览器控制台：**
```
[DEBUG] User message clicked: {messageId: "...", chartData: {...}}
[Dashboard] activeChart: echarts
```

---

## 常见问题

### Q1: 旧会话没有图表数据？
**原因：** 功能是新增的，旧会话的 history.json 中没有 `chart_data` 字段。

**解决：** 重新运行分析，新的结果会被绑定。

### Q2: 点击消息后显示"无可视化结果"？
**检查：**
1. `history.json` 中该消息是否有 `chart_data` 字段
2. 浏览器控制台是否有 `chartData: null`
3. 后端日志是否有 "Attached chart" 消息

### Q3: 图表类型检测失败？
**原因：** LLM 输出格式不正确（缺少代码块标记）

**检查：**
- 是否有 ` ```json:echarts ` 标记（注意冒号）
- 是否有完整的 JSON 对象
- 日志中是否有匹配成功的消息

---

## 总结

✅ **功能完整性：**
1. ✅ 数据结构支持（ChatMessage.ChartData）
2. ✅ 图表检测与绑定（attachChartToUserMessage）
3. ✅ 持久化存储（SaveThreads → history.json）
4. ✅ 加载恢复（LoadThreads → 反序列化）
5. ✅ 前端交互（点击事件 → 仪表盘更新）
6. ✅ 视觉指示（hasChart → 图标和悬停效果）

✅ **数据流完整：**
```
生成 → 检测 → 绑定 → 保存 → 加载 → 点击 → 显示
```

这是一个端到端的完整实现，确保了分析结果与用户请求的永久绑定！
