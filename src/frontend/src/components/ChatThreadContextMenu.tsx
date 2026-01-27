import React, { useEffect, useRef } from 'react';
import { Download, Briefcase, Brain, FolderOpen, Check } from 'lucide-react';
import { useLanguage } from '../i18n';

interface ChatThreadContextMenuProps {
    position: { x: number; y: number };
    threadId: string;
    onClose: () => void;
    onAction: (action: 'export' | 'view_memory' | 'view_results_directory' | 'toggle_intent_understanding', threadId: string) => void;
    autoIntentUnderstanding?: boolean;
}

const ChatThreadContextMenu: React.FC<ChatThreadContextMenuProps> = ({ position, threadId, onClose, onAction, autoIntentUnderstanding = true }) => {
    const { t } = useLanguage();
    const menuRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                onClose();
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, [onClose]);

    const handleAction = (action: 'export' | 'view_memory' | 'view_results_directory' | 'toggle_intent_understanding') => {
        onAction(action, threadId);
        // Don't close menu for toggle action so user can see the state change
        if (action !== 'toggle_intent_understanding') {
            onClose();
        }
    };

    return (
        <div
            ref={menuRef}
            className="fixed bg-white border border-slate-200 rounded-lg shadow-xl z-[9999] w-48 py-1 overflow-hidden"
            style={{ top: position.y, left: position.x }}
            onContextMenu={(e) => {
                e.preventDefault();
                e.stopPropagation();
            }}
        >
            <button
                onClick={(e) => { e.stopPropagation(); handleAction('view_memory'); }}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <Brain className="w-4 h-4 text-slate-400" />
                {t('view_memory')}
            </button>
            <button
                onClick={(e) => { e.stopPropagation(); handleAction('view_results_directory'); }}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <FolderOpen className="w-4 h-4 text-slate-400" />
                {t('view_results_directory')}
            </button>
            <div className="h-px bg-slate-100 my-1" />
            <button
                onClick={(e) => { e.stopPropagation(); handleAction('toggle_intent_understanding'); }}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <div className={`w-4 h-4 border rounded flex items-center justify-center ${autoIntentUnderstanding ? 'bg-blue-500 border-blue-500' : 'border-slate-300'}`}>
                    {autoIntentUnderstanding && <Check className="w-3 h-3 text-white" />}
                </div>
                {t('auto_intent_understanding')}
            </button>
            <div className="h-px bg-slate-100 my-1" />
            <button
                onClick={(e) => { e.stopPropagation(); handleAction('export'); }}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <Download className="w-4 h-4 text-slate-400" />
                Export
            </button>
        </div>
    );
};

export default ChatThreadContextMenu;
