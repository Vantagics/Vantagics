# React Hook错误修复

## 问题描述
重新打开应用程序，点击有图表的分析请求时立即报错：
```
Minified React error #310; visit https://reactjs.org/docs/error-decoder.html?invariant=310 for the full message or use the non-minified dev environment for full errors and additional helpful warnings.
```

## 错误分析

### React错误#310
React错误#310通常表示违反了React Hook的使用规则，具体包括：
1. **Hook在条件语句中使用**: Hook不能在if语句、循环或嵌套函数中调用
2. **Hook顺序不一致**: 每次渲染时Hook的调用顺序必须相同
3. **Hook在非组件函数中使用**: Hook只能在React组件或自定义Hook中使用

### 具体问题定位
通过错误堆栈分析，问题出现在：
1. **Dashboard组件**: 在`renderChart`函数内部使用了`React.useMemo`
2. **Chart组件**: 在条件返回语句之后使用了Hook

### 违反的Hook规则
```typescript
// ❌ 错误：在函数内部使用Hook
const renderChart = () => {
    const chartKey = React.useMemo(() => {
        // ...
    }, [chartType, currentChartIndex, chartData]);
};

// ❌ 错误：在条件返回之后使用Hook
const Chart = ({ options }) => {
    if (!options) {
        return <div>Error</div>; // 提前返回
    }
    const enhancedOptions = useMemo(() => { // Hook在条件返回之后
        // ...
    }, [options]);
};
```

## 解决方案

### 1. Dashboard组件修复
将`useMemo`移出函数，改为普通的计算：

```typescript
// ✅ 修复前：在函数内部使用useMemo
const renderChart = () => {
    const chartKey = React.useMemo(() => {
        const contentHash = typeof chartData === 'string' 
            ? chartData.substring(0, 50) 
            : JSON.stringify(chartData).substring(0, 50);
        return `chart-${chartType}-${currentChartIndex}-${contentHash.replace(/[^a-zA-Z0-9]/g, '')}`;
    }, [chartType, currentChartIndex, chartData]);
};

// ✅ 修复后：使用普通计算
const renderChart = () => {
    const contentHash = typeof chartData === 'string' 
        ? chartData.substring(0, 50) 
        : JSON.stringify(chartData).substring(0, 50);
    const chartKey = `chart-${chartType}-${currentChartIndex}-${contentHash.replace(/[^a-zA-Z0-9]/g, '')}`;
};
```

### 2. Chart组件修复
将所有Hook移到组件顶部，在任何条件返回之前：

```typescript
// ✅ 修复前：Hook在条件返回之后
const Chart = ({ options }) => {
    if (!options) {
        return <div>Error</div>; // 提前返回
    }
    const enhancedOptions = useMemo(() => { ... }, [options]); // 违反Hook规则
};

// ✅ 修复后：Hook在组件顶部
const Chart = ({ options }) => {
    const chartRef = useRef(null);
    
    useEffect(() => {
        // 清理逻辑
    }, []);
    
    const enhancedOptions = useMemo(() => {
        // 在useMemo内部处理无效options
        if (!options || typeof options !== 'object') {
            return { title: { text: '图表数据格式错误' }, series: [] };
        }
        // 正常处理逻辑
    }, [options]);
    
    // 条件返回移到Hook之后
    if (!options || typeof options !== 'object') {
        return <div>Error</div>;
    }
};
```

## 修复特点

### 1. Hook规则合规
- **顶层调用**: 所有Hook都在组件的顶层调用
- **顺序一致**: 每次渲染时Hook的调用顺序相同
- **无条件调用**: Hook不在条件语句中调用

### 2. 错误处理改进
- **useMemo内部处理**: 在useMemo内部处理无效数据
- **默认值提供**: 为无效数据提供安全的默认配置
- **双重检查**: 既在useMemo内部处理，也在组件层面检查

### 3. 性能优化保持
- **计算优化**: 保持图表key的计算逻辑
- **内存管理**: 保持ECharts实例的清理逻辑
- **渲染优化**: 保持组件的重新创建机制

## 技术实现

### Dashboard.tsx 修复
```typescript
const renderChart = () => {
    // 移除useMemo，使用普通计算
    const contentHash = typeof chartData === 'string' 
        ? chartData.substring(0, 50) 
        : JSON.stringify(chartData).substring(0, 50);
    const chartKey = `chart-${chartType}-${currentChartIndex}-${contentHash.replace(/[^a-zA-Z0-9]/g, '')}`;
    
    // 使用计算出的key
    <Chart key={chartKey} options={validatedOptions} height="400px" />
};
```

### Chart.tsx 修复
```typescript
const Chart: React.FC<ChartProps> = ({ options, height = '400px' }) => {
    // 所有Hook在组件顶部
    const chartRef = useRef<any>(null);
    
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
    
    const enhancedOptions = React.useMemo(() => {
        // 在useMemo内部处理无效options
        if (!options || typeof options !== 'object') {
            return {
                title: { text: '图表数据格式错误' },
                series: []
            };
        }
        
        // 正常处理逻辑
        try {
            // ... 图表配置处理
        } catch (error) {
            return {
                title: { text: '图表配置错误' },
                series: []
            };
        }
    }, [options]);
    
    // 条件返回在Hook之后
    if (!options || typeof options !== 'object') {
        return <div>Error UI</div>;
    }
    
    return <ReactECharts ... />;
};
```

## 测试验证

### 测试场景
1. **应用启动**: 重新打开应用程序，验证无错误
2. **图表点击**: 点击有图表的分析请求，验证正常显示
3. **图表切换**: 在不同图表之间切换，验证稳定性
4. **错误数据**: 测试无效图表数据的处理
5. **开发工具**: 在开发环境中验证无Hook警告

### 预期结果
- 应用程序正常启动，无React错误
- 图表正常显示和切换
- 控制台无Hook相关警告
- 错误数据得到正确处理
- 保持原有的功能和性能

## 相关文件
- `src/frontend/src/components/Dashboard.tsx`: 移除函数内部的useMemo
- `src/frontend/src/components/Chart.tsx`: 重新组织Hook的调用顺序

## 注意事项
1. **Hook规则**: 严格遵循React Hook的使用规则
2. **性能影响**: 移除useMemo可能有轻微性能影响，但确保了稳定性
3. **错误处理**: 保持了原有的错误处理逻辑
4. **向后兼容**: 修改不影响现有功能