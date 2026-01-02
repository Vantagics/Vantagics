# Specification: Fix Browse Directory Dialog (Flash and Disappear)

## Overview
The "Browse" button for the Data Cache Directory triggers a directory selection dialog that immediately disappears on macOS. While other context-dependent actions (like opening settings from the menu) work, this specific dialog interaction is unstable.

## Functional Requirements
- **Backend (Go):**
    - Ensure `SelectDirectory` handles the Wails context robustly.
    - Add logging to track when the dialog is opened and what it returns.
    - Check if `runtime.OpenDirectoryDialog` requires any specific options to remain stable on macOS.
- **Frontend (React):**
    - Ensure the button click doesn't trigger a double event or a race condition with the modal state.
    - Add logging to the frontend to catch any errors returned by the bridge.

## Non-Functional Requirements
- **Stability:** The dialog must remain open until the user selects a directory or cancels.
- **Observability:** Better logging to diagnose bridge failures.

## Acceptance Criteria
- [ ] User can click "Browse" and the system directory picker remains open.
- [ ] Selecting a directory correctly updates the input field.
- [ ] Canceling the dialog doesn't crash the app or cause weird behavior.

## Out of Scope
- Implementing custom file pickers (staying with system dialogs).
