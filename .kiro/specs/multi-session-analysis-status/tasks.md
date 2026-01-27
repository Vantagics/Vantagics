# Implementation Plan: Multi-Session Analysis Status Indicator

## Overview

本实现计划将多会话分析状态指示器功能分解为可执行的编码任务。实现将扩展现有的 `LoadingStateManager`，创建新的 React Hooks 和组件，并在三个关键位置（聊天区域、历史会话列表、仪表盘标题）集成状态显示。

## Tasks

- [x] 1. 增强 LoadingStateManager
  - [x] 1.1 扩展 SessionLoadingState 接口，添加 progress 和 error 字段
    - 在 `src/frontend/src/managers/LoadingStateManager.ts` 中更新接口定义
    - 添加 `progress` 对象（stage, progress, message, step, total）
    - 添加 `error` 对象（code, message）
    - _Requirements: 4.1, 4.2_

  - [x] 1.2 实现 updateProgress 和 setError 方法
    - 添加 `updateProgress(threadId, progress)` 方法
    - 添加 `setError(threadId, error)` 方法
    - 确保状态更新触发订阅者通知
    - _Requirements: 4.3, 5.1, 5.3_

  - [x] 1.3 实现会话特定订阅机制
    - 添加 `subscribeToSession(threadId, listener)` 方法
    - 实现只通知特定会话变化的逻辑
    - 返回取消订阅函数
    - _Requirements: 4.4_

  - [x] 1.4 添加 getLoadingCount 和 getSessionState 方法
    - 实现 `getLoadingCount()` 返回当前加载中的会话数量
    - 实现 `getSessionState(threadId)` 返回特定会话的完整状态
    - _Requirements: 3.1, 3.2, 4.1_

  - [x] 1.5 编写 LoadingStateManager 属性测试
    - **Property 1: Session State Independence**
    - **Validates: Requirements 1.5, 2.3, 4.1**

- [x] 2. 创建 React Hooks
  - [x] 2.1 增强 useLoadingState Hook
    - 在 `src/frontend/src/hooks/useLoadingState.ts` 中更新实现
    - 添加 `loadingCount` 和 `isAnyLoading` 返回值
    - 添加 `getProgress` 和 `getError` 方法
    - _Requirements: 3.1, 3.2_

  - [x] 2.2 创建 useSessionStatus Hook
    - 创建 `src/frontend/src/hooks/useSessionStatus.ts`
    - 实现针对单个会话的状态订阅
    - 返回 isLoading, progress, error, elapsedTime
    - _Requirements: 1.1, 1.2, 5.5_

  - [x] 2.3 编写 Hooks 属性测试
    - **Property 9: Session Switch State Display**
    - **Validates: Requirements 5.5**

- [x] 3. Checkpoint - 确保核心状态管理测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 4. 创建状态指示器组件
  - [x] 4.1 创建 AnalysisStatusIndicator 组件
    - 创建 `src/frontend/src/components/AnalysisStatusIndicator.tsx`
    - 实现三种显示模式：inline（内联）、compact（紧凑）、full（完整）
    - 显示加载动画、进度信息、错误状态
    - _Requirements: 1.1, 1.2, 1.4_

  - [x] 4.2 创建 GlobalAnalysisStatus 组件
    - 创建 `src/frontend/src/components/GlobalAnalysisStatus.tsx`
    - 显示全局加载状态和活跃分析数量
    - 实现悬停显示详情的 tooltip
    - _Requirements: 3.1, 3.2, 3.3_

  - [x] 4.3 编写组件属性测试
    - **Property 6: Global Status Aggregation**
    - **Validates: Requirements 3.1, 3.2**

- [x] 5. 集成到 ChatSidebar
  - [x] 5.1 在聊天区域集成 AnalysisStatusIndicator
    - 修改 `src/frontend/src/components/ChatSidebar.tsx`
    - 在 AI 回复前显示加载状态指示器
    - 使用 useSessionStatus Hook 获取当前会话状态
    - _Requirements: 1.1, 1.2, 1.3_

  - [x] 5.2 在历史会话列表集成 Spinner
    - 在会话列表项中添加加载状态指示
    - 使用 useLoadingState Hook 获取所有加载中的会话
    - 为加载中的会话显示转圈动画
    - _Requirements: 2.1, 2.2, 2.3_

  - [x] 5.3 编写 ChatSidebar 集成测试
    - **Property 2: Loading State Lifecycle**
    - **Validates: Requirements 1.1, 1.3, 4.2, 4.3**

- [x] 6. 集成到 DraggableDashboard
  - [x] 6.1 在仪表盘标题集成 GlobalAnalysisStatus
    - 修改 `src/frontend/src/components/DraggableDashboard.tsx`
    - 在标题右侧添加全局状态指示器
    - 显示正在进行的分析数量
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [x] 6.2 编写 Dashboard 集成测试
    - **Property 3: Progress Update Propagation**
    - **Validates: Requirements 1.2, 3.4**

- [x] 7. Checkpoint - 确保 UI 集成测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 8. 事件处理和状态同步
  - [x] 8.1 增强后端事件监听
    - 确保 LoadingStateManager 正确处理 analysis-progress 事件
    - 确保正确处理 analysis-completed 事件
    - 确保正确处理 analysis-error 事件
    - _Requirements: 5.1, 5.2, 5.3_

  - [x] 8.2 实现事件路由逻辑
    - 确保事件根据 threadId 正确路由到对应会话
    - 处理无效或缺失 threadId 的情况
    - _Requirements: 5.4_

  - [x] 8.3 编写事件处理属性测试
    - **Property 5: Event Routing Correctness**
    - **Validates: Requirements 5.1, 5.2, 5.4**

- [x] 9. 错误处理和清理
  - [x] 9.1 实现超时自动清理
    - 确保超时机制正确清理过期的加载状态
    - 配置默认超时时间（2分钟）
    - _Requirements: 4.6_

  - [x] 9.2 实现会话删除时的状态清理
    - 在删除会话时清理 LoadingStateManager 中的状态
    - 确保全局状态正确更新
    - _Requirements: 4.5_

  - [x] 9.3 编写清理和超时属性测试
    - **Property 8: Cleanup and Timeout**
    - **Validates: Requirements 4.5, 4.6**

- [x] 10. Final Checkpoint - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

## Notes

- 所有任务（包括测试任务）都是必需的
- 每个任务都引用了具体的需求以确保可追溯性
- 检查点确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况
