import React, { useState, useEffect } from 'react';
import { GetConfig, SaveConfig, SelectDirectory, GetPythonEnvironments, ValidatePython } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';

type Tab = 'llm' | 'system' | 'drivers' | 'runenv';

interface PreferenceModalProps {
    isOpen: boolean;
    onClose: () => void;
}

const PreferenceModal: React.FC<PreferenceModalProps> = ({ isOpen, onClose }) => {
    const [activeTab, setActiveTab] = useState<Tab>('llm');
    const [config, setConfig] = useState<main.Config>({
        llmProvider: 'OpenAI',
        apiKey: '',
        baseUrl: '',
        modelName: '',
        maxTokens: 4096,
        darkMode: false,
        localCache: true,
        language: 'English',
        claudeHeaderStyle: 'Anthropic',
        dataCacheDir: '',
        pythonPath: ''
    });
    const [isTesting, setIsTesting] = useState(false);
    const [testResult, setTestResult] = useState<{success: boolean, message: string} | null>(null);

    useEffect(() => {
        if (isOpen) {
            GetConfig().then(data => {
                setConfig(data);
            }).catch(console.error);
            setTestResult(null);
        }

        // Listen for directory selection result
        const unsubscribe = EventsOn("directory-selected", (path: string) => {
            console.log('Event directory-selected received:', path);
            if (path) {
                setConfig(prev => ({ ...prev, dataCacheDir: path }));
            }
        });

        return () => {
            unsubscribe();
        };
    }, [isOpen]);

    const handleSave = async () => {
        try {
            await SaveConfig(config);
            onClose();
        } catch (err) {
            console.error('Failed to save config:', err);
            alert('Failed to save configuration: ' + err);
        }
    };

    const handleBrowseDirectory = () => {
        console.log('handleBrowseDirectory triggered (event-based)');
        SelectDirectory();
    };

    const handleTestConnection = async () => {
        setIsTesting(true);
        setTestResult(null);
        try {
            // @ts-ignore - We will implement this in App.go
            const result = await window.go.main.App.TestLLMConnection(config);
            setTestResult(result);
        } catch (err) {
            setTestResult({success: false, message: String(err)});
        } finally {
            setIsTesting(false);
        }
    };

    if (!isOpen) return null;

    const isAnthropic = config.llmProvider === 'Anthropic';
    const isOpenAICompatible = config.llmProvider === 'OpenAI-Compatible';
    const isClaudeCompatible = config.llmProvider === 'Claude-Compatible';

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[800px] h-[600px] rounded-xl shadow-2xl flex overflow-hidden text-slate-900">
                {/* Sidebar */}
                <div className="w-64 bg-slate-50 border-r border-slate-200 p-4 flex flex-col">
                    <h2 className="text-xl font-bold text-slate-800 mb-6 px-2">Preferences</h2>
                    <nav className="space-y-1">
                        {(['llm', 'system', 'drivers', 'runenv'] as const).map((tab) => (
                            <button
                                key={tab}
                                onClick={() => setActiveTab(tab)}
                                className={`w-full text-left px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                                    activeTab === tab ? 'bg-blue-100 text-blue-700' : 'text-slate-600 hover:bg-slate-100'
                                }`}
                            >
                                {tab === 'llm' && 'LLM Configuration'}
                                {tab === 'system' && 'System Parameters'}
                                {tab === 'drivers' && 'Data Source Drivers'}
                                {tab === 'runenv' && 'Run Environment'}
                            </button>
                        ))}
                    </nav>
                </div>

                {/* Content Area */}
                <div className="flex-1 flex flex-col min-w-0">
                    <div className="flex-1 p-8 overflow-y-auto">
                        {activeTab === 'llm' && (
                            <div className="space-y-6">
                                <h3 className="text-lg font-semibold text-slate-800 border-b border-slate-200 pb-2">LLM Model Configuration</h3>
                                <div className="grid gap-4">
                                    <div>
                                        <label htmlFor="llmProvider" className="block text-sm font-medium text-slate-700 mb-1">Provider Type</label>
                                        <select 
                                            id="llmProvider"
                                            value={config.llmProvider}
                                            onChange={(e) => setConfig({...config, llmProvider: e.target.value})}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        >
                                            <option value="OpenAI">OpenAI</option>
                                            <option value="Anthropic">Anthropic (Claude)</option>
                                            <option value="OpenAI-Compatible">OpenAI-Compatible (Local, DeepSeek, etc.)</option>
                                            <option value="Claude-Compatible">Claude-Compatible (Proxies, Bedrock, etc.)</option>
                                        </select>
                                    </div>
                                    
                                    {(isOpenAICompatible || isClaudeCompatible || config.llmProvider === 'OpenAI') && (
                                        <div className="animate-in fade-in slide-in-from-top-1 duration-200">
                                            <label htmlFor="baseUrl" className="block text-sm font-medium text-slate-700 mb-1">
                                                API Base URL {config.llmProvider === 'OpenAI' ? '(Optional)' : ''}
                                            </label>
                                            <input 
                                                id="baseUrl"
                                                type="text" 
                                                value={config.baseUrl}
                                                onChange={(e) => setConfig({...config, baseUrl: e.target.value})}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                placeholder={
                                                    isOpenAICompatible ? "http://localhost:11434" : 
                                                    isClaudeCompatible ? "https://bedrock-runtime.us-east-1.amazonaws.com" :
                                                    "https://api.openai.com/v1"
                                                }
                                                autoCapitalize="none"
                                                autoCorrect="off"
                                                spellCheck={false}
                                            />
                                            <p className="mt-1 text-[10px] text-slate-400 italic">
                                                {isOpenAICompatible 
                                                    ? "Base URL for the compatible API (e.g., Ollama, LM Studio, DeepSeek)" 
                                                    : isClaudeCompatible 
                                                        ? "Base URL for Claude proxy (e.g., AWS Bedrock, Vertex AI, One API)"
                                                        : "Leave empty for official OpenAI API"}
                                            </p>
                                        </div>
                                    )}

                                    {isClaudeCompatible && (
                                        <div className="animate-in fade-in slide-in-from-top-1 duration-200">
                                            <label htmlFor="headerStyle" className="block text-sm font-medium text-slate-700 mb-1">
                                                Header Style
                                            </label>
                                            <select 
                                                id="headerStyle"
                                                value={config.claudeHeaderStyle || 'Anthropic'}
                                                onChange={(e) => setConfig({...config, claudeHeaderStyle: e.target.value})}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            >
                                                <option value="Anthropic">Anthropic (x-api-key)</option>
                                                <option value="OpenAI">OpenAI (Authorization: Bearer)</option>
                                            </select>
                                            <p className="mt-1 text-[10px] text-slate-400 italic">
                                                Select "OpenAI" if your proxy uses Bearer tokens (e.g., some One API setups).
                                            </p>
                                        </div>
                                    )}

                                    <div>
                                        <label htmlFor="apiKey" className="block text-sm font-medium text-slate-700 mb-1">
                                            API Key {isOpenAICompatible ? '(Optional)' : ''}
                                        </label>
                                        <input 
                                            id="apiKey"
                                            type="password" 
                                            value={config.apiKey}
                                            onChange={(e) => setConfig({...config, apiKey: e.target.value})}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            placeholder={isAnthropic ? "sk-ant-..." : "sk-..."}
                                            autoCapitalize="none"
                                            autoCorrect="off"
                                            spellCheck={false}
                                        />
                                    </div>
                                    <div>
                                        <label htmlFor="modelName" className="block text-sm font-medium text-slate-700 mb-1">Model Name</label>
                                        <input 
                                            id="modelName"
                                            type="text" 
                                            value={config.modelName}
                                            onChange={(e) => setConfig({...config, modelName: e.target.value})}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            placeholder={isAnthropic ? "claude-3-5-sonnet-20240620" : (isOpenAICompatible ? "llama3" : "gpt-4o")}
                                            autoCapitalize="none"
                                            autoCorrect="off"
                                            spellCheck={false}
                                        />
                                    </div>

                                    <div>
                                        <label htmlFor="maxTokens" className="block text-sm font-medium text-slate-700 mb-1">Max Tokens</label>
                                        <input 
                                            id="maxTokens"
                                            type="number" 
                                            value={config.maxTokens}
                                            onChange={(e) => setConfig({...config, maxTokens: parseInt(e.target.value) || 0})}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        />
                                    </div>
                                    
                                    <div className="pt-2 flex items-center gap-4">
                                        <button 
                                            onClick={handleTestConnection}
                                            disabled={isTesting}
                                            className={`px-4 py-2 text-xs font-semibold rounded-md transition-colors ${
                                                isTesting ? 'bg-slate-100 text-slate-400 cursor-not-allowed' : 'bg-slate-100 text-slate-700 hover:bg-slate-200'
                                            }`}
                                        >
                                            {isTesting ? 'Testing...' : 'Test Connection'}
                                        </button>
                                        
                                        {testResult && (
                                            <div className={`text-xs font-medium animate-in fade-in slide-in-from-left-1 ${
                                                testResult.success ? 'text-green-600' : 'text-red-600'
                                            }`}>
                                                {testResult.success ? '✓ Connection successful!' : `✗ ${testResult.message}`}
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </div>
                        )}
                        {activeTab === 'system' && (
                            <div className="space-y-6">
                                <h3 className="text-lg font-semibold text-slate-800 border-b border-slate-200 pb-2">System Parameters</h3>
                                <div className="space-y-4">
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <span className="block text-sm font-medium text-slate-700">Dark Mode</span>
                                            <span className="block text-xs text-slate-500">Enable dark appearance for the UI</span>
                                        </div>
                                        <input 
                                            type="checkbox" 
                                            checked={config.darkMode}
                                            onChange={(e) => setConfig({...config, darkMode: e.target.checked})}
                                        />
                                    </div>
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <span className="block text-sm font-medium text-slate-700">Local Cache</span>
                                            <span className="block text-xs text-slate-500">Store query results locally</span>
                                        </div>
                                        <input 
                                            type="checkbox" 
                                            checked={config.localCache}
                                            onChange={(e) => setConfig({...config, localCache: e.target.checked})}
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-1">Language</label>
                                        <select 
                                            value={config.language}
                                            onChange={(e) => setConfig({...config, language: e.target.value})}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm"
                                        >
                                            <option>English</option>
                                            <option>简体中文</option>
                                        </select>
                                    </div>
                                    <div>
                                        <label htmlFor="dataCacheDir" className="block text-sm font-medium text-slate-700 mb-1">Data Cache Directory</label>
                                        <input 
                                            id="dataCacheDir"
                                            type="text" 
                                            value={config.dataCacheDir}
                                            onChange={(e) => setConfig({...config, dataCacheDir: e.target.value})}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            placeholder="~/RapidBI"
                                            autoCapitalize="none"
                                            autoCorrect="off"
                                            spellCheck={false}
                                        />
                                        <p className="mt-1 text-[10px] text-slate-400 italic">
                                            The directory used to store application data. Must exist on your system.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        )}
                        {activeTab === 'drivers' && <DriverSettings />}
                        {activeTab === 'runenv' && <RunEnvSettings config={config} setConfig={setConfig} />}
                    </div>
                    
                    {/* Footer */}
                    <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-3">
                        <button onClick={onClose} className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-200 rounded-md">
                            Cancel
                        </button>
                        <button onClick={handleSave} className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md shadow-sm">
                            Save Changes
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

const DriverSettings: React.FC = () => {
    const drivers = [
        { name: 'PostgreSQL', version: '14.2', installed: true },
        { name: 'MySQL', version: '8.0', installed: true },
        { name: 'SQLite', version: '3.39', installed: true },
    ];

    return (
        <div className="space-y-6">
            <h3 className="text-lg font-semibold text-slate-800 border-b border-slate-200 pb-2">Data Source Drivers</h3>
            <div className="border border-slate-200 rounded-lg overflow-hidden">
                <table className="w-full text-sm text-left">
                    <thead className="bg-slate-50 text-slate-700 font-medium">
                        <tr>
                            <th className="p-3 border-b border-slate-200">Driver Name</th>
                            <th className="p-3 border-b border-slate-200">Version</th>
                            <th className="p-3 border-b border-slate-200">Status</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                        {drivers.map((driver) => (
                            <tr key={driver.name}>
                                <td className="p-3 font-medium text-slate-700">{driver.name}</td>
                                <td className="p-3 text-slate-500">{driver.version}</td>
                                <td className="p-3 text-green-600 font-medium">Installed</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

interface RunEnvSettingsProps {
    config: main.Config;
    setConfig: (config: main.Config) => void;
}

const RunEnvSettings: React.FC<RunEnvSettingsProps> = ({ config, setConfig }) => {
    const [envs, setEnvs] = useState<main.PythonEnvironment[]>([]);
    const [loading, setLoading] = useState(false);
    const [validation, setValidation] = useState<main.PythonValidationResult | null>(null);
    const [validating, setValidating] = useState(false);

    useEffect(() => {
        setLoading(true);
        GetPythonEnvironments()
            .then(envs => {
                setEnvs(envs);
            })
            .catch(console.error)
            .finally(() => setLoading(false));
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

    // Check if current pythonPath is in the list
    const isKnownEnv = envs.some(e => e.path === config.pythonPath);

    return (
        <div className="space-y-6">
            <h3 className="text-lg font-semibold text-slate-800 border-b border-slate-200 pb-2">Python Runtime Environment</h3>
            <div className="space-y-4">
                <div>
                    <label htmlFor="pythonPath" className="block text-sm font-medium text-slate-700 mb-1">Select Python Environment</label>
                    {loading ? (
                        <div className="text-sm text-slate-500 animate-pulse">Scanning for Python environments...</div>
                    ) : (
                        <select
                            id="pythonPath"
                            value={config.pythonPath}
                            onChange={(e) => setConfig({ ...config, pythonPath: e.target.value })}
                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                        >
                            <option value="">Select an environment...</option>
                            {config.pythonPath && !isKnownEnv && (
                                <option value={config.pythonPath}>
                                    {config.pythonPath} (Saved)
                                </option>
                            )}
                            {envs.map((env) => (
                                <option key={env.path} value={env.path}>
                                    {env.type} - {env.version} ({env.path})
                                </option>
                            ))}
                        </select>
                    )}
                    <p className="mt-1 text-[10px] text-slate-400 italic">
                        Select the Python interpreter to use for executing generated scripts.
                    </p>
                </div>

                {validating && (
                    <div className="text-sm text-blue-600 animate-pulse">Validating environment...</div>
                )}

                {validation && !validating && (
                    <div className={`p-4 rounded-lg border ${validation.valid ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'}`}>
                        <div className="flex items-center justify-between mb-2">
                            <span className={`font-semibold ${validation.valid ? 'text-green-800' : 'text-red-800'}`}>
                                {validation.valid ? '✓ Environment Valid' : '✗ Environment Invalid'}
                            </span>
                            <span className="text-xs text-slate-500">{validation.version}</span>
                        </div>
                        
                        {!validation.valid && validation.error && (
                            <div className="text-sm text-red-700 mb-2">{validation.error}</div>
                        )}

                        {validation.missingPackages && validation.missingPackages.length > 0 && (
                            <div>
                                <span className="text-sm font-medium text-amber-700 block mb-1">Missing Recommended Packages:</span>
                                <ul className="list-disc list-inside text-xs text-amber-600">
                                    {validation.missingPackages.map(pkg => (
                                        <li key={pkg}>{pkg}</li>
                                    ))}
                                </ul>
                            </div>
                        )}
                        
                        {validation.valid && (!validation.missingPackages || validation.missingPackages.length === 0) && (
                            <div className="text-xs text-green-700">All recommended packages (pandas, matplotlib) are installed.</div>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
};

export default PreferenceModal;