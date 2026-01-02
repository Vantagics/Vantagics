# Specification: Disable Auto-Input Corrections for Technical Settings

## Overview
Technical input fields in the Settings modal (Model Name, API Base URL, API Key, Data Cache Directory) are currently prone to unwanted browser/OS-level corrections like auto-capitalization and spell-checking. These corrections can break configurations (e.g., changing `gpt-4` to `Gpt-4`). This track ensures all sensitive technical fields have these automatic features disabled.

## Functional Requirements
- **Update Settings Inputs:** Modify `src/frontend/src/components/PreferenceModal.tsx` to apply the following attributes to technical input fields:
    - `autoCapitalize="none"`
    - `autoCorrect="off"`
    - `spellCheck={false}`
- **Target Fields:**
    - Model Name
    - API Base URL
    - API Key
    - Data Cache Directory

## Acceptance Criteria
- [ ] Typing in the "Model Name" field does not automatically capitalize the first letter.
- [ ] The "API Base URL" field does not trigger spell-check warnings.
- [ ] The "API Key" field remains exactly as typed without correction.
- [ ] The "Data Cache Directory" path remains lowercase if typed that way.

## Out of Scope
- Changing inputs in the chat interface (where capitalization might be desired).
