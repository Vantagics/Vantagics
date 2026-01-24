/**
 * LoadingStateManager - 多会话并发加载状态管理器
 * 
 * 统一管理所有分析会话的加载状态，支持：
 * - 多会话并发分析
 * - 独立的进度跟踪
 * - 自动超时清理
 * - 事件驱动的状态更新
 */

import { createLogger } from '../utils/systemLog';

const logger = createLogger('LoadingStateManager');

/**
 * SessionLoadingState - 会话加载状态接口
 * 
 * 定义单个分析会话的完整状态信息，包括：
 * - 基本状态（加载中/完成）
 * - 进度信息（阶段、百分比、消息）
 * - 错误信息（错误代码和消息）
 * 
 * Requirements: 4.1, 4.2
 */
export interface SessionLoadingState {
    threadId: string;           // 会话唯一标识
    isLoading: boolean;         // 是否正在加载
    startTime: number;          // 开始时间戳
    progress?: {
        stage: string;          // 当前阶段: 'initializing' | 'analyzing' | 'generating' | 'complete'
        progress: number;       // 进度百分比 0-100
        message: string;        // 显示消息
        step: number;           // 当前步骤
        total: number;          // 总步骤数
    };
    error?: {
        code: string;           // 错误代码
        message: string;        // 错误消息
    };
}

/**
 * LoadingSession - 向后兼容的加载会话接口
 * @deprecated 请使用 SessionLoadingState 接口
 */
export interface LoadingSession {
    threadId: string;
    startTime: number;
    progress?: {
        stage: string;
        progress: number;
        message: string;
        step: number;
        total: number;
    };
}

type LoadingStateListener = (loadingThreadIds: Set<string>) => void;
type SessionStateListener = (state: SessionLoadingState | undefined) => void;

class LoadingStateManager {
    private static instance: LoadingStateManager;
    private loadingSessions: Map<string, SessionLoadingState> = new Map();
    private listeners: Set<LoadingStateListener> = new Set();
    private sessionListeners: Map<string, Set<SessionStateListener>> = new Map();
    private timeoutIds: Map<string, number> = new Map();
    private readonly TIMEOUT_MS = 120000; // 2分钟超时
    private initialized = false;

    private constructor() {
        // 私有构造函数，确保单例
    }

    static getInstance(): LoadingStateManager {
        if (!LoadingStateManager.instance) {
            LoadingStateManager.instance = new LoadingStateManager();
        }
        return LoadingStateManager.instance;
    }

    /**
     * 初始化管理器，注册全局事件监听
     * 只需调用一次
     */
    initialize(): void {
        if (this.initialized) {
            logger.info('[LoadingStateManager] Already initialized, skipping');
            return;
        }
        
        this.initialized = true;
        logger.info('[LoadingStateManager] Initializing...');

        // 监听前端 CustomEvent (来自 ChatSidebar)
        window.addEventListener('chat-loading-frontend', this.handleLoadingEvent);
        
        // 监听后端 Wails 事件
        this.setupWailsListeners();
        
        logger.info('[LoadingStateManager] Initialized successfully');
    }

    /**
     * 设置 Wails 事件监听器
     */
    private setupWailsListeners(): void {
        // 动态导入 Wails runtime 以避免循环依赖
        import('../../wailsjs/runtime/runtime').then(({ EventsOn }) => {
            // 监听后端加载状态事件
            EventsOn('chat-loading', (data: any) => {
                logger.info(`[LoadingStateManager] chat-loading (backend): ${JSON.stringify(data)}`);
                this.processLoadingData(data);
            });

            // 监听分析完成事件
            EventsOn('analysis-completed', (payload: any) => {
                logger.info(`[LoadingStateManager] analysis-completed: ${JSON.stringify(payload)}`);
                const threadId = payload?.threadId;
                if (threadId) {
                    this.setLoading(threadId, false);
                }
            });

            // 监听分析错误事件
            EventsOn('analysis-error', (payload: any) => {
                logger.info(`[LoadingStateManager] analysis-error: ${JSON.stringify(payload)}`);
                const threadId = payload?.threadId;
                if (threadId) {
                    // 使用 setError 方法设置错误状态
                    const error = {
                        code: payload?.code || 'ANALYSIS_ERROR',
                        message: payload?.message || '分析过程中发生错误'
                    };
                    this.setError(threadId, error);
                }
            });

            // 监听分析取消事件
            EventsOn('analysis-cancelled', (data: any) => {
                logger.info(`[LoadingStateManager] analysis-cancelled: ${JSON.stringify(data)}`);
                const threadId = data?.threadId;
                if (threadId) {
                    this.setLoading(threadId, false);
                }
            });

            // 监听进度更新事件
            EventsOn('analysis-progress', (update: any) => {
                if (update?.threadId) {
                    this.updateProgress(update.threadId, update);
                }
            });

            // 监听分析队列状态事件（并发控制等待）
            EventsOn('analysis-queue-status', (data: any) => {
                logger.info(`[LoadingStateManager] analysis-queue-status: ${JSON.stringify(data)}`);
                const threadId = data?.threadId;
                if (threadId) {
                    if (data.status === 'waiting') {
                        // 更新进度显示等待状态
                        this.updateProgress(threadId, {
                            stage: 'waiting',
                            progress: 0,
                            message: data.message || '等待分析队列中...',
                            step: 0,
                            total: 0
                        });
                    } else if (data.status === 'starting') {
                        // 开始分析，更新进度
                        this.updateProgress(threadId, {
                            stage: 'initializing',
                            progress: 0,
                            message: data.message || '开始分析',
                            step: 0,
                            total: 0
                        });
                    }
                }
            });

            logger.info('[LoadingStateManager] Wails listeners registered');
        }).catch(err => {
            logger.error(`[LoadingStateManager] Failed to setup Wails listeners: ${err}`);
        });
    }

    /**
     * 处理前端 CustomEvent
     */
    private handleLoadingEvent = (event: Event): void => {
        const customEvent = event as CustomEvent;
        const data = customEvent.detail;
        logger.info(`[LoadingStateManager] chat-loading-frontend: ${JSON.stringify(data)}`);
        this.processLoadingData(data);
    };

    /**
     * 处理加载状态数据
     */
    private processLoadingData(data: any): void {
        if (typeof data === 'boolean') {
            // 旧格式：布尔值，无法确定 threadId，忽略
            logger.warn('[LoadingStateManager] Received boolean loading state without threadId, ignoring');
            return;
        }
        
        if (data && typeof data === 'object' && data.threadId) {
            this.setLoading(data.threadId, data.loading);
        }
    }

    /**
     * 设置会话加载状态
     */
    setLoading(threadId: string, loading: boolean): void {
        logger.info(`[LoadingStateManager] setLoading: threadId=${threadId}, loading=${loading}`);
        
        if (loading) {
            // 开始加载
            const existingSession = this.loadingSessions.get(threadId);
            this.loadingSessions.set(threadId, {
                threadId,
                isLoading: true,
                startTime: existingSession?.startTime ?? Date.now(),
                progress: existingSession?.progress,
                error: undefined // 清除之前的错误
            });
            
            // 设置超时自动清理
            this.clearTimeout(threadId);
            const timeoutId = window.setTimeout(() => {
                logger.warn(`[LoadingStateManager] Timeout for threadId=${threadId}, auto-clearing`);
                this.setLoading(threadId, false);
            }, this.TIMEOUT_MS);
            this.timeoutIds.set(threadId, timeoutId);
        } else {
            // 结束加载
            const existingSession = this.loadingSessions.get(threadId);
            if (existingSession) {
                // 保留会话状态但标记为不再加载
                existingSession.isLoading = false;
                // 如果没有错误，可以清理会话
                if (!existingSession.error) {
                    this.loadingSessions.delete(threadId);
                }
            }
            this.clearTimeout(threadId);
        }
        
        this.notifyListeners();
        this.notifySessionListeners(threadId);
    }

    /**
     * 更新会话进度
     * 
     * 如果会话不存在，会自动创建一个新的加载会话
     * 
     * Requirements: 4.3, 5.1
     */
    updateProgress(threadId: string, progress: SessionLoadingState['progress']): void {
        logger.info(`[LoadingStateManager] updateProgress: threadId=${threadId}, progress=${JSON.stringify(progress)}`);
        
        let session = this.loadingSessions.get(threadId);
        
        if (!session) {
            // 如果会话不存在，创建一个新的加载会话
            logger.info(`[LoadingStateManager] Creating new session for progress update: threadId=${threadId}`);
            session = {
                threadId,
                isLoading: true,
                startTime: Date.now()
            };
            this.loadingSessions.set(threadId, session);
            
            // 设置超时自动清理
            this.clearTimeout(threadId);
            const timeoutId = window.setTimeout(() => {
                logger.warn(`[LoadingStateManager] Timeout for threadId=${threadId}, auto-clearing`);
                this.setLoading(threadId, false);
            }, this.TIMEOUT_MS);
            this.timeoutIds.set(threadId, timeoutId);
        }
        
        // 更新进度信息
        session.progress = progress;
        
        // 通知所有订阅者
        this.notifyListeners();
        this.notifySessionListeners(threadId);
    }

    /**
     * 设置会话错误状态
     * 
     * 设置错误后，会话将不再处于加载状态，但会保留错误信息
     * 
     * Requirements: 5.3
     */
    setError(threadId: string, error: SessionLoadingState['error']): void {
        logger.info(`[LoadingStateManager] setError: threadId=${threadId}, error=${JSON.stringify(error)}`);
        
        let session = this.loadingSessions.get(threadId);
        
        if (!session) {
            // 如果会话不存在，创建一个新的会话来存储错误
            logger.info(`[LoadingStateManager] Creating new session for error: threadId=${threadId}`);
            session = {
                threadId,
                isLoading: false,
                startTime: Date.now()
            };
            this.loadingSessions.set(threadId, session);
        }
        
        // 设置错误状态并标记为不再加载
        session.error = error;
        session.isLoading = false;
        
        // 清除超时定时器
        this.clearTimeout(threadId);
        
        // 通知所有订阅者
        this.notifyListeners();
        this.notifySessionListeners(threadId);
    }

    /**
     * 清除会话状态
     * 
     * 完全移除会话的所有状态信息
     * 
     * Requirements: 4.5
     */
    clearSession(threadId: string): void {
        logger.info(`[LoadingStateManager] clearSession: threadId=${threadId}`);
        
        this.loadingSessions.delete(threadId);
        this.clearTimeout(threadId);
        
        // 通知所有订阅者
        this.notifyListeners();
        this.notifySessionListeners(threadId);
        
        // 清理会话特定的监听器
        this.sessionListeners.delete(threadId);
    }

    /**
     * 清除超时定时器
     */
    private clearTimeout(threadId: string): void {
        const timeoutId = this.timeoutIds.get(threadId);
        if (timeoutId) {
            window.clearTimeout(timeoutId);
            this.timeoutIds.delete(threadId);
        }
    }

    /**
     * 获取当前所有加载中的会话ID
     */
    getLoadingThreadIds(): Set<string> {
        const loadingIds = new Set<string>();
        this.loadingSessions.forEach((session, threadId) => {
            if (session.isLoading) {
                loadingIds.add(threadId);
            }
        });
        return loadingIds;
    }

    /**
     * 获取当前加载中的会话数量
     * 
     * Requirements: 3.1, 3.2
     */
    getLoadingCount(): number {
        return this.getLoadingThreadIds().size;
    }

    /**
     * 检查指定会话是否正在加载
     */
    isLoading(threadId: string): boolean {
        const session = this.loadingSessions.get(threadId);
        return session?.isLoading ?? false;
    }

    /**
     * 获取会话的完整状态
     * 
     * Requirements: 4.1
     */
    getSessionState(threadId: string): SessionLoadingState | undefined {
        return this.loadingSessions.get(threadId);
    }

    /**
     * 获取会话的进度信息
     */
    getProgress(threadId: string): SessionLoadingState['progress'] | undefined {
        return this.loadingSessions.get(threadId)?.progress;
    }

    /**
     * 获取会话的错误信息
     */
    getError(threadId: string): SessionLoadingState['error'] | undefined {
        return this.loadingSessions.get(threadId)?.error;
    }

    /**
     * 订阅状态变化
     */
    subscribe(listener: LoadingStateListener): () => void {
        this.listeners.add(listener);
        // 立即通知当前状态
        listener(this.getLoadingThreadIds());
        
        // 返回取消订阅函数
        return () => {
            this.listeners.delete(listener);
        };
    }

    /**
     * 订阅特定会话的状态变化
     * 
     * Requirements: 4.4
     */
    subscribeToSession(threadId: string, listener: SessionStateListener): () => void {
        let sessionListenerSet = this.sessionListeners.get(threadId);
        if (!sessionListenerSet) {
            sessionListenerSet = new Set();
            this.sessionListeners.set(threadId, sessionListenerSet);
        }
        sessionListenerSet.add(listener);
        
        // 立即通知当前状态
        listener(this.getSessionState(threadId));
        
        // 返回取消订阅函数
        return () => {
            const listenerSet = this.sessionListeners.get(threadId);
            if (listenerSet) {
                listenerSet.delete(listener);
                if (listenerSet.size === 0) {
                    this.sessionListeners.delete(threadId);
                }
            }
        };
    }

    /**
     * 通知所有监听器
     */
    private notifyListeners(): void {
        const loadingThreadIds = this.getLoadingThreadIds();
        logger.info(`[LoadingStateManager] Notifying ${this.listeners.size} listeners, loadingThreadIds=${JSON.stringify([...loadingThreadIds])}`);
        this.listeners.forEach(listener => {
            try {
                listener(loadingThreadIds);
            } catch (err) {
                logger.error(`[LoadingStateManager] Listener error: ${err}`);
            }
        });
    }

    /**
     * 通知特定会话的监听器
     */
    private notifySessionListeners(threadId: string): void {
        const sessionListenerSet = this.sessionListeners.get(threadId);
        if (sessionListenerSet && sessionListenerSet.size > 0) {
            const state = this.getSessionState(threadId);
            logger.info(`[LoadingStateManager] Notifying ${sessionListenerSet.size} session listeners for threadId=${threadId}`);
            sessionListenerSet.forEach(listener => {
                try {
                    listener(state);
                } catch (err) {
                    logger.error(`[LoadingStateManager] Session listener error: ${err}`);
                }
            });
        }
    }

    /**
     * 清理所有状态（用于测试或重置）
     */
    reset(): void {
        this.loadingSessions.clear();
        this.timeoutIds.forEach(id => window.clearTimeout(id));
        this.timeoutIds.clear();
        this.sessionListeners.clear();
        this.notifyListeners();
    }
}

// 导出单例实例
export const loadingStateManager = LoadingStateManager.getInstance();

// 导出类型
export type { LoadingStateListener, SessionStateListener };

// 导出 React Hook
export function useLoadingState(): {
    loadingThreadIds: Set<string>;
    loadingCount: number;
    isAnyLoading: boolean;
    isLoading: (threadId: string) => boolean;
    getProgress: (threadId: string) => SessionLoadingState['progress'] | undefined;
    getError: (threadId: string) => SessionLoadingState['error'] | undefined;
    getSessionState: (threadId: string) => SessionLoadingState | undefined;
} {
    // 这个 hook 需要在 React 组件中使用 useState 和 useEffect
    // 由于这是纯 TS 文件，我们只导出管理器方法
    // 实际的 React hook 在 useLoadingState.ts 中实现
    const loadingThreadIds = loadingStateManager.getLoadingThreadIds();
    return {
        loadingThreadIds,
        loadingCount: loadingThreadIds.size,
        isAnyLoading: loadingThreadIds.size > 0,
        isLoading: (threadId: string) => loadingStateManager.isLoading(threadId),
        getProgress: (threadId: string) => loadingStateManager.getProgress(threadId),
        getError: (threadId: string) => loadingStateManager.getError(threadId),
        getSessionState: (threadId: string) => loadingStateManager.getSessionState(threadId)
    };
}

export default loadingStateManager;
