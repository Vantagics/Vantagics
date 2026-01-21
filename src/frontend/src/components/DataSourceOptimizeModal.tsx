import React, { useState, useEffect, useRef } from 'react';
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
    const [logs, setLogs] = useState<string[]>([]);
    const logsEndRef = useRef<HTMLDivElement>(null);

    const addLog = (message: string) => {
        const timestamp = new Date().toLocaleTimeString('zh-CN', { hour12: false });
        setLogs(prev => [...prev, `[${timestamp}] ${message}`]);
    };

    useEffect(() => {
        if (logs.length > 0) {
            logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
        }
    }, [logs]);

    useEffect(() => {
        if (isOpen) {
            // Reset state when modal opens
            setStep('loading');
            setSuggestions([]);
            setResult(null);
            setError(null);
            setLogs([]);
            // Load suggestions
            loadSuggestions();
        }
    }, [isOpen, dataSourceId]);

    const loadSuggestions = async () => {
        setStep('loading');
        setError(null);
        addLog('🔍 开始分析数据库结构...');

        try {
            addLog('📊 正在获取表信息和列定义...');
            const suggestionsResult = await GetOptimizeSuggestions(dataSourceId);

            if (suggestionsResult.success && suggestionsResult.suggestions.length > 0) {
                addLog(`✅ 分析完成，找到 ${suggestionsResult.suggestions.length} 个优化建议`);
                setSuggestions(suggestionsResult.suggestions);
                setStep('confirm');
            } else if (suggestionsResult.error) {
                addLog(`❌ 分析失败: ${suggestionsResult.error}`);
                setError(suggestionsResult.error);
                setStep('complete');
            } else {
                addLog('ℹ️ 未找到优化建议，数据库可能已经优化');
                setError('未找到优化建议，数据库可能已经优化或不需要优化');
                setStep('complete');
            }
        } catch (err) {
            const errorMsg = err instanceof Error ? err.message : String(err);
            addLog(`❌ 分析出错: ${errorMsg}`);
            setError(errorMsg);
            setStep('complete');
        }
    };

    const applyOptimizations = async () => {
        setStep('applying');
        setLogs([]);

        try {
            addLog(`⚡ 开始执行优化，共 ${suggestions.length} 个索引...`);

            const optimizeResult = await ApplyOptimizeSuggestions(dataSourceId, suggestions);

            // Log each result
            let successCount = 0;
            optimizeResult.suggestions.forEach((sug, idx) => {
                if (sug.applied) {
                    addLog(`✅ [${idx + 1}/${suggestions.length}] 成功创建索引: ${sug.index_name}`);
                    successCount++;
                } else {
                    addLog(`❌ [${idx + 1}/${suggestions.length}] 创建失败: ${sug.index_name} - ${sug.error || '未知错误'}`);
                }
            });

            addLog(`\n📊 优化完成: 成功 ${successCount}/${suggestions.length} 个索引`);

            setResult(optimizeResult);
            setStep('complete');
        } catch (err) {
            const errorMsg = err instanceof Error ? err.message : String(err);
            addLog(`❌ 优化执行出错: ${errorMsg}`);
            setError(errorMsg);
            setStep('complete');
        }
    };

    const handleClose = () => {
        onClose();
        // Emit event to refresh data sources list
        if (step === 'complete' && result?.success) {
            import('../../wailsjs/runtime/runtime').then(({ EventsEmit }) => {
                EventsEmit('data-source-optimized', dataSourceId);
            });
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[10000]" onClick={handleClose}>
            <div
                className="bg-white rounded-lg shadow-2xl w-full max-w-2xl max-h-[80vh] overflow-hidden flex flex-col"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b border-slate-200 bg-gradient-to-r from-amber-50 to-orange-50">
                    <div className="flex items-center gap-2">
                        <div className="p-1.5 bg-amber-100 rounded-lg">
                            <Zap className="w-5 h-5 text-amber-600" />
                        </div>
                        <div>
                            <h2 className="text-lg font-bold text-slate-800">数据源优化</h2>
                            <p className="text-xs text-slate-500">{dataSourceName}</p>
                        </div>
                    </div>
                    <button
                        onClick={handleClose}
                        className="p-1.5 hover:bg-white/50 rounded-lg transition-colors"
                    >
                        <X className="w-4 h-4 text-slate-500" />
                    </button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-4">
                    {/* Execution Logs */}
                    {(step === 'loading' || step === 'applying') && (
                        <div className="space-y-3">
                            <div className="flex items-center gap-2 mb-2">
                                <Loader className="w-4 h-4 text-amber-500 animate-spin" />
                                <span className="text-sm font-medium text-slate-700">
                                    {step === 'loading' ? '正在分析...' : '正在执行优化...'}
                                </span>
                            </div>

                            <div className="bg-slate-900 rounded-lg p-3 font-mono text-xs text-green-400 h-64 overflow-y-auto">
                                {logs.map((log, idx) => (
                                    <div key={idx} className="mb-1">{log}</div>
                                ))}
                                <div ref={logsEndRef} />
                            </div>
                        </div>
                    )}

                    {/* Confirm Step */}
                    {step === 'confirm' && suggestions.length > 0 && (
                        <div className="space-y-3">
                            <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
                                <div className="flex items-start gap-2">
                                    <AlertTriangle className="w-4 h-4 text-blue-500 flex-shrink-0 mt-0.5" />
                                    <div>
                                        <h3 className="font-semibold text-sm text-blue-800 mb-0.5">确认优化操作</h3>
                                        <p className="text-xs text-blue-600">
                                            系统将执行以下 {suggestions.length} 个索引创建操作。
                                        </p>
                                    </div>
                                </div>
                            </div>

                            <div className="space-y-2 max-h-80 overflow-y-auto">
                                {suggestions.map((suggestion, index) => (
                                    <div
                                        key={index}
                                        className="border border-slate-200 rounded-lg p-3 bg-slate-50 text-sm"
                                    >
                                        <div className="flex items-center gap-2 mb-1">
                                            <span className="flex-shrink-0 w-5 h-5 bg-amber-100 rounded-full flex items-center justify-center text-xs font-bold text-amber-600">
                                                {index + 1}
                                            </span>
                                            <span className="font-semibold text-slate-800">{suggestion.index_name}</span>
                                            <span className="text-xs px-1.5 py-0.5 bg-slate-200 text-slate-600 rounded">
                                                {suggestion.table_name}
                                            </span>
                                        </div>

                                        <p className="text-xs text-slate-600 mb-1 ml-7">
                                            <span className="font-medium">列：</span>
                                            {suggestion.columns.join(', ')}
                                        </p>

                                        <p className="text-xs text-slate-600 ml-7">
                                            {suggestion.reason}
                                        </p>
                                    </div>
                                ))}
                            </div>

                            {/* Execution Log Preview */}
                            {logs.length > 0 && (
                                <div className="bg-slate-900 rounded-lg p-3 font-mono text-xs text-green-400 max-h-32 overflow-y-auto">
                                    {logs.map((log, idx) => (
                                        <div key={idx} className="mb-1">{log}</div>
                                    ))}
                                </div>
                            )}
                        </div>
                    )}

                    {/* Complete Step */}
                    {step === 'complete' && (
                        <div className="space-y-3">
                            {error && (
                                <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
                                    <div className="flex items-start gap-2">
                                        <XCircle className="w-4 h-4 text-blue-500 flex-shrink-0 mt-0.5" />
                                        <div>
                                            <h3 className="font-semibold text-sm text-blue-800 mb-0.5">优化失败</h3>
                                            <p className="text-xs text-blue-600">{error}</p>
                                        </div>
                                    </div>
                                </div>
                            )}

                            {result && (
                                <div className={`rounded-lg p-3 text-sm ${result.success ? 'bg-green-50 border border-green-200' : 'bg-amber-50 border border-amber-200'}`}>
                                    <div className="flex items-start gap-2">
                                        {result.success ? (
                                            <CheckCircle className="w-4 h-4 text-green-500 flex-shrink-0 mt-0.5" />
                                        ) : (
                                            <Database className="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" />
                                        )}
                                        <div>
                                            <h3 className={`font-semibold mb-0.5 ${result.success ? 'text-green-800' : 'text-amber-800'}`}>
                                                {result.summary}
                                            </h3>
                                            <p className={`text-xs ${result.success ? 'text-green-600' : 'text-amber-600'}`}>
                                                共分析 {result.suggestions.length} 个索引建议
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            )}

                            {/* Execution Logs */}
                            {logs.length > 0 && (
                                <div>
                                    <h3 className="text-sm font-semibold text-slate-800 mb-2">执行日志</h3>
                                    <div className="bg-slate-900 rounded-lg p-3 font-mono text-xs text-green-400 max-h-64 overflow-y-auto">
                                        {logs.map((log, idx) => (
                                            <div key={idx} className="mb-1 whitespace-pre-line">{log}</div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="p-3 border-t border-slate-200 bg-slate-50 flex justify-end gap-2">
                    {step === 'confirm' && (
                        <>
                            <button
                                onClick={handleClose}
                                className="px-3 py-1.5 text-sm bg-slate-200 text-slate-700 rounded-lg hover:bg-slate-300 transition-colors font-medium"
                            >
                                取消
                            </button>
                            <button
                                onClick={applyOptimizations}
                                className="px-3 py-1.5 text-sm bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition-colors font-medium flex items-center gap-1.5"
                            >
                                <Zap className="w-3.5 h-3.5" />
                                确认优化
                            </button>
                        </>
                    )}
                    {(step === 'complete' || step === 'loading') && (
                        <button
                            onClick={handleClose}
                            className="px-3 py-1.5 text-sm bg-slate-600 text-white rounded-lg hover:bg-slate-700 transition-colors font-medium"
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
