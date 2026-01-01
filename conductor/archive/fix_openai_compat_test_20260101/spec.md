# Specification: Fix OpenAI-Compatible Connection Test (404 Error)

## Overview
Users are experiencing a "404 Not Found" error when testing the connection for the "OpenAI-Compatible" provider, despite providing a correct base URL (e.g., `http://localhost:11434`). This indicates that the automatically appended path suffix is incorrect for some providers (like Ollama or LM Studio) or that the path manipulation logic is flawed.

## Functional Requirements
- **Fix Path Appending:** Refine the logic that appends `/v1/chat/completions` to ensure it works correctly with various base URL formats.
- **Provider-Specific Handling:** Consider if different "OpenAI-Compatible" providers (like Ollama) need different default paths (e.g., `/api/chat` vs `/v1/chat/completions`).
- **Improved Error Reporting:** If a 404 occurs, provide a more helpful message suggesting the user check their Base URL or path.
- **Support Full URL Input:** Ensure that if a user provides a full URL (already containing a path), the app does not append an additional suffix.

## Non-Functional Requirements
- **Robustness:** The URL construction should be resilient to trailing slashes and common variations in API endpoints.
- **Testability:** Add unit tests covering various Base URL inputs and their expected output URLs.

## Acceptance Criteria
- [ ] Users can successfully test the connection to local providers (like Ollama) using just the base URL.
- [ ] Entering a full URL including the path also works correctly.
- [ ] No more "404 Not Found" errors due to incorrect path appending.
- [ ] Unit tests pass for various URL scenarios.

## Out of Scope
- Support for non-HTTP based local models.
- Automatic discovery of local model servers.
