# Track Plan: Enable Context Menu for Inputs

## Phase 1: Enable Native Context Menu [checkpoint: b430c6b]
Goal: Unblock the standard system context menu for editable elements.

- [x] Task: CSS - Allow Text Selection in Inputs (2b5d97d)
    - [ ] Write Tests: Verify `-webkit-user-select: text` is applied to inputs
    - [ ] Implement: Add global CSS rules for inputs and textareas in `style.css`
- [x] Task: JS - Ensure Context Menu Events Propagation (90bbe97)
    - [ ] Write Tests: Verify `contextmenu` event is not prevented on inputs
    - [ ] Implement: Add global event listener to allow standard behavior for editable elements
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Enable Native Context Menu' (Protocol in workflow.md)

## Phase 2: Custom Fallback Context Menu (Optional/Contingency) [checkpoint: 78b0667]
Goal: Provide a custom menu if native menus are still blocked by Wails/WebKit on macOS.

- [x] Task: Frontend - Custom Context Menu Component (ac37274)
    - [ ] Write Tests: Verify menu renders at cursor position and has Copy/Paste/Select All
    - [ ] Implement: Create `ContextMenu.tsx` and integrate into a global provider or wrapper
- [x] Task: UI/UX - Integrate Custom Menu with Inputs (ec828bb)
    - [ ] Write Tests: Verify right-click on `PreferenceModal` inputs triggers custom menu
    - [ ] Implement: Hook inputs to the custom context menu logic
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Custom Fallback Context Menu' (Protocol in workflow.md)

## Phase 3: Final Verification & Cleanup
Goal: Ensure the behavior is consistent and bug-free across the app.

- [x] Task: Integration - Verify all app inputs (7916937)
    - [ ] Write Tests: Regression testing for chat and settings inputs
    - [ ] Implement: Manual check of all interactive text fields
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Final Verification & Cleanup' (Protocol in workflow.md)
