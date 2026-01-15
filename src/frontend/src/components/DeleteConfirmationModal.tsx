import React, { useState } from 'react';
import { useLanguage } from '../i18n';
import { Loader2 } from 'lucide-react';

interface DeleteConfirmationModalProps {
    isOpen: boolean;
    sourceName: string;
    onClose: () => void;
    onConfirm: () => void | Promise<void>;
    type?: 'data_source' | 'thread'; // 添加类型参数
}

const DeleteConfirmationModal: React.FC<DeleteConfirmationModalProps> = ({
    isOpen,
    sourceName,
    onClose,
    onConfirm,
    type = 'data_source'
}) => {
    const { t } = useLanguage();
    const [isDeleting, setIsDeleting] = useState(false);

    if (!isOpen) return null;

    // 根据类型选择不同的文本
    const getTitle = () => {
        return type === 'thread' ? t('delete_thread_title') : t('delete_data_source_title');
    };

    const getMessage = () => {
        return type === 'thread' ? t('delete_thread_message') : t('delete_data_source_message');
    };

    const getConfirmButton = () => {
        return type === 'thread' ? t('delete_thread_confirm') : t('delete_data_source_confirm');
    };

    const handleConfirm = async () => {
        setIsDeleting(true);
        try {
            await onConfirm();
            // onConfirm should close the modal by calling onClose or setting state
        } catch (error) {
            console.error('[DeleteConfirmationModal] Error during deletion:', error);
        } finally {
            setIsDeleting(false);
        }
    };

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm" onClick={isDeleting ? undefined : onClose}>
            <div
                className="bg-white w-[400px] rounded-xl shadow-2xl overflow-hidden text-slate-900 p-6"
                onClick={e => e.stopPropagation()}
            >
                <h3 className="text-lg font-bold text-slate-800 mb-2">{getTitle()}</h3>
                <p className="text-sm text-slate-600 mb-6">
                    {getMessage().replace('{0}', sourceName)}
                </p>
                <div className="flex justify-end gap-3">
                    <button
                        onClick={onClose}
                        disabled={isDeleting}
                        className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={handleConfirm}
                        disabled={isDeleting}
                        className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                        {isDeleting && <Loader2 className="w-4 h-4 animate-spin" />}
                        {isDeleting ? t('deleting') || '删除中...' : getConfirmButton()}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default DeleteConfirmationModal;
