# Implementation Plan: Data Source Startup Overview

## Overview

This implementation plan breaks down the data source startup overview feature into discrete coding tasks. The approach follows a bottom-up strategy: backend API first, then frontend components, and finally integration. Each task builds on previous work to ensure incremental progress and early validation.

## Tasks

- [ ] 1. Implement backend statistics API
  - [x] 1.1 Add DataSourceStatistics and DataSourceSummary types to Go codebase
    - Create new types in src/agent/datasource_types.go or src/app.go
    - Define DataSourceStatistics struct with TotalCount, BreakdownByType, DataSources fields
    - Define DataSourceSummary struct with ID, Name, Type fields
    - Add JSON tags for proper serialization
    - _Requirements: 1.2, 1.3, 1.4_

  - [x] 1.2 Implement GetDataSourceStatistics method in App
    - Add method to src/app.go
    - Load data sources using existing dataSourceService.LoadDataSources()
    - Calculate total count (length of data sources array)
    - Build breakdown map by iterating and grouping by Type field
    - Build DataSourceSummary array for selection UI
    - Handle empty data source list (return zero count, empty breakdown)
    - Handle errors from LoadDataSources (wrap and return)
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [ ]* 1.3 Write property test for statistics calculation
    - **Property 1: Statistics Calculation Correctness**
    - Generate random lists of DataSource objects with various Type values
    - Call GetDataSourceStatistics and verify:
      - TotalCount equals length of input list
      - Sum of all BreakdownByType values equals TotalCount
      - Each Type in breakdown appears in input data sources
      - DataSources array length equals TotalCount
    - Test with empty list (edge case)
    - Run 100+ iterations
    - **Validates: Requirements 1.2, 1.3, 1.4**

  - [ ]* 1.4 Write unit tests for GetDataSourceStatistics
    - Test with empty data source list (returns zero count, empty breakdown)
    - Test with single data source (returns count=1, breakdown with one entry)
    - Test with multiple data sources of same type (correct grouping)
    - Test with multiple data sources of different types (correct breakdown)
    - Test error handling when dataSourceService is nil
    - Test error handling when LoadDataSources fails
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 7.1, 7.2_

- [ ] 2. Implement backend analysis initiation API
  - [x] 2.1 Implement StartDataSourceAnalysis method in App
    - Add method to src/app.go
    - Validate dataSourceService is initialized
    - Load data sources to verify target exists
    - Return error if data source ID not found
    - Generate unique thread ID (format: "ds-analysis-{id}-{timestamp}")
    - Construct analysis prompt in Chinese (mention data source name and type)
    - Call existing SendMessage with thread ID and prompt
    - Log analysis initiation
    - Return thread ID on success
    - _Requirements: 4.1, 4.2, 4.5_

  - [ ]* 2.2 Write property test for analysis initiation
    - **Property 5: Analysis Initiation Correctness**
    - Generate random valid data source IDs from loaded data sources
    - Call StartDataSourceAnalysis and verify:
      - Returns non-empty thread ID on success
      - Thread ID follows expected format
      - Returns error for non-existent IDs
    - Run 100+ iterations
    - **Validates: Requirements 4.1, 4.2**

  - [ ]* 2.3 Write unit tests for StartDataSourceAnalysis
    - Test with valid data source ID (returns thread ID)
    - Test with invalid data source ID (returns error)
    - Test error handling when dataSourceService is nil
    - Test error handling when LoadDataSources fails
    - Test error handling when SendMessage fails
    - Verify thread ID format
    - Verify logging occurs
    - _Requirements: 4.1, 4.2, 4.5, 7.1, 7.2_

- [ ] 3. Checkpoint - Backend API complete
  - Ensure all backend tests pass
  - Verify Wails bindings are generated correctly
  - Test API methods manually using Wails dev tools
  - Ask the user if questions arise

- [ ] 4. Implement frontend data source overview component
  - [x] 4.1 Create DataSourceOverview component
    - Create src/frontend/src/components/DataSourceOverview.tsx
    - Define TypeScript interfaces (DataSourceStatistics, DataSourceSummary)
    - Implement component with state for statistics, loading, error
    - Add useEffect to fetch statistics on mount
    - Call GetDataSourceStatistics from Wails bindings
    - Implement loading state UI (spinner + text)
    - Implement error state UI (error message + retry button)
    - Implement empty state UI (no data sources message)
    - Implement statistics display (total count + breakdown list)
    - Add CSS classes for styling
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 5.1_

  - [ ]* 4.2 Write property test for statistics rendering
    - **Property 2: Statistics Rendering Completeness**
    - Generate random DataSourceStatistics objects with various breakdowns
    - Render DataSourceOverview with test data
    - Verify:
      - Total count is displayed
      - All breakdown entries are rendered
      - Each entry shows type name and count
      - Correct number of breakdown items
    - Run 100+ iterations
    - **Validates: Requirements 2.2, 2.3, 2.4**

  - [ ]* 4.3 Write unit tests for DataSourceOverview
    - Test loading state renders spinner and text
    - Test error state renders error message and retry button
    - Test retry button calls loadStatistics again
    - Test empty state renders "no data sources" message
    - Test statistics display with single data source
    - Test statistics display with multiple types
    - Test component calls GetDataSourceStatistics on mount
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

- [ ] 5. Implement smart insight for one-click analysis
  - [x] 5.1 Create DataSourceAnalysisInsight component
    - Create src/frontend/src/components/DataSourceAnalysisInsight.tsx
    - Accept statistics and onAnalyzeClick props
    - Implement state for showSelection and analyzing
    - Implement handleAnalyzeClick:
      - If multiple sources: show selection modal
      - If single source: call startAnalysis directly
    - Implement startAnalysis function:
      - Call StartDataSourceAnalysis from Wails bindings
      - Handle success (notify parent, navigate, or update UI)
      - Handle errors (show alert with message)
      - Update analyzing state
    - Generate insight text based on data source count
    - Render SmartInsight component with text, icon, onClick
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 4.1, 4.2, 4.3, 4.4, 4.5_

  - [ ]* 5.2 Write property test for smart insight structure
    - **Property 3: Smart Insight Structure Completeness**
    - Generate random DataSourceStatistics with varying counts
    - Render DataSourceAnalysisInsight
    - Verify:
      - Insight text mentions data sources
      - Insight text reflects actual count
      - onClick handler is defined
      - Icon is set
    - Run 100+ iterations
    - **Validates: Requirements 3.1, 3.2, 3.3**

  - [ ]* 5.3 Write unit tests for DataSourceAnalysisInsight
    - Test with single data source (no modal shown, direct analysis)
    - Test with multiple data sources (modal shown)
    - Test handleAnalyzeClick with single source calls startAnalysis
    - Test handleAnalyzeClick with multiple sources sets showSelection
    - Test startAnalysis success flow
    - Test startAnalysis error handling (shows alert)
    - Test analyzing state updates correctly
    - Test insight text generation for different counts
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 4.1, 4.2, 4.3, 4.4, 4.5_

- [ ] 6. Implement data source selection modal
  - [x] 6.1 Create DataSourceSelectionModal component
    - Create src/frontend/src/components/DataSourceSelectionModal.tsx
    - Accept dataSources, onSelect, onCancel props
    - Render modal overlay with click-to-close
    - Render modal content with title
    - Render list of data sources (map over dataSources)
    - Each item shows name and type
    - Each item has onClick to call onSelect with ID
    - Render cancel button
    - Stop propagation on modal content click
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [ ]* 6.2 Write property test for selection UI completeness
    - **Property 4: Selection UI Completeness**
    - Generate random lists of DataSourceSummary (length > 1)
    - Render DataSourceSelectionModal
    - Verify:
      - All data sources are rendered
      - Each item displays name and type
      - Cancel button is present
      - Each item has click handler
    - Run 100+ iterations
    - **Validates: Requirements 6.1, 6.2, 6.4**

  - [ ]* 6.3 Write unit tests for DataSourceSelectionModal
    - Test renders all data sources from props
    - Test each data source shows name and type
    - Test clicking data source calls onSelect with correct ID
    - Test clicking cancel button calls onCancel
    - Test clicking overlay calls onCancel
    - Test clicking modal content doesn't close modal
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [ ] 7. Integrate components into main application
  - [x] 7.1 Add DataSourceOverview to App.tsx
    - Import DataSourceOverview component
    - Add component to main content area (prominent location)
    - Pass onAnalyzeClick handler
    - Implement handler to open chat sidebar or navigate to analysis
    - Ensure component loads on application startup
    - _Requirements: 5.1, 5.2, 5.3_

  - [ ]* 7.2 Write integration tests for startup flow
    - Test App.tsx renders DataSourceOverview on mount
    - Test DataSourceOverview fetches statistics on mount
    - Test clicking smart insight triggers analysis
    - Test analysis opens chat or navigates correctly
    - Test error states don't break app rendering
    - _Requirements: 5.1, 5.2, 5.3, 5.5, 7.3_

- [ ] 8. Add styling and polish
  - [x] 8.1 Create CSS for DataSourceOverview
    - Add styles to src/frontend/src/styles/ or component file
    - Style overview header, total count display
    - Style breakdown list (type name, count)
    - Style loading state (spinner, text)
    - Style error state (message, retry button)
    - Style empty state
    - Ensure responsive design
    - Match existing application design system
    - _Requirements: 2.2, 2.3, 2.4, 2.5, 2.6, 5.2_

  - [x] 8.2 Create CSS for DataSourceSelectionModal
    - Style modal overlay (semi-transparent background)
    - Style modal content (centered, white background, shadow)
    - Style data source list items (hover effects, cursor pointer)
    - Style name and type display
    - Style cancel button
    - Ensure modal is accessible (keyboard navigation, focus management)
    - _Requirements: 6.1, 6.2, 6.4_

- [ ] 9. Final checkpoint - End-to-end testing
  - Test complete flow: startup → statistics display → click insight → select source → analysis starts
  - Test with no data sources (empty state)
  - Test with single data source (direct analysis)
  - Test with multiple data sources (selection modal)
  - Test error scenarios (network failure, invalid data)
  - Test retry functionality
  - Verify logging and error messages
  - Ensure all tests pass
  - Ask the user if questions arise

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Integration tests validate end-to-end flows
- The implementation follows existing patterns in the codebase (Wails bindings, React components, Go services)
