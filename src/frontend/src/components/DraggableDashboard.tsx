/**
 * Draggable Dashboard Component
 * 
 * 完整的可拖拽仪表盘，整合真实数据展示和拖拽功能
 * 使用新的统一数据系统 (useDashboardData Hook)
 */

import React, { useState, useEffect } from 'react';
import { Edit3, Lock, Unlock, Save, X, Download, FileText, Image, Table, FileSpreadsheet, ChevronLeft, ChevronRight, Presentation, FileChartColumn, ClipboardList } from 'lucide-react';
import MetricCard from './MetricCard';
import SmartInsight from './SmartInsight';
import DataTable from './DataTable';
import Chart from './Chart';
import ImageModal from './ImageModal';
import ChartModal from './ChartModal';
import Toast, { ToastType } from './Toast';
import { main } from '../../wailsjs/go/models';
import { useLanguage } from '../i18n';
import { SaveLayout, LoadLayout, SelectSaveFile, GetSessionFileAsBase64, DownloadSessionFile, GenerateCSVThumbnail, GenerateFilePreview, GenerateReport, PrepareReport, ExportReport } from '../../wailsjs/go/main/App';
import { EventsEmit } from '../../wailsjs/runtime/runtime';
import { database } from '../../wailsjs/go/models';
import { createLogger } from '../utils/systemLog';
import { useDashboardData } from '../hooks/useDashboardData';
import { renderFilePreview } from '../utils/FilePreviewRenderer';
import { GlobalAnalysisStatus } from './GlobalAnalysisStatus';

const logger = createLogger('DraggableDashboard');

/**
 * 简单的内联 markdown 渲染函数
 * 处理标题中的加粗文本 (**text** 或 __text__)
 */
const renderInlineMarkdown = (text: string): React.ReactNode => {
    if (!text) return text;
    
    // 匹配 **text** 或 __text__ 格式的加粗文本
    const boldPattern = /(\*\*|__)(.+?)\1/g;
    const parts: React.ReactNode[] = [];
    let lastIndex = 0;
    let match;
    let keyIndex = 0;
    
    while ((match = boldPattern.exec(text)) !== null) {
        // 添加匹配前的普通文本
        if (match.index > lastIndex) {
            parts.push(text.slice(lastIndex, match.index));
        }
        // 添加加粗文本
        parts.push(<strong key={keyIndex++}>{match[2]}</strong>);
        lastIndex = match.index + match[0].length;
    }
    
    // 添加剩余的普通文本
    if (lastIndex < text.length) {
        parts.push(text.slice(lastIndex));
    }
    
    return parts.length > 0 ? parts : text;
};

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
    columns?: string[];
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
    
    // 创建兼容变量，从新系统获取数据 - memoized to prevent unnecessary re-renders
    const data = React.useMemo(() => ({
        metrics: dashboardData.metrics.map(m => ({
            title: m.title,
            value: m.value,
            change: m.change || ''
        })),
        insights: dashboardData.insights.map(i => ({
            text: i.text,
            icon: i.icon || 'lightbulb',
            dataSourceId: i.dataSourceId,
            sourceName: i.sourceName
        }))
    }), [dashboardData.metrics, dashboardData.insights]);
    
    // 构建兼容的 activeChart 对象 - memoized to avoid expensive JSON.stringify on every render
    const activeChart = React.useMemo<{ type: 'echarts' | 'image' | 'table' | 'csv', data: any, chartData?: any } | null>(() => {
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
                        ...dashboardData.allTableData.map(t => ({ type: 'table', data: t.rows, columns: t.columns }))
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
                    charts: dashboardData.allTableData.map(t => ({ type: 'table', data: t.rows, columns: t.columns }))
                }
            };
        }
        return null;
    }, [dashboardData.hasECharts, dashboardData.echartsData, dashboardData.allEChartsData, 
        dashboardData.hasImages, dashboardData.images, 
        dashboardData.hasTables, dashboardData.tableData, dashboardData.allTableData]);
    
    const [isEditMode, setIsEditMode] = useState(false);
    const [filePreviewsLoading, setFilePreviewsLoading] = useState<Record<string, boolean>>({});
    const [filePreviews, setFilePreviews] = useState<Record<string, string>>({});
    const [currentImageIndex, setCurrentImageIndex] = useState(0);
    
    // Memoize parsed ECharts configs to avoid re-parsing JSON on every render
    const parsedEChartsConfigs = React.useMemo(() => {
        if (!dashboardData.allEChartsData || dashboardData.allEChartsData.length === 0) return [];
        return dashboardData.allEChartsData.map((chartData) => {
            try {
                if (typeof chartData === 'string') {
                    try {
                        return JSON.parse(chartData);
                    } catch {
                        const cleanedData = cleanEChartsJsonString(chartData);
                        return JSON.parse(cleanedData);
                    }
                }
                return chartData;
            } catch (e) {
                logger.error(`[parsedEChartsConfigs] Failed to parse: ${e}`);
                return null;
            }
        });
    }, [dashboardData.allEChartsData]);
    
    // 图表/图片放大模态框状态
    const [imageModalOpen, setImageModalOpen] = useState(false);
    const [chartModalOpen, setChartModalOpen] = useState(false);
    const [modalImageUrl, setModalImageUrl] = useState<string>('');
    const [modalChartOptions, setModalChartOptions] = useState<any>(null);
    
    // 导出功能状态
    const [exportDropdownOpen, setExportDropdownOpen] = useState(false);
    const [toast, setToast] = useState<{ message: string; type: ToastType } | null>(null);
    const [isGeneratingReport, setIsGeneratingReport] = useState(false);
    const [preparedReportId, setPreparedReportId] = useState<string | null>(null);

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

    // 检查是否有可导出的内容（只检查真正的分析结果，不包括数据源统计）
    const hasExportableContent = () => {
        return dashboardData.hasRealAnalysisResults;
    };

    // 生成报告（使用LLM生成正式分析报告，缓存后可多次导出不同格式）
    const prepareReport = async () => {
        try {
            setExportDropdownOpen(false);
            setIsGeneratingReport(true);
            setPreparedReportId(null);
            logger.debug('Starting report preparation...');

            // 尝试从洞察数据中获取数据源名称
            let dataSourceName = '';
            if (dashboardData.hasInsights) {
                for (const insight of dashboardData.insights) {
                    if (insight.sourceName) {
                        dataSourceName = insight.sourceName;
                        break;
                    }
                }
            }
            if (!dataSourceName && dashboardData.hasTables && dashboardData.allTableData.length > 0) {
                const firstTable = dashboardData.allTableData[0];
                if (firstTable.title) {
                    dataSourceName = firstTable.title;
                }
            }

            const reportData: any = {
                userRequest: userRequestText || '',
                dataSourceName: dataSourceName,
                metrics: [],
                insights: [],
                chartImages: [],
                tableData: null,
                allTableData: [],
                format: ''
            };

            // 收集指标数据
            if (dashboardData.hasMetrics) {
                reportData.metrics = dashboardData.metrics.map((metric) => ({
                    title: metric.title || '',
                    value: metric.value || '',
                    change: metric.change || ''
                }));
            }

            // 收集洞察数据
            if (dashboardData.hasInsights) {
                reportData.insights = dashboardData.insights.map((insight) =>
                    insight.text || ''
                );
            }

            // 收集所有图表图片
            const chartImages: string[] = [];
            const echartsComponents = document.querySelectorAll('.echarts-for-react');
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
                        }
                    }
                } catch (e) {
                    console.error(`[DraggableDashboard] Failed to capture EChart ${i}:`, e);
                }
            }

            // Canvas fallback
            if (chartImages.length === 0) {
                const canvasElements = document.querySelectorAll('canvas');
                for (let i = 0; i < canvasElements.length; i++) {
                    const canvas = canvasElements[i];
                    const parent = canvas.parentElement;
                    if (parent && (parent.classList.contains('echarts-for-react') ||
                                   parent.querySelector('.echarts-for-react') ||
                                   canvas.width > 200)) {
                        try {
                            chartImages.push(canvas.toDataURL('image/png'));
                        } catch (e) { /* skip */ }
                    }
                }
            }

            // dashboardData images
            if (dashboardData.hasImages) {
                for (const img of dashboardData.images) {
                    if (typeof img === 'string' && img.startsWith('data:image') && !chartImages.includes(img)) {
                        chartImages.push(img);
                    }
                }
            }

            if (chartImages.length > 0) {
                reportData.chartImages = chartImages;
            }

            // 收集表格数据
            if (dashboardData.hasTables && dashboardData.tableData) {
                const tableData = dashboardData.tableData;
                if (tableData.columns && tableData.columns.length > 0 && tableData.rows && tableData.rows.length > 0) {
                    reportData.tableData = {
                        columns: tableData.columns.map(col => ({ title: col, dataType: 'string' })),
                        data: tableData.rows.map((row: Record<string, any>) =>
                            tableData.columns.map(col => row[col] === null || row[col] === undefined ? '' : row[col])
                        )
                    };
                }
            }

            // 收集所有表格
            if (dashboardData.hasTables && dashboardData.allTableData.length > 0) {
                reportData.allTableData = dashboardData.allTableData.map((table, index) => ({
                    name: table.title || `表格${index + 1}`,
                    table: {
                        columns: table.columns.map(col => ({ title: col, dataType: 'string' })),
                        data: table.rows.map((row: Record<string, any>) =>
                            table.columns.map(col => row[col] === null || row[col] === undefined ? '' : row[col])
                        )
                    }
                }));
            }

            logger.debug(`Report data prepared: metrics=${reportData.metrics.length}, insights=${reportData.insights.length}, charts=${chartImages.length}`);

            const reportId = await PrepareReport(reportData);

            setIsGeneratingReport(false);
            setPreparedReportId(reportId);
            setExportDropdownOpen(true);
            setToast({ message: t('report_ready'), type: 'success' });
        } catch (error) {
            setIsGeneratingReport(false);
            console.error('[DraggableDashboard] Report preparation failed:', error);
            setToast({
                message: (t('generate_report_failed')) + (error instanceof Error ? error.message : String(error)),
                type: 'error'
            });
        }
    };

    // 导出已生成的报告
    const exportReportAs = async (format: 'word' | 'pdf') => {
        if (!preparedReportId) return;
        try {
            await ExportReport(preparedReportId, format);
            setToast({ message: t('generate_report_success'), type: 'success' });
        } catch (error) {
            console.error('[DraggableDashboard] Report export failed:', error);
            setToast({
                message: (t('generate_report_failed')) + (error instanceof Error ? error.message : String(error)),
                type: 'error'
            });
        }
    };

    // 清理 ECharts 配置字符串中的 JavaScript 函数
    const cleanEChartsJsonString = (jsonStr: string): string => {
        let result = jsonStr;
        // 移除 JavaScript 函数定义
        const functionPattern = /,?\s*"?(\w+)"?\s*:\s*function\s*\([^)]*\)\s*\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}/g;
        result = result.replace(functionPattern, '');
        // 清理可能残留的尾随逗号
        result = result.replace(/,(\s*[}\]])/g, '$1');
        // 清理可能残留的前导逗号
        result = result.replace(/([{[]\s*),/g, '$1');
        return result;
    };
    
    // 安全解析 ECharts 配置
    const safeParseEChartsConfig = (data: any): any => {
        if (typeof data === 'string') {
            try {
                return JSON.parse(data);
            } catch (e) {
                // 如果解析失败，尝试清理 JavaScript 函数后再解析
                try {
                    const cleanedData = cleanEChartsJsonString(data);
                    return JSON.parse(cleanedData);
                } catch (e2) {
                    logger.error(`[safeParseEChartsConfig] Failed to parse ECharts config: ${e2}`);
                    throw e;
                }
            }
        }
        return data;
    };

    // 双击图表放大显示
    const handleChartDoubleClick = () => {
        if (!activeChart) return;
        
        if (activeChart.type === 'echarts' && typeof activeChart.data === 'string') {
            try {
                const options = safeParseEChartsConfig(activeChart.data);
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
        const isDataSourceInsight = insight.dataSourceId && insight.dataSourceId !== '';
        logger.warn(`[InsightClick] dsId=${insight.dataSourceId}, isDS=${isDataSourceInsight}`);
        
        if (isDataSourceInsight) {
            logger.warn(`[InsightClick] onInsightClick defined: ${!!onInsightClick}`);
            if (onInsightClick) {
                logger.warn(`[InsightClick] Calling onInsightClick...`);
                onInsightClick(insight);
                logger.warn(`[InsightClick] onInsightClick called`);
            }
            return;
        }
        
        // 优先使用回调函数，由父组件统一管理分析请求
        if (onInsightClick) {
            onInsightClick(insight);
        } else if (activeThreadId) {
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
        if (['pptx', 'ppt'].includes(ext)) {
            return <Presentation size={20} className="text-orange-600" />;
        }
        if (['json', 'xml'].includes(ext)) {
            return <Table size={20} className="text-amber-600" />;
        }
        return <FileText size={20} className="text-orange-600" />;
    };

    // 获取文件预览（图片、Excel、PPT、CSV）
    const loadFilePreview = async (file: main.SessionFile) => {
        if (!activeThreadId || filePreviews[file.path] || filePreviewsLoading[file.path]) return;
        
        const ext = file.name.split('.').pop()?.toLowerCase() || '';
        const isImage = file.type === 'image' || ['png', 'jpg', 'jpeg', 'gif', 'webp'].includes(ext);
        const isPreviewable = ['csv', 'xlsx', 'xls', 'pptx'].includes(ext);
        
        if (!isImage && !isPreviewable) return;
        
        setFilePreviewsLoading(prev => ({ ...prev, [file.path]: true }));
        try {
            if (isImage) {
                const base64Data = await GetSessionFileAsBase64(activeThreadId, file.name);
                if (base64Data) {
                    setFilePreviews(prev => ({ ...prev, [file.path]: base64Data }));
                }
            } else if (isPreviewable) {
                // Excel/PPT/CSV: 获取结构化预览数据，前端渲染为图片
                const previewJson = await GenerateFilePreview(activeThreadId, file.name);
                if (previewJson) {
                    const previewImage = renderFilePreview(previewJson);
                    if (previewImage) {
                        setFilePreviews(prev => ({ ...prev, [file.path]: previewImage }));
                    }
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
    // 所有组件宽度设为100（全宽），垂直堆叠布局
    const defaultLayout: LayoutItem[] = [
        { id: 'metric-area', type: 'metric', x: 0, y: 0, w: 100, h: EDIT_MODE_HEIGHTS.metric, data: null },
        { id: 'chart-area', type: 'chart', x: 0, y: 90, w: 100, h: EDIT_MODE_HEIGHTS.chart, data: null },
        { id: 'insight-area', type: 'insight', x: 0, y: 200, w: 100, h: EDIT_MODE_HEIGHTS.insight, data: null },
        { id: 'table-area', type: 'table', x: 0, y: 280, w: 100, h: EDIT_MODE_HEIGHTS.table, data: null },
        { id: 'image-area', type: 'image', x: 0, y: 360, w: 100, h: EDIT_MODE_HEIGHTS.image, data: null },
        { id: 'file_download-area', type: 'file_download', x: 0, y: 450, w: 100, h: EDIT_MODE_HEIGHTS.file_download, data: null },
    ];
    const [layout, setLayout] = useState<LayoutItem[]>(defaultLayout);
    const [draggedItem, setDraggedItem] = useState<string | null>(null);
    const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 });
    // 新增：调整大小状态
    const [resizingItem, setResizingItem] = useState<string | null>(null);
    const [resizeStart, setResizeStart] = useState({ x: 0, y: 0, w: 0, h: 0 });

    // 检查某种类型的组件是否有数据
    // 直接使用 dashboardData 而不是 activeChart，确保所有数据类型都能正确检测
    const hasDataForType = (type: string): boolean => {
        switch (type) {
            case 'metric':
                return dashboardData.hasMetrics;
            case 'insight':
                return dashboardData.hasInsights;
            case 'chart':
                // 图表组件：检查是否有 ECharts 数据
                return dashboardData.hasECharts;
            case 'table':
                // 表格数据：直接使用 dashboardData.hasTables
                return dashboardData.hasTables;
            case 'image':
                // 图片组件：直接使用 dashboardData.hasImages
                return dashboardData.hasImages;
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
        const displayItems = layout.filter(item => hasDataForType(item.type));
        
        // 诊断日志：记录布局中的所有类型和过滤结果
        const allTypes = layout.map(item => item.type).join(',');
        const displayTypes = displayItems.map(item => item.type).join(',');
        logger.warn(`[getDisplayLayout] layout types=[${allTypes}], display types=[${displayTypes}], hasECharts=${dashboardData.hasECharts}`);
        
        return displayItems;
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
                        // 确保所有默认类型都存在，补充缺失的类型
                        const requiredTypes: LayoutItem['type'][] = ['metric', 'chart', 'insight', 'table', 'image', 'file_download'];
                        let maxY = Math.max(...convertedLayout.map(i => i.y + i.h), 0);
                        
                        for (const reqType of requiredTypes) {
                            if (!seenTypes.has(reqType)) {
                                const minH = MIN_HEIGHTS[reqType] || 56;
                                convertedLayout.push({
                                    id: `${reqType}-area`,
                                    type: reqType,
                                    x: 0,
                                    y: maxY,
                                    w: 100,
                                    h: minH,
                                    data: null
                                });
                                maxY += minH + 10;
                                logger.warn(`[loadSavedLayout] Added missing type: ${reqType}`);
                            }
                        }
                        
                        logger.warn(`[loadSavedLayout] Final ${convertedLayout.length} items: ${convertedLayout.map(i => i.type).join(',')}`);
                        setLayout(convertedLayout);
                    } else {
                        logger.warn(`[loadSavedLayout] No items in saved layout, using default`);
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
                        name: t('component_metric'), 
                        desc: t('component_metric_desc'),
                        icon: '📊',
                        bgColor: 'bg-blue-50',
                        borderColor: 'border-blue-200',
                        textColor: 'text-blue-700'
                    };
                case 'insight':
                    return { 
                        name: t('component_insight'), 
                        desc: t('component_insight_desc'),
                        icon: '💡',
                        bgColor: 'bg-purple-50',
                        borderColor: 'border-purple-200',
                        textColor: 'text-purple-700'
                    };
                case 'chart':
                    return { 
                        name: t('component_chart'), 
                        desc: t('component_chart_desc'),
                        icon: '📈',
                        bgColor: 'bg-green-50',
                        borderColor: 'border-green-200',
                        textColor: 'text-green-700'
                    };
                case 'table':
                    return { 
                        name: t('component_table'), 
                        desc: t('component_table_desc'),
                        icon: '📋',
                        bgColor: 'bg-amber-50',
                        borderColor: 'border-amber-200',
                        textColor: 'text-amber-700'
                    };
                case 'image':
                    return { 
                        name: t('component_image'), 
                        desc: t('component_image_desc'),
                        icon: '🖼️',
                        bgColor: 'bg-cyan-50',
                        borderColor: 'border-cyan-200',
                        textColor: 'text-cyan-700'
                    };
                case 'file_download':
                case 'file':
                    return { 
                        name: t('component_file'), 
                        desc: t('component_file_desc'),
                        icon: '📁',
                        bgColor: 'bg-orange-50',
                        borderColor: 'border-orange-200',
                        textColor: 'text-orange-700'
                    };
                default:
                    return { 
                        name: t('component_generic'), 
                        desc: t('component_generic_desc'),
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
                case 'metric': return t('area_metric');
                case 'insight': return t('area_insight');
                case 'chart': return t('area_chart');
                case 'table': return t('area_table');
                case 'image': return t('area_image');
                case 'file_download': return t('area_file_download');
                default: return t('area_component');
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
                        <div key={idx} className="bg-blue-50 dark:bg-[#1a2332] rounded-lg border border-blue-100 dark:border-[#264f78]">
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
                            className="bg-purple-50 dark:bg-[#2a1e2e] rounded-lg p-3 border border-purple-100 dark:border-[#5a3d5f]"
                        >
                            <SmartInsight
                                text={insight.text || ''}
                                icon={insight.icon || 'lightbulb'}
                                threadId={activeThreadId || undefined}
                                onClick={() => {
                                    const dsId = insight.dataSourceId;
                                    const dsName = insight.sourceName || '';
                                    logger.warn(`[onClick1] dsId=${dsId}, dsName=${dsName}`);
                                    if (dsId) {
                                        // 使用与手工"开始新分析"相同的流程
                                        logger.warn(`[onClick1] Emitting start-new-chat event...`);
                                        EventsEmit('start-new-chat', {
                                            dataSourceId: dsId,
                                            dataSourceName: dsName,
                                            sessionName: `分析: ${dsName}`,
                                            keepChatOpen: true
                                        });
                                        logger.warn(`[onClick1] Event emitted`);
                                    } else if (onInsightClick) {
                                        onInsightClick(insight);
                                    }
                                }}
                            />
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
                                threadId={activeThreadId || undefined}
                            />
                        );
                    }
                    return null;
                case 'chart':
                    if (item.data?.type === 'echarts' && typeof item.data.data === 'string') {
                        try {
                            const options = safeParseEChartsConfig(item.data.data);
                            return (
                                <div 
                                    className="cursor-zoom-in group/chart relative"
                                    onDoubleClick={(e) => {
                                        e.stopPropagation();
                                        setModalChartOptions(options);
                                        setChartModalOpen(true);
                                    }}
                                    title={t('double_click_to_zoom')}
                                >
                                    <Chart options={options} height={`${item.h - 32}px`} />
                                    <div className="absolute top-2 right-2 opacity-0 group-hover/chart:opacity-100 transition-opacity bg-black/50 text-white text-xs px-2 py-1 rounded">
                                        {t('double_click_to_zoom')}
                                    </div>
                                </div>
                            );
                        } catch (e) {
                            return <div className="text-red-500">{t('chart_error')}</div>;
                        }
                    } else if (item.data?.type === 'image') {
                        return (
                            <div 
                                className="cursor-zoom-in group/img relative w-full h-full"
                                onDoubleClick={(e) => {
                                    e.stopPropagation();
                                    handleImageDoubleClick(item.data.data);
                                }}
                                title={t('double_click_to_zoom')}
                            >
                                <img 
                                    src={item.data.data} 
                                    alt={t('chart_image')} 
                                    className="w-full h-full object-contain"
                                />
                                <div className="absolute top-2 right-2 opacity-0 group-hover/img:opacity-100 transition-opacity bg-black/50 text-white text-xs px-2 py-1 rounded">
                                    {t('double_click_to_zoom')}
                                </div>
                            </div>
                        );
                    }
                    return null;
                case 'table':
                    if (Array.isArray(item.data)) {
                        return <DataTable data={item.data} columns={item.columns} />;
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
                    rounded-xl overflow-hidden bg-white dark:bg-[#252526]
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
                            title={t('remove_component')}
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
                        title={t('drag_to_resize')}
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
            if (!dashboardData.hasMetrics || dashboardData.metrics.length === 0) {
                return <div className="p-4 text-center text-slate-400 dark:text-[#808080] text-sm">暂无指标数据</div>;
            }
            const metrics = dashboardData.metrics;
            return (
                <div className="grid grid-cols-4 gap-2 p-2">
                    {metrics.map((metric, idx: number) => (
                        <div key={idx} className="bg-blue-50 dark:bg-[#1a2332] rounded-lg border border-blue-100 dark:border-[#264f78]">
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
            if (!dashboardData.hasInsights || dashboardData.insights.length === 0) {
                return <div className="p-4 text-center text-slate-400 dark:text-[#808080] text-sm">暂无洞察数据</div>;
            }
            const insights = dashboardData.insights;
            return (
                <div className="grid grid-cols-3 gap-2 p-2">
                    {insights.map((insight, idx: number) => (
                        <div 
                            key={idx} 
                            className="bg-purple-50 dark:bg-[#2a1e2e] rounded-lg p-3 border border-purple-100 dark:border-[#5a3d5f]"
                        >
                            <SmartInsight
                                text={insight.text || ''}
                                icon={insight.icon || 'lightbulb'}
                                threadId={activeThreadId || undefined}
                                onClick={() => {
                                    const dsId = insight.dataSourceId;
                                    const dsName = insight.sourceName || '';
                                    logger.warn(`[onClick2] dsId=${dsId}, dsName=${dsName}`);
                                    if (dsId) {
                                        // 使用与手工"开始新分析"相同的流程
                                        logger.warn(`[onClick2] Emitting start-new-chat event...`);
                                        EventsEmit('start-new-chat', {
                                            dataSourceId: dsId,
                                            dataSourceName: dsName,
                                            sessionName: `分析: ${dsName}`,
                                            keepChatOpen: true
                                        });
                                        logger.warn(`[onClick2] Event emitted`);
                                    } else if (onInsightClick) {
                                        onInsightClick(insight);
                                    }
                                }}
                            />
                        </div>
                    ))}
                </div>
            );
        };

        // 渲染图表
        const renderChart = () => {
            if (!dashboardData.hasECharts || parsedEChartsConfigs.length === 0) {
                return <div className="p-4 text-center text-slate-400 dark:text-[#808080] text-sm">暂无图表数据</div>;
            }
            
            return (
                <div className="space-y-4">
                    {parsedEChartsConfigs.map((options, index) => {
                        if (!options) {
                            return <div key={index} className="text-red-500 p-4">{t('chart_error')}</div>;
                        }
                        try {
                            const chartStepDescription = dashboardData.allEChartsMetadata?.[index]?.step_description;
                            
                            return (
                                <div key={index}>
                                    {chartStepDescription && (
                                        <div className="px-3 py-1.5 text-xs text-slate-500 dark:text-[#999]">📋 {chartStepDescription}</div>
                                    )}
                                    <div 
                                        className="cursor-zoom-in group relative"
                                        onDoubleClick={() => {
                                            setModalChartOptions(options);
                                            setChartModalOpen(true);
                                        }}
                                        title={t('double_click_to_zoom')}
                                    >
                                        <Chart options={options} height="300px" />
                                        <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity bg-black/50 text-white text-xs px-2 py-1 rounded">
                                            {t('double_click_to_zoom')}
                                        </div>
                                    </div>
                                </div>
                            );
                        } catch (e) {
                            logger.error(`[renderChart] Failed to render chart ${index}: ${e}`);
                            console.error(`Failed to render chart ${index}:`, e);
                            return <div key={index} className="text-red-500 p-4">{t('chart_error')}: {String(e)}</div>;
                        }
                    })}
                </div>
            );
        };

        // 渲染表格
        const renderTable = () => {
            // 直接使用 dashboardData 中的表格数据
            if (!dashboardData.hasTables || !dashboardData.allTableData || dashboardData.allTableData.length === 0) {
                return <div className="p-4 text-center text-slate-400 text-sm">{t('no_data_available')}</div>;
            }

            const validTables: { data: typeof dashboardData.allTableData[0]; originalIndex: number }[] = [];
            dashboardData.allTableData.forEach((t, i) => {
                if (t.rows && t.rows.length > 0) validTables.push({ data: t, originalIndex: i });
            });
            const totalTables = validTables.length;
            
            // 渲染所有表格，每个表格都显示标题
            return (
                <div className="space-y-4">
                    {validTables.map(({ data: tableData, originalIndex }, index) => {
                        // 优先使用后端提供的标题，否则从列名生成
                        const cols = tableData.columns || Object.keys(tableData.rows[0] || {});
                        const titleHint = cols.slice(0, 3).join(' / ') + (cols.length > 3 ? ' ...' : '');
                        
                        // 使用后端提供的 title，如果没有则生成默认标题
                        let tableTitle: string;
                        const hasBackendTitle = tableData.title && tableData.title.trim();
                        if (hasBackendTitle) {
                            // 使用后端提供的标题
                            // 如果标题已经以数字编号开头（如 "1. xxx"），不再添加额外编号
                            const startsWithNumber = /^\d+[\.\、]/.test(tableData.title!.trim());
                            tableTitle = (totalTables > 1 && !startsWithNumber)
                                ? `${index + 1}. ${tableData.title}`
                                : tableData.title!;
                        } else {
                            // 生成默认标题
                            tableTitle = totalTables > 1
                                ? `${t('area_table')} ${index + 1}${titleHint ? ` — ${titleHint}` : ''}`
                                : `${t('area_table')}${titleHint ? ` — ${titleHint}` : ''}`;
                        }
                        
                        // 从 metadata 中读取 step_description
                        const stepDescription = dashboardData.allTableMetadata?.[originalIndex]?.step_description;
                        
                        // 显示标题栏的条件：多表格时总是显示，单表格时如果有后端提供的标题或 step_description 也显示
                        const showTitleBar = totalTables > 1 || hasBackendTitle || !!stepDescription;
                        
                        return (
                            <div key={index} className="border border-slate-200 dark:border-[#3c3c3c] rounded-lg overflow-hidden">
                                {/* 表格标题栏 */}
                                {showTitleBar && (
                                    <div className="px-3 py-2 bg-slate-50 dark:bg-[#252526] border-b border-slate-200 dark:border-[#3c3c3c]">
                                        <div className="flex items-center gap-2">
                                            <span className="flex-1 text-sm font-medium text-slate-700 dark:text-[#d4d4d4]">{renderInlineMarkdown(tableTitle)}</span>
                                            <span className="text-xs text-slate-400 dark:text-[#808080]">{tableData.rows.length} {t('rows')}</span>
                                        </div>
                                        {stepDescription && (
                                            <div className="mt-1 text-xs text-slate-500 dark:text-[#999]">📋 {stepDescription}</div>
                                        )}
                                    </div>
                                )}
                                <DataTable data={tableData.rows} columns={tableData.columns} />
                            </div>
                        );
                    })}
                </div>
            );
        };

        // 渲染文件下载 - 带预览图和下载功能
        const renderFileDownload = () => {
            if (!sessionFiles || sessionFiles.length === 0) {
                return <div className="p-4 text-center text-slate-400 dark:text-[#808080] text-sm">暂无文件</div>;
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
                        const isExcel = ['xlsx', 'xls'].includes(ext);
                        const isPPT = ext === 'pptx';
                        const isCsv = file.type === 'csv' || ext === 'csv';
                        const preview = filePreviews[file.path];
                        const isLoading = filePreviewsLoading[file.path];
                        
                        return (
                            <div 
                                key={idx} 
                                className="flex flex-col bg-white dark:bg-[#252526] rounded-lg border border-slate-200 dark:border-[#3c3c3c] overflow-hidden hover:border-blue-400 dark:hover:border-[#5b8ab5] hover:shadow-lg cursor-pointer transition-all group"
                                onClick={() => handleFileDownload(file)}
                                title={`点击下载: ${file.name}`}
                            >
                                {/* 预览区域 */}
                                <div className="h-32 bg-slate-100 dark:bg-[#2d2d30] flex items-center justify-center overflow-hidden">
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
                                    ) : isExcel || isCsv ? (
                                        <div className="flex flex-col items-center text-green-600">
                                            <FileSpreadsheet size={32} />
                                            <span className="text-xs mt-1">{isExcel ? 'Excel' : 'CSV'}</span>
                                        </div>
                                    ) : isPPT ? (
                                        <div className="flex flex-col items-center text-orange-600">
                                            <Presentation size={32} />
                                            <span className="text-xs mt-1">PPT</span>
                                        </div>
                                    ) : (
                                        <div className="flex flex-col items-center text-orange-600">
                                            <FileText size={32} />
                                            <span className="text-xs mt-1 uppercase">{ext || t('file_label')}</span>
                                        </div>
                                    )}
                                </div>
                                
                                {/* 文件信息 */}
                                <div className="p-2 border-t border-slate-100 dark:border-[#3c3c3c]">
                                    <div className="flex items-center gap-1.5">
                                        {getFileIcon(file.name, file.type)}
                                        <span className="text-xs text-slate-700 dark:text-[#d4d4d4] truncate flex-1" title={file.name}>
                                            {file.name}
                                        </span>
                                    </div>
                                    <div className="flex items-center justify-between mt-1">
                                        <span className="text-xs text-slate-400 dark:text-[#808080]">
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
                    // 直接使用 dashboardData.images
                    const images = dashboardData.images;
                    
                    if (images.length === 0) {
                        return <div className="p-4 text-center text-slate-400 dark:text-[#808080] text-sm">暂无图片</div>;
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
                                title={t('double_click_to_zoom')}
                            >
                                <img src={currentImage} alt={`Image ${validIndex + 1}`} className="max-w-full max-h-full object-contain" />
                                <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity bg-black/50 text-white text-xs px-2 py-1 rounded">
                                    {t('double_click_to_zoom')}
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
                                        title={t('previous_chart')}
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
                                        title={t('next_chart')}
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
                    flex flex-col rounded-xl overflow-hidden bg-white dark:bg-[#252526] shadow-md
                    hover:shadow-lg transition-shadow duration-200
                    w-full
                `}
            >
                {/* 区域标题条 - 表格区域显示表格数量 */}
                <div className={`
                    flex-shrink-0 px-3 py-1.5 flex items-center gap-1.5
                    ${componentInfo.bgColor} ${componentInfo.borderColor}
                    border-b
                `}>
                    <span className="text-base">{componentInfo.icon}</span>
                    <span className={`text-sm font-medium ${componentInfo.textColor}`}>
                        {item.type === 'table' && dashboardData.allTableData.length > 1
                            ? `${areaTitle} (${dashboardData.allTableData.filter(t => t.rows && t.rows.length > 0).length})`
                            : item.type === 'table' && dashboardData.allTableData.length === 1 && dashboardData.allTableData[0]?.title
                                ? renderInlineMarkdown(dashboardData.allTableData[0].title)
                                : areaTitle
                        }
                    </span>
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
            case 'metric': return t('area_metric');
            case 'insight': return t('area_insight');
            case 'chart': return t('area_chart');
            case 'table': return t('area_table');
            case 'image': return t('area_image');
            case 'file_download': return t('area_file_download');
            default: return t('area_component');
        }
    };

    return (
        <div className="relative h-full w-full bg-slate-50 dark:bg-[#1e1e1e] flex flex-col">
            {/* 顶部标题栏 */}
            <div className="flex-shrink-0 bg-white dark:bg-[#252526] border-b border-slate-200 dark:border-[#3c3c3c] pr-6 pl-4 py-3">
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
                                    ? 'bg-green-50 border border-green-300 text-green-700 hover:bg-green-100 dark:bg-[#2d3d2d] dark:border-[#3d5a3d] dark:text-[#6a9955] dark:hover:bg-[#3d5a3d]' 
                                    : 'bg-slate-50 border border-slate-200 text-slate-600 hover:bg-slate-100 hover:border-slate-300 dark:bg-[#2d2d30] dark:border-[#3c3c3c] dark:text-[#808080] dark:hover:bg-[#3c3c3c]'
                                }
                            `}
                            title={isEditMode ? t('save_layout') : t('edit_layout')}
                        >
                            {isEditMode ? (
                                <>
                                    <Save size={14} />
                                    <span>{t('save')}</span>
                                </>
                            ) : (
                                <>
                                    <Edit3 size={14} />
                                    <span>{t('edit')}</span>
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
                                title={t('auto_arrange')}
                            >
                                <span>📐</span>
                                <span>排列</span>
                            </button>
                        )}

                        {/* 分隔线 */}
                        <div className="h-6 w-px bg-slate-200 dark:bg-[#3c3c3c]"></div>

                        {/* 标题和用户请求 */}
                        <div className="flex flex-col">
                            <div className="flex items-center gap-3">
                                <h1 className="text-lg font-semibold text-slate-700 dark:text-[#d4d4d4] flex items-center gap-2">
                                    <span>📊</span>
                                    {t('smart_analysis_dashboard')}
                                    {isEditMode && (
                                        <span className="text-xs font-normal px-2 py-0.5 bg-amber-50 text-amber-600 border border-amber-200 rounded-full">
                                            {t('editing')}
                                        </span>
                                    )}
                                </h1>
                                {/* Global Analysis Status - Requirements: 3.1, 3.2, 3.3, 3.4 */}
                                <GlobalAnalysisStatus />

                            </div>
                            {userRequestText && (
                                <p className="text-xs text-slate-500 dark:text-[#808080] mt-0.5 max-w-md truncate" title={userRequestText}>
                                    {userRequestText}
                                </p>
                            )}
                        </div>
                    </div>

                    {/* 右侧：数据导出按钮 - 仅在有可导出内容时显示 */}
                    {hasExportableContent() && (
                        <div className="flex items-center gap-2">
                            <div className="relative export-dropdown-container">
                                <button
                                    onClick={() => {
                                        if (preparedReportId) {
                                            setExportDropdownOpen(!exportDropdownOpen);
                                        } else {
                                            prepareReport();
                                        }
                                    }}
                                    className={`px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition-all text-sm cursor-pointer ${
                                        preparedReportId
                                            ? 'bg-green-50 border border-green-200 text-green-600 hover:bg-green-100'
                                            : 'bg-purple-50 border border-purple-200 text-purple-600 hover:bg-purple-100'
                                    }`}
                                    title={preparedReportId ? (t('report_ready_title')) : (t('reports_button_title'))}
                                >
                                    <ClipboardList size={14} />
                                    <span>{preparedReportId ? (t('export_report')) : (t('reports'))}</span>
                                </button>

                                {/* 格式选择下拉菜单 - 仅在报告已生成后显示 */}
                                {exportDropdownOpen && preparedReportId && (
                                    <div className="absolute right-0 top-full mt-2 w-56 bg-white dark:bg-[#252526] rounded-xl shadow-xl border border-slate-200 dark:border-[#3c3c3c] py-1.5 z-50">
                                        <div className="px-3 py-1.5 text-xs font-medium text-slate-400 dark:text-[#808080] uppercase tracking-wider">{t('select_export_format')}</div>
                                        <button
                                            onClick={() => exportReportAs('word')}
                                            className="w-full flex items-center gap-2.5 px-3 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                        >
                                            <FileChartColumn size={16} className="flex-shrink-0 text-indigo-500" />
                                            <span className="whitespace-nowrap">{t('export_as_word')}</span>
                                        </button>
                                        <button
                                            onClick={() => exportReportAs('pdf')}
                                            className="w-full flex items-center gap-2.5 px-3 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                        >
                                            <FileChartColumn size={16} className="flex-shrink-0 text-rose-500" />
                                            <span className="whitespace-nowrap">{t('export_as_pdf')}</span>
                                        </button>
                                        <div className="border-t border-slate-100 dark:border-[#3c3c3c] mt-1 pt-1">
                                            <button
                                                onClick={() => { setPreparedReportId(null); setExportDropdownOpen(false); prepareReport(); }}
                                                className="w-full flex items-center gap-2.5 px-3 py-2 text-sm text-slate-400 dark:text-[#808080] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                            >
                                                <span className="whitespace-nowrap">{t('regenerate_report')}</span>
                                            </button>
                                        </div>
                                    </div>
                                )}
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* 编辑模式下的控件库面板 - 只显示有数据的组件类型 */}
            {isEditMode && (
                <div className="flex-shrink-0 bg-white dark:bg-[#252526] border-b border-slate-200 dark:border-[#3c3c3c] pr-6 pl-4 py-3">
                    <div className="flex items-center gap-4">
                        <span className="text-sm font-medium text-slate-600 dark:text-[#808080]">控件库：</span>
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
                className="relative flex-1 overflow-auto pr-6 py-6"
                onMouseMove={handleDrag}
                onMouseUp={handleDragEnd}
                onMouseLeave={handleDragEnd}
                onClick={onDashboardClick}
            >
                {getDisplayLayout().length === 0 ? (
                    <div className="flex items-center justify-center h-full">
                        <div className="text-center text-slate-400 dark:text-[#808080]">
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
                            {t('edit_mode_hint')}
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

            {/* 报告生成进度遮罩 */}
            {isGeneratingReport && (
                <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60]">
                    <div className="bg-white dark:bg-[#252526] rounded-2xl shadow-2xl p-8 flex flex-col items-center gap-5 min-w-[320px]">
                        {/* 旋转动画 */}
                        <div className="relative w-16 h-16">
                            <div className="absolute inset-0 rounded-full border-4 border-slate-200 dark:border-[#3c3c3c]"></div>
                            <div className="absolute inset-0 rounded-full border-4 border-transparent border-t-[#5b7a9d] animate-spin"></div>
                            <div className="absolute inset-2 rounded-full border-4 border-transparent border-t-[#7b9bb8] animate-spin" style={{ animationDirection: 'reverse', animationDuration: '1.5s' }}></div>
                        </div>
                        {/* 进度条 */}
                        <div className="w-full bg-slate-100 dark:bg-[#3c3c3c] rounded-full h-2 overflow-hidden">
                            <div className="h-full bg-gradient-to-r from-[#5b7a9d] via-[#7b9bb8] to-[#5b7a9d] rounded-full animate-pulse" style={{ width: '100%', animation: 'reportProgress 2s ease-in-out infinite' }}></div>
                        </div>
                        <p className="text-sm font-medium text-slate-700 dark:text-[#d4d4d4]">{t('generate_report_processing')}</p>
                        <p className="text-xs text-slate-400 dark:text-[#808080]">{t('generate_report_llm_hint')}</p>
                        <style>{`
                            @keyframes reportProgress {
                                0% { transform: translateX(-100%); }
                                50% { transform: translateX(0%); }
                                100% { transform: translateX(100%); }
                            }
                        `}</style>
                    </div>
                </div>
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
