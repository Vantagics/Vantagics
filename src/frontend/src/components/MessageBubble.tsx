import React from 'react';
import ReactMarkdown from 'react-markdown';
import MetricCard from './MetricCard';
import { User, Bot } from 'lucide-react';

interface MessageBubbleProps {
    role: 'user' | 'assistant';
    content: string;
    payload?: string;
}

const MessageBubble: React.FC<MessageBubbleProps> = ({ role, content, payload }) => {
    const isUser = role === 'user';
    let parsedPayload: any = null;

    if (payload) {
        try {
            parsedPayload = JSON.parse(payload);
        } catch (e) {
            console.error("Failed to parse payload", e);
        }
    }

    return (
        <div className={`flex items-start gap-4 ${isUser ? 'flex-row-reverse' : 'flex-row'} animate-in fade-in slide-in-from-bottom-2 duration-300`}>
            <div className={`flex-shrink-0 w-9 h-9 rounded-xl flex items-center justify-center shadow-sm ${
                isUser 
                    ? 'bg-slate-200 text-slate-600' 
                    : 'bg-gradient-to-br from-blue-500 to-indigo-600 text-white'
            }`}>
                {isUser ? <User className="w-5 h-5" /> : <Bot className="w-5 h-5" />}
            </div>

            <div 
                className={`max-w-[75%] rounded-2xl px-5 py-3.5 shadow-sm ${
                    isUser 
                        ? 'bg-blue-600 text-white rounded-tr-none' 
                        : 'bg-white border border-slate-100 text-slate-700 rounded-tl-none ring-1 ring-slate-50'
                }`}
            >
                <div className={`prose prose-sm font-normal leading-relaxed ${isUser ? 'prose-invert text-white' : 'text-slate-700'} max-w-none`}>
                    <ReactMarkdown>{content}</ReactMarkdown>
                </div>

                {parsedPayload && parsedPayload.type === 'visual_insight' && (
                    <div className="mt-4 pt-4 border-t border-slate-100">
                        <MetricCard 
                            title={parsedPayload.data.title}
                            value={parsedPayload.data.value}
                            change={parsedPayload.data.change}
                        />
                    </div>
                )}

                {parsedPayload && parsedPayload.type === 'actions' && (
                    <div className="mt-4 flex flex-wrap gap-2">
                        {parsedPayload.actions.map((action: any) => (
                            <button 
                                key={action.id}
                                className={`px-4 py-1.5 rounded-full text-xs font-bold transition-all border ${
                                    isUser
                                        ? 'bg-white/20 border-white/30 text-white hover:bg-white/30'
                                        : 'bg-blue-50 border-blue-100 text-blue-600 hover:bg-blue-100'
                                }`}
                            >
                                {action.label}
                            </button>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};

export default MessageBubble;