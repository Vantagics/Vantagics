import React, { useState, useEffect, useRef } from 'react';
import { X, MessageSquare, Plus, Trash2, Send, Loader2, ChevronLeft, ChevronRight, Settings, Upload } from 'lucide-react';
import { GetChatHistory, SaveChatHistory, SendMessage, DeleteThread, ClearHistory, GetDataSources, CreateChatThread, UpdateThreadTitle, ExportSessionHTML, AssetizeSession } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';
import MessageBubble from './MessageBubble';
import { useLanguage } from '../i18n';
import DeleteConfirmationModal from './DeleteConfirmationModal';
import ChatThreadContextMenu from './ChatThreadContextMenu';
import MemoryViewModal from './MemoryViewModal';

interface ChatSidebarProps {
    isOpen: boolean;
    onClose: () => void;
}

const ChatSidebar: React.FC<ChatSidebarProps> = ({ isOpen, onClose }) => {
    const { t } = useLanguage();
    const [threads, setThreads] = useState<main.ChatThread[]>([]);
    const [activeThreadId, setActiveThreadId] = useState<string | null>(null);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
    const [showClearConfirm, setShowClearConfirm] = useState(false);
    const [dataSources, setDataSources] = useState<any[]>([]);
    const [deleteThreadTarget, setDeleteThreadTarget] = useState<{ id: string, title: string } | null>(null);
    const [memoryModalTarget, setMemoryModalTarget] = useState<string | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number, y: number, threadId: string } | null>(null);
    const [blankAreaContextMenu, setBlankAreaContextMenu] = useState<{ x: number, y: number } | null>(null);
    
    // Resizing State
    const [sidebarWidth, setSidebarWidth] = useState(650);
    const [historyWidth, setHistoryWidth] = useState(208);
    const [isResizingSidebar, setIsResizingSidebar] = useState(false);
    const [isResizingHistory, setIsResizingHistory] = useState(false);

    const messagesEndRef = useRef<HTMLDivElement>(null);
    const blankMenuRef = useRef<HTMLDivElement>(null);

    const activeThread = threads.find(t => t.id === activeThreadId);
    // Find associated data source name
    const activeDataSource = dataSources.find(ds => ds.id === activeThread?.data_source_id);

    useEffect(() => {
        if (isOpen) {
            loadThreads();
            loadDataSources();
        }
    }, [isOpen]);

    // Resizing Effect
    useEffect(() => {
        const handleMouseMove = (e: MouseEvent) => {
            if (isResizingSidebar) {
                const newWidth = window.innerWidth - e.clientX;
                if (newWidth > 400 && newWidth < window.innerWidth - 100) {
                    setSidebarWidth(newWidth);
                }
            } else if (isResizingHistory) {
                const sidebarLeft = window.innerWidth - sidebarWidth;
                const newWidth = e.clientX - sidebarLeft;
                if (newWidth > 150 && newWidth < sidebarWidth - 300) {
                    setHistoryWidth(newWidth);
                }
            }
        };

        const handleMouseUp = () => {
            setIsResizingSidebar(false);
            setIsResizingHistory(false);
            document.body.style.cursor = 'default';
        };

        if (isResizingSidebar || isResizingHistory) {
            window.addEventListener('mousemove', handleMouseMove);
            window.addEventListener('mouseup', handleMouseUp);
        }

        return () => {
            window.removeEventListener('mousemove', handleMouseMove);
            window.removeEventListener('mouseup', handleMouseUp);
        };
    }, [isResizingSidebar, isResizingHistory, sidebarWidth]);

    useEffect(() => {
        // Listen for new chat creation from Sidebar
        const unsubscribeStart = EventsOn('start-new-chat', (data: any) => {
            handleCreateThread(data.dataSourceId, data.sessionName);
        });

        // Listen for open chat request from Sidebar context menu
        const unsubscribeOpen = EventsOn('open-chat', (thread: main.ChatThread) => {
            // Ensure we have the latest threads
            loadThreads().then(() => {
                setActiveThreadId(thread.id);
            });
        });

        // Listen for thread updates (e.g. background analysis errors)
        const unsubscribeUpdate = EventsOn('thread-updated', (threadId: string) => {
            loadThreads();
        });

        // Listen for loading state from backend (for async tasks like suggestions)
        const unsubscribeLoading = EventsOn('chat-loading', (loading: boolean) => {
            setIsLoading(loading);
        });

        // Listen for insight analysis request from Dashboard
        const unsubscribeAnalyze = EventsOn('analyze-insight', (text: string) => {
            // Open sidebar if closed (managed by parent, but we can't control parent state easily here without prop)
            // But we can just send message. The parent App handles opening via prop? 
            // Wait, isOpen is a prop. We cannot change it here.
            // But we can emit an event to App?
            // Actually, clicking insight usually implies user wants to chat.
            // We can assume App will listen to this too or we emit "open-chat-sidebar".
            
            // For now, let's just trigger send message.
            handleSendMessage(text);
        });

        return () => {
            if (unsubscribeStart) unsubscribeStart();
            if (unsubscribeOpen) unsubscribeOpen();
            if (unsubscribeUpdate) unsubscribeUpdate();
            if (unsubscribeLoading) unsubscribeLoading();
            if (unsubscribeAnalyze) unsubscribeAnalyze();
        };
    }, [threads]);

    useEffect(() => {
        const scrollToBottom = () => {
            messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
        };

        // Use a small timeout to ensure DOM has updated
        const timeoutId = setTimeout(scrollToBottom, 100);
        return () => clearTimeout(timeoutId);
    }, [activeThread?.messages, isLoading]);

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (blankMenuRef.current && !blankMenuRef.current.contains(event.target as Node)) {
                setBlankAreaContextMenu(null);
            }
        };
        if (blankAreaContextMenu) {
            document.addEventListener('mousedown', handleClickOutside);
            return () => document.removeEventListener('mousedown', handleClickOutside);
        }
    }, [blankAreaContextMenu]);

    const loadThreads = async () => {
        try {
            const history = await GetChatHistory();
            setThreads(history);
            if (history.length > 0 && !activeThreadId) {
                setActiveThreadId(history[0].id);
            }
        } catch (err) {
            console.error('Failed to load chat history:', err);
        }
    };

    const loadDataSources = async () => {
        try {
            const ds = await GetDataSources();
            setDataSources(ds || []);
        } catch (err) {
            console.error('Failed to load data sources:', err);
        }
    };

    const handleCreateThread = async (dataSourceId?: string, title?: string) => {
        try {
            const newThread = await CreateChatThread(dataSourceId || '', title || 'New Chat');
            setThreads(prev => [newThread, ...prev]);
            setActiveThreadId(newThread.id);
        } catch (err) {
            console.error('Failed to create thread:', err);
        }
    };

    const handleDeleteThread = (id: string, e: React.MouseEvent) => {
        e.stopPropagation();
        const thread = threads.find(t => t.id === id);
        if (thread) {
            setDeleteThreadTarget({ id: thread.id, title: thread.title });
        }
    };

    const confirmDeleteThread = async () => {
        if (!deleteThreadTarget) return;
        try {
            await DeleteThread(deleteThreadTarget.id);
            const updatedThreads = threads.filter(t => t.id !== deleteThreadTarget.id);
            setThreads(updatedThreads);
            if (activeThreadId === deleteThreadTarget.id) {
                setActiveThreadId(updatedThreads.length > 0 ? updatedThreads[0].id : null);
            }
            setDeleteThreadTarget(null);
        } catch (err) {
            console.error('Failed to delete thread:', err);
        }
    };

    const handleContextMenu = (e: React.MouseEvent, threadId: string) => {
        e.preventDefault();
        setContextMenu({ x: e.clientX, y: e.clientY, threadId });
    };

    const handleBlankAreaContextMenu = (e: React.MouseEvent) => {
        // Only show menu if clicking directly on the container (blank area)
        if (e.target === e.currentTarget) {
            e.preventDefault();
            setBlankAreaContextMenu({ x: e.clientX, y: e.clientY });
        }
    };

    const handleImportAction = () => {
        console.log('Import action triggered');
        alert('Import functionality (Not implemented)');
        setBlankAreaContextMenu(null);
    };

    const handleContextAction = async (action: 'export' | 'assetize' | 'view_memory', threadId: string) => {
        console.log(`Action ${action} on thread ${threadId}`);
        if (action === 'view_memory') {
            setMemoryModalTarget(threadId);
        } else if (action === 'export') {
            try {
                await ExportSessionHTML(threadId);
            } catch (e) {
                console.error("Export failed:", e);
            }
        } else if (action === 'assetize') {
            try {
                await AssetizeSession(threadId);
            } catch (e) {
                console.error("Assetize failed:", e);
            }
        }
    };

    const handleSendMessage = async (text?: string) => {
        const msgText = text || input;
        if (!msgText.trim() || isLoading) return;

        let currentThreads = [...threads];
        let currentThread = currentThreads.find(t => t.id === activeThreadId);

        // If no active thread, create one first
        if (!currentThread) {
            try {
                const title = msgText.slice(0, 30);
                const newThread = await CreateChatThread('', title);
                currentThread = newThread;
                currentThreads = [newThread, ...currentThreads];
                setActiveThreadId(newThread.id);
                // Update local threads immediately to show UI
                setThreads(currentThreads);
            } catch (err) {
                console.error("Failed to create thread on send:", err);
                return;
            }
        }

        const userMsg = new main.ChatMessage();
        userMsg.id = Date.now().toString();
        userMsg.role = 'user';
        userMsg.content = msgText;
        userMsg.timestamp = Math.floor(Date.now() / 1000);

        if (!currentThread.messages) currentThread.messages = [];
        currentThread.messages.push(userMsg);
        
        if (currentThread.messages.length === 1 && currentThread.title === 'New Chat') {
             const newTitle = msgText.slice(0, 30) + (msgText.length > 30 ? '...' : '');
             try {
                 const uniqueTitle = await UpdateThreadTitle(currentThread.id, newTitle);
                 currentThread.title = uniqueTitle;
             } catch (err) {
                 console.error("Failed to rename thread:", err);
             }
        }

        setThreads([...currentThreads]);
        setInput('');
        setIsLoading(true);

        try {
            const response = await SendMessage(currentThread?.id || '', msgText);
            const assistantMsg = new main.ChatMessage();
            assistantMsg.id = (Date.now() + 1).toString();
            assistantMsg.role = 'assistant';
            assistantMsg.content = response;
            assistantMsg.timestamp = Math.floor(Date.now() / 1000);

            currentThread.messages.push(assistantMsg);
            setThreads([...currentThreads]);
            await SaveChatHistory(currentThreads);
        } catch (error) {
            console.error(error);
            const errorMsg = new main.ChatMessage();
            errorMsg.id = (Date.now() + 1).toString();
            errorMsg.role = 'assistant';
            errorMsg.content = 'Sorry, I encountered an error. Please check your connection and API keys.';
            errorMsg.timestamp = Math.floor(Date.now() / 1000);
            currentThread.messages.push(errorMsg);
            setThreads([...currentThreads]);
        } finally {
            setIsLoading(false);
        }
    };

    const handleClearHistory = () => {
        setShowClearConfirm(true);
    };

    const confirmClearHistory = async () => {
        try {
            await ClearHistory();
            setThreads([]);
            setActiveThreadId(null);
            setShowClearConfirm(false);
        } catch (err) {
            console.error('Failed to clear history:', err);
            setShowClearConfirm(false);
        }
    };

    const cancelClearHistory = () => {
        setShowClearConfirm(false);
    };

    return (
        <>
            <div 
                className={`fixed inset-0 bg-slate-900/40 backdrop-blur-[2px] transition-opacity duration-300 z-40 ${isOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'}`}
                onClick={onClose}
            />
            
            <div 
                data-testid="chat-sidebar"
                style={{ width: sidebarWidth }}
                className={`fixed inset-y-0 right-0 bg-white shadow-2xl transform transition-transform duration-300 ease-in-out z-50 flex overflow-hidden border-l border-slate-200 ${isOpen ? 'translate-x-0' : 'translate-x-full'}`}
            >
                {/* Sidebar Resizer (Left Edge) */}
                <div 
                    className="absolute left-0 top-0 bottom-0 w-1 hover:bg-blue-400 cursor-col-resize z-[60] transition-colors"
                    onMouseDown={() => { setIsResizingSidebar(true); document.body.style.cursor = 'col-resize'; }}
                />

                {/* Thread List Sidebar */}
                <div 
                    style={{ width: isSidebarCollapsed ? 0 : historyWidth }}
                    className="bg-slate-50 border-r border-slate-200 flex flex-col transition-all duration-300 overflow-hidden relative flex-shrink-0"
                >
                    <div className="p-4 border-b border-slate-200 flex items-center justify-between bg-white/50 backdrop-blur-sm sticky top-0 z-10"
                        style={{ '--wails-draggable': 'drag' } as any}
                    >
                        <span className="font-bold text-slate-900 text-[11px] uppercase tracking-[0.1em]">{t('history')}</span>
                        <div className="w-4" /> 
                    </div>
                    
                    <div
                        className="flex-1 overflow-y-auto p-2 space-y-1.5 scrollbar-hide"
                        onContextMenu={handleBlankAreaContextMenu}
                    >
                        {threads.map(thread => (
                            <div 
                                key={thread.id}
                                onClick={() => setActiveThreadId(thread.id)}
                                onContextMenu={(e) => handleContextMenu(e, thread.id)}
                                className={`group flex items-center justify-between p-2.5 rounded-xl cursor-pointer text-xs transition-all border ${
                                    activeThreadId === thread.id 
                                        ? 'bg-white border-blue-200 text-blue-700 shadow-sm ring-1 ring-blue-100' 
                                        : 'text-slate-600 hover:bg-white hover:border-slate-200 border-transparent'
                                }`}
                            >
                                <div className="flex items-center gap-2.5 truncate pr-1">
                                    <MessageSquare className={`w-4 h-4 flex-shrink-0 ${activeThreadId === thread.id ? 'text-blue-500' : 'text-slate-400'}`} />
                                    <span className="truncate leading-tight">{thread.title}</span>
                                </div>
                                <button 
                                    onClick={(e) => handleDeleteThread(thread.id, e)}
                                    className="opacity-0 group-hover:opacity-100 p-1.5 hover:text-red-500 transition-all rounded-lg hover:bg-red-50 text-slate-400"
                                >
                                    <Trash2 className="w-3.5 h-3.5" />
                                </button>
                            </div>
                        ))}
                        {threads.length === 0 && (
                            <div className="text-center py-12 px-4">
                                <div className="w-10 h-10 bg-slate-100 rounded-full flex items-center justify-center mx-auto mb-3">
                                    <MessageSquare className="w-5 h-5 text-slate-300" />
                                </div>
                                <p className="text-[10px] text-slate-400 font-medium">No threads yet</p>
                            </div>
                        )}
                    </div>

                    <div className="p-3 border-t border-slate-200 bg-white/50">
                        <button 
                            onClick={handleClearHistory}
                            className="w-full flex items-center justify-center gap-2 py-2.5 text-[10px] font-bold text-slate-500 hover:text-red-600 transition-colors rounded-xl hover:bg-red-50"
                        >
                            <Trash2 className="w-3.5 h-3.5" />
                            {t('clear_history')}
                        </button>
                    </div>

                    {/* History Resizer (Right Edge of History Panel) */}
                    <div 
                        className="absolute right-0 top-0 bottom-0 w-1 hover:bg-blue-400 cursor-col-resize z-20 transition-colors"
                        onMouseDown={(e) => { e.preventDefault(); setIsResizingHistory(true); document.body.style.cursor = 'col-resize'; }}
                    />
                </div>

                {/* Main Chat Area */}
                <div className="flex-1 flex flex-col min-w-0 bg-white relative">
                    <button 
                        onClick={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
                        className={`absolute left-0 top-1/2 -translate-y-1/2 -translate-x-1/2 z-50 bg-white border border-slate-200 rounded-full p-1.5 shadow-lg hover:bg-slate-50 text-slate-400 hover:text-blue-500 transition-all hover:scale-110 ${isSidebarCollapsed ? 'translate-x-3' : ''}`}
                    >
                        {isSidebarCollapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
                    </button>

                    <div className="h-16 flex items-center justify-between px-6 border-b border-slate-100 bg-white/80 backdrop-blur-md z-10 relative"
                        style={{ '--wails-draggable': 'drag' } as any}
                    >
                        <div className="flex items-center gap-3.5">
                            <div className="bg-gradient-to-br from-blue-500 to-indigo-600 p-2 rounded-xl shadow-md shadow-blue-200">
                                <MessageSquare className="w-5 h-5 text-white" />
                            </div>
                            <div>
                                <h3 className="font-bold text-slate-900 text-sm tracking-tight">{t('ai_assistant')}</h3>
                                <div className="flex items-center gap-2 mt-1">
                                    <span className="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse" />
                                    <p className="text-[10px] text-slate-500 font-medium truncate max-w-[200px]">
                                        {activeThread?.title || t('ready_to_help')}
                                    </p>
                                    {activeDataSource && (
                                        <span className="text-[9px] px-1.5 py-0.5 bg-slate-100 text-slate-500 rounded-full border border-slate-200">
                                            {activeDataSource.name}
                                        </span>
                                    )}
                                </div>
                            </div>
                        </div>
                        <div className="flex items-center gap-1">
                             <button 
                                onClick={onClose}
                                aria-label="Close sidebar"
                                className="p-2 hover:bg-slate-100 rounded-full text-slate-400 hover:text-slate-600 transition-all"
                                style={{ '--wails-draggable': 'no-drag' } as any}
                            >
                                <X className="w-5 h-5" />
                            </button>
                        </div>
                        {isLoading && (
                            <div className="absolute bottom-0 left-0 right-0 h-1 z-20 overflow-hidden">
                                <div className="h-full w-1/3 bg-blue-500 animate-progress-indeterminate rounded-full"></div>
                            </div>
                        )}
                    </div>

                    <div className="flex-1 overflow-y-auto p-6 space-y-8 bg-slate-50/10 scrollbar-thin scrollbar-thumb-slate-200 scrollbar-track-transparent">
                        {activeThread?.messages.map((msg, index) => (
                            <MessageBubble 
                                key={msg.id || index} 
                                role={msg.role as 'user' | 'assistant'} 
                                content={msg.content} 
                                onActionClick={(action) => handleSendMessage(action.value || action.label)}
                            />
                        ))}
                        {isLoading && (
                            <div className="flex justify-start animate-in fade-in slide-in-from-bottom-2 duration-300">
                                <div className="bg-white border border-slate-200 rounded-2xl px-5 py-3.5 shadow-sm rounded-bl-none flex items-center gap-2">
                                    <Loader2 className="w-4 h-4 animate-spin text-blue-600" />
                                    <span className="text-xs text-slate-500 font-medium">{t('ai_thinking')}</span>
                                </div>
                            </div>
                        )}
                        {!activeThread && !isLoading && (
                            <div className="h-full flex flex-col items-center justify-center text-center px-8 animate-in fade-in zoom-in-95 duration-500">
                                <div className="bg-gradient-to-br from-blue-50 to-indigo-50 p-6 rounded-[2.5rem] mb-6 shadow-inner ring-1 ring-white">
                                    <MessageSquare className="w-10 h-10 text-blue-500" />
                                </div>
                                <h4 className="text-slate-900 font-extrabold text-xl tracking-tight mb-3">{t('insights_at_fingertips')}</h4>
                                <p className="text-sm text-slate-500 max-w-[280px] leading-relaxed font-medium">
                                    {t('ask_about_sales')}
                                </p>
                            </div>
                        )}
                        <div ref={messagesEndRef} />
                    </div>

                    <div className="p-6 border-t border-slate-100 bg-white">
                        <div className="flex items-stretch gap-3 max-w-2xl mx-auto w-full">
                            <input 
                                type="text"
                                value={input}
                                onChange={(e) => setInput(e.target.value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()}
                                placeholder={t('what_to_analyze')}
                                className="flex-1 bg-slate-50 border border-slate-200 rounded-2xl px-6 py-1.5 text-sm font-normal text-slate-900 focus:ring-4 focus:ring-blue-100 focus:bg-white focus:border-blue-300 transition-all outline-none shadow-sm hover:border-slate-300"
                            />
                            <button 
                                onClick={() => handleSendMessage()}
                                disabled={isLoading || !input.trim()}
                                className="aspect-square bg-blue-600 text-white hover:bg-blue-700 rounded-2xl disabled:bg-slate-200 disabled:text-slate-400 transition-all shadow-md active:scale-95 flex items-center justify-center"
                            >
                                <Send className="w-5 h-5" />
                            </button>
                        </div>
                        <div className="flex items-center justify-center gap-4 mt-4">
                            <p className="text-[10px] text-slate-400 font-medium flex items-center gap-1">
                                <span className="w-1 h-1 bg-slate-300 rounded-full" />
                                {t('data_driven_reasoning')}
                            </p>
                            <p className="text-[10px] text-slate-400 font-medium flex items-center gap-1">
                                <span className="w-1 h-1 bg-slate-300 rounded-full" />
                                {t('visualized_summaries')}
                            </p>
                        </div>
                    </div>
                </div>
            </div>

            {/* Confirmation Modal */}
            {showClearConfirm && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
                    <div className="bg-white rounded-xl shadow-2xl p-6 w-[320px] transform transition-all animate-in zoom-in-95 duration-200">
                        <h3 className="text-lg font-bold text-slate-900 mb-2">{t('clear_history_confirm_title')}</h3>
                        <p className="text-sm text-slate-500 mb-6">
                            {t('clear_history_confirm_desc')}
                        </p>
                        <div className="flex justify-end gap-3">
                            <button
                                onClick={cancelClearHistory}
                                className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-lg transition-colors"
                            >
                                {t('cancel')}
                            </button>
                            <button
                                onClick={confirmClearHistory}
                                className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-lg shadow-sm transition-colors"
                            >
                                {t('clear')}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            <DeleteConfirmationModal
                isOpen={!!deleteThreadTarget}
                sourceName={deleteThreadTarget?.title || ''}
                onClose={() => setDeleteThreadTarget(null)}
                onConfirm={confirmDeleteThread}
            />

            <MemoryViewModal
                isOpen={!!memoryModalTarget}
                threadId={memoryModalTarget || ''}
                onClose={() => setMemoryModalTarget(null)}
            />

            {contextMenu && (
                <ChatThreadContextMenu
                    position={{ x: contextMenu.x, y: contextMenu.y }}
                    threadId={contextMenu.threadId}
                    onClose={() => setContextMenu(null)}
                    onAction={handleContextAction}
                />
            )}

            {blankAreaContextMenu && (
                <div
                    ref={blankMenuRef}
                    className="fixed bg-white border border-slate-200 rounded-lg shadow-xl z-[9999] w-40 py-1 overflow-hidden"
                    style={{ top: blankAreaContextMenu.y, left: blankAreaContextMenu.x }}
                    onContextMenu={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                    }}
                >
                    <button
                        onClick={(e) => { e.stopPropagation(); handleImportAction(); }}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
                    >
                        <Upload className="w-4 h-4 text-slate-400" />
                        Import
                    </button>
                </div>
            )}
        </>
    );
};

export default ChatSidebar;