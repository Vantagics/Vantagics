import React, { useState, useEffect, useRef, useCallback } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2, Package, X, Share2, Edit3, Trash2, Lock, Zap, Download, Play, AlertTriangle, XCircle, CheckCircle2, ChevronDown, ChevronRight, FileCode2, Database, Clock, RefreshCw, Link, ArrowLeftRight } from 'lucide-react';
import { ListLocalQuickAnalysisPacks, DeleteLocalPack, GetDataSources, LoadQuickAnalysisPackByPath, LoadQuickAnalysisPackWithPassword, ExecuteQuickAnalysisPack, GetUsageLicenses, PurchaseAdditionalUses, RenewSubscription, GetMySharedPackNames, GetShareURL, GetMyPurchasedPacks, DownloadMarketplacePack, GetMyPublishedPacks, ReplaceMarketplacePack, RefreshPurchasedPackLicenses } from '../../wailsjs/go/main/App';
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
    total_uses: number;
    expires_at: string;
    subscription_months: number;
    blocked?: boolean;
}

interface DataSourceInfo {
    id: string;
    name: string;
    type: string;
}

interface PurchasedPackRemoteInfo {
    listing_id: number;
    pack_name: string;
    pack_description: string;
    source_name: string;
    author_name: string;
    share_mode: string;
    credits_price: number;
    created_at: string;
}

// Exported pure function: check if a purchased pack is already downloaded locally
export function isPurchasedPackDownloaded(
    purchasedPack: PurchasedPackRemoteInfo,
    localPacks: LocalPackInfo[]
): boolean {
    return localPacks.some(lp => lp.pack_name === purchasedPack.pack_name);
}

interface PackManagerPageProps {
    isOpen: boolean;
    onClose: () => void;
    onSharePack?: (pack: LocalPackInfo) => void;
    shareVersion?: number;
}

const PackManagerPage: React.FC<PackManagerPageProps> = ({ isOpen, onClose, onSharePack, shareVersion }) => {
    const { t } = useLanguage();
    const [packs, setPacks] = useState<LocalPackInfo[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; pack: LocalPackInfo } | null>(null);
    const contextMenuRef = useRef<HTMLDivElement>(null);

    // Usage license state (pack_name -> license info)
    const [licenseMap, setLicenseMap] = useState<Map<string, UsageLicenseInfo>>(new Map());
    const [licenseActionLoading, setLicenseActionLoading] = useState<string | null>(null);

    // Shared pack names (packs already shared to marketplace)
    const [sharedPackNames, setSharedPackNames] = useState<Set<string>>(new Set());

    // Delete confirmation state
    const [deleteTarget, setDeleteTarget] = useState<LocalPackInfo | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);

    // Edit metadata dialog state
    const [editTarget, setEditTarget] = useState<LocalPackInfo | null>(null);

    const backdropMouseDown = useRef(false);

    // Toast notification state for share URL
    const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

    // Purchased packs state
    const [purchasedPacks, setPurchasedPacks] = useState<PurchasedPackRemoteInfo[]>([]);
    const [purchasedLoading, setPurchasedLoading] = useState(false);
    const [downloadingIds, setDownloadingIds] = useState<Set<number>>(new Set());
    const [downloadErrors, setDownloadErrors] = useState<Map<number, string>>(new Map());

    // Install flow state
    const [installTarget, setInstallTarget] = useState<LocalPackInfo | null>(null);
    const [dataSources, setDataSources] = useState<DataSourceInfo[]>([]);
    const [loadingDataSources, setLoadingDataSources] = useState(false);
    const [selectedDataSourceId, setSelectedDataSourceId] = useState<string | null>(null);

    // Replace flow state
    interface PublishedPackOption {
        listing_id: number;
        pack_name: string;
        source_name: string;
    }
    const [replaceTarget, setReplaceTarget] = useState<LocalPackInfo | null>(null);
    const [publishedPacks, setPublishedPacks] = useState<PublishedPackOption[]>([]);
    const [loadingPublished, setLoadingPublished] = useState(false);
    const [selectedListingId, setSelectedListingId] = useState<number | null>(null);
    const [isReplacing, setIsReplacing] = useState(false);

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

    const loadSharedPackNames = useCallback(async () => {
        try {
            const names = await GetMySharedPackNames();
            setSharedPackNames(new Set(names || []));
        } catch {
            // Silently ignore - shared status is optional display info
        }
    }, []);

    const loadPurchasedPacks = useCallback(async () => {
        setPurchasedLoading(true);
        try {
            const result = await GetMyPurchasedPacks();
            setPurchasedPacks(result || []);
        } catch {
            setPurchasedPacks([]);
        } finally {
            setPurchasedLoading(false);
        }
    }, []);

    const handleRefreshPurchased = useCallback(async () => {
        setPurchasedLoading(true);
        try {
            // Sync license info from server first, then refresh lists
            await RefreshPurchasedPackLicenses().catch(() => {});
            const [purchasedResult] = await Promise.all([
                GetMyPurchasedPacks(),
                loadLicenses(),
            ]);
            setPurchasedPacks(purchasedResult || []);
        } catch {
            setPurchasedPacks([]);
        } finally {
            setPurchasedLoading(false);
        }
    }, [loadLicenses]);

    useEffect(() => {
        if (isOpen) {
            loadPacks();
            loadLicenses();
            loadSharedPackNames();
            loadPurchasedPacks();
            setContextMenu(null);
            setDeleteTarget(null);
            setEditTarget(null);
            setInstallTarget(null);
        }
    }, [isOpen, loadPacks]);

    // Refresh shared pack names when shareVersion changes (after successful share)
    useEffect(() => {
        if (isOpen && shareVersion) {
            loadSharedPackNames();
        }
    }, [shareVersion, isOpen, loadSharedPackNames]);

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
                if (replaceTarget && !isReplacing) {
                    setReplaceTarget(null);
                } else if (installTarget) {
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

    const handleGetShareURL = async (packName: string) => {
        try {
            await GetShareURL(packName);
            setToast({ message: t('pack_manager_url_copied') || '已复制到剪贴板', type: 'success' });
            setTimeout(() => setToast(null), 2000);
        } catch (err: any) {
            const msg = err?.message || err?.toString() || '';
            setToast({ message: (t('pack_manager_url_copy_failed') || '获取链接失败') + (msg ? `: ${msg}` : ''), type: 'error' });
            setTimeout(() => setToast(null), 2000);
        }
    };

    const handleEditMetadata = (pack: LocalPackInfo) => {
        if (!pack.is_encrypted) {
            setEditTarget(pack);
        }
        setContextMenu(null);
    };

    const handleReplaceClick = async (pack: LocalPackInfo) => {
        setContextMenu(null);
        setReplaceTarget(pack);
        setSelectedListingId(null);
        setLoadingPublished(true);
        try {
            const packs = await GetMyPublishedPacks(pack.source_name);
            setPublishedPacks((packs || []).map((p: any) => ({
                listing_id: p.id,
                pack_name: p.pack_name,
                source_name: p.source_name,
            })));
        } catch {
            setPublishedPacks([]);
        } finally {
            setLoadingPublished(false);
        }
    };

    const handleReplaceConfirm = async () => {
        if (!replaceTarget || !selectedListingId) return;
        setIsReplacing(true);
        try {
            await ReplaceMarketplacePack(replaceTarget.file_path, selectedListingId);
            setToast({ message: t('pack_manager_replace_success'), type: 'success' });
            setTimeout(() => setToast(null), 3000);
            setReplaceTarget(null);
            loadSharedPackNames();
        } catch (err: any) {
            const msg = err?.message || err?.toString() || '';
            setToast({ message: (t('pack_manager_replace_failed')) + (msg ? `: ${msg}` : ''), type: 'error' });
            setTimeout(() => setToast(null), 3000);
        } finally {
            setIsReplacing(false);
        }
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

    const handleRenew = async (license: UsageLicenseInfo, months: number = 1) => {
        setLicenseActionLoading(license.pack_name);
        try {
            await RenewSubscription(license.listing_id, months);
            await loadLicenses();
        } catch {
            // Error handled silently - user can retry
        } finally {
            setLicenseActionLoading(null);
        }
    };

    const handleDownloadPurchased = async (pack: PurchasedPackRemoteInfo) => {
        setDownloadingIds(prev => new Set(prev).add(pack.listing_id));
        setDownloadErrors(prev => { const m = new Map(prev); m.delete(pack.listing_id); return m; });
        try {
            await DownloadMarketplacePack(pack.listing_id);
            setToast({ message: t('pack_manager_download_success'), type: 'success' });
            setTimeout(() => setToast(null), 2000);
            // Refresh both lists
            loadPacks();
            loadPurchasedPacks();
        } catch (err: any) {
            const msg = err?.message || err?.toString() || '';
            setDownloadErrors(prev => new Map(prev).set(pack.listing_id, msg));
            setToast({ message: (t('pack_manager_download_failed')) + (msg ? `: ${msg}` : ''), type: 'error' });
            setTimeout(() => setToast(null), 3000);
        } finally {
            setDownloadingIds(prev => { const s = new Set(prev); s.delete(pack.listing_id); return s; });
        }
    };

    const getLicenseStatus = (packName: string): { license: UsageLicenseInfo; isExpired: boolean; isExhausted: boolean } | null => {
        const license = licenseMap.get(packName);
        if (!license || license.pricing_model === 'free') return null;

        if (license.pricing_model === 'per_use') {
            return { license, isExpired: false, isExhausted: license.remaining_uses <= 0 };
        }
        // subscription — use server-validated blocked flag
        if (license.blocked) {
            return { license, isExpired: true, isExhausted: false };
        }
        return { license, isExpired: false, isExhausted: false };
    };

    if (!isOpen) return null;

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onMouseDown={(e) => {
                if (e.target === e.currentTarget) backdropMouseDown.current = true;
            }}
            onMouseUp={(e) => {
                if (e.target === e.currentTarget && backdropMouseDown.current) {
                    onClose();
                }
                backdropMouseDown.current = false;
            }}
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

                    {!loading && !error && packs.filter(p => !p.file_name.startsWith('marketplace_pack_')).length === 0 && (
                        <div className="flex flex-col items-center justify-center py-12 text-slate-400 dark:text-[#6e6e6e]">
                            <Package className="w-12 h-12 mb-3 opacity-50" />
                            <p className="text-sm">{t('pack_manager_empty')}</p>
                        </div>
                    )}

                    {!loading && !error && packs.filter(p => !p.file_name.startsWith('marketplace_pack_')).length > 0 && (
                        <div className="space-y-2">
                            {packs.filter(p => !p.file_name.startsWith('marketplace_pack_')).map(pack => (
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
                                            {/* Local packs have no authorization limits — no license status shown */}
                                        </div>
                                        {/* Action buttons */}
                                        <div className="flex items-center gap-1 flex-shrink-0 mt-0.5">
                                            {sharedPackNames.has(pack.pack_name) ? (
                                                <>
                                                    <span className="flex items-center gap-1 px-2 py-1 text-xs text-green-600 dark:text-green-400">
                                                        <CheckCircle2 className="w-3.5 h-3.5" />
                                                        {t('pack_manager_shared')}
                                                    </span>
                                                    <button
                                                        onClick={e => { e.stopPropagation(); handleGetShareURL(pack.pack_name); }}
                                                        title={t('pack_manager_get_url') || '获取 URL'}
                                                        className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-slate-400 dark:text-[#808080] hover:text-blue-500 dark:hover:text-blue-400"
                                                    >
                                                        <Link className="w-3.5 h-3.5" />
                                                    </button>
                                                </>
                                            ) : (
                                                <>
                                                    <button
                                                        onClick={e => { e.stopPropagation(); handleSharePack(pack); }}
                                                        title={t('pack_manager_share_to_market')}
                                                        className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-slate-400 dark:text-[#808080] hover:text-blue-500 dark:hover:text-blue-400"
                                                    >
                                                        <Share2 className="w-3.5 h-3.5" />
                                                    </button>
                                                    <button
                                                        onClick={e => { e.stopPropagation(); handleReplaceClick(pack); }}
                                                        title={t('pack_manager_replace')}
                                                        className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-slate-400 dark:text-[#808080] hover:text-orange-500 dark:hover:text-orange-400"
                                                    >
                                                        <ArrowLeftRight className="w-3.5 h-3.5" />
                                                    </button>
                                                </>
                                            )}
                                            <button
                                                onClick={e => { e.stopPropagation(); handleInstallClick(pack); }}
                                                title={t('pack_manager_install')}
                                                className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-slate-400 dark:text-[#808080] hover:text-green-500 dark:hover:text-green-400"
                                            >
                                                <Play className="w-3.5 h-3.5" />
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

                    {/* Purchased Packs Section */}
                    <div className="mt-4">
                        <div className="flex items-center justify-between mb-2">
                            <h4 className="text-xs font-semibold text-slate-500 dark:text-[#8e8e8e] uppercase tracking-wide">
                                {t('pack_manager_purchased_section')}
                            </h4>
                            <button
                                onClick={handleRefreshPurchased}
                                disabled={purchasedLoading}
                                title={t('pack_manager_refresh_purchased')}
                                className="flex items-center gap-1 px-2 py-1 text-xs rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-blue-500 dark:text-blue-400 hover:text-blue-600 dark:hover:text-blue-300 disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                {purchasedLoading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RefreshCw className="w-3.5 h-3.5" />}
                                {t('pack_manager_refresh_purchased_btn')}
                            </button>
                        </div>
                    {purchasedLoading && (
                        <div className="flex items-center justify-center py-4 gap-2 text-xs text-slate-400 dark:text-[#6e6e6e]">
                            <Loader2 className="w-4 h-4 animate-spin" />
                        </div>
                    )}
                    {!purchasedLoading && purchasedPacks.length === 0 && (
                        <p className="text-center py-4 text-xs text-slate-400 dark:text-[#6e6e6e]">
                            {t('pack_manager_no_purchased')}
                        </p>
                    )}
                    {!purchasedLoading && purchasedPacks.length > 0 && (
                            <div className="space-y-2">
                                {purchasedPacks.map(pp => {
                                    const alreadyDownloaded = isPurchasedPackDownloaded(pp, packs);
                                    const isDownloading = downloadingIds.has(pp.listing_id);
                                    const dlError = downloadErrors.get(pp.listing_id);
                                    return (
                                        <div
                                            key={pp.listing_id}
                                            className={`p-3 rounded-lg border transition-colors ${alreadyDownloaded ? 'border-slate-200 dark:border-[#3e3e42] bg-slate-50/50 dark:bg-[#2a2a2e]' : 'border-dashed border-slate-300 dark:border-[#4e4e52] hover:bg-slate-50 dark:hover:bg-[#2d2d30]'}`}
                                        >
                                            <div className="flex items-start gap-3">
                                                <div className={`mt-0.5 flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center ${alreadyDownloaded ? 'bg-blue-50 dark:bg-blue-900/30' : 'bg-green-50 dark:bg-green-900/30'}`}>
                                                    <Package className={`w-4 h-4 ${alreadyDownloaded ? 'text-blue-500 dark:text-blue-400' : 'text-green-500 dark:text-green-400'}`} />
                                                </div>
                                                <div className="flex-1 min-w-0">
                                                    <p className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4] truncate">
                                                        {pp.pack_name}
                                                    </p>
                                                    {pp.pack_description && (
                                                        <p className="text-xs text-slate-500 dark:text-[#8e8e8e] mt-0.5 truncate">
                                                            {pp.pack_description}
                                                        </p>
                                                    )}
                                                    <div className="flex items-center gap-3 mt-1.5 text-xs text-slate-400 dark:text-[#6e6e6e]">
                                                        {pp.source_name && <span>{t('pack_manager_source')}: {pp.source_name}</span>}
                                                        {pp.author_name && <span>{t('pack_manager_author')}: {pp.author_name}</span>}
                                                    </div>
                                                    {dlError && (
                                                        <p className="text-xs text-red-500 mt-1 truncate">{dlError}</p>
                                                    )}
                                                    {/* Usage license status for purchased packs */}
                                                    {(() => {
                                                        const status = getLicenseStatus(pp.pack_name);
                                                        // Show free/unlimited for free packs
                                                        if (!status && pp.share_mode === 'free') {
                                                            return (
                                                                <div className="mt-1.5">
                                                                    <span className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
                                                                        <CheckCircle2 className="w-3 h-3" />
                                                                        {t('pack_manager_free_unlimited')}
                                                                    </span>
                                                                </div>
                                                            );
                                                        }
                                                        // Show "not synced" hint for paid packs without license data
                                                        if (!status && (pp.share_mode === 'per_use' || pp.share_mode === 'subscription')) {
                                                            return (
                                                                <div className="mt-1.5">
                                                                    <span className="text-xs text-amber-500 dark:text-amber-400 flex items-center gap-1">
                                                                        <AlertTriangle className="w-3 h-3" />
                                                                        {t('pack_manager_license_not_synced')}
                                                                    </span>
                                                                </div>
                                                            );
                                                        }
                                                        if (!status) return null;
                                                        const { license, isExpired, isExhausted } = status;
                                                        const isLoading = licenseActionLoading === pp.pack_name;
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
                                                            const usedCount = license.total_uses - license.remaining_uses;
                                                            return (
                                                                <div className="mt-1.5 flex items-center gap-2">
                                                                    <span className="text-xs text-green-600 dark:text-green-400">
                                                                        {t('pack_manager_uses_detail').replace('{0}', String(usedCount)).replace('{1}', String(license.total_uses))}
                                                                    </span>
                                                                </div>
                                                            );
                                                        }
                                                        // subscription
                                                        if (isExpired) {
                                                            return (
                                                                <div className="flex items-center gap-2 mt-1.5">
                                                                    <span className="text-xs text-red-500 dark:text-red-400 flex items-center gap-1">
                                                                        <AlertTriangle className="w-3 h-3" />
                                                                        {t('pack_manager_expired')}
                                                                    </span>
                                                                    <button
                                                                        onClick={e => { e.stopPropagation(); handleRenew(license, 1); }}
                                                                        disabled={isLoading}
                                                                        className="text-xs px-2 py-0.5 rounded bg-blue-500 hover:bg-blue-600 text-white disabled:opacity-50 flex items-center gap-1"
                                                                    >
                                                                        {isLoading ? <Loader2 className="w-3 h-3 animate-spin" /> : <RefreshCw className="w-3 h-3" />}
                                                                        {t('pack_manager_renew_monthly')}
                                                                    </button>
                                                                    <button
                                                                        onClick={e => { e.stopPropagation(); handleRenew(license, 12); }}
                                                                        disabled={isLoading}
                                                                        className="text-xs px-2 py-0.5 rounded bg-green-500 hover:bg-green-600 text-white disabled:opacity-50 flex items-center gap-1"
                                                                    >
                                                                        {t('pack_manager_renew_yearly')}
                                                                    </button>
                                                                </div>
                                                            );
                                                        }
                                                        const remainingDays = license.expires_at ? Math.max(0, Math.ceil((new Date(license.expires_at).getTime() - Date.now()) / (1000 * 60 * 60 * 24))) : 0;
                                                        return (
                                                            <div className="mt-1.5">
                                                                <span className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
                                                                    <Clock className="w-3 h-3" />
                                                                    {t('pack_manager_subscription_detail')
                                                                        .replace('{0}', String(license.subscription_months))
                                                                        .replace('{1}', String(remainingDays))}
                                                                </span>
                                                            </div>
                                                        );
                                                    })()}
                                                </div>
                                                {alreadyDownloaded ? (
                                                    <div className="flex items-center gap-1 flex-shrink-0">
                                                        <button
                                                            onClick={() => {
                                                                const localPack = packs.find(lp => lp.pack_name === pp.pack_name);
                                                                if (localPack) handleInstallClick(localPack);
                                                            }}
                                                            title={t('pack_manager_install')}
                                                            className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-slate-400 dark:text-[#808080] hover:text-green-500 dark:hover:text-green-400"
                                                        >
                                                            <Play className="w-3.5 h-3.5" />
                                                        </button>
                                                        <button
                                                            onClick={() => handleDownloadPurchased(pp)}
                                                            disabled={isDownloading}
                                                            title={t('pack_manager_redownload') || '重新下载'}
                                                            className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-[#3e3e42] transition-colors text-slate-400 dark:text-[#808080] hover:text-blue-500 dark:hover:text-blue-400 disabled:opacity-50 disabled:cursor-not-allowed"
                                                        >
                                                            {isDownloading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Download className="w-3.5 h-3.5" />}
                                                        </button>
                                                        <span className="flex items-center gap-1 px-2 py-1.5 text-xs text-green-600 dark:text-green-400">
                                                            <CheckCircle2 className="w-3.5 h-3.5" />
                                                            {t('pack_manager_downloaded')}
                                                        </span>
                                                    </div>
                                                ) : (
                                                    <button
                                                        onClick={() => handleDownloadPurchased(pp)}
                                                        disabled={isDownloading}
                                                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md bg-green-500 hover:bg-green-600 text-white disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex-shrink-0"
                                                    >
                                                        {isDownloading ? (
                                                            <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                                        ) : (
                                                            <Download className="w-3.5 h-3.5" />
                                                        )}
                                                        {isDownloading ? t('pack_manager_downloading') : t('pack_manager_download')}
                                                    </button>
                                                )}
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                    )}
                    </div>
                </div>

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
                        disabled={sharedPackNames.has(contextMenu.pack.pack_name)}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2 disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-transparent dark:disabled:hover:bg-transparent"
                    >
                        {sharedPackNames.has(contextMenu.pack.pack_name) ? (
                            <>
                                <CheckCircle2 className="w-4 h-4 text-green-500" />
                                {t('pack_manager_shared')}
                            </>
                        ) : (
                            <>
                                <Share2 className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                                {t('pack_manager_share_to_market')}
                            </>
                        )}
                    </button>
                    <button
                        onClick={() => handleInstallClick(contextMenu.pack)}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                    >
                        <Play className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                        {t('pack_manager_install')}
                    </button>
                    {!sharedPackNames.has(contextMenu.pack.pack_name) && (
                        <button
                            onClick={() => handleReplaceClick(contextMenu.pack)}
                            className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                        >
                            <ArrowLeftRight className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                            {t('pack_manager_replace')}
                        </button>
                    )}
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
                    onMouseDown={(e) => {
                        if (e.target === e.currentTarget) backdropMouseDown.current = true;
                    }}
                    onMouseUp={(e) => {
                        if (e.target === e.currentTarget && backdropMouseDown.current && !isDeleting) {
                            setDeleteTarget(null);
                        }
                        backdropMouseDown.current = false;
                    }}
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
                    onMouseDown={(e) => {
                        if (e.target === e.currentTarget) backdropMouseDown.current = true;
                    }}
                    onMouseUp={(e) => {
                        if (e.target === e.currentTarget && backdropMouseDown.current && installPhase !== 'loading' && installPhase !== 'installing') {
                            setInstallTarget(null);
                        }
                        backdropMouseDown.current = false;
                    }}
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

                                    {/* License warning for marketplace packs */}
                                    {(() => {
                                        if (!installTarget) return null;
                                        const status = getLicenseStatus(installTarget.pack_name);
                                        if (!status) return null;
                                        const { isExpired: isSubExpired, isExhausted: isUsesExhausted } = status;
                                        if (isUsesExhausted) {
                                            return (
                                                <div className="flex items-start gap-2 p-2.5 mb-3 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800/40">
                                                    <XCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                                    <div className="text-sm">
                                                        <p className="text-red-700 dark:text-red-400 font-medium">{t('import_pack_uses_exhausted')}</p>
                                                        <p className="text-red-600 dark:text-red-300 mt-0.5">{t('import_pack_license_blocked_hint')}</p>
                                                    </div>
                                                </div>
                                            );
                                        }
                                        if (isSubExpired) {
                                            return (
                                                <div className="flex items-start gap-2 p-2.5 mb-3 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800/40">
                                                    <XCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                                    <div className="text-sm">
                                                        <p className="text-red-700 dark:text-red-400 font-medium">{t('import_pack_subscription_expired')}</p>
                                                        <p className="text-red-600 dark:text-red-300 mt-0.5">{t('import_pack_license_blocked_hint')}</p>
                                                    </div>
                                                </div>
                                            );
                                        }
                                        if (status.license.pricing_model === 'per_use' && status.license.remaining_uses > 0) {
                                            return (
                                                <div className="flex items-center gap-2 p-2 mb-3 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800/40">
                                                    <AlertTriangle className="w-4 h-4 text-amber-500 flex-shrink-0" />
                                                    <p className="text-sm text-amber-700 dark:text-amber-400">
                                                        {t('import_pack_remaining_uses').replace('{0}', String(status.license.remaining_uses))}
                                                    </p>
                                                </div>
                                            );
                                        }
                                        return null;
                                    })()}
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
                                    disabled={installPhase === 'installing' || !!(installLoadResult?.validation?.missing_tables?.length) || (installLoadResult?.has_python_steps && !installLoadResult?.python_configured) || !!(installTarget && getLicenseStatus(installTarget.pack_name)?.isExhausted) || !!(installTarget && getLicenseStatus(installTarget.pack_name)?.isExpired)}
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

            {/* Replace Pack Dialog */}
            {replaceTarget && (
                <div
                    className="fixed inset-0 z-[10000] flex items-center justify-center bg-black/50"
                    onMouseDown={(e) => {
                        if (e.target === e.currentTarget) backdropMouseDown.current = true;
                    }}
                    onMouseUp={(e) => {
                        if (e.target === e.currentTarget && backdropMouseDown.current && !isReplacing) {
                            setReplaceTarget(null);
                        }
                        backdropMouseDown.current = false;
                    }}
                >
                    <div
                        className="bg-white dark:bg-[#252526] w-[480px] max-h-[60vh] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] flex flex-col"
                        onClick={e => e.stopPropagation()}
                    >
                        <div className="px-6 py-4 border-b border-slate-200 dark:border-[#3e3e42]">
                            <h3 className="text-base font-bold text-slate-800 dark:text-[#d4d4d4]">
                                {t('pack_manager_replace_title')}
                            </h3>
                            <p className="text-xs text-slate-500 dark:text-[#8e8e8e] mt-1">
                                {t('pack_manager_replace_desc')}
                            </p>
                        </div>
                        <div className="flex-1 overflow-y-auto p-4">
                            {/* Warning */}
                            <div className="flex items-start gap-2 p-2.5 mb-4 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800/40">
                                <AlertTriangle className="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0" />
                                <p className="text-xs text-amber-700 dark:text-amber-400">
                                    {t('pack_manager_replace_warning')}
                                </p>
                            </div>

                            {/* Local pack info */}
                            <div className="mb-3 p-2.5 bg-slate-50 dark:bg-[#1e1e1e] rounded-lg border border-slate-200 dark:border-[#3e3e42]">
                                <p className="text-xs text-slate-500 dark:text-[#8e8e8e]">{t('pack_manager_source')}: {replaceTarget.source_name}</p>
                                <p className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4] mt-0.5">{replaceTarget.pack_name}</p>
                            </div>

                            {/* Published packs list */}
                            {loadingPublished ? (
                                <div className="flex items-center justify-center py-8 gap-2 text-sm text-slate-500 dark:text-[#8e8e8e]">
                                    <Loader2 className="w-4 h-4 animate-spin" />
                                    {t('pack_manager_replace_loading')}
                                </div>
                            ) : publishedPacks.length === 0 ? (
                                <p className="text-center py-8 text-sm text-slate-400 dark:text-[#6e6e6e]">
                                    {t('pack_manager_replace_no_packs')}
                                </p>
                            ) : (
                                <div className="space-y-2">
                                    {publishedPacks.map(pp => (
                                        <button
                                            key={pp.listing_id}
                                            onClick={() => setSelectedListingId(pp.listing_id)}
                                            className={`w-full text-left p-3 rounded-lg border-2 transition-all ${
                                                selectedListingId === pp.listing_id
                                                    ? 'border-orange-500 bg-orange-50 dark:bg-orange-900/20 dark:border-orange-500'
                                                    : 'border-slate-200 dark:border-[#3e3e42] hover:border-orange-300 dark:hover:border-orange-700'
                                            }`}
                                        >
                                            <p className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4]">{pp.pack_name}</p>
                                            <p className="text-xs text-slate-400 dark:text-[#6e6e6e] mt-0.5">{t('pack_manager_source')}: {pp.source_name}</p>
                                        </button>
                                    ))}
                                </div>
                            )}
                        </div>
                        <div className="flex justify-end gap-3 px-6 py-4 border-t border-slate-200 dark:border-[#3e3e42]">
                            <button
                                onClick={() => setReplaceTarget(null)}
                                disabled={isReplacing}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                {t('cancel')}
                            </button>
                            <button
                                onClick={handleReplaceConfirm}
                                disabled={!selectedListingId || isReplacing}
                                className="px-4 py-2 text-sm font-medium text-white bg-orange-500 hover:bg-orange-600 rounded-md shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                            >
                                {isReplacing && <Loader2 className="w-4 h-4 animate-spin" />}
                                {isReplacing ? t('pack_manager_replacing') : t('pack_manager_replace_confirm')}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* EditPackMetadataDialog */}
            {editTarget && (
                <EditPackMetadataDialog
                    pack={editTarget}
                    isShared={sharedPackNames.has(editTarget.pack_name)}
                    onClose={() => setEditTarget(null)}
                    onSaved={() => { setEditTarget(null); loadPacks(); }}
                />
            )}

            {/* Toast notification */}
            {toast && (
                <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-[10001] px-4 py-2 rounded-lg shadow-lg text-sm font-medium flex items-center gap-2 animate-fade-in"
                    style={{
                        backgroundColor: toast.type === 'success' ? '#059669' : '#dc2626',
                        color: 'white',
                    }}
                >
                    {toast.type === 'success' ? <CheckCircle2 className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
                    {toast.message}
                </div>
            )}
        </div>,
        document.body
    );
};

export default PackManagerPage;
export type { LocalPackInfo };
