# Requirements Document: Dashboard Image Display Fix

## Introduction

When users perform analysis requests, the backend detects images in the response and sends them via `dashboard-update` events. However, these images are not appearing in the dashboard's image display area on the right side of the interface. The images may appear in chat messages but fail to render in the dedicated dashboard image component.

This spec addresses the end-to-end image display pipeline from backend detection through frontend rendering.

## Glossary

- **Dashboard**: The right-side panel displaying metrics, insights, charts, and images
- **Image Component**: The dedicated area in the dashboard for displaying images
- **activeChart**: Frontend state object containing the current chart/image data with type and data properties
- **dashboard-update**: Backend event emitted when images are detected in analysis responses
- **Image Data**: Base64-encoded image data or file paths returned from analysis
- **Session**: A chat thread/conversation session with a unique ID

## Requirements

### Requirement 1: Backend Image Detection and Transmission

**User Story:** As a backend system, I want to detect images in analysis responses and transmit them to the frontend, so that users can see visual results of their analysis.

#### Acceptance Criteria

1. WHEN an analysis response contains an image (base64 or markdown format), THE Backend SHALL detect it and extract the image data
2. WHEN an image is detected, THE Backend SHALL emit a `dashboard-update` event with the image data and session ID
3. WHEN emitting the dashboard-update event, THE Backend SHALL include: sessionId, type='image', and data (base64 or file path)
4. IF multiple images are detected in a single response, THE Backend SHALL emit separate events for each image
5. WHEN an image is detected, THE Backend SHALL log the detection with "[CHART] Detected inline base64 image" or similar message

### Requirement 2: Frontend Event Reception and State Management

**User Story:** As a frontend application, I want to receive image data from the backend and update the dashboard state, so that images are available for display.

#### Acceptance Criteria

1. WHEN a `dashboard-update` event is received with type='image', THE Frontend SHALL extract the image data from the event payload
2. WHEN the event includes a sessionId, THE Frontend SHALL verify it matches the current active session before updating
3. WHEN image data is received, THE Frontend SHALL update the `activeChart` state with: type='image', data=<image_data>
4. WHEN activeChart is updated with image data, THE Frontend SHALL trigger a re-render of the dashboard component
5. IF the image data is a file path, THE Frontend SHALL convert it to a base64 data URL before storing in activeChart

### Requirement 3: Dashboard Image Component Rendering

**User Story:** As a dashboard component, I want to display images from the activeChart state, so that users can see analysis results.

#### Acceptance Criteria

1. WHEN the dashboard is rendered and activeChart.type='image', THE Image Component SHALL be visible in the dashboard
2. WHEN activeChart contains image data, THE Image Component SHALL display the image using the data URL
3. WHEN an image is displayed, THE Image Component SHALL support double-click to open a full-screen modal
4. WHEN no image data is available, THE Image Component SHALL display an empty state placeholder
5. WHEN an image fails to load, THE Image Component SHALL display an error state with the image source for debugging

### Requirement 4: Image Data Format Handling

**User Story:** As the image display system, I want to handle various image data formats, so that images from different sources can be displayed correctly.

#### Acceptance Criteria

1. WHEN image data is a base64 string, THE System SHALL convert it to a data URL with appropriate MIME type
2. WHEN image data is a file path (file:// or relative path), THE System SHALL load it via the GetSessionFileAsBase64 API
3. WHEN image data is an HTTP URL, THE System SHALL use it directly without conversion
4. WHEN image data format is unknown, THE System SHALL attempt to detect the format and handle it appropriately
5. WHEN image loading fails, THE System SHALL log the error and display a user-friendly error message

### Requirement 5: Session-Specific Image Display

**User Story:** As a multi-session system, I want to display images specific to the current session, so that users see correct results when switching between sessions.

#### Acceptance Criteria

1. WHEN a user switches to a different session, THE Dashboard SHALL clear the previous session's image
2. WHEN a new session is selected, THE Dashboard SHALL display only images from that session
3. WHEN a `dashboard-update` event is received for a different session, THE Dashboard SHALL not update if another session is active
4. WHEN the activeSessionId changes, THE Dashboard SHALL update activeChart to show the new session's image
5. WHEN a session has no image data, THE Dashboard SHALL display an empty state

### Requirement 6: Image Display Area Integration

**User Story:** As a dashboard layout system, I want to integrate the image display area with other dashboard components, so that images are properly positioned and visible.

#### Acceptance Criteria

1. WHEN the dashboard is in view mode, THE Image Component SHALL be visible if image data exists
2. WHEN the dashboard is in edit mode, THE Image Component SHALL show a placeholder with edit controls
3. WHEN the image component has data, THE Component SHALL occupy the designated image display area
4. WHEN the image component has no data, THE Component SHALL be hidden in view mode but visible in edit mode
5. WHEN the dashboard layout is saved, THE Image Component position and size SHALL be preserved

## Notes

- Images may come from multiple sources: inline base64 in responses, generated chart.png files, or markdown image references
- The system must handle both synchronous image detection and asynchronous file loading
- Error handling is critical for user experience - failed images should not break the dashboard
- Session management is important for multi-threaded analysis scenarios
