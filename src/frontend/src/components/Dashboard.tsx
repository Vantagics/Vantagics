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

interface DashboardProps {
    data: main.DashboardData | null;
    activeChart?: { type: 'echarts' | 'image', data: string } | null;
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
        return null;
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
