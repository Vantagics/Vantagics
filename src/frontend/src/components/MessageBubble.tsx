import React from 'react';
import ReactMarkdown from 'react-markdown';
import MetricCard from './MetricCard';
import Chart from './Chart';
import DataTable from './DataTable';
import { User, Bot } from 'lucide-react';

interface MessageBubbleProps {
    role: 'user' | 'assistant';
    content: string;
    payload?: string;
    onActionClick?: (action: any) => void;
}

const MessageBubble: React.FC<MessageBubbleProps> = ({ role, content, payload, onActionClick }) => {
    const isUser = role === 'user';
    let parsedPayload: any = null;

    if (payload) {
        try {
            parsedPayload = JSON.parse(payload);
        } catch (e) {
            console.error("Failed to parse payload", e);
        }
    }

    // Auto-extract numbered list actions
    const extractedActions: any[] = [];
    if (!isUser) {
        const lines = content.split('\n');
        for (const line of lines) {
            // Match lines starting with "1. ", "2. ", etc.
            const match = line.match(/^(\d+)\.\s+(.*)$/);
            if (match) {
                const rawLabel = match[2].trim();
                // Avoid extracting very long text or non-titles
                if (rawLabel.length > 0 && rawLabel.length < 100) {
                    extractedActions.push({
                        id: `auto_${match[1]}`,
                        label: rawLabel,
                        // Value should be clean text for the LLM input (no markdown)
                        value: rawLabel.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '')
                    });
                }
            }
        }
    }

    const allActions = [
        ...(parsedPayload && parsedPayload.type === 'actions' ? parsedPayload.actions : []),
        ...extractedActions
    ];

    const renderButtonLabel = (label: string) => {
        // Split by bold markers **text**
        const parts = label.split(/(\*\*.*?\*\*)/g);
        return parts.map((part, i) => {
            if (part.startsWith('**') && part.endsWith('**')) {
                return <strong key={i} className="font-black underline-offset-2">{part.slice(2, -2)}</strong>;
            }
            return part;
        });
    };

    const cleanedContent = content.replace(/```[ \t]*json:dashboard[\s\S]*?```/g, '').trim();

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
                className={`max-w-[85%] rounded-2xl px-5 py-3.5 shadow-sm ${
                    isUser 
                        ? 'bg-blue-600 text-white rounded-tr-none' 
                        : 'bg-white border border-slate-100 text-slate-700 rounded-tl-none ring-1 ring-slate-50'
                }`}
            >
                <div className={`prose prose-sm font-normal leading-relaxed ${isUser ? 'prose-invert text-white' : 'text-slate-700'} max-w-none`}>
                    <ReactMarkdown
                        components={{
                            code(props) {
                                const {children, className, node, ...rest} = props;
                                const match = /language-(\w+)/.exec(className || '');
                                // Handle specific custom languages (formats: language-json:echarts or just json:echarts if passed directly)
                                // ReactMarkdown usually prefixes with language-
                                
                                const isECharts = className?.includes('json:echarts');
                                const isTable = className?.includes('json:table');

                                if (isECharts) {
                                    try {
                                        const data = JSON.parse(String(children).replace(/\n$/, ''));
                                        return <Chart options={data} />;
                                    } catch (e) {
                                        console.error("Failed to parse ECharts JSON", e);
                                    }
                                }

                                if (isTable) {
                                    try {
                                        const data = JSON.parse(String(children).replace(/\n$/, ''));
                                        return <DataTable data={data} />;
                                    } catch (e) {
                                        console.error("Failed to parse Table JSON", e);
                                    }
                                }
                                
                                return <code {...rest} className={className}>{children}</code>;
                            }
                        }}
                    >
                        {cleanedContent}
                    </ReactMarkdown>
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
                
                {parsedPayload && parsedPayload.type === 'echarts' && (
                     <div className="mt-4 pt-4 border-t border-slate-100">
                        <Chart options={parsedPayload.data} />
                    </div>
                )}
                
                {parsedPayload && parsedPayload.type === 'table' && (
                     <div className="mt-4 pt-4 border-t border-slate-100">
                        <DataTable data={parsedPayload.data} />
                    </div>
                )}

                {allActions.length > 0 && (
                    <div className="mt-4 flex flex-wrap gap-2">
                        {allActions.map((action: any) => (
                            <button 
                                key={action.id}
                                onClick={() => onActionClick && onActionClick(action)}
                                className={`px-4 py-1.5 rounded-full text-xs font-medium transition-all border ${
                                    isUser
                                        ? 'bg-white/20 border-white/30 text-white hover:bg-white/30'
                                        : 'bg-blue-50 border-blue-100 text-blue-600 hover:bg-blue-100'
                                } shadow-sm hover:shadow-md active:scale-95`}
                            >
                                {renderButtonLabel(action.label)}
                            </button>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};

export default MessageBubble;