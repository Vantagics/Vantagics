import React from 'react';
import { useLanguage } from '../i18n';

interface DeleteConfirmationModalProps {
    isOpen: boolean;
    sourceName: string;
    onClose: () => void;
    onConfirm: () => void;
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

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm" onClick={onClose}>
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
                        className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-md transition-colors"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={onConfirm}
                        className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md shadow-sm transition-colors"
                    >
                        {getConfirmButton()}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default DeleteConfirmationModal;
