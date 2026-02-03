import React, { useState, useEffect } from 'react';
import { X, BarChart3, CreditCard, Globe } from 'lucide-react';
import { useLanguage } from '../i18n';
import { GetActivationStatus } from '../../wailsjs/go/main/App';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

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
    }>({ activated: false });

    useEffect(() => {
        if (isOpen) {
            GetActivationStatus().then((status) => {
                setActivationStatus({
                    activated: status.activated || false,
                    sn: status.sn || '',
                    expires_at: status.expires_at || '',
                    daily_analysis_limit: status.daily_analysis_limit || 0,
                    daily_analysis_count: status.daily_analysis_count || 0,
                });
            }).catch(() => {
                setActivationStatus({ activated: false });
            });
        }
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
    
    // Determine if trial or official license (trial has daily_analysis_limit > 0)
    const isTrial = activationStatus.activated && activationStatus.daily_analysis_limit !== undefined && activationStatus.daily_analysis_limit > 0;

    const handleSubscribe = () => {
        BrowserOpenURL(PURCHASE_URL);
    };

    if (!isOpen) return null;

    const isChinese = language === '简体中文';

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-sm">
                {/* Header with Logo */}
                <div className="p-5 bg-gradient-to-br from-blue-500 to-purple-600 rounded-t-xl text-white text-center relative">
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
                        {isChinese ? '观数据之界，见商业全貌' : 'See Beyond Data'}
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
                    <div className="p-3 bg-slate-50 rounded-lg space-y-2">
                        <div className="flex justify-between items-center">
                            <span className="text-slate-500 text-xs">{t('working_mode')}</span>
                            <div className="flex items-center gap-1.5">
                                {activationStatus.activated ? (
                                    <>
                                        <span className={`text-xs font-medium px-2 py-0.5 rounded ${isTrial ? 'bg-orange-100 text-orange-700' : 'bg-green-100 text-green-700'}`}>
                                            {isTrial ? (t('trial_license') || '试用版') : (t('official_license') || '正式版')}
                                        </span>
                                    </>
                                ) : (
                                    <span className="text-xs font-medium px-2 py-0.5 rounded bg-slate-100 text-slate-700">
                                        {t('opensource_license')}
                                    </span>
                                )}
                            </div>
                        </div>
                        {activationStatus.activated && activationStatus.sn && (
                            <div className="flex justify-between items-center">
                                <span className="text-slate-500 text-xs">{t('serial_number')}</span>
                                <span className="text-slate-700 font-mono text-xs">{activationStatus.sn}</span>
                            </div>
                        )}
                        {activationStatus.activated && activationStatus.expires_at && (
                            <div className="flex justify-between items-center">
                                <span className="text-slate-500 text-xs">{t('expires')}</span>
                                <span className={`text-xs ${isExpired ? 'text-red-600 font-medium' : daysUntilExpiration !== null && daysUntilExpiration <= 31 ? 'text-orange-600' : 'text-slate-700'}`}>
                                    {activationStatus.expires_at}
                                    {isExpired && ` (${t('expired') || '已过期'})`}
                                    {!isExpired && daysUntilExpiration !== null && daysUntilExpiration <= 31 && ` (${daysUntilExpiration}${t('days_remaining') || '天后到期'})`}
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
                                    : 'bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white'
                            }`}
                        >
                            <CreditCard className="w-4 h-4" />
                            {isExpired 
                                ? (t('renew_subscription') || '续订授权') 
                                : (t('subscribe_now') || '立即订阅')}
                        </button>
                    )}

                    {/* Daily Analysis Usage */}
                    {activationStatus.activated && activationStatus.daily_analysis_limit !== undefined && activationStatus.daily_analysis_limit > 0 && (
                        <div className="p-3 bg-blue-50 rounded-lg">
                            <div className="flex items-center justify-between mb-2">
                                <div className="flex items-center gap-1.5">
                                    <BarChart3 className="w-3.5 h-3.5 text-blue-600" />
                                    <span className="text-xs font-medium text-slate-700">{t('daily_analysis_usage') || '今日分析'}</span>
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
        </div>
    );
};

export default AboutModal;
