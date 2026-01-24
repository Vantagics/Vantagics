/**
 * useDashboardData Hook
 * 
 * 为DraggableDashboard提供数据访问的Hook
 * 直接使用新的AnalysisResultManager
 */

import { useMemo } from 'react';
import { useAnalysisResults } from './useAnalysisResults';
import {
  AnalysisResultItem,
  NormalizedTableData,
  NormalizedMetricData,
  NormalizedInsightData,
} from '../types/AnalysisResult';

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
  
  // 加载状态
  isLoading: boolean;
  error: string | null;
}

/**
 * useDashboardData Hook
 * 
 * 直接使用新的统一数据系统
 */
export function useDashboardData(): DashboardDataSource {
  const analysisResults = useAnalysisResults();
  
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
    
    // Metrics
    const hasMetrics = analysisResults.metrics.length > 0;
    const metrics = analysisResults.metrics;
    
    // Insights
    const hasInsights = analysisResults.insights.length > 0;
    const insights = analysisResults.insights;
    
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
      metrics,
      hasInsights,
      insights,
      hasFiles,
      files,
      isLoading: analysisResults.isLoading,
      error: analysisResults.error,
    };
  }, [analysisResults]);
  
  return result;
}

export default useDashboardData;
