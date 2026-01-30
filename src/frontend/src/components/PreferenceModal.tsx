import React, { useState, useEffect } from 'react';
import { GetConfig, SaveConfig, SelectDirectory, GetPythonEnvironments, ValidatePython, InstallPythonPackages, CreateVantageDataEnvironment, CheckVantageDataEnvironmentExists, DiagnosePythonInstallation, GetSkills, EnableSkill, DisableSkill, ReloadSkills, GetLogStats, CleanupLogs } from '../../wailsjs/go/main/App';
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
import { main, agent, config as configModel } from '../../wailsjs/go/models';
import { useLanguage } from '../i18n';
import Toast, { ToastType } from './Toast';
import MCPServiceModal from './MCPServiceModal';
import { Plus, Edit2, Trash2, Server, Power, PowerOff, CheckCircle, AlertCircle, Zap, RefreshCw, Search, Filter, Tag, BookOpen, X, MapPin, Archive } from 'lucide-react';
import { countries, getCityDisplayName, getCountryDisplayName, City, Country } from '../data/cities';

type Tab = 'llm' | 'system' | 'session' | 'mcp' | 'search' | 'network' | 'runenv' | 'skills' | 'intent';

// Skill types
interface SkillInfo {
    id: string;
    name: string;
    description: string;
    version: string;
    author: string;
    category: string;
    keywords: string[];
    required_columns: string[];
    tools: string[];
    enabled: boolean;
    icon: string;
    tags: string[];
}

// Use Wails generated type
type MCPService = configModel.MCPService;

// Search API Config type
interface SearchAPIConfig {
    id: string;
    name: string;
    description: string;
    apiKey?: string;
    customId?: string;
    enabled: boolean;
    tested: boolean;
}

interface PreferenceModalProps {
    isOpen: boolean;
    onClose: () => void;
    onOpenSkills?: () => void;
}

const PreferenceModal: React.FC<PreferenceModalProps> = ({ isOpen, onClose, onOpenSkills }) => {
    const { t } = useLanguage();
    const [activeTab, setActiveTab] = useState<Tab>('system');
    const [config, setConfig] = useState<configModel.Config>(configModel.Config.createFrom({
        llmProvider: 'OpenAI',
        apiKey: '',
        baseUrl: '',
        modelName: '',
        maxTokens: 4096,
        darkMode: false,
        enableMemory: false,
        autoAnalysisSuggestions: true,
        autoIntentUnderstanding: true,
        localCache: true,
        language: 'English',
        claudeHeaderStyle: 'Anthropic',
        dataCacheDir: '',
        pythonPath: '',
        maxPreviewRows: 100,
        maxConcurrentAnalysis: 5,
        detailedLog: false,
        mcpServices: []
    }));
    const [isTesting, setIsTesting] = useState(false);
    const [testResult, setTestResult] = useState<{ success: boolean, message: string } | null>(null);
    const [toast, setToast] = useState<{ message: string; type: ToastType } | null>(null);
    const [mcpModalOpen, setMcpModalOpen] = useState(false);
    const [editingMcpService, setEditingMcpService] = useState<MCPService | null>(null);
    const [testingSearchAPI, setTestingSearchAPI] = useState<string | null>(null); // Track which API is being tested
    const [searchAPITestResults, setSearchAPITestResults] = useState<{ [key: string]: { success: boolean; message: string } | null }>({});
    const [logStats, setLogStats] = useState<{ totalSizeMB: number; logCount: number; archiveCount: number; logDir: string } | null>(null);
    const [isCleaningLogs, setIsCleaningLogs] = useState(false);

    // Helper function to update config while maintaining Config class instance
    const updateConfig = (updates: Partial<configModel.Config>) => {
        setConfig(configModel.Config.createFrom({ ...config, ...updates }));
    };

    useEffect(() => {
        if (isOpen) {
            GetConfig().then(data => {
                // Filter out DuckDuckGo from search APIs (deprecated)
                if (data.searchAPIs) {
                    data.searchAPIs = data.searchAPIs.filter((api: any) => api.id !== 'duckduckgo');
                }
                // Reset activeSearchAPI if it was duckduckgo
                if (data.activeSearchAPI === 'duckduckgo') {
                    data.activeSearchAPI = '';
                }
                setConfig(data);
                // Always load log stats
                loadLogStats();
            }).catch(console.error);
            setTestResult(null);
        }
    }, [isOpen]);

    // Load log statistics
    const loadLogStats = async () => {
        try {
            const stats = await GetLogStats();
            setLogStats(stats);
        } catch (err) {
            console.error('Failed to get log stats:', err);
            setLogStats(null);
        }
    };

    // Handle log cleanup
    const handleCleanupLogs = async () => {
        setIsCleaningLogs(true);
        try {
            await CleanupLogs();
            setToast({ message: t('log_cleanup_success'), type: 'success' });
            // Reload stats after cleanup
            await loadLogStats();
        } catch (err) {
            console.error('Failed to cleanup logs:', err);
            setToast({ message: t('log_cleanup_error') + ': ' + err, type: 'error' });
        } finally {
            setIsCleaningLogs(false);
        }
    };

    const handleSave = async () => {
        try {
            console.log('[Config] Saving config:', config);
            console.log('[Config] MCP Services:', config.mcpServices);
            await SaveConfig(config);
            // Show success toast
            setToast({ message: t('settings_save_success') || '配置保存成功', type: 'success' });
            // Close modal after a short delay to allow toast to be visible
            setTimeout(() => {
                onClose();
            }, 500);
        } catch (err) {
            console.error('Failed to save config:', err);
            setToast({ message: t('settings_save_failed') + ': ' + err, type: 'error' });
        }
    };

    const handleBrowseDirectory = async () => {
        try {
            const path = await SelectDirectory();
            if (path) {
                updateConfig({ dataCacheDir: path });
            }
        } catch (err) {
            console.error('Failed to select directory:', err);
        }
    };

    const handleTestConnection = async () => {
        setIsTesting(true);
        setTestResult(null);
        try {
            // @ts-ignore - We will implement this in App.go
            const result = await window.go.main.App.TestLLMConnection(config);
            setTestResult(result);
        } catch (err) {
            setTestResult({ success: false, message: String(err) });
        } finally {
            setIsTesting(false);
        }
    };

    const handleTestSearchAPI = async (apiId: string) => {
        const searchAPIs = config.searchAPIs || getDefaultSearchAPIs();
        const apiConfig = searchAPIs.find((api: SearchAPIConfig) => api.id === apiId);
        if (!apiConfig) return;

        // Validation - only Serper requires API key
        if (apiConfig.id === 'serper' && !apiConfig.apiKey) {
            setSearchAPITestResults(prev => ({
                ...prev,
                [apiId]: {
                    success: false,
                    message: 'Serper.dev requires an API key'
                }
            }));
            return;
        }

        setTestingSearchAPI(apiId);
        setSearchAPITestResults(prev => ({ ...prev, [apiId]: null }));

        try {
            // @ts-ignore - TestSearchAPI is defined in App.go
            const result = await window.go.main.App.TestSearchAPI(apiConfig);
            setSearchAPITestResults(prev => ({ ...prev, [apiId]: result }));
            if (result.success) {
                // Update tested status
                const updatedAPIs = searchAPIs.map((api: SearchAPIConfig) =>
                    api.id === apiId ? { ...api, tested: true } : api
                );
                updateConfig({ searchAPIs: updatedAPIs });
            }
        } catch (err) {
            setSearchAPITestResults(prev => ({
                ...prev,
                [apiId]: { success: false, message: String(err) }
            }));
        } finally {
            setTestingSearchAPI(null);
        }
    };

    const getDefaultSearchAPIs = (): SearchAPIConfig[] => [
        {
            id: 'serper',
            name: 'Serper (Google Search)',
            description: 'Google Search API via Serper.dev (requires API key)',
            apiKey: '',
            enabled: false,
            tested: false
        },
        {
            id: 'uapi_pro',
            name: 'UAPI Pro',
            description: 'UAPI Pro search service with structured data (API key optional)',
            apiKey: '',
            enabled: false,
            tested: false
        }
    ];

    const updateSearchAPIConfig = (id: string, field: string, value: any) => {
        const searchAPIs = config.searchAPIs || getDefaultSearchAPIs();
        const updatedAPIs = searchAPIs.map((api: SearchAPIConfig) =>
            api.id === id ? { ...api, [field]: value, tested: false } : api
        );
        updateConfig({ searchAPIs: updatedAPIs });
        // Clear test result for this API when config changes
        setSearchAPITestResults(prev => ({ ...prev, [id]: null }));
    };

    if (!isOpen) return null;

    const isAnthropic = config.llmProvider === 'Anthropic';
    const isGemini = config.llmProvider === 'Gemini';
    const isOpenAICompatible = config.llmProvider === 'OpenAI-Compatible';
    const isClaudeCompatible = config.llmProvider === 'Claude-Compatible';

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[800px] h-[600px] rounded-xl shadow-2xl flex overflow-hidden text-slate-900">
                {/* Sidebar */}
                <div className="w-64 bg-slate-50 border-r border-slate-200 p-4 flex flex-col">
                    <h2 className="text-xl font-bold text-slate-800 mb-6 px-2">{t('preferences')}</h2>
                    <nav className="space-y-1">
                        {(['system', 'session', 'llm', 'search', 'network', 'mcp', 'runenv', 'skills', 'intent'] as const).map((tab) => (
                            <button
                                key={tab}
                                onClick={() => setActiveTab(tab)}
                                className={`w-full text-left px-4 py-2 rounded-lg text-sm font-medium transition-colors ${activeTab === tab ? 'bg-blue-100 text-blue-700' : 'text-slate-600 hover:bg-slate-100'
                                    }`}
                            >
                                {tab === 'system' && t('system_params')}
                                {tab === 'session' && t('session_management')}
                                {tab === 'llm' && t('llm_config')}
                                {tab === 'search' && t('search_engine')}
                                {tab === 'network' && t('network_settings')}
                                {tab === 'mcp' && t('mcp_services')}
                                {tab === 'runenv' && t('run_env')}
                                {tab === 'skills' && (t('skills_management') || 'Skills管理')}
                                {tab === 'intent' && (t('intent_enhancement') || '意图增强')}
                            </button>
                        ))}
                    </nav>
                </div>

                {/* Content Area */}
                <div className="flex-1 flex flex-col min-w-0">
                    <div className="flex-1 p-8 overflow-y-auto">
                        {activeTab === 'llm' && (
                            <div className="space-y-6">
                                <h3 className="text-lg font-semibold text-slate-800 border-b border-slate-200 pb-2">{t('llm_config')}</h3>
                                <div className="grid gap-4">
                                    <div>
                                        <label htmlFor="llmProvider" className="block text-sm font-medium text-slate-700 mb-1">{t('provider_type')}</label>
                                        <select
                                            id="llmProvider"
                                            value={config.llmProvider}
                                            onChange={(e) => updateConfig({ llmProvider: e.target.value })}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        >
                                            <option value="OpenAI">OpenAI</option>
                                            <option value="Anthropic">Anthropic (Claude)</option>
                                            <option value="Gemini">Google Gemini</option>
                                            <option value="OpenAI-Compatible">OpenAI-Compatible (Local, DeepSeek, etc.)</option>
                                            <option value="Claude-Compatible">Claude-Compatible (Proxies, Bedrock, etc.)</option>
                                        </select>
                                    </div>

                                    {(isOpenAICompatible || isClaudeCompatible) && (
                                        <div className="animate-in fade-in slide-in-from-top-1 duration-200">
                                            <label htmlFor="baseUrl" className="block text-sm font-medium text-slate-700 mb-1">
                                                {t('api_base_url')}
                                            </label>
                                            <input
                                                id="baseUrl"
                                                type="text"
                                                value={config.baseUrl}
                                                onChange={(e) => updateConfig({ baseUrl: e.target.value })}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                placeholder={
                                                    isOpenAICompatible ? "http://localhost:11434" :
                                                        "https://bedrock-runtime.us-east-1.amazonaws.com"
                                                }
                                                autoCapitalize="none"
                                                autoCorrect="off"
                                                spellCheck={false}
                                            />
                                            <p className="mt-1 text-[10px] text-slate-400 italic">
                                                {isOpenAICompatible
                                                    ? t('api_base_url_openai_compatible_desc')
                                                    : t('api_base_url_claude_compatible_desc')}
                                            </p>
                                        </div>
                                    )}

                                    {isClaudeCompatible && (
                                        <div className="animate-in fade-in slide-in-from-top-1 duration-200">
                                            <label htmlFor="headerStyle" className="block text-sm font-medium text-slate-700 mb-1">
                                                {t('header_style')}
                                            </label>
                                            <select
                                                id="headerStyle"
                                                value={config.claudeHeaderStyle || 'Anthropic'}
                                                onChange={(e) => updateConfig({ claudeHeaderStyle: e.target.value })}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            >
                                                <option value="Anthropic">Anthropic (x-api-key)</option>
                                                <option value="OpenAI">OpenAI (Authorization: Bearer)</option>
                                            </select>
                                            <p className="mt-1 text-[10px] text-slate-400 italic">
                                                {t('header_style_desc')}
                                            </p>
                                        </div>
                                    )}

                                    <div>
                                        <label htmlFor="apiKey" className="block text-sm font-medium text-slate-700 mb-1">
                                            {t('api_key')} {isOpenAICompatible ? '(Optional)' : ''}
                                        </label>
                                        <input
                                            id="apiKey"
                                            type="password"
                                            value={config.apiKey}
                                            onChange={(e) => updateConfig({ apiKey: e.target.value })}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            placeholder={isAnthropic ? "sk-ant-..." : (isGemini ? "AIza..." : "sk-...")}
                                            autoCapitalize="none"
                                            autoCorrect="off"
                                            spellCheck={false}
                                        />
                                    </div>
                                    <div>
                                        <label htmlFor="modelName" className="block text-sm font-medium text-slate-700 mb-1">{t('model_name')}</label>
                                        <input
                                            id="modelName"
                                            type="text"
                                            value={config.modelName}
                                            onChange={(e) => updateConfig({ modelName: e.target.value })}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            placeholder={isAnthropic ? "claude-3-5-sonnet-20240620" : (isGemini ? "gemini-3-pro" : (isOpenAICompatible ? "llama3" : "gpt-4o"))}
                                            autoCapitalize="none"
                                            autoCorrect="off"
                                            spellCheck={false}
                                        />
                                    </div>

                                    <div>
                                        <label htmlFor="maxTokens" className="block text-sm font-medium text-slate-700 mb-1">{t('max_tokens')}</label>
                                        <input
                                            id="maxTokens"
                                            type="number"
                                            value={config.maxTokens}
                                            onChange={(e) => updateConfig({ maxTokens: parseInt(e.target.value) || 0 })}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        />
                                    </div>

                                    <div className="pt-2 flex items-center gap-4">
                                        <button
                                            onClick={handleTestConnection}
                                            disabled={isTesting}
                                            className={`px-4 py-2 text-xs font-semibold rounded-md transition-colors ${isTesting ? 'bg-slate-100 text-slate-400 cursor-not-allowed' : 'bg-slate-100 text-slate-700 hover:bg-slate-200'
                                                }`}
                                        >
                                            {isTesting ? t('testing') : t('test_connection')}
                                        </button>

                                        {testResult && (
                                            <div className={`text-xs font-medium animate-in fade-in slide-in-from-left-1 ${testResult.success ? 'text-green-600' : 'text-red-600'
                                                }`}>
                                                {testResult.success ? `✓ ${t('connection_successful')}` : `✗ ${testResult.message}`}
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </div>
                        )}
                        {activeTab === 'system' && (
                            <div className="space-y-6">
                                <h3 className="text-lg font-semibold text-slate-800 border-b border-slate-200 pb-2">{t('system_params')}</h3>
                                <div className="space-y-4">
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <span className="block text-sm font-medium text-slate-700">{t('dark_mode')}</span>
                                            <span className="block text-xs text-slate-500">{t('dark_mode_desc')}</span>
                                        </div>
                                        <input
                                            type="checkbox"
                                            checked={config.darkMode}
                                            onChange={(e) => updateConfig({ darkMode: e.target.checked })}
                                        />
                                    </div>
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <span className="block text-sm font-medium text-slate-700">{t('local_cache')}</span>
                                            <span className="block text-xs text-slate-500">{t('local_cache_desc')}</span>
                                        </div>
                                        <input
                                            type="checkbox"
                                            checked={config.localCache}
                                            onChange={(e) => updateConfig({ localCache: e.target.checked })}
                                        />
                                    </div>
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <span className="block text-sm font-medium text-slate-700">{t('detailed_log')}</span>
                                            <span className="block text-xs text-slate-500">{t('detailed_log_desc')}</span>
                                        </div>
                                        <input
                                            type="checkbox"
                                            checked={config.detailedLog}
                                            onChange={(e) => updateConfig({ detailedLog: e.target.checked })}
                                        />
                                    </div>
                                    
                                    {/* Log Management - Independent of detailed log setting */}
                                    <div className="p-3 bg-slate-50 rounded-lg border border-slate-200">
                                        <div className="flex items-center justify-between mb-2">
                                            <span className="text-sm font-medium text-slate-700">{t('log_stats')}</span>
                                            <button
                                                onClick={loadLogStats}
                                                className="text-xs text-blue-600 hover:text-blue-800"
                                            >
                                                <RefreshCw className="w-3 h-3" />
                                            </button>
                                        </div>
                                        {logStats ? (
                                            <div className="space-y-1 text-xs text-slate-600">
                                                <div className="flex justify-between">
                                                    <span>{t('log_stats_total')}:</span>
                                                    <span className="font-medium">{logStats.totalSizeMB.toFixed(2)} MB</span>
                                                </div>
                                                <div className="flex justify-between">
                                                    <span>{t('log_stats_logs')}:</span>
                                                    <span className="font-medium">{logStats.logCount}</span>
                                                </div>
                                                <div className="flex justify-between">
                                                    <span>{t('log_stats_archives')}:</span>
                                                    <span className="font-medium">{logStats.archiveCount}</span>
                                                </div>
                                            </div>
                                        ) : (
                                            <p className="text-xs text-slate-500 italic">{t('loading')}</p>
                                        )}
                                        
                                        <div className="mt-3 pt-3 border-t border-slate-200">
                                            <label htmlFor="logMaxSizeMB" className="block text-xs font-medium text-slate-600 mb-1">{t('log_max_size')}</label>
                                            <input
                                                id="logMaxSizeMB"
                                                type="number"
                                                value={config.logMaxSizeMB || 100}
                                                onChange={(e) => updateConfig({ logMaxSizeMB: parseInt(e.target.value) || 100 })}
                                                className="w-full border border-slate-300 rounded-md p-1.5 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                min="1"
                                                max="10000"
                                            />
                                            <p className="mt-1 text-[10px] text-slate-400 italic">
                                                {t('log_max_size_desc')}
                                            </p>
                                        </div>
                                        
                                        <button
                                            onClick={handleCleanupLogs}
                                            disabled={isCleaningLogs}
                                            className="mt-2 w-full flex items-center justify-center gap-2 px-3 py-1.5 text-xs bg-amber-100 text-amber-800 rounded hover:bg-amber-200 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                                        >
                                            <Archive className="w-3 h-3" />
                                            {isCleaningLogs ? t('cleaning') : t('log_cleanup')}
                                        </button>
                                        <p className="mt-1 text-[10px] text-slate-400 italic">
                                            {t('log_cleanup_desc')}
                                        </p>
                                    </div>
                                    
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-1">{t('language')}</label>
                                        <select
                                            value={config.language}
                                            onChange={(e) => updateConfig({ language: e.target.value })}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm"
                                        >
                                            <option>English</option>
                                            <option>简体中文</option>
                                        </select>
                                    </div>
                                    <div>
                                        <label htmlFor="maxPreviewRows" className="block text-sm font-medium text-slate-700 mb-1">{t('max_preview_rows')}</label>
                                        <input
                                            id="maxPreviewRows"
                                            type="number"
                                            value={config.maxPreviewRows}
                                            onChange={(e) => updateConfig({ maxPreviewRows: parseInt(e.target.value) || 100 })}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            min="1"
                                            max="10000"
                                        />
                                        <p className="mt-1 text-[10px] text-slate-400 italic">
                                            {t('max_preview_rows_desc')}
                                        </p>
                                    </div>
                                    <div>
                                        <label htmlFor="dataCacheDir" className="block text-sm font-medium text-slate-700 mb-1">{t('data_cache_dir')}</label>
                                        <div className="flex gap-2">
                                            <input
                                                id="dataCacheDir"
                                                type="text"
                                                value={config.dataCacheDir}
                                                onChange={(e) => updateConfig({ dataCacheDir: e.target.value })}
                                                className="flex-1 border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                placeholder="~/VantageData"
                                                autoCapitalize="none"
                                                autoCorrect="off"
                                                spellCheck={false}
                                            />
                                            <button
                                                onClick={handleBrowseDirectory}
                                                className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 border border-slate-300 rounded-md transition-colors whitespace-nowrap"
                                                type="button"
                                            >
                                                {t('browse')}
                                            </button>
                                        </div>
                                        <p className="mt-1 text-[10px] text-slate-400 italic">
                                            {t('data_cache_dir_desc')}
                                        </p>
                                    </div>
                                    
                                    {/* Location Settings */}
                                    <div className="pt-4 border-t border-slate-200">
                                        <div className="flex items-center gap-2 mb-3">
                                            <MapPin className="w-4 h-4 text-blue-500" />
                                            <label className="text-sm font-medium text-slate-700">{t('location_settings') || '位置设置'}</label>
                                        </div>
                                        <p className="text-xs text-slate-500 mb-3">
                                            {t('location_settings_desc') || '设置您的位置，用于天气查询、附近地点等位置相关功能'}
                                        </p>
                                        <div className="grid grid-cols-2 gap-3">
                                            <div>
                                                <label className="block text-xs font-medium text-slate-600 mb-1">{t('country') || '国家或地区'}</label>
                                                <select
                                                    value={config.location?.country || ''}
                                                    onChange={(e) => {
                                                        const selectedCountry = countries.find(c => c.nameEn === e.target.value);
                                                        updateConfig({
                                                            location: {
                                                                country: e.target.value,
                                                                city: '',
                                                                latitude: 0,
                                                                longitude: 0
                                                            }
                                                        });
                                                    }}
                                                    className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                >
                                                    <option value="">{t('select_country') || '选择国家或地区'}</option>
                                                    {countries.map(country => (
                                                        <option key={country.code} value={country.nameEn}>
                                                            {getCountryDisplayName(country, config.language === '简体中文' ? 'zh' : 'en')}
                                                        </option>
                                                    ))}
                                                </select>
                                            </div>
                                            <div>
                                                <label className="block text-xs font-medium text-slate-600 mb-1">{t('city') || '城市'}</label>
                                                <select
                                                    value={config.location?.city || ''}
                                                    onChange={(e) => {
                                                        const selectedCountry = countries.find(c => c.nameEn === config.location?.country);
                                                        const selectedCity = selectedCountry?.cities.find(city => city.nameEn === e.target.value);
                                                        updateConfig({
                                                            location: {
                                                                ...config.location,
                                                                city: e.target.value,
                                                                latitude: selectedCity?.lat || 0,
                                                                longitude: selectedCity?.lng || 0
                                                            }
                                                        });
                                                    }}
                                                    className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                    disabled={!config.location?.country}
                                                >
                                                    <option value="">{t('select_city') || '选择城市'}</option>
                                                    {config.location?.country && countries
                                                        .find(c => c.nameEn === config.location?.country)
                                                        ?.cities.map(city => (
                                                            <option key={city.nameEn} value={city.nameEn}>
                                                                {getCityDisplayName(city, config.language === '简体中文' ? 'zh' : 'en')}
                                                            </option>
                                                        ))
                                                    }
                                                </select>
                                            </div>
                                        </div>
                                        {config.location?.city && (
                                            <p className="mt-2 text-xs text-green-600 flex items-center gap-1">
                                                <CheckCircle className="w-3 h-3" />
                                                {t('location_set') || '位置已设置'}: {config.location.city}, {config.location.country}
                                            </p>
                                        )}
                                    </div>
                                </div>
                            </div>
                        )}
                        {activeTab === 'session' && (
                            <div className="space-y-6">
                                <h3 className="text-lg font-semibold text-slate-800 border-b border-slate-200 pb-2">{t('session_management')}</h3>
                                <p className="text-sm text-slate-500">{t('session_management_desc')}</p>
                                <div className="space-y-4">
                                    {/* Enable Memory */}
                                    <div className="flex items-center justify-between py-3 border-b border-slate-100">
                                        <div className="flex-1">
                                            <span className="block text-sm font-medium text-slate-700">{t('enable_memory')}</span>
                                            <span className="block text-xs text-slate-500 mt-1">{t('enable_memory_desc')}</span>
                                        </div>
                                        <input
                                            type="checkbox"
                                            checked={config.enableMemory}
                                            onChange={(e) => updateConfig({ enableMemory: e.target.checked })}
                                            className="w-4 h-4 text-blue-600 focus:ring-2 focus:ring-blue-500 rounded"
                                        />
                                    </div>
                                    
                                    {/* Auto Analysis Suggestions */}
                                    <div className="flex items-center justify-between py-3 border-b border-slate-100">
                                        <div className="flex-1">
                                            <span className="block text-sm font-medium text-slate-700">{t('auto_analysis_suggestions')}</span>
                                            <span className="block text-xs text-slate-500 mt-1">{t('auto_analysis_suggestions_desc')}</span>
                                        </div>
                                        <input
                                            type="checkbox"
                                            checked={config.autoAnalysisSuggestions !== false}
                                            onChange={(e) => updateConfig({ autoAnalysisSuggestions: e.target.checked })}
                                            className="w-4 h-4 text-blue-600 focus:ring-2 focus:ring-blue-500 rounded"
                                        />
                                    </div>
                                    
                                    {/* Auto Intent Understanding */}
                                    <div className="flex items-center justify-between py-3 border-b border-slate-100">
                                        <div className="flex-1">
                                            <label className="text-sm font-medium text-slate-700">
                                                {t('auto_intent_understanding')}
                                            </label>
                                            <p className="text-xs text-slate-500 mt-1">
                                                {t('auto_intent_understanding_desc')}
                                            </p>
                                        </div>
                                        <input
                                            type="checkbox"
                                            checked={config.autoIntentUnderstanding !== false}
                                            onChange={(e) => {
                                                const newValue = e.target.checked;
                                                updateConfig({ autoIntentUnderstanding: newValue });
                                                
                                                // Show warning toast when user disables intent understanding
                                                if (!newValue) {
                                                    setToast({
                                                        message: t('intent_understanding_disabled_warning') || '建议仅专业用户关闭意图理解功能',
                                                        type: 'warning'
                                                    });
                                                }
                                            }}
                                            className="w-4 h-4 text-blue-600 focus:ring-2 focus:ring-blue-500 rounded"
                                        />
                                    </div>
                                    
                                    {/* Max Concurrent Analysis */}
                                    <div className="py-3 border-b border-slate-100">
                                        <label htmlFor="maxConcurrentAnalysis" className="block text-sm font-medium text-slate-700 mb-1">{t('max_concurrent_analysis')}</label>
                                        <input
                                            id="maxConcurrentAnalysis"
                                            type="number"
                                            value={config.maxConcurrentAnalysis || 5}
                                            onChange={(e) => {
                                                const value = parseInt(e.target.value) || 5;
                                                const clampedValue = Math.max(1, Math.min(10, value));
                                                updateConfig({ maxConcurrentAnalysis: clampedValue });
                                            }}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            min="1"
                                            max="10"
                                        />
                                        <p className="mt-1 text-xs text-slate-500">
                                            {t('max_concurrent_analysis_hint')}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        )}
                        {activeTab === 'search' && (
                            <div className="space-y-6">
                                <div className="flex items-center justify-between border-b border-slate-200 pb-4">
                                    <div>
                                        <h3 className="text-lg font-semibold text-slate-800">{t('search_api_config')}</h3>
                                        <p className="text-sm text-slate-500 mt-1">{t('search_api_config_desc')}</p>
                                    </div>
                                </div>

                                {/* Search API Selection */}
                                <div className="space-y-4">
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-3">
                                            {t('select_search_api')}
                                        </label>
                                        <div className="space-y-3">
                                            {(config.searchAPIs || getDefaultSearchAPIs()).map((api: SearchAPIConfig) => {
                                                const isTestingThis = testingSearchAPI === api.id;
                                                const testResult = searchAPITestResults[api.id];
                                                const isActive = config.activeSearchAPI === api.id;
                                                
                                                return (
                                                    <div key={api.id} className={`border-2 rounded-lg p-4 ${
                                                        isActive ? 'border-blue-500 bg-blue-50' : 'border-slate-200'
                                                    }`}>
                                                        <div className="flex items-start justify-between">
                                                            <div className="flex items-start gap-3 flex-1">
                                                                <input
                                                                    type="radio"
                                                                    name="activeSearchAPI"
                                                                    checked={isActive}
                                                                    onChange={() => updateConfig({ activeSearchAPI: api.id })}
                                                                    className="mt-1 w-4 h-4 text-blue-600 focus:ring-2 focus:ring-blue-500"
                                                                />
                                                                <div className="flex-1">
                                                                    <div className="flex items-center gap-2 mb-1">
                                                                        <h4 className="text-sm font-semibold text-slate-900">{api.name}</h4>
                                                                        {api.tested && (
                                                                            <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium text-green-700 bg-green-100 rounded-full">
                                                                                <CheckCircle className="w-3 h-3" />
                                                                                {t('tested_badge')}
                                                                            </span>
                                                                        )}
                                                                        {isActive && (
                                                                            <span className="inline-flex items-center px-2 py-0.5 text-xs font-medium text-blue-700 bg-blue-100 rounded-full">
                                                                                {t('active_badge')}
                                                                            </span>
                                                                        )}
                                                                    </div>
                                                                    <p className="text-xs text-slate-500 mb-3">{api.description}</p>
                                                                    
                                                                    {/* API Key Input for Serper and UAPI Pro */}
                                                                    {(api.id === 'serper' || api.id === 'uapi_pro') && (
                                                                        <div className="mb-3">
                                                                            <label className="block text-xs font-medium text-slate-700 mb-1">
                                                                                {t('api_key')} {api.id === 'serper' && <span className="text-red-500">*</span>}
                                                                                {api.id === 'uapi_pro' && <span className="text-slate-400">({t('optional')})</span>}
                                                                            </label>
                                                                            <div className="flex gap-2">
                                                                                <input
                                                                                    type="password"
                                                                                    value={api.apiKey || ''}
                                                                                    onChange={(e) => updateSearchAPIConfig(api.id, 'apiKey', e.target.value)}
                                                                                    placeholder={api.id === 'uapi_pro' ? t('enter_api_key_optional_placeholder', api.name) : t('enter_api_key_placeholder', api.name)}
                                                                                    className="flex-1 px-2 py-1.5 text-xs border border-slate-300 rounded focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                                                                                />
                                                                                <a
                                                                                    href={api.id === 'serper' ? 'https://serper.dev' : 'https://uapis.cn'}
                                                                                    target="_blank"
                                                                                    rel="noopener noreferrer"
                                                                                    className="px-3 py-1.5 bg-green-600 text-white text-xs rounded hover:bg-green-700 transition-colors whitespace-nowrap"
                                                                                    title={t('get_api_key')}
                                                                                >
                                                                                    {t('get_key_button')}
                                                                                </a>
                                                                            </div>
                                                                        </div>
                                                                    )}
                                                                    
                                                                    {/* Test Button */}
                                                                    <button
                                                                        onClick={() => handleTestSearchAPI(api.id)}
                                                                        disabled={isTestingThis}
                                                                        className="px-3 py-1.5 text-xs bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-slate-400 disabled:cursor-not-allowed transition-colors"
                                                                    >
                                                                        {isTestingThis ? t('testing') : t('test_connection_button')}
                                                                    </button>
                                                                    
                                                                    {/* Test Result */}
                                                                    {testResult && (
                                                                        <div className={`mt-2 p-2 rounded text-xs ${
                                                                            testResult.success ? 'bg-green-50' : 'bg-red-50'
                                                                        }`}>
                                                                            <div className="flex items-start gap-2">
                                                                                {testResult.success ? (
                                                                                    <CheckCircle className="text-green-600 flex-shrink-0 mt-0.5" size={14} />
                                                                                ) : (
                                                                                    <AlertCircle className="text-red-600 flex-shrink-0 mt-0.5" size={14} />
                                                                                )}
                                                                                <span className={testResult.success ? 'text-green-800' : 'text-red-800'}>
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

                                    {/* Info Box */}
                                    <div className="bg-blue-50 p-4 rounded-lg">
                                        <div className="flex items-start gap-3">
                                            <AlertCircle className="text-blue-600 flex-shrink-0 mt-0.5" size={20} />
                                            <div className="text-sm text-blue-800">
                                                <p className="font-medium mb-1">{t('search_api_info_title')}</p>
                                                <ul className="list-disc list-inside space-y-1 text-xs">
                                                    <li>{t('serper_info')}</li>
                                                    <li>{t('uapi_pro_info')}</li>
                                                </ul>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        )}
                        {activeTab === 'mcp' && (
                            <div className="space-y-6">
                                <div className="flex items-center justify-between border-b border-slate-200 pb-4">
                                    <div>
                                        <h3 className="text-lg font-semibold text-slate-800">{t('mcp_services_title')}</h3>
                                        <p className="text-sm text-slate-500 mt-1">{t('mcp_services_description')}</p>
                                    </div>
                                    <button
                                        onClick={() => {
                                            setEditingMcpService(null);
                                            setMcpModalOpen(true);
                                        }}
                                        className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors shadow-sm"
                                    >
                                        <Plus className="w-4 h-4" />
                                        {t('add_mcp_service')}
                                    </button>
                                </div>

                                {/* MCP Services List */}
                                <div className="space-y-3">
                                    {(!config.mcpServices || config.mcpServices.length === 0) ? (
                                        <div className="text-center py-12 bg-slate-50 rounded-lg border-2 border-dashed border-slate-200">
                                            <Server className="w-12 h-12 text-slate-300 mx-auto mb-3" />
                                            <p className="text-sm text-slate-500">{t('no_mcp_services')}</p>
                                            <button
                                                onClick={() => {
                                                    setEditingMcpService(null);
                                                    setMcpModalOpen(true);
                                                }}
                                                className="mt-4 text-sm text-blue-600 hover:text-blue-700 font-medium"
                                            >
                                                {t('add_mcp_service')}
                                            </button>
                                        </div>
                                    ) : (
                                        config.mcpServices.map((service: MCPService) => (
                                            <div
                                                key={service.id}
                                                className="flex items-start gap-4 p-4 bg-white border border-slate-200 rounded-lg hover:shadow-md transition-shadow"
                                            >
                                                <div className={`p-2 rounded-lg ${service.enabled ? 'bg-green-100' : 'bg-slate-100'}`}>
                                                    {service.enabled ? (
                                                        <Power className="w-5 h-5 text-green-600" />
                                                    ) : (
                                                        <PowerOff className="w-5 h-5 text-slate-400" />
                                                    )}
                                                </div>
                                                <div className="flex-1 min-w-0">
                                                    <div className="flex items-start justify-between gap-2">
                                                        <div className="flex-1 min-w-0">
                                                            <div className="flex items-center gap-2">
                                                                <h4 className="text-sm font-semibold text-slate-800 truncate">
                                                                    {service.name}
                                                                </h4>
                                                                {/* Test status badge */}
                                                                {service.tested ? (
                                                                    <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium text-green-700 bg-green-100 rounded-full">
                                                                        <CheckCircle className="w-3 h-3" />
                                                                        {t('tested')}
                                                                    </span>
                                                                ) : (
                                                                    <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium text-amber-700 bg-amber-100 rounded-full">
                                                                        <AlertCircle className="w-3 h-3" />
                                                                        {t('not_tested')}
                                                                    </span>
                                                                )}
                                                            </div>
                                                            {service.description && (
                                                                <p className="text-xs text-slate-500 mt-1">
                                                                    {service.description}
                                                                </p>
                                                            )}
                                                            <p className="text-xs text-slate-400 mt-2 font-mono truncate">
                                                                {service.url}
                                                            </p>
                                                            {/* Warning if enabled but not tested */}
                                                            {service.enabled && !service.tested && (
                                                                <p className="text-xs text-amber-600 mt-2 flex items-center gap-1">
                                                                    <AlertCircle className="w-3 h-3" />
                                                                    {t('mcp_not_tested_warning')}
                                                                </p>
                                                            )}
                                                        </div>
                                                        <div className="flex items-center gap-1 flex-shrink-0">
                                                            <button
                                                                onClick={() => {
                                                                    setEditingMcpService(service);
                                                                    setMcpModalOpen(true);
                                                                }}
                                                                className="p-2 text-slate-600 hover:bg-slate-100 rounded-lg transition-colors"
                                                                title={t('edit_mcp_service')}
                                                            >
                                                                <Edit2 className="w-4 h-4" />
                                                            </button>
                                                            <button
                                                                onClick={() => {
                                                                    const newServices = config.mcpServices.filter(
                                                                        (s: MCPService) => s.id !== service.id
                                                                    );
                                                                    updateConfig({ mcpServices: newServices });
                                                                }}
                                                                className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                                                                title={t('delete_mcp_service')}
                                                            >
                                                                <Trash2 className="w-4 h-4" />
                                                            </button>
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>
                                        ))
                                    )}
                                </div>
                            </div>
                        )}
                        {activeTab === 'network' && (
                            <div className="space-y-6">
                                <div className="flex items-center justify-between border-b border-slate-200 pb-4">
                                    <div>
                                        <h3 className="text-lg font-semibold text-slate-800">{t('network_settings')}</h3>
                                        <p className="text-sm text-slate-500 mt-1">{t('network_settings_description')}</p>
                                    </div>
                                </div>

                                <NetworkSettings config={config} updateConfig={updateConfig} />
                            </div>
                        )}
                        {activeTab === 'runenv' && <RunEnvSettings config={config} setConfig={setConfig} updateConfig={updateConfig} />}
                        {activeTab === 'skills' && <SkillsSettings onOpenSkills={onOpenSkills} />}
                        {activeTab === 'intent' && <IntentEnhancementSettings config={config} updateConfig={updateConfig} />}
                    </div>

                    {/* Footer */}
                    <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-3">
                        <button onClick={onClose} className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-md">
                            {t('cancel')}
                        </button>
                        <button onClick={handleSave} className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md shadow-sm">
                            {t('save_changes')}
                        </button>
                    </div>
                </div>
            </div>
            {toast && (
                <Toast
                    message={toast.message}
                    type={toast.type}
                    onClose={() => setToast(null)}
                />
            )}
            <MCPServiceModal
                isOpen={mcpModalOpen}
                service={editingMcpService}
                onClose={() => {
                    setMcpModalOpen(false);
                    setEditingMcpService(null);
                }}
                onSave={(service: MCPService) => {
                    console.log('[MCP] Saving service:', service);
                    if (editingMcpService) {
                        // Update existing service
                        const newServices = config.mcpServices.map((s: MCPService) =>
                            s.id === service.id ? service : s
                        );
                        console.log('[MCP] Updated services:', newServices);
                        updateConfig({ mcpServices: newServices });
                    } else {
                        // Add new service
                        const newServices = [...(config.mcpServices || []), service];
                        console.log('[MCP] Added service, new list:', newServices);
                        updateConfig({ mcpServices: newServices });
                    }
                }}
            />
        </div>
    );
};

interface NetworkSettingsProps {
    config: configModel.Config;
    updateConfig: (updates: Partial<configModel.Config>) => void;
}

const NetworkSettings: React.FC<NetworkSettingsProps> = ({ config, updateConfig }) => {
    const { t } = useLanguage();
    const [testing, setTesting] = useState(false);
    const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

    // Initialize proxy config if it doesn't exist
    const proxyConfig = config.proxyConfig || {
        enabled: false,
        protocol: 'http',
        host: '',
        port: 0,
        username: '',
        password: '',
        tested: false
    };

    const updateProxyConfig = (updates: Partial<typeof proxyConfig>) => {
        const newProxyConfig = { ...proxyConfig, ...updates };
        
        // Clear tested flag if connection details change
        if (updates.protocol || updates.host || updates.port || updates.username || updates.password) {
            newProxyConfig.tested = false;
        }
        
        updateConfig({ proxyConfig: newProxyConfig });
    };

    const handleTestProxy = async () => {
        if (!proxyConfig.host || proxyConfig.port <= 0) {
            setTestResult({ success: false, message: t('proxy_test_failed') + ': Host and port are required' });
            return;
        }

        setTesting(true);
        setTestResult(null);
        
        try {
            // @ts-ignore - TestProxy is defined in App.go
            const result = await window.go.main.App.TestProxy(proxyConfig);
            setTestResult(result);
            
            if (result.success) {
                updateProxyConfig({ tested: true });
            }
        } catch (err) {
            setTestResult({ success: false, message: String(err) });
        } finally {
            setTesting(false);
        }
    };

    return (
        <div className="space-y-6">
            {/* Enable Proxy Toggle */}
            <div className="flex items-center justify-between p-4 bg-slate-50 rounded-lg border border-slate-200">
                <div className="flex-1">
                    <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-slate-700">{t('proxy_enabled')}</span>
                        {proxyConfig.tested && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium text-green-700 bg-green-100 rounded-full">
                                <CheckCircle className="w-3 h-3" />
                                {t('proxy_tested')}
                            </span>
                        )}
                        {!proxyConfig.tested && proxyConfig.host && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium text-amber-700 bg-amber-100 rounded-full">
                                <AlertCircle className="w-3 h-3" />
                                {t('proxy_not_tested')}
                            </span>
                        )}
                    </div>
                    <p className="text-xs text-slate-500 mt-1">
                        {proxyConfig.enabled && !proxyConfig.tested && t('proxy_test_required')}
                    </p>
                </div>
                <input
                    type="checkbox"
                    checked={proxyConfig.enabled}
                    onChange={(e) => updateProxyConfig({ enabled: e.target.checked })}
                    disabled={!proxyConfig.tested && proxyConfig.host !== ''}
                    className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                />
            </div>

            {/* Proxy Configuration */}
            <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                            {t('proxy_protocol')}
                        </label>
                        <select
                            value={proxyConfig.protocol}
                            onChange={(e) => updateProxyConfig({ protocol: e.target.value })}
                            className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        >
                            <option value="http">HTTP</option>
                            <option value="https">HTTPS</option>
                            <option value="socks5">SOCKS5</option>
                        </select>
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                            {t('proxy_port')}
                        </label>
                        <input
                            type="number"
                            value={proxyConfig.port || ''}
                            onChange={(e) => updateProxyConfig({ port: parseInt(e.target.value) || 0 })}
                            placeholder="8080"
                            min="1"
                            max="65535"
                            className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                    </div>
                </div>

                <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">
                        {t('proxy_host')}
                    </label>
                    <input
                        type="text"
                        value={proxyConfig.host}
                        onChange={(e) => updateProxyConfig({ host: e.target.value })}
                        placeholder="proxy.example.com or 192.168.1.1"
                        className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                </div>

                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                            {t('proxy_username')}
                        </label>
                        <input
                            type="text"
                            value={proxyConfig.username}
                            onChange={(e) => updateProxyConfig({ username: e.target.value })}
                            placeholder="username"
                            autoComplete="off"
                            className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                            {t('proxy_password')}
                        </label>
                        <input
                            type="password"
                            value={proxyConfig.password}
                            onChange={(e) => updateProxyConfig({ password: e.target.value })}
                            placeholder="password"
                            autoComplete="off"
                            className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                    </div>
                </div>

                {/* Test Button */}
                <div className="pt-2 flex items-center gap-4">
                    <button
                        onClick={handleTestProxy}
                        disabled={testing || !proxyConfig.host || proxyConfig.port <= 0}
                        className={`px-4 py-2 text-sm font-semibold rounded-md transition-colors ${
                            testing || !proxyConfig.host || proxyConfig.port <= 0
                                ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                                : 'bg-blue-600 text-white hover:bg-blue-700'
                        }`}
                    >
                        {testing ? t('testing_proxy') : t('test_proxy')}
                    </button>

                    {testResult && (
                        <div className={`text-sm font-medium animate-in fade-in slide-in-from-left-1 ${
                            testResult.success ? 'text-green-600' : 'text-red-600'
                        }`}>
                            {testResult.success ? `✓ ${t('proxy_test_success')}` : `✗ ${testResult.message}`}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

interface RunEnvSettingsProps {
    config: configModel.Config;
    setConfig: (config: configModel.Config) => void;
    updateConfig: (updates: Partial<configModel.Config>) => void;
}

const RunEnvSettings: React.FC<RunEnvSettingsProps> = ({ config, setConfig, updateConfig }) => {
    const { t } = useLanguage();
    const [envs, setEnvs] = useState<agent.PythonEnvironment[]>([]);
    const [loading, setLoading] = useState(false);
    const [validation, setValidation] = useState<agent.PythonValidationResult | null>(null);
    const [validating, setValidating] = useState(false);
    const [installing, setInstalling] = useState(false);
    const [creatingEnv, setCreatingEnv] = useState(false);
    const [showCreateButton, setShowCreateButton] = useState(false);
    const [diagnosing, setDiagnosing] = useState(false);
    const [notification, setNotification] = useState<{ type: 'success' | 'error' | 'info', message: string } | null>(null);

    // Auto-hide notification after 5 seconds for success/info, 10 seconds for error
    useEffect(() => {
        if (notification) {
            const timeout = setTimeout(() => {
                setNotification(null);
            }, notification.type === 'error' ? 10000 : 5000);

            return () => clearTimeout(timeout);
        }
    }, [notification]);

    const loadEnvironments = async () => {
        setLoading(true);
        try {
            const environments = await GetPythonEnvironments();
            setEnvs(environments);

            // Check if we should show the "Create VantageData Environment" button
            const hasVirtualEnvSupport = environments.some(env =>
                env.type.toLowerCase().includes('conda') ||
                env.type.toLowerCase().includes('virtualenv') ||
                env.type.toLowerCase().includes('venv')
            );

            const hasVantageDataEnv = await CheckVantageDataEnvironmentExists();
            setShowCreateButton(hasVirtualEnvSupport && !hasVantageDataEnv);
        } catch (error) {
            console.error('Failed to load environments:', error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadEnvironments();
    }, []);

    useEffect(() => {
        if (config.pythonPath) {
            setValidating(true);
            ValidatePython(config.pythonPath)
                .then(setValidation)
                .catch(console.error)
                .finally(() => setValidating(false));
        } else {
            setValidation(null);
        }
    }, [config.pythonPath]);

    const handleInstallPackages = async () => {
        if (!config.pythonPath || !validation?.missingPackages || validation.missingPackages.length === 0) {
            return;
        }

        setInstalling(true);
        try {
            await InstallPythonPackages(config.pythonPath, validation.missingPackages);

            // Re-validate the environment after installation
            setValidating(true);
            const newValidation = await ValidatePython(config.pythonPath);
            setValidation(newValidation);

            if (newValidation.missingPackages && newValidation.missingPackages.length === 0) {
                setNotification({ type: 'success', message: t('packages_install_success') });
            } else {
                setNotification({ type: 'info', message: t('packages_install_partial', newValidation.missingPackages?.length || 0) });
            }
        } catch (error) {
            console.error('Package installation failed:', error);
            setNotification({ type: 'error', message: t('packages_install_failed', String(error)) });
        } finally {
            setInstalling(false);
            setValidating(false);
        }
    };

    const handleCreateVantageDataEnvironment = async () => {
        setCreatingEnv(true);
        try {
            const pythonPath = await CreateVantageDataEnvironment();

            // Refresh the environment list
            await loadEnvironments();

            // Auto-select the new environment
            updateConfig({ pythonPath });

            setNotification({ type: 'success', message: t('env_create_success') });
        } catch (error) {
            console.error('Environment creation failed:', error);

            // Show detailed error message with suggestions
            const errorMessage = String(error);
            let userMessage = t('env_create_failed_title') + '\n\n';

            if (errorMessage.includes('No suitable Python interpreter found')) {
                // Extract the detailed diagnostic information from the error
                const diagnosticStart = errorMessage.indexOf('Detection attempts:');
                const diagnosticEnd = errorMessage.indexOf('To resolve this issue');

                if (diagnosticStart !== -1 && diagnosticEnd !== -1) {
                    const diagnosticInfo = errorMessage.substring(diagnosticStart, diagnosticEnd);
                    userMessage += t('diagnostic_info') + '\n' + diagnosticInfo + '\n\n';
                }

                userMessage += t('solution') + '\n\n';
                userMessage += t('install_anaconda') + '\n';
                userMessage += t('anaconda_download') + '\n';
                userMessage += t('anaconda_verify') + '\n\n';
                userMessage += t('install_python') + '\n';
                userMessage += t('python_download') + '\n';
                userMessage += t('python_add_path') + '\n';
                userMessage += t('python_venv_included') + '\n\n';
                userMessage += t('verify_installation') + '\n';
                userMessage += t('verify_terminal') + '\n';
                userMessage += t('verify_conda') + '\n';
                userMessage += t('verify_python') + '\n';
                userMessage += t('verify_venv') + '\n\n';
                userMessage += t('restart_app_retry') + '\n\n';
                userMessage += t('contact_support');
            } else if (errorMessage.includes('conda')) {
                userMessage += t('conda_env_failed') + '\n\n';
                userMessage += t('conda_solution') + '\n';
                userMessage += t('conda_verify_install') + '\n';
                userMessage += t('conda_check_path') + '\n';
                userMessage += t('conda_try_version') + '\n';
                userMessage += t('restart_terminal_app') + '\n\n';
                userMessage += t('error_details', errorMessage);
            } else if (errorMessage.includes('venv')) {
                userMessage += t('venv_failed') + '\n\n';
                userMessage += t('venv_solution') + '\n';
                userMessage += t('venv_check_version') + '\n';
                userMessage += t('venv_check_module') + '\n';
                userMessage += t('venv_try_help') + '\n';
                userMessage += t('venv_reinstall') + '\n\n';
                userMessage += t('error_details', errorMessage);
            } else {
                userMessage += t('error_details', errorMessage) + '\n\n';
                userMessage += t('general_solution') + '\n';
                userMessage += t('check_network') + '\n';
                userMessage += t('check_disk_space') + '\n';
                userMessage += t('run_as_admin') + '\n';
                userMessage += t('restart_retry');
            }

            // Show error notification instead of blocking alert
            setNotification({ type: 'error', message: userMessage });
        } finally {
            setCreatingEnv(false);
        }
    };

    const handleDiagnosePython = async () => {
        setDiagnosing(true);
        try {
            const diagnostics = await DiagnosePythonInstallation();

            // Format diagnostic information for display
            let diagnosticText = t('python_diagnostic_report') + '\n\n';

            // System info
            diagnosticText += t('system_info', diagnostics.os, diagnostics.arch) + '\n\n';

            // Conda info
            const conda = diagnostics.conda as any;
            diagnosticText += t('conda_detection') + '\n';
            if (conda.found) {
                diagnosticText += t('conda_found', conda.path) + '\n';
                if (conda.working) {
                    diagnosticText += t('conda_version', conda.version) + '\n';
                } else {
                    diagnosticText += t('conda_failed', conda.error) + '\n';
                }
            } else {
                diagnosticText += t('conda_not_found', conda.error) + '\n';
            }
            diagnosticText += '\n';

            // Python commands info
            const pythonCommands = diagnostics.python_commands as any;
            diagnosticText += t('python_cmd_detection') + '\n';
            for (const [cmd, info] of Object.entries(pythonCommands)) {
                const cmdInfo = info as any;
                if (cmdInfo.found) {
                    diagnosticText += t('cmd_found', cmd, cmdInfo.path) + '\n';
                    if (cmdInfo.working) {
                        diagnosticText += t('cmd_version', cmdInfo.version) + '\n';
                        if (cmdInfo.venv_support) {
                            diagnosticText += t('cmd_venv_support') + '\n';
                        } else {
                            diagnosticText += t('cmd_venv_no_support', cmdInfo.venv_error) + '\n';
                        }
                    } else {
                        diagnosticText += t('cmd_failed', cmdInfo.error) + '\n';
                    }
                } else {
                    diagnosticText += t('cmd_not_found', cmd) + '\n';
                }
            }
            diagnosticText += '\n';

            // Existing environments
            const existingEnvs = diagnostics.existing_environments as any[];
            diagnosticText += t('existing_envs', existingEnvs.length) + '\n';
            existingEnvs.forEach((env, index) => {
                diagnosticText += `  ${index + 1}. ${env.type} - ${env.version}\n`;
                diagnosticText += t('env_path', env.path) + '\n';
            });

            // Show diagnostic results
            const textarea = document.createElement('textarea');
            textarea.value = diagnosticText;
            textarea.style.width = '100%';
            textarea.style.height = '400px';
            textarea.style.fontFamily = 'monospace';
            textarea.style.fontSize = '12px';
            textarea.readOnly = true;

            const modal = document.createElement('div');
            modal.style.position = 'fixed';
            modal.style.top = '0';
            modal.style.left = '0';
            modal.style.width = '100%';
            modal.style.height = '100%';
            modal.style.backgroundColor = 'rgba(0,0,0,0.5)';
            modal.style.display = 'flex';
            modal.style.alignItems = 'center';
            modal.style.justifyContent = 'center';
            modal.style.zIndex = '10000';

            const content = document.createElement('div');
            content.style.backgroundColor = 'white';
            content.style.padding = '20px';
            content.style.borderRadius = '8px';
            content.style.maxWidth = '80%';
            content.style.maxHeight = '80%';
            content.style.overflow = 'auto';

            const title = document.createElement('h3');
            title.textContent = t('python_diagnostic_report');
            title.style.marginTop = '0';

            const closeBtn = document.createElement('button');
            closeBtn.textContent = t('close');
            closeBtn.style.marginTop = '10px';
            closeBtn.style.padding = '8px 16px';
            closeBtn.style.backgroundColor = '#3b82f6';
            closeBtn.style.color = 'white';
            closeBtn.style.border = 'none';
            closeBtn.style.borderRadius = '4px';
            closeBtn.style.cursor = 'pointer';
            closeBtn.onclick = () => document.body.removeChild(modal);

            const copyBtn = document.createElement('button');
            copyBtn.textContent = t('copy_to_clipboard');
            copyBtn.style.marginTop = '10px';
            copyBtn.style.marginLeft = '10px';
            copyBtn.style.padding = '8px 16px';
            copyBtn.style.backgroundColor = '#10b981';
            copyBtn.style.color = 'white';
            copyBtn.style.border = 'none';
            copyBtn.style.borderRadius = '4px';
            copyBtn.style.cursor = 'pointer';
            copyBtn.onclick = () => {
                navigator.clipboard.writeText(diagnosticText);
                copyBtn.textContent = t('copied');
                setTimeout(() => copyBtn.textContent = t('copy_to_clipboard'), 2000);
            };

            content.appendChild(title);
            content.appendChild(textarea);
            content.appendChild(closeBtn);
            content.appendChild(copyBtn);
            modal.appendChild(content);
            document.body.appendChild(modal);

        } catch (error) {
            setNotification({ type: 'error', message: t('diagnostic_failed', String(error)) });
        } finally {
            setDiagnosing(false);
        }
    };

    // Check if current pythonPath is in the list
    const isKnownEnv = envs.some(e => e.path === config.pythonPath);

    return (
        <div className="space-y-6">
            <h3 className="text-lg font-semibold text-slate-800 border-b border-slate-200 pb-2">{t('python_runtime_env')}</h3>
            <div className="space-y-4">
                <div>
                    <label htmlFor="pythonPath" className="block text-sm font-medium text-slate-700 mb-1">{t('select_python_env')}</label>
                    {loading ? (
                        <div className="text-sm text-slate-500 animate-pulse">{t('scanning_python_envs')}</div>
                    ) : (
                        <select
                            id="pythonPath"
                            value={config.pythonPath}
                            onChange={(e) => updateConfig({ pythonPath: e.target.value })}
                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                        >
                            <option value="">{t('select_env_placeholder')}</option>
                            {config.pythonPath && !isKnownEnv && (
                                <option value={config.pythonPath}>
                                    {config.pythonPath} ({t('saved_env')})
                                </option>
                            )}
                            {envs.map((env) => (
                                <option key={env.path} value={env.path}>
                                    {env.type} - {env.version} ({env.path})
                                </option>
                            ))}
                        </select>
                    )}

                    {/* Create VantageData Environment Button */}
                    {showCreateButton && (
                        <div className="mt-3">
                            <button
                                onClick={handleCreateVantageDataEnvironment}
                                disabled={creatingEnv || loading}
                                className="flex items-center gap-2 px-4 py-2 bg-green-600 text-white text-sm rounded-md hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                            >
                                {creatingEnv ? (
                                    <>
                                        <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                                        {t('creating_env')}
                                    </>
                                ) : (
                                    <>
                                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                                        </svg>
                                        {t('create_vantagedata_env')}
                                    </>
                                )}
                            </button>
                            <p className="mt-1 text-xs text-slate-500">
                                {t('create_env_desc')}
                            </p>
                        </div>
                    )}

                    <p className="mt-1 text-[10px] text-slate-400 italic">
                        {t('select_python_interpreter_desc')}
                    </p>

                    {/* Python Diagnostic Button */}
                    <div className="mt-3">
                        <button
                            onClick={handleDiagnosePython}
                            disabled={diagnosing || loading}
                            className="flex items-center gap-2 px-3 py-1 bg-slate-100 text-slate-700 text-xs rounded-md hover:bg-slate-200 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            {diagnosing ? (
                                <>
                                    <div className="w-3 h-3 border-2 border-slate-400 border-t-transparent rounded-full animate-spin"></div>
                                    {t('diagnosing')}
                                </>
                            ) : (
                                <>
                                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                    </svg>
                                    {t('python_diagnostic')}
                                </>
                            )}
                        </button>
                        <p className="mt-1 text-xs text-slate-500">
                            {t('python_diagnostic_desc')}
                        </p>
                    </div>
                </div>

                {creatingEnv && (
                    <div className="p-4 bg-blue-50 border border-blue-200 rounded-lg">
                        <div className="flex items-center gap-3">
                            <div className="w-5 h-5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
                            <div>
                                <p className="text-sm font-medium text-blue-800">{t('creating_vantagedata_env')}</p>
                                <p className="text-xs text-blue-600">{t('creating_env_wait')}</p>
                            </div>
                        </div>
                    </div>
                )}

                {validating && (
                    <div className="text-sm text-blue-600 animate-pulse">{t('validating_env')}</div>
                )}

                {validation && !validating && (
                    <div className={`p-4 rounded-lg border ${validation.valid ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'}`}>
                        <div className="flex items-center justify-between mb-2">
                            <span className={`font-semibold ${validation.valid ? 'text-green-800' : 'text-red-800'}`}>
                                {validation.valid ? '✓ ' + t('env_valid') : '✗ ' + t('env_invalid')}
                            </span>
                            <span className="text-xs text-slate-500">{validation.version}</span>
                        </div>

                        {!validation.valid && validation.error && (
                            <div className="text-sm text-red-700 mb-2">{validation.error}</div>
                        )}

                        {validation.missingPackages && validation.missingPackages.length > 0 && (
                            <div>
                                <div className="flex items-center justify-between mb-2">
                                    <span className="text-sm font-medium text-amber-700">{t('missing_packages')}</span>
                                    <button
                                        onClick={handleInstallPackages}
                                        disabled={installing || validating}
                                        className="px-3 py-1 bg-blue-600 text-white text-xs rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                                    >
                                        {installing ? t('installing_packages') : t('install_packages')}
                                    </button>
                                </div>
                                <ul className="list-disc list-inside text-xs text-amber-600 mb-2">
                                    {validation.missingPackages.map(pkg => (
                                        <li key={pkg}>{pkg}</li>
                                    ))}
                                </ul>
                                {installing && (
                                    <div className="text-xs text-blue-600 animate-pulse">
                                        {t('installing_packages_wait')}
                                    </div>
                                )}
                            </div>
                        )}

                        {validation.valid && (!validation.missingPackages || validation.missingPackages.length === 0) && (
                            <div className="text-xs text-green-700">{t('all_packages_installed')}</div>
                        )}
                    </div>
                )}

                {/* Notification Component */}
                {notification && (
                    <div className={`fixed top-4 right-4 max-w-md p-4 rounded-lg shadow-lg border z-50 animate-in slide-in-from-right-2 ${notification.type === 'success' ? 'bg-green-50 border-green-200 text-green-800' :
                        notification.type === 'error' ? 'bg-red-50 border-red-200 text-red-800' :
                            'bg-blue-50 border-blue-200 text-blue-800'
                        }`}>
                        <div className="flex items-start justify-between">
                            <div className="flex items-start gap-3">
                                <div className="flex-shrink-0 mt-0.5">
                                    {notification.type === 'success' && (
                                        <svg className="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                        </svg>
                                    )}
                                    {notification.type === 'error' && (
                                        <svg className="w-5 h-5 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                        </svg>
                                    )}
                                    {notification.type === 'info' && (
                                        <svg className="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                        </svg>
                                    )}
                                </div>
                                <div className="flex-1">
                                    <div className="text-sm font-medium mb-1">
                                        {notification.type === 'success' && t('operation_success')}
                                        {notification.type === 'error' && t('operation_failed')}
                                        {notification.type === 'info' && t('info_message')}
                                    </div>
                                    <div className="text-xs whitespace-pre-line">
                                        {notification.message}
                                    </div>
                                </div>
                            </div>
                            <button
                                onClick={() => setNotification(null)}
                                className="flex-shrink-0 ml-2 text-gray-400 hover:text-gray-600 transition-colors"
                            >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};

// Skills Settings Component
const SkillsSettings: React.FC<{ onOpenSkills?: () => void }> = ({ onOpenSkills }) => {
    const { t } = useLanguage();
    const [skills, setSkills] = useState<SkillInfo[]>([]);
    const [filteredSkills, setFilteredSkills] = useState<SkillInfo[]>([]);
    const [selectedCategory, setSelectedCategory] = useState<string>('all');
    const [searchQuery, setSearchQuery] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [selectedSkill, setSelectedSkill] = useState<SkillInfo | null>(null);

    useEffect(() => {
        loadSkills();
    }, []);

    useEffect(() => {
        filterSkills();
    }, [skills, selectedCategory, searchQuery]);

    const loadSkills = async () => {
        setIsLoading(true);
        try {
            const loadedSkills = await GetSkills() as unknown as SkillInfo[];
            setSkills(loadedSkills || []);
        } catch (error) {
            console.error('Failed to load skills:', error);
        } finally {
            setIsLoading(false);
        }
    };

    const filterSkills = () => {
        let filtered = [...(skills || [])];

        if (selectedCategory !== 'all') {
            filtered = filtered.filter(s => s.category === selectedCategory);
        }

        if (searchQuery.trim()) {
            const query = searchQuery.toLowerCase();
            filtered = filtered.filter(s =>
                s.name.toLowerCase().includes(query) ||
                s.description.toLowerCase().includes(query) ||
                s.keywords.some(k => k.toLowerCase().includes(query)) ||
                s.tags.some(t => t.toLowerCase().includes(query))
            );
        }

        setFilteredSkills(filtered);
    };

    const handleToggleSkill = async (skillId: string, currentlyEnabled: boolean) => {
        try {
            if (currentlyEnabled) {
                await DisableSkill(skillId);
            } else {
                await EnableSkill(skillId);
            }
            await loadSkills();
        } catch (error) {
            console.error('Failed to toggle skill:', error);
        }
    };

    const handleReloadSkills = async () => {
        setIsLoading(true);
        try {
            await ReloadSkills();
            await loadSkills();
        } catch (error) {
            console.error('Failed to reload skills:', error);
        } finally {
            setIsLoading(false);
        }
    };

    const categories = ['all', ...Array.from(new Set((skills || []).map(s => s.category)))];
    const enabledCount = (skills || []).filter(s => s.enabled).length;

    const getCategoryIcon = (category: string) => {
        const icons: { [key: string]: string } = {
            user_analytics: '👥',
            sales_analytics: '💰',
            marketing: '📢',
            product: '📦',
            custom: '🔧',
            all: '📚'
        };
        return icons[category] || '📊';
    };

    const getIconComponent = (iconName: string) => {
        const icons: { [key: string]: React.ReactNode } = {
            users: <Tag className="w-5 h-5" />,
            filter: <Filter className="w-5 h-5" />,
            zap: <Zap className="w-5 h-5" />,
            chart: <BookOpen className="w-5 h-5" />,
        };
        return icons[iconName] || <BookOpen className="w-5 h-5" />;
    };

    return (
        <div className="space-y-4 h-full flex flex-col">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-slate-200 pb-4">
                <div>
                    <h3 className="text-lg font-semibold text-slate-800">Skills 插件管理</h3>
                    <p className="text-sm text-slate-500 mt-1">
                        {skills.length} 个插件 · {enabledCount} 个已启用
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={onOpenSkills}
                        className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors flex items-center gap-2"
                        title="打开 Skills 管理页面"
                    >
                        <Plus className="w-4 h-4" />
                        Skills 管理
                    </button>
                    <button
                        onClick={handleReloadSkills}
                        disabled={isLoading}
                        className="p-2 hover:bg-slate-100 rounded-lg transition-colors text-slate-600 hover:text-blue-600"
                        title="重新加载Skills"
                    >
                        <RefreshCw className={`w-5 h-5 ${isLoading ? 'animate-spin' : ''}`} />
                    </button>
                </div>
            </div>

            {/* Toolbar */}
            <div className="flex flex-col sm:flex-row gap-3">
                {/* Search */}
                <div className="flex-1 relative">
                    <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                    <input
                        type="text"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        placeholder="搜索 Skills..."
                        className="w-full pl-10 pr-4 py-2 bg-white border border-slate-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                    />
                </div>

                {/* Category Filter */}
                <div className="flex gap-2 overflow-x-auto scrollbar-hide">
                    {categories.map(cat => (
                        <button
                            key={cat}
                            onClick={() => setSelectedCategory(cat)}
                            className={`px-3 py-2 rounded-lg text-xs font-medium transition-all whitespace-nowrap flex items-center gap-1.5 ${
                                selectedCategory === cat
                                    ? 'bg-blue-600 text-white shadow-sm'
                                    : 'bg-white text-slate-600 hover:bg-slate-100 border border-slate-200'
                            }`}
                        >
                            <span>{getCategoryIcon(cat)}</span>
                            <span className="capitalize">{cat === 'all' ? '全部' : cat}</span>
                        </button>
                    ))}
                </div>
            </div>

            {/* Skills List */}
            <div className="flex-1 overflow-y-auto">
                {isLoading ? (
                    <div className="flex items-center justify-center h-32">
                        <RefreshCw className="w-6 h-6 animate-spin text-blue-600" />
                    </div>
                ) : filteredSkills.length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-32 text-slate-400">
                        <Zap className="w-12 h-12 mb-2 opacity-20" />
                        <p className="text-sm font-medium">未找到匹配的 Skills</p>
                    </div>
                ) : (
                    <div className="space-y-2">
                        {filteredSkills.map(skill => (
                            <div
                                key={skill.id}
                                className={`group bg-white border rounded-lg p-3 hover:shadow-md transition-all cursor-pointer ${
                                    skill.enabled
                                        ? 'border-blue-200 hover:border-blue-300'
                                        : 'border-slate-200 hover:border-slate-300 opacity-60'
                                }`}
                                onClick={() => setSelectedSkill(skill)}
                            >
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3 flex-1 min-w-0">
                                        <div className={`p-2 rounded-lg flex-shrink-0 ${
                                            skill.enabled ? 'bg-blue-100 text-blue-600' : 'bg-slate-100 text-slate-400'
                                        }`}>
                                            {getIconComponent(skill.icon)}
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-2">
                                                <h4 className="font-semibold text-slate-900 text-sm truncate">
                                                    {skill.name}
                                                </h4>
                                                <span className="text-xs text-slate-400">v{skill.version}</span>
                                            </div>
                                            <p className="text-xs text-slate-500 truncate mt-0.5">
                                                {skill.description}
                                            </p>
                                        </div>
                                    </div>
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            handleToggleSkill(skill.id, skill.enabled);
                                        }}
                                        className={`p-2 rounded-lg transition-all flex-shrink-0 ${
                                            skill.enabled
                                                ? 'bg-green-100 text-green-600 hover:bg-green-200'
                                                : 'bg-slate-100 text-slate-400 hover:bg-slate-200'
                                        }`}
                                        title={skill.enabled ? '禁用' : '启用'}
                                    >
                                        {skill.enabled ? <Power className="w-4 h-4" /> : <PowerOff className="w-4 h-4" />}
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* Skill Detail Modal */}
            {selectedSkill && (
                <SkillDetailModalInSettings
                    skill={selectedSkill}
                    onClose={() => setSelectedSkill(null)}
                    onToggle={(enabled) => handleToggleSkill(selectedSkill.id, enabled)}
                />
            )}
        </div>
    );
};

// Skill Detail Modal for Settings
interface SkillDetailModalInSettingsProps {
    skill: SkillInfo;
    onClose: () => void;
    onToggle: (currentlyEnabled: boolean) => void;
}

const SkillDetailModalInSettings: React.FC<SkillDetailModalInSettingsProps> = ({ skill, onClose, onToggle }) => {
    return (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-[60] flex items-center justify-center p-4">
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-2xl max-h-[80vh] overflow-hidden flex flex-col">
                {/* Header */}
                <div className="p-4 border-b border-slate-200 bg-gradient-to-r from-blue-50 to-indigo-50">
                    <div className="flex items-start justify-between">
                        <div className="flex items-center gap-3">
                            <div className={`p-2 rounded-lg ${
                                skill.enabled ? 'bg-blue-600 text-white' : 'bg-slate-300 text-slate-600'
                            }`}>
                                <Zap className="w-5 h-5" />
                            </div>
                            <div>
                                <h2 className="text-lg font-bold text-slate-900">{skill.name}</h2>
                                <p className="text-xs text-slate-600">
                                    v{skill.version} · by {skill.author}
                                </p>
                            </div>
                        </div>
                        <button
                            onClick={onClose}
                            className="p-1.5 hover:bg-white/80 rounded-lg transition-colors text-slate-400 hover:text-slate-600"
                        >
                            <X className="w-4 h-4" />
                        </button>
                    </div>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-4 space-y-4">
                    {/* Description */}
                    <div>
                        <h3 className="text-xs font-bold text-slate-900 mb-1 uppercase tracking-wider">描述</h3>
                        <p className="text-sm text-slate-700">{skill.description}</p>
                    </div>

                    {/* Required Columns */}
                    <div>
                        <h3 className="text-xs font-bold text-slate-900 mb-2 uppercase tracking-wider">数据要求</h3>
                        <div className="bg-slate-50 rounded-lg p-3 border border-slate-200">
                            <div className="flex flex-wrap gap-1.5">
                                {skill.required_columns.map(col => (
                                    <span
                                        key={col}
                                        className="px-2 py-1 bg-white border border-slate-300 rounded text-xs font-mono text-slate-700"
                                    >
                                        {col}
                                    </span>
                                ))}
                            </div>
                        </div>
                    </div>

                    {/* Keywords */}
                    <div>
                        <h3 className="text-xs font-bold text-slate-900 mb-2 uppercase tracking-wider">触发关键词</h3>
                        <div className="flex flex-wrap gap-1.5">
                            {skill.keywords.map(keyword => (
                                <span
                                    key={keyword}
                                    className="px-2 py-1 bg-blue-50 text-blue-700 rounded text-xs font-medium border border-blue-200"
                                >
                                    "{keyword}"
                                </span>
                            ))}
                        </div>
                    </div>

                    {/* Tags */}
                    <div>
                        <h3 className="text-xs font-bold text-slate-900 mb-2 uppercase tracking-wider">标签</h3>
                        <div className="flex flex-wrap gap-1.5">
                            {skill.tags.map(tag => (
                                <span
                                    key={tag}
                                    className="px-2 py-1 bg-slate-100 text-slate-600 rounded text-xs"
                                >
                                    #{tag}
                                </span>
                            ))}
                        </div>
                    </div>

                    {/* Tools */}
                    <div>
                        <h3 className="text-xs font-bold text-slate-900 mb-2 uppercase tracking-wider">使用工具</h3>
                        <div className="flex gap-2">
                            {skill.tools.map(tool => (
                                <span
                                    key={tool}
                                    className="px-3 py-1.5 bg-gradient-to-r from-blue-50 to-cyan-50 text-blue-700 rounded text-xs font-bold border border-blue-200"
                                >
                                    {tool.toUpperCase()}
                                </span>
                            ))}
                        </div>
                    </div>
                </div>

                {/* Footer */}
                <div className="p-4 border-t border-slate-200 bg-slate-50 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <span className="text-sm text-slate-600">状态:</span>
                        <span className={`px-2 py-0.5 rounded-full text-xs font-bold ${
                            skill.enabled
                                ? 'bg-green-100 text-green-700'
                                : 'bg-slate-200 text-slate-600'
                        }`}>
                            {skill.enabled ? '✓ 已启用' : '✗ 已禁用'}
                        </span>
                    </div>
                    <button
                        onClick={() => onToggle(skill.enabled)}
                        className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                            skill.enabled
                                ? 'bg-slate-200 text-slate-700 hover:bg-slate-300'
                                : 'bg-green-600 text-white hover:bg-green-700'
                        }`}
                    >
                        {skill.enabled ? '禁用' : '启用'}
                    </button>
                </div>
            </div>
        </div>
    );
};

// Intent Enhancement Settings Component
interface IntentEnhancementSettingsProps {
    config: configModel.Config;
    updateConfig: (updates: Partial<configModel.Config>) => void;
}

const IntentEnhancementSettings: React.FC<IntentEnhancementSettingsProps> = ({ config, updateConfig }) => {
    const { t } = useLanguage();

    // Get or create default intent enhancement config
    const getIntentConfig = () => {
        return config.intentEnhancement || configModel.IntentEnhancementConfig.createFrom({
            enable_context_enhancement: true,
            enable_preference_learning: true,
            enable_dynamic_dimensions: true,
            enable_few_shot_examples: true,
            enable_caching: true,
            cache_similarity_threshold: 0.85,
            cache_expiration_hours: 24,
            max_cache_entries: 1000,
            max_history_records: 10,
        });
    };

    const intentConfig = getIntentConfig();

    // Update intent enhancement config
    const updateIntentConfig = (updates: Partial<configModel.IntentEnhancementConfig>) => {
        const newIntentConfig = configModel.IntentEnhancementConfig.createFrom({
            ...intentConfig,
            ...updates,
        });
        updateConfig({ intentEnhancement: newIntentConfig });
    };

    // Reset to defaults
    const resetToDefaults = () => {
        const defaultConfig = configModel.IntentEnhancementConfig.createFrom({
            enable_context_enhancement: true,
            enable_preference_learning: true,
            enable_dynamic_dimensions: true,
            enable_few_shot_examples: true,
            enable_caching: true,
            cache_similarity_threshold: 0.85,
            cache_expiration_hours: 24,
            max_cache_entries: 1000,
            max_history_records: 10,
        });
        updateConfig({ intentEnhancement: defaultConfig });
    };

    // Check if all features are enabled
    const allFeaturesEnabled = intentConfig.enable_context_enhancement &&
        intentConfig.enable_preference_learning &&
        intentConfig.enable_dynamic_dimensions &&
        intentConfig.enable_few_shot_examples &&
        intentConfig.enable_caching;

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-slate-200 pb-4">
                <div>
                    <h3 className="text-lg font-semibold text-slate-800">
                        {t('intent_enhancement_settings') || '意图增强设置'}
                    </h3>
                    <p className="text-sm text-slate-500 mt-1">
                        {t('intent_enhancement_description') || '配置AI驱动的意图理解增强功能'}
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    {allFeaturesEnabled ? (
                        <span className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-green-700 bg-green-100 rounded-full">
                            <CheckCircle className="w-3 h-3" />
                            {t('all_features_enabled') || '所有功能已启用'}
                        </span>
                    ) : (
                        <span className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-amber-700 bg-amber-100 rounded-full">
                            <AlertCircle className="w-3 h-3" />
                            {t('some_features_disabled') || '部分功能已禁用'}
                        </span>
                    )}
                </div>
            </div>

            {/* Feature Toggles */}
            <div className="space-y-4">
                <h4 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">
                    {t('intent_enhancement') || '增强功能'}
                </h4>

                {/* Context Enhancement */}
                <div className="flex items-center justify-between py-3 border-b border-slate-100">
                    <div className="flex-1">
                        <label className="text-sm font-medium text-slate-700">
                            {t('enable_context_enhancement') || '上下文增强'}
                        </label>
                        <p className="text-xs text-slate-500 mt-1">
                            {t('enable_context_enhancement_desc') || '将历史分析记录作为上下文，提供更相关的建议'}
                        </p>
                    </div>
                    <input
                        type="checkbox"
                        checked={intentConfig.enable_context_enhancement}
                        onChange={(e) => updateIntentConfig({ enable_context_enhancement: e.target.checked })}
                        className="w-4 h-4 text-blue-600 focus:ring-2 focus:ring-blue-500 rounded"
                    />
                </div>

                {/* Preference Learning */}
                <div className="flex items-center justify-between py-3 border-b border-slate-100">
                    <div className="flex-1">
                        <label className="text-sm font-medium text-slate-700">
                            {t('enable_preference_learning') || '偏好学习'}
                        </label>
                        <p className="text-xs text-slate-500 mt-1">
                            {t('enable_preference_learning_desc') || '从您的意图选择中学习，优化建议排序'}
                        </p>
                    </div>
                    <input
                        type="checkbox"
                        checked={intentConfig.enable_preference_learning}
                        onChange={(e) => updateIntentConfig({ enable_preference_learning: e.target.checked })}
                        className="w-4 h-4 text-blue-600 focus:ring-2 focus:ring-blue-500 rounded"
                    />
                </div>

                {/* Dynamic Dimensions */}
                <div className="flex items-center justify-between py-3 border-b border-slate-100">
                    <div className="flex-1">
                        <label className="text-sm font-medium text-slate-700">
                            {t('enable_dynamic_dimensions') || '动态维度'}
                        </label>
                        <p className="text-xs text-slate-500 mt-1">
                            {t('enable_dynamic_dimensions_desc') || '根据数据特征自动调整分析维度建议'}
                        </p>
                    </div>
                    <input
                        type="checkbox"
                        checked={intentConfig.enable_dynamic_dimensions}
                        onChange={(e) => updateIntentConfig({ enable_dynamic_dimensions: e.target.checked })}
                        className="w-4 h-4 text-blue-600 focus:ring-2 focus:ring-blue-500 rounded"
                    />
                </div>

                {/* Few-shot Examples */}
                <div className="flex items-center justify-between py-3 border-b border-slate-100">
                    <div className="flex-1">
                        <label className="text-sm font-medium text-slate-700">
                            {t('enable_few_shot_examples') || 'Few-shot示例'}
                        </label>
                        <p className="text-xs text-slate-500 mt-1">
                            {t('enable_few_shot_examples_desc') || '包含领域特定示例以提高建议质量'}
                        </p>
                    </div>
                    <input
                        type="checkbox"
                        checked={intentConfig.enable_few_shot_examples}
                        onChange={(e) => updateIntentConfig({ enable_few_shot_examples: e.target.checked })}
                        className="w-4 h-4 text-blue-600 focus:ring-2 focus:ring-blue-500 rounded"
                    />
                </div>

                {/* Caching */}
                <div className="flex items-center justify-between py-3 border-b border-slate-100">
                    <div className="flex-1">
                        <label className="text-sm font-medium text-slate-700">
                            {t('enable_caching') || '意图缓存'}
                        </label>
                        <p className="text-xs text-slate-500 mt-1">
                            {t('enable_caching_desc') || '缓存相似请求以减少LLM调用并提高响应速度'}
                        </p>
                    </div>
                    <input
                        type="checkbox"
                        checked={intentConfig.enable_caching}
                        onChange={(e) => updateIntentConfig({ enable_caching: e.target.checked })}
                        className="w-4 h-4 text-blue-600 focus:ring-2 focus:ring-blue-500 rounded"
                    />
                </div>
            </div>

            {/* Cache Settings (only shown when caching is enabled) */}
            {intentConfig.enable_caching && (
                <div className="space-y-4 pt-4 border-t border-slate-200">
                    <h4 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">
                        {t('cache_settings') || '缓存设置'}
                    </h4>

                    {/* Similarity Threshold */}
                    <div className="space-y-2">
                        <label className="block text-sm font-medium text-slate-700">
                            {t('cache_similarity_threshold') || '相似度阈值'}
                        </label>
                        <div className="flex items-center gap-4">
                            <input
                                type="range"
                                min="0.5"
                                max="1.0"
                                step="0.05"
                                value={intentConfig.cache_similarity_threshold}
                                onChange={(e) => updateIntentConfig({ cache_similarity_threshold: parseFloat(e.target.value) })}
                                className="flex-1 h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer"
                            />
                            <span className="text-sm font-mono text-slate-600 w-12 text-right">
                                {intentConfig.cache_similarity_threshold.toFixed(2)}
                            </span>
                        </div>
                        <p className="text-xs text-slate-500">
                            {t('cache_similarity_threshold_desc') || '缓存命中所需的最小相似度分数（0-1）'}
                        </p>
                    </div>

                    {/* Cache Expiration */}
                    <div className="space-y-2">
                        <label className="block text-sm font-medium text-slate-700">
                            {t('cache_expiration_hours') || '缓存过期时间（小时）'}
                        </label>
                        <input
                            type="number"
                            min="1"
                            max="168"
                            value={intentConfig.cache_expiration_hours}
                            onChange={(e) => updateIntentConfig({ cache_expiration_hours: parseInt(e.target.value) || 24 })}
                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                        />
                        <p className="text-xs text-slate-500">
                            {t('cache_expiration_hours_desc') || '缓存建议的有效时长'}
                        </p>
                    </div>

                    {/* Max Cache Entries */}
                    <div className="space-y-2">
                        <label className="block text-sm font-medium text-slate-700">
                            {t('max_cache_entries') || '最大缓存条目数'}
                        </label>
                        <input
                            type="number"
                            min="100"
                            max="10000"
                            value={intentConfig.max_cache_entries}
                            onChange={(e) => updateIntentConfig({ max_cache_entries: parseInt(e.target.value) || 1000 })}
                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                        />
                        <p className="text-xs text-slate-500">
                            {t('max_cache_entries_desc') || '缓存的最大意图建议数量'}
                        </p>
                    </div>
                </div>
            )}

            {/* Context Enhancement Settings (only shown when context enhancement is enabled) */}
            {intentConfig.enable_context_enhancement && (
                <div className="space-y-4 pt-4 border-t border-slate-200">
                    <h4 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">
                        {t('enable_context_enhancement') || '上下文设置'}
                    </h4>

                    {/* Max History Records */}
                    <div className="space-y-2">
                        <label className="block text-sm font-medium text-slate-700">
                            {t('max_history_records') || '最大历史记录数'}
                        </label>
                        <input
                            type="number"
                            min="1"
                            max="50"
                            value={intentConfig.max_history_records}
                            onChange={(e) => updateIntentConfig({ max_history_records: parseInt(e.target.value) || 10 })}
                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                        />
                        <p className="text-xs text-slate-500">
                            {t('max_history_records_desc') || '上下文中包含的历史分析记录数量'}
                        </p>
                    </div>
                </div>
            )}

            {/* Reset Button */}
            <div className="pt-4 border-t border-slate-200">
                <button
                    onClick={resetToDefaults}
                    className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-md transition-colors"
                >
                    {t('reset_to_defaults') || '恢复默认设置'}
                </button>
            </div>

            {/* Info Box */}
            <div className="bg-blue-50 p-4 rounded-lg">
                <div className="flex items-start gap-3">
                    <Zap className="text-blue-600 flex-shrink-0 mt-0.5" size={20} />
                    <div className="text-sm text-blue-800">
                        <p className="font-medium mb-1">
                            {config.language === '简体中文' ? '关于意图增强' : 'About Intent Enhancement'}
                        </p>
                        <ul className="list-disc list-inside space-y-1 text-xs">
                            <li>{config.language === '简体中文' ? '上下文增强：利用历史分析记录提供更相关的建议' : 'Context Enhancement: Uses historical analysis records for more relevant suggestions'}</li>
                            <li>{config.language === '简体中文' ? '偏好学习：根据您的选择习惯优化建议排序' : 'Preference Learning: Optimizes suggestion ranking based on your selection habits'}</li>
                            <li>{config.language === '简体中文' ? '动态维度：根据数据特征自动推荐分析维度' : 'Dynamic Dimensions: Automatically recommends analysis dimensions based on data characteristics'}</li>
                            <li>{config.language === '简体中文' ? 'Few-shot示例：使用领域示例提高建议质量' : 'Few-shot Examples: Uses domain examples to improve suggestion quality'}</li>
                            <li>{config.language === '简体中文' ? '意图缓存：缓存相似请求以加快响应速度' : 'Intent Caching: Caches similar requests for faster response times'}</li>
                        </ul>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default PreferenceModal;