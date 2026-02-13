import React, { useState, useEffect, useCallback } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2, Package, X, Download, ShoppingCart } from 'lucide-react';
import {
    GetMarketplaceCategories,
    BrowseMarketplacePacks,
    DownloadMarketplacePack,
    GetMarketplaceCreditsBalance,
} from '../../wailsjs/go/main/App';

interface PackCategory {
    id: number;
    name: string;
    description: string;
    is_preset: boolean;
    pack_count: number;
}

interface PackListingInfo {
    id: number;
    category_id: number;
    category_name: string;
    pack_name: string;
    pack_description: string;
    source_name: string;
    author_name: string;
    share_mode: string;  // 'free', 'per_use', 'time_limited', 'subscription'
    credits_price: number;
    valid_days: number;
    billing_cycle: string;  // 'monthly' or 'yearly'
    download_count: number;
    created_at: string;
}

interface MarketBrowsePageProps {
    onClose: () => void;
}

function formatBillingInfo(pack: PackListingInfo, t: (key: string) => string): { label: string; variant: 'free' | 'paid' } {
    switch (pack.share_mode) {
        case 'free':
            return { label: t('market_browse_free'), variant: 'free' };
        case 'per_use':
            return { label: t('market_browse_per_use').replace('{price}', String(pack.credits_price)), variant: 'paid' };
        case 'time_limited':
            return {
                label: t('market_browse_time_limited').replace('{price}', String(pack.credits_price)).replace('{days}', String(pack.valid_days)),
                variant: 'paid',
            };
        case 'subscription': {
            const cycleKey = pack.billing_cycle === 'yearly' ? 'market_browse_sub_yearly' : 'market_browse_sub_monthly';
            return {
                label: t(cycleKey).replace('{price}', String(pack.credits_price)),
                variant: 'paid',
            };
        }
        default:
            return { label: t('market_browse_free'), variant: 'free' };
    }
}

const MarketBrowsePage: React.FC<MarketBrowsePageProps> = ({ onClose }) => {
    const { t } = useLanguage();
    const [categories, setCategories] = useState<PackCategory[]>([]);
    const [packs, setPacks] = useState<PackListingInfo[]>([]);
    const [selectedCategoryID, setSelectedCategoryID] = useState<number>(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [creditsBalance, setCreditsBalance] = useState<number>(0);
    const [downloadingID, setDownloadingID] = useState<number | null>(null);
    const [successMsg, setSuccessMsg] = useState<string | null>(null);

    const fetchCategories = useCallback(async () => {
        try {
            const cats = await GetMarketplaceCategories();
            setCategories(cats || []);
        } catch (_) {
            // categories are optional for browsing
        }
    }, []);

    const fetchBalance = useCallback(async () => {
        try {
            const bal = await GetMarketplaceCreditsBalance();
            setCreditsBalance(bal);
        } catch (_) {
            // balance fetch is best-effort
        }
    }, []);

    const fetchPacks = useCallback(async (categoryID: number) => {
        setLoading(true);
        setError(null);
        try {
            const result = await BrowseMarketplacePacks(categoryID);
            setPacks(result || []);
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Failed to load packs');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchCategories();
        fetchBalance();
        fetchPacks(0);
    }, [fetchCategories, fetchBalance, fetchPacks]);

    const handleCategoryChange = (catID: number) => {
        setSelectedCategoryID(catID);
        fetchPacks(catID);
    };

    const handleDownload = async (pack: PackListingInfo) => {
        if (downloadingID !== null) return;
        setDownloadingID(pack.id);
        setError(null);
        setSuccessMsg(null);
        try {
            const filePath = await DownloadMarketplacePack(pack.id);
            setSuccessMsg(`${t('market_browse_download_success')}${filePath ? ': ' + filePath : ''}`);
            // Refresh balance after purchase
            fetchBalance();
            setTimeout(() => setSuccessMsg(null), 3000);
        } catch (err: any) {
            const msg = err?.message || err?.toString() || 'Download failed';
            setError(msg);
        } finally {
            setDownloadingID(null);
        }
    };

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose();
        };
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [onClose]);

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[640px] max-h-[80vh] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] flex flex-col"
                onClick={e => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-[#3e3e42]">
                    <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4]">
                        {t('market_browse_title')}
                    </h3>
                    <div className="flex items-center gap-3">
                        <span className="text-xs text-slate-500 dark:text-[#8e8e8e]">
                            {t('market_browse_balance')}: {creditsBalance} {t('market_browse_credits')}
                        </span>
                        <button
                            onClick={onClose}
                            className="p-1 rounded-lg hover:bg-slate-100 dark:hover:bg-[#2d2d30] transition-colors"
                        >
                            <X className="w-5 h-5 text-slate-400 dark:text-[#808080]" />
                        </button>
                    </div>
                </div>

                {/* Category filter tabs */}
                <div className="flex items-center gap-2 px-6 py-3 border-b border-slate-200 dark:border-[#3e3e42] overflow-x-auto">
                    <button
                        onClick={() => handleCategoryChange(0)}
                        className={`px-3 py-1 text-xs font-medium rounded-full transition-colors whitespace-nowrap ${
                            selectedCategoryID === 0
                                ? 'bg-blue-600 text-white'
                                : 'bg-slate-100 dark:bg-[#2d2d30] text-slate-600 dark:text-[#b0b0b0] hover:bg-slate-200 dark:hover:bg-[#3e3e42]'
                        }`}
                    >
                        {t('market_browse_all')}
                    </button>
                    {categories.map(cat => (
                        <button
                            key={cat.id}
                            onClick={() => handleCategoryChange(cat.id)}
                            className={`px-3 py-1 text-xs font-medium rounded-full transition-colors whitespace-nowrap ${
                                selectedCategoryID === cat.id
                                    ? 'bg-blue-600 text-white'
                                    : 'bg-slate-100 dark:bg-[#2d2d30] text-slate-600 dark:text-[#b0b0b0] hover:bg-slate-200 dark:hover:bg-[#3e3e42]'
                            }`}
                        >
                            {cat.name}
                        </button>
                    ))}
                </div>

                {/* Success message */}
                {successMsg && (
                    <div className="mx-6 mt-3 px-3 py-2 text-sm text-green-700 dark:text-green-400 bg-green-50 dark:bg-green-900/20 rounded-lg">
                        {successMsg}
                    </div>
                )}

                {/* Error message */}
                {error && (
                    <div className="mx-6 mt-3 px-3 py-2 text-sm text-red-500 bg-red-50 dark:bg-red-900/20 rounded-lg">
                        {error}
                    </div>
                )}

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-6">
                    {loading && (
                        <div className="flex items-center justify-center py-12 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                            <Loader2 className="w-5 h-5 animate-spin" />
                            {t('market_browse_loading')}
                        </div>
                    )}

                    {!loading && !error && packs.length === 0 && (
                        <div className="flex flex-col items-center justify-center py-12 text-slate-400 dark:text-[#6e6e6e]">
                            <Package className="w-12 h-12 mb-3 opacity-50" />
                            <p className="text-sm">{t('market_browse_empty')}</p>
                        </div>
                    )}

                    {!loading && packs.length > 0 && (
                        <div className="space-y-2">
                            {packs.map(pack => (
                                <div
                                    key={pack.id}
                                    className="p-3 rounded-lg border border-slate-200 dark:border-[#3e3e42] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                >
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex-1 min-w-0">
                                            <p className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4] truncate">
                                                {pack.pack_name}
                                            </p>
                                            {pack.pack_description && (
                                                <p className="text-xs text-slate-500 dark:text-[#8e8e8e] mt-0.5 line-clamp-2">
                                                    {pack.pack_description}
                                                </p>
                                            )}
                                            <div className="flex items-center gap-3 mt-1.5 text-xs text-slate-400 dark:text-[#6e6e6e]">
                                                {pack.source_name && <span>{pack.source_name}</span>}
                                                {pack.author_name && <span>{pack.author_name}</span>}
                                                <span>{pack.download_count} {t('market_browse_downloads')}</span>
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2 shrink-0">
                                            {(() => {
                                                const billing = formatBillingInfo(pack, t);
                                                return (
                                                    <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${
                                                        billing.variant === 'free'
                                                            ? 'text-green-700 dark:text-green-400 bg-green-50 dark:bg-green-900/20'
                                                            : 'text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20'
                                                    }`}>
                                                        {billing.label}
                                                    </span>
                                                );
                                            })()}
                                            <button
                                                onClick={() => handleDownload(pack)}
                                                disabled={downloadingID !== null}
                                                className="px-3 py-1.5 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
                                            >
                                                {downloadingID === pack.id ? (
                                                    <>
                                                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                                        {t('market_browse_downloading')}
                                                    </>
                                                ) : pack.share_mode !== 'free' ? (
                                                    <>
                                                        <ShoppingCart className="w-3.5 h-3.5" />
                                                        {t('market_browse_buy_download')}
                                                    </>
                                                ) : (
                                                    <>
                                                        <Download className="w-3.5 h-3.5" />
                                                        {t('market_browse_download')}
                                                    </>
                                                )}
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>
        </div>,
        document.body
    );
};

export default MarketBrowsePage;
