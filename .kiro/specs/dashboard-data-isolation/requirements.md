# 需求文档

## 简介

本文档定义了仪表盘数据隔离功能的需求。该功能旨在解决当显示分析结果时（分析完成自动加载或点击历史分析请求加载），仪表盘中之前的数据（特别是数据源相关数据）没有被清除，导致新旧数据混合显示的问题。

## 术语表

- **Dashboard（仪表盘）**: 显示分析结果的主界面组件，包含指标卡片、图表、表格、洞察等
- **AnalysisResultManager**: 集中管理所有分析结果数据的单例管理器
- **DataSourceStatistics（数据源统计）**: 系统中所有数据源的统计信息，在无分析结果时显示
- **Session（会话）**: 用户与系统的一次交互会话，包含多个消息
- **Message（消息）**: 会话中的单个分析请求及其结果
- **useDashboardData**: 为仪表盘提供数据访问的 React Hook

## 需求

### 需求 1：新分析开始时清除旧数据

**用户故事：** 作为用户，我希望当开始新的分析时，仪表盘能够清除所有旧数据，以便我能看到干净的分析结果而不是混合数据。

#### 验收标准

1. WHEN 用户发起新的分析请求 THEN Dashboard_System SHALL 立即清除当前显示的所有分析结果数据
2. WHEN 新分析开始 THEN Dashboard_System SHALL 清除 dataSourceStatistics 状态
3. WHEN 新分析开始 THEN Dashboard_System SHALL 显示加载状态直到新数据到达
4. WHEN 新分析结果到达 THEN Dashboard_System SHALL 仅显示该分析的结果数据

### 需求 2：历史分析请求数据隔离

**用户故事：** 作为用户，我希望当点击历史分析请求时，仪表盘只显示该请求对应的分析结果，以便我能准确查看历史数据。

#### 验收标准

1. WHEN 用户点击历史分析请求 THEN Dashboard_System SHALL 清除当前显示的所有数据
2. WHEN 用户点击历史分析请求 THEN Dashboard_System SHALL 仅加载并显示该请求对应的分析结果
3. WHEN 历史分析结果加载完成 THEN Dashboard_System SHALL 不显示任何数据源统计信息
4. IF 历史分析请求没有关联的分析结果 THEN Dashboard_System SHALL 显示空状态而非数据源统计

### 需求 3：数据源统计信息显示控制

**用户故事：** 作为用户，我希望数据源统计信息只在没有活跃分析结果时显示，以避免与分析结果混淆。

#### 验收标准

1. WHILE 存在任何分析结果数据 THEN Dashboard_System SHALL 隐藏数据源统计信息
2. WHEN 所有分析结果被清除 THEN Dashboard_System SHALL 显示数据源统计信息
3. WHEN 切换到新会话且无分析结果 THEN Dashboard_System SHALL 显示数据源统计信息
4. THE Dashboard_System SHALL 确保 hasAnyAnalysisResults 检查在所有边界情况下正确工作

### 需求 4：会话和消息切换时的数据隔离

**用户故事：** 作为用户，我希望切换会话或消息时，数据能够正确隔离，以便每个会话/消息的数据独立显示。

#### 验收标准

1. WHEN 用户切换到不同会话 THEN Dashboard_System SHALL 清除当前会话的所有显示数据
2. WHEN 用户切换到不同会话 THEN Dashboard_System SHALL 重置 dataSourceStatistics 状态
3. WHEN 用户在同一会话内切换消息 THEN Dashboard_System SHALL 清除当前消息的分析结果
4. WHEN 用户在同一会话内切换消息 THEN Dashboard_System SHALL 仅显示新选中消息的分析结果
5. IF 切换目标没有分析结果 THEN Dashboard_System SHALL 显示空状态或数据源统计（根据上下文）

### 需求 5：数据状态同步

**用户故事：** 作为用户，我希望仪表盘的数据状态与分析结果管理器保持同步，以确保显示的数据始终是最新和正确的。

#### 验收标准

1. WHEN AnalysisResultManager 清除数据 THEN useDashboardData SHALL 同步清除 dataSourceStatistics
2. WHEN AnalysisResultManager 切换会话 THEN useDashboardData SHALL 重新加载对应的数据源统计
3. WHEN AnalysisResultManager 选择新消息 THEN useDashboardData SHALL 更新显示数据
4. THE useDashboardData SHALL 订阅 AnalysisResultManager 的状态变更事件
5. WHEN 状态变更事件触发 THEN useDashboardData SHALL 重新评估 hasAnyAnalysisResults 条件
