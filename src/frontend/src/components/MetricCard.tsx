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
        <div className="bg-white rounded-xl shadow-sm p-6 flex flex-col justify-between hover:shadow-md transition-shadow duration-200">
            <h3 className="text-slate-500 text-sm font-medium uppercase tracking-wider mb-2">{title}</h3>
            <div className="flex items-end justify-between">
                <span className="text-3xl font-bold text-slate-800">{value}</span>
                <span className={`text-sm font-semibold ${changeColor} bg-opacity-10 px-2 py-1 rounded-full ${isPositive ? 'bg-green-50' : 'bg-red-50'}`}>
                    {change}
                </span>
            </div>
        </div>
    );
};

export default MetricCard;
