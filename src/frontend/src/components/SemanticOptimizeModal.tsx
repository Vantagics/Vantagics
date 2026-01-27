import React, { useState, useEffect } from 'react';
import { X, Sparkles, Loader2, CheckCircle, AlertCircle } from 'lucide-react';
import { useLanguage } from '../i18n';
import { SemanticOptimizeDataSource } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

interface SemanticOptimizeModalProps {
    isOpen: boolean;
    dataSourceId: string;
    dataSourceName: string;
    onClose: () => void;
    onSuccess: () => void;
}

const SemanticOptimizeModal: React.FC<SemanticOptimizeModalProps> = ({
    isOpen,
    dataSourceId,
    dataSourceName,
    onClose,
    onSuccess
}) => {
    const { t } = useLanguage();
    const [isOptimizing, setIsOptimizing] = useState(false);
    const [progress, setProgress] = useState('');
    const [error, setError] = useState('');
    const [completed, setCompleted] = useState(false);

    useEffect(() => {
        if (!isOpen) {
            // Reset state when modal closes
            setIsOptimizing(false);
            setProgress('');
            setError('');
            setCompleted(false);
            return;
        }

        // Listen for progress events
        const unsubscribeProgress = EventsOn('semantic-optimize-progress', (data: any) => {
            if (data && data.message) {
                setProgress(data.message);
            }
        });

        // Listen for completion events
        const unsubscribeCompleted = EventsOn('semantic-optimize-completed', (data: any) => {
            setCompleted(true);
            setIsOptimizing(false);
            setProgress(t('semantic_optimize_success') || '语义优化完成！');
            
            // Auto close after 2 seconds
            setTimeout(() => {
                onSuccess();
                onClose();
            }, 2000);
        });

        return () => {
            if (unsubscribeProgress) unsubscribeProgress();
            if (unsubscribeCompleted) unsubscribeCompleted();
        };
    }, [isOpen, onSuccess, onClose, t]);

    const handleOptimize = async () => {
        setIsOptimizing(true);
        setError('');
        setProgress(t('semantic_optimizing') || '正在开始优化...');

        try {
            await SemanticOptimizeDataSource(dataSourceId);
        } catch (err) {
            setError(String(err));
            setIsOptimizing(false);
            setProgress('');
        }
    };

    const handleClose = () => {
        if (!isOptimizing) {
            onClose();
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-[10001] flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[500px] rounded-xl shadow-2xl flex flex-col overflow-hidden">
                {/* Header */}
                <div className="p-4 border-b border-slate-200 flex justify-between items-center bg-gradient-to-r from-purple-50 to-blue-50">
                    <div className="flex items-center gap-2">
                        <Sparkles className="w-5 h-5 text-purple-600" />
                        <h2 className="text-lg font-bold text-slate-800">
                            {t('semantic_optimize') || '数据源语义优化'}
                        </h2>
                    </div>
                    {!isOptimizing && (
                        <button onClick={handleClose} className="text-slate-500 hover:text-slate-700">
                            <X className="w-5 h-5" />
                        </button>
                    )}
                </div>

                {/* Content */}
                <div className="p-6">
                    {!isOptimizing && !completed && !error && (
                        <div>
                            <p className="text-sm text-slate-700 mb-4">
                                {t('semantic_optimize_desc') || '将使用 AI 分析数据源结构和样本数据，为字段生成更有意义的名称。'}
                            </p>
                            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
                                <p className="text-sm text-blue-800">
                                    <strong>{t('data_source')}:</strong> {dataSourceName}
                                </p>
                                <p className="text-xs text-blue-600 mt-2">
                                    {t('semantic_optimize_note') || '将创建名为 "{name}_语义优化" 的新数据源，原数据源保持不变。'}
                                        .replace('{name}', dataSourceName)
                                </p>
                            </div>
                            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
                                <p className="text-xs text-yellow-800">
                                    <strong>{t('note')}:</strong> {t('semantic_optimize_warning') || '优化过程可能需要几分钟，请耐心等待。'}
                                </p>
                            </div>
                        </div>
                    )}

                    {isOptimizing && (
                        <div className="flex flex-col items-center py-8">
                            <Loader2 className="w-12 h-12 text-purple-600 animate-spin mb-4" />
                            <p className="text-sm text-slate-700 text-center">
                                {progress}
                            </p>
                            <div className="mt-4 w-full bg-slate-200 rounded-full h-2">
                                <div className="bg-gradient-to-r from-purple-500 to-blue-500 h-2 rounded-full animate-pulse" 
                                     style={{ width: '60%' }}></div>
                            </div>
                        </div>
                    )}

                    {completed && (
                        <div className="flex flex-col items-center py-8">
                            <CheckCircle className="w-12 h-12 text-green-600 mb-4" />
                            <p className="text-sm text-green-700 font-medium">
                                {progress}
                            </p>
                        </div>
                    )}

                    {error && (
                        <div className="flex flex-col items-center py-8">
                            <AlertCircle className="w-12 h-12 text-red-600 mb-4" />
                            <p className="text-sm text-red-700 font-medium mb-2">
                                {t('semantic_optimize_failed') || '语义优化失败'}
                            </p>
                            <p className="text-xs text-red-600 text-center">
                                {error}
                            </p>
                        </div>
                    )}
                </div>

                {/* Footer */}
                {!isOptimizing && !completed && (
                    <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-2">
                        <button
                            onClick={handleClose}
                            className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 hover:bg-slate-50 rounded-md"
                        >
                            {t('cancel') || '取消'}
                        </button>
                        <button
                            onClick={handleOptimize}
                            className="px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-purple-600 to-blue-600 hover:from-purple-700 hover:to-blue-700 rounded-md flex items-center gap-2"
                        >
                            <Sparkles className="w-4 h-4" />
                            {t('start_optimize') || '开始优化'}
                        </button>
                    </div>
                )}

                {error && (
                    <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end">
                        <button
                            onClick={handleClose}
                            className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 hover:bg-slate-50 rounded-md"
                        >
                            {t('close') || '关闭'}
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
};

export default SemanticOptimizeModal;
