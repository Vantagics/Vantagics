/**
 * Responsive Layout System for Dashboard Drag-Drop Layout
 * 
 * Handles responsive behavior across different screen sizes and devices
 */

export interface Breakpoint {
  name: string;
  minWidth: number;
  maxWidth?: number;
  columns: number;
  margin: [number, number];
  containerPadding: [number, number];
  rowHeight: number;
}

export interface ResponsiveConfig {
  breakpoints: Breakpoint[];
  defaultBreakpoint: string;
}

export interface ViewportInfo {
  width: number;
  height: number;
  breakpoint: Breakpoint;
  isMobile: boolean;
  isTablet: boolean;
  isDesktop: boolean;
  orientation: 'portrait' | 'landscape';
}

/**
 * Default responsive breakpoints
 */
export const DEFAULT_BREAKPOINTS: Breakpoint[] = [
  {
    name: 'xs',
    minWidth: 0,
    maxWidth: 575,
    columns: 4,
    margin: [5, 5],
    containerPadding: [5, 5],
    rowHeight: 60,
  },
  {
    name: 'sm',
    minWidth: 576,
    maxWidth: 767,
    columns: 8,
    margin: [8, 8],
    containerPadding: [8, 8],
    rowHeight: 70,
  },
  {
    name: 'md',
    minWidth: 768,
    maxWidth: 991,
    columns: 12,
    margin: [10, 10],
    containerPadding: [10, 10],
    rowHeight: 80,
  },
  {
    name: 'lg',
    minWidth: 992,
    maxWidth: 1199,
    columns: 18,
    margin: [12, 12],
    containerPadding: [12, 12],
    rowHeight: 90,
  },
  {
    name: 'xl',
    minWidth: 1200,
    maxWidth: 1599,
    columns: 24,
    margin: [15, 15],
    containerPadding: [15, 15],
    rowHeight: 100,
  },
  {
    name: 'xxl',
    minWidth: 1600,
    columns: 30,
    margin: [20, 20],
    containerPadding: [20, 20],
    rowHeight: 110,
  },
];

/**
 * Responsive layout manager
 */
export class ResponsiveLayoutManager {
  private config: ResponsiveConfig;
  private currentViewport: ViewportInfo | null = null;
  private listeners: Set<(viewport: ViewportInfo) => void> = new Set();
  private resizeObserver: ResizeObserver | null = null;
  private container: HTMLElement | null = null;

  constructor(config?: Partial<ResponsiveConfig>) {
    this.config = {
      breakpoints: DEFAULT_BREAKPOINTS,
      defaultBreakpoint: 'lg',
      ...config,
    };

    this.initializeViewport();
    this.setupResizeObserver();
  }

  /**
   * Initialize viewport information
   */
  private initializeViewport(): void {
    if (typeof window !== 'undefined') {
      this.updateViewport();
      window.addEventListener('resize', this.handleResize);
      window.addEventListener('orientationchange', this.handleOrientationChange);
    }
  }

  /**
   * Setup resize observer for container
   */
  private setupResizeObserver(): void {
    if (typeof window !== 'undefined' && 'ResizeObserver' in window) {
      this.resizeObserver = new ResizeObserver(this.handleContainerResize);
    }
  }

  /**
   * Set container element to observe
   */
  setContainer(container: HTMLElement): void {
    if (this.container && this.resizeObserver) {
      this.resizeObserver.unobserve(this.container);
    }

    this.container = container;

    if (this.container && this.resizeObserver) {
      this.resizeObserver.observe(this.container);
    }
  }

  /**
   * Handle window resize
   */
  private handleResize = (): void => {
    this.updateViewport();
  };

  /**
   * Handle orientation change
   */
  private handleOrientationChange = (): void => {
    // Delay to allow orientation change to complete
    setTimeout(() => {
      this.updateViewport();
    }, 100);
  };

  /**
   * Handle container resize
   */
  private handleContainerResize = (entries: ResizeObserverEntry[]): void => {
    if (entries.length > 0) {
      this.updateViewport();
    }
  };

  /**
   * Update viewport information
   */
  private updateViewport(): void {
    if (typeof window === 'undefined') return;

    const width = this.container ? this.container.clientWidth : window.innerWidth;
    const height = this.container ? this.container.clientHeight : window.innerHeight;
    const breakpoint = this.getBreakpointForWidth(width);

    const viewport: ViewportInfo = {
      width,
      height,
      breakpoint,
      isMobile: breakpoint.name === 'xs' || breakpoint.name === 'sm',
      isTablet: breakpoint.name === 'md',
      isDesktop: breakpoint.name === 'lg' || breakpoint.name === 'xl' || breakpoint.name === 'xxl',
      orientation: width > height ? 'landscape' : 'portrait',
    };

    // Only notify if viewport changed significantly
    if (!this.currentViewport || this.hasViewportChanged(this.currentViewport, viewport)) {
      this.currentViewport = viewport;
      this.notifyListeners(viewport);
    }
  }

  /**
   * Check if viewport has changed significantly
   */
  private hasViewportChanged(oldViewport: ViewportInfo, newViewport: ViewportInfo): boolean {
    return (
      oldViewport.breakpoint.name !== newViewport.breakpoint.name ||
      oldViewport.orientation !== newViewport.orientation ||
      Math.abs(oldViewport.width - newViewport.width) > 50 ||
      Math.abs(oldViewport.height - newViewport.height) > 50
    );
  }

  /**
   * Get breakpoint for given width
   */
  private getBreakpointForWidth(width: number): Breakpoint {
    for (let i = this.config.breakpoints.length - 1; i >= 0; i--) {
      const breakpoint = this.config.breakpoints[i];
      if (width >= breakpoint.minWidth && (!breakpoint.maxWidth || width <= breakpoint.maxWidth)) {
        return breakpoint;
      }
    }

    // Fallback to default breakpoint
    return this.config.breakpoints.find(bp => bp.name === this.config.defaultBreakpoint) ||
           this.config.breakpoints[0];
  }

  /**
   * Get current viewport information
   */
  getCurrentViewport(): ViewportInfo | null {
    return this.currentViewport;
  }

  /**
   * Get breakpoint by name
   */
  getBreakpoint(name: string): Breakpoint | null {
    return this.config.breakpoints.find(bp => bp.name === name) || null;
  }

  /**
   * Get all breakpoints
   */
  getAllBreakpoints(): Breakpoint[] {
    return [...this.config.breakpoints];
  }

  /**
   * Subscribe to viewport changes
   */
  subscribe(callback: (viewport: ViewportInfo) => void): () => void {
    this.listeners.add(callback);

    // Call immediately with current viewport
    if (this.currentViewport) {
      callback(this.currentViewport);
    }

    // Return unsubscribe function
    return () => {
      this.listeners.delete(callback);
    };
  }

  /**
   * Notify listeners of viewport changes
   */
  private notifyListeners(viewport: ViewportInfo): void {
    this.listeners.forEach(callback => callback(viewport));
  }

  /**
   * Convert layout items for different breakpoints
   */
  convertLayoutForBreakpoint(
    layout: any[],
    fromBreakpoint: string,
    toBreakpoint: string
  ): any[] {
    const fromBp = this.getBreakpoint(fromBreakpoint);
    const toBp = this.getBreakpoint(toBreakpoint);

    if (!fromBp || !toBp) {
      return layout;
    }

    const columnRatio = toBp.columns / fromBp.columns;

    return layout.map(item => ({
      ...item,
      x: Math.round(item.x * columnRatio),
      w: Math.max(1, Math.round(item.w * columnRatio)),
      // Keep height the same for now, could be adjusted based on rowHeight ratio
    }));
  }

  /**
   * Get optimal component size for current viewport
   */
  getOptimalComponentSize(
    componentType: string,
    defaultSize: { w: number; h: number }
  ): { w: number; h: number } {
    if (!this.currentViewport) {
      return defaultSize;
    }

    const { breakpoint, isMobile, isTablet } = this.currentViewport;

    // Adjust sizes based on viewport
    let w = defaultSize.w;
    let h = defaultSize.h;

    if (isMobile) {
      // On mobile, make components wider and shorter
      w = Math.min(breakpoint.columns, Math.max(2, Math.round(defaultSize.w * 1.2)));
      h = Math.max(2, Math.round(defaultSize.h * 0.8));
    } else if (isTablet) {
      // On tablet, slightly adjust sizes
      w = Math.min(breakpoint.columns, Math.max(2, Math.round(defaultSize.w * 1.1)));
      h = Math.max(2, Math.round(defaultSize.h * 0.9));
    }

    // Component-specific adjustments
    switch (componentType) {
      case 'table':
        if (isMobile) {
          w = breakpoint.columns; // Full width on mobile
          h = Math.max(4, h);
        }
        break;
      case 'file_download':
        if (isMobile) {
          w = breakpoint.columns; // Full width on mobile
        }
        break;
      case 'metrics':
        if (isMobile) {
          w = Math.max(2, Math.round(breakpoint.columns / 2)); // Half width on mobile
        }
        break;
    }

    return { w, h };
  }

  /**
   * Check if drag/resize should be disabled
   */
  shouldDisableDragResize(): boolean {
    if (!this.currentViewport) {
      return false;
    }

    // Disable on very small screens for better UX
    return this.currentViewport.breakpoint.name === 'xs';
  }

  /**
   * Get touch-friendly handle size
   */
  getTouchHandleSize(): number {
    if (!this.currentViewport) {
      return 8;
    }

    // Larger handles on touch devices
    return this.currentViewport.isMobile ? 16 : 8;
  }

  /**
   * Get responsive grid configuration
   */
  getResponsiveGridConfig(): any {
    if (!this.currentViewport) {
      return null;
    }

    const { breakpoint } = this.currentViewport;

    return {
      columns: breakpoint.columns,
      rowHeight: breakpoint.rowHeight,
      margin: breakpoint.margin,
      containerPadding: breakpoint.containerPadding,
      // Add responsive-specific options
      compactType: this.currentViewport.isMobile ? 'vertical' : null,
      preventCollision: false,
      isDraggable: !this.shouldDisableDragResize(),
      isResizable: !this.shouldDisableDragResize(),
    };
  }

  /**
   * Cleanup
   */
  cleanup(): void {
    if (typeof window !== 'undefined') {
      window.removeEventListener('resize', this.handleResize);
      window.removeEventListener('orientationchange', this.handleOrientationChange);
    }

    if (this.resizeObserver && this.container) {
      this.resizeObserver.unobserve(this.container);
    }

    this.listeners.clear();
  }
}

/**
 * Global responsive layout manager instance
 */
let globalResponsiveManager: ResponsiveLayoutManager | null = null;

/**
 * Get or create global responsive layout manager
 */
export function getResponsiveLayoutManager(config?: Partial<ResponsiveConfig>): ResponsiveLayoutManager {
  if (!globalResponsiveManager) {
    globalResponsiveManager = new ResponsiveLayoutManager(config);
  }
  return globalResponsiveManager;
}

/**
 * React hook for responsive layout
 */
export function useResponsiveLayout(config?: Partial<ResponsiveConfig>) {
  const manager = getResponsiveLayoutManager(config);
  const [viewport, setViewport] = React.useState<ViewportInfo | null>(
    manager.getCurrentViewport()
  );

  React.useEffect(() => {
    const unsubscribe = manager.subscribe(setViewport);
    return unsubscribe;
  }, [manager]);

  const setContainer = React.useCallback((container: HTMLElement) => {
    manager.setContainer(container);
  }, [manager]);

  const getOptimalSize = React.useCallback((
    componentType: string,
    defaultSize: { w: number; h: number }
  ) => {
    return manager.getOptimalComponentSize(componentType, defaultSize);
  }, [manager, viewport]);

  const convertLayout = React.useCallback((
    layout: any[],
    fromBreakpoint: string,
    toBreakpoint: string
  ) => {
    return manager.convertLayoutForBreakpoint(layout, fromBreakpoint, toBreakpoint);
  }, [manager]);

  return {
    viewport,
    setContainer,
    getOptimalSize,
    convertLayout,
    shouldDisableDragResize: manager.shouldDisableDragResize(),
    getTouchHandleSize: manager.getTouchHandleSize(),
    getResponsiveGridConfig: manager.getResponsiveGridConfig(),
  };
}

// Add React import for hooks
import React from 'react';