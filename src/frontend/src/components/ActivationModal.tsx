import React, { useState, useEffect } from 'react';
import { X, Key, CheckCircle, AlertCircle, Loader2, Mail, ExternalLink } from 'lucide-react';
import { ActivateLicense, GetActivationStatus, DeactivateLicense } from '../../wailsjs/go/main/App';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { useLanguage } from '../i18n';

interface ActivationModalProps {
    isOpen: boolean;
    onClose: () => void;
    onActivated?: () => void;
}

const INVITE_URL = 'https://vantagedata.chat/invite';

const ActivationModal: React.FC<ActivationModalProps> = ({ isOpen, onClose, onActivated }) => {
    const { t } = useLanguage();
    const [serverURL, setServerURL] = useState('https://license.vantagedata.chat');
    const [sn, setSN] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [status, setStatus] = useState<any>(null);
    
    // Request SN state
    const [showRequestForm, setShowRequestForm] = useState(false);
    const [email, setEmail] = useState('');
    const [isRequesting, setIsRequesting] = useState(false);
    const [requestMessage, setRequestMessage] = useState<{ type: 'success' | 'error', text: string, isNotInvited?: boolean } | null>(null);

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
            setShowRequestForm(false);
            setRequestMessage(null);
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
        if (!serverURL || !sn) {
            setError(t('please_fill_server_and_sn') || '请填写服务器地址和序列号');
            return;
        }

        setIsLoading(true);
        setError(null);

        try {
            const result = await ActivateLicense(serverURL, sn);
            if (result.success) {
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
            setSN('');
        } catch (err) {
            console.error('Failed to deactivate:', err);
        }
    };

    const handleRequestSN = async () => {
        if (!serverURL) {
            setRequestMessage({ type: 'error', text: t('please_fill_server') || '请先填写服务器地址' });
            return;
        }
        if (!email || !email.includes('@')) {
            setRequestMessage({ type: 'error', text: t('please_enter_valid_email') || '请输入有效的邮箱地址' });
            return;
        }

        setIsRequesting(true);
        setRequestMessage(null);

        try {
            const response = await fetch(`${serverURL}/request-sn`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email }),
            });
            const result = await response.json();
            
            if (result.success) {
                setSN(result.sn);
                setShowRequestForm(false);
                setRequestMessage({ type: 'success', text: t('sn_request_success') || '序列号申请成功！' });
            } else {
                // Check if it's an "not invited" error
                if (result.code === 'EMAIL_NOT_WHITELISTED' || result.code === 'not_invited' || result.message?.includes('not invited') || result.message?.includes('未被邀请')) {
                    setRequestMessage({ type: 'error', text: t('email_not_invited_text') || '当前未被邀请使用，请点击下方链接获取帮助。', isNotInvited: true });
                } else {
                    // Use code for localization if available
                    setRequestMessage({ type: 'error', text: getLocalizedError(result.code, result.message) });
                }
            }
        } catch (err: any) {
            setRequestMessage({ type: 'error', text: (t('server_connection_failed') || '连接服务器失败') + ': ' + err.toString() });
        } finally {
            setIsRequesting(false);
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[10000]">
            <div className="bg-white rounded-xl shadow-2xl w-[450px] overflow-hidden">
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b border-slate-200 bg-gradient-to-r from-indigo-50 to-purple-50">
                    <div className="flex items-center gap-2">
                        <div className="p-1.5 bg-indigo-100 rounded-lg">
                            <Key className="w-5 h-5 text-indigo-600" />
                        </div>
                        <h2 className="text-lg font-bold text-slate-800">
                            {t('activation_title') || '产品激活'}
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

                            <div>
                                <div className="flex items-center justify-between mb-1">
                                    <label className="block text-sm font-medium text-slate-700">
                                        序列号 (SN)
                                    </label>
                                    <button
                                        onClick={() => setShowRequestForm(!showRequestForm)}
                                        className="text-xs text-indigo-600 hover:text-indigo-800"
                                    >
                                        {showRequestForm ? '返回' : '没有序列号？申请试用'}
                                    </button>
                                </div>
                                
                                {showRequestForm ? (
                                    <div className="space-y-3 p-3 bg-slate-50 rounded-lg border border-slate-200">
                                        <div>
                                            <label className="block text-xs text-slate-600 mb-1">
                                                输入邮箱申请试用序列号
                                            </label>
                                            <div className="flex gap-2">
                                                <input
                                                    type="email"
                                                    value={email}
                                                    onChange={(e) => setEmail(e.target.value)}
                                                    className="flex-1 px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none text-sm"
                                                    placeholder="your@email.com"
                                                />
                                                <button
                                                    onClick={handleRequestSN}
                                                    disabled={isRequesting}
                                                    className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:bg-slate-300 flex items-center gap-1 text-sm whitespace-nowrap"
                                                >
                                                    {isRequesting ? (
                                                        <Loader2 className="w-4 h-4 animate-spin" />
                                                    ) : (
                                                        <Mail className="w-4 h-4" />
                                                    )}
                                                    申请
                                                </button>
                                            </div>
                                            <p className="mt-1 text-xs text-amber-600">
                                                ⚠️ 请输入真实邮箱，以便日后找回序列号
                                            </p>
                                        </div>
                                        {requestMessage && (
                                            <div className={`text-xs ${requestMessage.type === 'success' ? 'text-green-600' : 'text-red-600'}`}>
                                                {requestMessage.text}
                                                {requestMessage.isNotInvited && (
                                                    <button
                                                        onClick={handleOpenInviteLink}
                                                        className="mt-1 flex items-center gap-1 text-blue-600 hover:text-blue-800 hover:underline"
                                                    >
                                                        <ExternalLink className="w-3 h-3" />
                                                        {INVITE_URL}
                                                    </button>
                                                )}
                                            </div>
                                        )}
                                        <p className="text-xs text-slate-400">
                                            * 每个邮箱仅可申请一次，每日限5次申请
                                        </p>
                                    </div>
                                ) : (
                                    <input
                                        type="text"
                                        value={sn}
                                        onChange={(e) => setSN(e.target.value.toUpperCase())}
                                        className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none text-sm font-mono"
                                        placeholder="XXXX-XXXX-XXXX-XXXX"
                                    />
                                )}
                                
                                {requestMessage && !showRequestForm && requestMessage.type === 'success' && (
                                    <p className="mt-1 text-xs text-green-600">{requestMessage.text}</p>
                                )}
                            </div>

                            <button
                                onClick={handleActivate}
                                disabled={isLoading}
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
                                输入序列号激活，或点击上方"申请试用"获取
                            </p>
                        </>
                    )}
                </div>
            </div>
        </div>
    );
};

export default ActivationModal;
