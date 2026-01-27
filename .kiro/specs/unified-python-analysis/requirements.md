# Requirements Document

## Introduction

本文档定义了统一Python分析流程优化功能的需求。当前系统在处理用户分析请求时需要多次LLM调用（请求分类→Schema获取→SQL生成→数据处理/可视化），每次调用都有网络延迟和处理时间，累积起来严重影响用户体验。

本优化的核心目标是将SQL生成、数据处理、图表生成合并成一次LLM调用，让LLM直接生成完整的Python代码，包含数据库连接、SQL执行、数据处理和可视化，一次执行得到最终结果。

## Glossary

- **Unified_Python_Generator**: 统一Python代码生成器，负责一次性生成包含完整分析流程的Python代码
- **Analysis_Request**: 用户的分析请求，包含查询意图和数据源信息
- **Python_Analysis_Code**: 生成的完整Python代码，包含sqlite3连接、SQL执行、pandas处理和matplotlib可视化
- **Code_Template**: 代码模板，提供标准化的代码结构和错误处理
- **Execution_Result**: Python代码执行结果，包含文本输出、图表文件和数据表格
- **Schema_Context**: 数据源的Schema上下文，包含表结构、字段类型和示例数据
- **Single_Pass_Analysis**: 单次LLM调用完成的完整分析流程

## Requirements

### Requirement 1: 统一代码生成

**User Story:** As a user, I want the system to generate complete analysis code in a single LLM call, so that I get faster responses without multiple round-trips.

#### Acceptance Criteria

1. WHEN a user submits a data analysis request, THE Unified_Python_Generator SHALL generate complete Python code in a single LLM call
2. THE generated Python_Analysis_Code SHALL include: sqlite3 database connection, SQL query execution, pandas data processing, and matplotlib/seaborn visualization (when applicable)
3. WHEN generating code, THE Unified_Python_Generator SHALL use the provided Schema_Context to construct accurate SQL queries
4. THE generated code SHALL be self-contained and executable without additional LLM interactions
5. WHEN the request requires visualization, THE generated code SHALL save charts to the session directory with appropriate filenames

### Requirement 2: 代码模板系统

**User Story:** As a developer, I want standardized code templates, so that generated code is consistent, safe, and handles errors properly.

#### Acceptance Criteria

1. THE Code_Template SHALL provide a standard structure with: imports, database connection, query execution, data processing, visualization, and result output sections
2. THE Code_Template SHALL include proper error handling with try-except blocks for database operations, data processing, and file I/O
3. THE Code_Template SHALL use parameterized database paths that are injected at runtime
4. WHEN database operations fail, THE generated code SHALL output clear error messages in Chinese
5. THE Code_Template SHALL include proper resource cleanup (closing database connections) in finally blocks

### Requirement 3: Schema上下文优化

**User Story:** As a user, I want the system to efficiently use schema information, so that generated SQL queries are accurate and efficient.

#### Acceptance Criteria

1. WHEN preparing Schema_Context for code generation, THE system SHALL include table names, column names, column types, and sample data
2. THE Schema_Context SHALL be formatted in a concise, LLM-friendly format that minimizes token usage
3. WHEN multiple tables are relevant, THE Schema_Context SHALL include relationship hints (foreign keys, common columns)
4. THE system SHALL cache Schema_Context to avoid redundant database queries within the same session
5. IF Schema_Context exceeds token limits, THEN THE system SHALL prioritize tables most relevant to the user's request

### Requirement 4: 执行结果处理

**User Story:** As a user, I want to see analysis results in a clear and organized format, so that I can understand the insights quickly.

#### Acceptance Criteria

1. WHEN Python code execution completes successfully, THE Execution_Result SHALL include: text summary, data tables (if applicable), and chart file paths (if applicable)
2. THE system SHALL parse Python stdout to extract structured results (tables, summaries, file paths)
3. WHEN charts are generated, THE system SHALL emit file-saved events with filename, type, and size
4. IF execution fails, THEN THE system SHALL return the error message and suggest potential fixes
5. THE system SHALL support multiple output formats: text, markdown table, JSON data, and image paths

### Requirement 5: 请求路由优化

**User Story:** As a user, I want the system to choose the optimal execution path, so that simple requests are handled quickly and complex requests get full analysis.

#### Acceptance Criteria

1. WHEN a request is classified as requiring data analysis with visualization, THE system SHALL use the Unified_Python_Generator path
2. WHEN a request is a simple data query without visualization, THE system SHALL use direct SQL execution for faster response
3. WHEN a request is a consultation/suggestion type, THE system SHALL NOT use the Unified_Python_Generator path
4. THE system SHALL detect request complexity and choose between: quick path (no LLM), single-pass analysis (1 LLM call), or multi-step analysis (multiple LLM calls)
5. WHEN using Unified_Python_Generator, THE system SHALL complete the analysis with at most 2 tool calls (schema fetch + python execution)

### Requirement 6: 代码安全性

**User Story:** As a system administrator, I want generated code to be safe and sandboxed, so that malicious or erroneous code cannot harm the system.

#### Acceptance Criteria

1. THE generated Python_Analysis_Code SHALL only access the specified database file and session directory
2. THE Code_Template SHALL NOT include any system commands, file deletions, or network operations outside of the analysis scope
3. WHEN generating code, THE Unified_Python_Generator SHALL validate that SQL queries are read-only (SELECT statements only)
4. THE system SHALL enforce execution timeout limits to prevent infinite loops or resource exhaustion
5. IF generated code attempts unsafe operations, THEN THE Python executor SHALL block execution and return an error

### Requirement 7: 提示词工程优化

**User Story:** As a developer, I want optimized prompts for code generation, so that the LLM produces high-quality, executable code consistently.

#### Acceptance Criteria

1. THE prompt for Unified_Python_Generator SHALL include: user request, schema context, code template structure, and output format requirements
2. THE prompt SHALL specify that generated code must be complete and executable without modifications
3. THE prompt SHALL include examples of well-formed analysis code for common scenarios
4. WHEN the user request is ambiguous, THE prompt SHALL instruct the LLM to make reasonable assumptions and document them in code comments
5. THE prompt SHALL specify Chinese language for all user-facing output (print statements, chart labels, error messages)

### Requirement 8: 性能监控

**User Story:** As a developer, I want to monitor the performance of the unified analysis path, so that I can measure improvements and identify bottlenecks.

#### Acceptance Criteria

1. THE system SHALL log timing metrics for: schema fetch, LLM code generation, and Python execution
2. THE system SHALL track the number of LLM calls per analysis request
3. WHEN analysis completes, THE system SHALL log total duration and compare against the baseline multi-call approach
4. THE system SHALL emit progress updates during long-running analyses
5. IF analysis takes longer than expected, THEN THE system SHALL log a warning with diagnostic information

