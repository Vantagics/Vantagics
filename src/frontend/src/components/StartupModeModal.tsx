import React, { useState, useEffect } from 'react';
import { Building2, Code2, Key, Mail, Loader2, CheckCircle, AlertCircle, ArrowLeft, Settings, ExternalLink } from 'lucide-react';
import { GetConfig, SaveConfig, GetActivationStatus, ActivateLicense, TestLLMConnection, RequestSN, LoadSavedActivation } from '../../wailsjs/go/main/App';
import { BrowserOpenURL, EventsEmit } from '../../wailsjs/runtime/runtime';
import { useLanguage } from '../i18n';
import { createLogger } from '../utils/systemLog';

const logger = createLogger('StartupModeModal');

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
    const serverURL = 'https://license.vantagedata.chat';
    const [sn, setSN] = useState('');
    const [activationEmail, setActivationEmail] = useState('');
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
                if (config.licenseEmail) {
                    setActivationEmail(config.licenseEmail);
                }
                // Try to load from local encrypted storage
                const loadResult = await LoadSavedActivation(config.licenseSN);
                if (loadResult.success) {
                    // Successfully loaded from local storage
                    setSuccessMessage(t('activation_loaded'));
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
        setMode('commercial');
        setCommercialStep('activating');
        try {
            const config = await GetConfig();
            const result = await TestLLMConnection(config);
            if (result.success) {
                onComplete();
            } else {
                setSuccessMessage(null);
                setError(t('llm_connection_failed'));
                setCommercialStep('check');
            }
        } catch (err: any) {
            setSuccessMessage(null);
            setError(err.toString());
            setCommercialStep('check');
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
        if (!activateSN) {
            setError(t('please_fill_server_and_sn'));
            return;
        }

        if (!activationEmail) {
            setError(t('activation_email_required'));
            return;
        }
        const atIndex = activationEmail.indexOf('@');
        if (atIndex < 1 || atIndex >= activationEmail.length - 1 || !activationEmail.substring(atIndex + 1).includes('.')) {
            setError(t('please_enter_valid_email'));
            return;
        }

        if (!serverURL) {
            setError(t('please_fill_server_and_sn'));
            return;
        }

        setIsLoading(true);
        setError(null);
        setCommercialStep('activating');

        try {
            const result = await ActivateLicense(serverURL, activateSN);
            if (result.success) {
                // Save SN and email to config
                const config = await GetConfig() as any;
                config.licenseSN = activateSN;
                config.licenseServerURL = serverURL;
                config.licenseEmail = activationEmail;
                await SaveConfig(config);
                
                setSuccessMessage(t('activation_success'));
                
                // Emit event to notify AboutModal to refresh activation status
                EventsEmit('activation-status-changed');
                
                // Verify LLM connection (verifyAndComplete handles its own loading state)
                await verifyAndComplete();
            } else {
                setError(result.message);
                setCommercialStep('check');
                setIsLoading(false);
            }
        } catch (err: any) {
            setError(err.toString());
            setCommercialStep('check');
            setIsLoading(false);
        }
    };

    const handleRequestSN = async () => {
        if (!email || !email.includes('@')) {
            setError(t('please_enter_valid_email'));
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
                setSN(result.sn);
                setActivationEmail(email);
                setSuccessMessage(t('sn_request_success'));
                // Auto-switch to activation step with the new SN
                setCommercialStep('check');
            } else {
                // Check if it's an "not invited" error
                if (result.code === 'not_invited' || result.message?.includes('not invited') || result.message?.includes('未被邀请')) {
                    setError(t('email_not_invited_text'));
                    setIsNotInvitedError(true);
                } else if (result.code) {
                    // Use localized error message based on error code
                    const localizedError = t(`license_error_${result.code}`);
                    setError(localizedError || result.message);
                    setIsNotInvitedError(false);
                } else {
                    setError(result.message);
                    setIsNotInvitedError(false);
                }
            }
        } catch (err: any) {
            setError(t('server_connection_failed'));
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
        <div className="fixed inset-0 bg-gradient-to-br from-slate-100 via-slate-50 to-[#f0f4f8] dark:from-[#1e1e1e] dark:via-[#1e1e1e] dark:to-[#1a2332] flex items-center justify-center z-[10000]">
            <div className="bg-white dark:bg-[#252526] rounded-2xl shadow-2xl w-[520px] overflow-hidden">
                {/* Header */}
                <div className="p-6 bg-gradient-to-r from-[#5b7a9d] to-[#6b8db5] text-white">
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
                                {mode === 'select' && (t('welcome_to_vantagedata'))}
                                {mode === 'commercial' && (t('commercial_mode'))}
                                {mode === 'opensource' && (t('opensource_mode'))}
                            </h1>
                            <p className="text-blue-100 text-sm mt-1">
                                {mode === 'select' && (t('select_usage_mode'))}
                                {mode === 'commercial' && (t('activate_with_sn'))}
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
                                {t('mode_selection_desc')}
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
                                            {t('commercial_mode')}
                                        </h3>
                                        <p className="text-slate-500 text-sm mt-1">
                                            {t('commercial_mode_desc')}
                                        </p>
                                        <div className="flex items-center gap-2 mt-3 text-xs text-blue-600">
                                            <CheckCircle className="w-4 h-4" />
                                            <span>{t('no_config_needed')}</span>
                                            <CheckCircle className="w-4 h-4 ml-2" />
                                            <span>{t('cloud_llm_service')}</span>
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
                                            {t('opensource_mode')}
                                        </h3>
                                        <p className="text-slate-500 text-sm mt-1">
                                            {t('opensource_mode_desc')}
                                        </p>
                                        <div className="flex items-center gap-2 mt-3 text-xs text-green-600">
                                            <Settings className="w-4 h-4" />
                                            <span>{t('custom_llm_config')}</span>
                                            <Code2 className="w-4 h-4 ml-2" />
                                            <span>{t('full_control')}</span>
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
                                                {t('serial_number')}
                                            </label>
                                            <button
                                                onClick={() => setCommercialStep('request')}
                                                className="text-xs text-blue-600 hover:text-blue-800"
                                            >
                                                {t('no_sn_request_trial')}
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

                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-1">
                                            {t('activation_email_label')}
                                        </label>
                                        <input
                                            type="email"
                                            value={activationEmail}
                                            onChange={(e) => setActivationEmail(e.target.value)}
                                            className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-sm"
                                            placeholder={t('activation_email_placeholder')}
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
                                                {t('activating')}
                                            </>
                                        ) : (
                                            <>
                                                <Key className="w-4 h-4" />
                                                {t('activate')}
                                            </>
                                        )}
                                    </button>
                                </>
                            )}

                            {commercialStep === 'request' && (
                                <>
                                    <p className="text-sm text-slate-600">
                                        {t('request_trial_desc')}
                                    </p>
                                    
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-1">
                                            {t('email_address')}
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
                                                {t('requesting')}
                                            </>
                                        ) : (
                                            <>
                                                <Mail className="w-4 h-4" />
                                                {t('request_trial_sn')}
                                            </>
                                        )}
                                    </button>

                                    <p className="text-xs text-slate-400 text-center">
                                        {t('trial_limit_note')}
                                    </p>
                                </>
                            )}

                            {commercialStep === 'activating' && (
                                <div className="py-8 text-center">
                                    <Loader2 className="w-12 h-12 text-blue-600 animate-spin mx-auto mb-4" />
                                    <p className="text-slate-600">
                                        {t('activating_and_verifying')}
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
