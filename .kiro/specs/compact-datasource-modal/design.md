# Design Document: Compact Data Source Modal

## Overview

本设计文档描述如何优化 Snowflake 和 BigQuery 数据源导入表单的布局，通过减少垂直间距、优化信息提示框、调整 textarea 高度等方式，使表单在标准笔记本屏幕（1366x768 及以上）上完整显示，确保用户无需滚动即可看到确认按钮。

### Design Goals

1. 将 Snowflake 和 BigQuery 表单的总高度控制在 600px 以内
2. 保持所有字段的可用性和可读性
3. 确保在 1280x720 最小分辨率下按钮始终可见
4. 使用渐进式优化策略，优先优化最占空间的元素

## Architecture

### Component Structure

```
AddDataSourceModal (React Component)
├── Modal Container (fixed positioning)
├── Header Section (固定高度)
├── Content Section (可滚动区域)
│   ├── Error Display
│   ├── Name Input
│   ├── Driver Type Select
│   └── Driver-Specific Forms
│       ├── Snowflake Form
│       │   ├── Info Box (优化目标)
│       │   └── Input Fields (优化间距)
│       └── BigQuery Form
│           ├── Info Box (优化目标)
│           ├── Input Fields (优化间距)
│           └── Textarea (优化行数)
└── Footer Section (固定在底部)
    ├── Cancel Button
    └── Import Button
```

### Layout Strategy

采用固定头部和底部、可滚动内容区域的布局模式：

1. **Header**: 固定高度，包含标题
2. **Content**: 使用 `max-height` 和 `overflow-y-auto` 实现滚动
3. **Footer**: 固定在底部，始终可见

## Components and Interfaces

### 1. Modal Container Modifications

**Current Implementation:**
```tsx
<div className="bg-white w-[500px] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900">
```

**Optimized Implementation:**
```tsx
<div className="bg-white w-[500px] max-h-[90vh] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900">
```

**Changes:**
- 添加 `max-h-[90vh]` 确保模态框不超过视口高度的 90%
- 保持 `flex flex-col` 布局以支持固定头部和底部

### 2. Content Section Modifications

**Current Implementation:**
```tsx
<div className="p-6 space-y-4">
```

**Optimized Implementation:**
```tsx
<div className="p-6 space-y-3 overflow-y-auto max-h-[calc(90vh-180px)]">
```

**Changes:**
- `space-y-4` → `space-y-3`: 减少字段间距从 16px 到 12px
- 添加 `overflow-y-auto`: 启用垂直滚动
- 添加 `max-h-[calc(90vh-180px)]`: 为头部（约 80px）和底部（约 100px）预留空间

### 3. Info Box Optimization

#### Snowflake Info Box

**Current Implementation:**
```tsx
<div className="p-3 bg-blue-50 border border-blue-200 rounded-lg">
    <p className="text-sm font-medium text-blue-800 mb-2">
        {t('snowflake_setup_guide') || '❄️ Snowflake Connection'}
    </p>
    <p className="text-xs text-blue-700">
        {t('snowflake_desc') || 'Connect to your Snowflake data warehouse...'}
    </p>
</div>
```

**Optimized Implementation:**
```tsx
<div className="p-2 bg-blue-50 border border-blue-200 rounded-lg">
    <p className="text-xs font-medium text-blue-800 mb-1 leading-tight">
        ❄️ {t('snowflake_setup_guide') || 'Snowflake Connection'}
    </p>
    <p className="text-xs text-blue-700 leading-snug">
        {t('snowflake_desc') || 'Connect to your Snowflake data warehouse...'}
    </p>
</div>
```

**Changes:**
- `p-3` → `p-2`: 减少内边距从 12px 到 8px
- `text-sm` → `text-xs`: 标题字体从 14px 减少到 12px
- `mb-2` → `mb-1`: 减少标题底部边距从 8px 到 4px
- 添加 `leading-tight` 和 `leading-snug`: 减少行高

#### BigQuery Info Box

**Current Implementation:**
```tsx
<div className="p-3 bg-blue-50 border border-blue-200 rounded-lg">
    <p className="text-sm font-medium text-blue-800 mb-2">
        {t('bigquery_setup_guide') || '📊 BigQuery Connection'}
    </p>
    <ol className="text-xs text-blue-700 space-y-1 list-decimal list-inside">
        <li>{t('bigquery_step1') || 'Go to Google Cloud Console'}</li>
        <li>{t('bigquery_step2') || 'Create a service account...'}</li>
        <li>{t('bigquery_step3') || 'Download the JSON key file'}</li>
        <li>{t('bigquery_step4') || 'Paste the JSON content below'}</li>
    </ol>
</div>
```

**Optimized Implementation:**
```tsx
<div className="p-2 bg-blue-50 border border-blue-200 rounded-lg">
    <p className="text-xs font-medium text-blue-800 mb-1 leading-tight">
        📊 {t('bigquery_setup_guide') || 'BigQuery Connection'}
    </p>
    <ol className="text-xs text-blue-700 space-y-0.5 list-decimal list-inside leading-snug">
        <li>{t('bigquery_step1') || 'Go to Google Cloud Console'}</li>
        <li>{t('bigquery_step2') || 'Create a service account...'}</li>
        <li>{t('bigquery_step3') || 'Download the JSON key file'}</li>
        <li>{t('bigquery_step4') || 'Paste the JSON content below'}</li>
    </ol>
</div>
```

**Changes:**
- `p-3` → `p-2`: 减少内边距
- `text-sm` → `text-xs`: 标题字体减小
- `mb-2` → `mb-1`: 减少标题底部边距
- `space-y-1` → `space-y-0.5`: 列表项间距从 4px 减少到 2px
- 添加 `leading-tight` 和 `leading-snug`: 减少行高

### 4. Form Field Spacing

**Current Pattern:**
```tsx
<div>
    <label className="block text-sm font-medium text-slate-700 mb-1">Label</label>
    <input className="w-full border border-slate-300 rounded-md p-2 text-sm..." />
    <p className="text-xs text-slate-500 mt-1">Hint text</p>
</div>
```

**Optimized Pattern:**
```tsx
<div>
    <label className="block text-sm font-medium text-slate-700 mb-1">Label</label>
    <input className="w-full border border-slate-300 rounded-md p-2 text-sm..." />
    <p className="text-xs text-slate-500 mt-0.5 leading-tight">Hint text</p>
</div>
```

**Changes:**
- `mt-1` → `mt-0.5`: 提示文本上边距从 4px 减少到 2px
- 添加 `leading-tight`: 减少提示文本行高

### 5. Textarea Optimization (BigQuery)

**Current Implementation:**
```tsx
<textarea
    value={config.bigqueryCredentials || ''}
    onChange={(e) => setConfig({ ...config, bigqueryCredentials: e.target.value })}
    className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none font-mono"
    placeholder='{"type": "service_account", "project_id": "...", ...}'
    rows={6}
    spellCheck={false}
    autoCorrect="off"
    autoComplete="off"
/>
```

**Optimized Implementation:**
```tsx
<textarea
    value={config.bigqueryCredentials || ''}
    onChange={(e) => setConfig({ ...config, bigqueryCredentials: e.target.value })}
    className="w-full border border-slate-300 rounded-md p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none font-mono resize-y"
    placeholder='{"type": "service_account", "project_id": "...", ...}'
    rows={4}
    spellCheck={false}
    autoCorrect="off"
    autoComplete="off"
/>
```

**Changes:**
- `rows={6}` → `rows={4}`: 默认行数从 6 减少到 4
- 添加 `resize-y`: 允许用户垂直调整大小

### 6. Warning Box Optimization (BigQuery)

**Current Implementation:**
```tsx
<div className="p-3 bg-amber-50 border border-amber-200 rounded-lg">
    <p className="text-xs text-amber-700">
        ⚠️ {t('bigquery_note') || 'Note: BigQuery integration requires...'}
    </p>
</div>
```

**Optimized Implementation:**
```tsx
<div className="p-2 bg-amber-50 border border-amber-200 rounded-lg">
    <p className="text-xs text-amber-700 leading-snug">
        ⚠️ {t('bigquery_note') || 'Note: BigQuery integration requires...'}
    </p>
</div>
```

**Changes:**
- `p-3` → `p-2`: 减少内边距
- 添加 `leading-snug`: 减少行高

## Data Models

无需修改数据模型，所有更改仅涉及 UI 样式。

## Correctness Properties


*属性（Property）是指在系统所有有效执行过程中都应该保持为真的特征或行为——本质上是关于系统应该做什么的形式化陈述。属性是人类可读规范和机器可验证正确性保证之间的桥梁。*

### Property 1: Modal Height Constraint

*For any* data source type (Snowflake or BigQuery), when the form is rendered, the total modal height should not exceed 600px.

**Validates: Requirements 1.1, 2.1**

### Property 2: Info Box Compact Styling

*For any* info box element in Snowflake or BigQuery forms, the computed padding should be 8px (p-2), font-size should be 12px or less (text-xs), and line-height should use tight or snug values.

**Validates: Requirements 1.2, 3.1, 3.2**

### Property 3: Button Visibility Across Resolutions

*For any* viewport size in the range [1280x720, 1920x1080], when the modal is displayed, the confirmation button should be visible within the viewport bounds without requiring scrolling.

**Validates: Requirements 1.3, 2.4, 5.2, 5.3, 5.4**

### Property 4: Form Field Spacing Reduction

*For any* form field container, the vertical spacing between fields should be 12px (space-y-3), and the spacing between labels and inputs should be at least 4px (mb-1).

**Validates: Requirements 1.5, 4.1, 4.2**

### Property 5: Textarea Row Count

*For the* BigQuery service account JSON textarea, the rows attribute should equal 4.

**Validates: Requirements 2.3, 8.1**

### Property 6: Info Box Height Reduction

*For any* info box in BigQuery form, the computed height after optimization should be less than the original height by at least 20%.

**Validates: Requirements 2.2**

### Property 7: Font Size Minimum

*For any* text element within info boxes, the computed font-size should be at least 11px.

**Validates: Requirements 3.3**

### Property 8: List Item Spacing

*For any* ordered or unordered list within info boxes, the spacing between list items should be 2px or less (space-y-0.5).

**Validates: Requirements 3.4**

### Property 9: Hint Text Spacing

*For any* hint text element (text-xs text-slate-500), the top margin should be 2px (mt-0.5) or less.

**Validates: Requirements 4.5**

### Property 10: Scrollable Content Area

*For any* modal where content height exceeds viewport height, the content area should have overflow-y-auto enabled and the footer should remain fixed at the bottom.

**Validates: Requirements 5.1**

### Property 11: Responsive Width

*For any* viewport size, the modal width should remain fixed at 500px.

**Validates: Requirements 6.4**

### Property 12: Scroll Activation Threshold

*For any* viewport with height less than 800px, the content area should have scrolling enabled (overflow-y-auto).

**Validates: Requirements 6.3**

### Property 13: Optional Field Labeling

*For any* optional form field, the label should contain the text "(Optional)" or equivalent localized text.

**Validates: Requirements 7.1**

### Property 14: Field Accessibility

*For any* form field (required or optional), the field should be present in the DOM and not hidden with display:none.

**Validates: Requirements 7.5**

### Property 15: Textarea Scrollbar

*For the* BigQuery textarea, when content exceeds visible rows, the scrollHeight should be greater than clientHeight (indicating scrollbar presence).

**Validates: Requirements 8.3**

### Property 16: Textarea Monospace Font

*For the* BigQuery service account JSON textarea, the computed font-family should include 'monospace' or a monospace font stack.

**Validates: Requirements 8.4**

## Error Handling

### Invalid Viewport Sizes

如果视口尺寸小于最小支持分辨率（1280x720），模态框应该：
1. 保持基本功能可用
2. 启用内容区域滚动
3. 确保按钮始终可见（固定在底部）

### Content Overflow

当表单内容超过可用空间时：
1. 内容区域自动启用垂直滚动
2. 头部和底部保持固定位置
3. 提供视觉提示（如阴影）表明有更多内容

### Browser Compatibility

确保 CSS 属性在主流浏览器中的兼容性：
- `max-h-[90vh]`: 所有现代浏览器支持
- `overflow-y-auto`: 所有浏览器支持
- `calc()`: 所有现代浏览器支持
- Tailwind 自定义类: 通过 PostCSS 编译，兼容性良好

## Testing Strategy

### Unit Testing

使用 React Testing Library 和 Jest 进行单元测试：

1. **Snapshot Tests**: 捕获优化前后的组件快照，确保样式变更符合预期
2. **Style Tests**: 验证关键 CSS 类的应用（space-y-3, p-2, text-xs 等）
3. **Attribute Tests**: 验证 textarea rows 属性、max-height 等关键属性
4. **Responsive Tests**: 使用 `window.matchMedia` 模拟不同视口尺寸

### Property-Based Testing

使用 fast-check (JavaScript) 进行基于属性的测试，每个测试至少运行 100 次迭代：

1. **Height Properties**: 生成随机表单状态，验证模态框高度约束
2. **Spacing Properties**: 验证各种间距值在允许范围内
3. **Visibility Properties**: 在随机视口尺寸下验证按钮可见性
4. **Style Properties**: 验证 CSS 属性值符合设计规范

### Visual Regression Testing

使用 Playwright 或 Cypress 进行视觉回归测试：

1. 捕获 Snowflake 和 BigQuery 表单的截图
2. 在不同分辨率下验证布局
3. 对比优化前后的视觉差异
4. 确保无意外的样式变更

### Manual Testing Checklist

1. ✓ 在 1366x768 分辨率下打开 Snowflake 表单，确认无需滚动即可看到确认按钮
2. ✓ 在 1366x768 分辨率下打开 BigQuery 表单，确认无需滚动即可看到确认按钮
3. ✓ 在 1280x720 分辨率下验证滚动功能正常
4. ✓ 验证所有文本仍然清晰可读
5. ✓ 验证表单字段仍然易于点击和填写
6. ✓ 在不同浏览器（Chrome, Firefox, Safari, Edge）中测试

### Test Configuration

所有基于属性的测试应配置为：
- 最小迭代次数: 100
- 标签格式: `Feature: compact-datasource-modal, Property {number}: {property_text}`
- 每个正确性属性对应一个独立的属性测试

### Testing Tools

- **Unit Tests**: Jest + React Testing Library
- **Property Tests**: fast-check
- **Visual Tests**: Playwright
- **E2E Tests**: Playwright
- **Style Validation**: jest-dom custom matchers

## Implementation Notes

### CSS Class Changes Summary

| Element | Current | Optimized | Savings |
|---------|---------|-----------|---------|
| Content container | `space-y-4` | `space-y-3` | 4px per field |
| Info box padding | `p-3` | `p-2` | 8px total |
| Info box title | `text-sm mb-2` | `text-xs mb-1` | ~6px |
| List spacing | `space-y-1` | `space-y-0.5` | 2px per item |
| Hint text margin | `mt-1` | `mt-0.5` | 2px per hint |
| Textarea rows | `rows={6}` | `rows={4}` | ~40px |

### Estimated Height Savings

**Snowflake Form:**
- Info box: ~15px
- Field spacing (7 fields × 4px): ~28px
- Hint text spacing (5 hints × 2px): ~10px
- **Total: ~53px reduction**

**BigQuery Form:**
- Info box: ~20px
- Field spacing (3 fields × 4px): ~12px
- Textarea: ~40px
- Warning box: ~8px
- Hint text spacing (3 hints × 2px): ~6px
- **Total: ~86px reduction**

### Accessibility Considerations

1. **Keyboard Navigation**: 所有优化不影响 Tab 键导航顺序
2. **Screen Readers**: 标签和提示文本仍然正确关联
3. **Touch Targets**: 输入框和按钮保持足够的点击区域（最小 44x44px）
4. **Contrast**: 文本颜色和背景色对比度符合 WCAG AA 标准
5. **Focus Indicators**: 保持清晰的焦点指示器

### Performance Impact

优化对性能的影响：
- **Rendering**: 减少 DOM 高度可能略微提升渲染性能
- **Layout Calculation**: 更简单的布局可能减少重排时间
- **Memory**: 无显著影响
- **Bundle Size**: 无影响（仅 CSS 类变更）

### Browser Support

- Chrome/Edge: 完全支持
- Firefox: 完全支持
- Safari: 完全支持
- 移动浏览器: 完全支持（虽然模态框主要用于桌面）

### Rollback Plan

如果优化导致问题：
1. 保留原始 CSS 类作为注释
2. 使用 Git 回滚到优化前的提交
3. 通过功能标志控制新旧样式切换
4. 收集用户反馈后再次调整
