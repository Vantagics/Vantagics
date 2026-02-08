# 授权模式切换功能实现总结

## 实现日期
2026-02-08

## 功能描述
用户可以在关于对话框中切换商业授权模式和开源模式，使用与应用启动时相同的激活对话框。

## 主要修改

### 1. 前端修改

#### AboutModal.tsx
- 移除了独立的 `ActivationModal` 组件
- 切换到商业模式时发送 `open-startup-mode-modal` 事件
- 添加 `activation-status-changed` 事件监听器，用于刷新激活状态
- 使用系统日志记录器

#### App.tsx
- 添加 `open-startup-mode-modal` 事件监听器
- 在主应用渲染块中添加 `StartupModeModal`（关键修复）
- 确保对话框在应用准备好后也能显示

#### StartupModeModal.tsx
- 激活成功后发送 `activation-status-changed` 事件
- 使用固定的授权服务器URL：`http://license.vantagedata.chat:6699`
- 添加系统日志记录

### 2. 后端修改
无需修改，使用现有的 `DeactivateLicense()` 和 `ActivateLicense()` 方法。

### 3. Bug 修复
- 修复了 `src/i18n/translations_en.go` 中的重复键
- 修复了 `src/i18n/translations_zh.go` 中的重复键
- 修复了 `StartupModeModal` 只在启动阶段渲染的问题

## 关键技术点

### 问题诊断
通过系统日志发现：
1. 事件正确发送和接收 ✅
2. 状态正确更新为 `true` ✅
3. 但组件没有渲染 ❌

### 根本原因
`StartupModeModal` 只在 `if (!isAppReady)` 条件块内渲染，导致应用准备好后无法显示。

### 解决方案
在主应用渲染块中也添加 `StartupModeModal`，使其在整个应用生命周期中都可用。

## 事件流程

### 切换到商业模式
```
AboutModal (用户点击确认)
  ↓
EventsEmit('open-startup-mode-modal')
  ↓
App.tsx 监听事件
  ↓
setShowStartupModeModal(true)
  ↓
StartupModeModal 显示
  ↓
用户激活
  ↓
EventsEmit('activation-status-changed')
  ↓
AboutModal 刷新状态
```

### 切换到开源模式
```
AboutModal (用户点击确认)
  ↓
DeactivateLicense()
  ↓
EventsEmit('open-settings', { tab: 'llm' })
  ↓
App.tsx 打开设置对话框（LLM页）
```

## 测试结果
✅ 功能正常工作
✅ 授权服务器URL与启动时一致
✅ 不向用户显示服务器URL
✅ 激活成功后状态自动刷新

## 相关文档
- `doc/LICENSE_MODE_SWITCH.md` - 详细的功能文档和测试指南
