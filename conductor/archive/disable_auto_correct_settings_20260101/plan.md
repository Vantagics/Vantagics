# Plan: Disable Auto-Input Corrections for Technical Settings

This plan outlines the steps to disable auto-capitalization, auto-correction, and spell-checking for technical configuration fields in the Preferences modal.

## Phase 1: Frontend Attribute Updates (React)

- [x] **Task 1: Update technical inputs in `PreferenceModal.tsx`.**
    - Locate the following input fields: `baseUrl`, `apiKey`, `modelName`, `dataCacheDir`.
    - Add the attributes: `autoCapitalize="none"`, `autoCorrect="off"`, `spellCheck={false}`.
- [x] **Task 2: Verify changes with automated tests.**
    - Update `src/frontend/src/components/PreferenceModal.test.tsx` to verify that these attributes are present on the target inputs.
- [ ] **Task: Conductor - User Manual Verification 'Frontend Updates' (Protocol in workflow.md)**

## Phase 2: Final Verification

- [x] **Task 1: Build and Manual Check.**
    - Rebuild the application and confirm that typing in these fields no longer triggers auto-capitalization or corrections.
- [x] **Task: Conductor - User Manual Verification 'Final Verification' (Protocol in workflow.md)** [checkpoint: 0361cc5]
