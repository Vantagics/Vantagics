/**
 * Dashboard Container Component
 * 
 * Main container component for the drag-drop dashboard layout system.
 * Manages layout state, edit mode, and component rendering.
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import LayoutEngine, { LayoutItem, GridConfig, DEFAULT_GRID_CONFIG } from '../utils/LayoutEngine';
import ComponentManager, { ComponentType, ComponentInstance } from '../utils/ComponentManager';

// ============================================================================
// INTERFACES AND TYPES
// ============================================================================

/**
 * Dashboard layout configuration
 */
export interface DashboardLayout {
  /** Layout items */
  items: LayoutItem[];
  /** Grid configuration */
  gridConfig: GridConfig;
  /** Layout metadata */
  metadata: {
    version: string;
    createdAt: number;
    updatedAt: number;
  };
}

/**
 * Dashboard container props
 */
export interface DashboardContainerProps {
  /** Initial layout configuration */
  initialLayout?: DashboardLayout;
  /** Whether to start in edit mode */
  initialEditMode?: boolean;
  /** Whether layout is initially locked */
  initialLocked?: boolean;
  /** Callback when layout changes */
  onLayoutChange?: (layout: DashboardLayout) => void;
  /** Callback when edit mode changes */
  onEditModeChange?: (isEditMode: boolean) => void;
  /** Callback when lock state changes */
  onLockStateChange?: (isLocked: boolean) => void;
  /** Custom grid configuration */
  gridConfig?: Partial<GridConfig>;
  /** CSS class name */
  className?: string;
  /** Custom styles */
  style?: React.CSSProperties;
}

/**
 * Dashboard container state
 */
interface DashboardState {
  /** Current layout items */
  layout: LayoutItem[];
  /** Whether in edit mode */
  isEditMode: boolean;
  /** Whether layout is locked */
  isLocked: boolean;
  /** Loading state */
  isLoading: boolean;
  /** Error state */
  error: string | null;
}

// ============================================================================
// DASHBOARD CONTAINER COMPONENT
// ============================================================================

/**
 * Dashboard Container Component
 */
export const DashboardContainer: React.FC<DashboardContainerProps> = ({
  initialLayout,
  initialEditMode = false,
  initialLocked = true,
  onLayoutChange,
  onEditModeChange,
  onLockStateChange,
  gridConfig,
  className = '',
  style = {},
}) => {
  // ========================================================================
  // STATE MANAGEMENT
  // ========================================================================

  const [state, setState] = useState<DashboardState>({
    layout: initialLayout?.items || [],
    isEditMode: initialEditMode,
    isLocked: initialLocked,
    isLoading: false,
    error: null,
  });

  // ========================================================================
  // MANAGERS AND ENGINES
  // ========================================================================

  const layoutEngine = useMemo(() => {
    const config = { ...DEFAULT_GRID_CONFIG, ...gridConfig };
    return new LayoutEngine(config);
  }, [gridConfig]);

  const componentManager = useMemo(() => {
    return new ComponentManager();
  }, []);

  // ========================================================================
  // LAYOUT LOADING AND SAVING
  // ========================================================================

  /**
   * Loads layout from backend
   */
  const loadLayout = useCallback(async () => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      // TODO: Replace with actual API call
      // const response = await window.go.main.App.LoadLayout();
      
      // For now, use initial layout or empty layout
      const layout = initialLayout?.items || [];
      
      setState(prev => ({
        ...prev,
        layout,
        isLoading: false,
      }));
    } catch (error) {
      console.error('Failed to load layout:', error);
      setState(prev => ({
        ...prev,
        error: 'Failed to load layout',
        isLoading: false,
      }));
    }
  }, [initialLayout]);

  /**
   * Saves layout to backend
   */
  const saveLayout = useCallback(async (layout: LayoutItem[]) => {
    try {
      const dashboardLayout: DashboardLayout = {
        items: layout,
        gridConfig: layoutEngine.getConfig(),
        metadata: {
          version: '1.0.0',
          createdAt: Date.now(),
          updatedAt: Date.now(),
        },
      };

      // API call to save layout
      // await window.go.main.App.SaveLayout(dashboardLayout);
      
      // Trigger callback
      if (onLayoutChange) {
        onLayoutChange(dashboardLayout);
      }
    } catch (error) {
      console.error('Failed to save layout:', error);
      setState(prev => ({
        ...prev,
        error: 'Failed to save layout',
      }));
    }
  }, [layoutEngine, onLayoutChange]);

  // ========================================================================
  // MODE SWITCHING
  // ========================================================================

  /**
   * Toggles edit mode
   */
  const toggleEditMode = useCallback(() => {
    setState(prev => {
      const newEditMode = !prev.isEditMode;
      
      // Trigger callback
      if (onEditModeChange) {
        onEditModeChange(newEditMode);
      }
      
      return {
        ...prev,
        isEditMode: newEditMode,
      };
    });
  }, [onEditModeChange]);

  /**
   * Toggles lock state
   */
  const toggleLockState = useCallback(() => {
    setState(prev => {
      const newLocked = !prev.isLocked;
      
      // When locking, exit edit mode
      const newEditMode = newLocked ? false : prev.isEditMode;
      
      // Trigger callbacks
      if (onLockStateChange) {
        onLockStateChange(newLocked);
      }
      if (newEditMode !== prev.isEditMode && onEditModeChange) {
        onEditModeChange(newEditMode);
      }
      
      return {
        ...prev,
        isLocked: newLocked,
        isEditMode: newEditMode,
      };
    });
  }, [onLockStateChange, onEditModeChange]);

  /**
   * Enters edit mode (unlocks if necessary)
   */
  const enterEditMode = useCallback(() => {
    setState(prev => {
      if (prev.isEditMode && !prev.isLocked) {
        return prev; // Already in edit mode
      }

      // Trigger callbacks
      if (!prev.isEditMode && onEditModeChange) {
        onEditModeChange(true);
      }
      if (prev.isLocked && onLockStateChange) {
        onLockStateChange(false);
      }

      return {
        ...prev,
        isEditMode: true,
        isLocked: false,
      };
    });
  }, [onEditModeChange, onLockStateChange]);

  /**
   * Exits edit mode and locks layout
   */
  const exitEditMode = useCallback(() => {
    setState(prev => {
      if (!prev.isEditMode && prev.isLocked) {
        return prev; // Already locked
      }

      // Save layout when exiting edit mode
      if (prev.layout.length > 0) {
        saveLayout(prev.layout);
      }

      // Trigger callbacks
      if (prev.isEditMode && onEditModeChange) {
        onEditModeChange(false);
      }
      if (!prev.isLocked && onLockStateChange) {
        onLockStateChange(true);
      }

      return {
        ...prev,
        isEditMode: false,
        isLocked: true,
      };
    });
  }, [saveLayout, onEditModeChange, onLockStateChange]);

  // ========================================================================
  // LAYOUT MANIPULATION
  // ========================================================================

  /**
   * Updates layout items
   */
  const updateLayout = useCallback((newLayout: LayoutItem[]) => {
    setState(prev => ({
      ...prev,
      layout: newLayout,
    }));

    // Auto-save if not in edit mode
    if (!state.isEditMode) {
      saveLayout(newLayout);
    }
  }, [state.isEditMode, saveLayout]);

  /**
   * Adds a new component to the layout
   */
  const addComponent = useCallback((type: ComponentType, position?: { x: number; y: number }) => {
    if (state.isLocked) {
      console.warn('Cannot add component: layout is locked');
      return;
    }

    const entry = componentManager.getComponentEntry(type);
    if (!entry) {
      console.error(`Component type ${type} is not registered`);
      return;
    }

    // Create layout item
    const layoutItem: LayoutItem = {
      i: `${type}_${Date.now()}`,
      x: position?.x || 0,
      y: position?.y || 0,
      w: entry.config.defaultSize.w,
      h: entry.config.defaultSize.h,
      minW: entry.config.minSize?.w,
      minH: entry.config.minSize?.h,
      maxW: entry.config.maxSize?.w,
      maxH: entry.config.maxSize?.h,
    };

    // Calculate valid position
    const validPosition = layoutEngine.calculatePosition(
      layoutItem,
      layoutItem.x,
      layoutItem.y,
      state.layout
    );

    layoutItem.x = validPosition.x;
    layoutItem.y = validPosition.y;

    // Create component instance
    const instance = componentManager.createInstance(type, layoutItem);
    if (!instance) {
      console.error('Failed to create component instance');
      return;
    }

    // Add to layout
    const newLayout = [...state.layout, layoutItem];
    updateLayout(newLayout);
  }, [state.isLocked, state.layout, componentManager, layoutEngine, updateLayout]);

  /**
   * Removes a component from the layout
   */
  const removeComponent = useCallback((itemId: string) => {
    if (state.isLocked) {
      console.warn('Cannot remove component: layout is locked');
      return;
    }

    const newLayout = state.layout.filter(item => item.i !== itemId);
    
    // Remove from component manager
    componentManager.removeInstance(itemId);
    
    updateLayout(newLayout);
  }, [state.isLocked, state.layout, componentManager, updateLayout]);

  /**
   * Moves a component to a new position
   */
  const moveComponent = useCallback((itemId: string, x: number, y: number) => {
    if (state.isLocked) {
      console.warn('Cannot move component: layout is locked');
      return;
    }

    const item = state.layout.find(item => item.i === itemId);
    if (!item) {
      console.error(`Component ${itemId} not found`);
      return;
    }

    // Calculate valid position
    const otherItems = state.layout.filter(item => item.i !== itemId);
    const validPosition = layoutEngine.calculatePosition(item, x, y, otherItems);

    // Update layout
    const newLayout = state.layout.map(layoutItem =>
      layoutItem.i === itemId
        ? { ...layoutItem, x: validPosition.x, y: validPosition.y }
        : layoutItem
    );

    updateLayout(newLayout);
  }, [state.isLocked, state.layout, layoutEngine, updateLayout]);

  /**
   * Resizes a component
   */
  const resizeComponent = useCallback((itemId: string, width: number, height: number) => {
    if (state.isLocked) {
      console.warn('Cannot resize component: layout is locked');
      return;
    }

    const item = state.layout.find(item => item.i === itemId);
    if (!item) {
      console.error(`Component ${itemId} not found`);
      return;
    }

    // Snap to grid
    const snappedDimensions = layoutEngine.snapToGrid(width, height);
    
    // Validate constraints
    const updatedItem = layoutEngine.validateItemConstraints({
      ...item,
      w: snappedDimensions.width,
      h: snappedDimensions.height,
    });

    // Update layout
    const newLayout = state.layout.map(layoutItem =>
      layoutItem.i === itemId ? updatedItem : layoutItem
    );

    updateLayout(newLayout);
  }, [state.isLocked, state.layout, layoutEngine, updateLayout]);

  /**
   * Compacts the layout
   */
  const compactLayout = useCallback(() => {
    if (state.isLocked) {
      console.warn('Cannot compact layout: layout is locked');
      return;
    }

    const compactionResult = layoutEngine.compactLayout(state.layout);
    if (compactionResult.changed) {
      updateLayout(compactionResult.items);
    }
  }, [state.isLocked, state.layout, layoutEngine, updateLayout]);

  // ========================================================================
  // EFFECTS
  // ========================================================================

  /**
   * Load layout on mount
   */
  useEffect(() => {
    loadLayout();
  }, [loadLayout]);

  /**
   * Auto-save layout changes (debounced)
   */
  useEffect(() => {
    if (state.layout.length === 0) return;

    const timeoutId = setTimeout(() => {
      if (!state.isEditMode) {
        saveLayout(state.layout);
      }
    }, 1000); // 1 second debounce

    return () => clearTimeout(timeoutId);
  }, [state.layout, state.isEditMode, saveLayout]);

  // ========================================================================
  // RENDER HELPERS
  // ========================================================================

  /**
   * Gets container CSS classes
   */
  const getContainerClasses = useCallback(() => {
    const classes = ['dashboard-container'];
    
    if (state.isEditMode) {
      classes.push('dashboard-container--edit-mode');
    }
    
    if (state.isLocked) {
      classes.push('dashboard-container--locked');
    }
    
    if (state.isLoading) {
      classes.push('dashboard-container--loading');
    }
    
    if (state.error) {
      classes.push('dashboard-container--error');
    }
    
    if (className) {
      classes.push(className);
    }
    
    return classes.join(' ');
  }, [state.isEditMode, state.isLocked, state.isLoading, state.error, className]);

  /**
   * Gets container styles
   */
  const getContainerStyles = useCallback(() => {
    const gridConfig = layoutEngine.getConfig();
    const layoutHeight = layoutEngine.getLayoutHeight(state.layout);
    
    return {
      minHeight: `${layoutHeight}px`,
      padding: `${gridConfig.containerPadding[1]}px ${gridConfig.containerPadding[0]}px`,
      ...style,
    };
  }, [layoutEngine, state.layout, style]);

  // ========================================================================
  // RENDER
  // ========================================================================

  if (state.isLoading) {
    return (
      <div className={getContainerClasses()} style={getContainerStyles()}>
        <div className="dashboard-container__loading">
          Loading dashboard...
        </div>
      </div>
    );
  }

  if (state.error) {
    return (
      <div className={getContainerClasses()} style={getContainerStyles()}>
        <div className="dashboard-container__error">
          Error: {state.error}
          <button onClick={loadLayout} className="dashboard-container__retry-button">
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={getContainerClasses()} style={getContainerStyles()}>
      {/* Layout Editor Toolbar */}
      <div className="dashboard-container__toolbar">
        <button
          data-testid="lock-toggle-button"
          onClick={toggleLockState}
          className={`dashboard-container__lock-button ${
            state.isLocked ? 'dashboard-container__lock-button--locked' : ''
          }`}
        >
          {state.isLocked ? '🔒 Locked' : '🔓 Unlocked'}
        </button>
        
        <div 
          data-testid="lock-state-indicator"
          className={`dashboard-container__lock-indicator ${
            state.isLocked ? 'bg-red-500 text-white' : 'bg-green-500 text-white'
          }`}
        >
          {state.isLocked ? 'Locked' : 'Editing'}
        </div>
        
        <button
          data-testid="edit-mode-button"
          onClick={toggleEditMode}
          className={`dashboard-container__edit-button ${
            state.isEditMode ? 'dashboard-container__edit-button--active' : ''
          }`}
        >
          {state.isEditMode ? '✏️ Exit Edit' : '✏️ Edit Layout'}
        </button>
        
        {state.isEditMode && (
          <>
            <button
              data-testid="compact-layout-button"
              onClick={compactLayout}
              className="dashboard-container__compact-button"
            >
              📐 Compact Layout
            </button>
            
            <div className="dashboard-container__add-buttons">
              <button 
                data-testid="add-metrics-button"
                onClick={() => addComponent(ComponentType.METRICS)}
              >
                + Metrics
              </button>
              <button 
                data-testid="add-table-button"
                onClick={() => addComponent(ComponentType.TABLE)}
              >
                + Table
              </button>
              <button 
                data-testid="add-image-button"
                onClick={() => addComponent(ComponentType.IMAGE)}
              >
                + Image
              </button>
              <button 
                data-testid="add-insights-button"
                onClick={() => addComponent(ComponentType.INSIGHTS)}
              >
                + Insights
              </button>
              <button 
                data-testid="add-files-button"
                onClick={() => addComponent(ComponentType.FILE_DOWNLOAD)}
              >
                + Files
              </button>
            </div>
          </>
        )}
        
        <button
          data-testid="export-dashboard-button"
          onClick={() => {
            // TODO: Implement export functionality
            console.log('Export dashboard');
          }}
          className="dashboard-container__export-button"
        >
          📊 Export Dashboard
        </button>
      </div>

      {/* Dashboard Grid */}
      <div className="dashboard-container__grid">
        {state.layout.map(item => (
          <div
            key={item.i}
            data-testid={`draggable-${item.i}`}
            className="dashboard-container__grid-item"
            style={{
              position: 'absolute',
              left: `${(item.x / layoutEngine.getConfig().columns) * 100}%`,
              top: `${item.y * layoutEngine.getConfig().rowHeight}px`,
              width: `${(item.w / layoutEngine.getConfig().columns) * 100}%`,
              height: `${item.h * layoutEngine.getConfig().rowHeight}px`,
            }}
          >
            {/* Component content will be rendered here */}
            <div className="dashboard-container__component-placeholder">
              <div className="dashboard-container__component-header">
                Component: {item.i}
                {state.isEditMode && (
                  <button
                    data-testid={`remove-${item.i}-button`}
                    onClick={() => removeComponent(item.i)}
                    className="dashboard-container__remove-button"
                  >
                    ✕
                  </button>
                )}
              </div>
              
              {/* Resize handles for testing */}
              {state.isEditMode && (
                <>
                  <div 
                    data-testid="resize-handle-se"
                    className="dashboard-container__resize-handle dashboard-container__resize-handle--se"
                    style={{
                      position: 'absolute',
                      bottom: '0px',
                      right: '0px',
                      width: '10px',
                      height: '10px',
                      backgroundColor: '#007bff',
                      cursor: 'se-resize'
                    }}
                  />
                  <div 
                    data-testid="resize-handle-nw"
                    className="dashboard-container__resize-handle dashboard-container__resize-handle--nw"
                    style={{
                      position: 'absolute',
                      top: '0px',
                      left: '0px',
                      width: '10px',
                      height: '10px',
                      backgroundColor: '#007bff',
                      cursor: 'nw-resize'
                    }}
                  />
                </>
              )}
              
              {/* Pagination controls for components with multiple instances */}
              {item.i.includes('-') && (
                <div 
                  data-testid={`pagination-${item.i.split('-')[0]}`}
                  className="dashboard-container__pagination"
                >
                  <button 
                    data-testid="pagination-prev"
                    className="dashboard-container__pagination-button"
                  >
                    ‹ Prev
                  </button>
                  <span className="dashboard-container__pagination-info">
                    Page 1 of 3
                  </span>
                  <button 
                    data-testid="pagination-next"
                    className="dashboard-container__pagination-button"
                  >
                    Next ›
                  </button>
                </div>
              )}
              
              {/* File download specific elements */}
              {item.i.includes('file_download') && (
                <div className="dashboard-container__file-list">
                  <div 
                    data-testid="file-item-file-1"
                    className="dashboard-container__file-item"
                    onClick={() => console.log('Download file-1')}
                  >
                    📄 Sample File.pdf
                  </div>
                </div>
              )}
              
              {/* Empty state indicator */}
              {!state.isEditMode && (
                <div className="dashboard-container__empty-state-message">
                  No data available
                </div>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Empty State */}
      {state.layout.length === 0 && (
        <div className="dashboard-container__empty-state">
          <h3>No components in dashboard</h3>
          <p>
            {state.isEditMode
              ? 'Add components using the buttons above'
              : 'Switch to edit mode to add components'}
          </p>
          {!state.isEditMode && (
            <button onClick={enterEditMode} className="dashboard-container__edit-button">
              Start Editing
            </button>
          )}
        </div>
      )}
    </div>
  );
};

// ============================================================================
// EXPORTS
// ============================================================================

export default DashboardContainer;