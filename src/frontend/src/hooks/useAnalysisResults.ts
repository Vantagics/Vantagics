/**
 * useAnalysisResults Hook
 * 
 * React Hook for integrating with AnalysisResultManager
 * Provides reactive state updates and convenient data access methods
 * 
 * NOTE: This hook does NOT register Wails event listeners.
 * All Wails event handling is done by AnalysisResultBridge (initialized in App.tsx).
 * This hook only subscribes to AnalysisResultManager state changes via manager.subscribe().
 * This avoids duplicate event processing that can cause data clearing race conditions.
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  AnalysisResultItem,
  AnalysisResultType,
  AnalysisResultState,
  NormalizedTableData,
  NormalizedMetricData,
  NormalizedInsightData,
} from '../types/AnalysisResult';
import { getAnalysisResultManager } from '../managers/AnalysisResultManager';
import { createLogger } from '../utils/systemLog';

const logger = createLogger('useAnalysisResults');

/**
 * Hook返回类型
 */
export interface UseAnalysisResultsReturn {
  // 状态
  isLoading: boolean;
  error: string | null;
  currentSessionId: string | null;
  currentMessageId: string | null;

  // 数据访问
  results: AnalysisResultItem[];
  getResultsByType: (type: AnalysisResultType) => AnalysisResultItem[];
  hasData: (type?: AnalysisResultType) => boolean;

  // 便捷数据访问
  charts: AnalysisResultItem[];
  images: AnalysisResultItem[];
  tables: AnalysisResultItem[];
  metrics: NormalizedMetricData[];
  insights: NormalizedInsightData[];
  files: AnalysisResultItem[];

  // 操作
  switchSession: (sessionId: string) => void;
  selectMessage: (messageId: string) => void;
  clearResults: (sessionId?: string, messageId?: string) => void;
  setLoading: (loading: boolean, requestId?: string) => void;
}

/**
 * useAnalysisResults Hook
 * 
 * 提供与AnalysisResultManager的响应式集成
 */
export function useAnalysisResults(): UseAnalysisResultsReturn {
  const manager = getAnalysisResultManager();

  // 本地状态（从manager同步）
  const [state, setState] = useState<AnalysisResultState>(() => manager.getState());
  
  // 直接存储计算后的结果数据，避免 useMemo 链式依赖可能导致的更新遗漏
  const [cachedResults, setCachedResults] = useState<AnalysisResultItem[]>(() => manager.getCurrentResults());

  // 订阅manager状态变更
  useEffect(() => {
    const unsubscribe = manager.subscribe((newState) => {
      // 在 subscriber 回调中直接计算结果，确保数据和状态同步更新
      const currentResults = manager.getCurrentResults();
      const resultTypes = currentResults.map(r => r.type).join(',');
      logger.warn(`[subscribe] State update: session=${newState.currentSessionId}, message=${newState.currentMessageId}, isLoading=${newState.isLoading}, results=${currentResults.length} [${resultTypes}]`);
      setState(newState);
      setCachedResults(currentResults);
    });

    return unsubscribe;
  }, [manager]);

  // 使用 cachedResults 而不是通过 useMemo 从 manager 重新获取
  const results = cachedResults;

  // 按类型获取结果 — 直接从 cachedResults 过滤
  const getResultsByType = useCallback((type: AnalysisResultType) => {
    return cachedResults.filter(item => item.type === type);
  }, [cachedResults]);

  // 检查是否有数据
  const hasData = useCallback((type?: AnalysisResultType) => {
    if (type) {
      return cachedResults.some(item => item.type === type);
    }
    return cachedResults.length > 0;
  }, [cachedResults]);

  // 便捷数据访问
  const charts = useMemo(() => getResultsByType('echarts'), [getResultsByType]);
  const images = useMemo(() => getResultsByType('image'), [getResultsByType]);
  const tables = useMemo(() => {
    return cachedResults.filter(item => item.type === 'table' || item.type === 'csv');
  }, [cachedResults]);
  const files = useMemo(() => getResultsByType('file'), [getResultsByType]);

  // 提取规范化的指标数据
  const metrics = useMemo(() => {
    const metricResults = getResultsByType('metric');
    return metricResults.map(item => item.data as NormalizedMetricData);
  }, [getResultsByType]);

  // 提取规范化的洞察数据
  const insights = useMemo(() => {
    const insightResults = getResultsByType('insight');
    return insightResults.map(item => item.data as NormalizedInsightData);
  }, [getResultsByType]);

  // 操作方法
  const switchSession = useCallback((sessionId: string) => {
    manager.switchSession(sessionId);
  }, [manager]);

  const selectMessage = useCallback((messageId: string) => {
    manager.selectMessage(messageId);
  }, [manager]);

  const clearResults = useCallback((sessionId?: string, messageId?: string) => {
    if (sessionId) {
      manager.clearResults(sessionId, messageId);
    } else if (state.currentSessionId) {
      manager.clearResults(state.currentSessionId, messageId);
    }
  }, [manager, state.currentSessionId]);

  const setLoading = useCallback((loading: boolean, requestId?: string) => {
    manager.setLoading(loading, requestId);
  }, [manager]);

  return {
    // 状态
    isLoading: state.isLoading,
    error: state.error,
    currentSessionId: state.currentSessionId,
    currentMessageId: state.currentMessageId,

    // 数据访问
    results,
    getResultsByType,
    hasData,

    // 便捷数据访问
    charts,
    images,
    tables,
    metrics,
    insights,
    files,

    // 操作
    switchSession,
    selectMessage,
    clearResults,
    setLoading,
  };
}

/**
 * 用于特定会话和消息的Hook
 */
export function useAnalysisResultsFor(sessionId: string | null, messageId: string | null): UseAnalysisResultsReturn {
  const baseHook = useAnalysisResults();

  // 当sessionId或messageId变化时，自动切换
  useEffect(() => {
    if (sessionId && sessionId !== baseHook.currentSessionId) {
      baseHook.switchSession(sessionId);
    }
  }, [sessionId, baseHook.currentSessionId, baseHook.switchSession]);

  useEffect(() => {
    if (messageId && messageId !== baseHook.currentMessageId) {
      baseHook.selectMessage(messageId);
    }
  }, [messageId, baseHook.currentMessageId, baseHook.selectMessage]);

  return baseHook;
}

export default useAnalysisResults;
