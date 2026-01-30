import React, { useState, useEffect } from 'react';
import { ReadChartDataFile } from '../../wailsjs/go/main/App';
import Chart from './Chart';
import { useLanguage } from '../i18n';

interface EChartsFileLoaderProps {
    fileRef: string;
    threadId: string | null;
    chartKey: string;
    onDoubleClick: () => void;
}

const EChartsFileLoader: React.FC<EChartsFileLoaderProps> = ({ fileRef, threadId, chartKey, onDoubleClick }) => {
    const { t } = useLanguage();
    const [fileData, setFileData] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const loadFileData = async () => {
            try {
                setLoading(true);
                setError(null);

                if (!threadId) {
                    throw new Error('No active thread ID available');
                }

                console.log("[EChartsFileLoader] Loading chart data from file:", fileRef, "for thread:", threadId);
                const data = await ReadChartDataFile(threadId, fileRef);
                console.log("[EChartsFileLoader] Successfully loaded chart data from file, length:", data.length);

                setFileData(data);
                setLoading(false);
            } catch (error) {
                console.error("[EChartsFileLoader] Failed to load chart data file:", error);
                setError(error instanceof Error ? error.message : String(error));
                setLoading(false);
            }
        };

        loadFileData();
    }, [fileRef, threadId]);

    // Show loading state
    if (loading) {
        return (
            <div className="w-full bg-blue-50 border border-blue-200 rounded-xl p-4 shadow-sm">
                <div className="flex items-center gap-3">
                    <div className="animate-spin">
                        <svg className="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24">
                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                    </div>
                    <div className="flex-1">
                        <p className="text-sm font-medium text-blue-800">{t('loading_chart_data')}</p>
                        <p className="text-xs text-blue-600 mt-1">{t('reading_chart_data')}</p>
                    </div>
                </div>
            </div>
        );
    }

    // Show error state
    if (error) {
        return (
            <div className="w-full bg-blue-50 border border-blue-200 rounded-xl p-4 shadow-sm">
                <div className="flex items-center gap-3">
                    <div className="bg-blue-100 p-2 rounded-lg">
                        <svg className="w-5 h-5 text-red-600" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                        </svg>
                    </div>
                    <div className="flex-1">
                        <p className="text-sm font-medium text-red-800">{t('failed_load_chart')}</p>
                        <p className="text-xs text-red-600 mt-1">{error}</p>
                    </div>
                </div>
            </div>
        );
    }

    // Render chart with loaded data
    if (!fileData) {
        return null;
    }

    try {
        // Clean and parse the loaded data
        let cleanedData = fileData
            .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
            .replace(/,?\s*"?matter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
            .replace(/,?\s*[a-zA-Z_$][a-zA-Z0-9_$]*\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
            .replace(/,(\s*[}\]])/g, '$1')
            .replace(/(\{\s*),/g, '$1');

        const options = JSON.parse(cleanedData);

        if (!options || typeof options !== 'object') {
            console.error("[EChartsFileLoader] Invalid ECharts options: not an object", options);
            return null;
        }

        const validatedOptions = {
            ...options,
            animation: options.animation !== false,
            series: options.series || []
        };

        return (
            <div
                className="cursor-zoom-in group relative"
                onDoubleClick={onDoubleClick}
                title={t('double_click_expand')}
            >
                <Chart
                    key={chartKey}
                    options={validatedOptions}
                    height="400px"
                />
                <div className="absolute top-4 right-4 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
                    <span className="bg-slate-800/80 text-white px-3 py-1 rounded-full text-xs font-medium shadow-sm backdrop-blur-sm">{t('double_click_expand')}</span>
                </div>
            </div>
        );
    } catch (e) {
        console.error("[EChartsFileLoader] Failed to parse chart data:", e);
        return (
            <div className="w-full bg-blue-50 border border-blue-200 rounded-xl p-4 shadow-sm">
                <div className="flex items-center gap-3">
                    <div className="bg-blue-100 p-2 rounded-lg">
                        <svg className="w-5 h-5 text-red-600" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                        </svg>
                    </div>
                    <div className="flex-1">
                        <p className="text-sm font-medium text-red-800">{t('cannot_display_chart')}</p>
                        <p className="text-xs text-red-600 mt-1">{t('error')}: {(e as Error).message}</p>
                    </div>
                </div>
            </div>
        );
    }
};

export default EChartsFileLoader;
