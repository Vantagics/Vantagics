# Implementation Plan: Smart Insight Click Behavior Optimization

## Overview

本实现计划将智能洞察点击行为优化功能分解为一系列增量式的编码任务。每个任务都建立在前一个任务的基础上，确保功能逐步完善并可随时测试验证。

## Tasks

- [ ] 1. 重构 App.tsx 状态管理
  - [x] 1.1 添加新的状态变量用于请求跟踪
    - 在 App.tsx 中添加 `pendingRequestId` 状态（string | null）
    - 添加 `lastCompletedRequestId` 状态（string | null）
    - 添加请求ID生成函数 `generateRequestId()`
    - _Requirements: 2.1, 4.1_

  - [ ]* 1.2 编写状态管理的单元测试
    - 测试 `generateRequestId()` 生成唯一ID
    - 测试状态变量初始化
    - 测试状态更新逻辑
    - _Requirements: 2.1, 4.1_

- [ ] 2. 实现洞察点击处理逻辑
  - [x] 2.1 修改 DraggableDashboard.tsx 的洞察点击处理
    - 更新 `handleInsightClick` 函数，接收 `onInsightClick` 回调
    - 确保点击时不清空当前显示数据
    - 传递洞察文本到父组件
    - _Requirements: 1.1, 6.1_

  - [x] 2.2 在 App.tsx 中实现 `handleInsightClick` 函数
    - 生成唯一的 requestId
    - 设置 `pendingRequestId` 和 `isAnalysisLoading` 状态
    - **关键**: 不修改 `dashboardData` 状态
    - 通过 EventsEmit 发送分析请求，包含 requestId
    - _Requirements: 1.1, 1.2, 2.1, 2.2, 4.2_

  - [ ]* 2.3 编写洞察点击的属性测试
    - **Property 1: Dashboard data persists during loading**
    - **Validates: Requirements 1.1, 1.2, 4.2**
    - 使用 fast-check 生成随机仪表盘数据
    - 验证点击后数据保持不变
    - 验证加载状态正确设置
    - _Requirements: 1.1, 1.2, 4.2_

- [ ] 3. 实现分析结果接收和验证逻辑
  - [x] 3.1 修改 `analysis-completed` 事件处理器
    - 在事件 payload 中添加 requestId 字段
    - 实现 requestId 匹配验证逻辑
    - 只有匹配的 requestId 才更新 `dashboardData`
    - 不匹配的结果记录日志并忽略
    - _Requirements: 1.3, 4.3, 4.4_

  - [x] 3.2 更新加载状态清除逻辑
    - 在结果匹配时清除 `pendingRequestId`
    - 设置 `lastCompletedRequestId`
    - 清除 `isAnalysisLoading` 标志
    - _Requirements: 2.3_

  - [ ]* 3.3 编写请求ID匹配的属性测试
    - **Property 3: Only matching requestId updates dashboard**
    - **Validates: Requirements 4.3, 4.4**
    - 生成随机的当前和过期 requestId
    - 验证过期结果被忽略
    - 验证匹配结果更新数据
    - _Requirements: 4.3, 4.4_

- [ ] 4. 实现请求取消和去重机制
  - [x] 4.1 实现快速连续点击的请求取消
    - 在 `handleInsightClick` 中检查是否有待处理请求
    - 如果有，取消前一个请求（更新 pendingRequestId）
    - 确保只有最新请求被处理
    - _Requirements: 5.3, 5.4_

  - [ ]* 4.2 编写请求取消的属性测试
    - **Property 4: Only most recent request is processed**
    - **Validates: Requirements 5.3, 5.4**
    - 生成随机的洞察点击序列
    - 验证只有最后一个请求被处理
    - 验证最终显示最后一个请求的结果
    - _Requirements: 5.3, 5.4_

- [x] 5. 检查点 - 核心功能验证
  - 手动测试洞察点击流程
  - 验证仪表盘内容在加载期间保持不变
  - 验证新结果正确显示
  - 验证快速点击只处理最后一个请求
  - 如有问题，询问用户

- [ ] 6. 实现错误处理机制
  - [x] 6.1 添加请求超时处理
    - 实现 30 秒超时机制
    - 超时后清除加载状态但保留数据
    - 显示超时错误提示
    - _Requirements: 2.4_

  - [x] 6.2 实现分析失败处理
    - 监听 `analysis-error` 事件
    - 验证 requestId 匹配
    - 清除加载状态，保留现有数据
    - 显示错误提示
    - _Requirements: 2.4_

  - [x] 6.3 实现会话切换时的请求清理
    - 在 `session-switched` 事件处理器中
    - 取消当前会话的待处理请求
    - 清除加载状态
    - _Requirements: 2.4_

  - [ ]* 6.4 编写错误处理的单元测试
    - 测试超时场景
    - 测试请求失败场景
    - 测试会话切换场景
    - 验证数据保持不变
    - _Requirements: 2.4_

- [ ] 7. 优化用户界面反馈
  - [ ] 7.1 在 DraggableDashboard 中添加加载覆盖层
    - 创建 LoadingOverlay 组件
    - 在 `isAnalysisLoading` 为 true 时显示
    - 显示"正在分析..."消息
    - 不阻止用户查看当前内容
    - _Requirements: 3.2_

  - [ ] 7.2 优化洞察项的点击反馈
    - 添加点击时的视觉反馈（高亮、动画）
    - 在加载期间禁用洞察项点击
    - 添加"点击深入分析"提示文本
    - _Requirements: 3.1_

  - [ ]* 7.3 编写 UI 组件的单元测试
    - 测试 LoadingOverlay 显示/隐藏
    - 测试洞察项点击反馈
    - 测试加载期间的交互状态
    - _Requirements: 3.1, 3.2_

- [ ] 8. 更新事件通信机制
  - [ ] 8.1 修改后端事件 payload 结构
    - 在 `chat-send-message-in-session` 事件中添加 requestId
    - 在 `analysis-completed` 事件中添加 requestId
    - 确保事件数据结构一致性
    - _Requirements: 6.1, 6.2, 6.3_

  - [ ]* 8.2 编写事件通信的属性测试
    - **Property 5: Event communication integrity**
    - **Validates: Requirements 6.1, 6.2, 6.3**
    - 验证事件正确发送和接收
    - 验证 requestId 正确传递
    - 验证事件处理器正确调用
    - _Requirements: 6.1, 6.2, 6.3_

- [ ] 9. 检查点 - 完整功能测试
  - 运行所有单元测试和属性测试
  - 手动测试完整用户流程
  - 测试错误场景（超时、失败）
  - 测试边缘情况（快速点击、会话切换）
  - 如有问题，询问用户

- [ ] 10. 性能优化和代码清理
  - [ ] 10.1 优化组件渲染性能
    - 使用 React.memo 包装 DraggableDashboard
    - 使用 useMemo 缓存计算结果
    - 使用 useCallback 优化事件处理器
    - _Requirements: 5.1, 5.2_

  - [ ] 10.2 添加事件去抖处理
    - 对快速连续点击进行去抖
    - 防止过多的请求发送
    - _Requirements: 5.3_

  - [ ] 10.3 清理代码和添加注释
    - 添加关键逻辑的注释
    - 移除调试日志
    - 统一代码风格
    - _Requirements: All_

- [ ] 11. 最终验证和文档更新
  - [ ] 11.1 运行完整测试套件
    - 运行所有单元测试
    - 运行所有属性测试
    - 验证测试覆盖率 ≥ 85%
    - _Requirements: All_

  - [ ] 11.2 更新相关文档
    - 更新组件使用说明
    - 更新事件通信文档
    - 添加故障排查指南
    - _Requirements: All_

  - [ ] 11.3 最终用户验收测试
    - 演示完整功能
    - 收集用户反馈
    - 确认所有需求已满足
    - _Requirements: All_

## Notes

- 标记 `*` 的任务为可选测试任务，可以跳过以加快 MVP 开发
- 每个任务都引用了具体的需求编号，确保可追溯性
- 检查点任务确保增量验证，及时发现问题
- 属性测试使用 fast-check 库，每个测试至少运行 100 次迭代
- 单元测试和属性测试是互补的，共同确保代码质量
