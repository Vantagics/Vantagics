import React from 'react';
import ReactMarkdown from 'react-markdown';
import { TrendingUp, UserCheck, AlertCircle, Star, Info } from 'lucide-react';

interface SmartInsightProps {
    text: string;
    icon: string;
    onClick?: () => void;
}

const iconMap: Record<string, React.ReactNode> = {
    'trending-up': <TrendingUp className="w-5 h-5 text-blue-500" />,
    'user-check': <UserCheck className="w-5 h-5 text-green-500" />,
    'alert-circle': <AlertCircle className="w-5 h-5 text-amber-500" />,
    'star': <Star className="w-5 h-5 text-purple-500" />,
    'info': <Info className="w-5 h-5 text-slate-500" />,
};

const SmartInsight: React.FC<SmartInsightProps> = ({ text, icon, onClick }) => {
    const IconComponent = iconMap[icon] || iconMap['info'];

    const handleClick = (e: React.MouseEvent) => {
        e.preventDefault();
        e.stopPropagation(); // 阻止事件冒泡，防止触发Dashboard的点击处理
        if (onClick) {
            onClick();
        }
    };

    return (
        <div 
            onClick={handleClick}
            className={`bg-white rounded-xl shadow-sm p-4 flex items-start gap-4 border-l-4 border-blue-500 hover:shadow-md transition-shadow duration-200 hover:bg-slate-50/50 ${onClick ? 'cursor-pointer active:scale-[0.99] transition-transform' : ''}`}
        >
            <div className="insight-icon bg-gradient-to-br from-slate-50 to-slate-100 p-2 rounded-lg shrink-0 shadow-inner">
                {IconComponent}
            </div>
            <div className="text-slate-700 text-sm leading-relaxed pt-1 prose prose-sm max-w-none">
                <ReactMarkdown
                    components={{
                        // 自定义markdown组件样式
                        p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                        strong: ({ children }) => <strong className="font-semibold text-slate-800">{children}</strong>,
                        em: ({ children }) => <em className="italic">{children}</em>,
                        ul: ({ children }) => <ul className="list-disc list-inside mb-2">{children}</ul>,
                        ol: ({ children }) => <ol className="list-decimal list-inside mb-2">{children}</ol>,
                        li: ({ children }) => <li className="mb-1">{children}</li>,
                        code: ({ children }) => <code className="bg-slate-100 px-1 py-0.5 rounded text-xs font-mono">{children}</code>,
                    }}
                >
                    {text}
                </ReactMarkdown>
            </div>
        </div>
    );
};

export default SmartInsight;
