import React from 'react';
import { useLanguage } from '../i18n';
import { Upload, Store, X, Briefcase } from 'lucide-react';

interface OnboardingWizardProps {
    isOpen: boolean;
    onClose: () => void;
    onSelectPlatform: (platform: string) => void;
    onSelectSelfImport: () => void;
}

const OnboardingWizard: React.FC<OnboardingWizardProps> = ({
    isOpen,
    onClose,
    onSelectPlatform,
    onSelectSelfImport
}) => {
    const { t } = useLanguage();

    if (!isOpen) return null;

    // E-commerce platforms
    const ecommercePlatforms = [
        { id: 'shopify', name: 'Shopify', icon: '🛍️', color: 'bg-green-50 hover:bg-green-100 border-green-200', textColor: 'text-green-700' },
        { id: 'bigcommerce', name: 'BigCommerce', icon: '🏪', color: 'bg-blue-50 hover:bg-blue-100 border-blue-200', textColor: 'text-blue-700' },
        { id: 'ebay', name: 'eBay', icon: '🛒', color: 'bg-yellow-50 hover:bg-yellow-100 border-yellow-200', textColor: 'text-yellow-700' },
        { id: 'etsy', name: 'Etsy', icon: '🎨', color: 'bg-orange-50 hover:bg-orange-100 border-orange-200', textColor: 'text-orange-700' }
    ];

    // Project management tools
    const projectTools = [
        { id: 'jira', name: 'Jira', icon: '📋', color: 'bg-indigo-50 hover:bg-indigo-100 border-indigo-200', textColor: 'text-indigo-700' }
    ];

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[480px] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900">
                {/* Header */}
                <div className="px-5 py-4 border-b border-slate-200 flex items-center justify-between">
                    <div>
                        <h2 className="text-lg font-bold text-slate-800">
                            {t('onboarding_welcome') || '👋 欢迎使用 VantageData'}
                        </h2>
                        <p className="text-xs text-slate-500 mt-0.5">
                            {t('onboarding_subtitle') || '让我们开始导入您的数据'}
                        </p>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-1.5 hover:bg-slate-100 rounded-full transition-colors"
                    >
                        <X className="w-4 h-4 text-slate-400" />
                    </button>
                </div>

                {/* Content */}
                <div className="px-5 py-4 space-y-4">
                    {/* E-commerce Section */}
                    <div>
                        <div className="flex items-center gap-1.5 mb-2">
                            <Store className="w-4 h-4 text-blue-500" />
                            <h3 className="text-xs font-semibold text-slate-700">
                                {t('onboarding_ecommerce_title') || '电商平台'}
                            </h3>
                            <span className="text-xs text-slate-400">- {t('onboarding_ecommerce_desc') || '导入店铺数据'}</span>
                        </div>
                        <div className="grid grid-cols-4 gap-2">
                            {ecommercePlatforms.map((platform) => (
                                <button
                                    key={platform.id}
                                    onClick={() => onSelectPlatform(platform.id)}
                                    className={`p-2 rounded-lg border ${platform.color} transition-all hover:scale-[1.02] hover:shadow-sm flex flex-col items-center gap-1`}
                                >
                                    <span className="text-lg">{platform.icon}</span>
                                    <span className={`font-medium text-xs ${platform.textColor}`}>
                                        {platform.name}
                                    </span>
                                </button>
                            ))}
                        </div>
                    </div>

                    {/* Project Management Section */}
                    <div>
                        <div className="flex items-center gap-1.5 mb-2">
                            <Briefcase className="w-4 h-4 text-indigo-500" />
                            <h3 className="text-xs font-semibold text-slate-700">
                                {t('onboarding_project_title') || '项目管理'}
                            </h3>
                            <span className="text-xs text-slate-400">- {t('onboarding_project_desc') || '导入项目数据'}</span>
                        </div>
                        <div className="grid grid-cols-4 gap-2">
                            {projectTools.map((platform) => (
                                <button
                                    key={platform.id}
                                    onClick={() => onSelectPlatform(platform.id)}
                                    className={`p-2 rounded-lg border ${platform.color} transition-all hover:scale-[1.02] hover:shadow-sm flex flex-col items-center gap-1`}
                                >
                                    <span className="text-lg">{platform.icon}</span>
                                    <span className={`font-medium text-xs ${platform.textColor}`}>
                                        {platform.name}
                                    </span>
                                </button>
                            ))}
                        </div>
                    </div>

                    {/* Divider */}
                    <div className="flex items-center gap-3">
                        <div className="flex-1 h-px bg-slate-200"></div>
                        <span className="text-xs text-slate-400">{t('onboarding_or') || '或者'}</span>
                        <div className="flex-1 h-px bg-slate-200"></div>
                    </div>

                    {/* Self Import Option */}
                    <button
                        onClick={onSelectSelfImport}
                        className="w-full p-3 rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 transition-all hover:shadow-sm flex items-center justify-center gap-2"
                    >
                        <Upload className="w-4 h-4 text-slate-600" />
                        <span className="font-medium text-sm text-slate-700">
                            {t('onboarding_self_import') || '自主导入数据'}
                        </span>
                        <span className="text-xs text-slate-400">
                            ({t('onboarding_self_import_desc') || 'Excel、CSV、JSON、MySQL'})
                        </span>
                    </button>
                </div>

                {/* Footer */}
                <div className="px-5 py-3 border-t border-slate-100 bg-slate-50">
                    <p className="text-xs text-slate-400 text-center">
                        {t('onboarding_skip_hint') || '您可以随时通过侧边栏添加更多数据源'}
                    </p>
                </div>
            </div>
        </div>
    );
};

export default OnboardingWizard;
