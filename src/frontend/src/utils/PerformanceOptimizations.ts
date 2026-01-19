/**
 * Performance Optimization Utilities
 * 
 * Collection of utilities and hooks for optimizing dashboard performance
 * including memoization, lazy loading, and efficient rendering.
 */

import React, { useMemo, useCallback, useRef, useEffect, useState } from 'react';
import { LayoutItem, ComponentType } from './ComponentManager';

// ============================================================================
// MEMOIZATION UTILITIES
// ============================================================================

/**
 * Memoized layout item comparison for React.memo
 */
export const layoutItemsEqual = (
  prevItems: LayoutItem[],
  nextItems: LayoutItem[]
): boolean => {
  if (prevItems.length !== nextItems.length) return false;
  
  return prevItems.every((prevItem, index) => {
    const nextItem = nextItems[index];
    return (
      prevItem.i === nextItem.i &&
      prevItem.x === nextItem.x &&
      prevItem.y === nextItem.y &&
      prevItem.w === nextItem.w &&
      prevItem.h === nextItem.h &&
      prevItem.type === nextItem.type &&
      prevItem.instanceIdx === nextItem.instanceIdx &&
      prevItem.static === nextItem.static
    );
  });
};

/**
 * Memoized component props comparison
 */
export const componentPropsEqual = (
  prevProps: any,
  nextProps: any
): boolean => {
  // Compare layout item
  if (!layoutItemsEqual([prevProps.layoutItem], [nextProps.layoutItem])) {
    return false;
  }
  
  // Compare other props
  return (
    prevProps.isEditMode === nextProps.isEditMode &&
    prevProps.isLocked === nextProps.isLocked &&
    prevProps.gridConfig === nextProps.gridConfig
  );
};

// ============================================================================
// LAZY LOADING UTILITIES
// ============================================================================

/**
 * Lazy component loader with error boundary
 */
export const createLazyComponent = <T extends React.ComponentType<any>>(
  importFn: () => Promise<{ default: T }>,
  fallback?: React.ComponentType
) => {
  const LazyComponent = React.lazy(importFn);
  
  return React.forwardRef<any, React.ComponentProps<T>>((props, ref) => (
    <React.Suspense 
      fallback={
        fallback ? 
        React.createElement(fallback) : 
        <div className="animate-pulse bg-gray-200 rounded h-32 w-full" />
      }
    >
      <LazyComponent {...props} ref={ref} />
    </React.Suspense>
  ));
};

/**
 * Intersection Observer hook for lazy loading
 */
export const useIntersectionObserver = (
  options: IntersectionObserverInit = {}
) => {
  const [isIntersecting, setIsIntersecting] = useState(false);
  const [hasIntersected, setHasIntersected] = useState(false);
  const targetRef = useRef<HTMLElement>(null);

  useEffect(() => {
    const target = targetRef.current;
    if (!target) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsIntersecting(entry.isIntersecting);
        if (entry.isIntersecting && !hasIntersected) {
          setHasIntersected(true);
        }
      },
      {
        threshold: 0.1,
        rootMargin: '50px',
        ...options
      }
    );

    observer.observe(target);

    return () => {
      observer.unobserve(target);
    };
  }, [hasIntersected, options]);

  return { targetRef, isIntersecting, hasIntersected };
};

/**
 * Lazy component wrapper with intersection observer
 */
export const LazyComponentWrapper: React.FC<{
  children: React.ReactNode;
  fallback?: React.ReactNode;
  className?: string;
}> = ({ children, fallback, className = '' }) => {
  const { targetRef, hasIntersected } = useIntersectionObserver();

  return (
    <div ref={targetRef} className={className}>
      {hasIntersected ? children : (
        fallback || (
          <div className="animate-pulse bg-gray-200 rounded h-32 w-full" />
        )
      )}
    </div>
  );
};

// ============================================================================
// DEBOUNCING AND THROTTLING
// ============================================================================

/**
 * Debounced callback hook
 */
export const useDebouncedCallback = <T extends (...args: any[]) => any>(
  callback: T,
  delay: number
): T => {
  const timeoutRef = useRef<NodeJS.Timeout>();

  return useCallback(
    ((...args: Parameters<T>) => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      
      timeoutRef.current = setTimeout(() => {
        callback(...args);
      }, delay);
    }) as T,
    [callback, delay]
  );
};

/**
 * Throttled callback hook
 */
export const useThrottledCallback = <T extends (...args: any[]) => any>(
  callback: T,
  delay: number
): T => {
  const lastCallRef = useRef<number>(0);
  const timeoutRef = useRef<NodeJS.Timeout>();

  return useCallback(
    ((...args: Parameters<T>) => {
      const now = Date.now();
      const timeSinceLastCall = now - lastCallRef.current;

      if (timeSinceLastCall >= delay) {
        lastCallRef.current = now;
        callback(...args);
      } else {
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current);
        }
        
        timeoutRef.current = setTimeout(() => {
          lastCallRef.current = Date.now();
          callback(...args);
        }, delay - timeSinceLastCall);
      }
    }) as T,
    [callback, delay]
  );
};

// ============================================================================
// VIRTUAL SCROLLING
// ============================================================================

/**
 * Virtual scrolling hook for large lists
 */
export const useVirtualScrolling = <T>(
  items: T[],
  itemHeight: number,
  containerHeight: number,
  overscan: number = 5
) => {
  const [scrollTop, setScrollTop] = useState(0);

  const visibleRange = useMemo(() => {
    const startIndex = Math.max(0, Math.floor(scrollTop / itemHeight) - overscan);
    const endIndex = Math.min(
      items.length - 1,
      Math.ceil((scrollTop + containerHeight) / itemHeight) + overscan
    );
    
    return { startIndex, endIndex };
  }, [scrollTop, itemHeight, containerHeight, items.length, overscan]);

  const visibleItems = useMemo(() => {
    return items.slice(visibleRange.startIndex, visibleRange.endIndex + 1);
  }, [items, visibleRange]);

  const totalHeight = items.length * itemHeight;
  const offsetY = visibleRange.startIndex * itemHeight;

  const handleScroll = useCallback((event: React.UIEvent<HTMLDivElement>) => {
    setScrollTop(event.currentTarget.scrollTop);
  }, []);

  return {
    visibleItems,
    totalHeight,
    offsetY,
    handleScroll,
    visibleRange
  };
};

// ============================================================================
// COMPONENT OPTIMIZATION HOOKS
// ============================================================================

/**
 * Optimized layout state hook with memoization
 */
export const useOptimizedLayoutState = (initialItems: LayoutItem[]) => {
  const [items, setItems] = useState(initialItems);
  
  const memoizedItems = useMemo(() => items, [items]);
  
  const updateItems = useCallback((newItems: LayoutItem[]) => {
    setItems(prevItems => {
      if (layoutItemsEqual(prevItems, newItems)) {
        return prevItems; // Prevent unnecessary re-renders
      }
      return newItems;
    });
  }, []);

  const updateItem = useCallback((itemId: string, updates: Partial<LayoutItem>) => {
    setItems(prevItems => 
      prevItems.map(item => 
        item.i === itemId ? { ...item, ...updates } : item
      )
    );
  }, []);

  return {
    items: memoizedItems,
    updateItems,
    updateItem
  };
};

/**
 * Optimized component visibility hook
 */
export const useOptimizedVisibility = (
  items: LayoutItem[],
  isEditMode: boolean,
  dataAvailability: Record<string, boolean>
) => {
  return useMemo(() => {
    const visibilityMap = new Map<string, boolean>();
    
    items.forEach(item => {
      const hasData = dataAvailability[item.i] ?? true;
      const isVisible = isEditMode || hasData;
      visibilityMap.set(item.i, isVisible);
    });
    
    return visibilityMap;
  }, [items, isEditMode, dataAvailability]);
};

/**
 * Optimized drag state hook
 */
export const useOptimizedDragState = () => {
  const [dragState, setDragState] = useState<{
    isDragging: boolean;
    draggedItem: string | null;
    dragOffset: { x: number; y: number } | null;
  }>({
    isDragging: false,
    draggedItem: null,
    dragOffset: null
  });

  const startDrag = useCallback((itemId: string, offset: { x: number; y: number }) => {
    setDragState({
      isDragging: true,
      draggedItem: itemId,
      dragOffset: offset
    });
  }, []);

  const updateDrag = useCallback((offset: { x: number; y: number }) => {
    setDragState(prev => ({
      ...prev,
      dragOffset: offset
    }));
  }, []);

  const endDrag = useCallback(() => {
    setDragState({
      isDragging: false,
      draggedItem: null,
      dragOffset: null
    });
  }, []);

  return {
    dragState,
    startDrag,
    updateDrag,
    endDrag
  };
};

// ============================================================================
// PERFORMANCE MONITORING
// ============================================================================

/**
 * Performance monitoring hook
 */
export const usePerformanceMonitor = (componentName: string) => {
  const renderCountRef = useRef(0);
  const lastRenderTimeRef = useRef(Date.now());

  useEffect(() => {
    renderCountRef.current += 1;
    const now = Date.now();
    const timeSinceLastRender = now - lastRenderTimeRef.current;
    lastRenderTimeRef.current = now;

    if (process.env.NODE_ENV === 'development') {
      console.log(`[Performance] ${componentName} render #${renderCountRef.current}, time since last: ${timeSinceLastRender}ms`);
    }
  });

  return {
    renderCount: renderCountRef.current,
    logPerformance: (operation: string, duration: number) => {
      if (process.env.NODE_ENV === 'development') {
        console.log(`[Performance] ${componentName} ${operation}: ${duration}ms`);
      }
    }
  };
};

/**
 * Measure component render time
 */
export const withPerformanceMeasurement = <P extends object>(
  Component: React.ComponentType<P>,
  componentName: string
) => {
  return React.memo((props: P) => {
    const startTime = performance.now();
    
    useEffect(() => {
      const endTime = performance.now();
      const renderTime = endTime - startTime;
      
      if (process.env.NODE_ENV === 'development' && renderTime > 16) {
        console.warn(`[Performance] ${componentName} slow render: ${renderTime.toFixed(2)}ms`);
      }
    });

    return <Component {...props} />;
  });
};

// ============================================================================
// MEMORY OPTIMIZATION
// ============================================================================

/**
 * Cleanup effect hook for preventing memory leaks
 */
export const useCleanupEffect = (cleanup: () => void, deps: React.DependencyList) => {
  useEffect(() => {
    return cleanup;
  }, deps);
};

/**
 * Weak map cache for component instances
 */
export class ComponentCache {
  private cache = new WeakMap<object, any>();

  get<T>(key: object): T | undefined {
    return this.cache.get(key);
  }

  set<T>(key: object, value: T): void {
    this.cache.set(key, value);
  }

  has(key: object): boolean {
    return this.cache.has(key);
  }

  delete(key: object): boolean {
    return this.cache.delete(key);
  }
}

// Global component cache instance
export const globalComponentCache = new ComponentCache();

// ============================================================================
// EXPORTS
// ============================================================================

export default {
  // Memoization
  layoutItemsEqual,
  componentPropsEqual,
  
  // Lazy loading
  createLazyComponent,
  useIntersectionObserver,
  LazyComponentWrapper,
  
  // Debouncing/Throttling
  useDebouncedCallback,
  useThrottledCallback,
  
  // Virtual scrolling
  useVirtualScrolling,
  
  // Component optimization
  useOptimizedLayoutState,
  useOptimizedVisibility,
  useOptimizedDragState,
  
  // Performance monitoring
  usePerformanceMonitor,
  withPerformanceMeasurement,
  
  // Memory optimization
  useCleanupEffect,
  ComponentCache,
  globalComponentCache
};