# Implementation Plan: Compact Data Source Modal

## Overview

本实现计划将优化 Snowflake 和 BigQuery 数据源导入表单的布局，通过减少垂直间距、优化信息提示框样式、调整 textarea 高度等方式，使表单总高度控制在 600px 以内，确保在标准笔记本屏幕（1366x768）上完整显示包括确认按钮。

实现策略采用渐进式优化：首先优化模态框容器和内容区域的布局结构，然后优化各个子组件的样式，最后添加测试验证所有优化目标。

## Tasks

- [x] 1. 优化模态框容器和布局结构
  - 修改 AddDataSourceModal 组件的根容器，添加 `max-h-[90vh]` 限制最大高度
  - 修改内容区域，添加 `overflow-y-auto` 和 `max-h-[calc(90vh-180px)]` 实现可滚动内容
  - 将字段间距从 `space-y-4` 改为 `space-y-3`（16px → 12px）
  - 确保头部和底部区域保持固定位置
  - _Requirements: 5.1, 5.2, 6.3_

- [x] 1.1 编写模态框布局结构的属性测试
  - **Property 1: Modal Height Constraint**
  - **Property 3: Button Visibility Across Resolutions**
  - **Property 10: Scrollable Content Area**
  - **Property 11: Responsive Width**
  - **Property 12: Scroll Activation Threshold**
  - **Validates: Requirements 1.1, 2.1, 5.1, 5.2, 5.3, 5.4, 6.3, 6.4**

- [x] 2. 优化 Snowflake 表单的 Info Box 样式
  - 将 Info Box 的 padding 从 `p-3` 改为 `p-2`（12px → 8px）
  - 将标题字体从 `text-sm` 改为 `text-xs`（14px → 12px）
  - 将标题底部边距从 `mb-2` 改为 `mb-1`（8px → 4px）
  - 为标题添加 `leading-tight`，为描述文本添加 `leading-snug` 减少行高
  - _Requirements: 1.2, 3.1, 3.2, 3.3_

- [x] 2.1 编写 Snowflake Info Box 的样式验证测试
  - **Property 2: Info Box Compact Styling**
  - **Property 7: Font Size Minimum**
  - **Validates: Requirements 1.2, 3.1, 3.2, 3.3**

- [x] 3. 优化 BigQuery 表单的 Info Box 样式
  - 将 Info Box 的 padding 从 `p-3` 改为 `p-2`
  - 将标题字体从 `text-sm` 改为 `text-xs`
  - 将标题底部边距从 `mb-2` 改为 `mb-1`
  - 将列表项间距从 `space-y-1` 改为 `space-y-0.5`（4px → 2px）
  - 为标题添加 `leading-tight`，为列表添加 `leading-snug`
  - _Requirements: 2.2, 3.1, 3.2, 3.4_

- [x] 3.1 编写 BigQuery Info Box 的样式验证测试
  - **Property 2: Info Box Compact Styling**
  - **Property 6: Info Box Height Reduction**
  - **Property 8: List Item Spacing**
  - **Validates: Requirements 2.2, 3.1, 3.2, 3.4**

- [x] 4. 优化 BigQuery 表单的 Textarea 组件
  - 将 Service Account JSON textarea 的 rows 属性从 6 改为 4
  - 添加 `resize-y` class 允许用户垂直调整大小
  - 确保保持 `font-mono` 等宽字体
  - 验证 placeholder 和其他属性保持不变
  - _Requirements: 2.3, 8.1, 8.3, 8.4_

- [x] 4.1 编写 Textarea 组件的属性测试
  - **Property 5: Textarea Row Count**
  - **Property 15: Textarea Scrollbar**
  - **Property 16: Textarea Monospace Font**
  - **Validates: Requirements 2.3, 8.1, 8.3, 8.4**

- [x] 5. 优化表单字段间距和提示文本
  - 确认所有表单字段容器使用 `space-y-3`（已在任务 1 中完成）
  - 将所有提示文本（hint text）的上边距从 `mt-1` 改为 `mt-0.5`（4px → 2px）
  - 为提示文本添加 `leading-tight` 减少行高
  - 验证标签和输入框之间的间距保持 `mb-1`（4px）
  - _Requirements: 1.5, 4.1, 4.2, 4.5_

- [x] 5.1 编写表单字段间距的属性测试
  - **Property 4: Form Field Spacing Reduction**
  - **Property 9: Hint Text Spacing**
  - **Validates: Requirements 1.5, 4.1, 4.2, 4.5**

- [x] 6. 优化 BigQuery Warning Box 样式
  - 将 Warning Box 的 padding 从 `p-3` 改为 `p-2`
  - 为文本添加 `leading-snug` 减少行高
  - 保持 `text-xs` 字体大小和 amber 配色方案
  - _Requirements: 3.1, 3.2_

- [x] 7. 验证可选字段标记和可访问性
  - 检查所有可选字段的标签是否包含 "(Optional)" 或本地化等效文本
  - 确保所有字段（必填和可选）在 DOM 中可访问，未使用 `display:none` 隐藏
  - 验证字段的视觉层次清晰，用户可以轻松区分必填和可选字段
  - _Requirements: 7.1, 7.3, 7.5_

- [x] 7.1 编写可选字段和可访问性的单元测试
  - **Property 13: Optional Field Labeling**
  - **Property 14: Field Accessibility**
  - **Validates: Requirements 7.1, 7.3, 7.5**

- [x] 8. Checkpoint - 运行所有测试并验证样式
  - 运行所有单元测试和属性测试，确保通过
  - 在浏览器中手动测试 Snowflake 和 BigQuery 表单
  - 验证在 1366x768 分辨率下表单完整显示
  - 验证在 1280x720 分辨率下滚动功能正常
  - 如有问题请向用户报告

- [x] 9. 添加视觉回归测试（可选）
  - 使用 Playwright 捕获 Snowflake 表单的截图
  - 使用 Playwright 捕获 BigQuery 表单的截图
  - 在多个分辨率下验证布局（1280x720, 1366x768, 1920x1080）
  - 设置基准截图用于未来的回归测试
  - _Requirements: 1.3, 2.4, 6.1, 6.2_

- [ ] 10. 添加可访问性测试（可选）
  - 验证键盘导航顺序正确（Tab 键导航）
  - 验证屏幕阅读器可以正确读取标签和提示文本
  - 验证所有交互元素的触摸目标至少 44x44px
  - 验证文本颜色对比度符合 WCAG AA 标准
  - 验证焦点指示器清晰可见
  - _Requirements: 4.3, 4.4_

- [x] 11. 最终验证和文档更新
  - 在所有主流浏览器中测试（Chrome, Firefox, Safari, Edge）
  - 验证所有 16 个正确性属性都已通过测试
  - 更新相关文档或注释，说明优化的 CSS 类变更
  - 确认所有需求都已满足
  - 向用户报告完成状态

## Notes

- 任务标记 `*` 的为可选任务，可以跳过以加快 MVP 交付
- 每个任务都引用了具体的需求编号以确保可追溯性
- Checkpoint 任务确保增量验证
- 属性测试验证通用正确性属性，每个测试至少运行 100 次迭代
- 单元测试验证特定示例和边缘情况
- 所有 CSS 类变更都使用 Tailwind CSS 工具类
- 优化预计可减少 Snowflake 表单约 53px，BigQuery 表单约 86px
- 保持所有字段的可用性、可读性和可访问性是首要原则
