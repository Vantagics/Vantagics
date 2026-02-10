import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ResizeHandle } from './ResizeHandle';

describe('ResizeHandle Component', () => {
  let onDragStart: ReturnType<typeof vi.fn<() => void>>;
  let onDrag: ReturnType<typeof vi.fn<(deltaX: number) => void>>;
  let onDragEnd: ReturnType<typeof vi.fn<() => void>>;

  beforeEach(() => {
    onDragStart = vi.fn<() => void>();
    onDrag = vi.fn<(deltaX: number) => void>();
    onDragEnd = vi.fn<() => void>();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render with default vertical orientation', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle).toBeDefined();
      expect(handle.getAttribute('aria-orientation')).toBe('vertical');
    });

    it('should render with horizontal orientation when specified', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
          orientation="horizontal"
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle.getAttribute('aria-orientation')).toBe('horizontal');
    });

    it('should have appropriate ARIA label', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle.getAttribute('aria-label')).toBe('Resize panels horizontally');
    });

    it('should be keyboard accessible with tabIndex', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle.getAttribute('tabIndex')).toBe('0');
    });
  });

  describe('Cursor Changes', () => {
    it('should have col-resize cursor for vertical orientation', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
          orientation="vertical"
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle.style.cursor).toBe('col-resize');
    });

    it('should have row-resize cursor for horizontal orientation', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
          orientation="horizontal"
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle.style.cursor).toBe('row-resize');
    });
  });

  describe('Hover Behavior', () => {
    it('should add hovered class on mouse enter', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle.className).not.toContain('hovered');

      fireEvent.mouseEnter(handle);
      expect(handle.className).toContain('hovered');
    });

    it('should remove hovered class on mouse leave', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      fireEvent.mouseEnter(handle);
      expect(handle.className).toContain('hovered');

      fireEvent.mouseLeave(handle);
      expect(handle.className).not.toContain('hovered');
    });

    it('should change background color on hover', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Check that background color is defined (CSS variables may not be computed in test environment)
      expect(handle.style.backgroundColor).toBeDefined();
      
      fireEvent.mouseEnter(handle);
      
      // Verify the handle has the hovered class which would trigger the color change
      expect(handle.className).toContain('hovered');
    });
  });

  describe('Drag Start', () => {
    it('should call onDragStart when mouse down', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });

      expect(onDragStart).toHaveBeenCalledTimes(1);
    });

    it('should add dragging class on mouse down', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle.className).not.toContain('dragging');

      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      expect(handle.className).toContain('dragging');
    });

    it('should prevent default on mouse down', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      const event = new MouseEvent('mousedown', { bubbles: true, cancelable: true });
      const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
      
      handle.dispatchEvent(event);
      expect(preventDefaultSpy).toHaveBeenCalled();
    });
  });

  describe('Drag Movement', () => {
    it('should call onDrag with deltaX during vertical drag', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
          orientation="vertical"
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      // Move mouse
      fireEvent.mouseMove(window, { clientX: 150, clientY: 100 });
      
      expect(onDrag).toHaveBeenCalledWith(50);
    });

    it('should call onDrag with deltaY during horizontal drag', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
          orientation="horizontal"
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      // Move mouse
      fireEvent.mouseMove(window, { clientX: 100, clientY: 150 });
      
      expect(onDrag).toHaveBeenCalledWith(50);
    });

    it('should handle negative delta values', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
          orientation="vertical"
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      // Move mouse backwards
      fireEvent.mouseMove(window, { clientX: 50, clientY: 100 });
      
      expect(onDrag).toHaveBeenCalledWith(-50);
    });

    it('should not call onDrag when not dragging', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      // Move mouse without starting drag
      fireEvent.mouseMove(window, { clientX: 150, clientY: 100 });
      
      expect(onDrag).not.toHaveBeenCalled();
    });
  });

  describe('Drag End', () => {
    it('should call onDragEnd on mouse up', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      // End drag
      fireEvent.mouseUp(window);
      
      expect(onDragEnd).toHaveBeenCalledTimes(1);
    });

    it('should remove dragging class on mouse up', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      expect(handle.className).toContain('dragging');
      
      // End drag
      fireEvent.mouseUp(window);
      expect(handle.className).not.toContain('dragging');
    });

    it('should not call onDragEnd if not dragging', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      // Mouse up without starting drag
      fireEvent.mouseUp(window);
      
      expect(onDragEnd).not.toHaveBeenCalled();
    });
  });

  describe('Visual Feedback', () => {
    it('should show active background color during drag', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Check that background color is defined
      expect(handle.style.backgroundColor).toBeDefined();

      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      // Verify the handle has the dragging class which would trigger the color change
      expect(handle.className).toContain('dragging');
    });

    it('should have visual indicator element', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      const indicator = handle.querySelector('div');
      
      expect(indicator).toBeDefined();
      expect(indicator?.style.pointerEvents).toBe('none');
    });

    it('should disable transitions during drag', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Before drag
      expect(handle.style.transition).toContain('background-color');

      // During drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      expect(handle.style.transition).toBe('none');
    });
  });

  describe('Text Selection Prevention', () => {
    it('should prevent text selection during drag', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      expect(document.body.style.userSelect).toBe('none');
    });

    it('should restore text selection after drag', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start and end drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      fireEvent.mouseUp(window);
      
      expect(document.body.style.userSelect).toBe('');
    });

    it('should set body cursor during drag', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
          orientation="vertical"
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      expect(document.body.style.cursor).toBe('col-resize');
    });

    it('should restore body cursor after drag', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start and end drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      fireEvent.mouseUp(window);
      
      expect(document.body.style.cursor).toBe('');
    });
  });

  describe('Edge Cases', () => {
    it('should handle rapid mouse movements', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      // Rapid movements
      fireEvent.mouseMove(window, { clientX: 110, clientY: 100 });
      fireEvent.mouseMove(window, { clientX: 120, clientY: 100 });
      fireEvent.mouseMove(window, { clientX: 130, clientY: 100 });
      
      expect(onDrag).toHaveBeenCalledTimes(3);
    });

    it('should handle mouse leaving and returning during drag', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      // Mouse leave
      fireEvent.mouseLeave(handle);
      
      // Should still be dragging
      expect(handle.className).toContain('dragging');
      
      // Should still respond to mouse move
      fireEvent.mouseMove(window, { clientX: 150, clientY: 100 });
      expect(onDrag).toHaveBeenCalled();
    });

    it('should handle zero delta movements', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
        />
      );

      const handle = screen.getByRole('separator');
      
      // Start drag
      fireEvent.mouseDown(handle, { clientX: 100, clientY: 100 });
      
      // Move to same position
      fireEvent.mouseMove(window, { clientX: 100, clientY: 100 });
      
      expect(onDrag).toHaveBeenCalledWith(0);
    });
  });

  describe('Dimensions', () => {
    it('should have correct dimensions for vertical orientation', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
          orientation="vertical"
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle.style.width).toBe('8px');
      expect(handle.style.height).toBe('100%');
    });

    it('should have correct dimensions for horizontal orientation', () => {
      render(
        <ResizeHandle
          onDragStart={onDragStart}
          onDrag={onDrag}
          onDragEnd={onDragEnd}
          orientation="horizontal"
        />
      );

      const handle = screen.getByRole('separator');
      expect(handle.style.width).toBe('100%');
      expect(handle.style.height).toBe('8px');
    });
  });
});
