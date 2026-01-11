# Plan: Advanced BI Analysis with Eino Agent Framework

This plan outlines the implementation of an Eino-powered BI agent capable of guided data analysis, Python-based tool execution, and multi-modal responses.

## Phase 1: Eino Core & Infrastructure
- [x] Task: Refactor `EinoService` in `src/agent/eino.go` to support Anthropic/Claude-compatible models and tool calling. [ca0f826]
- [x] Task: Implement a stateful Eino graph (using `compose.Graph`) that incorporates conversational memory. [f334bd4]
- [x] Task: Create a base `ChatService` integration to route specific "Analysis" messages to the Eino agent. [5f447ad]
- [x] Task: Write unit tests for the Eino graph construction and memory persistence. [d439317]
- [x] Task: Conductor - User Manual Verification 'Eino Core & Infrastructure' (Protocol in workflow.md)

## Phase 2: Data-Driven Tools (Python execution)
- [x] Task: Implement `PythonExecutorTool` as an Eino-compatible component that uses `PythonService` to run generated code. [7720307]
- [x] Task: Implement `DataSourceContextTool` to provide the agent with schema details and data samples. [7720307]
- [x] Task: Develop a multi-modal response parser in Go to identify and structure text, charts (images), and table data from tool outputs. [7720307]
- [x] Task: Write tests for Python tool execution and output parsing. [7720307]
- [x] Task: Conductor - User Manual Verification 'Data-Driven Tools' (Protocol in workflow.md)

## Phase 3: Context-Awareness & UI
- [x] Task: Refactor `EinoService.RunAnalysis` to accept `dataSourceID` and dynamically inject the Data Source Schema/Summary into the System Prompt. [e0a449e]
- [x] Task: Update `App.SendMessage` to pass the active thread's `dataSourceID` to `EinoService`. [611d05e]
- [x] Task: Clean up redundant `memoryService` injection code in `App.CreateChatThread`. [611d05e]
- [x] Task: Update `MessageBubble.tsx` and `ChatArea.tsx` to render interactive charts (using ECharts or similar) and scrollable data tables. [611d05e]
- [x] Task: Implement UI support for "Interactive Suggestion" buttons sent by the agent. [611d05e]
- [x] Task: Test frontend rendering of multi-modal messages and interactive components. [611d05e]
- [~] Task: Conductor - User Manual Verification 'Context-Awareness & UI' (Protocol in workflow.md)

## Phase 4: Advanced Skills & Plugin System
- [ ] Task: Implement a dynamic tool registration mechanism that scans a designated "Skills" directory for user-defined Python scripts.
- [ ] Task: Define a standard Python wrapper/interface that allows these scripts to be easily consumed by Eino as tools.
- [ ] Task: Add a UI indicator or "Skill Manager" view (or simple status) to show which custom skills are active.
- [ ] Task: Verify that the agent can successfully select and execute a user-defined "Skill".
- [ ] Task: Conductor - User Manual Verification 'Advanced Skills & Plugin System' (Protocol in workflow.md)

## Phase 5: Export & Final Integration
- [ ] Task: Implement `ExportTool` for the agent to generate Excel and PDF reports from analysis results.
- [ ] Task: Conduct a full end-to-end "Guided Analysis" flow test.
- [ ] Task: Update documentation and user guide regarding "Advanced Skills" and BI features.
- [ ] Task: Conductor - User Manual Verification 'Export & Final Integration' (Protocol in workflow.md)
