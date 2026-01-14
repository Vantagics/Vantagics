import React, { useEffect, useRef, useState } from 'react';
import { GetChatHistoryByDataSource } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';
import { MessageSquare, Download, Info, Play } from 'lucide-react';
import { useLanguage } from '../i18n';

interface SourceContextMenuProps {
    position: { x: number; y: number };
    sourceId: string;
    onClose: () => void;
    onSelectThread: (thread: main.ChatThread) => void;
    onExport: () => void;
    onProperties: () => void;
    onStartAnalysis: () => void;
}

const SourceContextMenu: React.FC<SourceContextMenuProps> = ({ position, sourceId, onClose, onSelectThread, onExport, onProperties, onStartAnalysis }) => {
    const { t } = useLanguage();
    const menuRef = useRef<HTMLDivElement>(null);
    const [threads, setThreads] = useState<main.ChatThread[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                onClose();
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, [onClose]);

    useEffect(() => {
        setIsLoading(true);
        GetChatHistoryByDataSource(sourceId)
            .then(t => setThreads(t || []))
            .catch(console.error)
            .finally(() => setIsLoading(false));
    }, [sourceId]);

    return (
        <div 
            ref={menuRef}
            className="fixed bg-white border border-slate-200 rounded-lg shadow-xl z-[9999] w-64 py-1 overflow-hidden"
            style={{ top: position.y, left: position.x }}
            onContextMenu={(e) => {
                e.preventDefault();
                e.stopPropagation();
            }}
        >
            <button 
                onClick={() => {
                    onStartAnalysis();
                    onClose();
                }}
                className="w-full text-left px-4 py-2 text-sm text-blue-600 font-medium hover:bg-blue-50 flex items-center gap-2"
            >
                <Play className="w-4 h-4 text-blue-500" />
                {t('start_new_analysis')}
            </button>

            <div className="h-px bg-slate-100 my-1" />

            <button 
                onClick={() => {
                    onProperties();
                    onClose();
                }}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <Info className="w-4 h-4 text-slate-400" />
                Properties
            </button>
            <button 
                onClick={() => {
                    onExport();
                    onClose();
                }}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <Download className="w-4 h-4 text-slate-400" />
                Export Data
            </button>
            
            <div className="h-px bg-slate-100 my-1" />

            <div className="px-3 py-2 border-b border-slate-100 bg-slate-50">
                <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Associated Chats</span>
            </div>
            
            <div className="max-h-64 overflow-y-auto">
                {isLoading ? (
                    <div className="p-4 text-center text-xs text-slate-400">Loading...</div>
                ) : !threads || threads.length === 0 ? (
                    <div className="p-4 text-center text-xs text-slate-400 italic">No associated chats found.</div>
                ) : (
                    threads?.map(thread => (
                        <button 
                            key={thread.id}
                            onClick={() => {
                                onSelectThread(thread);
                                onClose();
                            }}
                            className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-blue-50 hover:text-blue-600 flex items-center gap-2 transition-colors"
                        >
                            <MessageSquare className="w-3 h-3 text-slate-400" />
                            <span className="truncate">{thread.title}</span>
                        </button>
                    ))
                )}
            </div>
        </div>
    );
};

export default SourceContextMenu;
