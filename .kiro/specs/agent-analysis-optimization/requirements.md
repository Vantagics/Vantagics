# Requirements Document

## Introduction

本文档定义了Agent分析流程优化功能的需求。基于对现有agent执行轨迹和代码的分析，发现当前流程存在Schema获取效率低、执行计划与实际执行不匹配、请求类型识别不够智能、缺少分步数据处理机制等问题。本优化旨在提高agent分析效率，减少不必要的工具调用，并支持更智能的请求分类和分步执行机制。

## Glossary

- **Analysis_Planner**: 分析规划器，负责分析用户请求并创建执行计划
- **Request_Classifier**: 请求分类器，负责识别用户请求的类型
- **Schema_Manager**: Schema管理器，负责智能获取和缓存数据库Schema信息
- **Step_Executor**: 分步执行器，负责按步骤执行分析任务并支持中间结果反馈
- **Execution_Validator**: 执行验证器，负责验证执行计划的合理性和执行过程的一致性
- **Quick_Path**: 快速路径，指不需要数据源即可完成的简单请求
- **Consultation_Request**: 咨询建议类请求，只需要基于Schema生成建议，不需要执行SQL
- **Multi_Step_Analysis**: 多步骤分析，需要分步执行并根据中间结果调整后续步骤的复杂分析

## Requirements

### Requirement 1: 增强请求分类

**User Story:** As a user, I want the system to intelligently classify my requests, so that it can choose the most efficient execution path.

#### Acceptance Criteria

1. WHEN a user submits a request, THE Request_Classifier SHALL classify it into one of the following types: `trivial`, `simple`, `data_query`, `visualization`, `calculation`, `web_search`, `consultation`, or `multi_step_analysis`
2. WHEN a request is classified as `consultation` (e.g., "对本数据源提出一些分析建议"), THE Analysis_Planner SHALL NOT include `execute_sql` in the execution plan
3. WHEN a request is classified as `multi_step_analysis`, THE Analysis_Planner SHALL create a plan with intermediate checkpoints for result validation
4. WHEN a request matches quick path patterns (time queries, simple calculations, unit conversions), THE Request_Classifier SHALL classify it as `trivial` or `simple` and bypass LLM planning
5. IF a request contains keywords like "建议", "分析方向", "可以做什么分析", THEN THE Request_Classifier SHALL classify it as `consultation`

### Requirement 2: 智能Schema获取

**User Story:** As a user, I want the system to fetch only the necessary schema information, so that analysis can be completed faster with fewer tool calls.

#### Acceptance Criteria

1. WHEN a request is classified as `consultation`, THE Schema_Manager SHALL only fetch table names and basic descriptions without detailed column information
2. WHEN a request is classified as `data_query` or `visualization`, THE Schema_Manager SHALL fetch complete schema including column names, types, and sample data
3. WHEN schema information has been cached for a data source, THE Schema_Manager SHALL return cached schema instead of making a new tool call
4. THE Schema_Manager SHALL provide a `getSchemaLevel` method that returns `basic` or `detailed` based on request type
5. WHEN fetching detailed schema, THE Schema_Manager SHALL fetch all relevant tables in a single call using the `table_names` parameter

### Requirement 3: 分步执行机制

**User Story:** As a user, I want complex analyses to be executed step by step with intermediate feedback, so that I can see progress and the system can adjust based on results.

#### Acceptance Criteria

1. WHEN a request is classified as `multi_step_analysis`, THE Step_Executor SHALL execute steps sequentially with intermediate result validation
2. WHEN a step completes, THE Step_Executor SHALL evaluate the result and determine if the next step should proceed, be modified, or be skipped
3. THE Step_Executor SHALL emit progress updates after each step completion
4. WHEN an intermediate step fails, THE Step_Executor SHALL attempt to recover by adjusting subsequent steps
5. THE Step_Executor SHALL support a maximum of 3 retry attempts per step before marking the step as failed

### Requirement 4: 执行计划验证

**User Story:** As a developer, I want the system to validate execution plans and track deviations, so that I can improve the planning accuracy over time.

#### Acceptance Criteria

1. WHEN an execution plan is created, THE Execution_Validator SHALL validate that the plan is consistent with the request classification
2. WHEN a `consultation` request has `execute_sql` in its plan, THE Execution_Validator SHALL flag this as an inconsistency and remove the SQL step
3. WHILE executing a plan, THE Execution_Validator SHALL track actual tool calls and compare them against planned steps
4. WHEN execution completes, THE Execution_Validator SHALL record the deviation metrics (planned vs actual calls, step order changes)
5. THE Execution_Validator SHALL log warnings when actual execution deviates more than 50% from the plan

### Requirement 5: 咨询建议类请求优化

**User Story:** As a user, I want consultation requests to be handled efficiently without unnecessary SQL execution, so that I get quick and relevant suggestions.

#### Acceptance Criteria

1. WHEN a `consultation` request is received, THE Analysis_Planner SHALL create a plan with only `get_data_source_context` (basic level) followed by direct response generation
2. THE Analysis_Planner SHALL generate analysis suggestions based on table names, relationships, and data source summary without executing any SQL
3. WHEN generating suggestions for a `consultation` request, THE system SHALL include: potential analysis dimensions, recommended visualizations, and example queries (as text, not executed)
4. THE system SHALL complete `consultation` requests with at most 1 tool call (basic schema fetch)

### Requirement 6: 执行计划格式优化

**User Story:** As a developer, I want execution plans to be more precise and actionable, so that the agent follows them accurately.

#### Acceptance Criteria

1. WHEN creating an execution plan, THE Analysis_Planner SHALL use exact tool names (`get_data_source_context`, `execute_sql`, `python_executor`) instead of generic names (`get_schema`, `sql_executor`)
2. WHEN a plan step specifies `get_data_source_context`, THE plan SHALL include whether basic or detailed schema is needed
3. WHEN a plan step specifies `execute_sql`, THE plan SHALL include the expected query type (aggregation, join, filter, etc.)
4. THE Analysis_Planner SHALL include estimated duration for each step based on historical data

### Requirement 7: Schema缓存机制

**User Story:** As a user, I want schema information to be cached across requests in the same session, so that repeated analyses don't require redundant schema fetches.

#### Acceptance Criteria

1. WHEN schema is fetched for a data source, THE Schema_Manager SHALL cache it with a TTL of 30 minutes
2. WHEN a cached schema exists and is not expired, THE Schema_Manager SHALL return the cached version with a "[Using cached schema]" indicator
3. WHEN the data source structure changes (detected via metadata), THE Schema_Manager SHALL invalidate the cache
4. THE Schema_Manager SHALL support both basic and detailed schema caching separately
5. WHEN cache is hit, THE system SHALL log the cache hit for monitoring purposes
