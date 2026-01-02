# Specification: Replace Native Confirm Dialog for Clear History

## Overview
The "Clear History" button currently uses the native `window.confirm()` method. On the user's macOS environment, this dialog fails to appear, preventing the action from executing. We will replace the native confirmation with a custom React modal (UI-based) to ensure reliability and consistent styling.

## Functional Requirements
- **Frontend (React):**
    - Create a state variable `isClearHistoryModalOpen` in `ChatSidebar.tsx`.
    - Implement a simple confirmation modal (or reuse `PreferenceModal` style if appropriate, but a small alert dialog is better) that renders when this state is true.
    - The modal should have "Cancel" and "Clear" buttons.
    - Update `handleClearHistory` to open this modal instead of calling `confirm()`.
    - The "Clear" button in the modal should trigger the actual `ClearHistory` backend call and state cleanup.

## Non-Functional Requirements
- **UX:** The modal should match the application's visual style (Tailwind CSS).
- **Accessibility:** Focus management (optional for this quick fix but good practice).

## Acceptance Criteria
- [ ] Clicking "Clear History" opens a custom UI modal asking for confirmation.
- [ ] Clicking "Cancel" closes the modal without action.
- [ ] Clicking "Clear" calls the backend `ClearHistory`, clears the local state, and closes the modal.
- [ ] Native `window.confirm` is removed.

## Out of Scope
- Creating a reusable generic "ConfirmationModal" component (we will inline it or create a simple local one to keep it focused, unless a pattern emerges).
