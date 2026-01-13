# 智能仪表盘和数据探索器点击自动隐藏会话功能

## 功能描述
当用户点击智能仪表盘区域或数据探索器区域时，会话侧边栏会自动隐藏，提供更好的用户体验和更大的查看空间。

## 实现细节

### 触发条件
会话侧边栏会在以下情况下自动隐藏：
- 点击仪表盘的空白区域
- 点击仪表盘的标题区域
- 点击数据探索器的空白区域
- 点击非交互元素的区域

### 不触发隐藏的情况
为了保持良好的用户体验，以下情况不会隐藏会话侧边栏：

1. **交互元素**：
   - 按钮 (`<button>`)
   - 链接 (`<a>`)
   - 输入框 (`<input>`, `<select>`, `<textarea>`)
   - 具有 `cursor-pointer` 类的元素
   - 具有 `cursor-zoom-in` 类的元素
   - 具有 `role="button"` 属性的元素

2. **图表区域**（仅Dashboard）：
   - 图表容器 (包含 "chart" 的类名)
   - Canvas 元素
   - SVG 元素
   - 表格 (`<table>`)
   - ECharts 容器 (`.echarts-container`)

3. **智能洞察和指标卡片**（仅Dashboard）：
   - 包含 "insight" 的类名元素
   - 包含 "metric" 的类名元素

4. **数据表格**（仅ContextPanel）：
   - 表格元素及其内容

## 代码实现

### Dashboard 组件修改
```typescript
interface DashboardProps {
    // ... 其他属性
    onDashboardClick?: () => void;
}

const handleDashboardClick = (e: React.MouseEvent) => {
    const target = e.target as HTMLElement;
    
    // 检查交互元素、图表区域、智能洞察卡片
    const isInteractiveElement = /* ... */;
    const isInChartArea = /* ... */;
    const isInInsightCard = /* ... */;
    
    // 只有在点击空白区域时才隐藏
    if (!isInteractiveElement && !isInChartArea && !isInInsightCard && onDashboardClick) {
        onDashboardClick();
    }
};
```

### ContextPanel 组件修改
```typescript
interface ContextPanelProps {
    width: number;
    onContextPanelClick?: () => void;
}

const handleContextPanelClick = (e: React.MouseEvent) => {
    const target = e.target as HTMLElement;
    
    // 检查交互元素和表格
    const isInteractiveElement = /* ... */;
    
    if (!isInteractiveElement && onContextPanelClick) {
        onContextPanelClick();
    }
};
```

### App 组件修改
```typescript
<Dashboard 
    data={dashboardData} 
    activeChart={activeChart} 
    userRequestText={selectedUserRequest}
    onDashboardClick={() => {
        if (isChatOpen) {
            setIsChatOpen(false);
        }
    }}
/>

<ContextPanel 
    width={contextPanelWidth}
    onContextPanelClick={() => {
        if (isChatOpen) {
            setIsChatOpen(false);
        }
    }}
/>
```

## 用户体验优势

1. **自动化操作**：用户无需手动点击关闭按钮
2. **直观交互**：点击主要内容区域自然地关闭侧边栏
3. **智能判断**：避免在用户与内容交互时意外关闭
4. **更大视野**：快速获得更多屏幕空间查看数据

## 测试场景

1. **正常隐藏**：
   - 点击仪表盘标题区域 → 会话侧边栏隐藏
   - 点击仪表盘空白区域 → 会话侧边栏隐藏
   - 点击数据探索器标题区域 → 会话侧边栏隐藏
   - 点击数据探索器空白区域 → 会话侧边栏隐藏

2. **保持显示**：
   - 点击智能洞察卡片 → 会话侧边栏保持显示
   - 点击图表区域 → 会话侧边栏保持显示
   - 点击按钮或链接 → 会话侧边栏保持显示
   - 点击表格数据 → 会话侧边栏保持显示
   - 点击数据探索器中的表格 → 会话侧边栏保持显示

3. **边界情况**：
   - 会话侧边栏已关闭时点击 → 无操作
   - 快速连续点击 → 正常响应
   - 在不同区域间切换点击 → 正常响应