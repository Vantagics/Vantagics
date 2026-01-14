import React, { useEffect, useRef } from 'react';
import { Download, Briefcase, Brain, FolderOpen } from 'lucide-react';
import { useLanguage } from '../i18n';

interface ChatThreadContextMenuProps {
    position: { x: number; y: number };
    threadId: string;
    onClose: () => void;
    onAction: (action: 'export' | 'assetize' | 'view_memory' | 'view_results_directory', threadId: string) => void;
}

const ChatThreadContextMenu: React.FC<ChatThreadContextMenuProps> = ({ position, threadId, onClose, onAction }) => {
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

    const handleAction = (action: 'export' | 'assetize' | 'view_memory' | 'view_results_directory') => {
        onAction(action, threadId);
        onClose();
    };

    return (
        <div 
            ref={menuRef}
            className="fixed bg-white border border-slate-200 rounded-lg shadow-xl z-[9999] w-40 py-1 overflow-hidden"
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
                onClick={(e) => { e.stopPropagation(); handleAction('export'); }}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <Download className="w-4 h-4 text-slate-400" />
                Export
            </button>
            <button
                onClick={(e) => { e.stopPropagation(); handleAction('assetize'); }}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <Briefcase className="w-4 h-4 text-slate-400" />
                Assetize
            </button>
        </div>
    );
};

export default ChatThreadContextMenu;
