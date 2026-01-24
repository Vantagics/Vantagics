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
    /** 自定义 CSS 类名 */
    className?: string;
}

/**
 * 格式化已用时间为可读字符串
 * @param ms 毫秒数
 * @returns 格式化的时间字符串
 */
function formatElapsedTime(ms: number): string {
    if (ms < 1000) {
        return '刚刚开始';
    }
    
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    
    if (minutes > 0) {
        return `${minutes}分${remainingSeconds}秒`;
    }
    return `${seconds}秒`;
}

/**
 * 获取阶段的中文显示名称
 * @param stage 阶段标识
 * @returns 中文阶段名称
 */
function getStageDisplayName(stage: string): string {
    const stageNames: Record<string, string> = {
        'waiting': '等待中',
        'initializing': '初始化中',
        'analyzing': '正在分析',
        'generating': '正在生成',
        'complete': '已完成',
        'error': '发生错误'
    };
    return stageNames[stage] || stage;
}

/**
 * Spinner 组件 - 转圈加载动画
 */
const Spinner: React.FC<{ size?: 'sm' | 'md' | 'lg'; className?: string }> = ({ 
    size = 'md', 
    className = '' 
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
            aria-label="加载中"
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
const InlineIndicator: React.FC<{ className?: string }> = ({ className = '' }) => (
    <span className={`inline-flex items-center ${className}`}>
        <Spinner size="sm" />
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
}> = ({ message, showMessage, className = '' }) => (
    <div className={`inline-flex items-center gap-2 ${className}`}>
        <Spinner size="sm" />
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
}> = ({ isOpen, onClose, onConfirm }) => {
    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="bg-white rounded-xl shadow-2xl p-6 w-[380px] transform transition-all animate-in zoom-in-95 duration-200">
                <div className="flex items-start gap-4 mb-4">
                    <div className="bg-amber-100 p-2 rounded-full">
                        <AlertTriangle className="w-6 h-6 text-amber-600" />
                    </div>
                    <div className="flex-1">
                        <h3 className="text-lg font-bold text-slate-900 mb-1">取消分析</h3>
                        <p className="text-sm text-slate-600">
                            确定要取消当前的分析任务吗？
                        </p>
                        <p className="text-xs text-slate-400 mt-2">
                            已经生成的结果将会丢失。
                        </p>
                    </div>
                </div>

                <div className="flex justify-end gap-3 mt-6">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                        继续分析
                    </button>
                    <button
                        onClick={onConfirm}
                        className="px-4 py-2 text-sm font-medium text-white bg-amber-600 hover:bg-amber-700 rounded-lg shadow-sm transition-colors"
                    >
                        确认取消
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
}> = ({ progress, elapsedTime, showMessage, showProgress, showCancelButton, onCancel, className = '' }) => {
    const [showConfirmDialog, setShowConfirmDialog] = useState(false);
    
    const displayMessage = progress?.message || '正在处理...';
    const displayStage = progress?.stage ? getStageDisplayName(progress.stage) : '';
    const progressPercent = progress?.progress ?? 0;
    const stepInfo = progress?.step && progress?.total 
        ? `(${progress.step}/${progress.total})` 
        : '';
    
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
                                    {displayStage} {stepInfo}
                                </span>
                            )}
                        </div>
                        <div className="flex items-center gap-2">
                            <span className="text-xs text-slate-500">
                                {formatElapsedTime(elapsedTime)}
                            </span>
                            {showCancelButton && onCancel && (
                                <button
                                    onClick={handleCancelClick}
                                    className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-slate-500 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
                                    title="取消分析"
                                >
                                    <XCircle className="w-3.5 h-3.5" />
                                    <span>取消</span>
                                </button>
                            )}
                        </div>
                    </div>
                    
                    {/* 进度条 */}
                    {showProgress && progressPercent > 0 && (
                        <ProgressBar progress={progressPercent} />
                    )}
                    
                    {/* 消息 */}
                    {showMessage && displayMessage && (
                        <p className="text-sm text-slate-600">
                            {displayMessage}
                        </p>
                    )}
                </div>
            </div>
            
            {/* 取消确认对话框 */}
            <CancelConfirmDialog
                isOpen={showConfirmDialog}
                onClose={handleCloseDialog}
                onConfirm={handleConfirmCancel}
            />
        </>
    );
};

/**
 * ErrorIndicator - 错误状态指示器
 */
const ErrorIndicator: React.FC<{
    error: { code: string; message: string };
    variant: 'inline' | 'compact' | 'full';
    className?: string;
}> = ({ error, variant, className = '' }) => {
    if (variant === 'inline') {
        return (
            <span className={`inline-flex items-center ${className}`} title={error.message}>
                <AlertCircle className="w-3 h-3 text-red-500" />
            </span>
        );
    }
    
    if (variant === 'compact') {
        return (
            <div className={`inline-flex items-center gap-2 ${className}`}>
                <AlertCircle className="w-4 h-4 text-red-500 flex-shrink-0" />
                <span className="text-sm text-red-600 truncate max-w-[200px]">
                    {error.message}
                </span>
            </div>
        );
    }
    
    // full variant
    return (
        <div className={`flex flex-col gap-2 p-3 bg-red-50 border border-red-200 rounded-lg ${className}`}>
            <div className="flex items-center gap-2">
                <AlertCircle className="w-4 h-4 text-red-500 flex-shrink-0" />
                <span className="text-sm font-medium text-red-700">
                    分析失败
                </span>
            </div>
            <p className="text-sm text-red-600">
                {error.message}
            </p>
            {error.code && error.code !== 'ANALYSIS_ERROR' && (
                <p className="text-xs text-red-400">
                    错误代码: {error.code}
                </p>
            )}
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
    className = ''
}) => {
    // 使用 useSessionStatus hook 获取会话状态
    const { isLoading, progress, error, elapsedTime } = useSessionStatus(threadId);
    
    // 计算显示消息
    const displayMessage = useMemo(() => {
        if (progress?.message) {
            return progress.message;
        }
        if (progress?.stage) {
            return getStageDisplayName(progress.stage);
        }
        return '正在分析...';
    }, [progress]);
    
    // 如果有错误，显示错误状态
    if (error) {
        return (
            <ErrorIndicator 
                error={error} 
                variant={variant} 
                className={className} 
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
            return <InlineIndicator className={className} />;
            
        case 'compact':
            return (
                <CompactIndicator 
                    message={displayMessage}
                    showMessage={showMessage}
                    className={className}
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
                />
            );
            
        default:
            return null;
    }
};

export default AnalysisStatusIndicator;
