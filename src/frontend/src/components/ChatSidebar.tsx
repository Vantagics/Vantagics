import React, { useState, useEffect, useRef } from 'react';
import { X, MessageSquare, Plus, Trash2, Send, Loader2, ChevronLeft, ChevronRight, Settings } from 'lucide-react';
import { GetChatHistory, SaveChatHistory, SendMessage, DeleteThread, ClearHistory } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';
import MessageBubble from './MessageBubble';

interface ChatSidebarProps {
    isOpen: boolean;
    onClose: () => void;
}

const ChatSidebar: React.FC<ChatSidebarProps> = ({ isOpen, onClose }) => {
    const [threads, setThreads] = useState<main.ChatThread[]>([]);
    const [activeThreadId, setActiveThreadId] = useState<string | null>(null);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
    const messagesEndRef = useRef<HTMLDivElement>(null);

    const activeThread = threads.find(t => t.id === activeThreadId);

    useEffect(() => {
        if (isOpen) {
            loadThreads();
        }
    }, [isOpen]);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [activeThread?.messages]);

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

    const handleCreateThread = () => {
        const newThread: main.ChatThread = {
            id: Date.now().toString(),
            title: 'New Chat',
            created_at: Math.floor(Date.now() / 1000),
            messages: []
        };
        const updatedThreads = [newThread, ...threads];
        setThreads(updatedThreads);
        setActiveThreadId(newThread.id);
        SaveChatHistory(updatedThreads).catch(console.error);
    };

    const handleDeleteThread = async (id: string, e: React.MouseEvent) => {
        e.stopPropagation();
        try {
            await DeleteThread(id);
            const updatedThreads = threads.filter(t => t.id !== id);
            setThreads(updatedThreads);
            if (activeThreadId === id) {
                setActiveThreadId(updatedThreads.length > 0 ? updatedThreads[0].id : null);
            }
        } catch (err) {
            console.error('Failed to delete thread:', err);
        }
    };

    const handleSendMessage = async () => {
        if (!input.trim() || isLoading) return;

        let currentThreads = [...threads];
        let currentThread = currentThreads.find(t => t.id === activeThreadId);

        if (!currentThread) {
            currentThread = {
                id: Date.now().toString(),
                title: input.slice(0, 30),
                created_at: Math.floor(Date.now() / 1000),
                messages: []
            };
            currentThreads = [currentThread, ...currentThreads];
            setActiveThreadId(currentThread.id);
        }

        const userMsg: main.ChatMessage = {
            id: Date.now().toString(),
            role: 'user',
            content: input,
            timestamp: Math.floor(Date.now() / 1000)
        };

        currentThread.messages.push(userMsg);
        if (currentThread.messages.length === 1) {
            currentThread.title = input.slice(0, 30) + (input.length > 30 ? '...' : '');
        }

        setThreads([...currentThreads]);
        setInput('');
        setIsLoading(true);

        try {
            const response = await SendMessage(input);
            const assistantMsg: main.ChatMessage = {
                id: (Date.now() + 1).toString(),
                role: 'assistant',
                content: response,
                timestamp: Math.floor(Date.now() / 1000)
            };
            currentThread.messages.push(assistantMsg);
            setThreads([...currentThreads]);
            await SaveChatHistory(currentThreads);
        } catch (error) {
            console.error(error);
            currentThread.messages.push({
                id: (Date.now() + 1).toString(),
                role: 'assistant',
                content: 'Sorry, I encountered an error. Please check your connection and API keys.',
                timestamp: Math.floor(Date.now() / 1000)
            });
            setThreads([...currentThreads]);
        } finally {
            setIsLoading(false);
        }
    };

    const handleClearHistory = async () => {
        if (confirm('Are you sure you want to clear all chat history?')) {
            try {
                await ClearHistory();
                setThreads([]);
                setActiveThreadId(null);
            } catch (err) {
                console.error('Failed to clear history:', err);
            }
        }
    };

    return (
        <>
            <div 
                className={`fixed inset-0 bg-slate-900/40 backdrop-blur-[2px] transition-opacity duration-300 z-40 ${isOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'}`}
                onClick={onClose}
            />
            
            <div 
                data-testid="chat-sidebar"
                className={`fixed inset-y-0 right-0 w-[650px] bg-white shadow-2xl transform transition-transform duration-300 ease-in-out z-50 flex overflow-hidden border-l border-slate-200 ${isOpen ? 'translate-x-0' : 'translate-x-full'}`}
            >
                {/* Thread List Sidebar */}
                <div className={`${isSidebarCollapsed ? 'w-0' : 'w-52'} bg-slate-50 border-r border-slate-200 flex flex-col transition-all duration-300 overflow-hidden relative`}>
                    <div className="p-4 border-b border-slate-200 flex items-center justify-between bg-white/50 backdrop-blur-sm sticky top-0 z-10">
                        <span className="font-bold text-slate-900 text-[11px] uppercase tracking-[0.1em]">History</span>
                        <button 
                            onClick={handleCreateThread}
                            className="p-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-all shadow-sm active:scale-95"
                            title="New Chat"
                        >
                            <Plus className="w-4 h-4" />
                        </button>
                    </div>
                    
                    <div className="flex-1 overflow-y-auto p-2 space-y-1.5 scrollbar-hide">
                        {threads.map(thread => (
                            <div 
                                key={thread.id}
                                onClick={() => setActiveThreadId(thread.id)}
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
                            CLEAR HISTORY
                        </button>
                    </div>
                </div>

                {/* Main Chat Area */}
                <div className="flex-1 flex flex-col min-w-0 bg-white relative">
                    <button 
                        onClick={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
                        className={`absolute left-0 top-1/2 -translate-y-1/2 -translate-x-1/2 z-50 bg-white border border-slate-200 rounded-full p-1.5 shadow-lg hover:bg-slate-50 text-slate-400 hover:text-blue-500 transition-all hover:scale-110 ${isSidebarCollapsed ? 'translate-x-3' : ''}`}
                    >
                        {isSidebarCollapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
                    </button>

                    <div className="h-16 flex items-center justify-between px-6 border-b border-slate-100 bg-white/80 backdrop-blur-md z-10">
                        <div className="flex items-center gap-3.5">
                            <div className="bg-gradient-to-br from-blue-500 to-indigo-600 p-2 rounded-xl shadow-md shadow-blue-200">
                                <MessageSquare className="w-5 h-5 text-white" />
                            </div>
                            <div>
                                <h3 className="font-bold text-slate-900 text-sm tracking-tight">AI Assistant</h3>
                                <div className="flex items-center gap-2 mt-1">
                                    <span className="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse" />
                                    <p className="text-[10px] text-slate-500 font-medium truncate max-w-[200px]">{activeThread?.title || 'Ready to help'}</p>
                                </div>
                            </div>
                        </div>
                        <div className="flex items-center gap-1">
                             <button 
                                onClick={onClose}
                                aria-label="Close sidebar"
                                className="p-2 hover:bg-slate-100 rounded-full text-slate-400 hover:text-slate-600 transition-all"
                            >
                                <X className="w-5 h-5" />
                            </button>
                        </div>
                    </div>

                    <div className="flex-1 overflow-y-auto p-6 space-y-8 bg-slate-50/10 scrollbar-thin scrollbar-thumb-slate-200 scrollbar-track-transparent">
                        {activeThread?.messages.map((msg, index) => (
                            <MessageBubble 
                                key={msg.id || index} 
                                role={msg.role as 'user' | 'assistant'} 
                                content={msg.content} 
                            />
                        ))}
                        {isLoading && (
                            <div className="flex justify-start animate-in fade-in slide-in-from-bottom-2 duration-300">
                                <div className="bg-white border border-slate-200 rounded-2xl px-5 py-3.5 shadow-sm rounded-bl-none flex items-center gap-2">
                                    <Loader2 className="w-4 h-4 animate-spin text-blue-600" />
                                    <span className="text-xs text-slate-500 font-medium">AI is thinking...</span>
                                </div>
                            </div>
                        )}
                        {!activeThread && !isLoading && (
                            <div className="h-full flex flex-col items-center justify-center text-center px-8 animate-in fade-in zoom-in-95 duration-500">
                                <div className="bg-gradient-to-br from-blue-50 to-indigo-50 p-6 rounded-[2.5rem] mb-6 shadow-inner ring-1 ring-white">
                                    <MessageSquare className="w-10 h-10 text-blue-500" />
                                </div>
                                <h4 className="text-slate-900 font-extrabold text-xl tracking-tight mb-3">Insights at your fingertips</h4>
                                <p className="text-sm text-slate-500 max-w-[280px] leading-relaxed font-medium">
                                    Ask about sales trends, customer behavior, or request a complex data analysis.
                                </p>
                                <button 
                                    onClick={handleCreateThread}
                                    className="mt-8 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-sm font-bold shadow-lg shadow-blue-200 transition-all active:scale-95 flex items-center gap-2"
                                >
                                    <Plus className="w-4 h-4" />
                                    Start New Analysis
                                </button>
                            </div>
                        )}
                        <div ref={messagesEndRef} />
                    </div>

                    <div className="p-6 border-t border-slate-100 bg-white">
                        <div className="relative group max-w-2xl mx-auto w-full">
                            <input 
                                type="text"
                                value={input}
                                onChange={(e) => setInput(e.target.value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()}
                                placeholder="What would you like to analyze?"
                                className="w-full bg-slate-50 border border-slate-200 rounded-2xl pl-6 pr-16 py-4.5 text-sm font-normal text-slate-900 focus:ring-4 focus:ring-blue-100 focus:bg-white focus:border-blue-300 transition-all outline-none shadow-sm group-hover:border-slate-300"
                                disabled={isLoading}
                            />
                            <button 
                                onClick={handleSendMessage}
                                disabled={isLoading || !input.trim()}
                                className="absolute right-2.5 top-1/2 -translate-y-1/2 p-2.5 bg-blue-600 text-white hover:bg-blue-700 rounded-xl disabled:bg-slate-200 disabled:text-slate-400 transition-all shadow-md active:scale-95"
                            >
                                <Send className="w-5 h-5" />
                            </button>
                        </div>
                        <div className="flex items-center justify-center gap-4 mt-4">
                            <p className="text-[10px] text-slate-400 font-medium flex items-center gap-1">
                                <span className="w-1 h-1 bg-slate-300 rounded-full" />
                                Data-driven reasoning
                            </p>
                            <p className="text-[10px] text-slate-400 font-medium flex items-center gap-1">
                                <span className="w-1 h-1 bg-slate-300 rounded-full" />
                                Visualized summaries
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
};

export default ChatSidebar;
