# Specification: Claude-Compatible LLM Provider

## Overview
This track implements a new "Claude-Compatible" LLM provider in the application's settings. This allows users to connect to third-party proxies, local gateways, or cloud-managed Claude instances (like AWS Bedrock or Google Vertex AI) that expose an Anthropic-like or OpenAI-like API for Claude models.

## Functional Requirements
- **New Provider Option:** Add "Claude-Compatible" to the LLM Provider dropdown in the Preference Modal.
- **Configurable Fields:**
    - **Base URL:** The custom endpoint for the Claude-compatible service.
    - **API Key:** The credential for authentication.
    - **Model Name:** The specific Claude model identifier.
    - **Max Tokens:** Adjustable limit for response length.
- **Header Flexibility:** Implement a toggle or auto-selection to support both:
    - **Anthropic Style:** `x-api-key` and `anthropic-version`.
    - **OpenAI Style:** `Authorization: Bearer <key>`.
- **UI Enhancements:**
    - Provide placeholder hints for common proxies (AWS Bedrock, Vertex AI, One API).
    - Update the `LLMService` in Go to handle the new provider logic and header variations.
    - Ensure the "Test Connection" feature works with this new provider.

## Non-Functional Requirements
- **Security:** Ensure API keys are handled securely within the existing configuration framework.
- **UX:** Clear labeling and helpful hints for users who might not be familiar with proxy configurations.

## Acceptance Criteria
- [ ] "Claude-Compatible" appears in the Provider dropdown.
- [ ] Users can save a custom Base URL and API Key for this provider.
- [ ] Users can toggle between Anthropic-style and OpenAI-style headers (or have it automatically handled based on URL/config).
- [ ] Successful chat completion using a Claude-compatible proxy.
- [ ] "Test Connection" button returns success when configured correctly.

## Out of Scope
- Direct integration with AWS SDK or Google Cloud SDK (must be via an HTTP proxy/gateway).
- Streaming responses (unless already supported by the base LLM service).
