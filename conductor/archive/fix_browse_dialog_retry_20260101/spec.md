# Specification: Fix Browse Directory Dialog (Retry)

## Overview
The "Browse" button for the Data Cache Directory triggers a directory selection dialog that immediately disappears on macOS. Previous attempts using `runtime.OpenDirectoryDialog` with delays and focus management failed. This track attempts alternative approaches, specifically focusing on ensuring the dialog is invoked on the correct thread or window context, or using a pure frontend-triggered file input if feasible as a fallback.

## Functional Requirements
- **Backend (Go):**
    - Investigate if `runtime.OpenDirectoryDialog` needs to be run on the main thread using `dispatch_async` equivalent or if Wails has a specific thread-safety mechanism we missed.
    - **Crucial:** Verify if `runtime.WindowSetAlwaysOnTop(a.ctx, false)` is actually working or if the window style itself (transparent title bar) is conflicting with the sheet.
- **Frontend (React):**
    - No major changes, just ensure it calls the backend correctly.

## Alternative Strategy: Hidden File Input
If the native dialog continues to fail, we can use a hidden `<input type="file" webkitdirectory />` on the frontend. This is a standard HTML5 feature that works in Wails/Electron environments and uses the native picker without needing backend orchestration.
- **Pros:** Native behavior, handled by the webview engine (WebKit).
- **Cons:** Might need specific permission handling, but usually works out of the box in Wails.

## Acceptance Criteria
- [ ] Clicking "Browse" opens a directory picker that stays open.
- [ ] Selecting a directory updates the path.

## Out of Scope
- Re-implementing the native dialog if the HTML5 fallback works perfectly.
