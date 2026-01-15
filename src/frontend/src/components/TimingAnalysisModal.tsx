import React from 'react';
import { X, Clock, Zap } from 'lucide-react';

interface TimingAnalysisModalProps {
    isOpen: boolean;
    onClose: () => void;
    timingData: any;
    messageContent: string;
}

const TimingAnalysisModal: React.FC<TimingAnalysisModalProps> = ({ isOpen, onClose, timingData, messageContent }) => {

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
        performanceRating = '优秀';
        performanceColor = 'text-green-600';
    } else if (totalSeconds < 60) {
        performanceRating = '良好';
        performanceColor = 'text-blue-600';
    } else if (totalSeconds < 120) {
        performanceRating = '一般';
        performanceColor = 'text-yellow-600';
    } else {
        performanceRating = '较慢';
        performanceColor = 'text-red-600';
    }

    // Format duration helper
    const formatDuration = (seconds: number): string => {
        if (seconds < 60) {
            return `${seconds.toFixed(1)}秒`;
        }
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}分${secs}秒`;
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
                <div className="flex items-center justify-between p-6 border-b border-slate-200 bg-gradient-to-r from-blue-50 to-indigo-50">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-blue-100 rounded-lg">
                            <Clock className="w-6 h-6 text-blue-600" />
                        </div>
                        <div>
                            <h2 className="text-xl font-bold text-slate-800">耗时分析</h2>
                            <p className="text-sm text-slate-500">Performance Analysis</p>
                        </div>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-2 hover:bg-white/50 rounded-lg transition-colors"
                    >
                        <X className="w-5 h-5 text-slate-500" />
                    </button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-6 space-y-6">
                    {/* Total Time Card */}
                    <div className="bg-gradient-to-br from-blue-50 to-indigo-50 rounded-xl p-6 border border-blue-100">
                        <div className="flex items-center justify-between mb-4">
                            <h3 className="text-lg font-semibold text-slate-800">总耗时</h3>
                            <div className={`px-3 py-1 rounded-full text-sm font-medium ${performanceColor} bg-white`}>
                                {performanceRating}
                            </div>
                        </div>
                        <div className="flex items-baseline gap-2">
                            <span className="text-5xl font-bold text-blue-600">{totalMinutes}</span>
                            <span className="text-2xl text-slate-600">分</span>
                            <span className="text-5xl font-bold text-blue-600">{totalSecondsRemainder}</span>
                            <span className="text-2xl text-slate-600">秒</span>
                        </div>
                        <div className="mt-2 text-sm text-slate-500">
                            总计 {totalSeconds.toFixed(2)} 秒
                        </div>
                    </div>

                    {/* Analysis Details */}
                    <div className="space-y-3">
                        <h3 className="text-lg font-semibold text-slate-800 flex items-center gap-2">
                            <Zap className="w-5 h-5 text-yellow-500" />
                            分析详情
                        </h3>
                        
                        <div className="bg-slate-50 rounded-lg p-4 space-y-3">
                            <div className="flex justify-between items-center py-2 border-b border-slate-200">
                                <span className="text-slate-600">分析类型</span>
                                <span className="font-medium text-slate-800">
                                    {analysisType === 'eino_service' ? 'AI 智能分析' : '标准分析'}
                                </span>
                            </div>
                            
                            <div className="flex justify-between items-center py-2 border-b border-slate-200">
                                <span className="text-slate-600">完成时间</span>
                                <span className="font-medium text-slate-800">{formattedTimestamp}</span>
                            </div>
                            
                            <div className="flex justify-between items-center py-2">
                                <span className="text-slate-600">响应长度</span>
                                <span className="font-medium text-slate-800">{messageContent.length} 字符</span>
                            </div>
                        </div>
                    </div>

                    {/* Stage Breakdown */}
                    {stages.length > 0 && (
                        <div className="space-y-3">
                            <h3 className="text-lg font-semibold text-slate-800 flex items-center gap-2">
                                <svg className="w-5 h-5 text-indigo-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                                </svg>
                                各阶段耗时
                            </h3>
                            
                            <div className="space-y-3">
                                {stages.map((stage: any, index: number) => {
                                    const colors = stageColors[index % stageColors.length];
                                    return (
                                        <div key={index} className={`${colors.bg} rounded-lg p-4`}>
                                            <div className="flex justify-between items-center mb-2">
                                                <div className="flex items-center gap-2">
                                                    <span className={`font-semibold ${colors.text}`}>{stage.name}</span>
                                                    <span className="text-xs text-slate-500">{stage.description}</span>
                                                </div>
                                                <div className="flex items-center gap-3">
                                                    <span className={`font-bold ${colors.text}`}>{formatDuration(stage.duration)}</span>
                                                    <span className={`px-2 py-1 rounded-full text-xs font-bold ${colors.text} bg-white`}>
                                                        {stage.percentage.toFixed(0)}%
                                                    </span>
                                                </div>
                                            </div>
                                            {/* Progress bar */}
                                            <div className="w-full bg-white rounded-full h-2 overflow-hidden">
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
                    <div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
                        <h4 className="font-semibold text-amber-800 mb-2">💡 性能提示</h4>
                        <ul className="text-sm text-amber-700 space-y-1">
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
                <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end">
                    <button
                        onClick={onClose}
                        className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
                    >
                        关闭
                    </button>
                </div>
            </div>
        </div>
    );
};

export default TimingAnalysisModal;
