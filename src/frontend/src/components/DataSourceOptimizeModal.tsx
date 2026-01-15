import React, { useState, useEffect } from 'react';
import { X, Zap, CheckCircle, XCircle, Loader, Database, AlertTriangle } from 'lucide-react';
import { GetOptimizeSuggestions, ApplyOptimizeSuggestions } from '../../wailsjs/go/main/App';

interface IndexSuggestion {
    table_name: string;
    index_name: string;
    columns: string[];
    reason: string;
    sql_command: string;
    applied: boolean;
    error?: string;
}

interface SuggestionsResult {
    data_source_id: string;
    data_source_name: string;
    suggestions: IndexSuggestion[];
    success: boolean;
    error?: string;
}

interface OptimizeResult {
    data_source_id: string;
    data_source_name: string;
    suggestions: IndexSuggestion[];
    summary: string;
    success: boolean;
    error?: string;
}

interface DataSourceOptimizeModalProps {
    isOpen: boolean;
    onClose: () => void;
    dataSourceId: string;
    dataSourceName: string;
}

type Step = 'loading' | 'confirm' | 'applying' | 'complete';

const DataSourceOptimizeModal: React.FC<DataSourceOptimizeModalProps> = ({
    isOpen,
    onClose,
    dataSourceId,
    dataSourceName
}) => {
    const [step, setStep] = useState<Step>('loading');
    const [suggestions, setSuggestions] = useState<IndexSuggestion[]>([]);
    const [result, setResult] = useState<OptimizeResult | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [applyProgress, setApplyProgress] = useState(0);

    useEffect(() => {
        if (isOpen) {
            // Reset state when modal opens
            setStep('loading');
            setSuggestions([]);
            setResult(null);
            setError(null);
            setApplyProgress(0);
            // Load suggestions
            loadSuggestions();
        }
    }, [isOpen, dataSourceId]);

    const loadSuggestions = async () => {
        setStep('loading');
        setError(null);
        
        try {
            const suggestionsResult = await GetOptimizeSuggestions(dataSourceId);
            if (suggestionsResult.success && suggestionsResult.suggestions.length > 0) {
                setSuggestions(suggestionsResult.suggestions);
                setStep('confirm');
            } else {
                setError('未找到优化建议，数据库可能已经优化或不需要优化');
                setStep('complete');
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
            setStep('complete');
        }
    };

    const applyOptimizations = async () => {
        setStep('applying');
        setApplyProgress(0);
        
        try {
            // Simulate progress
            const progressInterval = setInterval(() => {
                setApplyProgress(prev => Math.min(prev + 10, 90));
            }, 200);
            
            const optimizeResult = await ApplyOptimizeSuggestions(dataSourceId, suggestions);
            
            clearInterval(progressInterval);
            setApplyProgress(100);
            
            setResult(optimizeResult);
            setStep('complete');
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
            setStep('complete');
        }
    };

    const handleClose = () => {
        onClose();
        // Emit event to refresh data sources list
        if (step === 'complete' && result?.success) {
            // Use Wails event system to notify Sidebar to refresh
            import('../../wailsjs/runtime/runtime').then(({ EventsEmit }) => {
                EventsEmit('data-source-optimized', dataSourceId);
            });
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[10000]" onClick={handleClose}>
            <div 
                className="bg-white rounded-xl shadow-2xl w-full max-w-4xl max-h-[85vh] overflow-hidden flex flex-col"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-slate-200 bg-gradient-to-r from-amber-50 to-orange-50">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-amber-100 rounded-lg">
                            <Zap className="w-6 h-6 text-amber-600" />
                        </div>
                        <div>
                            <h2 className="text-xl font-bold text-slate-800">数据源优化</h2>
                            <p className="text-sm text-slate-500">{dataSourceName}</p>
                        </div>
                    </div>
                    <button
                        onClick={handleClose}
                        className="p-2 hover:bg-white/50 rounded-lg transition-colors"
                    >
                        <X className="w-5 h-5 text-slate-500" />
                    </button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-6">
                    {/* Loading Step */}
                    {step === 'loading' && (
                        <div className="flex flex-col items-center justify-center py-12">
                            <Loader className="w-12 h-12 text-amber-500 animate-spin mb-4" />
                            <p className="text-slate-600 text-lg font-medium">正在分析数据库结构...</p>
                            <p className="text-slate-400 text-sm mt-2">AI 正在生成索引优化建议</p>
                        </div>
                    )}

                    {/* Confirm Step */}
                    {step === 'confirm' && suggestions.length > 0 && (
                        <div className="space-y-4">
                            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                                <div className="flex items-start gap-3">
                                    <AlertTriangle className="w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5" />
                                    <div>
                                        <h3 className="font-semibold text-blue-800 mb-1">确认优化操作</h3>
                                        <p className="text-sm text-blue-600">
                                            系统将执行以下 {suggestions.length} 个索引创建操作。请仔细检查后确认执行。
                                        </p>
                                    </div>
                                </div>
                            </div>

                            <div className="space-y-3">
                                <h3 className="text-lg font-semibold text-slate-800">优化建议</h3>
                                
                                {suggestions.map((suggestion, index) => (
                                    <div 
                                        key={index}
                                        className="border border-slate-200 rounded-lg p-4 bg-slate-50"
                                    >
                                        <div className="flex items-start gap-3">
                                            <div className="flex-shrink-0 w-8 h-8 bg-amber-100 rounded-full flex items-center justify-center">
                                                <span className="text-sm font-bold text-amber-600">{index + 1}</span>
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-center gap-2 mb-2">
                                                    <span className="font-semibold text-slate-800">
                                                        {suggestion.index_name}
                                                    </span>
                                                    <span className="text-xs px-2 py-0.5 bg-slate-200 text-slate-600 rounded">
                                                        {suggestion.table_name}
                                                    </span>
                                                </div>
                                                
                                                <p className="text-sm text-slate-600 mb-2">
                                                    <span className="font-medium">列：</span>
                                                    {suggestion.columns.join(', ')}
                                                </p>
                                                
                                                <p className="text-sm text-slate-600 mb-3">
                                                    <span className="font-medium">原因：</span>
                                                    {suggestion.reason}
                                                </p>
                                                
                                                <div className="bg-slate-800 rounded p-2 overflow-x-auto">
                                                    <code className="text-xs text-green-400 font-mono">
                                                        {suggestion.sql_command}
                                                    </code>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Applying Step */}
                    {step === 'applying' && (
                        <div className="flex flex-col items-center justify-center py-12">
                            <Loader className="w-12 h-12 text-amber-500 animate-spin mb-4" />
                            <p className="text-slate-600 text-lg font-medium mb-4">正在执行优化...</p>
                            
                            {/* Progress Bar */}
                            <div className="w-full max-w-md">
                                <div className="bg-slate-200 rounded-full h-3 overflow-hidden">
                                    <div 
                                        className="bg-gradient-to-r from-amber-500 to-orange-500 h-full transition-all duration-300 ease-out"
                                        style={{ width: `${applyProgress}%` }}
                                    />
                                </div>
                                <p className="text-center text-sm text-slate-500 mt-2">{applyProgress}%</p>
                            </div>
                        </div>
                    )}

                    {/* Complete Step */}
                    {step === 'complete' && (
                        <div className="space-y-4">
                            {error && (
                                <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                                    <div className="flex items-start gap-3">
                                        <XCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
                                        <div>
                                            <h3 className="font-semibold text-red-800 mb-1">优化失败</h3>
                                            <p className="text-sm text-red-600">{error}</p>
                                        </div>
                                    </div>
                                </div>
                            )}

                            {result && (
                                <>
                                    {/* Summary */}
                                    <div className={`rounded-lg p-4 ${result.success ? 'bg-green-50 border border-green-200' : 'bg-amber-50 border border-amber-200'}`}>
                                        <div className="flex items-start gap-3">
                                            {result.success ? (
                                                <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5" />
                                            ) : (
                                                <Database className="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
                                            )}
                                            <div>
                                                <h3 className={`font-semibold mb-1 ${result.success ? 'text-green-800' : 'text-amber-800'}`}>
                                                    {result.summary}
                                                </h3>
                                                <p className={`text-sm ${result.success ? 'text-green-600' : 'text-amber-600'}`}>
                                                    共分析 {result.suggestions.length} 个索引建议
                                                </p>
                                            </div>
                                        </div>
                                    </div>

                                    {/* Detailed Results */}
                                    {result.suggestions.length > 0 && (
                                        <div className="space-y-3">
                                            <h3 className="text-lg font-semibold text-slate-800">执行结果</h3>
                                            
                                            {result.suggestions.map((suggestion, index) => (
                                                <div 
                                                    key={index}
                                                    className={`border rounded-lg p-4 ${
                                                        suggestion.applied 
                                                            ? 'bg-green-50 border-green-200' 
                                                            : 'bg-red-50 border-red-200'
                                                    }`}
                                                >
                                                    <div className="flex items-start gap-3">
                                                        {suggestion.applied ? (
                                                            <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5" />
                                                        ) : (
                                                            <XCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
                                                        )}
                                                        <div className="flex-1 min-w-0">
                                                            <div className="flex items-center gap-2 mb-2">
                                                                <span className="font-semibold text-slate-800">
                                                                    {suggestion.index_name}
                                                                </span>
                                                                <span className="text-xs px-2 py-0.5 bg-slate-200 text-slate-600 rounded">
                                                                    {suggestion.table_name}
                                                                </span>
                                                            </div>
                                                            
                                                            {suggestion.error && (
                                                                <p className="text-sm text-red-600 mt-2">
                                                                    <span className="font-medium">错误：</span>
                                                                    {suggestion.error}
                                                                </p>
                                                            )}
                                                        </div>
                                                    </div>
                                                </div>
                                            ))}
                                        </div>
                                    )}

                                    {/* Tips */}
                                    <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                                        <h4 className="font-semibold text-blue-800 mb-2">💡 优化提示</h4>
                                        <ul className="text-sm text-blue-700 space-y-1">
                                            <li>• 索引可以显著提升查询速度，特别是对大型数据表</li>
                                            <li>• 索引会占用额外的存储空间</li>
                                            <li>• 过多的索引可能会降低写入性能</li>
                                            <li>• 建议定期分析查询性能，调整索引策略</li>
                                        </ul>
                                    </div>
                                </>
                            )}
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-2">
                    {step === 'confirm' && (
                        <>
                            <button
                                onClick={handleClose}
                                className="px-4 py-2 bg-slate-200 text-slate-700 rounded-lg hover:bg-slate-300 transition-colors font-medium"
                            >
                                取消
                            </button>
                            <button
                                onClick={applyOptimizations}
                                className="px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition-colors font-medium flex items-center gap-2"
                            >
                                <Zap className="w-4 h-4" />
                                确认优化
                            </button>
                        </>
                    )}
                    {(step === 'complete' || step === 'loading') && (
                        <button
                            onClick={handleClose}
                            className="px-4 py-2 bg-slate-600 text-white rounded-lg hover:bg-slate-700 transition-colors font-medium"
                        >
                            关闭
                        </button>
                    )}
                </div>
            </div>
        </div>
    );
};

export default DataSourceOptimizeModal;
