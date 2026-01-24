# Design Document: Dashboard Image Display Fix

## Overview

This design addresses the end-to-end image display pipeline for the dashboard. The system detects images in analysis responses on the backend, transmits them via events to the frontend, and renders them in the dedicated dashboard image component. The solution handles multiple image formats (base64, file paths, HTTP URLs), manages session-specific images, and provides proper error handling and user feedback.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Analysis Response                            │
│                  (contains image data)                           │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│              Backend Image Detection (app.go)                    │
│  • Regex patterns for base64 images                             │
│  • Markdown image reference detection                           │
│  • File path extraction                                         │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│         Event Emission (dashboard-update)                        │
│  • Payload: { sessionId, type: 'image', data: <image> }        │
│  • Separate event per image                                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│           Frontend Event Reception (App.tsx)                     │
│  • EventsOn('dashboard-update') listener                        │
│  • Session ID validation                                        │
│  • State update: setActiveChart()                               │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│        Image Data Format Conversion                              │
│  • Base64 → data URL with MIME type                             │
│  • File path → GetSessionFileAsBase64 API call                  │
│  • HTTP URL → use directly                                      │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│      Dashboard Image Component Rendering                         │
│  • Display image from activeChart.data                          │
│  • Show empty state if no data                                  │
│  • Show error state if loading fails                            │
│  • Support double-click for full-screen modal                   │
└─────────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Backend Image Detection (app.go)

**Responsibility**: Detect images in analysis responses and emit events

**Key Functions**:
- `detectAndEmitImages(response string, threadID string)` - Main detection function
- Pattern matching for:
  - Base64 images: `data:image/[type];base64,[data]`
  - Markdown images: `![alt](path)`
  - File references: `files/[filename]`

**Event Payload Structure**:
```go
{
  "sessionId": "thread-123",
  "type": "image",
  "data": "data:image/png;base64,iVBORw0KGgo..." // or file path
}
```

### 2. Frontend Event Handler (App.tsx)

**Responsibility**: Receive events and update dashboard state

**Key Logic**:
```typescript
EventsOn("dashboard-update", (payload: any) => {
  if (payload.type === 'image') {
    // Validate session ID
    if (payload.sessionId === activeSessionId) {
      // Convert image data if needed
      const imageData = await convertImageData(payload.data);
      // Update state
      setActiveChart({
        type: 'image',
        data: imageData
      });
    }
  }
});
```

**State Management**:
- `activeChart`: Current image/chart data
- `activeSessionId`: Current session ID
- `sessionCharts`: Map of session ID to chart data

### 3. Image Data Converter

**Responsibility**: Convert various image formats to displayable format

**Conversion Logic**:
```typescript
async function convertImageData(data: string): Promise<string> {
  // Already a data URL
  if (data.startsWith('data:')) {
    return data;
  }
  
  // HTTP URL
  if (data.startsWith('http://') || data.startsWith('https://')) {
    return data;
  }
  
  // File path - load via API
  if (data.startsWith('file://') || data.startsWith('files/')) {
    return await GetSessionFileAsBase64(threadId, extractFilename(data));
  }
  
  // Base64 string without data URL prefix
  if (isBase64(data)) {
    return `data:image/png;base64,${data}`;
  }
  
  throw new Error(`Unknown image format: ${data}`);
}
```

### 4. Dashboard Image Component (DraggableDashboard.tsx)

**Responsibility**: Render images in the dashboard

**Rendering States**:
- **Empty State**: No image data available
- **Loading State**: Image is being loaded
- **Error State**: Image failed to load
- **Display State**: Image successfully loaded and displayed

**Features**:
- Double-click to open full-screen modal
- Hover effects for interactivity
- Error messages with image source for debugging
- Edit mode controls for layout management

## Data Models

### Image Data Structure
```typescript
interface ImageData {
  type: 'image';
  data: string; // data URL, HTTP URL, or file path
  chartData?: ChartData; // optional additional chart data
}

interface ActiveChart {
  type: 'echarts' | 'image' | 'table' | 'csv';
  data: any;
  chartData?: ChartData;
}
```

### Event Payload
```typescript
interface DashboardUpdatePayload {
  sessionId: string;
  type: 'image' | 'echarts' | 'table' | 'csv';
  data: any;
  chartData?: ChartData;
}
```

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Image Detection Completeness
*For any* analysis response containing images in supported formats (base64, markdown, file paths), the backend SHALL detect all images and emit corresponding events.
**Validates: Requirements 1.1, 1.4**

### Property 2: Event Payload Correctness
*For any* detected image, the emitted `dashboard-update` event SHALL contain all required fields: sessionId, type='image', and data.
**Validates: Requirements 1.2, 1.3**

### Property 3: Session-Specific Event Filtering
*For any* `dashboard-update` event with a sessionId, the frontend SHALL only update activeChart if the sessionId matches the current activeSessionId.
**Validates: Requirements 2.2, 5.3**

### Property 4: Image Data State Update
*For any* valid image data received in a dashboard-update event, the activeChart state SHALL be updated with type='image' and the image data.
**Validates: Requirements 2.3, 2.4**

### Property 5: Image Format Conversion Round-Trip
*For any* image data in supported formats (base64, file path, HTTP URL), converting to displayable format and then displaying SHALL result in the same visual image.
**Validates: Requirements 4.1, 4.2, 4.3**

### Property 6: Component Visibility Based on Data
*For any* dashboard state, the Image Component SHALL be visible if and only if activeChart.type='image' and activeChart.data is not null/empty.
**Validates: Requirements 3.1, 6.1, 6.4**

### Property 7: Session Isolation
*For any* two different sessions with different images, switching between sessions SHALL display the correct image for each session without mixing data.
**Validates: Requirements 5.1, 5.2, 5.4**

### Property 8: Error State Display
*For any* image that fails to load, the Image Component SHALL display an error state with the image source visible for debugging purposes.
**Validates: Requirements 3.5, 4.5**

### Property 9: Layout Persistence
*For any* saved dashboard layout, the Image Component's position and size SHALL be preserved when the layout is reloaded.
**Validates: Requirements 6.5**

### Property 10: Empty State Handling
*For any* session without image data, the Image Component SHALL display an empty state placeholder in edit mode and be hidden in view mode.
**Validates: Requirements 3.4, 5.5, 6.2, 6.4**

## Error Handling

### Backend Error Handling
- **Invalid Image Format**: Log warning, skip image, continue processing
- **Event Emission Failure**: Log error, attempt retry with exponential backoff
- **Multiple Images**: Process each independently, emit separate events

### Frontend Error Handling
- **Event Reception Failure**: Log error, maintain current state
- **Session ID Mismatch**: Silently ignore event (expected behavior)
- **Image Loading Failure**: Display error state with source URL
- **Format Conversion Failure**: Display error state with format details
- **API Call Failure** (GetSessionFileAsBase64): Display error state, log error

### User-Facing Error Messages
- "Failed to load image" - Generic loading failure
- "Image format not supported" - Unsupported format
- "Session not found" - Session ID mismatch
- "File not found" - File path doesn't exist

## Testing Strategy

### Unit Tests

**Backend Image Detection**:
- Test detection of base64 images in various formats
- Test detection of markdown image references
- Test detection of file path references
- Test handling of multiple images in single response
- Test edge cases: empty images, malformed data, special characters

**Frontend Event Handling**:
- Test event reception and payload extraction
- Test session ID validation and filtering
- Test state updates with various image formats
- Test error handling for invalid payloads

**Image Data Conversion**:
- Test base64 to data URL conversion
- Test file path to base64 conversion (mock API)
- Test HTTP URL pass-through
- Test format detection and error handling

**Image Component Rendering**:
- Test rendering with valid image data
- Test rendering with no data (empty state)
- Test rendering with invalid data (error state)
- Test modal open/close on double-click
- Test layout preservation

### Property-Based Tests

**Property 1: Image Detection Completeness**
- Generate random analysis responses with 0-5 images in various formats
- Verify all images are detected and events emitted
- Minimum 100 iterations

**Property 2: Event Payload Correctness**
- Generate random images and verify event payloads contain required fields
- Verify payload structure matches specification
- Minimum 100 iterations

**Property 3: Session-Specific Event Filtering**
- Generate events for multiple sessions
- Verify only matching session updates state
- Minimum 100 iterations

**Property 4: Image Data State Update**
- Generate random image data in supported formats
- Verify activeChart state is updated correctly
- Minimum 100 iterations

**Property 5: Image Format Conversion Round-Trip**
- Generate random images in each format
- Convert and verify visual equivalence
- Minimum 100 iterations

**Property 6: Component Visibility Based on Data**
- Generate random combinations of activeChart states
- Verify component visibility matches specification
- Minimum 100 iterations

**Property 7: Session Isolation**
- Generate multiple sessions with different images
- Switch between sessions and verify correct image displays
- Minimum 100 iterations

**Property 8: Error State Display**
- Generate invalid image URLs and formats
- Verify error states display correctly
- Minimum 100 iterations

**Property 9: Layout Persistence**
- Generate random layout configurations
- Save and reload, verify preservation
- Minimum 100 iterations

**Property 10: Empty State Handling**
- Generate sessions without image data
- Verify empty state displays in edit mode, hidden in view mode
- Minimum 100 iterations

## Implementation Notes

1. **Image Detection Patterns**: Use regex patterns that are robust to whitespace and encoding variations
2. **Event Ordering**: Ensure images are emitted in the order they appear in the response
3. **Performance**: Lazy-load images to avoid blocking the UI
4. **Caching**: Cache converted image data to avoid repeated conversions
5. **Logging**: Add detailed logging for debugging image display issues
6. **Accessibility**: Ensure images have alt text and are keyboard accessible
7. **Mobile Support**: Ensure images display correctly on mobile devices
