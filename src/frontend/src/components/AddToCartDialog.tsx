import React, { useState } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { main } from '../../wailsjs/go/models';

type PackListingInfo = main.PackListingInfo;

interface AddToCartDialogProps {
    pack: PackListingInfo;
    onConfirm: (quantity: number, isYearly: boolean) => void;
    onClose: () => void;
}

const AddToCartDialog: React.FC<AddToCartDialogProps> = ({ pack, onConfirm, onClose }) => {
    const { t } = useLanguage();
    const isPerUse = pack.share_mode === 'per_use';
    const isSubscription = pack.share_mode === 'subscription';

    const [quantity, setQuantity] = useState<number>(1);
    const [quantityStr, setQuantityStr] = useState<string>('1');
    const [isYearly, setIsYearly] = useState(false);
    const backdropMouseDown = React.useRef(false);

    const handleConfirm = () => {
        onConfirm(quantity >= 1 ? quantity : 1, isYearly);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape') {
            onClose();
        } else if (e.key === 'Enter') {
            handleConfirm();
        }
    };

    const inputClass = "w-full px-3 py-2 text-sm border border-slate-300 dark:border-[#3e3e42] rounded-lg bg-white dark:bg-[#1e1e1e] text-slate-900 dark:text-[#d4d4d4] placeholder-slate-400 dark:placeholder-[#6e6e6e] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[110] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onMouseDown={(e) => {
                if (e.target === e.currentTarget) {
                    backdropMouseDown.current = true;
                } else {
                    backdropMouseDown.current = false;
                }
            }}
            onMouseUp={(e) => {
                if (e.target === e.currentTarget && backdropMouseDown.current) {
                    onClose();
                }
                backdropMouseDown.current = false;
            }}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[380px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
                role="dialog"
                aria-modal="true"
                aria-labelledby="add-to-cart-dialog-title"
            >
                <h3 id="add-to-cart-dialog-title" className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-1">
                    {t('add_to_cart_title')}
                </h3>
                <p className="text-sm text-slate-500 dark:text-[#8e8e8e] mb-4 truncate">
                    {pack.pack_name}
                </p>

                {/* Per-use: quantity input */}
                {isPerUse && (
                    <div className="mb-4">
                        <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                            {t('purchase_quantity')}
                        </label>
                        <input
                            type="number"
                            min={1}
                            step={1}
                            value={quantityStr}
                            onChange={e => {
                                const raw = e.target.value;
                                setQuantityStr(raw);
                                const v = parseInt(raw, 10);
                                if (!isNaN(v) && v >= 1) setQuantity(v);
                            }}
                            onBlur={() => {
                                if (quantity >= 1) setQuantityStr(String(quantity));
                                else { setQuantity(1); setQuantityStr('1'); }
                            }}
                            className={inputClass}
                        />
                    </div>
                )}

                {/* Subscription: monthly/yearly toggle + quantity */}
                {isSubscription && (
                    <>
                        <div className="mb-4">
                            <div className="flex gap-2 mb-3">
                                <button
                                    onClick={() => { setIsYearly(false); setQuantity(1); setQuantityStr('1'); }}
                                    className={`flex-1 px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
                                        !isYearly
                                            ? 'bg-blue-600 text-white'
                                            : 'bg-slate-100 dark:bg-[#2d2d30] text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-200 dark:hover:bg-[#3e3e42]'
                                    }`}
                                >
                                    {t('purchase_mode_monthly')}
                                </button>
                                <button
                                    onClick={() => { setIsYearly(true); setQuantity(1); setQuantityStr('1'); }}
                                    className={`flex-1 px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
                                        isYearly
                                            ? 'bg-blue-600 text-white'
                                            : 'bg-slate-100 dark:bg-[#2d2d30] text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-200 dark:hover:bg-[#3e3e42]'
                                    }`}
                                >
                                    {t('purchase_mode_yearly')}
                                </button>
                            </div>
                        </div>

                        <div className="mb-4">
                            <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                                {isYearly ? t('purchase_years') : t('purchase_months')}
                            </label>
                            <input
                                type="number"
                                min={1}
                                max={isYearly ? 3 : 12}
                                step={1}
                                value={quantityStr}
                                onChange={e => {
                                    const raw = e.target.value;
                                    setQuantityStr(raw);
                                    const v = parseInt(raw, 10);
                                    const max = isYearly ? 3 : 12;
                                    if (!isNaN(v) && v >= 1 && v <= max) setQuantity(v);
                                }}
                                onBlur={() => {
                                    if (quantity >= 1) setQuantityStr(String(quantity));
                                    else { setQuantity(1); setQuantityStr('1'); }
                                }}
                                className={inputClass}
                            />
                            {isYearly && (
                                <p className="mt-2 text-xs text-green-600 dark:text-green-400">
                                    🎁 {t('purchase_yearly_bonus')}
                                </p>
                            )}
                        </div>
                    </>
                )}

                {/* Buttons */}
                <div className="flex justify-end gap-3">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={handleConfirm}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors"
                    >
                        {t('add_to_cart_confirm')}
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default AddToCartDialog;
