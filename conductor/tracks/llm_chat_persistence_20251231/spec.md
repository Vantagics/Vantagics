# Specification: LLM Chat Integration

## 1. Overview
Implement a fully functional LLM (Large Language Model) chat interface within the RapidBI application. This allows users to interact with an AI assistant for data analysis and general queries, with support for configurable LLM providers (OpenAI, Anthropic). The chat history will be persistent, and the interface will support Markdown rendering and multiple chat threads.

## 2. Functional Requirements

### 2.1 LLM Configuration
*   **Provider Selection:** Users can select between "OpenAI" (and compatible) and "Anthropic" providers via a Settings modal.
*   **API Key Management:** Users can securely input and save their API keys.
*   **Endpoint Configuration:** Users can override the base URL (useful for local models like Ollama or LocalAI).
*   **Model Selection:** Users can specify the model name (e.g., `gpt-4`, `claude-3-opus`).
*   **Parameter Tuning:** Users can adjust `MaxTokens`.

### 2.2 Chat Interface
*   **Chat Window:** A dedicated chat area displaying the conversation history.
*   **Input:** Text input field for user queries.
*   **Markdown Support:** The chat must render Markdown content (code blocks, headers, lists, bold/italic text) from the LLM response.
*   **Loading State:** Visual indication (e.g., typing indicator) while waiting for the LLM response.
*   **Error Handling:** Display user-friendly error messages if the API call fails (e.g., invalid key, network error).

### 2.3 Chat Management
*   **Persistence:** Chat history and active threads must be saved to a local file/database (`~/rapidbi/chat_history.json` or similar) and loaded on application start.
*   **Multi-Thread Support:**
    *   Sidebar or menu to list past chat sessions.
    *   Ability to create a "New Chat".
    *   Ability to switch between chat threads.
*   **Clear Chat:** Option to clear the current chat history or delete a thread.

### 2.4 Backend (Go)
*   **API Integration:** Extend `LLMService` to fully support the selected providers.
*   **Persistence Layer:** Implement methods to `SaveHistory`, `LoadHistory`, `GetThreads`, `DeleteThread`.
*   **Context Management:** (Optional/Future) Pass relevant app context (e.g., dashboard metrics) to the LLM if currently visible.

## 3. User Experience (UX)
*   **Settings Access:** Accessible via the "File -> Settings" menu or a gear icon in the UI.
*   **Visuals:** Clean, modern interface consistent with the existing dashboard design.
*   **Feedback:** Immediate feedback on user actions (sending message, changing settings).

## 4. Technical Constraints
*   **Language:** Go (Backend), React/TypeScript (Frontend).
*   **Framework:** Wails.
*   **Storage:** Local file system (JSON) for MVP.

## 5. Out of Scope
*   Streaming responses (MVP will wait for full response, though architecture should allow future upgrade).
*   File attachments/uploads to LLM.
*   Voice input/output.
