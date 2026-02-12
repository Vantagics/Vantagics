# 授权模式切换功能

## 功能概述

用户可以在关于对话框中切换商业授权模式和开源模式。

## 功能特性

### 1. 从商业模式切换到开源模式

**前置条件检查：**
- 检查是否有正在进行的分析任务
- 如果有活动分析，阻止切换并显示错误消息

**切换流程：**
1. 用户点击"切换到开源模式"按钮
2. 显示确认对话框，警告用户授权将被取消
3. 用户确认后：
   - 调用后端 `DeactivateLicense()` 方法
   - 后端检查是否有活动分析（`HasActiveAnalysis()`）
   - 如果有活动分析，返回错误
   - 如果没有活动分析，清除授权数据
4. 切换成功后：
   - 关闭关于对话框
   - 自动打开设置对话框
   - 切换到 LLM 配置页
   - 用户必须配置自己的 LLM API

### 2. 从开源模式切换到商业模式

**切换流程：**
1. 用户点击"切换到商业模式"按钮
2. 显示确认对话框
3. 用户确认后：
   - 关闭关于对话框
   - 发送 `open-startup-mode-modal` 事件
   - 打开启动模式对话框（与应用启动时使用的相同）
   - 用户输入序列号进行激活
   - 激活成功后，发送 `activation-status-changed` 事件
   - 关于对话框监听该事件并刷新激活状态

**重要说明：**
- 授权服务器URL使用固定值（与启动时激活一致）：`https://license.vantagedata.chat`
- 不向用户显示服务器URL输入框
- 使用与启动时相同的激活对话框（`StartupModeModal`），确保体验一致

## 技术实现

### 后端 (Go)

**新增方法：**

```go
// HasActiveAnalysis 检查是否有活动的分析会话
func (a *App) HasActiveAnalysis() bool

// DeactivateLicense 取消激活授权（修改为返回error）
func (a *App) DeactivateLicense() error
```

**修改点：**
- `DeactivateLicense()` 现在返回 `error` 类型
- 在取消激活前检查是否有活动分析
- 如果有活动分析，返回本地化的错误消息

### 前端 (TypeScript/React)

**修改文件：**

1. `src/frontend/src/components/AboutModal.tsx`
   - 移除独立的 `ActivationModal` 组件引用
   - 修改 `handleConfirm` 方法，发送 `open-startup-mode-modal` 事件
   - 添加 `activation-status-changed` 事件监听器，用于刷新激活状态
   - 成功取消激活后发送 `open-settings` 事件

2. `src/frontend/src/App.tsx`
   - 添加 `open-startup-mode-modal` 事件监听器
   - 监听到事件后打开 `StartupModeModal`
   - 修改 `open-settings` 事件监听器，支持传递 `tab` 参数
   - 设置 `preferenceInitialTab` 状态

3. `src/frontend/src/components/StartupModeModal.tsx`
   - 激活成功后发送 `activation-status-changed` 事件
   - 通知其他组件（如 `AboutModal`）刷新激活状态

4. `src/frontend/src/components/PreferenceModal.tsx`
   - 已支持 `initialTab` 属性
   - 打开时自动切换到指定标签页

## 国际化

所有相关的翻译键已在 `src/frontend/src/i18n.ts` 中定义：

- `switch_to_commercial` - 切换到商业模式
- `switch_to_opensource` - 切换到开源模式
- `confirm_switch_to_commercial` - 确认切换到商业模式
- `confirm_switch_to_commercial_desc` - 商业模式切换说明
- `confirm_switch_to_opensource` - 确认切换到开源模式
- `confirm_switch_to_opensource_desc` - 开源模式切换说明
- `deactivate_failed` - 取消激活失败

## 用户体验

### 成功场景
1. 用户在没有活动分析时切换模式
2. 系统顺利完成切换
3. 切换到商业模式时，打开与启动时相同的激活对话框
4. 激活成功后，关于对话框自动刷新显示新的激活状态
5. 切换到开源模式时，自动打开设置对话框的 LLM 配置页
6. 用户配置 LLM API 后可继续使用

### 失败场景
1. 用户在有活动分析时尝试切换
2. 系统显示错误消息："当前有正在进行的分析任务，无法切换模式"
3. 确认对话框保持打开，用户可以：
   - 点击"取消"关闭对话框
   - 等待分析完成后重试

## 事件流程

### 切换到商业模式
```
AboutModal (用户点击) 
  → 确认对话框 
  → EventsEmit('open-startup-mode-modal') 
  → App.tsx 监听事件 
  → 打开 StartupModeModal 
  → 用户激活 
  → EventsEmit('activation-status-changed') 
  → AboutModal 监听事件 
  → 刷新激活状态
```

### 切换到开源模式
```
AboutModal (用户点击) 
  → 确认对话框 
  → DeactivateLicense() 
  → EventsEmit('open-settings', { tab: 'llm' }) 
  → App.tsx 监听事件 
  → 打开设置对话框（LLM页）
```

## 测试建议

1. **正常切换测试**
   - 在没有活动分析时切换模式
   - 验证启动模式对话框正确打开
   - 验证激活成功后关于对话框状态刷新
   - 验证设置对话框自动打开并切换到 LLM 页

2. **阻止切换测试**
   - 启动一个分析任务
   - 尝试切换模式
   - 验证显示错误消息
   - 验证授权状态未改变

3. **国际化测试**
   - 在中文环境下测试
   - 在英文环境下测试
   - 验证所有消息正确显示

4. **边界情况测试**
   - 网络错误时的处理
   - 快速连续点击切换按钮
   - 在切换过程中关闭对话框
   - 激活对话框与关于对话框的交互

5. **授权服务器一致性测试**
   - 验证启动时激活使用的服务器URL
   - 验证关于对话框切换时使用相同的服务器URL
   - 验证服务器URL不显示给用户
