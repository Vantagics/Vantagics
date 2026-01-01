# Plan: Fix Browse Directory Dialog (Flash and Disappear)

This plan outlines the steps to diagnose and fix the directory selection dialog issue on macOS.

## Phase 1: Diagnostics and Backend Stability (Go)

- [x] **Task 1: Add Logging to `SelectDirectory`.**
- [x] **Task 2: Verify Context Initialization.**
    - Ensure `a.ctx` is properly handled and not being overwritten or cleared.
- [x] **Task: Conductor - User Manual Verification 'Backend Stability' (Protocol in workflow.md)**

## Phase 2: Frontend Robustness (React)

- [x] **Task 1: Refine Browse Button in `PreferenceModal.tsx`.**
- [x] **Task 2: Prevent Event Bubbling.**
- [x] **Task: Conductor - User Manual Verification 'Frontend Robustness' (Protocol in workflow.md)**

## Phase 3: Enhanced Backend Fix (Retry)

- [x] **Task 1: Add robust options and delay to `SelectDirectory`.**
- [x] **Task 2: Diagnostic Dialog Test.**
- [x] **Task 3: Minimalist Dialog and Threading fix.**
- [x] **Task: Conductor - User Manual Verification 'Enhanced Fix' (Protocol in workflow.md)** [Status: Failed/Incomplete - Dialog still flashes]

## Phase 4: Final Verification
