/**
 * Draggable Dashboard Component
 * 
 * 完整的可拖拽仪表盘，整合真实数据展示和拖拽功能
 * 使用新的统一数据系统 (useDashboardData Hook)
 */

import React, { useState, useEffect } from 'react';
import { Edit3, Lock, Unlock, Save, X, Download, FileText, Image, Table, FileSpreadsheet, ChevronLeft, ChevronRight, FileImage, Presentation } from 'lucide-react';
import MetricCard from './MetricCard';
import SmartInsight from './SmartInsight';
import DataTable from './DataTable';
import Chart from './Chart';
import ImageModal from './ImageModal';
import ChartModal from './ChartModal';
import Toast, { ToastType } from './Toast';
import { main } from '../../wailsjs/go/models';
import { useLanguage } from '../i18n';
import { SaveLayout, LoadLayout, SelectSaveFile, GetSessionFileAsBase64, DownloadSessionFile, GenerateCSVThumbnail, ExportDashboardToPDF, ExportDashboardToPPT, ExportSessionFilesToZip } from '../../wailsjs/go/main/App';
import { EventsEmit } from '../../wailsjs/runtime/runtime';
import { database } from '../../wailsjs/go/models';
import { createLogger } from '../utils/systemLog';
import { useDashboardData } from '../hooks/useDashboardData';
import { GlobalAnalysisStatus } from './GlobalAnalysisStatus';

const logger = createLogger('DraggableDashboard');

interface DraggableDashboardProps {
    data: main.DashboardData | null;  // 保留接口兼容性，但不再使用
    activeChart?: { type: 'echarts' | 'image' | 'table' | 'csv', data: any, chartData?: main.ChartData } | null;  // 保留接口兼容性，但不再使用
    userRequestText?: string | null;
    onDashboardClick?: () => void;
    isChatOpen?: boolean;
    activeThreadId?: string | null;
    sessionFiles?: main.SessionFile[];
    selectedMessageId?: string | null;
    onInsightClick?: (insight: any) => void;
}

interface LayoutItem {
    id: string;
    type: 'metric' | 'insight' | 'chart' | 'table' | 'image' | 'file_download';
    x: number;
    y: number;
    w: number;
    h: number;
    data: any;
}

const DraggableDashboard: React.FC<DraggableDashboardProps> = ({
    // data 和 activeChart 不再使用，改用 useDashboardData Hook
    userRequestText,
    onDashboardClick,
    isChatOpen,
    activeThreadId,
    sessionFiles,
    selectedMessageId,
    onInsightClick
}) => {
    const { t } = useLanguage();
    
    // 使用新的统一数据 Hook
    const dashboardData = useDashboardData();
    
    // 创建兼容变量，从新系统获取数据
    // 这样可以最小化对现有代码的修改
    const data = {
        metrics: dashboardData.metrics.map(m => ({
            title: m.title,
            value: m.value,
            change: m.change || ''
        })),
        insights: dashboardData.insights.map(i => ({
            text: i.text,
            icon: i.icon || 'lightbulb',
            data_source_id: i.dataSourceId,
            source_name: i.sourceName
        }))
    };
    
    // 构建兼容的 activeChart 对象
    const activeChart: { type: 'echarts' | 'image' | 'table' | 'csv', data: any, chartData?: any } | null = (() => {
        if (dashboardData.hasECharts) {
            return {
                type: 'echarts' as const,
                data: typeof dashboardData.echartsData === 'string' 
                    ? dashboardData.echartsData 
                    : JSON.stringify(dashboardData.echartsData),
                chartData: {
                    charts: [
                        ...dashboardData.allEChartsData.map(d => ({ type: 'echarts', data: typeof d === 'string' ? d : JSON.stringify(d) })),
                        ...dashboardData.images.map(img => ({ type: 'image', data: img })),
                        ...dashboardData.allTableData.map(t => ({ type: 'table', data: t.rows }))
                    ]
                }
            };
        }
        if (dashboardData.hasImages) {
            return {
                type: 'image' as const,
                data: dashboardData.images[0],
                chartData: {
                    charts: dashboardData.images.map(img => ({ type: 'image', data: img }))
                }
            };
        }
        if (dashboardData.hasTables && dashboardData.tableData) {
            return {
                type: 'table' as const,
                data: dashboardData.tableData.rows,
                chartData: {
                    charts: dashboardData.allTableData.map(t => ({ type: 'table', data: t.rows }))
                }
            };
        }
        return null;
    })();
    
    const [isEditMode, setIsEditMode] = useState(false);
    const [filePreviewsLoading, setFilePreviewsLoading] = useState<Record<string, boolean>>({});
    const [filePreviews, setFilePreviews] = useState<Record<string, string>>({});
    const [currentImageIndex, setCurrentImageIndex] = useState(0);
    
    // 图表/图片放大模态框状态
    const [imageModalOpen, setImageModalOpen] = useState(false);
    const [chartModalOpen, setChartModalOpen] = useState(false);
    const [modalImageUrl, setModalImageUrl] = useState<string>('');
    const [modalChartOptions, setModalChartOptions] = useState<any>(null);
    
    // 导出功能状态
    const [exportDropdownOpen, setExportDropdownOpen] = useState(false);
    const [toast, setToast] = useState<{ message: string; type: ToastType } | null>(null);

    // 点击外部关闭导出下拉菜单
    useEffect(() => {
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
        return dashboardData.hasMetrics || dashboardData.hasInsights || dashboardData.hasECharts || dashboardData.hasImages || dashboardData.hasTables;
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
            
            await ExportSessionFilesToZip(activeThreadId, selectedMessageId);
            
            setToast({ message: '数据文件导出成功！', type: 'success' });
        } catch (error) {
            console.error('[DraggableDashboard] Data files export failed:', error);
            setToast({
                message: '数据文件导出失败: ' + (error instanceof Error ? error.message : String(error)),
                type: 'error'
            });
        }
    };

    // 导出为PDF（使用后端chromedp生成）
    const exportAsPDF = async () => {
        try {
            setExportDropdownOpen(false);
            logger.debug('Starting PDF export...');

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

            // 方法1: 收集页面上所有ECharts组件
            const echartsComponents = document.querySelectorAll('.echarts-for-react');
            logger.debug(`Found ECharts components on page: ${echartsComponents.length}`);

            for (let i = 0; i < echartsComponents.length; i++) {
                try {
                    const component = echartsComponents[i] as any;
                    
                    if (component?.getEchartsInstance) {
                        const instance = component.getEchartsInstance();
                        if (instance) {
                            const dataURL = instance.getDataURL({
                                type: 'png',
                                pixelRatio: 4,
                                backgroundColor: '#fff'
                            });
                            chartImages.push(dataURL);
                            logger.debug(`Captured EChart ${i + 1}, size: ${dataURL.length} bytes`);
                        }
                    }
                } catch (e) {
                    console.error(`[DraggableDashboard] Failed to capture EChart ${i}:`, e);
                }
            }

            // 方法2: 尝试通过Canvas元素捕获（备用方案）
            if (chartImages.length === 0) {
                logger.debug('No ECharts captured via component method, trying Canvas fallback');
                const canvasElements = document.querySelectorAll('canvas');
                
                for (let i = 0; i < canvasElements.length; i++) {
                    const canvas = canvasElements[i];
                    const parent = canvas.parentElement;
                    
                    if (parent && (parent.classList.contains('echarts-for-react') || 
                                   parent.querySelector('.echarts-for-react') ||
                                   canvas.width > 200)) {
                        try {
                            const dataURL = canvas.toDataURL('image/png');
                            chartImages.push(dataURL);
                            logger.debug(`Captured canvas ${i + 1} as fallback, size: ${dataURL.length} bytes`);
                        } catch (e) {
                            console.error(`[DraggableDashboard] Failed to capture canvas ${i}:`, e);
                        }
                    }
                }
            }

            // 收集chartData.charts中的所有image类型
            if (activeChart?.chartData?.charts) {
                for (const chart of activeChart.chartData.charts) {
                    if (chart.type === 'image' && typeof chart.data === 'string' && chart.data.startsWith('data:image')) {
                        chartImages.push(chart.data);
                        logger.debug(`Added image from chartData, size: ${chart.data.length}`);
                    }
                }
            }

            // 也检查activeChart.data（直接图片）
            if (activeChart?.type === 'image' && typeof activeChart.data === 'string' && activeChart.data.startsWith('data:image')) {
                if (!chartImages.includes(activeChart.data)) {
                    chartImages.push(activeChart.data);
                    logger.debug(`Added direct image, size: ${activeChart.data.length}`);
                }
            }

            logger.debug(`Total images collected: ${chartImages.length}`);

            // 添加图表图片到导出数据
            if (chartImages.length > 0) {
                exportData.chartImages = chartImages;
            }

            // 收集表格数据
            if (activeChart?.chartData?.charts) {
                const tableCharts = activeChart.chartData.charts.filter(
                    (chart: any) => chart.type === 'table'
                );

                if (tableCharts.length > 0) {
                    const firstTable = tableCharts[0];
                    try {
                        let tableDataRaw = firstTable.data;

                        if (typeof tableDataRaw === 'string') {
                            tableDataRaw = tableDataRaw
                                .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                .replace(/,(\s*[}\]])/g, '$1');
                            tableDataRaw = JSON.parse(tableDataRaw);
                        }

                        if (Array.isArray(tableDataRaw) && tableDataRaw.length > 0) {
                            const columns = Object.keys(tableDataRaw[0]).map(key => ({
                                title: key,
                                dataType: 'string'
                            }));

                            const rows = tableDataRaw.map((row: any) =>
                                Object.values(row).map(v => v === null || v === undefined ? '' : v)
                            );

                            exportData.tableData = {
                                columns: columns,
                                data: rows
                            };

                            logger.debug(`Table data extracted: ${columns.length} columns, ${rows.length} rows`);
                        }
                    } catch (e) {
                        console.error('[DraggableDashboard] Failed to parse table data:', e);
                    }
                }
            }

            logger.debug(`Export data prepared: metrics=${exportData.metrics.length}, insights=${exportData.insights.length}`);

            // 调用后端API生成PDF
            await ExportDashboardToPDF(exportData);

            logger.debug('PDF export completed successfully');
            setToast({ message: 'PDF导出成功！', type: 'success' });
        } catch (error) {
            console.error('[DraggableDashboard] PDF export failed:', error);
            setToast({
                message: 'PDF导出失败: ' + (error instanceof Error ? error.message : String(error)),
                type: 'error'
            });
        }
    };

    // 导出为PPT
    const exportAsPPT = async () => {
        try {
            setExportDropdownOpen(false);
            logger.debug('Starting PPT export...');

            // 收集仪表盘数据（与PDF导出相同的逻辑）
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

            // 收集页面上所有ECharts组件
            const echartsComponents = document.querySelectorAll('.echarts-for-react');
            logger.debug(`Found ECharts components on page: ${echartsComponents.length}`);

            for (let i = 0; i < echartsComponents.length; i++) {
                try {
                    const component = echartsComponents[i] as any;
                    
                    if (component?.getEchartsInstance) {
                        const instance = component.getEchartsInstance();
                        if (instance) {
                            const dataURL = instance.getDataURL({
                                type: 'png',
                                pixelRatio: 4,
                                backgroundColor: '#fff'
                            });
                            chartImages.push(dataURL);
                            logger.debug(`Captured EChart ${i + 1}, size: ${dataURL.length} bytes`);
                        }
                    }
                } catch (e) {
                    console.error(`[DraggableDashboard] Failed to capture EChart ${i}:`, e);
                }
            }

            // 备用方案：通过Canvas元素捕获
            if (chartImages.length === 0) {
                logger.debug('No ECharts captured via component method, trying Canvas fallback');
                const canvasElements = document.querySelectorAll('canvas');
                
                for (let i = 0; i < canvasElements.length; i++) {
                    const canvas = canvasElements[i];
                    const parent = canvas.parentElement;
                    
                    if (parent && (parent.classList.contains('echarts-for-react') || 
                                   parent.querySelector('.echarts-for-react') ||
                                   canvas.width > 200)) {
                        try {
                            const dataURL = canvas.toDataURL('image/png');
                            chartImages.push(dataURL);
                            logger.debug(`Captured canvas ${i + 1} as fallback, size: ${dataURL.length} bytes`);
                        } catch (e) {
                            console.error(`[DraggableDashboard] Failed to capture canvas ${i}:`, e);
                        }
                    }
                }
            }

            // 收集chartData.charts中的所有image类型
            if (activeChart?.chartData?.charts) {
                for (const chart of activeChart.chartData.charts) {
                    if (chart.type === 'image' && typeof chart.data === 'string' && chart.data.startsWith('data:image')) {
                        chartImages.push(chart.data);
                        logger.debug(`Added image from chartData, size: ${chart.data.length}`);
                    }
                }
            }

            // 也检查activeChart.data（直接图片）
            if (activeChart?.type === 'image' && typeof activeChart.data === 'string' && activeChart.data.startsWith('data:image')) {
                if (!chartImages.includes(activeChart.data)) {
                    chartImages.push(activeChart.data);
                    logger.debug(`Added direct image, size: ${activeChart.data.length}`);
                }
            }

            logger.debug(`Total images collected: ${chartImages.length}`);

            // 添加图表图片到导出数据
            if (chartImages.length > 0) {
                exportData.chartImages = chartImages;
            }

            // 收集表格数据
            if (activeChart?.chartData?.charts) {
                const tableCharts = activeChart.chartData.charts.filter(
                    (chart: any) => chart.type === 'table'
                );

                if (tableCharts.length > 0) {
                    const firstTable = tableCharts[0];
                    try {
                        let tableDataRaw = firstTable.data;

                        if (typeof tableDataRaw === 'string') {
                            tableDataRaw = tableDataRaw
                                .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                                .replace(/,(\s*[}\]])/g, '$1');
                            tableDataRaw = JSON.parse(tableDataRaw);
                        }

                        if (Array.isArray(tableDataRaw) && tableDataRaw.length > 0) {
                            const columns = Object.keys(tableDataRaw[0]).map(key => ({
                                title: key,
                                dataType: 'string'
                            }));

                            const rows = tableDataRaw.map((row: any) =>
                                Object.values(row).map(v => v === null || v === undefined ? '' : v)
                            );

                            exportData.tableData = {
                                columns: columns,
                                data: rows
                            };

                            logger.debug(`Table data extracted: ${columns.length} columns, ${rows.length} rows`);
                        }
                    } catch (e) {
                        console.error('[DraggableDashboard] Failed to parse table data:', e);
                    }
                }
            }

            logger.debug(`Export data prepared: metrics=${exportData.metrics.length}, insights=${exportData.insights.length}`);

            // 调用后端API生成PPT
            await ExportDashboardToPPT(exportData);

            logger.debug('PPT export completed successfully');
            setToast({ message: 'PPT导出成功！', type: 'success' });
        } catch (error) {
            console.error('[DraggableDashboard] PPT export failed:', error);
            setToast({
                message: 'PPT导出失败: ' + (error instanceof Error ? error.message : String(error)),
                type: 'error'
            });
        }
    };

    // 双击图表放大显示
    const handleChartDoubleClick = () => {
        if (!activeChart) return;
        
        if (activeChart.type === 'echarts' && typeof activeChart.data === 'string') {
            try {
                const options = JSON.parse(activeChart.data);
                setModalChartOptions(options);
                setChartModalOpen(true);
            } catch (e) {
                console.error('Failed to parse chart options:', e);
            }
        }
    };

    // 双击图片放大显示
    const handleImageDoubleClick = (imageUrl: string) => {
        setModalImageUrl(imageUrl);
        setImageModalOpen(true);
    };

    // 点击洞察项，传递到父组件处理
    // 不清空当前显示数据，保持仪表盘内容稳定
    const handleInsightClick = (insight: any) => {
        // 优先使用回调函数，由父组件统一管理分析请求
        // 传递完整的洞察对象，包含 data_source_id 等信息
        if (onInsightClick) {
            onInsightClick(insight);
        } else if (activeThreadId) {
            // 降级方案：直接通过事件发送分析请求
            const insightText = typeof insight === 'string' ? insight : insight.text;
            EventsEmit('chat-send-message-in-session', {
                text: `请深入分析：${insightText}`,
                threadId: activeThreadId
            });
        }
    };

    // 获取文件图标
    const getFileIcon = (fileName: string, fileType: string) => {
        const ext = fileName.split('.').pop()?.toLowerCase() || '';
        if (fileType === 'image' || ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg'].includes(ext)) {
            return <Image size={20} className="text-cyan-600" />;
        }
        if (['csv', 'xlsx', 'xls'].includes(ext)) {
            return <FileSpreadsheet size={20} className="text-green-600" />;
        }
        if (['json', 'xml'].includes(ext)) {
            return <Table size={20} className="text-amber-600" />;
        }
        return <FileText size={20} className="text-orange-600" />;
    };

    // 获取文件预览（图片或CSV预览）- 参考Dashboard的做法
    const loadFilePreview = async (file: main.SessionFile) => {
        if (!activeThreadId || filePreviews[file.path] || filePreviewsLoading[file.path]) return;
        
        const ext = file.name.split('.').pop()?.toLowerCase() || '';
        const isImage = file.type === 'image' || ['png', 'jpg', 'jpeg', 'gif', 'webp'].includes(ext);
        const isCsv = file.type === 'csv' || ['csv'].includes(ext);
        
        // 只处理图片和CSV文件
        if (!isImage && !isCsv) return;
        
        setFilePreviewsLoading(prev => ({ ...prev, [file.path]: true }));
        try {
            if (isImage) {
                // 图片：直接获取base64数据
                const base64Data = await GetSessionFileAsBase64(activeThreadId, file.name);
                if (base64Data) {
                    // GetSessionFileAsBase64 返回的已经是完整的 data:image/xxx;base64,xxx 格式
                    setFilePreviews(prev => ({ ...prev, [file.path]: base64Data }));
                }
            } else if (isCsv) {
                // CSV：生成预览缩略图
                const base64Data = await GenerateCSVThumbnail(activeThreadId, file.name);
                if (base64Data) {
                    setFilePreviews(prev => ({ ...prev, [file.path]: base64Data }));
                }
            }
        } catch (error) {
            console.error('Failed to load file preview:', error);
        } finally {
            setFilePreviewsLoading(prev => ({ ...prev, [file.path]: false }));
        }
    };

    // 加载所有文件预览
    useEffect(() => {
        if (sessionFiles && activeThreadId) {
            sessionFiles.forEach(file => loadFilePreview(file));
        }
    }, [sessionFiles, activeThreadId]);

    // 重置图片索引当activeChart变化时
    useEffect(() => {
        setCurrentImageIndex(0);
    }, [activeChart]);

    // 下载文件
    const handleFileDownload = async (file: main.SessionFile) => {
        if (!activeThreadId) return;
        
        try {
            // 弹出保存对话框
            const ext = file.name.split('.').pop() || '*';
            const savePath = await SelectSaveFile(file.name, `*.${ext}`);
            
            if (savePath) {
                // 下载文件到指定路径
                await DownloadSessionFile(activeThreadId, file.name);
            }
        } catch (error) {
            console.error('Failed to download file:', error);
        }
    };
    
    // 组件最小高度配置（非编辑模式下的基础高度）
    const MIN_HEIGHTS: Record<string, number> = {
        metric: 60,
        chart: 80,
        insight: 56,
        table: 56,
        image: 60,
        file_download: 56
    };

    // 编辑模式下的高度（增加20%）
    const EDIT_MODE_HEIGHTS: Record<string, number> = {
        metric: Math.round(60 * 1.2),      // 72
        chart: Math.round(80 * 1.2),       // 96
        insight: Math.round(56 * 1.2),     // 67
        table: Math.round(56 * 1.2),       // 67
        image: Math.round(60 * 1.2),       // 72
        file_download: Math.round(56 * 1.2) // 67
    };
    
    // 默认布局：编辑模式下显示所有可用占位组件（使用编辑模式高度）
    const defaultLayout: LayoutItem[] = [
        { id: 'metric-area', type: 'metric', x: 0, y: 0, w: 100, h: EDIT_MODE_HEIGHTS.metric, data: null },
        { id: 'chart-area', type: 'chart', x: 0, y: 90, w: 100, h: EDIT_MODE_HEIGHTS.chart, data: null },
        { id: 'insight-area', type: 'insight', x: 0, y: 200, w: 100, h: EDIT_MODE_HEIGHTS.insight, data: null },
        { id: 'table-area', type: 'table', x: 0, y: 290, w: 50, h: EDIT_MODE_HEIGHTS.table, data: null },
        { id: 'image-area', type: 'image', x: 52, y: 290, w: 24, h: EDIT_MODE_HEIGHTS.image, data: null },
        { id: 'file_download-area', type: 'file_download', x: 78, y: 290, w: 22, h: EDIT_MODE_HEIGHTS.file_download, data: null },
    ];
    const [layout, setLayout] = useState<LayoutItem[]>(defaultLayout);
    const [draggedItem, setDraggedItem] = useState<string | null>(null);
    const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 });
    // 新增：调整大小状态
    const [resizingItem, setResizingItem] = useState<string | null>(null);
    const [resizeStart, setResizeStart] = useState({ x: 0, y: 0, w: 0, h: 0 });

    // 检查某种类型的组件是否有数据
    const hasDataForType = (type: string): boolean => {
        switch (type) {
            case 'metric':
                return !!(data?.metrics && Array.isArray(data.metrics) && data.metrics.length > 0);
            case 'insight':
                return !!(data?.insights && Array.isArray(data.insights) && data.insights.length > 0);
            case 'chart':
                // 图表组件：只有当 activeChart 是 echarts 类型时才显示
                return !!(activeChart?.type === 'echarts' && activeChart?.data);
            case 'table':
                // 表格数据：检查多种来源
                // 调试日志
                import('../utils/systemLog').then(({ SystemLog }) => {
                    SystemLog.debug('DraggableDashboard', `[hasDataForType:table] activeChart?.type: ${activeChart?.type}`);
                    SystemLog.debug('DraggableDashboard', `[hasDataForType:table] activeChart?.data type: ${typeof activeChart?.data}, isArray: ${Array.isArray(activeChart?.data)}`);
                    if (Array.isArray(activeChart?.data)) {
                        SystemLog.debug('DraggableDashboard', `[hasDataForType:table] activeChart?.data length: ${activeChart?.data?.length}`);
                    }
                });
                
                // 1. activeChart.type === 'table' 或 'csv'，data可以是数组或字符串
                if ((activeChart?.type === 'table' || activeChart?.type === 'csv') && activeChart?.data) {
                    // 检查data是否有效（数组或非空字符串）
                    if (Array.isArray(activeChart.data) && activeChart.data.length > 0) {
                        return true;
                    }
                    if (typeof activeChart.data === 'string' && activeChart.data.length > 0) {
                        return true;
                    }
                }
                // 2. chartData.charts 数组中有 table 类型
                if (activeChart?.chartData?.charts) {
                    const hasTableChart = activeChart.chartData.charts.some(
                        (chart: any) => chart.type === 'table' && chart.data
                    );
                    if (hasTableChart) return true;
                }
                return false;
            case 'image':
                // 图片组件：检查多种来源
                // 1. activeChart.type === 'image'
                if (activeChart?.type === 'image' && activeChart?.data) {
                    return true;
                }
                // 2. chartData.charts 数组中有 image 类型
                if (activeChart?.chartData?.charts) {
                    const hasImageChart = activeChart.chartData.charts.some(
                        (chart: any) => chart.type === 'image' && chart.data
                    );
                    if (hasImageChart) return true;
                }
                return false;
            case 'file_download':
                // 文件下载：只显示与当前选中消息关联的文件
                if (!sessionFiles || sessionFiles.length === 0 || !selectedMessageId) return false;
                return sessionFiles.some(file => file.message_id === selectedMessageId);
            default:
                return false;
        }
    };

    // 获取用于显示的布局（非编辑模式下过滤没有数据的组件）
    const getDisplayLayout = (): LayoutItem[] => {
        if (isEditMode) {
            // 编辑模式：显示所有组件
            return layout;
        }
        // 非编辑模式：只显示有数据的组件
        return layout.filter(item => hasDataForType(item.type));
    };

    // 初始化时使用默认布局（包含所有组件类型）
    // 不再根据数据自动生成布局，而是保持用户编辑的布局
    // 数据变化时不改变布局，只影响非编辑模式下的显示过滤

    // 加载保存的布局（保留所有组件类型，不过滤）
    useEffect(() => {
        const loadSavedLayout = async () => {
            try {
                const savedLayout = await LoadLayout('default-user');
                if (savedLayout && savedLayout.items && savedLayout.items.length > 0) {
                    // 转换保存的布局到我们的格式，并去重同类型控件
                    // 不过滤没有数据的组件，保留完整布局供编辑模式使用
                    const seenTypes = new Set<string>();
                    const convertedLayout: LayoutItem[] = [];
                    
                    for (const item of savedLayout.items) {
                        const type = item.i.split('-')[0] as LayoutItem['type'];
                        
                        // 每种类型只保留第一个
                        if (seenTypes.has(type)) continue;
                        seenTypes.add(type);
                        
                        const minH = MIN_HEIGHTS[type] || 56;
                        convertedLayout.push({
                            id: `${type}-area`, // 统一使用 type-area 格式
                            type: type,
                            x: item.x,
                            y: item.y,
                            w: item.w,
                            h: Math.max(item.h, minH),
                            data: null // 数据从全局获取
                        });
                    }
                    
                    if (convertedLayout.length > 0) {
                        setLayout(convertedLayout);
                    }
                }
            } catch (error) {
                console.error('Failed to load layout:', error);
            }
        };

        loadSavedLayout();
    }, []);

    // 保存布局
    const handleSaveLayout = async () => {
        try {
            const layoutConfig = new database.LayoutConfiguration({
                id: '',
                userId: 'default-user',
                isLocked: !isEditMode,
                items: layout.map(item => ({
                    i: item.id,
                    x: item.x,
                    y: item.y,
                    w: item.w,
                    h: item.h,
                    minW: 20,
                    minH: 60,
                    maxW: 100,
                    maxH: 800
                })),
                createdAt: Date.now(),
                updatedAt: Date.now()
            });

            await SaveLayout(layoutConfig);
            console.log('Layout saved successfully');
        } catch (error) {
            console.error('Failed to save layout:', error);
        }
    };

    // 开始拖拽
    const handleDragStart = (e: React.MouseEvent, itemId: string) => {
        if (!isEditMode) return;
        
        const item = layout.find(i => i.id === itemId);
        if (!item) return;

        const rect = (e.target as HTMLElement).getBoundingClientRect();
        setDragOffset({
            x: e.clientX - rect.left,
            y: e.clientY - rect.top
        });
        setDraggedItem(itemId);
    };

    // 拖拽中
    const handleDrag = (e: MouseEvent | React.MouseEvent) => {
        if (!draggedItem || !isEditMode) return;

        const container = document.getElementById('dashboard-container');
        if (!container) return;

        const containerRect = container.getBoundingClientRect();
        const newX = ((e.clientX - containerRect.left - dragOffset.x) / containerRect.width) * 100;
        const newY = e.clientY - containerRect.top - dragOffset.y;

        setLayout(prev => prev.map(item => 
            item.id === draggedItem
                ? { ...item, x: Math.max(0, Math.min(100 - item.w, newX)), y: Math.max(0, newY) }
                : item
        ));
    };

    // 结束拖拽
    const handleDragEnd = () => {
        // 如果是调整大小结束，重新排列布局
        if (resizingItem) {
            setLayout(prev => autoArrangeLayout(prev));
        }
        setDraggedItem(null);
        setResizingItem(null);
        if (isEditMode) {
            handleSaveLayout();
        }
    };

    // 自动排列布局 - 当组件大小改变时，其他组件自动调整位置
    const autoArrangeLayout = (currentLayout: LayoutItem[]): LayoutItem[] => {
        if (currentLayout.length === 0) return currentLayout;

        // 按y坐标排序，然后按x坐标排序
        const sortedItems = [...currentLayout].sort((a, b) => {
            if (Math.abs(a.y - b.y) < 20) { // 同一行（y差距小于20px）
                return a.x - b.x;
            }
            return a.y - b.y;
        });

        const arrangedItems: LayoutItem[] = [];
        let currentRowY = 0;
        let currentRowX = 0;
        let currentRowMaxHeight = 0;
        const gap = 2; // 组件间距（百分比）
        const verticalGap = 10; // 垂直间距（像素）

        for (const item of sortedItems) {
            // 检查当前行是否能放下这个组件
            if (currentRowX + item.w > 100) {
                // 换行
                currentRowY += currentRowMaxHeight + verticalGap;
                currentRowX = 0;
                currentRowMaxHeight = 0;
            }

            // 放置组件
            arrangedItems.push({
                ...item,
                x: currentRowX,
                y: currentRowY
            });

            // 更新当前行状态
            currentRowX += item.w + gap;
            currentRowMaxHeight = Math.max(currentRowMaxHeight, item.h);
        }

        return arrangedItems;
    };

    // 开始调整大小
    const handleResizeStart = (e: React.MouseEvent, itemId: string) => {
        if (!isEditMode) return;
        e.stopPropagation();
        e.preventDefault();
        
        const item = layout.find(i => i.id === itemId);
        if (!item) return;

        setResizeStart({
            x: e.clientX,
            y: e.clientY,
            w: item.w,
            h: item.h
        });
        setResizingItem(itemId);
    };

    // 调整大小中
    const handleResize = (e: MouseEvent) => {
        if (!resizingItem || !isEditMode) return;

        const container = document.getElementById('dashboard-container');
        if (!container) return;

        const containerRect = container.getBoundingClientRect();
        const deltaX = e.clientX - resizeStart.x;
        const deltaY = e.clientY - resizeStart.y;

        // 计算新的宽度（百分比）和高度（像素）
        const deltaWPercent = (deltaX / containerRect.width) * 100;
        const newW = Math.max(15, Math.min(100, resizeStart.w + deltaWPercent)); // 最小15%，最大100%
        const newH = Math.max(40, resizeStart.h + deltaY); // 最小40px

        setLayout(prev => prev.map(item => 
            item.id === resizingItem
                ? { ...item, w: newW, h: newH }
                : item
        ));
    };

    // 添加全局拖拽和调整大小事件监听
    useEffect(() => {
        if (!isEditMode || (!draggedItem && !resizingItem)) return;

        const handleMouseMove = (e: MouseEvent) => {
            if (draggedItem) {
                handleDrag(e);
            } else if (resizingItem) {
                handleResize(e);
            }
        };
        const handleMouseUp = () => handleDragEnd();

        // 添加到 document 级别，这样即使鼠标移出仪表盘区域也能继续操作
        document.addEventListener('mousemove', handleMouseMove);
        document.addEventListener('mouseup', handleMouseUp);

        return () => {
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', handleMouseUp);
        };
    }, [isEditMode, draggedItem, resizingItem, dragOffset, resizeStart]);

    // 渲染组件
    const renderComponent = (item: LayoutItem) => {
        // 获取该类型组件的最小高度
        const minH = MIN_HEIGHTS[item.type] || 56;
        
        // 根据数据量计算实际高度（非编辑模式）
        const calculateAutoHeight = (): number => {
            if (isEditMode) {
                // 编辑模式使用用户设置的高度
                return Math.max(item.h, minH);
            }
            
            const titleBarHeight = 32; // 标题条高度
            const padding = 16; // 内边距
            const itemHeight = 80; // 单个项目高度
            const gap = 8; // 项目间距
            
            switch (item.type) {
                case 'metric': {
                    // 关键指标：4列布局
                    const metricsCount = data?.metrics?.length || 0;
                    if (metricsCount === 0) return minH;
                    const cols = 4;
                    const rows = Math.ceil(metricsCount / cols);
                    return titleBarHeight + padding + rows * (itemHeight + gap);
                }
                case 'insight': {
                    // 自动洞察：3列布局
                    const insightsCount = data?.insights?.length || 0;
                    if (insightsCount === 0) return minH;
                    const cols = 3;
                    const rows = Math.ceil(insightsCount / cols);
                    const insightItemHeight = 100; // 洞察项目稍高
                    return titleBarHeight + padding + rows * (insightItemHeight + gap);
                }
                case 'chart': {
                    // 图表：根据是否有数据决定高度
                    if (!activeChart) return minH;
                    return Math.max(item.h, 300); // 图表最小300px
                }
                case 'table': {
                    // 表格：根据数据行数计算
                    if (!Array.isArray(item.data) || item.data.length === 0) return minH;
                    const rowHeight = 40;
                    const headerHeight = 48;
                    const maxRows = 10; // 最多显示10行
                    const displayRows = Math.min(item.data.length, maxRows);
                    return titleBarHeight + headerHeight + displayRows * rowHeight + padding;
                }
                case 'image': {
                    // 图片：保持用户设置或默认高度
                    return Math.max(item.h, 150);
                }
                case 'file_download': {
                    // 文件下载：根据与当前消息关联的文件数量计算
                    const filteredFiles = selectedMessageId 
                        ? sessionFiles?.filter(file => file.message_id === selectedMessageId) || []
                        : sessionFiles || [];
                    const filesCount = filteredFiles.length;
                    if (filesCount === 0) return minH;
                    const fileItemHeight = 48;
                    return titleBarHeight + padding + Math.min(filesCount, 5) * fileItemHeight;
                }
                default:
                    return Math.max(item.h, minH);
            }
        };
        
        const autoHeight = calculateAutoHeight();
        
        const style: React.CSSProperties = {
            position: 'absolute',
            left: `${item.x}%`,
            top: `${item.y}px`,
            width: `${item.w}%`,
            height: isEditMode ? `${Math.max(item.h, minH)}px` : 'auto', // 非编辑模式使用auto
            minHeight: `${minH}px`,
            cursor: isEditMode ? 'move' : 'default',
            transition: draggedItem === item.id ? 'none' : 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
            zIndex: draggedItem === item.id ? 1000 : 1
        };

        // 获取组件类型的中文名称和淡雅配色
        const getComponentInfo = (type: string) => {
            switch (type) {
                case 'metric':
                    return { 
                        name: '关键指标卡片', 
                        desc: '显示KPI数值',
                        icon: '📊',
                        bgColor: 'bg-blue-50',
                        borderColor: 'border-blue-200',
                        textColor: 'text-blue-700'
                    };
                case 'insight':
                    return { 
                        name: 'AI分析洞察', 
                        desc: '智能分析结论',
                        icon: '💡',
                        bgColor: 'bg-purple-50',
                        borderColor: 'border-purple-200',
                        textColor: 'text-purple-700'
                    };
                case 'chart':
                    return { 
                        name: '数据可视化图表', 
                        desc: '图形化展示数据',
                        icon: '📈',
                        bgColor: 'bg-green-50',
                        borderColor: 'border-green-200',
                        textColor: 'text-green-700'
                    };
                case 'table':
                    return { 
                        name: '数据表格', 
                        desc: '表格形式展示',
                        icon: '📋',
                        bgColor: 'bg-amber-50',
                        borderColor: 'border-amber-200',
                        textColor: 'text-amber-700'
                    };
                case 'image':
                    return { 
                        name: '图片展示区', 
                        desc: '显示图片内容',
                        icon: '🖼️',
                        bgColor: 'bg-cyan-50',
                        borderColor: 'border-cyan-200',
                        textColor: 'text-cyan-700'
                    };
                case 'file_download':
                case 'file':
                    return { 
                        name: '文件下载区', 
                        desc: '下载生成的文件',
                        icon: '📁',
                        bgColor: 'bg-orange-50',
                        borderColor: 'border-orange-200',
                        textColor: 'text-orange-700'
                    };
                default:
                    return { 
                        name: '通用组件', 
                        desc: '自定义内容',
                        icon: '📦',
                        bgColor: 'bg-gray-50',
                        borderColor: 'border-gray-200',
                        textColor: 'text-gray-700'
                    };
            }
        };

        const componentInfo = getComponentInfo(item.type);

        // 获取区域标题
        const getAreaTitle = (type: string) => {
            switch (type) {
                case 'metric': return '关键指标';
                case 'insight': return '自动洞察';
                case 'chart': return '数据图表';
                case 'table': return '数据表格';
                case 'image': return '图片';
                case 'file_download': return '文件下载';
                default: return '组件';
            }
        };

        const areaTitle = getAreaTitle(item.type);

        // 渲染多个指标（4列自动排布）
        const renderMultipleMetrics = () => {
            if (item.type !== 'metric' || !data?.metrics || !Array.isArray(data.metrics) || data.metrics.length === 0) {
                return null;
            }

            const metrics = data.metrics;
            const cols = Math.min(4, metrics.length); // 最多4列

            return (
                <div className="grid gap-2 p-2" style={{ 
                    gridTemplateColumns: `repeat(${cols}, 1fr)`,
                }}>
                    {metrics.map((metric: any, idx: number) => (
                        <div key={idx} className="bg-blue-50 rounded-lg border border-blue-100">
                            <MetricCard
                                title={metric.title || ''}
                                value={metric.value || ''}
                                change={metric.change || ''}
                            />
                        </div>
                    ))}
                </div>
            );
        };

        // 渲染多个洞察（3列自动排布）- 可点击发起分析
        const renderMultipleInsights = () => {
            if (item.type !== 'insight' || !data?.insights || !Array.isArray(data.insights) || data.insights.length === 0) {
                return null;
            }

            const insights = data.insights;
            const cols = Math.min(3, insights.length); // 最多3列

            return (
                <div className="grid gap-2 p-2" style={{ 
                    gridTemplateColumns: `repeat(${cols}, 1fr)`,
                }}>
                    {insights.map((insight: any, idx: number) => (
                        <div 
                            key={idx} 
                            className="bg-purple-50 rounded-lg p-3 border border-purple-100 cursor-pointer hover:bg-purple-100 hover:border-purple-300 hover:shadow-md transition-all group"
                            onClick={() => handleInsightClick(insight)}
                            title="点击深入分析此洞察"
                        >
                            <SmartInsight
                                text={insight.text || ''}
                                icon={insight.icon || 'lightbulb'}
                            />
                            <div className="mt-2 text-xs text-purple-500 opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
                                <span>🔍</span>
                                <span>点击深入分析</span>
                            </div>
                        </div>
                    ))}
                </div>
            );
        };

        const content = (() => {
            switch (item.type) {
                case 'metric':
                    // 单个指标数据时直接显示
                    if (item.data?.title) {
                        return (
                            <MetricCard
                                title={item.data?.title || ''}
                                value={item.data?.value || ''}
                                change={item.data?.change || ''}
                            />
                        );
                    }
                    return null;
                case 'insight':
                    // 单个洞察数据时直接显示
                    if (item.data?.text) {
                        return (
                            <SmartInsight
                                text={item.data?.text || ''}
                                icon={item.data?.icon || 'lightbulb'}
                            />
                        );
                    }
                    return null;
                case 'chart':
                    if (item.data?.type === 'echarts' && typeof item.data.data === 'string') {
                        try {
                            const options = JSON.parse(item.data.data);
                            return (
                                <div 
                                    className="cursor-zoom-in group/chart relative"
                                    onDoubleClick={(e) => {
                                        e.stopPropagation();
                                        setModalChartOptions(options);
                                        setChartModalOpen(true);
                                    }}
                                    title="双击放大查看"
                                >
                                    <Chart options={options} height={`${item.h - 32}px`} />
                                    <div className="absolute top-2 right-2 opacity-0 group-hover/chart:opacity-100 transition-opacity bg-black/50 text-white text-xs px-2 py-1 rounded">
                                        双击放大
                                    </div>
                                </div>
                            );
                        } catch (e) {
                            return <div className="text-red-500">Chart error</div>;
                        }
                    } else if (item.data?.type === 'image') {
                        return (
                            <div 
                                className="cursor-zoom-in group/img relative w-full h-full"
                                onDoubleClick={(e) => {
                                    e.stopPropagation();
                                    handleImageDoubleClick(item.data.data);
                                }}
                                title="双击放大查看"
                            >
                                <img 
                                    src={item.data.data} 
                                    alt="Chart" 
                                    className="w-full h-full object-contain"
                                />
                                <div className="absolute top-2 right-2 opacity-0 group-hover/img:opacity-100 transition-opacity bg-black/50 text-white text-xs px-2 py-1 rounded">
                                    双击放大
                                </div>
                            </div>
                        );
                    }
                    return null;
                case 'table':
                    if (Array.isArray(item.data)) {
                        return <DataTable data={item.data} />;
                    }
                    return null;
                default:
                    return null;
            }
        })();

        return (
            <div
                key={item.id}
                style={style}
                onMouseDown={(e) => handleDragStart(e, item.id)}
                className={`
                    group relative flex flex-col
                    ${isEditMode ? `ring-2 ring-offset-2 ${componentInfo.borderColor.replace('border-', 'ring-')} ring-opacity-50` : ''}
                    ${(draggedItem === item.id || resizingItem === item.id) ? 'opacity-90 shadow-2xl scale-[1.02]' : 'shadow-md'}
                    rounded-xl overflow-hidden bg-white
                    hover:shadow-xl transition-all duration-200
                `}
            >
                {/* 区域标题条 - 自动显示 */}
                <div className={`
                    flex-shrink-0 px-3 py-1.5 flex items-center justify-between
                    ${componentInfo.bgColor} ${componentInfo.borderColor}
                    border-b
                `}>
                    <div className="flex items-center gap-1.5">
                        <span className="text-base">{componentInfo.icon}</span>
                        <span className={`text-sm font-medium ${componentInfo.textColor}`}>{areaTitle}</span>
                    </div>
                    
                    {/* 编辑模式下的删除按钮 */}
                    {isEditMode && (
                        <button
                            onClick={(e) => {
                                e.stopPropagation();
                                setLayout(prev => prev.filter(i => i.id !== item.id));
                            }}
                            className={`
                                ${componentInfo.textColor}
                                hover:text-red-600
                                p-0.5 rounded transition-colors
                            `}
                            title="删除组件"
                        >
                            <X size={14} />
                        </button>
                    )}
                </div>

                {/* 编辑模式下显示占位提示（当没有实际内容时） - 排除metric和insight */}
                {isEditMode && !content && item.type !== 'insight' && item.type !== 'metric' && (
                    <div className={`
                        flex-1 flex flex-col items-center justify-center
                        ${componentInfo.bgColor} bg-opacity-30
                        border-2 border-dashed ${componentInfo.borderColor} rounded-b-xl m-1
                    `}>
                        <span className={`text-xs ${componentInfo.textColor} opacity-70`}>
                            {componentInfo.desc}
                        </span>
                    </div>
                )}

                {/* 关键指标区域特殊处理：编辑模式显示提示，非编辑模式自动4列排布 */}
                {item.type === 'metric' && (
                    <>
                        {isEditMode && !data?.metrics?.length && (
                            <div className={`
                                flex-1 flex flex-col items-center justify-center
                                ${componentInfo.bgColor} bg-opacity-30
                                border-2 border-dashed ${componentInfo.borderColor} rounded-b-xl m-1
                            `}>
                                <span className={`text-xs ${componentInfo.textColor} opacity-70`}>
                                    多个指标将按4列自动排布
                                </span>
                            </div>
                        )}
                        {!isEditMode && renderMultipleMetrics()}
                        {isEditMode && data?.metrics?.length > 0 && (
                            <div className="flex-1 p-2 text-center text-sm text-blue-600">
                                已有 {data.metrics.length} 个指标，将按4列自动排布
                            </div>
                        )}
                    </>
                )}

                {/* 洞察区域特殊处理：编辑模式显示提示，非编辑模式自动排布多个洞察 */}
                {item.type === 'insight' && (
                    <>
                        {isEditMode && !data?.insights?.length && (
                            <div className={`
                                flex-1 flex flex-col items-center justify-center
                                ${componentInfo.bgColor} bg-opacity-30
                                border-2 border-dashed ${componentInfo.borderColor} rounded-b-xl m-1
                            `}>
                                <span className={`text-xs ${componentInfo.textColor} opacity-70`}>
                                    多个洞察将自动排布在此区域
                                </span>
                            </div>
                        )}
                        {!isEditMode && renderMultipleInsights()}
                        {isEditMode && data?.insights?.length > 0 && (
                            <div className="flex-1 p-2 text-center text-sm text-purple-600">
                                已有 {data.insights.length} 条洞察，将自动排布
                            </div>
                        )}
                    </>
                )}

                {/* 其他组件内容 */}
                {content && item.type !== 'insight' && item.type !== 'metric' && (
                    <div className="flex-1 overflow-auto">
                        {content}
                    </div>
                )}

                {/* 拖拽时的视觉反馈 */}
                {(draggedItem === item.id || resizingItem === item.id) && (
                    <div className={`
                        absolute inset-0 
                        ${componentInfo.bgColor} bg-opacity-30
                        pointer-events-none rounded-xl 
                        border-2 ${componentInfo.borderColor} border-dashed
                    `} />
                )}

                {/* 编辑模式下的调整大小手柄 - 右下角 */}
                {isEditMode && (
                    <div
                        onMouseDown={(e) => handleResizeStart(e, item.id)}
                        className={`
                            absolute bottom-0 right-0 z-30
                            w-4 h-4 cursor-se-resize
                            ${componentInfo.bgColor} ${componentInfo.borderColor}
                            border-t-2 border-l-2 rounded-tl-md
                            hover:bg-blue-100 hover:border-blue-400
                            transition-colors duration-150
                            flex items-center justify-center
                        `}
                        title="拖拽调整大小"
                    >
                        <svg width="8" height="8" viewBox="0 0 8 8" className={componentInfo.textColor}>
                            <path d="M7 1v6H1" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
                        </svg>
                    </div>
                )}
            </div>
        );
    };

    // 非编辑模式下的流式布局渲染 - 高度根据数据量自动计算
    const renderFlowComponent = (item: LayoutItem) => {
        const componentInfo = getComponentInfoStatic(item.type);
        const areaTitle = getAreaTitleStatic(item.type);

        // 渲染多个指标（4列自动排布）
        const renderMetricsGrid = () => {
            if (!data?.metrics || !Array.isArray(data.metrics) || data.metrics.length === 0) {
                return <div className="p-4 text-center text-slate-400 text-sm">暂无指标数据</div>;
            }
            const metrics = data.metrics;
            return (
                <div className="grid grid-cols-4 gap-2 p-2">
                    {metrics.map((metric: any, idx: number) => (
                        <div key={idx} className="bg-blue-50 rounded-lg border border-blue-100">
                            <MetricCard
                                title={metric.title || ''}
                                value={metric.value || ''}
                                change={metric.change || ''}
                            />
                        </div>
                    ))}
                </div>
            );
        };

        // 渲染多个洞察（3列自动排布）- 可点击发起分析
        const renderInsightsGrid = () => {
            if (!data?.insights || !Array.isArray(data.insights) || data.insights.length === 0) {
                return <div className="p-4 text-center text-slate-400 text-sm">暂无洞察数据</div>;
            }
            const insights = data.insights;
            return (
                <div className="grid grid-cols-3 gap-2 p-2">
                    {insights.map((insight: any, idx: number) => (
                        <div 
                            key={idx} 
                            className="bg-purple-50 rounded-lg p-3 border border-purple-100 cursor-pointer hover:bg-purple-100 hover:border-purple-300 hover:shadow-md transition-all group"
                            onClick={() => handleInsightClick(insight)}
                            title="点击深入分析此洞察"
                        >
                            <SmartInsight
                                text={insight.text || ''}
                                icon={insight.icon || 'lightbulb'}
                            />
                            <div className="mt-2 text-xs text-purple-500 opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
                                <span>🔍</span>
                                <span>点击深入分析</span>
                            </div>
                        </div>
                    ))}
                </div>
            );
        };

        // 渲染图表
        const renderChart = () => {
            if (!activeChart) {
                return <div className="p-4 text-center text-slate-400 text-sm">暂无图表数据</div>;
            }
            if (activeChart.type === 'echarts' && typeof activeChart.data === 'string') {
                try {
                    const options = JSON.parse(activeChart.data);
                    return (
                        <div 
                            className="cursor-zoom-in group relative"
                            onDoubleClick={handleChartDoubleClick}
                            title="双击放大查看"
                        >
                            <Chart options={options} height="300px" />
                            <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity bg-black/50 text-white text-xs px-2 py-1 rounded">
                                双击放大
                            </div>
                        </div>
                    );
                } catch (e) {
                    return <div className="text-red-500 p-4">图表解析错误</div>;
                }
            } else if (activeChart.type === 'image') {
                return (
                    <div 
                        className="cursor-zoom-in group relative"
                        onDoubleClick={() => handleImageDoubleClick(activeChart.data)}
                        title="双击放大查看"
                    >
                        <img src={activeChart.data} alt="Chart" className="w-full object-contain max-h-96" />
                        <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity bg-black/50 text-white text-xs px-2 py-1 rounded">
                            双击放大
                        </div>
                    </div>
                );
            }
            return null;
        };

        // 渲染表格
        const renderTable = () => {
            // 使用系统日志调试
            import('../utils/systemLog').then(({ SystemLog }) => {
                SystemLog.debug('DraggableDashboard', `[renderTable] activeChart: ${JSON.stringify(activeChart ? { type: activeChart.type, hasData: !!activeChart.data, hasChartData: !!activeChart.chartData } : null)}`);
                SystemLog.debug('DraggableDashboard', `[renderTable] activeChart?.chartData?.charts: ${JSON.stringify(activeChart?.chartData?.charts?.map((c: any) => ({ type: c.type, hasData: !!c.data })))}`);
            });
            
            // 从activeChart.chartData.charts中提取表格数据
            let tableData: any[] = [];
            
            if (activeChart?.chartData?.charts) {
                // 查找所有table类型的图表
                const tableCharts = activeChart.chartData.charts.filter(
                    (chart: any) => chart.type === 'table'
                );
                
                import('../utils/systemLog').then(({ SystemLog }) => {
                    SystemLog.debug('DraggableDashboard', `[renderTable] tableCharts found: ${tableCharts.length}`);
                });
                
                if (tableCharts.length > 0 && tableCharts[0].data) {
                    // 使用第一个表格的数据
                    const firstTableData = tableCharts[0].data;
                    
                    import('../utils/systemLog').then(({ SystemLog }) => {
                        SystemLog.debug('DraggableDashboard', `[renderTable] firstTableData type: ${typeof firstTableData}, preview: ${typeof firstTableData === 'string' ? firstTableData.substring(0, 100) : 'array'}`);
                    });
                    
                    if (typeof firstTableData === 'string') {
                        try {
                            tableData = JSON.parse(firstTableData);
                        } catch (e) {
                            console.error('[renderTable] Failed to parse table data:', e);
                        }
                    } else if (Array.isArray(firstTableData)) {
                        tableData = firstTableData;
                    }
                }
            }
            
            // 兼容旧格式：直接从activeChart.data获取
            if (tableData.length === 0 && activeChart?.type === 'table' && activeChart?.data) {
                import('../utils/systemLog').then(({ SystemLog }) => {
                    SystemLog.debug('DraggableDashboard', `[renderTable] Trying old format - activeChart.type: ${activeChart.type}`);
                });
                if (typeof activeChart.data === 'string') {
                    try {
                        tableData = JSON.parse(activeChart.data);
                    } catch (e) {
                        console.error('Failed to parse table data:', e);
                    }
                } else if (Array.isArray(activeChart.data)) {
                    tableData = activeChart.data;
                }
            }
            
            // 兼容CSV类型
            if (tableData.length === 0 && activeChart?.type === 'csv' && activeChart?.data) {
                if (typeof activeChart.data === 'string') {
                    try {
                        tableData = JSON.parse(activeChart.data);
                    } catch (e) {
                        console.error('Failed to parse CSV data:', e);
                    }
                } else if (Array.isArray(activeChart.data)) {
                    tableData = activeChart.data;
                }
            }
            
            // 如果item.data有数据，也使用它（兼容性）
            if (tableData.length === 0 && Array.isArray(item.data) && item.data.length > 0) {
                tableData = item.data;
            }
            
            if (tableData.length === 0) {
                return <div className="p-4 text-center text-slate-400 text-sm">暂无表格数据</div>;
            }
            return <DataTable data={tableData} />;
        };

        // 渲染文件下载 - 带预览图和下载功能
        const renderFileDownload = () => {
            if (!sessionFiles || sessionFiles.length === 0) {
                return <div className="p-4 text-center text-slate-400 text-sm">暂无文件</div>;
            }
            
            // 只显示与当前选中消息关联的文件
            const filteredFiles = selectedMessageId 
                ? sessionFiles.filter(file => file.message_id === selectedMessageId)
                : sessionFiles;
            
            if (filteredFiles.length === 0) {
                return <div className="p-4 text-center text-slate-400 text-sm">当前分析请求无关联文件</div>;
            }
            
            return (
                <div className="p-2 grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                    {filteredFiles.map((file: main.SessionFile, idx: number) => {
                        const ext = file.name.split('.').pop()?.toLowerCase() || '';
                        const isImage = file.type === 'image' || ['png', 'jpg', 'jpeg', 'gif', 'webp'].includes(ext);
                        const isCsv = file.type === 'csv' || ['csv'].includes(ext);
                        const preview = filePreviews[file.path];
                        const isLoading = filePreviewsLoading[file.path];
                        const hasPreview = isImage || isCsv;
                        
                        return (
                            <div 
                                key={idx} 
                                className="flex flex-col bg-white rounded-lg border border-slate-200 overflow-hidden hover:border-blue-400 hover:shadow-lg cursor-pointer transition-all group"
                                onClick={() => handleFileDownload(file)}
                                title={`点击下载: ${file.name}`}
                            >
                                {/* 预览区域 - 图片和CSV显示缩略图 */}
                                <div className="h-32 bg-slate-100 flex items-center justify-center overflow-hidden">
                                    {isLoading ? (
                                        <div className="animate-pulse text-slate-400 text-xs">加载预览...</div>
                                    ) : preview ? (
                                        <img 
                                            src={preview} 
                                            alt={file.name} 
                                            className="w-full h-full object-contain group-hover:scale-105 transition-transform duration-200"
                                        />
                                    ) : isImage ? (
                                        <div className="flex flex-col items-center text-cyan-600">
                                            <Image size={32} />
                                            <span className="text-xs mt-1">图片文件</span>
                                        </div>
                                    ) : isCsv ? (
                                        <div className="flex flex-col items-center text-green-600">
                                            <FileSpreadsheet size={32} />
                                            <span className="text-xs mt-1">数据表格</span>
                                        </div>
                                    ) : (
                                        <div className="flex flex-col items-center text-orange-600">
                                            <FileText size={32} />
                                            <span className="text-xs mt-1 uppercase">{ext || '文件'}</span>
                                        </div>
                                    )}
                                </div>
                                
                                {/* 文件信息 */}
                                <div className="p-2 border-t border-slate-100">
                                    <div className="flex items-center gap-1.5">
                                        {getFileIcon(file.name, file.type)}
                                        <span className="text-xs text-slate-700 truncate flex-1" title={file.name}>
                                            {file.name}
                                        </span>
                                    </div>
                                    <div className="flex items-center justify-between mt-1">
                                        <span className="text-xs text-slate-400">
                                            {file.size ? `${(file.size / 1024).toFixed(1)} KB` : ''}
                                        </span>
                                        <span className="text-xs text-blue-500 opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-0.5">
                                            <Download size={12} />
                                            <span>下载</span>
                                        </span>
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            );
        };

        // 根据类型渲染内容
        const renderContent = () => {
            switch (item.type) {
                case 'metric': return renderMetricsGrid();
                case 'insight': return renderInsightsGrid();
                case 'chart': return renderChart();
                case 'table': return renderTable();
                case 'image': {
                    // 收集所有图片
                    const images: string[] = [];
                    
                    // 从chartData.charts中提取所有image类型的图片
                    if (activeChart?.chartData?.charts) {
                        for (const chart of activeChart.chartData.charts) {
                            if (chart.type === 'image' && typeof chart.data === 'string' && chart.data.startsWith('data:image')) {
                                images.push(chart.data);
                            }
                        }
                    }
                    
                    // 如果没有从charts数组中找到，尝试使用activeChart.data（兼容旧格式）
                    if (images.length === 0 && activeChart?.type === 'image' && typeof activeChart.data === 'string') {
                        images.push(activeChart.data);
                    }
                    
                    if (images.length === 0) {
                        return <div className="p-4 text-center text-slate-400 text-sm">暂无图片</div>;
                    }
                    
                    // 确保当前索引在有效范围内
                    const validIndex = Math.min(currentImageIndex, images.length - 1);
                    const currentImage = images[validIndex];
                    
                    return (
                        <div className="relative h-full flex flex-col">
                            {/* 图片显示区域 */}
                            <div 
                                className="flex-1 cursor-zoom-in group relative flex items-center justify-center p-2"
                                onDoubleClick={() => handleImageDoubleClick(currentImage)}
                                title="双击放大查看"
                            >
                                <img src={currentImage} alt={`Image ${validIndex + 1}`} className="max-w-full max-h-full object-contain" />
                                <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity bg-black/50 text-white text-xs px-2 py-1 rounded">
                                    双击放大
                                </div>
                            </div>
                            
                            {/* 导航控制条（仅在多张图片时显示） */}
                            {images.length > 1 && (
                                <div className="flex-shrink-0 flex items-center justify-center gap-3 py-2 px-3 bg-slate-50 border-t border-slate-200">
                                    <button
                                        onClick={() => setCurrentImageIndex(Math.max(0, currentImageIndex - 1))}
                                        disabled={currentImageIndex === 0}
                                        className={`p-1.5 rounded transition-colors ${
                                            currentImageIndex === 0
                                                ? 'text-slate-300 cursor-not-allowed'
                                                : 'text-slate-600 hover:bg-slate-200 hover:text-slate-800'
                                        }`}
                                        title="前一张"
                                    >
                                        <ChevronLeft size={18} />
                                    </button>
                                    
                                    <span className="text-sm text-slate-600 font-medium min-w-[60px] text-center">
                                        {validIndex + 1} / {images.length}
                                    </span>
                                    
                                    <button
                                        onClick={() => setCurrentImageIndex(Math.min(images.length - 1, currentImageIndex + 1))}
                                        disabled={currentImageIndex >= images.length - 1}
                                        className={`p-1.5 rounded transition-colors ${
                                            currentImageIndex >= images.length - 1
                                                ? 'text-slate-300 cursor-not-allowed'
                                                : 'text-slate-600 hover:bg-slate-200 hover:text-slate-800'
                                        }`}
                                        title="后一张"
                                    >
                                        <ChevronRight size={18} />
                                    </button>
                                </div>
                            )}
                        </div>
                    );
                }
                case 'file_download': return renderFileDownload();
                default: return <div className="p-4 text-center text-slate-400 text-sm">未知组件类型</div>;
            }
        };

        return (
            <div
                key={item.id}
                className={`
                    flex flex-col rounded-xl overflow-hidden bg-white shadow-md
                    hover:shadow-lg transition-shadow duration-200
                `}
                style={{ width: `${item.w}%` }}
            >
                {/* 区域标题条 */}
                <div className={`
                    flex-shrink-0 px-3 py-1.5 flex items-center gap-1.5
                    ${componentInfo.bgColor} ${componentInfo.borderColor}
                    border-b
                `}>
                    <span className="text-base">{componentInfo.icon}</span>
                    <span className={`text-sm font-medium ${componentInfo.textColor}`}>{areaTitle}</span>
                </div>

                {/* 内容区域 - 高度自动 */}
                <div className="flex-1">
                    {renderContent()}
                </div>
            </div>
        );
    };

    // 静态辅助函数（避免在renderFlowComponent中重复定义）
    const getComponentInfoStatic = (type: string) => {
        switch (type) {
            case 'metric': return { icon: '📊', bgColor: 'bg-blue-50', borderColor: 'border-blue-200', textColor: 'text-blue-700' };
            case 'insight': return { icon: '💡', bgColor: 'bg-purple-50', borderColor: 'border-purple-200', textColor: 'text-purple-700' };
            case 'chart': return { icon: '📈', bgColor: 'bg-green-50', borderColor: 'border-green-200', textColor: 'text-green-700' };
            case 'table': return { icon: '📋', bgColor: 'bg-amber-50', borderColor: 'border-amber-200', textColor: 'text-amber-700' };
            case 'image': return { icon: '🖼️', bgColor: 'bg-cyan-50', borderColor: 'border-cyan-200', textColor: 'text-cyan-700' };
            case 'file_download': return { icon: '📁', bgColor: 'bg-orange-50', borderColor: 'border-orange-200', textColor: 'text-orange-700' };
            default: return { icon: '📦', bgColor: 'bg-gray-50', borderColor: 'border-gray-200', textColor: 'text-gray-700' };
        }
    };

    const getAreaTitleStatic = (type: string) => {
        switch (type) {
            case 'metric': return '关键指标';
            case 'insight': return '自动洞察';
            case 'chart': return '数据图表';
            case 'table': return '数据表格';
            case 'image': return '图片';
            case 'file_download': return '文件下载';
            default: return '组件';
        }
    };

    return (
        <div className="relative h-full w-full bg-slate-50 flex flex-col">
            {/* 顶部标题栏 */}
            <div className="flex-shrink-0 bg-white border-b border-slate-200 px-6 py-3">
                <div className="flex items-center justify-between">
                    {/* 左侧：编辑按钮 + 标题 */}
                    <div className="flex items-center gap-4">
                        {/* 编辑布局按钮 - 淡雅风格 */}
                        <button
                            onClick={() => {
                                setIsEditMode(!isEditMode);
                                if (isEditMode) {
                                    handleSaveLayout();
                                }
                            }}
                            className={`
                                px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition-all text-sm
                                ${isEditMode 
                                    ? 'bg-green-50 border border-green-300 text-green-700 hover:bg-green-100' 
                                    : 'bg-slate-50 border border-slate-200 text-slate-600 hover:bg-slate-100 hover:border-slate-300'
                                }
                            `}
                            title={isEditMode ? "保存布局" : "编辑布局"}
                        >
                            {isEditMode ? (
                                <>
                                    <Save size={14} />
                                    <span>保存</span>
                                </>
                            ) : (
                                <>
                                    <Edit3 size={14} />
                                    <span>编辑</span>
                                </>
                            )}
                        </button>

                        {/* 自动排列按钮 - 仅编辑模式显示，淡雅风格 */}
                        {isEditMode && (
                            <button
                                onClick={() => {
                                    setLayout(prev => autoArrangeLayout(prev));
                                }}
                                className="px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition-all text-sm
                                    bg-indigo-50 border border-indigo-200 text-indigo-600 hover:bg-indigo-100"
                                title="自动排列所有组件"
                            >
                                <span>📐</span>
                                <span>排列</span>
                            </button>
                        )}

                        {/* 分隔线 */}
                        <div className="h-6 w-px bg-slate-200"></div>

                        {/* 标题和用户请求 */}
                        <div className="flex flex-col">
                            <div className="flex items-center gap-3">
                                <h1 className="text-lg font-semibold text-slate-700 flex items-center gap-2">
                                    <span>📊</span>
                                    智能分析仪表盘
                                    {isEditMode && (
                                        <span className="text-xs font-normal px-2 py-0.5 bg-amber-50 text-amber-600 border border-amber-200 rounded-full">
                                            编辑中
                                        </span>
                                    )}
                                </h1>
                                {/* Global Analysis Status - Requirements: 3.1, 3.2, 3.3, 3.4 */}
                                <GlobalAnalysisStatus />

                            </div>
                            {userRequestText && (
                                <p className="text-xs text-slate-500 mt-0.5 max-w-md truncate" title={userRequestText}>
                                    {userRequestText}
                                </p>
                            )}
                        </div>
                    </div>

                    {/* 右侧：数据导出按钮 */}
                    <div className="flex items-center gap-2">
                        {hasExportableContent() && (
                            <div className="relative export-dropdown-container">
                                <button
                                    onClick={() => setExportDropdownOpen(!exportDropdownOpen)}
                                    className="px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition-all text-sm
                                        bg-purple-50 border border-purple-200 text-purple-600 hover:bg-purple-100"
                                    title="导出仪表盘数据"
                                >
                                    <Download size={14} />
                                    <span>导出</span>
                                </button>

                                {/* 导出下拉菜单 */}
                                {exportDropdownOpen && (
                                    <div className="absolute right-0 top-full mt-2 w-48 bg-white rounded-lg shadow-lg border border-slate-200 py-2 z-50">
                                        <button
                                            onClick={exportAsPDF}
                                            className="w-full flex items-center gap-3 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                                        >
                                            <FileImage size={16} className="text-red-600" />
                                            <span>导出为 PDF</span>
                                        </button>
                                        <button
                                            onClick={exportAsPPT}
                                            className="w-full flex items-center gap-3 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                                        >
                                            <Presentation size={16} className="text-orange-600" />
                                            <span>导出为 PPT</span>
                                        </button>
                                        <button
                                            onClick={exportDataFiles}
                                            className="w-full flex items-center gap-3 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                                        >
                                            <Download size={16} className="text-green-600" />
                                            <span>导出数据文件</span>
                                        </button>
                                    </div>
                                )}
                            </div>
                        )}
                    </div>
                </div>
            </div>

            {/* 编辑模式下的控件库面板 - 只显示有数据的组件类型 */}
            {isEditMode && (
                <div className="flex-shrink-0 bg-white border-b border-slate-200 px-6 py-3">
                    <div className="flex items-center gap-4">
                        <span className="text-sm font-medium text-slate-600">控件库：</span>
                        <div className="flex items-center gap-2 flex-wrap">
                            {/* 关键指标 - 编辑模式下始终显示 */}
                            <button
                                onClick={() => {
                                    if (layout.some(i => i.type === 'metric')) return;
                                    const newItem: LayoutItem = {
                                        id: 'metric-area',
                                        type: 'metric',
                                        x: 0,
                                        y: Math.max(...layout.map(i => i.y + i.h), 0) + 10,
                                        w: 100,
                                        h: EDIT_MODE_HEIGHTS.metric,
                                        data: null
                                    };
                                    setLayout(prev => [...prev, newItem]);
                                }}
                                disabled={layout.some(i => i.type === 'metric')}
                                className={`px-3 py-2 border rounded-lg transition-colors flex items-center gap-1.5 text-sm
                                    ${layout.some(i => i.type === 'metric') 
                                        ? 'bg-gray-100 border-gray-200 text-gray-400 cursor-not-allowed' 
                                        : 'bg-blue-50 border-blue-200 text-blue-700 hover:bg-blue-100'}`}
                            >
                                <span>📊</span>
                                <span>关键指标</span>
                            </button>

                            {/* 数据图表 */}
                            <button
                                onClick={() => {
                                    if (layout.some(i => i.type === 'chart')) return;
                                    const newItem: LayoutItem = {
                                        id: 'chart-area',
                                        type: 'chart',
                                        x: 0,
                                        y: Math.max(...layout.map(i => i.y + i.h), 0) + 10,
                                        w: 100,
                                        h: EDIT_MODE_HEIGHTS.chart,
                                        data: null
                                    };
                                    setLayout(prev => [...prev, newItem]);
                                }}
                                disabled={layout.some(i => i.type === 'chart')}
                                className={`px-3 py-2 border rounded-lg transition-colors flex items-center gap-1.5 text-sm
                                    ${layout.some(i => i.type === 'chart') 
                                        ? 'bg-gray-100 border-gray-200 text-gray-400 cursor-not-allowed' 
                                        : 'bg-green-50 border-green-200 text-green-700 hover:bg-green-100'}`}
                            >
                                <span>📈</span>
                                <span>数据图表</span>
                            </button>

                            {/* 自动洞察 */}
                            <button
                                onClick={() => {
                                    if (layout.some(i => i.type === 'insight')) return;
                                    const newItem: LayoutItem = {
                                        id: 'insight-area',
                                        type: 'insight',
                                        x: 0,
                                        y: Math.max(...layout.map(i => i.y + i.h), 0) + 10,
                                        w: 100,
                                        h: EDIT_MODE_HEIGHTS.insight,
                                        data: null
                                    };
                                    setLayout(prev => [...prev, newItem]);
                                }}
                                disabled={layout.some(i => i.type === 'insight')}
                                className={`px-3 py-2 border rounded-lg transition-colors flex items-center gap-1.5 text-sm
                                    ${layout.some(i => i.type === 'insight') 
                                        ? 'bg-gray-100 border-gray-200 text-gray-400 cursor-not-allowed' 
                                        : 'bg-purple-50 border-purple-200 text-purple-700 hover:bg-purple-100'}`}
                            >
                                <span>💡</span>
                                <span>自动洞察</span>
                            </button>

                            {/* 数据表格 */}
                            <button
                                onClick={() => {
                                    if (layout.some(i => i.type === 'table')) return;
                                    const newItem: LayoutItem = {
                                        id: 'table-area',
                                        type: 'table',
                                        x: 0,
                                        y: Math.max(...layout.map(i => i.y + i.h), 0) + 10,
                                        w: 100,
                                        h: EDIT_MODE_HEIGHTS.table,
                                        data: null
                                    };
                                    setLayout(prev => [...prev, newItem]);
                                }}
                                disabled={layout.some(i => i.type === 'table')}
                                className={`px-3 py-2 border rounded-lg transition-colors flex items-center gap-1.5 text-sm
                                    ${layout.some(i => i.type === 'table') 
                                        ? 'bg-gray-100 border-gray-200 text-gray-400 cursor-not-allowed' 
                                        : 'bg-amber-50 border-amber-200 text-amber-700 hover:bg-amber-100'}`}
                            >
                                <span>📋</span>
                                <span>数据表格</span>
                            </button>

                            {/* 图片 */}
                            <button
                                onClick={() => {
                                    if (layout.some(i => i.type === 'image')) return;
                                    const newItem: LayoutItem = {
                                        id: 'image-area',
                                        type: 'image',
                                        x: 0,
                                        y: Math.max(...layout.map(i => i.y + i.h), 0) + 10,
                                        w: 50,
                                        h: EDIT_MODE_HEIGHTS.image,
                                        data: null
                                    };
                                    setLayout(prev => [...prev, newItem]);
                                }}
                                disabled={layout.some(i => i.type === 'image')}
                                className={`px-3 py-2 border rounded-lg transition-colors flex items-center gap-1.5 text-sm
                                    ${layout.some(i => i.type === 'image') 
                                        ? 'bg-gray-100 border-gray-200 text-gray-400 cursor-not-allowed' 
                                        : 'bg-cyan-50 border-cyan-200 text-cyan-700 hover:bg-cyan-100'}`}
                            >
                                <span>🖼️</span>
                                <span>图片</span>
                            </button>

                            {/* 文件下载 */}
                            <button
                                onClick={() => {
                                    if (layout.some(i => i.type === 'file_download')) return;
                                    const newItem: LayoutItem = {
                                        id: 'file_download-area',
                                        type: 'file_download',
                                        x: 0,
                                        y: Math.max(...layout.map(i => i.y + i.h), 0) + 10,
                                        w: 50,
                                        h: EDIT_MODE_HEIGHTS.file_download,
                                        data: null
                                    };
                                    setLayout(prev => [...prev, newItem]);
                                }}
                                disabled={layout.some(i => i.type === 'file_download')}
                                className={`px-3 py-2 border rounded-lg transition-colors flex items-center gap-1.5 text-sm
                                    ${layout.some(i => i.type === 'file_download') 
                                        ? 'bg-gray-100 border-gray-200 text-gray-400 cursor-not-allowed' 
                                        : 'bg-orange-50 border-orange-200 text-orange-700 hover:bg-orange-100'}`}
                            >
                                <span>📁</span>
                                <span>文件下载</span>
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* 仪表盘容器 */}
            <div
                id="dashboard-container"
                className="relative flex-1 overflow-auto p-6"
                onMouseMove={handleDrag}
                onMouseUp={handleDragEnd}
                onMouseLeave={handleDragEnd}
                onClick={onDashboardClick}
            >
                {getDisplayLayout().length === 0 ? (
                    <div className="flex items-center justify-center h-full">
                        <div className="text-center text-slate-400">
                            <p className="text-lg">{t('no_data_available')}</p>
                            <p className="text-sm mt-2">{t('start_analysis_to_see_results')}</p>
                        </div>
                    </div>
                ) : isEditMode ? (
                    // 编辑模式：使用绝对定位，显示所有组件
                    <div className="relative" style={{ minHeight: '100vh' }}>
                        {layout.map(item => renderComponent(item))}
                    </div>
                ) : (
                    // 非编辑模式：使用流式布局，只显示有数据的组件
                    <div className="flex flex-col gap-4">
                        {getDisplayLayout()
                            .sort((a, b) => a.y - b.y || a.x - b.x)
                            .map(item => renderFlowComponent(item))}
                    </div>
                )}

                {/* 编辑模式提示 - 小字体 */}
                {isEditMode && layout.length > 0 && (
                    <div className="fixed bottom-4 left-1/2 transform -translate-x-1/2 bg-blue-600 text-white px-4 py-2 rounded-lg shadow-lg z-50">
                        <p className="text-sm font-medium">
                            💡 拖拽组件移动位置，拖拽右下角调整大小，点击 X 删除，完成后点击"保存布局"
                        </p>
                    </div>
                )}
            </div>

            {/* 图片放大模态框 */}
            <ImageModal
                isOpen={imageModalOpen}
                imageUrl={modalImageUrl}
                onClose={() => setImageModalOpen(false)}
            />

            {/* 图表放大模态框 */}
            {modalChartOptions && (
                <ChartModal
                    isOpen={chartModalOpen}
                    options={modalChartOptions}
                    onClose={() => setChartModalOpen(false)}
                />
            )}

            {/* Toast 提示 */}
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

export default DraggableDashboard;
