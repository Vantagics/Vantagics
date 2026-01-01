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

- [ ] **Task 1: Update Frontend Models.**
    - Update `src/frontend/wailsjs/go/models.ts` to include `dataCacheDir`.
- [ ] **Task 2: Update PreferenceModal UI.**
    - Add a text input for "Data Cache Directory" in the "System Parameters" tab of `src/frontend/src/components/PreferenceModal.tsx`.
    - Handle state change for this field.
- [ ] **Task 3: Update Frontend Tests.**
    - Update `src/frontend/src/components/PreferenceModal.test.tsx` to verify the new field exists and interacts correctly.
- [ ] **Task: Conductor - User Manual Verification 'Frontend Implementation' (Protocol in workflow.md)**

## Phase 3: Integration and Verification

- [ ] **Task 1: End-to-End Verification.**
    - Build and run the app.
    - Verify default path.
    - Try setting an invalid path (expect error).
    - Try setting a valid path (expect success).
- [ ] **Task: Conductor - User Manual Verification 'Final Verification' (Protocol in workflow.md)**