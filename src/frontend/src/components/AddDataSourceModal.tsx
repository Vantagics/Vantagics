import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom';
import { CheckCircle, Loader2 } from 'lucide-react';
import { AddDataSource, SelectExcelFile, SelectCSVFile, SelectJSONFile, SelectFolder, TestMySQLConnection, GetMySQLDatabases, GetConfig, OpenExternalURL, GetJiraProjects } from '../../wailsjs/go/main/App';
import { useLanguage } from '../i18n';
import { SystemLog } from '../utils/systemLog';

// Declare OAuth functions that will be available after rebuild
declare function StartShopifyOAuth(shop: string): Promise<string>;
declare function WaitForShopifyOAuth(): Promise<{accessToken: string, shop: string, scope: string}>;
declare function CancelShopifyOAuth(): Promise<void>;
declare function OpenShopifyOAuthInBrowser(url: string): Promise<void>;

interface AddDataSourceModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSuccess: (dataSource: any) => void;
    preSelectedDriverType?: string | null;
}

const AddDataSourceModal: React.FC<AddDataSourceModalProps> = ({ isOpen, onClose, onSuccess, preSelectedDriverType }) => {
    const { t } = useLanguage();
    const [name, setName] = useState('');
    const [driverType, setDriverType] = useState('excel');
    const [config, setConfig] = useState<Record<string, string>>({
        filePath: '',
        host: 'localhost',
        port: '',
        user: '',
        database: ''
    });
    const [isStoreLocally, setIsStoreLocally] = useState(false);

    const [isImporting, setIsImporting] = useState(false);
    const [isTesting, setIsTesting] = useState(false);
    const [availableDatabases, setAvailableDatabases] = useState<string[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [showToast, setShowToast] = useState(false);
    
    // Shopify OAuth state
    const [shopifyOAuthEnabled, setShopifyOAuthEnabled] = useState(false);
    const [isOAuthInProgress, setIsOAuthInProgress] = useState(false);
    const [oauthStatus, setOauthStatus] = useState<string>('');

    // Jira project selection state
    const [jiraProjects, setJiraProjects] = useState<Array<{key: string, name: string, id: string}>>([]);
    const [isLoadingJiraProjects, setIsLoadingJiraProjects] = useState(false);
    const [jiraProjectsError, setJiraProjectsError] = useState<string | null>(null);
    const [jiraCredentialsValid, setJiraCredentialsValid] = useState(false);

    // Etsy mode state: 'online' (API) or 'offline' (local files)
    const [etsyMode, setEtsyMode] = useState<'online' | 'offline'>('online');

    // Check if Shopify OAuth is configured - run every time modal opens or driver type changes
    useEffect(() => {
        if (isOpen && driverType === 'shopify') {
            SystemLog.info('AddDataSourceModal', 'Checking Shopify OAuth config...');
            GetConfig().then((cfg: any) => {
                const hasClientId = !!cfg.shopifyClientId;
                const hasSecret = !!cfg.shopifyClientSecret;
                SystemLog.info('AddDataSourceModal', `OAuth config check: ClientID=${hasClientId}, Secret=${hasSecret}`);
                setShopifyOAuthEnabled(hasClientId && hasSecret);
            }).catch((err) => {
                SystemLog.error('AddDataSourceModal', `Failed to get config: ${err}`);
                setShopifyOAuthEnabled(false);
            });
        }
    }, [isOpen, driverType]);

    // Set driver type from preSelectedDriverType prop
    useEffect(() => {
        if (isOpen && preSelectedDriverType) {
            setDriverType(preSelectedDriverType);
            setConfig(prev => ({ ...prev, filePath: '' }));
            // Sync etsyMode when pre-selected
            if (preSelectedDriverType === 'etsy_offline') {
                setEtsyMode('offline');
            } else if (preSelectedDriverType === 'etsy') {
                setEtsyMode('online');
            }
        }
    }, [isOpen, preSelectedDriverType]);

    if (!isOpen) return null;

    const handleBrowseFile = async () => {
        try {
            let path = '';
            if (driverType === 'excel') {
                path = await SelectExcelFile();
            } else if (driverType === 'csv') {
                path = await SelectFolder("Select CSV Directory");
            } else if (driverType === 'json') {
                path = await SelectJSONFile();
            } else if (driverType === 'etsy_offline') {
                path = await SelectFolder("Select Etsy Data Directory");
            }

            if (path) {
                setConfig(prev => ({ ...prev, filePath: path }));
                // Auto-fill name if empty
                if (!name) {
                    const fileName = path.split(/[/\\]/).pop();
                    if (fileName) {
                        setName(fileName.replace(/\.[^/.]+$/, ""));
                    }
                }
            }
        } catch (err) {
            console.error('Failed to select file:', err);
        }
    };

    const handleTestConnection = async () => {
        if (!config.host || !config.user) {
            setError('Please provide Host and User for connection test.');
            return;
        }
        setIsTesting(true);
        setError(null);
        setAvailableDatabases([]);
        try {
            await TestMySQLConnection(config.host, config.port, config.user, config.password || '');

            // Try to fetch databases
            try {
                const dbs = await GetMySQLDatabases(config.host, config.port, config.user, config.password || '');
                if (dbs && dbs.length > 0) {
                    setAvailableDatabases(dbs);
                }
            } catch (e) {
                console.warn("Could not fetch databases:", e);
            }

            // Show toast notification instead of alert
            setShowToast(true);
            setTimeout(() => setShowToast(false), 3000);
        } catch (err: any) {
            setError(t('test_connection_failed') + err);
        } finally {
            setIsTesting(false);
        }
    };

    // Fetch Jira projects when credentials are provided
    const handleFetchJiraProjects = async () => {
        if (!config.jiraBaseUrl || !config.jiraUsername || !config.jiraApiToken) {
            setJiraProjectsError(t('jira_credentials_required'));
            return;
        }

        setIsLoadingJiraProjects(true);
        setJiraProjectsError(null);
        setJiraProjects([]);
        setJiraCredentialsValid(false);

        try {
            const projects = await GetJiraProjects(
                config.jiraInstanceType || 'cloud',
                config.jiraBaseUrl,
                config.jiraUsername,
                config.jiraApiToken
            );
            setJiraProjects(projects || []);
            setJiraCredentialsValid(true);
            if (projects && projects.length > 0) {
                // Auto-fill name if empty
                if (!name && projects.length === 1) {
                    setName(projects[0].name);
                }
            }
        } catch (err: any) {
            setJiraProjectsError(err.toString());
            setJiraCredentialsValid(false);
        } finally {
            setIsLoadingJiraProjects(false);
        }
    };

    const handleImport = async () => {
        if (!name) {
            setError('Please enter a data source name');
            return;
        }
        if ((driverType === 'excel' || driverType === 'csv' || driverType === 'json' || driverType === 'etsy_offline') && !config.filePath) {
            setError(driverType === 'excel' ? 'Please select an Excel file' : driverType === 'json' ? 'Please select a JSON file' : driverType === 'etsy_offline' ? 'Please select an Etsy data folder' : 'Please select a CSV file');
            return;
        }
        if (driverType === 'shopify' && (!config.shopifyStore || !config.shopifyAccessToken)) {
            setError('Please provide Shopify store URL and access token');
            return;
        }
        if (driverType === 'bigcommerce' && (!config.bigcommerceStoreHash || !config.bigcommerceAccessToken)) {
            setError('Please provide BigCommerce store hash and access token');
            return;
        }
        if (driverType === 'ebay' && !config.ebayAccessToken) {
            setError('Please provide eBay OAuth access token');
            return;
        }
        if (driverType === 'etsy' && !config.etsyAccessToken) {
            setError('Please provide Etsy OAuth access token');
            return;
        }
        if (driverType === 'jira' && (!config.jiraBaseUrl || !config.jiraUsername || !config.jiraApiToken)) {
            setError('Please provide Jira URL, username/email, and password/API token');
            return;
        }
        if (driverType === 'snowflake' && (!config.snowflakeAccount || !config.snowflakeUser || !config.snowflakePassword)) {
            setError('Please provide Snowflake account, username, and password');
            return;
        }
        if (driverType === 'bigquery' && (!config.bigqueryProjectId || !config.bigqueryCredentials)) {
            setError('Please provide BigQuery project ID and service account credentials');
            return;
        }
        // Financial data source validation
        if (driverType === 'sp_global') {
            const apiKey = (config.financialApiKey || '').trim();
            const apiSecret = (config.financialApiSecret || '').trim();
            if (!apiKey || !apiSecret) {
                setError(t('financial_missing_sp_global'));
                return;
            }
            config.financialApiKey = apiKey;
            config.financialApiSecret = apiSecret;
            config.financialProvider = 'sp_global';
        }
        if (driverType === 'lseg') {
            const apiKey = (config.financialApiKey || '').trim();
            const username = (config.financialUsername || '').trim();
            const password = (config.financialPassword || '').trim();
            if (!apiKey || !username || !password) {
                setError(t('financial_missing_lseg'));
                return;
            }
            config.financialApiKey = apiKey;
            config.financialUsername = username;
            config.financialPassword = password;
            config.financialProvider = 'lseg';
        }
        if (driverType === 'pitchbook') {
            const apiKey = (config.financialApiKey || '').trim();
            if (!apiKey) {
                setError(t('financial_missing_pitchbook'));
                return;
            }
            config.financialApiKey = apiKey;
            config.financialProvider = 'pitchbook';
        }
        if (driverType === 'bloomberg') {
            const apiKey = (config.financialApiKey || '').trim();
            const certPath = (config.financialCertPath || '').trim();
            if (!apiKey && !certPath) {
                setError(t('financial_missing_bloomberg'));
                return;
            }
            config.financialApiKey = apiKey;
            config.financialCertPath = certPath;
            config.financialProvider = 'bloomberg';
        }
        if (driverType === 'morningstar') {
            const apiKey = (config.financialApiKey || '').trim();
            if (!apiKey) {
                setError(t('financial_missing_morningstar'));
                return;
            }
            config.financialApiKey = apiKey;
            config.financialProvider = 'morningstar';
        }
        if (driverType === 'iex_cloud') {
            const token = (config.financialToken || '').trim();
            const symbols = (config.financialSymbols || '').trim();
            if (!token) {
                setError(t('financial_missing_iex_token'));
                return;
            }
            if (!symbols) {
                setError(t('financial_missing_iex_symbols'));
                return;
            }
            config.financialToken = token;
            config.financialSymbols = symbols;
            config.financialProvider = 'iex_cloud';
        }
        if (driverType === 'alpha_vantage') {
            const apiKey = (config.financialApiKey || '').trim();
            const symbols = (config.financialSymbols || '').trim();
            if (!apiKey) {
                setError(t('financial_missing_av_key'));
                return;
            }
            if (!symbols) {
                setError(t('financial_missing_av_symbols'));
                return;
            }
            config.financialApiKey = apiKey;
            config.financialSymbols = symbols;
            config.financialProvider = 'alpha_vantage';
        }
        if (driverType === 'quandl') {
            const apiKey = (config.financialApiKey || '').trim();
            const datasetCode = (config.financialDatasetCode || '').trim();
            if (!apiKey) {
                setError(t('financial_missing_quandl_key'));
                return;
            }
            if (!datasetCode) {
                setError(t('financial_missing_quandl_dataset'));
                return;
            }
            if (!/^[A-Za-z0-9_]+\/[A-Za-z0-9_]+$/.test(datasetCode)) {
                setError(t('financial_invalid_quandl_dataset'));
                return;
            }
            config.financialApiKey = apiKey;
            config.financialDatasetCode = datasetCode;
            config.financialProvider = 'quandl';
        }

        setIsImporting(true);
        setError(null);
        try {
            const newDataSource = await AddDataSource(name, driverType, {
                ...config,
                storeLocally: isStoreLocally.toString()
            });

            // Just refresh the list
            onSuccess(null);

            onClose();
            // Reset form
            setName('');
            setDriverType('excel');
            setEtsyMode('online');
            setIsStoreLocally(false);
            setConfig({
                filePath: '',
                host: 'localhost',
                port: '',
                user: '',
                database: ''
            });
        } catch (err) {
            setError(String(err));
        } finally {
            setIsImporting(false);
        }
    };

    return ReactDOM.createPortal(
        <>
            {/* Toast Notification */}
            {showToast && (
                <div className="fixed top-4 right-4 z-[10001] animate-slide-in-right">
                    <div className="bg-green-500 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2">
                        <CheckCircle className="w-5 h-5" />
                        <span className="font-medium">{t('test_connection_success')}</span>
                    </div>
                </div>
            )}

            <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm">
                <div className="bg-white dark:bg-[#252526] w-[500px] max-h-[90vh] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900 dark:text-[#d4d4d4]">
                    <div className="p-6 border-b border-slate-200 dark:border-[#3c3c3c]">
                        <h2 className="text-xl font-bold text-slate-800 dark:text-[#d4d4d4]">{t('add_data_source')}</h2>
                    </div>

                    <div className="p-6 space-y-3 overflow-y-auto max-h-[calc(90vh-180px)]">
                        {error && (
                            <div className="p-3 bg-blue-50 border border-blue-200 text-blue-700 text-sm rounded-md break-words">
                                {error}
                            </div>
                        )}

                        <div>
                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('source_name')}</label>
                            <input
                                type="text"
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                placeholder={t('source_name_placeholder')}
                                spellCheck={false}
                                autoCorrect="off"
                                autoComplete="off"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('driver_type')}</label>
                            <select
                                value={driverType === 'etsy_offline' ? 'etsy' : driverType}
                                onChange={(e) => {
                                    const val = e.target.value;
                                    if (val === 'etsy') {
                                        // Default to online mode, let sub-selector handle the actual driverType
                                        setEtsyMode('online');
                                        setDriverType('etsy');
                                    } else {
                                        setDriverType(val);
                                    }
                                    setConfig(prev => ({ ...prev, filePath: '' })); // Reset file path on change
                                }}
                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                            >
                                <option value="excel">Excel</option>
                                <option value="csv">CSV</option>
                                <option value="json">JSON</option>
                                <option value="mysql">MySQL</option>
                                <option value="postgresql">PostgreSQL</option>
                                <option value="doris">Doris</option>
                                <option value="snowflake">Snowflake</option>
                                <option value="bigquery">BigQuery</option>
                                <option value="shopify">Shopify API</option>
                                <option value="bigcommerce">BigCommerce API</option>
                                <option value="ebay">eBay API</option>
                                <option value="etsy">Etsy</option>
                                <option value="jira">Jira</option>
                                <optgroup label={t('financial_data_group')}>
                                    <option value="sp_global">S&P Global</option>
                                    <option value="lseg">LSEG (Refinitiv)</option>
                                    <option value="pitchbook">PitchBook</option>
                                    <option value="bloomberg">Bloomberg Data License</option>
                                    <option value="morningstar">Morningstar</option>
                                    <option value="iex_cloud">IEX Cloud</option>
                                    <option value="alpha_vantage">Alpha Vantage</option>
                                    <option value="quandl">Quandl (Nasdaq Data Link)</option>
                                </optgroup>
                            </select>
                        </div>

                        {/* Etsy mode sub-selector: Online (API) vs Offline (local files) */}
                        {(driverType === 'etsy' || driverType === 'etsy_offline') && (
                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">{t('etsy_data_mode')}</label>
                                <div className="grid grid-cols-2 gap-3">
                                    <button
                                        type="button"
                                        onClick={() => { if (etsyMode !== 'online') { setEtsyMode('online'); setDriverType('etsy'); setConfig(prev => ({ ...prev, filePath: '' })); } }}
                                        className={`p-3 rounded-lg border-2 text-left transition-all ${etsyMode === 'online' ? 'border-blue-500 bg-blue-50 dark:bg-[#1a2332]' : 'border-slate-200 dark:border-[#3c3c3c] hover:border-slate-300'}`}
                                    >
                                        <div className="flex items-center gap-2 mb-1">
                                            <span className="text-lg">🌐</span>
                                            <span className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4]">{t('etsy_online_mode')}</span>
                                        </div>
                                        <p className="text-xs text-slate-500 dark:text-[#808080]">{t('etsy_online_desc')}</p>
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => { if (etsyMode !== 'offline') { setEtsyMode('offline'); setDriverType('etsy_offline'); setConfig(prev => ({ ...prev, filePath: '' })); } }}
                                        className={`p-3 rounded-lg border-2 text-left transition-all ${etsyMode === 'offline' ? 'border-blue-500 bg-blue-50 dark:bg-[#1a2332]' : 'border-slate-200 dark:border-[#3c3c3c] hover:border-slate-300'}`}
                                    >
                                        <div className="flex items-center gap-2 mb-1">
                                            <span className="text-lg">📁</span>
                                            <span className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4]">{t('etsy_offline_mode')}</span>
                                        </div>
                                        <p className="text-xs text-slate-500 dark:text-[#808080]">{t('etsy_offline_desc')}</p>
                                    </button>
                                </div>
                            </div>
                        )}

                        {driverType === 'excel' || driverType === 'csv' || driverType === 'json' || driverType === 'etsy_offline' ? (
                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">
                                    {driverType === 'csv' ? (t('csv_folder_path')) : driverType === 'etsy_offline' ? (t('etsy_offline_folder_path')) : t('file_path')}
                                </label>
                                {driverType === 'csv' && (
                                    <p className="text-xs text-slate-500 mb-2">
                                        {t('csv_folder_hint')}
                                    </p>
                                )}
                                {driverType === 'etsy_offline' && (
                                    <p className="text-xs text-slate-500 mb-2">
                                        {t('etsy_offline_folder_hint')}
                                    </p>
                                )}
                                <div className="flex gap-2">
                                    <input
                                        type="text"
                                        value={config.filePath}
                                        readOnly
                                        className="flex-1 border border-slate-300 rounded-md p-2 text-sm bg-slate-50 outline-none"
                                        placeholder={driverType === 'excel' ? "Select excel file..." : driverType === 'json' ? "Select JSON file..." : driverType === 'etsy_offline' ? "Select Etsy data folder..." : "Select csv folder..."}
                                    />
                                    <button
                                        onClick={handleBrowseFile}
                                        className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 border border-slate-300 rounded-md transition-colors"
                                    >
                                        {t('browse')}
                                    </button>
                                </div>
                            </div>
                        ) : driverType === 'shopify' ? (
                            <div className="space-y-4">
                                {/* OAuth Mode (if configured) */}
                                {shopifyOAuthEnabled ? (
                                    <>
                                        <div className="p-3 bg-green-50 border border-green-200 rounded-lg">
                                            <p className="text-sm font-medium text-green-800 mb-2">
                                                🔐 {t('shopify_oauth_mode')}
                                            </p>
                                            <p className="text-xs text-green-700">
                                                {t('shopify_oauth_desc')}
                                            </p>
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('shopify_store_url')}</label>
                                            <input
                                                type="text"
                                                value={config.shopifyStore || ''}
                                                onChange={(e) => setConfig({ ...config, shopifyStore: e.target.value })}
                                                onKeyDown={(e) => e.stopPropagation()}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                placeholder="your-store.myshopify.com"
                                                spellCheck={false}
                                                autoCorrect="off"
                                                autoComplete="off"
                                                disabled={isOAuthInProgress}
                                            />
                                        </div>
                                        {config.shopifyAccessToken ? (
                                            <div className="p-3 bg-green-50 border border-green-200 rounded-lg flex items-center gap-2">
                                                <CheckCircle className="w-5 h-5 text-green-600" />
                                                <span className="text-sm text-green-700">{t('shopify_authorized')}</span>
                                            </div>
                                        ) : (
                                            <button
                                                onClick={async () => {
                                                    if (!config.shopifyStore) {
                                                        setError(t('shopify_store_required'));
                                                        return;
                                                    }
                                                    setIsOAuthInProgress(true);
                                                    setOauthStatus(t('shopify_oauth_starting'));
                                                    setError(null);
                                                    try {
                                                        // @ts-ignore - Will be available after rebuild
                                                        const authURL = await window.go.main.App.StartShopifyOAuth(config.shopifyStore);
                                                        setOauthStatus(t('shopify_oauth_waiting'));
                                                        // @ts-ignore
                                                        await window.go.main.App.OpenShopifyOAuthInBrowser(authURL);
                                                        // @ts-ignore
                                                        const result = await window.go.main.App.WaitForShopifyOAuth();
                                                        setConfig({ 
                                                            ...config, 
                                                            shopifyStore: result.shop,
                                                            shopifyAccessToken: result.accessToken 
                                                        });
                                                        setOauthStatus('');
                                                    } catch (err: any) {
                                                        setError(err.toString());
                                                    } finally {
                                                        setIsOAuthInProgress(false);
                                                        setOauthStatus('');
                                                    }
                                                }}
                                                disabled={isOAuthInProgress || !config.shopifyStore}
                                                className={`w-full px-4 py-2 text-sm font-medium text-white rounded-md shadow-sm flex items-center justify-center gap-2 ${
                                                    isOAuthInProgress || !config.shopifyStore
                                                        ? 'bg-slate-400 cursor-not-allowed'
                                                        : 'bg-green-600 hover:bg-green-700'
                                                }`}
                                            >
                                                {isOAuthInProgress ? (
                                                    <>
                                                        <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
                                                        {oauthStatus}
                                                    </>
                                                ) : (
                                                    <>🔗 {t('shopify_authorize')}</>
                                                )}
                                            </button>
                                        )}
                                        {isOAuthInProgress && (
                                            <button
                                                onClick={async () => {
                                                    // @ts-ignore
                                                    await window.go.main.App.CancelShopifyOAuth();
                                                    setIsOAuthInProgress(false);
                                                    setOauthStatus('');
                                                }}
                                                className="w-full px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-md"
                                            >
                                                {t('cancel')}
                                            </button>
                                        )}
                                    </>
                                ) : (
                                    /* Manual Token Mode (fallback) */
                                    <>
                                        <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg">
                                            <p className="text-sm font-medium text-amber-800 mb-2">
                                                {t('shopify_setup_guide')}
                                            </p>
                                            <ol className="text-xs text-amber-700 space-y-1 list-decimal list-inside">
                                                <li>{t('shopify_step1')} → Settings → Apps</li>
                                                <li>{t('shopify_step2')}</li>
                                                <li>{t('shopify_step3')}</li>
                                                <li>{t('shopify_step4')}</li>
                                            </ol>
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('shopify_store_url')}</label>
                                            <input
                                                type="text"
                                                value={config.shopifyStore || ''}
                                                onChange={(e) => setConfig({ ...config, shopifyStore: e.target.value })}
                                                onKeyDown={(e) => e.stopPropagation()}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                placeholder="your-store.myshopify.com"
                                                spellCheck={false}
                                                autoCorrect="off"
                                                autoComplete="off"
                                            />
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('shopify_access_token')}</label>
                                            <input
                                                type="password"
                                                value={config.shopifyAccessToken || ''}
                                                onChange={(e) => setConfig({ ...config, shopifyAccessToken: e.target.value })}
                                                onKeyDown={(e) => e.stopPropagation()}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                placeholder="shpat_..."
                                                spellCheck={false}
                                                autoCorrect="off"
                                                autoComplete="off"
                                            />
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('api_version')}</label>
                                            <input
                                                type="text"
                                                value={config.shopifyAPIVersion || '2024-01'}
                                                onChange={(e) => setConfig({ ...config, shopifyAPIVersion: e.target.value })}
                                                onKeyDown={(e) => e.stopPropagation()}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                                placeholder="2024-01"
                                                spellCheck={false}
                                                autoCorrect="off"
                                                autoComplete="off"
                                            />
                                        </div>
                                    </>
                                )}
                            </div>
                        ) : driverType === 'bigcommerce' ? (
                            <div className="space-y-4">
                                <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg">
                                    <p className="text-sm font-medium text-amber-800 mb-2">
                                        {t('bigcommerce_setup_guide')}
                                    </p>
                                    <ol className="text-xs text-amber-700 space-y-1 list-decimal list-inside">
                                        <li>{t('bigcommerce_step1')} → Settings → API Accounts</li>
                                        <li>{t('bigcommerce_step2')}</li>
                                        <li>{t('bigcommerce_step3')}</li>
                                        <li>{t('bigcommerce_step4')}</li>
                                    </ol>
                                    <p className="text-xs text-amber-600 mt-2">
                                        {t('bigcommerce_path_hint')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('bigcommerce_store_hash')}</label>
                                    <input
                                        type="text"
                                        value={config.bigcommerceStoreHash || ''}
                                        onChange={(e) => setConfig({ ...config, bigcommerceStoreHash: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="abc123"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('bigcommerce_store_hash_hint')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('bigcommerce_access_token')}</label>
                                    <input
                                        type="password"
                                        value={config.bigcommerceAccessToken || ''}
                                        onChange={(e) => setConfig({ ...config, bigcommerceAccessToken: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder={t('api_access_token_placeholder')}
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('bigcommerce_token_hint')}
                                    </p>
                                </div>
                            </div>
                        ) : driverType === 'ebay' ? (
                            <div className="space-y-4">
                                <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg">
                                    <p className="text-sm font-medium text-amber-800 mb-2">
                                        {t('ebay_setup_guide')}
                                    </p>
                                    <ol className="text-xs text-amber-700 space-y-1 list-decimal list-inside">
                                        <li>{t('ebay_step1')} → <button onClick={() => OpenExternalURL('https://developer.ebay.com/my/keys')} className="text-blue-600 underline hover:text-blue-800">developer.ebay.com</button></li>
                                        <li>{t('ebay_step2')}</li>
                                        <li>{t('ebay_step3')}</li>
                                        <li>{t('ebay_step4')}</li>
                                    </ol>
                                    <p className="text-xs text-amber-600 mt-2">
                                        {t('ebay_scopes_hint')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('ebay_access_token')}</label>
                                    <input
                                        type="password"
                                        value={config.ebayAccessToken || ''}
                                        onChange={(e) => setConfig({ ...config, ebayAccessToken: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="v^1.1#i^1#p^3#..."
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('ebay_environment')}</label>
                                    <select
                                        value={config.ebayEnvironment || 'production'}
                                        onChange={(e) => setConfig({ ...config, ebayEnvironment: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                    >
                                        <option value="production">Production</option>
                                        <option value="sandbox">Sandbox</option>
                                    </select>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('ebay_apis')}</label>
                                    <div className="space-y-2 p-3 bg-slate-50 rounded-md border border-slate-200">
                                        <label className="flex items-center gap-2 cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={config.ebayApiFulfillment !== 'false'}
                                                onChange={(e) => setConfig({ ...config, ebayApiFulfillment: e.target.checked ? 'true' : 'false' })}
                                                className="rounded border-slate-300 text-blue-600"
                                            />
                                            <span className="text-sm text-slate-700">Fulfillment API</span>
                                            <span className="text-xs text-slate-500">({t('ebay_fulfillment_desc')})</span>
                                        </label>
                                        <label className="flex items-center gap-2 cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={config.ebayApiFinances !== 'false'}
                                                onChange={(e) => setConfig({ ...config, ebayApiFinances: e.target.checked ? 'true' : 'false' })}
                                                className="rounded border-slate-300 text-blue-600"
                                            />
                                            <span className="text-sm text-slate-700">Finances API</span>
                                            <span className="text-xs text-slate-500">({t('ebay_finances_desc')})</span>
                                        </label>
                                        <label className="flex items-center gap-2 cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={config.ebayApiAnalytics !== 'false'}
                                                onChange={(e) => setConfig({ ...config, ebayApiAnalytics: e.target.checked ? 'true' : 'false' })}
                                                className="rounded border-slate-300 text-blue-600"
                                            />
                                            <span className="text-sm text-slate-700">Analytics API</span>
                                            <span className="text-xs text-slate-500">({t('ebay_analytics_desc')})</span>
                                        </label>
                                    </div>
                                </div>
                            </div>
                        ) : driverType === 'etsy' ? (
                            <div className="space-y-4">
                                <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg">
                                    <p className="text-sm font-medium text-amber-800 mb-2">
                                        {t('etsy_setup_guide')}
                                    </p>
                                    <ol className="text-xs text-amber-700 space-y-1 list-decimal list-inside">
                                        <li>{t('etsy_step1')} → <button onClick={() => OpenExternalURL('https://www.etsy.com/developers/your-apps')} className="text-blue-600 underline hover:text-blue-800">etsy.com/developers</button></li>
                                        <li>{t('etsy_step2')}</li>
                                        <li>{t('etsy_step3')}</li>
                                        <li>{t('etsy_step4')}</li>
                                    </ol>
                                    <p className="text-xs text-amber-600 mt-2">
                                        {t('etsy_scopes_hint')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('etsy_access_token')}</label>
                                    <input
                                        type="password"
                                        value={config.etsyAccessToken || ''}
                                        onChange={(e) => setConfig({ ...config, etsyAccessToken: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="xxxxxxxx.xxxxxxxxxxxxxxxxxxxxxxxxx"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg">
                                    <p className="text-xs text-blue-700">
                                        💡 {t('etsy_auto_detect_hint')}
                                    </p>
                                </div>
                            </div>
                        ) : driverType === 'jira' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('jira_instance_type')}</label>
                                    <select
                                        value={config.jiraInstanceType || 'cloud'}
                                        onChange={(e) => setConfig({ ...config, jiraInstanceType: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                    >
                                        <option value="cloud">{t('jira_cloud')}</option>
                                        <option value="server">{t('jira_server')}</option>
                                    </select>
                                </div>
                                {/* Setup guide based on instance type */}
                                <div className="p-3 bg-indigo-50 border border-indigo-200 rounded-lg">
                                    <p className="text-sm font-medium text-indigo-800 mb-2">
                                        {t('jira_setup_guide')}
                                    </p>
                                    {config.jiraInstanceType === 'server' ? (
                                        <ol className="text-xs text-indigo-700 space-y-1 list-decimal list-inside">
                                            <li>{t('jira_server_step1')}</li>
                                            <li>{t('jira_server_step2')}</li>
                                            <li>{t('jira_server_step3')}</li>
                                            <li>{t('jira_server_step4')}</li>
                                        </ol>
                                    ) : (
                                        <ol className="text-xs text-indigo-700 space-y-1 list-decimal list-inside">
                                            <li>{t('jira_cloud_step1')}</li>
                                            <li>{t('jira_cloud_step2')}</li>
                                            <li>{t('jira_cloud_step3')}</li>
                                            <li>{t('jira_cloud_step4')}</li>
                                        </ol>
                                    )}
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('jira_base_url')}</label>
                                    <input
                                        type="text"
                                        value={config.jiraBaseUrl || ''}
                                        onChange={(e) => setConfig({ ...config, jiraBaseUrl: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder={config.jiraInstanceType === 'server' ? 'https://jira.yourcompany.com' : 'https://yourcompany.atlassian.net'}
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {config.jiraInstanceType === 'server' 
                                            ? (t('jira_server_url_hint'))
                                            : (t('jira_cloud_url_hint'))}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">
                                        {config.jiraInstanceType === 'server' ? (t('jira_username')) : (t('jira_email'))}
                                    </label>
                                    <input
                                        type="text"
                                        value={config.jiraUsername || ''}
                                        onChange={(e) => setConfig({ ...config, jiraUsername: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder={config.jiraInstanceType === 'server' ? 'username' : 'your.email@company.com'}
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">
                                        {config.jiraInstanceType === 'server' ? (t('jira_password')) : (t('jira_api_token'))}
                                    </label>
                                    <input
                                        type="password"
                                        value={config.jiraApiToken || ''}
                                        onChange={(e) => setConfig({ ...config, jiraApiToken: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder={config.jiraInstanceType === 'server' ? '••••••••' : 'ATATT3xFfGF0...'}
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    {config.jiraInstanceType === 'cloud' && (
                                        <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                            {t('jira_api_token_hint')}
                                        </p>
                                    )}
                                </div>
                                {/* Fetch Projects Button */}
                                <div>
                                    <button
                                        type="button"
                                        onClick={handleFetchJiraProjects}
                                        disabled={isLoadingJiraProjects || !config.jiraBaseUrl || !config.jiraUsername || !config.jiraApiToken}
                                        className="w-full px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 disabled:bg-slate-300 disabled:cursor-not-allowed flex items-center justify-center gap-2 text-sm"
                                    >
                                        {isLoadingJiraProjects ? (
                                            <>
                                                <Loader2 className="w-4 h-4 animate-spin" />
                                                {t('jira_loading_projects')}
                                            </>
                                        ) : jiraCredentialsValid ? (
                                            <>
                                                <CheckCircle className="w-4 h-4" />
                                                {t('jira_credentials_valid')}
                                            </>
                                        ) : (
                                            t('jira_fetch_projects')
                                        )}
                                    </button>
                                    {jiraProjectsError && (
                                        <p className="text-xs text-red-600 mt-0.5 leading-tight">{jiraProjectsError}</p>
                                    )}
                                </div>
                                {/* Project Selection */}
                                {jiraProjects.length > 0 && (
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-1">
                                            {t('jira_select_project')}
                                        </label>
                                        <select
                                            value={config.jiraProjectKey || ''}
                                            onChange={(e) => {
                                                setConfig({ ...config, jiraProjectKey: e.target.value });
                                                // Auto-fill name if empty
                                                if (!name && e.target.value) {
                                                    const selectedProject = jiraProjects.find(p => p.key === e.target.value);
                                                    if (selectedProject) {
                                                        setName(selectedProject.name);
                                                    }
                                                }
                                            }}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        >
                                            <option value="">{t('jira_all_projects')}</option>
                                            {jiraProjects.map((project) => (
                                                <option key={project.key} value={project.key}>
                                                    {project.key} - {project.name}
                                                </option>
                                            ))}
                                        </select>
                                        <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                            {t('jira_project_select_hint')}
                                        </p>
                                    </div>
                                )}
                                <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg">
                                    <p className="text-xs text-blue-700">
                                        💡 {t('jira_import_hint')}
                                    </p>
                                </div>
                            </div>
                        ) : driverType === 'snowflake' ? (
                            <div className="space-y-4">
                                <div className="p-2 bg-blue-50 border border-blue-200 rounded-lg">
                                    <p className="text-xs font-medium text-blue-800 mb-1 leading-tight">
                                        {t('snowflake_setup_guide')}
                                    </p>
                                    <p className="text-xs text-blue-700 leading-snug">
                                        {t('snowflake_desc')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('snowflake_account')}</label>
                                    <input
                                        type="text"
                                        value={config.snowflakeAccount || ''}
                                        onChange={(e) => setConfig({ ...config, snowflakeAccount: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="xy12345.us-east-1"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('snowflake_account_hint')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('user')}</label>
                                    <input
                                        type="text"
                                        value={config.snowflakeUser || ''}
                                        onChange={(e) => setConfig({ ...config, snowflakeUser: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder={t('username_placeholder')}
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('password')}</label>
                                    <input
                                        type="password"
                                        value={config.snowflakePassword || ''}
                                        onChange={(e) => setConfig({ ...config, snowflakePassword: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="••••••••"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('snowflake_warehouse')}</label>
                                    <input
                                        type="text"
                                        value={config.snowflakeWarehouse || ''}
                                        onChange={(e) => setConfig({ ...config, snowflakeWarehouse: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="COMPUTE_WH"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('database')}</label>
                                    <input
                                        type="text"
                                        value={config.snowflakeDatabase || ''}
                                        onChange={(e) => setConfig({ ...config, snowflakeDatabase: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="MY_DATABASE"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('snowflake_schema')}</label>
                                    <input
                                        type="text"
                                        value={config.snowflakeSchema || ''}
                                        onChange={(e) => setConfig({ ...config, snowflakeSchema: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="PUBLIC"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('snowflake_role')}</label>
                                    <input
                                        type="text"
                                        value={config.snowflakeRole || ''}
                                        onChange={(e) => setConfig({ ...config, snowflakeRole: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="ACCOUNTADMIN"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                            </div>
                        ) : driverType === 'bigquery' ? (
                            <div className="space-y-4">
                                <div className="p-2 bg-blue-50 border border-blue-200 rounded-lg">
                                    <p className="text-xs font-medium text-blue-800 mb-1 leading-tight">
                                        {t('bigquery_setup_guide')}
                                    </p>
                                    <ol className="text-xs text-blue-700 space-y-0.5 list-decimal list-inside leading-snug">
                                        <li>{t('bigquery_step1')}</li>
                                        <li>{t('bigquery_step2')}</li>
                                        <li>{t('bigquery_step3')}</li>
                                        <li>{t('bigquery_step4')}</li>
                                    </ol>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('bigquery_project_id')}</label>
                                    <input
                                        type="text"
                                        value={config.bigqueryProjectId || ''}
                                        onChange={(e) => setConfig({ ...config, bigqueryProjectId: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="my-gcp-project"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('bigquery_project_hint')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('bigquery_dataset')}</label>
                                    <input
                                        type="text"
                                        value={config.bigqueryDatasetId || ''}
                                        onChange={(e) => setConfig({ ...config, bigqueryDatasetId: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="my_dataset"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('bigquery_dataset_hint')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('bigquery_credentials')}</label>
                                    <textarea
                                        value={config.bigqueryCredentials || ''}
                                        onChange={(e) => setConfig({ ...config, bigqueryCredentials: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none font-mono resize-y"
                                        placeholder='{"type": "service_account", "project_id": "...", ...}'
                                        rows={4}
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('bigquery_credentials_hint')}
                                    </p>
                                </div>
                                <div className="p-2 bg-amber-50 border border-amber-200 rounded-lg">
                                    <p className="text-xs text-amber-700 leading-snug">
                                        ⚠️ {t('bigquery_note')}
                                    </p>
                                </div>
                            </div>
                        ) : driverType === 'sp_global' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_api_key')}</label>
                                    <input
                                        type="password"
                                        value={config.financialApiKey || ''}
                                        onChange={(e) => setConfig({ ...config, financialApiKey: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="S&P Global API Key"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_api_secret')}</label>
                                    <input
                                        type="password"
                                        value={config.financialApiSecret || ''}
                                        onChange={(e) => setConfig({ ...config, financialApiSecret: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="S&P Global API Secret"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_datasets')}</label>
                                    <div className="space-y-2 p-3 bg-slate-50 rounded-md border border-slate-200">
                                        {[
                                            { key: 'companies', label: t('sp_dataset_companies') },
                                            { key: 'financials', label: t('sp_dataset_financials') },
                                            { key: 'credit_ratings', label: t('sp_dataset_credit_ratings') },
                                            { key: 'market_data', label: t('sp_dataset_market_data') },
                                        ].map(ds => (
                                            <label key={ds.key} className="flex items-center gap-2 cursor-pointer">
                                                <input
                                                    type="checkbox"
                                                    checked={(config.financialDatasets || '').split(',').filter(Boolean).includes(ds.key)}
                                                    onChange={(e) => {
                                                        const current = (config.financialDatasets || '').split(',').filter(Boolean);
                                                        const updated = e.target.checked ? [...current, ds.key] : current.filter(d => d !== ds.key);
                                                        setConfig({ ...config, financialDatasets: updated.join(',') });
                                                    }}
                                                    className="rounded border-slate-300 text-blue-600"
                                                />
                                                <span className="text-sm text-slate-700">{ds.label}</span>
                                            </label>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        ) : driverType === 'lseg' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_app_key')}</label>
                                    <input
                                        type="password"
                                        value={config.financialApiKey || ''}
                                        onChange={(e) => setConfig({ ...config, financialApiKey: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="LSEG App Key"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_username')}</label>
                                    <input
                                        type="text"
                                        value={config.financialUsername || ''}
                                        onChange={(e) => setConfig({ ...config, financialUsername: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder={t('username_placeholder')}
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('password')}</label>
                                    <input
                                        type="password"
                                        value={config.financialPassword || ''}
                                        onChange={(e) => setConfig({ ...config, financialPassword: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="••••••••"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_datasets')}</label>
                                    <div className="space-y-2 p-3 bg-slate-50 rounded-md border border-slate-200">
                                        {[
                                            { key: 'historical_prices', label: t('lseg_dataset_historical_prices') },
                                            { key: 'fundamentals', label: t('lseg_dataset_fundamentals') },
                                            { key: 'esg', label: t('lseg_dataset_esg') },
                                        ].map(ds => (
                                            <label key={ds.key} className="flex items-center gap-2 cursor-pointer">
                                                <input
                                                    type="checkbox"
                                                    checked={(config.financialDatasets || '').split(',').filter(Boolean).includes(ds.key)}
                                                    onChange={(e) => {
                                                        const current = (config.financialDatasets || '').split(',').filter(Boolean);
                                                        const updated = e.target.checked ? [...current, ds.key] : current.filter(d => d !== ds.key);
                                                        setConfig({ ...config, financialDatasets: updated.join(',') });
                                                    }}
                                                    className="rounded border-slate-300 text-blue-600"
                                                />
                                                <span className="text-sm text-slate-700">{ds.label}</span>
                                            </label>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        ) : driverType === 'pitchbook' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_api_key')}</label>
                                    <input
                                        type="password"
                                        value={config.financialApiKey || ''}
                                        onChange={(e) => setConfig({ ...config, financialApiKey: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="PitchBook API Key"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_datasets')}</label>
                                    <div className="space-y-2 p-3 bg-slate-50 rounded-md border border-slate-200">
                                        {[
                                            { key: 'companies', label: t('pb_dataset_companies') },
                                            { key: 'deals', label: t('pb_dataset_deals') },
                                            { key: 'funds', label: t('pb_dataset_funds') },
                                            { key: 'investors', label: t('pb_dataset_investors') },
                                        ].map(ds => (
                                            <label key={ds.key} className="flex items-center gap-2 cursor-pointer">
                                                <input
                                                    type="checkbox"
                                                    checked={(config.financialDatasets || '').split(',').filter(Boolean).includes(ds.key)}
                                                    onChange={(e) => {
                                                        const current = (config.financialDatasets || '').split(',').filter(Boolean);
                                                        const updated = e.target.checked ? [...current, ds.key] : current.filter(d => d !== ds.key);
                                                        setConfig({ ...config, financialDatasets: updated.join(',') });
                                                    }}
                                                    className="rounded border-slate-300 text-blue-600"
                                                />
                                                <span className="text-sm text-slate-700">{ds.label}</span>
                                            </label>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        ) : driverType === 'bloomberg' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_api_key')}</label>
                                    <input
                                        type="password"
                                        value={config.financialApiKey || ''}
                                        onChange={(e) => setConfig({ ...config, financialApiKey: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="Bloomberg API Key"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_cert_path')}</label>
                                    <input
                                        type="text"
                                        value={config.financialCertPath || ''}
                                        onChange={(e) => setConfig({ ...config, financialCertPath: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="/path/to/certificate.pem"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('bloomberg_cert_hint')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_datasets')}</label>
                                    <div className="space-y-2 p-3 bg-slate-50 rounded-md border border-slate-200">
                                        {[
                                            { key: 'reference_data', label: t('bb_dataset_reference_data') },
                                            { key: 'pricing', label: t('bb_dataset_pricing') },
                                            { key: 'corporate_actions', label: t('bb_dataset_corporate_actions') },
                                        ].map(ds => (
                                            <label key={ds.key} className="flex items-center gap-2 cursor-pointer">
                                                <input
                                                    type="checkbox"
                                                    checked={(config.financialDatasets || '').split(',').filter(Boolean).includes(ds.key)}
                                                    onChange={(e) => {
                                                        const current = (config.financialDatasets || '').split(',').filter(Boolean);
                                                        const updated = e.target.checked ? [...current, ds.key] : current.filter(d => d !== ds.key);
                                                        setConfig({ ...config, financialDatasets: updated.join(',') });
                                                    }}
                                                    className="rounded border-slate-300 text-blue-600"
                                                />
                                                <span className="text-sm text-slate-700">{ds.label}</span>
                                            </label>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        ) : driverType === 'morningstar' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_api_key')}</label>
                                    <input
                                        type="password"
                                        value={config.financialApiKey || ''}
                                        onChange={(e) => setConfig({ ...config, financialApiKey: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="Morningstar API Key"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_datasets')}</label>
                                    <div className="space-y-2 p-3 bg-slate-50 rounded-md border border-slate-200">
                                        {[
                                            { key: 'funds', label: t('ms_dataset_funds') },
                                            { key: 'stocks', label: t('ms_dataset_stocks') },
                                            { key: 'portfolio', label: t('ms_dataset_portfolio') },
                                        ].map(ds => (
                                            <label key={ds.key} className="flex items-center gap-2 cursor-pointer">
                                                <input
                                                    type="checkbox"
                                                    checked={(config.financialDatasets || '').split(',').filter(Boolean).includes(ds.key)}
                                                    onChange={(e) => {
                                                        const current = (config.financialDatasets || '').split(',').filter(Boolean);
                                                        const updated = e.target.checked ? [...current, ds.key] : current.filter(d => d !== ds.key);
                                                        setConfig({ ...config, financialDatasets: updated.join(',') });
                                                    }}
                                                    className="rounded border-slate-300 text-blue-600"
                                                />
                                                <span className="text-sm text-slate-700">{ds.label}</span>
                                            </label>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        ) : driverType === 'iex_cloud' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_api_token')}</label>
                                    <input
                                        type="password"
                                        value={config.financialToken || ''}
                                        onChange={(e) => setConfig({ ...config, financialToken: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="pk_..."
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_symbols')}</label>
                                    <input
                                        type="text"
                                        value={config.financialSymbols || ''}
                                        onChange={(e) => setConfig({ ...config, financialSymbols: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="AAPL,MSFT,GOOGL"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('financial_symbols_hint')}
                                    </p>
                                </div>
                            </div>
                        ) : driverType === 'alpha_vantage' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_api_key')}</label>
                                    <input
                                        type="password"
                                        value={config.financialApiKey || ''}
                                        onChange={(e) => setConfig({ ...config, financialApiKey: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="Alpha Vantage API Key"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_data_type')}</label>
                                    <select
                                        value={config.financialDataType || 'time_series'}
                                        onChange={(e) => setConfig({ ...config, financialDataType: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                    >
                                        <option value="time_series">{t('av_type_time_series')}</option>
                                        <option value="forex">{t('av_type_forex')}</option>
                                        <option value="crypto">{t('av_type_crypto')}</option>
                                        <option value="technical_indicators">{t('av_type_technical_indicators')}</option>
                                    </select>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_symbols')}</label>
                                    <input
                                        type="text"
                                        value={config.financialSymbols || ''}
                                        onChange={(e) => setConfig({ ...config, financialSymbols: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="AAPL,MSFT,GOOGL"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('financial_symbols_hint')}
                                    </p>
                                </div>
                            </div>
                        ) : driverType === 'quandl' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_api_key')}</label>
                                    <input
                                        type="password"
                                        value={config.financialApiKey || ''}
                                        onChange={(e) => setConfig({ ...config, financialApiKey: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="Quandl API Key"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('financial_dataset_code')}</label>
                                    <input
                                        type="text"
                                        value={config.financialDatasetCode || ''}
                                        onChange={(e) => setConfig({ ...config, financialDatasetCode: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="WIKI/AAPL"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-0.5 leading-tight">
                                        {t('quandl_dataset_code_hint')}
                                    </p>
                                </div>
                            </div>
                        ) : (
                            <div className="grid grid-cols-2 gap-4">
                                <div className="col-span-2">
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('host')}</label>
                                    <input
                                        type="text"
                                        value={config.host}
                                        onChange={(e) => setConfig({ ...config, host: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('port')}</label>
                                    <input
                                        type="text"
                                        value={config.port}
                                        onChange={(e) => setConfig({ ...config, port: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder={driverType === 'mysql' ? '3306' : driverType === 'postgresql' ? '5432' : ''}
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('database')}</label>
                                    {availableDatabases.length > 0 ? (
                                        <div className="flex gap-2">
                                            <select
                                                value={config.database}
                                                onChange={(e) => setConfig({ ...config, database: e.target.value })}
                                                className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            >
                                                <option value="">-- Select Database --</option>
                                                {availableDatabases.map(db => (
                                                    <option key={db} value={db}>{db}</option>
                                                ))}
                                            </select>
                                            <button
                                                onClick={() => setAvailableDatabases([])}
                                                className="px-2 text-slate-400 hover:text-slate-600"
                                                title={t('switch_to_manual_entry')}
                                            >
                                                ✕
                                            </button>
                                        </div>
                                    ) : (
                                        <input
                                            type="text"
                                            value={config.database}
                                            onChange={(e) => setConfig({ ...config, database: e.target.value })}
                                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                            spellCheck={false}
                                            autoCorrect="off"
                                            autoComplete="off"
                                            placeholder={isTesting ? "Listing databases..." : ""}
                                        />
                                    )}
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('user')}</label>
                                    <input
                                        type="text"
                                        value={config.user}
                                        onChange={(e) => setConfig({ ...config, user: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('password')}</label>
                                    <input
                                        type="password"
                                        value={config.password || ''}
                                        onChange={(e) => setConfig({ ...config, password: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                </div>
                                <div className="col-span-2 flex items-center justify-between mt-2">
                                    <div className="flex items-center gap-2">
                                        <input
                                            type="checkbox"
                                            id="storeLocally"
                                            checked={isStoreLocally}
                                            onChange={(e) => setIsStoreLocally(e.target.checked)}
                                            className="rounded border-slate-300 text-blue-600 shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                                        />
                                        <label htmlFor="storeLocally" className="text-sm text-slate-700 select-none cursor-pointer">
                                            {t('store_locally')}
                                        </label>
                                    </div>
                                    <button
                                        onClick={handleTestConnection}
                                        disabled={isTesting}
                                        className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${isTesting ? 'bg-slate-100 text-slate-400' : 'bg-slate-100 text-slate-700 hover:bg-slate-200'}`}
                                    >
                                        {isTesting ? 'Testing...' : (t('test_connection'))}
                                    </button>
                                </div>
                            </div>
                        )}


                    </div>

                    <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-3">
                        <button
                            onClick={onClose}
                            disabled={isImporting}
                            className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-200 rounded-md"
                        >
                            {t('cancel')}
                        </button>
                        <button
                            onClick={handleImport}
                            disabled={isImporting}
                            className={`px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md shadow-sm flex items-center gap-2 ${isImporting ? 'opacity-70 cursor-not-allowed' : ''}`}
                        >
                            {isImporting ? (
                                <>
                                    <span className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
                                    {t('importing')}
                                </>
                            ) : (
                                t('import')
                            )}
                        </button>
                    </div>
                </div>
            </div>
        </>,
        document.body
    );
};

export default AddDataSourceModal;
