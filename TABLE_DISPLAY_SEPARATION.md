# Table显示分离功能

## 功能描述
将LLM分析输出的`json:table`内容从会话区隐藏，只在仪表盘区域以表格形式显示，类似于ECharts的处理方式。

## 问题背景
用户反馈LLM分析输出的`json:table`代码块不应该直接在会话区显示，而应该在仪表盘区域以表格形式展示，提供更好的数据可视化体验。

## 解决方案

### 1. 隐藏会话区的Table代码块
在MessageBubble组件中，将`json:table`代码块从会话区隐藏：

#### 内容清理逻辑更新
```typescript
// 更新注释和清理逻辑
// Keep: json:dashboard (for dashboard display)
// Hide: json:echarts, json:table (shown on dashboard instead), SQL queries, Python code

const cleanedContent = contentWithPlaceholders
    .replace(/```[ \t]*json:dashboard[\s\S]*?```/g, '')
    .replace(/```[ \t]*json:echarts[\s\S]*?```/g, '') // 隐藏ECharts代码
    .replace(/```[ \t]*json:table[\s\S]*?```/g, '') // 隐藏Table代码，在仪表盘显示
    .replace(/```[ \t]*(sql|SQL)[\s\S]*?```/g, '')
    .replace(/```[ \t]*(python|Python|py)[\s\S]*?```/g, '')
```

#### ReactMarkdown代码组件更新
```typescript
code(props) {
    const isECharts = className?.includes('json:echarts');
    const isTable = className?.includes('json:table');
    const isSql = className?.includes('sql') || className?.includes('SQL');
    const isPython = className?.includes('python') || className?.includes('Python') || className?.includes('py');

    // Hide SQL, Python, ECharts, and Table code blocks
    // ECharts and Tables are shown on dashboard instead
    if (isSql || isPython || isECharts || isTable) {
        return null; // 隐藏这些技术代码块
    }

    return <code {...rest} className={className}>{children}</code>;
}
```

### 2. 移除会话区的Table渲染
移除MessageBubble组件中直接渲染表格的代码：

```typescript
// 移除以下代码段
{parsedPayload && parsedPayload.type === 'table' && (
    <div className="mt-4 pt-4 border-t border-slate-100">
        <DataTable data={parsedPayload.data} />
    </div>
)}
```

### 3. 仪表盘表格显示
Dashboard组件已经具备完整的表格显示功能：

#### 单表格显示
```typescript
if (chartType === 'table') {
    const tableData = chartData as any[];
    if (!tableData || !Array.isArray(tableData) || tableData.length === 0) return null;
    
    const columns = Object.keys(tableData[0]);
    return (
        <div className="w-full bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
            // 表格渲染逻辑
        </div>
    );
}
```

#### 多表格显示
```typescript
const renderDataTables = () => {
    if (!activeChart?.chartData?.charts) return null;
    
    const tableCharts = activeChart.chartData.charts.filter(
        chart => chart.type === 'table'
    );
    
    if (tableCharts.length === 0) return null;
    
    return (
        <div className="mt-6 space-y-4">
            <h3 className="text-md font-semibold text-slate-700 flex items-center gap-2">
                <Table className="w-5 h-5 text-blue-500" />
                {t('analysis_data') || 'Analysis Data'}
            </h3>
            {tableCharts.map((chart, tableIndex) => {
                // 多表格渲染逻辑
            })}
        </div>
    );
};
```

## 实现效果

### 会话区
- ✅ 隐藏`json:table`代码块
- ✅ 隐藏SQL查询代码
- ✅ 隐藏Python代码
- ✅ 保持ECharts代码隐藏
- ✅ 保持清洁的对话界面

### 仪表盘区
- ✅ 显示表格数据
- ✅ 支持多表格显示
- ✅ 提供表格操作功能（导出等）
- ✅ 与ECharts图表并列显示
- ✅ 响应式表格设计

## 用户体验改进

### 1. 清洁的会话界面
- 用户在会话区看不到技术代码块
- 专注于LLM的分析文本内容
- 避免代码干扰阅读体验

### 2. 专业的数据展示
- 表格在仪表盘区域专业展示
- 提供更好的数据可视化
- 支持表格操作和交互

### 3. 一致的显示逻辑
- 与ECharts处理方式保持一致
- 技术内容统一在仪表盘显示
- 会话区专注于文本交流

## 技术实现细节

### 修改的文件
- `src/frontend/src/components/MessageBubble.tsx`
  - 更新内容清理逻辑
  - 更新ReactMarkdown代码组件
  - 移除会话区表格渲染

### 保持不变的文件
- `src/frontend/src/components/Dashboard.tsx`
  - 已有完整的表格显示功能
  - 支持单表格和多表格显示
  - 无需修改

### 后端兼容性
- 后端发送`json:table`数据的逻辑无需修改
- Dashboard通过`dashboard-update`事件接收数据
- 表格数据通过`chartData.charts`数组传递

## 测试场景

### 场景1：单个表格显示
**输入：**
```
分析结果如下：

```json:table
[
  {"分析维度": "客户地理分布分析", "具体建议": "分析各州/城市的客户集中度"},
  {"分析维度": "客户重复购买分析", "具体建议": "通过customer_id分析重复购买行为"}
]
```
```

**预期效果：**
- 会话区：只显示"分析结果如下："文本，不显示JSON代码
- 仪表盘：显示完整的分析建议表格

### 场景2：混合内容显示
**输入：**
```
数据分析完成，主要发现：

1. 客户分布不均衡
2. 重复购买率较低

```json:table
[{"指标": "客户总数", "数值": "99441"}, {"指标": "重复购买率", "数值": "3.2%"}]
```

```json:echarts
{"title": {"text": "客户分布图"}, "series": [{"type": "bar", "data": [120, 200, 150]}]}
```
```

**预期效果：**
- 会话区：显示文本内容和编号列表，不显示JSON代码
- 仪表盘：同时显示表格和ECharts图表

## 总结

通过这次修改，实现了：

1. **会话区清洁化**：隐藏所有技术代码块，专注于文本交流
2. **仪表盘专业化**：统一在仪表盘显示数据可视化内容
3. **用户体验优化**：提供更清晰的数据展示和更好的交互体验
4. **一致性保持**：与ECharts处理方式保持一致的设计逻辑

这个改进让用户能够在会话区专注于与LLM的文本交流，同时在仪表盘区域获得专业的数据可视化体验。