# ECharts显示分离修复

## 问题描述
LLM分析后返回的ECharts代码在会话区对话框中显示，影响用户体验。用户不需要看到技术性的ECharts代码，只需要在仪表盘上看到可视化的图表。

## 需求
- ECharts图表应该只在仪表盘上显示
- 会话区对话框中不显示ECharts代码
- 保持其他功能正常（表格数据、图片等）

## 解决方案

### 当前流程分析
1. **后端处理**: 
   - LLM返回包含ECharts代码的响应
   - 后端检测并提取ECharts数据
   - 通过`dashboard-update`事件发送到仪表盘
   - 将图表数据附加到用户消息的`ChartData`字段

2. **前端显示**:
   - 仪表盘接收`dashboard-update`事件并显示图表
   - MessageBubble组件处理会话区的消息显示
   - 需要隐藏技术性代码，只显示用户友好的内容

### 修改内容

#### MessageBubble.tsx 修改

##### 1. 内容清理增强
```typescript
// 修改前 - 保留ECharts代码块
const cleanedContent = contentWithPlaceholders
    .replace(/```[ \t]*json:dashboard[\s\S]*?```/g, '')
    .replace(/```[ \t]*(sql|SQL)[\s\S]*?```/g, '')
    .replace(/```[ \t]*(python|Python|py)[\s\S]*?```/g, '')

// 修改后 - 隐藏ECharts代码块
const cleanedContent = contentWithPlaceholders
    .replace(/```[ \t]*json:dashboard[\s\S]*?```/g, '')
    .replace(/```[ \t]*json:echarts[\s\S]*?```/g, '') // 新增：隐藏ECharts代码
    .replace(/```[ \t]*(sql|SQL)[\s\S]*?```/g, '')
    .replace(/```[ \t]*(python|Python|py)[\s\S]*?```/g, '')
```

##### 2. ReactMarkdown代码组件修改
```typescript
// 修改前 - 渲染ECharts组件
if (isECharts) {
    try {
        const data = JSON.parse(String(children).replace(/\n$/, ''));
        return <Chart options={data} />;
    } catch (e) {
        console.error("Failed to parse ECharts JSON", e);
    }
}

// 修改后 - 隐藏ECharts代码块
if (isSql || isPython || isECharts) {
    // Return null to hide these technical code blocks
    return null;
}
```

## 技术细节

### 隐藏的代码类型
1. **ECharts代码**: `json:echarts` 格式的代码块
2. **SQL查询**: `sql` 或 `SQL` 格式的代码块  
3. **Python代码**: `python`, `Python`, `py` 格式的代码块

### 保留的内容类型
1. **表格数据**: `json:table` 格式仍在会话区显示
2. **Markdown图片**: 保持正常显示
3. **普通文本**: 分析结果和说明文字
4. **操作按钮**: 智能提取的分析建议按钮

### 数据流程
```
LLM响应 → 后端检测ECharts → 发送到仪表盘 → 附加到用户消息
                                    ↓
                              会话区隐藏代码，只显示文本
```

## 用户体验改进

### 会话区体验
- **简洁界面**: 不显示技术性代码，界面更清爽
- **专注内容**: 用户只看到分析结果和说明
- **操作便利**: 保留分析建议按钮，方便继续交互

### 仪表盘体验  
- **可视化展示**: ECharts图表在仪表盘上正常显示
- **交互功能**: 支持图表缩放、导出等功能
- **多图表支持**: 支持一个分析结果包含多个图表

## 测试验证

### 测试场景
1. **ECharts生成**: 
   - 发送需要图表分析的请求
   - 验证会话区不显示ECharts代码
   - 验证仪表盘正常显示图表

2. **混合内容**:
   - 包含文本、ECharts、表格的响应
   - 验证会话区只显示文本和表格
   - 验证ECharts在仪表盘显示

3. **其他代码类型**:
   - 包含SQL、Python代码的响应
   - 验证这些代码也被隐藏
   - 验证不影响正常文本显示

### 预期结果
- 会话区对话框干净整洁，无技术代码
- 仪表盘正常显示ECharts图表
- 表格数据在会话区正常显示
- 分析建议按钮正常工作
- 图片和普通文本正常显示

## 相关文件
- `src/frontend/src/components/MessageBubble.tsx`: 修改消息显示逻辑
- `src/app.go`: 后端图表数据检测和传递（无需修改）
- `src/frontend/src/components/Dashboard.tsx`: 仪表盘图表显示（无需修改）

## 注意事项
1. **向后兼容**: 修改不影响现有的图表显示功能
2. **表格保留**: `json:table` 格式的表格仍在会话区显示，因为表格数据对用户有直接价值
3. **调试支持**: 开发者仍可在浏览器开发工具中查看完整响应内容
4. **灵活配置**: 如需调整隐藏规则，只需修改正则表达式匹配模式