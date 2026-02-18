import React, { useState, useEffect, useCallback, useMemo } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2, Package, X, Download, ShoppingCart, Search, Trash2, CreditCard, RefreshCw, Bell } from 'lucide-react';
import {
    GetMarketplaceCategories,
    BrowseMarketplacePacks,
    DownloadMarketplacePack,
    GetMarketplaceCreditsBalance,
    PurchaseAdditionalUses,
    RenewSubscription,
    EnsureMarketplaceAuth,
    OpenExternalURL,
    GetMarketplaceNotifications,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';
import { filterByShareMode, filterByKeyword, sortPacks, ShareModeFilter, SortField, SortDirection } from '../utils/marketplaceFilter';
import { CartItem, calculateItemCost, calculateTotalCost } from '../utils/marketplaceCart';
import AddToCartDialog from './AddToCartDialog';
import InsufficientBalanceDialog from './InsufficientBalanceDialog';

type PackListingInfo = main.PackListingInfo;

interface PackCategory {
    id: number;
    name: string;
    description: string;
    is_preset: boolean;
    pack_count: number;
}

interface MarketBrowseDialogProps {
    onClose: () => void;
}

const TOPUP_URL = 'https://market.vantagics.com/user/credits';

/**
 * Determines whether the purchased badge should be visible for a pack listing.
 */
export function shouldShowPurchasedBadge(purchased: boolean): boolean {
    return purchased;
}

/**
 * Determines the action button type for a pack listing.
 * Returns 'download' for free packs, 'add_to_cart' for paid packs.
 * Note: the add-to-cart button is hidden in the UI when the pack is already purchased.
 */
export function getActionButtonType(shareMode: string): 'download' | 'add_to_cart' {
    return shareMode === 'free' ? 'download' : 'add_to_cart';
}

function formatBillingInfo(pack: PackListingInfo, t: (key: string) => string): { label: string; variant: 'free' | 'paid' } {
    switch (pack.share_mode) {
        case 'free':
            return { label: t('market_browse_free'), variant: 'free' };
        case 'per_use':
            return { label: t('market_browse_per_use').replace('{price}', String(pack.credits_price)), variant: 'paid' };
        case 'subscription':
            return {
                label: t('market_browse_sub_monthly').replace('{price}', String(pack.credits_price)),
                variant: 'paid',
            };
        default:
            return { label: t('market_browse_free'), variant: 'free' };
    }
}

const MarketBrowseDialog: React.FC<MarketBrowseDialogProps> = ({ onClose }) => {
    const { t } = useLanguage();
    const [categories, setCategories] = useState<PackCategory[]>([]);
    const [packs, setPacks] = useState<PackListingInfo[]>([]);
    const [selectedCategoryID, setSelectedCategoryID] = useState<number>(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [creditsBalance, setCreditsBalance] = useState<number>(0);
    const [downloadingID, setDownloadingID] = useState<number | null>(null);
    const [successMsg, setSuccessMsg] = useState<string | null>(null);
    const [refreshingBalance, setRefreshingBalance] = useState(false);

    // Notifications state
    const [notifications, setNotifications] = useState<any[]>([]);

    // Filter / Sort / Search state
    const [shareModeFilter, setShareModeFilter] = useState<ShareModeFilter>('all');
    const [sortField, setSortField] = useState<SortField>('created_at');
    const [sortDirection, setSortDirection] = useState<SortDirection>('desc');
    const [searchKeyword, setSearchKeyword] = useState('');

    // Cart state
    const [cartItems, setCartItems] = useState<CartItem[]>([]);
    const [addToCartTarget, setAddToCartTarget] = useState<PackListingInfo | null>(null);
    const [isCheckingOut, setIsCheckingOut] = useState(false);
    const [showInsufficientBalance, setShowInsufficientBalance] = useState(false);

    const filteredAndSortedPacks = useMemo(() => {
        let result = filterByShareMode(packs, shareModeFilter);
        result = filterByKeyword(result, searchKeyword);
        result = sortPacks(result, sortField, sortDirection);
        return result;
    }, [packs, shareModeFilter, searchKeyword, sortField, sortDirection]);

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

    const fetchNotifications = useCallback(async () => {
        try {
            const msgs = await GetMarketplaceNotifications();
            setNotifications(msgs || []);
        } catch (_) {
            // notifications fetch is best-effort
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
        // Ensure marketplace auth before fetching packs so the server can
        // identify the user and return correct purchased status.
        EnsureMarketplaceAuth()
            .catch(() => { /* best-effort auth */ })
            .finally(() => {
                fetchCategories();
                fetchBalance();
                fetchNotifications();
                fetchPacks(0);
            });
    }, [fetchCategories, fetchBalance, fetchNotifications, fetchPacks]);

    const handleCategoryChange = (catID: number) => {
        setSelectedCategoryID(catID);
        fetchPacks(catID);
    };

    const handleFreeDownload = async (pack: PackListingInfo) => {
        if (downloadingID !== null) return;
        setDownloadingID(pack.id);
        setError(null);
        setSuccessMsg(null);
        try {
            const filePath = await DownloadMarketplacePack(pack.id);
            setSuccessMsg(`${t('market_browse_download_success')}${filePath ? ': ' + filePath : ''}`);
            fetchBalance();
            setTimeout(() => setSuccessMsg(null), 3000);
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Download failed');
        } finally {
            setDownloadingID(null);
        }
    };

    const handleAddToCart = (pack: PackListingInfo) => {
        setAddToCartTarget(pack);
    };

    const handleAddToCartConfirm = (quantity: number, isYearly: boolean) => {
        if (!addToCartTarget) return;
        setCartItems(prev => [...prev, { pack: addToCartTarget, quantity, isYearly }]);
        setAddToCartTarget(null);
    };

    const handleRemoveCartItem = (index: number) => {
        setCartItems(prev => prev.filter((_, i) => i !== index));
    };

    const handleCheckout = async () => {
        const totalCost = calculateTotalCost(cartItems);
        if (creditsBalance < totalCost) {
            setShowInsufficientBalance(true);
            return;
        }
        setIsCheckingOut(true);
        setError(null);
        try {
            await EnsureMarketplaceAuth();
            for (const item of cartItems) {
                if (item.pack.share_mode === 'per_use') {
                    await PurchaseAdditionalUses(item.pack.id, item.quantity);
                } else if (item.pack.share_mode === 'subscription') {
                    const months = item.isYearly ? 12 * item.quantity : item.quantity;
                    await RenewSubscription(item.pack.id, months);
                }
            }
            setCartItems([]);
            setSuccessMsg(t('market_browse_checkout_success'));
            fetchBalance();
            fetchPacks(selectedCategoryID);
            setTimeout(() => setSuccessMsg(null), 3000);
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Checkout failed');
        } finally {
            setIsCheckingOut(false);
        }
    };

    const handleTopUp = () => {
        OpenExternalURL(TOPUP_URL);
    };

    const handleRefreshBalance = async () => {
        setRefreshingBalance(true);
        try {
            const bal = await GetMarketplaceCreditsBalance();
            setCreditsBalance(bal);
        } catch (_) {
            // best-effort
        } finally {
            setRefreshingBalance(false);
        }
    };

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // Don't close market page if a sub-dialog (e.g. AddToCartDialog) is open
            if (addToCartTarget || showInsufficientBalance) return;
            if (e.key === 'Escape') onClose();
        };
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [onClose, addToCartTarget, showInsufficientBalance]);

    const sortOptions = [
        { field: 'created_at' as SortField, direction: 'desc' as SortDirection, label: t('sort_time_desc') },
        { field: 'created_at' as SortField, direction: 'asc' as SortDirection, label: t('sort_time_asc') },
        { field: 'credits_price' as SortField, direction: 'desc' as SortDirection, label: t('sort_price_desc') },
        { field: 'credits_price' as SortField, direction: 'asc' as SortDirection, label: t('sort_price_asc') },
    ];

    const totalCost = calculateTotalCost(cartItems);

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[720px] max-h-[85vh] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] flex flex-col"
                onClick={e => e.stopPropagation()}
                role="dialog"
                aria-modal="true"
                aria-labelledby="market-browse-dialog-title"
            >
                {/* Header: title + balance + top-up + close */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-[#3e3e42]">
                    <h3 id="market-browse-dialog-title" className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4]">
                        {t('market_browse_title')}
                    </h3>
                    <div className="flex items-center gap-3">
                        <span className="text-xs text-slate-500 dark:text-[#8e8e8e]">
                            {t('market_browse_balance')}: {creditsBalance} {t('market_browse_credits')}
                        </span>
                        <button
                            onClick={handleRefreshBalance}
                            disabled={refreshingBalance}
                            className="p-1 rounded-lg hover:bg-slate-100 dark:hover:bg-[#2d2d30] transition-colors disabled:opacity-50"
                            title={t('market_browse_refresh_balance')}
                        >
                            <RefreshCw className={`w-3.5 h-3.5 text-slate-500 dark:text-[#8e8e8e] ${refreshingBalance ? 'animate-spin' : ''}`} />
                        </button>
                        <button
                            onClick={handleTopUp}
                            className="px-2.5 py-1 text-xs font-medium text-blue-600 dark:text-blue-400 border border-blue-300 dark:border-blue-600 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors"
                        >
                            {t('market_browse_topup')}
                        </button>
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

                {/* Toolbar: share mode filter + sort + search */}
                <div className="flex items-center gap-2 px-6 py-2.5 border-b border-slate-200 dark:border-[#3e3e42]">
                    {/* Share mode filter buttons */}
                    <div className="flex items-center gap-1">
                        {([
                            { value: 'all' as ShareModeFilter, label: t('filter_all') },
                            { value: 'free' as ShareModeFilter, label: t('filter_free') },
                            { value: 'per_use' as ShareModeFilter, label: t('filter_per_use') },
                            { value: 'subscription' as ShareModeFilter, label: t('filter_subscription') },
                        ]).map(opt => (
                            <button
                                key={opt.value}
                                onClick={() => setShareModeFilter(opt.value)}
                                className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors ${
                                    shareModeFilter === opt.value
                                        ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                                        : 'text-slate-500 dark:text-[#8e8e8e] hover:bg-slate-100 dark:hover:bg-[#2d2d30]'
                                }`}
                            >
                                {opt.label}
                            </button>
                        ))}
                    </div>

                    {/* Sort dropdown */}
                    <select
                        value={`${sortField}|${sortDirection}`}
                        onChange={e => {
                            const parts = e.target.value.split('|');
                            setSortField(parts[0] as SortField);
                            setSortDirection(parts[1] as SortDirection);
                        }}
                        className="ml-auto px-2 py-1 text-xs border border-slate-200 dark:border-[#3e3e42] rounded-md bg-white dark:bg-[#1e1e1e] text-slate-700 dark:text-[#d4d4d4] focus:outline-none focus:ring-1 focus:ring-blue-500"
                    >
                        {sortOptions.map(opt => (
                            <option key={`${opt.field}|${opt.direction}`} value={`${opt.field}|${opt.direction}`}>
                                {opt.label}
                            </option>
                        ))}
                    </select>

                    {/* Search input */}
                    <div className="relative">
                        <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400 dark:text-[#6e6e6e]" />
                        <input
                            type="text"
                            value={searchKeyword}
                            onChange={e => setSearchKeyword(e.target.value)}
                            placeholder={t('search_packs_placeholder')}
                            className="pl-7 pr-3 py-1 text-xs w-40 border border-slate-200 dark:border-[#3e3e42] rounded-md bg-white dark:bg-[#1e1e1e] text-slate-700 dark:text-[#d4d4d4] placeholder-slate-400 dark:placeholder-[#6e6e6e] focus:outline-none focus:ring-1 focus:ring-blue-500"
                        />
                    </div>
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

                {/* Notification banners */}
                {notifications.length > 0 && (
                    <div className="mx-6 mt-3 space-y-2">
                        <div className="flex items-center gap-1.5 text-xs font-medium text-amber-700 dark:text-amber-400">
                            <Bell className="w-3.5 h-3.5" />
                            {t('market_notifications_title')}
                        </div>
                        {notifications.map((n: any) => (
                            <div
                                key={n.id}
                                className="px-3 py-2 text-sm rounded-lg border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-900/20"
                            >
                                <p className="font-medium text-amber-800 dark:text-amber-300">{n.title}</p>
                                <p className="text-amber-700 dark:text-amber-400 mt-0.5 text-xs">{n.content}</p>
                            </div>
                        ))}
                    </div>
                )}

                {/* Pack list */}
                <div className="flex-1 overflow-y-auto p-6">
                    {loading && (
                        <div className="flex items-center justify-center py-12 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                            <Loader2 className="w-5 h-5 animate-spin" />
                            {t('market_browse_loading')}
                        </div>
                    )}

                    {!loading && !error && filteredAndSortedPacks.length === 0 && (
                        <div className="flex flex-col items-center justify-center py-12 text-slate-400 dark:text-[#6e6e6e]">
                            <Package className="w-12 h-12 mb-3 opacity-50" />
                            <p className="text-sm">{t('market_browse_empty')}</p>
                        </div>
                    )}

                    {!loading && filteredAndSortedPacks.length > 0 && (
                        <div className="space-y-2">
                            {filteredAndSortedPacks.map(pack => (
                                <div
                                    key={pack.id}
                                    className="p-3 rounded-lg border border-slate-200 dark:border-[#3e3e42] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                >
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex-1 min-w-0">
                                            <p className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4] truncate">
                                                {pack.pack_name}
                                                {pack.purchased && (
                                                    <span className="ml-1.5 px-1.5 py-0.5 text-xs font-medium rounded-full text-blue-700 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/20">
                                                        {t('market_browse_purchased')}
                                                    </span>
                                                )}
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
                                            {pack.share_mode === 'free' ? (
                                                <button
                                                    onClick={() => handleFreeDownload(pack)}
                                                    disabled={downloadingID !== null}
                                                    className="px-3 py-1.5 text-xs font-medium text-white bg-green-600 hover:bg-green-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
                                                >
                                                    {downloadingID === pack.id ? (
                                                        <>
                                                            <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                                            {t('market_browse_downloading')}
                                                        </>
                                                    ) : (
                                                        <>
                                                            <Download className="w-3.5 h-3.5" />
                                                            {t('market_browse_free_download')}
                                                        </>
                                                    )}
                                                </button>
                                            ) : !pack.purchased ? (
                                                <button
                                                    onClick={() => handleAddToCart(pack)}
                                                    className="px-3 py-1.5 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors flex items-center gap-1.5"
                                                >
                                                    <ShoppingCart className="w-3.5 h-3.5" />
                                                    {t('market_browse_add_to_cart')}
                                                </button>
                                            ) : null}
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Cart summary footer */}
                {cartItems.length > 0 && (
                    <div className="border-t border-slate-200 dark:border-[#3e3e42] px-6 py-3">
                        <div className="flex items-center justify-between mb-2">
                            <span className="text-sm font-medium text-slate-700 dark:text-[#d4d4d4]">
                                {t('cart_summary').replace('{count}', String(cartItems.length))}
                            </span>
                            <span className="text-sm font-bold text-slate-800 dark:text-[#d4d4d4]">
                                {t('cart_total').replace('{cost}', String(totalCost))}
                            </span>
                        </div>
                        <div className="max-h-24 overflow-y-auto space-y-1 mb-2">
                            {cartItems.map((item, idx) => (
                                <div key={idx} className="flex items-center justify-between text-xs text-slate-600 dark:text-[#b0b0b0]">
                                    <span className="truncate flex-1 mr-2">
                                        {item.pack.pack_name}
                                        {item.pack.share_mode === 'per_use' && ` × ${item.quantity}`}
                                        {item.pack.share_mode === 'subscription' && (item.isYearly ? ` × ${item.quantity}yr` : ` × ${item.quantity}mo`)}
                                    </span>
                                    <span className="mr-2 whitespace-nowrap">{calculateItemCost(item)} {t('market_browse_credits')}</span>
                                    <button
                                        onClick={() => handleRemoveCartItem(idx)}
                                        className="p-0.5 text-red-400 hover:text-red-600 dark:text-red-400 dark:hover:text-red-300 transition-colors"
                                        title={t('cart_remove')}
                                    >
                                        <Trash2 className="w-3.5 h-3.5" />
                                    </button>
                                </div>
                            ))}
                        </div>
                        <button
                            onClick={handleCheckout}
                            disabled={isCheckingOut || cartItems.length === 0}
                            className="w-full px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                        >
                            {isCheckingOut ? (
                                <>
                                    <Loader2 className="w-4 h-4 animate-spin" />
                                    {t('market_browse_checking_out')}
                                </>
                            ) : (
                                <>
                                    <CreditCard className="w-4 h-4" />
                                    {t('market_browse_checkout')}
                                </>
                            )}
                        </button>
                    </div>
                )}
            </div>

            {/* AddToCartDialog */}
            {addToCartTarget && (
                <AddToCartDialog
                    pack={addToCartTarget}
                    onConfirm={handleAddToCartConfirm}
                    onClose={() => setAddToCartTarget(null)}
                />
            )}

            {/* InsufficientBalanceDialog */}
            {showInsufficientBalance && (
                <InsufficientBalanceDialog
                    currentBalance={creditsBalance}
                    totalCost={totalCost}
                    onTopUp={() => {
                        setShowInsufficientBalance(false);
                        handleTopUp();
                    }}
                    onClose={() => setShowInsufficientBalance(false)}
                />
            )}
        </div>,
        document.body
    );
};

export default MarketBrowseDialog;
