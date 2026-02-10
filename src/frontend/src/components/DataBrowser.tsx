import React, { useCallback, useEffect, useRef, useState, useMemo } from 'react';
import { X, Database, Search, ChevronLeft, ChevronRight, Table2, Columns3, AlertCircle, Loader2, Hash, Trash2, Check, Pencil } from 'lucide-react';
import { GetDataSourceTables, GetDataSourceTableData, GetDataSourceTableCount, RenameColumn, DeleteColumn, DeleteDataSource, DeleteTable, GetConfig } from '../../wailsjs/go/main/App';
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
import DeleteColumnConfirmationModal from './DeleteColumnConfirmationModal';
import { useLanguage } from '../i18n';
import './DataBrowser.css';

/**
 * DataBrowser Props Interface
 * Requirements: 7.1, 7.2, 7.3, 7.5, 7.6, 7.9, 7.10, 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7, 8.8
 */
export interface DataBrowserProps {
    /** Whether the data browser panel is open */
    isOpen: boolean;
    /** The data source ID to browse, null when no source selected */
    sourceId: string | null;
    /** The data source name to display in the header - Requirement 8.1 */
    sourceName?: string | null;
    /** Callback to close the data browser */
    onClose: () => void;
    /** Current width of the data browser panel in pixels */
    width: number;
    /** Callback when the width changes via resize handle drag */
    onWidthChange: (width: number) => void;
}

/** Table info with metadata */
export interface TableInfo {
    name: string;
    rowCount: number;
    columnCount: number;
}

/** Column info with type */
export interface ColumnInfo {
    name: string;
    type: string;
}

/** Minimum width for the data browser panel */
const MIN_WIDTH = 300;
/** Maximum width ratio relative to container */
const MAX_WIDTH_RATIO = 0.9;
/** Number of rows per page for pagination */
export const ROWS_PER_PAGE = 15;

/**
 * DataBrowser Component
 *
 * A slide-out panel for browsing data source contents.
 * Slides in from the right over the CenterPanel.
 * Does NOT overlay the Left or Right panels.
 *
 * Requirements:
 * - 7.1: Slide in from the right when "Browse Data" is selected
 * - 7.2: Overlay the Center_Panel (chat area) when visible
 * - 7.3: NOT overlay the Left_Panel or Right_Panel
 * - 7.5: Include a close button (X) in its header
 * - 7.6: Slide out when close button is clicked
 * - 7.9: Dim or blur the Center_Panel content behind it
 * - 7.10: Allow users to resize by dragging its left edge
 * - 8.1: Display the selected data source name in the header
 * - 8.2: Display a list of tables/sheets in the data source
 * - 8.3: Display columns and data types when a table is selected
 * - 8.4: Display sample data rows (first 10-20 rows)
 * - 8.5: Provide pagination controls for browsing additional data rows
 * - 8.6: Display row counts and column statistics
 * - 8.7: Display error message when data loading fails
 * - 8.8: Include search/filter capability for finding tables and columns
 */
const DataBrowser: React.FC<DataBrowserProps> = ({
    isOpen,
    sourceId,
    sourceName,
    onClose,
    width,
    onWidthChange,
}) => {
    const [isResizing, setIsResizing] = useState(false);
    const resizeStartX = useRef<number>(0);
    const resizeStartWidth = useRef<number>(0);
    const containerRef = useRef<HTMLDivElement>(null);

    // Data browser content state - Requirements 8.2, 8.3, 8.4
    const [tables, setTables] = useState<string[]>([]);
    const [selectedTable, setSelectedTable] = useState<string | null>(null);
    const [tableData, setTableData] = useState<Record<string, any>[]>([]);
    const [tableRowCount, setTableRowCount] = useState<number>(0);
    const [currentPage, setCurrentPage] = useState<number>(1);
    const [isLoadingTables, setIsLoadingTables] = useState(false);
    const [isLoadingData, setIsLoadingData] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [searchQuery, setSearchQuery] = useState('');

    // Column editing state
    const [editingColumn, setEditingColumn] = useState<{ columnName: string; newName: string } | null>(null);
    const [editError, setEditError] = useState<string | null>(null);
    const [savingColumn, setSavingColumn] = useState(false);
    const editInputRef = useRef<HTMLInputElement>(null);

    // Column delete state
    const [deleteColumnTarget, setDeleteColumnTarget] = useState<{ columnName: string; isLastColumn: boolean; isLastTable: boolean } | null>(null);

    // i18n
    const { t } = useLanguage();

    // Preview limit from config
    const [previewLimit, setPreviewLimit] = useState(100);

    /**
     * Derive columns from the first row of table data.
     * Requirement 8.3
     */
    const columns: ColumnInfo[] = useMemo(() => {
        if (tableData.length === 0) return [];
        const firstRow = tableData[0];
        return Object.keys(firstRow).map((key) => ({
            name: key,
            type: inferColumnType(firstRow[key]),
        }));
    }, [tableData]);

    /**
     * Filter tables based on search query.
     * Requirement 8.8
     */
    const filteredTables = useMemo(() => {
        if (!searchQuery.trim()) return tables;
        const query = searchQuery.toLowerCase();
        return tables.filter((table) => table.toLowerCase().includes(query));
    }, [tables, searchQuery]);

    /**
     * Filter columns based on search query (when a table is selected).
     * Requirement 8.8
     */
    const filteredColumns = useMemo(() => {
        if (!searchQuery.trim()) return columns;
        const query = searchQuery.toLowerCase();
        return columns.filter(
            (col) =>
                col.name.toLowerCase().includes(query) ||
                col.type.toLowerCase().includes(query)
        );
    }, [columns, searchQuery]);

    /**
     * Paginated data rows.
     * Requirement 8.5
     */
    const paginatedRows = useMemo(() => {
        const start = (currentPage - 1) * ROWS_PER_PAGE;
        const end = start + ROWS_PER_PAGE;
        return tableData.slice(start, end);
    }, [tableData, currentPage]);

    const totalPages = Math.max(1, Math.ceil(tableData.length / ROWS_PER_PAGE));

    /**
     * Load tables when the data browser opens with a source.
     * Requirement 8.2
     */
    useEffect(() => {
        if (isOpen && sourceId) {
            loadTables(sourceId);
        }
        if (!isOpen) {
            // Reset state when closing
            setTables([]);
            setSelectedTable(null);
            setTableData([]);
            setTableRowCount(0);
            setCurrentPage(1);
            setError(null);
            setSearchQuery('');
            setEditingColumn(null);
            setEditError(null);
            setDeleteColumnTarget(null);
        }
    }, [isOpen, sourceId]);

    /**
     * Load preview limit from config.
     */
    useEffect(() => {
        GetConfig().then(config => {
            if (config.maxPreviewRows) {
                setPreviewLimit(config.maxPreviewRows);
            }
        }).catch(() => {});

        const unsub = EventsOn('config-updated', () => {
            GetConfig().then(config => {
                if (config.maxPreviewRows) {
                    setPreviewLimit(config.maxPreviewRows);
                }
            }).catch(() => {});
        });
        return () => { if (unsub) unsub(); };
    }, []);

    /**
     * Focus edit input when editing starts.
     */
    useEffect(() => {
        if (editingColumn && editInputRef.current) {
            editInputRef.current.focus();
            const len = editingColumn.newName.length;
            editInputRef.current.setSelectionRange(len, len);
        }
    }, [editingColumn?.columnName]);

    /**
     * Load table list from backend.
     * Requirement 8.2
     */
    const loadTables = async (id: string) => {
        setIsLoadingTables(true);
        setError(null);
        try {
            const tableNames = await GetDataSourceTables(id);
            setTables(tableNames || []);
        } catch (err) {
            console.error('Failed to load tables:', err);
            setError(t('unable_load_tables'));
            setTables([]);
        } finally {
            setIsLoadingTables(false);
        }
    };

    /**
     * Load table data and row count when a table is selected.
     * Requirements 8.3, 8.4, 8.6
     */
    const loadTableData = async (tableName: string) => {
        if (!sourceId) return;
        setIsLoadingData(true);
        setError(null);
        try {
            const [data, rowCount] = await Promise.all([
                GetDataSourceTableData(sourceId, tableName),
                GetDataSourceTableCount(sourceId, tableName),
            ]);
            setTableData(data || []);
            setTableRowCount(rowCount || 0);
            setCurrentPage(1);
        } catch (err) {
            console.error('Failed to load table data:', err);
            setError(t('unable_load_table_data', tableName));
            setTableData([]);
            setTableRowCount(0);
        } finally {
            setIsLoadingData(false);
        }
    };

    /**
     * Handle table selection.
     * Requirement 8.3
     */
    const handleTableSelect = useCallback(
        (tableName: string) => {
            setSelectedTable(tableName);
            setSearchQuery('');
            loadTableData(tableName);
        },
        [sourceId]
    );

    /**
     * Handle going back to table list.
     */
    const handleBackToTables = useCallback(() => {
        setSelectedTable(null);
        setTableData([]);
        setTableRowCount(0);
        setCurrentPage(1);
        setSearchQuery('');
        setError(null);
        setEditingColumn(null);
        setEditError(null);
    }, []);

    /**
     * Column rename: double-click to start editing.
     */
    const handleColumnDoubleClick = useCallback((columnName: string) => {
        setEditingColumn({ columnName, newName: columnName });
        setEditError(null);
    }, []);

    /**
     * Validate column name.
     */
    const validateColumnName = useCallback((newName: string, originalName: string): string | null => {
        if (!newName.trim()) return t('column_name_empty');
        const invalidChars = [' ', "'", '"', ';', '--', '/*', '*/', '\t', '\n', '\r'];
        for (const char of invalidChars) {
            if (newName.includes(char)) return t('column_name_invalid_char').replace('{char}', char === ' ' ? t('space') : char);
        }
        if (/^[0-9]/.test(newName)) return t('column_name_starts_with_number');
        if (tableData.length > 0) {
            const cols = Object.keys(tableData[0]);
            if (cols.some(c => c === newName && c !== originalName)) return t('column_name_duplicate').replace('{name}', newName);
        }
        return null;
    }, [tableData, t]);

    /**
     * Save column rename.
     */
    const handleColumnSave = useCallback(async () => {
        if (!editingColumn || !sourceId || !selectedTable) return;
        const { columnName, newName } = editingColumn;
        if (newName === columnName) {
            setEditingColumn(null);
            setEditError(null);
            return;
        }
        const err = validateColumnName(newName, columnName);
        if (err) { setEditError(err); return; }

        setSavingColumn(true);
        try {
            await RenameColumn(sourceId, selectedTable, columnName, newName);
            const data = await GetDataSourceTableData(sourceId, selectedTable);
            setTableData(data || []);
            EventsEmit('column-renamed', { dataSourceId: sourceId, tableName: selectedTable, oldColumnName: columnName, newColumnName: newName });
            setEditingColumn(null);
            setEditError(null);
        } catch (e) {
            console.error('Failed to rename column:', e);
            setEditError(t('rename_failed', String(e)));
        } finally {
            setSavingColumn(false);
        }
    }, [editingColumn, sourceId, selectedTable, validateColumnName]);

    /**
     * Cancel column editing.
     */
    const handleColumnCancel = useCallback(() => {
        setEditingColumn(null);
        setEditError(null);
    }, []);

    /**
     * Handle key down in column edit input.
     */
    const handleEditKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'Enter') { e.preventDefault(); handleColumnSave(); }
        else if (e.key === 'Escape') { handleColumnCancel(); }
        else { e.stopPropagation(); }
    }, [handleColumnSave, handleColumnCancel]);

    /**
     * Start column delete flow.
     */
    const handleDeleteColumn = useCallback((columnName: string) => {
        let isLastColumn = false;
        if (tableData.length > 0) {
            isLastColumn = Object.keys(tableData[0]).length === 1;
        }
        const isLastTable = tables.length === 1;
        setDeleteColumnTarget({ columnName, isLastColumn, isLastTable });
    }, [tableData, tables]);

    /**
     * Confirm column delete.
     */
    const confirmDeleteColumn = useCallback(async () => {
        if (!deleteColumnTarget || !sourceId || !selectedTable) return;
        const { columnName, isLastColumn, isLastTable } = deleteColumnTarget;
        try {
            if (isLastColumn && isLastTable) {
                await DeleteDataSource(sourceId);
                EventsEmit('data-source-deleted', sourceId);
                onClose();
            } else if (isLastColumn) {
                await DeleteTable(sourceId, selectedTable);
                await loadTables(sourceId);
                setSelectedTable(null);
                setTableData([]);
            } else {
                await DeleteColumn(sourceId, selectedTable, columnName);
                const data = await GetDataSourceTableData(sourceId, selectedTable);
                setTableData(data || []);
                EventsEmit('column-deleted', { dataSourceId: sourceId, tableName: selectedTable, columnName });
            }
            setDeleteColumnTarget(null);
        } catch (e) {
            console.error('Failed to delete column:', e);
            setError(t('delete_failed_msg', String(e)));
            setDeleteColumnTarget(null);
        }
    }, [deleteColumnTarget, sourceId, selectedTable, onClose]);

    /**
     * Handle page change.
     * Requirement 8.5
     */
    const handlePageChange = useCallback(
        (page: number) => {
            if (page >= 1 && page <= totalPages) {
                setCurrentPage(page);
            }
        },
        [totalPages]
    );

    /**
     * Handle Escape key to close the data browser.
     * If editing a column, Escape cancels the edit instead.
     * Requirement 7.6, 11.5
     */
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape' && isOpen) {
                if (editingColumn) {
                    // Cancel column editing first
                    setEditingColumn(null);
                    setEditError(null);
                } else {
                    onClose();
                }
            }
        };

        if (isOpen) {
            document.addEventListener('keydown', handleKeyDown);
            return () => document.removeEventListener('keydown', handleKeyDown);
        }
    }, [isOpen, onClose, editingColumn]);

    /**
     * Handle resize drag start on the right edge.
     * Requirement 7.10
     */
    const handleResizeMouseDown = useCallback(
        (e: React.MouseEvent) => {
            e.preventDefault();
            e.stopPropagation();
            setIsResizing(true);
            resizeStartX.current = e.clientX;
            resizeStartWidth.current = width;

            // Prevent text selection during drag
            document.body.style.userSelect = 'none';
            document.body.style.cursor = 'col-resize';
        },
        [width]
    );

    /**
     * Handle resize drag movement.
     * Dragging right edge to the right increases width, to the left decreases width.
     */
    const handleResizeMouseMove = useCallback(
        (e: MouseEvent) => {
            if (!isResizing) return;

            const deltaX = e.clientX - resizeStartX.current;
            const parentWidth =
                containerRef.current?.parentElement?.clientWidth || window.innerWidth;
            const maxWidth = parentWidth * MAX_WIDTH_RATIO;
            const newWidth = Math.max(
                MIN_WIDTH,
                Math.min(maxWidth, resizeStartWidth.current + deltaX)
            );

            onWidthChange(newWidth);
        },
        [isResizing, onWidthChange]
    );

    /**
     * Handle resize drag end.
     */
    const handleResizeMouseUp = useCallback(() => {
        if (!isResizing) return;
        setIsResizing(false);
        document.body.style.userSelect = '';
        document.body.style.cursor = '';
    }, [isResizing]);

    /**
     * Set up global mouse event listeners for resize drag.
     */
    useEffect(() => {
        if (isResizing) {
            window.addEventListener('mousemove', handleResizeMouseMove);
            window.addEventListener('mouseup', handleResizeMouseUp);

            return () => {
                window.removeEventListener('mousemove', handleResizeMouseMove);
                window.removeEventListener('mouseup', handleResizeMouseUp);
            };
        }
    }, [isResizing, handleResizeMouseMove, handleResizeMouseUp]);

    /**
     * Handle backdrop click to close the data browser.
     * Requirement 7.6
     */
    const handleBackdropClick = useCallback(() => {
        if (isOpen) {
            onClose();
        }
    }, [isOpen, onClose]);

    /**
     * Render the table list view.
     * Requirement 8.2
     */
    const renderTableList = () => {
        if (isLoadingTables) {
            return (
                <div className="db-loading" data-testid="db-loading-tables">
                    <Loader2 className="db-loading-spinner" size={24} />
                    <span>{t('loading_tables')}</span>
                </div>
            );
        }

        if (filteredTables.length === 0 && tables.length > 0) {
            return (
                <div className="db-empty-state" data-testid="db-no-search-results">
                    <Search size={24} />
                    <span>{t('no_tables_match', searchQuery)}</span>
                </div>
            );
        }

        if (tables.length === 0) {
            return (
                <div className="db-empty-state" data-testid="db-no-tables">
                    <Table2 size={24} />
                    <span>{t('no_tables_in_source')}</span>
                </div>
            );
        }

        return (
            <div className="db-table-list" data-testid="db-table-list" role="list">
                {filteredTables.map((table) => (
                    <button
                        key={table}
                        className="db-table-item"
                        data-testid={`db-table-item-${table}`}
                        onClick={() => handleTableSelect(table)}
                        role="listitem"
                    >
                        <Table2 size={14} className="db-table-icon" />
                        <span className="db-table-name">{table}</span>
                        <ChevronRight size={14} className="db-table-arrow" />
                    </button>
                ))}
            </div>
        );
    };

    /**
     * Render the column and data view for a selected table.
     * Requirements 8.3, 8.4, 8.6
     */
    const renderTableDetail = () => {
        if (isLoadingData) {
            return (
                <div className="db-loading" data-testid="db-loading-data">
                    <Loader2 className="db-loading-spinner" size={24} />
                    <span>{t('loading_table_data_browser')}</span>
                </div>
            );
        }

        return (
            <>
                {/* Statistics - Requirement 8.6 */}
                <div className="db-stats" data-testid="db-stats">
                    <div className="db-stat-item">
                        <Hash size={12} />
                        <span>
                            {tableData.length < tableRowCount
                                ? t('rows_stats', tableData.length.toLocaleString(), tableRowCount.toLocaleString(), String(previewLimit))
                                : t('rows_total', tableRowCount.toLocaleString())}
                        </span>
                    </div>
                    <div className="db-stat-item">
                        <Columns3 size={12} />
                        <span>{t('columns_count', String(columns.length))}</span>
                    </div>
                </div>

                {/* Edit error message */}
                {editError && (
                    <div className="db-error" style={{ margin: '4px 16px' }} data-testid="db-edit-error">
                        <AlertCircle size={14} />
                        <span>{editError}</span>
                    </div>
                )}

                {/* Columns with rename/delete - Requirement 8.3 */}
                <div className="db-section">
                    <div className="db-section-header" data-testid="db-columns-header">
                        <Columns3 size={14} />
                        <span>{t('columns_label', String(filteredColumns.length))}</span>
                    </div>
                    <div className="db-columns-list" data-testid="db-columns-list" role="list">
                        {filteredColumns.map((col) => (
                            <div
                                key={col.name}
                                className="db-column-item"
                                data-testid={`db-column-${col.name}`}
                                role="listitem"
                            >
                                {editingColumn?.columnName === col.name ? (
                                    <div className="db-column-edit">
                                        <input
                                            ref={editInputRef}
                                            className="db-column-edit-input"
                                            value={editingColumn.newName}
                                            onChange={(e) => setEditingColumn({ ...editingColumn, newName: e.target.value })}
                                            onKeyDown={handleEditKeyDown}
                                            disabled={savingColumn}
                                            data-testid={`db-column-edit-input-${col.name}`}
                                        />
                                        <button
                                            className="db-column-edit-btn db-column-save-btn"
                                            onClick={handleColumnSave}
                                            disabled={savingColumn}
                                            title={t('save')}
                                            data-testid={`db-column-save-${col.name}`}
                                        >
                                            {savingColumn ? <Loader2 size={12} className="db-loading-spinner" /> : <Check size={12} />}
                                        </button>
                                        <button
                                            className="db-column-edit-btn db-column-cancel-btn"
                                            onClick={handleColumnCancel}
                                            disabled={savingColumn}
                                            title={t('cancel')}
                                            data-testid={`db-column-cancel-${col.name}`}
                                        >
                                            <X size={12} />
                                        </button>
                                    </div>
                                ) : (
                                    <>
                                        <span
                                            className="db-column-name"
                                            onDoubleClick={() => handleColumnDoubleClick(col.name)}
                                            title={t('double_click_to_rename')}
                                        >
                                            {col.name}
                                        </span>
                                        <span className="db-column-type" data-testid={`db-column-type-${col.name}`}>
                                            {col.type}
                                        </span>
                                        <div className="db-column-actions">
                                            <button
                                                className="db-column-action-btn"
                                                onClick={() => handleColumnDoubleClick(col.name)}
                                                title={t('rename_column')}
                                                data-testid={`db-column-rename-${col.name}`}
                                            >
                                                <Pencil size={11} />
                                            </button>
                                            <button
                                                className="db-column-action-btn db-column-delete-btn"
                                                onClick={() => handleDeleteColumn(col.name)}
                                                title={t('delete_column')}
                                                data-testid={`db-column-delete-${col.name}`}
                                            >
                                                <Trash2 size={11} />
                                            </button>
                                        </div>
                                    </>
                                )}
                            </div>
                        ))}
                    </div>
                </div>

                {/* Sample Data - Requirement 8.4 */}
                <div className="db-section db-data-section">
                    <div className="db-section-header" data-testid="db-data-header">
                        <Table2 size={14} />
                        <span>{t('sample_data')}</span>
                    </div>
                    {tableData.length > 0 ? (
                        <div className="db-data-table-wrapper" data-testid="db-data-table-wrapper">
                            <table className="db-data-table" data-testid="db-data-table">
                                <thead>
                                    <tr>
                                        {columns.map((col) => (
                                            <th key={col.name}>{col.name}</th>
                                        ))}
                                    </tr>
                                </thead>
                                <tbody>
                                    {paginatedRows.map((row, rowIdx) => (
                                        <tr key={rowIdx} data-testid={`db-data-row-${rowIdx}`}>
                                            {columns.map((col) => (
                                                <td key={col.name}>
                                                    {formatCellValue(row[col.name])}
                                                </td>
                                            ))}
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    ) : (
                        <div className="db-empty-state" data-testid="db-no-data">
                            <span>{t('no_data_available_browser')}</span>
                        </div>
                    )}

                    {/* Pagination - Requirement 8.5 */}
                    {tableData.length > ROWS_PER_PAGE && (
                        <div className="db-pagination" data-testid="db-pagination">
                            <button
                                className="db-pagination-btn"
                                data-testid="db-pagination-prev"
                                onClick={() => handlePageChange(currentPage - 1)}
                                disabled={currentPage <= 1}
                                aria-label={t('previous_page')}
                            >
                                <ChevronLeft size={14} />
                            </button>
                            <span className="db-pagination-info" data-testid="db-pagination-info">
                                {t('page_info', String(currentPage), String(totalPages))}
                            </span>
                            <button
                                className="db-pagination-btn"
                                data-testid="db-pagination-next"
                                onClick={() => handlePageChange(currentPage + 1)}
                                disabled={currentPage >= totalPages}
                                aria-label={t('next_page')}
                            >
                                <ChevronRight size={14} />
                            </button>
                        </div>
                    )}
                </div>
            </>
        );
    };

    return (
        <>
            {/* Backdrop dim/blur effect on center panel - Requirement 7.9 */}
            <div
                data-testid="data-browser-backdrop"
                className={`data-browser-backdrop ${isOpen ? 'visible' : ''}`}
                onClick={handleBackdropClick}
                aria-hidden="true"
            />

            {/* Slide-out panel - Requirements 7.1, 7.2, 7.3 */}
            <div
                ref={containerRef}
                data-testid="data-browser"
                className={`data-browser ${isOpen ? 'open' : ''}`}
                style={{ width: `${width}px` }}
                role="dialog"
                aria-label={t('data_browser_title')}
                aria-modal={isOpen}
                aria-hidden={!isOpen}
            >
                {/* Resize handle on right edge - Requirement 7.10 */}
                <div
                    data-testid="data-browser-resize-handle"
                    className={`data-browser-resize-handle ${isResizing ? 'dragging' : ''}`}
                    onMouseDown={handleResizeMouseDown}
                    role="separator"
                    aria-orientation="vertical"
                    aria-label={t('resize_data_browser')}
                />

                {/* Header with data source name and close button - Requirements 7.5, 8.1 */}
                <div className="data-browser-header" data-testid="data-browser-header">
                    <div className="data-browser-header-left">
                        {selectedTable ? (
                            <button
                                className="db-back-button"
                                data-testid="db-back-button"
                                onClick={handleBackToTables}
                                aria-label={t('back_to_table_list')}
                            >
                                <ChevronLeft size={16} />
                            </button>
                        ) : (
                            <Database
                                className="data-browser-header-icon"
                                size={16}
                            />
                        )}
                        <span className="data-browser-title" data-testid="data-browser-title">
                            {selectedTable
                                ? selectedTable
                                : sourceName || (sourceId ? t('data_browser_title') : t('no_data_source_title'))}
                        </span>
                    </div>
                    <button
                        data-testid="data-browser-close-button"
                        className="data-browser-close-button"
                        onClick={onClose}
                        aria-label={t('close_data_browser')}
                    >
                        <X size={16} />
                    </button>
                </div>

                {/* Search bar - Requirement 8.8 */}
                {sourceId && (
                    <div className="db-search-bar" data-testid="db-search-bar">
                        <Search size={14} className="db-search-icon" />
                        <input
                            type="text"
                            data-testid="db-search-input"
                            className="db-search-input"
                            placeholder={
                                selectedTable
                                    ? t('filter_columns')
                                    : t('search_tables')
                            }
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            aria-label={
                                selectedTable
                                    ? t('filter_columns')
                                    : t('search_tables')
                            }
                        />
                    </div>
                )}

                {/* Content area - Requirements 8.2, 8.3, 8.4, 8.5, 8.6, 8.7, 8.8 */}
                <div
                    className="data-browser-content"
                    data-testid="data-browser-content"
                >
                    {/* Error state - Requirement 8.7 */}
                    {error && (
                        <div className="db-error" data-testid="db-error">
                            <AlertCircle size={20} />
                            <span>{error}</span>
                        </div>
                    )}

                    {!error && sourceId ? (
                        selectedTable ? renderTableDetail() : renderTableList()
                    ) : !error ? (
                        <div className="db-empty-state" data-testid="data-browser-no-source">
                            <Database size={32} className="db-empty-icon" />
                            <span>{t('no_data_source_selected')}</span>
                        </div>
                    ) : null}
                </div>
            </div>

            {/* Delete Column Confirmation Modal */}
            <DeleteColumnConfirmationModal
                isOpen={!!deleteColumnTarget}
                columnName={deleteColumnTarget?.columnName || ''}
                tableName={selectedTable || ''}
                isLastColumn={deleteColumnTarget?.isLastColumn}
                isLastTable={deleteColumnTarget?.isLastTable}
                dataSourceName={sourceName || ''}
                onClose={() => setDeleteColumnTarget(null)}
                onConfirm={confirmDeleteColumn}
            />
        </>
    );
};

/**
 * Infer column type from a sample value.
 */
export function inferColumnType(value: any): string {
    if (value === null || value === undefined) return 'unknown';
    if (typeof value === 'number') {
        return Number.isInteger(value) ? 'integer' : 'float';
    }
    if (typeof value === 'boolean') return 'boolean';
    if (typeof value === 'string') {
        // Check for date-like strings
        if (/^\d{4}-\d{2}-\d{2}/.test(value)) return 'date';
        // Check for numeric strings
        if (/^-?\d+(\.\d+)?$/.test(value)) return 'numeric';
        return 'text';
    }
    if (typeof value === 'object') return 'json';
    return 'unknown';
}

/**
 * Format a cell value for display.
 */
export function formatCellValue(value: any): string {
    if (value === null || value === undefined) return '—';
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
}

export default DataBrowser;
