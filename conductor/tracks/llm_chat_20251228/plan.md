# Track Plan: LLM-Based Chat Interface

## Phase 1: Sidebar Infrastructure & UI Scaffolding [checkpoint: af80de0]
Goal: Create the visual container for the chat and the message components.
- [Note: Added React Markdown to tech stack for chat rendering (2025-12-28)]

- [x] Task: Frontend - Chat Sidebar Component (4bb0a49)
    - [ ] Write Tests: Verify sidebar toggles visibility and renders correctly
    - [ ] Implement: Create `ChatSidebar.tsx` with toggle logic
- [x] Task: Frontend - Message Bubble Component (f1b1b6d)
    - [ ] Write Tests: Verify Markdown rendering and different sender styles
    - [ ] Implement: Create `MessageBubble.tsx` with `react-markdown`
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Sidebar Infrastructure & UI Scaffolding' (Protocol in workflow.md)

## Phase 2: LLM Service Integration (Backend)
Goal: Implement the communication layer with OpenAI and Anthropic.

- [x] Task: Backend - LLM Client Factory (36b860d)
    - [ ] Write Tests: Verify switching between providers and correct API calls (mocked)
    - [ ] Implement: Create LLM client logic in `llm_service.go`
- [x] Task: Backend - Chat Handler in App (efbe739)
    - [ ] Write Tests: Verify `SendMessage` Wails method returns response
    - [ ] Implement: Add `SendMessage` to `app.go`
- [ ] Task: Conductor - User Manual Verification 'Phase 2: LLM Service Integration (Backend)' (Protocol in workflow.md)

## Phase 3: Visual Insights & Contextual Actions
Goal: Enable the LLM to return UI widgets and action buttons.

- [ ] Task: Frontend - Insight & Action Handlers
    - [ ] Write Tests: Verify components render based on specific JSON payloads in messages
    - [ ] Implement: Update `MessageBubble.tsx` to handle `visual_insight` and `actions` types
- [ ] Task: Integration - Chat Data Analysis Flow
    - [ ] Write Tests: Mock full end-to-end flow from input to visual result
    - [ ] Implement: Connect the frontend input to the backend LLM service and display results
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Visual Insights & Contextual Actions' (Protocol in workflow.md)

## Phase 4: Configuration & Polishing
Goal: Allow user to configure API keys and refine the UX.

- [ ] Task: Frontend - Provider & API Key Settings
    - [ ] Write Tests: Verify API key saving and provider selection updates state
    - [ ] Implement: Update `PreferenceModal.tsx` with LLM settings
- [ ] Task: UI/UX - Final Polish & Transitions
    - [ ] Write Tests: Regression testing for UI components
    - [ ] Implement: Add smooth sliding transitions and loading indicators for the chat
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Final Polishing & Verification' (Protocol in workflow.md)
