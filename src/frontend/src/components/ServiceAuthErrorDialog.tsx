import React from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { ExternalLink } from 'lucide-react';

interface ServiceAuthErrorDialogProps {
    show: boolean;
    errorMessage: string;
    onRetry: () => void;
    onClose: () => void;
}

const ServiceAuthErrorDialog: React.FC<ServiceAuthErrorDialogProps> = ({ show, errorMessage, onRetry, onClose }) => {
    const { t } = useLanguage();

    if (!show) return null;

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape') {
            onClose();
        }
    };

    const handleDirectVisit = () => {
        BrowserOpenURL('https://service.vantagics.com');
        onClose();
    };

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[380px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
                tabIndex={-1}
                role="dialog"
                aria-modal="true"
                aria-labelledby="service-auth-error-title"
            >
                <h3 id="service-auth-error-title" className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-1">
                    {t('service_login_error_title')}
                </h3>
                <p className="text-sm text-red-500 dark:text-red-400 mt-3 mb-5">
                    {errorMessage}
                </p>

                <div className="flex justify-end gap-3">
                    <button
                        onClick={handleDirectVisit}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors flex items-center gap-1.5"
                    >
                        <ExternalLink className="w-3.5 h-3.5" />
                        {t('service_login_direct_visit')}
                    </button>
                    <button
                        onClick={onRetry}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
                    >
                        {t('service_login_retry')}
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default ServiceAuthErrorDialog;
