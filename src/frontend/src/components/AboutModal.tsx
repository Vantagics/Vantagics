import React, { useState, useEffect } from 'react';
import { X, BarChart3, CreditCard, Globe } from 'lucide-react';
import { useLanguage } from '../i18n';
import { GetActivationStatus, DeactivateLicense } from '../../wailsjs/go/main/App';
import { BrowserOpenURL, EventsEmit } from '../../wailsjs/runtime/runtime';
import { createLogger } from '../utils/systemLog';

const logger = createLogger('AboutModal');

interface AboutModalProps {
    isOpen: boolean;
    onClose: () => void;
}

const PURCHASE_URL = 'https://vantagedata.chat/purchase';
const WEBSITE_URL = 'https://vantagedata.chat';

const AboutModal: React.FC<AboutModalProps> = ({ isOpen, onClose }) => {
    const { t, language } = useLanguage();
    const [activationStatus, setActivationStatus] = useState<{
        activated: boolean;
        sn?: string;
        expires_at?: string;
        daily_analysis_limit?: number;
        daily_analysis_count?: number;
        total_credits?: number;
        used_credits?: number;
        credits_mode?: boolean;
        email?: string;
    }>({ activated: false });

    // State for license mode switch feature
    const [showConfirmDialog, setShowConfirmDialog] = useState(false);
    const [confirmAction, setConfirmAction] = useState<'toCommercial' | 'toOpenSource' | null>(null);
    const [isDeactivating, setIsDeactivating] = useState(false);
    const [deactivateError, setDeactivateError] = useState<string | null>(null);

    useEffect(() => {
        if (isOpen) {
            GetActivationStatus().then((status) => {
                setActivationStatus({
                    activated: status.activated || false,
                    sn: status.sn || '',
                    expires_at: status.expires_at || '',
                    daily_analysis_limit: status.daily_analysis_limit || 0,
                    daily_analysis_count: status.daily_analysis_count || 0,
                    total_credits: status.total_credits || 0,
                    used_credits: status.used_credits || 0,
                    credits_mode: status.credits_mode || false,
                    email: status.email || '',
                });
            }).catch(() => {
                setActivationStatus({ activated: false });
            });
        }
    }, [isOpen]);

    // Listen for activation success event to refresh status
    useEffect(() => {
        if (!isOpen) return;
        
        const refreshStatus = () => {
            GetActivationStatus().then((status) => {
                setActivationStatus({
                    activated: status.activated || false,
                    sn: status.sn || '',
                    expires_at: status.expires_at || '',
                    daily_analysis_limit: status.daily_analysis_limit || 0,
                    daily_analysis_count: status.daily_analysis_count || 0,
                    total_credits: status.total_credits || 0,
                    used_credits: status.used_credits || 0,
                    credits_mode: status.credits_mode || false,
                    email: status.email || '',
                });
            }).catch(() => {
                setActivationStatus({ activated: false });
            });
        };
        
        // Import EventsOn dynamically to avoid circular dependency
        import('../../wailsjs/runtime/runtime').then(({ EventsOn }) => {
            const unsubscribe = EventsOn('activation-status-changed', refreshStatus);
            return () => {
                if (unsubscribe) unsubscribe();
            };
        });
    }, [isOpen]);

    // Calculate days until expiration
    const getDaysUntilExpiration = (): number | null => {
        if (!activationStatus.expires_at) return null;
        const expiresDate = new Date(activationStatus.expires_at);
        const today = new Date();
        const diffTime = expiresDate.getTime() - today.getTime();
        const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
        return diffDays;
    };

    const daysUntilExpiration = getDaysUntilExpiration();
    const showSubscribeButton = activationStatus.activated && daysUntilExpiration !== null && daysUntilExpiration <= 31;
    const isExpired = daysUntilExpiration !== null && daysUntilExpiration <= 0;
    
    // Determine if trial or official license
    // Trial = has limits: daily_analysis_limit > 0 (次数有限制) or credits_mode with total_credits > 0 (credits有限制)
    // credits_mode with total_credits == 0 means unlimited, not trial
    const isTrial = activationStatus.activated && (
        (!activationStatus.credits_mode && activationStatus.daily_analysis_limit !== undefined && activationStatus.daily_analysis_limit > 0) ||
        (activationStatus.credits_mode === true && activationStatus.total_credits !== undefined && activationStatus.total_credits > 0)
    );

    const handleSubscribe = () => {
        BrowserOpenURL(PURCHASE_URL);
    };

    // Handle mode switch button click
    const handleSwitchClick = () => {
        if (activationStatus.activated) {
            setConfirmAction('toOpenSource');
        } else {
            setConfirmAction('toCommercial');
        }
        setShowConfirmDialog(true);
    };

    // Handle confirm action
    const handleConfirm = async () => {
        if (confirmAction === 'toCommercial') {
            // Close the confirm dialog and AboutModal, then emit event to open StartupModeModal
            setShowConfirmDialog(false);
            onClose();
            // Delay event emission to ensure AboutModal is fully closed
            setTimeout(() => {
                EventsEmit('open-startup-mode-modal');
            }, 100);
        } else if (confirmAction === 'toOpenSource') {
            // Deactivate the license
            setIsDeactivating(true);
            setDeactivateError(null);
            try {
                await DeactivateLicense();
                // Refresh activation status
                const status = await GetActivationStatus();
                setActivationStatus({
                    activated: status.activated || false,
                    sn: status.sn || '',
                    expires_at: status.expires_at || '',
                    daily_analysis_limit: status.daily_analysis_limit || 0,
                    daily_analysis_count: status.daily_analysis_count || 0,
                    total_credits: status.total_credits || 0,
                    used_credits: status.used_credits || 0,
                    credits_mode: status.credits_mode || false,
                    email: status.email || '',
                });
                setShowConfirmDialog(false);
                setConfirmAction(null);
                
                // Close AboutModal and open Settings with LLM tab
                onClose();
                // Emit event to open settings with LLM tab
                EventsEmit('open-settings', { tab: 'llm' });
            } catch (error: any) {
                // Show error message
                const errorMsg = error?.message || error?.toString() || t('deactivate_failed');
                setDeactivateError(errorMsg);
            } finally {
                setIsDeactivating(false);
            }
        }
    };

    // Handle cancel action
    const handleCancel = () => {
        setShowConfirmDialog(false);
        setConfirmAction(null);
        setDeactivateError(null);
    };

    if (!isOpen) return null;

    const isChinese = language === '简体中文';

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-white dark:bg-[#252526] rounded-xl shadow-2xl w-full max-w-sm">
                {/* Header with Logo */}
                <div className="p-5 bg-gradient-to-br from-[#5b7a9d] to-[#7b9bb8] rounded-t-xl text-white text-center relative">
                    <button
                        onClick={onClose}
                        className="absolute right-3 top-3 p-1 hover:bg-white/20 rounded-lg transition-colors"
                    >
                        <X className="w-4 h-4" />
                    </button>
                    <div className="w-14 h-14 bg-white/20 rounded-xl mx-auto mb-3 flex items-center justify-center">
                        <svg className="w-9 h-9" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M3 3v18h18" />
                            <path d="M18 17V9" />
                            <path d="M13 17V5" />
                            <path d="M8 17v-3" />
                        </svg>
                    </div>
                    <h1 className="text-xl font-bold">{isChinese ? '观界' : 'VantageData'}</h1>
                    <p className="text-white/80 text-xs mt-1">
                        {isChinese ? '观数据之界，见商业全貌' : 'See Beyond Data. Master Your Vantage.'}
                    </p>
                </div>

                {/* Content */}
                <div className="p-4 space-y-3 text-sm">
                    {/* Version Info */}
                    <div className="grid grid-cols-2 gap-2 text-xs">
                        <div className="flex justify-between">
                            <span className="text-slate-500">{t('version')}</span>
                            <span className="text-slate-700 font-medium">1.0.0</span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-slate-500">{t('build_date')}</span>
                            <span className="text-slate-700 font-medium">2026-01-18</span>
                        </div>
                    </div>

                    {/* License Info */}
                    <div className="p-3 bg-slate-50 dark:bg-[#2d2d30] rounded-lg space-y-2">
                        <div className="flex justify-between items-center">
                            <span className="text-slate-500 text-xs">{t('working_mode')}</span>
                            <div className="flex items-center gap-2">
                                {/* Current Mode Badge */}
                                {activationStatus.activated ? (
                                    <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                                        isTrial 
                                            ? 'bg-gradient-to-r from-orange-400 to-amber-500 text-white shadow-sm' 
                                            : 'bg-gradient-to-r from-emerald-500 to-green-600 text-white shadow-sm'
                                    }`}>
                                        {isTrial ? (t('trial_license')) : (t('official_license'))}
                                    </span>
                                ) : (
                                    <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-gradient-to-r from-slate-500 to-slate-600 text-white shadow-sm">
                                        {t('opensource_license')}
                                    </span>
                                )}
                                {/* Mode Switch Button - Outlined style for distinction */}
                                <button
                                    onClick={handleSwitchClick}
                                    disabled={isDeactivating}
                                    className={`text-xs px-2.5 py-1 rounded border transition-all disabled:opacity-50
                                        ${activationStatus.activated 
                                            ? 'border-slate-300 text-slate-500 hover:border-slate-400 hover:text-slate-600 hover:bg-slate-50' 
                                            : 'border-blue-300 text-blue-500 hover:border-blue-400 hover:text-blue-600 hover:bg-blue-50'
                                        }`}
                                >
                                    {activationStatus.activated 
                                        ? t('switch_to_opensource') 
                                        : t('switch_to_commercial')}
                                </button>
                            </div>
                        </div>
                        {activationStatus.activated && activationStatus.sn && (
                            <div className="flex justify-between items-center">
                                <span className="text-slate-500 text-xs">{t('serial_number')}</span>
                                <span className="text-slate-700 font-mono text-xs">{activationStatus.sn}</span>
                            </div>
                        )}
                        {activationStatus.activated && activationStatus.email && (
                            <div className="flex justify-between items-center">
                                <span className="text-slate-500 text-xs">{t('license_email')}</span>
                                <span className="text-slate-700 text-xs">{activationStatus.email}</span>
                            </div>
                        )}
                        {activationStatus.activated && activationStatus.expires_at && (
                            <div className="flex justify-between items-center">
                                <span className="text-slate-500 text-xs">{t('expires')}</span>
                                <span className={`text-xs ${isExpired ? 'text-red-600 font-medium' : daysUntilExpiration !== null && daysUntilExpiration <= 31 ? 'text-orange-600' : 'text-slate-700'}`}>
                                    {activationStatus.expires_at}
                                    {isExpired && ` (${t('expired')})`}
                                    {!isExpired && daysUntilExpiration !== null && daysUntilExpiration <= 31 && ` (${daysUntilExpiration}${t('days_remaining')})`}
                                </span>
                            </div>
                        )}
                    </div>

                    {/* Subscribe Button - Show when less than 31 days until expiration */}
                    {showSubscribeButton && (
                        <button
                            onClick={handleSubscribe}
                            className={`w-full flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                                isExpired 
                                    ? 'bg-red-600 hover:bg-red-700 text-white' 
                                    : 'bg-gradient-to-r from-[#5b7a9d] to-[#7b9bb8] hover:from-[#456a8a] hover:to-[#6b8db5] text-white'
                            }`}
                        >
                            <CreditCard className="w-4 h-4" />
                            {isExpired 
                                ? (t('renew_subscription')) 
                                : (t('subscribe_now'))}
                        </button>
                    )}

                    {/* Credits Usage - shown when credits_mode is true */}
                    {activationStatus.activated && activationStatus.credits_mode === true && (
                        <div className="p-3 bg-blue-50 rounded-lg">
                            <div className="flex items-center justify-between mb-2">
                                <div className="flex items-center gap-1.5">
                                    <BarChart3 className="w-3.5 h-3.5 text-blue-600" />
                                    <span className="text-xs font-medium text-slate-700">{t('credits_usage')}</span>
                                </div>
                                <span className="text-xs text-slate-600">
                                    {activationStatus.total_credits === 0 ? t('unlimited') : `${activationStatus.used_credits || 0} / ${activationStatus.total_credits}`}
                                </span>
                            </div>
                            {activationStatus.total_credits !== undefined && activationStatus.total_credits > 0 && (
                            <div className="w-full bg-blue-200 rounded-full h-1.5">
                                <div 
                                    className={`h-1.5 rounded-full transition-all ${
                                        (activationStatus.used_credits || 0) >= activationStatus.total_credits ? 'bg-red-500' : 'bg-blue-500'
                                    }`}
                                    style={{ width: `${Math.min(100, ((activationStatus.used_credits || 0) / activationStatus.total_credits) * 100)}%` }}
                                />
                            </div>
                            )}
                        </div>
                    )}

                    {/* Daily Analysis Usage - shown when NOT in credits mode */}
                    {activationStatus.activated && activationStatus.credits_mode !== true && activationStatus.daily_analysis_limit !== undefined && activationStatus.daily_analysis_limit > 0 && (
                        <div className="p-3 bg-blue-50 rounded-lg">
                            <div className="flex items-center justify-between mb-2">
                                <div className="flex items-center gap-1.5">
                                    <BarChart3 className="w-3.5 h-3.5 text-blue-600" />
                                    <span className="text-xs font-medium text-slate-700">{t('daily_analysis_usage')}</span>
                                </div>
                                <span className="text-xs text-slate-600">
                                    {activationStatus.daily_analysis_count || 0} / {activationStatus.daily_analysis_limit}
                                </span>
                            </div>
                            <div className="w-full bg-blue-200 rounded-full h-1.5">
                                <div 
                                    className={`h-1.5 rounded-full transition-all ${
                                        (activationStatus.daily_analysis_count || 0) >= activationStatus.daily_analysis_limit ? 'bg-red-500' : 'bg-blue-500'
                                    }`}
                                    style={{ width: `${Math.min(100, ((activationStatus.daily_analysis_count || 0) / activationStatus.daily_analysis_limit) * 100)}%` }}
                                />
                            </div>
                        </div>
                    )}

                    {/* Copyright */}
                    <p className="text-center text-xs text-slate-400 pt-1">
                        © 2026 VantageData
                    </p>

                    {/* Website Link */}
                    <button
                        onClick={() => BrowserOpenURL(WEBSITE_URL)}
                        className="w-full flex items-center justify-center gap-1.5 text-xs text-blue-600 hover:text-blue-700 hover:underline transition-colors"
                    >
                        <Globe className="w-3.5 h-3.5" />
                        vantagedata.chat
                    </button>
                </div>
            </div>

            {/* Confirmation Dialog */}
            {showConfirmDialog && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60]">
                    <div className="bg-white dark:bg-[#252526] rounded-lg shadow-xl w-full max-w-sm p-4">
                        <h3 className="text-lg font-semibold text-slate-800 dark:text-[#d4d4d4] mb-2">
                            {confirmAction === 'toCommercial' 
                                ? t('confirm_switch_to_commercial') 
                                : t('confirm_switch_to_opensource')}
                        </h3>
                        <p className="text-sm text-slate-600 mb-4">
                            {confirmAction === 'toCommercial' 
                                ? t('confirm_switch_to_commercial_desc') 
                                : t('confirm_switch_to_opensource_desc')}
                        </p>
                        {deactivateError && (
                            <p className="text-sm text-red-600 mb-4">{deactivateError}</p>
                        )}
                        <div className="flex justify-end gap-2">
                            <button
                                onClick={handleCancel}
                                disabled={isDeactivating}
                                className="px-4 py-2 text-sm text-slate-600 hover:bg-slate-100 
                                           rounded-lg transition-colors disabled:opacity-50"
                            >
                                {t('cancel')}
                            </button>
                            <button
                                onClick={handleConfirm}
                                disabled={isDeactivating}
                                className={`px-4 py-2 text-sm text-white rounded-lg transition-colors
                                    ${confirmAction === 'toOpenSource' 
                                        ? 'bg-orange-500 hover:bg-orange-600' 
                                        : 'bg-blue-500 hover:bg-blue-600'}
                                    disabled:opacity-50`}
                            >
                                {isDeactivating ? '...' : t('confirm')}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default AboutModal;
