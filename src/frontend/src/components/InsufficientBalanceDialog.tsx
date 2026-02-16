import React from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';

interface InsufficientBalanceDialogProps {
    currentBalance: number;
    totalCost: number;
    onTopUp: () => void;
    onClose: () => void;
}

const InsufficientBalanceDialog: React.FC<InsufficientBalanceDialogProps> = ({ currentBalance, totalCost, onTopUp, onClose }) => {
    const { t } = useLanguage();
    const deficit = totalCost - currentBalance;
    const backdropMouseDown = React.useRef(false);

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape') {
            onClose();
        }
    };

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[110] flex items-center justify-center bg-black/50 backdrop-blur-sm"
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
                className="bg-white dark:bg-[#252526] w-[380px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
                role="dialog"
                aria-modal="true"
                aria-labelledby="insufficient-balance-dialog-title"
            >
                <h3 id="insufficient-balance-dialog-title" className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-4">
                    {t('insufficient_balance')}
                </h3>

                <div className="space-y-2 mb-6">
                    <div className="flex justify-between text-sm">
                        <span className="text-slate-500 dark:text-[#8e8e8e]">{t('current_balance_label')}</span>
                        <span className="font-medium">{currentBalance} {t('market_browse_credits')}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                        <span className="text-slate-500 dark:text-[#8e8e8e]">{t('balance_needed')}</span>
                        <span className="font-medium">{totalCost} {t('market_browse_credits')}</span>
                    </div>
                    <div className="border-t border-slate-200 dark:border-[#3e3e42] pt-2 flex justify-between text-sm">
                        <span className="text-red-500 dark:text-red-400 font-medium">{t('balance_diff')}</span>
                        <span className="text-red-500 dark:text-red-400 font-medium">{deficit} {t('market_browse_credits')}</span>
                    </div>
                </div>

                <div className="flex justify-end gap-3">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={onTopUp}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors"
                    >
                        {t('go_topup')}
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default InsufficientBalanceDialog;
