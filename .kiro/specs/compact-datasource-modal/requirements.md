# Requirements Document

## Introduction

本功能旨在优化 Snowflake 和 BigQuery 数据源导入表单的布局，减少垂直高度，确保在各种屏幕分辨率下用户都能看到底部的确认按钮。当前表单过高导致在某些屏幕上显示不完整，影响用户体验。

## Glossary

- **Modal**: 模态对话框，用于添加数据源的弹窗界面
- **Form_Height**: 表单的垂直高度，包括所有输入字段和说明文本
- **Viewport**: 用户可见的浏览器窗口区域
- **Compact_Layout**: 紧凑布局，通过减少间距、优化元素排列来降低整体高度
- **Info_Box**: 信息提示框，用于显示设置指南和说明的区域
- **Textarea**: 多行文本输入框，用于输入 JSON 等长文本内容
- **Confirmation_Button**: 确认按钮，位于模态框底部的"导入"按钮

## Requirements

### Requirement 1: 减少 Snowflake 表单高度

**User Story:** 作为用户，我希望 Snowflake 导入表单更紧凑，这样我可以在不滚动的情况下看到所有字段和确认按钮。

#### Acceptance Criteria

1. WHEN 用户选择 Snowflake 数据源类型 THEN THE Modal SHALL 显示所有必需字段且总高度不超过 600px
2. WHEN 显示 Snowflake 表单 THEN THE Info_Box SHALL 使用更紧凑的样式减少垂直空间占用
3. WHEN 显示 Snowflake 表单 THEN THE Modal SHALL 在标准笔记本屏幕（1366x768）上完整显示包括 Confirmation_Button
4. THE Modal SHALL 保持所有输入字段的可用性和可读性
5. THE Modal SHALL 减少字段之间的垂直间距至合理范围（不小于 8px）

### Requirement 2: 减少 BigQuery 表单高度

**User Story:** 作为用户，我希望 BigQuery 导入表单更紧凑，这样我可以在不滚动的情况下看到所有字段和确认按钮。

#### Acceptance Criteria

1. WHEN 用户选择 BigQuery 数据源类型 THEN THE Modal SHALL 显示所有必需字段且总高度不超过 600px
2. WHEN 显示 BigQuery 表单 THEN THE Info_Box SHALL 折叠或简化四步说明以减少高度
3. WHEN 显示 BigQuery 表单 THEN THE Textarea SHALL 减少默认行数至 4 行同时保持可用性
4. WHEN 显示 BigQuery 表单 THEN THE Modal SHALL 在标准笔记本屏幕（1366x768）上完整显示包括 Confirmation_Button
5. THE Modal SHALL 保持 JSON 输入框的可编辑性和可读性

### Requirement 3: 优化信息提示框样式

**User Story:** 作为用户，我希望设置指南信息更简洁，这样可以减少表单的整体高度。

#### Acceptance Criteria

1. WHEN 显示包含多步骤说明的 Info_Box THEN THE Info_Box SHALL 使用更小的字体和行高
2. WHEN 显示 Info_Box THEN THE Info_Box SHALL 减少内边距（padding）至最小可读值
3. THE Info_Box SHALL 保持文本的可读性（字体不小于 11px）
4. WHERE Info_Box 包含列表 THEN THE Modal SHALL 减少列表项之间的间距
5. THE Info_Box SHALL 在需要时支持折叠/展开功能以节省空间

### Requirement 4: 优化表单字段间距

**User Story:** 作为用户，我希望表单字段排列更紧凑，这样可以在有限的屏幕空间内看到更多内容。

#### Acceptance Criteria

1. WHEN 显示任何数据源表单 THEN THE Modal SHALL 将字段间距从 16px 减少至 12px
2. WHEN 显示表单字段 THEN THE Modal SHALL 保持标签和输入框之间的间距不小于 4px
3. THE Modal SHALL 确保字段间距减少后不影响触摸操作的可用性
4. THE Modal SHALL 在字段密集排列时保持视觉层次清晰
5. WHEN 字段包含提示文本 THEN THE Modal SHALL 减少提示文本的上下边距

### Requirement 5: 确保按钮可见性

**User Story:** 作为用户，我希望无论表单内容多少，我都能看到底部的确认和取消按钮。

#### Acceptance Criteria

1. WHEN Modal 内容高度超过 Viewport THEN THE Modal SHALL 使内容区域可滚动而按钮区域固定在底部
2. WHEN 用户滚动表单内容 THEN THE Confirmation_Button SHALL 始终保持可见
3. THE Modal SHALL 在最小支持分辨率（1280x720）下完整显示按钮区域
4. WHEN Modal 打开时 THEN THE Modal SHALL 自动调整位置确保按钮在 Viewport 内
5. THE Modal SHALL 在内容滚动时提供视觉提示表明有更多内容

### Requirement 6: 保持响应式布局

**User Story:** 作为用户，我希望优化后的表单在不同屏幕尺寸下都能正常工作。

#### Acceptance Criteria

1. WHEN 在不同分辨率下显示 Modal THEN THE Modal SHALL 自动调整布局保持可用性
2. THE Modal SHALL 在 1280x720 到 1920x1080 分辨率范围内正常显示
3. WHEN 屏幕高度小于 800px THEN THE Modal SHALL 启用内容区域滚动
4. THE Modal SHALL 保持固定宽度（500px）以维持表单可读性
5. WHEN 在小屏幕上显示 THEN THE Modal SHALL 优先显示必填字段和按钮

### Requirement 7: 优化可选字段显示

**User Story:** 作为用户，我希望可选字段以更节省空间的方式显示，这样可以减少表单的初始高度。

#### Acceptance Criteria

1. WHERE 字段标记为可选（Optional）THEN THE Modal SHALL 在标签中使用更简洁的标记方式
2. WHEN 显示多个可选字段 THEN THE Modal SHALL 考虑使用折叠区域或更紧凑的布局
3. THE Modal SHALL 确保用户能够轻松识别必填和可选字段
4. WHEN 用户与可选字段交互 THEN THE Modal SHALL 提供清晰的视觉反馈
5. THE Modal SHALL 保持所有字段的可访问性无论是否折叠

### Requirement 8: 优化 Textarea 组件

**User Story:** 作为用户，我希望 JSON 输入框占用更少的垂直空间，同时仍然可以方便地输入和查看内容。

#### Acceptance Criteria

1. WHEN 显示 BigQuery 的 Service Account JSON 输入框 THEN THE Textarea SHALL 默认显示 4 行而非 6 行
2. THE Textarea SHALL 支持自动扩展以适应更多内容
3. WHEN 用户输入内容超过可见行数 THEN THE Textarea SHALL 显示滚动条
4. THE Textarea SHALL 保持等宽字体以便于阅读 JSON 格式
5. THE Textarea SHALL 在焦点状态下提供清晰的视觉边界

