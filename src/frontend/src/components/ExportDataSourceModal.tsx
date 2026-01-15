import React, { useState, useEffect } from 'react';
import { useLanguage } from '../i18n';
import { SelectSaveFile, ExportToCSV, ExportToSQL, ExportToMySQL, GetDataSourceTables, UpdateMySQLExportConfig, TestMySQLConnection, GetMySQLDatabases, ShowMessage } from '../../wailsjs/go/main/App';
import { main, agent } from '../../wailsjs/go/models';

interface ExportDataSourceModalProps {
    isOpen: boolean;
    sourceId: string;
    sourceName: string;
    onClose: () => void;
    dataSource?: agent.DataSource;
}

const ExportDataSourceModal: React.FC<ExportDataSourceModalProps> = ({ isOpen, sourceId, sourceName, onClose, dataSource }) => {
    const { t } = useLanguage();
    const [exportType, setExportType] = useState<'csv' | 'sql' | 'mysql'>('csv');
    const [tables, setTables] = useState<string[]>([]);
    const [selectedTables, setSelectedTables] = useState<string[]>([]);
    const [isExporting, setIsExporting] = useState(false);

    // MySQL step tracking
    const [mysqlStep, setMysqlStep] = useState<'connection' | 'database'>('connection');
    const [isConnecting, setIsConnecting] = useState(false);
    const [isConnected, setIsConnected] = useState(false);
    const [availableDatabases, setAvailableDatabases] = useState<string[]>([]);

    // MySQL Config
    const [mysqlConfig, setMysqlConfig] = useState({
        host: '',
        port: '3306',
        user: '',
        password: '',
        database: ''
    });

    useEffect(() => {
        if (isOpen && sourceId) {
            GetDataSourceTables(sourceId).then(tbls => {
                setTables(tbls || []);
                // Default: select all tables
                setSelectedTables(tbls || []);
            }).catch(console.error);

            // Reset MySQL states
            setMysqlStep('connection');
            setIsConnected(false);
            setAvailableDatabases([]);

            // Load saved MySQL export config if exists
            if (dataSource?.config?.mysql_export_config) {
                const savedConfig = dataSource.config.mysql_export_config;
                setMysqlConfig({
                    host: savedConfig.host || '',
                    port: savedConfig.port || '3306',
                    user: savedConfig.user || '',
                    password: savedConfig.password || '',
                    database: savedConfig.database || ''
                });
            } else {
                // Reset to default if no saved config
                setMysqlConfig({
                    host: '',
                    port: '3306',
                    user: '',
                    password: '',
                    database: ''
                });
            }
        }
    }, [isOpen, sourceId, dataSource]);

    if (!isOpen) return null;

    const handleTestConnection = async () => {
        if (!mysqlConfig.host || !mysqlConfig.user) {
            alert(t('please_connect_mysql'));
            return;
        }

        setIsConnecting(true);
        try {
            await TestMySQLConnection(mysqlConfig.host, mysqlConfig.port, mysqlConfig.user, mysqlConfig.password);

            // Connection successful, get databases
            const dbs = await GetMySQLDatabases(mysqlConfig.host, mysqlConfig.port, mysqlConfig.user, mysqlConfig.password);
            setAvailableDatabases(dbs || []);
            setIsConnected(true);
            setMysqlStep('database');

            // If there's a saved database, select it
            if (mysqlConfig.database && dbs?.includes(mysqlConfig.database)) {
                // Database already set
            } else if (dbs && dbs.length > 0) {
                setMysqlConfig(prev => ({ ...prev, database: dbs[0] }));
            }
        } catch (err: any) {
            console.error("Connection test failed:", err);
            const errorMessage = err?.message || err?.toString() || "Unknown error";
            alert(`Connection failed:\n\n${errorMessage}`);
            setIsConnected(false);
        } finally {
            setIsConnecting(false);
        }
    };

    const handleBackToConnection = () => {
        setMysqlStep('connection');
        setIsConnected(false);
    };

    const handleToggleTable = (tableName: string) => {
        setSelectedTables(prev =>
            prev.includes(tableName)
                ? prev.filter(t => t !== tableName)
                : [...prev, tableName]
        );
    };

    const handleSelectAll = () => {
        setSelectedTables(tables);
    };

    const handleDeselectAll = () => {
        setSelectedTables([]);
    };

    const handleExport = async () => {
        if (selectedTables.length === 0) {
            ShowMessage("warning", t('export'), t('please_select_table'));
            return;
        }

        setIsExporting(true);
        let destination = '';

        // Helper for string formatting
        const formatMsg = (key: string, ...args: string[]) => {
            let msg = t(key);
            args.forEach((arg, i) => {
                msg = msg.replace(`{${i}}`, arg);
            });
            return msg;
        };

        try {
            if (exportType === 'csv') {
                const fileName = selectedTables.length === 1 ? `${selectedTables[0]}.csv` : `${sourceName}.csv`;
                const path = await SelectSaveFile(fileName, "*.csv");
                if (path) {
                    destination = path;
                    await ExportToCSV(sourceId, selectedTables, path);
                    ShowMessage("info", "Export Success", formatMsg('export_success', sourceName, destination));
                    onClose();
                } else {
                    setIsExporting(false); // Cancelled
                    return;
                }
            } else if (exportType === 'sql') {
                const fileName = selectedTables.length === 1 ? `${selectedTables[0]}.sql` : `${sourceName}.sql`;
                const path = await SelectSaveFile(fileName, "*.sql");
                if (path) {
                    destination = path;
                    await ExportToSQL(sourceId, selectedTables, path);
                    ShowMessage("info", "Export Success", formatMsg('export_success', sourceName, destination));
                    onClose();
                } else {
                    setIsExporting(false); // Cancelled
                    return;
                }
            } else if (exportType === 'mysql') {
                if (!isConnected) {
                    ShowMessage("warning", t('connection'), t('please_connect_mysql'));
                    setIsExporting(false);
                    return;
                }
                if (!mysqlConfig.database) {
                    ShowMessage("warning", t('database'), t('please_select_database'));
                    setIsExporting(false);
                    return;
                }

                destination = `${mysqlConfig.host}:${mysqlConfig.port}/${mysqlConfig.database}`;

                // Check if source and target are the same
                if (dataSource && (dataSource.type === 'mysql' || dataSource.type === 'doris')) {
                    const srcHost = (dataSource.config.host || 'localhost').trim().toLowerCase();
                    const srcPort = (dataSource.config.port || '3306').trim();
                    const srcDb = (dataSource.config.database || '').trim().toLowerCase();

                    const dstHost = (mysqlConfig.host || 'localhost').trim().toLowerCase();
                    const dstPort = (mysqlConfig.port || '3306').trim();
                    const dstDb = (mysqlConfig.database || '').trim().toLowerCase();

                    const normalize = (h: string) => (h === '127.0.0.1' || h === '::1') ? 'localhost' : h;

                    if (normalize(srcHost) === normalize(dstHost) && srcPort === dstPort && srcDb === dstDb) {
                        ShowMessage("error", "Export Error", t('err_same_source_target'));
                        setIsExporting(false);
                        return;
                    }
                }

                await ExportToMySQL(sourceId, selectedTables, mysqlConfig.host, mysqlConfig.port, mysqlConfig.user, mysqlConfig.password, mysqlConfig.database);

                // Save MySQL export config for future use
                try {
                    await UpdateMySQLExportConfig(sourceId, mysqlConfig.host, mysqlConfig.port, mysqlConfig.user, mysqlConfig.password, mysqlConfig.database);
                } catch (saveErr) {
                    console.error("Failed to save MySQL config:", saveErr);
                    // Don't fail the export if config save fails
                }

                ShowMessage("info", "Export Success", formatMsg('export_success', sourceName, destination));
                onClose();
            }
        } catch (err: any) {
            console.error("Export error:", err);
            const errorMessage = err?.message || err?.toString() || "Unknown error occurred";
            ShowMessage("error", "Export Failed", formatMsg('export_failed', sourceName, destination || 'unknown', errorMessage));
        } finally {
            setIsExporting(false);
        }
    };

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[500px] rounded-xl shadow-2xl overflow-hidden text-slate-900 p-6 flex flex-col max-h-[90vh]">
                <h3 className="text-lg font-bold text-slate-800 mb-4">{t('export_data')}</h3>
                
                <div className="space-y-4 overflow-y-auto flex-1 p-1">
                    <div>
                        <div className="flex items-center justify-between mb-2">
                            <label className="block text-sm font-medium text-slate-700">{t('select_tables')}</label>
                            <div className="flex gap-2">
                                <button
                                    type="button"
                                    onClick={handleSelectAll}
                                    className="text-xs text-blue-600 hover:text-blue-800"
                                >
                                    {t('select_all')}
                                </button>
                                <span className="text-xs text-slate-400">|</span>
                                <button
                                    type="button"
                                    onClick={handleDeselectAll}
                                    className="text-xs text-blue-600 hover:text-blue-800"
                                >
                                    {t('deselect_all')}
                                </button>
                            </div>
                        </div>
                        <div className="border border-slate-300 rounded-md p-3 max-h-40 overflow-y-auto bg-slate-50">
                            {tables.length === 0 ? (
                                <p className="text-sm text-slate-500">{t('no_tables_found')}</p>
                            ) : (
                                <div className="space-y-2">
                                    {tables.map(table => (
                                        <label key={table} className="flex items-center gap-2 cursor-pointer hover:bg-slate-100 p-1 rounded">
                                            <input
                                                type="checkbox"
                                                checked={selectedTables.includes(table)}
                                                onChange={() => handleToggleTable(table)}
                                                className="w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                                            />
                                            <span className="text-sm text-slate-700">{table}</span>
                                        </label>
                                    ))}
                                </div>
                            )}
                        </div>
                        <p className="text-xs text-slate-500 mt-1">
                            {t('tables_selected').replace('{0}', selectedTables.length.toString()).replace('{1}', tables.length.toString())}
                        </p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">{t('export_format')}</label>
                        <div className="flex gap-4">
                            <label className="flex items-center gap-2 text-sm text-slate-700 cursor-pointer">
                                <input 
                                    type="radio" 
                                    name="exportType" 
                                    checked={exportType === 'csv'} 
                                    onChange={() => setExportType('csv')}
                                /> CSV
                            </label>
                            <label className="flex items-center gap-2 text-sm text-slate-700 cursor-pointer">
                                <input 
                                    type="radio" 
                                    name="exportType" 
                                    checked={exportType === 'sql'} 
                                    onChange={() => setExportType('sql')}
                                /> SQL
                            </label>
                            <label className="flex items-center gap-2 text-sm text-slate-700 cursor-pointer">
                                <input 
                                    type="radio" 
                                    name="exportType" 
                                    checked={exportType === 'mysql'} 
                                    onChange={() => setExportType('mysql')}
                                /> {t('mysql_database')}
                            </label>
                        </div>
                    </div>

                    {exportType === 'mysql' && (
                        <div className="space-y-3 p-4 bg-slate-50 rounded-lg border border-slate-200">
                            {/* Step indicator */}
                            <div className="flex items-center justify-between mb-2">
                                <div className="flex items-center gap-2">
                                    <div className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium ${mysqlStep === 'connection' || !isConnected ? 'bg-blue-600 text-white' : 'bg-green-500 text-white'}`}>
                                        {isConnected ? '✓' : '1'}
                                    </div>
                                    <span className="text-xs font-medium text-slate-600">{t('connection')}</span>
                                    <div className="w-8 h-px bg-slate-300"></div>
                                    <div className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium ${mysqlStep === 'database' && isConnected ? 'bg-blue-600 text-white' : 'bg-slate-300 text-slate-500'}`}>
                                        2
                                    </div>
                                    <span className="text-xs font-medium text-slate-600">{t('select_database')}</span>
                                </div>
                                {isConnected && mysqlStep === 'database' && (
                                    <button
                                        onClick={handleBackToConnection}
                                        className="text-xs text-blue-600 hover:text-blue-800"
                                    >
                                        ← {t('back')}
                                    </button>
                                )}
                            </div>

                            {/* Step 1: Connection */}
                            {mysqlStep === 'connection' && (
                                <>
                                    <div className="grid grid-cols-3 gap-3">
                                        <div className="col-span-2">
                                            <label className="block text-xs font-medium text-slate-500 mb-1">Host *</label>
                                            <input
                                                type="text"
                                                value={mysqlConfig.host}
                                                onChange={(e) => setMysqlConfig({...mysqlConfig, host: e.target.value})}
                                                className="w-full border border-slate-300 rounded-md p-1.5 text-sm"
                                                placeholder="localhost"
                                                autoComplete="off"
                                                spellCheck={false}
                                                disabled={isConnecting}
                                            />
                                        </div>
                                        <div>
                                            <label className="block text-xs font-medium text-slate-500 mb-1">Port</label>
                                            <input
                                                type="text"
                                                value={mysqlConfig.port}
                                                onChange={(e) => setMysqlConfig({...mysqlConfig, port: e.target.value})}
                                                className="w-full border border-slate-300 rounded-md p-1.5 text-sm"
                                                placeholder="3306"
                                                autoComplete="off"
                                                spellCheck={false}
                                                disabled={isConnecting}
                                            />
                                        </div>
                                    </div>
                                    <div className="grid grid-cols-2 gap-3">
                                        <div>
                                            <label className="block text-xs font-medium text-slate-500 mb-1">{t('username')} *</label>
                                            <input
                                                type="text"
                                                value={mysqlConfig.user}
                                                onChange={(e) => setMysqlConfig({...mysqlConfig, user: e.target.value})}
                                                className="w-full border border-slate-300 rounded-md p-1.5 text-sm"
                                                placeholder="root"
                                                autoComplete="off"
                                                spellCheck={false}
                                                disabled={isConnecting}
                                            />
                                        </div>
                                        <div>
                                            <label className="block text-xs font-medium text-slate-500 mb-1">Password</label>
                                            <input
                                                type="password"
                                                value={mysqlConfig.password}
                                                onChange={(e) => setMysqlConfig({...mysqlConfig, password: e.target.value})}
                                                className="w-full border border-slate-300 rounded-md p-1.5 text-sm"
                                                autoComplete="off"
                                                spellCheck={false}
                                                disabled={isConnecting}
                                            />
                                        </div>
                                    </div>
                                    <button
                                        onClick={handleTestConnection}
                                        disabled={isConnecting || !mysqlConfig.host || !mysqlConfig.user}
                                        className="w-full bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium py-2 px-4 rounded-md disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                                    >
                                        {isConnecting ? (
                                            <>
                                                <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
                                                {t('connecting')}
                                            </>
                                        ) : (
                                            t('connect_to_server')
                                        )}
                                    </button>
                                </>
                            )}

                            {/* Step 2: Database selection */}
                            {mysqlStep === 'database' && isConnected && (
                                <div>
                                    <label className="block text-xs font-medium text-slate-500 mb-1">{t('select_database')} *</label>
                                    <div className="flex flex-col gap-2">
                                        <select
                                            value={availableDatabases.includes(mysqlConfig.database) ? mysqlConfig.database : ''}
                                            onChange={(e) => {
                                                if (e.target.value) {
                                                    setMysqlConfig({...mysqlConfig, database: e.target.value});
                                                }
                                            }}
                                            className="w-full border border-slate-300 rounded-md p-1.5 text-sm"
                                        >
                                            <option value="">{t('select_existing_database')}</option>
                                            {availableDatabases.map(db => (
                                                <option key={db} value={db}>{db}</option>
                                            ))}
                                        </select>
                                        
                                        <div className="relative">
                                            <div className="absolute inset-0 flex items-center">
                                                <span className="w-full border-t border-slate-200" />
                                            </div>
                                            <div className="relative flex justify-center text-xs uppercase">
                                                <span className="bg-slate-50 px-2 text-slate-500">{t('or_create_new')}</span>
                                            </div>
                                        </div>

                                        <input
                                            type="text"
                                            value={mysqlConfig.database}
                                            onChange={(e) => setMysqlConfig({...mysqlConfig, database: e.target.value})}
                                            className="w-full border border-slate-300 rounded-md p-1.5 text-sm"
                                            placeholder={t('enter_new_database_name')}
                                            spellCheck={false}
                                        />
                                    </div>
                                    
                                    <p className="text-xs text-slate-500 mt-1">
                                        {t('connected_to').replace('{0}', mysqlConfig.host).replace('{1}', mysqlConfig.port)}
                                    </p>
                                </div>
                            )}
                        </div>
                    )}
                </div>

                <div className="mt-6 flex justify-end gap-3 pt-4 border-t border-slate-100">
                    <button 
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-md transition-colors"
                    >
                        {t('cancel')}
                    </button>
                    <button 
                        onClick={handleExport}
                        disabled={isExporting}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md shadow-sm transition-colors disabled:opacity-50 flex items-center gap-2"
                    >
                        {isExporting ? (
                            <>
                                <span className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
                                {t('exporting')}
                            </>
                        ) : t('export')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default ExportDataSourceModal;
