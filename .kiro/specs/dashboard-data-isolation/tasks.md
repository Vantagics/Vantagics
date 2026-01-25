# 实现计划: 仪表盘数据隔离

## 概述

本实现计划将仪表盘数据隔离功能分解为可执行的编码任务。核心目标是确保 `useDashboardData` Hook 能够正确响应 `AnalysisResultManager` 的状态变更，在适当时机清除或重新加载 `dataSourceStatistics`。

## 任务

- [x] 1. 扩展 AnalysisResultManager 事件系统
  - [x] 1.1 添加 analysis-started 事件触发
    - 在 `setLoading(true)` 时触发 `analysis-started` 事件
    - 事件携带 sessionId、messageId、requestId
    - _Requirements: 1.1, 1.2_
  
  - [x] 1.2 添加 session-switched 事件触发
    - 在 `switchSession` 方法中触发事件
    - 事件携带 fromSessionId 和 toSessionId
    - _Requirements: 4.1, 4.2_
  
  - [x] 1.3 添加 message-selected 事件触发
    - 在 `selectMessage` 方法中触发事件
    - 事件携带 sessionId、fromMessageId、toMessageId
    - _Requirements: 4.3, 4.4_

- [x] 2. 改进 useDashboardData Hook
  - [x] 2.1 添加 dataSourceStatistics 响应式管理
    - 订阅 AnalysisResultManager 状态变更
    - 在分析开始时清除 dataSourceStatistics
    - 在会话切换时重置 dataSourceStatistics
    - _Requirements: 1.2, 4.2, 5.1_
  
  - [x] 2.2 实现 hasAnyAnalysisResults 边界检查优化
    - 确保检查覆盖所有数据类型
    - 处理空数组和 null 值边界情况
    - _Requirements: 3.4_
  
  - [x] 2.3 添加数据源统计条件加载逻辑
    - 只在无分析结果时加载数据源统计
    - 在分析结果清除后重新加载
    - _Requirements: 3.1, 3.2, 3.3_
  
  - [x] 2.4 编写属性测试：数据清除一致性
    - **Property 1: 数据清除一致性**
    - **Validates: Requirements 1.1, 2.1, 4.1, 4.3**
  
  - [x] 2.5 编写属性测试：数据源统计清除同步
    - **Property 2: 数据源统计清除同步**
    - **Validates: Requirements 1.2, 4.2, 5.1**

- [x] 3. 检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 4. 实现数据隔离逻辑
  - [x] 4.1 改进 selectMessage 方法的数据清除逻辑
    - 切换消息时清除当前消息的分析结果
    - 保留新消息的已有数据（如果有）
    - _Requirements: 4.3, 4.4_
  
  - [x] 4.2 改进历史数据恢复逻辑
    - 在 analysis-result-restore 事件处理中先清除旧数据
    - 确保只显示恢复的数据
    - _Requirements: 2.1, 2.2_
  
  - [x] 4.3 添加历史请求无结果时的空状态处理
    - 检测历史请求是否有关联结果
    - 无结果时显示空状态而非数据源统计
    - _Requirements: 2.4_
  
  - [x] 4.4 编写属性测试：数据隔离性
    - **Property 3: 数据隔离性**
    - **Validates: Requirements 1.4, 2.2, 4.4**
  
  - [x] 4.5 编写属性测试：数据源统计显示互斥性
    - **Property 4: 数据源统计显示互斥性**
    - **Validates: Requirements 2.3, 3.1, 3.2, 3.3**

- [x] 5. 检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 6. 实现状态同步机制
  - [x] 6.1 添加 useAnalysisResults 状态变更监听
    - 在 useDashboardData 中监听 analysisResults 变化
    - 状态变更时重新评估 hasAnyAnalysisResults
    - _Requirements: 5.3, 5.5_
  
  - [x] 6.2 实现加载状态同步
    - 确保 isLoading 状态正确传递
    - 分析开始时设置 isLoading 为 true
    - 数据到达或错误时设置为 false
    - _Requirements: 1.3_
  
  - [x] 6.3 编写属性测试：hasAnyAnalysisResults 边界正确性
    - **Property 5: hasAnyAnalysisResults 边界正确性**
    - **Validates: Requirements 3.4**
  
  - [x] 6.4 编写属性测试：状态同步响应性
    - **Property 6: 状态同步响应性**
    - **Validates: Requirements 5.2, 5.3, 5.5**
  
  - [x] 6.5 编写属性测试：加载状态一致性
    - **Property 7: 加载状态一致性**
    - **Validates: Requirements 1.3**

- [x] 7. 最终检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

## 备注

- 所有任务都是必需的，包括属性测试
- 每个任务都引用了具体的需求以便追溯
- 检查点确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证特定示例和边界情况
