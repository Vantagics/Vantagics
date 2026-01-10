import React, { useEffect, useRef } from 'react';
import ReactECharts from 'echarts-for-react';

interface ChartProps {
    options: any;
    height?: string;
}

const Chart: React.FC<ChartProps> = ({ options, height = '400px' }) => {
    return (
        <div className="w-full rounded-xl border border-slate-200 bg-white p-4 shadow-sm my-4">
            <ReactECharts 
                option={options} 
                style={{ height: height, width: '100%' }}
                theme="light"
            />
        </div>
    );
};

export default Chart;
