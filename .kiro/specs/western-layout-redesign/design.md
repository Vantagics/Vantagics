# Design Document: Western Layout Redesign

## Overview

This design document describes the implementation approach for transforming the current overlay-based UI into a fixed three-panel layout optimized for Western user habits. The redesign replaces the collapsible/overlay pattern with a stable, always-visible three-panel structure: Left Panel (data sources + historical sessions), Center Panel (fixed chat area), and Right Panel (dashboard).

### Key Design Decisions

1. **Fixed Three-Panel Layout**: Eliminates overlays and collapsible panels for better spatial stability
2. **Slide-Out Data Browser**: Provides data exploration without disrupting the main workflow
3. **Unified Left Panel**: Combines data sources and historical sessions in a single vertical panel
4. **Persistent Center Chat**: Makes conversation context always visible
5. **Resizable Panels**: Allows users to customize their workspace proportions

## Architecture

### Component Hierarchy

```
App
├── LeftPanel (new)
│   ├── DataSourcesSection
│   │   ├── DataSourceList
│   │   └── AddDataSourceButton
│   ├── NewSessionButton (new)
│   └── HistoricalSessionsSection (new)
│       └── SessionList
├── ResizeHandle (between Left and Center)
├── CenterPanel (replaces ChatSidebar)
│   ├── ChatHeader
│   ├── MessageList
│   ├── MessageInput
│   └── DataBrowser (slide-out overlay, new)
│       ├── DataBrowserHeader
│       ├── TableList
│       ├── ColumnView
│       └── DataPreview
├── ResizeHandle (between Center and Right)
└── RightPanel (replaces current dashboard area)
    └── DraggableDashboard
        ├── MetricsCards
        ├── Charts
        ├── Insights
        └── FileDownloads
```

### Layout Structure

The layout uses CSS Flexbox for the main three-panel structure:

```
┌─────────────────────────────────────────────────────────────┐
│  App Container (display: flex, flex-direction: row)         │
│  ┌──────────┬────────────────────────┬──────────────────┐  │
│  │          │                        │                  │  │
│  │  Left    │  Center Panel          │  Right Panel     │  │
│  │  Panel   │  (Chat Area)           │  (Dashboard)     │  │
│  │          │                        │                  │  │
│  │  Data    │  ┌──────────────────┐  │  Metrics         │  │
│  │  Sources │  │ Chat Messages    │  │  Charts          │  │
│  │          │  │                  │  │  Insights        │  │
│  │  [New    │  │                  │  │  Files           │  │
│  │  Session]│  │                  │  │                  │  │
│  │          │  └──────────────────┘  │                  │  │
│  │  History │  Message Input         │                  │  │
│  │  Sessions│                        │                  │  │
│  │          │  [Data Browser Overlay]│                  │  │
│  └──────────┴────────────────────────┴──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. LeftPanel Component

**Purpose**: Displays data sources, new session button, and historical sessions in a unified vertical panel.

**Props**:
```typescript
interface LeftPanelProps {
  width: number;                    // Panel width in pixels
  onWidthChange: (width: number) => void;
  onDataSourceSelect: (sourceId: string) => void;
  onSessionSelect: (sessionId: string) => void;
  onNewSession: () => void;
  onBrowseData: (sourceId: string) => void;
  selectedDataSourceId: string | null;
  selectedSessionId: string | null;
}
```

**State**:
```typescript
interface LeftPanelState {
  dataSources: DataSource[];
  sessions: AnalysisSession[];
  isLoadingDataSources: boolean;
  isLoadingSessions: boolean;
  contextMenu: ContextMenuState | null;
}
```

**Key Methods**:
- `fetchDataSources()`: Loads data sources from backend
- `fetchSessions()`: Loads historical sessions from backend
- `handleDataSourceContextMenu(e, sourceId)`: Shows context menu with "Browse Data" option
- `handleSessionContextMenu(e, sessionId)`: Shows session context menu
- `handleNewSessionClick()`: Triggers new session creation dialog

### 2. DataSourcesSection Component

**Purpose**: Displays the list of data sources with add button.

**Props**:
```typescript
interface DataSourcesSectionProps {
  dataSources: DataSource[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onContextMenu: (e: React.MouseEvent, id: string) => void;
  onAdd: () => void;
}
```

**Rendering**:
- Header with "Data Sources" title and add button
- Scrollable list of data sources
- Each item shows: icon, name, type indicator
- Selected item highlighted
- Empty state when no data sources

### 3. HistoricalSessionsSection Component

**Purpose**: Displays the list of previous analysis sessions.

**Props**:
```typescript
interface HistoricalSessionsSectionProps {
  sessions: AnalysisSession[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onContextMenu: (e: React.MouseEvent, id: string) => void;
}
```

**Rendering**:
- Header with "Historical Sessions" title
- Scrollable list of sessions (virtualized for performance)
- Each item shows: session name, date, data source name
- Selected item highlighted
- Empty state when no sessions

### 4. NewSessionButton Component

**Purpose**: Provides a prominent button to create new analysis sessions.

**Props**:
```typescript
interface NewSessionButtonProps {
  onClick: () => void;
  disabled: boolean;
  selectedDataSourceName: string | null;
}
```

**Rendering**:
- Full-width button with icon and text
- Shows selected data source name if available
- Disabled state when no data source selected
- Tooltip explaining the requirement

### 5. CenterPanel Component

**Purpose**: Displays the fixed chat interface with message history and input.

**Props**:
```typescript
interface CenterPanelProps {
  width: number;
  sessionId: string | null;
  messages: Message[];
  isLoading: boolean;
  onSendMessage: (text: string) => void;
  onMessageClick: (messageId: string) => void;
  dataBrowserOpen: boolean;
  dataBrowserSourceId: string | null;
  onCloseBrowser: () => void;
}
```

**State**:
```typescript
interface CenterPanelState {
  inputText: string;
  isComposing: boolean;
  scrollPosition: number;
}
```

**Key Methods**:
- `handleSendMessage()`: Sends user message to backend
- `handleMessageClick(messageId)`: Loads analysis results for clicked message
- `scrollToBottom()`: Auto-scrolls to latest message
- `renderMessage(message)`: Renders individual message bubble

### 6. DataBrowser Component

**Purpose**: Slide-out panel for browsing data source contents.

**Props**:
```typescript
interface DataBrowserProps {
  isOpen: boolean;
  sourceId: string | null;
  onClose: () => void;
  width: number;
  onWidthChange: (width: number) => void;
}
```

**State**:
```typescript
interface DataBrowserState {
  tables: TableInfo[];
  selectedTable: string | null;
  columns: ColumnInfo[];
  dataRows: any[];
  currentPage: number;
  totalRows: number;
  isLoading: boolean;
  error: string | null;
}
```

**Key Methods**:
- `loadTables()`: Fetches table list from data source
- `loadTableData(tableName)`: Loads columns and sample data
- `handleTableSelect(tableName)`: Switches to selected table
- `handlePageChange(page)`: Loads different page of data
- `slideIn()`: Animates panel entrance
- `slideOut()`: Animates panel exit

**Animation**:
```css
.data-browser {
  position: absolute;
  right: 0;
  top: 0;
  height: 100%;
  background: white;
  box-shadow: -4px 0 12px rgba(0, 0, 0, 0.1);
  transform: translateX(100%);
  transition: transform 300ms cubic-bezier(0.4, 0, 0.2, 1);
}

.data-browser.open {
  transform: translateX(0);
}
```

### 7. RightPanel Component

**Purpose**: Displays the dashboard with analysis results.

**Props**:
```typescript
interface RightPanelProps {
  width: number;
  onWidthChange: (width: number) => void;
  dashboardData: DashboardData | null;
  activeChart: ChartData | null;
  sessionFiles: SessionFile[];
  selectedMessageId: string | null;
  onInsightClick: (insight: string) => void;
}
```

**Rendering**:
- Wraps existing DraggableDashboard component
- No major changes to dashboard functionality
- Maintains all existing metrics, charts, insights display

### 8. ResizeHandle Component

**Purpose**: Provides draggable handles for resizing panels.

**Props**:
```typescript
interface ResizeHandleProps {
  onDragStart: () => void;
  onDrag: (deltaX: number) => void;
  onDragEnd: () => void;
  orientation: 'vertical' | 'horizontal';
}
```

**Behavior**:
- Visual indicator on hover (color change, cursor change)
- Captures mouse events for dragging
- Emits delta values during drag
- Prevents text selection during drag
- Shows visual feedback during active drag

## Data Models

### AnalysisSession

```typescript
interface AnalysisSession {
  id: string;
  name: string;
  dataSourceId: string;
  dataSourceName: string;
  createdAt: number;
  updatedAt: number;
  messageCount: number;
  lastMessage: string;
}
```

### DataSource

```typescript
interface DataSource {
  id: string;
  name: string;
  type: 'excel' | 'mysql' | 'postgresql' | 'doris' | 'csv' | 'json';
  config: any;
  createdAt: number;
}
```

### TableInfo

```typescript
interface TableInfo {
  name: string;
  rowCount: number;
  columnCount: number;
  schema: string;
}
```

### ColumnInfo

```typescript
interface ColumnInfo {
  name: string;
  type: string;
  nullable: boolean;
  primaryKey: boolean;
}
```

### PanelWidths

```typescript
interface PanelWidths {
  left: number;
  center: number;  // Calculated as remaining space
  right: number;
}
```

## State Management

### App-Level State

The main App component manages the following state:

```typescript
interface AppState {
  // Panel dimensions
  panelWidths: PanelWidths;
  
  // Selection state
  selectedDataSourceId: string | null;
  selectedSessionId: string | null;
  selectedMessageId: string | null;
  
  // Data browser state
  dataBrowserOpen: boolean;
  dataBrowserSourceId: string | null;
  
  // Chat state
  activeSessionId: string | null;
  messages: Message[];
  isAnalysisLoading: boolean;
  
  // Dashboard state
  dashboardData: DashboardData | null;
  activeChart: ChartData | null;
  sessionFiles: SessionFile[];
}
```

### State Transitions

1. **Opening Data Browser**:
   ```
   User clicks "Browse Data" → 
   Set dataBrowserOpen = true →
   Set dataBrowserSourceId = sourceId →
   DataBrowser slides in over CenterPanel
   ```

2. **Closing Data Browser**:
   ```
   User clicks X button or presses Escape →
   Set dataBrowserOpen = false →
   DataBrowser slides out →
   CenterPanel becomes fully visible
   ```

3. **Selecting Session**:
   ```
   User clicks session in HistoricalSessionsSection →
   Set selectedSessionId = sessionId →
   Load session messages →
   Update CenterPanel with messages →
   Load session results →
   Update RightPanel with results
   ```

4. **Creating New Session**:
   ```
   User clicks NewSessionButton →
   Open NewSessionModal →
   User enters session name →
   Create session with selectedDataSourceId →
   Set selectedSessionId = newSessionId →
   Update CenterPanel to show new empty session
   ```

5. **Resizing Panels**:
   ```
   User drags ResizeHandle →
   Calculate new widths based on mouse position →
   Update panelWidths state →
   Panels re-render with new widths →
   On drag end, persist widths to localStorage
   ```

## Layout Calculations

### Panel Width Constraints

```typescript
const PANEL_CONSTRAINTS = {
  left: {
    min: 180,
    max: 400,
    default: 256
  },
  center: {
    min: 400,
    max: Infinity  // Fills remaining space
  },
  right: {
    min: 280,
    max: 600,
    default: 384
  }
};
```

### Width Calculation Logic

```typescript
function calculatePanelWidths(
  totalWidth: number,
  leftWidth: number,
  rightWidth: number
): PanelWidths {
  // Enforce constraints
  const constrainedLeft = Math.max(
    PANEL_CONSTRAINTS.left.min,
    Math.min(PANEL_CONSTRAINTS.left.max, leftWidth)
  );
  
  const constrainedRight = Math.max(
    PANEL_CONSTRAINTS.right.min,
    Math.min(PANEL_CONSTRAINTS.right.max, rightWidth)
  );
  
  // Calculate center width (remaining space)
  const centerWidth = totalWidth - constrainedLeft - constrainedRight;
  
  // Ensure center meets minimum
  if (centerWidth < PANEL_CONSTRAINTS.center.min) {
    // Reduce right panel to accommodate
    const adjustedRight = Math.max(
      PANEL_CONSTRAINTS.right.min,
      totalWidth - constrainedLeft - PANEL_CONSTRAINTS.center.min
    );
    
    return {
      left: constrainedLeft,
      center: totalWidth - constrainedLeft - adjustedRight,
      right: adjustedRight
    };
  }
  
  return {
    left: constrainedLeft,
    center: centerWidth,
    right: constrainedRight
  };
}
```

### Resize Handle Logic

```typescript
function handleResizeDrag(
  handlePosition: 'left' | 'right',
  deltaX: number,
  currentWidths: PanelWidths,
  totalWidth: number
): PanelWidths {
  if (handlePosition === 'left') {
    // Dragging between left and center
    const newLeftWidth = currentWidths.left + deltaX;
    return calculatePanelWidths(
      totalWidth,
      newLeftWidth,
      currentWidths.right
    );
  } else {
    // Dragging between center and right
    const newRightWidth = currentWidths.right - deltaX;
    return calculatePanelWidths(
      totalWidth,
      currentWidths.left,
      newRightWidth
    );
  }
}
```

## Migration Strategy

### Phase 1: Create New Components

1. Create `LeftPanel.tsx` with DataSourcesSection and HistoricalSessionsSection
2. Create `NewSessionButton.tsx`
3. Create `CenterPanel.tsx` by refactoring ChatSidebar
4. Create `DataBrowser.tsx` for slide-out data browsing
5. Create `RightPanel.tsx` as wrapper for DraggableDashboard
6. Create `ResizeHandle.tsx` for panel resizing

### Phase 2: Update App.tsx Layout

1. Replace current layout structure with three-panel flexbox
2. Add ResizeHandle components between panels
3. Implement panel width state management
4. Add localStorage persistence for panel widths
5. Wire up event handlers for all components

### Phase 3: Migrate State and Events

1. Move chat state from ChatSidebar to CenterPanel
2. Move data source selection to LeftPanel
3. Add session selection handling
4. Implement data browser open/close logic
5. Update all EventsOn listeners to work with new structure

### Phase 4: Remove Old Components

1. Remove ChatSidebar overlay logic
2. Remove ContextPanel (old data browser)
3. Remove collapse/expand button logic
4. Clean up unused state variables
5. Remove unused CSS classes

### Phase 5: Testing and Polish

1. Test panel resizing with various window sizes
2. Test data browser slide-in/out animations
3. Test session switching and message loading
4. Test keyboard shortcuts
5. Test accessibility with screen readers
6. Performance testing with large datasets

## Error Handling

### Data Loading Errors

```typescript
// In LeftPanel
try {
  const sessions = await GetChatThreads();
  setSessions(sessions);
} catch (error) {
  console.error('Failed to load sessions:', error);
  showToast('error', 'Failed to load historical sessions');
  setSessions([]);
}
```

### Data Browser Errors

```typescript
// In DataBrowser
try {
  const tables = await GetDataSourceTables(sourceId);
  setTables(tables);
} catch (error) {
  console.error('Failed to load tables:', error);
  setError('Unable to load data source tables. Please try again.');
  setTables([]);
}
```

### Panel Resize Errors

```typescript
// In App.tsx
try {
  localStorage.setItem('panelWidths', JSON.stringify(panelWidths));
} catch (error) {
  console.warn('Failed to save panel widths:', error);
  // Continue without persistence
}
```

## Testing Strategy

### Unit Tests

1. **LeftPanel Component**
   - Test data source list rendering
   - Test session list rendering
   - Test context menu interactions
   - Test new session button click

2. **CenterPanel Component**
   - Test message rendering
   - Test message input handling
   - Test auto-scroll behavior
   - Test loading states

3. **DataBrowser Component**
   - Test slide-in/out animations
   - Test table selection
   - Test data loading
   - Test pagination
   - Test close button

4. **ResizeHandle Component**
   - Test drag start/end
   - Test drag delta calculations
   - Test cursor changes
   - Test visual feedback

5. **Panel Width Calculations**
   - Test constraint enforcement
   - Test width calculations with various inputs
   - Test edge cases (minimum window size)
   - Test localStorage persistence

### Integration Tests

1. **Session Workflow**
   - Create new session → verify CenterPanel updates
   - Select historical session → verify data loads
   - Send message → verify dashboard updates

2. **Data Browser Workflow**
   - Open data browser → verify slide-in animation
   - Select table → verify data loads
   - Close browser → verify CenterPanel visible

3. **Panel Resizing**
   - Drag left handle → verify left and center resize
   - Drag right handle → verify center and right resize
   - Resize to minimum → verify constraints enforced
   - Reload page → verify widths restored

### Property-Based Tests

Will be defined in the Correctness Properties section below.


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property Reflection

After analyzing all acceptance criteria, I identified the following categories of redundancy:

1. **Panel visibility properties**: Multiple criteria test that panels remain visible. These can be combined into comprehensive visibility properties.
2. **Resize behavior properties**: Several criteria test different aspects of resizing. These can be unified into properties about constraint enforcement and persistence.
3. **Data rendering properties**: Many criteria test that data is rendered when available. These follow the same pattern and can be generalized.
4. **Interaction properties**: Click, right-click, and keyboard interactions follow similar patterns across different components.

The properties below represent the unique, non-redundant validation requirements.

### Layout Structure Properties

**Property 1: Panel width conservation**
*For any* window width and panel configuration, the sum of left panel width, center panel width, and right panel width SHALL equal the total available width.
**Validates: Requirements 1.4, 1.6**

**Property 2: Panel width constraints**
*For any* resize operation, the resulting panel widths SHALL satisfy: left >= 180px AND left <= 400px AND center >= 400px AND right >= 280px AND right <= 600px.
**Validates: Requirements 9.4, 9.5**

**Property 3: Panel persistence round-trip**
*For any* valid panel width configuration, saving to localStorage then loading SHALL produce equivalent panel widths (within 1px tolerance).
**Validates: Requirements 9.1, 9.2**

### Data Rendering Properties

**Property 4: List rendering completeness**
*For any* non-empty array of data sources, the rendered list SHALL contain exactly one list item per data source with matching IDs.
**Validates: Requirements 2.2, 3.2**

**Property 5: Session chronological ordering**
*For any* list of sessions with timestamps, the rendered order SHALL be reverse chronological (newest first), meaning for all adjacent pairs (i, i+1), session[i].timestamp >= session[i+1].timestamp.
**Validates: Requirements 3.7**

**Property 6: Data source type indicators**
*For any* data source in the list, its rendered element SHALL contain a type indicator (icon or color class) corresponding to its type property.
**Validates: Requirements 2.5**

**Property 7: Session metadata completeness**
*For any* rendered session item, it SHALL display all required metadata fields: name, date, and data source name.
**Validates: Requirements 3.5**

### Interaction Properties

**Property 8: Selection state consistency**
*For any* selectable item (data source or session), clicking it SHALL result in exactly one item being marked as selected in its list.
**Validates: Requirements 2.3, 3.3**

**Property 9: Context menu trigger**
*For any* right-clickable element (data source or session), right-clicking SHALL display a context menu containing the expected options for that element type.
**Validates: Requirements 2.4, 3.6**

**Property 10: Data browser toggle**
*For any* data browser state (open or closed), triggering the toggle action (Ctrl+B or close button) SHALL result in the opposite state.
**Validates: Requirements 7.6, 11.4**

**Property 11: Message send immediacy**
*For any* valid message text, sending the message SHALL result in it appearing in the conversation history within the same render cycle.
**Validates: Requirements 5.6**

**Property 12: Insight click propagation**
*For any* insight in the dashboard, clicking it SHALL trigger a message send event with the insight text as the message content.
**Validates: Requirements 6.6**

### Data Browser Properties

**Property 13: Data browser overlay positioning**
*For any* data browser open state, the browser element SHALL overlay only the center panel, with its left edge >= left panel width AND its right edge <= (total width - right panel width).
**Validates: Requirements 7.2, 7.3**

**Property 14: Center panel persistence**
*For any* data browser state (open or closed), the center panel element SHALL remain in the DOM (not unmounted).
**Validates: Requirements 7.7**

**Property 15: Data browser content loading**
*For any* data source with tables, opening the data browser for that source SHALL trigger loading of the table list, and selecting a table SHALL trigger loading of its columns and sample data.
**Validates: Requirements 8.2, 8.3, 8.4**

**Property 16: Data browser search filtering**
*For any* search query in the data browser, the filtered table list SHALL contain only tables whose names contain the query string (case-insensitive).
**Validates: Requirements 8.8**

### Keyboard Navigation Properties

**Property 17: Panel focus shortcuts**
*For any* panel (left, center, right), pressing its corresponding shortcut (Ctrl+1/2/3) SHALL move focus to that panel.
**Validates: Requirements 11.1, 11.2, 11.3**

**Property 18: Escape key data browser close**
*For any* data browser open state, pressing Escape SHALL close the data browser.
**Validates: Requirements 11.5**

**Property 19: Tab navigation containment**
*For any* focused panel, pressing Tab SHALL move focus to the next focusable element within that panel, wrapping to the first element after the last.
**Validates: Requirements 11.7**

**Property 20: ARIA attribute presence**
*For any* interactive element (button, list item, panel), it SHALL have appropriate ARIA attributes (role, label, or labelledby).
**Validates: Requirements 11.8**

### State Management Properties

**Property 21: Session switch data consistency**
*For any* session switch operation, the center panel SHALL display messages from the new session AND the right panel SHALL display dashboard data from the new session.
**Validates: Requirements 3.3, 3.4, 6.7**

**Property 22: Loading state visibility**
*For any* analysis loading state (true or false), the center panel SHALL display a loading indicator if and only if the state is true.
**Validates: Requirements 5.8**

**Property 23: Empty state fallback**
*For any* empty data condition (no data sources, no sessions, no analysis results), the corresponding panel SHALL display an appropriate empty state message.
**Validates: Requirements 2.6, 3.8, 5.4, 6.4**

### Resize Behavior Properties

**Property 24: Resize handle drag effect**
*For any* resize handle drag operation with delta X, the adjacent panel widths SHALL change by approximately delta X (subject to constraints), and the change SHALL be reflected in real-time during the drag.
**Validates: Requirements 1.9**

**Property 25: Resize debouncing**
*For any* rapid sequence of resize events (>10 events within 100ms), the system SHALL process at most one resize calculation per 16ms (60fps).
**Validates: Requirements 13.5**

### Performance Properties

**Property 26: List virtualization**
*For any* list with more than 50 items (sessions or messages), the DOM SHALL contain at most 20 rendered list items at any time (only visible items plus buffer).
**Validates: Requirements 13.2, 13.3**

**Property 27: Lazy loading**
*For any* data browser table list, table data SHALL be loaded only when the table is selected, not when the browser opens.
**Validates: Requirements 13.6**

### Theme Support Properties

**Property 28: Theme consistency**
*For any* theme mode (light or dark), all panels SHALL use colors from the corresponding theme palette, with no hardcoded colors that don't respect the theme.
**Validates: Requirements 12.7**

**Property 29: Hover state presence**
*For any* interactive element (button, list item, link), it SHALL have defined hover styles that differ from its default state.
**Validates: Requirements 12.6**

### Migration Compatibility Properties

**Property 30: Feature preservation**
*For any* existing feature (chat send, dashboard display, data source management), the feature SHALL work identically in the new layout as in the old layout.
**Validates: Requirements 10.3, 10.4, 10.5, 10.8**


## Testing Strategy

### Dual Testing Approach

This feature requires both unit tests and property-based tests to ensure comprehensive coverage:

- **Unit tests**: Verify specific examples, edge cases, component rendering, and integration points
- **Property tests**: Verify universal properties across all inputs using randomized testing

Together, these approaches provide comprehensive validation: unit tests catch concrete bugs and verify specific scenarios, while property tests verify general correctness across a wide range of inputs.

### Property-Based Testing Configuration

**Testing Library**: We will use `fast-check` for TypeScript/React property-based testing.

**Configuration**:
- Minimum 100 iterations per property test (due to randomization)
- Each property test must reference its design document property
- Tag format: `Feature: western-layout-redesign, Property {number}: {property_text}`

**Example Property Test Structure**:
```typescript
import fc from 'fast-check';

// Feature: western-layout-redesign, Property 1: Panel width conservation
test('panel widths sum to total width', () => {
  fc.assert(
    fc.property(
      fc.integer({ min: 1024, max: 3840 }), // window width
      fc.integer({ min: 180, max: 400 }),   // left width
      fc.integer({ min: 280, max: 600 }),   // right width
      (totalWidth, leftWidth, rightWidth) => {
        const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);
        expect(result.left + result.center + result.right).toBe(totalWidth);
      }
    ),
    { numRuns: 100 }
  );
});
```

### Unit Testing Strategy

#### Component Unit Tests

1. **LeftPanel Component**
   - Renders data sources section with correct header
   - Renders historical sessions section below data sources
   - Renders new session button between sections
   - Handles data source selection
   - Handles session selection
   - Shows context menus on right-click
   - Displays empty states when no data

2. **CenterPanel Component**
   - Renders chat interface with message list
   - Renders message input at bottom
   - Displays welcome message when no session active
   - Displays loading indicator during analysis
   - Auto-scrolls to latest message
   - Handles message send
   - Remains in DOM when data browser is open

3. **DataBrowser Component**
   - Slides in from right with animation
   - Displays data source name in header
   - Loads and displays table list
   - Loads columns and data when table selected
   - Provides pagination controls
   - Filters tables based on search query
   - Closes on X button click
   - Closes on Escape key press
   - Resizes when left edge dragged

4. **RightPanel Component**
   - Renders dashboard with metrics
   - Renders charts when available
   - Renders insights when available
   - Renders file downloads when available
   - Handles insight click
   - Updates when session switches
   - Shows empty state when no data

5. **ResizeHandle Component**
   - Changes cursor on hover
   - Captures mouse events on drag
   - Emits delta values during drag
   - Prevents text selection during drag
   - Shows visual feedback during drag

#### Integration Tests

1. **Complete Session Workflow**
   - Select data source → new session button enabled
   - Click new session → dialog opens with pre-selected source
   - Create session → center panel shows empty chat
   - Send message → message appears and analysis starts
   - Analysis completes → dashboard updates with results

2. **Data Browser Workflow**
   - Right-click data source → context menu appears
   - Click "Browse Data" → browser slides in
   - Select table → columns and data load
   - Search for table → list filters
   - Click X → browser slides out
   - Center panel visible again

3. **Panel Resizing Workflow**
   - Drag left handle → left and center resize
   - Drag right handle → center and right resize
   - Resize below minimum → constraint enforced
   - Resize above maximum → constraint enforced
   - Reload page → widths restored from localStorage

4. **Session Switching Workflow**
   - Click session in history → session loads in center
   - Messages display in center panel
   - Dashboard updates with session results
   - Click different session → content switches
   - Previous session data cleared

5. **Keyboard Navigation Workflow**
   - Press Ctrl+1 → left panel focused
   - Press Ctrl+2 → center panel focused
   - Press Ctrl+3 → right panel focused
   - Press Ctrl+B → data browser toggles
   - Press Escape with browser open → browser closes
   - Press Tab → focus moves within panel

### Property-Based Test Mapping

Each correctness property will be implemented as a property-based test:

| Property | Test Focus | Generators Needed |
|----------|-----------|-------------------|
| Property 1 | Width conservation | Window widths, panel widths |
| Property 2 | Width constraints | Resize operations, panel widths |
| Property 3 | Persistence round-trip | Panel configurations |
| Property 4 | List rendering | Data source arrays |
| Property 5 | Chronological ordering | Session arrays with timestamps |
| Property 6 | Type indicators | Data sources with types |
| Property 7 | Metadata completeness | Session objects |
| Property 8 | Selection consistency | Click events, item lists |
| Property 9 | Context menu trigger | Right-click events, element types |
| Property 10 | Browser toggle | Browser states, toggle actions |
| Property 11 | Message immediacy | Message texts |
| Property 12 | Insight propagation | Insight objects |
| Property 13 | Browser positioning | Panel widths, browser states |
| Property 14 | Panel persistence | Browser states |
| Property 15 | Content loading | Data sources with tables |
| Property 16 | Search filtering | Search queries, table lists |
| Property 17 | Focus shortcuts | Keyboard events, panel IDs |
| Property 18 | Escape close | Browser states, keyboard events |
| Property 19 | Tab containment | Focus states, focusable elements |
| Property 20 | ARIA attributes | Interactive elements |
| Property 21 | Session switch consistency | Session IDs, session data |
| Property 22 | Loading visibility | Loading states |
| Property 23 | Empty state fallback | Empty data conditions |
| Property 24 | Resize drag effect | Drag deltas, panel widths |
| Property 25 | Resize debouncing | Event sequences |
| Property 26 | List virtualization | Large item arrays |
| Property 27 | Lazy loading | Table selections |
| Property 28 | Theme consistency | Theme modes, color values |
| Property 29 | Hover states | Interactive elements |
| Property 30 | Feature preservation | Feature operations |

### Test Data Generators

For property-based testing, we need generators for:

```typescript
// Window dimensions
const windowWidthGen = fc.integer({ min: 1024, max: 3840 });

// Panel widths
const leftWidthGen = fc.integer({ min: 180, max: 400 });
const rightWidthGen = fc.integer({ min: 280, max: 600 });

// Data sources
const dataSourceGen = fc.record({
  id: fc.uuid(),
  name: fc.string({ minLength: 1, maxLength: 50 }),
  type: fc.constantFrom('excel', 'mysql', 'postgresql', 'doris', 'csv', 'json'),
  createdAt: fc.integer({ min: 0, max: Date.now() })
});

const dataSourceArrayGen = fc.array(dataSourceGen, { minLength: 0, maxLength: 100 });

// Sessions
const sessionGen = fc.record({
  id: fc.uuid(),
  name: fc.string({ minLength: 1, maxLength: 100 }),
  dataSourceId: fc.uuid(),
  dataSourceName: fc.string({ minLength: 1, maxLength: 50 }),
  createdAt: fc.integer({ min: 0, max: Date.now() }),
  updatedAt: fc.integer({ min: 0, max: Date.now() }),
  messageCount: fc.integer({ min: 0, max: 1000 })
});

const sessionArrayGen = fc.array(sessionGen, { minLength: 0, maxLength: 100 });

// Messages
const messageGen = fc.record({
  id: fc.uuid(),
  role: fc.constantFrom('user', 'assistant'),
  content: fc.string({ minLength: 1, maxLength: 1000 }),
  timestamp: fc.integer({ min: 0, max: Date.now() })
});

const messageArrayGen = fc.array(messageGen, { minLength: 0, maxLength: 500 });

// Tables
const tableGen = fc.record({
  name: fc.string({ minLength: 1, maxLength: 50 }),
  rowCount: fc.integer({ min: 0, max: 1000000 }),
  columnCount: fc.integer({ min: 1, max: 100 })
});

const tableArrayGen = fc.array(tableGen, { minLength: 0, maxLength: 50 });

// Search queries
const searchQueryGen = fc.string({ minLength: 0, maxLength: 50 });

// Drag deltas
const dragDeltaGen = fc.integer({ min: -500, max: 500 });

// Theme modes
const themeModeGen = fc.constantFrom('light', 'dark');
```

### Edge Cases to Test

1. **Minimum window size** (1024px width)
2. **Maximum window size** (3840px width)
3. **Empty data states** (no sources, no sessions, no messages)
4. **Single item lists** (one source, one session)
5. **Very long lists** (100+ sources, 500+ messages)
6. **Very long text** (long data source names, long messages)
7. **Rapid interactions** (fast clicking, rapid resizing)
8. **Concurrent operations** (resize while loading data)
9. **Network failures** (failed data loads)
10. **Invalid data** (malformed session data, missing fields)

### Performance Testing

While not part of property-based testing, we should measure:

1. **Initial render time** (target: <500ms)
2. **Session switch time** (target: <300ms)
3. **Data browser open time** (target: <200ms)
4. **Resize responsiveness** (target: 60fps)
5. **Memory usage** with large datasets
6. **Virtualization effectiveness** (DOM node count)

### Accessibility Testing

1. **Screen reader compatibility** (NVDA, JAWS, VoiceOver)
2. **Keyboard-only navigation** (all features accessible)
3. **Focus indicators** (visible and clear)
4. **Color contrast** (WCAG AA compliance)
5. **ARIA attributes** (correct roles and labels)

### Browser Compatibility Testing

Test on:
- Chrome (latest)
- Firefox (latest)
- Safari (latest)
- Edge (latest)

### Regression Testing

After implementation, verify that:
1. All existing features still work
2. No performance degradation
3. No accessibility regressions
4. No visual regressions (screenshot comparison)

