# Plan: Python "Run Env" Configuration

This plan outlines the steps to add Python environment detection and configuration to the settings modal.

## Phase 1: Backend Python Probe (Go)

- [~] **Task 1: Update `Config` struct.**
    - Add `PythonPath` to `Config` in `src/app.go`.
- [~] **Task 2: Implement Python detection logic.**
    - Create `src/python_service.go`.
    - Implement `ProbePythonEnvironments` to search for system paths, Conda envs, and venvs.
- [~] **Task 3: Implement Python validation logic.**
    - Implement `ValidatePythonEnvironment(path string)` to get version and check for `pandas`/`matplotlib`.
- [~] **Task 4: Bind methods to `App`.**
    - Add `GetPythonEnvironments` and `ValidatePython` to the `App` struct in `src/app.go`.
- [x] **Task: Conductor - User Manual Verification 'Backend Python Probe' (Protocol in workflow.md)** [checkpoint: bff41f6]

## Phase 2: Frontend "Run Env" UI (React)

- [~] **Task 1: Update Frontend Models.**
    - Update `src/frontend/wailsjs/go/models.ts` to include `pythonPath`.
- [x] **Task 2: Add "Run Env" tab to `PreferenceModal.tsx`.**
    - Add the new tab button in the sidebar.
    - Implement the "Run Env" content area with a dropdown and validation info display.
- [x] **Task 3: Integrate with Backend Probe.**
    - Call `GetPythonEnvironments` on tab mount.
    - Call `ValidatePython` when an environment is selected.
- [x] **Task: Conductor - User Manual Verification 'Frontend UI' (Protocol in workflow.md)** [checkpoint: manual_verify_frontend]

## Phase 3: Integration and Polish

- [x] **Task 1: Persistence Verification.**
    - Ensure the selected `PythonPath` is correctly saved and reloaded.
- [x] **Task 2: Refine styling and UX.**
    - Add loading indicators while probing/validating.
- [x] **Task: Conductor - User Manual Verification 'Final Verification' (Protocol in workflow.md)** [checkpoint: manual_verified_final]
