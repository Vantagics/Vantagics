# Implementation Tasks

## Task 1: 创建统一数据模型和类型定义

- [x] 1.1 创建 `src/frontend/src/types/AnalysisResult.ts` 定义统一数据模型
  - 定义 `AnalysisResultItem` 接口
  - 定义 `AnalysisResultType` 类型
  - 定义 `ResultMetadata` 接口
  - 定义 `AnalysisResultBatch` 接口
  - 定义 `AnalysisResultState` 接口

- [x] 1.2 创建 `src/frontend/src/utils/DataNormalizer.ts` 数据规范化器
  - 实现 `normalizeECharts()` 方法
  - 实现 `normalizeImage()` 方法
  - 实现 `normalizeTable()` 方法
  - 实现 `normalizeMetric()` 方法
  - 实现 `normalizeInsight()` 方法
  - 实现 `normalize()` 统一入口方法

## Task 2: 实现 AnalysisResultManager 状态管理器

- [x] 2.1 创建 `src/frontend/src/managers/AnalysisResultManager.ts`
  - 实现单例模式
  - 实现 `updateResults()` 方法
  - 实现 `clearResults()` 方法
  - 实现 `getResults()` 方法
  - 实现 `getResultsByType()` 方法
  - 实现 `hasData()` 方法

- [x] 2.2 实现会话管理功能
  - 实现 `switchSession()` 方法
  - 实现 `getCurrentSession()` 方法
  - 实现 `selectMessage()` 方法

- [x] 2.3 实现状态订阅机制
  - 实现 `subscribe()` 方法
  - 实现 `notifySubscribers()` 方法
  - 实现加载状态管理 `setLoading()` / `isLoading()`

## Task 3: 后端事件聚合器实现

- [x] 3.1 创建 `src/event_aggregator.go` 事件聚合器
  - 实现 `EventAggregator` 结构体
  - 实现 `AddItem()` 方法
  - 实现 `FlushNow()` 方法
  - 实现定时聚合逻辑（50ms窗口）

- [x] 3.2 修改 `src/app.go` 集成事件聚合器
  - 添加 `eventAggregator` 字段到 App 结构体 ✓
  - 在 startup 中初始化 EventAggregator ✓
  - 替换 `dashboard-update` 事件为 `analysis-result-update` ✓
  - 替换 `dashboard-data-update` 事件 ✓
  - 移除 `update-dashboard-insights` 事件 ✓
  - 移除 `update-dashboard-metrics` 事件 ✓
  - 移除 `metrics-extracted` 事件 ✓
  - 替换 `clear-dashboard-data` 事件为 `analysis-result-clear` ✓

## Task 4: 前端事件监听重构

- [x] 4.1 修改 `src/frontend/src/App.tsx` 事件监听
  - 创建 `AnalysisResultBridge.ts` 桥接层 ✓
  - 添加 `analysis-result-update` 事件监听 ✓
  - 添加 `analysis-result-clear` 事件监听 ✓
  - 添加 `analysis-result-loading` 事件监听 ✓
  - 添加 `analysis-result-error` 事件监听 ✓
  - 在 App.tsx 中初始化 Bridge ✓

- [x] 4.2 简化 App.tsx 状态管理 ✓
  - 移除 `activeChart` 状态 ✓
  - 移除 `sessionCharts` 状态 ✓
  - 移除 `sessionInsights` 状态 ✓
  - 移除 `sessionMetrics` 状态 ✓
  - 移除 `dashboardData` 状态 ✓
  - 移除旧的事件监听器 (`dashboard-update`, `dashboard-data-update`, `clear-dashboard-data`, `update-dashboard-insights`, `update-dashboard-metrics`, `metrics-extracted`) ✓
  - 集成 `AnalysisResultManager` ✓

## Task 5: DraggableDashboard 组件重构

- [x] 5.1 重构数据获取逻辑
  - 创建 `useDashboardData` Hook 整合新旧数据源 ✓
  - 创建 `useAnalysisResults` Hook 提供响应式数据访问 ✓
  - 从 `AnalysisResultManager` 获取数据（通过Hook）✓

- [x] 5.2 简化渲染逻辑 ✓
  - 使用 `useDashboardData` Hook 获取数据 ✓
  - 创建兼容变量 `data` 和 `activeChart` 从新系统获取数据 ✓
  - 移除对旧 props 的依赖 ✓

## Task 6: 数据持久化集成

- [x] 6.1 修改后端数据存储
  - 更新 `ChatMessage` 结构支持新格式 ✓
  - 实现 `AnalysisResultItem` 的序列化/反序列化 ✓
  - 确保数据完整性 ✓

- [x] 6.2 实现数据恢复逻辑
  - 会话切换时加载历史数据 ✓
  - 消息点击时恢复关联数据 ✓
  - 处理数据加载失败情况 ✓

## Task 7: 测试和验证

- [x] 7.1 单元测试
  - 测试 `DataNormalizer` 各类型规范化 ✓ (15 tests passed)
  - 测试 `AnalysisResultManager` 状态管理 ✓ (10 tests passed)
  - 测试事件聚合逻辑 ✓ (5 Go tests passed)

- [x] 7.2 集成测试
  - 测试完整数据流（后端到前端）✓
  - 测试会话切换数据恢复 ✓
  - 测试并发更新处理 ✓

## 完成总结

所有任务已完成！实现了：
1. 统一数据模型 (`AnalysisResult.ts`)
2. 数据规范化器 (`DataNormalizer.ts`)
3. 状态管理器 (`AnalysisResultManager.ts`)
4. 后端事件聚合器 (`event_aggregator.go`)
5. 事件桥接层 (`AnalysisResultBridge.ts`)
6. React Hooks (`useAnalysisResults.ts`, `useDashboardData.ts`)
7. 数据持久化 (ChatMessage.AnalysisResults)
8. 数据恢复逻辑 (analysis-result-restore 事件)
