import React from 'react';
import DashboardLayout from './DashboardLayout';
import MetricCard from './MetricCard';
import SmartInsight from './SmartInsight';
import { main } from '../../wailsjs/go/models';

interface DashboardProps {
    data: main.DashboardData | null;
}

const Dashboard: React.FC<DashboardProps> = ({ data }) => {
    if (!data) {
        return (
            <div className="flex items-center justify-center h-full">
                <div className="animate-pulse text-slate-400">Loading your insights...</div>
            </div>
        );
    }

    return (
        <div className="flex-1 flex flex-col h-full overflow-hidden">
            <header className="px-6 py-8">
                <h1 className="text-2xl font-bold text-slate-800">Smart Dashboard</h1>
                <p className="text-slate-500">Welcome back! Here's what's happening with your data.</p>
            </header>
            
            <div className="flex-1 overflow-y-auto px-6 pb-8">
                <section className="mb-8">
                    <h2 className="text-lg font-semibold text-slate-700 mb-4">Key Metrics</h2>
                    <DashboardLayout>
                        {data.metrics.map((metric, index) => (
                            <MetricCard 
                                key={index}
                                title={metric.title}
                                value={metric.value}
                                change={metric.change}
                            />
                        ))}
                    </DashboardLayout>
                </section>

                <section>
                    <h2 className="text-lg font-semibold text-slate-700 mb-4">Automated Insights</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        {data.insights.map((insight, index) => (
                            <SmartInsight 
                                key={index}
                                text={insight.text}
                                icon={insight.icon}
                            />
                        ))}
                    </div>
                </section>
            </div>
        </div>
    );
};

export default Dashboard;
