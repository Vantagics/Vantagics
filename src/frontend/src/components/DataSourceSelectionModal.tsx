import React from 'react';
import { X, Database } from 'lucide-react';
import { useLanguage } from '../i18n';
import '../styles/datasource-selection-modal.css';

interface DataSourceSummary {
    id: string;
    name: string;
    type: string;
}

interface DataSourceSelectionModalProps {
    dataSources: DataSourceSummary[];
    onSelect: (dataSourceId: string) => void;
    onCancel?: () => void;
    onClose?: () => void;
    isOpen?: boolean;
}

const DataSourceSelectionModal: React.FC<DataSourceSelectionModalProps> = ({
    dataSources,
    onSelect,
    onCancel,
    onClose
}) => {
    const { t } = useLanguage();
    
    // Support both onCancel and onClose for flexibility
    const handleClose = () => {
        if (onClose) onClose();
        if (onCancel) onCancel();
    };

    return (
        <div 
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={handleClose}
        >
            <div 
                className="bg-white rounded-xl shadow-2xl max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-slate-200">
                    <div className="flex items-center gap-3">
                        <Database className="w-6 h-6 text-blue-500" />
                        <div>
                            <h3 className="text-lg font-semibold text-slate-900">
                                {t('select_data_source') || '选择要分析的数据源'}
                            </h3>
                            <p className="text-sm text-slate-600 mt-1">
                                {t('select_data_source_desc') || '请选择一个数据源开始智能分析'}
                            </p>
                        </div>
                    </div>
                    <button
                        onClick={handleClose}
                        className="text-slate-400 hover:text-slate-600 transition-colors"
                    >
                        <X size={24} />
                    </button>
                </div>

                {/* Data Source List */}
                <div className="flex-1 overflow-y-auto p-6">
                    <div className="space-y-3">
                        {dataSources.map((ds) => (
                            <button
                                key={ds.id}
                                onClick={() => onSelect(ds.id)}
                                className="w-full text-left p-4 border-2 border-slate-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-all duration-200 group"
                            >
                                <div className="flex items-center justify-between">
                                    <div className="flex-1 min-w-0">
                                        <div className="ds-name font-semibold text-slate-900 group-hover:text-blue-600 mb-1">
                                            {ds.name}
                                        </div>
                                        <div className="ds-type text-sm text-slate-600 uppercase font-medium">
                                            {ds.type}
                                        </div>
                                    </div>
                                    <div className="ml-4 text-slate-400 group-hover:text-blue-500 transition-colors">
                                        <svg 
                                            className="w-5 h-5" 
                                            fill="none" 
                                            stroke="currentColor" 
                                            viewBox="0 0 24 24"
                                        >
                                            <path 
                                                strokeLinecap="round" 
                                                strokeLinejoin="round" 
                                                strokeWidth={2} 
                                                d="M9 5l7 7-7 7" 
                                            />
                                        </svg>
                                    </div>
                                </div>
                            </button>
                        ))}
                    </div>
                </div>

                {/* Footer */}
                <div className="flex items-center justify-end p-6 border-t border-slate-200 bg-slate-50">
                    <button
                        onClick={handleClose}
                        className="cancel-button px-4 py-2 text-sm text-slate-600 hover:text-slate-900 font-medium"
                    >
                        {t('cancel') || '取消'}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default DataSourceSelectionModal;
