# Plan: Fix OpenAI-Compatible Connection Test (404 Error)

This plan outlines the steps to fix the "404 Not Found" error when testing connections to OpenAI-Compatible LLM providers by refining the URL construction logic and adding comprehensive tests.

## Phase 1: Diagnostics and Unit Testing (Go)

This phase focuses on reproducing the issue through unit tests and identifying the exact scenarios where path appending fails.

- [x] **Task 1: Add comprehensive unit tests for URL construction.**
    - Expand `src/llm_service_test.go` to include a dedicated test for `chatOpenAI` and `chatClaudeCompatible` URL construction.
    - Test cases:
        - `http://localhost:11434` (Ollama default)
        - `http://localhost:11434/` (Trailing slash)
        - `http://localhost:11434/v1` (Custom base)
        - `http://localhost:11434/api/chat` (Full custom path)
- [x] **Task 2: Identify the point of failure.**
    - Run the tests and confirm which cases result in incorrect URLs.
- [ ] **Task: Conductor - User Manual Verification 'Diagnostics' (Protocol in workflow.md)**

## Phase 2: Refine URL Logic (Go)

This phase involves implementing a more robust URL construction mechanism.

- [x] **Task 1: Implement a smarter path appending function.**
- [x] **Task 2: Handle specific provider variations (if necessary).**
- [x] **Task: Conductor - User Manual Verification 'Refine URL Logic' (Protocol in workflow.md)** [checkpoint: 2999617]

## Phase 3: Verification and Error Handling

This phase ensures the fix works in the UI and improves the user experience during failures.

- [x] **Task 1: Improve 404 error messaging.**
    - Update the error handling in `src/llm_service.go` to provide a more actionable message when a 404 occurs.
- [x] **Task 2: Final End-to-End Test.**
    - Manually verify the "Test Connection" button in the UI using a local mock server or a real Ollama instance.
- [x] **Task: Conductor - User Manual Verification 'Final Verification' (Protocol in workflow.md)** [checkpoint: 2c285ee]
