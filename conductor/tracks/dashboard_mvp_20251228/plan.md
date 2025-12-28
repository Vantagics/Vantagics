# Track Plan: Smart Dashboard MVP

## Phase 1: Project Foundation & Mock Data Service
Goal: Set up the backend structure to serve data and establish the frontend scaffolding.

- [x] Task: Backend - Define Data Structures in Go (901d880)
    - [ ] Write Tests: Define expected JSON structure for Dashboard Data
    - [ ] Implement: Create Go structs for metrics and insights in `app.go`
- [ ] Task: Backend - Implement Mock Data Provider
    - [ ] Write Tests: Verify `GetDashboardData` returns expected mock values
    - [ ] Implement: Add `GetDashboardData` method to `App` struct
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Project Foundation & Mock Data Service' (Protocol in workflow.md)

## Phase 2: Core Dashboard UI
Goal: Build the visual structure of the dashboard.

- [ ] Task: Frontend - Dashboard Layout Component
    - [ ] Write Tests: Verify Layout renders children and has basic Tailwind classes
    - [ ] Implement: Create `DashboardLayout.tsx` with a responsive grid
- [ ] Task: Frontend - Metric Card Component
    - [ ] Write Tests: Verify `MetricCard` displays title and value correctly
    - [ ] Implement: Create `MetricCard.tsx` with approachable styling
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Core Dashboard UI' (Protocol in workflow.md)

## Phase 3: Smart Insights & Data Integration
Goal: Connect the UI to the backend and display automated insights.

- [ ] Task: Frontend - Smart Insight Component
    - [ ] Write Tests: Verify `SmartInsight` renders the insight text and an icon
    - [ ] Implement: Create `SmartInsight.tsx` following the visual richness guideline
- [ ] Task: Frontend - Integrate Backend Data
    - [ ] Write Tests: Mock Wails runtime and verify data fetching logic
    - [ ] Implement: Fetch data from Go backend and populate the dashboard components
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Smart Insights & Data Integration' (Protocol in workflow.md)

## Phase 4: Final Polishing & Verification
Goal: Ensure the MVP meets all guidelines and is ready for review.

- [ ] Task: UI/UX - Apply Final Styling & Animations
    - [ ] Write Tests: Ensure no regressions in component rendering
    - [ ] Implement: Add subtle transitions and refine the Tailwind theme for "Visual Richness"
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Final Polishing & Verification' (Protocol in workflow.md)
