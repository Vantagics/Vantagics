import React, { useState, useEffect } from 'react';
import { Building2, Code2, Key, Mail, Loader2, CheckCircle, AlertCircle, ArrowLeft, Settings, ExternalLink } from 'lucide-react';
import { GetConfig, SaveConfig, GetActivationStatus, ActivateLicense, TestLLMConnection, RequestSN, LoadSavedActivation } from '../../wailsjs/go/main/App';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { useLanguage } from '../i18n';

interface StartupModeModalProps {
    isOpen: boolean;
    onComplete: () => void;
    onOpenSettings: () => void;
}

type Mode = 'select' | 'commercial' | 'opensource';
type CommercialStep = 'check' | 'request' | 'activating';

const StartupModeModal: React.FC<StartupModeModalProps> = ({ isOpen, onComplete, onOpenSettings }) => {
    const { t } = useLanguage();
    const [mode, setMode] = useState<Mode>('select');
    const [commercialStep, setCommercialStep] = useState<CommercialStep>('check');
    
    // Commercial mode state - server URL is fixed, not user-configurable
    const serverURL = 'http://license.vantagedata.chat:6699';
    const [sn, setSN] = useState('');
    const [email, setEmail] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [isNotInvitedError, setIsNotInvitedError] = useState(false);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);

    const INVITE_URL = 'https://vantagedata.chat/invite';

    const handleOpenInviteLink = () => {
        BrowserOpenURL(INVITE_URL);
    };

    useEffect(() => {
        if (isOpen) {
            checkExistingActivation();
        }
    }, [isOpen]);

    const checkExistingActivation = async () => {
        try {
            // First check if already activated in memory
            const status = await GetActivationStatus();
            if (status.activated) {
                // Already activated, verify LLM and complete
                await verifyAndComplete();
                return;
            }
            
            // Try to load saved activation from local storage
            const config = await GetConfig() as any;
            if (config.licenseSN) {
                setSN(config.licenseSN);
                // Try to load from local encrypted storage
                const loadResult = await LoadSavedActivation(config.licenseSN);
                if (loadResult.success) {
                    // Successfully loaded from local storage
                    setSuccessMessage(t('activation_loaded') || '已从本地加载激活信息');
                    await verifyAndComplete();
                    return;
                }
                // If local load failed, user needs to re-activate online
                setMode('commercial');
                setCommercialStep('check');
            }
        } catch (err) {
            console.error('Failed to check activation status:', err);
        }
    };

    const verifyAndComplete = async () => {
        setIsLoading(true);
        setError(null);
        try {
            const config = await GetConfig();
            const result = await TestLLMConnection(config);
            if (result.success) {
                onComplete();
            } else {
                setError(t('llm_connection_failed') || 'LLM连接失败，请检查配置');
            }
        } catch (err: any) {
            setError(err.toString());
        } finally {
            setIsLoading(false);
        }
    };

    const handleSelectCommercial = () => {
        setMode('commercial');
        setCommercialStep('check');
        setError(null);
        // Check if we have saved SN
        checkSavedSN();
    };

    const checkSavedSN = async () => {
        try {
            const config = await GetConfig() as any;
            if (config.licenseSN) {
                setSN(config.licenseSN);
                // Auto-activate with saved SN
                handleActivate(config.licenseSN);
            }
        } catch (err) {
            console.error('Failed to check saved SN:', err);
        }
    };

    const handleSelectOpenSource = () => {
        setMode('opensource');
        onOpenSettings();
    };

    const handleActivate = async (snToUse?: string) => {
        const activateSN = snToUse || sn;
        if (!serverURL || !activateSN) {
            setError(t('please_fill_server_and_sn') || '请填写服务器地址和序列号');
            return;
        }

        setIsLoading(true);
        setError(null);
        setCommercialStep('activating');

        try {
            const result = await ActivateLicense(serverURL, activateSN);
            if (result.success) {
                // Save SN to config
                const config = await GetConfig() as any;
                config.licenseSN = activateSN;
                config.licenseServerURL = serverURL;
                await SaveConfig(config);
                
                setSuccessMessage(t('activation_success') || '激活成功！');
                // Verify LLM connection
                setTimeout(() => verifyAndComplete(), 1000);
            } else {
                setError(result.message);
                setCommercialStep('check');
            }
        } catch (err: any) {
            setError(err.toString());
            setCommercialStep('check');
        } finally {
            setIsLoading(false);
        }
    };

    const handleRequestSN = async () => {
        if (!email || !email.includes('@')) {
            setError(t('please_enter_valid_email') || '请输入有效的邮箱地址');
            setIsNotInvitedError(false);
            return;
        }

        setIsLoading(true);
        setError(null);
        setIsNotInvitedError(false);

        try {
            // Use Go backend to request SN (avoids CORS issues)
            const result = await RequestSN(serverURL, email);
            
            if (result.success) {
                setSN(result.sn || '');
                setSuccessMessage(t('sn_request_success') || '序列号申请成功！');
                // Auto-switch to activation step with the new SN
                setCommercialStep('check');
            } else {
                // Check if it's an "not invited" error
                if (result.code === 'not_invited' || result.message?.includes('not invited') || result.message?.includes('未被邀请')) {
                    setError(t('email_not_invited_text') || '当前未被邀请使用，请点击下方链接获取帮助。');
                    setIsNotInvitedError(true);
                } else if (result.code === 'rate_limit') {
                    setError(result.message || t('request_rate_limit') || '请求次数已达上限，请明天再试');
                    setIsNotInvitedError(false);
                } else {
                    setError(result.message);
                    setIsNotInvitedError(false);
                }
            }
        } catch (err: any) {
            setError(t('server_connection_failed') || '连接服务器失败: ' + err.toString());
            setIsNotInvitedError(false);
        } finally {
            setIsLoading(false);
        }
    };

    const handleBack = () => {
        if (mode === 'commercial' && commercialStep === 'request') {
            setCommercialStep('check');
        } else {
            setMode('select');
        }
        setError(null);
        setIsNotInvitedError(false);
        setSuccessMessage(null);
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-gradient-to-br from-slate-100 via-slate-50 to-blue-50 flex items-center justify-center z-[10000]">
            <div className="bg-white rounded-2xl shadow-2xl w-[520px] overflow-hidden">
                {/* Header */}
                <div className="p-6 bg-gradient-to-r from-blue-600 to-indigo-600 text-white">
                    <div className="flex items-center gap-3">
                        {mode !== 'select' && (
                            <button 
                                onClick={handleBack}
                                className="p-1.5 hover:bg-white/20 rounded-lg transition-colors"
                            >
                                <ArrowLeft className="w-5 h-5" />
                            </button>
                        )}
                        <div>
                            <h1 className="text-xl font-bold">
                                {mode === 'select' && (t('welcome_to_vantagedata') || '欢迎使用 VantageData')}
                                {mode === 'commercial' && (t('commercial_mode') || '商业软件模式')}
                                {mode === 'opensource' && (t('opensource_mode') || '开源模式')}
                            </h1>
                            <p className="text-blue-100 text-sm mt-1">
                                {mode === 'select' && (t('select_usage_mode') || '请选择您的使用模式')}
                                {mode === 'commercial' && (t('activate_with_sn') || '使用序列号激活')}
                            </p>
                        </div>
                    </div>
                </div>

                {/* Content */}
                <div className="p-6">
                    {/* Mode Selection */}
                    {mode === 'select' && (
                        <div className="space-y-4">
                            <p className="text-slate-600 text-sm mb-6">
                                {t('mode_selection_desc') || '首次使用需要选择使用模式。商业模式提供云端LLM服务，开源模式需要自行配置LLM。'}
                            </p>
                            
                            {/* Commercial Mode Card */}
                            <button
                                onClick={handleSelectCommercial}
                                className="w-full p-5 border-2 border-slate-200 rounded-xl hover:border-blue-500 hover:bg-blue-50/50 transition-all text-left group"
                            >
                                <div className="flex items-start gap-4">
                                    <div className="p-3 bg-blue-100 rounded-xl group-hover:bg-blue-200 transition-colors">
                                        <Building2 className="w-6 h-6 text-blue-600" />
                                    </div>
                                    <div className="flex-1">
                                        <h3 className="font-semibold text-slate-800 text-lg">
                                            {t('commercial_mode') || '商业软件模式'}
                                        </h3>
                                        <p className="text-slate-500 text-sm mt-1">
                                            {t('commercial_mode_desc') || '使用序列号激活，享受云端LLM服务，无需自行配置'}
                                        </p>
                                        <div className="flex items-center gap-2 mt-3 text-xs text-blue-600">
                                            <CheckCircle className="w-4 h-4" />
                                            <span>{t('no_config_needed') || '无需配置'}</span>
                                            <CheckCircle className="w-4 h-4 ml-2" />
                                            <span>{t('cloud_llm_service') || '云端LLM服务'}</span>
                                        </div>
                                    </div>
                                </div>
                            </button>

                            {/* Open Source Mode Card */}
                            <button
                                onClick={handleSelectOpenSource}
                                className="w-full p-5 border-2 border-slate-200 rounded-xl hover:border-green-500 hover:bg-green-50/50 transition-all text-left group"
                            >
                                <div className="flex items-start gap-4">
                                    <div className="p-3 bg-green-100 rounded-xl group-hover:bg-green-200 transition-colors">
                                        <Code2 className="w-6 h-6 text-green-600" />
                                    </div>
                                    <div className="flex-1">
                                        <h3 className="font-semibold text-slate-800 text-lg">
                                            {t('opensource_mode') || '开源模式'}
                                        </h3>
                                        <p className="text-slate-500 text-sm mt-1">
                                            {t('opensource_mode_desc') || '自行配置LLM API，支持OpenAI、Anthropic、本地模型等'}
                                        </p>
                                        <div className="flex items-center gap-2 mt-3 text-xs text-green-600">
                                            <Settings className="w-4 h-4" />
                                            <span>{t('custom_llm_config') || '自定义LLM配置'}</span>
                                            <Code2 className="w-4 h-4 ml-2" />
                                            <span>{t('full_control') || '完全控制'}</span>
                                        </div>
                                    </div>
                                </div>
                            </button>
                        </div>
                    )}

                    {/* Commercial Mode - Activation */}
                    {mode === 'commercial' && (
                        <div className="space-y-4">
                            {/* Error/Success Messages */}
                            {error && (
                                <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
                                    <div className="flex items-start gap-2">
                                        <AlertCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                        <span className="text-sm text-red-700">{error}</span>
                                    </div>
                                    {isNotInvitedError && (
                                        <button
                                            onClick={handleOpenInviteLink}
                                            className="mt-2 ml-6 flex items-center gap-1 text-sm text-blue-600 hover:text-blue-800 hover:underline"
                                        >
                                            <ExternalLink className="w-3.5 h-3.5" />
                                            {INVITE_URL}
                                        </button>
                                    )}
                                </div>
                            )}
                            {successMessage && (
                                <div className="p-3 bg-green-50 border border-green-200 rounded-lg flex items-start gap-2">
                                    <CheckCircle className="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
                                    <span className="text-sm text-green-700">{successMessage}</span>
                                </div>
                            )}

                            {commercialStep === 'check' && (
                                <>
                                    <div>
                                        <div className="flex items-center justify-between mb-1">
                                            <label className="block text-sm font-medium text-slate-700">
                                                {t('serial_number') || '序列号 (SN)'}
                                            </label>
                                            <button
                                                onClick={() => setCommercialStep('request')}
                                                className="text-xs text-blue-600 hover:text-blue-800"
                                            >
                                                {t('no_sn_request_trial') || '没有序列号？申请试用'}
                                            </button>
                                        </div>
                                        <input
                                            type="text"
                                            value={sn}
                                            onChange={(e) => setSN(e.target.value.toUpperCase())}
                                            className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-sm font-mono"
                                            placeholder="XXXX-XXXX-XXXX-XXXX"
                                        />
                                    </div>

                                    <button
                                        onClick={() => handleActivate()}
                                        disabled={isLoading || !sn}
                                        className="w-full py-2.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-slate-300 disabled:cursor-not-allowed flex items-center justify-center gap-2 transition-colors"
                                    >
                                        {isLoading ? (
                                            <>
                                                <Loader2 className="w-4 h-4 animate-spin" />
                                                {t('activating') || '正在激活...'}
                                            </>
                                        ) : (
                                            <>
                                                <Key className="w-4 h-4" />
                                                {t('activate') || '激活'}
                                            </>
                                        )}
                                    </button>
                                </>
                            )}

                            {commercialStep === 'request' && (
                                <>
                                    <p className="text-sm text-slate-600">
                                        {t('request_trial_desc') || '输入您的邮箱地址，我们将发送试用序列号给您。'}
                                    </p>
                                    
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-1">
                                            {t('email_address') || '邮箱地址'}
                                        </label>
                                        <input
                                            type="email"
                                            value={email}
                                            onChange={(e) => setEmail(e.target.value)}
                                            className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-sm"
                                            placeholder="your@email.com"
                                        />
                                    </div>

                                    <button
                                        onClick={handleRequestSN}
                                        disabled={isLoading || !email}
                                        className="w-full py-2.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-slate-300 disabled:cursor-not-allowed flex items-center justify-center gap-2 transition-colors"
                                    >
                                        {isLoading ? (
                                            <>
                                                <Loader2 className="w-4 h-4 animate-spin" />
                                                {t('requesting') || '正在申请...'}
                                            </>
                                        ) : (
                                            <>
                                                <Mail className="w-4 h-4" />
                                                {t('request_trial_sn') || '申请试用序列号'}
                                            </>
                                        )}
                                    </button>

                                    <p className="text-xs text-slate-400 text-center">
                                        {t('trial_limit_note') || '* 每个邮箱仅可申请一次试用'}
                                    </p>
                                </>
                            )}

                            {commercialStep === 'activating' && (
                                <div className="py-8 text-center">
                                    <Loader2 className="w-12 h-12 text-blue-600 animate-spin mx-auto mb-4" />
                                    <p className="text-slate-600">
                                        {t('activating_and_verifying') || '正在激活并验证LLM服务...'}
                                    </p>
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default StartupModeModal;
