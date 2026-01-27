/**
 * useLoadingState - React Hook for loading state management
 * 
 * 提供响应式的加载状态访问，自动订阅 LoadingStateManager 的状态变化
 * 
 * Requirements: 3.1, 3.2
 */

import { useState, useEffect, useCallback } from 'react';
import { loadingStateManager, SessionLoadingState } from '../managers/LoadingStateManager';
import { createLogger } from '../utils/systemLog';

const logger = createLogger('useLoadingState');

/**
 * UseLoadingStateResult - Hook 返回值接口
 * 
 * 提供全局加载状态的访问和查询方法
 */
export interface UseLoadingStateResult {
    /** 所有正在加载的会话ID集合 */
    loadingThreadIds: Set<string>;
    /** 当前加载中的会话数量 */
    loadingCount: number;
    /** 是否有任何会话正在加载 */
    isAnyLoading: boolean;
    /** 检查指定会话是否正在加载 */
    isLoading: (threadId: string | null | undefined) => boolean;
    /** 获取指定会话的进度信息 */
    getProgress: (threadId: string) => SessionLoadingState['progress'] | undefined;
    /** 获取指定会话的错误信息 */
    getError: (threadId: string) => SessionLoadingState['error'] | undefined;
}

/**
 * React Hook: 订阅并响应加载状态变化
 * 
 * 使用 React useState 和 useEffect 订阅 LoadingStateManager 的状态变化，
 * 并在组件卸载时自动清理订阅。
 * 
 * @returns UseLoadingStateResult 包含加载状态和查询方法的对象
 */
export function useLoadingState(): UseLoadingStateResult {
    const [loadingThreadIds, setLoadingThreadIds] = useState<Set<string>>(
        () => loadingStateManager.getLoadingThreadIds()
    );

    useEffect(() => {
        logger.info(`[useLoadingState] Subscribing to LoadingStateManager`);
        // 订阅状态变化
        const unsubscribe = loadingStateManager.subscribe((newLoadingThreadIds) => {
            logger.info(`[useLoadingState] Received update: [${[...newLoadingThreadIds].join(',')}]`);
            setLoadingThreadIds(new Set(newLoadingThreadIds));
        });

        // 清理订阅
        return () => {
            logger.info(`[useLoadingState] Unsubscribing from LoadingStateManager`);
            unsubscribe();
        };
    }, []);

    /**
     * 检查指定会话是否正在加载
     */
    const isLoading = useCallback((threadId: string | null | undefined): boolean => {
        if (!threadId) return false;
        return loadingThreadIds.has(threadId);
    }, [loadingThreadIds]);

    /**
     * 获取指定会话的进度信息
     */
    const getProgress = useCallback((threadId: string): SessionLoadingState['progress'] | undefined => {
        return loadingStateManager.getProgress(threadId);
    }, []);

    /**
     * 获取指定会话的错误信息
     */
    const getError = useCallback((threadId: string): SessionLoadingState['error'] | undefined => {
        return loadingStateManager.getError(threadId);
    }, []);

    // 计算派生状态
    const loadingCount = loadingThreadIds.size;
    const isAnyLoading = loadingCount > 0;

    return {
        loadingThreadIds,
        loadingCount,
        isAnyLoading,
        isLoading,
        getProgress,
        getError
    };
}

export default useLoadingState;
