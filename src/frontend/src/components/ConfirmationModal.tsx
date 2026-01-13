import React from 'react';
import { useLanguage } from '../i18n';

interface ConfirmationModalProps {
    isOpen: boolean;
    title: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    onClose: () => void;
    onConfirm: () => void;
}

const ConfirmationModal: React.FC<ConfirmationModalProps> = ({ 
    isOpen, 
    title, 
    message, 
    confirmText, 
    cancelText, 
    onClose, 
    onConfirm 
}) => {
    const { t } = useLanguage();

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="bg-white rounded-xl shadow-2xl p-6 w-[400px] transform transition-all animate-in zoom-in-95 duration-200">
                <h3 className="text-lg font-bold text-slate-900 mb-2">{title}</h3>
                <p className="text-sm text-slate-500 mb-6">
                    {message}
                </p>
                <div className="flex justify-end gap-3">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                        {cancelText || t('cancel')}
                    </button>
                    <button
                        onClick={() => { onConfirm(); onClose(); }}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors"
                    >
                        {confirmText || t('confirm')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default ConfirmationModal;
