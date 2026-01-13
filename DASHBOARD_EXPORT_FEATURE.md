# 仪表盘导出功能

## 功能描述
为智能仪表盘添加导出功能，支持将仪表盘内容导出为HTML和PDF格式，方便用户保存和分享分析报告。

## 功能特性

### 1. 智能显示控制
- **条件显示**：只有当仪表盘有可导出内容时才显示导出按钮
- **内容检测**：自动检测核心指标、自动洞察和图表数据
- **位置优化**：导出按钮位于仪表盘右上角，不影响主要内容

### 2. 双格式支持

#### HTML导出
- **完整样式**：包含完整的CSS样式，确保美观的显示效果
- **响应式设计**：支持不同屏幕尺寸的显示
- **打印友好**：包含打印样式，支持纸质输出
- **自动下载**：生成后自动下载到用户设备

#### PDF导出
- **打印预览**：通过浏览器打印功能生成PDF
- **A4格式**：优化为A4纸张大小
- **分页控制**：避免内容跨页断裂
- **即时生成**：无需额外插件或服务

### 3. 内容结构

#### 报告头部
```html
<div class="header">
    <h1>智能仪表盘报告</h1>
    <p>生成时间: 2024-01-15 14:30:25</p>
</div>
```

#### 分析请求信息
```html
<div class="request-info">
    <h3>📊 分析请求</h3>
    <p>用户的具体分析请求内容</p>
</div>
```

#### 核心指标区域
```html
<div class="section">
    <h2>核心指标</h2>
    <div class="metrics-grid">
        <div class="metric-card">
            <div class="metric-title">指标名称</div>
            <div class="metric-value">指标数值</div>
            <div class="metric-change">变化趋势</div>
        </div>
    </div>
</div>
```

#### 图表信息区域
```html
<div class="section">
    <h2>分析图表</h2>
    <div class="chart-section">
        <p>图表类型: ECHARTS</p>
        <p>此报告包含交互式图表，请在原系统中查看完整的可视化效果。</p>
    </div>
</div>
```

#### 自动洞察区域
```html
<div class="section">
    <h2>自动洞察</h2>
    <div class="insights-grid">
        <div class="insight-card">
            <div class="insight-text">洞察内容</div>
        </div>
    </div>
</div>
```

## 技术实现

### 1. 状态管理
```typescript
const [exportDropdownOpen, setExportDropdownOpen] = useState(false);
```

### 2. 内容检测
```typescript
const hasExportableContent = () => {
    const hasMetrics = data?.metrics && Array.isArray(data.metrics) && data.metrics.length > 0;
    const hasInsights = data?.insights && Array.isArray(data.insights) && data.insights.length > 0;
    const hasChart = activeChart !== null;
    return hasMetrics || hasInsights || hasChart;
};
```

### 3. HTML生成
```typescript
const exportAsHTML = () => {
    const timestamp = new Date().toLocaleString('zh-CN');
    let htmlContent = `<!DOCTYPE html>...`;
    
    // 动态添加各个区域的内容
    if (userRequestText) { /* 添加分析请求 */ }
    if (data?.metrics) { /* 添加核心指标 */ }
    if (activeChart) { /* 添加图表信息 */ }
    if (data?.insights) { /* 添加自动洞察 */ }
    
    // 创建并下载文件
    const blob = new Blob([htmlContent], { type: 'text/html;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `dashboard-report-${timestamp}.html`;
    link.click();
};
```

### 4. PDF生成
```typescript
const exportAsPDF = () => {
    const printWindow = window.open('', '_blank');
    printWindow.document.write(printContent);
    printWindow.document.close();
    
    printWindow.onload = () => {
        setTimeout(() => {
            printWindow.print();
            printWindow.close();
        }, 500);
    };
};
```

### 5. 交互控制
```typescript
// 点击外部关闭下拉菜单
React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
        if (exportDropdownOpen) {
            const target = event.target as HTMLElement;
            if (!target.closest('.export-dropdown-container')) {
                setExportDropdownOpen(false);
            }
        }
    };
    
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
}, [exportDropdownOpen]);
```

## 样式设计

### 1. 导出按钮
```css
.export-button {
    background: #3b82f6;
    color: white;
    padding: 8px 16px;
    border-radius: 8px;
    transition: background-color 0.2s;
}

.export-button:hover {
    background: #2563eb;
}
```

### 2. 下拉菜单
```css
.dropdown-menu {
    position: absolute;
    right: 0;
    top: 100%;
    margin-top: 8px;
    width: 192px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 10px 25px rgba(0,0,0,0.1);
    border: 1px solid #e2e8f0;
    z-index: 50;
}
```

### 3. HTML报告样式
```css
/* 现代化设计 */
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

/* 响应式网格 */
.metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 20px;
}

/* 打印优化 */
@media print {
    body { background: white; }
    .section { page-break-inside: avoid; }
}
```

### 4. PDF报告样式
```css
/* A4纸张优化 */
@page {
    margin: 20mm;
    size: A4;
}

/* 分页控制 */
.section {
    page-break-inside: avoid;
    margin-bottom: 25px;
}

/* 简洁设计 */
.metric-card {
    border: 1px solid #e2e8f0;
    padding: 15px;
    text-align: center;
}
```

## 用户体验

### 1. 智能显示
- ✅ 只有在有内容时才显示导出按钮
- ✅ 避免空报告的生成
- ✅ 提供清晰的视觉反馈

### 2. 操作便捷
- ✅ 一键导出，无需复杂配置
- ✅ 自动文件命名（包含时间戳）
- ✅ 支持两种常用格式

### 3. 内容完整
- ✅ 包含所有仪表盘显示的内容
- ✅ 保持原有的数据结构
- ✅ 添加时间戳和来源信息

### 4. 格式优化
- ✅ HTML：适合在线查看和分享
- ✅ PDF：适合打印和正式文档

## 文件命名规则

### HTML文件
```
dashboard-report-2024-01-15T14-30-25.html
```

### PDF文件
```
通过浏览器打印对话框，用户可自定义文件名
默认建议：dashboard-report-YYYY-MM-DD.pdf
```

## 错误处理

### 1. 导出失败
```typescript
try {
    // 导出逻辑
} catch (error) {
    console.error("[Dashboard] Export failed:", error);
    alert('导出失败，请重试');
}
```

### 2. 弹窗阻止
```typescript
const printWindow = window.open('', '_blank');
if (!printWindow) {
    alert('请允许弹出窗口以完成PDF导出');
    return;
}
```

### 3. 内容验证
```typescript
const hasExportableContent = () => {
    // 验证是否有可导出的内容
    return hasMetrics || hasInsights || hasChart;
};
```

## 浏览器兼容性

### 支持的浏览器
- ✅ Chrome 80+
- ✅ Firefox 75+
- ✅ Safari 13+
- ✅ Edge 80+

### 功能支持
- ✅ Blob API（HTML下载）
- ✅ Window.open（PDF打印）
- ✅ CSS Grid（布局）
- ✅ CSS Flexbox（对齐）

## 性能优化

### 1. 按需加载
- 只有在有内容时才渲染导出按钮
- 延迟生成HTML内容直到用户点击

### 2. 内存管理
- 及时清理Blob URL
- 关闭打印窗口释放资源

### 3. 用户反馈
- 提供加载状态指示
- 显示操作完成确认

## 扩展性

### 1. 新增格式
- 可轻松添加Excel、Word等格式
- 模块化的导出函数设计

### 2. 自定义样式
- 支持主题切换
- 可配置的样式模板

### 3. 高级功能
- 批量导出
- 定时导出
- 云端存储集成

## 总结

仪表盘导出功能提供了：

1. **智能化**：自动检测内容，按需显示
2. **多格式**：支持HTML和PDF两种主流格式
3. **美观性**：专业的报告样式设计
4. **易用性**：一键导出，操作简单
5. **完整性**：包含所有仪表盘内容
6. **兼容性**：支持主流浏览器和设备

这个功能大大提升了用户体验，让用户能够方便地保存、分享和打印分析报告，满足了不同场景下的使用需求。