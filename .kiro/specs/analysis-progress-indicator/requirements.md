# Requirements Document

## Introduction

本功能在智能分析仪表盘标题右侧添加一个分析进行中指示器。当有分析任务正在执行时，显示一个带有旋转动画的加载指示器和"分析进行中"提示文字。当分析完成或用户取消分析时，该指示器自动消失。

## Glossary

- **Analysis_Progress_Indicator**: 分析进行中指示器组件，用于显示当前分析任务的执行状态
- **Dashboard_Header**: 仪表盘标题栏，位于 DraggableDashboard 组件顶部
- **Analysis_Loading_State**: 分析加载状态，由 `isAnalysisLoading` 属性控制
- **Spinner**: 旋转加载动画，用于视觉反馈

## Requirements

### Requirement 1: 分析进行中指示器显示

**User Story:** As a user, I want to see a visual indicator when analysis is in progress, so that I know the system is working on my request.

#### Acceptance Criteria

1. WHEN `isAnalysisLoading` is true, THE Analysis_Progress_Indicator SHALL display a spinning animation and the text "分析进行中"
2. WHEN `isAnalysisLoading` is false, THE Analysis_Progress_Indicator SHALL be hidden
3. THE Analysis_Progress_Indicator SHALL be positioned to the right of the dashboard title "智能分析仪表盘"

### Requirement 2: 指示器视觉设计

**User Story:** As a user, I want the progress indicator to have a clear and non-intrusive design, so that it provides feedback without distracting from the dashboard content.

#### Acceptance Criteria

1. THE Spinner SHALL use a circular rotating animation with smooth transition
2. THE Analysis_Progress_Indicator SHALL use a subtle color scheme consistent with the dashboard design (blue tones)
3. THE Analysis_Progress_Indicator SHALL have appropriate spacing from the dashboard title
4. THE Analysis_Progress_Indicator text and spinner SHALL be vertically aligned

### Requirement 3: 国际化支持

**User Story:** As a user, I want the progress indicator text to be displayed in my preferred language, so that I can understand the status message.

#### Acceptance Criteria

1. THE Analysis_Progress_Indicator text SHALL support both English ("Analysis in progress") and Chinese ("分析进行中") languages
2. WHEN the system language changes, THE Analysis_Progress_Indicator text SHALL update accordingly

### Requirement 4: 状态同步

**User Story:** As a user, I want the indicator to accurately reflect the analysis state, so that I have reliable feedback about the system status.

#### Acceptance Criteria

1. WHEN an analysis task starts, THE Analysis_Progress_Indicator SHALL appear immediately
2. WHEN an analysis task completes successfully, THE Analysis_Progress_Indicator SHALL disappear immediately
3. WHEN a user cancels an analysis task, THE Analysis_Progress_Indicator SHALL disappear immediately
4. WHEN an analysis task fails with an error, THE Analysis_Progress_Indicator SHALL disappear immediately
