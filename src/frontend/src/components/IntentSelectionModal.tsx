import React from 'react';
import { X } from 'lucide-react';
import { useLanguage } from '../i18n';

interface IntentSuggestion {
    id: string;
    title: string;
    description: string;
    icon: string;
    query: string;
}

interface IntentSelectionModalProps {
    suggestions: IntentSuggestion[];
    onSelect: (suggestion: IntentSuggestion) => void;
    onCancel: () => void;
    onSkip: () => void;
}

const IntentSelectionModal: React.FC<IntentSelectionModalProps> = ({
    suggestions,
    onSelect,
    onCancel,
    onSkip
}) => {
    const { t } = useLanguage();

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white rounded-xl shadow-2xl max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-slate-200">
                    <div>
                        <h3 className="text-lg font-semibold text-slate-900">
                            {t('select_your_intent') || '请选择您的分析意图'}
                        </h3>
                        <p className="text-sm text-slate-600 mt-1">
                            {t('intent_selection_desc') || '系统理解了您的请求，请选择最符合您意图的分析方向'}
                        </p>
                    </div>
                    <button
                        onClick={onCancel}
                        className="text-slate-400 hover:text-slate-600 transition-colors"
                    >
                        <X size={24} />
                    </button>
                </div>

                {/* Suggestions List */}
                <div className="flex-1 overflow-y-auto p-6">
                    <div className="space-y-3">
                        {suggestions.map((suggestion, index) => (
                            <button
                                key={suggestion.id}
                                onClick={() => onSelect(suggestion)}
                                className="w-full text-left p-4 border-2 border-slate-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-all duration-200 group"
                            >
                                <div className="flex items-start gap-3">
                                    <span className="text-3xl flex-shrink-0 group-hover:scale-110 transition-transform">
                                        {suggestion.icon}
                                    </span>
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2 mb-1">
                                            <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-slate-100 text-slate-600 text-xs font-medium group-hover:bg-blue-100 group-hover:text-blue-600">
                                                {index + 1}
                                            </span>
                                            <h4 className="font-semibold text-slate-900 group-hover:text-blue-600">
                                                {suggestion.title}
                                            </h4>
                                        </div>
                                        <p className="text-sm text-slate-600 leading-relaxed">
                                            {suggestion.description}
                                        </p>
                                    </div>
                                </div>
                            </button>
                        ))}
                    </div>
                </div>

                {/* Footer */}
                <div className="flex items-center justify-between p-6 border-t border-slate-200 bg-slate-50">
                    <button
                        onClick={onSkip}
                        className="text-sm text-slate-600 hover:text-slate-900 underline"
                    >
                        {t('skip_and_analyze') || '跳过并直接分析'}
                    </button>
                    <button
                        onClick={onCancel}
                        className="px-4 py-2 text-sm text-slate-600 hover:text-slate-900 font-medium"
                    >
                        {t('cancel') || '取消'}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default IntentSelectionModal;
