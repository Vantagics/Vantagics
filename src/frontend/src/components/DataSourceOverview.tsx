import React, { useState, useEffect } from 'react';
import { Database, RefreshCw, AlertCircle } from 'lucide-react';
import { GetDataSourceStatistics } from '../../wailsjs/go/main/App';
import { agent } from '../../wailsjs/go/models';
import DataSourceAnalysisInsight from './DataSourceAnalysisInsight';
import '../styles/datasource-overview.css';

interface DataSourceOverviewProps {
    onAnalyzeClick?: (dataSourceId: string) => void;
}

const DataSourceOverview: React.FC<DataSourceOverviewProps> = ({ onAnalyzeClick }) => {
    const [statistics, setStatistics] = useState<agent.DataSourceStatistics | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        loadStatistics();
    }, []);

    const loadStatistics = async () => {
        try {
            setLoading(true);
            setError(null);
            const stats = await GetDataSourceStatistics();
            setStatistics(stats);
        } catch (err) {
            setError(err instanceof Error ? err.message : '加载数据源统计信息失败');
            console.error('Failed to load data source statistics:', err);
        } finally {
            setLoading(false);
        }
    };

    // Loading state UI
    if (loading) {
        return (
            <div className="data-source-overview bg-white rounded-lg shadow-sm border border-slate-200 p-6">
                <div className="flex items-center justify-center space-x-3">
                    <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-500"></div>
                    <p className="text-slate-600">加载数据源信息...</p>
                </div>
            </div>
        );
    }

    // Error state UI
    if (error) {
        return (
            <div className="data-source-overview bg-white rounded-lg shadow-sm border border-red-200 p-6">
                <div className="flex items-start space-x-3">
                    <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
                    <div className="flex-1">
                        <p className="text-red-700 font-medium mb-2">加载失败</p>
                        <p className="text-red-600 text-sm mb-3">{error}</p>
                        <button
                            onClick={loadStatistics}
                            className="inline-flex items-center space-x-2 px-4 py-2 bg-red-50 hover:bg-red-100 text-red-700 rounded-md transition-colors text-sm font-medium"
                        >
                            <RefreshCw className="w-4 h-4" />
                            <span>重试</span>
                        </button>
                    </div>
                </div>
            </div>
        );
    }

    // Empty state UI
    if (!statistics || statistics.total_count === 0) {
        return (
            <div className="data-source-overview bg-white rounded-lg shadow-sm border border-slate-200 p-6">
                <div className="flex items-center justify-center space-x-3 text-slate-500">
                    <Database className="w-5 h-5" />
                    <p>暂无数据源</p>
                </div>
            </div>
        );
    }

    // Statistics display UI
    return (
        <div className="data-source-overview bg-white rounded-lg shadow-sm border border-slate-200 p-6">
            {/* Header with total count */}
            <div className="overview-header flex items-center justify-between mb-4">
                <div className="flex items-center space-x-2">
                    <Database className="w-5 h-5 text-blue-500" />
                    <h3 className="text-lg font-semibold text-slate-800">数据源概览</h3>
                </div>
                <div className="total-count flex items-center space-x-2 bg-blue-50 px-4 py-2 rounded-lg">
                    <span className="text-sm text-slate-600">总数:</span>
                    <span className="text-xl font-bold text-blue-600">{statistics.total_count}</span>
                </div>
            </div>

            {/* Breakdown by type */}
            <div className="breakdown">
                <h4 className="text-sm font-medium text-slate-700 mb-3">按类型统计</h4>
                <div className="breakdown-list grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                    {Object.entries(statistics.breakdown_by_type)
                        .sort(([, a], [, b]) => b - a) // Sort by count descending
                        .map(([type, count]) => (
                            <div
                                key={type}
                                className="breakdown-item bg-slate-50 rounded-lg p-3 border border-slate-200 hover:border-blue-300 hover:bg-blue-50 transition-colors"
                            >
                                <div className="flex items-center justify-between">
                                    <span className="type-name text-sm font-medium text-slate-700 uppercase">
                                        {type}
                                    </span>
                                    <span className="type-count text-lg font-bold text-blue-600">
                                        {count}
                                    </span>
                                </div>
                            </div>
                        ))}
                </div>
            </div>

            {/* Smart Insight for One-Click Analysis */}
            <div className="mt-4">
                <DataSourceAnalysisInsight 
                    statistics={statistics}
                    onAnalyzeClick={onAnalyzeClick}
                />
            </div>
        </div>
    );
};

export default DataSourceOverview;
