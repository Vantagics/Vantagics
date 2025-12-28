import React from 'react';
import ReactMarkdown from 'react-markdown';
import MetricCard from './MetricCard';

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
        <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} mb-2`}>
            <div 
                className={`max-w-[85%] rounded-2xl px-4 py-3 text-sm shadow-sm ${
                    isUser 
                        ? 'bg-blue-600 text-white rounded-br-none' 
                        : 'bg-white border border-slate-200 text-slate-700 rounded-bl-none'
                }`}
            >
                <div className={`prose prose-sm ${isUser ? 'prose-invert text-white' : 'text-slate-700'} max-w-none`}>
                    <ReactMarkdown>{content}</ReactMarkdown>
                </div>

                {parsedPayload && parsedPayload.type === 'visual_insight' && (
                    <div className="mt-3">
                        <MetricCard 
                            title={parsedPayload.data.title}
                            value={parsedPayload.data.value}
                            change={parsedPayload.data.change}
                        />
                    </div>
                )}

                {parsedPayload && parsedPayload.type === 'actions' && (
                    <div className="mt-3 flex flex-wrap gap-2">
                        {parsedPayload.actions.map((action: any) => (
                            <button 
                                key={action.id}
                                className="px-3 py-1 bg-blue-50 text-blue-600 rounded-full text-xs font-medium hover:bg-blue-100 transition-colors border border-blue-100"
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
