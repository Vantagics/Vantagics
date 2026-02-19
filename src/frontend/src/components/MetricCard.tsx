import React from 'react';

interface MetricCardProps {
    title: string;
    value: string;
    change: string;
}

const MetricCard: React.FC<MetricCardProps> = ({ title, value, change }) => {
    const isPositive = change.startsWith('+');
    const changeColor = isPositive ? 'text-green-500' : 'text-red-500';

    return (
        <div className="bg-white dark:bg-[#252526] rounded-xl p-6 flex flex-col justify-between hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-all duration-200 border border-slate-200 dark:border-[#3c3c3c] relative overflow-hidden">
            <div className="relative z-10">
                <h3 className="text-slate-500 dark:text-[#808080] text-sm font-medium tracking-wider mb-2">{title}</h3>
                <div className="flex items-end justify-between">
                    <span className="text-3xl font-bold text-slate-800 dark:text-[#d4d4d4]">{value}</span>
                    <span className={`text-sm font-semibold ${changeColor} bg-opacity-10 px-2 py-1 rounded-full ${isPositive ? 'bg-green-50' : 'bg-red-50'}`}>
                        {change}
                    </span>
                </div>
            </div>
        </div>
    );
};

export default MetricCard;
