import React, { useState, useEffect, useCallback, useMemo } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2, CheckSquare, Square, AlertTriangle } from 'lucide-react';
import { ExportQuickAnalysisPackSelected, GetConfig, GetThreadExportableRequests } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

interface ExportPackDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (author: string) => void;
    threadId: string;
}

type DialogStep = 'loading' | 'select' | 'form';

const ExportPackDialog: React.FC<ExportPackDialogProps> = ({
    isOpen,
    onClose,
    onConfirm,
    threadId,
}) => {
    const { t } = useLanguage();
    const [step, setStep] = useState<DialogStep>('loading');
    const [requests, setRequests] = useState<main.ExportableRequest[]>([]);
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
    const [packName, setPackName] = useState('');
    const [author, setAuthor] = useState('');
    const [isExporting, setIsExporting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [successPath, setSuccessPath] = useState<string | null>(null);
    const [loadError, setLoadError] = useState<string | null>(null);
    const backdropMouseDown = React.useRef(false);

    // Load exportable requests when dialog opens
    useEffect(() => {
        if (!isOpen) return;

        setStep('loading');
        setPackName('');
        setAuthor('');
        setIsExporting(false);
        setError(null);
        setSuccessPath(null);
        setLoadError(null);
        setRequests([]);
        setSelectedIds(new Set());

        GetConfig().then(cfg => {
            if (cfg.authorSignature) {
                setAuthor(cfg.authorSignature);
            }
        }).catch(console.error);

        GetThreadExportableRequests(threadId)
            .then((result) => {
                const reqs = result || [];
                setRequests(reqs);
                const selected = new Set<string>();
                for (const req of reqs) {
                    if (!req.is_auto_suggestion) {
                        selected.add(req.request_id);
                    }
                }
                setSelectedIds(selected);
                setStep('select');
            })
            .catch((err: any) => {
                setLoadError(err?.message || err?.toString() || t('operation_failed'));
                setStep('select');
            });
    }, [isOpen, threadId]);

    const toggleRequest = useCallback((requestId: string) => {
        setSelectedIds(prev => {
            const next = new Set(prev);
            if (next.has(requestId)) {
                next.delete(requestId);
            } else {
                next.add(requestId);
            }
            return next;
        });
    }, []);

    const toggleAll = useCallback(() => {
        setSelectedIds(prev => {
            if (prev.size === requests.length) {
                return new Set();
            }
            return new Set(requests.map(r => r.request_id));
        });
    }, [requests]);

    const handleNext = useCallback(() => {
        if (selectedIds.size === 0) return;
        setStep('form');
    }, [selectedIds.size]);

    const handleBack = useCallback(() => {
        setStep('select');
        setError(null);
        setSuccessPath(null);
    }, []);

    const packNameTrimmed = packName.trim();
    const authorTrimmed = author.trim();
    const canSubmit = step === 'form' && packNameTrimmed !== '' && authorTrimmed !== '' && !isExporting;

    const handleConfirm = useCallback(async () => {
        if (!canSubmit) return;

        setIsExporting(true);
        setError(null);
        setSuccessPath(null);
        try {
            const savedPath = await ExportQuickAnalysisPackSelected(
                threadId, packNameTrimmed, authorTrimmed, '', Array.from(selectedIds)
            );
            setSuccessPath(savedPath);
            onConfirm(authorTrimmed);
        } catch (err: any) {
            setError(err?.message || err?.toString() || t('export_failed_title'));
        } finally {
            setIsExporting(false);
        }
    }, [canSubmit, threadId, packNameTrimmed, authorTrimmed, selectedIds, onConfirm]);

    const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'Escape' && !isExporting) {
            onClose();
        } else if (e.key === 'Enter') {
            if (step === 'select' && selectedIds.size > 0) {
                handleNext();
            } else if (canSubmit) {
                handleConfirm();
            }
        }
    }, [isExporting, onClose, step, selectedIds.size, handleNext, canSubmit, handleConfirm]);

    const isBusy = step === 'loading' || isExporting;
    const allSelected = useMemo(() => selectedIds.size === requests.length && requests.length > 0, [selectedIds.size, requests.length]);

    if (!isOpen) return null;

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onMouseDown={(e) => {
                if (e.target === e.currentTarget) backdropMouseDown.current = true;
            }}
            onMouseUp={(e) => {
                if (e.target === e.currentTarget && backdropMouseDown.current && !isBusy) {
                    onClose();
                }
                backdropMouseDown.current = false;
            }}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[520px] max-h-[80vh] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6 flex flex-col"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
            >
                <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-4">
                    {t('export_pack_title')}
                </h3>

                {/* Step 1: Loading */}
                {step === 'loading' && (
                    <div className="flex items-center justify-center py-8">
                        <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
                        <span className="ml-2 text-sm text-slate-500 dark:text-[#8e8e8e]">
                            {t('export_pack_loading_requests')}
                        </span>
                    </div>
                )}

                {/* Step 2: Select requests */}
                {step === 'select' && (
                    <>
                        {loadError ? (
                            <div className="flex items-center gap-2 py-4 text-sm text-red-500">
                                <AlertTriangle className="w-4 h-4 flex-shrink-0" />
                                <span>{loadError}</span>
                            </div>
                        ) : requests.length === 0 ? (
                            <div className="py-4 text-sm text-slate-500 dark:text-[#8e8e8e]">
                                {t('export_pack_no_requests')}
                            </div>
                        ) : (
                            <>
                                <p className="text-sm text-slate-500 dark:text-[#8e8e8e] mb-3">
                                    {t('export_pack_select_hint')}
                                </p>

                                {/* Select all toggle */}
                                <div
                                    className="flex items-center gap-2 mb-2 pb-2 border-b border-slate-200 dark:border-[#3e3e42] cursor-pointer select-none hover:bg-slate-50 dark:hover:bg-[#2d2d30] rounded px-2 py-1 -mx-2"
                                    onClick={toggleAll}
                                    role="checkbox"
                                    aria-checked={allSelected}
                                    tabIndex={0}
                                    onKeyDown={(e) => { if (e.key === ' ' || e.key === 'Enter') { e.preventDefault(); toggleAll(); } }}
                                >
                                    {allSelected ? (
                                        <CheckSquare className="w-4 h-4 text-blue-500 flex-shrink-0" />
                                    ) : (
                                        <Square className="w-4 h-4 text-slate-400 dark:text-[#6e6e6e] flex-shrink-0" />
                                    )}
                                    <span className="text-sm font-medium text-slate-600 dark:text-[#b0b0b0]">
                                        {t('export_pack_select_all')} ({requests.length})
                                    </span>
                                </div>

                                {/* Request list */}
                                <div className="overflow-y-auto max-h-[40vh] space-y-1">
                                    {requests.map((req) => {
                                        const checked = selectedIds.has(req.request_id);
                                        return (
                                            <div
                                                key={req.request_id}
                                                className="flex items-start gap-2 cursor-pointer select-none hover:bg-slate-50 dark:hover:bg-[#2d2d30] rounded px-2 py-2 -mx-2"
                                                onClick={() => toggleRequest(req.request_id)}
                                                role="checkbox"
                                                aria-checked={checked}
                                                tabIndex={0}
                                                onKeyDown={(e) => { if (e.key === ' ' || e.key === 'Enter') { e.preventDefault(); toggleRequest(req.request_id); } }}
                                            >
                                                {checked ? (
                                                    <CheckSquare className="w-4 h-4 text-blue-500 flex-shrink-0 mt-0.5" />
                                                ) : (
                                                    <Square className="w-4 h-4 text-slate-400 dark:text-[#6e6e6e] flex-shrink-0 mt-0.5" />
                                                )}
                                                <div className="flex-1 min-w-0">
                                                    <p className="text-sm text-slate-800 dark:text-[#d4d4d4] break-words leading-snug">
                                                        {req.user_request}
                                                    </p>
                                                    <div className="flex items-center gap-2 mt-1">
                                                        <span className="text-xs text-slate-400 dark:text-[#6e6e6e]">
                                                            {req.step_count} {t('export_pack_steps')}
                                                        </span>
                                                        {req.is_auto_suggestion && (
                                                            <span className="text-xs px-1.5 py-0.5 rounded bg-amber-50 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400">
                                                                {t('export_pack_auto_suggestion')}
                                                            </span>
                                                        )}
                                                    </div>
                                                </div>
                                            </div>
                                        );
                                    })}
                                </div>
                            </>
                        )}

                        {/* Buttons for select step */}
                        <div className="flex justify-end gap-3 mt-4 pt-4 border-t border-slate-200 dark:border-[#3e3e42]">
                            <button
                                onClick={onClose}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors"
                            >
                                {t('cancel')}
                            </button>
                            <button
                                onClick={handleNext}
                                disabled={selectedIds.size === 0 || !!loadError}
                                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                {t('export_pack_next')}
                            </button>
                        </div>
                    </>
                )}

                {/* Step 3: Form (pack name + author) */}
                {step === 'form' && (
                    <>
                        <p className="text-sm text-slate-500 dark:text-[#8e8e8e] mb-3">
                            {t('export_pack_selected_count').replace('{0}', String(selectedIds.size)).replace('{1}', String(requests.length))}
                        </p>

                        {/* Pack name input (required) */}
                        <div className="mb-4">
                            <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                                {t('export_pack_name')} <span className="text-red-500">*</span>
                            </label>
                            <input
                                type="text"
                                value={packName}
                                onChange={e => setPackName(e.target.value)}
                                placeholder={t('export_pack_name_placeholder')}
                                disabled={isExporting}
                                autoFocus
                                className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-[#3e3e42] rounded-lg bg-white dark:bg-[#1e1e1e] text-slate-900 dark:text-[#d4d4d4] placeholder-slate-400 dark:placeholder-[#6e6e6e] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                            />
                        </div>

                        {/* Author input (required) */}
                        <div className="mb-4">
                            <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                                {t('export_pack_author')} <span className="text-red-500">*</span>
                            </label>
                            <input
                                type="text"
                                value={author}
                                onChange={e => setAuthor(e.target.value)}
                                placeholder={t('export_pack_author_placeholder')}
                                disabled={isExporting}
                                className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-[#3e3e42] rounded-lg bg-white dark:bg-[#1e1e1e] text-slate-900 dark:text-[#d4d4d4] placeholder-slate-400 dark:placeholder-[#6e6e6e] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                            />
                        </div>

                        {/* Success message */}
                        {successPath && (
                            <p className="mb-4 text-sm text-green-600 dark:text-green-400 break-all">
                                {t('export_pack_success')}{successPath}
                            </p>
                        )}

                        {/* Error message */}
                        {error && (
                            <p className="mb-4 text-sm text-red-500">{error}</p>
                        )}

                        {/* Buttons for form step */}
                        <div className="flex justify-between mt-4 pt-4 border-t border-slate-200 dark:border-[#3e3e42]">
                            <button
                                onClick={handleBack}
                                disabled={isExporting}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                {t('export_pack_back')}
                            </button>
                            <div className="flex gap-3">
                                <button
                                    onClick={onClose}
                                    disabled={isExporting}
                                    className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    {t('cancel')}
                                </button>
                                <button
                                    onClick={handleConfirm}
                                    disabled={!canSubmit}
                                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                                >
                                    {isExporting && <Loader2 className="w-4 h-4 animate-spin" />}
                                    {isExporting ? t('export_pack_exporting') : t('export')}
                                </button>
                            </div>
                        </div>
                    </>
                )}
            </div>
        </div>,
        document.body
    );
};

export default ExportPackDialog;
