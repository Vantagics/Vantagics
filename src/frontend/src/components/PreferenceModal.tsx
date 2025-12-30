import React, { useState, useEffect } from 'react';
import { GetConfig, SaveConfig } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

type Tab = 'llm' | 'system' | 'drivers';

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
        language: 'English'
    });
    const [isTesting, setIsTesting] = useState(false);
    const [testResult, setTestResult] = useState<{success: boolean, message: string} | null>(null);

    useEffect(() => {
        if (isOpen) {
            GetConfig().then(setConfig).catch(console.error);
            setTestResult(null);
        }
    }, [isOpen]);

    const handleSave = async () => {
        try {
            await SaveConfig(config);
            onClose();
        } catch (err) {
            console.error('Failed to save config:', err);
            alert('Failed to save configuration');
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
            setTestResult({success: false, message: String(err)});
        } finally {
            setIsTesting(false);
        }
    };

    if (!isOpen) return null;

    const isAnthropic = config.llmProvider === 'Anthropic';
    const isOpenAICompatible = config.llmProvider === 'OpenAI-Compatible';

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[800px] h-[600px] rounded-xl shadow-2xl flex overflow-hidden text-slate-900">
                {/* Sidebar */}
                <div className="w-64 bg-slate-50 border-r border-slate-200 p-4 flex flex-col">
                    <h2 className="text-xl font-bold text-slate-800 mb-6 px-2">Preferences</h2>
                    <nav className="space-y-1">
                        {(['llm', 'system', 'drivers'] as const).map((tab) => (
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
                                        </select>
                                    </div>
                                    
                                    {(isOpenAICompatible || config.llmProvider === 'OpenAI') && (
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
                                                placeholder={isOpenAICompatible ? "http://localhost:11434" : "https://api.openai.com/v1"}
                                            />
                                            <p className="mt-1 text-[10px] text-slate-400 italic">
                                                {isOpenAICompatible 
                                                    ? "Base URL for the compatible API (e.g., Ollama, LM Studio, DeepSeek)" 
                                                    : "Leave empty for official OpenAI API"}
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
                                </div>
                            </div>
                        )}
                        {activeTab === 'drivers' && <DriverSettings />}
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

export default PreferenceModal;
