# Plan: LLM Chat Integration

## Phase 1: Persistence & Backend Services [checkpoint: 6115136]
- [x] Task: Create `ChatService` in Go to handle persistence (Load/Save history). c848a27
- [x] Task: Implement `DeleteThread` and `ClearHistory` in `App.go`. d9af5c3
- [x] Task: Update `LLMService` to support OpenAI-compatible base URL and Anthropic protocol. c451379
- [x] Task: Unit tests for `ChatService` and `LLMService`. c451379
- [x] Task: Conductor - User Manual Verification 'Phase 1: Persistence & Backend Services' (Protocol in workflow.md) (6115136)

## Phase 2: Configuration & Settings UI [checkpoint: c605d46]
- [x] Task: Update `PreferenceModal.tsx` to include fields for OpenAI-compatible base URL. 62e6cbf
- [x] Task: Add validation for API keys and URLs in the Settings UI. 62e6cbf
- [x] Task: Implement "Test Connection" button in Settings. 62e6cbf
- [x] Task: Unit tests for updated `PreferenceModal`. 62e6cbf
- [x] Task: Conductor - User Manual Verification 'Phase 2: Configuration & Settings UI' (Protocol in workflow.md) (c605d46)

## Phase 3: Enhanced Chat Interface
- [ ] Task: Implement multi-thread sidebar in `ChatArea.tsx` or a new `ChatSidebar.tsx`.
- [ ] Task: Add "New Chat" and "Clear Chat" functionality.
- [ ] Task: Integrate `react-markdown` and `syntax-highlighter` for Markdown rendering.
- [ ] Task: Add loading indicators (typing state) during API calls.
- [ ] Task: Unit tests for `ChatArea` and Markdown rendering.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Enhanced Chat Interface' (Protocol in workflow.md)

## Phase 4: Integration & Polishing
- [ ] Task: Wire up frontend chat state with backend persistence.
- [ ] Task: Ensure smooth switching between chat threads.
- [ ] Task: Final UI/UX polish (consistent styling, transitions).
- [ ] Task: Full build and verification (`wails build`).
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Integration & Polishing' (Protocol in workflow.md)
