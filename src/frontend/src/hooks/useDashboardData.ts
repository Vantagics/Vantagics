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
 */

import { useMemo, useState, useEffect, useCallback } from 'react';
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
 */
export function useDashboardData(): DashboardDataSource {
  const analysisResults = useAnalysisResults();
  const [dataSourceStatistics, setDataSourceStatistics] = useState<agent.DataSourceStatistics | null>(null);
  const [isDataSourceStatsLoading, setIsDataSourceStatsLoading] = useState<boolean>(false);
  
  // 跟踪是否正在查看历史请求的空结果 (Requirement 2.4)
  // 当此标志为 true 时，即使没有分析结果，也不应显示数据源统计
  const [isViewingHistoricalEmptyResult, setIsViewingHistoricalEmptyResult] = useState<boolean>(false);
  
  // 加载数据源统计的函数
  const loadDataSourceStatistics = useCallback(async () => {
    setIsDataSourceStatsLoading(true);
    try {
      const stats = await GetDataSourceStatistics();
      setDataSourceStatistics(stats);
      logger.debug('Data source statistics loaded successfully');
    } catch (err) {
      logger.error(`Failed to load data source statistics: ${err}`);
      setDataSourceStatistics(null);
    } finally {
      setIsDataSourceStatsLoading(false);
    }
  }, []);
  
  // 清除数据源统计的函数
  const clearDataSourceStatistics = useCallback(() => {
    logger.debug('Clearing data source statistics');
    setDataSourceStatistics(null);
  }, []);
  
  // 强制刷新数据源统计
  const refreshDataSourceStats = useCallback(() => {
    logger.debug('Refreshing data source statistics');
    loadDataSourceStatistics();
  }, [loadDataSourceStatistics]);
  
  // 清除所有数据
  const clearAllData = useCallback(() => {
    logger.debug('Clearing all dashboard data');
    clearDataSourceStatistics();
    const manager = getAnalysisResultManager();
    manager.clearAll();
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
      logger.debug('Viewing historical empty result, not loading data source statistics');
      return;
    }
    
    // 只在没有分析结果且数据源统计为空时加载
    if (!hasAnyAnalysisResults && dataSourceStatistics === null && !isDataSourceStatsLoading) {
      logger.info('No analysis results and no data source statistics, loading data source statistics');
      loadDataSourceStatistics();
    }
  }, [hasAnyAnalysisResults, dataSourceStatistics, isDataSourceStatsLoading, loadDataSourceStatistics, isViewingHistoricalEmptyResult]);
  
  // 订阅 AnalysisResultManager 事件
  useEffect(() => {
    const manager = getAnalysisResultManager();
    
    // 订阅 analysis-started 事件 - 分析开始时清除 dataSourceStatistics
    const unsubscribeAnalysisStarted = manager.on('analysis-started', (event) => {
      logger.info(`Analysis started event received: session=${event.sessionId}, message=${event.messageId}, request=${event.requestId}`);
      // 清除数据源统计，确保新分析结果不会与旧数据混合
      clearDataSourceStatistics();
      // 重置历史空结果标志，因为新分析开始了
      setIsViewingHistoricalEmptyResult(false);
    });
    
    // 订阅 session-switched 事件 - 会话切换时重置 dataSourceStatistics
    const unsubscribeSessionSwitched = manager.on('session-switched', (event) => {
      logger.info(`Session switched event received: from=${event.fromSessionId}, to=${event.toSessionId}`);
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
      logger.info(`Message selected event received: session=${event.sessionId}, from=${event.fromMessageId}, to=${event.toMessageId}`);
      // 消息切换时清除数据源统计，确保历史数据恢复时不会混合显示
      clearDataSourceStatistics();
      // 注意：不在这里重置 isViewingHistoricalEmptyResult，因为 historical-empty-result 事件会在之后触发
    });
    
    // 订阅 historical-empty-result 事件 - 历史请求无结果时显示空状态 (Requirement 2.4)
    // 当历史分析请求没有关联的分析结果时，设置标志以阻止加载数据源统计
    const unsubscribeHistoricalEmptyResult = manager.on('historical-empty-result', (event) => {
      logger.info(`Historical empty result event received: session=${event.sessionId}, message=${event.messageId}`);
      // 设置标志，阻止加载数据源统计
      setIsViewingHistoricalEmptyResult(true);
      // 确保数据源统计被清除
      clearDataSourceStatistics();
    });
    
    // 清理函数
    return () => {
      unsubscribeAnalysisStarted();
      unsubscribeSessionSwitched();
      unsubscribeMessageSelected();
      unsubscribeHistoricalEmptyResult();
    };
  }, [clearDataSourceStatistics]);
  
  const result = useMemo(() => {
    // ECharts
    const echartsItems = analysisResults.charts;
    const hasECharts = echartsItems.length > 0;
    const echartsData = hasECharts ? echartsItems[0].data : null;
    const allEChartsData = echartsItems.map(item => item.data);
    
    // Images
    const imageItems = analysisResults.images;
    const hasImages = imageItems.length > 0;
    const images = imageItems.map(item => item.data as string);
    
    // Tables
    const tableItems = analysisResults.tables;
    const hasTables = tableItems.length > 0;
    const tableData = hasTables ? (tableItems[0].data as NormalizedTableData) : null;
    const allTableData = tableItems.map(item => item.data as NormalizedTableData);
    
    // 使用已计算的 hasAnyAnalysisResults（在 useMemo 外部计算以支持条件加载逻辑）
    
    // 判断是否应该显示数据源统计 (Requirement 2.4)
    // 当正在查看历史请求的空结果时，不显示数据源统计
    const shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
    
    // 新增：是否有真正的分析结果（不包括数据源统计）
    // 用于导出按钮的显示判断
    const hasRealAnalysisResults = hasAnyAnalysisResults;
    
    // Metrics - 只有在没有分析结果且不是历史空结果时才显示数据源指标
    const analysisMetrics = analysisResults.metrics;
    const dataSourceMetrics: NormalizedMetricData[] = [];
    
    // 只有在应该显示数据源统计时才添加数据源统计信息
    if (shouldShowDataSourceStats && dataSourceStatistics && dataSourceStatistics.total_count > 0) {
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
    }
    
    // 有分析结果时只显示分析指标，没有时显示数据源指标（除非是历史空结果）
    const allMetrics = hasAnyAnalysisResults ? analysisMetrics : [...dataSourceMetrics, ...analysisMetrics];
    const hasMetrics = allMetrics.length > 0;
    
    // Insights - 只有在没有分析结果且不是历史空结果时才显示数据源洞察
    const analysisInsights = analysisResults.insights;
    const dataSourceInsights: NormalizedInsightData[] = [];
    
    // 只有在应该显示数据源统计时才添加数据源洞察
    if (shouldShowDataSourceStats && dataSourceStatistics && dataSourceStatistics.data_sources && dataSourceStatistics.data_sources.length > 0) {
      logger.info(`Generating insights for ${dataSourceStatistics.data_sources.length} data sources`);
      dataSourceStatistics.data_sources.forEach((ds: any) => {
        const insight = {
          text: `${ds.name} (${ds.type.toUpperCase()}) - 点击启动智能分析`,
          icon: 'database',
          dataSourceId: ds.id,
          sourceName: ds.name
        };
        logger.debug(`Created data source insight: id=${ds.id}, name=${ds.name}, type=${ds.type}`);
        dataSourceInsights.push(insight);
      });
    } else if (hasAnyAnalysisResults) {
      logger.debug('Has analysis results, hiding data source insights');
    } else if (isViewingHistoricalEmptyResult) {
      logger.debug('Viewing historical empty result, showing empty state instead of data source insights');
    } else {
      logger.debug('No data sources available for insights');
    }
    
    // 有分析结果时只显示分析洞察，没有时显示数据源洞察（除非是历史空结果）
    // 限制最多显示9个洞察项
    const combinedInsights = hasAnyAnalysisResults ? analysisInsights : [...dataSourceInsights, ...analysisInsights];
    const allInsights = combinedInsights.slice(0, 9);
    const hasInsights = allInsights.length > 0;
    
    // Files
    const hasFiles = analysisResults.files.length > 0;
    const files = analysisResults.files;
    
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
    };
  }, [analysisResults, dataSourceStatistics, isDataSourceStatsLoading, hasAnyAnalysisResults, isViewingHistoricalEmptyResult, refreshDataSourceStats, clearAllData]);
  
  return result;
}

export default useDashboardData;
