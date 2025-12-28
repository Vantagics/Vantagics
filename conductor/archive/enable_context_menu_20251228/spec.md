# Track Specification: Enable Context Menu for Inputs

## Overview
This track focuses on enabling standard text manipulation capabilities (Copy, Paste, Select All) via a right-click context menu for all input fields in the application. This is particularly important for non-technical users who rely on these menus for configuring API keys and URLs.

## Goals
- Enable the native system context menu for all `<input>` and `<textarea>` elements.
- Ensure compatibility across macOS (primary development platform) and other OSs.
- Provide a consistent user experience for text editing.

## Functional Requirements
- **Standard Actions:** The menu must include at least "Copy", "Paste", "Cut", and "Select All".
- **Global Availability:** All input fields, including those in the `PreferenceModal` and Chat, must support this.
- **MacOS Specifics:** Ensure that Wails/WebKit on macOS correctly shows the system menu for editable elements.

## Technical Requirements
- **CSS Implementation:** Use `-webkit-user-select: text` and ensure Wails doesn't intercept the `contextmenu` event globally in a way that blocks inputs.
- **JS/React Logic:** If native menus remain blocked by Wails/WebKit, implement a lightweight custom context menu that performs standard operations using the `Clipboard API` and `Selection API`.
- **TDD:** Verify that inputs are not blocking the `contextmenu` event.

## Acceptance Criteria
- [ ] User can right-click any input field and see a menu.
- [ ] User can copy text from an input.
- [ ] User can paste text into an input.
- [ ] User can select all text in an input via the menu.
- [ ] The behavior works correctly in the `PreferenceModal` for API keys and URLs.
