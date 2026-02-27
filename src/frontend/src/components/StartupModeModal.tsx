import React, { useState, useEffect } from 'react';
import { Building2, Code2, Key, Mail, Loader2, CheckCircle, AlertCircle, ArrowLeft, Settings, ExternalLink, Gift } from 'lucide-react';
import { GetConfig, GetEffectiveConfig, SaveConfig, GetActivationStatus, ActivateLicense, TestLLMConnection, RequestSN, RequestFreeSN, RequestOpenSourceSN, LoadSavedActivation } from '../../wailsjs/go/main/App';
import { BrowserOpenURL, EventsEmit } from '../../wailsjs/runtime/runtime';
import { useLanguage } from '../i18n';
import { createLogger } from '../utils/systemLog';

const logger = createLogger('StartupModeModal');

interface StartupModeModalProps {
    isOpen: boolean;
    onComplete: () => void;
    onOpenSettings: () => void;
    initialMode?: Mode;
}

type Mode = 'select' | 'commercial' | 'opensource' | 'free';
type CommercialStep = 'check' | 'activating';

const StartupModeModal: React.FC<StartupModeModalProps> = ({ isOpen, onComplete, onOpenSettings, initialMode }) => {
    const { t } = useLanguage();
    const [mode, setMode] = useState<Mode>('select');
    const [commercialStep, setCommercialStep] = useState<CommercialStep>('check');

    // Set initial mode when modal opens with initialMode prop
    useEffect(() => {
        if (isOpen && initialMode) {
            setMode(initialMode);
            if (initialMode === 'free') {
                setFreeEmail('');
                setError(null);
            } else if (initialMode === 'opensource') {
                setOssEmail('');
                setError(null);
            }
        }
    }, [isOpen, initialMode]);
    
    // Commercial mode state - server URL is fixed, not user-configurable
    const serverURL = 'https://license.vantagics.com';
    const [sn, setSN] = useState('');
    const [activationEmail, setActivationEmail] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [isNotInvitedError, setIsNotInvitedError] = useState(false);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);

    // Free registration state
    const [freeEmail, setFreeEmail] = useState('');

    // Open source registration state
    const [ossEmail, setOssEmail] = useState('');

    const INVITE_URL = 'https://vantagics.com/invite';

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
            // Use GetEffectiveConfig to include license-provided LLM settings
            const config = await GetEffectiveConfig();
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
        setError(null);
        setOssEmail('');
    };

    const handleSelectFree = () => {
        setMode('free');
        setError(null);
        setFreeEmail('');
    };

    const handleFreeRegister = async () => {
        if (!freeEmail || !freeEmail.includes('@')) {
            setError(t('please_enter_valid_email'));
            return;
        }

        setIsLoading(true);
        setError(null);

        try {
            // Step 1: Request free SN
            const result = await RequestFreeSN(serverURL, freeEmail);
            if (!result.success) {
                if (result.code) {
                    const localizedError = t(`license_error_${result.code}`);
                    setError(localizedError || result.message);
                } else {
                    setError(result.message);
                }
                setIsLoading(false);
                return;
            }

            // Step 2: Activate the free SN
            const activateResult = await ActivateLicense(serverURL, result.sn);
            if (!activateResult.success) {
                setError(activateResult.message);
                setIsLoading(false);
                return;
            }

            // Step 3: Save config
            const config = await GetConfig() as any;
            config.licenseSN = result.sn;
            config.licenseServerURL = serverURL;
            config.licenseEmail = freeEmail;
            await SaveConfig(config);

            // Step 4: Emit event and complete
            EventsEmit('activation-status-changed');
            onComplete();
        } catch (err: any) {
            setError(err.toString());
        } finally {
            setIsLoading(false);
        }
    };

    const handleOpenSourceRegister = async () => {
        if (!ossEmail || !ossEmail.includes('@')) {
            setError(t('please_enter_valid_email'));
            return;
        }

        setIsLoading(true);
        setError(null);

        try {
            // Step 1: Request open source SN
            const result = await RequestOpenSourceSN(serverURL, ossEmail);
            if (!result.success) {
                if (result.code) {
                    const localizedError = t(`license_error_${result.code}`);
                    setError(localizedError || result.message);
                } else {
                    setError(result.message);
                }
                setIsLoading(false);
                return;
            }

            // Step 2: Activate the open source SN
            const activateResult = await ActivateLicense(serverURL, result.sn);
            if (!activateResult.success) {
                setError(activateResult.message);
                setIsLoading(false);
                return;
            }

            // Step 3: Save config
            const config = await GetConfig() as any;
            config.licenseSN = result.sn;
            config.licenseServerURL = serverURL;
            config.licenseEmail = ossEmail;
            await SaveConfig(config);

            // Step 4: Emit event and open settings for LLM configuration
            EventsEmit('activation-status-changed');
            onOpenSettings();
        } catch (err: any) {
            setError(err.toString());
        } finally {
            setIsLoading(false);
        }
    };

    const handleActivate = async (snToUse?: string) => {
        const activateSN = snToUse;

        // If no saved SN provided, we need to request one via email
        if (!activateSN) {
            if (!activationEmail) {
                setError(t('activation_email_required'));
                return;
            }
            const atIndex = activationEmail.indexOf('@');
            if (atIndex < 1 || atIndex >= activationEmail.length - 1 || !activationEmail.substring(atIndex + 1).includes('.')) {
                setError(t('please_enter_valid_email'));
                return;
            }
        }

        if (!serverURL) {
            setError(t('please_fill_server_and_sn'));
            return;
        }

        setIsLoading(true);
        setError(null);
        setIsNotInvitedError(false);
        setCommercialStep('activating');

        try {
            let finalSN = activateSN;

            // Step 1: If no SN provided, request one via email
            if (!finalSN) {
                const snResult = await RequestSN(serverURL, activationEmail);
                if (!snResult.success) {
                    // Handle not_invited error with invite link
                    if (snResult.code === 'not_invited' || snResult.message?.includes('not invited') || snResult.message?.includes('未被邀请')) {
                        setError(t('email_not_invited_text'));
                        setIsNotInvitedError(true);
                    } else if (snResult.code) {
                        const localizedError = t(`license_error_${snResult.code}`);
                        setError(localizedError || snResult.message);
                    } else {
                        setError(snResult.message);
                    }
                    setCommercialStep('check');
                    setIsLoading(false);
                    return;
                }
                finalSN = snResult.sn;
            }

            // Step 2: Activate with the SN
            const result = await ActivateLicense(serverURL, finalSN);
            if (result.success) {
                // Step 3: Save SN and email to config
                const config = await GetConfig() as any;
                config.licenseSN = finalSN;
                config.licenseServerURL = serverURL;
                config.licenseEmail = activationEmail;
                await SaveConfig(config);
                
                setSuccessMessage(t('activation_success'));
                
                // Step 4: Emit event to notify AboutModal to refresh activation status
                EventsEmit('activation-status-changed');
                
                // Step 5: Verify LLM connection (verifyAndComplete handles its own loading state)
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

    const handleBack = () => {
        setMode('select');
        setError(null);
        setIsNotInvitedError(false);
        setSuccessMessage(null);
        setFreeEmail('');
        setOssEmail('');
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
                                {mode === 'select' && (t('welcome_to_vantagics'))}
                                {mode === 'commercial' && (t('commercial_mode'))}
                                {mode === 'opensource' && (t('opensource_mode'))}
                                {mode === 'free' && (t('free_registration'))}
                            </h1>
                            <p className="text-blue-100 text-sm mt-1">
                                {mode === 'select' && (t('select_usage_mode'))}
                                {mode === 'commercial' && (t('activate_with_sn'))}
                                {mode === 'free' && (t('free_registration_subtitle'))}
                                {mode === 'opensource' && (t('oss_registration_subtitle'))}
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
                            
                            {/* Free Mode Card */}
                            <button
                                onClick={handleSelectFree}
                                className="w-full p-5 border-2 border-slate-200 rounded-xl hover:border-purple-500 hover:bg-purple-50/50 transition-all text-left group"
                            >
                                <div className="flex items-start gap-4">
                                    <div className="p-3 bg-purple-100 rounded-xl group-hover:bg-purple-200 transition-colors">
                                        <Gift className="w-6 h-6 text-purple-600" />
                                    </div>
                                    <div className="flex-1">
                                        <h3 className="font-semibold text-slate-800 text-lg">
                                            {t('free_registration')}
                                        </h3>
                                        <p className="text-slate-500 text-sm mt-1">
                                            {t('free_registration_desc')}
                                        </p>
                                        <div className="flex items-center gap-2 mt-3 text-xs text-purple-600">
                                            <Mail className="w-4 h-4" />
                                            <span>{t('email_registration')}</span>
                                            <Gift className="w-4 h-4 ml-2" />
                                            <span>{t('permanent_free')}</span>
                                        </div>
                                    </div>
                                </div>
                            </button>

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
                                        disabled={isLoading || !activationEmail}
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

                    {/* Free Registration Mode */}
                    {mode === 'free' && (
                        <div className="space-y-4">
                            {error && (
                                <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
                                    <div className="flex items-start gap-2">
                                        <AlertCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                        <span className="text-sm text-red-700">{error}</span>
                                    </div>
                                </div>
                            )}

                            <p className="text-sm text-slate-600">
                                {t('free_registration_email_hint')}
                            </p>

                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">
                                    {t('email_address')}
                                </label>
                                <input
                                    type="email"
                                    value={freeEmail}
                                    onChange={(e) => setFreeEmail(e.target.value)}
                                    className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-purple-500 outline-none text-sm"
                                    placeholder="your@email.com"
                                    disabled={isLoading}
                                />
                            </div>

                            <button
                                onClick={handleFreeRegister}
                                disabled={isLoading || !freeEmail}
                                className="w-full py-2.5 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:bg-slate-300 disabled:cursor-not-allowed flex items-center justify-center gap-2 transition-colors"
                            >
                                {isLoading ? (
                                    <>
                                        <Loader2 className="w-4 h-4 animate-spin" />
                                        {t('registering')}
                                    </>
                                ) : (
                                    <>
                                        <Gift className="w-4 h-4" />
                                        {t('free_registration')}
                                    </>
                                )}
                            </button>

                            <p className="text-xs text-slate-400 text-center">
                                {t('free_mode_limitation_note')}
                            </p>
                        </div>
                    )}

                    {/* Open Source Registration Mode */}
                    {mode === 'opensource' && (
                        <div className="space-y-4">
                            {error && (
                                <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
                                    <div className="flex items-start gap-2">
                                        <AlertCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                                        <span className="text-sm text-red-700">{error}</span>
                                    </div>
                                </div>
                            )}

                            <p className="text-sm text-slate-600">
                                {t('oss_registration_email_hint')}
                            </p>

                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">
                                    {t('email_address')}
                                </label>
                                <input
                                    type="email"
                                    value={ossEmail}
                                    onChange={(e) => setOssEmail(e.target.value)}
                                    className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-green-500 outline-none text-sm"
                                    placeholder="your@email.com"
                                    disabled={isLoading}
                                />
                            </div>

                            <button
                                onClick={handleOpenSourceRegister}
                                disabled={isLoading || !ossEmail}
                                className="w-full py-2.5 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:bg-slate-300 disabled:cursor-not-allowed flex items-center justify-center gap-2 transition-colors"
                            >
                                {isLoading ? (
                                    <>
                                        <Loader2 className="w-4 h-4 animate-spin" />
                                        {t('registering')}
                                    </>
                                ) : (
                                    <>
                                        <Code2 className="w-4 h-4" />
                                        {t('oss_register_and_activate')}
                                    </>
                                )}
                            </button>

                            <p className="text-xs text-slate-400 text-center">
                                {t('oss_mode_limitation_note')}
                            </p>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default StartupModeModal;
