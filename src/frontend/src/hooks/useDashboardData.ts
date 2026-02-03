/**
 * useDashboardData Hook
 * 
 * 为DraggableDashboard提供数据访问的Hook
 * 直接使用新的AnalysisResultManager
 * 包含数据源统计信息
 * 
 * 响应式管理 dataSourceStatistics:
 * - 订阅 AnalysisResultManager 状态变更
 * - 在分析开始时清除 dataSourceStatistics
 * - 在会话切换时重置 dataSourceStatistics
 * - 历史请求无结果时显示空状态而非数据源统计 (Requirement 2.4)
 * 
 * 状态变更监听机制 (Task 6.1, Requirements 5.3, 5.5):
 * - 通过 useAnalysisResults hook 获取 analysisResults
 * - useAnalysisResults 内部订阅 AnalysisResultManager 的状态变更
 * - 当 AnalysisResultManager 状态变更时（如选择新消息、更新数据等）
 * - analysisResults 会自动更新
 * - hasAnyAnalysisResults 通过 useMemo 依赖 analysisResults 的所有数据数组
 * - 因此状态变更会自动触发 hasAnyAnalysisResults 的重新评估
 * 
 * 加载状态管理 (Task 8.1, Requirements 4.1, 4.3):
 * - 加载状态与 AnalysisResultManager 的分析状态同步
 * - 订阅 analysis-started 事件，分析开始时设置加载状态
 * - 订阅 data-restored 事件，数据恢复完成时清除加载状态
 * - 监听 analysisResults.isLoading 变化，记录状态变更日志
 * - 分析完成或出错时自动清除加载状态
 * 
 * 空状态显示逻辑 (Task 8.3, Requirement 5.4):
 * - 当历史请求无结果时，显示空状态而非数据源统计
 * - shouldShowEmptyState: 指示是否应该显示空状态
 * - isViewingHistoricalEmptyResult: 指示是否正在查看历史空结果
 * - 空状态决策有详细的调试日志
 */

import { useMemo, useState, useEffect, useCallback, useRef } from 'react';
import { useAnalysisResults } from './useAnalysisResults';
import {
  AnalysisResultItem,
  NormalizedTableData,
  NormalizedMetricData,
  NormalizedInsightData,
} from '../types/AnalysisResult';
import { GetDataSourceStatistics } from '../../wailsjs/go/main/App';
import { agent } from '../../wailsjs/go/models';
import { createLogger } from '../utils/systemLog';
import { getAnalysisResultManager } from '../managers/AnalysisResultManager';

const logger = createLogger('useDashboardData');

export interface DashboardDataSource {
  // 图表数据
  hasECharts: boolean;
  echartsData: any | null;
  allEChartsData: any[];
  
  // 图片数据
  hasImages: boolean;
  images: string[];
  
  // 表格数据
  hasTables: boolean;
  tableData: NormalizedTableData | null;
  allTableData: NormalizedTableData[];
  
  // 指标数据
  hasMetrics: boolean;
  metrics: NormalizedMetricData[];
  
  // 洞察数据
  hasInsights: boolean;
  insights: NormalizedInsightData[];
  
  // 文件数据
  hasFiles: boolean;
  files: AnalysisResultItem[];
  
  // 数据源统计信息
  dataSourceStatistics: agent.DataSourceStatistics | null;
  
  // 数据源统计加载状态
  isDataSourceStatsLoading: boolean;
  
  // 加载状态
  isLoading: boolean;
  error: string | null;
  
  // 新增：强制刷新数据源统计的方法
  refreshDataSourceStats: () => void;
  
  // 新增：清除所有数据的方法
  clearAllData: () => void;
  
  // 新增：是否有真正的分析结果（不包括数据源统计）
  // 用于导出按钮的显示判断
  hasRealAnalysisResults: boolean;
  
  // 新增：是否应该显示空状态 (Task 8.3, Requirement 5.4)
  // 当历史请求无结果时为 true，此时不显示数据源统计
  shouldShowEmptyState: boolean;
  
  // 新增：是否正在查看历史空结果 (Task 8.3, Requirement 5.4)
  // 用于组件判断是否显示空状态提示
  isViewingHistoricalEmptyResult: boolean;
}

/**
 * useDashboardData Hook
 * 
 * 直接使用新的统一数据系统
 * 包含数据源统计信息
 * 
 * 响应式管理:
 * - 订阅 AnalysisResultManager 的 analysis-started 和 session-switched 事件
 * - 在分析开始时清除 dataSourceStatistics
 * - 在会话切换时重置 dataSourceStatistics
 * - 历史请求无结果时显示空状态而非数据源统计 (Requirement 2.4)
 * 
 * 加载状态管理 (Task 8.1, Requirements 4.1, 4.3):
 * - 加载状态与 AnalysisResultManager 的分析状态同步
 * - 分析开始时显示加载状态 (Requirement 4.1)
 * - 分析完成时加载状态消失 (Requirement 4.3)
 * - 记录所有状态变更日志便于调试
 */
export function useDashboardData(): DashboardDataSource {
  const analysisResults = useAnalysisResults();
  const [dataSourceStatistics, setDataSourceStatistics] = useState<agent.DataSourceStatistics | null>(null);
  const [isDataSourceStatsLoading, setIsDataSourceStatsLoading] = useState<boolean>(false);
  
  // 跟踪是否正在查看历史请求的空结果 (Requirement 2.4)
  // 当此标志为 true 时，即使没有分析结果，也不应显示数据源统计
  const [isViewingHistoricalEmptyResult, setIsViewingHistoricalEmptyResult] = useState<boolean>(false);
  
  // 使用 ref 跟踪上一次的加载状态，用于状态变更日志 (Task 8.1)
  const prevLoadingRef = useRef<boolean>(analysisResults.isLoading);
  const prevErrorRef = useRef<string | null>(analysisResults.error);
  
  // Hook 生命周期日志 (Task 9.2)
  useEffect(() => {
    logger.info('[Lifecycle] useDashboardData hook mounted');
    return () => {
      logger.info('[Lifecycle] useDashboardData hook unmounted');
    };
  }, []);
  
  /**
   * 加载状态变更日志 (Task 8.1, Requirements 4.1, 4.3)
   * 
   * 监听 analysisResults.isLoading 变化，记录状态变更
   * - 当 isLoading 从 false 变为 true 时，记录分析开始
   * - 当 isLoading 从 true 变为 false 时，记录分析完成或取消
   */
  useEffect(() => {
    const prevLoading = prevLoadingRef.current;
    const currentLoading = analysisResults.isLoading;
    
    if (prevLoading !== currentLoading) {
      if (currentLoading) {
        logger.info('[LoadingState] Analysis started - loading state set to true (Requirement 4.1)');
      } else {
        logger.info('[LoadingState] Analysis completed/cancelled - loading state set to false (Requirement 4.3)');
      }
      prevLoadingRef.current = currentLoading;
    }
  }, [analysisResults.isLoading]);
  
  /**
   * 错误状态变更日志 (Task 8.1)
   * 
   * 监听 analysisResults.error 变化，记录错误状态变更
   */
  useEffect(() => {
    const prevError = prevErrorRef.current;
    const currentError = analysisResults.error;
    
    if (prevError !== currentError) {
      if (currentError) {
        logger.warn(`[LoadingState] Error occurred - loading state cleared: ${currentError}`);
      } else if (prevError) {
        logger.info('[LoadingState] Error cleared');
      }
      prevErrorRef.current = currentError;
    }
  }, [analysisResults.error]);
  
  // 加载数据源统计的函数
  const loadDataSourceStatistics = useCallback(async () => {
    logger.debug('[DataSourceStats] Starting to load data source statistics');
    setIsDataSourceStatsLoading(true);
    try {
      const stats = await GetDataSourceStatistics();
      setDataSourceStatistics(stats);
      logger.info(`[DataSourceStats] Loaded successfully: total=${stats?.total_count || 0}, types=${Object.keys(stats?.breakdown_by_type || {}).length}`);
    } catch (err) {
      logger.error(`[DataSourceStats] Failed to load: ${err}`);
      setDataSourceStatistics(null);
    } finally {
      setIsDataSourceStatsLoading(false);
    }
  }, []);
  
  // 清除数据源统计的函数
  const clearDataSourceStatistics = useCallback(() => {
    logger.debug('[DataSourceStats] Clearing data source statistics');
    setDataSourceStatistics(null);
  }, []);
  
  // 强制刷新数据源统计
  const refreshDataSourceStats = useCallback(() => {
    logger.info('[DataSourceStats] Force refreshing data source statistics');
    loadDataSourceStatistics();
  }, [loadDataSourceStatistics]);
  
  // 清除所有数据
  const clearAllData = useCallback(() => {
    logger.info('[ClearAll] Clearing all dashboard data');
    clearDataSourceStatistics();
    const manager = getAnalysisResultManager();
    manager.clearAll();
    logger.debug('[ClearAll] All dashboard data cleared');
  }, [clearDataSourceStatistics]);
  
  /**
   * 计算是否有任何分析结果
   * 
   * 这个值用于决定是否应该加载/显示数据源统计
   * 
   * 状态变更监听机制 (Task 6.1):
   * - 此 useMemo 依赖于 analysisResults 的所有数据数组
   * - 当 AnalysisResultManager 状态变更时（如选择新消息、更新数据等）
   * - useAnalysisResults hook 会通过 subscribe 机制接收到状态变更
   * - 状态变更会触发 analysisResults 的更新
   * - 由于 useMemo 的依赖数组包含所有 analysisResults 数据
   * - hasAnyAnalysisResults 会自动重新评估
   * 
   * Validates: Requirements 3.1, 3.2, 3.3, 5.3, 5.5
   */
  const hasAnyAnalysisResults = useMemo(() => {
    const safeArrayLength = (arr: any[] | null | undefined): number => {
      if (arr === null || arr === undefined) return 0;
      if (!Array.isArray(arr)) return 0;
      return arr.length;
    };
    
    const chartsCount = safeArrayLength(analysisResults.charts);
    const imagesCount = safeArrayLength(analysisResults.images);
    const tablesCount = safeArrayLength(analysisResults.tables);
    const metricsCount = safeArrayLength(analysisResults.metrics);
    const insightsCount = safeArrayLength(analysisResults.insights);
    const filesCount = safeArrayLength(analysisResults.files);
    
    const hasResults = (
      chartsCount > 0 ||
      imagesCount > 0 ||
      tablesCount > 0 ||
      metricsCount > 0 ||
      insightsCount > 0 ||
      filesCount > 0
    );
    
    // 记录状态变更评估结果，便于调试
    logger.debug(`hasAnyAnalysisResults re-evaluated: ${hasResults} (charts=${chartsCount}, images=${imagesCount}, tables=${tablesCount}, metrics=${metricsCount}, insights=${insightsCount}, files=${filesCount})`);
    
    return hasResults;
  }, [analysisResults.charts, analysisResults.images, analysisResults.tables, 
      analysisResults.metrics, analysisResults.insights, analysisResults.files]);
  
  /**
   * 条件加载数据源统计逻辑
   * 
   * 只在无分析结果时加载数据源统计 (Requirement 3.1)
   * 在分析结果清除后重新加载 (Requirement 3.2)
   * 切换到新会话且无分析结果时显示 (Requirement 3.3)
   * 历史请求无结果时显示空状态而非数据源统计 (Requirement 2.4)
   * 
   * Validates: Requirements 2.4, 3.1, 3.2, 3.3
   */
  useEffect(() => {
    // 如果正在查看历史请求的空结果，不加载数据源统计 (Requirement 2.4)
    if (isViewingHistoricalEmptyResult) {
      logger.debug('[ConditionalLoad] Viewing historical empty result, skipping data source statistics load');
      return;
    }
    
    // 只在没有分析结果且数据源统计为空时加载
    if (!hasAnyAnalysisResults && dataSourceStatistics === null && !isDataSourceStatsLoading) {
      logger.info('[ConditionalLoad] No analysis results and no data source statistics, triggering load');
      loadDataSourceStatistics();
    } else {
      logger.debug(`[ConditionalLoad] Skip load: hasResults=${hasAnyAnalysisResults}, hasStats=${dataSourceStatistics !== null}, isLoading=${isDataSourceStatsLoading}`);
    }
  }, [hasAnyAnalysisResults, dataSourceStatistics, isDataSourceStatsLoading, loadDataSourceStatistics, isViewingHistoricalEmptyResult]);
  
  // 订阅 AnalysisResultManager 事件
  useEffect(() => {
    const manager = getAnalysisResultManager();
    
    // 订阅 analysis-started 事件 - 分析开始时清除 dataSourceStatistics
    // Task 8.1: 确保加载状态与分析状态同步 (Requirement 4.1)
    const unsubscribeAnalysisStarted = manager.on('analysis-started', (event) => {
      logger.info(`[StateChange] Analysis started event received: session=${event.sessionId}, message=${event.messageId}, request=${event.requestId}`);
      logger.debug('[StateChange] Clearing data source statistics for new analysis');
      // 清除数据源统计，确保新分析结果不会与旧数据混合
      clearDataSourceStatistics();
      // 重置历史空结果标志，因为新分析开始了
      setIsViewingHistoricalEmptyResult(false);
    });
    
    // 订阅 session-switched 事件 - 会话切换时重置 dataSourceStatistics
    const unsubscribeSessionSwitched = manager.on('session-switched', (event) => {
      logger.info(`[StateChange] Session switched event received: from=${event.fromSessionId}, to=${event.toSessionId}`);
      logger.debug('[StateChange] Clearing data source statistics for session switch');
      // 会话切换时清除数据源统计
      // 条件加载逻辑会在下一次渲染时根据 hasAnyAnalysisResults 决定是否重新加载
      clearDataSourceStatistics();
      // 重置历史空结果标志，因为切换了会话
      setIsViewingHistoricalEmptyResult(false);
    });
    
    // 订阅 message-selected 事件 - 消息切换时清除 dataSourceStatistics
    // 这对于历史数据恢复特别重要 (Requirements 2.1, 2.2)
    // 当用户点击历史分析请求时，会触发 message-selected 事件
    // 此时需要清除 dataSourceStatistics，确保只显示恢复的数据
    const unsubscribeMessageSelected = manager.on('message-selected', (event) => {
      logger.info(`[StateChange] Message selected event received: session=${event.sessionId}, from=${event.fromMessageId}, to=${event.toMessageId}`);
      logger.debug('[StateChange] Clearing data source statistics for message selection');
      // 消息切换时清除数据源统计，确保历史数据恢复时不会混合显示
      clearDataSourceStatistics();
      // 注意：不在这里重置 isViewingHistoricalEmptyResult，因为 historical-empty-result 事件会在之后触发
    });
    
    // 订阅 historical-empty-result 事件 - 历史请求无结果时显示空状态 (Requirement 2.4)
    // 当历史分析请求没有关联的分析结果时，设置标志以阻止加载数据源统计
    const unsubscribeHistoricalEmptyResult = manager.on('historical-empty-result', (event) => {
      logger.info(`[StateChange] Historical empty result event received: session=${event.sessionId}, message=${event.messageId}`);
      logger.debug('[StateChange] Setting historical empty result flag, blocking data source statistics loading');
      // 设置标志，阻止加载数据源统计
      setIsViewingHistoricalEmptyResult(true);
      // 确保数据源统计被清除
      clearDataSourceStatistics();
    });
    
    // 订阅 data-restored 事件 - 数据恢复完成时记录日志 (Task 8.1)
    // 确保加载状态在数据恢复后正确清除 (Requirement 4.3)
    const unsubscribeDataRestored = manager.on('data-restored', (event) => {
      logger.info(`[StateChange] Data restored event received: session=${event.sessionId}, message=${event.messageId}, items=${event.validCount}/${event.itemCount}`);
      logger.debug(`[StateChange] Restored items by type: ${JSON.stringify(event.itemsByType)}`);
      // 数据恢复完成后，加载状态应该已经被 AnalysisResultManager 清除
      // 这里只记录日志，不需要额外操作
    });
    
    // 清理函数
    return () => {
      unsubscribeAnalysisStarted();
      unsubscribeSessionSwitched();
      unsubscribeMessageSelected();
      unsubscribeHistoricalEmptyResult();
      unsubscribeDataRestored();
    };
  }, [clearDataSourceStatistics]);
  
  const result = useMemo(() => {
    logger.debug('[DataTransform] Starting data transformation');
    
    // ECharts
    const echartsItems = analysisResults.charts;
    const hasECharts = echartsItems.length > 0;
    const echartsData = hasECharts ? echartsItems[0].data : null;
    const allEChartsData = echartsItems.map(item => item.data);
    logger.warn(`[DataTransform] ECharts: count=${echartsItems.length}, hasData=${hasECharts}`);
    if (hasECharts && echartsData) {
      logger.warn(`[DataTransform] ECharts data type: ${typeof echartsData}, preview: ${JSON.stringify(echartsData).substring(0, 100)}...`);
    }
    
    // Images
    const imageItems = analysisResults.images;
    const hasImages = imageItems.length > 0;
    const images = imageItems.map(item => item.data as string);
    logger.debug(`[DataTransform] Images: count=${imageItems.length}, hasData=${hasImages}`);
    
    // Tables
    const tableItems = analysisResults.tables;
    const hasTables = tableItems.length > 0;
    const tableData = hasTables ? (tableItems[0].data as NormalizedTableData) : null;
    const allTableData = tableItems.map(item => item.data as NormalizedTableData);
    logger.debug(`[DataTransform] Tables: count=${tableItems.length}, hasData=${hasTables}`);
    
    // 使用已计算的 hasAnyAnalysisResults（在 useMemo 外部计算以支持条件加载逻辑）
    
    // 判断是否应该显示数据源统计 (Requirement 2.4)
    // 当正在查看历史请求的空结果时，不显示数据源统计
    const shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
    logger.debug(`[DataTransform] shouldShowDataSourceStats=${shouldShowDataSourceStats} (hasResults=${hasAnyAnalysisResults}, historicalEmpty=${isViewingHistoricalEmptyResult})`);
    
    // 新增：是否有真正的分析结果（不包括数据源统计）
    // 用于导出按钮的显示判断
    const hasRealAnalysisResults = hasAnyAnalysisResults;
    
    /**
     * 空状态显示逻辑 (Task 8.3, Requirement 5.4)
     * 
     * shouldShowEmptyState 为 true 的条件：
     * 1. 正在查看历史请求的空结果 (isViewingHistoricalEmptyResult = true)
     * 2. 没有任何分析结果 (hasAnyAnalysisResults = false)
     * 3. 不在加载状态 (isLoading = false)
     * 
     * 当 shouldShowEmptyState 为 true 时：
     * - 不显示数据源统计
     * - 显示空状态提示（如"该历史请求没有分析结果"）
     */
    const shouldShowEmptyState = isViewingHistoricalEmptyResult && !hasAnyAnalysisResults && !analysisResults.isLoading;
    
    // 空状态决策日志 (Task 8.3)
    logger.debug(`[EmptyState] Decision: shouldShowEmptyState=${shouldShowEmptyState}`);
    logger.debug(`[EmptyState] Factors: isViewingHistoricalEmptyResult=${isViewingHistoricalEmptyResult}, hasAnyAnalysisResults=${hasAnyAnalysisResults}, isLoading=${analysisResults.isLoading}`);
    if (shouldShowEmptyState) {
      logger.info('[EmptyState] Showing empty state for historical request with no results (Requirement 5.4)');
    } else if (isViewingHistoricalEmptyResult && hasAnyAnalysisResults) {
      logger.debug('[EmptyState] Historical empty result flag set but has results - showing results');
    } else if (isViewingHistoricalEmptyResult && analysisResults.isLoading) {
      logger.debug('[EmptyState] Historical empty result flag set but still loading - waiting');
    }
    
    // Metrics - 只有在没有分析结果且不是历史空结果时才显示数据源指标
    const analysisMetrics = analysisResults.metrics;
    const dataSourceMetrics: NormalizedMetricData[] = [];
    
    // 只有在应该显示数据源统计时才添加数据源统计信息
    if (shouldShowDataSourceStats && dataSourceStatistics && dataSourceStatistics.total_count > 0) {
      logger.debug(`[DataTransform] Adding data source metrics: total=${dataSourceStatistics.total_count}`);
      // Total data sources metric
      dataSourceMetrics.push({
        title: '数据源总数',
        value: String(dataSourceStatistics.total_count),
        change: ''
      });
      
      // Breakdown by type - show top 3 types
      const sortedTypes = Object.entries(dataSourceStatistics.breakdown_by_type)
        .sort(([, a], [, b]) => b - a)
        .slice(0, 3);
      
      sortedTypes.forEach(([type, count]) => {
        dataSourceMetrics.push({
          title: `${type.toUpperCase()} 数据源`,
          value: String(count),
          change: ''
        });
      });
    } else if (shouldShowEmptyState) {
      logger.debug('[DataTransform] Empty state active - not adding data source metrics (Requirement 5.4)');
    }
    
    // 有分析结果时只显示分析指标，没有时显示数据源指标（除非是历史空结果）
    const allMetrics = hasAnyAnalysisResults ? analysisMetrics : [...dataSourceMetrics, ...analysisMetrics];
    const hasMetrics = allMetrics.length > 0;
    logger.debug(`[DataTransform] Metrics: analysisCount=${analysisMetrics.length}, dataSourceCount=${dataSourceMetrics.length}, total=${allMetrics.length}`);
    
    // Insights - 只有在没有分析结果且不是历史空结果时才显示数据源洞察
    const analysisInsights = analysisResults.insights;
    const dataSourceInsights: NormalizedInsightData[] = [];
    
    // 只有在应该显示数据源统计时才添加数据源洞察
    if (shouldShowDataSourceStats && dataSourceStatistics && dataSourceStatistics.data_sources && dataSourceStatistics.data_sources.length > 0) {
      logger.debug(`[DataTransform] Generating insights for ${dataSourceStatistics.data_sources.length} data sources`);
      dataSourceStatistics.data_sources.forEach((ds: any) => {
        const insight = {
          text: `${ds.name} (${ds.type.toUpperCase()}) - 点击启动智能分析`,
          icon: 'database',
          dataSourceId: ds.id,
          sourceName: ds.name
        };
        logger.debug(`[DataTransform] Created data source insight: id=${ds.id}, name=${ds.name}, type=${ds.type}`);
        dataSourceInsights.push(insight);
      });
    } else if (hasAnyAnalysisResults) {
      logger.debug('[DataTransform] Has analysis results, hiding data source insights');
    } else if (shouldShowEmptyState) {
      logger.debug('[DataTransform] Empty state active - not adding data source insights (Requirement 5.4)');
    } else if (isViewingHistoricalEmptyResult) {
      logger.debug('[DataTransform] Viewing historical empty result, showing empty state instead of data source insights');
    } else {
      logger.debug('[DataTransform] No data sources available for insights');
    }
    
    // 有分析结果时只显示分析洞察，没有时显示数据源洞察（除非是历史空结果）
    // 限制最多显示9个洞察项
    const combinedInsights = hasAnyAnalysisResults ? analysisInsights : [...dataSourceInsights, ...analysisInsights];
    const allInsights = combinedInsights.slice(0, 9);
    const hasInsights = allInsights.length > 0;
    logger.debug(`[DataTransform] Insights: analysisCount=${analysisInsights.length}, dataSourceCount=${dataSourceInsights.length}, total=${allInsights.length}`);
    
    // Files
    const hasFiles = analysisResults.files.length > 0;
    const files = analysisResults.files;
    logger.debug(`[DataTransform] Files: count=${files.length}, hasData=${hasFiles}`);
    
    // 记录最终的加载状态 (Task 8.1)
    logger.debug(`[DataTransform] Final state: isLoading=${analysisResults.isLoading}, error=${analysisResults.error || 'none'}, hasRealResults=${hasRealAnalysisResults}, shouldShowEmptyState=${shouldShowEmptyState}`);
    
    logger.debug('[DataTransform] Data transformation completed');
    
    return {
      hasECharts,
      echartsData,
      allEChartsData,
      hasImages,
      images,
      hasTables,
      tableData,
      allTableData,
      hasMetrics,
      metrics: allMetrics,
      hasInsights,
      insights: allInsights,
      hasFiles,
      files,
      dataSourceStatistics,
      isDataSourceStatsLoading,
      isLoading: analysisResults.isLoading,
      error: analysisResults.error,
      refreshDataSourceStats,
      clearAllData,
      // 新增：是否有真正的分析结果（不包括数据源统计）
      hasRealAnalysisResults,
      // 新增：是否应该显示空状态 (Task 8.3, Requirement 5.4)
      shouldShowEmptyState,
      // 新增：是否正在查看历史空结果 (Task 8.3, Requirement 5.4)
      isViewingHistoricalEmptyResult,
    };
  }, [analysisResults, dataSourceStatistics, isDataSourceStatsLoading, hasAnyAnalysisResults, isViewingHistoricalEmptyResult, refreshDataSourceStats, clearAllData]);
  
  return result;
}

export default useDashboardData;
