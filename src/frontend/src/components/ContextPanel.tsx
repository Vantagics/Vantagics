import React from 'react';

const ContextPanel: React.FC = () => {
    return (
        <div className="w-96 bg-white border-l border-slate-200 flex flex-col h-full shadow-sm">
            <div 
                className="p-4 pt-8 border-b border-slate-200 bg-slate-50 flex justify-between items-center"
                style={{ WebkitAppRegion: 'drag' } as any}
            >
                <h2 className="text-lg font-semibold text-slate-700">Context / Results</h2>
                <span className="text-xs font-mono text-slate-400">Read-only</span>
            </div>
            
            <div className="flex-1 overflow-y-auto p-4 bg-slate-50/50">
                {/* Mock Report / Grid */}
                <div className="bg-white border border-slate-200 rounded-lg p-4 mb-4 shadow-sm">
                    <h3 className="text-sm font-bold text-slate-800 mb-2">Analysis Report: Q3 Sales</h3>
                    <div className="w-full h-32 bg-blue-50 rounded mb-2 flex items-center justify-center text-blue-300">
                        [Chart Placeholder]
                    </div>
                    <p className="text-xs text-slate-500 leading-relaxed">
                        Sales increased by 15% compared to Q2. The primary driver was the new subscription model.
                    </p>
                </div>

                <div className="bg-white border border-slate-200 rounded-lg overflow-hidden shadow-sm">
                    <table className="w-full text-left text-xs">
                        <thead className="bg-slate-100 text-slate-600 font-medium border-b border-slate-200">
                            <tr>
                                <th className="p-2">Region</th>
                                <th className="p-2">Revenue</th>
                                <th className="p-2">Growth</th>
                            </tr>
                        </thead>
                        <tbody className="text-slate-500">
                            <tr className="border-b border-slate-100">
                                <td className="p-2">North</td>
                                <td className="p-2">$120k</td>
                                <td className="p-2 text-green-600">+12%</td>
                            </tr>
                            <tr className="border-b border-slate-100">
                                <td className="p-2">East</td>
                                <td className="p-2">$95k</td>
                                <td className="p-2 text-red-500">-2%</td>
                            </tr>
                            <tr>
                                <td className="p-2">West</td>
                                <td className="p-2">$145k</td>
                                <td className="p-2 text-green-600">+8%</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    );
};

export default ContextPanel;
