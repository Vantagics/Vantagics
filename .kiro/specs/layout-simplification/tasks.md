# 实施计划：布局简化

## 概述

分两条主线实施：移除数据源区域（需求 1、2、4）和修复拖拽调整大小（需求 3）。使用 TypeScript/React，属性测试使用 fast-check。

## 任务

- [x] 1. 修复 ResizeHandle 拖拽功能
  - [x] 1.1 重构 ResizeHandle 组件，使用 useRef 替代 useState 跟踪拖拽状态
    - 添加 `isDraggingRef` 用于事件回调中读取拖拽状态，避免闭包陈旧问题
    - 使用 `onDragRef`、`onDragEndRef`、`onDragStartRef` 存储最新的回调引用
    - 在 `handleMouseDown` 中直接注册全局 mousemove/mouseup 监听器，在 `handleMouseUp` 中移除
    - 移除依赖 `isDragging` 状态的 `useEffect` 事件注册逻辑
    - 保留 `isDragging` useState 仅用于视觉渲染（CSS 类名切换）
    - _Requirements: 3.1, 3.2, 3.5, 3.6_

  - [x] 1.2 为 ResizeHandle 编写拖拽交互单元测试
    - 模拟 mousedown → mousemove → mouseup 序列，验证 onDrag 回调被正确调用且 deltaX 值正确
    - 验证拖拽结束后 onDragEnd 被调用
    - 验证非拖拽状态下 mousemove 不触发 onDrag
    - _Requirements: 3.1, 3.2_

- [x] 2. 检查点 - 确保 ResizeHandle 修复后拖拽功能正常
  - 确保所有测试通过，如有问题请向用户确认。

- [x] 3. 简化 LeftPanel 组件
  - [x] 3.1 从 LeftPanel 中移除 DataSourcesSection 相关代码
    - 移除 DataSourcesSection 的 import 和渲染
    - 移除 `dataSources` 状态、`fetchDataSources` 函数
    - 移除 `isLoadingDataSources` 状态及其加载中的 UI
    - 移除数据源事件监听器（data-source-added、data-source-deleted、data-source-renamed）
    - 移除 `handleDataSourceContextMenu` 函数
    - 移除数据源相关的上下文菜单渲染（Browse Data 选项）
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x] 3.2 简化 LeftPanel 的 Props 接口
    - 移除 `onDataSourceSelect`、`onBrowseData`、`selectedDataSourceId`、`onWidthChange` props
    - 更新 LeftPanelProps 接口定义
    - 移除 NewSessionButton 的 `disabled` 属性（不再依赖 selectedDataSourceId）
    - 移除 NewSessionButton 的 `selectedDataSourceName` 属性
    - _Requirements: 2.3, 4.1, 4.3_

  - [x] 3.3 更新 App.tsx 中 LeftPanel 的调用代码
    - 移除传递给 LeftPanel 的已删除 props（onDataSourceSelect、onBrowseData、selectedDataSourceId、onWidthChange）
    - 确保保留 `selectedDataSourceId` 状态（其他组件仍在使用）
    - _Requirements: 4.2_

  - [x] 3.4 为简化后的 LeftPanel 编写单元测试
    - 验证渲染结果不包含 DataSourcesSection（无 "Data Sources" 标题）
    - 验证渲染结果包含 NewSessionButton 和 HistoricalSessionsSection
    - 验证 GetDataSources 未被调用
    - 验证 NewSessionButton 始终可点击（无 disabled 属性）
    - _Requirements: 1.1, 1.2, 1.3, 2.2, 2.3_

- [x] 4. 检查点 - 确保 LeftPanel 简化后界面正常
  - 确保所有测试通过，如有问题请向用户确认。

- [x] 5. 属性测试
  - [x] 5.1 编写面板宽度约束不变量的属性测试
    - **Property 1: 面板宽度约束不变量**
    - 使用 fast-check 生成随机的 totalWidth、leftWidth、rightWidth
    - 验证 calculatePanelWidths 输出始终满足最小值约束且三面板宽度之和等于 totalWidth
    - **Validates: Requirements 3.3**

  - [x] 5.2 编写拖拽调整大小正确性的属性测试
    - **Property 2: 拖拽调整大小的正确性**
    - 使用 fast-check 生成随机的 handlePosition、deltaX、currentWidths、totalWidth
    - 验证 handleResizeDrag 输出满足所有面板约束且宽度之和等于 totalWidth
    - **Validates: Requirements 3.1, 3.2**

  - [x] 5.3 编写面板宽度持久化往返一致性的属性测试
    - **Property 3: 面板宽度持久化往返一致性**
    - 使用 fast-check 生成随机的合法 PanelWidths
    - 执行 savePanelWidths 后 loadPanelWidths，验证 left 和 right 值一致
    - **Validates: Requirements 3.4**

- [x] 6. 最终检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请向用户确认。

## 备注

- 标记 `*` 的任务为可选任务，可跳过以加快 MVP 进度
- 每个任务引用了具体的需求编号以便追溯
- 检查点确保增量验证
- 属性测试验证通用正确性属性，单元测试验证具体示例和边界情况
