import React, { useState, useEffect } from 'react';
import { useLanguage } from '../i18n';
import { X, AlertCircle, CheckCircle, Key, Globe } from 'lucide-react';

interface UAPIConfig {
    enabled: boolean;
    apiToken: string;
    baseUrl?: string;
    tested: boolean;
}

interface UAPIConfigModalProps {
    isOpen: boolean;
    config: UAPIConfig | null;
    onClose: () => void;
    onSave: (config: UAPIConfig) => void;
}

const UAPIConfigModal: React.FC<UAPIConfigModalProps> = ({ isOpen, config, onClose, onSave }) => {
    const { t } = useLanguage();
    const [enabled, setEnabled] = useState(false);
    const [apiToken, setApiToken] = useState('');
    const [baseUrl, setBaseUrl] = useState('');
    const [testing, setTesting] = useState(false);
    const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
    const [tested, setTested] = useState(false);

    useEffect(() => {
        if (isOpen && config) {
            setEnabled(config.enabled);
            setApiToken(config.apiToken || '');
            setBaseUrl(config.baseUrl || '');
            setTested(config.tested);
            setTestResult(null);
        } else if (isOpen) {
            // Default values for new config
            setEnabled(false);
            setApiToken('');
            setBaseUrl('');
            setTested(false);
            setTestResult(null);
        }
    }, [isOpen, config]);

    const handleTest = async () => {
        if (!apiToken.trim()) {
            setTestResult({ 
                success: false, 
                message: t('uapi_token_required') || 'API Token is required' 
            });
            return;
        }

        setTesting(true);
        setTestResult(null);

        try {
            // Test UAPI connection
            // @ts-ignore - TestUAPIConnection is defined in App.go
            const result = await window.go.main.App.TestUAPIConnection(apiToken.trim(), baseUrl.trim());
            setTestResult(result);
            if (result.success) {
                setTested(true);
            } else {
                setTested(false);
            }
        } catch (err) {
            setTestResult({ 
                success: false, 
                message: String(err) 
            });
            setTested(false);
        } finally {
            setTesting(false);
        }
    };

    const handleSave = () => {
        // Validation
        if (enabled && !apiToken.trim()) {
            setTestResult({ 
                success: false, 
                message: t('uapi_token_required') || 'API Token is required when UAPI is enabled' 
            });
            return;
        }

        if (enabled && !tested) {
            setTestResult({ 
                success: false, 
                message: t('uapi_test_required') || 'Please test the connection before saving' 
            });
            return;
        }

        const newConfig: UAPIConfig = {
            enabled,
            apiToken: apiToken.trim(),
            baseUrl: baseUrl.trim() || undefined,
            tested
        };

        onSave(newConfig);
        onClose();
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-2xl mx-4">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                        {t('uapi_config_title') || 'UAPI Configuration'}
                    </h2>
                    <button
                        onClick={onClose}
                        className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                    >
                        <X size={24} />
                    </button>
                </div>

                {/* Body */}
                <div className="p-6 space-y-6">
                    {/* Enable Toggle */}
                    <div className="flex items-center justify-between">
                        <div>
                            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                {t('uapi_enable') || 'Enable UAPI Search'}
                            </label>
                            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                {t('uapi_enable_desc') || 'Enable structured data search using UAPI'}
                            </p>
                        </div>
                        <label className="relative inline-flex items-center cursor-pointer">
                            <input
                                type="checkbox"
                                checked={enabled}
                                onChange={(e) => setEnabled(e.target.checked)}
                                className="sr-only peer"
                            />
                            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
                        </label>
                    </div>

                    {/* API Token */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            <Key size={16} className="inline mr-2" />
                            {t('uapi_token') || 'API Token'}
                            <span className="text-red-500 ml-1">*</span>
                        </label>
                        <input
                            type="password"
                            value={apiToken}
                            onChange={(e) => {
                                setApiToken(e.target.value);
                                setTested(false);
                                setTestResult(null);
                            }}
                            placeholder={t('uapi_token_placeholder') || 'Enter your UAPI API token'}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
                            disabled={!enabled}
                        />
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            {t('uapi_token_help') || 'Get your API token from UAPI dashboard'}
                        </p>
                    </div>

                    {/* Base URL (Optional) */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            <Globe size={16} className="inline mr-2" />
                            {t('uapi_base_url') || 'Base URL (Optional)'}
                        </label>
                        <input
                            type="text"
                            value={baseUrl}
                            onChange={(e) => {
                                setBaseUrl(e.target.value);
                                setTested(false);
                                setTestResult(null);
                            }}
                            placeholder="https://api.uapi.nl"
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
                            disabled={!enabled}
                        />
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            {t('uapi_base_url_help') || 'Leave empty to use default UAPI endpoint'}
                        </p>
                    </div>

                    {/* Test Button */}
                    <div>
                        <button
                            onClick={handleTest}
                            disabled={testing || !enabled || !apiToken.trim()}
                            className="w-full px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
                        >
                            {testing ? (t('testing') || 'Testing...') : (t('test_connection') || 'Test Connection')}
                        </button>
                    </div>

                    {/* Test Result */}
                    {testResult && (
                        <div className={`p-4 rounded-md ${testResult.success ? 'bg-green-50 dark:bg-green-900/20' : 'bg-red-50 dark:bg-red-900/20'}`}>
                            <div className="flex items-start">
                                {testResult.success ? (
                                    <CheckCircle className="text-green-600 dark:text-green-400 mr-3 flex-shrink-0" size={20} />
                                ) : (
                                    <AlertCircle className="text-red-600 dark:text-red-400 mr-3 flex-shrink-0" size={20} />
                                )}
                                <div className="flex-1">
                                    <p className={`text-sm ${testResult.success ? 'text-green-800 dark:text-green-200' : 'text-red-800 dark:text-red-200'}`}>
                                        {testResult.message}
                                    </p>
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Info Box */}
                    <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-md">
                        <div className="flex items-start">
                            <AlertCircle className="text-blue-600 dark:text-blue-400 mr-3 flex-shrink-0" size={20} />
                            <div className="text-sm text-blue-800 dark:text-blue-200">
                                <p className="font-medium mb-1">
                                    {t('uapi_info_title') || 'About UAPI'}
                                </p>
                                <p>
                                    {t('uapi_info_desc') || 'UAPI provides normalized, structured data from various sources including social media, gaming platforms, and web content. Visit docs.uapi.nl for more information.'}
                                </p>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Footer */}
                <div className="flex justify-end gap-3 p-6 border-t border-gray-200 dark:border-gray-700">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors"
                    >
                        {t('cancel') || 'Cancel'}
                    </button>
                    <button
                        onClick={handleSave}
                        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
                    >
                        {t('save') || 'Save'}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default UAPIConfigModal;
