/**
 * AnalysisStatusIndicator - 分析状态指示器组件
 * 
 * 可复用的状态指示器组件，支持三种显示模式：
 * - inline: 内联模式，仅显示转圈动画
 * - compact: 紧凑模式，显示转圈动画和简短消息
 * - full: 完整模式，显示转圈动画、进度条、消息和已用时间
 * 
 * Requirements: 1.1, 1.2, 1.4
 */

import React, { useMemo, useState } from 'react';
import { useSessionStatus } from '../hooks/useSessionStatus';
import { AlertCircle, Loader2, XCircle, AlertTriangle } from 'lucide-react';
import { useLanguage } from '../i18n';

/**
 * AnalysisStatusIndicatorProps - 组件属性接口
 */
export interface AnalysisStatusIndicatorProps {
    /** 会话ID */
    threadId: string;
    /** 显示模式：inline（内联）、compact（紧凑）、full（完整） */
    variant: 'inline' | 'compact' | 'full';
    /** 是否显示消息（仅对 compact 和 full 模式有效） */
    showMessage?: boolean;
    /** 是否显示进度条（仅对 full 模式有效） */
    showProgress?: boolean;
    /** 是否显示取消按钮（仅对 full 模式有效） */
    showCancelButton?: boolean;
    /** 取消分析的回调函数 */
    onCancel?: () => void;
    /** 重试分析的回调函数 */
    onRetry?: () => void;
    /** 关闭错误提示的回调函数 */
    onDismissError?: () => void;
    /** 自定义 CSS 类名 */
    className?: string;
}

/**
 * 格式化已用时间为可读字符串
 * @param ms 毫秒数
 * @param t 翻译函数
 * @returns 格式化的时间字符串
 */
function formatElapsedTime(ms: number, t: (key: string) => string): string {
    if (ms < 1000) {
        return t('just_started');
    }
    
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    
    if (minutes > 0) {
        return `${minutes}${t('minutes')}${remainingSeconds}${t('seconds')}`;
    }
    return `${seconds}${t('seconds')}`;
}

/**
 * 获取阶段的显示名称
 * @param stage 阶段标识
 * @param t 翻译函数
 * @returns 阶段名称
 */
function getStageDisplayName(stage: string, t: (key: string) => string): string {
    const stageKeys: Record<string, string> = {
        'waiting': 'stage_waiting',
        'initializing': 'stage_initializing',
        'schema': 'stage_schema',
        'query': 'stage_query',
        'analysis': 'stage_analyzing',
        'visualization': 'stage_visualization',
        'generating': 'stage_generating',
        'exporting': 'stage_exporting',
        'searching': 'stage_searching',
        'complete': 'stage_complete',
        'error': 'stage_error'
    };
    return t(stageKeys[stage] || stage);
}

/**
 * Spinner 组件 - 转圈加载动画
 */
const Spinner: React.FC<{ size?: 'sm' | 'md' | 'lg'; className?: string; label?: string }> = ({ 
    size = 'md', 
    className = '',
    label
}) => {
    const sizeClasses = {
        sm: 'w-3 h-3 border-2',
        md: 'w-4 h-4 border-2',
        lg: 'w-6 h-6 border-3'
    };
    
    return (
        <div 
            className={`${sizeClasses[size]} border-blue-200 border-t-blue-600 rounded-full animate-spin ${className}`}
            role="status"
            aria-label={label}
        />
    );
};

/**
 * ProgressBar 组件 - 进度条
 */
const ProgressBar: React.FC<{ progress: number; className?: string }> = ({ 
    progress, 
    className = '' 
}) => {
    // 确保进度值在 0-100 之间
    const clampedProgress = Math.max(0, Math.min(100, progress));
    
    return (
        <div className={`w-full bg-slate-200 rounded-full h-1.5 overflow-hidden ${className}`}>
            <div 
                className="bg-blue-600 h-full rounded-full transition-all duration-300 ease-out"
                style={{ width: `${clampedProgress}%` }}
                role="progressbar"
                aria-valuenow={clampedProgress}
                aria-valuemin={0}
                aria-valuemax={100}
            />
        </div>
    );
};

/**
 * InlineIndicator - 内联模式指示器
 * 仅显示一个小型转圈动画
 */
const InlineIndicator: React.FC<{ className?: string; t: (key: string) => string }> = ({ className = '', t }) => (
    <span className={`inline-flex items-center ${className}`}>
        <Spinner size="sm" label={t('loading')} />
    </span>
);

/**
 * CompactIndicator - 紧凑模式指示器
 * 显示转圈动画和简短消息
 */
const CompactIndicator: React.FC<{
    message?: string;
    showMessage: boolean;
    className?: string;
    t: (key: string) => string;
}> = ({ message, showMessage, className = '', t }) => (
    <div className={`inline-flex items-center gap-2 ${className}`}>
        <Spinner size="sm" label={t('loading')} />
        {showMessage && message && (
            <span className="text-sm text-slate-600 truncate max-w-[200px]">
                {message}
            </span>
        )}
    </div>
);

/**
 * CancelConfirmDialog - 取消确认对话框
 * 内置的轻量级确认对话框，用于取消分析前的用户确认
 */
const CancelConfirmDialog: React.FC<{
    isOpen: boolean;
    onClose: () => void;
    onConfirm: () => void;
    t: (key: string) => string;
}> = ({ isOpen, onClose, onConfirm, t }) => {
    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="bg-white rounded-xl shadow-2xl p-6 w-[380px] transform transition-all animate-in zoom-in-95 duration-200">
                <div className="flex items-start gap-4 mb-4">
                    <div className="bg-amber-100 p-2 rounded-full">
                        <AlertTriangle className="w-6 h-6 text-amber-600" />
                    </div>
                    <div className="flex-1">
                        <h3 className="text-lg font-bold text-slate-900 mb-1">{t('cancel_analysis_dialog_title')}</h3>
                        <p className="text-sm text-slate-600">
                            {t('cancel_analysis_dialog_message')}
                        </p>
                        <p className="text-xs text-slate-400 mt-2">
                            {t('cancel_analysis_dialog_note')}
                        </p>
                    </div>
                </div>

                <div className="flex justify-end gap-3 mt-6">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                        {t('continue_analysis')}
                    </button>
                    <button
                        onClick={onConfirm}
                        className="px-4 py-2 text-sm font-medium text-white bg-amber-600 hover:bg-amber-700 rounded-lg shadow-sm transition-colors"
                    >
                        {t('confirm_cancel')}
                    </button>
                </div>
            </div>
        </div>
    );
};

/**
 * FullIndicator - 完整模式指示器
 * 显示转圈动画、进度条、消息和已用时间
 * 包含 AI 助手图标，与聊天消息气泡样式保持一致
 * 支持取消按钮（需要用户确认）
 */
const FullIndicator: React.FC<{
    progress?: {
        stage: string;
        progress: number;
        message: string;
        step: number;
        total: number;
    };
    elapsedTime: number;
    showMessage: boolean;
    showProgress: boolean;
    showCancelButton: boolean;
    onCancel?: () => void;
    className?: string;
    t: (key: string) => string;
}> = ({ progress, elapsedTime, showMessage, showProgress, showCancelButton, onCancel, className = '', t }) => {
    const [showConfirmDialog, setShowConfirmDialog] = useState(false);
    
    const displayStage = progress?.stage ? getStageDisplayName(progress.stage, t) : '';
    const progressPercent = progress?.progress ?? 0;
    const stepInfo = progress?.step && progress?.total 
        ? `${progress.step}/${progress.total}` 
        : '';
    
    // Resolve the message: if it's an i18n key (starts with "progress."), translate it; otherwise use as-is
    const resolvedMessage = useMemo(() => {
        if (!progress?.message) return t('processing');
        const msg = progress.message;
        // If it looks like an i18n key, try to translate
        if (msg.startsWith('progress.')) {
            const translated = t(msg);
            // If translation returns the key itself, it wasn't found — use a fallback
            return translated === msg ? msg.replace('progress.', '').replace(/_/g, ' ') : translated;
        }
        return msg;
    }, [progress?.message, t]);
    
    const handleCancelClick = () => {
        setShowConfirmDialog(true);
    };
    
    const handleConfirmCancel = () => {
        setShowConfirmDialog(false);
        onCancel?.();
    };
    
    const handleCloseDialog = () => {
        setShowConfirmDialog(false);
    };
    
    return (
        <>
            <div className={`flex items-start gap-4 ${className}`}>
                {/* AI 助手图标 - 与 MessageBubble 中的 AI 消息样式一致 */}
                <div className="flex-shrink-0 w-9 h-9 rounded-xl flex items-center justify-center shadow-sm bg-gradient-to-br from-blue-500 to-indigo-600 text-white">
                    <Loader2 className="w-5 h-5 animate-spin" />
                </div>
                
                {/* 状态内容区域 */}
                <div className="flex-1 flex flex-col gap-2 p-3 bg-white border border-slate-100 rounded-2xl rounded-tl-none shadow-sm">
                    {/* 顶部：阶段 + 已用时间 + 取消按钮 */}
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                            <Loader2 className="w-4 h-4 text-blue-600 animate-spin" />
                            {displayStage && (
                                <span className="text-sm font-medium text-blue-700">
                                    {displayStage}
                                </span>
                            )}
                            {stepInfo && (
                                <span className="text-xs text-slate-400 font-mono">
                                    [{stepInfo}]
                                </span>
                            )}
                        </div>
                        <div className="flex items-center gap-2">
                            <span className="text-xs text-slate-500">
                                {formatElapsedTime(elapsedTime, t)}
                            </span>
                            {showCancelButton && onCancel && (
                                <button
                                    onClick={handleCancelClick}
                                    className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-slate-500 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
                                    title={t('cancel')}
                                >
                                    <XCircle className="w-3.5 h-3.5" />
                                    <span>{t('cancel')}</span>
                                </button>
                            )}
                        </div>
                    </div>
                    
                    {/* 进度条 */}
                    {showProgress && progressPercent > 0 && (
                        <ProgressBar progress={progressPercent} />
                    )}
                    
                    {/* 消息 */}
                    {showMessage && resolvedMessage && (
                        <p className="text-sm text-slate-600">
                            {resolvedMessage}
                        </p>
                    )}
                </div>
            </div>
            
            {/* 取消确认对话框 */}
            <CancelConfirmDialog
                isOpen={showConfirmDialog}
                onClose={handleCloseDialog}
                onConfirm={handleConfirmCancel}
                t={t}
            />
        </>
    );
};

/**
 * ErrorIndicator - 错误状态指示器
 * 显示分析过程中发生的错误，支持不同的显示模式
 */
const ErrorIndicator: React.FC<{
    error: { code: string; message: string };
    variant: 'inline' | 'compact' | 'full';
    className?: string;
    onRetry?: () => void;
    onDismiss?: () => void;
    t: (key: string) => string;
}> = ({ error, variant, className = '', onRetry, onDismiss, t }) => {
    // 根据错误代码获取图标和颜色
    const getErrorStyle = (code: string) => {
        switch (code) {
            case 'ANALYSIS_TIMEOUT':
                return { icon: AlertTriangle, bgColor: 'bg-amber-50', borderColor: 'border-amber-200', textColor: 'text-amber-700', iconColor: 'text-amber-500' };
            case 'NETWORK_ERROR':
                return { icon: AlertCircle, bgColor: 'bg-orange-50', borderColor: 'border-orange-200', textColor: 'text-orange-700', iconColor: 'text-orange-500' };
            case 'DATABASE_ERROR':
                return { icon: AlertCircle, bgColor: 'bg-blue-50', borderColor: 'border-blue-200', textColor: 'text-blue-700', iconColor: 'text-blue-500' };
            case 'PYTHON_ERROR':
                return { icon: AlertCircle, bgColor: 'bg-purple-50', borderColor: 'border-purple-200', textColor: 'text-purple-700', iconColor: 'text-purple-500' };
            case 'LLM_ERROR':
                return { icon: AlertCircle, bgColor: 'bg-indigo-50', borderColor: 'border-indigo-200', textColor: 'text-indigo-700', iconColor: 'text-indigo-500' };
            default:
                return { icon: AlertCircle, bgColor: 'bg-red-50', borderColor: 'border-red-200', textColor: 'text-red-700', iconColor: 'text-red-500' };
        }
    };
    
    const style = getErrorStyle(error.code);
    const IconComponent = style.icon;
    
    if (variant === 'inline') {
        return (
            <span className={`inline-flex items-center ${className}`} title={error.message}>
                <IconComponent className={`w-3 h-3 ${style.iconColor}`} />
            </span>
        );
    }
    
    if (variant === 'compact') {
        return (
            <div className={`inline-flex items-center gap-2 ${className}`}>
                <IconComponent className={`w-4 h-4 ${style.iconColor} flex-shrink-0`} />
                <span className={`text-sm ${style.textColor} truncate max-w-[200px]`}>
                    {error.message.startsWith('progress.') ? t(error.message) : error.message}
                </span>
            </div>
        );
    }
    
    // full variant - 完整的错误显示，包含重试和关闭按钮
    return (
        <div className={`flex items-start gap-4 ${className}`}>
            {/* 错误图标 - 与 AI 助手图标样式一致 */}
            <div className={`flex-shrink-0 w-9 h-9 rounded-xl flex items-center justify-center shadow-sm ${style.bgColor}`}>
                <IconComponent className={`w-5 h-5 ${style.iconColor}`} />
            </div>
            
            {/* 错误内容区域 */}
            <div className={`flex-1 flex flex-col gap-2 p-3 ${style.bgColor} border ${style.borderColor} rounded-2xl rounded-tl-none shadow-sm`}>
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <IconComponent className={`w-4 h-4 ${style.iconColor} flex-shrink-0`} />
                        <span className={`text-sm font-medium ${style.textColor}`}>
                            {t('analysis_failed')}
                        </span>
                    </div>
                    {onDismiss && (
                        <button
                            onClick={onDismiss}
                            className="text-slate-400 hover:text-slate-600 transition-colors"
                            title={t('close')}
                        >
                            <XCircle className="w-4 h-4" />
                        </button>
                    )}
                </div>
                
                <p className={`text-sm ${style.textColor}`}>
                    {error.message.startsWith('progress.') ? t(error.message) : error.message}
                </p>
                
                {error.code && error.code !== 'ANALYSIS_ERROR' && (
                    <p className="text-xs text-slate-400">
                        {t('error')}: {error.code}
                    </p>
                )}
                
                {onRetry && (
                    <div className="flex justify-end mt-1">
                        <button
                            onClick={onRetry}
                            className={`px-3 py-1 text-xs font-medium ${style.textColor} hover:bg-white/50 rounded-md transition-colors`}
                        >
                            {t('retry')}
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
};

/**
 * AnalysisStatusIndicator - 主组件
 * 
 * 根据会话状态和显示模式渲染相应的指示器
 * 
 * Requirements: 1.1, 1.2, 1.4
 */
export const AnalysisStatusIndicator: React.FC<AnalysisStatusIndicatorProps> = ({
    threadId,
    variant,
    showMessage = true,
    showProgress = true,
    showCancelButton = true,
    onCancel,
    onRetry,
    onDismissError,
    className = ''
}) => {
    // 使用国际化
    const { t } = useLanguage();
    
    // 使用 useSessionStatus hook 获取会话状态
    const { isLoading, progress, error, elapsedTime } = useSessionStatus(threadId);
    
    // 计算显示消息（解析 i18n key）
    const displayMessage = useMemo(() => {
        if (progress?.message) {
            const msg = progress.message;
            // If it looks like an i18n key, translate it
            if (msg.startsWith('progress.')) {
                const translated = t(msg);
                return translated === msg ? msg.replace('progress.', '').replace(/_/g, ' ') : translated;
            }
            return msg;
        }
        if (progress?.stage) {
            return getStageDisplayName(progress.stage, t);
        }
        return t('analyzing');
    }, [progress, t]);
    
    // 如果有错误，显示错误状态
    if (error) {
        return (
            <ErrorIndicator 
                error={error} 
                variant={variant} 
                className={className}
                onRetry={onRetry}
                onDismiss={onDismissError}
                t={t}
            />
        );
    }
    
    // 如果不在加载状态，不显示任何内容
    if (!isLoading) {
        return null;
    }
    
    // 根据 variant 渲染不同的指示器
    switch (variant) {
        case 'inline':
            return <InlineIndicator className={className} t={t} />;
            
        case 'compact':
            return (
                <CompactIndicator 
                    message={displayMessage}
                    showMessage={showMessage}
                    className={className}
                    t={t}
                />
            );
            
        case 'full':
            return (
                <FullIndicator 
                    progress={progress}
                    elapsedTime={elapsedTime}
                    showMessage={showMessage}
                    showProgress={showProgress}
                    showCancelButton={showCancelButton}
                    onCancel={onCancel}
                    className={className}
                    t={t}
                />
            );
            
        default:
            return null;
    }
};

export default AnalysisStatusIndicator;
