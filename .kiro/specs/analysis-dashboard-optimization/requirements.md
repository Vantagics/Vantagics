# Requirements Document

## Introduction

本文档定义了分析逻辑与仪表盘数据显示优化功能的需求。该功能旨在解决三个核心问题：
1. Agent分析后应优先生成可视化结果（ECharts图表、数据表格、图片等）
2. 确保所有分析结果都能正确传递到前端仪表盘并显示
3. 仪表盘各控件宽度应占满仪表盘区域宽度（100%宽度布局）

## Glossary

- **Agent**: 执行数据分析的AI代理，基于EinoService实现
- **Dashboard**: 前端仪表盘组件（DraggableDashboard），用于显示分析结果
- **ECharts**: 一种交互式图表库，用于生成可视化图表
- **EventAggregator**: 后端事件聚合器，负责收集分析结果并发送到前端
- **AnalysisResultManager**: 前端分析结果管理器，作为所有分析结果数据的单一数据源
- **Visualization_Result**: 可视化分析结果，包括ECharts图表、图片、表格等
- **Analysis_Prompt**: 发送给LLM的分析提示词，指导Agent生成分析代码
- **Layout_Item**: 仪表盘布局项，定义控件的位置和尺寸

## Requirements

### Requirement 1: 优先生成可视化结果

**User Story:** As a user, I want the Agent to prioritize generating visual results (charts, tables, images), so that I can better understand the analysis outcomes.

#### Acceptance Criteria

1. WHEN a user submits a data analysis request, THE Analysis_Prompt SHALL include explicit instructions to generate visualization results
2. WHEN the analysis involves numerical data or trends, THE Agent SHALL generate at least one ECharts chart or image
3. WHEN the analysis involves tabular data, THE Agent SHALL output the data in json:table format
4. IF the analysis request is ambiguous about visualization needs, THEN THE Agent SHALL default to generating a chart when data permits
5. WHEN generating Python code for analysis, THE Analysis_Prompt SHALL include mandatory chart generation instructions with plt.savefig()
6. THE Analysis_Prompt SHALL specify that charts must be saved to FILES_DIR using os.path.join()

### Requirement 2: 增强提示词中的可视化指令

**User Story:** As a developer, I want the analysis prompt to strongly emphasize visualization requirements, so that the LLM consistently generates visual outputs.

#### Acceptance Criteria

1. THE Analysis_Prompt SHALL include a dedicated "分析要求" section with visualization instructions marked as mandatory (⭐⭐⭐)
2. WHEN classification hints indicate NeedsVisualization=true, THE Analysis_Prompt SHALL include chart type recommendations
3. THE Analysis_Prompt SHALL include code examples showing correct chart saving patterns
4. THE Analysis_Prompt SHALL include a "重要警告" section about file saving requirements
5. WHEN no classification hints are available, THE Analysis_Prompt SHALL still encourage visualization generation

### Requirement 3: 数据传递完整性

**User Story:** As a user, I want all analysis results to be correctly displayed on the dashboard, so that I don't miss any important findings.

#### Acceptance Criteria

1. WHEN the Agent generates an ECharts chart, THE EventAggregator SHALL capture and emit the chart data to the frontend
2. WHEN the Agent generates an image file, THE EventAggregator SHALL capture and emit the image data to the frontend
3. WHEN the Agent generates table data, THE EventAggregator SHALL capture and emit the table data to the frontend
4. WHEN the EventAggregator emits data, THE AnalysisResultManager SHALL receive and store the data with correct sessionId and messageId
5. IF sessionId or messageId is empty, THEN THE EventAggregator SHALL log a warning but continue processing (graceful degradation)
6. WHEN data is received by AnalysisResultManager, THE Dashboard SHALL update to display the new data within 100ms

### Requirement 4: 数据解析可靠性

**User Story:** As a developer, I want the response parsing to reliably extract all visualization data, so that no generated content is lost.

#### Acceptance Criteria

1. WHEN parsing LLM response, THE Parser SHALL extract all json:echarts code blocks
2. WHEN parsing LLM response, THE Parser SHALL extract all base64 image data
3. WHEN parsing LLM response, THE Parser SHALL extract all json:table code blocks
4. WHEN parsing LLM response, THE Parser SHALL detect and load chart files saved by Python execution
5. IF JSON parsing fails, THEN THE Parser SHALL log the error with the first 500 characters of the problematic content
6. WHEN multiple charts/images/tables exist in one response, THE Parser SHALL collect all of them (not just the first)

### Requirement 5: 仪表盘控件全宽布局

**User Story:** As a user, I want dashboard components to use the full width of the dashboard area, so that I can see more data at once.

#### Acceptance Criteria

1. THE Dashboard default layout SHALL set all Layout_Item widths to 100% (w: 100)
2. WHEN a new layout is created, THE Layout_Item width SHALL default to 100
3. WHEN loading a saved layout, THE Dashboard SHALL preserve the saved width values
4. WHEN in edit mode, THE Dashboard SHALL allow users to adjust Layout_Item widths
5. WHEN displaying charts, THE Chart component SHALL expand to fill its container width
6. WHEN displaying tables, THE Table component SHALL expand to fill its container width

### Requirement 6: 布局持久化与恢复

**User Story:** As a user, I want my dashboard layout preferences to be saved and restored, so that I don't have to reconfigure the layout each time.

#### Acceptance Criteria

1. WHEN a user saves a layout, THE LayoutService SHALL persist the layout configuration to the database
2. WHEN the Dashboard loads, THE LayoutService SHALL restore the previously saved layout
3. IF no saved layout exists, THEN THE Dashboard SHALL use the default full-width layout
4. WHEN layout items are resized, THE Dashboard SHALL update the layout configuration
5. THE Layout configuration SHALL include x, y, w, h values for each Layout_Item

### Requirement 7: 实时数据更新

**User Story:** As a user, I want to see analysis results appear on the dashboard in real-time as they are generated, so that I can monitor the analysis progress.

#### Acceptance Criteria

1. WHEN the EventAggregator receives new data, THE Dashboard SHALL update within 50ms (flush delay)
2. WHEN analysis is in progress, THE Dashboard SHALL show a loading indicator
3. WHEN analysis completes, THE Dashboard SHALL display all results and clear the loading indicator
4. WHEN analysis fails, THE Dashboard SHALL display an error message with recovery suggestions
5. WHEN switching between sessions, THE Dashboard SHALL clear old data and display data for the new session

### Requirement 8: 数据类型支持

**User Story:** As a user, I want the dashboard to support various data types, so that I can view different kinds of analysis results.

#### Acceptance Criteria

1. THE Dashboard SHALL support displaying ECharts interactive charts
2. THE Dashboard SHALL support displaying PNG/JPEG images
3. THE Dashboard SHALL support displaying data tables with sorting and pagination
4. THE Dashboard SHALL support displaying metrics (title, value, change)
5. THE Dashboard SHALL support displaying insights (text, icon)
6. THE Dashboard SHALL support displaying file download links
