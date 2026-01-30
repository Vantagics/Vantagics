import React, { useEffect, useRef } from 'react';
import { useLanguage } from '../i18n';
import { Sparkles, Settings } from 'lucide-react';

interface DataSourceModeSelectorProps {
    isOpen: boolean;
    position: { x: number; y: number };
    onClose: () => void;
    onSelectBeginnerMode: () => void;
    onSelectExpertMode: () => void;
}

const DataSourceModeSelector: React.FC<DataSourceModeSelectorProps> = ({
    isOpen,
    position,
    onClose,
    onSelectBeginnerMode,
    onSelectExpertMode
}) => {
    const { t } = useLanguage();
    const menuRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (!isOpen) return;

        const handleClickOutside = (e: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
                onClose();
            }
        };

        const handleEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                onClose();
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        document.addEventListener('keydown', handleEscape);

        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
            document.removeEventListener('keydown', handleEscape);
        };
    }, [isOpen, onClose]);

    if (!isOpen) return null;

    return (
        <div
            ref={menuRef}
            className="fixed z-50 bg-white rounded-lg shadow-xl border border-slate-200 py-1 min-w-[180px]"
            style={{ left: position.x, top: position.y }}
        >
            <button
                onClick={() => {
                    onSelectBeginnerMode();
                    onClose();
                }}
                className="w-full px-4 py-2.5 text-left hover:bg-blue-50 flex items-center gap-3 transition-colors"
            >
                <Sparkles className="w-4 h-4 text-blue-500" />
                <span className="text-sm text-slate-700">{t('beginner_wizard') || '新手向导'}</span>
            </button>
            <button
                onClick={() => {
                    onSelectExpertMode();
                    onClose();
                }}
                className="w-full px-4 py-2.5 text-left hover:bg-slate-100 flex items-center gap-3 transition-colors"
            >
                <Settings className="w-4 h-4 text-slate-500" />
                <span className="text-sm text-slate-700">{t('expert_interface') || '专家界面'}</span>
            </button>
        </div>
    );
};

export default DataSourceModeSelector;
