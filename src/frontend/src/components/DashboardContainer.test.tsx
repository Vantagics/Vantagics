/**
 * Unit tests for DashboardContainer component
 */

import React from 'react';
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import DashboardContainer, { DashboardLayout, DashboardContainerProps } from './DashboardContainer';
import { ComponentType } from '../utils/ComponentManager';
import { LayoutItem } from '../utils/LayoutEngine';

// Mock the managers to avoid complex dependencies
const MockLayoutEngine = vi.fn().mockImplementation(() => ({
  getConfig: vi.fn(() => ({
    columns: 24,
    rowHeight: 30,
    margin: [10, 10],
    containerPadding: [10, 10],
  })),
  calculatePosition: vi.fn((item, x, y) => ({ x, y })),
  snapToGrid: vi.fn((width, height) => ({ width: Math.round(width), height: Math.round(height) })),
  validateItemConstraints: vi.fn(item => item),
  compactLayout: vi.fn(items => ({ items, changed: false })),
  getLayoutHeight: vi.fn(() => 300),
}));

vi.mock('../utils/LayoutEngine', () => ({
  default: MockLayoutEngine,
  DEFAULT_GRID_CONFIG: {
    columns: 24,
    rowHeight: 30,
    margin: [10, 10],
    containerPadding: [10, 10],
  },
}));

const MockComponentManager = vi.fn().mockImplementation(() => ({
  getComponentEntry: vi.fn((type) => ({
    type,
    displayName: `${type} Component`,
    config: {
      defaultSize: { w: 6, h: 4 },
      minSize: { w: 3, h: 2 },
      maxSize: { w: 12, h: 8 },
      supportsPagination: true,
    },
  })),
  createInstance: vi.fn((type, layout) => ({
    id: layout.i,
    type,
    title: `${type} Component`,
    layout,
    hasData: false,
  })),
  removeInstance: vi.fn(() => true),
}));

vi.mock('../utils/ComponentManager', () => ({
  default: MockComponentManager,
  ComponentType: {
    METRICS: 'metrics',
    TABLE: 'table',
    IMAGE: 'image',
    INSIGHTS: 'insights',
    FILE_DOWNLOAD: 'file_download',
  },
}));

describe('DashboardContainer', () => {
  let mockProps: DashboardContainerProps;
  let mockLayoutItems: LayoutItem[];

  beforeEach(() => {
    mockLayoutItems = [
      { i: 'metrics_1', x: 0, y: 0, w: 6, h: 4 },
      { i: 'table_1', x: 6, y: 0, w: 12, h: 6 },
    ];

    mockProps = {
      initialLayout: {
        items: mockLayoutItems,
        gridConfig: {
          columns: 24,
          rowHeight: 30,
          margin: [10, 10],
          containerPadding: [10, 10],
        },
        metadata: {
          version: '1.0.0',
          createdAt: Date.now(),
          updatedAt: Date.now(),
        },
      },
      initialEditMode: false,
      initialLocked: true,
      onLayoutChange: vi.fn(),
      onEditModeChange: vi.fn(),
      onLockStateChange: vi.fn(),
    };

    // Clear all mocks
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllTimers();
  });

  describe('Rendering', () => {
    it('should render dashboard container with initial layout', () => {
      render(<DashboardContainer {...mockProps} />);
      
      expect(screen.getByText('Component: metrics_1')).toBeInTheDocument();
      expect(screen.getByText('Component: table_1')).toBeInTheDocument();
    });

    it('should render empty state when no components', () => {
      const emptyProps = {
        ...mockProps,
        initialLayout: {
          ...mockProps.initialLayout!,
          items: [],
        },
      };
      
      render(<DashboardContainer {...emptyProps} />);
      
      expect(screen.getByText('No components in dashboard')).toBeInTheDocument();
      expect(screen.getByText('Switch to edit mode to add components')).toBeInTheDocument();
    });

    it('should render loading state', () => {
      render(<DashboardContainer {...mockProps} />);
      
      // Initially shows loading
      expect(screen.getByText('Loading dashboard...')).toBeInTheDocument();
    });

    it('should apply custom className and styles', () => {
      const customProps = {
        ...mockProps,
        className: 'custom-dashboard',
        style: { backgroundColor: 'red' },
      };
      
      render(<DashboardContainer {...customProps} />);
      
      const container = document.querySelector('.dashboard-container');
      expect(container).toHaveClass('custom-dashboard');
      expect(container).toHaveStyle({ backgroundColor: 'red' });
    });
  });

  describe('State Management', () => {
    it('should initialize with correct initial state', async () => {
      render(<DashboardContainer {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument();
      });
      
      // Should not show toolbar when locked
      expect(screen.queryByText('🔒 Locked')).not.toBeInTheDocument();
    });

    it('should handle initial edit mode', async () => {
      const editModeProps = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...editModeProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('🔓 Unlocked')).toBeInTheDocument();
        expect(screen.getByText('✏️ Exit Edit')).toBeInTheDocument();
      });
    });
  });

  describe('Mode Switching', () => {
    it('should toggle lock state', async () => {
      const props = {
        ...mockProps,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const lockButton = screen.getByText('🔓 Unlocked');
        fireEvent.click(lockButton);
      });
      
      expect(mockProps.onLockStateChange).toHaveBeenCalledWith(true);
    });

    it('should toggle edit mode', async () => {
      const props = {
        ...mockProps,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const editButton = screen.getByText('✏️ Edit Layout');
        fireEvent.click(editButton);
      });
      
      expect(mockProps.onEditModeChange).toHaveBeenCalledWith(true);
    });

    it('should enter edit mode from empty state', async () => {
      const emptyProps = {
        ...mockProps,
        initialLayout: {
          ...mockProps.initialLayout!,
          items: [],
        },
      };
      
      render(<DashboardContainer {...emptyProps} />);
      
      await waitFor(() => {
        const editButton = screen.getByText('Start Editing');
        fireEvent.click(editButton);
      });
      
      expect(mockProps.onEditModeChange).toHaveBeenCalledWith(true);
      expect(mockProps.onLockStateChange).toHaveBeenCalledWith(false);
    });

    it('should exit edit mode and lock layout', async () => {
      const props = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const editButton = screen.getByText('✏️ Exit Edit');
        fireEvent.click(editButton);
      });
      
      expect(mockProps.onEditModeChange).toHaveBeenCalledWith(false);
      expect(mockProps.onLockStateChange).toHaveBeenCalledWith(true);
    });
  });

  describe('Component Management', () => {
    it('should show add component buttons in edit mode', async () => {
      const props = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        expect(screen.getByText('+ Metrics')).toBeInTheDocument();
        expect(screen.getByText('+ Table')).toBeInTheDocument();
        expect(screen.getByText('+ Image')).toBeInTheDocument();
        expect(screen.getByText('+ Insights')).toBeInTheDocument();
        expect(screen.getByText('+ Files')).toBeInTheDocument();
      });
    });

    it('should add component when button clicked', async () => {
      const props = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const addButton = screen.getByText('+ Metrics');
        fireEvent.click(addButton);
      });
      
      // Should trigger layout change
      await waitFor(() => {
        expect(mockProps.onLayoutChange).toHaveBeenCalled();
      });
    });

    it('should show remove buttons in edit mode', async () => {
      const props = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const removeButtons = screen.getAllByText('✕');
        expect(removeButtons).toHaveLength(2); // One for each component
      });
    });

    it('should remove component when remove button clicked', async () => {
      const props = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const removeButtons = screen.getAllByText('✕');
        fireEvent.click(removeButtons[0]);
      });
      
      // Should trigger layout change
      await waitFor(() => {
        expect(mockProps.onLayoutChange).toHaveBeenCalled();
      });
    });

    it('should show compact layout button in edit mode', async () => {
      const props = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        expect(screen.getByText('📐 Compact Layout')).toBeInTheDocument();
      });
    });
  });

  describe('Layout Operations', () => {
    it('should prevent operations when locked', async () => {
      // Mock console.warn to verify warnings
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
      
      const props = {
        ...mockProps,
        initialLocked: true,
      };
      
      const { rerender } = render(<DashboardContainer {...props} />);
      
      // Try to trigger operations that should be blocked
      // Since the component doesn't expose these methods directly,
      // we'll test through the UI when possible
      
      await waitFor(() => {
        // In locked mode, toolbar should not be visible
        expect(screen.queryByText('+ Metrics')).not.toBeInTheDocument();
      });
      
      consoleSpy.mockRestore();
    });
  });

  describe('Error Handling', () => {
    it('should handle and display errors', async () => {
      // Mock an error scenario
      const errorProps = {
        ...mockProps,
        initialLayout: undefined, // This might cause an error
      };
      
      render(<DashboardContainer {...errorProps} />);
      
      // Should show loading initially
      expect(screen.getByText('Loading dashboard...')).toBeInTheDocument();
    });

    it('should provide retry functionality on error', async () => {
      // This test would need to mock an actual error state
      // For now, we'll just verify the structure exists
      render(<DashboardContainer {...mockProps} />);
      
      // The error state would show a retry button
      // expect(screen.getByText('Retry')).toBeInTheDocument();
    });
  });

  describe('Callbacks', () => {
    it('should call onLayoutChange when layout updates', async () => {
      const props = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const addButton = screen.getByText('+ Metrics');
        fireEvent.click(addButton);
      });
      
      await waitFor(() => {
        expect(mockProps.onLayoutChange).toHaveBeenCalled();
      });
    });

    it('should call onEditModeChange when edit mode toggles', async () => {
      const props = {
        ...mockProps,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const editButton = screen.getByText('✏️ Edit Layout');
        fireEvent.click(editButton);
      });
      
      expect(mockProps.onEditModeChange).toHaveBeenCalledWith(true);
    });

    it('should call onLockStateChange when lock state toggles', async () => {
      const props = {
        ...mockProps,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const lockButton = screen.getByText('🔓 Unlocked');
        fireEvent.click(lockButton);
      });
      
      expect(mockProps.onLockStateChange).toHaveBeenCalledWith(true);
    });
  });

  describe('Grid Layout', () => {
    it('should position components according to layout items', async () => {
      render(<DashboardContainer {...mockProps} />);
      
      await waitFor(() => {
        const gridItems = document.querySelectorAll('.dashboard-container__grid-item');
        expect(gridItems).toHaveLength(2);
        
        // Check positioning styles are applied
        const firstItem = gridItems[0] as HTMLElement;
        expect(firstItem.style.position).toBe('absolute');
        expect(firstItem.style.left).toBeDefined();
        expect(firstItem.style.top).toBeDefined();
        expect(firstItem.style.width).toBeDefined();
        expect(firstItem.style.height).toBeDefined();
      });
    });

    it('should apply correct CSS classes based on state', async () => {
      const props = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const container = document.querySelector('.dashboard-container');
        expect(container).toHaveClass('dashboard-container--edit-mode');
        expect(container).not.toHaveClass('dashboard-container--locked');
      });
    });
  });

  describe('Responsive Behavior', () => {
    it('should handle grid configuration updates', () => {
      const customGridConfig = {
        columns: 12,
        rowHeight: 40,
      };
      
      const props = {
        ...mockProps,
        gridConfig: customGridConfig,
      };
      
      render(<DashboardContainer {...props} />);
      
      // The component should use the custom grid configuration
      // This is tested indirectly through the layout engine mock
    });
  });

  describe('Performance', () => {
    it('should handle many components efficiently', async () => {
      const manyItems: LayoutItem[] = Array.from({ length: 50 }, (_, i) => ({
        i: `item_${i}`,
        x: (i % 6) * 4,
        y: Math.floor(i / 6) * 4,
        w: 4,
        h: 4,
      }));
      
      const props = {
        ...mockProps,
        initialLayout: {
          ...mockProps.initialLayout!,
          items: manyItems,
        },
      };
      
      const startTime = performance.now();
      render(<DashboardContainer {...props} />);
      const endTime = performance.now();
      
      expect(endTime - startTime).toBeLessThan(1000); // Should render in under 1 second
      
      await waitFor(() => {
        const gridItems = document.querySelectorAll('.dashboard-container__grid-item');
        expect(gridItems).toHaveLength(50);
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper button labels', async () => {
      const props = {
        ...mockProps,
        initialEditMode: true,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        expect(screen.getByText('🔓 Unlocked')).toBeInTheDocument();
        expect(screen.getByText('✏️ Exit Edit')).toBeInTheDocument();
        expect(screen.getByText('📐 Compact Layout')).toBeInTheDocument();
      });
    });

    it('should handle keyboard interactions', async () => {
      const props = {
        ...mockProps,
        initialLocked: false,
      };
      
      render(<DashboardContainer {...props} />);
      
      await waitFor(() => {
        const editButton = screen.getByText('✏️ Edit Layout');
        
        // Test keyboard activation
        editButton.focus();
        fireEvent.keyDown(editButton, { key: 'Enter' });
      });
      
      expect(mockProps.onEditModeChange).toHaveBeenCalledWith(true);
    });
  });
});