# 会话冲突防护功能

## 概述

为了防止在分析进行中启动新会话造成干扰，系统现在会检测并阻止这种冲突情况。

## 功能说明

### 问题场景

用户在进行数据分析时（例如运行复杂的 SQL 查询或 Python 脚本），如果点击仪表盘上的自动化洞察项试图启动新的分析会话，可能会导致：

1. **资源冲突** - 同时运行多个分析任务
2. **上下文混乱** - 不同会话的结果混在一起
3. **性能下降** - 多个任务竞争系统资源

### 解决方案

系统实现了会话冲突检测机制：

1. **后端检查** - `CreateChatThread` 方法检查 `isChatGenerating` 标志
2. **拒绝创建** - 如果有活动会话，返回友好的错误消息
3. **前端提示** - 显示消息模态框，告知用户等待当前分析完成
4. **简化流程** - 直接尝试创建会话，冲突时自动显示警告（无需额外确认对话框）

## 实现细节

### 后端实现 (`src/app.go`)

```go
func (a *App) CreateChatThread(dataSourceID, title string) (ChatThread, error) {
    if a.chatService == nil {
        return ChatThread{}, fmt.Errorf("chat service not initialized")
    }

    // Check if there's an active analysis session running
    if a.isChatGenerating {
        return ChatThread{}, fmt.Errorf("当前有分析会话进行中，创建新的会话将影响现有分析会话。请等待当前分析完成后再创建新会话。")
    }

    thread, err := a.chatService.CreateThread(dataSourceID, title)
    if err != nil {
        return ChatThread{}, err
    }

    return thread, nil
}
```

**关键点：**
- 检查 `isChatGenerating` 标志（在 `SendMessage` 中设置）
- 返回中文错误消息，用户友好
- 阻止新会话创建，保护当前分析

### 前端实现 (`src/frontend/src/components/ChatSidebar.tsx`)

```typescript
const handleCreateThread = async (dataSourceId?: string, title?: string) => {
    try {
        const newThread = await CreateChatThread(dataSourceId || '', title || 'New Chat');
        setThreads(prev => [newThread, ...prev]);
        setActiveThreadId(newThread.id);
        return newThread;
    } catch (err: any) {
        console.error('Failed to create thread:', err);

        // Check if error is about active session conflict
        const errorMsg = err?.message || String(err);
        if (errorMsg.includes('分析会话进行中') || errorMsg.includes('active analysis')) {
            // Show user-friendly error message via MessageModal
            EventsEmit('show-message-modal', {
                type: 'warning',
                title: t('session_conflict_title') || '会话冲突',
                message: errorMsg
            });
        } else {
            // Generic error
            EventsEmit('show-message-modal', {
                type: 'error',
                title: t('create_session_failed') || '创建会话失败',
                message: errorMsg
            });
        }

        return null;
    }
};
```

**关键点：**
- 捕获错误并检查错误消息
- 使用消息模态框显示警告（而不是 alert）
- 区分会话冲突错误和其他错误
- 返回 null，阻止后续的自动发送消息操作

## 用户体验流程

### 正常情况

```
用户点击洞察项
    ↓
创建新会话成功
    ↓
打开聊天界面
    ↓
自动发送分析提示
    ↓
开始分析
```

### 冲突情况

```
用户点击洞察项
    ↓
尝试创建新会话
    ↓
后端检测到活动会话
    ↓
返回错误消息
    ↓
前端显示模态框提示：
"当前有分析会话进行中，创建新的会话将影响现有分析会话。
请等待当前分析完成后再创建新会话。"
    ↓
用户点击确定关闭提示
    ↓
当前分析继续进行，新会话未创建
```

## 错误消息

### 中文
```
当前有分析会话进行中，创建新的会话将影响现有分析会话。请等待当前分析完成后再创建新会话。
```

### 英文（建议）
```
An analysis session is currently in progress. Creating a new session may interfere with the ongoing analysis. Please wait for the current analysis to complete before creating a new session.
```

## 扩展建议

### 1. 显示进度指示

在仪表盘上显示当前分析的进度：

```tsx
{isChatGenerating && (
    <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
        <div className="flex items-center gap-2">
            <Loader2 className="w-5 h-5 animate-spin text-blue-600" />
            <span className="text-blue-700 font-medium">分析进行中...</span>
        </div>
    </div>
)}
```

### 2. 禁用洞察项

当有活动会话时，禁用洞察项的点击：

```tsx
<SmartInsight
    insight={insight}
    onClick={() => handleInsightClick(insight)}
    disabled={isChatGenerating}  // 新增
/>
```

### 3. 队列机制

实现会话队列，允许用户添加多个分析到队列，按顺序执行：

```
队列：[分析A（进行中）, 分析B（等待）, 分析C（等待）]
```

### 4. 多会话支持

如果系统资源允许，可以支持同时运行多个会话（每个会话独立的 `isChatGenerating` 标志）。

## 测试场景

### 场景 1：正常创建会话

1. 打开应用，无活动会话
2. 点击仪表盘洞察项
3. ✅ 成功创建会话
4. ✅ 自动发送分析提示

### 场景 2：会话冲突

1. 打开应用
2. 发送一个分析请求（开始分析）
3. 在分析进行中，点击仪表盘洞察项
4. ❌ 显示冲突警告
5. ✅ 不创建新会话
6. ✅ 当前分析继续进行

### 场景 3：分析完成后创建

1. 打开应用
2. 发送一个分析请求并等待完成
3. 分析完成后，点击仪表盘洞察项
4. ✅ 成功创建新会话

## 相关代码文件

- **后端：** `src/app.go` - CreateChatThread 方法
- **前端：** `src/frontend/src/components/ChatSidebar.tsx` - handleCreateThread 方法
- **前端：** `src/frontend/src/components/Dashboard.tsx` - 洞察项点击处理

## 总结

这个功能确保了：
- ✅ **数据安全** - 避免会话间的数据混乱
- ✅ **用户体验** - 友好的错误提示
- ✅ **系统稳定** - 防止资源冲突
- ✅ **简单实现** - 最小化代码更改

用户现在可以放心地进行数据分析，不必担心误操作导致的问题。
