import React, { useState, useEffect } from 'react';
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
import { GetDataSourceTables, GetDataSourceTableData, GetDataSourceTableCount, DeleteDataSource, GetConfig } from '../../wailsjs/go/main/App';
import { Table, Database, FileText, ChevronRight, List, Trash2 } from 'lucide-react';
import { useLanguage } from '../i18n';

interface ContextPanelProps {
    width: number;
    onContextPanelClick?: () => void;
}

const ContextPanel: React.FC<ContextPanelProps> = ({ width, onContextPanelClick }) => {
    const { t } = useLanguage();
    const [selectedSource, setSelectedSource] = useState<any>(null);
    const [tables, setTables] = useState<string[]>([]);
    const [tableCounts, setTableCounts] = useState<Record<string, number>>({});
    const [selectedTable, setSelectedTable] = useState<string | null>(null);
    const [tableData, setTableData] = useState<any[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [previewLimit, setPreviewLimit] = useState(100);

    useEffect(() => {
        GetConfig().then(config => {
            if (config.maxPreviewRows) {
                setPreviewLimit(config.maxPreviewRows);
            }
        }).catch(console.error);

        // Listen for config updates
        const unsubscribeConfig = EventsOn('config-updated', () => {
             GetConfig().then(config => {
                if (config.maxPreviewRows) {
                    setPreviewLimit(config.maxPreviewRows);
                }
            }).catch(console.error);
        });

        const unsubscribeSelect = EventsOn('data-source-selected', (source: any) => {
            setSelectedSource(source);
            setSelectedTable(null);
            setTableData([]);
            setTableCounts({});
            loadTables(source.id);
        });

        const unsubscribeDelete = EventsOn('data-source-deleted', (deletedId: string) => {
            if (selectedSource?.id === deletedId) {
                setSelectedSource(null);
                setSelectedTable(null);
                setTableData([]);
                setTables([]);
                setTableCounts({});
            }
        });

        return () => {
            if (unsubscribeConfig) unsubscribeConfig();
            if (unsubscribeSelect) unsubscribeSelect();
            if (unsubscribeDelete) unsubscribeDelete();
        };
    }, [selectedSource]);

    const loadTables = async (sourceId: string) => {
        setIsLoading(true);
        try {
            const t = await GetDataSourceTables(sourceId);
            setTables(t || []);
            
            // Fetch counts for each table
            const counts: Record<string, number> = {};
            if (t) {
                for (const tableName of t) {
                    try {
                        const count = await GetDataSourceTableCount(sourceId, tableName);
                        counts[tableName] = count;
                    } catch (e) {
                        console.error(`Failed to get count for table ${tableName}:`, e);
                    }
                }
                setTableCounts(counts);
            }
        } catch (err) {
            console.error('Failed to load tables:', err);
        } finally {
            setIsLoading(false);
        }
    };

    const handleDeleteSource = async () => {
        if (!selectedSource) return;
        if (confirm(`Are you sure you want to delete data source "${selectedSource.name}"?`)) {
            try {
                await DeleteDataSource(selectedSource.id);
                const deletedId = selectedSource.id;
                setSelectedSource(null);
                setSelectedTable(null);
                setTableData([]);
                setTables([]);
                setTableCounts({});
                // Tell sidebar to refresh
                EventsEmit('data-source-deleted', deletedId);
            } catch (err) {
                alert('Delete failed: ' + err);
            }
        }
    };

    const handleTableClick = async (tableName: string) => {
        setSelectedTable(tableName);
        setIsLoading(true);
        try {
            const data = await GetDataSourceTableData(selectedSource.id, tableName);
            setTableData(data || []);
        } catch (err) {
            console.error('Failed to load table data:', err);
        } finally {
            setIsLoading(false);
        }
    };

    const handleContextPanelClick = (e: React.MouseEvent) => {
        // 只有当点击的是非交互元素时才隐藏聊天
        const target = e.target as HTMLElement;
        const isInteractiveElement = target.tagName === 'BUTTON' || 
                                   target.tagName === 'A' || 
                                   target.tagName === 'INPUT' || 
                                   target.tagName === 'SELECT' || 
                                   target.tagName === 'TEXTAREA' ||
                                   target.closest('button') ||
                                   target.closest('a') ||
                                   target.closest('[role="button"]') ||
                                   target.closest('.cursor-pointer') ||
                                   target.closest('table');
        
        if (!isInteractiveElement && onContextPanelClick) {
            onContextPanelClick();
        }
    };

    return (
        <div 
            className="bg-white border-r border-slate-200 flex flex-col h-full shadow-sm flex-shrink-0"
            style={{ width: width }}
            onClick={handleContextPanelClick}
        >
            <div 
                className="p-4 pt-8 border-b border-slate-200 bg-slate-50 flex justify-between items-center"
            >
                <h2 className="text-lg font-semibold text-slate-700 flex items-center gap-2">
                    <Database className="w-4 h-4 text-blue-500" />
                    {t('data_explorer')}
                </h2>
            </div>
            
            <div className="flex-1 overflow-y-auto bg-slate-50/50">
                {!selectedSource ? (
                    <div className="h-full flex flex-col items-center justify-center p-8 text-center text-slate-400">
                        <div className="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mb-4">
                            <List className="w-8 h-8 text-slate-300" />
                        </div>
                        <p className="text-sm font-medium">{t('select_data_source_to_explore')}</p>
                    </div>
                ) : (
                    <div className="flex flex-col h-full">
                        {/* Source Header */}
                        <div className="p-4 bg-white border-b border-slate-200 flex justify-between items-start">
                            <div>
                                <div className="flex items-center gap-2 mb-1">
                                    <FileText className="w-4 h-4 text-green-600" />
                                    <span className="text-sm font-bold text-slate-800">
                                        {selectedSource.name} <span className="text-slate-400 font-normal">({tables.length})</span>
                                    </span>
                                </div>
                                <div className="text-[10px] text-slate-400 font-mono uppercase tracking-wider">{selectedSource.type} {t('source')}</div>
                            </div>
                        </div>

                        {/* Content Area */}
                        <div className="flex-1 overflow-y-auto">
                            {!selectedTable ? (
                                <div className="p-2">
                                    <h3 className="text-xs font-bold text-slate-500 uppercase tracking-widest p-2 mb-1">{t('tables')}</h3>
                                    <div className="space-y-1">
                                        {tables.map((table) => (
                                            <button
                                                key={table}
                                                onClick={() => handleTableClick(table)}
                                                className="w-full flex items-center justify-between p-3 bg-white border border-slate-200 rounded-lg hover:border-blue-300 hover:bg-blue-50 transition-all text-left shadow-sm group"
                                            >
                                                <div className="flex items-center gap-3">
                                                    <Table className="w-4 h-4 text-slate-400 group-hover:text-blue-500" />
                                                    <span className="text-sm font-medium text-slate-700">
                                                        {table} <span className="text-slate-400 font-normal">({tableCounts[table] !== undefined ? tableCounts[table] : '-'})</span>
                                                    </span>
                                                </div>
                                                <ChevronRight className="w-4 h-4 text-slate-300" />
                                            </button>
                                        ))}
                                    </div>
                                </div>
                            ) : (
                                <div className="flex flex-col h-full bg-white">
                                    <div className="p-2 border-b border-slate-100 flex items-center gap-2 bg-slate-50">
                                        <button 
                                            onClick={() => setSelectedTable(null)}
                                            className="text-[10px] font-bold text-blue-600 hover:text-blue-700 px-2 py-1 rounded hover:bg-blue-50 transition-colors"
                                        >
                                            {t('back_to_tables')}
                                        </button>
                                        <span className="text-xs text-slate-400">/</span>
                                        <span className="text-xs font-bold text-slate-700 truncate">{selectedTable}</span>
                                    </div>
                                    
                                    <div className="flex-1 overflow-auto">
                                        {isLoading ? (
                                            <div className="p-8 text-center text-slate-400 text-xs">{t('loading_data')}</div>
                                        ) : !tableData || tableData.length === 0 ? (
                                            <div className="p-8 text-center text-slate-400 text-xs italic">{t('no_data_in_table')}</div>
                                        ) : (
                                            <div className="inline-block min-w-full align-middle">
                                                <table className="min-w-full divide-y divide-slate-200">
                                                    <thead className="bg-slate-50 sticky top-0 z-10">
                                                        <tr>
                                                            {Object.keys(tableData[0]).map((col) => (
                                                                <th 
                                                                    key={col} 
                                                                    className="px-3 py-2 text-left text-[10px] font-bold text-slate-500 uppercase tracking-wider border-r border-slate-200 last:border-0"
                                                                >
                                                                    {col}
                                                                </th>
                                                            ))}
                                                        </tr>
                                                    </thead>
                                                    <tbody className="bg-white divide-y divide-slate-100">
                                                        {tableData.map((row, i) => (
                                                            <tr key={i} className="hover:bg-slate-50">
                                                                {Object.values(row).map((val: any, j) => (
                                                                    <td 
                                                                        key={j} 
                                                                        className="px-3 py-2 text-[10px] text-slate-600 border-r border-slate-100 last:border-0 truncate max-w-[150px]"
                                                                        title={String(val)}
                                                                    >
                                                                        {val === null ? <span className="text-slate-300 italic">null</span> : String(val)}
                                                                    </td>
                                                                ))}
                                                            </tr>
                                                        ))}
                                                    </tbody>
                                                </table>
                                            </div>
                                        )}
                                    </div>
                                    <div className="p-2 border-t border-slate-100 bg-slate-50 text-[10px] text-slate-400 text-center">
                                        {t('showing_preview_rows').replace('{0}', previewLimit.toString())}
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};

export default ContextPanel;