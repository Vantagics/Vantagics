# Implementation Plan: Dashboard Image Display Fix

## Overview

This implementation plan breaks down the image display fix into discrete, incremental tasks. Each task builds on previous work and includes integrated testing to catch issues early. The plan follows a bottom-up approach: backend detection → frontend reception → data conversion → component rendering → integration testing.

## Tasks

- [x] 1. Backend Image Detection Implementation
  - [x] 1.1 Create image detection patterns and regex validators
    - Implement regex patterns for base64 images: `data:image/[type];base64,[data]`
    - Implement regex patterns for markdown images: `![alt](path)`
    - Implement regex patterns for file references: `files/[filename]`
    - Create utility functions to validate detected patterns
    - _Requirements: 1.1_
  
  - [ ]* 1.2 Write property tests for image detection
    - **Property 1: Image Detection Completeness**
    - **Validates: Requirements 1.1, 1.4**
  
  - [x] 1.3 Implement image extraction and event emission
    - Create `detectAndEmitImages()` function in app.go
    - Extract image data from detected patterns
    - Emit `dashboard-update` events with correct payload structure
    - Handle multiple images by emitting separate events
    - Add logging for each detected image
    - _Requirements: 1.2, 1.3, 1.4, 1.5_
  
  - [ ]* 1.4 Write property tests for event emission
    - **Property 2: Event Payload Correctness**
    - **Validates: Requirements 1.2, 1.3**

- [x] 2. Frontend Event Reception and State Management
  - [x] 2.1 Implement dashboard-update event listener
    - Add EventsOn('dashboard-update') listener in App.tsx
    - Extract payload and validate structure
    - Log received events for debugging
    - _Requirements: 2.1_
  
  - [ ]* 2.2 Write property tests for event reception
    - **Property 3: Session-Specific Event Filtering**
    - **Validates: Requirements 2.2, 5.3**
  
  - [x] 2.3 Implement session ID validation and filtering
    - Validate sessionId matches activeSessionId before updating
    - Silently ignore events from other sessions
    - Handle case where sessionId is not provided (fallback behavior)
    - _Requirements: 2.2_
  
  - [x] 2.4 Implement activeChart state update
    - Update activeChart with type='image' and image data
    - Trigger component re-render on state change
    - Maintain backward compatibility with existing chart types
    - _Requirements: 2.3, 2.4_
  
  - [ ]* 2.5 Write property tests for state management
    - **Property 4: Image Data State Update**
    - **Validates: Requirements 2.3, 2.4**

- [x] 3. Image Data Format Conversion
  - [x] 3.1 Implement image format detection
    - Detect base64 strings (with or without data URL prefix)
    - Detect HTTP/HTTPS URLs
    - Detect file paths (file://, files/, relative paths)
    - Detect unknown formats and handle gracefully
    - _Requirements: 4.1, 4.2, 4.3, 4.4_
  
  - [x] 3.2 Implement base64 to data URL conversion
    - Convert base64 strings to proper data URLs
    - Detect MIME type from base64 data or use default (image/png)
    - Handle edge cases: empty strings, invalid base64
    - _Requirements: 4.1_
  
  - [x] 3.3 Implement file path to base64 conversion
    - Extract filename from file paths (file://, files/, relative)
    - Call GetSessionFileAsBase64 API with threadId and filename
    - Handle API errors and timeouts
    - Cache results to avoid repeated API calls
    - _Requirements: 4.2, 2.5_
  
  - [x] 3.4 Implement HTTP URL pass-through
    - Validate HTTP/HTTPS URLs
    - Use URLs directly without conversion
    - Handle CORS issues gracefully
    - _Requirements: 4.3_
  
  - [ ]* 3.5 Write property tests for format conversion
    - **Property 5: Image Format Conversion Round-Trip**
    - **Validates: Requirements 4.1, 4.2, 4.3**
  
  - [x] 3.6 Implement error handling for format conversion
    - Log errors with details for debugging
    - Return error state instead of throwing
    - Display user-friendly error messages
    - _Requirements: 4.4, 4.5_

- [x] 4. Dashboard Image Component Rendering
  - [x] 4.1 Implement image display rendering
    - Render image from activeChart.data
    - Support double-click to open full-screen modal
    - Add hover effects for interactivity
    - Handle image loading states
    - _Requirements: 3.1, 3.2, 3.3_
  
  - [x] 4.2 Implement empty state rendering
    - Display placeholder when no image data available
    - Show helpful message in edit mode
    - Hide component in view mode when no data
    - _Requirements: 3.4, 6.2, 6.4_
  
  - [x] 4.3 Implement error state rendering
    - Display error message when image fails to load
    - Show image source URL for debugging
    - Provide user-friendly error text
    - _Requirements: 3.5, 4.5_
  
  - [ ]* 4.4 Write property tests for component rendering
    - **Property 6: Component Visibility Based on Data**
    - **Validates: Requirements 3.1, 6.1, 6.4**
  
  - [x] 4.5 Implement modal for full-screen image view
    - Create ImageModal component if not exists
    - Open modal on double-click
    - Close modal on escape key or close button
    - Display image at full resolution
    - _Requirements: 3.3_

- [x] 5. Session Management Integration
  - [x] 5.1 Implement session-specific image storage
    - Store images per session in sessionCharts map
    - Update sessionCharts when new images received
    - Clear images when session is deleted
    - _Requirements: 5.1, 5.2_
  
  - [x] 5.2 Implement session switching logic
    - Update activeChart when activeSessionId changes
    - Load correct image for new session
    - Clear previous session's image
    - _Requirements: 5.1, 5.4_
  
  - [ ]* 5.3 Write property tests for session management
    - **Property 7: Session Isolation**
    - **Validates: Requirements 5.1, 5.2, 5.4**
  
  - [x] 5.4 Implement empty state for sessions without images
    - Display empty state when session has no image
    - Show in edit mode, hide in view mode
    - _Requirements: 5.5, 6.4_

- [x] 6. Dashboard Layout Integration
  - [x] 6.1 Ensure image component visibility in view mode
    - Image component visible when activeChart.type='image'
    - Image component hidden when no data in view mode
    - Proper spacing and alignment with other components
    - _Requirements: 6.1, 6.3_
  
  - [x] 6.2 Implement edit mode controls
    - Show placeholder with edit controls in edit mode
    - Allow component removal in edit mode
    - Show component even without data in edit mode
    - _Requirements: 6.2_
  
  - [x] 6.3 Implement layout persistence
    - Save image component position and size
    - Restore layout on reload
    - Preserve layout across sessions
    - _Requirements: 6.5_
  
  - [ ]* 6.4 Write property tests for layout integration
    - **Property 9: Layout Persistence**
    - **Validates: Requirements 6.5**

- [x] 7. Error Handling and Logging
  - [x] 7.1 Implement comprehensive error handling
    - Backend: Handle invalid image formats, event emission failures
    - Frontend: Handle event reception failures, session mismatches
    - Component: Handle image loading failures, format errors
    - _Requirements: 4.4, 4.5_
  
  - [x] 7.2 Implement detailed logging
    - Log image detection with "[CHART] Detected..." messages
    - Log event emissions with payload details
    - Log state updates and session changes
    - Log errors with full context for debugging
    - _Requirements: 1.5_
  
  - [ ]* 7.3 Write property tests for error handling
    - **Property 8: Error State Display**
    - **Validates: Requirements 3.5, 4.5**

- [x] 8. Integration Testing
  - [x] 8.1 Test end-to-end image display flow
    - Create analysis response with image
    - Verify backend detects and emits event
    - Verify frontend receives and updates state
    - Verify component renders image correctly
    - _Requirements: 1.1, 2.1, 3.1_
  
  - [x] 8.2 Test multiple image handling
    - Create response with multiple images
    - Verify all images are detected and emitted
    - Verify last image is displayed (or implement carousel)
    - _Requirements: 1.4_
  
  - [x] 8.3 Test session switching with images
    - Create multiple sessions with different images
    - Switch between sessions
    - Verify correct image displays for each session
    - _Requirements: 5.1, 5.2, 5.4_
  
  - [x] 8.4 Test error scenarios
    - Invalid image formats
    - Missing files
    - Network errors
    - Verify error states display correctly
    - _Requirements: 3.5, 4.5_
  
  - [ ]* 8.5 Write integration property tests
    - **Property 10: Empty State Handling**
    - **Validates: Requirements 3.4, 5.5, 6.2, 6.4**

- [x] 9. Checkpoint - Ensure all tests pass
  - Ensure all unit tests pass
  - Ensure all property tests pass (minimum 100 iterations each)
  - Ensure integration tests pass
  - Verify no regressions in existing functionality
  - Ask the user if questions arise

- [x] 10. Code Review and Documentation
  - [x] 10.1 Review backend image detection code
    - Verify regex patterns are correct and efficient
    - Check error handling and logging
    - Verify event payload structure
  
  - [x] 10.2 Review frontend event handling code
    - Verify event listener setup and cleanup
    - Check session ID validation logic
    - Verify state updates are correct
  
  - [x] 10.3 Review image component code
    - Verify rendering logic for all states
    - Check accessibility (alt text, keyboard navigation)
    - Verify modal functionality
  
  - [x] 10.4 Update documentation
    - Add comments explaining image detection patterns
    - Document event payload structure
    - Add troubleshooting guide for image display issues

- [ ] 11. Property-Based Testing Implementation
  - [ ] 11.1 Write property tests for image detection
    - **Property 1: Image Detection Completeness**
    - Generate random analysis responses with 0-5 images in various formats
    - Verify all images are detected and events emitted
    - Minimum 100 iterations
    - **Validates: Requirements 1.1, 1.4**
  
  - [ ] 11.2 Write property tests for event payload correctness
    - **Property 2: Event Payload Correctness**
    - Generate random images and verify event payloads contain required fields
    - Verify payload structure matches specification
    - Minimum 100 iterations
    - **Validates: Requirements 1.2, 1.3**
  
  - [ ] 11.3 Write property tests for session-specific event filtering
    - **Property 3: Session-Specific Event Filtering**
    - Generate events for multiple sessions
    - Verify only matching session updates state
    - Minimum 100 iterations
    - **Validates: Requirements 2.2, 5.3**
  
  - [ ] 11.4 Write property tests for image data state update
    - **Property 4: Image Data State Update**
    - Generate random image data in supported formats
    - Verify activeChart state is updated correctly
    - Minimum 100 iterations
    - **Validates: Requirements 2.3, 2.4**
  
  - [ ] 11.5 Write property tests for image format conversion
    - **Property 5: Image Format Conversion Round-Trip**
    - Generate random images in each format
    - Convert and verify visual equivalence
    - Minimum 100 iterations
    - **Validates: Requirements 4.1, 4.2, 4.3**
  
  - [ ] 11.6 Write property tests for component visibility
    - **Property 6: Component Visibility Based on Data**
    - Generate random combinations of activeChart states
    - Verify component visibility matches specification
    - Minimum 100 iterations
    - **Validates: Requirements 3.1, 6.1, 6.4**
  
  - [ ] 11.7 Write property tests for session isolation
    - **Property 7: Session Isolation**
    - Generate multiple sessions with different images
    - Switch between sessions and verify correct image displays
    - Minimum 100 iterations
    - **Validates: Requirements 5.1, 5.2, 5.4**
  
  - [ ] 11.8 Write property tests for error state display
    - **Property 8: Error State Display**
    - Generate invalid image URLs and formats
    - Verify error states display correctly
    - Minimum 100 iterations
    - **Validates: Requirements 3.5, 4.5**
  
  - [ ] 11.9 Write property tests for layout persistence
    - **Property 9: Layout Persistence**
    - Generate random layout configurations
    - Save and reload, verify preservation
    - Minimum 100 iterations
    - **Validates: Requirements 6.5**
  
  - [ ] 11.10 Write property tests for empty state handling
    - **Property 10: Empty State Handling**
    - Generate sessions without image data
    - Verify empty state displays in edit mode, hidden in view mode
    - Minimum 100 iterations
    - **Validates: Requirements 3.4, 5.5, 6.2, 6.4**

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Property tests should run minimum 100 iterations each
- Integration tests should cover happy path and error scenarios
- All code should include error handling and logging
- Backward compatibility must be maintained with existing chart types

## Implementation Status Summary

**Completed (10 sections):**
- Backend image detection with regex patterns for base64, markdown, and file references
- Image extraction and event emission via `detectAndEmitImages()` in app.go
- Frontend event listener with session ID validation and filtering
- Image data format conversion (base64, file paths, HTTP URLs)
- Dashboard image component rendering with empty/error/loading states
- Session-specific image storage and switching logic
- Dashboard layout integration with edit mode controls
- Comprehensive error handling and logging
- End-to-end integration testing
- Code review and documentation

**Remaining (1 section):**
- Property-Based Testing Implementation (10 properties across 10 tasks)
  - These are optional but recommended for formal correctness verification
  - Each property test should run minimum 100 iterations
  - Tests validate universal properties across all valid inputs

