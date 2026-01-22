import React, { useState, useEffect } from 'react';
import { X, Server, AlertCircle, CheckCircle, Loader } from 'lucide-react';
import { useLanguage } from '../i18n';
import { TestMCPService } from '../../wailsjs/go/main/App';
import { config as configModel } from '../../wailsjs/go/models';

// Use Wails generated type
type MCPService = configModel.MCPService;

interface MCPServiceModalProps {
    isOpen: boolean;
    service: MCPService | null; // null for new service
    onClose: () => void;
    onSave: (service: MCPService) => void;
}

const MCPServiceModal: React.FC<MCPServiceModalProps> = ({
    isOpen,
    service,
    onClose,
    onSave
}) => {
    const { t } = useLanguage();
    const [formData, setFormData] = useState<MCPService>({
        id: '',
        name: '',
        description: '',
        url: '',
        enabled: true,
        tested: false
    });
    const [errors, setErrors] = useState<{ [key: string]: string }>({});
    const [isTesting, setIsTesting] = useState(false);
    const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

    useEffect(() => {
        if (isOpen) {
            if (service) {
                setFormData(service);
            } else {
                setFormData({
                    id: `mcp-${Date.now()}`,
                    name: '',
                    description: '',
                    url: '',
                    enabled: true,
                    tested: false
                });
            }
            setErrors({});
            setTestResult(null);
        }
    }, [isOpen, service]);

    // Clear test result when URL changes
    useEffect(() => {
        if (formData.url !== service?.url) {
            setTestResult(null);
            // Clear tested status when URL changes - requires re-testing
            setFormData(prev => ({ ...prev, tested: false }));
        }
    }, [formData.url, service?.url]);

    const validateForm = (): boolean => {
        const newErrors: { [key: string]: string } = {};

        if (!formData.name.trim()) {
            newErrors.name = t('mcp_service_name_required') || 'Service name is required';
        }

        if (!formData.url.trim()) {
            newErrors.url = t('mcp_service_url_required') || 'Service URL is required';
        } else if (!formData.url.match(/^https?:\/\/.+/)) {
            newErrors.url = t('mcp_service_url_invalid') || 'Please enter a valid URL';
        }

        setErrors(newErrors);
        return Object.keys(newErrors).length === 0;
    };

    const handleTest = async () => {
        if (!formData.url.trim()) {
            setTestResult({
                success: false,
                message: t('mcp_service_url_required') || 'Service URL is required'
            });
            return;
        }

        setIsTesting(true);
        setTestResult(null);

        try {
            const result = await TestMCPService(formData.url);
            setTestResult(result);
            
            // Update formData.tested when test is successful
            if (result.success) {
                setFormData(prev => ({ ...prev, tested: true }));
            }
        } catch (err) {
            setTestResult({
                success: false,
                message: String(err)
            });
        } finally {
            setIsTesting(false);
        }
    };

    const handleSave = () => {
        if (!validateForm()) {
            return;
        }

        // Check if service has been tested successfully
        if (!testResult || !testResult.success) {
            setErrors({
                ...errors,
                url: t('mcp_test_required') || 'Please test the service connection before saving'
            });
            return;
        }

        // Create service with tested flag set to true
        // IMPORTANT: Explicitly set tested to true when test was successful
        const serviceToSave: MCPService = {
            id: formData.id,
            name: formData.name.trim(),
            description: formData.description.trim(),
            url: formData.url.trim(),
            enabled: formData.enabled,
            tested: true // Explicitly set to true since testResult.success is verified above
        };

        onSave(serviceToSave);
        onClose();
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && e.ctrlKey) {
            handleSave();
        } else if (e.key === 'Escape') {
            onClose();
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-2xl">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-slate-200">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-blue-100 rounded-lg">
                            <Server className="w-5 h-5 text-blue-600" />
                        </div>
                        <h2 className="text-xl font-semibold text-slate-800">
                            {service ? t('edit_mcp_service') : t('add_mcp_service')}
                        </h2>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-1 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                        <X className="w-5 h-5 text-slate-400" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 space-y-4" onKeyDown={handleKeyDown}>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-2">
                            {t('mcp_service_name')} <span className="text-red-500">*</span>
                        </label>
                        <input
                            type="text"
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                            className={`w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                                errors.name ? 'border-red-300' : 'border-slate-300'
                            }`}
                            placeholder="e.g., Web Search, Database Tools"
                            autoFocus
                        />
                        {errors.name && (
                            <p className="mt-1 text-sm text-red-600 flex items-center gap-1">
                                <AlertCircle className="w-4 h-4" />
                                {errors.name}
                            </p>
                        )}
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-2">
                            {t('mcp_service_description')}
                        </label>
                        <input
                            type="text"
                            value={formData.description}
                            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                            className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                            placeholder="Brief description of what this service provides"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-2">
                            {t('mcp_service_url')} <span className="text-red-500">*</span>
                        </label>
                        <input
                            type="text"
                            value={formData.url}
                            onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                            className={`w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm ${
                                errors.url ? 'border-red-300' : 'border-slate-300'
                            }`}
                            placeholder="https://mcp.example.com/api"
                        />
                        {errors.url && (
                            <p className="mt-1 text-sm text-red-600 flex items-center gap-1">
                                <AlertCircle className="w-4 h-4" />
                                {errors.url}
                            </p>
                        )}
                        {!testResult && formData.url && (
                            <p className="mt-2 text-xs text-amber-600 flex items-center gap-1">
                                <AlertCircle className="w-3 h-3" />
                                {t('mcp_test_required')}
                            </p>
                        )}
                    </div>

                    {/* Test Button and Result */}
                    <div className="flex items-center gap-4">
                        <button
                            onClick={handleTest}
                            disabled={isTesting || !formData.url.trim()}
                            className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                        >
                            {isTesting ? (
                                <>
                                    <Loader className="w-4 h-4 animate-spin" />
                                    {t('testing_mcp_service')}
                                </>
                            ) : (
                                <>
                                    <Server className="w-4 h-4" />
                                    {t('test_mcp_service')}
                                </>
                            )}
                        </button>

                        {testResult && (
                            <div
                                className={`flex items-center gap-2 text-sm font-medium animate-in fade-in slide-in-from-left-1 ${
                                    testResult.success ? 'text-green-600' : 'text-red-600'
                                }`}
                            >
                                {testResult.success ? (
                                    <CheckCircle className="w-4 h-4" />
                                ) : (
                                    <AlertCircle className="w-4 h-4" />
                                )}
                                {testResult.message}
                            </div>
                        )}
                    </div>

                    <div className="flex items-center gap-2 pt-2">
                        <input
                            type="checkbox"
                            id="enabled"
                            checked={formData.enabled}
                            onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                            className="w-4 h-4 text-blue-600 border-slate-300 rounded focus:ring-2 focus:ring-blue-500"
                        />
                        <label htmlFor="enabled" className="text-sm font-medium text-slate-700">
                            {t('mcp_service_enabled')}
                        </label>
                    </div>
                </div>

                {/* Footer */}
                <div className="flex items-center justify-end gap-3 p-6 border-t border-slate-200 bg-slate-50">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-200 rounded-lg transition-colors"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={handleSave}
                        disabled={!testResult || !testResult.success}
                        className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors flex items-center gap-2 ${
                            testResult && testResult.success
                                ? 'text-white bg-blue-600 hover:bg-blue-700'
                                : 'text-slate-400 bg-slate-200 cursor-not-allowed'
                        }`}
                        title={
                            !testResult || !testResult.success
                                ? t('mcp_test_required')
                                : undefined
                        }
                    >
                        <Server className="w-4 h-4" />
                        {testResult && testResult.success ? (
                            <>
                                <CheckCircle className="w-4 h-4" />
                                {t('save_changes')}
                            </>
                        ) : (
                            t('save_changes')
                        )}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default MCPServiceModal;
