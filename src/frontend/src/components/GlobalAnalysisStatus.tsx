/**
 * GlobalAnalysisStatus - 全局分析状态指示器组件
 * 
 * 在仪表盘标题区域显示全局分析状态汇总，包括：
 * - 当有一个或多个会话正在分析时显示转圈动画
 * - 当所有分析完成时隐藏指示器
 * - 悬停时显示活跃分析数量的 tooltip
 * 
 * Requirements: 3.1, 3.2, 3.3
 */

import React, { useState, useRef, useEffect } from 'react';
import { useLoadingState } from '../hooks/useLoadingState';
import { Loader2 } from 'lucide-react';

/**
 * GlobalAnalysisStatusProps - 组件属性接口
 */
export interface GlobalAnalysisStatusProps {
    /** 自定义 CSS 类名 */
    className?: string;
}

/**
 * Tooltip 组件 - 悬停提示
 */
const Tooltip: React.FC<{
    content: string;
    visible: boolean;
    targetRef: React.RefObject<HTMLDivElement>;
}> = ({ content, visible, targetRef }) => {
    const [position, setPosition] = useState({ top: 0, left: 0 });

    useEffect(() => {
        if (visible && targetRef.current) {
            const rect = targetRef.current.getBoundingClientRect();
            setPosition({
                top: rect.bottom + 8,
                left: rect.left + rect.width / 2
            });
        }
    }, [visible, targetRef]);

    if (!visible) return null;

    return (
        <div
            className="fixed z-50 px-3 py-2 text-sm text-white bg-slate-800 rounded-lg shadow-lg transform -translate-x-1/2 whitespace-nowrap"
            style={{ top: position.top, left: position.left }}
            role="tooltip"
        >
            {content}
            {/* Tooltip arrow */}
            <div 
                className="absolute -top-1 left-1/2 transform -translate-x-1/2 w-2 h-2 bg-slate-800 rotate-45"
            />
        </div>
    );
};

/**
 * GlobalAnalysisStatus - 主组件
 * 
 * 显示全局加载状态和活跃分析数量
 * 
 * Requirements: 3.1, 3.2, 3.3
 */
export const GlobalAnalysisStatus: React.FC<GlobalAnalysisStatusProps> = ({
    className = ''
}) => {
    const { loadingCount, isAnyLoading } = useLoadingState();
    const [showTooltip, setShowTooltip] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);

    // 如果没有正在进行的分析，不显示任何内容
    // Requirements: 3.2 - 当所有分析完成时隐藏指示器
    if (!isAnyLoading) {
        return null;
    }

    // 生成 tooltip 内容
    const tooltipContent = loadingCount === 1
        ? '1 个分析正在进行中'
        : `${loadingCount} 个分析正在进行中`;

    return (
        <>
            <div
                ref={containerRef}
                className={`inline-flex items-center gap-2 px-3 py-1.5 bg-blue-50 border border-blue-200 rounded-full cursor-default transition-all duration-200 hover:bg-blue-100 ${className}`}
                onMouseEnter={() => setShowTooltip(true)}
                onMouseLeave={() => setShowTooltip(false)}
                role="status"
                aria-label={tooltipContent}
                data-testid="global-analysis-status"
            >
                {/* 转圈动画 - Requirements: 3.1 */}
                <Loader2 
                    className="w-4 h-4 text-blue-600 animate-spin" 
                    aria-hidden="true"
                />
                
                {/* 活跃分析数量 */}
                <span className="text-sm font-medium text-blue-700">
                    {loadingCount}
                </span>
            </div>

            {/* Tooltip - Requirements: 3.3 */}
            <Tooltip
                content={tooltipContent}
                visible={showTooltip}
                targetRef={containerRef}
            />
        </>
    );
};

export default GlobalAnalysisStatus;
