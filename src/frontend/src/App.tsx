import React, { useState, useEffect, useRef } from 'react';
import Sidebar from './components/Sidebar';
import Dashboard from './components/Dashboard';
import ContextPanel from './components/ContextPanel';
import PreferenceModal from './components/PreferenceModal';
import ChatSidebar from './components/ChatSidebar';
import MessageBubble from './components/MessageBubble';
import ContextMenu from './components/ContextMenu';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { GetDashboardData, SendMessage } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';
import { Send, Loader2 } from 'lucide-react';
import './App.css';

interface Message {
    role: 'user' | 'assistant';
    content: string;
    payload?: string;
}

function App() {
    const [isPreferenceOpen, setIsPreferenceOpen] = useState(false);
    const [isChatOpen, setIsChatOpen] = useState(false);
    const [dashboardData, setDashboardData] = useState<main.DashboardData | null>(null);
    const [messages, setMessages] = useState<Message[]>([
        { role: 'assistant', content: 'Hello! I am your AI Business Intelligence assistant. How can I help you analyze your data today?' }
    ]);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const messagesEndRef = useRef<HTMLDivElement>(null);

    // Context Menu State
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; target: HTMLElement } | null>(null);

    useEffect(() => {
        // Fetch dashboard data
        GetDashboardData().then(setDashboardData).catch(console.error);

        // Listen for menu event
        const unsubscribe = EventsOn("open-settings", () => {
            setIsPreferenceOpen(true);
        });

        // Global Context Menu Listener
        const handleContextMenu = (e: MouseEvent) => {
            const target = e.target as HTMLElement;
            if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
                e.preventDefault();
                setContextMenu({ x: e.clientX, y: e.clientY, target });
            }
        };

        window.addEventListener('contextmenu', handleContextMenu);

        return () => {
            window.removeEventListener('contextmenu', handleContextMenu);
        };
    }, []);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    const handleSendMessage = async () => {
        if (!input.trim() || isLoading) return;

        const userMessage = input;
        setInput('');
        setMessages(prev => [...prev, { role: 'user', content: userMessage }]);
        setIsLoading(true);

        try {
            const response = await SendMessage(userMessage);
            setMessages(prev => [...prev, { role: 'assistant', content: response }]);
        } catch (error) {
            console.error(error);
            setMessages(prev => [...prev, { role: 'assistant', content: 'Sorry, I encountered an error. Please check your connection and API keys.' }]);
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="flex h-screen w-screen bg-slate-50 overflow-hidden font-sans text-slate-900 relative">
            <Sidebar 
                onOpenSettings={() => setIsPreferenceOpen(true)} 
                onToggleChat={() => setIsChatOpen(!isChatOpen)}
            />
            <Dashboard data={dashboardData} />
            <ContextPanel />
            
            <ChatSidebar isOpen={isChatOpen} onClose={() => setIsChatOpen(false)}>
                <div className="flex-1 flex flex-col gap-4">
                    {messages.map((msg, index) => (
                        <MessageBubble key={index} role={msg.role} content={msg.content} payload={msg.payload} />
                    ))}
                    {isLoading && (
                        <div className="flex justify-start mb-2">
                            <div className="bg-white border border-slate-200 rounded-2xl px-4 py-3 shadow-sm rounded-bl-none">
                                <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                            </div>
                        </div>
                    )}
                    <div ref={messagesEndRef} />
                </div>
                <div className="mt-4 pt-4 border-t border-slate-100">
                    <div className="relative">
                        <input 
                            type="text"
                            value={input}
                            onChange={(e) => setInput(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()}
                            placeholder="Type a message..."
                            className="w-full bg-slate-100 border-none rounded-xl pl-4 pr-12 py-3 text-sm focus:ring-2 focus:ring-blue-500 transition-all outline-none"
                            disabled={isLoading}
                        />
                        <button 
                            onClick={handleSendMessage}
                            disabled={isLoading || !input.trim()}
                            aria-label="Send message"
                            className="absolute right-2 top-1/2 -translate-y-1/2 p-2 text-blue-600 hover:bg-blue-50 rounded-lg disabled:text-slate-300 disabled:hover:bg-transparent transition-colors"
                        >
                            <Send className="w-5 h-5" />
                        </button>
                    </div>
                </div>
            </ChatSidebar>

            <PreferenceModal 
                isOpen={isPreferenceOpen} 
                onClose={() => setIsPreferenceOpen(false)} 
            />

            {contextMenu && (
                <ContextMenu 
                    position={{ x: contextMenu.x, y: contextMenu.y }}
                    target={contextMenu.target}
                    onClose={() => setContextMenu(null)}
                />
            )}
        </div>
    );
}

export default App;