import React, { useState } from 'react';
import DashboardLayout from './DashboardLayout';
import MetricCard from './MetricCard';
import SmartInsight from './SmartInsight';
import Chart from './Chart';
import ImageModal from './ImageModal';
import ChartModal from './ChartModal';
import ConfirmationModal from './ConfirmationModal';
import { main } from '../../wailsjs/go/models';
import { useLanguage } from '../i18n';
import { EventsEmit } from '../../wailsjs/runtime/runtime';
import { Download, Table, BarChart3 } from 'lucide-react';

interface DashboardProps {
    data: main.DashboardData | null;
    activeChart?: { type: 'echarts' | 'image' | 'table' | 'csv', data: any } | null;
}

const Dashboard: React.FC<DashboardProps> = ({ data, activeChart }) => {
    const { t } = useLanguage();
    const [imageModalOpen, setImageModalOpen] = useState(false);
    const [chartModalOpen, setChartModalOpen] = useState(false);
    const [confirmModal, setConfirmModal] = useState<{ isOpen: boolean, insight: any | null }>({ isOpen: false, insight: null });

    if (!data) {
        return (
            <div className="flex items-center justify-center h-full">
                <div className="animate-pulse text-slate-400">{t('loading_insights')}</div>
            </div>
        );
    }

    const renderChart = () => {
        if (!activeChart) return null;

        if (activeChart.type === 'image') {
            return (
                <div
                    className="w-full bg-white rounded-xl border border-slate-200 p-4 shadow-sm mb-8 flex justify-center cursor-zoom-in group relative"
                    onDoubleClick={() => setImageModalOpen(true)}
                    title="Double click to expand"
                >
                    <img src={activeChart.data} alt="Analysis Chart" className="max-h-[400px] object-contain group-hover:scale-[1.01] transition-transform duration-300" />
                    <div className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity bg-black/5 pointer-events-none rounded-xl">
                        <span className="bg-white/90 px-3 py-1 rounded-full text-xs font-medium text-slate-600 shadow-sm backdrop-blur-sm">Double click to zoom</span>
                    </div>
                </div>
            );
        }

        if (activeChart.type === 'echarts') {
            try {
                const options = JSON.parse(activeChart.data);
                return (
                    <div
                        className="mb-8 cursor-zoom-in group relative"
                        onDoubleClick={() => setChartModalOpen(true)}
                        title="Double click to expand"
                    >
                        <Chart options={options} height="400px" />
                        <div className="absolute top-4 right-4 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
                            <span className="bg-slate-800/80 text-white px-3 py-1 rounded-full text-xs font-medium shadow-sm backdrop-blur-sm">Double click to expand</span>
                        </div>
                    </div>
                );
            } catch (e) {
                console.error("Failed to parse ECharts options for dashboard", e);
                return null;
            }
        }

        if (activeChart.type === 'table') {
            const tableData = activeChart.data as any[];
            if (!tableData || tableData.length === 0) return null;

            const columns = Object.keys(tableData[0]);
            return (
                <div className="w-full bg-white rounded-xl border border-slate-200 shadow-sm mb-8 overflow-hidden">
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

        if (activeChart.type === 'csv') {
            return (
                <div className="w-full bg-white rounded-xl border border-slate-200 p-4 shadow-sm mb-8">
                    <div className="flex items-center gap-3">
                        <div className="bg-green-100 p-2 rounded-lg">
                            <Download className="w-5 h-5 text-green-600" />
                        </div>
                        <div className="flex-1">
                            <p className="text-sm font-medium text-slate-700">{t('data_file_ready') || 'Data File Ready'}</p>
                            <p className="text-xs text-slate-400">{t('click_to_download') || 'Click to download'}</p>
                        </div>
                        <a
                            href={activeChart.data}
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

    const handleInsightClick = (insight: any) => {
        if (insight.data_source_id) {
            setConfirmModal({ isOpen: true, insight: insight });
        } else {
            EventsEmit("analyze-insight", insight.text);
        }
    };

    const confirmAnalysis = () => {
        const insight = confirmModal.insight;
        if (insight && insight.data_source_id) {
             EventsEmit('start-new-chat', {
                dataSourceId: insight.data_source_id,
                sessionName: `${t('analysis_session_prefix')}${insight.source_name || insight.text}`
            });
        }
    };

    return (
        <div className="flex-1 flex flex-col h-full overflow-hidden">
            <header className="px-6 py-8" style={{ '--wails-draggable': 'drag' } as any}>
                <h1 className="text-2xl font-bold text-slate-800">{t('smart_dashboard')}</h1>
                <p className="text-slate-500">{t('welcome_back')}</p>
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
                    imageUrl={activeChart?.type === 'image' ? activeChart.data : ''}
                    onClose={() => setImageModalOpen(false)}
                />

                {activeChart?.type === 'echarts' && (
                    <ChartModal
                        isOpen={chartModalOpen}
                        options={JSON.parse(activeChart.data)}
                        onClose={() => setChartModalOpen(false)}
                    />
                )}

                <ConfirmationModal
                    isOpen={confirmModal.isOpen}
                    title={t('start_analysis_title')}
                    message={confirmModal.insight ? t('start_analysis_confirm').replace('{0}', confirmModal.insight.text.split(':')[0]) : ''}
                    onClose={() => setConfirmModal({ isOpen: false, insight: null })}
                    onConfirm={confirmAnalysis}
                />

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
            </div>
        </div>
    );
};

export default Dashboard;
