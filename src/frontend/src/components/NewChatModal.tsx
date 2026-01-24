import React, { useState } from 'react';
import { useLanguage } from '../i18n';
import { CheckSessionNameExists } from '../../wailsjs/go/main/App';
import { MessageCircle } from 'lucide-react';

interface NewChatModalProps {
    isOpen: boolean;
    dataSourceId: string;
    onClose: () => void;
    onSubmit: (sessionName: string) => void;
    onStartFreeChat?: () => void; // New: callback for free chat
}

const NewChatModal: React.FC<NewChatModalProps> = ({ isOpen, dataSourceId, onClose, onSubmit, onStartFreeChat }) => {
    const { t } = useLanguage();
    const [sessionName, setSessionName] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [isValidating, setIsValidating] = useState(false);

    if (!isOpen) return null;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        const trimmedName = sessionName.trim();
        if (!trimmedName) return;

        setIsValidating(true);
        setError(null);
        try {
            const exists = await CheckSessionNameExists(dataSourceId, trimmedName);
            if (exists) {
                setError(t('session_already_exists').replace('{0}', trimmedName));
                setIsValidating(false);
                return;
            }
            onSubmit(trimmedName);
            setSessionName('');
            onClose();
        } catch (err) {
            setError(String(err));
        } finally {
            setIsValidating(false);
        }
    };

    const handleFreeChat = () => {
        setSessionName('');
        onClose();
        if (onStartFreeChat) {
            onStartFreeChat();
        }
    };

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[400px] rounded-xl shadow-2xl overflow-hidden text-slate-900 p-6">
                <h3 className="text-lg font-bold text-slate-800 mb-4">{t('start_new_analysis')}</h3>
                <form onSubmit={handleSubmit}>
                    <div className="mb-4">
                        {error && (
                            <div className="mb-3 p-2 bg-red-50 border border-red-100 text-red-600 text-xs rounded-lg">
                                {error}
                            </div>
                        )}
                        <label htmlFor="sessionName" className="block text-sm font-medium text-slate-700 mb-1">{t('session_name')}</label>
                        <input
                            id="sessionName"
                            type="text"
                            value={sessionName}
                            onChange={(e) => {
                                setSessionName(e.target.value);
                                if (error) setError(null);
                            }}
                            className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                            placeholder={t('session_name_placeholder')}
                            autoFocus
                            disabled={isValidating}
                        />
                    </div>
                    <div className="flex justify-between items-center">
                        {/* Left side - Free Chat button */}
                        <button 
                            type="button"
                            onClick={handleFreeChat}
                            className="px-3 py-2 text-sm font-medium text-purple-600 hover:bg-purple-50 rounded-md transition-colors flex items-center gap-1.5"
                            disabled={isValidating}
                        >
                            <MessageCircle className="w-4 h-4" />
                            {t('free_chat')}
                        </button>
                        
                        {/* Right side - Cancel and Start buttons */}
                        <div className="flex gap-3">
                            <button 
                                type="button"
                                onClick={onClose}
                                className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-md transition-colors"
                                disabled={isValidating}
                            >
                                {t('cancel')}
                            </button>
                            <button 
                                type="submit"
                                disabled={!sessionName.trim() || isValidating}
                                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                            >
                                {isValidating ? (
                                    <>
                                        <span className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
                                        {t('validating')}
                                    </>
                                ) : (
                                    t('start_chat')
                                )}
                            </button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default NewChatModal;
