import React, { useState, useEffect } from 'react';
import { GetConfig, SaveConfig, SelectDirectory, GetPythonEnvironments, ValidatePython, InstallPythonPackages, CreateRapidBIEnvironment, CheckRapidBIEnvironmentExists, DiagnosePythonInstallation } from '../../wailsjs/go/main/App';
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
import { main, agent, config as configModel } from '../../wailsjs/go/models';
import { useLanguage } from '../i18n';

type Tab = 'llm' | 'system' | 'runenv';

interface PreferenceModalProps {
    isOpen: boolean;
    onClose: () => void;
}

const PreferenceModal: React.FC<PreferenceModalProps> = ({ isOpen, onClose }) => {
    const { t } = useLanguage();
    const [activeTab, setActiveTab] = useState<Tab>('llm');
    const [config, setConfig] = useState<configModel.Config>({
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
        pythonPath: '',
        maxPreviewRows: 100,
        detailedLog: false
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
    }, [isOpen]);

    const handleSave = async () => {
        try {
            await SaveConfig(config);
            // Note: config-updated event is now sent by the backend
            onClose();
        } catch (err) {
            console.error('Failed to save config:', err);
            alert('Failed to save configuration: ' + err);
        }
    };

    const handleBrowseDirectory = async () => {
        try {
            const path = await SelectDirectory();
            if (path) {
                setConfig(prev => ({ ...prev, dataCacheDir: path }));
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
                    <h2 className="text-xl font-bold text-slate-800 mb-6 px-2">{t('preferences')}</h2>
                    <nav className="space-y-1">
                        {(['llm', 'system', 'runenv'] as const).map((tab) => (
                            <button
                                key={tab}
                                onClick={() => setActiveTab(tab)}
                                className={`w-full text-left px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                                    activeTab === tab ? 'bg-blue-100 text-blue-700' : 'text-slate-600 hover:bg-slate-100'
                                }`}
                            >
                                {tab === 'llm' && t('llm_config')}
                                {tab === 'system' && t('system_params')}
                                {tab === 'runenv' && t('run_env')}
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
                                            onChange={(e) => setConfig({...config, llmProvider: e.target.value})}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        >
                                            <option value="OpenAI">OpenAI</option>
                                            <option value="Anthropic">Anthropic (Claude)</option>
                                            <option value="OpenAI-Compatible">OpenAI-Compatible (Local, DeepSeek, etc.)</option>
                                            <option value="Claude-Compatible">Claude-Compatible (Proxies, Bedrock, etc.)</option>
                                        </select>
                                    </div>
                                    
                                    {(isOpenAICompatible || isClaudeCompatible) && (
                                        <div className="animate-in fade-in slide-in-from-top-1 duration-200">
                                            <label htmlFor="baseUrl" className="block text-sm font-medium text-slate-700 mb-1">
                                                API Base URL
                                            </label>
                                            <input 
                                                id="baseUrl"
                                                type="text" 
                                                value={config.baseUrl}
                                                onChange={(e) => setConfig({...config, baseUrl: e.target.value})}
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
                                                    ? "Base URL for the compatible API (e.g., Ollama, LM Studio, DeepSeek)" 
                                                    : "Base URL for Claude proxy (e.g., AWS Bedrock, Vertex AI, One API)"}
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
                                            {t('api_key')} {isOpenAICompatible ? '(Optional)' : ''}
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
                                        <label htmlFor="modelName" className="block text-sm font-medium text-slate-700 mb-1">{t('model_name')}</label>
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
                                        <label htmlFor="maxTokens" className="block text-sm font-medium text-slate-700 mb-1">{t('max_tokens')}</label>
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
                                <h3 className="text-lg font-semibold text-slate-800 border-b border-slate-200 pb-2">{t('system_params')}</h3>
                                <div className="space-y-4">
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <span className="block text-sm font-medium text-slate-700">{t('dark_mode')}</span>
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
                                            <span className="block text-sm font-medium text-slate-700">{t('local_cache')}</span>
                                            <span className="block text-xs text-slate-500">Store query results locally</span>
                                        </div>
                                        <input 
                                            type="checkbox" 
                                            checked={config.localCache}
                                            onChange={(e) => setConfig({...config, localCache: e.target.checked})}
                                        />
                                    </div>
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <span className="block text-sm font-medium text-slate-700">{t('detailed_log')}</span>
                                            <span className="block text-xs text-slate-500">Enable detailed logging for debugging</span>
                                        </div>
                                        <input 
                                            type="checkbox" 
                                            checked={config.detailedLog}
                                            onChange={(e) => setConfig({...config, detailedLog: e.target.checked})}
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-1">{t('language')}</label>
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
                                        <label htmlFor="maxPreviewRows" className="block text-sm font-medium text-slate-700 mb-1">{t('max_preview_rows')}</label>
                                        <input 
                                            id="maxPreviewRows"
                                            type="number" 
                                            value={config.maxPreviewRows}
                                            onChange={(e) => setConfig({...config, maxPreviewRows: parseInt(e.target.value) || 100})}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            min="1"
                                            max="10000"
                                        />
                                        <p className="mt-1 text-[10px] text-slate-400 italic">
                                            Number of rows to display in the data preview window (default 100).
                                        </p>
                                    </div>
                                    <div>
                                        <label htmlFor="dataCacheDir" className="block text-sm font-medium text-slate-700 mb-1">{t('data_cache_dir')}</label>
                                        <div className="flex gap-2">
                                            <input
                                                id="dataCacheDir"
                                                type="text"
                                                value={config.dataCacheDir}
                                                onChange={(e) => setConfig({...config, dataCacheDir: e.target.value})}
                                                className="flex-1 border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                placeholder="~/RapidBI"
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
                                            The directory used to store application data. Must exist on your system.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        )}
                        {activeTab === 'runenv' && <RunEnvSettings config={config} setConfig={setConfig} />}
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
        </div>
    );
};

interface RunEnvSettingsProps {
    config: configModel.Config;
    setConfig: (config: configModel.Config) => void;
}

const RunEnvSettings: React.FC<RunEnvSettingsProps> = ({ config, setConfig }) => {
    const [envs, setEnvs] = useState<agent.PythonEnvironment[]>([]);
    const [loading, setLoading] = useState(false);
    const [validation, setValidation] = useState<agent.PythonValidationResult | null>(null);
    const [validating, setValidating] = useState(false);
    const [installing, setInstalling] = useState(false);
    const [creatingEnv, setCreatingEnv] = useState(false);
    const [showCreateButton, setShowCreateButton] = useState(false);
    const [diagnosing, setDiagnosing] = useState(false);
    const [notification, setNotification] = useState<{type: 'success' | 'error' | 'info', message: string} | null>(null);

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
            
            // Check if we should show the "Create RapidBI Environment" button
            const hasVirtualEnvSupport = environments.some(env => 
                env.type.toLowerCase().includes('conda') || 
                env.type.toLowerCase().includes('virtualenv') ||
                env.type.toLowerCase().includes('venv')
            );
            
            const hasRapidBIEnv = await CheckRapidBIEnvironmentExists();
            setShowCreateButton(hasVirtualEnvSupport && !hasRapidBIEnv);
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
                setNotification({type: 'success', message: '所有缺失的包已成功安装！'});
            } else {
                setNotification({type: 'info', message: `安装完成。仍有 ${newValidation.missingPackages?.length || 0} 个包未能安装。`});
            }
        } catch (error) {
            console.error('Package installation failed:', error);
            setNotification({type: 'error', message: `包安装失败: ${error}`});
        } finally {
            setInstalling(false);
            setValidating(false);
        }
    };

    const handleCreateRapidBIEnvironment = async () => {
        setCreatingEnv(true);
        try {
            const pythonPath = await CreateRapidBIEnvironment();
            
            // Refresh the environment list
            await loadEnvironments();
            
            // Auto-select the new environment
            setConfig({ ...config, pythonPath });
            
            setNotification({type: 'success', message: 'RapidBI专用环境创建成功！已自动选择该环境。'});
        } catch (error) {
            console.error('Environment creation failed:', error);
            
            // Show detailed error message with suggestions
            const errorMessage = String(error);
            let userMessage = '环境创建失败\n\n';
            
            if (errorMessage.includes('No suitable Python interpreter found')) {
                // Extract the detailed diagnostic information from the error
                const diagnosticStart = errorMessage.indexOf('Detection attempts:');
                const diagnosticEnd = errorMessage.indexOf('To resolve this issue');
                
                if (diagnosticStart !== -1 && diagnosticEnd !== -1) {
                    const diagnosticInfo = errorMessage.substring(diagnosticStart, diagnosticEnd);
                    userMessage += '诊断信息：\n' + diagnosticInfo + '\n\n';
                }
                
                userMessage += '解决方案：\n\n';
                userMessage += '1. 安装Anaconda（推荐）\n';
                userMessage += '   • 下载地址：https://www.anaconda.com/\n';
                userMessage += '   • 安装后确保conda命令可用\n\n';
                userMessage += '2. 安装Python 3.3+\n';
                userMessage += '   • 下载地址：https://www.python.org/downloads/\n';
                userMessage += '   • 安装时勾选"Add Python to PATH"\n';
                userMessage += '   • Python 3.3+默认包含venv模块\n\n';
                userMessage += '3. 验证安装\n';
                userMessage += '   • 打开命令提示符/终端\n';
                userMessage += '   • 尝试：conda --version（Anaconda）\n';
                userMessage += '   • 尝试：python --version 或 python3 --version\n';
                userMessage += '   • 尝试：python -m venv --help\n\n';
                userMessage += '安装完成后，请重启RapidBI并重试。\n\n';
                userMessage += '如果问题仍然存在，请将上述诊断信息发送给技术支持。';
            } else if (errorMessage.includes('conda')) {
                userMessage += 'Conda环境创建失败\n\n';
                userMessage += '可能的解决方案：\n';
                userMessage += '• 确保Anaconda/Miniconda已正确安装\n';
                userMessage += '• 检查conda命令是否在PATH中\n';
                userMessage += '• 尝试在命令行运行：conda --version\n';
                userMessage += '• 重启命令提示符和RapidBI\n\n';
                userMessage += `错误详情：${errorMessage}`;
            } else if (errorMessage.includes('venv')) {
                userMessage += 'Python虚拟环境创建失败\n\n';
                userMessage += '可能的解决方案：\n';
                userMessage += '• 确保Python版本为3.3或更高\n';
                userMessage += '• 检查venv模块是否可用\n';
                userMessage += '• 尝试在命令行运行：python -m venv --help\n';
                userMessage += '• 重新安装Python并确保勾选"Add to PATH"\n\n';
                userMessage += `错误详情：${errorMessage}`;
            } else {
                userMessage += `错误详情：${errorMessage}\n\n`;
                userMessage += '建议：\n';
                userMessage += '• 检查网络连接\n';
                userMessage += '• 确保有足够的磁盘空间\n';
                userMessage += '• 以管理员权限运行RapidBI\n';
                userMessage += '• 重启应用程序后重试';
            }
            
            // Show error notification instead of blocking alert
            setNotification({type: 'error', message: userMessage});
        } finally {
            setCreatingEnv(false);
        }
    };

    const handleDiagnosePython = async () => {
        setDiagnosing(true);
        try {
            const diagnostics = await DiagnosePythonInstallation();
            
            // Format diagnostic information for display
            let diagnosticText = 'Python安装诊断报告\n\n';
            
            // System info
            diagnosticText += `系统信息：${diagnostics.os} ${diagnostics.arch}\n\n`;
            
            // Conda info
            const conda = diagnostics.conda as any;
            diagnosticText += 'Conda检测：\n';
            if (conda.found) {
                diagnosticText += `  ✓ 找到conda：${conda.path}\n`;
                if (conda.working) {
                    diagnosticText += `  ✓ 版本：${conda.version}\n`;
                } else {
                    diagnosticText += `  ✗ conda命令失败：${conda.error}\n`;
                }
            } else {
                diagnosticText += `  ✗ 未找到conda：${conda.error}\n`;
            }
            diagnosticText += '\n';
            
            // Python commands info
            const pythonCommands = diagnostics.python_commands as any;
            diagnosticText += 'Python命令检测：\n';
            for (const [cmd, info] of Object.entries(pythonCommands)) {
                const cmdInfo = info as any;
                if (cmdInfo.found) {
                    diagnosticText += `  ✓ ${cmd}：${cmdInfo.path}\n`;
                    if (cmdInfo.working) {
                        diagnosticText += `    版本：${cmdInfo.version}\n`;
                        if (cmdInfo.venv_support) {
                            diagnosticText += `    ✓ 支持venv\n`;
                        } else {
                            diagnosticText += `    ✗ 不支持venv：${cmdInfo.venv_error}\n`;
                        }
                    } else {
                        diagnosticText += `    ✗ 命令失败：${cmdInfo.error}\n`;
                    }
                } else {
                    diagnosticText += `  ✗ ${cmd}：未找到\n`;
                }
            }
            diagnosticText += '\n';
            
            // Existing environments
            const envs = diagnostics.existing_environments as any[];
            diagnosticText += `现有Python环境（${envs.length}个）：\n`;
            envs.forEach((env, index) => {
                diagnosticText += `  ${index + 1}. ${env.type} - ${env.version}\n`;
                diagnosticText += `     路径：${env.path}\n`;
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
            title.textContent = 'Python安装诊断报告';
            title.style.marginTop = '0';
            
            const closeBtn = document.createElement('button');
            closeBtn.textContent = '关闭';
            closeBtn.style.marginTop = '10px';
            closeBtn.style.padding = '8px 16px';
            closeBtn.style.backgroundColor = '#3b82f6';
            closeBtn.style.color = 'white';
            closeBtn.style.border = 'none';
            closeBtn.style.borderRadius = '4px';
            closeBtn.style.cursor = 'pointer';
            closeBtn.onclick = () => document.body.removeChild(modal);
            
            const copyBtn = document.createElement('button');
            copyBtn.textContent = '复制到剪贴板';
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
                copyBtn.textContent = '已复制！';
                setTimeout(() => copyBtn.textContent = '复制到剪贴板', 2000);
            };
            
            content.appendChild(title);
            content.appendChild(textarea);
            content.appendChild(closeBtn);
            content.appendChild(copyBtn);
            modal.appendChild(content);
            document.body.appendChild(modal);
            
        } catch (error) {
            setNotification({type: 'error', message: `诊断失败：${error}`});
        } finally {
            setDiagnosing(false);
        }
    };

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
                    
                    {/* Create RapidBI Environment Button */}
                    {showCreateButton && (
                        <div className="mt-3">
                            <button
                                onClick={handleCreateRapidBIEnvironment}
                                disabled={creatingEnv || loading}
                                className="flex items-center gap-2 px-4 py-2 bg-green-600 text-white text-sm rounded-md hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                            >
                                {creatingEnv ? (
                                    <>
                                        <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                                        创建中...
                                    </>
                                ) : (
                                    <>
                                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                                        </svg>
                                        创建RapidBI环境
                                    </>
                                )}
                            </button>
                            <p className="mt-1 text-xs text-slate-500">
                                自动创建专用虚拟环境并安装所有必需的包
                            </p>
                        </div>
                    )}
                    
                    <p className="mt-1 text-[10px] text-slate-400 italic">
                        Select the Python interpreter to use for executing generated scripts.
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
                                    诊断中...
                                </>
                            ) : (
                                <>
                                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                    </svg>
                                    Python安装诊断
                                </>
                            )}
                        </button>
                        <p className="mt-1 text-xs text-slate-500">
                            检测Python安装状态，帮助解决环境问题
                        </p>
                    </div>
                </div>

                {creatingEnv && (
                    <div className="p-4 bg-blue-50 border border-blue-200 rounded-lg">
                        <div className="flex items-center gap-3">
                            <div className="w-5 h-5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
                            <div>
                                <p className="text-sm font-medium text-blue-800">正在创建RapidBI专用环境...</p>
                                <p className="text-xs text-blue-600">这可能需要几分钟时间，请耐心等待</p>
                            </div>
                        </div>
                    </div>
                )}

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
                                <div className="flex items-center justify-between mb-2">
                                    <span className="text-sm font-medium text-amber-700">Missing Recommended Packages:</span>
                                    <button
                                        onClick={handleInstallPackages}
                                        disabled={installing || validating}
                                        className="px-3 py-1 bg-blue-600 text-white text-xs rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                                    >
                                        {installing ? '安装中...' : '环境处置'}
                                    </button>
                                </div>
                                <ul className="list-disc list-inside text-xs text-amber-600 mb-2">
                                    {validation.missingPackages.map(pkg => (
                                        <li key={pkg}>{pkg}</li>
                                    ))}
                                </ul>
                                {installing && (
                                    <div className="text-xs text-blue-600 animate-pulse">
                                        正在安装缺失的包，请稍候...
                                    </div>
                                )}
                            </div>
                        )}
                        
                        {validation.valid && (!validation.missingPackages || validation.missingPackages.length === 0) && (
                            <div className="text-xs text-green-700">All required packages (matplotlib, numpy, pandas, mlxtend, sqlite3) are installed.</div>
                        )}
                    </div>
                )}
                
                {/* Notification Component */}
                {notification && (
                    <div className={`fixed top-4 right-4 max-w-md p-4 rounded-lg shadow-lg border z-50 animate-in slide-in-from-right-2 ${
                        notification.type === 'success' ? 'bg-green-50 border-green-200 text-green-800' :
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
                                        {notification.type === 'success' && '操作成功'}
                                        {notification.type === 'error' && '操作失败'}
                                        {notification.type === 'info' && '提示信息'}
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

export default PreferenceModal;