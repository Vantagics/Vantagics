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
            <div className="bg-white w-[450px] rounded-xl shadow-2xl flex flex-col overflow-hidden">
                <div className="p-4 border-b border-slate-200 flex justify-between items-center bg-red-50">
                    <div className="flex items-center gap-2">
                        <AlertTriangle className="w-5 h-5 text-red-600" />
                        <h2 className="text-lg font-bold text-red-800">{t('confirm_delete')}</h2>
                    </div>
                    <button onClick={onClose} className="text-slate-500 hover:text-slate-700">
                        <X className="w-5 h-5" />
                    </button>
                </div>

                <div className="p-6">
                    <p className="text-sm text-slate-700 mb-4">
                        {t('delete_table_warning').replace('{tableName}', tableName)}
                    </p>
                    
                    {isLastTable && (
                        <div className="bg-orange-50 border border-orange-200 rounded-lg p-3 mb-4">
                            <p className="text-xs text-orange-800 mb-2">
                                <strong>{t('warning')}:</strong> {t('delete_last_table_warning')}
                            </p>
                            <p className="text-xs text-orange-700">
                                {t('delete_last_table_consequence').replace('{dataSourceName}', dataSourceName)}
                            </p>
                        </div>
                    )}
                    
                    <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
                        <p className="text-xs text-yellow-800">
                            <strong>{t('warning')}:</strong> {t('delete_table_irreversible')}
                        </p>
                    </div>
                </div>

                <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-2">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 hover:bg-slate-50 rounded-md"
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
