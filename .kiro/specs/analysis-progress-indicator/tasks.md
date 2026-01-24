# Implementation Plan: Analysis Progress Indicator

## Overview

本实现计划将分析进行中指示器功能分解为具体的编码任务。该功能涉及修改 DraggableDashboard 组件、App.tsx 和 i18n 配置文件。

**重要**: 由于现有的 `isAnalysisLoading` 状态管理不可靠，本实现包含状态可靠性保障措施。

## Tasks

- [x] 1. 添加国际化翻译键
  - 在 `src/frontend/src/i18n.ts` 中添加 `analysis_in_progress` 翻译键
  - English: "Analysis in progress"
  - 简体中文: "分析进行中"
  - _Requirements: 3.1_

- [x] 2. 实现状态可靠性保障
  - [x] 2.1 在 App.tsx 中添加应用启动时重置加载状态
    - 在组件挂载时调用 `manager.setLoading(false)`
    - 确保应用启动时加载状态为 false
    - _Requirements: 4.2, 4.3, 4.4_
  
  - [x] 2.2 添加加载状态超时保护
    - 当 `isLoading` 为 true 时启动 60 秒超时计时器
    - 超时后自动重置加载状态为 false
    - 加载状态变为 false 时清除计时器
    - _Requirements: 4.2, 4.3, 4.4_

- [x] 3. 实现分析进行中指示器
  - [x] 3.1 在 DraggableDashboard 组件中添加指示器 JSX
    - 在标题 "智能分析仪表盘" 右侧添加条件渲染的指示器
    - 使用 `isAnalysisLoading` prop 控制显示/隐藏
    - 包含旋转动画 spinner 和文字提示
    - 使用 Tailwind CSS 实现样式
    - _Requirements: 1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 2.4_
  
  - [x] 3.2 使用 i18n 翻译函数显示文字
    - 调用 `t('analysis_in_progress')` 获取翻译文本
    - _Requirements: 3.1, 3.2_

- [x] 4. Checkpoint - 验证功能实现
  - 确保指示器在分析进行中时正确显示
  - 确保指示器在分析完成/取消/失败时正确隐藏
  - 确保应用重启后加载状态为 false
  - 确保中英文切换正常工作
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. 编写单元测试
  - [x] 5.1 编写指示器可见性测试
    - 测试 `isAnalysisLoading=true` 时指示器可见
    - 测试 `isAnalysisLoading=false` 时指示器不可见
    - **Property 1: Indicator Visibility Matches Loading State**
    - **Validates: Requirements 1.1, 1.2, 4.1, 4.2, 4.3, 4.4**
  
  - [x] 5.2 编写国际化测试
    - 测试英文环境下显示正确文本
    - 测试中文环境下显示正确文本
    - **Property 2: Internationalization Text Correctness**
    - **Validates: Requirements 3.1, 3.2**

## Notes

- 所有任务均为必需任务
- 该功能实现简单，主要涉及 UI 渲染逻辑
- 利用现有的 `isAnalysisLoading` 状态，但增加可靠性保障
- 样式使用 Tailwind CSS，与现有设计风格保持一致
- 状态可靠性保障是关键，确保取消分析后关闭程序再打开不会显示加载状态
