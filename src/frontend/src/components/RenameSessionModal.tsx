import React, { useState, useEffect, useRef } from 'react';
import { useLanguage } from '../i18n';
import { CheckSessionNameExists } from '../../wailsjs/go/main/App';

interface RenameSessionModalProps {
    isOpen: boolean;
    currentTitle: string;
    threadId: string;
    dataSourceId: string;
    dataSourceName?: string;
    onClose: () => void;
    onConfirm: (newTitle: string) => void;
}

const RenameSessionModal: React.FC<RenameSessionModalProps> = ({
    isOpen,
    currentTitle,
    threadId,
    dataSourceId,
    dataSourceName,
    onClose,
    onConfirm,
}) => {
    const { t } = useLanguage();
    const [name, setName] = useState(currentTitle);
    const [error, setError] = useState('');
    const [isChecking, setIsChecking] = useState(false);
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (isOpen) {
            setName(currentTitle);
            setError('');
            setTimeout(() => inputRef.current?.select(), 50);
        }
    }, [isOpen, currentTitle]);

    if (!isOpen) return null;

    const handleConfirm = async () => {
        const trimmed = name.trim();
        if (!trimmed) {
            setError(t('session_name_required') || 'Session name is required');
            return;
        }
        if (trimmed === currentTitle) {
            onClose();
            return;
        }
        // Check duplicate within same data source
        setIsChecking(true);
        try {
            const exists = await CheckSessionNameExists(dataSourceId, trimmed, threadId);
            if (exists) {
                setError(t('session_name_duplicate') || 'A session with this name already exists for this data source');
                setIsChecking(false);
                return;
            }
            setIsChecking(false);
            onConfirm(trimmed);
        } catch (e) {
            setError(String(e));
            setIsChecking(false);
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleConfirm();
        } else if (e.key === 'Escape') {
            onClose();
        }
    };

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[10000]" onClick={onClose} role="dialog" aria-modal="true" aria-label={t('rename_session') || 'Rename Session'}>
            <div
                className="bg-white dark:bg-[#252526] rounded-lg shadow-xl w-96 p-5"
                onClick={(e) => e.stopPropagation()}
            >
                <h3 className="text-base font-semibold text-slate-800 dark:text-[#d4d4d4] mb-4">
                    {t('rename_session') || 'Rename Session'}
                </h3>
                <div className="mb-3">
                    <input
                        ref={inputRef}
                        type="text"
                        value={name}
                        onChange={(e) => { setName(e.target.value); setError(''); }}
                        onKeyDown={handleKeyDown}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-[#3c3c3c] rounded-md text-sm bg-white dark:bg-[#1e1e1e] text-slate-800 dark:text-[#d4d4d4] focus:outline-none focus:ring-2 focus:ring-blue-500"
                        placeholder={t('session_name_placeholder') || 'e.g. Sales Analysis Q1'}
                        aria-label={t('session_name') || 'Session Name'}
                    />
                    {dataSourceName && (
                        <p className="mt-1.5 text-xs text-slate-400 dark:text-[#808080]">
                            {t('rename_session_note') || 'The data source name in parentheses cannot be changed.'}
                        </p>
                    )}
                    {error && (
                        <p className="mt-1.5 text-xs text-red-500">{error}</p>
                    )}
                </div>
                <div className="flex justify-end gap-2">
                    <button
                        onClick={onClose}
                        className="px-4 py-1.5 text-sm rounded-md border border-slate-300 dark:border-[#3c3c3c] text-slate-600 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30]"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={handleConfirm}
                        disabled={isChecking || !name.trim()}
                        className="px-4 py-1.5 text-sm rounded-md bg-blue-500 text-white hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {isChecking ? (t('validating') || 'Validating...') : (t('confirm') || 'Confirm')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default RenameSessionModal;
