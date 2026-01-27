# Requirements Document

## Introduction

本文档定义了多会话分析状态指示器功能的需求。该功能旨在为用户提供清晰的视觉反馈，显示多个并发分析会话的处理状态。系统需要支持多个分析会话同时进行，每个会话有独立的状态管理器，并在多个位置（聊天区域、历史会话列表、仪表盘标题）显示相应的状态指示。

## Glossary

- **Session_Status_Manager**: 会话状态管理器，负责管理单个分析会话的状态，包括加载状态、进度信息和错误状态
- **Loading_State_Manager**: 全局加载状态管理器，统一管理所有分析会话的加载状态，支持多会话并发
- **Analysis_Session**: 分析会话，用户与 AI 进行数据分析交互的独立会话单元
- **Progress_Indicator**: 进度指示器，显示当前分析任务的处理进度和状态
- **Spinner**: 转圈动画，用于指示正在进行中的操作
- **Chat_Sidebar**: 聊天侧边栏组件，包含历史会话列表和当前会话的消息区域
- **Dashboard_Header**: 仪表盘标题区域，显示当前分析状态的汇总信息
- **Thread_ID**: 会话唯一标识符，用于区分不同的分析会话

## Requirements

### Requirement 1: 会话内分析状态显示

**User Story:** As a user, I want to see the analysis status within each chat session before the AI responds, so that I know the AI is processing my request.

#### Acceptance Criteria

1. WHEN an analysis request is sent in a session THEN THE Progress_Indicator SHALL display a loading spinner with status message in the chat area before the AI response
2. WHEN the analysis is in progress THEN THE Progress_Indicator SHALL show the current processing stage (e.g., "正在分析数据...", "正在生成图表...")
3. WHEN the analysis completes THEN THE Progress_Indicator SHALL be replaced by the AI's response message
4. IF an analysis error occurs THEN THE Progress_Indicator SHALL display an error state with appropriate message
5. WHEN multiple sessions have concurrent analyses THEN each session SHALL display its own independent Progress_Indicator

### Requirement 2: 历史会话列表状态指示

**User Story:** As a user, I want to see which sessions have ongoing analyses in the history list, so that I can quickly identify active sessions.

#### Acceptance Criteria

1. WHEN a session has an ongoing analysis THEN THE Chat_Sidebar SHALL display a spinning indicator before the session title in the history list
2. WHEN the analysis completes THEN THE Spinner SHALL be removed from the session list item
3. WHEN multiple sessions have concurrent analyses THEN each active session SHALL display its own Spinner independently
4. WHILE a session is loading THEN THE Spinner SHALL animate continuously until the analysis completes or fails

### Requirement 3: 仪表盘全局状态汇总

**User Story:** As a user, I want to see a summary of all ongoing analyses in the dashboard header, so that I can monitor overall system activity.

#### Acceptance Criteria

1. WHEN one or more sessions have ongoing analyses THEN THE Dashboard_Header SHALL display a spinning indicator on the right side of the title
2. WHEN all analyses complete THEN THE Spinner in Dashboard_Header SHALL be hidden
3. WHEN the user hovers over the global Spinner THEN THE System SHALL display a tooltip showing the count of active analyses
4. THE Dashboard_Header status indicator SHALL update in real-time as analyses start and complete

### Requirement 4: 多会话并发状态管理

**User Story:** As a developer, I want each session to have its own independent state manager, so that concurrent analyses don't interfere with each other.

#### Acceptance Criteria

1. THE Session_Status_Manager SHALL maintain independent state for each Analysis_Session identified by Thread_ID
2. WHEN a new analysis starts THEN THE Loading_State_Manager SHALL register the session with its Thread_ID and initial state
3. WHEN an analysis completes or fails THEN THE Loading_State_Manager SHALL update the session state and notify all subscribers
4. THE Loading_State_Manager SHALL support subscribing to state changes for specific Thread_IDs
5. WHEN a session is deleted THEN THE Loading_State_Manager SHALL clean up the associated state
6. THE Loading_State_Manager SHALL handle timeout scenarios and automatically clear stale loading states after a configurable duration

### Requirement 5: 状态同步与事件传播

**User Story:** As a developer, I want the status updates to propagate correctly between backend and frontend, so that the UI always reflects the actual analysis state.

#### Acceptance Criteria

1. WHEN the backend emits an analysis-progress event THEN THE Loading_State_Manager SHALL update the corresponding session state
2. WHEN the backend emits an analysis-completed event THEN THE Loading_State_Manager SHALL mark the session as complete
3. WHEN the backend emits an analysis-error event THEN THE Loading_State_Manager SHALL mark the session as failed with error information
4. THE System SHALL use Thread_ID to correctly route status updates to the appropriate session
5. WHEN a session switches THEN THE UI SHALL display the correct loading state for the newly active session
