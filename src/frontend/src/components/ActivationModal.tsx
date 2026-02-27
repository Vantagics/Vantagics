import React, { useState, useEffect } from 'react';
import { X, Key, CheckCircle, AlertCircle, Loader2, ExternalLink } from 'lucide-react';
import { ActivateLicense, GetActivationStatus, DeactivateLicense, GetConfig, SaveConfig, RequestSN } from '../../wailsjs/go/main/App';
import { BrowserOpenURL, EventsEmit } from '../../wailsjs/runtime/runtime';
import { useLanguage } from '../i18n';

interface ActivationModalProps {
    isOpen: boolean;
    onClose: () => void;
    onActivated?: () => void;
    hideServerURL?: boolean; // Hide server URL input (use default)
}

const INVITE_URL = 'https://vantagics.com/invite';

const ActivationModal: React.FC<ActivationModalProps> = ({ isOpen, onClose, onActivated, hideServerURL = false }) => {
    const { t } = useLanguage();
    const [serverURL, setServerURL] = useState('https://license.vantagics.com');
    const [activationEmail, setActivationEmail] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [isNotInvitedError, setIsNotInvitedError] = useState(false);
    const [status, setStatus] = useState<any>(null);

    // Helper function to get localized error message from server response code
    const getLocalizedError = (code: string | undefined, fallbackMessage: string): string => {
        if (code) {
            const translationKey = `license_error_${code}`;
            const translated = t(translationKey);
            // If translation exists and is different from the key, use it
            if (translated && translated !== translationKey) {
                return translated;
            }
        }
        return fallbackMessage;
    };

    const handleOpenInviteLink = () => {
        BrowserOpenURL(INVITE_URL);
    };

    useEffect(() => {
        if (isOpen) {
            loadStatus();
        }
    }, [isOpen]);

    const loadStatus = async () => {
        try {
            const result = await GetActivationStatus();
            setStatus(result);
        } catch (err) {
            console.error('Failed to get activation status:', err);
        }
    };

    const handleActivate = async () => {
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
        setIsNotInvitedError(false);

        try {
            // Step 1: Request SN via email
            const snResult = await RequestSN(serverURL, activationEmail);
            if (!snResult.success) {
                // Handle not_invited error with invite link
                if (snResult.code === 'not_invited' || snResult.message?.includes('not invited') || snResult.message?.includes('未被邀请')) {
                    setError(t('email_not_invited_text'));
                    setIsNotInvitedError(true);
                } else if (snResult.code) {
                    setError(getLocalizedError(snResult.code, snResult.message));
                } else {
                    setError(snResult.message);
                }
                setIsLoading(false);
                return;
            }
            const finalSN = snResult.sn;

            // Step 2: Activate with the SN
            const result = await ActivateLicense(serverURL, finalSN);
            if (result.success) {
                // Step 3: Save SN and email to config
                try {
                    const cfg = await GetConfig();
                    (cfg as any).licenseSN = finalSN;
                    (cfg as any).licenseServerURL = serverURL;
                    (cfg as any).licenseEmail = activationEmail;
                    await SaveConfig(cfg);
                } catch (e) {
                    console.warn('Failed to save license config:', e);
                }

                // Step 4: Emit event to notify other components
                EventsEmit('activation-status-changed');

                await loadStatus();
                onActivated?.();
            } else {
                // Use code for localization if available
                setError(getLocalizedError(result.code, result.message));
            }
        } catch (err: any) {
            setError(err.toString());
        } finally {
            setIsLoading(false);
        }
    };

    const handleDeactivate = async () => {
        try {
            await DeactivateLicense();
            await loadStatus();
        } catch (err) {
            console.error('Failed to deactivate:', err);
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[10000]">
            <div className="bg-white dark:bg-[#252526] rounded-xl shadow-2xl w-[450px] overflow-hidden">
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b border-slate-200 dark:border-[#3c3c3c] bg-gradient-to-r from-[#f0f4f8] to-[#eaeff5] dark:from-[#1e1e2e] dark:to-[#2a1e2e]">
                    <div className="flex items-center gap-2">
                        <div className="p-1.5 bg-[#dce5ef] rounded-lg">
                            <Key className="w-5 h-5 text-[#5b7a9d]" />
                        </div>
                        <h2 className="text-lg font-bold text-slate-800">
                            {t('activation_title')}
                        </h2>
                    </div>
                    <button onClick={onClose} className="p-1.5 hover:bg-white/50 rounded-lg">
                        <X className="w-4 h-4 text-slate-500" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 space-y-4">
                    {/* Current Status */}
                    {status?.activated ? (
                        <div className="p-4 bg-green-50 border border-green-200 rounded-lg">
                            <div className="flex items-center gap-2 mb-2">
                                <CheckCircle className="w-5 h-5 text-green-600" />
                                <span className="font-medium text-green-800">已激活</span>
                            </div>
                            <div className="text-sm text-green-700 space-y-1">
                                <p>到期时间: {status.expires_at}</p>
                                {status.has_llm && <p>✓ LLM 服务 ({status.llm_type})</p>}
                                {status.has_search && <p>✓ 搜索服务 ({status.search_type})</p>}
                            </div>
                            <button
                                onClick={handleDeactivate}
                                className="mt-3 px-4 py-1.5 bg-red-100 text-red-700 rounded-lg text-sm hover:bg-red-200"
                            >
                                取消激活
                            </button>
                        </div>
                    ) : (
                        <>
                            {error && (
                                <div className="p-3 bg-red-50 border border-red-200 rounded-lg flex items-start gap-2">
                                    <AlertCircle className="w-4 h-4 text-red-500 mt-0.5" />
                                    <span className="text-sm text-red-700">{error}</span>
                                </div>
                            )}
                            {isNotInvitedError && (
                                <button
                                    onClick={handleOpenInviteLink}
                                    className="mt-2 ml-6 flex items-center gap-1 text-sm text-blue-600 hover:text-blue-800 hover:underline"
                                >
                                    <ExternalLink className="w-3.5 h-3.5" />
                                    {INVITE_URL}
                                </button>
                            )}

                            {!hideServerURL && (
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">
                                        授权服务器地址
                                    </label>
                                    <input
                                        type="text"
                                        value={serverURL}
                                        onChange={(e) => setServerURL(e.target.value)}
                                        className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none text-sm"
                                        placeholder="http://server:7799"
                                    />
                                </div>
                            )}

                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">
                                    {t('activation_email_label')}
                                </label>
                                <input
                                    type="email"
                                    value={activationEmail}
                                    onChange={(e) => setActivationEmail(e.target.value)}
                                    className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none text-sm"
                                    placeholder={t('activation_email_placeholder')}
                                />
                            </div>

                            <button
                                onClick={handleActivate}
                                disabled={isLoading || !activationEmail}
                                className="w-full py-2.5 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:bg-slate-300 flex items-center justify-center gap-2"
                            >
                                {isLoading ? (
                                    <>
                                        <Loader2 className="w-4 h-4 animate-spin" />
                                        正在激活...
                                    </>
                                ) : (
                                    '激活'
                                )}
                            </button>

                            <p className="text-xs text-slate-500 text-center">
                                {t('activation_email_hint') !== 'activation_email_hint' ? t('activation_email_hint') : '输入邮箱即可自动激活商业授权'}
                            </p>
                        </>
                    )}
                </div>
            </div>
        </div>
    );
};

export default ActivationModal;