# Track Plan: Fix Paste Functionality & Build Automation

## Phase 1: Fix Context Menu & Paste logic
Goal: Ensure the custom context menu works correctly and doesn't trigger the browser menu.

- [x] Task: JS - Prevent Native Menu during Custom Menu interaction (b0e5e79)
    - [ ] Write Tests: Verify `contextmenu` events on custom menu items are prevented
    - [ ] Implement: Add `e.preventDefault()` and `e.stopPropagation()` to `ContextMenu` buttons
- [ ] Task: Frontend - Reliable Paste using Wails Runtime
    - [ ] Write Tests: Mock Wails Clipboard API and verify paste logic in `ContextMenu`
    - [ ] Implement: Update `handleAction('paste')` to use `window.runtime.ClipboardGetText()` (or similar)
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Fix Context Menu & Paste logic' (Protocol in workflow.md)

## Phase 2: Workflow Automation
Goal: Update the project workflow to include mandatory build verification.

- [ ] Task: Conductor - Update Workflow File
    - [ ] Write Tests: N/A (Manual verification of file content)
    - [ ] Implement: Add build commands to the "Standard Task Workflow" and "Quality Gates" in `workflow.md`
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Workflow Automation' (Protocol in workflow.md)

## Phase 3: Final Verification
Goal: Confirm both fixes work end-to-end.

- [ ] Task: Integration - Verify Paste and Build
    - [ ] Write Tests: regression test for input paste
    - [ ] Implement: Final full build and manual paste check
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Final Verification' (Protocol in workflow.md)
