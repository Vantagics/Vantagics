# Track Plan: Fix Paste Functionality & Build Automation

## Phase 1: Fix Context Menu & Paste logic [checkpoint: 94b2249]
Goal: Ensure the custom context menu works correctly and doesn't trigger the browser menu.

- [x] Task: JS - Prevent Native Menu during Custom Menu interaction (b0e5e79)
    - [ ] Write Tests: Verify `contextmenu` events on custom menu items are prevented
    - [ ] Implement: Add `e.preventDefault()` and `e.stopPropagation()` to `ContextMenu` buttons
- [x] Task: Frontend - Reliable Paste using Wails Runtime (f6ddc90)
    - [ ] Write Tests: Mock Wails Clipboard API and verify paste logic in `ContextMenu`
    - [ ] Implement: Update `handleAction('paste')` to use `window.runtime.ClipboardGetText()` (or similar)
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Fix Context Menu & Paste logic' (Protocol in workflow.md)

## Phase 2: Workflow Automation [checkpoint: b2e3b8e]
Goal: Update the project workflow to include mandatory build verification.

- [x] Task: Conductor - Update Workflow File (fe4d7c5)
    - [ ] Write Tests: N/A (Manual verification of file content)
    - [ ] Implement: Add build commands to the "Standard Task Workflow" and "Quality Gates" in `workflow.md`
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Workflow Automation' (Protocol in workflow.md)

## Phase 3: Final Verification [checkpoint: b2d64d1]
Goal: Confirm both fixes work end-to-end.

- [x] Task: Integration - Verify Paste and Build (e86ea35)
    - [x] Write Tests: regression test for input paste
    - [x] Implement: Final full build and manual paste check
- [x] Task: Conductor - User Manual Verification 'Phase 3: Final Verification' (Protocol in workflow.md) (b2d64d1)
