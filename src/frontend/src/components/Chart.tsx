import React, { useEffect, useRef } from 'react';
import ReactECharts from 'echarts-for-react';

interface ChartProps {
    options: any;
    height?: string;
}

const Chart: React.FC<ChartProps> = ({ options, height = '400px' }) => {
    const chartRef = useRef<any>(null);

    // 组件卸载时清理ECharts实例
    useEffect(() => {
        return () => {
            if (chartRef.current) {
                const echartsInstance = chartRef.current.getEchartsInstance();
                if (echartsInstance) {
                    echartsInstance.dispose();
                }
            }
        };
    }, []);

    // Add Chinese font support to ECharts options
    const enhancedOptions = React.useMemo(() => {
        // 如果options无效，返回一个默认配置
        if (!options || typeof options !== 'object') {
            return {
                title: { text: '图表数据格式错误' },
                series: []
            };
        }

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
            console.error('Chart: Error processing options', error);
            return {
                title: { text: '图表配置错误' },
                series: []
            };
        }
    }, [options]);

    // 验证输入的options（在Hook之后进行）
    if (!options || typeof options !== 'object') {
        console.error('Chart: Invalid options provided', options);
        return (
            <div className="w-full rounded-xl border border-red-200 bg-red-50 p-4 shadow-sm my-4">
                <div className="text-red-600 text-sm">
                    图表数据格式错误，无法显示图表
                </div>
            </div>
        );
    }

    return (
        <div className="w-full rounded-xl border border-slate-200 bg-white p-4 shadow-sm my-4">
            <ReactECharts
                ref={chartRef}
                option={enhancedOptions}
                style={{ height: height, width: '100%' }}
                theme="light"
                onError={(error) => {
                    console.error('ECharts rendering error:', error);
                }}
                opts={{
                    renderer: 'canvas' // 使用canvas渲染，更稳定
                }}
                notMerge={true} // 不合并配置，每次都重新设置
                lazyUpdate={false} // 不延迟更新
            />
        </div>
    );
};

export default Chart;
