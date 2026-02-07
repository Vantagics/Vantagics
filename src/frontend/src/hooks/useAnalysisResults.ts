/**
 * useAnalysisResults Hook
 * 
 * React Hook for integrating with AnalysisResultManager
 * Provides reactive state updates and convenient data access methods
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  AnalysisResultItem,
  AnalysisResultType,
  AnalysisResultState,
  AnalysisResultBatch,
  NormalizedTableData,
  NormalizedMetricData,
  NormalizedInsightData,
} from '../types/AnalysisResult';
import { getAnalysisResultManager, AnalysisResultManagerImpl } from '../managers/AnalysisResultManager';
import { EventsOn } from '../../wailsjs/runtime/runtime';
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

  // 订阅manager状态变更
  useEffect(() => {
    const unsubscribe = manager.subscribe((newState) => {
      setState(newState);
    });

    return unsubscribe;
  }, [manager]);

  // 监听后端事件
  useEffect(() => {
    // 监听 analysis-result-update 事件
    const unsubscribeUpdate = EventsOn('analysis-result-update', (payload: AnalysisResultBatch) => {
      logger.debug(`Received analysis-result-update: ${payload.items?.length || 0} items, session=${payload.sessionId}, message=${payload.messageId}`);
      manager.updateResults(payload);
    });

    // 监听 analysis-result-clear 事件
    const unsubscribeClear = EventsOn('analysis-result-clear', (payload: { sessionId: string; messageId?: string }) => {
      logger.debug(`Received analysis-result-clear: session=${payload.sessionId}`);
      manager.clearResults(payload.sessionId, payload.messageId);
    });

    // 监听 analysis-result-loading 事件
    const unsubscribeLoading = EventsOn('analysis-result-loading', (payload: { sessionId: string; loading: boolean; requestId?: string }) => {
      logger.debug(`Received analysis-result-loading: loading=${payload.loading}`);
      manager.setLoading(payload.loading, payload.requestId);
    });

    // 监听 analysis-cancelled 事件，清除仪表盘的加载状态
    const unsubscribeCancelled = EventsOn('analysis-cancelled', (payload: { threadId: string; message?: string }) => {
      logger.debug(`Received analysis-cancelled: threadId=${payload.threadId}`);
      manager.setLoading(false);
    });

    return () => {
      unsubscribeUpdate();
      unsubscribeClear();
      unsubscribeLoading();
      unsubscribeCancelled();
    };
  }, [manager]);

  // 获取当前结果
  const results = useMemo(() => {
    return manager.getCurrentResults();
  }, [state, manager]);

  // 按类型获取结果
  const getResultsByType = useCallback((type: AnalysisResultType) => {
    return manager.getCurrentResultsByType(type);
  }, [state, manager]);

  // 检查是否有数据
  const hasData = useCallback((type?: AnalysisResultType) => {
    return manager.hasCurrentData(type);
  }, [state, manager]);

  // 便捷数据访问
  const charts = useMemo(() => {
    const result = getResultsByType('echarts');
    logger.warn(`[useAnalysisResults] charts count: ${result.length}`);
    if (result.length > 0) {
      result.forEach((item, i) => {
        logger.warn(`[useAnalysisResults] chart[${i}] data type: ${typeof item.data}, keys: ${typeof item.data === 'object' ? Object.keys(item.data).join(',') : 'N/A'}`);
      });
    }
    return result;
  }, [getResultsByType]);
  const images = useMemo(() => getResultsByType('image'), [getResultsByType]);
  const tables = useMemo(() => {
    const tableResults = getResultsByType('table');
    const csvResults = getResultsByType('csv');
    return [...tableResults, ...csvResults];
  }, [getResultsByType]);
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
