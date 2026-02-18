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
            setProgress(t('semantic_optimize_success'));
            
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
        setProgress(t('semantic_optimizing'));

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
            <div className="bg-white dark:bg-[#252526] w-[500px] rounded-xl shadow-2xl flex flex-col overflow-hidden">
                {/* Header */}
                <div className="p-4 border-b border-slate-200 dark:border-[#3c3c3c] flex justify-between items-center bg-gradient-to-r from-[#f0f4f8] to-[#eaeff5] dark:from-[#2a1e2e] dark:to-[#1a2332]">
                    <div className="flex items-center gap-2">
                        <Sparkles className="w-5 h-5 text-[#5b7a9d] dark:text-[#c586c0]" />
                        <h2 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4]">
                            {t('semantic_optimize')}
                        </h2>
                    </div>
                    {!isOptimizing && (
                        <button onClick={handleClose} className="text-slate-500 hover:text-slate-700 dark:text-[#808080] dark:hover:text-[#d4d4d4]">
                            <X className="w-5 h-5" />
                        </button>
                    )}
                </div>

                {/* Content */}
                <div className="p-6">
                    {!isOptimizing && !completed && !error && (
                        <div>
                            <p className="text-sm text-slate-700 dark:text-[#d4d4d4] mb-4">
                                {t('semantic_optimize_desc')}
                            </p>
                            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
                                <p className="text-sm text-blue-800">
                                    <strong>{t('data_source')}:</strong> {dataSourceName}
                                </p>
                                <p className="text-xs text-blue-600 mt-2">
                                    {t('semantic_optimize_note')}
                                </p>
                            </div>
                            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
                                <p className="text-xs text-yellow-800">
                                    <strong>{t('note')}:</strong> {t('semantic_optimize_warning')}
                                </p>
                            </div>
                        </div>
                    )}

                    {isOptimizing && (
                        <div className="flex flex-col items-center py-8">
                            <Loader2 className="w-12 h-12 text-[#5b7a9d] dark:text-[#c586c0] animate-spin mb-4" />
                            <p className="text-sm text-slate-700 dark:text-[#d4d4d4] text-center">
                                {progress}
                            </p>
                            <div className="mt-4 w-full bg-slate-200 dark:bg-[#3c3c3c] rounded-full h-2">
                                <div className="bg-gradient-to-r from-[#5b7a9d] to-[#7b9bb8] h-2 rounded-full animate-pulse" 
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
                                {t('semantic_optimize_failed')}
                            </p>
                            <p className="text-xs text-red-600 text-center">
                                {error}
                            </p>
                        </div>
                    )}
                </div>

                {/* Footer */}
                {!isOptimizing && !completed && (
                    <div className="p-4 border-t border-slate-200 dark:border-[#3c3c3c] bg-slate-50 dark:bg-[#1e1e1e] flex justify-end gap-2">
                        <button
                            onClick={handleClose}
                            className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] bg-white dark:bg-[#3c3c3c] border border-slate-300 dark:border-[#4d4d4d] hover:bg-slate-50 dark:hover:bg-[#4d4d4d] rounded-md"
                        >
                            {t('cancel')}
                        </button>
                        <button
                            onClick={handleOptimize}
                            className="px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-[#5b7a9d] to-[#6b8db5] hover:from-[#456a8a] hover:to-[#5b7a9d] rounded-md flex items-center gap-2"
                        >
                            <Sparkles className="w-4 h-4" />
                            {t('start_optimize')}
                        </button>
                    </div>
                )}

                {error && (
                    <div className="p-4 border-t border-slate-200 dark:border-[#3c3c3c] bg-slate-50 dark:bg-[#1e1e1e] flex justify-end">
                        <button
                            onClick={handleClose}
                            className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] bg-white dark:bg-[#3c3c3c] border border-slate-300 dark:border-[#4d4d4d] hover:bg-slate-50 dark:hover:bg-[#4d4d4d] rounded-md"
                        >
                            {t('close')}
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
};

export default SemanticOptimizeModal;
