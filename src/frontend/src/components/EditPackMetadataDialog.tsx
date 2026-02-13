import React, { useState } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2 } from 'lucide-react';
import { UpdatePackMetadata } from '../../wailsjs/go/main/App';
import type { LocalPackInfo } from './PackManagerPage';

interface EditPackMetadataDialogProps {
    pack: LocalPackInfo;
    onClose: () => void;
    onSaved: () => void;
}

const EditPackMetadataDialog: React.FC<EditPackMetadataDialogProps> = ({ pack, onClose, onSaved }) => {
    const { t } = useLanguage();
    const [description, setDescription] = useState(pack.description);
    const [author, setAuthor] = useState(pack.author);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleSave = async () => {
        setSaving(true);
        setError(null);
        try {
            await UpdatePackMetadata(pack.file_path, description, author);
            onSaved();
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Failed to save metadata');
        } finally {
            setSaving(false);
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape') {
            onClose();
        }
    };

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[10000] flex items-center justify-center bg-black/50"
            onClick={saving ? undefined : onClose}
            onKeyDown={handleKeyDown}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[420px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                onClick={e => e.stopPropagation()}
            >
                <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-1">
                    {t('edit_metadata_title')}
                </h3>
                <p className="text-xs text-slate-400 dark:text-[#6e6e6e] mb-4 truncate">
                    {pack.pack_name}
                </p>

                <div className="space-y-3">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                            {t('edit_metadata_description')}
                        </label>
                        <textarea
                            value={description}
                            onChange={e => setDescription(e.target.value)}
                            rows={3}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-[#3c3c3c] rounded-md text-sm bg-white dark:bg-[#1e1e1e] text-slate-800 dark:text-[#d4d4d4] focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                            {t('edit_metadata_author')}
                        </label>
                        <input
                            type="text"
                            value={author}
                            onChange={e => setAuthor(e.target.value)}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-[#3c3c3c] rounded-md text-sm bg-white dark:bg-[#1e1e1e] text-slate-800 dark:text-[#d4d4d4] focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                    </div>
                </div>

                {error && (
                    <p className="mt-3 text-xs text-red-500">{error}</p>
                )}

                <div className="flex justify-end gap-3 mt-5">
                    <button
                        onClick={onClose}
                        disabled={saving}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={handleSave}
                        disabled={saving}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-500 hover:bg-blue-600 rounded-md shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                        {saving && <Loader2 className="w-4 h-4 animate-spin" />}
                        {t('save_changes')}
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default EditPackMetadataDialog;
