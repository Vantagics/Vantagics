import React, { useState } from 'react';
import SmartInsight from './SmartInsight';
import DataSourceSelectionModal from './DataSourceSelectionModal';
import { StartDataSourceAnalysis } from '../../wailsjs/go/main/App';
import { agent } from '../../wailsjs/go/models';
import { EventsEmit } from '../../wailsjs/runtime/runtime';

interface DataSourceAnalysisInsightProps {
    statistics: agent.DataSourceStatistics;
    onAnalyzeClick?: (dataSourceId: string) => void;
}

const DataSourceAnalysisInsight: React.FC<DataSourceAnalysisInsightProps> = ({ 
    statistics, 
    onAnalyzeClick 
}) => {
    const [showSelection, setShowSelection] = useState(false);
    const [analyzing, setAnalyzing] = useState(false);
    
    const handleAnalyzeClick = async () => {
        // If multiple data sources, show selection modal
        if (statistics.data_sources.length > 1) {
            setShowSelection(true);
            return;
        }
        
        // If single data source, analyze directly
        if (statistics.data_sources.length === 1) {
            await startAnalysis(statistics.data_sources[0].id);
        }
    };
    
    const startAnalysis = async (dataSourceId: string) => {
        try {
            setAnalyzing(true);
            console.log('[DataSourceAnalysisInsight] 🚀 Starting analysis for dataSourceId:', dataSourceId);
            
            const threadId = await StartDataSourceAnalysis(dataSourceId);
            
            console.log('[DataSourceAnalysisInsight] ✅ Analysis started successfully:', { dataSourceId, threadId });
            console.log('[DataSourceAnalysisInsight] Backend should emit: analysis-session-created event');
            
            // Ensure chat panel is visible (session switching handled by AnalysisResultBridge)
            console.log('[DataSourceAnalysisInsight] Emitting ensure-chat-open event');
            EventsEmit('ensure-chat-open', {});
            
            // Notify parent or navigate to analysis view
            if (onAnalyzeClick) {
                console.log('[DataSourceAnalysisInsight] Calling onAnalyzeClick callback');
                onAnalyzeClick(dataSourceId);
            }
            
            // Could also navigate to chat with the thread ID
            // navigate(`/chat/${threadId}`);
            
        } catch (err) {
            console.error('[DataSourceAnalysisInsight] ❌ Failed to start analysis:', err);
            alert('分析启动失败: ' + (err instanceof Error ? err.message : '未知错误'));
        } finally {
            setAnalyzing(false);
            setShowSelection(false);
        }
    };
    
    // Generate insight text based on data source count
    const insightText = statistics.total_count === 1
        ? `发现 1 个数据源 (${statistics.data_sources[0].type})，点击开始智能分析`
        : `发现 ${statistics.total_count} 个数据源，点击选择并开始智能分析`;
    
    return (
        <>
            <SmartInsight
                text={analyzing ? '正在启动分析...' : insightText}
                icon="trending-up"
                onClick={analyzing ? undefined : handleAnalyzeClick}
            />
            
            {showSelection && (
                <DataSourceSelectionModal
                    isOpen={showSelection}
                    dataSources={statistics.data_sources}
                    onSelect={startAnalysis}
                    onClose={() => setShowSelection(false)}
                />
            )}
        </>
    );
};

export default DataSourceAnalysisInsight;
