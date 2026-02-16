import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2 } from 'lucide-react';
import { ExportQuickAnalysisPack } from '../../wailsjs/go/main/App';

interface ExportPackDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (author: string) => void;
    threadId: string;
}

const ExportPackDialog: React.FC<ExportPackDialogProps> = ({
    isOpen,
    onClose,
    onConfirm,
    threadId,
}) => {
    const { t } = useLanguage();
    const [packName, setPackName] = useState('');
    const [author, setAuthor] = useState('');
    const [isExporting, setIsExporting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [successPath, setSuccessPath] = useState<string | null>(null);
    const backdropMouseDown = React.useRef(false);

    // Reset form when dialog opens
    useEffect(() => {
        if (isOpen) {
            setPackName('');
            setAuthor('');
            setIsExporting(false);
            setError(null);
            setSuccessPath(null);
        }
    }, [isOpen]);

    if (!isOpen) return null;

    const packNameTrimmed = packName.trim();
    const authorTrimmed = author.trim();
    const isPackNameEmpty = packNameTrimmed === '';
    const isAuthorEmpty = authorTrimmed === '';
    const canSubmit = !isPackNameEmpty && !isAuthorEmpty && !isExporting;

    const handleConfirm = async () => {
        if (!canSubmit) return;

        setIsExporting(true);
        setError(null);
        setSuccessPath(null);
        try {
            const savedPath = await ExportQuickAnalysisPack(threadId, packNameTrimmed, authorTrimmed, '');
            setSuccessPath(savedPath);
            onConfirm(authorTrimmed);
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Export failed');
        } finally {
            setIsExporting(false);
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && canSubmit) {
            handleConfirm();
        } else if (e.key === 'Escape' && !isExporting) {
            onClose();
        }
    };

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onMouseDown={(e) => {
                if (e.target === e.currentTarget) backdropMouseDown.current = true;
            }}
            onMouseUp={(e) => {
                if (e.target === e.currentTarget && backdropMouseDown.current && !isExporting) {
                    onClose();
                }
                backdropMouseDown.current = false;
            }}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[420px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
            >
                <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-4">
                    {t('export_pack_title')}
                </h3>

                {/* Pack name input (required) */}
                <div className="mb-4">
                    <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                        {t('export_pack_name')} <span className="text-red-500">*</span>
                    </label>
                    <input
                        type="text"
                        value={packName}
                        onChange={e => setPackName(e.target.value)}
                        placeholder={t('export_pack_name_placeholder')}
                        disabled={isExporting}
                        autoFocus
                        className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-[#3e3e42] rounded-lg bg-white dark:bg-[#1e1e1e] text-slate-900 dark:text-[#d4d4d4] placeholder-slate-400 dark:placeholder-[#6e6e6e] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                    />
                </div>

                {/* Author input (required) */}
                <div className="mb-4">
                    <label className="block text-sm font-medium text-slate-700 dark:text-[#b0b0b0] mb-1">
                        {t('export_pack_author')} <span className="text-red-500">*</span>
                    </label>
                    <input
                        type="text"
                        value={author}
                        onChange={e => setAuthor(e.target.value)}
                        placeholder={t('export_pack_author_placeholder')}
                        disabled={isExporting}
                        className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-[#3e3e42] rounded-lg bg-white dark:bg-[#1e1e1e] text-slate-900 dark:text-[#d4d4d4] placeholder-slate-400 dark:placeholder-[#6e6e6e] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                    />
                </div>

                {/* Success message */}
                {successPath && (
                    <p className="mb-4 text-sm text-green-600 dark:text-green-400 break-all">
                        {t('export_pack_success')}{successPath}
                    </p>
                )}

                {/* Error message */}
                {error && (
                    <p className="mb-4 text-sm text-red-500">{error}</p>
                )}

                {/* Buttons */}
                <div className="flex justify-end gap-3">
                    <button
                        onClick={onClose}
                        disabled={isExporting}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={handleConfirm}
                        disabled={!canSubmit}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                        {isExporting && <Loader2 className="w-4 h-4 animate-spin" />}
                        {isExporting ? t('export_pack_exporting') : t('export')}
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default ExportPackDialog;
