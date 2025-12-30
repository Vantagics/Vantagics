# Track Specification: Fix Paste Functionality & Build Automation

## Overview
This track addresses a bug where the "Paste" action in the custom context menu fails and triggers a secondary native menu instead. It also involves automating the project's rebuild process by updating the official Conductor workflow.

## Goals
- Fix the `Paste` functionality in the `ContextMenu` component.
- Prevent the browser's native context menu from appearing when clicking custom menu items.
- Update `conductor/workflow.md` to require a full build after each track implementation.

## Functional Requirements
- **Reliable Paste:** The "Paste" button must successfully insert text from the system clipboard into the target input field.
- **Wails Integration:** Use the Wails backend clipboard bridge if browser APIs are restricted.
- **No Double Menus:** Ensure `contextmenu` events are fully handled and prevented from bubbling to the system level when the custom menu is active.
- **Workflow Update:** The `conductor/workflow.md` must be modified to include a "Build Verification" step.

## Technical Requirements
- **Clipboard Handling:** Implement `ClipboardGetText` (or similar) from Wails runtime if `navigator.clipboard` is insufficient.
- **Event Handling:** Use `e.stopPropagation()` and `e.preventDefault()` correctly in `ContextMenu.tsx`.
- **Workflow Modification:** Add `npm run build` and `go build` (with correct tags) to the "Before Marking Task Complete" or "Phase Completion" sections of the workflow.

## Acceptance Criteria
- [ ] User can paste text into API key and URL fields via the custom context menu without seeing a second menu.
- [ ] The application is automatically rebuilt at the end of implementing this (and future) tracks.
- [ ] `conductor/workflow.md` explicitly lists building as a mandatory step.
