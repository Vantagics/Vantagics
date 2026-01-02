# Plan: Fix Browse Directory Dialog (Retry)

This plan explores alternative solutions, prioritizing the HTML5 `webkitdirectory` approach as it bypasses the backend context complexity that seems to be causing the issue.

## Phase 1: HTML5 Directory Picker Fallback (React)

- [~] **Task 1: Implement Hidden Input in `PreferenceModal.tsx`.**
    - Add a hidden `<input type="file" webkitdirectory directory />` element.
    - Create a `ref` to trigger it when "Browse" is clicked.
    - Handle the `onChange` event to get the selected path.
    - **Note:** In Wails/Electron, `file.path` usually gives the full path.
- [~] **Task 2: Verify Path Retrieval.**
    - Ensure that the full absolute path is retrieved (which is needed for the backend), not just a fake `C:\fakepath`.
- [x] **Task 3: Refine HTML5 Trigger.**
- [ ] **Task: Conductor - User Manual Verification 'HTML5 Picker' (Protocol in workflow.md)**

## Phase 2: Cleanup (Fallback to Text Input)

- [x] **Task 1: Remove Browse Button.**
    - Since the dialog is consistently unstable on this system, remove the "Browse" button.
    - Users must manually enter the path.
    - Remove the backend `SelectDirectory` code.
- [x] **Task: Conductor - User Manual Verification 'Cleanup' (Protocol in workflow.md)** [checkpoint: clean]

## Phase 3: Final Verification

- [ ] **Task 1: Remove Backend `SelectDirectory` code.**
    - If the frontend solution works, remove the unused backend code to keep it clean.
- [ ] **Task: Conductor - User Manual Verification 'Cleanup' (Protocol in workflow.md)**

## Phase 3: Final Verification

- [x] **Task 1: Build and Test.**
    - Rebuild and confirm the fix on macOS.
- [x] **Task: Conductor - User Manual Verification 'Final Verification' (Protocol in workflow.md)** [checkpoint: removed]
