/**
 * useSessionStatus - React Hook for single session status tracking
 * 
 * 提供针对单个会话的状态订阅，包括加载状态、进度信息、错误状态和已用时间
 * 
 * Requirements: 1.1, 1.2, 5.5
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { loadingStateManager, SessionLoadingState } from '../managers/LoadingStateManager';
import { createLogger } from '../utils/systemLog';

const logger = createLogger('useSessionStatus');

/**
 * UseSessionStatusResult - Hook 返回值接口
 * 
 * 提供单个会话的完整状态信息
 */
export interface UseSessionStatusResult {
    /** 会话是否正在加载 */
    isLoading: boolean;
    /** 会话的进度信息 */
    progress: SessionLoadingState['progress'] | undefined;
    /** 会话的错误信息 */
    error: SessionLoadingState['error'] | undefined;
    /** 加载开始时间戳 */
    startTime: number | undefined;
    /** 已用时间（毫秒） */
    elapsedTime: number;
}

/** 默认返回值，用于 threadId 为 null 或会话不存在时 */
const DEFAULT_RESULT: UseSessionStatusResult = {
    isLoading: false,
    progress: undefined,
    error: undefined,
    startTime: undefined,
    elapsedTime: 0
};

/** 已用时间更新间隔（毫秒） */
const ELAPSED_TIME_UPDATE_INTERVAL = 100;

/**
 * React Hook: 订阅并响应单个会话的状态变化
 * 
 * 使用 LoadingStateManager.subscribeToSession() 订阅特定会话的状态变化，
 * 并在会话加载时使用 setInterval 跟踪已用时间。
 * 
 * @param threadId 会话ID，可以为 null
 * @returns UseSessionStatusResult 包含会话状态的对象
 * 
 * Requirements: 1.1, 1.2, 5.5
 */
export function useSessionStatus(threadId: string | null): UseSessionStatusResult {
    // 会话状态
    const [sessionState, setSessionState] = useState<SessionLoadingState | undefined>(
        () => threadId ? loadingStateManager.getSessionState(threadId) : undefined
    );
    
    // 已用时间
    const [elapsedTime, setElapsedTime] = useState<number>(0);
    
    // 用于存储 interval ID 的 ref
    const intervalRef = useRef<number | null>(null);
    
    // 用于存储开始时间的 ref（避免闭包问题）
    const startTimeRef = useRef<number | undefined>(undefined);

    /**
     * 清理 interval
     */
    const clearElapsedTimeInterval = useCallback(() => {
        if (intervalRef.current !== null) {
            window.clearInterval(intervalRef.current);
            intervalRef.current = null;
            logger.info(`[useSessionStatus] Cleared elapsed time interval for threadId=${threadId}`);
        }
    }, [threadId]);

    /**
     * 启动已用时间计时器
     */
    const startElapsedTimeInterval = useCallback((startTime: number) => {
        // 先清理现有的 interval
        clearElapsedTimeInterval();
        
        startTimeRef.current = startTime;
        
        // 立即计算一次已用时间
        setElapsedTime(Date.now() - startTime);
        
        // 启动定时更新
        intervalRef.current = window.setInterval(() => {
            if (startTimeRef.current !== undefined) {
                setElapsedTime(Date.now() - startTimeRef.current);
            }
        }, ELAPSED_TIME_UPDATE_INTERVAL);
        
        logger.info(`[useSessionStatus] Started elapsed time interval for threadId=${threadId}, startTime=${startTime}`);
    }, [threadId, clearElapsedTimeInterval]);

    /**
     * 当 threadId 变化时，立即同步更新状态
     * 这确保在切换会话时不会出现状态闪烁
     */
    useEffect(() => {
        if (threadId) {
            const currentState = loadingStateManager.getSessionState(threadId);
            logger.info(`[useSessionStatus] threadId changed to ${threadId}, immediate state: ${JSON.stringify(currentState)}`);
            setSessionState(currentState);
            
            // 如果新会话正在加载，立即启动计时器
            if (currentState?.isLoading && currentState.startTime) {
                startElapsedTimeInterval(currentState.startTime);
            } else {
                clearElapsedTimeInterval();
                setElapsedTime(currentState?.startTime ? Date.now() - currentState.startTime : 0);
            }
        } else {
            setSessionState(undefined);
            setElapsedTime(0);
            clearElapsedTimeInterval();
        }
    }, [threadId]); // 只依赖 threadId，不依赖 startElapsedTimeInterval 和 clearElapsedTimeInterval

    /**
     * 订阅会话状态变化
     */
    useEffect(() => {
        // 如果 threadId 为 null，重置状态并返回
        if (!threadId) {
            logger.info(`[useSessionStatus] threadId is null, resetting state`);
            setSessionState(undefined);
            setElapsedTime(0);
            clearElapsedTimeInterval();
            return;
        }

        logger.info(`[useSessionStatus] Subscribing to session: threadId=${threadId}`);

        // 订阅会话状态变化
        const unsubscribe = loadingStateManager.subscribeToSession(threadId, (state) => {
            logger.info(`[useSessionStatus] Received state update for threadId=${threadId}: ${JSON.stringify(state)}`);
            setSessionState(state);

            // 根据加载状态管理计时器
            if (state?.isLoading && state.startTime) {
                // 会话正在加载，启动计时器
                startElapsedTimeInterval(state.startTime);
            } else {
                // 会话不在加载，停止计时器
                clearElapsedTimeInterval();
                
                // 如果有开始时间，计算最终的已用时间
                if (state?.startTime) {
                    setElapsedTime(Date.now() - state.startTime);
                } else {
                    setElapsedTime(0);
                }
            }
        });

        // 清理函数
        return () => {
            logger.info(`[useSessionStatus] Unsubscribing from session: threadId=${threadId}`);
            unsubscribe();
            clearElapsedTimeInterval();
        };
    }, [threadId, startElapsedTimeInterval, clearElapsedTimeInterval]);

    // 如果 threadId 为 null，返回默认值
    if (!threadId) {
        return DEFAULT_RESULT;
    }

    // 构建返回值
    return {
        isLoading: sessionState?.isLoading ?? false,
        progress: sessionState?.progress,
        error: sessionState?.error,
        startTime: sessionState?.startTime,
        elapsedTime
    };
}

export default useSessionStatus;
