import React, { useState, useEffect } from 'react';
import { useLanguage } from '../i18n';
import { X, AlertCircle, CheckCircle } from 'lucide-react';

interface SearchEngine {
    id: string;
    name: string;
    url: string;
    enabled: boolean;
    tested: boolean;
}

interface SearchEngineModalProps {
    isOpen: boolean;
    engine: SearchEngine | null;
    onClose: () => void;
    onSave: (engine: SearchEngine) => void;
}

const SearchEngineModal: React.FC<SearchEngineModalProps> = ({ isOpen, engine, onClose, onSave }) => {
    const { t } = useLanguage();
    const [name, setName] = useState('');
    const [url, setUrl] = useState('');
    const [testing, setTesting] = useState(false);
    const [testingTools, setTestingTools] = useState(false);
    const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
    const [tested, setTested] = useState(false);

    useEffect(() => {
        if (isOpen) {
            if (engine) {
                // Editing existing engine
                setName(engine.name);
                setUrl(engine.url);
                setTested(engine.tested);
                setTestResult(null);
            } else {
                // Adding new engine
                setName('');
                setUrl('');
                setTested(false);
                setTestResult(null);
            }
        }
    }, [isOpen, engine]);

    const handleTest = async () => {
        if (!url) {
            setTestResult({ success: false, message: t('mcp_service_url_required') });
            return;
        }

        setTesting(true);
        setTestResult(null);

        try {
            // @ts-ignore - TestSearchEngine is defined in App.go
            const result = await window.go.main.App.TestSearchEngine(url);
            setTestResult(result);
            if (result.success) {
                setTested(true);
            }
        } catch (err) {
            setTestResult({ success: false, message: String(err) });
        } finally {
            setTesting(false);
        }
    };

    const handleTestSearchTools = async () => {
        if (!url) {
            setTestResult({ success: false, message: t('mcp_service_url_required') });
            return;
        }

        setTestingTools(true);
        setTestResult(null);

        try {
            // @ts-ignore - TestSearchTools is defined in App.go
            const result = await window.go.main.App.TestSearchTools(url);
            setTestResult(result);
            if (result.success) {
                setTested(true);
            }
        } catch (err) {
            setTestResult({ success: false, message: String(err) });
        } finally {
            setTestingTools(false);
        }
    };

    const handleSave = () => {
        // Validation
        if (!name.trim()) {
            setTestResult({ success: false, message: t('mcp_service_name_required') });
            return;
        }

        if (!url.trim()) {
            setTestResult({ success: false, message: t('mcp_service_url_required') });
            return;
        }

        if (!tested) {
            setTestResult({ success: false, message: t('search_engine_test_required') });
            return;
        }

        // Generate ID for new engines
        const engineId = engine?.id || `custom-${Date.now()}`;

        const newEngine: SearchEngine = {
            id: engineId,
            name: name.trim(),
            url: url.trim(),
            enabled: true,
            tested: tested
        };

        onSave(newEngine);
        onClose();
    };

    const handleUrlChange = (newUrl: string) => {
        setUrl(newUrl);
        // Clear tested status when URL changes
        if (newUrl !== engine?.url) {
            setTested(false);
            setTestResult(null);
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[500px] rounded-xl shadow-2xl overflow-hidden text-slate-900">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-slate-200">
                    <h2 className="text-xl font-semibold text-slate-800">
                        {engine ? t('edit_search_engine') : t('add_custom_engine')}
                    </h2>
                    <button
                        onClick={onClose}
                        className="p-1 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                        <X className="w-5 h-5 text-slate-500" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 space-y-4">
                    <div>
                        <label htmlFor="engineName" className="block text-sm font-medium text-slate-700 mb-1">
                            {t('search_engine_name')}
                        </label>
                        <input
                            id="engineName"
                            type="text"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            placeholder="e.g., DuckDuckGo"
                            className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                            autoFocus
                        />
                    </div>

                    <div>
                        <label htmlFor="engineUrl" className="block text-sm font-medium text-slate-700 mb-1">
                            {t('search_engine_url')}
                        </label>
                        <input
                            id="engineUrl"
                            type="text"
                            value={url}
                            onChange={(e) => handleUrlChange(e.target.value)}
                            placeholder="e.g., duckduckgo.com"
                            className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                        <p className="mt-1 text-xs text-slate-500">
                            {t('search_engine_url_hint')}
                        </p>
                    </div>

                    {/* Test Button */}
                    <div className="pt-2">
                        <button
                            onClick={handleTest}
                            disabled={testing || !url}
                            className={`w-full px-4 py-2 text-sm font-semibold rounded-lg transition-colors ${
                                testing || !url
                                    ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                                    : 'bg-blue-600 text-white hover:bg-blue-700'
                            }`}
                        >
                            {testing ? t('testing_mcp_service') : t('test_connection')}
                        </button>
                    </div>

                    {/* Test Search Tools Button */}
                    <div className="pt-2">
                        <button
                            onClick={handleTestSearchTools}
                            disabled={testingTools || !url}
                            className={`w-full px-4 py-2 text-sm font-semibold rounded-lg transition-colors ${
                                testingTools || !url
                                    ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                                    : 'bg-green-600 text-white hover:bg-green-700'
                            }`}
                        >
                            {testingTools ? t('testing_search_tools') : t('test_search_tools')}
                        </button>
                        <p className="mt-1 text-xs text-slate-500">
                            {t('test_search_tools_hint')}
                        </p>
                    </div>

                    {/* Test Result */}
                    {testResult && (
                        <div
                            className={`flex items-start gap-2 p-3 rounded-lg ${
                                testResult.success
                                    ? 'bg-green-50 text-green-800'
                                    : 'bg-red-50 text-red-800'
                            }`}
                        >
                            {testResult.success ? (
                                <CheckCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
                            ) : (
                                <AlertCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
                            )}
                            <div className="flex-1">
                                <p className="text-sm font-medium">
                                    {testResult.success ? t('mcp_test_success') : t('mcp_test_failed')}
                                </p>
                                <p className="text-xs mt-1">{testResult.message}</p>
                            </div>
                        </div>
                    )}

                    {/* Warning if not tested */}
                    {!tested && !testResult && (
                        <div className="flex items-start gap-2 p-3 bg-amber-50 text-amber-800 rounded-lg">
                            <AlertCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
                            <p className="text-sm">{t('search_engine_test_required')}</p>
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="flex items-center justify-end gap-3 p-6 border-t border-slate-200 bg-slate-50">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                        {t('cancel')}
                    </button>
                    <button
                        onClick={handleSave}
                        disabled={!tested}
                        className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                            tested
                                ? 'bg-blue-600 text-white hover:bg-blue-700'
                                : 'bg-slate-200 text-slate-400 cursor-not-allowed'
                        }`}
                    >
                        {t('save_changes')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default SearchEngineModal;
