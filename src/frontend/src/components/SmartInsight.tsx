import React from 'react';
import { TrendingUp, UserCheck, AlertCircle, Star, Info } from 'lucide-react';

interface SmartInsightProps {
    text: string;
    icon: string;
}

const iconMap: Record<string, React.ReactNode> = {
    'trending-up': <TrendingUp className="w-5 h-5 text-blue-500" />,
    'user-check': <UserCheck className="w-5 h-5 text-green-500" />,
    'alert-circle': <AlertCircle className="w-5 h-5 text-amber-500" />,
    'star': <Star className="w-5 h-5 text-purple-500" />,
    'info': <Info className="w-5 h-5 text-slate-500" />,
};

const SmartInsight: React.FC<SmartInsightProps> = ({ text, icon }) => {
    const IconComponent = iconMap[icon] || iconMap['info'];

    return (
        <div className="bg-white rounded-xl shadow-sm p-4 flex items-start gap-4 border-l-4 border-blue-500 hover:shadow-md transition-shadow duration-200 hover:bg-slate-50/50">
            <div className="insight-icon bg-gradient-to-br from-slate-50 to-slate-100 p-2 rounded-lg shrink-0 shadow-inner">
                {IconComponent}
            </div>
            <p className="text-slate-700 text-sm leading-relaxed pt-1">
                {text}
            </p>
        </div>
    );
};

export default SmartInsight;
