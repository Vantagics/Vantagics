# Implementation Plan: Western Layout Redesign

## Overview

This implementation plan transforms the current overlay-based UI into a fixed three-panel layout optimized for Western user habits. The implementation follows a phased approach: create new components, update the main layout, migrate state and events, remove old components, and polish the experience.

## Tasks

- [ ] 1. Create core layout components
  - [x] 1.1 Create ResizeHandle component
    - Implement draggable resize handle with visual feedback
    - Add mouse event handlers for drag start, drag, and drag end
    - Implement cursor change on hover
    - Add visual indicator during active drag
    - _Requirements: 1.7, 1.8, 1.9_
  
  - [ ]* 1.2 Write property test for ResizeHandle
    - **Property 24: Resize drag effect**
    - **Validates: Requirements 1.9**
  
  - [x] 1.3 Create PanelWidths utility module
    - Implement calculatePanelWidths function with constraint enforcement
    - Implement handleResizeDrag function for resize calculations
    - Add panel width constraints constants
    - Add localStorage persistence functions
    - _Requirements: 1.4, 9.1, 9.2, 9.4, 9.5_
  
  - [ ]* 1.4 Write property tests for panel width calculations
    - **Property 1: Panel width conservation**
    - **Property 2: Panel width constraints**
    - **Property 3: Panel persistence round-trip**
    - **Validates: Requirements 1.4, 1.6, 9.1, 9.2, 9.4, 9.5**

- [ ] 2. Create LeftPanel component structure
  - [x] 2.1 Create LeftPanel container component
    - Implement main LeftPanel component with props interface
    - Add state management for data sources and sessions
    - Implement fetchDataSources and fetchSessions methods
    - Add context menu state management
    - _Requirements: 2.1, 3.1_
  
  - [x] 2.2 Create DataSourcesSection component
    - Implement data sources list rendering
    - Add data source selection handling
    - Add context menu trigger on right-click
    - Implement empty state display
    - Add "Add Data Source" button in header
    - Display type indicators (icons/colors) for each source
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7_
  
  - [ ]* 2.3 Write property tests for DataSourcesSection
    - **Property 4: List rendering completeness**
    - **Property 6: Data source type indicators**
    - **Property 8: Selection state consistency**
    - **Property 9: Context menu trigger**
    - **Validates: Requirements 2.2, 2.3, 2.4, 2.5**
  
  - [x] 2.4 Create NewSessionButton component
    - Implement button with icon and text
    - Add disabled state when no data source selected
    - Add tooltip explaining requirements
    - Display selected data source name when available
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_
  
  - [ ]* 2.5 Write unit tests for NewSessionButton
    - Test button renders with correct text
    - Test disabled state when no source selected
    - Test click handler triggers dialog
    - Test tooltip displays correctly
    - _Requirements: 4.1, 4.2, 4.4_
  
  - [x] 2.6 Create HistoricalSessionsSection component
    - Implement sessions list rendering with virtualization
    - Add session selection handling
    - Display session metadata (name, date, data source)
    - Implement reverse chronological ordering
    - Add context menu trigger on right-click
    - Implement empty state display
    - _Requirements: 3.1, 3.2, 3.3, 3.5, 3.6, 3.7, 3.8_
  
  - [ ]* 2.7 Write property tests for HistoricalSessionsSection
    - **Property 5: Session chronological ordering**
    - **Property 7: Session metadata completeness**
    - **Property 8: Selection state consistency**
    - **Property 9: Context menu trigger**
    - **Validates: Requirements 3.2, 3.3, 3.5, 3.6, 3.7**

- [x] 3. Checkpoint - Verify LeftPanel components
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 4. Create CenterPanel component
  - [x] 4.1 Create CenterPanel container component
    - Refactor ChatSidebar into CenterPanel (remove overlay logic)
    - Implement fixed positioning in center of layout
    - Add message list rendering with virtualization
    - Add message input area at bottom
    - Implement auto-scroll to latest message
    - Display loading indicator during analysis
    - Display welcome message when no session active
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8_
  
  - [ ]* 4.2 Write property tests for CenterPanel
    - **Property 11: Message send immediacy**
    - **Property 22: Loading state visibility**
    - **Property 26: List virtualization**
    - **Validates: Requirements 5.6, 5.8, 13.3**
  
  - [x] 4.3 Create DataBrowser component structure
    - Implement slide-out panel with absolute positioning
    - Add slide-in/out animations using CSS transforms
    - Implement close button in header
    - Add resize handle on left edge
    - Implement backdrop dim/blur effect on center panel
    - _Requirements: 7.1, 7.2, 7.3, 7.5, 7.6, 7.9, 7.10_
  
  - [x] 4.4 Implement DataBrowser content display
    - Add data source name display in header
    - Implement table list loading and rendering
    - Add table selection handling
    - Implement column and data type display
    - Add sample data rows display (10-20 rows)
    - Implement pagination controls
    - Add row count and column statistics display
    - Implement search/filter for tables and columns
    - Add error state display
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7, 8.8_
  
  - [ ]* 4.5 Write property tests for DataBrowser
    - **Property 10: Data browser toggle**
    - **Property 13: Data browser overlay positioning**
    - **Property 14: Center panel persistence**
    - **Property 15: Data browser content loading**
    - **Property 16: Data browser search filtering**
    - **Property 27: Lazy loading**
    - **Validates: Requirements 7.2, 7.3, 7.6, 7.7, 8.2, 8.3, 8.4, 8.8, 13.6**

- [ ] 5. Create RightPanel component
  - [x] 5.1 Create RightPanel wrapper component
    - Implement RightPanel as wrapper for DraggableDashboard
    - Add fixed positioning on right side of layout
    - Ensure scrollable overflow when content exceeds height
    - Maintain all existing dashboard functionality
    - _Requirements: 6.1, 6.2, 6.5_
  
  - [ ]* 5.2 Write property tests for RightPanel
    - **Property 12: Insight click propagation**
    - **Property 21: Session switch data consistency**
    - **Property 23: Empty state fallback**
    - **Validates: Requirements 6.3, 6.4, 6.6, 6.7**

- [x] 6. Checkpoint - Verify all panel components
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. Update App.tsx with new three-panel layout
  - [x] 7.1 Replace current layout with three-panel flexbox structure
    - Remove ChatSidebar overlay logic
    - Remove ContextPanel (old data browser)
    - Remove collapse/expand button logic
    - Implement three-panel flexbox container
    - Add LeftPanel, CenterPanel, RightPanel components
    - Add ResizeHandle components between panels
    - _Requirements: 1.1, 1.2, 10.1, 10.2_
  
  - [x] 7.2 Implement panel width state management
    - Add panelWidths state to App component
    - Implement resize handlers for both resize handles
    - Add localStorage persistence on resize end
    - Load saved widths on component mount
    - Implement default widths fallback
    - Add window resize handler for proportional scaling
    - _Requirements: 1.6, 1.9, 1.10, 9.1, 9.2, 9.3, 9.7_
  
  - [ ]* 7.3 Write property tests for App layout
    - **Property 1: Panel width conservation**
    - **Property 2: Panel width constraints**
    - **Property 3: Panel persistence round-trip**
    - **Property 24: Resize handle drag effect**
    - **Property 25: Resize debouncing**
    - **Validates: Requirements 1.4, 1.6, 1.9, 9.1, 9.2, 9.4, 9.5, 13.5**

- [ ] 8. Migrate state and event handling
  - [x] 8.1 Update data source selection state
    - Move selectedDataSourceId to App state
    - Wire up onDataSourceSelect handler to LeftPanel
    - Update data browser open logic to use selected source
    - _Requirements: 2.3, 7.1_
  
  - [x] 8.2 Update session selection state
    - Add selectedSessionId to App state
    - Wire up onSessionSelect handler to LeftPanel
    - Implement session loading in CenterPanel
    - Update RightPanel with session results
    - _Requirements: 3.3, 3.4_
  
  - [x] 8.3 Implement data browser state management
    - Add dataBrowserOpen and dataBrowserSourceId to App state
    - Wire up onBrowseData handler from LeftPanel context menu
    - Implement data browser open/close logic
    - Add Escape key handler to close data browser
    - _Requirements: 7.1, 7.6, 11.5_
  
  - [x] 8.4 Update event listeners for new layout
    - Update 'session-switched' event handler
    - Update 'data-source-selected' event handler
    - Update 'start-new-chat' event handler
    - Update 'dashboard-update' event handler
    - Ensure all existing events work with new structure
    - _Requirements: 10.3, 10.4, 10.5, 10.6_
  
  - [ ]* 8.5 Write integration tests for state management
    - **Property 21: Session switch data consistency**
    - **Property 30: Feature preservation**
    - **Validates: Requirements 3.3, 3.4, 6.7, 10.3, 10.4, 10.5**

- [ ] 9. Implement keyboard navigation
  - [x] 9.1 Add panel focus keyboard shortcuts
    - Implement Ctrl+1 (Cmd+1) to focus LeftPanel
    - Implement Ctrl+2 (Cmd+2) to focus CenterPanel
    - Implement Ctrl+3 (Cmd+3) to focus RightPanel
    - Add visual focus indicators for focused panel
    - _Requirements: 11.1, 11.2, 11.3, 11.6_
  
  - [x] 9.2 Add data browser keyboard shortcuts
    - Implement Ctrl+B (Cmd+B) to toggle data browser
    - Implement Escape to close data browser when open
    - _Requirements: 11.4, 11.5_
  
  - [x] 9.3 Implement tab navigation within panels
    - Add tab navigation containment for each panel
    - Ensure tab wraps to first element after last
    - _Requirements: 11.7_
  
  - [-] 9.4 Add ARIA attributes for accessibility
    - Add ARIA roles to all panels
    - Add ARIA labels to all interactive elements
    - Add ARIA live regions for dynamic content
    - Ensure screen reader compatibility
    - _Requirements: 11.8_
  
  - [ ]* 9.5 Write property tests for keyboard navigation
    - **Property 17: Panel focus shortcuts**
    - **Property 18: Escape key data browser close**
    - **Property 19: Tab navigation containment**
    - **Property 20: ARIA attribute presence**
    - **Validates: Requirements 11.1, 11.2, 11.3, 11.5, 11.7, 11.8**

- [ ] 10. Checkpoint - Verify keyboard navigation and accessibility
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 11. Implement visual design and theming
  - [ ] 11.1 Apply consistent spacing and styling
    - Implement 8px/16px/24px spacing grid across all panels
    - Add visual separators (borders/shadows) between panels
    - Apply consistent typography hierarchy
    - _Requirements: 12.2, 12.3, 12.5_
  
  - [ ] 11.2 Implement hover states
    - Add hover styles to all interactive elements
    - Ensure hover states are visually distinct
    - _Requirements: 12.6_
  
  - [ ] 11.3 Add theme support
    - Ensure all panels support light and dark themes
    - Verify no hardcoded colors that don't respect theme
    - Test theme switching with new layout
    - _Requirements: 12.7_
  
  - [ ] 11.4 Add resize animations
    - Implement smooth transitions for panel resizing (200-300ms)
    - Use CSS transforms for data browser slide animations
    - Ensure 60fps animation performance
    - _Requirements: 12.8, 13.7_
  
  - [ ]* 11.5 Write property tests for theming
    - **Property 28: Theme consistency**
    - **Property 29: Hover state presence**
    - **Validates: Requirements 12.6, 12.7**

- [ ] 12. Implement performance optimizations
  - [ ] 12.1 Add list virtualization
    - Implement virtualization for HistoricalSessionsSection (>50 items)
    - Implement virtualization for CenterPanel message list (>50 messages)
    - Ensure only visible items plus buffer are rendered
    - _Requirements: 13.2, 13.3_
  
  - [ ] 12.2 Implement lazy loading for data browser
    - Load table list only when data browser opens
    - Load table data only when table is selected
    - Implement pagination for large data sets
    - _Requirements: 13.6_
  
  - [ ] 12.3 Add resize debouncing
    - Debounce resize operations to max 60fps (16ms)
    - Prevent excessive re-renders during resize
    - _Requirements: 13.5_
  
  - [ ]* 12.4 Write property tests for performance
    - **Property 25: Resize debouncing**
    - **Property 26: List virtualization**
    - **Property 27: Lazy loading**
    - **Validates: Requirements 13.2, 13.3, 13.5, 13.6**

- [ ] 13. Clean up and remove old code
  - [ ] 13.1 Remove unused components and code
    - Remove ChatSidebar overlay-related code
    - Remove ContextPanel component
    - Remove collapse/expand button logic
    - Remove unused state variables
    - Remove unused CSS classes
    - _Requirements: 10.7_
  
  - [ ] 13.2 Update CSS and styling
    - Remove overlay-related CSS
    - Remove collapse animation CSS
    - Add new three-panel layout CSS
    - Ensure responsive design works correctly
    - _Requirements: 12.1, 12.2, 12.3_

- [ ] 14. Integration testing and polish
  - [ ]* 14.1 Write integration tests for complete workflows
    - Test complete session creation workflow
    - Test data browser workflow
    - Test panel resizing workflow
    - Test session switching workflow
    - Test keyboard navigation workflow
    - _Requirements: All requirements_
  
  - [ ] 14.2 Test with various window sizes
    - Test minimum window size (1024px)
    - Test maximum window size (3840px)
    - Test responsive behavior
    - Test minimum width warning message
    - _Requirements: 9.6_
  
  - [ ] 14.3 Test edge cases
    - Test with empty data (no sources, no sessions)
    - Test with single items
    - Test with very long lists (100+ items)
    - Test with very long text
    - Test rapid interactions
    - Test concurrent operations
    - _Requirements: 2.6, 3.8, 5.4, 6.4_
  
  - [ ] 14.4 Accessibility testing
    - Test with screen readers (NVDA, JAWS, VoiceOver)
    - Test keyboard-only navigation
    - Verify focus indicators are visible
    - Check color contrast (WCAG AA)
    - Verify ARIA attributes are correct
    - _Requirements: 11.1-11.8_
  
  - [ ] 14.5 Browser compatibility testing
    - Test on Chrome (latest)
    - Test on Firefox (latest)
    - Test on Safari (latest)
    - Test on Edge (latest)
    - _Requirements: All requirements_

- [ ] 15. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional property-based tests and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Integration tests validate end-to-end workflows
- The implementation follows a phased approach: create components → update layout → migrate state → clean up → polish
