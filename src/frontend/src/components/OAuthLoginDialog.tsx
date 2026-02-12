import React, { useState } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2 } from 'lucide-react';
import { MarketplaceLogin } from '../../wailsjs/go/main/App';

interface OAuthLoginDialogProps {
    onSuccess: (token: string) => void;
    onClose: () => void;
}

type OAuthProvider = 'google' | 'apple' | 'facebook' | 'amazon';

const providers: { id: OAuthProvider; label: string; color: string }[] = [
    { id: 'google', label: 'Google', color: 'bg-white dark:bg-[#1e1e1e] border border-slate-300 dark:border-[#3e3e42] text-slate-800 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30]' },
    { id: 'apple', label: 'Apple', color: 'bg-black text-white hover:bg-gray-800 dark:bg-white dark:text-black dark:hover:bg-gray-200' },
    { id: 'facebook', label: 'Facebook', color: 'bg-[#1877F2] text-white hover:bg-[#166FE5]' },
    { id: 'amazon', label: 'Amazon', color: 'bg-[#FF9900] text-black hover:bg-[#E88B00]' },
];

const OAuthLoginDialog: React.FC<OAuthLoginDialogProps> = ({ onSuccess, onClose }) => {
    const { t } = useLanguage();
    const [loading, setLoading] = useState<OAuthProvider | null>(null);
    const [error, setError] = useState<string | null>(null);

    const handleLogin = async (provider: OAuthProvider) => {
        setLoading(provider);
        setError(null);
        try {
            await MarketplaceLogin(provider);
            onSuccess(provider);
        } catch (err: any) {
            setError(err?.message || err?.toString() || t('oauth_login_error'));
        } finally {
            setLoading(null);
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape' && !loading) {
            onClose();
        }
    };

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={loading ? undefined : onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[380px] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] p-6"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKeyDown}
                tabIndex={-1}
            >
                <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4] mb-1">
                    {t('oauth_login_title')}
                </h3>
                <p className="text-sm text-slate-500 dark:text-[#8e8e8e] mb-5">
                    {t('oauth_login_desc')}
                </p>

                <div className="flex flex-col gap-3">
                    {providers.map(({ id, label, color }) => (
                        <button
                            key={id}
                            onClick={() => handleLogin(id)}
                            disabled={loading !== null}
                            className={`w-full px-4 py-2.5 text-sm font-medium rounded-lg transition-colors flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed ${color}`}
                        >
                            {loading === id && <Loader2 className="w-4 h-4 animate-spin" />}
                            {loading === id ? t('oauth_logging_in') : label}
                        </button>
                    ))}
                </div>

                {error && (
                    <p className="mt-4 text-sm text-red-500 text-center">{error}</p>
                )}

                <div className="flex justify-end mt-5">
                    <button
                        onClick={onClose}
                        disabled={loading !== null}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {t('cancel')}
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default OAuthLoginDialog;
