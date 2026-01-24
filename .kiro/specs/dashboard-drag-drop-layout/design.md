# Design Document: Dashboard Drag-Drop Layout

## Overview

This design implements a comprehensive dashboard redesign that enables users to customize their dashboard layout through drag-and-drop interactions. The system consists of a grid-based layout engine, a component management system, a layout editor with lock/unlock functionality, and an enhanced export service that filters components based on data availability.

The architecture follows a clear separation between the React/TypeScript frontend (handling UI interactions, drag-drop logic, and layout rendering) and the Go backend (managing layout persistence, data retrieval, and export operations). The design leverages existing Wails infrastructure for frontend-backend communication.

Key design principles:
- **Grid-based positioning**: All components snap to a grid system for consistent alignment
- **Component modularity**: Each component type (metrics, tables, images, insights) implements a common interface
- **State management**: Layout configuration is managed centrally and persisted to backend storage
- **Progressive enhancement**: Layout editor mode adds functionality without disrupting normal dashboard usage
- **Data-driven visibility**: Components automatically hide when empty, keeping the dashboard clean

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Frontend (React/TypeScript)              │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌─────────────────────────────────┐ │
│  │  Dashboard       │  │  Layout Editor                  │ │
│  │  Container       │  │  - Lock/Unlock Toggle           │ │
│  │                  │  │  - Edit Mode State              │ │
│  └────────┬─────────┘  └──────────────┬──────────────────┘ │
│           │                            │                     │
│  ┌────────▼────────────────────────────▼──────────────────┐ │
│  │         Layout Engine                                   │ │
│  │  - Grid System (24 columns)                            │ │
│  │  - Collision Detection                                 │ │
│  │  - Snap-to-Grid Logic                                  │ │
│  └────────┬────────────────────────────────────────────────┘ │
│           │                                                   │
│  ┌────────▼────────────────────────────────────────────────┐ │
│  │         Component Manager                               │ │
│  │  - Component Registry                                   │ │
│  │  - Instance Management                                  │ │
│  │  - Pagination State                                     │ │
│  └────────┬────────────────────────────────────────────────┘ │
│           │                                                   │
│  ┌────────▼────────────────────────────────────────────────┐ │
│  │         Draggable Components                            │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │ │
│  │  │ Metrics  │ │  Tables  │ │  Images  │ │ Insights │  │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │ │
│  │  ┌──────────────────────┐                              │ │
│  │  │  File Download Area  │                              │ │
│  │  └──────────────────────┘                              │ │
│  └─────────────────────────────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────┘
                            │ Wails Bridge
┌───────────────────────────▼─────────────────────────────────┐
│                     Backend (Go)                             │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐ │
│  │         Layout Service                                   │ │
│  │  - SaveLayout(config)                                   │ │
│  │  - LoadLayout() -> config                               │ │
│  │  - GetDefaultLayout() -> config                         │ │
│  └────────┬────────────────────────────────────────────────┘ │
│           │                                                   │
│  ┌────────▼────────────────────────────────────────────────┐ │
│  │         Data Service                                     │ │
│  │  - GetComponentData(type, id) -> data                   │ │
│  │  - CheckComponentHasData(type, id) -> bool              │ │
│  └────────┬────────────────────────────────────────────────┘ │
│           │                                                   │
│  ┌────────▼────────────────────────────────────────────────┐ │
│  │         File Service                                     │ │
│  │  - GetFilesByCategory(category) -> files                │ │
│  │  - HasFiles() -> bool                                   │ │
│  │  - DownloadFile(fileID) -> path                         │ │
│  └────────┬────────────────────────────────────────────────┘ │
│           │                                                   │
│  ┌────────▼────────────────────────────────────────────────┐ │
│  │         Export Service (Enhanced)                        │ │
│  │  - ExportDashboard(config) -> file                      │ │
│  │  - FilterEmptyComponents(components) -> filtered        │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │         Storage Layer                                    │ │
│  │  - SQLite Database (layout_configs table)              │ │
│  │  - JSON File Fallback                                   │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

**Layout Modification Flow:**
1. User drags/resizes component in frontend
2. Layout Engine calculates new position/size with grid snapping
3. Component Manager updates local state
4. Frontend calls `SaveLayout()` via Wails bridge
5. Backend persists to storage
6. Confirmation returned to frontend

**Component Visibility Flow:**
1. Dashboard loads layout configuration
2. For each component, frontend calls `CheckComponentHasData()`
3. Backend queries data service
4. If locked mode and no data: component hidden
5. If edit mode: component shown with empty state indicator

**Export Flow:**
1. User initiates export
2. Frontend calls `ExportDashboard()` with current layout config
3. Backend calls `FilterEmptyComponents()` to identify non-empty components
4. Export Service generates output for filtered components only
5. File returned to user with metadata about included components

## Components and Interfaces

### Frontend Components

#### DashboardContainer Component
```typescript
interface DashboardContainerProps {
  initialLayout?: LayoutConfiguration;
}

interface DashboardContainerState {
  layout: LayoutConfiguration;
  isEditMode: boolean;
  isLocked: boolean;
  components: ComponentInstance[];
}

// Main container managing overall dashboard state
class DashboardContainer extends React.Component<
  DashboardContainerProps,
  DashboardContainerState
> {
  // Handles layout changes, mode switching, and persistence
}
```

#### LayoutEngine
```typescript
interface GridConfig {
  columns: number;        // 24 columns
  rowHeight: number;      // pixels per row unit
  margin: [number, number]; // [horizontal, vertical] margins
  containerPadding: [number, number];
}

interface LayoutItem {
  i: string;              // unique component instance ID
  x: number;              // grid column position (0-23)
  y: number;              // grid row position
  w: number;              // width in grid units
  h: number;              // height in grid units
  minW?: number;          // minimum width
  minH?: number;          // minimum height
  maxW?: number;          // maximum width
  maxH?: number;          // maximum height
  static?: boolean;       // if true, cannot be dragged/resized
}

class LayoutEngine {
  // Calculates valid positions with collision detection
  calculatePosition(item: LayoutItem, x: number, y: number): Position;
  
  // Snaps dimensions to grid
  snapToGrid(width: number, height: number): Dimensions;
  
  // Detects and resolves overlaps
  detectCollisions(items: LayoutItem[]): CollisionResult;
  
  // Compacts layout to remove gaps
  compactLayout(items: LayoutItem[]): LayoutItem[];
}
```

#### ComponentManager
```typescript
interface ComponentInstance {
  id: string;
  type: ComponentType;
  instanceIndex: number;  // for pagination (0, 1, 2...)
  data: any;
  hasData: boolean;
  layoutItem: LayoutItem;
}

enum ComponentType {
  METRICS = 'metrics',
  TABLE = 'table',
  IMAGE = 'image',
  INSIGHTS = 'insights',
  FILE_DOWNLOAD = 'file_download'
}

class ComponentManager {
  // Registers component types and their renderers
  registerComponent(type: ComponentType, renderer: ComponentRenderer): void;
  
  // Creates new component instance
  createInstance(type: ComponentType): ComponentInstance;
  
  // Manages pagination state for component groups
  getPaginationState(type: ComponentType): PaginationState;
  
  // Updates component data and visibility
  updateComponentData(id: string, data: any): void;
}
```

#### DraggableComponent
```typescript
interface DraggableComponentProps {
  instance: ComponentInstance;
  isEditMode: boolean;
  isLocked: boolean;
  onDragStart: (id: string) => void;
  onDrag: (id: string, x: number, y: number) => void;
  onDragStop: (id: string, x: number, y: number) => void;
  onResize: (id: string, width: number, height: number) => void;
  onResizeStop: (id: string, width: number, height: number) => void;
}

// Wrapper component providing drag and resize functionality
const DraggableComponent: React.FC<DraggableComponentProps> = (props) => {
  // Renders component with drag handles and resize handles
  // Handles drag/resize events and calls parent callbacks
};
```

#### PaginationControl
```typescript
interface PaginationControlProps {
  componentType: ComponentType;
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  visible: boolean;
}

// Renders left/right navigation for component instances
const PaginationControl: React.FC<PaginationControlProps> = (props) => {
  // Similar to existing image/echart pagination
};
```

#### FileDownloadComponent
```typescript
interface FileDownloadComponentProps {
  instance: ComponentInstance;
  isEditMode: boolean;
}

interface FileCategory {
  name: string;
  files: FileInfo[];
}

interface FileInfo {
  id: string;
  name: string;
  size: number;
  createdAt: Date;
  downloadUrl: string;
}

// Displays file download area with two categories
const FileDownloadComponent: React.FC<FileDownloadComponentProps> = (props) => {
  // Renders two categories: "All Files" and "User Request Related Files"
  // Each category shows a list of downloadable files
  // Handles file download on click
  // Shows empty state when no files in a category
};
```

#### LayoutEditor
```typescript
interface LayoutEditorProps {
  isLocked: boolean;
  onToggleLock: () => void;
  onAddComponent: (type: ComponentType) => void;
  onRemoveComponent: (id: string) => void;
}

// Toolbar for layout editing controls
const LayoutEditor: React.FC<LayoutEditorProps> = (props) => {
  // Renders lock/unlock button, add component buttons, etc.
};
```

### Backend Services

#### Layout Service (Go)
```go
type LayoutConfiguration struct {
    ID          string       `json:"id"`
    UserID      string       `json:"userId"`
    IsLocked    bool         `json:"isLocked"`
    Items       []LayoutItem `json:"items"`
    CreatedAt   time.Time    `json:"createdAt"`
    UpdatedAt   time.Time    `json:"updatedAt"`
}

type LayoutItem struct {
    I           string  `json:"i"`
    X           int     `json:"x"`
    Y           int     `json:"y"`
    W           int     `json:"w"`
    H           int     `json:"h"`
    MinW        int     `json:"minW,omitempty"`
    MinH        int     `json:"minH,omitempty"`
    MaxW        int     `json:"maxW,omitempty"`
    MaxH        int     `json:"maxH,omitempty"`
    Static      bool    `json:"static"`
    Type        string  `json:"type"`
    InstanceIdx int     `json:"instanceIdx"`
}

type LayoutService struct {
    db *sql.DB
}

// Saves layout configuration to database
func (s *LayoutService) SaveLayout(config LayoutConfiguration) error

// Loads layout configuration from database
func (s *LayoutService) LoadLayout(userID string) (*LayoutConfiguration, error)

// Returns default layout configuration
func (s *LayoutService) GetDefaultLayout() LayoutConfiguration
```

#### Data Service (Go)
```go
type DataService struct {
    // Existing data source connections
}

// Retrieves data for a specific component instance
func (s *DataService) GetComponentData(componentType string, instanceID string) (interface{}, error)

// Checks if component has any data
func (s *DataService) CheckComponentHasData(componentType string, instanceID string) (bool, error)

// Batch check for multiple components
func (s *DataService) BatchCheckHasData(components []string) (map[string]bool, error)
```

#### Enhanced Export Service (Go)
```go
type ExportService struct {
    dataService   *DataService
    layoutService *LayoutService
}

type ExportRequest struct {
    LayoutConfig LayoutConfiguration `json:"layoutConfig"`
    Format       string              `json:"format"` // "xlsx", "csv", "json"
}

type ExportResult struct {
    FilePath          string   `json:"filePath"`
    IncludedComponents []string `json:"includedComponents"`
    ExcludedComponents []string `json:"excludedComponents"`
}

// Exports dashboard data, filtering empty components
func (s *ExportService) ExportDashboard(req ExportRequest) (*ExportResult, error)

// Filters out components without data
func (s *ExportService) FilterEmptyComponents(items []LayoutItem) ([]LayoutItem, error)
```

#### File Service (Go)
```go
type FileService struct {
    dataDir string
}

type FileCategory string

const (
    AllFiles              FileCategory = "all_files"
    UserRequestRelated    FileCategory = "user_request_related"
)

type FileInfo struct {
    ID          string       `json:"id"`
    Name        string       `json:"name"`
    Size        int64        `json:"size"`
    CreatedAt   time.Time    `json:"createdAt"`
    Category    FileCategory `json:"category"`
    DownloadURL string       `json:"downloadUrl"`
}

// Gets files for a specific category
func (s *FileService) GetFilesByCategory(category FileCategory) ([]FileInfo, error)

// Checks if file download component has any files
func (s *FileService) HasFiles() (bool, error)

// Initiates file download
func (s *FileService) DownloadFile(fileID string) (string, error)
```

## Data Models

### Layout Configuration Storage

**Database Schema (SQLite):**
```sql
CREATE TABLE layout_configs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    is_locked BOOLEAN DEFAULT FALSE,
    layout_data TEXT NOT NULL,  -- JSON serialized LayoutItem array
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

CREATE INDEX idx_layout_user ON layout_configs(user_id);
```

**JSON Structure (layout_data field):**
```json
{
  "items": [
    {
      "i": "metrics-0",
      "x": 0,
      "y": 0,
      "w": 6,
      "h": 4,
      "minW": 3,
      "minH": 2,
      "type": "metrics",
      "instanceIdx": 0,
      "static": false
    },
    {
      "i": "table-0",
      "x": 6,
      "y": 0,
      "w": 12,
      "h": 8,
      "minW": 6,
      "minH": 4,
      "type": "table",
      "instanceIdx": 0,
      "static": false
    },
    {
      "i": "image-0",
      "x": 18,
      "y": 0,
      "w": 6,
      "h": 6,
      "minW": 4,
      "minH": 4,
      "type": "image",
      "instanceIdx": 0,
      "static": false
    },
    {
      "i": "image-1",
      "x": 18,
      "y": 0,
      "w": 6,
      "h": 6,
      "minW": 4,
      "minH": 4,
      "type": "image",
      "instanceIdx": 1,
      "static": false
    }
  ]
}
```

### Component State Model

**Frontend State:**
```typescript
interface ComponentState {
  instances: Map<string, ComponentInstance>;
  paginationState: Map<ComponentType, PaginationState>;
  visibilityMap: Map<string, boolean>;
  dataCache: Map<string, any>;
}

interface PaginationState {
  currentPage: number;
  totalPages: number;
  instancesPerPage: number;
}
```

### Default Layout Configuration

The system provides a default layout when no saved configuration exists:

```typescript
const DEFAULT_LAYOUT: LayoutConfiguration = {
  id: 'default',
  userId: '',
  isLocked: false,
  items: [
    // Key metrics at top-left
    { i: 'metrics-0', x: 0, y: 0, w: 8, h: 4, minW: 4, minH: 2, type: 'metrics', instanceIdx: 0 },
    
    // Data table in center
    { i: 'table-0', x: 0, y: 4, w: 16, h: 8, minW: 8, minH: 6, type: 'table', instanceIdx: 0 },
    
    // Images on right side
    { i: 'image-0', x: 16, y: 0, w: 8, h: 6, minW: 4, minH: 4, type: 'image', instanceIdx: 0 },
    
    // Insights below images
    { i: 'insights-0', x: 16, y: 6, w: 8, h: 6, minW: 4, minH: 4, type: 'insights', instanceIdx: 0 },
    
    // File download area at bottom
    { i: 'file_download-0', x: 0, y: 12, w: 24, h: 6, minW: 8, minH: 4, type: 'file_download', instanceIdx: 0 }
  ],
  createdAt: new Date(),
  updatedAt: new Date()
};
```

## Correctness Properties


*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Drag Operation Persistence
*For any* component and any valid grid position, when a user completes a drag operation (click, drag, release), the component's position in the Layout_Configuration should be updated to reflect the new position.
**Validates: Requirements 1.3**

### Property 2: Invalid Position Reversion
*For any* component and any invalid position (overlapping, out of bounds), when a user attempts to drop the component, the component should return to its original position and the Layout_Configuration should remain unchanged.
**Validates: Requirements 1.4**

### Property 3: Lock State Prevents Editing
*For any* component, when the Layout_Editor is locked, all drag and resize operations should be disabled and all editing UI elements (drag handles, resize handles) should be hidden.
**Validates: Requirements 1.5, 2.6, 4.2, 4.3**

### Property 4: Resize Operation Persistence
*For any* component and any valid dimensions within min/max constraints, when a user completes a resize operation, the component's dimensions in the Layout_Configuration should be updated to reflect the new size.
**Validates: Requirements 2.3**

### Property 5: Size Constraint Enforcement
*For any* component, when resizing would violate minimum or maximum size constraints, the system should prevent the resize operation and maintain the current valid size.
**Validates: Requirements 2.4, 2.5**

### Property 6: Pagination Visibility
*For any* component type, when multiple instances exist (count > 1), pagination controls should be visible; when only one instance exists, pagination controls should be hidden.
**Validates: Requirements 3.1, 3.4**

### Property 7: Pagination Navigation
*For any* component type with multiple instances, when a user navigates to page N, only the instance at index N should be visible and all other instances of that type should be hidden.
**Validates: Requirements 3.2, 3.3**

### Property 8: Pagination State Persistence
*For any* component type with pagination, when switching between locked and unlocked Layout_Editor modes, the current page selection should be preserved.
**Validates: Requirements 3.5**

### Property 9: Edit Mode Activation
*For any* dashboard state, when the Layout_Editor is unlocked, all components should display drag handles and resize handles, and all drag/resize operations should be enabled.
**Validates: Requirements 4.1, 4.4**

### Property 10: Component Visibility Based on Data
*For any* component in locked mode, if the component has no data, it should be hidden from the dashboard; if it has data, it should be visible according to the Layout_Configuration.
**Validates: Requirements 5.1, 5.2**

### Property 11: Group Visibility
*For any* component type, when all instances of that type have no data in locked mode, the entire component group including pagination controls should be hidden.
**Validates: Requirements 5.3**

### Property 12: Edit Mode Shows All Components
*For any* component, when the Layout_Editor is unlocked, the component should be visible regardless of data availability, with empty components displaying a visual indicator.
**Validates: Requirements 5.4, 5.5**

### Property 13: Layout Configuration Round-Trip
*For any* valid Layout_Configuration, saving the configuration and then loading it should produce an equivalent configuration with the same component positions, sizes, and lock state.
**Validates: Requirements 6.1, 6.2, 6.3, 6.4**

### Property 14: Export Filters Empty Components
*For any* dashboard state, when exporting data, the export output should include only components that have data, and the export result should list which components were included and excluded.
**Validates: Requirements 7.1, 7.2, 7.3, 7.4**

### Property 15: Component Type Consistency
*For any* component type (metrics, table, image, insights, file_download), all instances should support the same drag, resize, and pagination behaviors consistently.
**Validates: Requirements 8.5, 8.6**

### Property 16: Grid Snapping
*For any* drag or resize operation, the final position and dimensions should align to grid boundaries (positions snap to grid columns/rows, dimensions snap to grid units).
**Validates: Requirements 9.2, 9.3**

### Property 17: Collision Prevention
*For any* layout state, no two components should overlap in their grid positions; if a drag/resize would cause overlap, the system should either prevent the operation or automatically reposition components.
**Validates: Requirements 9.4**

### Property 18: Responsive Layout Preservation
*For any* layout configuration, when the viewport size changes, the relative positions of components should be preserved (components maintain their grid positions and scale proportionally).
**Validates: Requirements 9.5**

### Property 19: Visual Feedback During Drag
*For any* drag operation, while dragging, the system should display a semi-transparent preview at the target position and change the cursor to indicate draggable state.
**Validates: Requirements 1.1, 1.2, 10.2**

### Property 20: Lock State Indicator
*For any* dashboard state, when the Layout_Editor is locked, a lock icon or indicator should be visible; when unlocked, an unlock icon or edit mode indicator should be visible.
**Validates: Requirements 10.4**

### Property 21: File Download Component Data Availability
*For any* file download component, when both file categories (all files and user-request-related files) are empty, the component should be hidden in locked mode; when at least one category has files, the component should be visible.
**Validates: Requirements 11.5**

### Property 22: File Download Category Display
*For any* file download component, when a category has no files, that category should display an empty state message; when a category has files, it should display the list of files with download functionality.
**Validates: Requirements 11.2, 11.4**

## Error Handling

### Frontend Error Handling

**Drag/Resize Validation Errors:**
- Invalid position (overlap, out of bounds): Revert to original position with visual feedback
- Invalid size (below min, above max): Clamp to valid range or revert
- Network error during save: Show error toast, maintain local state, retry on next operation

**Data Loading Errors:**
- Component data fetch failure: Display error state in component, allow retry
- Layout configuration load failure: Fall back to default layout, log error
- Batch data check timeout: Assume components have data (fail open for visibility)

**State Synchronization Errors:**
- Concurrent modification conflict: Last write wins, show warning to user
- Invalid layout data from backend: Validate and sanitize, fall back to default if unrecoverable

### Backend Error Handling

**Database Errors:**
- Connection failure: Return error to frontend, log for monitoring
- Constraint violation: Return validation error with details
- Serialization error: Log error, return default configuration

**Data Service Errors:**
- Component data not found: Return empty result (hasData = false)
- Query timeout: Return error, allow frontend to retry
- Invalid component type: Return validation error

**Export Service Errors:**
- No components with data: Return user-friendly error, prevent empty file generation
- File write failure: Return error with details, clean up partial files
- Format conversion error: Log error, return error response

### Error Response Format

```go
type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}

// Error codes
const (
    ErrInvalidLayout     = "INVALID_LAYOUT"
    ErrDatabaseError     = "DATABASE_ERROR"
    ErrNotFound          = "NOT_FOUND"
    ErrValidation        = "VALIDATION_ERROR"
    ErrExportFailed      = "EXPORT_FAILED"
    ErrNoData            = "NO_DATA"
)
```

## Testing Strategy

### Dual Testing Approach

This feature requires both unit tests and property-based tests to ensure comprehensive coverage:

**Unit Tests** focus on:
- Specific examples of drag/resize operations
- Edge cases (minimum/maximum sizes, boundary positions)
- Error conditions (invalid positions, load failures)
- Integration between components (layout engine + component manager)
- UI rendering for specific states

**Property-Based Tests** focus on:
- Universal properties that hold for all inputs
- Comprehensive input coverage through randomization
- State invariants across operations
- Round-trip properties (save/load, serialize/deserialize)

### Property-Based Testing Configuration

**Library Selection:**
- Frontend (TypeScript): Use `fast-check` library for property-based testing
- Backend (Go): Use `gopter` library for property-based testing

**Test Configuration:**
- Minimum 100 iterations per property test (due to randomization)
- Each property test must reference its design document property
- Tag format: `Feature: dashboard-drag-drop-layout, Property {number}: {property_text}`

**Example Property Test Structure (TypeScript):**
```typescript
import fc from 'fast-check';

describe('Feature: dashboard-drag-drop-layout', () => {
  it('Property 1: Drag Operation Persistence', () => {
    fc.assert(
      fc.property(
        fc.record({
          componentId: fc.string(),
          startX: fc.integer(0, 23),
          startY: fc.integer(0, 100),
          endX: fc.integer(0, 23),
          endY: fc.integer(0, 100),
        }),
        (testCase) => {
          // Test that drag operation persists position
          // Feature: dashboard-drag-drop-layout, Property 1
        }
      ),
      { numRuns: 100 }
    );
  });
});
```

**Example Property Test Structure (Go):**
```go
import "github.com/leanovate/gopter"

func TestProperty13_LayoutConfigurationRoundTrip(t *testing.T) {
    // Feature: dashboard-drag-drop-layout, Property 13: Layout Configuration Round-Trip
    properties := gopter.NewProperties(nil)
    
    properties.Property("save then load produces equivalent config", 
        prop.ForAll(
            func(config LayoutConfiguration) bool {
                // Test round-trip property
                saved := layoutService.SaveLayout(config)
                loaded := layoutService.LoadLayout(config.UserID)
                return reflect.DeepEqual(config, loaded)
            },
            genLayoutConfiguration(),
        ),
    )
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### Unit Test Coverage

**Frontend Unit Tests:**
- DashboardContainer state management
- LayoutEngine grid calculations and collision detection
- ComponentManager instance creation and pagination
- DraggableComponent event handling
- PaginationControl navigation
- LayoutEditor mode switching

**Backend Unit Tests:**
- LayoutService CRUD operations
- DataService component data retrieval
- ExportService filtering and export generation
- Database schema validation
- Error handling for all services

### Integration Tests

**Frontend Integration:**
- Complete drag-drop workflow (click → drag → drop → save)
- Complete resize workflow (grab handle → resize → release → save)
- Mode switching with state preservation
- Component visibility based on data availability
- Export workflow with component filtering

**Backend Integration:**
- Layout save/load with database
- Data service integration with export service
- Error propagation through service layers

### Test Data Generators

**Property Test Generators:**
```typescript
// TypeScript generators for fast-check
const genComponentType = fc.constantFrom('metrics', 'table', 'image', 'insights', 'file_download');
const genLayoutItem = fc.record({
  i: fc.string(),
  x: fc.integer(0, 23),
  y: fc.integer(0, 100),
  w: fc.integer(1, 24),
  h: fc.integer(1, 20),
  type: genComponentType,
  instanceIdx: fc.integer(0, 10),
});
const genLayoutConfiguration = fc.record({
  id: fc.uuid(),
  userId: fc.uuid(),
  isLocked: fc.boolean(),
  items: fc.array(genLayoutItem, { minLength: 1, maxLength: 20 }),
});
```

```go
// Go generators for gopter
func genLayoutItem() gopter.Gen {
    return gopter.CombineGens(
        gen.Identifier(),
        gen.IntRange(0, 23),
        gen.IntRange(0, 100),
        gen.IntRange(1, 24),
        gen.IntRange(1, 20),
    ).Map(func(vals []interface{}) LayoutItem {
        return LayoutItem{
            I: vals[0].(string),
            X: vals[1].(int),
            Y: vals[2].(int),
            W: vals[3].(int),
            H: vals[4].(int),
        }
    })
}
```

### Manual Testing Checklist

- [ ] Drag components to various positions and verify persistence
- [ ] Resize components to min/max boundaries
- [ ] Test pagination with multiple instances
- [ ] Lock/unlock layout and verify behavior changes
- [ ] Test with empty components in both modes
- [ ] Export dashboard with mixed empty/non-empty components
- [ ] Test responsive behavior at different screen sizes
- [ ] Verify visual feedback during all interactions
- [ ] Test error scenarios (network failures, invalid data)
- [ ] Cross-browser compatibility (Chrome, Firefox, Safari, Edge)
