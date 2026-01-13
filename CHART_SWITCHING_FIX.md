# 图表切换错误修复

## 问题描述
当用户在不同的分析问题之间切换时，会出现ECharts错误。具体场景：
1. 点击第一个分析问题，显示其关联图表
2. 点击第二个分析问题，显示其关联图表
3. 再次点击第一个分析问题时，出现ECharts错误

## 问题分析

### 根本原因
1. **组件复用问题**: React没有重新创建Chart组件实例，而是尝试更新现有的ECharts实例
2. **状态冲突**: 不同图表的配置和数据在同一个ECharts实例中产生冲突
3. **实例清理不当**: ECharts实例在图表切换时没有正确清理和重新初始化
4. **配置合并问题**: ECharts默认会合并新旧配置，可能导致配置冲突

### 技术细节
- React的组件复用机制导致Chart组件没有重新挂载
- ECharts实例的`setOption`方法默认会合并配置
- 不同图表类型的配置结构可能不兼容
- 组件卸载时ECharts实例没有正确释放

## 解决方案

### 1. 添加稳定的Key属性
为Chart组件生成基于内容的唯一key，确保图表切换时组件重新创建：

```typescript
// Generate a stable key for the chart based on content
const chartKey = React.useMemo(() => {
    const contentHash = typeof chartData === 'string' 
        ? chartData.substring(0, 50) 
        : JSON.stringify(chartData).substring(0, 50);
    return `chart-${chartType}-${currentChartIndex}-${contentHash.replace(/[^a-zA-Z0-9]/g, '')}`;
}, [chartType, currentChartIndex, chartData]);

<Chart 
    key={chartKey}
    options={validatedOptions} 
    height="400px" 
/>
```

### 2. 改进Chart组件生命周期管理
添加ECharts实例的正确清理和配置：

```typescript
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

    return (
        <ReactECharts
            ref={chartRef}
            option={enhancedOptions}
            notMerge={true} // 不合并配置，每次都重新设置
            lazyUpdate={false} // 不延迟更新
            opts={{
                renderer: 'canvas'
            }}
        />
    );
};
```

### 3. ECharts配置优化
- **notMerge={true}**: 每次设置选项时不与之前的配置合并
- **lazyUpdate={false}**: 立即更新图表，不延迟
- **实例清理**: 组件卸载时正确释放ECharts实例

## 修复特点

### 1. 组件隔离
- **唯一Key**: 每个图表都有基于内容的唯一标识
- **强制重创建**: 图表切换时强制重新创建组件实例
- **状态隔离**: 不同图表的状态完全隔离

### 2. 内存管理
- **实例清理**: 组件卸载时正确释放ECharts实例
- **引用管理**: 使用useRef管理ECharts实例引用
- **防止泄漏**: 避免ECharts实例的内存泄漏

### 3. 配置安全
- **不合并配置**: 避免新旧配置的冲突
- **立即更新**: 确保图表立即反映新的配置
- **错误处理**: 保持现有的错误处理机制

## 技术实现

### Dashboard.tsx 修改
```typescript
// 生成稳定的图表key
const chartKey = React.useMemo(() => {
    const contentHash = typeof chartData === 'string' 
        ? chartData.substring(0, 50) 
        : JSON.stringify(chartData).substring(0, 50);
    return `chart-${chartType}-${currentChartIndex}-${contentHash.replace(/[^a-zA-Z0-9]/g, '')}`;
}, [chartType, currentChartIndex, chartData]);

// 使用key确保组件重新创建
<Chart 
    key={chartKey}
    options={validatedOptions} 
    height="400px" 
/>
```

### Chart.tsx 修改
```typescript
const Chart: React.FC<ChartProps> = ({ options, height = '400px' }) => {
    const chartRef = useRef<any>(null);

    // 清理ECharts实例
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

    return (
        <ReactECharts
            ref={chartRef}
            option={enhancedOptions}
            notMerge={true} // 关键：不合并配置
            lazyUpdate={false} // 关键：立即更新
            opts={{ renderer: 'canvas' }}
        />
    );
};
```

## 测试验证

### 测试场景
1. **基本切换**: 在两个不同的分析问题之间切换
2. **多次切换**: 多次在同一组分析问题之间切换
3. **不同图表类型**: 在不同类型的图表之间切换（柱状图、折线图、饼图等）
4. **快速切换**: 快速连续点击不同的分析问题
5. **内存测试**: 长时间切换测试内存泄漏

### 预期结果
- 图表切换时不再出现ECharts错误
- 每次切换都能正确显示对应的图表
- 图表渲染性能良好
- 没有内存泄漏问题
- 控制台没有错误信息

## 相关文件
- `src/frontend/src/components/Dashboard.tsx`: 添加图表key生成逻辑
- `src/frontend/src/components/Chart.tsx`: 改进生命周期管理和ECharts配置

## 注意事项
1. **性能影响**: 使用key强制重创建组件会有轻微的性能开销，但能确保稳定性
2. **内容哈希**: 使用图表内容的哈希作为key的一部分，确保内容变化时组件重新创建
3. **向后兼容**: 修改不影响现有的图表显示功能
4. **调试支持**: 保持现有的调试日志和错误处理机制