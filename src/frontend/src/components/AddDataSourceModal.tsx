import React, { useState } from 'react';
import { CheckCircle } from 'lucide-react';
import { AddDataSource, SelectExcelFile, SelectCSVFile, SelectJSONFile, SelectFolder, TestMySQLConnection, GetMySQLDatabases } from '../../wailsjs/go/main/App';
import { useLanguage } from '../i18n';

interface AddDataSourceModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSuccess: (dataSource: any) => void;
}

const AddDataSourceModal: React.FC<AddDataSourceModalProps> = ({ isOpen, onClose, onSuccess }) => {
    const { t } = useLanguage();
    const [name, setName] = useState('');
    const [driverType, setDriverType] = useState('excel');
    const [config, setConfig] = useState<Record<string, string>>({
        filePath: '',
        host: 'localhost',
        port: '',
        user: '',
        database: ''
    });
    const [isStoreLocally, setIsStoreLocally] = useState(false);
    const [shouldOptimize, setShouldOptimize] = useState(true); // Default to true
    const [isImporting, setIsImporting] = useState(false);
    const [isTesting, setIsTesting] = useState(false);
    const [availableDatabases, setAvailableDatabases] = useState<string[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [showToast, setShowToast] = useState(false);

    if (!isOpen) return null;

    const handleBrowseFile = async () => {
        try {
            let path = '';
            if (driverType === 'excel') {
                path = await SelectExcelFile();
            } else if (driverType === 'csv') {
                path = await SelectFolder("Select CSV Directory");
            } else if (driverType === 'json') {
                path = await SelectJSONFile();
            }

            if (path) {
                setConfig(prev => ({ ...prev, filePath: path }));
                // Auto-fill name if empty
                if (!name) {
                    const fileName = path.split(/[/\\]/).pop();
                    if (fileName) {
                        setName(fileName.replace(/\.[^/.]+$/, ""));
                    }
                }
            }
        } catch (err) {
            console.error('Failed to select file:', err);
        }
    };

    const handleTestConnection = async () => {
        if (!config.host || !config.user) {
            setError('Please provide Host and User for connection test.');
            return;
        }
        setIsTesting(true);
        setError(null);
        setAvailableDatabases([]);
        try {
            await TestMySQLConnection(config.host, config.port, config.user, config.password || '');

            // Try to fetch databases
            try {
                const dbs = await GetMySQLDatabases(config.host, config.port, config.user, config.password || '');
                if (dbs && dbs.length > 0) {
                    setAvailableDatabases(dbs);
                }
            } catch (e) {
                console.warn("Could not fetch databases:", e);
            }

            // Show toast notification instead of alert
            setShowToast(true);
            setTimeout(() => setShowToast(false), 3000);
        } catch (err: any) {
            setError(t('test_connection_failed') || 'Connection failed: ' + err);
        } finally {
            setIsTesting(false);
        }
    };

    const handleImport = async () => {
        if (!name) {
            setError('Please enter a data source name');
            return;
        }
        if ((driverType === 'excel' || driverType === 'csv' || driverType === 'json') && !config.filePath) {
            setError(driverType === 'excel' ? 'Please select an Excel file' : driverType === 'json' ? 'Please select a JSON file' : 'Please select a CSV file');
            return;
        }

        setIsImporting(true);
        setError(null);
        try {
            const newDataSource = await AddDataSource(name, driverType, {
                ...config,
                storeLocally: isStoreLocally.toString()
            });

            // Pass the data source and optimization flag to parent
            if (shouldOptimize && newDataSource?.config?.db_path && !newDataSource?.config?.optimized) {
                // Will trigger optimization in parent component
                onSuccess(newDataSource);
            } else {
                // Just refresh the list
                onSuccess(null);
            }

            onClose();
            // Reset form
            setName('');
            setDriverType('excel');
            setIsStoreLocally(false);
            setShouldOptimize(true);
            setConfig({
                filePath: '',
                host: 'localhost',
                port: '',
                user: '',
                database: ''
            });
        } catch (err) {
            setError(String(err));
        } finally {
            setIsImporting(false);
        }
    };

    return (
        <>
            {/* Toast Notification */}
            {showToast && (
                <div className="fixed top-4 right-4 z-[10001] animate-slide-in-right">
                    <div className="bg-green-500 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2">
                        <CheckCircle className="w-5 h-5" />
                        <span className="font-medium">{t('test_connection_success') || 'Connection successful!'}</span>
                    </div>
                </div>
            )}

            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
                <div className="bg-white w-[500px] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900">
                    <div className="p-6 border-b border-slate-200">
                        <h2 className="text-xl font-bold text-slate-800">{t('add_data_source')}</h2>
                    </div>

                    <div className="p-6 space-y-4">
                        {error && (
                            <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-sm rounded-md break-words">
                                {error}
                            </div>
                        )}

                        <div>
                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('source_name')}</label>
                            <input
                                type="text"
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                placeholder="e.g. Sales 2023"
                                spellCheck={false}
                                autoCorrect="off"
                                autoComplete="off"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('driver_type')}</label>
                            <select
                                value={driverType}
                                onChange={(e) => {
                                    setDriverType(e.target.value);
                                    setConfig(prev => ({ ...prev, filePath: '' })); // Reset file path on change
                                }}
                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                            >
                                <option value="excel">Excel</option>
                                <option value="csv">CSV</option>
                                <option value="json">JSON</option>
                                <option value="mysql">MySQL</option>
                                <option value="postgresql">PostgreSQL</option>
                                <option value="doris">Doris</option>
                            </select>
                        </div>

                        {driverType === 'excel' || driverType === 'csv' || driverType === 'json' ? (
                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">{t('file_path')}</label>
                                <div className="flex gap-2">
                                    <input
                                        type="text"
                                        value={config.filePath}
                                        readOnly
                                        className="flex-1 border border-slate-300 rounded-md p-2 text-sm bg-slate-50 outline-none"
                                        placeholder={driverType === 'excel' ? "Select excel file..." : driverType === 'json' ? "Select JSON file..." : "Select csv folder..."}
                                    />
                                    <button
                                        onClick={handleBrowseFile}
                                        className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 border border-slate-300 rounded-md transition-colors"
                                    >
                                        {t('browse')}
                                    </button>
                                </div>
                            </div>
                        ) : (
                            <div className="grid grid-cols-2 gap-4">
                                <div className="col-span-2">
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('host')}</label>
                                    <input
                                        type="text"
                                        value={config.host}
                                        onChange={(e) => setConfig({ ...config, host: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('port')}</label>
                                    <input
                                        type="text"
                                        value={config.port}
                                        onChange={(e) => setConfig({ ...config, port: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder={driverType === 'mysql' ? '3306' : driverType === 'postgresql' ? '5432' : ''}
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('database')}</label>
                                    {availableDatabases.length > 0 ? (
                                        <div className="flex gap-2">
                                            <select
                                                value={config.database}
                                                onChange={(e) => setConfig({ ...config, database: e.target.value })}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            >
                                                <option value="">-- Select Database --</option>
                                                {availableDatabases.map(db => (
                                                    <option key={db} value={db}>{db}</option>
                                                ))}
                                            </select>
                                            <button
                                                onClick={() => setAvailableDatabases([])}
                                                className="px-2 text-slate-400 hover:text-slate-600"
                                                title="Switch to manual entry"
                                            >
                                                ✕
                                            </button>
                                        </div>
                                    ) : (
                                        <input
                                            type="text"
                                            value={config.database}
                                            onChange={(e) => setConfig({ ...config, database: e.target.value })}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            spellCheck={false}
                                            autoCorrect="off"
                                            autoComplete="off"
                                            placeholder={isTesting ? "Listing databases..." : ""}
                                        />
                                    )}
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('user')}</label>
                                    <input
                                        type="text"
                                        value={config.user}
                                        onChange={(e) => setConfig({ ...config, user: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('password') || 'Password'}</label>
                                    <input
                                        type="password"
                                        value={config.password || ''}
                                        onChange={(e) => setConfig({ ...config, password: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div className="col-span-2 flex items-center justify-between mt-2">
                                    <div className="flex items-center gap-2">
                                        <input
                                            type="checkbox"
                                            id="storeLocally"
                                            checked={isStoreLocally}
                                            onChange={(e) => setIsStoreLocally(e.target.checked)}
                                            className="rounded border-slate-300 text-blue-600 shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                                        />
                                        <label htmlFor="storeLocally" className="text-sm text-slate-700 select-none cursor-pointer">
                                            {t('store_locally')}
                                        </label>
                                    </div>
                                    <button
                                        onClick={handleTestConnection}
                                        disabled={isTesting}
                                        className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${isTesting ? 'bg-slate-100 text-slate-400' : 'bg-slate-100 text-slate-700 hover:bg-slate-200'}`}
                                    >
                                        {isTesting ? 'Testing...' : (t('test_connection') || 'Test Connection')}
                                    </button>
                                </div>
                            </div>
                        )}

                        {/* Optimize checkbox - shown for all local databases */}
                        {(driverType === 'excel' || driverType === 'csv' || driverType === 'json' || isStoreLocally) && (
                            <div className="flex items-center gap-2 p-3 bg-amber-50 border border-amber-200 rounded-lg">
                                <input
                                    type="checkbox"
                                    id="shouldOptimize"
                                    checked={shouldOptimize}
                                    onChange={(e) => setShouldOptimize(e.target.checked)}
                                    className="rounded border-amber-300 text-amber-600 shadow-sm focus:border-amber-300 focus:ring focus:ring-amber-200 focus:ring-opacity-50"
                                />
                                <label htmlFor="shouldOptimize" className="text-sm text-slate-700 select-none cursor-pointer flex-1">
                                    <span className="font-medium">{t('optimize_after_import') || '导入后优化数据'}</span>
                                    <span className="block text-xs text-slate-500 mt-0.5">
                                        {t('optimize_description') || '自动创建索引以提升查询性能'}
                                    </span>
                                </label>
                            </div>
                        )}
                    </div>

                    <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-3">
                        <button
                            onClick={onClose}
                            disabled={isImporting}
                            className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-200 rounded-md"
                        >
                            {t('cancel')}
                        </button>
                        <button
                            onClick={handleImport}
                            disabled={isImporting}
                            className={`px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md shadow-sm flex items-center gap-2 ${isImporting ? 'opacity-70 cursor-not-allowed' : ''}`}
                        >
                            {isImporting ? (
                                <>
                                    <span className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
                                    {t('importing')}
                                </>
                            ) : (
                                t('import')
                            )}
                        </button>
                    </div>
                </div>
            </div>
        </>
    );
};

export default AddDataSourceModal;
