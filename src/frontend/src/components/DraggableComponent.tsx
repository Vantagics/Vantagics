/**
 * DraggableComponent Wrapper
 * 
 * A wrapper component that adds drag and resize functionality to dashboard components.
 * Provides visual feedback, handles mouse events, and integrates with the layout engine.
 */

import React, { useState, useRef, useCallback, useEffect, useMemo } from 'react';
import { ComponentInstance } from '../utils/ComponentManager';

// ============================================================================
// INTERFACES AND TYPES
// ============================================================================

/**
 * Drag state information
 */
interface DragState {
  /** Whether component is currently being dragged */
  isDragging: boolean;
  /** Initial mouse position when drag started */
  startPosition: { x: number; y: number };
  /** Initial component position when drag started */
  startLayout: { x: number; y: number };
  /** Current drag offset */
  dragOffset: { x: number; y: number };
}

/**
 * Resize state information
 */
interface ResizeState {
  /** Whether component is currently being resized */
  isResizing: boolean;
  /** Which resize handle is being used */
  resizeHandle: ResizeHandle | null;
  /** Initial mouse position when resize started */
  startPosition: { x: number; y: number };
  /** Initial component dimensions when resize started */
  startLayout: { x: number; y: number; w: number; h: number };
  /** Current resize offset */
  resizeOffset: { x: number; y: number };
}

/**
 * Resize handle positions
 */
type ResizeHandle = 
  | 'nw' | 'n' | 'ne'
  | 'w'  |      'e'
  | 'sw' | 's' | 'se';

/**
 * DraggableComponent props
 */
export interface DraggableComponentProps {
  /** Component instance data */
  instance: ComponentInstance;
  /** Whether component is in edit mode */
  isEditMode: boolean;
  /** Whether layout is locked */
  isLocked: boolean;
  /** Grid configuration for snapping */
  gridConfig: {
    columns: number;
    rowHeight: number;
    columnWidth: number;
    margin: [number, number];
  };
  /** Callback when component is dragged */
  onDrag?: (instanceId: string, x: number, y: number) => void;
  /** Callback when drag operation completes */
  onDragStop?: (instanceId: string, x: number, y: number) => void;
  /** Callback when component is resized */
  onResize?: (instanceId: string, width: number, height: number) => void;
  /** Callback when resize operation completes */
  onResizeStop?: (instanceId: string, width: number, height: number) => void;
  /** Callback when component is selected */
  onSelect?: (instanceId: string) => void;
  /** Whether component is currently selected */
  isSelected?: boolean;
  /** Child component to render */
  children: React.ReactNode;
  /** Custom CSS classes */
  className?: string;
  /** Custom styles */
  style?: React.CSSProperties;
}

// ============================================================================
// DRAGGABLE COMPONENT
// ============================================================================

/**
 * DraggableComponent wrapper that adds drag and resize functionality
 */
export const DraggableComponent: React.FC<DraggableComponentProps> = ({
  instance,
  isEditMode,
  isLocked,
  gridConfig,
  onDrag,
  onDragStop,
  onResize,
  onResizeStop,
  onSelect,
  isSelected = false,
  children,
  className = '',
  style = {},
}) => {
  // ========================================================================
  // STATE MANAGEMENT
  // ========================================================================

  const [dragState, setDragState] = useState<DragState>({
    isDragging: false,
    startPosition: { x: 0, y: 0 },
    startLayout: { x: 0, y: 0 },
    dragOffset: { x: 0, y: 0 },
  });

  const [resizeState, setResizeState] = useState<ResizeState>({
    isResizing: false,
    resizeHandle: null,
    startPosition: { x: 0, y: 0 },
    startLayout: { x: 0, y: 0, w: 0, h: 0 },
    resizeOffset: { x: 0, y: 0 },
  });

  const [isHovered, setIsHovered] = useState(false);

  // ========================================================================
  // REFS
  // ========================================================================

  const componentRef = useRef<HTMLDivElement>(null);
  const dragHandleRef = useRef<HTMLDivElement>(null);

  // ========================================================================
  // COMPUTED VALUES
  // ========================================================================

  /**
   * Whether drag/resize operations are enabled
   */
  const isInteractive = useMemo(() => {
    return isEditMode && !isLocked;
  }, [isEditMode, isLocked]);

  /**
   * Component position and size in pixels
   */
  const componentStyle = useMemo(() => {
    const { layout } = instance;
    const { columnWidth, rowHeight, margin } = gridConfig;

    let x = layout.x * columnWidth + layout.x * margin[0];
    let y = layout.y * rowHeight + layout.y * margin[1];
    let width = layout.w * columnWidth + (layout.w - 1) * margin[0];
    let height = layout.h * rowHeight + (layout.h - 1) * margin[1];

    // Apply drag offset
    if (dragState.isDragging) {
      x += dragState.dragOffset.x;
      y += dragState.dragOffset.y;
    }

    // Apply resize offset
    if (resizeState.isResizing) {
      const handle = resizeState.resizeHandle;
      if (handle?.includes('e')) {
        width += resizeState.resizeOffset.x;
      }
      if (handle?.includes('w')) {
        x += resizeState.resizeOffset.x;
        width -= resizeState.resizeOffset.x;
      }
      if (handle?.includes('s')) {
        height += resizeState.resizeOffset.y;
      }
      if (handle?.includes('n')) {
        y += resizeState.resizeOffset.y;
        height -= resizeState.resizeOffset.y;
      }
    }

    return {
      position: 'absolute' as const,
      left: `${x}px`,
      top: `${y}px`,
      width: `${width}px`,
      height: `${height}px`,
      zIndex: dragState.isDragging || resizeState.isResizing ? 1000 : 1,
      ...style,
    };
  }, [instance.layout, gridConfig, dragState, resizeState, style]);

  /**
   * CSS classes for the component
   */
  const componentClasses = useMemo(() => {
    const classes = ['draggable-component'];
    
    if (className) {
      classes.push(className);
    }
    
    if (isInteractive) {
      classes.push('draggable-component--interactive');
    }
    
    if (dragState.isDragging) {
      classes.push('draggable-component--dragging');
    }
    
    if (resizeState.isResizing) {
      classes.push('draggable-component--resizing');
    }
    
    if (isSelected) {
      classes.push('draggable-component--selected');
    }
    
    if (isHovered && isInteractive) {
      classes.push('draggable-component--hovered');
    }
    
    if (!instance.hasData && isEditMode) {
      classes.push('draggable-component--empty');
    }
    
    return classes.join(' ');
  }, [className, isInteractive, dragState.isDragging, resizeState.isResizing, isSelected, isHovered, instance.hasData, isEditMode]);

  // ========================================================================
  // DRAG HANDLERS
  // ========================================================================

  /**
   * Handles drag start
   */
  const handleDragStart = useCallback((event: React.MouseEvent) => {
    if (!isInteractive) return;
    
    event.preventDefault();
    event.stopPropagation();

    const startPosition = { x: event.clientX, y: event.clientY };
    const startLayout = { x: instance.layout.x, y: instance.layout.y };

    setDragState({
      isDragging: true,
      startPosition,
      startLayout,
      dragOffset: { x: 0, y: 0 },
    });

    // Select component
    if (onSelect) {
      onSelect(instance.id);
    }
  }, [isInteractive, instance.layout, instance.id, onSelect]);

  /**
   * Handles drag movement
   */
  const handleDragMove = useCallback((event: MouseEvent) => {
    if (!dragState.isDragging) return;

    const currentPosition = { x: event.clientX, y: event.clientY };
    const dragOffset = {
      x: currentPosition.x - dragState.startPosition.x,
      y: currentPosition.y - dragState.startPosition.y,
    };

    setDragState(prev => ({
      ...prev,
      dragOffset,
    }));

    // Calculate grid position
    const { columnWidth, rowHeight } = gridConfig;
    const gridX = Math.round(dragOffset.x / columnWidth);
    const gridY = Math.round(dragOffset.y / rowHeight);
    const newX = Math.max(0, dragState.startLayout.x + gridX);
    const newY = Math.max(0, dragState.startLayout.y + gridY);

    // Trigger drag callback
    if (onDrag) {
      onDrag(instance.id, newX, newY);
    }
  }, [dragState, gridConfig, instance.id, onDrag]);

  /**
   * Handles drag end
   */
  const handleDragEnd = useCallback(() => {
    if (!dragState.isDragging) return;

    // Calculate final position
    const { columnWidth, rowHeight } = gridConfig;
    const gridX = Math.round(dragState.dragOffset.x / columnWidth);
    const gridY = Math.round(dragState.dragOffset.y / rowHeight);
    const newX = Math.max(0, dragState.startLayout.x + gridX);
    const newY = Math.max(0, dragState.startLayout.y + gridY);

    // Reset drag state
    setDragState({
      isDragging: false,
      startPosition: { x: 0, y: 0 },
      startLayout: { x: 0, y: 0 },
      dragOffset: { x: 0, y: 0 },
    });

    // Trigger drag stop callback
    if (onDragStop) {
      onDragStop(instance.id, newX, newY);
    }
  }, [dragState, gridConfig, instance.id, onDragStop]);

  // ========================================================================
  // RESIZE HANDLERS
  // ========================================================================

  /**
   * Handles resize start
   */
  const handleResizeStart = useCallback((event: React.MouseEvent, handle: ResizeHandle) => {
    if (!isInteractive) return;
    
    event.preventDefault();
    event.stopPropagation();

    const startPosition = { x: event.clientX, y: event.clientY };
    const startLayout = {
      x: instance.layout.x,
      y: instance.layout.y,
      w: instance.layout.w,
      h: instance.layout.h,
    };

    setResizeState({
      isResizing: true,
      resizeHandle: handle,
      startPosition,
      startLayout,
      resizeOffset: { x: 0, y: 0 },
    });

    // Select component
    if (onSelect) {
      onSelect(instance.id);
    }
  }, [isInteractive, instance.layout, instance.id, onSelect]);

  /**
   * Handles resize movement
   */
  const handleResizeMove = useCallback((event: MouseEvent) => {
    if (!resizeState.isResizing) return;

    const currentPosition = { x: event.clientX, y: event.clientY };
    const resizeOffset = {
      x: currentPosition.x - resizeState.startPosition.x,
      y: currentPosition.y - resizeState.startPosition.y,
    };

    setResizeState(prev => ({
      ...prev,
      resizeOffset,
    }));

    // Calculate new dimensions
    const { columnWidth, rowHeight } = gridConfig;
    const handle = resizeState.resizeHandle!;
    
    let newWidth = resizeState.startLayout.w;
    let newHeight = resizeState.startLayout.h;

    if (handle.includes('e')) {
      const deltaW = Math.round(resizeOffset.x / columnWidth);
      newWidth = Math.max(1, resizeState.startLayout.w + deltaW);
    }
    if (handle.includes('w')) {
      const deltaW = Math.round(-resizeOffset.x / columnWidth);
      newWidth = Math.max(1, resizeState.startLayout.w + deltaW);
    }
    if (handle.includes('s')) {
      const deltaH = Math.round(resizeOffset.y / rowHeight);
      newHeight = Math.max(1, resizeState.startLayout.h + deltaH);
    }
    if (handle.includes('n')) {
      const deltaH = Math.round(-resizeOffset.y / rowHeight);
      newHeight = Math.max(1, resizeState.startLayout.h + deltaH);
    }

    // Apply constraints
    const config = instance.config;
    if (config?.minSize) {
      newWidth = Math.max(newWidth, config.minSize.w);
      newHeight = Math.max(newHeight, config.minSize.h);
    }
    if (config?.maxSize) {
      newWidth = Math.min(newWidth, config.maxSize.w);
      newHeight = Math.min(newHeight, config.maxSize.h);
    }

    // Trigger resize callback
    if (onResize) {
      onResize(instance.id, newWidth, newHeight);
    }
  }, [resizeState, gridConfig, instance.id, instance.config, onResize]);

  /**
   * Handles resize end
   */
  const handleResizeEnd = useCallback(() => {
    if (!resizeState.isResizing) return;

    // Calculate final dimensions
    const { columnWidth, rowHeight } = gridConfig;
    const handle = resizeState.resizeHandle!;
    
    let newWidth = resizeState.startLayout.w;
    let newHeight = resizeState.startLayout.h;

    if (handle.includes('e')) {
      const deltaW = Math.round(resizeState.resizeOffset.x / columnWidth);
      newWidth = Math.max(1, resizeState.startLayout.w + deltaW);
    }
    if (handle.includes('w')) {
      const deltaW = Math.round(-resizeState.resizeOffset.x / columnWidth);
      newWidth = Math.max(1, resizeState.startLayout.w + deltaW);
    }
    if (handle.includes('s')) {
      const deltaH = Math.round(resizeState.resizeOffset.y / rowHeight);
      newHeight = Math.max(1, resizeState.startLayout.h + deltaH);
    }
    if (handle.includes('n')) {
      const deltaH = Math.round(-resizeState.resizeOffset.y / rowHeight);
      newHeight = Math.max(1, resizeState.startLayout.h + deltaH);
    }

    // Apply constraints
    const config = instance.config;
    if (config?.minSize) {
      newWidth = Math.max(newWidth, config.minSize.w);
      newHeight = Math.max(newHeight, config.minSize.h);
    }
    if (config?.maxSize) {
      newWidth = Math.min(newWidth, config.maxSize.w);
      newHeight = Math.min(newHeight, config.maxSize.h);
    }

    // Reset resize state
    setResizeState({
      isResizing: false,
      resizeHandle: null,
      startPosition: { x: 0, y: 0 },
      startLayout: { x: 0, y: 0, w: 0, h: 0 },
      resizeOffset: { x: 0, y: 0 },
    });

    // Trigger resize stop callback
    if (onResizeStop) {
      onResizeStop(instance.id, newWidth, newHeight);
    }
  }, [resizeState, gridConfig, instance.id, instance.config, onResizeStop]);

  // ========================================================================
  // MOUSE EVENT HANDLERS
  // ========================================================================

  /**
   * Handles component click
   */
  const handleClick = useCallback((event: React.MouseEvent) => {
    if (!isInteractive) return;
    
    event.stopPropagation();
    
    if (onSelect) {
      onSelect(instance.id);
    }
  }, [isInteractive, instance.id, onSelect]);

  /**
   * Handles mouse enter
   */
  const handleMouseEnter = useCallback(() => {
    if (isInteractive) {
      setIsHovered(true);
    }
  }, [isInteractive]);

  /**
   * Handles mouse leave
   */
  const handleMouseLeave = useCallback(() => {
    setIsHovered(false);
  }, []);

  // ========================================================================
  // EFFECTS
  // ========================================================================

  /**
   * Set up global mouse event listeners for drag and resize
   */
  useEffect(() => {
    if (dragState.isDragging) {
      document.addEventListener('mousemove', handleDragMove);
      document.addEventListener('mouseup', handleDragEnd);
      document.body.style.cursor = 'grabbing';
      document.body.style.userSelect = 'none';
    }

    if (resizeState.isResizing) {
      document.addEventListener('mousemove', handleResizeMove);
      document.addEventListener('mouseup', handleResizeEnd);
      document.body.style.userSelect = 'none';
      
      // Set appropriate cursor based on resize handle
      const handle = resizeState.resizeHandle;
      if (handle === 'nw' || handle === 'se') {
        document.body.style.cursor = 'nw-resize';
      } else if (handle === 'ne' || handle === 'sw') {
        document.body.style.cursor = 'ne-resize';
      } else if (handle === 'n' || handle === 's') {
        document.body.style.cursor = 'ns-resize';
      } else if (handle === 'w' || handle === 'e') {
        document.body.style.cursor = 'ew-resize';
      }
    }

    return () => {
      document.removeEventListener('mousemove', handleDragMove);
      document.removeEventListener('mouseup', handleDragEnd);
      document.removeEventListener('mousemove', handleResizeMove);
      document.removeEventListener('mouseup', handleResizeEnd);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [dragState.isDragging, resizeState.isResizing, handleDragMove, handleDragEnd, handleResizeMove, handleResizeEnd, resizeState.resizeHandle]);

  // ========================================================================
  // RENDER HELPERS
  // ========================================================================

  /**
   * Renders resize handles
   */
  const renderResizeHandles = () => {
    if (!isInteractive || !isSelected) return null;

    const handles: ResizeHandle[] = ['nw', 'n', 'ne', 'w', 'e', 'sw', 's', 'se'];

    return handles.map(handle => (
      <div
        key={handle}
        className={`draggable-component__resize-handle draggable-component__resize-handle--${handle}`}
        onMouseDown={(e) => handleResizeStart(e, handle)}
        style={{
          position: 'absolute',
          width: '8px',
          height: '8px',
          backgroundColor: '#007bff',
          border: '1px solid #fff',
          borderRadius: '2px',
          cursor: getCursorForHandle(handle),
          zIndex: 1001,
          ...getHandlePosition(handle),
        }}
      />
    ));
  };

  /**
   * Gets cursor style for resize handle
   */
  const getCursorForHandle = (handle: ResizeHandle): string => {
    switch (handle) {
      case 'nw':
      case 'se':
        return 'nw-resize';
      case 'ne':
      case 'sw':
        return 'ne-resize';
      case 'n':
      case 's':
        return 'ns-resize';
      case 'w':
      case 'e':
        return 'ew-resize';
      default:
        return 'default';
    }
  };

  /**
   * Gets position styles for resize handle
   */
  const getHandlePosition = (handle: ResizeHandle): React.CSSProperties => {
    const offset = -4; // Half of handle size
    
    switch (handle) {
      case 'nw':
        return { top: offset, left: offset };
      case 'n':
        return { top: offset, left: '50%', transform: 'translateX(-50%)' };
      case 'ne':
        return { top: offset, right: offset };
      case 'w':
        return { top: '50%', left: offset, transform: 'translateY(-50%)' };
      case 'e':
        return { top: '50%', right: offset, transform: 'translateY(-50%)' };
      case 'sw':
        return { bottom: offset, left: offset };
      case 's':
        return { bottom: offset, left: '50%', transform: 'translateX(-50%)' };
      case 'se':
        return { bottom: offset, right: offset };
      default:
        return {};
    }
  };

  /**
   * Renders drag handle
   */
  const renderDragHandle = () => {
    if (!isInteractive || (!isHovered && !isSelected)) return null;

    return (
      <div
        ref={dragHandleRef}
        className="draggable-component__drag-handle"
        onMouseDown={handleDragStart}
        style={{
          position: 'absolute',
          top: '4px',
          right: '4px',
          width: '20px',
          height: '20px',
          backgroundColor: 'rgba(0, 123, 255, 0.8)',
          borderRadius: '4px',
          cursor: 'grab',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 1001,
          color: 'white',
          fontSize: '12px',
        }}
        title="Drag to move"
      >
        ⋮⋮
      </div>
    );
  };

  /**
   * Renders empty state indicator
   */
  const renderEmptyIndicator = () => {
    if (instance.hasData || !isEditMode) return null;

    return (
      <div
        className="draggable-component__empty-indicator"
        style={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          padding: '8px 12px',
          backgroundColor: 'rgba(255, 193, 7, 0.9)',
          color: '#856404',
          borderRadius: '4px',
          fontSize: '12px',
          fontWeight: 'bold',
          zIndex: 1000,
          pointerEvents: 'none',
        }}
      >
        No Data
      </div>
    );
  };

  // ========================================================================
  // RENDER
  // ========================================================================

  return (
    <div
      ref={componentRef}
      className={componentClasses}
      style={componentStyle}
      onClick={handleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {/* Component content */}
      <div className="draggable-component__content">
        {children}
      </div>

      {/* Drag handle */}
      {renderDragHandle()}

      {/* Resize handles */}
      {renderResizeHandles()}

      {/* Empty state indicator */}
      {renderEmptyIndicator()}

      {/* Selection border */}
      {isSelected && (
        <div
          className="draggable-component__selection-border"
          style={{
            position: 'absolute',
            top: '-2px',
            left: '-2px',
            right: '-2px',
            bottom: '-2px',
            border: '2px solid #007bff',
            borderRadius: '4px',
            pointerEvents: 'none',
            zIndex: 999,
          }}
        />
      )}
    </div>
  );
};

// ============================================================================
// EXPORTS
// ============================================================================

export default DraggableComponent;