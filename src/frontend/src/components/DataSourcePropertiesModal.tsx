import React, { useState, useEffect } from 'react';
import { useLanguage } from '../i18n';
import { agent } from '../../wailsjs/go/models';
import { X } from 'lucide-react';

interface DataSourcePropertiesModalProps {
    isOpen: boolean;
    dataSource: agent.DataSource | null;
    onClose: () => void;
}

const DataSourcePropertiesModal: React.FC<DataSourcePropertiesModalProps> = ({ isOpen, dataSource, onClose }) => {
    const { t } = useLanguage();
    const [localDataSource, setLocalDataSource] = useState<agent.DataSource | null>(dataSource);

    // Update local state when dataSource prop changes
    useEffect(() => {
        setLocalDataSource(dataSource);
    }, [dataSource]);

    if (!isOpen || !localDataSource) return null;

    const config = localDataSource.config as any || {};
    const isRemote = ['mysql', 'postgresql', 'doris'].includes(localDataSource.type);

    return (
        <div className="fixed inset-0 z-[10000] flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[500px] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900">
                <div className="p-4 border-b border-slate-200 flex justify-between items-center bg-slate-50">
                    <h2 className="text-lg font-bold text-slate-800">{t('properties')}</h2>
                    <button onClick={onClose} className="text-slate-500 hover:text-slate-700">
                        <X className="w-5 h-5" />
                    </button>
                </div>

                <div className="p-6 space-y-4 max-h-[70vh] overflow-y-auto">
                    <div className="grid grid-cols-3 gap-4">
                        <div className="text-sm font-medium text-slate-500 text-right">{t('source_name')}:</div>
                        <div className="col-span-2 text-sm text-slate-800 font-medium">{localDataSource.name}</div>
                    </div>

                    <div className="grid grid-cols-3 gap-4">
                        <div className="text-sm font-medium text-slate-500 text-right">{t('driver_type')}:</div>
                        <div className="col-span-2 text-sm text-slate-800 capitalize">{localDataSource.type}</div>
                    </div>

                    <div className="grid grid-cols-3 gap-4">
                        <div className="text-sm font-medium text-slate-500 text-right">{t('created_at')}:</div>
                        <div className="col-span-2 text-sm text-slate-800">{new Date(localDataSource.created_at).toLocaleString()}</div>
                    </div>

                    {localDataSource.analysis?.summary && (
                        <>
                            <div className="border-t border-slate-100 my-4"></div>
                            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                                <div className="flex items-start gap-2">
                                    <div className="text-sm font-bold text-blue-900 mb-2">{t('data_summary')}</div>
                                </div>
                                <div className="text-sm text-slate-700 leading-relaxed whitespace-pre-wrap">
                                    {localDataSource.analysis.summary}
                                </div>
                            </div>
                        </>
                    )}

                    {localDataSource.analysis?.schema && localDataSource.analysis.schema.length > 0 && (
                        <>
                            <div className="border-t border-slate-100 my-4"></div>
                            <div className="bg-slate-50 border border-slate-200 rounded-lg p-4">
                                <div className="text-sm font-bold text-slate-900 mb-3">{t('table_schema')}</div>
                                <div className="space-y-3 max-h-60 overflow-y-auto">
                                    {localDataSource.analysis.schema.map((table, tableIdx) => (
                                        <div key={tableIdx} className="bg-white border border-slate-200 rounded-md p-3">
                                            <div className="text-xs font-bold text-slate-700 mb-2">{table.table_name}</div>
                                            <div className="flex flex-wrap gap-1.5">
                                                {table.columns && table.columns.map((col, colIdx) => (
                                                    <span
                                                        key={colIdx}
                                                        className="text-xs px-2 py-1 bg-slate-100 text-slate-600 rounded border border-slate-200"
                                                    >
                                                        {col}
                                                    </span>
                                                ))}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </>
                    )}

                    <div className="border-t border-slate-100 my-4"></div>

                    {isRemote ? (
                        <>
                            <div className="grid grid-cols-3 gap-4">
                                <div className="text-sm font-medium text-slate-500 text-right">{t('host')}:</div>
                                <div className="col-span-2 text-sm text-slate-800">{config.host || '-'}</div>
                            </div>
                            <div className="grid grid-cols-3 gap-4">
                                <div className="text-sm font-medium text-slate-500 text-right">{t('port')}:</div>
                                <div className="col-span-2 text-sm text-slate-800">{config.port || '-'}</div>
                            </div>
                            <div className="grid grid-cols-3 gap-4">
                                <div className="text-sm font-medium text-slate-500 text-right">{t('database')}:</div>
                                <div className="col-span-2 text-sm text-slate-800">{config.database || '-'}</div>
                            </div>
                             <div className="grid grid-cols-3 gap-4">
                                <div className="text-sm font-medium text-slate-500 text-right">{t('user')}:</div>
                                <div className="col-span-2 text-sm text-slate-800">{config.user || '-'}</div>
                            </div>
                             <div className="grid grid-cols-3 gap-4">
                                <div className="text-sm font-medium text-slate-500 text-right">{t('store_locally')}:</div>
                                <div className="col-span-2 text-sm text-slate-800">
                                    {config.store_locally ? t('yes') : t('no')}
                                </div>
                            </div>
                        </>
                    ) : (
                        <div className="grid grid-cols-3 gap-4">
                            <div className="text-sm font-medium text-slate-500 text-right">{t('file_path')}:</div>
                            <div className="col-span-2 text-sm text-slate-800 break-all">{config.original_file || '-'}</div>
                        </div>
                    )}

                    {config.db_path && (
                         <div className="grid grid-cols-3 gap-4">
                            <div className="text-sm font-medium text-slate-500 text-right">{t('db_path')}:</div>
                            <div className="col-span-2 text-sm text-slate-800 break-all text-xs text-slate-400">{config.db_path}</div>
                        </div>
                    )}

                    {config.db_path && (
                        <div className="grid grid-cols-3 gap-4">
                            <div className="text-sm font-medium text-slate-500 text-right">{t('optimization_status')}:</div>
                            <div className="col-span-2">
                                {config.optimized ? (
                                    <span className="inline-flex items-center px-2 py-1 text-xs font-medium text-green-700 bg-green-100 border border-green-300 rounded">
                                        ✓ {t('optimized')}
                                    </span>
                                ) : (
                                    <span className="inline-flex items-center px-2 py-1 text-xs font-medium text-slate-600 bg-slate-100 border border-slate-300 rounded">
                                        {t('not_optimized')}
                                    </span>
                                )}
                            </div>
                        </div>
                    )}
                </div>

                <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 hover:bg-slate-50 rounded-md shadow-sm"
                    >
                        {t('close')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default DataSourcePropertiesModal;
