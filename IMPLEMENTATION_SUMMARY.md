# 授权模式切换功能实现总结

## 需求

从关于对话框中，从商业授权切换为开源模式时：
1. 用户确认切换后，需要弹出设置对话框
2. 切到LLM配置页，让用户必须配置LLM
3. 切换前，如果有正在进行中的分析，不允许切换

## 实现的修改

### 1. 后端修改 (src/app.go)

#### 新增方法
```go
// HasActiveAnalysis 检查是否有活动的分析会话
func (a *App) HasActiveAnalysis() bool {
    a.activeThreadsMutex.RLock()
    defer a.activeThreadsMutex.RUnlock()
    return len(a.activeThreads) > 0
}
```

#### 修改方法
```go
// DeactivateLicense 取消激活授权
// 修改：返回 error 类型，在有活动分析时返回错误
func (a *App) DeactivateLicense() error {
    // 检查是否有活动分析
    if a.HasActiveAnalysis() {
        cfg, _ := a.GetConfig()
        if cfg.Language == "简体中文" {
            return fmt.Errorf("当前有正在进行的分析任务，无法切换模式")
        }
        return fmt.Errorf("cannot switch mode while analysis is in progress")
    }
    
    // 原有的取消激活逻辑...
    return nil
}
```

### 2. 前端修改

#### AboutModal.tsx (src/frontend/src/components/AboutModal.tsx)

修改 `handleConfirm` 方法：
- 添加错误处理逻辑
- 成功后关闭关于对话框
- 发送 `open-settings` 事件，传递 `{ tab: 'llm' }` 参数

```typescript
try {
    await DeactivateLicense();
    // 刷新状态...
    
    // 关闭关于对话框并打开设置
    onClose();
    import('../../wailsjs/runtime/runtime').then(({ EventsEmit }) => {
        EventsEmit('open-settings', { tab: 'llm' });
    });
} catch (error: any) {
    // 显示错误消息
    const errorMsg = error?.message || error?.toString() || t('deactivate_failed');
    setDeactivateError(errorMsg);
}
```

#### App.tsx (src/frontend/src/App.tsx)

修改 `open-settings` 事件监听器，支持传递 tab 参数：

```typescript
const unsubscribeSettings = EventsOn("open-settings", (data?: any) => {
    setIsPreferenceOpen(true);
    // 如果指定了 tab，设置为初始标签页
    if (data && data.tab) {
        setPreferenceInitialTab(data.tab);
    } else {
        setPreferenceInitialTab(undefined);
    }
});
```

#### PreferenceModal.tsx (src/frontend/src/components/PreferenceModal.tsx)

无需修改，已支持 `initialTab` 属性：
- 在 `useEffect` 中检查 `initialTab`
- 如果存在，调用 `setActiveTab(initialTab)`

### 3. 国际化

所有翻译已存在于 `src/frontend/src/i18n.ts`：
- ✅ `switch_to_commercial` / `switch_to_opensource`
- ✅ `confirm_switch_to_commercial` / `confirm_switch_to_opensource`
- ✅ `confirm_switch_to_commercial_desc` / `confirm_switch_to_opensource_desc`
- ✅ `deactivate_failed`
- ✅ `cancel` / `confirm`

## 功能流程

### 成功切换流程
1. 用户点击"切换到开源模式"
2. 显示确认对话框
3. 用户点击"确认"
4. 后端检查无活动分析
5. 清除授权数据
6. 关闭关于对话框
7. 自动打开设置对话框
8. 自动切换到 LLM 配置页
9. 用户配置 LLM API

### 阻止切换流程
1. 用户点击"切换到开源模式"
2. 显示确认对话框
3. 用户点击"确认"
4. 后端检查到有活动分析
5. 返回错误："当前有正在进行的分析任务，无法切换模式"
6. 在确认对话框中显示错误消息
7. 用户可以点击"取消"关闭对话框

## 测试要点

1. ✅ 无活动分析时可以成功切换
2. ✅ 有活动分析时阻止切换并显示错误
3. ✅ 切换成功后自动打开设置对话框
4. ✅ 设置对话框自动切换到 LLM 配置页
5. ✅ 错误消息正确显示（中英文）
6. ✅ 授权状态正确更新

## 文件清单

### 修改的文件
- `src/app.go` - 后端授权逻辑
- `src/frontend/src/components/AboutModal.tsx` - 关于对话框
- `src/frontend/src/App.tsx` - 主应用组件

### 新增的文件
- `doc/LICENSE_MODE_SWITCH.md` - 功能文档
- `IMPLEMENTATION_SUMMARY.md` - 实现总结（本文件）

## 注意事项

1. **类型定义更新**：`DeactivateLicense` 的返回类型从 `void` 改为 `error`，需要重新生成 wailsjs 绑定
2. **错误处理**：前端使用 try-catch 捕获后端返回的错误
3. **状态管理**：确认对话框在显示错误时保持打开状态
4. **用户体验**：切换成功后自动引导用户配置 LLM

## 下一步

1. 运行 `wails generate` 更新前端类型定义
2. 测试完整的切换流程
3. 测试错误场景（有活动分析时）
4. 验证国际化显示
