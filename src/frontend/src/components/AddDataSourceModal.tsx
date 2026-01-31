import React, { useState, useEffect } from 'react';
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
    const [shouldOptimize, setShouldOptimize] = useState(true); // Default to true
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
            setError(t('test_connection_failed') || 'Connection failed: ' + err);
        } finally {
            setIsTesting(false);
        }
    };

    // Fetch Jira projects when credentials are provided
    const handleFetchJiraProjects = async () => {
        if (!config.jiraBaseUrl || !config.jiraUsername || !config.jiraApiToken) {
            setJiraProjectsError(t('jira_credentials_required') || 'Please fill in URL, username/email, and password/API token first');
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
        if ((driverType === 'excel' || driverType === 'csv' || driverType === 'json') && !config.filePath) {
            setError(driverType === 'excel' ? 'Please select an Excel file' : driverType === 'json' ? 'Please select a JSON file' : 'Please select a CSV file');
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

        setIsImporting(true);
        setError(null);
        try {
            const newDataSource = await AddDataSource(name, driverType, {
                ...config,
                storeLocally: isStoreLocally.toString()
            });

            // Pass the data source and optimization flag to parent
            if (shouldOptimize && newDataSource?.config?.db_path && !newDataSource?.config?.optimized) {
                // Will trigger optimization in parent component
                onSuccess(newDataSource);
            } else {
                // Just refresh the list
                onSuccess(null);
            }

            onClose();
            // Reset form
            setName('');
            setDriverType('excel');
            setIsStoreLocally(false);
            setShouldOptimize(true);
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

    return (
        <>
            {/* Toast Notification */}
            {showToast && (
                <div className="fixed top-4 right-4 z-[10001] animate-slide-in-right">
                    <div className="bg-green-500 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2">
                        <CheckCircle className="w-5 h-5" />
                        <span className="font-medium">{t('test_connection_success') || 'Connection successful!'}</span>
                    </div>
                </div>
            )}

            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
                <div className="bg-white w-[500px] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900">
                    <div className="p-6 border-b border-slate-200">
                        <h2 className="text-xl font-bold text-slate-800">{t('add_data_source')}</h2>
                    </div>

                    <div className="p-6 space-y-4">
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
                                placeholder="e.g. Sales 2023"
                                spellCheck={false}
                                autoCorrect="off"
                                autoComplete="off"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('driver_type')}</label>
                            <select
                                value={driverType}
                                onChange={(e) => {
                                    setDriverType(e.target.value);
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
                                <option value="shopify">Shopify API</option>
                                <option value="bigcommerce">BigCommerce API</option>
                                <option value="ebay">eBay API</option>
                                <option value="etsy">Etsy API</option>
                                <option value="jira">Jira</option>
                            </select>
                        </div>

                        {driverType === 'excel' || driverType === 'csv' || driverType === 'json' ? (
                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">
                                    {driverType === 'csv' ? (t('csv_folder_path') || 'CSV Folder Path') : t('file_path')}
                                </label>
                                {driverType === 'csv' && (
                                    <p className="text-xs text-slate-500 mb-2">
                                        {t('csv_folder_hint') || '📁 Select a folder containing CSV files. Each CSV file in the folder will be imported as a separate data table.'}
                                    </p>
                                )}
                                <div className="flex gap-2">
                                    <input
                                        type="text"
                                        value={config.filePath}
                                        readOnly
                                        className="flex-1 border border-slate-300 rounded-md p-2 text-sm bg-slate-50 outline-none"
                                        placeholder={driverType === 'excel' ? "Select excel file..." : driverType === 'json' ? "Select JSON file..." : "Select csv folder..."}
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
                                                🔐 {t('shopify_oauth_mode') || 'One-Click Authorization'}
                                            </p>
                                            <p className="text-xs text-green-700">
                                                {t('shopify_oauth_desc') || 'Simply enter your store URL and click "Authorize" to connect securely.'}
                                            </p>
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('shopify_store_url') || 'Store URL'}</label>
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
                                                <span className="text-sm text-green-700">{t('shopify_authorized') || 'Store authorized successfully!'}</span>
                                            </div>
                                        ) : (
                                            <button
                                                onClick={async () => {
                                                    if (!config.shopifyStore) {
                                                        setError(t('shopify_store_required') || 'Please enter your store URL');
                                                        return;
                                                    }
                                                    setIsOAuthInProgress(true);
                                                    setOauthStatus(t('shopify_oauth_starting') || 'Starting authorization...');
                                                    setError(null);
                                                    try {
                                                        // @ts-ignore - Will be available after rebuild
                                                        const authURL = await window.go.main.App.StartShopifyOAuth(config.shopifyStore);
                                                        setOauthStatus(t('shopify_oauth_waiting') || 'Waiting for authorization in browser...');
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
                                                    <>🔗 {t('shopify_authorize') || 'Authorize with Shopify'}</>
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
                                                {t('cancel') || 'Cancel'}
                                            </button>
                                        )}
                                    </>
                                ) : (
                                    /* Manual Token Mode (fallback) */
                                    <>
                                        <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg">
                                            <p className="text-sm font-medium text-amber-800 mb-2">
                                                {t('shopify_setup_guide') || '📋 How to get your Access Token:'}
                                            </p>
                                            <ol className="text-xs text-amber-700 space-y-1 list-decimal list-inside">
                                                <li>{t('shopify_step1') || 'Go to your Shopify Admin'} → Settings → Apps</li>
                                                <li>{t('shopify_step2') || 'Click "Develop apps" → Create an app'}</li>
                                                <li>{t('shopify_step3') || 'Configure Admin API scopes (read_orders, read_products, read_customers)'}</li>
                                                <li>{t('shopify_step4') || 'Install app and copy the Admin API access token'}</li>
                                            </ol>
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('shopify_store_url') || 'Store URL'}</label>
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
                                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('shopify_access_token') || 'Access Token'}</label>
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
                                            <label className="block text-sm font-medium text-slate-700 mb-1">{t('api_version') || 'API Version'}</label>
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
                                        {t('bigcommerce_setup_guide') || '📋 How to get your credentials:'}
                                    </p>
                                    <ol className="text-xs text-amber-700 space-y-1 list-decimal list-inside">
                                        <li>{t('bigcommerce_step1') || 'Go to BigCommerce Admin'} → Settings → API Accounts</li>
                                        <li>{t('bigcommerce_step2') || 'Click "Create API Account" → "Create V2/V3 API Token"'}</li>
                                        <li>{t('bigcommerce_step3') || 'Set OAuth Scopes: Products, Orders, Customers (read-only)'}</li>
                                        <li>{t('bigcommerce_step4') || 'Save and copy the Access Token and API Path'}</li>
                                    </ol>
                                    <p className="text-xs text-amber-600 mt-2">
                                        {t('bigcommerce_path_hint') || 'API Path format: https://api.bigcommerce.com/stores/{store_hash}/v3'}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('bigcommerce_store_hash') || 'Store Hash'}</label>
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
                                    <p className="text-xs text-slate-500 mt-1">
                                        {t('bigcommerce_store_hash_hint') || 'Found in your BigCommerce API path: api.bigcommerce.com/stores/{store_hash}'}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('bigcommerce_access_token') || 'Access Token'}</label>
                                    <input
                                        type="password"
                                        value={config.bigcommerceAccessToken || ''}
                                        onChange={(e) => setConfig({ ...config, bigcommerceAccessToken: e.target.value })}
                                        onKeyDown={(e) => e.stopPropagation()}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                        placeholder="API Access Token"
                                        spellCheck={false}
                                        autoCorrect="off"
                                        autoComplete="off"
                                    />
                                    <p className="text-xs text-slate-500 mt-1">
                                        {t('bigcommerce_token_hint') || 'API Account Access Token from BigCommerce'}
                                    </p>
                                </div>
                            </div>
                        ) : driverType === 'ebay' ? (
                            <div className="space-y-4">
                                <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg">
                                    <p className="text-sm font-medium text-amber-800 mb-2">
                                        {t('ebay_setup_guide') || '📋 How to get your Access Token:'}
                                    </p>
                                    <ol className="text-xs text-amber-700 space-y-1 list-decimal list-inside">
                                        <li>{t('ebay_step1') || 'Go to eBay Developer Program'} → <button onClick={() => OpenExternalURL('https://developer.ebay.com/my/keys')} className="text-blue-600 underline hover:text-blue-800">developer.ebay.com</button></li>
                                        <li>{t('ebay_step2') || 'Create or select an application'}</li>
                                        <li>{t('ebay_step3') || 'Generate User Token with required OAuth scopes'}</li>
                                        <li>{t('ebay_step4') || 'Copy the OAuth User Token and paste below'}</li>
                                    </ol>
                                    <p className="text-xs text-amber-600 mt-2">
                                        {t('ebay_scopes_hint') || 'Required scopes: sell.fulfillment, sell.finances, sell.analytics.readonly'}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('ebay_access_token') || 'OAuth Access Token'}</label>
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
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('ebay_environment') || 'Environment'}</label>
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
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('ebay_apis') || 'APIs to Import'}</label>
                                    <div className="space-y-2 p-3 bg-slate-50 rounded-md border border-slate-200">
                                        <label className="flex items-center gap-2 cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={config.ebayApiFulfillment !== 'false'}
                                                onChange={(e) => setConfig({ ...config, ebayApiFulfillment: e.target.checked ? 'true' : 'false' })}
                                                className="rounded border-slate-300 text-blue-600"
                                            />
                                            <span className="text-sm text-slate-700">Fulfillment API</span>
                                            <span className="text-xs text-slate-500">({t('ebay_fulfillment_desc') || 'Orders, buyer info, payments'})</span>
                                        </label>
                                        <label className="flex items-center gap-2 cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={config.ebayApiFinances !== 'false'}
                                                onChange={(e) => setConfig({ ...config, ebayApiFinances: e.target.checked ? 'true' : 'false' })}
                                                className="rounded border-slate-300 text-blue-600"
                                            />
                                            <span className="text-sm text-slate-700">Finances API</span>
                                            <span className="text-xs text-slate-500">({t('ebay_finances_desc') || 'Transactions, fees, payouts'})</span>
                                        </label>
                                        <label className="flex items-center gap-2 cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={config.ebayApiAnalytics !== 'false'}
                                                onChange={(e) => setConfig({ ...config, ebayApiAnalytics: e.target.checked ? 'true' : 'false' })}
                                                className="rounded border-slate-300 text-blue-600"
                                            />
                                            <span className="text-sm text-slate-700">Analytics API</span>
                                            <span className="text-xs text-slate-500">({t('ebay_analytics_desc') || 'Traffic, conversion, seller metrics'})</span>
                                        </label>
                                    </div>
                                </div>
                            </div>
                        ) : driverType === 'etsy' ? (
                            <div className="space-y-4">
                                <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg">
                                    <p className="text-sm font-medium text-amber-800 mb-2">
                                        {t('etsy_setup_guide') || '📋 How to get your Access Token:'}
                                    </p>
                                    <ol className="text-xs text-amber-700 space-y-1 list-decimal list-inside">
                                        <li>{t('etsy_step1') || 'Go to Etsy Developer Portal'} → <button onClick={() => OpenExternalURL('https://www.etsy.com/developers/your-apps')} className="text-blue-600 underline hover:text-blue-800">etsy.com/developers</button></li>
                                        <li>{t('etsy_step2') || 'Create a new App (or use existing one)'}</li>
                                        <li>{t('etsy_step3') || 'In App settings, generate an OAuth token with required scopes'}</li>
                                        <li>{t('etsy_step4') || 'Copy the Access Token and paste below'}</li>
                                    </ol>
                                    <p className="text-xs text-amber-600 mt-2">
                                        {t('etsy_scopes_hint') || 'Required scopes: listings_r, transactions_r, shops_r'}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('etsy_access_token') || 'OAuth Access Token'}</label>
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
                                        💡 {t('etsy_auto_detect_hint') || 'Shop ID will be automatically detected from your token'}
                                    </p>
                                </div>
                            </div>
                        ) : driverType === 'jira' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('jira_instance_type') || 'Instance Type'}</label>
                                    <select
                                        value={config.jiraInstanceType || 'cloud'}
                                        onChange={(e) => setConfig({ ...config, jiraInstanceType: e.target.value })}
                                        className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                                    >
                                        <option value="cloud">{t('jira_cloud') || 'Jira Cloud'}</option>
                                        <option value="server">{t('jira_server') || 'Jira Server / Data Center'}</option>
                                    </select>
                                </div>
                                {/* Setup guide based on instance type */}
                                <div className="p-3 bg-indigo-50 border border-indigo-200 rounded-lg">
                                    <p className="text-sm font-medium text-indigo-800 mb-2">
                                        {t('jira_setup_guide') || '📋 How to get your credentials:'}
                                    </p>
                                    {config.jiraInstanceType === 'server' ? (
                                        <ol className="text-xs text-indigo-700 space-y-1 list-decimal list-inside">
                                            <li>{t('jira_server_step1') || 'Use your Jira Server login credentials'}</li>
                                            <li>{t('jira_server_step2') || 'Username is your Jira username (not email)'}</li>
                                            <li>{t('jira_server_step3') || 'Password is your Jira password'}</li>
                                            <li>{t('jira_server_step4') || 'Ensure your account has project access'}</li>
                                        </ol>
                                    ) : (
                                        <ol className="text-xs text-indigo-700 space-y-1 list-decimal list-inside">
                                            <li>{t('jira_cloud_step1') || 'Go to Atlassian Account Settings'}</li>
                                            <li>{t('jira_cloud_step2') || 'Navigate to Security → API tokens'}</li>
                                            <li>{t('jira_cloud_step3') || 'Click "Create API token" and copy it'}</li>
                                            <li>{t('jira_cloud_step4') || 'Use your Atlassian email as username'}</li>
                                        </ol>
                                    )}
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('jira_base_url') || 'Jira URL'}</label>
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
                                    <p className="text-xs text-slate-500 mt-1">
                                        {config.jiraInstanceType === 'server' 
                                            ? (t('jira_server_url_hint') || 'e.g., jira.your-company.com')
                                            : (t('jira_cloud_url_hint') || 'e.g., your-domain.atlassian.net')}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">
                                        {config.jiraInstanceType === 'server' ? (t('jira_username') || 'Username') : (t('jira_email') || 'Email')}
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
                                        {config.jiraInstanceType === 'server' ? (t('jira_password') || 'Password') : (t('jira_api_token') || 'API Token')}
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
                                        <p className="text-xs text-slate-500 mt-1">
                                            {t('jira_api_token_hint') || 'Generate from Atlassian Account Settings → Security → API tokens'}
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
                                                {t('jira_loading_projects') || 'Loading projects...'}
                                            </>
                                        ) : jiraCredentialsValid ? (
                                            <>
                                                <CheckCircle className="w-4 h-4" />
                                                {t('jira_credentials_valid') || 'Credentials verified'}
                                            </>
                                        ) : (
                                            t('jira_fetch_projects') || 'Verify & Fetch Projects'
                                        )}
                                    </button>
                                    {jiraProjectsError && (
                                        <p className="text-xs text-red-600 mt-1">{jiraProjectsError}</p>
                                    )}
                                </div>
                                {/* Project Selection */}
                                {jiraProjects.length > 0 && (
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-1">
                                            {t('jira_select_project') || 'Select Project (Optional)'}
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
                                            <option value="">{t('jira_all_projects') || '-- All accessible projects --'}</option>
                                            {jiraProjects.map((project) => (
                                                <option key={project.key} value={project.key}>
                                                    {project.key} - {project.name}
                                                </option>
                                            ))}
                                        </select>
                                        <p className="text-xs text-slate-500 mt-1">
                                            {t('jira_project_select_hint') || 'Select a specific project or leave empty to import all'}
                                        </p>
                                    </div>
                                )}
                                <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg">
                                    <p className="text-xs text-blue-700">
                                        💡 {t('jira_import_hint') || 'Will import: Issues, Projects, Users, Sprints'}
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
                                                title="Switch to manual entry"
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
                                    <label className="block text-sm font-medium text-slate-700 mb-1">{t('password') || 'Password'}</label>
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
                                        {isTesting ? 'Testing...' : (t('test_connection') || 'Test Connection')}
                                    </button>
                                </div>
                            </div>
                        )}

                        {/* Optimize checkbox - shown for all local databases */}
                        {(driverType === 'excel' || driverType === 'csv' || driverType === 'json' || driverType === 'shopify' || driverType === 'bigcommerce' || driverType === 'ebay' || driverType === 'etsy' || driverType === 'jira' || isStoreLocally) && (
                            <div className="flex items-center gap-2 p-3 bg-amber-50 border border-amber-200 rounded-lg">
                                <input
                                    type="checkbox"
                                    id="shouldOptimize"
                                    checked={shouldOptimize}
                                    onChange={(e) => setShouldOptimize(e.target.checked)}
                                    className="rounded border-amber-300 text-amber-600 shadow-sm focus:border-amber-300 focus:ring focus:ring-amber-200 focus:ring-opacity-50"
                                />
                                <label htmlFor="shouldOptimize" className="text-sm text-slate-700 select-none cursor-pointer flex-1">
                                    <span className="font-medium">{t('optimize_after_import') || '导入后优化数据'}</span>
                                    <span className="block text-xs text-slate-500 mt-0.5">
                                        {t('optimize_description') || '自动创建索引以提升查询性能'}
                                    </span>
                                </label>
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
        </>
    );
};

export default AddDataSourceModal;
