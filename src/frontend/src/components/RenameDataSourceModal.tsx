import React, { useState, useEffect } from 'react';
import { X, Edit3, AlertCircle } from 'lucide-react';
import { useLanguage } from '../i18n';

interface RenameDataSourceModalProps {
    isOpen: boolean;
    currentName: string;
    onClose: () => void;
    onRename: (newName: string) => Promise<void>;
}

const RenameDataSourceModal: React.FC<RenameDataSourceModalProps> = ({
    isOpen,
    currentName,
    onClose,
    onRename
}) => {
    const { t } = useLanguage();
    const [newName, setNewName] = useState(currentName);
    const [isRenaming, setIsRenaming] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (isOpen) {
            setNewName(currentName);
            setError(null);
        }
    }, [isOpen, currentName]);

    const handleRename = async () => {
        const trimmedName = newName.trim();
        
        if (!trimmedName) {
            setError(t('data_source_name_required') || 'Data source name is required');
            return;
        }

        if (trimmedName === currentName) {
            onClose();
            return;
        }

        setIsRenaming(true);
        setError(null);

        try {
            await onRename(trimmedName);
            onClose();
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsRenaming(false);
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !isRenaming) {
            handleRename();
        } else if (e.key === 'Escape') {
            onClose();
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-md">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-slate-200">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-blue-100 rounded-lg">
                            <Edit3 className="w-5 h-5 text-blue-600" />
                        </div>
                        <h2 className="text-xl font-semibold text-slate-800">
                            {t('rename_data_source') || 'Rename Data Source'}
                        </h2>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-1 hover:bg-slate-100 rounded-lg transition-colors"
                        disabled={isRenaming}
                    >
                        <X className="w-5 h-5 text-slate-400" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-2">
                            {t('data_source_name') || 'Data Source Name'}
                        </label>
                        <input
                            type="text"
                            value={newName}
                            onChange={(e) => setNewName(e.target.value)}
                            onKeyDown={handleKeyDown}
                            className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            placeholder={t('enter_data_source_name') || 'Enter data source name'}
                            autoFocus
                            disabled={isRenaming}
                        />
                    </div>

                    {error && (
                        <div className="flex items-start gap-2 p-3 bg-red-50 border border-red-200 rounded-lg">
                            <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
                            <p className="text-sm text-red-800">{error}</p>
                        </div>
                    )}

                    <div className="text-xs text-slate-500">
                        <p>{t('rename_data_source_note') || 'Note: The data source name must be unique.'}</p>
                    </div>
                </div>

                {/* Footer */}
                <div className="flex items-center justify-end gap-3 p-6 border-t border-slate-200 bg-slate-50">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-200 rounded-lg transition-colors"
                        disabled={isRenaming}
                    >
                        {t('cancel') || 'Cancel'}
                    </button>
                    <button
                        onClick={handleRename}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                        disabled={isRenaming || !newName.trim()}
                    >
                        {isRenaming ? (
                            <>
                                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                                {t('renaming') || 'Renaming...'}
                            </>
                        ) : (
                            <>
                                <Edit3 className="w-4 h-4" />
                                {t('rename') || 'Rename'}
                            </>
                        )}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default RenameDataSourceModal;
