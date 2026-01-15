/* 
 * Frontend Integration for Working Context
 * Add this code to Dashboard.tsx or a new useWorkingContext hook
 */

import { EventsEmit } from '../../wailsjs/runtime/runtime';
import { UpdateWorkingContext } from '../../wailsjs/go/main/App';

// Helper function to calculate statistics from chart data
function calculateStatistics(data: any, chartType: string): any {
    // For table/CSV data (array of objects)
    if (Array.isArray(data)) {
        const rowCount = data.length;
        const aggregates: Record<string, number> = {};

        // Calculate aggregates for numeric columns
        if (data.length > 0) {
            const numericColumns = Object.keys(data[0]).filter(key =>
                typeof data[0][key] === 'number'
            );

            numericColumns.forEach(col => {
                const values = data.map(row => row[col]).filter(v => typeof v === 'number');
                if (values.length > 0) {
                    aggregates[`${col}_avg`] = values.reduce((a, b) => a + b, 0) / values.length;
                    aggregates[`${col}_max`] = Math.max(...values);
                    aggregates[`${col}_min`] = Math.min(...values);
                }
            });
        }

        return {
            row_count: rowCount,
            aggregates: aggregates,
            outliers: [] // TODO: Implement outlier detection
        };
    }

    // For ECharts JSON string
    if (typeof data === 'string' && chartType === 'echarts') {
        try {
            const chartConfig = JSON.parse(data);
            return extractEChartsStatistics(chartConfig);
        } catch (e) {
            console.error('[WorkingContext] Failed to parse echarts data:', e);
            return { row_count: 0, aggregates: {}, outliers: [] };
        }
    }

    return { row_count: 0, aggregates: {}, outliers: [] };
}

// Extract statistics from ECharts configuration
function extractEChartsStatistics(chartConfig: any): any {
    const stats = {
        row_count: 0,
        aggregates: {} as Record<string, number>,
        outliers: [] as any[]
    };

    // Extract data from series
    if (chartConfig.series && Array.isArray(chartConfig.series)) {
        chartConfig.series.forEach((series: any) => {
            if (series.data && Array.isArray(series.data)) {
                stats.row_count += series.data.length;

                // Extract numeric values
                const values = series.data
                    .map((d: any) => typeof d === 'number' ? d : (typeof d === 'object' && d.value ? d.value : null))
                    .filter((v: any) => v !== null && typeof v === 'number');

                if (values.length > 0) {
                    const seriesName = series.name || 'series';
                    stats.aggregates[`${seriesName}_avg`] = values.reduce((a: number, b: number) => a + b, 0) / values.length;
                    stats.aggregates[`${seriesName}_max`] = Math.max(...values);
                    stats.aggregates[`${seriesName}_min`] = Math.min(...values);
                }
            }
        });
    }

    return stats;
}

// Hook to manage working context updates
export function useWorkingContext(sessionId: string | null) {
    const updateContext = async (updates: any) => {
        if (!sessionId) return;

        try {
            await UpdateWorkingContext(sessionId, updates);
            console.log('[WorkingContext] Updated for session:', sessionId);
        } catch (error) {
            console.error('[WorkingContext] Update failed:', error);
        }
    };

    // Update when active chart changes
    const onChartChange = (chart: any) => {
        if (!chart || !sessionId) return;

        const dataSummary = calculateStatistics(chart.data, chart.type);

        updateContext({
            active_chart: {
                type: chart.type,
                data_summary: dataSummary
            },
            operation: {
                action: 'view_chart',
                target: chart.type,
                value: ''
            }
        });
    };

    // Update when filters change
    const onFilterChange = (filters: Record<string, string>) => {
        if (!sessionId) return;

        updateContext({
            active_filters: filters,
            operation: {
                action: 'filter',
                target: Object.keys(filters).join(','),
                value: Object.values(filters).join(',')
            }
        });
    };

    // Update when highlights are  added
    const onHighlight = (type: string, description: string, dataPoints?: string[]) => {
        if (!sessionId) return;

        updateContext({
            highlights: [{
                type: type,
                description: description,
                data_points: dataPoints || []
            }]
        });
    };

    return {
        onChartChange,
        onFilterChange,
        onHighlight
    };
}

/* 
 * Example usage in Dashboard.tsx:
 * 
 * const { onChartChange, onFilterChange, onHighlight } = useWorkingContext(activeSessionId);
 * 
 * useEffect(() => {
 *     if (activeChart) {
 *         onChartChange(activeChart);
 *     }
 * }, [activeChart]);
 * 
 * // When user applies filter:
 * const handleFilterApply = (filters) => {
 *     setCurrentFilters(filters);
 *     onFilterChange(filters);
 * };
 * 
 * // When outliers are detected:
 * const highlightOutliers = (outliers) => {
 *     onHighlight('outlier', 'Sales values above threshold', outliers.map(o => o.label));
 * };
 */
