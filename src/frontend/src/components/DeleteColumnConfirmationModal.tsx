import React from 'react';
import { AlertTriangle, X } from 'lucide-react';
import { useLanguage } from '../i18n';

interface DeleteColumnConfirmationModalProps {
    isOpen: boolean;
    columnName: string;
    tableName: string;
    isLastColumn?: boolean;
    isLastTable?: boolean;
    dataSourceName?: string;
    onClose: () => void;
    onConfirm: () => void;
}

const DeleteColumnConfirmationModal: React.FC<DeleteColumnConfirmationModalProps> = ({
    isOpen,
    columnName,
    tableName,
    isLastColumn = false,
    isLastTable = false,
    dataSourceName = '',
    onClose,
    onConfirm
}) => {
    const { t } = useLanguage();

    if (!isOpen) return null;

    // Determine the warning level
    const willDeleteDataSource = isLastColumn && isLastTable;

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-md mx-4 overflow-hidden">
                {/* Header */}
                <div className={`flex items-center justify-between px-6 py-4 border-b border-slate-200 ${willDeleteDataSource ? 'bg-orange-50' : 'bg-red-50'}`}>
                    <div className="flex items-center gap-3">
                        <div className={`w-10 h-10 rounded-full flex items-center justify-center ${willDeleteDataSource ? 'bg-orange-100' : 'bg-red-100'}`}>
                            <AlertTriangle className={`w-5 h-5 ${willDeleteDataSource ? 'text-orange-600' : 'text-red-600'}`} />
                        </div>
                        <h2 className="text-lg font-semibold text-slate-800">{t('delete_column_title')}</h2>
                    </div>
                    <button
                        onClick={onClose}
                        className={`p-2 rounded-lg transition-colors ${willDeleteDataSource ? 'hover:bg-orange-100' : 'hover:bg-red-100'}`}
                    >
                        <X className="w-5 h-5 text-slate-500" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6">
                    <p className="text-slate-600 mb-4">
                        {t('delete_column_warning').replace('{columnName}', columnName).replace('{tableName}', tableName)}
                    </p>
                    
                    {willDeleteDataSource && (
                        <div className="bg-orange-50 border border-orange-200 rounded-lg p-3 mb-4">
                            <p className="text-sm text-orange-800 font-medium mb-1">
                                {t('warning')}: {t('delete_last_column_warning')}
                            </p>
                            <p className="text-xs text-orange-700">
                                {t('delete_last_column_consequence').replace('{dataSourceName}', dataSourceName)}
                            </p>
                        </div>
                    )}
                    
                    <p className="text-sm text-red-600 bg-red-50 p-3 rounded-lg">
                        {t('delete_column_irreversible')}
                    </p>
                </div>

                {/* Footer */}
                <div className="flex justify-end gap-3 px-6 py-4 border-t border-slate-200 bg-slate-50">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={onConfirm}
                        className={`px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors ${willDeleteDataSource ? 'bg-orange-600 hover:bg-orange-700' : 'bg-red-600 hover:bg-red-700'}`}
                    >
                        {willDeleteDataSource ? t('delete_datasource') : t('delete_column_confirm')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default DeleteColumnConfirmationModal;
