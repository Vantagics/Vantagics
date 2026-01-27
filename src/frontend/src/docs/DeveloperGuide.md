# Dashboard Drag-Drop Layout - Developer Guide

## Overview

This guide provides comprehensive documentation for developers working with the Dashboard Drag-Drop Layout system. It covers architecture, APIs, extension patterns, and best practices.

## Architecture Overview

### System Components

```
Dashboard Drag-Drop Layout System
├── Backend Infrastructure
│   ├── Database Layer (Go)
│   │   ├── LayoutService - Layout persistence
│   │   ├── FileService - File management
│   │   ├── DataService - Data availability checks
│   │   └── ExportService - Dashboard export
│   └── Wails Bridge - Frontend-Backend communication
├── Frontend Core (TypeScript/React)
│   ├── Layout Engine - Grid calculations and positioning
│   ├── Component Manager - Component registry and lifecycle
│   ├── Visibility Manager - Component visibility logic
│   └── Dashboard Container - Main orchestration component
├── Draggable Components
│   ├── DraggableComponent - Base wrapper for drag/resize
│   ├── Component Types - Metrics, Table, Image, Insights, Files
│   ├── Pagination Control - Multi-instance pagination
│   └── Layout Editor - Edit mode toolbar
├── UI/UX Polish
│   ├── Visual Feedback - Drag/resize visual indicators
│   ├── Loading States - Component loading management
│   ├── Error Handling - User-friendly error system
│   ├── Responsive Layout - Multi-device support
│   ├── Keyboard Shortcuts - Power user shortcuts
│   └── Accessibility - Screen reader and keyboard support
└── Testing & Quality
    ├── Unit Tests - Component and utility tests
    ├── Property-Based Tests - Correctness verification
    └── Integration Tests - End-to-end workflows
```

### Data Flow

```
User Interaction → Dashboard Container → Layout Engine → Component Manager
                                    ↓
Backend Services ← Wails Bridge ← State Updates → Visual Feedback
                                    ↓
Database Persistence ← Layout Service ← Component Updates → UI Updates
```

## Core APIs

### Layout Engine (`src/frontend/src/utils/LayoutEngine.ts`)

The Layout Engine handles all grid calculations, positioning, and collision detection.

```typescript
import LayoutEngine, { LayoutItem, GridConfig } from '../utils/LayoutEngine';

const layoutEngine = new LayoutEngine(gridConfig);

// Calculate valid position for component
const position = layoutEngine.calculatePosition(item, x, y, otherItems);

// Snap coordinates to grid
const snapped = layoutEngine.snapToGrid(x, y);

// Detect collisions between components
const collisions = layoutEngine.detectCollisions(item, otherItems);

// Compact layout to remove gaps
const result = layoutEngine.compactLayout(items);

// Validate item constraints
const validItem = layoutEngine.validateItemConstraints(item);
```

#### Key Methods

- `calculatePosition(item, x, y, otherItems)` - Find valid position avoiding collisions
- `snapToGrid(x, y)` - Snap coordinates to grid boundaries
- `detectCollisions(item, otherItems)` - Check for component overlaps
- `compactLayout(items)` - Remove vertical gaps in layout
- `validateItemConstraints(item)` - Enforce size constraints

### Component Manager (`src/frontend/src/utils/ComponentManager.ts`)

Manages component registration, instantiation, and lifecycle.

```typescript
import ComponentManager, { ComponentType } from '../utils/ComponentManager';

const componentManager = new ComponentManager();

// Register custom component type
componentManager.registerComponent(ComponentType.CUSTOM, {
  displayName: 'Custom Component',
  config: {
    defaultSize: { w: 6, h: 4 },
    minSize: { w: 3, h: 2 },
    maxSize: { w: 12, h: 8 },
    supportsPagination: true,
  },
  factory: (props) => <CustomComponent {...props} />,
});

// Create component instance
const instance = componentManager.createInstance(ComponentType.METRICS, layoutItem);

// Get pagination state
const paginationState = componentManager.getPaginationState(ComponentType.METRICS);
```

#### Component Registration

```typescript
interface ComponentConfig {
  defaultSize: { w: number; h: number };
  minSize?: { w: number; h: number };
  maxSize?: { w: number; h: number };
  supportsPagination: boolean;
  allowMultiple?: boolean;
  category?: string;
}

interface ComponentEntry {
  type: ComponentType;
  displayName: string;
  config: ComponentConfig;
  factory: ComponentFactory;
}
```

### Dashboard Container (`src/frontend/src/components/DashboardContainer.tsx`)

Main orchestration component that manages layout state and user interactions.

```typescript
import { DashboardContainer, DashboardLayout } from './DashboardContainer';

<DashboardContainer
  initialLayout={layout}
  initialEditMode={false}
  initialLocked={true}
  onLayoutChange={(layout) => console.log('Layout changed:', layout)}
  onEditModeChange={(isEditMode) => console.log('Edit mode:', isEditMode)}
  onLockStateChange={(isLocked) => console.log('Lock state:', isLocked)}
  gridConfig={{ columns: 24, rowHeight: 30 }}
  className="custom-dashboard"
/>
```

#### Props Interface

```typescript
interface DashboardContainerProps {
  initialLayout?: DashboardLayout;
  initialEditMode?: boolean;
  initialLocked?: boolean;
  onLayoutChange?: (layout: DashboardLayout) => void;
  onEditModeChange?: (isEditMode: boolean) => void;
  onLockStateChange?: (isLocked: boolean) => void;
  gridConfig?: Partial<GridConfig>;
  className?: string;
  style?: React.CSSProperties;
}
```

### Draggable Component (`src/frontend/src/components/DraggableComponent.tsx`)

Base wrapper component that adds drag and resize functionality to any component.

```typescript
import { DraggableComponent } from './DraggableComponent';

<DraggableComponent
  layoutItem={item}
  isEditMode={true}
  isLocked={false}
  onDrag={(item, x, y) => handleDrag(item, x, y)}
  onResize={(item, width, height) => handleResize(item, width, height)}
  onRemove={(itemId) => handleRemove(itemId)}
  gridConfig={gridConfig}
>
  <YourComponent />
</DraggableComponent>
```

## Extension Patterns

### Adding New Component Types

1. **Define Component Type**
```typescript
// Add to ComponentType enum
export enum ComponentType {
  // ... existing types
  CUSTOM_CHART = 'custom_chart',
}
```

2. **Create Component Implementation**
```typescript
// src/components/CustomChartComponent.tsx
export const CustomChartComponent: React.FC<ComponentProps> = ({ data, config }) => {
  return (
    <div className="custom-chart-component">
      {/* Your component implementation */}
    </div>
  );
};
```

3. **Create Draggable Wrapper**
```typescript
// src/components/DraggableCustomChart.tsx
export const DraggableCustomChart: React.FC<DraggableComponentProps> = (props) => {
  return (
    <DraggableComponent {...props}>
      <CustomChartComponent />
    </DraggableComponent>
  );
};
```

4. **Register Component**
```typescript
// Register in ComponentManager
componentManager.registerComponent(ComponentType.CUSTOM_CHART, {
  displayName: 'Custom Chart',
  config: {
    defaultSize: { w: 8, h: 6 },
    minSize: { w: 4, h: 3 },
    maxSize: { w: 16, h: 12 },
    supportsPagination: false,
  },
  factory: (props) => <DraggableCustomChart {...props} />,
});
```

5. **Add Data Availability Check**
```typescript
// Backend: Add to DataService
func (ds *DataService) CheckCustomChartHasData(componentID string) (bool, error) {
    // Implementation to check if custom chart has data
    return hasData, nil
}
```

### Custom Layout Algorithms

Extend the Layout Engine with custom positioning algorithms:

```typescript
class CustomLayoutEngine extends LayoutEngine {
  customCompactLayout(items: LayoutItem[]): CompactionResult {
    // Custom compaction algorithm
    const compactedItems = this.applyCustomAlgorithm(items);
    return {
      items: compactedItems,
      changed: true,
    };
  }

  private applyCustomAlgorithm(items: LayoutItem[]): LayoutItem[] {
    // Your custom algorithm implementation
    return items;
  }
}
```

### Custom Visual Feedback

Extend the Visual Feedback system:

```typescript
class CustomVisualFeedback extends VisualFeedback {
  showCustomIndicator(bounds: Bounds, type: string): void {
    const indicator = document.createElement('div');
    indicator.className = `custom-indicator custom-indicator--${type}`;
    // Custom indicator implementation
    this.container?.appendChild(indicator);
  }
}
```

## Backend Integration

### Database Schema

The system uses the following database tables:

```sql
-- Layout configurations
CREATE TABLE layout_configs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    layout_data TEXT NOT NULL, -- JSON
    is_default BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- File metadata
CREATE TABLE file_metadata (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_size INTEGER,
    mime_type TEXT,
    category TEXT, -- 'all_files' or 'user_request_related'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Service Layer

#### Layout Service (`src/database/layout_service.go`)

```go
type LayoutService struct {
    db *sql.DB
}

func (ls *LayoutService) SaveLayout(config LayoutConfig) error {
    // Save layout configuration to database
}

func (ls *LayoutService) LoadLayout(userID string) (*LayoutConfig, error) {
    // Load layout configuration from database
}

func (ls *LayoutService) GetDefaultLayout() (*LayoutConfig, error) {
    // Get default layout configuration
}
```

#### File Service (`src/database/file_service.go`)

```go
type FileService struct {
    db *sql.DB
}

func (fs *FileService) GetFilesByCategory(category string) ([]FileMetadata, error) {
    // Get files by category
}

func (fs *FileService) HasFiles(category string) (bool, error) {
    // Check if category has files
}

func (fs *FileService) DownloadFile(fileID, filename string) (string, error) {
    // Handle file download
}
```

### Wails Bridge

The frontend communicates with the backend through Wails bridge methods:

```go
// app.go
func (a *App) SaveLayout(config LayoutConfig) error {
    return a.layoutService.SaveLayout(config)
}

func (a *App) LoadLayout() (*LayoutConfig, error) {
    return a.layoutService.LoadLayout(a.getCurrentUserID())
}

func (a *App) CheckComponentHasData(componentType, componentID string) (bool, error) {
    return a.dataService.CheckComponentHasData(componentType, componentID)
}
```

## Testing Strategies

### Unit Testing

Test individual components and utilities:

```typescript
// Component tests
describe('DraggableComponent', () => {
  it('should handle drag operations', () => {
    const onDrag = vi.fn();
    render(<DraggableComponent onDrag={onDrag} />);
    // Test drag interaction
  });
});

// Utility tests
describe('LayoutEngine', () => {
  it('should calculate valid positions', () => {
    const engine = new LayoutEngine();
    const position = engine.calculatePosition(item, x, y, otherItems);
    expect(position).toEqual({ x: expectedX, y: expectedY });
  });
});
```

### Property-Based Testing

Verify correctness properties:

```typescript
import fc from 'fast-check';

describe('Layout Properties', () => {
  it('should maintain grid constraints', () => {
    fc.assert(fc.property(
      genLayoutItem(),
      (item) => {
        const engine = new LayoutEngine();
        const snapped = engine.snapToGrid(item.x, item.y);
        expect(snapped.x % engine.getConfig().margin[0]).toBe(0);
      }
    ));
  });
});
```

### Integration Testing

Test complete workflows:

```typescript
describe('Dashboard Integration', () => {
  it('should complete drag-drop workflow', async () => {
    render(<DashboardContainer />);
    
    // Enter edit mode
    fireEvent.click(screen.getByTestId('edit-button'));
    
    // Drag component
    const component = screen.getByTestId('draggable-component');
    fireEvent.mouseDown(component);
    fireEvent.mouseMove(document, { clientX: 200, clientY: 150 });
    fireEvent.mouseUp(document);
    
    // Verify layout saved
    await waitFor(() => {
      expect(mockSaveLayout).toHaveBeenCalled();
    });
  });
});
```

## Performance Optimization

### Rendering Optimization

1. **Memoization**
```typescript
const MemoizedComponent = React.memo(DraggableComponent, (prevProps, nextProps) => {
  return prevProps.layoutItem.x === nextProps.layoutItem.x &&
         prevProps.layoutItem.y === nextProps.layoutItem.y;
});
```

2. **Virtual Scrolling**
```typescript
// For large numbers of components
const VirtualizedDashboard = () => {
  const visibleComponents = useMemo(() => {
    return components.filter(isComponentVisible);
  }, [components, viewport]);
  
  return visibleComponents.map(component => 
    <DraggableComponent key={component.id} {...component} />
  );
};
```

3. **Debounced Updates**
```typescript
const debouncedSave = useMemo(
  () => debounce(saveLayout, 1000),
  [saveLayout]
);
```

### Animation Performance

1. **CSS Transforms**
```css
.draggable-component {
  transform: translate3d(var(--x), var(--y), 0);
  will-change: transform;
}
```

2. **RequestAnimationFrame**
```typescript
const animateComponent = (element: HTMLElement, targetX: number, targetY: number) => {
  const animate = () => {
    // Update position
    element.style.transform = `translate3d(${targetX}px, ${targetY}px, 0)`;
    requestAnimationFrame(animate);
  };
  requestAnimationFrame(animate);
};
```

## Best Practices

### Component Development

1. **Separation of Concerns**
   - Keep layout logic separate from component logic
   - Use composition over inheritance
   - Implement single responsibility principle

2. **State Management**
   - Use local state for UI interactions
   - Use global state for layout configuration
   - Implement optimistic updates with rollback

3. **Error Handling**
   - Always handle async operations
   - Provide fallback UI states
   - Log errors for debugging

### Layout Design

1. **Grid System**
   - Use consistent grid units
   - Respect minimum/maximum sizes
   - Consider responsive breakpoints

2. **User Experience**
   - Provide visual feedback for all interactions
   - Implement undo/redo functionality
   - Support keyboard navigation

3. **Performance**
   - Minimize re-renders
   - Use efficient algorithms
   - Implement lazy loading

### Accessibility

1. **ARIA Labels**
```typescript
<div
  role="button"
  aria-label="Drag handle for metrics component"
  aria-describedby="drag-instructions"
  tabIndex={0}
>
```

2. **Keyboard Support**
```typescript
const handleKeyDown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'ArrowUp':
      moveComponent('up');
      break;
    case 'Enter':
      activateComponent();
      break;
  }
};
```

3. **Screen Reader Support**
```typescript
const announceChange = (message: string) => {
  const announcer = document.getElementById('sr-announcer');
  if (announcer) {
    announcer.textContent = message;
  }
};
```

## Troubleshooting

### Common Issues

1. **Components Not Dragging**
   - Check if edit mode is enabled
   - Verify drag handles are properly configured
   - Ensure event handlers are attached

2. **Layout Not Saving**
   - Check backend service connectivity
   - Verify user permissions
   - Check for validation errors

3. **Performance Issues**
   - Profile component re-renders
   - Check for memory leaks
   - Optimize animation performance

### Debugging Tools

1. **Layout Debugger**
```typescript
const LayoutDebugger = () => {
  const [showGrid, setShowGrid] = useState(false);
  
  return (
    <div className={`layout-debugger ${showGrid ? 'show-grid' : ''}`}>
      <button onClick={() => setShowGrid(!showGrid)}>
        Toggle Grid
      </button>
    </div>
  );
};
```

2. **Performance Monitor**
```typescript
const usePerformanceMonitor = () => {
  useEffect(() => {
    const observer = new PerformanceObserver((list) => {
      list.getEntries().forEach((entry) => {
        console.log('Performance:', entry);
      });
    });
    observer.observe({ entryTypes: ['measure'] });
    
    return () => observer.disconnect();
  }, []);
};
```

## Migration Guide

### From Static Layout

1. **Wrap Components**
```typescript
// Before
<MetricsComponent />

// After
<DraggableComponent layoutItem={item}>
  <MetricsComponent />
</DraggableComponent>
```

2. **Add Layout State**
```typescript
const [layout, setLayout] = useState<LayoutItem[]>([]);
const [isEditMode, setIsEditMode] = useState(false);
```

3. **Implement Persistence**
```typescript
const saveLayout = async (newLayout: LayoutItem[]) => {
  await window.go.main.App.SaveLayout({
    items: newLayout,
    gridConfig: DEFAULT_GRID_CONFIG,
    metadata: { version: '1.0.0', createdAt: Date.now(), updatedAt: Date.now() }
  });
};
```

### Version Compatibility

The system maintains backward compatibility through:

1. **Layout Version Detection**
```typescript
const migrateLayout = (layout: any): DashboardLayout => {
  if (!layout.metadata?.version) {
    // Migrate from v0 to v1
    return migrateFromV0(layout);
  }
  return layout;
};
```

2. **Graceful Degradation**
```typescript
const ComponentRenderer = ({ type, ...props }) => {
  const Component = componentManager.getComponent(type);
  if (!Component) {
    return <FallbackComponent type={type} />;
  }
  return <Component {...props} />;
};
```

This developer guide provides the foundation for working with and extending the Dashboard Drag-Drop Layout system. For specific implementation details, refer to the individual component documentation and test files.