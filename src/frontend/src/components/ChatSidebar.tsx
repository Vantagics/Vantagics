import React, { useState, useEffect, useRef } from 'react';
import { X, MessageSquare, Plus, Trash2, Send, Loader2, ChevronLeft, ChevronRight } from 'lucide-react';
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
            // Create a new thread if none exists
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
        // Update title if it's the first message
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
            {/* Backdrop */}
            <div 
                className={`fixed inset-0 bg-slate-900/20 backdrop-blur-[2px] transition-opacity duration-300 z-40 ${isOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'}`}
                onClick={onClose}
            />
            
            {/* Sidebar Container */}
            <div 
                data-testid="chat-sidebar"
                className={`fixed inset-y-0 right-0 w-[600px] bg-white border-l border-slate-200 shadow-2xl transform transition-transform duration-300 ease-in-out z-50 flex overflow-hidden ${isOpen ? 'translate-x-0' : 'translate-x-full'}`}
            >
                {/* Thread List Sidebar */}
                <div className={`${isSidebarCollapsed ? 'w-0' : 'w-48'} bg-slate-50 border-r border-slate-200 flex flex-col transition-all duration-300 overflow-hidden relative`}>
                    <div className="p-3 border-b border-slate-200 flex items-center justify-between bg-white/50 backdrop-blur-sm sticky top-0">
                        <span className="font-bold text-slate-800 text-xs uppercase tracking-wider">Chats</span>
                        <button 
                            onClick={handleCreateThread}
                            className="p-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors shadow-sm"
                            title="New Chat"
                        >
                            <Plus className="w-3.5 h-3.5" />
                        </button>
                    </div>
                    
                    <div className="flex-1 overflow-y-auto p-2 space-y-1">
                        {threads.map(thread => (
                            <div 
                                key={thread.id}
                                onClick={() => setActiveThreadId(thread.id)}
                                className={`group flex items-center justify-between p-2 rounded-lg cursor-pointer text-xs transition-all ${
                                    activeThreadId === thread.id 
                                        ? 'bg-blue-100 text-blue-700 font-medium' 
                                        : 'text-slate-600 hover:bg-slate-100'
                                }`}
                            >
                                <div className="flex items-center gap-2 truncate pr-1">
                                    <MessageSquare className={`w-3.5 h-3.5 flex-shrink-0 ${activeThreadId === thread.id ? 'text-blue-500' : 'text-slate-400'}`} />
                                    <span className="truncate">{thread.title}</span>
                                </div>
                                <button 
                                    onClick={(e) => handleDeleteThread(thread.id, e)}
                                    className="opacity-0 group-hover:opacity-100 p-1 hover:text-red-500 transition-all rounded-md hover:bg-red-50"
                                >
                                    <Trash2 className="w-3 h-3" />
                                </button>
                            </div>
                        ))}
                        {threads.length === 0 && (
                            <div className="text-center py-8 px-2">
                                <p className="text-[10px] text-slate-400 italic">No chat history</p>
                            </div>
                        )}
                    </div>

                    <div className="p-2 border-t border-slate-200 bg-white/50">
                        <button 
                            onClick={handleClearHistory}
                            className="w-full flex items-center justify-center gap-2 py-2 text-[10px] font-medium text-slate-400 hover:text-red-500 transition-colors rounded-md"
                        >
                            <Trash2 className="w-3 h-3" />
                            Clear All
                        </button>
                    </div>
                </div>

                {/* Main Chat Area */}
                <div className="flex-1 flex flex-col min-w-0 bg-white relative">
                    {/* Sidebar Toggle Button */}
                    <button 
                        onClick={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
                        className={`absolute left-0 top-1/2 -translate-y-1/2 -translate-x-1/2 z-50 bg-white border border-slate-200 rounded-full p-1 shadow-md hover:bg-slate-50 text-slate-400 hover:text-slate-600 transition-all ${isSidebarCollapsed ? 'translate-x-2' : ''}`}
                    >
                        {isSidebarCollapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
                    </button>

                    <div className="h-14 flex items-center justify-between px-6 border-b border-slate-100 bg-white/80 backdrop-blur-sm z-10">
                        <div className="flex items-center gap-3 text-slate-700">
                            <div className="bg-blue-50 p-1.5 rounded-lg">
                                <MessageSquare className="w-5 h-5 text-blue-600" />
                            </div>
                            <div>
                                <h3 className="font-semibold text-sm leading-none">AI Assistant</h3>
                                <p className="text-[10px] text-slate-400 mt-1">{activeThread?.title || 'Data Analysis'}</p>
                            </div>
                        </div>
                        <button 
                            onClick={onClose}
                            className="p-2 hover:bg-slate-100 rounded-full text-slate-400 hover:text-slate-600 transition-colors"
                        >
                            <X className="w-5 h-5" />
                        </button>
                    </div>

                    <div className="flex-1 overflow-y-auto p-6 space-y-6 bg-slate-50/20">
                        {activeThread?.messages.map((msg, index) => (
                            <MessageBubble 
                                key={msg.id || index} 
                                role={msg.role as 'user' | 'assistant'} 
                                content={msg.content} 
                            />
                        ))}
                        {isLoading && (
                            <div className="flex justify-start">
                                <div className="bg-white border border-slate-200 rounded-2xl px-4 py-3 shadow-sm rounded-bl-none">
                                    <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                                </div>
                            </div>
                        )}
                        {!activeThread && !isLoading && (
                            <div className="h-full flex flex-col items-center justify-center text-center px-4">
                                <div className="bg-blue-50 p-4 rounded-2xl mb-4">
                                    <MessageSquare className="w-8 h-8 text-blue-500" />
                                </div>
                                <h4 className="text-slate-700 font-semibold mb-2">How can I help you today?</h4>
                                <p className="text-sm text-slate-400 max-w-[250px]">
                                    Ask me about your data, sales performance, or any business insights.
                                </p>
                            </div>
                        )}
                        <div ref={messagesEndRef} />
                    </div>

                    <div className="p-6 border-t border-slate-100 bg-white">
                        <div className="relative group">
                            <input 
                                type="text"
                                value={input}
                                onChange={(e) => setInput(e.target.value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()}
                                placeholder="Ask a question about your data..."
                                className="w-full bg-slate-50 border border-slate-200 rounded-2xl pl-5 pr-14 py-4 text-sm focus:ring-2 focus:ring-blue-500 focus:bg-white focus:border-transparent transition-all outline-none shadow-sm group-hover:border-slate-300"
                                disabled={isLoading}
                            />
                            <button 
                                onClick={handleSendMessage}
                                disabled={isLoading || !input.trim()}
                                className="absolute right-2 top-1/2 -translate-y-1/2 p-2.5 bg-blue-600 text-white hover:bg-blue-700 rounded-xl disabled:bg-slate-200 disabled:text-slate-400 transition-all shadow-md hover:shadow-lg active:scale-95"
                            >
                                <Send className="w-5 h-5" />
                            </button>
                        </div>
                        <p className="text-center text-[10px] text-slate-400 mt-3">
                            RapidBI AI Assistant can provide data-driven insights but always verify critical information.
                        </p>
                    </div>
                </div>
            </div>
        </>
    );
};

export default ChatSidebar;