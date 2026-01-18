import React, { useState, useEffect } from 'react';
import { X } from 'lucide-react';

interface DataSource {
    id: string;
    name: string;
    type: string;
}

interface DataSourceSelectionModalProps {
    isOpen: boolean;
    dataSources: DataSource[];
    onClose: () => void;
    onSelect: (dataSourceId: string) => void;
}

const DataSourceSelectionModal: React.FC<DataSourceSelectionModalProps> = ({
    isOpen,
    dataSources,
    onClose,
    onSelect
}) => {
    const [selectedId, setSelectedId] = useState<string>('');

    useEffect(() => {
        if (isOpen && dataSources.length > 0) {
            setSelectedId(dataSources[0].id);
        }
    }, [isOpen, dataSources]);

    if (!isOpen) return null;

    const handleConfirm = () => {
        if (selectedId) {
            onSelect(selectedId);
        }
    };

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[100000]">
            <div className="bg-white rounded-lg shadow-xl w-full max-w-md">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200">
                    <h2 className="text-lg font-semibold text-slate-800">选择目标数据源</h2>
                    <button
                        onClick={onClose}
                        className="text-slate-400 hover:text-slate-600 transition-colors"
                    >
                        <X className="w-5 h-5" />
                    </button>
                </div>

                {/* Body */}
                <div className="px-6 py-4">
                    <p className="text-sm text-slate-600 mb-4">
                        选择要导入分析过程的目标数据源
                    </p>

                    <div className="space-y-2 max-h-64 overflow-y-auto">
                        {dataSources.map((ds) => (
                            <label
                                key={ds.id}
                                className={`block p-3 border rounded-lg cursor-pointer transition-colors ${selectedId === ds.id
                                        ? 'border-blue-500 bg-blue-50'
                                        : 'border-slate-200 hover:border-blue-300 hover:bg-slate-50'
                                    }`}
                            >
                                <input
                                    type="radio"
                                    name="dataSource"
                                    value={ds.id}
                                    checked={selectedId === ds.id}
                                    onChange={(e) => setSelectedId(e.target.value)}
                                    className="sr-only"
                                />
                                <div className="flex items-center gap-3">
                                    <span className={`flex-shrink-0 w-3 h-3 rounded-full ${ds.type === 'excel' ? 'bg-green-500' :
                                            ['mysql', 'postgresql', 'doris'].includes(ds.type) ? 'bg-blue-500' :
                                                'bg-gray-400'
                                        }`}></span>
                                    <div className="flex-1">
                                        <div className="font-medium text-slate-800">{ds.name}</div>
                                        <div className="text-xs text-slate-500">{ds.type.toUpperCase()}</div>
                                    </div>
                                    {selectedId === ds.id && (
                                        <svg className="w-5 h-5 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
                                            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                                        </svg>
                                    )}
                                </div>
                            </label>
                        ))}
                    </div>
                </div>

                {/* Footer */}
                <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-slate-200 bg-slate-50">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 transition-colors"
                    >
                        取消
                    </button>
                    <button
                        onClick={handleConfirm}
                        disabled={!selectedId}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:bg-slate-300 disabled:cursor-not-allowed transition-colors"
                    >
                        继续导入
                    </button>
                </div>
            </div>
        </div>
    );
};

export default DataSourceSelectionModal;
