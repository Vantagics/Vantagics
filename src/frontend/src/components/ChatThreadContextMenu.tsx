import React, { useEffect, useRef } from 'react';
import { Brain, FolderOpen, Check, Eraser, FileText, PackageOpen, Pencil } from 'lucide-react';
import { useLanguage } from '../i18n';

interface ChatThreadContextMenuProps {
    position: { x: number; y: number };
    threadId: string;
    onClose: () => void;
    onAction: (action: 'view_memory' | 'view_results_directory' | 'toggle_intent_understanding' | 'clear_messages' | 'comprehensive_report' | 'export_quick_analysis_pack' | 'rename', threadId: string) => void;
    autoIntentUnderstanding?: boolean;
    isFreeChatThread?: boolean;
    isReplaySession?: boolean;
    isGeneratingComprehensiveReport?: boolean;
}

const ChatThreadContextMenu: React.FC<ChatThreadContextMenuProps> = ({ position, threadId, onClose, onAction, autoIntentUnderstanding = true, isFreeChatThread = false, isReplaySession = false, isGeneratingComprehensiveReport = false }) => {
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

    const handleAction = (action: 'view_memory' | 'view_results_directory' | 'toggle_intent_understanding' | 'clear_messages' | 'comprehensive_report' | 'export_quick_analysis_pack' | 'rename') => {
        onAction(action, threadId);
        // Don't close menu for toggle action so user can see the state change
        if (action !== 'toggle_intent_understanding') {
            onClose();
        }
    };

    return (
        <div
            ref={menuRef}
            className="fixed bg-white dark:bg-[#252526] border border-slate-200 dark:border-[#3c3c3c] rounded-lg shadow-xl z-[9999] w-48 py-1 overflow-hidden"
            style={{ top: position.y, left: position.x }}
            onContextMenu={(e) => {
                e.preventDefault();
                e.stopPropagation();
            }}
        >
            {isFreeChatThread ? (
                <button
                    onClick={(e) => { e.stopPropagation(); handleAction('clear_messages'); }}
                    className="w-full text-left px-4 py-2 text-sm text-red-600 dark:text-[#f14c4c] hover:bg-red-50 dark:hover:bg-[#2e1e1e] flex items-center gap-2"
                >
                    <Eraser className="w-4 h-4 text-red-400 dark:text-[#f14c4c]" />
                    {t('clear_chat_history') || '清除历史会话'}
                </button>
            ) : (
                <>
                    <button
                        onClick={(e) => { e.stopPropagation(); handleAction('rename'); }}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                    >
                        <Pencil className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                        {t('context_menu_rename')}
                    </button>
                    {!isReplaySession && (
                        <>
                            <div className="h-px bg-slate-100 dark:bg-[#3c3c3c] my-1" />
                            <button
                                onClick={(e) => { e.stopPropagation(); handleAction('view_memory'); }}
                                className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                            >
                                <Brain className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                                {t('view_memory')}
                            </button>
                            <button
                                onClick={(e) => { e.stopPropagation(); handleAction('view_results_directory'); }}
                                className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                            >
                                <FolderOpen className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                                {t('view_results_directory')}
                            </button>
                            <div className="h-px bg-slate-100 dark:bg-[#3c3c3c] my-1" />
                            <button
                                onClick={(e) => { e.stopPropagation(); handleAction('toggle_intent_understanding'); }}
                                className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                            >
                                <div className={`w-4 h-4 border rounded flex items-center justify-center ${autoIntentUnderstanding ? 'bg-blue-500 border-blue-500' : 'border-slate-300'}`}>
                                    {autoIntentUnderstanding && <Check className="w-3 h-3 text-white" />}
                                </div>
                                {t('auto_intent_understanding')}
                            </button>
                            <div className="h-px bg-slate-100 dark:bg-[#3c3c3c] my-1" />
                            <button
                                onClick={(e) => { e.stopPropagation(); handleAction('export_quick_analysis_pack'); }}
                                className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                            >
                                <PackageOpen className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                                {t('export_quick_analysis_pack')}
                            </button>
                        </>
                    )}
                    <div className="h-px bg-slate-100 dark:bg-[#3c3c3c] my-1" />
                    <button
                        onClick={(e) => { e.stopPropagation(); if (!isGeneratingComprehensiveReport) handleAction('comprehensive_report'); }}
                        className={`w-full text-left px-4 py-2 text-sm flex items-center gap-2 ${
                            isGeneratingComprehensiveReport 
                                ? 'text-slate-300 dark:text-[#4d4d4d] cursor-not-allowed' 
                                : 'text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30]'
                        }`}
                        title={isGeneratingComprehensiveReport 
                            ? (t('comprehensive_report_generating') || '正在生成综合报告...') 
                            : t('comprehensive_report_button_title')}
                        disabled={isGeneratingComprehensiveReport}
                    >
                        <FileText className={`w-4 h-4 ${isGeneratingComprehensiveReport ? 'text-slate-300' : 'text-blue-500'}`} />
                        {t('comprehensive_report')}
                        {isGeneratingComprehensiveReport && (
                            <span className="ml-auto text-[10px] text-indigo-400 animate-pulse">...</span>
                        )}
                    </button>
                </>
            )}
        </div>
    );
};

export default ChatThreadContextMenu;
