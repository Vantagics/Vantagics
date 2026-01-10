# Plan: Advanced BI Analysis with Eino Agent Framework

This plan outlines the implementation of an Eino-powered BI agent capable of guided data analysis, Python-based tool execution, and multi-modal responses.

## Phase 1: Eino Core & Infrastructure
- [x] Task: Refactor `EinoService` in `src/agent/eino.go` to support Anthropic/Claude-compatible models and tool calling. [ca0f826]
- [x] Task: Implement a stateful Eino graph (using `compose.Graph`) that incorporates conversational memory. [f334bd4]
- [ ] Task: Create a base `ChatService` integration to route specific "Analysis" messages to the Eino agent.
- [ ] Task: Write unit tests for the Eino graph construction and memory persistence.
- [ ] Task: Conductor - User Manual Verification 'Eino Core & Infrastructure' (Protocol in workflow.md)

## Phase 2: Data-Driven Tools (Python execution)
- [ ] Task: Implement `PythonExecutorTool` as an Eino-compatible component that uses `PythonService` to run generated code.
- [ ] Task: Implement `DataSourceContextTool` to provide the agent with schema details and data samples.
- [ ] Task: Develop a multi-modal response parser in Go to identify and structure text, charts (images), and table data from tool outputs.
- [ ] Task: Write tests for Python tool execution and output parsing.
- [ ] Task: Conductor - User Manual Verification 'Data-Driven Tools' (Protocol in workflow.md)

## Phase 3: Frontend UI & UX Enhancements
- [ ] Task: Update `MessageBubble.tsx` and `ChatArea.tsx` to render interactive charts (using ECharts or similar) and scrollable data tables.
- [ ] Task: Implement UI support for "Interactive Suggestion" buttons sent by the agent.
- [ ] Task: Add a "Proactive Insight" trigger in the frontend that requests initial analysis when a data source is first connected to a chat.
- [ ] Task: Test frontend rendering of multi-modal messages and interactive components.
- [ ] Task: Conductor - User Manual Verification 'UI & UX Enhancements' (Protocol in workflow.md)

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
