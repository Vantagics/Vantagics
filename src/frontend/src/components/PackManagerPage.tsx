import React, { useState, useEffect, useRef, useCallback } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2, Package, X, Share2, Edit3, Trash2, Lock, Zap, Download, AlertTriangle, XCircle, CheckCircle2, ChevronDown, ChevronRight, FileCode2, Database, Clock, RefreshCw } from 'lucide-react';
import { ListLocalQuickAnalysisPacks, DeleteLocalPack, GetDataSources, LoadQuickAnalysisPackByPath, LoadQuickAnalysisPackWithPassword, ExecuteQuickAnalysisPack, GetUsageLicenses, PurchaseAdditionalUses, RenewSubscription } from '../../wailsjs/go/main/App';
import EditPackMetadataDialog from './EditPackMetadataDialog';
import { main } from '../../wailsjs/go/models';

interface LocalPackInfo {
    file_name: string;
    file_path: string;
    pack_name: string;
    description: string;
    source_name: string;
    author: string;
    created_at: string;
    is_encrypted: boolean;
}

interface UsageLicenseInfo {
    listing_id: number;
    pack_name: string;
    pricing_model: string;
    remaining_uses: number;
    expires_at: string;
    billing_cycle: string;
}

interface DataSourceInfo {
    id: string;
    name: string;
    type: string;
}

interface PackManagerPageProps {
    isOpen: boolean;
    onClose: () => void;
    onSharePack?: (pack: LocalPackInfo) => void;
}

const PackManagerPage: React.FC<PackManagerPageProps> = ({ isOpen, onClose, onSharePack }) => {
    const { t } = useLanguage();
    const [packs, setPacks] = useState<LocalPackInfo[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; pack: LocalPackInfo } | null>(null);
    const contextMenuRef = useRef<HTMLDivElement>(null);

    // Usage license state (pack_name -> license info)
    const [licenseMap, setLicenseMap] = useState<Map<string, UsageLicenseInfo>>(new Map());
    const [licenseActionLoading, setLicenseActionLoading] = useState<string | null>(null);

    // Delete confirmation state
    const [deleteTarget, setDeleteTarget] = useState<LocalPackInfo | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);

    // Edit metadata dialog state
    const [editTarget, setEditTarget] = useState<LocalPackInfo | null>(null);

    // Install flow state
    const [installTarget, setInstallTarget] = useState<LocalPackInfo | null>(null);
    const [dataSources, setDataSources] = useState<DataSourceInfo[]>([]);
    const [loadingDataSources, setLoadingDataSources] = useState(false);
    const [selectedDataSourceId, setSelectedDataSourceId] = useState<string | null>(null);

    // Install preview state (after data source selection)
    type InstallPhase = 'select-ds' | 'loading' | 'password' | 'preview' | 'installing';
    const [installPhase, setInstallPhase] = useState<InstallPhase>('select-ds');
    const [installPassword, setInstallPassword] = useState('');
    const [installLoadResult, setInstallLoadResult] = useState<main.PackLoadResult | null>(null);
    const [installError, setInstallError] = useState<string | null>(null);
    const [installExpandedSteps, setInstallExpandedSteps] = useState<Set<number>>(new Set());

    const loadPacks = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const result = await ListLocalQuickAnalysisPacks();
            setPacks(result || []);
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Failed to load packs');
        } finally {
            setLoading(false);
        }
    }, []);

    const loadLicenses = useCallback(async () => {
        try {
            const licenses = await GetUsageLicenses();
            const map = new Map<string, UsageLicenseInfo>();
            if (licenses) {
                for (const lic of licenses) {
                    if (lic.pack_name) {
                        map.set(lic.pack_name, lic);
                    }
                }
            }
            setLicenseMap(map);
        } catch {
            // Silently ignore - licenses are optional display info
        }
    }, []);

    useEffect(() => {
        if (isOpen) {
            loadPacks();
            loadLicenses();
            setContextMenu(null);
            setDeleteTarget(null);
            setEditTarget(null);
            setInstallTarget(null);
        }
    }, [isOpen, loadPacks]);

    // Close context menu on outside click
    useEffect(() => {
        if (!contextMenu) return;
        const handleClick = (e: MouseEvent) => {
            if (contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
                setContextMenu(null);
            }
        };
        document.addEventListener('mousedown', handleClick);
        return () => document.removeEventListener('mousedown', handleClick);
    }, [contextMenu]);

    // Close on Escape
    useEffect(() => {
        if (!isOpen) return;
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                if (installTarget) {
                    setInstallTarget(null);
                } else if (deleteTarget) {
                    setDeleteTarget(null);
                } else if (contextMenu) {
                    setContextMenu(null);
                } else {
                    onClose();
                }
            }
        };
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [isOpen, contextMenu, deleteTarget, installTarget, onClose]);

    const handleContextMenu = (e: React.MouseEvent, pack: LocalPackInfo) => {
        e.preventDefault();
        e.stopPropagation();
        setContextMenu({ x: e.clientX, y: e.clientY, pack });
    };

    const handleSharePack = (pack: LocalPackInfo) => {
        if (onSharePack) {
            onSharePack(pack);
        }
        setContextMenu(null);
    };

    const handleEditMetadata = (pack: LocalPackInfo) => {
        if (!pack.is_encrypted) {
            setEditTarget(pack);
        }
        setContextMenu(null);
    };

    const handleDeleteClick = (pack: LocalPackInfo) => {
        setDeleteTarget(pack);
        setContextMenu(null);
    };

    const handleInstallClick = async (pack: LocalPackInfo) => {
        setContextMenu(null);
        setInstallTarget(pack);
        setInstallPhase('select-ds');
        setLoadingDataSources(true);
        setSelectedDataSourceId(null);
        setInstallPassword('');
        setInstallLoadResult(null);
        setInstallError(null);
        setInstallExpandedSteps(new Set());
        try {
            const ds = await GetDataSources();
            setDataSources((ds || []).map((d: any) => ({ id: d.id, name: d.name, type: d.type })));
        } catch {
            setDataSources([]);
        } finally {
            setLoadingDataSources(false);
        }
    };

    const handleInstallConfirm = async () => {
        if (!installTarget || !selectedDataSourceId) return;
        setInstallPhase('loading');
        setInstallError(null);
        try {
            const result = await LoadQuickAnalysisPackByPath(installTarget.file_path, selectedDataSourceId);
            if (result.needs_password) {
                setInstallLoadResult(result);
                setInstallPhase('password');
            } else {
                setInstallLoadResult(result);
                setInstallPhase('preview');
            }
        } catch (err: any) {
            setInstallError(err?.message || err?.toString() || 'Failed to load pack');
            setInstallPhase('select-ds');
        }
    };

    const handleInstallPasswordSubmit = async () => {
        if (!installTarget || !installLoadResult?.file_path || !selectedDataSourceId || !installPassword.trim()) return;
        setInstallPhase('loading');
        setInstallError(null);
        try {
            const result = await LoadQuickAnalysisPackWithPassword(installLoadResult.file_path, selectedDataSourceId, installPassword);
            if (result.needs_password) {
                setInstallError(t('import_pack_wrong_password'));
                setInstallPhase('password');
            } else {
                setInstallLoadResult(result);
                setInstallPhase('preview');
            }
        } catch (err: any) {
            setInstallError(err?.message || err?.toString() || t('import_pack_wrong_password'));
            setInstallPhase('password');
        }
    };

    const handleInstallExecute = async () => {
        if (!installTarget || !installLoadResult?.file_path || !selectedDataSourceId) return;
        setInstallPhase('installing');
        setInstallError(null);
        try {
            await ExecuteQuickAnalysisPack(installLoadResult.file_path, selectedDataSourceId, installPassword);
            setInstallTarget(null);
        } catch (err: any) {
            setInstallError(err?.message || err?.toString() || 'Install failed');
            setInstallPhase('preview');
        }
    };

    const toggleInstallStep = (stepId: number) => {
        setInstallExpandedSteps(prev => {
            const next = new Set(prev);
            if (next.has(stepId)) next.delete(stepId);
            else next.add(stepId);
            return next;
        });
    };

    const handleDeleteConfirm = async () => {
        if (!deleteTarget) return;
        setIsDeleting(true);
        try {
            await DeleteLocalPack(deleteTarget.file_path);
            setDeleteTarget(null);
            loadPacks();
        } catch (err: any) {
            const msg = err?.message || err?.toString() || '';
            setError(t('pack_manager_delete_error').replace('{0}', msg));
            setDeleteTarget(null);
        } finally {
            setIsDeleting(false);
        }
    };

    const handleRepurchase = async (license: UsageLicenseInfo) => {
        setLicenseActionLoading(license.pack_name);
        try {
            await PurchaseAdditionalUses(license.listing_id, 1);
            await loadLicenses();
        } catch {
            // Error handled silently - user can retry
        } finally {
            setLicenseActionLoading(null);
        }
    };

    const handleRenew = async (license: UsageLicenseInfo) => {
        setLicenseActionLoading(license.pack_name);
        try {
            await RenewSubscription(license.listing_id);
            await loadLicenses();
        } catch {
            // Error handled silently - user can retry
        } finally {
            setLicenseActionLoading(null);
        }
    };

    const getLicenseStatus = (packName: string): { license: UsageLicenseInfo; isExpired: boolean; isExhausted: boolean } | null => {
        const license = licenseMap.get(packName);
        if (!license || license.pricing_model === 'free') return null;

        if (license.pricing_model === 'per_use') {
            return { license, isExpired: false, isExhausted: license.remaining_uses <= 0 };
        }
        // time_limited or subscription
        if (license.expires_at) {
            const isExpired = new Date(license.expires_at) <= new Date();
            return { license, isExpired, isExhausted: false };
        }
        return { license, isExpired: false, isExhausted: false };
    };

    if (!isOpen) return null;

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[560px] max-h-[70vh] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] flex flex-col"
                onClick={e => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-[#3e3e42]">
                    <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4]">
                        {t('pack_manager_title')}
                    </h3>
                    <button
                        onClick={onClose}
                        className="p-1 rounded-lg hover:bg-slate-100 dark:hover:bg-[#2d2d30] transition-colors"
                    >
                        <X className="w-5 h-5 text-slate-400 dark:text-[#808080]" />
                    </button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-6">
                    {loading && (
                        <div className="flex items-center justify-center py-12 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                            <Loader2 className="w-5 h-5 animate-spin" />
                            {t('pack_manager_loading')}
                        </div>
                    )}

                    {error && !loading && (
                        <div className="text-center py-12">
                            <p className="text-sm text-red-500">{error}</p>
                            <button
                                onClick={loadPacks}
                                className="mt-3 px-4 py-2 text-sm font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
                            >
                                {t('retry')}
                            </button>
                        </div>
                    )}

                    {!loading && !error && packs.length === 0 && (
                        <div className="flex flex-col items-center justify-center py-12 text-slate-400 dark:text-[#6e6e6e]">
                            <Package className="w-12 h-12 mb-3 opacity-50" />
                            <p className="text-sm">{t('pack_manager_empty')}</p>
                        </div>
                    )}

                    {!loading && !error && packs.length > 0 && (
                        <div className="space-y-2">
                            {packs.map(pack => (
                                <div
                                    key={pack.file_path}
                                    className="p-3 rounded-lg border border-slate-200 dark:border-[#3e3e42] hover:bg-slate-50 dark:hover:bg-[#2d2d30] cursor-context-menu transition-colors"
                                    onContextMenu={e => handleContextMenu(e, pack)}
                                >
                                    <div className="flex items-start gap-3">
                                        <div className="mt-0.5 flex-shrink-0 w-8 h-8 rounded-lg bg-blue-50 dark:bg-blue-900/30 flex items-center justify-center">
                                            <Zap className="w-4 h-4 text-blue-500 dark:text-blue-400" />
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-2">
                                                <p className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4] truncate">
                                                    {pack.pack_name}
                                                </p>
                                                {pack.is_encrypted && (
                                                    <span className="flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
                                                        <Lock className="w-3 h-3" />
                                                        {t('pack_manager_encrypted')}
                                                    </span>
                                                )}
                                            </div>
                                            {pack.description && (
                                                <p className="text-xs text-slate-500 dark:text-[#8e8e8e] mt-0.5 truncate">
                                                    {pack.description}
                                                </p>
                                            )}
                                            <div className="flex items-center gap-3 mt-1.5 text-xs text-slate-400 dark:text-[#6e6e6e]">
                                                <span>{t('pack_manager_source')}: {pack.source_name}</span>
                                                {pack.author && <span>{t('pack_manager_author')}: {pack.author}</span>}
                                            </div>
                                            {/* Usage license status */}
                                            {(() => {
                                                const status = getLicenseStatus(pack.pack_name);
                                                if (!status) return null;
                                                const { license, isExpired, isExhausted } = status;
                                                const isLoading = licenseActionLoading === pack.pack_name;
                                                if (license.pricing_model === 'per_use') {
                                                    if (isExhausted) {
                                                        return (
                                                            <div className="flex items-center gap-2 mt-1.5">
                                                                <span className="text-xs text-red-500 dark:text-red-400 flex items-center gap-1">
                                                                    <AlertTriangle className="w-3 h-3" />
                                                                    {t('pack_manager_uses_exhausted')}
                                                                </span>
                                                                <button
                                                                    onClick={e => { e.stopPropagation(); handleRepurchase(license); }}
                                                                    disabled={isLoading}
                                                                    className="text-xs px-2 py-0.5 rounded bg-blue-500 hover:bg-blue-600 text-white disabled:opacity-50 flex items-center gap-1"
                                                                >
                                                                    {isLoading ? <Loader2 className="w-3 h-3 animate-spin" /> : <RefreshCw className="w-3 h-3" />}
                                                                    {t('pack_manager_repurchase')}
                                                                </button>
                                                            </div>
                                                        );
                                                    }
                                                    return (
                                                        <div className="mt-1.5">
                                                            <span className="text-xs text-green-600 dark:text-green-400">
                                                                {t('pack_manager_remaining_uses').replace('{0}', String(license.remaining_uses))}
                                                            </span>
                                                        </div>
                                                    );
                                                }
                                                // time_limited or subscription
                                                if (isExpired) {
                                                    return (
                                                        <div className="flex items-center gap-2 mt-1.5">
                                                            <span className="text-xs text-red-500 dark:text-red-400 flex items-center gap-1">
                                                                <AlertTriangle className="w-3 h-3" />
                                                                {t('pack_manager_expired')}
                                                            </span>
                                                            <button
                                                                onClick={e => { e.stopPropagation(); handleRenew(license); }}
                                                                disabled={isLoading}
                                                                className="text-xs px-2 py-0.5 rounded bg-blue-500 hover:bg-blue-600 text-white disabled:opacity-50 flex items-center gap-1"
                                                            >
                                                                {isLoading ? <Loader2 className="w-3 h-3 animate-spin" /> : <RefreshCw className="w-3 h-3" />}
                                                                {t('pack_manager_renew')}
                                                            </button>
                                                        </div>
                                                    );
                                                }
                                                const dateStr = license.expires_at ? new Date(license.expires_at).toLocaleDateString() : '';
                                                return (
                                                    <div className="mt-1.5">
                                                        <span className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
                                                            <Clock className="w-3 h-3" />
                                                            {t('pack_manager_valid_until').replace('{0}', dateStr)}
                                                        </span>
                                                    </div>
                                                );
                                            })()}
                                        </div>
                                        {/* Action buttons */}
                                        <div className="flex items-center gap-1 flex-shrink-0 mt-0.5">
                                            <button
                                                onClick={e => { e.stopPropagation(); handleSharePack(pack); }}
                                                title={t('pack_manager_share_to_market')}
                                                className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-slate-400 dark:text-[#808080] hover:text-blue-500 dark:hover:text-blue-400"
                                            >
                                                <Share2 className="w-3.5 h-3.5" />
                                            </button>
                                            <button
                                                onClick={e => { e.stopPropagation(); handleInstallClick(pack); }}
                                                title={t('pack_manager_install')}
                                                className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-slate-400 dark:text-[#808080] hover:text-green-500 dark:hover:text-green-400"
                                            >
                                                <Download className="w-3.5 h-3.5" />
                                            </button>
                                            <button
                                                onClick={e => { e.stopPropagation(); handleDeleteClick(pack); }}
                                                title={t('pack_manager_delete')}
                                                className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-slate-400 dark:text-[#808080] hover:text-red-500 dark:hover:text-red-400"
                                            >
                                                <Trash2 className="w-3.5 h-3.5" />
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Hint */}
                {!loading && packs.length > 0 && (
                    <div className="px-6 py-3 border-t border-slate-200 dark:border-[#3e3e42] text-xs text-slate-400 dark:text-[#6e6e6e] text-center">
                        {t('pack_manager_hint')}
                    </div>
                )}
            </div>

            {/* Context Menu */}
            {contextMenu && (
                <div
                    ref={contextMenuRef}
                    className="fixed bg-white dark:bg-[#252526] border border-slate-200 dark:border-[#3c3c3c] rounded-lg shadow-xl z-[9999] w-48 py-1 overflow-hidden"
                    style={{ top: contextMenu.y, left: contextMenu.x }}
                    onContextMenu={e => { e.preventDefault(); e.stopPropagation(); }}
                >
                    <button
                        onClick={() => handleSharePack(contextMenu.pack)}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                    >
                        <Share2 className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                        {t('pack_manager_share_to_market')}
                    </button>
                    <button
                        onClick={() => handleInstallClick(contextMenu.pack)}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                    >
                        <Download className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                        {t('pack_manager_install')}
                    </button>
                    <button
                        onClick={() => handleEditMetadata(contextMenu.pack)}
                        disabled={contextMenu.pack.is_encrypted}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2 disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-transparent dark:disabled:hover:bg-transparent"
                    >
                        <Edit3 className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                        {t('pack_manager_edit_metadata')}
                    </button>
                    <button
                        onClick={() => handleDeleteClick(contextMenu.pack)}
                        className="w-full text-left px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                    >
                        <Trash2 className="w-4 h-4" />
                        {t('pack_manager_delete')}
                    </button>
                </div>
            )}

            {/* Delete Confirmation Dialog */}
            {deleteTarget && (
                <div
                    className="fixed inset-0 z-[10000] flex items-center justify-center bg-black/50"
                    onClick={isDeleting ? undefined : () => setDeleteTarget(null)}
                >
                    <div
                        className="bg-white dark:bg-[#252526] w-[400px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                        onClick={e => e.stopPropagation()}
                    >
                        <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-2">
                            {t('pack_manager_delete_title')}
                        </h3>
                        <p className="text-sm text-slate-600 dark:text-[#9d9d9d] mb-6">
                            {t('pack_manager_delete_message').replace('{0}', deleteTarget.pack_name)}
                        </p>
                        <div className="flex justify-end gap-3">
                            <button
                                onClick={() => setDeleteTarget(null)}
                                disabled={isDeleting}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                {t('cancel')}
                            </button>
                            <button
                                onClick={handleDeleteConfirm}
                                disabled={isDeleting}
                                className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                            >
                                {isDeleting && <Loader2 className="w-4 h-4 animate-spin" />}
                                {t('pack_manager_delete_confirm')}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Install Dialog */}
            {installTarget && (
                <div
                    className="fixed inset-0 z-[10000] flex items-center justify-center bg-black/50"
                    onClick={installPhase === 'loading' || installPhase === 'installing' ? undefined : () => setInstallTarget(null)}
                >
                    <div
                        className="bg-white dark:bg-[#252526] w-[500px] max-h-[70vh] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] flex flex-col"
                        onClick={e => e.stopPropagation()}
                    >
                        <div className="px-6 py-4 border-b border-slate-200 dark:border-[#3e3e42]">
                            <h3 className="text-base font-bold text-slate-800 dark:text-[#d4d4d4]">
                                {t('pack_manager_install')} - {installTarget.pack_name}
                            </h3>
                            {installPhase === 'select-ds' && (
                                <p className="text-xs text-slate-400 dark:text-[#6e6e6e] mt-1">
                                    {t('pack_manager_install_select_datasource')}
                                </p>
                            )}
                        </div>
                        <div className="flex-1 overflow-y-auto p-4">
                            {/* Phase: Select Data Source */}
                            {installPhase === 'select-ds' && (
                                <>
                                    {installError && <p className="mb-3 text-sm text-red-500">{installError}</p>}
                                    {loadingDataSources ? (
                                        <div className="flex items-center justify-center py-8 gap-2 text-sm text-slate-500 dark:text-[#8e8e8e]">
                                            <Loader2 className="w-4 h-4 animate-spin" />
                                            {t('pack_manager_loading')}
                                        </div>
                                    ) : dataSources.length === 0 ? (
                                        <p className="text-center py-8 text-sm text-slate-400 dark:text-[#6e6e6e]">
                                            {t('pack_manager_empty')}
                                        </p>
                                    ) : (
                                        <div className="space-y-2">
                                            {dataSources.map(ds => (
                                                <button
                                                    key={ds.id}
                                                    onClick={() => setSelectedDataSourceId(ds.id)}
                                                    className={`w-full text-left p-3 rounded-lg border-2 transition-all ${
                                                        selectedDataSourceId === ds.id
                                                            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 dark:border-blue-500'
                                                            : 'border-slate-200 dark:border-[#3e3e42] hover:border-blue-300 dark:hover:border-blue-700'
                                                    }`}
                                                >
                                                    <p className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4]">{ds.name}</p>
                                                    <p className="text-xs text-slate-400 dark:text-[#6e6e6e] mt-0.5 uppercase">{ds.type}</p>
                                                </button>
                                            ))}
                                        </div>
                                    )}
                                </>
                            )}

                            {/* Phase: Loading */}
                            {installPhase === 'loading' && (
                                <div className="flex items-center justify-center py-8 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                                    <Loader2 className="w-5 h-5 animate-spin" />
                                    {t('import_pack_loading')}
                                </div>
                            )}

                            {/* Phase: Password */}
                            {installPhase === 'password' && (
                                <div>
                                    <p className="text-sm text-slate-600 dark:text-[#b0b0b0] mb-4">
                                        {t('import_pack_password_required')}
                                    </p>
                                    <input
                                        type="password"
                                        value={installPassword}
                                        onChange={e => { setInstallPassword(e.target.value); setInstallError(null); }}
                                        placeholder={t('import_pack_password_placeholder')}
                                        autoFocus
                                        onKeyDown={e => { if (e.key === 'Enter') handleInstallPasswordSubmit(); }}
                                        className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-[#3e3e42] rounded-lg bg-white dark:bg-[#1e1e1e] text-slate-900 dark:text-[#d4d4d4] placeholder-slate-400 dark:placeholder-[#6e6e6e] focus:outline-none focus:ring-2 focus:ring-blue-500"
                                    />
                                    {installError && <p className="mt-3 text-sm text-red-500">{installError}</p>}
                                </div>
                            )}

                            {/* Phase: Preview / Installing */}
                            {(installPhase === 'preview' || installPhase === 'installing') && installLoadResult?.pack && (
                                <>
                                    {/* Metadata */}
                                    <div className="mb-4 p-3 bg-slate-50 dark:bg-[#1e1e1e] rounded-lg border border-slate-200 dark:border-[#3e3e42]">
                                        <div className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-sm">
                                            <span className="text-slate-500 dark:text-[#8e8e8e]">{t('import_pack_author')}:</span>
                                            <span className="font-medium">{installLoadResult.pack.metadata.author}</span>
                                            <span className="text-slate-500 dark:text-[#8e8e8e]">{t('import_pack_source_name')}:</span>
                                            <span>{installLoadResult.pack.metadata.source_name}</span>
                                            <span className="text-slate-500 dark:text-[#8e8e8e]">{t('import_pack_steps_count')}:</span>
                                            <span>{installLoadResult.pack.executable_steps?.length ?? 0}</span>
                                        </div>
                                    </div>

                                    {/* Schema validation */}
                                    {installLoadResult.validation && (() => {
                                        const v = installLoadResult.validation;
                                        const hasMissingTables = v.missing_tables && v.missing_tables.length > 0;
                                        const hasMissingColumns = v.missing_columns && v.missing_columns.length > 0;
                                        return (
                                            <div className="mb-4">
                                                <p className="text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-2">
                                                    {t('import_pack_schema_validation')}
                                                </p>
                                                {hasMissingTables && (
                                                    <div className="flex items-start gap-2 p-2.5 mb-2 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800/40">
                                                        <XCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                                        <div className="text-sm">
                                                            <p className="text-red-700 dark:text-red-400 font-medium">{t('import_pack_missing_tables')}</p>
                                                            <p className="text-red-600 dark:text-red-300 mt-1">{v.missing_tables.join(', ')}</p>
                                                        </div>
                                                    </div>
                                                )}
                                                {hasMissingColumns && (
                                                    <div className="flex items-start gap-2 p-2.5 mb-2 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800/40">
                                                        <AlertTriangle className="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0" />
                                                        <div className="text-sm">
                                                            <p className="text-amber-700 dark:text-amber-400 font-medium">{t('import_pack_missing_columns')}</p>
                                                            <p className="text-amber-600 dark:text-amber-300 mt-1">
                                                                {v.missing_columns.map((c: any) => `${c.table_name}.${c.column_name}`).join(', ')}
                                                            </p>
                                                        </div>
                                                    </div>
                                                )}
                                                {!hasMissingTables && !hasMissingColumns && (
                                                    <div className="flex items-center gap-2 p-2.5 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800/40">
                                                        <CheckCircle2 className="w-4 h-4 text-green-500 flex-shrink-0" />
                                                        <span className="text-sm text-green-700 dark:text-green-400">{t('import_pack_schema_compatible')}</span>
                                                    </div>
                                                )}
                                            </div>
                                        );
                                    })()}

                                    {/* Python warning */}
                                    {installLoadResult.has_python_steps && !installLoadResult.python_configured && (
                                        <div className="flex items-start gap-2 p-2.5 mb-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800/40">
                                            <XCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                            <p className="text-sm text-red-700 dark:text-red-400">{t('import_pack_python_not_configured')}</p>
                                        </div>
                                    )}

                                    {/* Steps */}
                                    {installLoadResult.pack.executable_steps && installLoadResult.pack.executable_steps.length > 0 && (
                                        <div className="mb-4">
                                            <p className="text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-2">{t('import_pack_contents')}</p>
                                            <div className="border border-slate-200 dark:border-[#3e3e42] rounded-lg overflow-hidden">
                                                {installLoadResult.pack.executable_steps.map((step: any) => {
                                                    const isSql = step.step_type === 'sql_query';
                                                    const isExpanded = installExpandedSteps.has(step.step_id);
                                                    return (
                                                        <div key={step.step_id} className="border-b border-slate-200 dark:border-[#3e3e42] last:border-b-0">
                                                            <button
                                                                type="button"
                                                                onClick={() => toggleInstallStep(step.step_id)}
                                                                className="w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                                            >
                                                                {isExpanded ? <ChevronDown className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" /> : <ChevronRight className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />}
                                                                {isSql ? <Database className="w-3.5 h-3.5 text-blue-500 flex-shrink-0" /> : <FileCode2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />}
                                                                <span className="text-slate-400 dark:text-[#6e6e6e] flex-shrink-0">{t('import_pack_step_label')} {step.step_id}</span>
                                                                <span className="truncate">{step.description}</span>
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

                                    {installError && <p className="mb-3 text-sm text-red-500">{installError}</p>}
                                </>
                            )}
                        </div>
                        <div className="flex justify-end gap-3 px-6 py-4 border-t border-slate-200 dark:border-[#3e3e42]">
                            <button
                                onClick={() => setInstallTarget(null)}
                                disabled={installPhase === 'loading' || installPhase === 'installing'}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                {t('cancel')}
                            </button>
                            {installPhase === 'select-ds' && (
                                <button
                                    onClick={handleInstallConfirm}
                                    disabled={!selectedDataSourceId}
                                    className="px-4 py-2 text-sm font-medium text-white bg-blue-500 hover:bg-blue-600 rounded-md shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    {t('confirm')}
                                </button>
                            )}
                            {installPhase === 'password' && (
                                <button
                                    onClick={handleInstallPasswordSubmit}
                                    disabled={!installPassword.trim()}
                                    className="px-4 py-2 text-sm font-medium text-white bg-blue-500 hover:bg-blue-600 rounded-md shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    {t('confirm')}
                                </button>
                            )}
                            {(installPhase === 'preview' || installPhase === 'installing') && (
                                <button
                                    onClick={handleInstallExecute}
                                    disabled={installPhase === 'installing' || !!(installLoadResult?.validation?.missing_tables?.length) || (installLoadResult?.has_python_steps && !installLoadResult?.python_configured)}
                                    className="px-4 py-2 text-sm font-medium text-white bg-blue-500 hover:bg-blue-600 rounded-md shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                                >
                                    {installPhase === 'installing' && <Loader2 className="w-4 h-4 animate-spin" />}
                                    {installPhase === 'installing' ? t('import_pack_importing') : t('import_pack_confirm')}
                                </button>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* EditPackMetadataDialog */}
            {editTarget && (
                <EditPackMetadataDialog
                    pack={editTarget}
                    onClose={() => setEditTarget(null)}
                    onSaved={() => { setEditTarget(null); loadPacks(); }}
                />
            )}
        </div>,
        document.body
    );
};

export default PackManagerPage;
export type { LocalPackInfo };
