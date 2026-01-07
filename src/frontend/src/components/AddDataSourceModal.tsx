import React, { useState } from 'react';
import { AddDataSource, SelectExcelFile, SelectCSVFile } from '../../wailsjs/go/main/App';
import { useLanguage } from '../i18n';

interface AddDataSourceModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSuccess: () => void;
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
    const [isImporting, setIsImporting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    if (!isOpen) return null;

    const handleBrowseFile = async () => {
        try {
            let path = '';
            if (driverType === 'excel') {
                path = await SelectExcelFile();
            } else if (driverType === 'csv') {
                path = await SelectCSVFile();
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

    const handleImport = async () => {
        if (!name) {
            setError('Please enter a data source name');
            return;
        }
        if ((driverType === 'excel' || driverType === 'csv') && !config.filePath) {
            setError(driverType === 'excel' ? 'Please select an Excel file' : 'Please select a CSV file');
            return;
        }

        setIsImporting(true);
        setError(null);
        try {
            await AddDataSource(name, driverType, config);
            onSuccess();
            onClose();
            // Reset form
            setName('');
            setDriverType('excel');
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
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[500px] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900">
                <div className="p-6 border-b border-slate-200">
                    <h2 className="text-xl font-bold text-slate-800">{t('add_data_source')}</h2>
                </div>

                <div className="p-6 space-y-4">
                    {error && (
                        <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-sm rounded-md">
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
                            <option value="mysql">MySQL</option>
                            <option value="postgresql">PostgreSQL</option>
                            <option value="doris">Doris</option>
                        </select>
                    </div>

                    {driverType === 'excel' || driverType === 'csv' ? (
                        <div>
                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('file_path')}</label>
                            <div className="flex gap-2">
                                <input
                                    type="text"
                                    value={config.filePath}
                                    readOnly
                                    className="flex-1 border border-slate-300 rounded-md p-2 text-sm bg-slate-50 outline-none"
                                    placeholder={driverType === 'excel' ? "Select excel file..." : "Select csv file..."}
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
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">{t('database')}</label>
                                <input
                                    type="text"
                                    value={config.database}
                                    onChange={(e) => setConfig({ ...config, database: e.target.value })}
                                    className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">{t('user')}</label>
                                <input
                                    type="text"
                                    value={config.user}
                                    onChange={(e) => setConfig({ ...config, user: e.target.value })}
                                    className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                />
                            </div>
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
    );
};

export default AddDataSourceModal;
