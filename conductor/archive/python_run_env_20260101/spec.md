# Specification: Python "Run Env" Configuration

## Overview
Add a new "Run Env" tab to the application's Preferences modal. This tab allows users to configure the Python environment used to execute generated scripts. The system will auto-detect common Python installations (System, Conda, Virtualenvs) and allow the user to choose one from a dropdown.

## Functional Requirements
- **Backend (Go):**
    - Implement a probe service to scan for:
        - System Python paths.
        - Anaconda/Miniconda environments.
        - Standard virtualenv locations.
    - Implement a validation service that runs the selected Python to:
        - Verify existence.
        - Retrieve the version string.
        - Check for the presence of `pandas` and `matplotlib`.
    - Update `Config` struct to store `PythonPath`.
- **Frontend (React):**
    - Add "Run Env" tab to `PreferenceModal.tsx`.
    - Display a dropdown populated with detected environments.
    - Show real-time validation results (Version, Missing Packages) when an environment is selected.
    - Persist the selected path in the application configuration.

## Non-Functional Requirements
- **UX:** Clear feedback if an environment is invalid or missing required libraries.
- **Performance:** Scans should be efficient and non-blocking (async).

## Acceptance Criteria
- [ ] User can see "Run Env" tab in Settings.
- [ ] Dropdown lists detected System and Conda environments.
- [ ] Selecting an environment displays its Python version.
- [ ] Selecting an environment indicates if `pandas` or `matplotlib` are missing.
- [ ] Selected Python path is saved correctly.

## Out of Scope
- Automatic installation of missing Python packages.
- Creation of new virtual environments from within the app.
