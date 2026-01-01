# Plan: Claude-Compatible LLM Provider Implementation

This plan outlines the steps to add support for Claude-Compatible LLM providers, allowing users to configure custom endpoints and header styles for Claude models.

## Phase 1: Backend Infrastructure (Go)

This phase focuses on updating the configuration structure and the LLM service to support the new provider and its specific requirements.

- [x] **Task 1: Add unit tests for Claude-Compatible provider.**
    - Write failing tests in `src/llm_service_test.go` that simulate a Claude-compatible endpoint with both Anthropic-style and OpenAI-style headers.
    - Verify that `NewLLMService` correctly handles the new provider type.
- [x] **Task 2: Update `Config` struct and `LLMService`.**
    - Add `ClaudeHeaderStyle` field to the `Config` struct in `src/app.go`.
    - Update `LLMService` struct in `src/llm_service.go` to include the header style preference.
    - Update `NewLLMService` to initialize the new field.
- [x] **Task 3: Implement `chatClaudeCompatible` logic.**
    - Implement the logic in `src/llm_service.go` to handle requests to Claude-compatible endpoints.
    - Ensure it respects the `ClaudeHeaderStyle` (Anthropic vs OpenAI) and correctly formats the request body and headers.
    - Verify that all tests pass.
- [x] **Task: Conductor - User Manual Verification 'Backend Infrastructure' (Protocol in workflow.md)** [checkpoint: e97fb60]

## Phase 2: Frontend Configuration UI (React)

This phase involves updating the settings UI to allow users to select and configure the Claude-Compatible provider.

- [x] **Task 1: Update `PreferenceModal.tsx` Provider dropdown.**
    - Add "Claude-Compatible" as a selectable option in the LLM Provider dropdown.
- [x] **Task 2: Add Header Style and Custom Endpoint UI.**
    - Add a selection (e.g., Radio or Select) for "Header Style" (Anthropic vs OpenAI) that appears when "Claude-Compatible" is selected.
    - Ensure the "Base URL" field is visible and correctly labeled for this provider.
- [x] **Task 3: Implement placeholders and hints for Claude Proxies.**
    - Add placeholder text and small hint tooltips/text for common proxies like AWS Bedrock, Google Vertex AI, and One API.
- [x] **Task: Conductor - User Manual Verification 'Frontend Configuration UI' (Protocol in workflow.md)** [checkpoint: 523466f]

## Phase 3: Final Integration and Verification

This phase ensures everything works together and provides a smooth user experience.

- [x] **Task 1: Verify Connection Test.**
    - Ensure the "Test Connection" button in the settings modal works correctly for the Claude-Compatible provider across different header styles.
- [x] **Task 2: End-to-End Chat Verification.**
    - Perform a manual chat session using a mock or real Claude-compatible proxy to ensure responses are correctly received and rendered.
- [x] **Task: Conductor - User Manual Verification 'Final Integration and Verification' (Protocol in workflow.md)** [checkpoint: 5e5f160]