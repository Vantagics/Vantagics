import React, { useState } from 'react';
import ReactECharts from 'echarts-for-react';
import DashboardLayout from './DashboardLayout';
import MetricCard from './MetricCard';
import SmartInsight from './SmartInsight';
import Chart from './Chart';
import ImageModal from './ImageModal';
import ChartModal from './ChartModal';
import { main } from '../../wailsjs/go/models';
import { useLanguage } from '../i18n';
import { useWorkingContext } from '../hooks/useWorkingContext';
import EChartsFileLoader from './EChartsFileLoader';
import TableFileLoader from './TableFileLoader';
import { EventsEmit } from '../../wailsjs/runtime/runtime';
import { ExportDashboardToPDF } from '../../wailsjs/go/main/App';
import { Download, Table, BarChart3, ChevronLeft, ChevronRight, FileText, FileImage } from 'lucide-react';
import { createLogger } from '../utils/systemLog';
import Toast, { ToastType } from './Toast';

const logger = createLogger('Dashboard');

interface DashboardProps {
    data: main.DashboardData | null;
    activeChart?: { type: 'echarts' | 'image' | 'table' | 'csv', data: any, chartData?: main.ChartData } | null;
    userRequestText?: string | null;
    onDashboardClick?: () => void;
    isChatOpen?: boolean;
    activeThreadId?: string | null;  // Track active thread for insight clicks
    isAnalysisLoading?: boolean;     // Analysis loading state
    loadingThreadId?: string | null; // Which thread is loading
    sessionFiles?: main.SessionFile[]; // Session files for download
    selectedMessageId?: string | null; // Current selected message ID for filtering files
}

const Dashboard: React.FC<DashboardProps> = ({ data, activeChart, userRequestText, onDashboardClick, isChatOpen, activeThreadId, isAnalysisLoading, loadingThreadId, sessionFiles, selectedMessageId }) => {
    const { t } = useLanguage();
    const [imageModalOpen, setImageModalOpen] = useState(false);
    const [chartModalOpen, setChartModalOpen] = useState(false);
    const [currentChartIndex, setCurrentChartIndex] = useState(0);
    const [exportDropdownOpen, setExportDropdownOpen] = useState(false);
    const [toast, setToast] = useState<{ message: string; type: ToastType } | null>(null);

    // Reset chart index when activeChart changes
    React.useEffect(() => {
        setCurrentChartIndex(0);
    }, [activeChart]);

    // 点击外部关闭导出下拉菜单
    React.useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (exportDropdownOpen) {
                const target = event.target as HTMLElement;
                if (!target.closest('.export-dropdown-container')) {
                    setExportDropdownOpen(false);
                }
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
        };
    }, [exportDropdownOpen]);

    // 检查是否有可导出的内容
    const hasExportableContent = () => {
        const hasMetrics = data?.metrics && Array.isArray(data.metrics) && data.metrics.length > 0;
        const hasInsights = data?.insights && Array.isArray(data.insights) && data.insights.length > 0;
        const hasChart = activeChart !== null;
        return hasMetrics || hasInsights || hasChart;
    };

    // 捕获ECharts图表为图片
    const captureEChartsAsImage = async (): Promise<string | null> => {
        try {
            // 方法1: 尝试通过ReactECharts组件实例获取
            const echartsComponent = document.querySelector('.echarts-for-react') as any;
            if (echartsComponent && echartsComponent.getEchartsInstance) {
                const echartsInstance = echartsComponent.getEchartsInstance();
                if (echartsInstance) {
                    console.log("[Dashboard] ECharts captured via getDataURL method");
                    const dataURL = echartsInstance.getDataURL({
                        type: 'png',
                        pixelRatio: 4, // 高分辨率 (提高到4倍以获得更清晰的图片)
                        backgroundColor: '#fff'
                    });
                    return dataURL;
                }
            }

            // 方法2: 尝试通过Canvas元素转换
            const canvasElements = document.querySelectorAll('canvas');
            for (const canvas of canvasElements) {
                const parent = canvas.parentElement;
                if (parent && (parent.classList.contains('echarts-for-react') ||
                    parent.style.height ||
                    canvas.width > 200)) {

                    console.log("[Dashboard] ECharts captured via Canvas toBlob method");
                    return new Promise((resolve) => {
                        canvas.toBlob((blob) => {
                            if (blob) {
                                const reader = new FileReader();
                                reader.onload = () => resolve(reader.result as string);
                                reader.readAsDataURL(blob);
                            } else {
                                resolve(null);
                            }
                        }, 'image/png');
                    });
                }
            }

            // 方法3: 尝试通过全局ECharts实例
            const globalEcharts = (window as any).echarts;
            if (globalEcharts) {
                const echartsContainer = document.querySelector('.echarts-for-react');
                if (echartsContainer) {
                    const instance = globalEcharts.getInstanceByDom(echartsContainer);
                    if (instance) {
                        console.log("[Dashboard] ECharts captured via global echarts instance");
                        const dataURL = instance.getDataURL({
                            type: 'png',
                            pixelRatio: 2,
                            backgroundColor: '#fff'
                        });
                        return dataURL;
                    }
                }
            }

            console.warn("[Dashboard] No ECharts instance found for capture");
            return null;
        } catch (error) {
            console.error("[Dashboard] Failed to capture ECharts as image:", error);
            return null;
        }
    };

    // 导出数据文件（ZIP 格式）
    const exportDataFiles = async () => {
        try {
            if (!activeThreadId) {
                setToast({ message: '无法导出：未选择会话', type: 'error' });
                return;
            }

            if (!selectedMessageId) {
                setToast({ message: '无法导出：未选择分析请求', type: 'error' });
                return;
            }

            setExportDropdownOpen(false);
            
            logger.debug(`Exporting files for thread ${activeThreadId}, message ${selectedMessageId}`);
            
            const { ExportSessionFilesToZip } = await import('../../wailsjs/go/main/App');
            await ExportSessionFilesToZip(activeThreadId, selectedMessageId);
            
            setToast({ message: '数据文件导出成功！', type: 'success' });
        } catch (error) {
            console.error('[Dashboard] Data files export failed:', error);
            setToast({
                message: '数据文件导出失败: ' + (error instanceof Error ? error.message : String(error)),
                type: 'error'
            });
        }
    };

    // 导出为PDF（使用后端chromedp生成）
    const exportAsPDF = async () => {
        try {
            console.log('[Dashboard] Starting PDF export...');

            // 收集仪表盘数据
            const exportData: any = {
                userRequest: userRequestText || '',
                metrics: [],
                insights: [],
                chartImage: ''
            };

            // 收集指标数据
            if (data?.metrics && Array.isArray(data.metrics)) {
                exportData.metrics = data.metrics.map((metric: any) => ({
                    title: metric.title || '',
                    value: metric.value || '',
                    change: metric.change || ''
                }));
            }

            // 收集洞察数据
            if (data?.insights && Array.isArray(data.insights)) {
                exportData.insights = data.insights.map((insight: any) =>
                    insight.text || insight.toString()
                );
            }

            // 收集所有图表图片
            const chartImages: string[] = [];

            // 方法1: 无条件收集页面上所有ECharts组件
            const echartsComponents = document.querySelectorAll('.echarts-for-react');
            console.log('[Dashboard] Found ECharts components on page:', echartsComponents.length);

            for (let i = 0; i < echartsComponents.length; i++) {
                try {
                    const component = echartsComponents[i] as any;
                    console.log(`[Dashboard] Processing EChart component ${i}:`, {
                        hasGetInstance: !!component?.getEchartsInstance,
                        componentType: component?.constructor?.name
                    });
                    
                    if (component?.getEchartsInstance) {
                        const instance = component.getEchartsInstance();
                        if (instance) {
                            const dataURL = instance.getDataURL({
                                type: 'png',
                                pixelRatio: 4,
                                backgroundColor: '#fff'
                            });
                            chartImages.push(dataURL);
                            console.log(`[Dashboard] ✓ Captured EChart ${i + 1}, size: ${dataURL.length} bytes`);
                        } else {
                            console.warn(`[Dashboard] ✗ EChart ${i} instance is null`);
                        }
                    } else {
                        console.warn(`[Dashboard] ✗ EChart ${i} has no getEchartsInstance method`);
                    }
                } catch (e) {
                    console.error(`[Dashboard] Failed to capture EChart ${i}:`, e);
                }
            }

            // 方法2: 尝试通过Canvas元素捕获（备用方案）
            if (chartImages.length === 0) {
                console.log('[Dashboard] No ECharts captured via component method, trying Canvas fallback');
                const canvasElements = document.querySelectorAll('canvas');
                console.log('[Dashboard] Found canvas elements:', canvasElements.length);
                
                for (let i = 0; i < canvasElements.length; i++) {
                    const canvas = canvasElements[i];
                    const parent = canvas.parentElement;
                    
                    // 检查是否是 ECharts 的 canvas
                    if (parent && (parent.classList.contains('echarts-for-react') || 
                                   parent.querySelector('.echarts-for-react') ||
                                   canvas.width > 200)) {
                        try {
                            const dataURL = canvas.toDataURL('image/png');
                            chartImages.push(dataURL);
                            console.log(`[Dashboard] ✓ Captured canvas ${i + 1} as fallback, size: ${dataURL.length} bytes`);
                        } catch (e) {
                            console.error(`[Dashboard] Failed to capture canvas ${i}:`, e);
                        }
                    }
                }
            }

            // 收集chartData.charts中的所有image类型
            if (activeChart?.chartData?.charts) {
                console.log('[Dashboard] chartData.charts:', activeChart.chartData.charts.map((c: any) => ({ type: c.type, hasData: !!c.data })));
                for (const chart of activeChart.chartData.charts) {
                    if (chart.type === 'image' && typeof chart.data === 'string' && chart.data.startsWith('data:image')) {
                        chartImages.push(chart.data);
                        console.log('[Dashboard] ✓ Added image from chartData, size:', chart.data.length);
                    }
                }
            }

            // 也检查activeChart.data（直接图片）
            if (activeChart?.type === 'image' && typeof activeChart.data === 'string' && activeChart.data.startsWith('data:image')) {
                if (!chartImages.includes(activeChart.data)) {
                    chartImages.push(activeChart.data);
                    console.log('[Dashboard] ✓ Added direct image, size:', activeChart.data.length);
                }
            }

            console.log('[Dashboard] ========================================');
            console.log('[Dashboard] Total images collected:', chartImages.length);
            console.log('[Dashboard] Image sizes:', chartImages.map(img => `${(img.length / 1024).toFixed(1)}KB`));
            console.log('[Dashboard] ========================================');

            // 添加图表图片到导出数据
            if (chartImages.length > 0) {
                exportData.chartImages = chartImages;
            }

            // 收集表格数据（从chartData.charts中提取type=table的数据）
            if (activeChart?.chartData?.charts) {
                console.log('[Dashboard] Checking chartData.charts for table data');

                // 找到所有table类型的图表
                const tableCharts = activeChart.chartData.charts.filter(
                    (chart: any) => chart.type === 'table'
                );

                console.log('[Dashboard] Found table charts:', tableCharts.length);

                if (tableCharts.length > 0) {
                    // 使用第一个table的数据
                    const firstTable = tableCharts[0];
                    try {
                        let tableDataRaw = firstTable.data;

                        // 如果是字符串，需要解析JSON
                        if (typeof tableDataRaw === 'string') {
                            // 清理可能的函数定义
                            tableDataRaw = tableDataRaw
                                .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                .replace(/,(\s*[}\]])/g, '$1');
                            tableDataRaw = JSON.parse(tableDataRaw);
                        }

                        if (Array.isArray(tableDataRaw) && tableDataRaw.length > 0) {
                            // 从第一行推断列
                            const columns = Object.keys(tableDataRaw[0]).map(key => ({
                                title: key,
                                dataType: 'string'
                            }));

                            // 转换为二维数组
                            const rows = tableDataRaw.map((row: any) =>
                                Object.values(row).map(v => v === null || v === undefined ? '' : v)
                            );

                            exportData.tableData = {
                                columns: columns,
                                data: rows
                            };

                            console.log('[Dashboard] Table data extracted:', {
                                columnsCount: columns.length,
                                rowsCount: rows.length
                            });
                        }
                    } catch (e) {
                        console.error('[Dashboard] Failed to parse table data:', e);
                    }
                }
            }


            console.log('[Dashboard] Export data prepared:', {
                metricsCount: exportData.metrics.length,
                insightsCount: exportData.insights.length,
                hasChart: !!exportData.chartImage
            });

            // 调用后端API生成PDF
            await ExportDashboardToPDF(exportData);

            console.log('[Dashboard] PDF export completed successfully');
            setToast({ message: 'PDF导出成功！', type: 'success' });
        } catch (error) {
            console.error('[Dashboard] PDF export failed:', error);
            setToast({
                message: 'PDF导出失败: ' + (error instanceof Error ? error.message : String(error)),
                type: 'error'
            });
        }
    };

    if (!data) {
        return (
            <div className="flex items-center justify-center h-full">
                <div className="animate-pulse text-slate-400">{t('loading_insights')}</div>
            </div>
        );
    }

    const renderChart = () => {
        if (!activeChart) {
            logger.debug("renderChart: activeChart is null");
            return null;
        }

        logger.debug(`renderChart: type=${activeChart.type}, hasChartData=${!!activeChart.chartData}`);

        // Extract charts array if chartData is available (multi-chart support)
        const charts = activeChart.chartData?.charts || [];
        logger.debug(`renderChart: charts array length=${charts.length}`);

        const hasMultipleCharts = charts.length > 1;

        // Use chart from charts array if available, otherwise fall back to single chart (activeChart.data)
        const currentChart = charts.length > 0 ? charts[currentChartIndex] : null;
        const chartType = currentChart ? currentChart.type : activeChart.type;
        const chartData = currentChart ? currentChart.data : activeChart.data;

        logger.debug(`renderChart: currentChart type=${chartType}, chartData type=${typeof chartData}`);
        if (typeof chartData === 'string') {
            logger.debug(`renderChart: chartData string length=${chartData.length}, preview=${chartData.substring(0, 100)}`);
        }

        // Generate a stable key for the chart based on content
        const contentHash = typeof chartData === 'string'
            ? chartData.substring(0, 50)
            : JSON.stringify(chartData).substring(0, 50);
        const chartKey = `chart-${chartType}-${currentChartIndex}-${contentHash.replace(/[^a-zA-Z0-9]/g, '')}`;

        const renderSingleChart = () => {
            if (chartType === 'image') {
                return (
                    <div
                        className="w-full bg-white rounded-xl border border-slate-200 p-4 shadow-sm flex justify-center cursor-zoom-in group relative"
                        onDoubleClick={() => setImageModalOpen(true)}
                        title="Double click to expand"
                    >
                        <img src={chartData} alt="Analysis Chart" className="max-h-[400px] object-contain group-hover:scale-[1.01] transition-transform duration-300" />
                        <div className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity bg-black/5 pointer-events-none rounded-xl">
                            <span className="bg-white/90 px-3 py-1 rounded-full text-xs font-medium text-slate-600 shadow-sm backdrop-blur-sm">{t('double_click_to_zoom')}</span>
                        </div>
                    </div>
                );
            }

            if (chartType === 'echarts') {
                // Check if chartData is a file reference - if so, use dedicated loader component
                if (typeof chartData === 'string' && chartData.startsWith('file://')) {
                    logger.debug(`Detected file reference for ECharts: ${chartData}`);
                    return <EChartsFileLoader
                        fileRef={chartData}
                        threadId={activeThreadId || null}
                        chartKey={chartKey}
                        onDoubleClick={() => setChartModalOpen(true)}
                    />;
                }

                // Otherwise, handle inline chart data
                try {
                    // Validate that chartData is a string before parsing
                    if (typeof chartData !== 'string') {
                        logger.error(`Invalid ECharts data: not a string, type=${typeof chartData}`);
                        return null;
                    }

                    // Check if the string contains function definitions (invalid JSON)
                    if (chartData.includes('function(') || chartData.includes('function (')) {
                        logger.error("Invalid ECharts data: contains function definitions");

                        // Try to clean the data by removing function definitions
                        try {
                            let cleanedData = chartData;

                            // Remove common function patterns that might appear in ECharts configs
                            cleanedData = cleanedData.replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '');
                            cleanedData = cleanedData.replace(/,?\s*"?matter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '');
                            cleanedData = cleanedData.replace(/,?\s*[a-zA-Z_$][a-zA-Z0-9_$]*\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '');

                            // Clean up any trailing commas that might be left
                            cleanedData = cleanedData.replace(/,(\s*[}\]])/g, '$1');

                            // Try to parse the cleaned data
                            const cleanedOptions = JSON.parse(cleanedData);

                            if (cleanedOptions && typeof cleanedOptions === 'object') {
                                logger.info("Successfully cleaned ECharts data by removing functions");

                                return (
                                    <div className="w-full">
                                        <ReactECharts
                                            option={cleanedOptions}
                                            style={{ height: '400px', width: '100%' }}
                                            opts={{ renderer: 'canvas' }}
                                            className="echarts-for-react"
                                        />
                                    </div>
                                );
                            }
                        } catch (cleanError) {
                            logger.error(`Failed to clean ECharts data: ${cleanError}`);
                        }

                        // If cleaning fails, show error message
                        return (
                            <div className="w-full bg-amber-50 border border-amber-200 rounded-xl p-4 shadow-sm">
                                <div className="flex items-center gap-3">
                                    <div className="bg-amber-100 p-2 rounded-lg">
                                        <svg className="w-5 h-5 text-amber-600" fill="currentColor" viewBox="0 0 20 20">
                                            <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                                        </svg>
                                    </div>
                                    <div className="flex-1">
                                        <p className="text-sm font-medium text-amber-800">{t('chart_data_format_error')}</p>
                                        <p className="text-xs text-amber-600 mt-1">{t('chart_contains_functions')}</p>
                                    </div>
                                </div>
                            </div>
                        );
                    }

                    // 清理chartData中的JavaScript函数
                    let cleanedChartData = chartData;
                    if (typeof chartData === 'string') {
                        cleanedChartData = chartData
                            .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                            .replace(/,?\s*"?matter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                            .replace(/,?\s*[a-zA-Z_$][a-zA-Z0-9_$]*\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                            .replace(/,(\s*[}\]])/g, '$1')
                            .replace(/(\{\s*),/g, '$1');
                    }

                    const options = JSON.parse(cleanedChartData);

                    // 验证ECharts选项的基本结构
                    if (!options || typeof options !== 'object') {
                        logger.error("Invalid ECharts options: not an object");
                        return null;
                    }

                    // 详细日志
                    logger.debug(`ECharts options parsed successfully`);
                    logger.debug(`Has title: ${!!options.title}`);
                    logger.debug(`Has series: ${!!options.series}, length: ${options.series?.length || 0}`);
                    logger.debug(`Has grid: ${!!options.grid}`);
                    logger.debug(`Has xAxis: ${!!options.xAxis}`);
                    logger.debug(`Has yAxis: ${!!options.yAxis}`);

                    // 修复常见的ECharts配置问题
                    const fixedOptions = { ...options };

                    // 修复pie图表不应该有gridIndex的问题
                    if (fixedOptions.series && Array.isArray(fixedOptions.series)) {
                        fixedOptions.series = fixedOptions.series.map((s: any) => {
                            if (s.type === 'pie' && s.gridIndex !== undefined) {
                                const { gridIndex, xAxisIndex, yAxisIndex, ...rest } = s;
                                logger.debug(`Removed gridIndex from pie chart: ${s.name}`);
                                return rest;
                            }
                            return s;
                        });
                    }

                    // 确保必要的属性存在
                    const validatedOptions = {
                        ...fixedOptions,
                        // 确保有基本的配置
                        animation: fixedOptions.animation !== false,
                        // 如果没有series，添加一个空的
                        series: fixedOptions.series || []
                    };

                    return (
                        <div
                            className="cursor-zoom-in group relative"
                            onDoubleClick={() => setChartModalOpen(true)}
                            title="Double click to expand"
                        >
                            <Chart
                                key={chartKey}
                                options={validatedOptions}
                                height="400px"
                            />
                            <div className="absolute top-4 right-4 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
                                <span className="bg-slate-800/80 text-white px-3 py-1 rounded-full text-xs font-medium shadow-sm backdrop-blur-sm">Double click to expand</span>
                            </div>
                        </div>
                    );
                } catch (e) {
                    logger.error(`Failed to parse ECharts options: ${e}`);
                    return (
                        <div className="w-full bg-red-50 border border-red-200 rounded-xl p-4 shadow-sm">
                            <div className="flex items-center gap-3">
                                <div className="bg-red-100 p-2 rounded-lg">
                                    <svg className="w-5 h-5 text-red-600" fill="currentColor" viewBox="0 0 20 20">
                                        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                                    </svg>
                                </div>
                                <div className="flex-1">
                                    <p className="text-sm font-medium text-red-800">Cannot Display Chart</p>
                                    <p className="text-xs text-red-600 mt-1">The chart data is malformed. Error: {(e as Error).message}</p>
                                </div>
                            </div>
                        </div>
                    );
                }
            }

            if (chartType === 'table') {
                // Check if chartData is a file reference
                if (typeof chartData === 'string' && chartData.startsWith('file://')) {
                    logger.debug(`Detected file reference for table data: ${chartData}`);
                    return <TableFileLoader fileRef={chartData} threadId={activeThreadId || null} />;
                }

                // Otherwise, handle inline table data
                // Parse if it's a JSON string
                let tableData = chartData;
                if (typeof chartData === 'string') {
                    try {
                        tableData = JSON.parse(chartData);
                        logger.debug(`Parsed table data from JSON string, rows: ${tableData.length}`);
                    } catch (e) {
                        logger.error(`Failed to parse table data: ${e}`);
                        return (
                            <div className="w-full bg-red-50 border border-red-200 rounded-xl p-4 shadow-sm">
                                <div className="flex items-center gap-3">
                                    <div className="bg-red-100 p-2 rounded-lg">
                                        <Table className="w-5 h-5 text-red-600" />
                                    </div>
                                    <div className="flex-1">
                                        <p className="text-sm font-medium text-red-800">Cannot Display Table</p>
                                        <p className="text-xs text-red-600 mt-1">The table data is malformed. Error: {(e as Error).message}</p>
                                    </div>
                                </div>
                            </div>
                        );
                    }
                }

                if (!tableData || !Array.isArray(tableData) || tableData.length === 0) {
                    logger.warn(`Invalid table data: not an array or empty`);
                    return null;
                }

                const columns = Object.keys(tableData[0]);
                return (
                    <div className="w-full bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
                        <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100 bg-slate-50">
                            <div className="flex items-center gap-2">
                                <Table className="w-4 h-4 text-blue-500" />
                                <span className="text-sm font-medium text-slate-700">{t('analysis_result') || 'Analysis Result'}</span>
                                <span className="text-xs text-slate-400">({tableData.length} rows)</span>
                            </div>
                            <button
                                onClick={() => downloadTableAsCSV(tableData, 'analysis_result.csv')}
                                className="flex items-center gap-1 px-2 py-1 text-xs text-blue-600 hover:bg-blue-50 rounded transition-colors"
                            >
                                <Download className="w-3 h-3" />
                                CSV
                            </button>
                        </div>
                        <div className="overflow-x-auto max-h-[400px] overflow-y-auto">
                            <table className="w-full text-sm">
                                <thead className="bg-slate-50 sticky top-0">
                                    <tr>
                                        {columns.map(col => (
                                            <th key={col} className="px-4 py-2 text-left text-xs font-semibold text-slate-600 border-b border-slate-200">
                                                {col}
                                            </th>
                                        ))}
                                    </tr>
                                </thead>
                                <tbody>
                                    {tableData.slice(0, 100).map((row, i) => (
                                        <tr key={i} className="hover:bg-slate-50 transition-colors">
                                            {columns.map(col => (
                                                <td key={col} className="px-4 py-2 text-slate-700 border-b border-slate-100 whitespace-nowrap">
                                                    {formatCellValue(row[col])}
                                                </td>
                                            ))}
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                            {tableData.length > 100 && (
                                <div className="px-4 py-2 text-center text-xs text-slate-400 bg-slate-50 border-t border-slate-100">
                                    Showing first 100 of {tableData.length} rows
                                </div>
                            )}
                        </div>
                    </div>
                );
            }

            if (chartType === 'csv') {
                return (
                    <div className="w-full bg-white rounded-xl border border-slate-200 p-4 shadow-sm">
                        <div className="flex items-center gap-3">
                            <div className="bg-green-100 p-2 rounded-lg">
                                <Download className="w-5 h-5 text-green-600" />
                            </div>
                            <div className="flex-1">
                                <p className="text-sm font-medium text-slate-700">{t('data_file_ready') || 'Data File Ready'}</p>
                                <p className="text-xs text-slate-400">{t('click_to_download') || 'Click to download'}</p>
                            </div>
                            <a
                                href={chartData}
                                download="analysis_data.csv"
                                className="px-4 py-2 bg-green-600 text-white text-sm font-medium rounded-lg hover:bg-green-700 transition-colors flex items-center gap-2"
                            >
                                <Download className="w-4 h-4" />
                                Download CSV
                            </a>
                        </div>
                    </div>
                );
            }

            return null;
        };

        return (
            <div className="space-y-3">
                {renderSingleChart()}

                {/* Multi-chart navigation buttons */}
                {hasMultipleCharts && (
                    <div className="flex items-center justify-center gap-3">
                        <button
                            onClick={() => setCurrentChartIndex(prev => Math.max(0, prev - 1))}
                            disabled={currentChartIndex === 0}
                            className="flex items-center gap-1 px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition-all shadow-sm"
                            title="Previous chart"
                        >
                            <ChevronLeft className="w-4 h-4" />
                            Previous
                        </button>
                        <span className="text-sm text-slate-600 font-medium">
                            {currentChartIndex + 1} / {charts.length}
                        </span>
                        <button
                            onClick={() => setCurrentChartIndex(prev => Math.min(charts.length - 1, prev + 1))}
                            disabled={currentChartIndex === charts.length - 1}
                            className="flex items-center gap-1 px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition-all shadow-sm"
                            title="Next chart"
                        >
                            Next
                            <ChevronRight className="w-4 h-4" />
                        </button>
                    </div>
                )}

                {/* Data tables display below charts */}
                {renderDataTables()}
            </div>
        );
    };

    // Render data tables from chartData (for JSON table data display)
    const renderDataTables = () => {
        if (!activeChart?.chartData?.charts) return null;

        // Extract all charts with type 'table'
        const tableCharts = activeChart.chartData.charts.filter(
            chart => chart.type === 'table'
        );

        if (tableCharts.length === 0) return null;

        // 如果当前显示的图表就是 table，不要重复显示
        // 检查当前图表索引对应的图表类型
        const charts = activeChart.chartData.charts || [];
        const currentChart = charts.length > 0 ? charts[currentChartIndex] : null;
        const isCurrentChartTable = currentChart && currentChart.type === 'table';

        // 如果只有一个 table 且正在显示，不重复渲染
        if (tableCharts.length === 1 && isCurrentChartTable) {
            return null;
        }

        return (
            <div className="mt-6 space-y-4">
                <h3 className="text-md font-semibold text-slate-700 flex items-center gap-2">
                    <Table className="w-5 h-5 text-blue-500" />
                    {t('analysis_data') || 'Analysis Data'}
                </h3>
                {tableCharts.map((chart, tableIndex) => {
                    // 跳过当前正在显示的 table
                    if (isCurrentChartTable && tableIndex === currentChartIndex) {
                        return null;
                    }

                    try {
                        // 清理表格数据中的JavaScript函数
                        let cleanedData = chart.data;
                        if (typeof chart.data === 'string') {
                            cleanedData = chart.data
                                .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                .replace(/,?\s*"?matter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                .replace(/,?\s*[a-zA-Z_$][a-zA-Z0-9_$]*\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                .replace(/,(\s*[}\]])/g, '$1')
                                .replace(/(\{\s*),/g, '$1');
                        }

                        const tableData = JSON.parse(cleanedData);
                        if (!tableData || !Array.isArray(tableData) || tableData.length === 0) {
                            return null;
                        }

                        const columns = Object.keys(tableData[0]);

                        return (
                            <div key={tableIndex} className="w-full bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
                                <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100 bg-slate-50">
                                    <div className="flex items-center gap-2">
                                        <Table className="w-4 h-4 text-blue-500" />
                                        <span className="text-sm font-medium text-slate-700">
                                            {tableCharts.length > 1 ? `${t('table') || 'Table'} ${tableIndex + 1}` : (t('data_table') || 'Data Table')}
                                        </span>
                                        <span className="text-xs text-slate-400">({tableData.length} rows)</span>
                                    </div>
                                    <button
                                        onClick={() => downloadTableAsCSV(tableData, `analysis_data_${tableIndex + 1}.csv`)}
                                        className="flex items-center gap-1 px-2 py-1 text-xs text-blue-600 hover:bg-blue-50 rounded transition-colors"
                                    >
                                        <Download className="w-3 h-3" />
                                        CSV
                                    </button>
                                </div>
                                <div className="overflow-x-auto max-h-[400px] overflow-y-auto">
                                    <table className="w-full text-sm">
                                        <thead className="bg-slate-50 sticky top-0">
                                            <tr>
                                                {columns.map(col => (
                                                    <th key={col} className="px-4 py-2 text-left text-xs font-semibold text-slate-600 border-b border-slate-200">
                                                        {col}
                                                    </th>
                                                ))}
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {tableData.slice(0, 100).map((row, i) => (
                                                <tr key={i} className="hover:bg-slate-50 transition-colors">
                                                    {columns.map(col => (
                                                        <td key={col} className="px-4 py-2 text-slate-700 border-b border-slate-100 whitespace-nowrap">
                                                            {formatCellValue(row[col])}
                                                        </td>
                                                    ))}
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                    {tableData.length > 100 && (
                                        <div className="px-4 py-2 text-center text-xs text-slate-400 bg-slate-50 border-t border-slate-100">
                                            Showing first 100 of {tableData.length} rows
                                        </div>
                                    )}
                                </div>
                            </div>
                        );
                    } catch (e) {
                        logger.error(`Failed to parse table data: ${e}`);
                        return null;
                    }
                })}
            </div>
        );
    };

    // Helper function to format cell values
    const formatCellValue = (value: any): string => {
        if (value === null || value === undefined) return '-';
        if (typeof value === 'number') {
            return value.toLocaleString();
        }
        return String(value);
    };

    // Helper function to download table as CSV
    const downloadTableAsCSV = (data: any[], filename: string) => {
        if (!data || data.length === 0) return;

        const columns = Object.keys(data[0]);
        const csvContent = [
            columns.join(','),
            ...data.map(row =>
                columns.map(col => {
                    const val = row[col];
                    if (val === null || val === undefined) return '';
                    const strVal = String(val);
                    // Escape quotes and wrap in quotes if contains comma
                    if (strVal.includes(',') || strVal.includes('"') || strVal.includes('\n')) {
                        return `"${strVal.replace(/"/g, '""')}"`;
                    }
                    return strVal;
                }).join(',')
            )
        ].join('\n');

        // 添加BOM以确保中文正确显示
        const BOM = '\uFEFF';
        const blob = new Blob([BOM + csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = filename;
        link.click();
    };

    // Helper function to download file
    const downloadFile = async (file: main.SessionFile) => {
        try {
            if (!activeThreadId) {
                logger.error('No active thread ID for file download');
                setToast({ message: t('download_failed') || 'Download failed', type: 'error' });
                return;
            }

            logger.debug(`Downloading file: ${file.name} from thread ${activeThreadId}`);
            
            // Call backend API to show save dialog and copy file
            const { DownloadSessionFile } = await import('../../wailsjs/go/main/App');
            await DownloadSessionFile(activeThreadId, file.name);
            
            logger.info(`File downloaded successfully: ${file.name}`);
            setToast({ message: t('download_success') || 'File saved successfully', type: 'success' });
        } catch (error) {
            logger.error(`Failed to download file: ${error}`);
            setToast({ message: t('download_failed') || 'Download failed: ' + (error instanceof Error ? error.message : String(error)), type: 'error' });
        }
    };

    // Helper function to get file thumbnail URL for images
    const getFileThumbnailUrl = async (file: main.SessionFile): Promise<string | null> => {
        if (file.type !== 'image' || !activeThreadId) {
            return null;
        }
        
        try {
            const { GetSessionFilePath } = await import('../../wailsjs/go/main/App');
            const filePath = await GetSessionFilePath(activeThreadId, file.name);
            // Convert Windows path to file URL
            return `file:///${filePath.replace(/\\/g, '/')}`;
        } catch (error) {
            logger.error(`Failed to get thumbnail URL: ${error}`);
            return null;
        }
    };

    // Helper function to get file icon based on type
    const getFileIcon = (fileType: string) => {
        switch (fileType) {
            case 'image':
                return <FileImage className="w-5 h-5 text-blue-500" />;
            case 'csv':
            case 'data':
                return <FileText className="w-5 h-5 text-green-500" />;
            default:
                return <Download className="w-5 h-5 text-slate-500" />;
        }
    };

    // Helper function to format file size
    const formatFileSize = (bytes: number): string => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
    };

    // Render session files download section
    const renderFilesSection = () => {
        logger.debug(`[renderFilesSection] Called with sessionFiles count: ${sessionFiles?.length || 0}, selectedMessageId: ${selectedMessageId}`);
        
        if (sessionFiles && sessionFiles.length > 0) {
            logger.debug(`[renderFilesSection] Session files: ${JSON.stringify(sessionFiles.map(f => ({
                name: f.name,
                message_id: f.message_id,
                type: f.type
            })))}`);
        }
        
        if (!sessionFiles || sessionFiles.length === 0) {
            logger.debug(`[renderFilesSection] No session files available`);
            return null;
        }

        // 过滤只显示当前消息的文件
        const filteredFiles = selectedMessageId 
            ? sessionFiles.filter(file => {
                const matches = file.message_id === selectedMessageId;
                logger.debug(`[renderFilesSection] File ${file.name}: message_id=${file.message_id}, matches=${matches}`);
                return matches;
            })
            : sessionFiles;

        logger.debug(`[renderFilesSection] Filtered ${filteredFiles.length} files for message ${selectedMessageId}`);
        
        if (filteredFiles.length === 0) {
            logger.warn(`[renderFilesSection] No files match selectedMessageId: ${selectedMessageId}`);
            // 如果没有匹配的文件，显示所有文件（可能是message_id不匹配的问题）
            logger.debug(`[renderFilesSection] Showing all ${sessionFiles.length} files as fallback`);
            return (
                <section className="mb-6 animate-in fade-in slide-in-from-top-2 duration-500">
                    <h2 className="text-lg font-semibold text-slate-700 mb-4 flex items-center gap-2">
                        <Download className="w-5 h-5 text-blue-500" />
                        {t('session_files') || 'Generated Files'}
                        <span className="text-xs text-amber-600 bg-amber-50 px-2 py-1 rounded">
                            (Showing all files - message filter not matched)
                        </span>
                    </h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        {sessionFiles.map((file, index) => (
                            <FileCard key={index} file={file} />
                        ))}
                    </div>
                </section>
            );
        }

        return (
            <section className="mb-6 animate-in fade-in slide-in-from-top-2 duration-500">
                <h2 className="text-lg font-semibold text-slate-700 mb-4 flex items-center gap-2">
                    <Download className="w-5 h-5 text-blue-500" />
                    {t('session_files') || 'Generated Files'}
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {filteredFiles.map((file, index) => (
                        <FileCard key={index} file={file} />
                    ))}
                </div>
            </section>
        );
    };

    // File card component with thumbnail support
    const FileCard: React.FC<{ file: main.SessionFile }> = ({ file }) => {
        const [thumbnailUrl, setThumbnailUrl] = React.useState<string | null>(null);
        const [thumbnailLoading, setThumbnailLoading] = React.useState(false);

        React.useEffect(() => {
            if (activeThreadId) {
                // Load thumbnail for image and CSV files
                const loadThumbnail = async () => {
                    setThumbnailLoading(true);
                    try {
                        if (file.type === 'image') {
                            // Load image thumbnail
                            const { GetSessionFileAsBase64 } = await import('../../wailsjs/go/main/App');
                            const base64Data = await GetSessionFileAsBase64(activeThreadId, file.name);
                            setThumbnailUrl(base64Data);
                        } else if (file.type === 'csv') {
                            // Generate CSV preview thumbnail
                            const { GenerateCSVThumbnail } = await import('../../wailsjs/go/main/App');
                            const base64Data = await GenerateCSVThumbnail(activeThreadId, file.name);
                            setThumbnailUrl(base64Data);
                        }
                    } catch (error) {
                        logger.error(`Failed to load thumbnail: ${error}`);
                        setThumbnailUrl(null);
                    } finally {
                        setThumbnailLoading(false);
                    }
                };
                
                if (file.type === 'image' || file.type === 'csv') {
                    loadThumbnail();
                }
            }
        }, [file, activeThreadId]);

        return (
            <div
                className="bg-white rounded-xl border border-slate-200 shadow-sm hover:shadow-md transition-all duration-200 cursor-pointer group overflow-hidden"
                onClick={() => downloadFile(file)}
            >
                {/* 图片/CSV 缩略图 */}
                {(file.type === 'image' || file.type === 'csv') && (
                    <div className="w-full h-32 bg-slate-100 overflow-hidden flex items-center justify-center">
                        {thumbnailLoading ? (
                            <div className="animate-pulse text-slate-400 text-xs">Loading preview...</div>
                        ) : thumbnailUrl ? (
                            <img 
                                src={thumbnailUrl} 
                                alt={file.name}
                                className="w-full h-full object-contain group-hover:scale-105 transition-transform duration-200"
                            />
                        ) : (
                            file.type === 'image' ? (
                                <FileImage className="w-8 h-8 text-slate-300" />
                            ) : (
                                <FileText className="w-8 h-8 text-slate-300" />
                            )
                        )}
                    </div>
                )}
                
                {/* 文件信息 */}
                <div className="p-4">
                    <div className="flex items-start gap-3">
                        <div className="flex-shrink-0 p-2 bg-slate-50 rounded-lg group-hover:bg-blue-50 transition-colors">
                            {getFileIcon(file.type)}
                        </div>
                        <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium text-slate-700 truncate group-hover:text-blue-600 transition-colors" title={file.name}>
                                {file.name}
                            </p>
                            <p className="text-xs text-slate-400 mt-1">
                                {formatFileSize(file.size)}
                            </p>
                            {file.created_at && (
                                <p className="text-xs text-slate-400 mt-0.5">
                                    {new Date(file.created_at * (file.created_at < 10000000000 ? 1000 : 1)).toLocaleString()}
                                </p>
                            )}
                        </div>
                        <div className="flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                            <Download className="w-4 h-4 text-blue-500" />
                        </div>
                    </div>
                </div>
            </div>
        );
    };

    const handleDashboardClick = (e: React.MouseEvent) => {
        // 只有当点击的是Dashboard容器本身或其直接子元素（非交互元素）时才隐藏聊天
        const target = e.target as HTMLElement;

        // 检查是否是交互元素
        const isInteractiveElement = target.tagName === 'BUTTON' ||
            target.tagName === 'A' ||
            target.tagName === 'INPUT' ||
            target.tagName === 'SELECT' ||
            target.tagName === 'TEXTAREA' ||
            target.closest('button') ||
            target.closest('a') ||
            target.closest('[role="button"]') ||
            target.closest('.cursor-pointer') ||
            target.closest('.cursor-zoom-in');

        // 检查是否在图表区域内（用户可能正在查看分析结果）
        const isInChartArea = target.closest('[class*="chart"]') ||
            target.closest('canvas') ||
            target.closest('svg') ||
            target.closest('table') ||
            target.closest('.echarts-container');

        // 检查是否在智能洞察卡片内
        const isInInsightCard = target.closest('[class*="insight"]') ||
            target.closest('[class*="metric"]');

        // 只有在点击空白区域时才隐藏聊天侧边栏
        // 如果聊天区已经打开且用户点击了智能洞察，不要隐藏（让用户继续使用）
        if (!isInteractiveElement && !isInChartArea && !isInInsightCard && onDashboardClick) {
            onDashboardClick();
        }
    };

    const handleInsightClick = (insight: any) => {
        // 检查是否有分析正在进行
        if (isAnalysisLoading && loadingThreadId) {
            logger.debug(`Analysis in progress for thread ${loadingThreadId}, blocking insight click`);
            setToast({
                message: t('analysis_in_progress') || '分析进行中，请等待当前分析完成后再发起新的分析',
                type: 'warning'
            });
            return;
        }

        // 区分洞察来源，决定不同的处理方式
        if (insight.source === 'llm_suggestion') {
            // LLM生成的洞察：在当前会话中继续分析
            logger.debug(`LLM insight clicked, continuing in current session: ${insight.text.substring(0, 50)}`);
            logger.debug(`Using activeThreadId: ${activeThreadId}`);

            // 优先使用 activeThreadId，确保在正确会话中发送
            if (activeThreadId) {
                EventsEmit("analyze-insight-in-session", {
                    text: insight.text,
                    threadId: activeThreadId,  // 直接传递 threadId
                    userMessageId: insight.userMessageId,  // 保留作为备份
                    continueInSession: true
                });
            } else {
                // 没有活动会话，回退到使用 userMessageId
                logger.warn('No activeThreadId, falling back to userMessageId');
                EventsEmit("analyze-insight-in-session", {
                    text: insight.text,
                    userMessageId: insight.userMessageId,
                    continueInSession: true
                });
            }
        } else if (insight.data_source_id) {
            // 系统洞察：创建新会话进行分析
            logger.debug(`System insight clicked, creating new session: ${insight.text.substring(0, 50)}`);
            EventsEmit('start-new-chat', {
                dataSourceId: insight.data_source_id,
                sessionName: `${t('analysis_session_prefix')}${insight.source_name || insight.text}`,
                keepChatOpen: true // 标记这是创建新会话，不要隐藏聊天区
            });
        } else {
            // 其他洞察：使用analyze-insight事件（向后兼容）
            logger.debug(`Generic insight clicked: ${insight.text.substring(0, 50)}`);
            EventsEmit("analyze-insight", insight.text);
        }
    };

    return (
        <div
            className="flex-1 flex flex-col h-full overflow-hidden"
            onClick={handleDashboardClick}
        >
            <header className="px-6 py-8 relative">
                <div className="flex items-start justify-between">
                    <div className="flex-1">
                        <h1 className="text-2xl font-bold text-slate-800">{t('smart_dashboard')}</h1>
                        <p className="text-slate-500">{t('welcome_back')}</p>
                    </div>

                    {/* 导出按钮 - 只有在有可导出内容时显示 */}
                    {hasExportableContent() && (
                        <div className="relative export-dropdown-container">
                            <button
                                onClick={() => setExportDropdownOpen(!exportDropdownOpen)}
                                className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors shadow-sm"
                                title="导出报告"
                            >
                                <Download className="w-4 h-4" />
                                <span className="text-sm font-medium">导出</span>
                            </button>

                            {/* 导出下拉菜单 */}
                            {exportDropdownOpen && (
                                <div className="absolute right-0 top-full mt-2 w-48 bg-white rounded-lg shadow-lg border border-slate-200 py-2 z-50">
                                    <button
                                        onClick={exportAsPDF}
                                        className="w-full flex items-center gap-3 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                                    >
                                        <FileImage className="w-4 h-4 text-red-600" />
                                        <span>导出为 PDF</span>
                                    </button>
                                    <button
                                        onClick={exportDataFiles}
                                        className="w-full flex items-center gap-3 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                                    >
                                        <Download className="w-4 h-4 text-green-600" />
                                        <span>导出数据文件</span>
                                    </button>
                                </div>
                            )}
                        </div>
                    )}
                </div>

                {userRequestText && (
                    <div className="mt-4 p-3 bg-blue-50 border border-blue-100 rounded-lg">
                        <div className="flex items-start gap-2">
                            <BarChart3 className="w-4 h-4 text-blue-600 mt-0.5 flex-shrink-0" />
                            <div className="flex-1">
                                <p className="text-xs font-semibold text-blue-900 uppercase tracking-wide mb-1">{t('analysis_request') || 'Analysis Request'}</p>
                                <p className="text-sm text-blue-800">{userRequestText}</p>
                            </div>
                        </div>
                        {!activeChart && (
                            <div className="mt-2 p-2 bg-amber-50 border border-amber-200 rounded text-xs text-amber-800 flex items-center gap-2">
                                <svg className="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                    <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                                </svg>
                                <span>{t('no_visualization_results')}</span>
                            </div>
                        )}
                    </div>
                )}
            </header>

            <div className="flex-1 overflow-y-auto px-6 pb-8">
                {/* 核心指标区域 - 显示在最上方 */}
                {(() => {
                    // 过滤掉无效的指标（没有具体数值的）
                    const validMetrics = data.metrics?.filter(metric => {
                        // 检查 value 是否有效
                        if (!metric.value || typeof metric.value !== 'string') {
                            return false;
                        }

                        const trimmedValue = metric.value.trim();

                        // 排除空字符串
                        if (trimmedValue === '') {
                            return false;
                        }

                        // 排除常见的占位符
                        const invalidValues = ['N/A', 'n/a', 'null', 'undefined', '-', '--', '...', 'TBD', 'tbd'];
                        if (invalidValues.includes(trimmedValue)) {
                            return false;
                        }

                        return true;
                    }) || [];

                    return validMetrics.length > 0 && (
                        <section className="mb-6 animate-in fade-in slide-in-from-top-2 duration-500">
                            <h2 className="text-lg font-semibold text-slate-700 mb-4">{t('key_metrics')}</h2>
                            <DashboardLayout>
                                {validMetrics.map((metric, index) => (
                                    <MetricCard
                                        key={index}
                                        title={metric.title}
                                        value={metric.value}
                                        change={metric.change}
                                    />
                                ))}
                            </DashboardLayout>
                        </section>
                    );
                })()}

                {/* 分析图表/表格区域 */}
                {activeChart && (
                    <section className="mb-6 animate-in fade-in slide-in-from-top-4 duration-500">
                        <h2 className="text-lg font-semibold text-slate-700 mb-4">
                            {activeChart.type === 'table' ? (t('analysis_data') || 'Analysis Data') : t('latest_analysis')}
                        </h2>
                        {renderChart()}
                    </section>
                )}

                {/* Session Files Download Section */}
                {renderFilesSection()}

                <ImageModal
                    isOpen={imageModalOpen}
                    imageUrl={(() => {
                        if (!activeChart) return '';
                        const charts = activeChart.chartData?.charts || [];
                        if (charts.length > 0) {
                            const currentChart = charts[currentChartIndex];
                            return currentChart?.type === 'image' ? currentChart.data : '';
                        }
                        return activeChart.type === 'image' ? activeChart.data : '';
                    })()}
                    onClose={() => setImageModalOpen(false)}
                />

                {activeChart?.type === 'echarts' && (
                    <ChartModal
                        isOpen={chartModalOpen}
                        options={(() => {
                            const charts = activeChart.chartData?.charts || [];
                            if (charts.length > 0) {
                                const currentChart = charts[currentChartIndex];
                                if (currentChart?.type === 'echarts') {
                                    // 清理数据中的JavaScript函数
                                    let cleanedData = currentChart.data;
                                    if (typeof currentChart.data === 'string') {
                                        cleanedData = currentChart.data
                                            .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                            .replace(/,?\s*"?matter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                            .replace(/,?\s*[a-zA-Z_$][a-zA-Z0-9_$]*\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                            .replace(/,(\s*[}\]])/g, '$1')
                                            .replace(/(\{\s*),/g, '$1');
                                    }
                                    try {
                                        return JSON.parse(cleanedData);
                                    } catch (e) {
                                        console.error("Failed to parse cleaned chart data:", e);
                                        return {};
                                    }
                                }
                                return {};
                            }
                            // 清理activeChart.data中的JavaScript函数
                            let cleanedData = activeChart.data;
                            if (typeof activeChart.data === 'string') {
                                cleanedData = activeChart.data
                                    .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                    .replace(/,?\s*"?matter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                    .replace(/,?\s*[a-zA-Z_$][a-zA-Z0-9_$]*\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                    .replace(/,(\s*[}\]])/g, '$1')
                                    .replace(/(\{\s*),/g, '$1');
                            }
                            try {
                                return JSON.parse(cleanedData);
                            } catch (e) {
                                console.error("Failed to parse cleaned active chart data:", e);
                                return {};
                            }
                        })()}
                        onClose={() => setChartModalOpen(false)}
                    />
                )}

                {/* 自动洞察区域 - 显示在最下方 */}
                {data.insights && Array.isArray(data.insights) && data.insights.length > 0 && (
                    <section className="mb-6">
                        <h2 className="text-lg font-semibold text-slate-700 mb-4">{t('automated_insights')}</h2>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            {data.insights.map((insight, index) => (
                                <SmartInsight
                                    key={index}
                                    text={insight.text}
                                    icon={insight.icon}
                                    onClick={() => handleInsightClick(insight)}
                                />
                            ))}
                        </div>
                    </section>
                )}
            </div>

            {/* Toast notification */}
            {toast && (
                <Toast
                    message={toast.message}
                    type={toast.type}
                    onClose={() => setToast(null)}
                />
            )}
        </div>
    );
};

export default Dashboard;
