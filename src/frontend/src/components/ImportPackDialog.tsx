import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2, AlertTriangle, XCircle, CheckCircle2, ChevronDown, ChevronRight, FileCode2, Database, Zap, ShoppingBag, Lock, FolderOpen, Package, ArrowLeft } from 'lucide-react';
import {
    LoadQuickAnalysisPack,
    LoadQuickAnalysisPackWithPassword,
    ExecuteQuickAnalysisPack,
    ListLocalQuickAnalysisPacks,
    LoadQuickAnalysisPackByPath,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

interface ImportPackDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: () => void;
    dataSourceId: string;
}

type DialogState = 'pack-list' | 'loading' | 'password' | 'preview' | 'executing';

export function getPackOrigin(fileName: string): 'local' | 'marketplace' {
    return fileName.startsWith('marketplace_') ? 'marketplace' : 'local';
}

export function canCloseDialog(state: string): boolean {
    return state !== 'loading' && state !== 'executing';
}

export function canImport(
    loadResult: { has_python_steps?: boolean; python_configured?: boolean; validation?: { missing_tables?: string[] } } | null,
    state: string
): boolean {
    if (state !== 'preview') return false;
    const hasMissingTables = loadResult?.validation?.missing_tables && loadResult.validation.missing_tables.length > 0;
    if (hasMissingTables) return false;
    if (loadResult?.has_python_steps && !loadResult?.python_configured) return false;
    return true;
}

const ImportPackDialog: React.FC<ImportPackDialogProps> = ({
    isOpen,
    onClose,
    onConfirm,
    dataSourceId,
}) => {
    const { t } = useLanguage();
    const [state, setState] = useState<DialogState>('pack-list');
    const [packs, setPacks] = useState<main.LocalPackInfo[]>([]);
    const [packsLoading, setPacksLoading] = useState(false);
    const [packsError, setPacksError] = useState<string | null>(null);
    const [password, setPassword] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [loadResult, setLoadResult] = useState<main.PackLoadResult | null>(null);
    const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set());

    // Load pack list when dialog opens
    useEffect(() => {
        if (!isOpen) return;
        setState('pack-list');
        setPacks([]);
        setPacksLoading(true);
        setPacksError(null);
        setPassword('');
        setError(null);
        setLoadResult(null);
        setExpandedSteps(new Set());

        ListLocalQuickAnalysisPacks()
            .then((result) => {
                setPacks(result || []);
                setPacksLoading(false);
            })
            .catch((err: any) => {
                setPacksError(err?.message || err?.toString() || 'Failed to load packs');
                setPacksLoading(false);
            });
    }, [isOpen, dataSourceId]);

    if (!isOpen) return null;

    const handlePackClick = async (pack: main.LocalPackInfo) => {
        setState('loading');
        setError(null);
        try {
            const result = await LoadQuickAnalysisPackByPath(pack.file_path, dataSourceId);
            if (result.needs_password) {
                setLoadResult(result);
                setState('password');
            } else {
                setLoadResult(result);
                setState('preview');
            }
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Failed to load pack');
            setState('pack-list');
        }
    };

    const handleBrowseFile = async () => {
        setState('loading');
        setError(null);
        try {
            const result = await LoadQuickAnalysisPack(dataSourceId);
            if (!result) {
                // User cancelled file selection
                setState('pack-list');
                return;
            }
            if (result.needs_password) {
                setLoadResult(result);
                setState('password');
            } else {
                setLoadResult(result);
                setState('preview');
            }
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Failed to load pack');
            setState('pack-list');
        }
    };

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
        setState('executing');
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
        if (e.key === 'Escape' && canCloseDialog(state)) {
            onClose();
        }
        if (e.key === 'Enter') {
            if (state === 'password') handlePasswordSubmit();
        }
    };

    const handleBack = () => {
        setState('pack-list');
        setLoadResult(null);
        setPassword('');
        setError(null);
        setExpandedSteps(new Set());
    };

    const validation = loadResult?.validation;
    const hasMissingTables = validation?.missing_tables && validation.missing_tables.length > 0;
    const hasMissingColumns = validation?.missing_columns && validation.missing_columns.length > 0;
    const canImportResult = canImport(loadResult, state);

    const toggleStep = (stepId: number) => {
        setExpandedSteps(prev => {
            const next = new Set(prev);
            if (next.has(stepId)) next.delete(stepId);
            else next.add(stepId);
            return next;
        });
    };

    const isLoading = !canCloseDialog(state);

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={isLoading ? undefined : onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[600px] max-h-[80vh] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6 flex flex-col"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
            >
                <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-4">
                    {t('import_pack_title')}
                </h3>

                {/* Pack list state */}
                {state === 'pack-list' && (
                    <div className="flex flex-col flex-1 min-h-0">
                        {packsLoading && (
                            <div className="flex items-center justify-center py-8 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                                <Loader2 className="w-5 h-5 animate-spin" />
                                {t('import_pack_loading')}
                            </div>
                        )}

                        {packsError && !packsLoading && (
                            <div className="flex flex-col items-center justify-center py-8 gap-3">
                                <p className="text-sm text-red-500">{packsError}</p>
                                <button
                                    onClick={() => {
                                        setPacksError(null);
                                        setPacksLoading(true);
                                        ListLocalQuickAnalysisPacks()
                                            .then((result) => {
                                                setPacks(result || []);
                                                setPacksLoading(false);
                                            })
                                            .catch((err: any) => {
                                                setPacksError(err?.message || err?.toString() || 'Failed to load packs');
                                                setPacksLoading(false);
                                            });
                                    }}
                                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors"
                                >
                                    {t('retry') || 'Retry'}
                                </button>
                            </div>
                        )}

                        {!packsLoading && !packsError && packs.length === 0 && (
                            <div className="flex flex-col items-center justify-center py-8 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                                <Package className="w-10 h-10 opacity-40" />
                                <p>{t('import_pack_empty_hint')}</p>
                            </div>
                        )}

                        {!packsLoading && !packsError && packs.length > 0 && (
                            <div className="overflow-y-auto max-h-[calc(80vh-200px)] space-y-1">
                                {packs.map((pack, idx) => {
                                    const origin = getPackOrigin(pack.file_name);
                                    return (
                                        <div
                                            key={pack.file_path || idx}
                                            onClick={() => handlePackClick(pack)}
                                            className="flex items-start gap-3 p-3 rounded-lg border border-slate-200 dark:border-[#3e3e42] hover:bg-slate-50 dark:hover:bg-[#2d2d30] cursor-pointer transition-colors"
                                        >
                                            <div className="flex items-center gap-1 mt-0.5 flex-shrink-0">
                                                {pack.is_encrypted && <Lock className="w-3.5 h-3.5 text-amber-500" />}
                                                {origin === 'local'
                                                    ? <Zap className="w-4 h-4 text-yellow-500" />
                                                    : <ShoppingBag className="w-4 h-4 text-purple-500" />
                                                }
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <p className="text-sm font-medium truncate">{pack.pack_name}</p>
                                                {pack.description && (
                                                    <p className="text-xs text-slate-500 dark:text-[#8e8e8e] mt-0.5 truncate">{pack.description}</p>
                                                )}
                                                <p className="text-xs text-slate-400 dark:text-[#6e6e6e] mt-1">
                                                    {pack.source_name && <span>{pack.source_name}</span>}
                                                    {pack.author && <span>{pack.source_name ? ' · ' : ''}{pack.author}</span>}
                                                    {pack.created_at && <span>{(pack.source_name || pack.author) ? ' · ' : ''}{formatDate(pack.created_at)}</span>}
                                                </p>
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        )}

                        {/* Bottom bar */}
                        <div className="flex justify-end gap-3 mt-4 pt-4 border-t border-slate-200 dark:border-[#3e3e42]">
                            <button
                                onClick={handleBrowseFile}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors flex items-center gap-2"
                            >
                                <FolderOpen className="w-4 h-4" />
                                {t('import_pack_browse_file')}
                            </button>
                            <button
                                onClick={onClose}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors"
                            >
                                {t('cancel')}
                            </button>
                        </div>
                    </div>
                )}

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
                                onClick={handleBack}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors flex items-center gap-1 mr-auto"
                            >
                                <ArrowLeft className="w-4 h-4" />
                                {t('import_pack_back')}
                            </button>
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
                {(state === 'preview' || state === 'executing') && (
                    <div className="overflow-y-auto max-h-[calc(80vh-100px)]">
                        {/* Error without pack data (file load failure) */}
                        {error && !loadResult?.pack && (
                            <div className="mb-4">
                                <p className="text-sm text-red-500">{error}</p>
                                <div className="flex justify-end mt-4">
                                    <button
                                        onClick={handleBack}
                                        disabled={state === 'executing'}
                                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
                                    >
                                        <ArrowLeft className="w-4 h-4" />
                                        {t('import_pack_back')}
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

                                {/* Python environment warning */}
                                {loadResult.has_python_steps && !loadResult.python_configured && (
                                    <div className="flex items-start gap-2 p-2.5 mb-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800/40">
                                        <XCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                        <p className="text-sm text-red-700 dark:text-red-400">
                                            {t('import_pack_python_not_configured')}
                                        </p>
                                    </div>
                                )}

                                {/* Pack contents directory */}
                                {loadResult.pack.executable_steps && loadResult.pack.executable_steps.length > 0 && (
                                    <div className="mb-4">
                                        <p className="text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-2">
                                            {t('import_pack_contents')}
                                        </p>
                                        <div className="border border-slate-200 dark:border-[#3e3e42] rounded-lg overflow-hidden">
                                            {loadResult.pack.executable_steps.map((step) => {
                                                const isSql = step.step_type === 'sql_query';
                                                const isExpanded = expandedSteps.has(step.step_id);
                                                return (
                                                    <div key={step.step_id} className="border-b border-slate-200 dark:border-[#3e3e42] last:border-b-0">
                                                        <button
                                                            type="button"
                                                            onClick={() => toggleStep(step.step_id)}
                                                            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                                        >
                                                            {isExpanded
                                                                ? <ChevronDown className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />
                                                                : <ChevronRight className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />
                                                            }
                                                            {isSql
                                                                ? <Database className="w-3.5 h-3.5 text-blue-500 flex-shrink-0" />
                                                                : <FileCode2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                                                            }
                                                            <span className="text-slate-400 dark:text-[#6e6e6e] flex-shrink-0">
                                                                {t('import_pack_step_label')} {step.step_id}
                                                            </span>
                                                            <span className="truncate">{step.description}</span>
                                                            <span className="ml-auto text-xs text-slate-400 dark:text-[#6e6e6e] flex-shrink-0">
                                                                {isSql ? t('import_pack_sql_scripts') : t('import_pack_python_scripts')}
                                                            </span>
                                                        </button>
                                                        {isExpanded && (
                                                            <div className="px-3 pb-2">
                                                                <pre className="text-xs bg-slate-50 dark:bg-[#1e1e1e] border border-slate-200 dark:border-[#3e3e42] rounded p-2 overflow-x-auto max-h-[200px] overflow-y-auto whitespace-pre-wrap break-all">
                                                                    <code>{step.code}</code>
                                                                </pre>
                                                            </div>
                                                        )}
                                                    </div>
                                                );
                                            })}
                                        </div>
                                    </div>
                                )}

                                {/* Error during import execution */}
                                {error && <p className="mb-4 text-sm text-red-500">{error}</p>}

                                {/* Buttons */}
                                <div className="flex justify-end gap-3">
                                    <button
                                        onClick={handleBack}
                                        disabled={state === 'executing'}
                                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1 mr-auto"
                                    >
                                        <ArrowLeft className="w-4 h-4" />
                                        {t('import_pack_back')}
                                    </button>
                                    <button
                                        onClick={onClose}
                                        disabled={state === 'executing'}
                                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                    >
                                        {t('cancel')}
                                    </button>
                                    <button
                                        onClick={handleConfirmImport}
                                        disabled={!canImportResult}
                                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                                    >
                                        {state === 'executing' && <Loader2 className="w-4 h-4 animate-spin" />}
                                        {state === 'executing' ? t('import_pack_importing') : t('import_pack_confirm')}
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
