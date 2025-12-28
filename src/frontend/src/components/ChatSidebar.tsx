import React from 'react';
import { X, MessageSquare } from 'lucide-react';

interface ChatSidebarProps {
    isOpen?: boolean;
    onClose?: () => void;
    children?: React.ReactNode;
}

const ChatSidebar: React.FC<ChatSidebarProps> = ({ isOpen = false, onClose, children }) => {
    return (
        <>
            {/* Backdrop */}
            <div 
                className={`fixed inset-0 bg-slate-900/20 backdrop-blur-[2px] transition-opacity duration-300 z-40 ${isOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'}`}
                onClick={onClose}
            />
            
            {/* Sidebar */}
            <div 
                data-testid="chat-sidebar"
                className={`fixed inset-y-0 right-0 w-[450px] bg-white border-l border-slate-200 shadow-2xl transform transition-transform duration-300 ease-in-out z-50 flex flex-col ${isOpen ? 'translate-x-0' : 'translate-x-full'}`}
            >
            <div className="h-16 flex items-center justify-between px-6 border-b border-slate-100 bg-slate-50/50 backdrop-blur-sm">
                <div className="flex items-center gap-2 text-slate-700">
                    <MessageSquare className="w-5 h-5" />
                    <span className="font-semibold">AI Assistant</span>
                </div>
                <button 
                    onClick={onClose}
                    aria-label="Close sidebar"
                    className="p-2 hover:bg-slate-100 rounded-full text-slate-400 hover:text-slate-600 transition-colors"
                >
                    <X className="w-5 h-5" />
                </button>
            </div>
            <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-4 bg-slate-50/30">
                {children}
            </div>
                    </div>
                </>
            );
        };
        export default ChatSidebar;
