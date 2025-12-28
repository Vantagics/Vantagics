# Track Plan: Enable Context Menu for Inputs

## Phase 1: Enable Native Context Menu
Goal: Unblock the standard system context menu for editable elements.

- [x] Task: CSS - Allow Text Selection in Inputs (2b5d97d)
    - [ ] Write Tests: Verify `-webkit-user-select: text` is applied to inputs
    - [ ] Implement: Add global CSS rules for inputs and textareas in `style.css`
- [ ] Task: JS - Ensure Context Menu Events Propagation
    - [ ] Write Tests: Verify `contextmenu` event is not prevented on inputs
    - [ ] Implement: Add global event listener to allow standard behavior for editable elements
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Enable Native Context Menu' (Protocol in workflow.md)

## Phase 2: Custom Fallback Context Menu (Optional/Contingency)
Goal: Provide a custom menu if native menus are still blocked by Wails/WebKit on macOS.

- [ ] Task: Frontend - Custom Context Menu Component
    - [ ] Write Tests: Verify menu renders at cursor position and has Copy/Paste/Select All
    - [ ] Implement: Create `ContextMenu.tsx` and integrate into a global provider or wrapper
- [ ] Task: UI/UX - Integrate Custom Menu with Inputs
    - [ ] Write Tests: Verify right-click on `PreferenceModal` inputs triggers custom menu
    - [ ] Implement: Hook inputs to the custom context menu logic
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Custom Fallback Context Menu' (Protocol in workflow.md)

## Phase 3: Final Verification & Cleanup
Goal: Ensure the behavior is consistent and bug-free across the app.

- [ ] Task: Integration - Verify all app inputs
    - [ ] Write Tests: Regression testing for chat and settings inputs
    - [ ] Implement: Manual check of all interactive text fields
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Final Verification & Cleanup' (Protocol in workflow.md)
