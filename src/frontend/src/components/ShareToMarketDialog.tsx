import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2 } from 'lucide-react';
import { GetMarketplaceCategories, SharePackToMarketplace } from '../../wailsjs/go/main/App';

interface PackCategory {
    id: number;
    name: string;
    description: string;
    is_preset: boolean;
    pack_count: number;
}

interface ShareToMarketDialogProps {
    packFilePath: string;
    packName: string;
    onClose: () => void;
    onSuccess: () => void;
}

const ShareToMarketDialog: React.FC<ShareToMarketDialogProps> = ({
    packFilePath,
    packName,
    onClose,
    onSuccess,
}) => {
    const { t } = useLanguage();
    const [categories, setCategories] = useState<PackCategory[]>([]);
    const [categoryID, setCategoryID] = useState<number>(0);
    const [shareMode, setShareMode] = useState<'free' | 'paid'>('free');
    const [creditsPrice, setCreditsPrice] = useState<string>('');
    const [isSharing, setIsSharing] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [loadingCategories, setLoadingCategories] = useState(true);

    useEffect(() => {
        const fetchCategories = async () => {
            setLoadingCategories(true);
            try {
                const cats = await GetMarketplaceCategories();
                setCategories(cats || []);
                if (cats && cats.length > 0) {
                    setCategoryID(cats[0].id);
                }
            } catch (err: any) {
                setError(err?.message || err?.toString() || 'Failed to load categories');
            } finally {
                setLoadingCategories(false);
            }
        };
        fetchCategories();
    }, []);

    const priceNum = parseInt(creditsPrice, 10);
    const isPriceValid = shareMode === 'free' || (Number.isInteger(priceNum) && priceNum > 0);
    const canSubmit = categoryID > 0 && isPriceValid && !isSharing && !loadingCategories;

    const handleConfirm = async () => {
        if (!canSubmit) return;
        setIsSharing(true);
        setError(null);
        try {
            const price = shareMode === 'paid' ? priceNum : 0;
            await SharePackToMarketplace(packFilePath, categoryID, shareMode, price);
            onSuccess();
            onClose();
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Share failed');
        } finally {
            setIsSharing(false);
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && canSubmit) {
            handleConfirm();
        } else if (e.key === 'Escape' && !isSharing) {
            onClose();
        }
    };

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={isSharing ? undefined : onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[420px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
            >
                <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-1">
                    {t('share_dialog_title')}
                </h3>
                <p className="text-sm text-slate-500 dark:text-[#8e8e8e] mb-4 truncate">
                    {packName}
                </p>

                {/* Category selector */}
                <div className="mb-4">
                    <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                        {t('share_dialog_category')} <span className="text-red-500">*</span>
                    </label>
                    {loadingCategories ? (
                        <div className="flex items-center gap-2 text-sm text-slate-400 dark:text-[#6e6e6e] py-2">
                            <Loader2 className="w-4 h-4 animate-spin" />
                        </div>
                    ) : (
                        <select
                            value={categoryID}
                            onChange={e => setCategoryID(Number(e.target.value))}
                            disabled={isSharing}
                            className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-[#3e3e42] rounded-lg bg-white dark:bg-[#1e1e1e] text-slate-900 dark:text-[#d4d4d4] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                        >
                            <option value={0} disabled>{t('share_dialog_select_category')}</option>
                            {categories.map(cat => (
                                <option key={cat.id} value={cat.id}>{cat.name}</option>
                            ))}
                        </select>
                    )}
                </div>

                {/* Share mode radio */}
                <div className="mb-4">
                    <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-2">
                        {t('share_dialog_share_mode')} <span className="text-red-500">*</span>
                    </label>
                    <div className="flex items-center gap-4">
                        <label className="flex items-center gap-2 cursor-pointer">
                            <input
                                type="radio"
                                name="shareMode"
                                value="free"
                                checked={shareMode === 'free'}
                                onChange={() => setShareMode('free')}
                                disabled={isSharing}
                                className="accent-blue-600"
                            />
                            <span className="text-sm text-slate-700 dark:text-[#d4d4d4]">{t('share_dialog_free')}</span>
                        </label>
                        <label className="flex items-center gap-2 cursor-pointer">
                            <input
                                type="radio"
                                name="shareMode"
                                value="paid"
                                checked={shareMode === 'paid'}
                                onChange={() => setShareMode('paid')}
                                disabled={isSharing}
                                className="accent-blue-600"
                            />
                            <span className="text-sm text-slate-700 dark:text-[#d4d4d4]">{t('share_dialog_paid')}</span>
                        </label>
                    </div>
                </div>

                {/* Credits price input (shown when paid) */}
                {shareMode === 'paid' && (
                    <div className="mb-4">
                        <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                            {t('share_dialog_credits_price')} <span className="text-red-500">*</span>
                        </label>
                        <input
                            type="number"
                            min="1"
                            step="1"
                            value={creditsPrice}
                            onChange={e => setCreditsPrice(e.target.value)}
                            disabled={isSharing}
                            placeholder="1"
                            className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-[#3e3e42] rounded-lg bg-white dark:bg-[#1e1e1e] text-slate-900 dark:text-[#d4d4d4] placeholder-slate-400 dark:placeholder-[#6e6e6e] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                        />
                    </div>
                )}

                {/* Error message */}
                {error && (
                    <p className="mb-4 text-sm text-red-500">{error}</p>
                )}

                {/* Buttons */}
                <div className="flex justify-end gap-3">
                    <button
                        onClick={onClose}
                        disabled={isSharing}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={handleConfirm}
                        disabled={!canSubmit}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                        {isSharing && <Loader2 className="w-4 h-4 animate-spin" />}
                        {isSharing ? t('share_dialog_sharing') : t('share_dialog_share')}
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default ShareToMarketDialog;
