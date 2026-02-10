import React, { useEffect, useRef } from 'react';
import { Download, Briefcase, Brain, FolderOpen, Check, Eraser, FileText } from 'lucide-react';
import { useLanguage } from '../i18n';

interface ChatThreadContextMenuProps {
    position: { x: number; y: number };
    threadId: string;
    onClose: () => void;
    onAction: (action: 'export' | 'view_memory' | 'view_results_directory' | 'toggle_intent_understanding' | 'clear_messages' | 'comprehensive_report', threadId: string) => void;
    autoIntentUnderstanding?: boolean;
    isFreeChatThread?: boolean;
}

const ChatThreadContextMenu: React.FC<ChatThreadContextMenuProps> = ({ position, threadId, onClose, onAction, autoIntentUnderstanding = true, isFreeChatThread = false }) => {
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

    const handleAction = (action: 'export' | 'view_memory' | 'view_results_directory' | 'toggle_intent_understanding' | 'clear_messages' | 'comprehensive_report') => {
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
            {!isFreeChatThread && (
                <button
                    onClick={(e) => { e.stopPropagation(); handleAction('comprehensive_report'); }}
                    className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
                    title={t('comprehensive_report_button_title')}
                >
                    <FileText className="w-4 h-4 text-blue-500" />
                    {t('comprehensive_report')}
                </button>
            )}
            {isFreeChatThread && (
                <>
                    <div className="h-px bg-slate-100 my-1" />
                    <button
                        onClick={(e) => { e.stopPropagation(); handleAction('clear_messages'); }}
                        className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 flex items-center gap-2"
                    >
                        <Eraser className="w-4 h-4 text-red-400" />
                        {t('clear_chat_history') || '清除历史会话'}
                    </button>
                </>
            )}
        </div>
    );
};

export default ChatThreadContextMenu;
