# Plan: Enforce Unique Chat Names per Data Source

## Phase 1: Backend Logic
- [x] **Task 1: Update ChatService to handle title uniqueness.**
    - Modify `ChatService` to check for existing titles with the same `data_source_id`.
    - Implement a helper function `ensureUniqueTitle(dataSourceId, title)` that appends a counter (e.g., "(1)", "(2)") if a duplicate exists.
    - Update `SaveThreads` or create a specific `CreateThread` / `RenameThread` method that uses this logic.

## Phase 2: Frontend Integration
- [x] **Task 2: Update Chat Creation.**
    - Ensure `NewChatModal` or `ChatSidebar` calls the backend to get the unique name (or relies on the backend to sanitize it before saving).
    - If the backend sanitizes it, the frontend needs to reload the thread list or receive the updated thread object.

## Phase 3: Verification
- [x] **Task 3: Unit Tests.**
    - Add tests in `chat_service_test.go` to verify uniqueness logic for same and different data sources.
- [x] **Task 4: Manual Verification.**
    - Create multiple "New Chat" sessions for the same data source.
    - Verify they get unique names (e.g., "New Chat", "New Chat (1)").
    - Verify creating a chat with the same name as an existing one results in a unique name.
