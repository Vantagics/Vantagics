import React from 'react';
import { X, AlertCircle, CheckCircle, AlertTriangle } from 'lucide-react';
import { useLanguage } from '../i18n';

interface ValidationIssue {
    type: string;
    table?: string;
    column?: string;
    message: string;
    severity: string; // "error", "warning"
}

interface ValidationResult {
    compatible: boolean;
    issues: ValidationIssue[];
}

interface ValidationResultsModalProps {
    isOpen: boolean;
    validationResult: ValidationResult | null;
    onClose: () => void;
    onProceed: () => void;
}

const ValidationResultsModal: React.FC<ValidationResultsModalProps> = ({
    isOpen,
    validationResult,
    onClose,
    onProceed
}) => {
    const { t } = useLanguage();

    if (!isOpen || !validationResult) return null;

    const errors = validationResult.issues.filter(i => i.severity === 'error');
    const warnings = validationResult.issues.filter(i => i.severity === 'warning');

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[100000]">
            <div className="bg-white dark:bg-[#252526] rounded-lg shadow-xl w-full max-w-2xl max-h-[80vh] flex flex-col">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-[#3c3c3c]">
                    <h2 className="text-lg font-semibold text-slate-800 dark:text-[#d4d4d4]">{t('validation_results_title')}</h2>
                    <button
                        onClick={onClose}
                        className="text-slate-400 hover:text-slate-600 transition-colors"
                    >
                        <X className="w-5 h-5" />
                    </button>
                </div>

                {/* Body */}
                <div className="flex-1 overflow-y-auto px-6 py-4">
                    {/* Overall Status */}
                    <div className={`flex items-start gap-3 p-4 rounded-lg mb-4 ${validationResult.compatible
                            ? 'bg-green-50 border border-green-200'
                            : 'bg-red-50 border border-red-200'
                        }`}>
                        {validationResult.compatible ? (
                            <CheckCircle className="w-6 h-6 text-green-600 flex-shrink-0 mt-0.5" />
                        ) : (
                            <AlertCircle className="w-6 h-6 text-red-600 flex-shrink-0 mt-0.5" />
                        )}
                        <div>
                            <div className={`font-semibold ${validationResult.compatible ? 'text-green-800' : 'text-red-800'
                                }`}>
                                {validationResult.compatible ? t('schema_compatible') : t('schema_incompatible')}
                            </div>
                            <div className={`text-sm mt-1 ${validationResult.compatible ? 'text-green-700' : 'text-red-700'
                                }`}>
                                {validationResult.compatible
                                    ? t('schema_compatible_desc')
                                    : t('schema_incompatible_desc')}
                            </div>
                        </div>
                    </div>

                    {/* Issues */}
                    {validationResult.issues.length > 0 ? (
                        <div className="space-y-4">
                            {/* Errors */}
                            {errors.length > 0 && (
                                <div>
                                    <h3 className="text-sm font-semibold text-red-800 mb-2 flex items-center gap-2">
                                        <AlertCircle className="w-4 h-4" />
                                        {t('validation_errors')} ({errors.length})
                                    </h3>
                                    <div className="space-y-2">
                                        {errors.map((issue, idx) => (
                                            <div key={idx} className="bg-red-50 border border-red-200 rounded-lg p-3">
                                                <div className="text-sm text-red-900">{issue.message}</div>
                                                {issue.table && (
                                                    <div className="text-xs text-red-700 mt-1">
                                                        {t('validation_type_label')}: {issue.type}
                                                        {issue.column && ` | ${t('validation_column_label')}: ${issue.column}`}
                                                    </div>
                                                )}
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* Warnings */}
                            {warnings.length > 0 && (
                                <div>
                                    <h3 className="text-sm font-semibold text-yellow-800 mb-2 flex items-center gap-2">
                                        <AlertTriangle className="w-4 h-4" />
                                        {t('validation_warnings')} ({warnings.length})
                                    </h3>
                                    <div className="space-y-2">
                                        {warnings.map((issue, idx) => (
                                            <div key={idx} className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
                                                <div className="text-sm text-yellow-900">{issue.message}</div>
                                                {issue.table && (
                                                    <div className="text-xs text-yellow-700 mt-1">
                                                        {t('validation_type_label')}: {issue.type}
                                                        {issue.column && ` | ${t('validation_column_label')}: ${issue.column}`}
                                                    </div>
                                                )}
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                    ) : (
                        <div className="text-center py-8 text-slate-500">
                            <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-3" />
                            <div className="font-medium">{t('validation_no_issues')}</div>
                            <div className="text-sm mt-1">{t('validation_all_present')}</div>
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-slate-200 dark:border-[#3c3c3c] bg-slate-50 dark:bg-[#1e1e1e]">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 transition-colors"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={onProceed}
                        disabled={!validationResult.compatible}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:bg-slate-300 disabled:cursor-not-allowed transition-colors"
                        title={!validationResult.compatible ? t('validation_cannot_proceed') : t('validation_proceed')}
                    >
                        {validationResult.compatible ? t('validation_proceed') : t('validation_cannot_import')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default ValidationResultsModal;
