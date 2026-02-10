import React from 'react';
import { useLanguage } from '../i18n';
import { X, AlertTriangle } from 'lucide-react';

interface DeleteTableConfirmationModalProps {
    isOpen: boolean;
    tableName: string;
    isLastTable?: boolean;
    dataSourceName?: string;
    onClose: () => void;
    onConfirm: () => void;
}

const DeleteTableConfirmationModal: React.FC<DeleteTableConfirmationModalProps> = ({
    isOpen,
    tableName,
    isLastTable = false,
    dataSourceName = '',
    onClose,
    onConfirm,
}) => {
    const { t } = useLanguage();

    console.log('[DEBUG] DeleteTableConfirmationModal render:', { isOpen, tableName, isLastTable, dataSourceName });

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-[10001] flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white dark:bg-[#252526] w-[450px] rounded-xl shadow-2xl flex flex-col overflow-hidden">
                <div className="p-4 border-b border-slate-200 dark:border-[#3c3c3c] flex justify-between items-center bg-red-50 dark:bg-[#2e1e1e]">
                    <div className="flex items-center gap-2">
                        <AlertTriangle className="w-5 h-5 text-red-600 dark:text-[#f14c4c]" />
                        <h2 className="text-lg font-bold text-red-800 dark:text-[#f14c4c]">{t('confirm_delete')}</h2>
                    </div>
                    <button onClick={onClose} className="text-slate-500 dark:text-[#808080] hover:text-slate-700 dark:hover:text-[#d4d4d4]">
                        <X className="w-5 h-5" />
                    </button>
                </div>

                <div className="p-6">
                    <p className="text-sm text-slate-700 dark:text-[#d4d4d4] mb-4">
                        {t('delete_table_warning').replace('{tableName}', tableName)}
                    </p>
                    
                    {isLastTable && (
                        <div className="bg-orange-50 dark:bg-[#2a2620] border border-orange-200 dark:border-[#5a5040] rounded-lg p-3 mb-4">
                            <p className="text-xs text-orange-800 dark:text-[#dcdcaa] mb-2">
                                <strong>{t('warning')}:</strong> {t('delete_last_table_warning')}
                            </p>
                            <p className="text-xs text-orange-700 dark:text-[#dcdcaa]">
                                {t('delete_last_table_consequence').replace('{dataSourceName}', dataSourceName)}
                            </p>
                        </div>
                    )}
                    
                    <div className="bg-yellow-50 dark:bg-[#2a2620] border border-yellow-200 dark:border-[#5a5040] rounded-lg p-3">
                        <p className="text-xs text-yellow-800 dark:text-[#dcdcaa]">
                            <strong>{t('warning')}:</strong> {t('delete_table_irreversible')}
                        </p>
                    </div>
                </div>

                <div className="p-4 border-t border-slate-200 dark:border-[#3c3c3c] bg-slate-50 dark:bg-[#1e1e1e] flex justify-end gap-2">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] bg-white dark:bg-[#3c3c3c] border border-slate-300 dark:border-[#4d4d4d] hover:bg-slate-50 dark:hover:bg-[#4d4d4d] rounded-md"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={onConfirm}
                        className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md"
                    >
                        {t('delete')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default DeleteTableConfirmationModal;
