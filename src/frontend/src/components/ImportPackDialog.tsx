import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2, AlertTriangle, XCircle, CheckCircle2 } from 'lucide-react';
import {
    LoadQuickAnalysisPack,
    LoadQuickAnalysisPackWithPassword,
    ExecuteQuickAnalysisPack,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

interface ImportPackDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: () => void;
    dataSourceId: string;
}

type DialogState = 'loading' | 'password' | 'preview' | 'importing';

const ImportPackDialog: React.FC<ImportPackDialogProps> = ({
    isOpen,
    onClose,
    onConfirm,
    dataSourceId,
}) => {
    const { t } = useLanguage();
    const [state, setState] = useState<DialogState>('loading');
    const [password, setPassword] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [loadResult, setLoadResult] = useState<main.PackLoadResult | null>(null);

    // Load pack when dialog opens
    useEffect(() => {
        if (!isOpen) return;
        setState('loading');
        setPassword('');
        setError(null);
        setLoadResult(null);

        LoadQuickAnalysisPack(dataSourceId)
            .then((result) => {
                if (result.needs_password) {
                    setState('password');
                    setLoadResult(result);
                } else {
                    setLoadResult(result);
                    setState('preview');
                }
            })
            .catch((err: any) => {
                setError(err?.message || err?.toString() || 'Failed to load pack');
                setState('preview');
            });
    }, [isOpen, dataSourceId]);

    if (!isOpen) return null;

    const handlePasswordSubmit = async () => {
        if (!password.trim() || !loadResult?.file_path) return;
        setError(null);
        setState('loading');
        try {
            const result = await LoadQuickAnalysisPackWithPassword(
                loadResult.file_path,
                dataSourceId,
                password
            );
            setLoadResult(result);
            if (result.needs_password) {
                setError(t('import_pack_wrong_password'));
                setState('password');
            } else {
                setState('preview');
            }
        } catch (err: any) {
            setError(err?.message || err?.toString() || t('import_pack_wrong_password'));
            setState('password');
        }
    };

    const handleConfirmImport = async () => {
        if (!loadResult?.file_path) return;
        setState('importing');
        setError(null);
        try {
            await ExecuteQuickAnalysisPack(loadResult.file_path, dataSourceId, password);
            onConfirm();
            onClose();
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Import failed');
            setState('preview');
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape' && state !== 'loading' && state !== 'importing') {
            onClose();
        }
        if (e.key === 'Enter') {
            if (state === 'password') handlePasswordSubmit();
        }
    };

    const validation = loadResult?.validation;
    const hasMissingTables = validation?.missing_tables && validation.missing_tables.length > 0;
    const hasMissingColumns = validation?.missing_columns && validation.missing_columns.length > 0;
    const canImport = !hasMissingTables && state === 'preview';

    const isLoading = state === 'loading' || state === 'importing';

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={isLoading ? undefined : onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[480px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
            >
                <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-4">
                    {t('import_pack_title')}
                </h3>

                {/* Loading state */}
                {state === 'loading' && (
                    <div className="flex items-center justify-center py-8 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                        <Loader2 className="w-5 h-5 animate-spin" />
                        {t('import_pack_loading')}
                    </div>
                )}

                {/* Password entry state */}
                {state === 'password' && (
                    <div>
                        <p className="text-sm text-slate-600 dark:text-[#b0b0b0] mb-4">
                            {t('import_pack_password_required')}
                        </p>
                        <div className="mb-4">
                            <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                                {t('import_pack_password')} <span className="text-red-500">*</span>
                            </label>
                            <input
                                type="password"
                                value={password}
                                onChange={e => { setPassword(e.target.value); setError(null); }}
                                placeholder={t('import_pack_password_placeholder')}
                                autoFocus
                                className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-[#3e3e42] rounded-lg bg-white dark:bg-[#1e1e1e] text-slate-900 dark:text-[#d4d4d4] placeholder-slate-400 dark:placeholder-[#6e6e6e] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            />
                        </div>
                        {error && <p className="mb-4 text-sm text-red-500">{error}</p>}
                        <div className="flex justify-end gap-3">
                            <button
                                onClick={onClose}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors"
                            >
                                {t('cancel')}
                            </button>
                            <button
                                onClick={handlePasswordSubmit}
                                disabled={!password.trim()}
                                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                {t('confirm')}
                            </button>
                        </div>
                    </div>
                )}

                {/* Preview state */}
                {(state === 'preview' || state === 'importing') && (
                    <div>
                        {/* Error without pack data (file load failure) */}
                        {error && !loadResult?.pack && (
                            <div className="mb-4">
                                <p className="text-sm text-red-500">{error}</p>
                                <div className="flex justify-end mt-4">
                                    <button
                                        onClick={onClose}
                                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors"
                                    >
                                        {t('close')}
                                    </button>
                                </div>
                            </div>
                        )}

                        {/* Pack metadata and validation */}
                        {loadResult?.pack && (
                            <>
                                {/* Metadata section */}
                                <div className="mb-4 p-3 bg-slate-50 dark:bg-[#1e1e1e] rounded-lg border border-slate-200 dark:border-[#3e3e42]">
                                    <div className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-sm">
                                        <span className="text-slate-500 dark:text-[#8e8e8e]">{t('import_pack_author')}:</span>
                                        <span className="font-medium">{loadResult.pack.metadata.author}</span>
                                        <span className="text-slate-500 dark:text-[#8e8e8e]">{t('import_pack_created_at')}:</span>
                                        <span>{formatDate(loadResult.pack.metadata.created_at)}</span>
                                        <span className="text-slate-500 dark:text-[#8e8e8e]">{t('import_pack_source_name')}:</span>
                                        <span>{loadResult.pack.metadata.source_name}</span>
                                        <span className="text-slate-500 dark:text-[#8e8e8e]">{t('import_pack_steps_count')}:</span>
                                        <span>{loadResult.pack.executable_steps?.length ?? 0}</span>
                                    </div>
                                </div>

                                {/* Schema validation summary */}
                                {validation && (
                                    <div className="mb-4">
                                        <p className="text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-2">
                                            {t('import_pack_schema_validation')}
                                        </p>

                                        {/* Missing tables - error */}
                                        {hasMissingTables && (
                                            <div className="flex items-start gap-2 p-2.5 mb-2 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800/40">
                                                <XCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                                <div className="text-sm">
                                                    <p className="text-red-700 dark:text-red-400 font-medium">
                                                        {t('import_pack_missing_tables')}
                                                    </p>
                                                    <p className="text-red-600 dark:text-red-300 mt-1">
                                                        {validation.missing_tables.join(', ')}
                                                    </p>
                                                </div>
                                            </div>
                                        )}

                                        {/* Missing columns - warning */}
                                        {hasMissingColumns && (
                                            <div className="flex items-start gap-2 p-2.5 mb-2 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800/40">
                                                <AlertTriangle className="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0" />
                                                <div className="text-sm">
                                                    <p className="text-amber-700 dark:text-amber-400 font-medium">
                                                        {t('import_pack_missing_columns')}
                                                    </p>
                                                    <p className="text-amber-600 dark:text-amber-300 mt-1">
                                                        {validation.missing_columns.map(
                                                            (c) => `${c.table_name}.${c.column_name}`
                                                        ).join(', ')}
                                                    </p>
                                                </div>
                                            </div>
                                        )}

                                        {/* All good */}
                                        {!hasMissingTables && !hasMissingColumns && (
                                            <div className="flex items-center gap-2 p-2.5 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800/40">
                                                <CheckCircle2 className="w-4 h-4 text-green-500 flex-shrink-0" />
                                                <span className="text-sm text-green-700 dark:text-green-400">
                                                    {t('import_pack_schema_compatible')}
                                                </span>
                                            </div>
                                        )}
                                    </div>
                                )}

                                {/* Error during import execution */}
                                {error && <p className="mb-4 text-sm text-red-500">{error}</p>}

                                {/* Buttons */}
                                <div className="flex justify-end gap-3">
                                    <button
                                        onClick={onClose}
                                        disabled={state === 'importing'}
                                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                    >
                                        {t('cancel')}
                                    </button>
                                    <button
                                        onClick={handleConfirmImport}
                                        disabled={!canImport}
                                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                                    >
                                        {state === 'importing' && <Loader2 className="w-4 h-4 animate-spin" />}
                                        {state === 'importing' ? t('import_pack_importing') : t('import_pack_confirm')}
                                    </button>
                                </div>
                            </>
                        )}
                    </div>
                )}
            </div>
        </div>,
        document.body
    );
};

function formatDate(rfc3339: string): string {
    try {
        const d = new Date(rfc3339);
        return d.toLocaleString();
    } catch {
        return rfc3339;
    }
}

export default ImportPackDialog;
