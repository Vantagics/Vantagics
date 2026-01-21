import React, { useState, useEffect, useRef } from 'react';
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
import { GetDataSourceTables, GetDataSourceTableData, GetDataSourceTableCount, DeleteDataSource, DeleteTable, GetConfig, RenameColumn } from '../../wailsjs/go/main/App';
import { Table, Database, FileText, ChevronRight, ChevronLeft, List, Trash2, Check, X } from 'lucide-react';
import { useLanguage } from '../i18n';
import DeleteTableConfirmationModal from './DeleteTableConfirmationModal';

interface ContextPanelProps {
    width: number;
    onContextPanelClick?: () => void;
    onCollapse?: () => void; // 折叠回调
}

interface EditingColumn {
    columnName: string;
    newName: string;
}

const ContextPanel: React.FC<ContextPanelProps> = ({ width, onContextPanelClick, onCollapse }) => {
    const { t } = useLanguage();
    const [selectedSource, setSelectedSource] = useState<any>(null);
    const [tables, setTables] = useState<string[]>([]);
    const [tableCounts, setTableCounts] = useState<Record<string, number>>({});
    const [selectedTable, setSelectedTable] = useState<string | null>(null);
    const [tableData, setTableData] = useState<any[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [previewLimit, setPreviewLimit] = useState(100);
    const [deleteTableTarget, setDeleteTableTarget] = useState<{ tableName: string; isLastTable?: boolean } | null>(null);
    
    // Column editing state
    const [editingColumn, setEditingColumn] = useState<EditingColumn | null>(null);
    const [editError, setEditError] = useState<string | null>(null);
    const [savingColumn, setSavingColumn] = useState(false);
    const inputRef = useRef<HTMLInputElement>(null);

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
            setEditingColumn(null);
            setEditError(null);
            loadTables(source.id);
        });

        const unsubscribeDelete = EventsOn('data-source-deleted', (deletedId: string) => {
            if (selectedSource?.id === deletedId) {
                setSelectedSource(null);
                setSelectedTable(null);
                setTableData([]);
                setTables([]);
                setTableCounts({});
                setEditingColumn(null);
                setEditError(null);
            }
        });

        return () => {
            if (unsubscribeConfig) unsubscribeConfig();
            if (unsubscribeSelect) unsubscribeSelect();
            if (unsubscribeDelete) unsubscribeDelete();
        };
    }, [selectedSource]);

    // Focus input when editing starts
    useEffect(() => {
        if (editingColumn && inputRef.current) {
            inputRef.current.focus();
            inputRef.current.select();
        }
    }, [editingColumn]);

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
        setEditingColumn(null);
        setEditError(null);
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

    const handleDeleteTable = (tableName: string, e: React.MouseEvent) => {
        e.preventDefault();
        e.stopPropagation();
        // Check if this is the last table
        const isLastTable = tables.length === 1;
        console.log('[DEBUG] Delete table clicked:', { tableName, isLastTable, tablesLength: tables.length, tables });
        setDeleteTableTarget({ tableName, isLastTable });
    };

    const confirmDeleteTable = async () => {
        if (!deleteTableTarget || !selectedSource) return;
        console.log('[DEBUG] Confirm delete table:', { deleteTableTarget, selectedSourceId: selectedSource.id, selectedSourceName: selectedSource.name });
        try {
            // If this is the last table, delete the entire data source
            if (deleteTableTarget.isLastTable) {
                console.log('[DEBUG] Deleting entire data source (last table)');
                await DeleteDataSource(selectedSource.id);
                EventsEmit('data-source-deleted', selectedSource.id);
                setSelectedSource(null);
                setSelectedTable(null);
                setTableData([]);
                setTables([]);
                setTableCounts({});
            } else {
                // Otherwise, just delete the table
                console.log('[DEBUG] Deleting only the table');
                await DeleteTable(selectedSource.id, deleteTableTarget.tableName);
                // Refresh table list
                await loadTables(selectedSource.id);
                // If the deleted table was selected, clear selection
                if (selectedTable === deleteTableTarget.tableName) {
                    setSelectedTable(null);
                    setTableData([]);
                }
            }
            setDeleteTableTarget(null);
        } catch (err) {
            console.error('Failed to delete table:', err);
            alert('Failed to delete table: ' + err);
        }
    };

    // Column editing handlers
    const handleColumnDoubleClick = (columnName: string) => {
        setEditingColumn({
            columnName,
            newName: columnName
        });
        setEditError(null);
    };

    const validateColumnName = (newName: string, originalName: string): string | null => {
        // Check if empty
        if (!newName.trim()) {
            return t('column_name_empty');
        }

        // Check for invalid characters
        const invalidChars = [' ', "'", '"', ';', '--', '/*', '*/', '\t', '\n', '\r'];
        for (const char of invalidChars) {
            if (newName.includes(char)) {
                return t('column_name_invalid_char').replace('{char}', char === ' ' ? t('space') : char);
            }
        }

        // Check if starts with number
        if (/^[0-9]/.test(newName)) {
            return t('column_name_starts_with_number');
        }

        // Check for duplicate names in the current table
        if (tableData.length > 0) {
            const columns = Object.keys(tableData[0]);
            const isDuplicate = columns.some(col => col === newName && col !== originalName);
            if (isDuplicate) {
                return t('column_name_duplicate').replace('{name}', newName);
            }
        }

        return null;
    };

    const handleColumnSave = async () => {
        if (!editingColumn || !selectedSource || !selectedTable) return;

        const { columnName, newName } = editingColumn;
        
        // Skip if name hasn't changed
        if (newName === columnName) {
            setEditingColumn(null);
            setEditError(null);
            return;
        }

        // Validate
        const error = validateColumnName(newName, columnName);
        if (error) {
            setEditError(error);
            return;
        }

        setSavingColumn(true);
        try {
            await RenameColumn(selectedSource.id, selectedTable, columnName, newName);
            
            // Reload table data to reflect the change
            const data = await GetDataSourceTableData(selectedSource.id, selectedTable);
            setTableData(data || []);
            
            // Emit event to notify other components (Sidebar, DataSourcePropertiesModal)
            EventsEmit('column-renamed', {
                dataSourceId: selectedSource.id,
                tableName: selectedTable,
                oldColumnName: columnName,
                newColumnName: newName
            });
            
            setEditingColumn(null);
            setEditError(null);
        } catch (err) {
            console.error('Failed to rename column:', err);
            setEditError(t('rename_column_failed') + ': ' + err);
        } finally {
            setSavingColumn(false);
        }
    };

    const handleColumnCancel = () => {
        setEditingColumn(null);
        setEditError(null);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleColumnSave();
        } else if (e.key === 'Escape') {
            handleColumnCancel();
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

    // Render column header (editable)
    const renderColumnHeader = (col: string) => {
        const isEditing = editingColumn?.columnName === col;

        if (isEditing) {
            return (
                <th 
                    key={col} 
                    className="px-2 py-2 text-left text-[10px] font-bold text-slate-500 uppercase tracking-wider border-r border-slate-200 last:border-0 bg-blue-50"
                    style={{ minWidth: '120px' }}
                >
                    <div className="flex items-center gap-1">
                        <input
                            ref={inputRef}
                            type="text"
                            value={editingColumn.newName}
                            onChange={(e) => setEditingColumn({
                                ...editingColumn,
                                newName: e.target.value
                            })}
                            onKeyDown={handleKeyDown}
                            className={`flex-1 min-w-[80px] px-2 py-1 text-[10px] border rounded focus:outline-none focus:ring-1 ${
                                editError 
                                    ? 'border-red-400 focus:ring-red-200 bg-red-50' 
                                    : 'border-blue-400 focus:ring-blue-200 bg-white'
                            }`}
                            disabled={savingColumn}
                        />
                        <button
                            onClick={handleColumnSave}
                            disabled={savingColumn}
                            className="p-0.5 text-green-600 hover:text-green-700 disabled:opacity-50 flex-shrink-0"
                            title={t('save')}
                        >
                            <Check className="w-3 h-3" />
                        </button>
                        <button
                            onClick={handleColumnCancel}
                            disabled={savingColumn}
                            className="p-0.5 text-red-600 hover:text-red-700 disabled:opacity-50 flex-shrink-0"
                            title={t('cancel')}
                        >
                            <X className="w-3 h-3" />
                        </button>
                    </div>
                    {editError && (
                        <div className="mt-1 text-[9px] text-red-600 normal-case tracking-normal font-normal whitespace-nowrap">
                            {editError}
                        </div>
                    )}
                </th>
            );
        }

        return (
            <th 
                key={col} 
                className="px-3 py-2 text-left text-[10px] font-bold text-slate-500 uppercase tracking-wider border-r border-slate-200 last:border-0 cursor-pointer hover:bg-blue-50 hover:text-blue-600 transition-colors"
                onDoubleClick={() => handleColumnDoubleClick(col)}
                title={t('double_click_to_edit')}
            >
                {col}
            </th>
        );
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
                {onCollapse && (
                    <button
                        onClick={onCollapse}
                        className="p-1 hover:bg-slate-200 rounded-md text-slate-500 hover:text-blue-600 transition-colors"
                        title="折叠数据浏览器"
                    >
                        <ChevronLeft className="w-4 h-4" />
                    </button>
                )}
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
                                            <div
                                                key={table}
                                                className="w-full flex items-center justify-between p-3 bg-white border border-slate-200 rounded-lg hover:border-blue-300 hover:bg-blue-50 transition-all shadow-sm group"
                                            >
                                                <button
                                                    onClick={() => handleTableClick(table)}
                                                    className="flex-1 flex items-center gap-3 text-left"
                                                >
                                                    <Table className="w-4 h-4 text-slate-400 group-hover:text-blue-500" />
                                                    <span className="text-sm font-medium text-slate-700">
                                                        {table} <span className="text-slate-400 font-normal">({tableCounts[table] !== undefined ? tableCounts[table] : '-'})</span>
                                                    </span>
                                                </button>
                                                <div className="flex items-center gap-1">
                                                    <button
                                                        onClick={(e) => handleDeleteTable(table, e)}
                                                        className="p-1 hover:text-red-600 transition-opacity opacity-0 group-hover:opacity-100"
                                                        title={t('delete_table')}
                                                    >
                                                        <Trash2 className="w-3 h-3" />
                                                    </button>
                                                    <ChevronRight className="w-4 h-4 text-slate-300" />
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            ) : (
                                <div className="flex flex-col h-full bg-white">
                                    <div className="p-2 border-b border-slate-100 flex items-center justify-between bg-slate-50">
                                        <div className="flex items-center gap-2">
                                            <button 
                                                onClick={() => {
                                                    setSelectedTable(null);
                                                    setEditingColumn(null);
                                                    setEditError(null);
                                                }}
                                                className="text-[10px] font-bold text-blue-600 hover:text-blue-700 px-2 py-1 rounded hover:bg-blue-50 transition-colors"
                                            >
                                                {t('back_to_tables')}
                                            </button>
                                            <span className="text-xs text-slate-400">/</span>
                                            <span className="text-xs font-bold text-slate-700 truncate">{selectedTable}</span>
                                        </div>
                                        <span className="text-[9px] text-slate-400 italic">{t('double_click_to_edit_column')}</span>
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
                                                            {Object.keys(tableData[0]).map((col) => renderColumnHeader(col))}
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

            <DeleteTableConfirmationModal
                isOpen={!!deleteTableTarget}
                tableName={deleteTableTarget?.tableName || ''}
                isLastTable={deleteTableTarget?.isLastTable || false}
                dataSourceName={selectedSource?.name || ''}
                onClose={() => setDeleteTableTarget(null)}
                onConfirm={confirmDeleteTable}
            />
        </div>
    );
};

export default ContextPanel;
