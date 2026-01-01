# Plan: Data Cache Directory Setting

This plan outlines the steps to add the Data Cache Directory setting to the system preferences.

## Phase 1: Backend Implementation (Go)

- [x] **Task 1: Update Config Struct and Default Logic.**
    - Modify `Config` struct in `src/app.go` to include `DataCacheDir`.
    - Update `GetConfig` to populate `DataCacheDir` with default (`~/RapidBI`) if empty.
- [x] **Task 2: Implement Validation Logic.**
    - Update `SaveConfig` in `src/app.go` to validate that `DataCacheDir` exists and is a directory. Return an error if not.
- [x] **Task 3: Update Tests.**
    - Add unit tests in `src/app_test.go` to verify default value generation and validation logic (valid vs invalid paths).
- [x] **Task: Conductor - User Manual Verification 'Backend Implementation' (Protocol in workflow.md)** [checkpoint: 70311a2]

## Phase 2: Frontend Implementation (React)

- [x] **Task 1: Update Frontend Models.**
- [x] **Task 2: Update PreferenceModal UI.**
- [x] **Task 3: Update Frontend Tests.**
- [x] **Task: Conductor - User Manual Verification 'Frontend Implementation' (Protocol in workflow.md)** [checkpoint: f67b1b3]

## Phase 3: Browse Button Implementation

- [x] **Task 1: Add Directory Picker to Backend.**
- [x] **Task 2: Update PreferenceModal with Browse Button.**
- [x] **Task 3: Update Frontend Tests.**
- [x] **Task: Conductor - User Manual Verification 'Browse Button' (Protocol in workflow.md)** [checkpoint: ad3c1bd]

## Phase 4: Integration and Verification

- [x] **Task 1: End-to-End Verification.**
- [x] **Task: Conductor - User Manual Verification 'Final Verification' (Protocol in workflow.md)** [checkpoint: ad3c1bd]