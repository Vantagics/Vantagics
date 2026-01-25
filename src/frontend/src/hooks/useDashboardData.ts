/**
 * useDashboardData Hook
 * 
 * 为DraggableDashboard提供数据访问的Hook
 * 直接使用新的AnalysisResultManager
 * 包含数据源统计信息
 */

import { useMemo, useState, useEffect } from 'react';
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
  
  // 加载状态
  isLoading: boolean;
  error: string | null;
}

/**
 * useDashboardData Hook
 * 
 * 直接使用新的统一数据系统
 * 包含数据源统计信息
 */
export function useDashboardData(): DashboardDataSource {
  const analysisResults = useAnalysisResults();
  const [dataSourceStatistics, setDataSourceStatistics] = useState<agent.DataSourceStatistics | null>(null);
  
  // Load data source statistics on mount
  useEffect(() => {
    const loadStatistics = async () => {
      try {
        const stats = await GetDataSourceStatistics();
        setDataSourceStatistics(stats);
      } catch (err) {
        console.error('Failed to load data source statistics:', err);
        setDataSourceStatistics(null);
      }
    };
    
    loadStatistics();
  }, []);
  
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
    
    // 检查是否有任何分析结果数据
    const hasAnyAnalysisResults = hasECharts || hasImages || hasTables || 
      analysisResults.metrics.length > 0 || analysisResults.insights.length > 0 || 
      analysisResults.files.length > 0;
    
    // Metrics - 只有在没有分析结果时才显示数据源指标
    const analysisMetrics = analysisResults.metrics;
    const dataSourceMetrics: NormalizedMetricData[] = [];
    
    // 只有在没有分析结果时才添加数据源统计信息
    if (!hasAnyAnalysisResults && dataSourceStatistics && dataSourceStatistics.total_count > 0) {
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
    
    // 有分析结果时只显示分析指标，没有时显示数据源指标
    const allMetrics = hasAnyAnalysisResults ? analysisMetrics : [...dataSourceMetrics, ...analysisMetrics];
    const hasMetrics = allMetrics.length > 0;
    
    // Insights - 只有在没有分析结果时才显示数据源洞察
    const analysisInsights = analysisResults.insights;
    const dataSourceInsights: NormalizedInsightData[] = [];
    
    // 只有在没有分析结果时才添加数据源洞察
    if (!hasAnyAnalysisResults && dataSourceStatistics && dataSourceStatistics.data_sources && dataSourceStatistics.data_sources.length > 0) {
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
    } else {
      logger.debug('No data sources available for insights');
    }
    
    // 有分析结果时只显示分析洞察，没有时显示数据源洞察
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
      isLoading: analysisResults.isLoading,
      error: analysisResults.error,
    };
  }, [analysisResults, dataSourceStatistics]);
  
  return result;
}

export default useDashboardData;
