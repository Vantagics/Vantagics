import { useEffect, useRef, useCallback } from 'react';
import { EventsOn } from '../../wailsjs/runtime/runtime';

/**
 * Custom hook for subscribing to Wails events with automatic cleanup.
 * Reduces boilerplate code for EventsOn + useEffect pattern.
 * 
 * @param eventName - The name of the Wails event to listen for
 * @param handler - Callback function to handle the event
 * @param dependencies - Optional dependency array (like useEffect)
 * 
 * @example
 * // Basic usage
 * useWailsEvent('analysis-progress', (data) => {
 *   console.log('Progress:', data);
 * });
 * 
 * @example
 * // With dependencies
 * useWailsEvent('thread-updated', (threadId) => {
 *   if (threadId === activeThreadId) {
 *     refreshData();
 *   }
 * }, [activeThreadId]);
 * 
 * @example
 * // With type safety
 * interface ProgressData {
 *   stage: string;
 *   progress: number;
 * }
 * useWailsEvent<ProgressData>('analysis-progress', (data) => {
 *   setProgress(data.progress);
 * });
 */
export function useWailsEvent<T = any>(
  eventName: string,
  handler: (data: T) => void,
  dependencies: React.DependencyList = []
): void {
  // Use ref to store the latest handler to avoid re-subscribing
  const handlerRef = useRef(handler);
  
  // Update ref when handler changes
  useEffect(() => {
    handlerRef.current = handler;
  }, [handler]);
  
  // Stable callback that uses the ref
  const stableHandler = useCallback((data: T) => {
    handlerRef.current(data);
  }, []);
  
  useEffect(() => {
    const cleanup = EventsOn(eventName, stableHandler);
    
    return () => {
      if (cleanup && typeof cleanup === 'function') {
        cleanup();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [eventName, stableHandler, ...dependencies]);
}

/**
 * Hook for subscribing to multiple Wails events at once.
 * 
 * @param events - Object mapping event names to handlers
 * @param dependencies - Optional dependency array
 * 
 * @example
 * useWailsEvents({
 *   'analysis-progress': (data) => setProgress(data),
 *   'analysis-completed': (data) => setCompleted(true),
 *   'analysis-error': (error) => setError(error)
 * });
 */
export function useWailsEvents(
  events: Record<string, (data: any) => void>,
  dependencies: React.DependencyList = []
): void {
  useEffect(() => {
    const cleanups: Array<() => void> = [];
    
    Object.entries(events).forEach(([eventName, handler]) => {
      const cleanup = EventsOn(eventName, handler);
      if (cleanup && typeof cleanup === 'function') {
        cleanups.push(cleanup);
      }
    });
    
    return () => {
      cleanups.forEach(cleanup => cleanup());
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [events, ...dependencies]);
}

/**
 * Hook for one-time event subscription (fires only once then unsubscribes).
 * 
 * @param eventName - The name of the Wails event to listen for
 * @param handler - Callback function to handle the event (called only once)
 * @param dependencies - Optional dependency array
 * 
 * @example
 * useWailsEventOnce('initialization-complete', () => {
 *   console.log('App initialized');
 *   setReady(true);
 * });
 */
export function useWailsEventOnce<T = any>(
  eventName: string,
  handler: (data: T) => void,
  dependencies: React.DependencyList = []
): void {
  const hasHandledRef = useRef(false);
  
  useEffect(() => {
    hasHandledRef.current = false;
  }, dependencies);
  
  useWailsEvent<T>(
    eventName,
    (data) => {
      if (!hasHandledRef.current) {
        hasHandledRef.current = true;
        handler(data);
      }
    },
    dependencies
  );
}

/**
 * Hook for conditional event subscription.
 * Only subscribes when the condition is true.
 * 
 * @param eventName - The name of the Wails event to listen for
 * @param handler - Callback function to handle the event
 * @param condition - Boolean condition to enable/disable subscription
 * @param dependencies - Optional dependency array
 * 
 * @example
 * useWailsEventConditional(
 *   'chat-message',
 *   (message) => addMessage(message),
 *   isChatOpen, // Only listen when chat is open
 *   [isChatOpen]
 * );
 */
export function useWailsEventConditional<T = any>(
  eventName: string,
  handler: (data: T) => void,
  condition: boolean,
  dependencies: React.DependencyList = []
): void {
  useEffect(() => {
    if (!condition) {
      return;
    }
    
    const cleanup = EventsOn(eventName, handler);
    
    return () => {
      if (cleanup && typeof cleanup === 'function') {
        cleanup();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [eventName, handler, condition, ...dependencies]);
}
