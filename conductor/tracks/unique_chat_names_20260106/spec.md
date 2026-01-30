# Specification: Unique Chat Names per Data Source

## Requirement
Within the context of a specific Data Source, chat thread titles must be unique. If a user attempts to create or rename a chat to a title that already exists for that data source, the system should automatically append a suffix (e.g., "(1)", "(2)") to ensure uniqueness.

## Scope
- **Scope**: Chat Service (Backend) and Chat Creation/Renaming flows.
- **Context**: `data_source_id`. Titles can be duplicated *across* different data sources, but not *within* the same one.

## Detailed Behavior
1.  **New Chat Creation**:
    - If user enters "Sales Analysis" and it exists, save as "Sales Analysis (1)".
    - If "Sales Analysis (1)" also exists, save as "Sales Analysis (2)".
2.  **Auto-Naming (First Message)**:
    - If the system auto-generates a title from the first message (e.g., "Analyze Q1"), apply the same uniqueness logic.

## Technical Implementation
- **Backend**: `ChatService.go`
    - Add `GenerateUniqueTitle(dataSourceID, desiredTitle) string`.
    - Use this when creating or updating a thread.
- **Frontend**:
    - Ideally, the frontend sends the user's desired title, and the backend returns the actual saved thread with the unique title.
