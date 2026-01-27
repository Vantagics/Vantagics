import React, { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import ReactECharts from 'echarts-for-react';

interface ChartModalProps {
    isOpen: boolean;
    options: any;
    onClose: () => void;
}

const ChartModal: React.FC<ChartModalProps> = ({ isOpen, options, onClose }) => {
    const [isVisible, setIsVisible] = useState(false);

    // 验证输入的options
    if (!options || typeof options !== 'object') {
        console.error('ChartModal: Invalid options provided', options);
        return null;
    }

    // Add Chinese font support to ECharts options
    const enhancedOptions = React.useMemo(() => {
        try {
            const fontConfig = {
                textStyle: {
                    fontFamily: 'Microsoft YaHei, SimHei, Arial Unicode MS, sans-serif'
                }
            };

            return {
                ...fontConfig,
                ...options,
                title: {
                    ...fontConfig.textStyle,
                    ...options.title
                },
                legend: {
                    textStyle: fontConfig.textStyle,
                    ...options.legend
                },
                xAxis: Array.isArray(options.xAxis)
                    ? options.xAxis.map((axis: any) => ({
                        ...axis,
                        axisLabel: { ...fontConfig.textStyle, ...axis.axisLabel },
                        nameTextStyle: { ...fontConfig.textStyle, ...axis.nameTextStyle }
                    }))
                    : options.xAxis ? {
                        ...options.xAxis,
                        axisLabel: { ...fontConfig.textStyle, ...options.xAxis?.axisLabel },
                        nameTextStyle: { ...fontConfig.textStyle, ...options.xAxis?.nameTextStyle }
                    } : undefined,
                yAxis: Array.isArray(options.yAxis)
                    ? options.yAxis.map((axis: any) => ({
                        ...axis,
                        axisLabel: { ...fontConfig.textStyle, ...axis.axisLabel },
                        nameTextStyle: { ...fontConfig.textStyle, ...axis.nameTextStyle }
                    }))
                    : options.yAxis ? {
                        ...options.yAxis,
                        axisLabel: { ...fontConfig.textStyle, ...options.yAxis?.axisLabel },
                        nameTextStyle: { ...fontConfig.textStyle, ...options.yAxis?.nameTextStyle }
                    } : undefined,
                tooltip: {
                    ...options.tooltip,
                    textStyle: fontConfig.textStyle
                },
                // 确保series存在
                series: options.series || []
            };
        } catch (error) {
            console.error('ChartModal: Error processing options', error);
            return {
                title: { text: '图表配置错误' },
                series: []
            };
        }
    }, [options]);

    useEffect(() => {
        if (isOpen) {
            setIsVisible(true);
        } else {
            const timer = setTimeout(() => setIsVisible(false), 200);
            return () => clearTimeout(timer);
        }
    }, [isOpen]);

    if (!isVisible) return null;

    return (
        <div 
            className={`fixed inset-0 z-[200] flex items-center justify-center transition-opacity duration-200 ${isOpen ? 'opacity-100' : 'opacity-0'}`}
            onClick={onClose}
        >
            {/* Backdrop */}
            <div className="absolute inset-0 bg-black/90 backdrop-blur-sm" />

            {/* Controls */}
            <div className="absolute top-4 right-4 flex gap-2 z-[210]">
                <button 
                    onClick={onClose}
                    className="p-2 bg-white/10 hover:bg-red-500/80 rounded-full text-white transition-colors"
                >
                    <X className="w-6 h-6" />
                </button>
            </div>

            {/* Chart Container */}
            <div 
                className="relative z-[205] w-[95vw] h-[90vh] bg-white rounded-xl p-8 shadow-2xl overflow-hidden"
                onClick={(e) => e.stopPropagation()}
            >
                <ReactECharts
                    option={enhancedOptions}
                    style={{ height: '100%', width: '100%' }}
                    theme="light"
                    onError={(error: any) => {
                        console.error('ECharts modal rendering error:', error);
                    }}
                    opts={{
                        renderer: 'canvas'
                    }}
                />
            </div>
        </div>
    );
};

export default ChartModal;
