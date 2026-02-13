import React from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2 } from 'lucide-react';

interface SNAuthErrorDialogProps {
    error: string;
    retrying: boolean;
    onRetry: () => void;
    onClose: () => void;
}

const SNAuthErrorDialog: React.FC<SNAuthErrorDialogProps> = ({ error, retrying, onRetry, onClose }) => {
    const { t } = useLanguage();

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape' && !retrying) {
            onClose();
        }
    };

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={retrying ? undefined : onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[380px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
                tabIndex={-1}
                role="dialog"
                aria-modal="true"
                aria-labelledby="sn-auth-error-title"
            >
                <h3 id="sn-auth-error-title" className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-1">
                    {t('sn_auth_error_title')}
                </h3>
                <p className="text-sm text-red-500 dark:text-red-400 mt-3 mb-5">
                    {error}
                </p>

                <div className="flex justify-end gap-3">
                    <button
                        onClick={onClose}
                        disabled={retrying}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={onRetry}
                        disabled={retrying}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {retrying && <Loader2 className="w-4 h-4 animate-spin" />}
                        {retrying ? t('sn_auth_retrying') : t('sn_auth_retry')}
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default SNAuthErrorDialog;
