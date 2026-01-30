import React from 'react';
import { useLanguage } from '../i18n';
import { ShoppingBag, Upload, Store, X } from 'lucide-react';

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

    const platforms = [
        {
            id: 'shopify',
            name: 'Shopify',
            icon: '🛍️',
            color: 'bg-green-50 hover:bg-green-100 border-green-200',
            textColor: 'text-green-700'
        },
        {
            id: 'bigcommerce',
            name: 'BigCommerce',
            icon: '🏪',
            color: 'bg-blue-50 hover:bg-blue-100 border-blue-200',
            textColor: 'text-blue-700'
        },
        {
            id: 'ebay',
            name: 'eBay',
            icon: '🛒',
            color: 'bg-yellow-50 hover:bg-yellow-100 border-yellow-200',
            textColor: 'text-yellow-700'
        },
        {
            id: 'etsy',
            name: 'Etsy',
            icon: '🎨',
            color: 'bg-orange-50 hover:bg-orange-100 border-orange-200',
            textColor: 'text-orange-700'
        }
    ];

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[520px] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900">
                {/* Header */}
                <div className="p-6 border-b border-slate-200 flex items-center justify-between">
                    <div>
                        <h2 className="text-xl font-bold text-slate-800">
                            {t('onboarding_welcome') || '👋 欢迎使用 RapidBI'}
                        </h2>
                        <p className="text-sm text-slate-500 mt-1">
                            {t('onboarding_subtitle') || '让我们开始导入您的数据'}
                        </p>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-2 hover:bg-slate-100 rounded-full transition-colors"
                    >
                        <X className="w-5 h-5 text-slate-400" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 space-y-6">
                    {/* Question */}
                    <div className="text-center">
                        <Store className="w-12 h-12 mx-auto text-blue-500 mb-3" />
                        <h3 className="text-lg font-semibold text-slate-700">
                            {t('onboarding_question') || '您是以下电商平台的店主吗？'}
                        </h3>
                        <p className="text-sm text-slate-500 mt-1">
                            {t('onboarding_question_desc') || '选择您的平台，我们将帮助您快速导入店铺数据'}
                        </p>
                    </div>

                    {/* Platform Grid */}
                    <div className="grid grid-cols-2 gap-3">
                        {platforms.map((platform) => (
                            <button
                                key={platform.id}
                                onClick={() => onSelectPlatform(platform.id)}
                                className={`p-4 rounded-lg border-2 ${platform.color} transition-all hover:scale-[1.02] hover:shadow-md flex items-center gap-3`}
                            >
                                <span className="text-2xl">{platform.icon}</span>
                                <span className={`font-medium ${platform.textColor}`}>
                                    {platform.name}
                                </span>
                            </button>
                        ))}
                    </div>

                    {/* Divider */}
                    <div className="flex items-center gap-4">
                        <div className="flex-1 h-px bg-slate-200"></div>
                        <span className="text-sm text-slate-400">
                            {t('onboarding_or') || '或者'}
                        </span>
                        <div className="flex-1 h-px bg-slate-200"></div>
                    </div>

                    {/* Self Import Option */}
                    <button
                        onClick={onSelectSelfImport}
                        className="w-full p-4 rounded-lg border-2 border-slate-200 bg-slate-50 hover:bg-slate-100 transition-all hover:shadow-md flex items-center justify-center gap-3"
                    >
                        <Upload className="w-5 h-5 text-slate-600" />
                        <span className="font-medium text-slate-700">
                            {t('onboarding_self_import') || '自主导入数据'}
                        </span>
                    </button>

                    <p className="text-xs text-slate-400 text-center">
                        {t('onboarding_self_import_desc') || '支持 Excel、CSV、JSON 文件，或连接 MySQL、PostgreSQL 数据库'}
                    </p>
                </div>

                {/* Footer */}
                <div className="p-4 border-t border-slate-100 bg-slate-50">
                    <p className="text-xs text-slate-400 text-center">
                        {t('onboarding_skip_hint') || '您可以随时通过侧边栏添加更多数据源'}
                    </p>
                </div>
            </div>
        </div>
    );
};

export default OnboardingWizard;
