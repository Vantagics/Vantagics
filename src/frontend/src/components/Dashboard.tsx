import React, { useState } from 'react';
import DashboardLayout from './DashboardLayout';
import MetricCard from './MetricCard';
import SmartInsight from './SmartInsight';
import Chart from './Chart';
import ImageModal from './ImageModal';
import ChartModal from './ChartModal';
import { main } from '../../wailsjs/go/models';
import { useLanguage } from '../i18n';
import { EventsEmit } from '../../wailsjs/runtime/runtime';
import { Download, Table, BarChart3, ChevronLeft, ChevronRight, FileText, FileImage } from 'lucide-react';

interface DashboardProps {
    data: main.DashboardData | null;
    activeChart?: { type: 'echarts' | 'image' | 'table' | 'csv', data: any, chartData?: main.ChartData } | null;
    userRequestText?: string | null;
    onDashboardClick?: () => void;
    isChatOpen?: boolean;
}

const Dashboard: React.FC<DashboardProps> = ({ data, activeChart, userRequestText, onDashboardClick, isChatOpen }) => {
    const { t } = useLanguage();
    const [imageModalOpen, setImageModalOpen] = useState(false);
    const [chartModalOpen, setChartModalOpen] = useState(false);
    const [currentChartIndex, setCurrentChartIndex] = useState(0);
    const [exportDropdownOpen, setExportDropdownOpen] = useState(false);

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
                        pixelRatio: 2, // 高分辨率
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

    // 导出为HTML（改进版本，支持图表转图片）
    const exportAsHTML = async () => {
        try {
            const timestamp = new Date().toLocaleString('zh-CN');
            
            // 获取图表图片（如果有ECharts）
            let chartImageData = null;
            if (activeChart && activeChart.type === 'echarts') {
                chartImageData = await captureEChartsAsImage();
            }
            
            let htmlContent = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>智能仪表盘报告 - ${timestamp}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #334155;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f8fafc;
        }
        .header {
            background: linear-gradient(135deg, #3b82f6, #6366f1);
            color: white;
            padding: 30px;
            border-radius: 12px;
            margin-bottom: 30px;
            text-align: center;
        }
        .header h1 {
            margin: 0 0 10px 0;
            font-size: 2.5em;
            font-weight: bold;
        }
        .header p {
            margin: 0;
            opacity: 0.9;
            font-size: 1.1em;
        }
        .request-info {
            background: #dbeafe;
            border: 1px solid #93c5fd;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 30px;
        }
        .request-info h3 {
            margin: 0 0 10px 0;
            color: #1e40af;
            font-size: 1.2em;
        }
        .section {
            background: white;
            border-radius: 12px;
            padding: 25px;
            margin-bottom: 25px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .section h2 {
            margin: 0 0 20px 0;
            color: #1e293b;
            font-size: 1.5em;
            border-bottom: 2px solid #e2e8f0;
            padding-bottom: 10px;
        }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
        }
        .metric-card {
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            padding: 20px;
            text-align: center;
        }
        .metric-title {
            font-size: 0.9em;
            color: #64748b;
            margin-bottom: 8px;
            font-weight: 500;
        }
        .metric-value {
            font-size: 1.8em;
            font-weight: bold;
            color: #1e293b;
            margin-bottom: 5px;
        }
        .metric-change {
            font-size: 0.8em;
            color: #059669;
            font-weight: 500;
        }
        .insights-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 15px;
        }
        .insight-card {
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            padding: 18px;
        }
        .insight-text {
            color: #475569;
            line-height: 1.5;
        }
        .chart-section {
            text-align: center;
            padding: 20px;
            background: #f8fafc;
            border-radius: 8px;
            border: 1px solid #e2e8f0;
        }
        .chart-image {
            max-width: 100%;
            height: auto;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            margin: 20px 0;
        }
        .chart-placeholder {
            padding: 40px;
            background: #f1f5f9;
            border: 2px dashed #cbd5e1;
            border-radius: 8px;
            color: #64748b;
            font-style: italic;
        }
        .footer {
            text-align: center;
            margin-top: 40px;
            padding: 20px;
            color: #64748b;
            font-size: 0.9em;
            border-top: 1px solid #e2e8f0;
        }
        @media print {
            body { background-color: white; }
            .section { 
                box-shadow: none; 
                border: 1px solid #e2e8f0;
                page-break-inside: avoid;
            }
            .chart-image {
                max-height: 400px;
                page-break-inside: avoid;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>智能仪表盘报告</h1>
        <p>生成时间: ${timestamp}</p>
    </div>`;

            // 添加分析请求信息
            if (userRequestText) {
                htmlContent += `
    <div class="request-info">
        <h3>📊 分析请求</h3>
        <p>${userRequestText}</p>
    </div>`;
            }

            // 添加核心指标
            if (data?.metrics && Array.isArray(data.metrics) && data.metrics.length > 0) {
                htmlContent += `
    <div class="section">
        <h2>核心指标</h2>
        <div class="metrics-grid">`;
                data.metrics.forEach(metric => {
                    htmlContent += `
            <div class="metric-card">
                <div class="metric-title">${metric.title}</div>
                <div class="metric-value">${metric.value}</div>
                ${metric.change ? `<div class="metric-change">${metric.change}</div>` : ''}
            </div>`;
                });
                htmlContent += `
        </div>
    </div>`;
            }

            // 添加图表（改进版本，包含实际图片）
            if (activeChart) {
                htmlContent += `
    <div class="section">
        <h2>分析图表</h2>
        <div class="chart-section">`;
                
                if (chartImageData) {
                    htmlContent += `
            <img src="${chartImageData}" alt="分析图表" class="chart-image" />
            <p style="margin-top: 15px; color: #64748b; font-size: 0.9em;">
                图表类型: ${activeChart.type.toUpperCase()} | 
                导出时间: ${timestamp}
            </p>`;
                } else if (activeChart.type === 'image' && activeChart.data) {
                    // 处理已有的图片数据
                    htmlContent += `
            <img src="${activeChart.data}" alt="分析图表" class="chart-image" />
            <p style="margin-top: 15px; color: #64748b; font-size: 0.9em;">
                图表类型: ${activeChart.type.toUpperCase()} | 
                导出时间: ${timestamp}
            </p>`;
                } else {
                    // 无法获取图片时的占位符
                    htmlContent += `
            <div class="chart-placeholder">
                <p>📊 ${activeChart.type.toUpperCase()} 图表</p>
                <p>此图表为交互式内容，请在原系统中查看完整效果</p>
            </div>`;
                }
                
                htmlContent += `
        </div>
    </div>`;
            }

            // 添加自动洞察
            if (data?.insights && Array.isArray(data.insights) && data.insights.length > 0) {
                htmlContent += `
    <div class="section">
        <h2>自动洞察</h2>
        <div class="insights-grid">`;
                data.insights.forEach(insight => {
                    htmlContent += `
            <div class="insight-card">
                <div class="insight-text">${insight.text}</div>
            </div>`;
                });
                htmlContent += `
        </div>
    </div>`;
            }

            htmlContent += `
    <div class="footer">
        <p>本报告由 RapidBI 智能仪表盘生成</p>
        <p>如需查看交互式图表和实时数据，请访问原系统</p>
    </div>
</body>
</html>`;

            // 创建并下载文件
            const blob = new Blob([htmlContent], { type: 'text/html;charset=utf-8' });
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.download = `dashboard-report-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.html`;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            URL.revokeObjectURL(url);
            
            console.log("[Dashboard] HTML export completed successfully");
        } catch (error) {
            console.error("[Dashboard] HTML export failed:", error);
            alert('HTML导出失败，请重试');
        }
    };

    // 导出为PDF（改进版本，支持图表转图片）
    const exportAsPDF = async () => {
        try {
            // 获取图表图片（如果有ECharts）
            let chartImageData = null;
            if (activeChart && activeChart.type === 'echarts') {
                chartImageData = await captureEChartsAsImage();
            }

            // 创建一个新窗口用于打印
            const printWindow = window.open('', '_blank');
            if (!printWindow) {
                alert('请允许弹出窗口以完成PDF导出');
                return;
            }

            const timestamp = new Date().toLocaleString('zh-CN');
            let printContent = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>智能仪表盘报告 - ${timestamp}</title>
    <style>
        @page {
            margin: 20mm;
            size: A4;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #334155;
            margin: 0;
            padding: 0;
        }
        .header {
            text-align: center;
            border-bottom: 2px solid #3b82f6;
            padding-bottom: 20px;
            margin-bottom: 30px;
        }
        .header h1 {
            color: #3b82f6;
            margin: 0 0 10px 0;
            font-size: 2.2em;
        }
        .header p {
            color: #64748b;
            margin: 0;
        }
        .request-info {
            background: #f1f5f9;
            border-left: 4px solid #3b82f6;
            padding: 15px;
            margin-bottom: 25px;
        }
        .section {
            margin-bottom: 25px;
            page-break-inside: avoid;
        }
        .section h2 {
            color: #1e293b;
            border-bottom: 1px solid #e2e8f0;
            padding-bottom: 8px;
            margin-bottom: 15px;
        }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 15px;
            margin-bottom: 20px;
        }
        .metric-card {
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            padding: 15px;
            text-align: center;
        }
        .metric-title {
            font-size: 0.9em;
            color: #64748b;
            margin-bottom: 5px;
        }
        .metric-value {
            font-size: 1.5em;
            font-weight: bold;
            color: #1e293b;
            margin-bottom: 3px;
        }
        .metric-change {
            font-size: 0.8em;
            color: #059669;
        }
        .chart-section {
            text-align: center;
            margin: 20px 0;
            page-break-inside: avoid;
        }
        .chart-image {
            max-width: 100%;
            max-height: 400px;
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            margin: 15px 0;
        }
        .chart-placeholder {
            padding: 30px;
            background: #f8fafc;
            border: 2px dashed #cbd5e1;
            border-radius: 6px;
            color: #64748b;
            font-style: italic;
            margin: 15px 0;
        }
        .insight-card {
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            padding: 12px;
            margin-bottom: 10px;
        }
        .insight-text {
            color: #475569;
            line-height: 1.4;
        }
        .footer {
            text-align: center;
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #e2e8f0;
            color: #64748b;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>智能仪表盘报告</h1>
        <p>生成时间: ${timestamp}</p>
    </div>`;

            // 添加分析请求信息
            if (userRequestText) {
                printContent += `
    <div class="request-info">
        <h3>📊 分析请求</h3>
        <p>${userRequestText}</p>
    </div>`;
            }

            // 添加核心指标
            if (data?.metrics && Array.isArray(data.metrics) && data.metrics.length > 0) {
                printContent += `
    <div class="section">
        <h2>核心指标</h2>
        <div class="metrics-grid">`;
                data.metrics.forEach(metric => {
                    printContent += `
            <div class="metric-card">
                <div class="metric-title">${metric.title}</div>
                <div class="metric-value">${metric.value}</div>
                ${metric.change ? `<div class="metric-change">${metric.change}</div>` : ''}
            </div>`;
                });
                printContent += `
        </div>
    </div>`;
            }

            // 添加图表（改进版本，包含实际图片）
            if (activeChart) {
                printContent += `
    <div class="section">
        <h2>分析图表</h2>
        <div class="chart-section">`;
                
                if (chartImageData) {
                    printContent += `
            <img src="${chartImageData}" alt="分析图表" class="chart-image" />
            <p style="margin-top: 10px; color: #64748b; font-size: 0.9em;">
                图表类型: ${activeChart.type.toUpperCase()} | 导出时间: ${timestamp}
            </p>`;
                } else if (activeChart.type === 'image' && activeChart.data) {
                    // 处理已有的图片数据
                    printContent += `
            <img src="${activeChart.data}" alt="分析图表" class="chart-image" />
            <p style="margin-top: 10px; color: #64748b; font-size: 0.9em;">
                图表类型: ${activeChart.type.toUpperCase()} | 导出时间: ${timestamp}
            </p>`;
                } else {
                    // 无法获取图片时的占位符
                    printContent += `
            <div class="chart-placeholder">
                <p>📊 ${activeChart.type.toUpperCase()} 图表</p>
                <p>此图表为交互式内容，请在原系统中查看完整效果</p>
            </div>`;
                }
                
                printContent += `
        </div>
    </div>`;
            }

            // 添加自动洞察
            if (data?.insights && Array.isArray(data.insights) && data.insights.length > 0) {
                printContent += `
    <div class="section">
        <h2>自动洞察</h2>`;
                data.insights.forEach(insight => {
                    printContent += `
        <div class="insight-card">
            <div class="insight-text">${insight.text}</div>
        </div>`;
                });
                printContent += `
    </div>`;
            }

            printContent += `
    <div class="footer">
        <p>本报告由 RapidBI 智能仪表盘生成</p>
        <p>如需查看交互式图表和实时数据，请访问原系统</p>
    </div>
</body>
</html>`;

            // 写入打印窗口并触发打印
            printWindow.document.write(printContent);
            printWindow.document.close();
            
            // 等待内容加载完成后打印
            printWindow.onload = () => {
                setTimeout(() => {
                    printWindow.print();
                    printWindow.close();
                }, 1000); // 增加延迟确保图片加载完成
            };
            
            console.log("[Dashboard] PDF export initiated successfully");
        } catch (error) {
            console.error("[Dashboard] PDF export failed:", error);
            alert('PDF导出失败，请重试');
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
            console.log("[Dashboard] renderChart: activeChart is null");
            return null;
        }

        console.log("[Dashboard] renderChart called with activeChart:", activeChart);
        console.log("[Dashboard] activeChart.type:", activeChart.type);
        console.log("[Dashboard] activeChart.chartData:", activeChart.chartData);

        // Extract charts array if chartData is available (multi-chart support)
        const charts = activeChart.chartData?.charts || [];
        console.log("[Dashboard] Extracted charts array:", charts);
        console.log("[Dashboard] Charts length:", charts.length);

        const hasMultipleCharts = charts.length > 1;

        // Use chart from charts array if available, otherwise fall back to single chart (activeChart.data)
        const currentChart = charts.length > 0 ? charts[currentChartIndex] : null;
        const chartType = currentChart ? currentChart.type : activeChart.type;
        const chartData = currentChart ? currentChart.data : activeChart.data;

        console.log("[Dashboard] currentChart:", currentChart);
        console.log("[Dashboard] chartType:", chartType);
        console.log("[Dashboard] chartData (first 100 chars):", typeof chartData === 'string' ? chartData.substring(0, 100) : chartData);

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
                            <span className="bg-white/90 px-3 py-1 rounded-full text-xs font-medium text-slate-600 shadow-sm backdrop-blur-sm">Double click to zoom</span>
                        </div>
                    </div>
                );
            }

            if (chartType === 'echarts') {
                try {
                    const options = JSON.parse(chartData);
                    
                    // 验证ECharts选项的基本结构
                    if (!options || typeof options !== 'object') {
                        console.error("Invalid ECharts options: not an object", options);
                        return null;
                    }
                    
                    // 确保必要的属性存在
                    const validatedOptions = {
                        ...options,
                        // 确保有基本的配置
                        animation: options.animation !== false,
                        // 如果没有series，添加一个空的
                        series: options.series || []
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
                    console.error("Failed to parse ECharts options for dashboard", e);
                    console.error("Raw chart data:", chartData);
                    return null;
                }
            }

            if (chartType === 'table') {
                const tableData = chartData as any[];
                if (!tableData || !Array.isArray(tableData) || tableData.length === 0) return null;

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

        return (
            <div className="mt-6 space-y-4">
                <h3 className="text-md font-semibold text-slate-700 flex items-center gap-2">
                    <Table className="w-5 h-5 text-blue-500" />
                    {t('analysis_data') || 'Analysis Data'}
                </h3>
                {tableCharts.map((chart, tableIndex) => {
                    try {
                        const tableData = JSON.parse(chart.data);
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
                        console.error("Failed to parse table data", e);
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

        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = filename;
        link.click();
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
        if (insight.data_source_id) {
            // Directly trigger new chat session creation
            // If there's a conflict, ChatSidebar will show the warning modal
            EventsEmit('start-new-chat', {
                dataSourceId: insight.data_source_id,
                sessionName: `${t('analysis_session_prefix')}${insight.source_name || insight.text}`,
                keepChatOpen: true // 标记这是创建新会话，不要隐藏聊天区
            });
        } else {
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
                                        onClick={exportAsHTML}
                                        className="w-full flex items-center gap-3 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                                    >
                                        <FileText className="w-4 h-4 text-blue-600" />
                                        <span>导出为 HTML</span>
                                    </button>
                                    <button
                                        onClick={exportAsPDF}
                                        className="w-full flex items-center gap-3 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                                    >
                                        <FileImage className="w-4 h-4 text-red-600" />
                                        <span>导出为 PDF</span>
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
                                <span>{t('no_visualization_results') || 'This analysis request has no visualization results yet.'}</span>
                            </div>
                        )}
                    </div>
                )}
            </header>

            <div className="flex-1 overflow-y-auto px-6 pb-8">
                {activeChart && (
                    <section className="animate-in fade-in slide-in-from-top-4 duration-500">
                        <h2 className="text-lg font-semibold text-slate-700 mb-4">Latest Analysis</h2>
                        {renderChart()}
                    </section>
                )}

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
                                return currentChart?.type === 'echarts' ? JSON.parse(currentChart.data) : {};
                            }
                            return JSON.parse(activeChart.data);
                        })()}
                        onClose={() => setChartModalOpen(false)}
                    />
                )}

                {/* 核心指标区域 - 只有当存在有效指标时才显示 */}
                {data.metrics && Array.isArray(data.metrics) && data.metrics.length > 0 && (
                    <section className="mb-8">
                        <h2 className="text-lg font-semibold text-slate-700 mb-4">{t('key_metrics')}</h2>
                        <DashboardLayout>
                            {data.metrics.map((metric, index) => (
                                <MetricCard
                                    key={index}
                                    title={metric.title}
                                    value={metric.value}
                                    change={metric.change}
                                />
                            ))}
                        </DashboardLayout>
                    </section>
                )}

                {/* 自动洞察区域 - 只有当存在有效洞察时才显示 */}
                {data.insights && Array.isArray(data.insights) && data.insights.length > 0 && (
                    <section>
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
        </div>
    );
};

export default Dashboard;
