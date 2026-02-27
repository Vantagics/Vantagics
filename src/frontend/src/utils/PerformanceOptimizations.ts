/**
 * Performance Optimization Utilities
 * 
 * 提供性能优化相关的工具函数和Hook
 */

import { useCallback, useRef, useEffect } from 'react';

/**
 * 防抖Hook - 延迟执行函数直到停止调用一段时间后
 * 
 * @param callback - 要防抖的函数
 * @param delay - 延迟时间（毫秒）
 * @returns 防抖后的函数
 * 
 * @example
 * const debouncedSearch = useDebounce((query: string) => {
 *   performSearch(query);
 * }, 300);
 */
export function useDebounce<T extends (...args: any[]) => any>(
  callback: T,
  delay: number
): (...args: Parameters<T>) => void {
  const timeoutRef = useRef<number>();
  const callbackRef = useRef(callback);

  // 更新回调引用
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  // 清理定时器
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        window.clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return useCallback(
    (...args: Parameters<T>) => {
      if (timeoutRef.current) {
        window.clearTimeout(timeoutRef.current);
      }
      timeoutRef.current = window.setTimeout(() => {
        callbackRef.current(...args);
      }, delay);
    },
    [delay]
  );
}

/**
 * 节流Hook - 限制函数在指定时间内只能执行一次
 * 
 * @param callback - 要节流的函数
 * @param delay - 节流时间（毫秒）
 * @returns 节流后的函数
 * 
 * @example
 * const throttledScroll = useThrottle((event: Event) => {
 *   handleScroll(event);
 * }, 100);
 */
export function useThrottle<T extends (...args: any[]) => any>(
  callback: T,
  delay: number
): (...args: Parameters<T>) => void {
  const lastRunRef = useRef<number>(0);
  const timeoutRef = useRef<number>();
  const callbackRef = useRef(callback);

  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        window.clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return useCallback(
    (...args: Parameters<T>) => {
      const now = Date.now();
      const timeSinceLastRun = now - lastRunRef.current;

      if (timeSinceLastRun >= delay) {
        callbackRef.current(...args);
        lastRunRef.current = now;
      } else {
        if (timeoutRef.current) {
          window.clearTimeout(timeoutRef.current);
        }
        timeoutRef.current = window.setTimeout(() => {
          callbackRef.current(...args);
          lastRunRef.current = Date.now();
        }, delay - timeSinceLastRun);
      }
    },
    [delay]
  );
}

/**
 * 深度比较Hook - 用于useMemo/useCallback的依赖比较
 * 
 * @param value - 要比较的值
 * @returns 如果值深度相等则返回之前的引用
 * 
 * @example
 * const memoizedValue = useMemo(() => {
 *   return expensiveCalculation(data);
 * }, [useDeepCompare(data)]);
 */
export function useDeepCompare<T>(value: T): T {
  const ref = useRef<T>(value);
  const signalRef = useRef<number>(0);

  if (!deepEqual(value, ref.current)) {
    ref.current = value;
    signalRef.current += 1;
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useCallback(() => ref.current, [signalRef.current])();
}

/**
 * 深度相等比较
 */
function deepEqual(a: any, b: any): boolean {
  if (a === b) return true;
  if (a == null || b == null) return false;
  if (typeof a !== 'object' || typeof b !== 'object') return false;

  const keysA = Object.keys(a);
  const keysB = Object.keys(b);

  if (keysA.length !== keysB.length) return false;

  for (const key of keysA) {
    if (!keysB.includes(key)) return false;
    if (!deepEqual(a[key], b[key])) return false;
  }

  return true;
}

/**
 * 虚拟滚动Hook - 用于大列表优化
 * 
 * @param items - 完整的项目列表
 * @param itemHeight - 每个项目的高度
 * @param containerHeight - 容器高度
 * @param overscan - 预渲染的额外项目数
 * @returns 可见项目和滚动处理函数
 * 
 * @example
 * const { visibleItems, onScroll, totalHeight } = useVirtualScroll(
 *   allItems,
 *   50,
 *   500,
 *   5
 * );
 */
export function useVirtualScroll<T>(
  items: T[],
  itemHeight: number,
  containerHeight: number,
  overscan: number = 3
) {
  const scrollTopRef = useRef(0);

  const totalHeight = items.length * itemHeight;
  const startIndex = Math.max(0, Math.floor(scrollTopRef.current / itemHeight) - overscan);
  const endIndex = Math.min(
    items.length,
    Math.ceil((scrollTopRef.current + containerHeight) / itemHeight) + overscan
  );

  const visibleItems = items.slice(startIndex, endIndex).map((item, index) => ({
    item,
    index: startIndex + index,
    offsetTop: (startIndex + index) * itemHeight,
  }));

  const onScroll = useCallback((event: React.UIEvent<HTMLElement>) => {
    scrollTopRef.current = event.currentTarget.scrollTop;
  }, []);

  return {
    visibleItems,
    onScroll,
    totalHeight,
    startIndex,
    endIndex,
  };
}

/**
 * 懒加载Hook - 延迟加载组件或数据
 * 
 * @param loader - 加载函数
 * @param deps - 依赖数组
 * @returns 加载状态和数据
 * 
 * @example
 * const { data, loading, error } = useLazyLoad(
 *   () => import('./HeavyComponent'),
 *   []
 * );
 */
export function useLazyLoad<T>(
  loader: () => Promise<T>,
  deps: React.DependencyList = []
) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setLoading(true);
      setError(null);

      try {
        const result = await loader();
        if (!cancelled) {
          setData(result);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err : new Error(String(err)));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    load();

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return { data, loading, error };
}

/**
 * 内存清理Hook - 自动清理大对象
 * 
 * @param cleanup - 清理函数
 * @param deps - 依赖数组
 * 
 * @example
 * useMemoryCleanup(() => {
 *   // 清理大对象
 *   largeDataCache.clear();
 * }, []);
 */
export function useMemoryCleanup(
  cleanup: () => void,
  deps: React.DependencyList = []
) {
  useEffect(() => {
    return () => {
      cleanup();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
}

// 导入useState
import { useState } from 'react';
