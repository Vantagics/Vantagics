import React from 'react';
import { X, Clock, Zap } from 'lucide-react';
import { useLanguage } from '../i18n';

interface TimingAnalysisModalProps {
    isOpen: boolean;
    onClose: () => void;
    timingData: any;
    messageContent: string;
}

const TimingAnalysisModal: React.FC<TimingAnalysisModalProps> = ({ isOpen, onClose, timingData, messageContent }) => {
    const { t } = useLanguage();

    if (!isOpen) return null;

    // Extract timing information
    const totalSeconds = timingData?.total_seconds || 0;
    const totalMinutes = timingData?.total_minutes || 0;
    const totalSecondsRemainder = timingData?.total_seconds_remainder || 0;
    const analysisType = timingData?.analysis_type || 'unknown';
    const timestamp = timingData?.timestamp || 0;
    const stages = timingData?.stages || [];

    // Format timestamp
    const formattedTimestamp = timestamp ? new Date(timestamp * 1000).toLocaleString('zh-CN') : 'N/A';

    // Calculate performance rating
    let performanceRating = '';
    let performanceColor = '';
    if (totalSeconds < 30) {
        performanceRating = t('performance_excellent');
        performanceColor = 'text-green-600';
    } else if (totalSeconds < 60) {
        performanceRating = t('performance_good');
        performanceColor = 'text-blue-600';
    } else if (totalSeconds < 120) {
        performanceRating = t('performance_average');
        performanceColor = 'text-yellow-600';
    } else {
        performanceRating = t('performance_slow');
        performanceColor = 'text-red-600';
    }

    // Format duration helper
    const formatDuration = (seconds: number): string => {
        if (seconds < 60) {
            return `${seconds.toFixed(1)}${t('second')}`;
        }
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}${t('minute')}${secs}${t('second')}`;
    };

    // Stage colors
    const stageColors = [
        { bg: 'bg-blue-100', text: 'text-blue-700', bar: 'bg-blue-500' },
        { bg: 'bg-green-100', text: 'text-green-700', bar: 'bg-green-500' },
        { bg: 'bg-purple-100', text: 'text-purple-700', bar: 'bg-purple-500' },
        { bg: 'bg-gray-100', text: 'text-gray-700', bar: 'bg-gray-500' }
    ];

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[10000]" onClick={onClose}>
            <div
                className="bg-white rounded-xl shadow-2xl w-full max-w-2xl max-h-[80vh] overflow-hidden flex flex-col"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 bg-gradient-to-r from-blue-50 to-indigo-50">
                    <div className="flex items-center gap-2.5">
                        <div className="p-1.5 bg-blue-100 rounded-lg">
                            <Clock className="w-5 h-5 text-blue-600" />
                        </div>
                        <div>
                            <h2 className="text-lg font-bold text-slate-800">{t('timing_analysis')}</h2>
                            <p className="text-xs text-slate-500">{t('performance_analysis')}</p>
                        </div>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-1.5 hover:bg-white/50 rounded-lg transition-colors"
                    >
                        <X className="w-4 h-4 text-slate-500" />
                    </button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-5 space-y-4">
                    {/* Total Time Card */}
                    <div className="bg-gradient-to-br from-blue-50 to-indigo-50 rounded-lg p-4 border border-blue-100">
                        <div className="flex items-center justify-between mb-3">
                            <h3 className="text-base font-semibold text-slate-800">{t('total_time')}</h3>
                            <div className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${performanceColor} bg-white`}>
                                {performanceRating}
                            </div>
                        </div>
                        <div className="flex items-baseline gap-1.5">
                            <span className="text-4xl font-bold text-blue-600">{totalMinutes}</span>
                            <span className="text-lg text-slate-600">{t('minute')}</span>
                            <span className="text-4xl font-bold text-blue-600">{totalSecondsRemainder}</span>
                            <span className="text-lg text-slate-600">{t('second')}</span>
                        </div>
                        <div className="mt-1.5 text-xs text-slate-500">
                            {t('total_time')} {totalSeconds.toFixed(2)} {t('second')}
                        </div>
                    </div>

                    {/* Analysis Details */}
                    <div className="space-y-2">
                        <h3 className="text-sm font-semibold text-slate-800 flex items-center gap-1.5">
                            <Zap className="w-4 h-4 text-yellow-500" />
                            {t('analysis_info')}
                        </h3>

                        <div className="bg-slate-50 rounded-lg p-3 space-y-2">
                            <div className="flex justify-between items-center py-1.5 border-b border-slate-200">
                                <span className="text-xs text-slate-600">{t('analysis_type')}</span>
                                <span className="text-xs font-medium text-slate-800">
                                    {analysisType === 'eino_service' ? 'AI' : 'Standard'}
                                </span>
                            </div>

                            <div className="flex justify-between items-center py-1.5">
                                <span className="text-xs text-slate-600">{t('analysis_time')}</span>
                                <span className="text-xs font-medium text-slate-800">{formattedTimestamp}</span>
                            </div>
                        </div>
                    </div>

                    {/* Stage Breakdown */}
                    {stages.length > 0 && (
                        <div className="space-y-2">
                            <h3 className="text-sm font-semibold text-slate-800 flex items-center gap-1.5">
                                <svg className="w-4 h-4 text-indigo-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                                </svg>
                                {t('stage_breakdown')}
                            </h3>

                            <div className="space-y-2">
                                {stages.map((stage: any, index: number) => {
                                    const colors = stageColors[index % stageColors.length];
                                    return (
                                        <div key={index} className={`${colors.bg} rounded-lg p-3`}>
                                            <div className="flex justify-between items-center mb-1.5">
                                                <div className="flex items-center gap-1.5">
                                                    <span className={`text-sm font-semibold ${colors.text}`}>{stage.name}</span>
                                                    <span className="text-[10px] text-slate-500">{stage.description}</span>
                                                </div>
                                                <div className="flex items-center gap-2">
                                                    <span className={`text-sm font-bold ${colors.text}`}>{formatDuration(stage.duration)}</span>
                                                    <span className={`px-1.5 py-0.5 rounded-full text-[10px] font-bold ${colors.text} bg-white`}>
                                                        {stage.percentage.toFixed(0)}%
                                                    </span>
                                                </div>
                                            </div>
                                            {/* Progress bar */}
                                            <div className="w-full bg-white rounded-full h-1.5 overflow-hidden">
                                                <div
                                                    className={`${colors.bar} h-full rounded-full transition-all duration-500`}
                                                    style={{ width: `${stage.percentage}%` }}
                                                />
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        </div>
                    )}

                    {/* Performance Tips */}
                    <div className="bg-amber-50 border border-amber-200 rounded-lg p-3">
                        <h4 className="text-xs font-semibold text-amber-800 mb-1.5">💡 性能提示</h4>
                        <ul className="text-[11px] text-amber-700 space-y-0.5">
                            {totalSeconds > 120 && (
                                <li>• 分析耗时较长，建议简化查询或优化数据源</li>
                            )}
                            {totalSeconds < 30 && (
                                <li>• 分析速度优秀，系统运行良好</li>
                            )}
                            <li>• 复杂的数据分析可能需要更长时间</li>
                            <li>• 网络状况和 LLM 服务响应速度会影响总耗时</li>
                        </ul>
                    </div>
                </div>

                {/* Footer */}
                <div className="px-5 py-3 border-t border-slate-200 bg-slate-50 flex justify-end">
                    <button
                        onClick={onClose}
                        className="px-5 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
                    >
                        {t('close')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default TimingAnalysisModal;
