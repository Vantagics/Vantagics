import React, { useState, useEffect } from 'react';
import { useLanguage } from '../i18n';
import { X, AlertCircle, CheckCircle, Key, Search, ExternalLink } from 'lucide-react';
import { OpenExternalURL } from '../../wailsjs/go/main/App';

interface SearchAPIConfig {
    id: string;
    name: string;
    description: string;
    apiKey?: string;
    customId?: string;
    enabled: boolean;
    tested: boolean;
}

interface TestingState {
    [key: string]: boolean; // API ID -> testing status
}

interface TestResultState {
    [key: string]: { success: boolean; message: string } | null; // API ID -> test result
}

interface SearchAPIConfigModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: (configs: SearchAPIConfig[], activeAPI: string) => void;
    currentConfigs: SearchAPIConfig[];
    activeAPI: string;
}

const SearchAPIConfigModal: React.FC<SearchAPIConfigModalProps> = ({
    isOpen,
    onClose,
    onSave,
    currentConfigs,
    activeAPI
}) => {
    const { t } = useLanguage();
    const [configs, setConfigs] = useState<SearchAPIConfig[]>([]);
    const [selectedAPI, setSelectedAPI] = useState<string>('');
    const [testing, setTesting] = useState<TestingState>({});
    const [testResults, setTestResults] = useState<TestResultState>({});

    useEffect(() => {
        if (isOpen) {
            // Filter out DuckDuckGo from existing configs
            const filteredConfigs = currentConfigs.filter(c => c.id !== 'duckduckgo');
            setConfigs(filteredConfigs.length > 0 ? [...filteredConfigs] : getDefaultConfigs());
            // Set selected API, defaulting to first available if activeAPI was duckduckgo
            const validActiveAPI = activeAPI && activeAPI !== 'duckduckgo' ? activeAPI : '';
            setSelectedAPI(validActiveAPI || (filteredConfigs.length > 0 ? filteredConfigs[0].id : 'serper'));
            setTestResults({});
            setTesting({});
        }
    }, [isOpen, currentConfigs, activeAPI]);

    const getDefaultConfigs = (): SearchAPIConfig[] => [
        {
            id: 'serper',
            name: 'Serper (Google Search)',
            description: 'Google Search API via Serper.dev (API key required)',
            apiKey: '',
            enabled: false,
            tested: false
        },
        {
            id: 'uapi_pro',
            name: 'UAPI Pro',
            description: 'UAPI Pro search service (API key optional, currently in free beta)',
            apiKey: '',
            enabled: false,
            tested: false
        }
    ];

    const updateConfig = (id: string, field: string, value: any) => {
        setConfigs(configs.map(cfg =>
            cfg.id === id ? { ...cfg, [field]: value, tested: false } : cfg
        ));
        // Clear test result for this API when config changes
        setTestResults(prev => ({ ...prev, [id]: null }));
    };

    const handleTestAPI = async (apiId: string) => {
        const config = configs.find(c => c.id === apiId);
        if (!config) return;

        // Validation - Serper requires API key
        if (config.id === 'serper' && !config.apiKey) {
            setTestResults(prev => ({
                ...prev,
                [apiId]: {
                    success: false,
                    message: t('serper_requires_key') || 'Serper.dev requires an API key'
                }
            }));
            return;
        }

        setTesting(prev => ({ ...prev, [apiId]: true }));
        setTestResults(prev => ({ ...prev, [apiId]: null }));

        try {
            // @ts-ignore - TestSearchAPI is defined in App.go
            const result = await window.go.main.App.TestSearchAPI(config);
            setTestResults(prev => ({ ...prev, [apiId]: result }));
            if (result.success) {
                // Update tested status
                setConfigs(configs.map(cfg =>
                    cfg.id === apiId ? { ...cfg, tested: true } : cfg
                ));
            }
        } catch (err) {
            setTestResults(prev => ({
                ...prev,
                [apiId]: { success: false, message: String(err) }
            }));
        } finally {
            setTesting(prev => ({ ...prev, [apiId]: false }));
        }
    };

    const handleSave = () => {
        const selectedConfig = configs.find(c => c.id === selectedAPI);
        
        // Validation
        if (!selectedConfig) {
            // Show error in the selected API's test result
            setTestResults(prev => ({
                ...prev,
                [selectedAPI]: {
                    success: false,
                    message: t('please_select_api') || 'Please select a search API'
                }
            }));
            return;
        }

        // Serper requires API key
        if (selectedConfig.id === 'serper' && !selectedConfig.apiKey) {
            setTestResults(prev => ({
                ...prev,
                [selectedAPI]: {
                    success: false,
                    message: t('serper_requires_key') || 'Serper.dev requires an API key'
                }
            }));
            return;
        }

        // Serper must be tested before saving
        if (selectedConfig.id === 'serper' && !selectedConfig.tested) {
            setTestResults(prev => ({
                ...prev,
                [selectedAPI]: {
                    success: false,
                    message: t('please_test_before_save') || 'Please test the connection before saving'
                }
            }));
            return;
        }

        // Mark selected API as enabled
        const updatedConfigs = configs.map(cfg => ({
            ...cfg,
            enabled: cfg.id === selectedAPI
        }));

        onSave(updatedConfigs, selectedAPI);
        onClose();
    };

    const getAPIIcon = (id: string) => {
        switch (id) {
            case 'serper':
                return <Search size={20} className="text-blue-500" />;
            case 'uapi_pro':
                return <Key size={20} className="text-purple-500" />;
            default:
                return <Search size={20} />;
        }
    };

    if (!isOpen) return null;

    const selectedConfig = configs.find(c => c.id === selectedAPI);

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-3xl mx-4 max-h-[90vh] overflow-y-auto">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700 sticky top-0 bg-white dark:bg-gray-800 z-10">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                        {t('search_api_config') || 'Search API Configuration'}
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
                    {/* API Selection */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                            {t('select_search_api') || 'Select Search API'}
                        </label>
                        <div className="space-y-3">
                            {configs.map((config) => {
                                const isTestingThis = testing[config.id] || false;
                                const testResult = testResults[config.id];
                                
                                return (
                                    <div key={config.id} className="border-2 rounded-lg overflow-hidden">
                                        <div
                                            className={`p-4 cursor-pointer transition-all ${
                                                selectedAPI === config.id
                                                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                                                    : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                                            }`}
                                            onClick={() => setSelectedAPI(config.id)}
                                        >
                                            <div className="flex items-start">
                                                <div className="flex-shrink-0 mt-1">
                                                    {getAPIIcon(config.id)}
                                                </div>
                                                <div className="ml-3 flex-1">
                                                    <div className="flex items-center justify-between">
                                                        <h3 className="text-sm font-medium text-gray-900 dark:text-white">
                                                            {config.name}
                                                        </h3>
                                                        <div className="flex items-center gap-2">
                                                            {config.tested && (
                                                                <div title={t('tested') || 'Tested'}>
                                                                    <CheckCircle size={16} className="text-green-500" />
                                                                </div>
                                                            )}
                                                            <input
                                                                type="radio"
                                                                checked={selectedAPI === config.id}
                                                                onChange={() => setSelectedAPI(config.id)}
                                                                className="ml-1"
                                                            />
                                                        </div>
                                                    </div>
                                                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                                        {config.description}
                                                    </p>
                                                    
                                                    {/* Test button for this API */}
                                                    <div className="mt-3">
                                                        <button
                                                            onClick={(e) => {
                                                                e.stopPropagation();
                                                                handleTestAPI(config.id);
                                                            }}
                                                            disabled={isTestingThis}
                                                            className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
                                                        >
                                                            {isTestingThis ? (t('testing') || 'Testing...') : (t('test') || 'Test')}
                                                        </button>
                                                    </div>
                                                    
                                                    {/* Test result for this API */}
                                                    {testResult && (
                                                        <div className={`mt-2 p-2 rounded text-xs ${testResult.success ? 'bg-green-50 dark:bg-green-900/20' : 'bg-red-50 dark:bg-red-900/20'}`}>
                                                            <div className="flex items-start">
                                                                {testResult.success ? (
                                                                    <CheckCircle className="text-green-600 dark:text-green-400 mr-2 flex-shrink-0" size={14} />
                                                                ) : (
                                                                    <AlertCircle className="text-red-600 dark:text-red-400 mr-2 flex-shrink-0" size={14} />
                                                                )}
                                                                <span className={testResult.success ? 'text-green-800 dark:text-green-200' : 'text-red-800 dark:text-red-200'}>
                                                                    {testResult.message}
                                                                </span>
                                                            </div>
                                                        </div>
                                                    )}
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                );
                            })}
                        </div>
                    </div>

                    {/* Configuration Fields */}
                    {selectedConfig && (
                        <div className="space-y-4 p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                            <h3 className="text-sm font-medium text-gray-900 dark:text-white">
                                {selectedConfig.name} {t('configuration') || 'Configuration'}
                            </h3>

                            {selectedConfig.id === 'serper' && (
                                <>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                            <Key size={16} className="inline mr-2" />
                                            {t('api_key') || 'API Key'}
                                            <span className="text-red-500 ml-1">*</span>
                                        </label>
                                        <div className="flex gap-2">
                                            <input
                                                type="password"
                                                value={selectedConfig.apiKey || ''}
                                                onChange={(e) => updateConfig(selectedConfig.id, 'apiKey', e.target.value)}
                                                placeholder={t('enter_serper_api_key') || 'Enter your Serper.dev API key'}
                                                className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
                                            />
                                            <button
                                                onClick={() => OpenExternalURL('https://serper.dev')}
                                                className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 transition-colors flex items-center gap-2 whitespace-nowrap"
                                                title={t('get_api_key') || 'Get API Key'}
                                            >
                                                <ExternalLink size={16} />
                                                {t('get_key') || 'Get Key'}
                                            </button>
                                        </div>
                                    </div>
                                    <div className="text-xs text-gray-500 dark:text-gray-400">
                                        <p>{t('serper_help') || 'Get your API key from Serper.dev dashboard'}</p>
                                    </div>
                                </>
                            )}

                            {selectedConfig.id === 'uapi_pro' && (
                                <>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                            <Key size={16} className="inline mr-2" />
                                            {t('api_key') || 'API Key'}
                                            <span className="text-gray-400 ml-1 text-xs">({t('optional') || 'Optional'})</span>
                                        </label>
                                        <div className="flex gap-2">
                                            <input
                                                type="password"
                                                value={selectedConfig.apiKey || ''}
                                                onChange={(e) => updateConfig(selectedConfig.id, 'apiKey', e.target.value)}
                                                placeholder={t('enter_uapi_api_key_optional') || 'Enter your UAPI Pro API key (optional)'}
                                                className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
                                            />
                                            <button
                                                onClick={() => OpenExternalURL('https://uapis.cn')}
                                                className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 transition-colors flex items-center gap-2 whitespace-nowrap"
                                                title={t('get_api_key') || 'Get API Key'}
                                            >
                                                <ExternalLink size={16} />
                                                {t('get_key') || 'Get Key'}
                                            </button>
                                        </div>
                                    </div>
                                    <div className="text-xs text-gray-500 dark:text-gray-400">
                                        <p>{t('uapi_pro_help_optional') || 'Currently in free beta - API key is optional. Get your key from UAPI dashboard for future use.'}</p>
                                    </div>
                                </>
                            )}
                        </div>
                    )}

                    {/* Info Box */}
                    <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-md">
                        <div className="flex items-start">
                            <AlertCircle className="text-blue-600 dark:text-blue-400 mr-3 flex-shrink-0" size={20} />
                            <div className="text-sm text-blue-800 dark:text-blue-200">
                                <p className="font-medium mb-1">
                                    {t('search_api_info_title') || 'About Search APIs'}
                                </p>
                                <ul className="list-disc list-inside space-y-1 text-xs">
                                    <li>{t('serper_info') || 'Serper: Google Search results via API, requires API key'}</li>
                                    <li>{t('uapi_pro_info') || 'UAPI Pro: Structured data with stable schemas, currently in free beta (API key optional)'}</li>
                                </ul>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Footer */}
                <div className="flex justify-end gap-3 p-6 border-t border-gray-200 dark:border-gray-700 sticky bottom-0 bg-white dark:bg-gray-800">
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

export default SearchAPIConfigModal;
