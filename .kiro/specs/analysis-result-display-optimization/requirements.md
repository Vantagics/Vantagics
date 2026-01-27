# Requirements Document

## Introduction

本文档定义了优化分析结果数据展示逻辑的需求。当前系统存在数据流复杂分散、数据格式不一致、状态管理混乱、渲染逻辑重复和数据丢失风险等问题。本优化旨在统一数据模型、简化数据流、规范化数据格式、集中状态管理和优化渲染逻辑。

## Glossary

- **Analysis_Result_Manager**: 分析结果管理器，负责统一管理所有分析结果数据的核心组件
- **Unified_Data_Model**: 统一数据模型，定义所有分析结果类型的标准数据结构
- **Data_Normalizer**: 数据规范化器，负责将不同格式的数据转换为统一格式
- **Event_Aggregator**: 事件聚合器，负责将多个事件合并为单一数据更新事件
- **Session_Data_Store**: 会话数据存储，按会话和消息ID组织的数据存储结构
- **Chart_Data**: 图表数据，包含ECharts配置、图片、表格、CSV等类型的可视化数据
- **Dashboard_State**: 仪表盘状态，包含当前显示的所有分析结果数据

## Requirements

### Requirement 1: 统一数据模型

**User Story:** As a developer, I want a unified data model for all analysis results, so that I can handle different data types consistently.

#### Acceptance Criteria

1. THE Unified_Data_Model SHALL define a standard wrapper structure containing type, data, metadata, and source fields
2. THE Unified_Data_Model SHALL support the following data types: echarts, image, table, csv, metric, insight
3. WHEN any analysis result is received, THE Data_Normalizer SHALL convert it to the Unified_Data_Model format
4. THE Unified_Data_Model SHALL include timestamp, messageId, and sessionId in metadata for traceability
5. THE Unified_Data_Model SHALL support batch data containing multiple items of different types

### Requirement 2: 简化事件数据流

**User Story:** As a developer, I want a simplified event system, so that I can reduce complexity and avoid race conditions.

#### Acceptance Criteria

1. THE Event_Aggregator SHALL consolidate multiple dashboard events into a single `analysis-result-update` event
2. WHEN the backend emits analysis results, THE Event_Aggregator SHALL batch all related data into one event payload
3. THE System SHALL deprecate the following events: `dashboard-update`, `dashboard-data-update`, `update-dashboard-insights`, `update-dashboard-metrics`, `metrics-extracted`
4. THE System SHALL use only three events for data flow: `analysis-result-update` (main data), `analysis-result-clear` (clear data), `analysis-result-loading` (loading state)
5. WHEN multiple data items are ready simultaneously, THE Event_Aggregator SHALL emit them in a single event to prevent race conditions

### Requirement 3: 集中状态管理

**User Story:** As a developer, I want centralized state management, so that I can avoid data inconsistency and simplify debugging.

#### Acceptance Criteria

1. THE Analysis_Result_Manager SHALL maintain a single source of truth for all analysis result data
2. THE Analysis_Result_Manager SHALL organize data by sessionId and messageId in a hierarchical structure
3. WHEN a session is switched, THE Analysis_Result_Manager SHALL load the corresponding session data without data loss
4. WHEN a message is selected, THE Analysis_Result_Manager SHALL retrieve all associated data (charts, metrics, insights, files)
5. THE Analysis_Result_Manager SHALL provide atomic update operations to prevent partial state updates
6. IF concurrent updates occur, THEN THE Analysis_Result_Manager SHALL queue updates and process them sequentially

### Requirement 4: 数据格式规范化

**User Story:** As a developer, I want consistent data formats, so that I can simplify rendering logic and reduce bugs.

#### Acceptance Criteria

1. WHEN ECharts data is received as a JSON string, THE Data_Normalizer SHALL parse and validate it before storage
2. WHEN image data is received, THE Data_Normalizer SHALL ensure it is in base64 data URL format with proper MIME type
3. WHEN table data is received, THE Data_Normalizer SHALL convert it to a standard array of objects format
4. WHEN CSV data is received, THE Data_Normalizer SHALL convert it to the same format as table data
5. THE Data_Normalizer SHALL validate all data against type-specific schemas before acceptance
6. IF data validation fails, THEN THE Data_Normalizer SHALL log the error and reject the invalid data

### Requirement 5: 优化渲染逻辑

**User Story:** As a developer, I want simplified rendering logic, so that I can reduce code duplication and improve maintainability.

#### Acceptance Criteria

1. THE DraggableDashboard SHALL receive pre-normalized data from Analysis_Result_Manager
2. THE DraggableDashboard SHALL NOT perform data extraction or transformation logic
3. WHEN checking data availability, THE DraggableDashboard SHALL use a single `hasData(type)` method from Analysis_Result_Manager
4. THE DraggableDashboard SHALL render each data type using dedicated render components without type-checking logic
5. WHEN data is updated, THE DraggableDashboard SHALL re-render only the affected components

### Requirement 6: 数据持久化和恢复

**User Story:** As a user, I want my analysis results to persist across sessions, so that I can continue my work without losing data.

#### Acceptance Criteria

1. WHEN analysis results are received, THE Session_Data_Store SHALL persist them to the backend immediately
2. WHEN a session is loaded, THE Session_Data_Store SHALL restore all associated analysis results
3. WHEN a message is clicked, THE Session_Data_Store SHALL load the complete data set for that message
4. THE Session_Data_Store SHALL maintain data integrity during concurrent read/write operations
5. IF data loading fails, THEN THE Session_Data_Store SHALL display an error message and retain any cached data
