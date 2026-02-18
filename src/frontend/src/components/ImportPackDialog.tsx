import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2, AlertTriangle, XCircle, CheckCircle2, ChevronDown, ChevronRight, FileCode2, Database, Zap, ShoppingBag, Lock, FolderOpen, Package, ArrowLeft, Download } from 'lucide-react';
import {
    LoadQuickAnalysisPack,
    LoadQuickAnalysisPackWithPassword,
    ExecuteQuickAnalysisPack,
    ListLocalQuickAnalysisPacks,
    LoadQuickAnalysisPackByPath,
    GetMyPurchasedPacks,
    DownloadMarketplacePack,
    DeleteLocalPack,
    GetUsageLicenses,
    RefreshPurchasedPackLicenses,
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

// Extract listing_id from marketplace pack file name (e.g. "marketplace_pack_123.qap" -> 123)
function extractListingIdFromFileName(fileName: string): number {
    const match = fileName.match(/^marketplace_pack_(\d+)\.qap$/);
    return match ? parseInt(match[1], 10) : 0;
}

// Get the listing_id for a local marketplace pack by matching against purchased packs
function getListingIdForPack(
    pack: main.LocalPackInfo,
    purchasedPacks: main.PurchasedPackInfo[]
): number {
    // Try extracting from file name first
    const fromFileName = extractListingIdFromFileName(pack.file_name);
    if (fromFileName > 0) return fromFileName;
    // Fallback: match by pack_name in purchased list
    const pp = purchasedPacks.find(p => p.pack_name === pack.pack_name);
    return pp?.listing_id || 0;
}

// Format a license status label for display in the pack list
function formatLicenseLabel(
    license: main.UsageLicense | undefined,
    shareMode: string | undefined,
    t: (key: string) => string
): { text: string; color: 'green' | 'amber' | 'red' } | null {
    // If we have a license record, use it
    if (license) {
        if (license.blocked) {
            return { text: t('import_pack_license_expired'), color: 'red' };
        }
        switch (license.pricing_model) {
            case 'free':
                return { text: t('import_pack_license_free'), color: 'green' };
            case 'per_use': {
                const remaining = license.remaining_uses ?? 0;
                if (remaining <= 0) {
                    return { text: t('import_pack_license_exhausted'), color: 'red' };
                }
                const total = license.total_uses ?? 0;
                return {
                    text: t('import_pack_license_per_use').replace('{0}', String(remaining)).replace('{1}', String(total)),
                    color: remaining <= 3 ? 'amber' : 'green',
                };
            }
            case 'subscription':
            case 'time_limited': {
                if (license.expires_at) {
                    try {
                        const d = new Date(license.expires_at);
                        const dateStr = d.toLocaleDateString();
                        return {
                            text: t('import_pack_license_subscription').replace('{0}', dateStr),
                            color: 'green',
                        };
                    } catch {
                        return { text: t('import_pack_license_subscription').replace('{0}', license.expires_at), color: 'green' };
                    }
                }
                return null;
            }
            default:
                return null;
        }
    }
    // No license record but we know the share mode from purchased info
    if (shareMode === 'free') {
        return { text: t('import_pack_license_free'), color: 'green' };
    }
    return null;
}

const LicenseBadge: React.FC<{ label: { text: string; color: 'green' | 'amber' | 'red' } }> = ({ label }) => {
    const colorClasses = {
        green: 'bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400',
        amber: 'bg-amber-50 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400',
        red: 'bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400',
    };
    return (
        <span className={`text-xs px-1.5 py-0.5 rounded flex-shrink-0 ${colorClasses[label.color]}`}>
            {label.text}
        </span>
    );
};

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
    const [purchasedPacks, setPurchasedPacks] = useState<main.PurchasedPackInfo[]>([]);
    const [purchasedLoading, setPurchasedLoading] = useState(false);
    const [purchasedFailed, setPurchasedFailed] = useState(false);
    const [downloadingId, setDownloadingId] = useState<number | null>(null);
    const [password, setPassword] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [loadResult, setLoadResult] = useState<main.PackLoadResult | null>(null);
    const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set());
    const backdropMouseDown = React.useRef(false);
    const [licenseMap, setLicenseMap] = useState<Map<number, main.UsageLicense>>(new Map());

    const loadAllPacks = () => {
        setPacksLoading(true);
        setPacksError(null);
        setPurchasedLoading(true);
        setPurchasedFailed(false);

        const localPromise = ListLocalQuickAnalysisPacks()
            .then((result) => {
                setPacks(result || []);
                setPacksLoading(false);
                return result || [];
            })
            .catch((err: any) => {
                setPacksError(err?.message || err?.toString() || 'Failed to load packs');
                setPacksLoading(false);
                return [] as main.LocalPackInfo[];
            });

        const purchasedPromise = GetMyPurchasedPacks()
            .then((result) => {
                setPurchasedPacks(result || []);
                return { packs: result || [], failed: false };
            })
            .catch(() => {
                setPurchasedPacks([]);
                setPurchasedFailed(true);
                return { packs: [] as main.PurchasedPackInfo[], failed: true };
            })
            .finally(() => {
                setPurchasedLoading(false);
            });

        // After both requests complete, clean up local marketplace pack files
        // that are no longer in the purchased list (server-side deleted)
        Promise.all([localPromise, purchasedPromise]).then(([localPacks, purchased]) => {
            if (purchased.failed) return; // Don't clean up if we couldn't verify with server
            const purchasedNames = new Set(purchased.packs.map(pp => pp.pack_name));
            const orphaned = localPacks.filter(
                p => p.file_name.startsWith('marketplace_pack_') && !purchasedNames.has(p.pack_name)
            );
            if (orphaned.length === 0) return;
            // Delete orphaned files and refresh the local list
            Promise.all(orphaned.map(p => DeleteLocalPack(p.file_path).catch(() => {}))).then(() => {
                ListLocalQuickAnalysisPacks()
                    .then((result) => setPacks(result || []))
                    .catch(() => {});
            });
        });

        // Load usage licenses for marketplace packs (for permission checks)
        RefreshPurchasedPackLicenses().catch(() => {}).finally(() => {
            GetUsageLicenses()
                .then((licenses) => {
                    const map = new Map<number, main.UsageLicense>();
                    if (licenses) {
                        for (const lic of licenses) {
                            if (lic.listing_id) map.set(lic.listing_id, lic);
                        }
                    }
                    setLicenseMap(map);
                })
                .catch(() => {});
        });
    };

    // Load pack list when dialog opens
    useEffect(() => {
        if (!isOpen) return;
        setState('pack-list');
        setPacks([]);
        setPurchasedPacks([]);
        setPacksLoading(true);
        setPacksError(null);
        setPassword('');
        setError(null);
        setLoadResult(null);
        setExpandedSteps(new Set());
        setDownloadingId(null);

        loadAllPacks();
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

    const handleDownloadAndInstall = async (pp: main.PurchasedPackInfo) => {
        setDownloadingId(pp.listing_id);
        try {
            const filePath = await DownloadMarketplacePack(pp.listing_id);
            // Refresh local packs list to include the newly downloaded pack
            const updatedPacks = await ListLocalQuickAnalysisPacks();
            setPacks(updatedPacks || []);
            // Find the downloaded pack and load it for install
            const downloaded = (updatedPacks || []).find(
                (p: main.LocalPackInfo) => p.file_path === filePath || p.pack_name === pp.pack_name
            );
            if (downloaded) {
                setDownloadingId(null);
                handlePackClick(downloaded);
            } else {
                setDownloadingId(null);
            }
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Download failed');
            setDownloadingId(null);
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

    // License check for marketplace packs
    const listingID = loadResult?.pack?.metadata?.listing_id || 0;
    const license = listingID > 0 ? licenseMap.get(listingID) : undefined;
    const isLicenseExhausted = license?.pricing_model === 'per_use' && (license.remaining_uses ?? 0) <= 0;
    // For subscription packs, use the server-validated 'blocked' flag instead of local expiry check.
    // The backend validates with the server after each execution and sets blocked=true if expired.
    const isLicenseExpired = !!license?.blocked;
    const licenseBlocked = isLicenseExhausted || isLicenseExpired;

    const toggleStep = (stepId: number) => {
        setExpandedSteps(prev => {
            const next = new Set(prev);
            if (next.has(stepId)) next.delete(stepId);
            else next.add(stepId);
            return next;
        });
    };

    const isLoading = !canCloseDialog(state);

    // Separate local-authored packs and marketplace-downloaded packs
    const localPacks = packs.filter(p => !p.file_name.startsWith('marketplace_pack_'));
    const allMarketplacePacks = packs.filter(p => p.file_name.startsWith('marketplace_pack_'));
    // Only keep marketplace local packs that are still in the purchased list (filter out server-side deleted ones)
    // If purchased packs request failed, show all marketplace packs as fallback to avoid hiding valid packs
    const purchasedPackNames = new Set(purchasedPacks.map(pp => pp.pack_name));
    const marketplacePacks = purchasedFailed
        ? allMarketplacePacks
        : allMarketplacePacks.filter(p => purchasedPackNames.has(p.pack_name));
    const marketplacePackNames = new Set(marketplacePacks.map(p => p.pack_name));
    // Purchased packs not yet downloaded locally
    const notDownloadedPurchased = purchasedPacks.filter(pp => !marketplacePackNames.has(pp.pack_name));
    const allLoading = packsLoading && purchasedLoading;
    const hasAnyPack = localPacks.length > 0 || marketplacePacks.length > 0 || notDownloadedPurchased.length > 0;

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onMouseDown={(e) => {
                if (e.target === e.currentTarget) backdropMouseDown.current = true;
            }}
            onMouseUp={(e) => {
                if (e.target === e.currentTarget && backdropMouseDown.current && !isLoading) {
                    onClose();
                }
                backdropMouseDown.current = false;
            }}
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
                        {allLoading && (
                            <div className="flex items-center justify-center py-8 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                                <Loader2 className="w-5 h-5 animate-spin" />
                                {t('import_pack_loading')}
                            </div>
                        )}

                        {packsError && !packsLoading && (
                            <div className="flex flex-col items-center justify-center py-8 gap-3">
                                <p className="text-sm text-red-500">{packsError}</p>
                                <button
                                    onClick={() => loadAllPacks()}
                                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors"
                                >
                                    {t('retry')}
                                </button>
                            </div>
                        )}

                        {!allLoading && !packsError && !hasAnyPack && (
                            <div className="flex flex-col items-center justify-center py-8 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                                <Package className="w-10 h-10 opacity-40" />
                                <p>{t('import_pack_empty_hint')}</p>
                            </div>
                        )}

                        {!allLoading && !packsError && hasAnyPack && (
                            <div className="overflow-y-auto max-h-[calc(80vh-200px)] space-y-1">
                                {/* Local packs */}
                                {localPacks.map((pack, idx) => (
                                    <div
                                        key={pack.file_path || `local-${idx}`}
                                        onClick={() => handlePackClick(pack)}
                                        className="flex items-start gap-3 p-3 rounded-lg border border-slate-200 dark:border-[#3e3e42] hover:bg-slate-50 dark:hover:bg-[#2d2d30] cursor-pointer transition-colors"
                                    >
                                        <div className="flex items-center gap-1 mt-0.5 flex-shrink-0">
                                            {pack.is_encrypted && <Lock className="w-3.5 h-3.5 text-amber-500" />}
                                            <Zap className="w-4 h-4 text-yellow-500" />
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
                                ))}

                                {/* Marketplace-downloaded packs (already local) */}
                                {marketplacePacks.map((pack, idx) => {
                                    const mpListingId = getListingIdForPack(pack, purchasedPacks);
                                    const mpLicense = mpListingId > 0 ? licenseMap.get(mpListingId) : undefined;
                                    const mpPurchased = purchasedPacks.find(pp => pp.pack_name === pack.pack_name);
                                    const mpLabel = formatLicenseLabel(mpLicense, mpPurchased?.share_mode, t);
                                    return (
                                    <div
                                        key={pack.file_path || `mp-${idx}`}
                                        onClick={() => handlePackClick(pack)}
                                        className="flex items-start gap-3 p-3 rounded-lg border border-slate-200 dark:border-[#3e3e42] hover:bg-slate-50 dark:hover:bg-[#2d2d30] cursor-pointer transition-colors"
                                    >
                                        <div className="flex items-center gap-1 mt-0.5 flex-shrink-0">
                                            <ShoppingBag className="w-4 h-4 text-purple-500" />
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-2">
                                                <p className="text-sm font-medium truncate">{pack.pack_name}</p>
                                                <span className="text-xs px-1.5 py-0.5 rounded bg-purple-50 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400 flex-shrink-0">
                                                    {t('import_pack_origin_marketplace')}
                                                </span>
                                                {mpLabel && <LicenseBadge label={mpLabel} />}
                                            </div>
                                            {pack.description && (
                                                <p className="text-xs text-slate-500 dark:text-[#8e8e8e] mt-0.5 truncate">{pack.description}</p>
                                            )}
                                            <p className="text-xs text-slate-400 dark:text-[#6e6e6e] mt-1">
                                                {pack.source_name && <span>{pack.source_name}</span>}
                                                {pack.author && <span>{pack.source_name ? ' · ' : ''}{pack.author}</span>}
                                            </p>
                                        </div>
                                    </div>
                                    );
                                })}

                                {/* Purchased but not yet downloaded */}
                                {notDownloadedPurchased.map((pp) => {
                                    const isDownloading = downloadingId === pp.listing_id;
                                    const ppLicense = pp.listing_id > 0 ? licenseMap.get(pp.listing_id) : undefined;
                                    const ppLabel = formatLicenseLabel(ppLicense, pp.share_mode, t);
                                    return (
                                        <div
                                            key={`purchased-${pp.listing_id}`}
                                            className="flex items-start gap-3 p-3 rounded-lg border border-dashed border-slate-300 dark:border-[#4e4e52] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                        >
                                            <div className="flex items-center gap-1 mt-0.5 flex-shrink-0">
                                                <ShoppingBag className="w-4 h-4 text-green-500" />
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-center gap-2">
                                                    <p className="text-sm font-medium truncate">{pp.pack_name}</p>
                                                    <span className="text-xs px-1.5 py-0.5 rounded bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400 flex-shrink-0">
                                                        {t('import_pack_purchased')}
                                                    </span>
                                                    {ppLabel && <LicenseBadge label={ppLabel} />}
                                                </div>
                                                {pp.pack_description && (
                                                    <p className="text-xs text-slate-500 dark:text-[#8e8e8e] mt-0.5 truncate">{pp.pack_description}</p>
                                                )}
                                                <p className="text-xs text-slate-400 dark:text-[#6e6e6e] mt-1">
                                                    {pp.source_name && <span>{pp.source_name}</span>}
                                                    {pp.author_name && <span>{pp.source_name ? ' · ' : ''}{pp.author_name}</span>}
                                                </p>
                                            </div>
                                            <button
                                                onClick={() => handleDownloadAndInstall(pp)}
                                                disabled={isDownloading || downloadingId !== null}
                                                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md bg-green-500 hover:bg-green-600 text-white disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex-shrink-0 mt-0.5"
                                            >
                                                {isDownloading ? (
                                                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                                ) : (
                                                    <Download className="w-3.5 h-3.5" />
                                                )}
                                                {isDownloading ? t('import_pack_downloading') : t('import_pack_download_install')}
                                            </button>
                                        </div>
                                    );
                                })}
                            </div>
                        )}

                        {error && (
                            <p className="text-sm text-red-500 mt-2">{error}</p>
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

                                {/* License warning for marketplace packs */}
                                {licenseBlocked && (
                                    <div className="flex items-start gap-2 p-2.5 mb-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800/40">
                                        <XCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                        <div className="text-sm">
                                            <p className="text-red-700 dark:text-red-400 font-medium">
                                                {isLicenseExhausted ? t('import_pack_uses_exhausted') : t('import_pack_subscription_expired')}
                                            </p>
                                            <p className="text-red-600 dark:text-red-300 mt-0.5">
                                                {t('import_pack_license_blocked_hint')}
                                            </p>
                                        </div>
                                    </div>
                                )}
                                {!licenseBlocked && license?.pricing_model === 'per_use' && (license.remaining_uses ?? 0) > 0 && (
                                    <div className="flex items-center gap-2 p-2 mb-4 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800/40">
                                        <AlertTriangle className="w-4 h-4 text-amber-500 flex-shrink-0" />
                                        <p className="text-sm text-amber-700 dark:text-amber-400">
                                            {t('import_pack_remaining_uses').replace('{0}', String(license.remaining_uses))}
                                        </p>
                                    </div>
                                )}

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
                                        disabled={!canImportResult || licenseBlocked}
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
