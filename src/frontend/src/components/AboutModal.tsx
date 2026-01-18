import React from 'react';
import { X, Info } from 'lucide-react';
import { useLanguage } from '../i18n';

interface AboutModalProps {
    isOpen: boolean;
    onClose: () => void;
}

const AboutModal: React.FC<AboutModalProps> = ({ isOpen, onClose }) => {
    const { t, language } = useLanguage();

    if (!isOpen) return null;

    const isChinese = language === '简体中文';

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-md">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-slate-200">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-blue-100 rounded-lg">
                            <Info className="w-5 h-5 text-blue-600" />
                        </div>
                        <h2 className="text-xl font-semibold text-slate-800">
                            {t('about')}
                        </h2>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-2 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                        <X className="w-5 h-5 text-slate-500" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-8">
                    {/* Logo/Icon */}
                    <div className="flex justify-center mb-6">
                        <div className="w-24 h-24 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl shadow-lg flex items-center justify-center">
                            <svg
                                className="w-16 h-16 text-white"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                strokeWidth="2"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                            >
                                {/* Data visualization icon */}
                                <path d="M3 3v18h18" />
                                <path d="M18 17V9" />
                                <path d="M13 17V5" />
                                <path d="M8 17v-3" />
                            </svg>
                        </div>
                    </div>

                    {/* App Name */}
                    <div className="text-center mb-4">
                        <h1 className="text-3xl font-bold text-slate-800 mb-2">
                            {isChinese ? '观界' : 'VantageData'}
                        </h1>
                        {isChinese && (
                            <p className="text-sm text-slate-500">VantageData</p>
                        )}
                    </div>

                    {/* Slogan */}
                    <div className="text-center mb-6">
                        <p className="text-slate-600 font-medium">
                            {isChinese 
                                ? '观数据之界，见商业全貌。' 
                                : 'See Beyond Data. Master Your Vantage.'}
                        </p>
                    </div>

                    {/* Divider */}
                    <div className="border-t border-slate-200 my-6"></div>

                    {/* Version Info */}
                    <div className="space-y-3 text-sm">
                        <div className="flex justify-between items-center">
                            <span className="text-slate-500">{t('version')}:</span>
                            <span className="text-slate-800 font-medium">1.0.0</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-slate-500">{t('build_date')}:</span>
                            <span className="text-slate-800 font-medium">2026-01-18</span>
                        </div>
                    </div>

                    {/* Divider */}
                    <div className="border-t border-slate-200 my-6"></div>

                    {/* Copyright */}
                    <div className="text-center text-xs text-slate-500">
                        <p>© 2026 VantageData. All rights reserved.</p>
                    </div>
                </div>

                {/* Footer */}
                <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-center rounded-b-xl">
                    <button
                        onClick={onClose}
                        className="px-6 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors"
                    >
                        {t('close')}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default AboutModal;
