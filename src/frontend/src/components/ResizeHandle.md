# ResizeHandle Component

## Overview

The `ResizeHandle` component provides a draggable handle for resizing panels in the three-panel layout. It offers visual feedback during hover and drag operations, making it easy for users to adjust panel widths.

## Features

- **Draggable Interface**: Smooth drag-and-drop resizing with mouse events
- **Visual Feedback**: 
  - Hover state with color change
  - Active drag state with distinct styling
  - Visual indicator line in the center
- **Cursor Changes**: Automatic cursor updates (col-resize/row-resize)
- **Text Selection Prevention**: Prevents text selection during drag operations
- **Accessibility**: ARIA attributes and keyboard navigation support
- **Orientation Support**: Works for both vertical and horizontal resizing

## Usage

```tsx
import { ResizeHandle } from './components/ResizeHandle';

function MyLayout() {
  const handleDragStart = () => {
    console.log('Drag started');
  };

  const handleDrag = (deltaX: number) => {
    // Update panel widths based on deltaX
    console.log('Dragging:', deltaX);
  };

  const handleDragEnd = () => {
    console.log('Drag ended');
    // Persist new widths to localStorage
  };

  return (
    <div style={{ display: 'flex' }}>
      <div style={{ width: '200px' }}>Left Panel</div>
      
      <ResizeHandle
        onDragStart={handleDragStart}
        onDrag={handleDrag}
        onDragEnd={handleDragEnd}
        orientation="vertical"
      />
      
      <div style={{ flex: 1 }}>Center Panel</div>
    </div>
  );
}
```

## Props

### `onDragStart: () => void` (required)
Callback fired when the user starts dragging the handle.

### `onDrag: (deltaX: number) => void` (required)
Callback fired during drag with the delta movement in pixels. For vertical orientation, this is the horizontal delta (deltaX). For horizontal orientation, this is the vertical delta (deltaY).

### `onDragEnd: () => void` (required)
Callback fired when the user releases the handle.

### `orientation?: 'vertical' | 'horizontal'` (optional)
The orientation of the resize handle. Defaults to `'vertical'`.
- `'vertical'`: For resizing panels horizontally (left-right)
- `'horizontal'`: For resizing panels vertically (top-bottom)

## Styling

The component uses CSS variables for theming:

```css
--resize-handle-bg: #e2e8f0           /* Default background */
--resize-handle-hover-bg: #94a3b8     /* Hover background */
--resize-handle-active-bg: #3b82f6    /* Active/dragging background */
--resize-handle-indicator: #cbd5e1    /* Indicator line color */
--resize-handle-indicator-hover: #64748b  /* Indicator hover color */
```

You can override these in your CSS:

```css
:root {
  --resize-handle-bg: #f0f0f0;
  --resize-handle-hover-bg: #d0d0d0;
  --resize-handle-active-bg: #0066cc;
}
```

## Accessibility

- **ARIA Role**: `separator` with appropriate `aria-orientation`
- **ARIA Label**: Descriptive label for screen readers
- **Keyboard Support**: `tabIndex={0}` for keyboard navigation
- **Focus Indicators**: Visual focus states for keyboard users

## Implementation Details

### Drag Behavior

1. **Mouse Down**: Sets dragging state, records initial position, calls `onDragStart`
2. **Mouse Move**: Calculates delta from last position, calls `onDrag` with delta
3. **Mouse Up**: Clears dragging state, calls `onDragEnd`, restores cursor

### State Management

The component maintains internal state for:
- `isDragging`: Whether a drag operation is in progress
- `isHovered`: Whether the mouse is hovering over the handle
- `dragStartX/Y`: Initial mouse position when drag started

### Event Listeners

Global mouse event listeners are attached during drag to ensure smooth operation even if the mouse moves outside the handle area.

## Requirements Validated

This component satisfies the following requirements from the western-layout-redesign spec:

- **Requirement 1.7**: Provides draggable resize handles between panels
- **Requirement 1.8**: Provides draggable resize handles between panels
- **Requirement 1.9**: Adjusts panel widths in real-time during drag

## Testing

The component includes comprehensive unit tests covering:
- Rendering with different orientations
- Cursor changes
- Hover behavior
- Drag start, movement, and end
- Visual feedback
- Text selection prevention
- Edge cases
- Dimensions

Run tests with:
```bash
npm test -- ResizeHandle.test.tsx
```

## Browser Compatibility

Works in all modern browsers that support:
- React 18+
- CSS Flexbox
- Mouse events
- CSS variables
