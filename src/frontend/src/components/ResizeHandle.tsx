import React, { useCallback, useEffect, useRef, useState } from 'react';

/**
 * ResizeHandle Component
 * 
 * A draggable handle for resizing panels in the three-panel layout.
 * Provides visual feedback during hover and drag operations.
 * 
 * Uses useRef for dragging state and callback refs to avoid React stale closure issues.
 * The isDragging useState is kept only for visual rendering (CSS class toggling).
 * 
 * Requirements: 1.7, 1.8, 1.9, 3.1, 3.2, 3.5, 3.6
 */

interface ResizeHandleProps {
  onDragStart: () => void;
  onDrag: (deltaX: number) => void;
  onDragEnd: () => void;
  orientation?: 'vertical' | 'horizontal';
}

export const ResizeHandle: React.FC<ResizeHandleProps> = ({
  onDragStart,
  onDrag,
  onDragEnd,
  orientation = 'vertical'
}) => {
  // isDragging state is kept ONLY for visual rendering (CSS class toggling)
  const [isDragging, setIsDragging] = useState(false);
  const [isHovered, setIsHovered] = useState(false);

  // useRef for dragging state to avoid stale closure issues in event callbacks
  const isDraggingRef = useRef(false);
  const dragStartX = useRef<number>(0);
  const dragStartY = useRef<number>(0);

  // Store latest callback references to avoid stale closures
  const onDragRef = useRef(onDrag);
  const onDragEndRef = useRef(onDragEnd);
  const onDragStartRef = useRef(onDragStart);

  // Keep callback refs up to date
  useEffect(() => {
    onDragRef.current = onDrag;
    onDragEndRef.current = onDragEnd;
    onDragStartRef.current = onDragStart;
  }, [onDrag, onDragEnd, onDragStart]);

  // Store orientation in a ref so event handlers always have the latest value
  const orientationRef = useRef(orientation);
  useEffect(() => {
    orientationRef.current = orientation;
  }, [orientation]);

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!isDraggingRef.current) return;

    const deltaX = e.clientX - dragStartX.current;
    const deltaY = e.clientY - dragStartY.current;

    if (orientationRef.current === 'vertical') {
      onDragRef.current(deltaX);
      dragStartX.current = e.clientX;
    } else {
      onDragRef.current(deltaY);
      dragStartY.current = e.clientY;
    }
  }, []);

  const handleMouseUp = useCallback(() => {
    if (!isDraggingRef.current) return;

    isDraggingRef.current = false;
    setIsDragging(false);
    onDragEndRef.current();

    // Restore text selection and cursor
    document.body.style.userSelect = '';
    document.body.style.cursor = '';

    // Remove global event listeners
    window.removeEventListener('mousemove', handleMouseMove);
    window.removeEventListener('mouseup', handleMouseUp);
  }, [handleMouseMove]);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();

    // Set ref for event callbacks (avoids stale closure)
    isDraggingRef.current = true;
    // Set state for visual rendering only
    setIsDragging(true);

    dragStartX.current = e.clientX;
    dragStartY.current = e.clientY;
    onDragStartRef.current();

    // Prevent text selection during drag
    document.body.style.userSelect = 'none';
    document.body.style.cursor = orientationRef.current === 'vertical' ? 'col-resize' : 'row-resize';

    // Register global event listeners directly in mousedown handler
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
  }, [handleMouseMove, handleMouseUp]);

  // Cleanup: remove global listeners if component unmounts while dragging
  useEffect(() => {
    return () => {
      if (isDraggingRef.current) {
        window.removeEventListener('mousemove', handleMouseMove);
        window.removeEventListener('mouseup', handleMouseUp);
        document.body.style.userSelect = '';
        document.body.style.cursor = '';
      }
    };
  }, [handleMouseMove, handleMouseUp]);

  const handleMouseEnter = useCallback(() => {
    setIsHovered(true);
  }, []);

  const handleMouseLeave = useCallback(() => {
    if (!isDraggingRef.current) {
      setIsHovered(false);
    }
  }, []);

  // Keyboard support for accessibility (arrow keys to resize)
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    const step = e.shiftKey ? 20 : 5; // Shift for larger steps
    let delta = 0;

    if (orientationRef.current === 'vertical') {
      if (e.key === 'ArrowLeft') delta = -step;
      else if (e.key === 'ArrowRight') delta = step;
    } else {
      if (e.key === 'ArrowUp') delta = -step;
      else if (e.key === 'ArrowDown') delta = step;
    }

    if (delta !== 0) {
      e.preventDefault();
      onDragStartRef.current();
      onDragRef.current(delta);
      onDragEndRef.current();
    }
  }, []);

  // Determine cursor style based on orientation
  const cursorStyle = orientation === 'vertical' ? 'col-resize' : 'row-resize';

  // Determine dimensions based on orientation
  const dimensionStyles = orientation === 'vertical'
    ? { width: '8px', height: '100%' }
    : { width: '100%', height: '8px' };

  return (
    <div
      className={`resize-handle ${orientation} ${isDragging ? 'dragging' : ''} ${isHovered ? 'hovered' : ''}`}
      onMouseDown={handleMouseDown}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      onKeyDown={handleKeyDown}
      style={{
        ...dimensionStyles,
        cursor: cursorStyle,
        position: 'relative',
        flexShrink: 0,
        backgroundColor: 'transparent',
        transition: isDragging ? 'none' : 'background-color 150ms ease-in-out',
        zIndex: 10,
      }}
      role="separator"
      aria-orientation={orientation}
      aria-label={`Resize ${orientation === 'vertical' ? 'panels horizontally' : 'panels vertically'}`}
      tabIndex={0}
    >
      {/* Visible line indicator centered in the hit area */}
      <div
        style={{
          position: 'absolute',
          top: 0,
          bottom: 0,
          left: '50%',
          transform: 'translateX(-50%)',
          width: orientation === 'vertical' ? '2px' : '100%',
          height: orientation === 'vertical' ? '100%' : '2px',
          backgroundColor: isDragging ? 'var(--resize-handle-active-bg, #3b82f6)' : 
                          isHovered ? 'var(--resize-handle-hover-bg, #94a3b8)' : 
                          'var(--resize-handle-bg, #e2e8f0)',
          transition: isDragging ? 'none' : 'background-color 150ms ease-in-out',
          pointerEvents: 'none',
        }}
      />
    </div>
  );
};

export default ResizeHandle;
