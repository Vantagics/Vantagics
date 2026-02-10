import React, { useEffect } from 'react';
import { X, AlertCircle, CheckCircle, Info, AlertTriangle } from 'lucide-react';
import { useLanguage } from '../i18n';

export type ToastType = 'info' | 'success' | 'warning' | 'error';

interface ToastProps {
    message: string;
    type?: ToastType;
    duration?: number;
    onClose: () => void;
}

const Toast: React.FC<ToastProps> = ({ message, type = 'info', duration = 3000, onClose }) => {
    const { t } = useLanguage();
    
    useEffect(() => {
        if (duration > 0) {
            const timer = setTimeout(() => {
                onClose();
            }, duration);
            return () => clearTimeout(timer);
        }
    }, [duration, onClose]);

    const getIcon = () => {
        switch (type) {
            case 'success':
                return <CheckCircle className="w-5 h-5 text-green-600" />;
            case 'warning':
                return <AlertTriangle className="w-5 h-5 text-amber-600" />;
            case 'error':
                return <AlertCircle className="w-5 h-5 text-red-600" />;
            default:
                return <Info className="w-5 h-5 text-blue-600" />;
        }
    };

    const getStyles = () => {
        switch (type) {
            case 'success':
                return 'bg-green-50 dark:bg-[#1e2a1e] border-green-200 dark:border-[#3d5a3d] text-green-800 dark:text-[#6a9955]';
            case 'warning':
                return 'bg-amber-50 dark:bg-[#2a2620] border-amber-200 dark:border-[#5a5040] text-amber-800 dark:text-[#dcdcaa]';
            case 'error':
                return 'bg-red-50 dark:bg-[#2e1e1e] border-red-200 dark:border-[#5a3d3d] text-red-800 dark:text-[#f14c4c]';
            default:
                return 'bg-blue-50 dark:bg-[#1a2332] border-blue-200 dark:border-[#264f78] text-blue-800 dark:text-[#569cd6]';
        }
    };

    return (
        <div
            className={`fixed top-4 right-4 z-[9999] flex items-center gap-3 px-4 py-3 rounded-lg border shadow-lg animate-in slide-in-from-top-2 fade-in duration-300 ${getStyles()}`}
            style={{ minWidth: '300px', maxWidth: '500px' }}
        >
            {getIcon()}
            <p className="flex-1 text-sm font-medium">{message}</p>
            <button
                onClick={onClose}
                className="p-1 hover:bg-black/5 rounded transition-colors"
                aria-label={t('close_button')}
            >
                <X className="w-4 h-4" />
            </button>
        </div>
    );
};

export default Toast;
