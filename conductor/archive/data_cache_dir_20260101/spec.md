# Specification: Data Cache Directory Setting

## Overview
Add a configuration option in the System Preferences to allow users to specify a custom "Data Cache Directory". This directory will be used to store application data (like local cache, chat history, etc.). The default should be `~/RapidBI` (or OS equivalent). When the user changes this setting, the application must verify that the directory exists (or create it if possible, though the requirement says "verify existence", implying validation).

## Functional Requirements
- **Backend (Go):**
    - Add `DataCacheDir` field to the `Config` struct.
    - Default `DataCacheDir` to `UserHomeDir/RapidBI` if empty.
    - Validate that the specified directory exists and is writable when saving configuration.
- **Frontend (React):**
    - Add an input field for "Data Cache Directory" in `PreferenceModal.tsx` under "System Parameters".
    - Display the current value.
    - Show an error if validation fails (handled by backend `SaveConfig` error return).

## Non-Functional Requirements
- **UX:** Clear label and maybe a placeholder showing the default.
- **Safety:** Ensure the application doesn't crash if the directory is deleted externally; it should probably fall back to default or recreate it on startup.

## Acceptance Criteria
- [ ] User can see the current Data Cache Directory in Settings.
- [ ] Default value is correctly set to `~/RapidBI` (or equivalent) on first run.
- [ ] User can change the path.
- [ ] If the user enters a non-existent path, saving should fail with an error message.
- [ ] The setting is persisted in `config.json`.

## Out of Scope
- Migrating existing data to the new directory (for now, just changing the pointer for future operations).
- "Browse" button to open a file picker (unless easily available via Wails, but text input is the MVP).
