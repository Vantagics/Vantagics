# Implementation Plan: License Mode Switch

## Overview

本实现计划将授权模式切换功能添加到 AboutModal 组件中。实现采用增量方式，先添加国际化文本，然后实现 UI 组件，最后添加事件处理逻辑。

## Tasks

- [x] 1. 添加国际化文本
  - [x] 1.1 在 i18n.ts 中添加英文翻译
    - 添加 `switch_to_commercial`, `switch_to_opensource` 按钮文本
    - 添加 `confirm_switch_to_commercial`, `confirm_switch_to_commercial_desc` 确认对话框文本
    - 添加 `confirm_switch_to_opensource`, `confirm_switch_to_opensource_desc` 确认对话框文本（含警告）
    - 添加 `deactivate_failed` 错误提示文本
    - _Requirements: 5.1, 5.2, 5.4_
  
  - [x] 1.2 在 i18n.ts 中添加中文翻译
    - 添加对应的中文翻译文本
    - _Requirements: 5.1, 5.2, 5.3_

- [x] 2. 实现 AboutModal 组件修改
  - [x] 2.1 添加状态管理
    - 添加 `showConfirmDialog`, `confirmAction`, `isDeactivating`, `deactivateError` 状态
    - 添加 `showActivationModal` 状态用于控制激活对话框
    - _Requirements: 2.1, 3.1_
  
  - [x] 2.2 实现切换按钮 UI
    - 在工作模式显示区域添加切换按钮
    - 根据 `activated` 状态显示不同的按钮文本
    - 使用 Tailwind CSS 保持样式一致性
    - _Requirements: 1.1, 1.2, 1.3, 1.4_
  
  - [x] 2.3 实现确认对话框 UI
    - 创建内联的确认对话框组件
    - 支持 title, message, confirmText, cancelText 属性
    - 支持 loading 状态和 warning 变体
    - _Requirements: 4.1, 4.2, 4.3, 4.4_
  
  - [x] 2.4 实现事件处理逻辑
    - 实现 `handleSwitchClick` 处理按钮点击
    - 实现 `handleConfirm` 处理确认操作
    - 实现 `handleCancel` 处理取消操作
    - 调用 `DeactivateLicense()` 并处理结果
    - _Requirements: 2.2, 2.3, 3.2, 3.3, 3.4, 3.5_
  
  - [x] 2.5 集成 ActivationModal
    - 导入 ActivationModal 组件
    - 在确认切换到商业模式后打开 ActivationModal
    - 处理激活成功后的状态刷新
    - _Requirements: 2.2, 2.4_

- [x] 3. Checkpoint - 功能验证
  - 确保所有功能正常工作
  - 测试开源模式切换到商业模式流程
  - 测试商业模式切换到开源模式流程
  - 测试取消操作
  - 测试错误处理
  - Ensure all tests pass, ask the user if questions arise.

- [x] 4. 添加测试
  - [x] 4.1 编写属性测试 - Button Text Correctness
    - **Property 1: Button text matches activation state and language**
    - 使用 fast-check 生成随机的激活状态和语言设置
    - 验证按钮文本与预期值匹配
    - **Validates: Requirements 1.2, 1.3**
  
  - [x] 4.2 编写属性测试 - Cancel Preserves State
    - **Property 2: Cancel preserves activation state**
    - 使用 fast-check 生成随机的初始激活状态
    - 验证取消操作后状态不变
    - **Validates: Requirements 2.3, 3.4**
  
  - [x] 4.3 编写属性测试 - Language Consistency
    - **Property 3: All text matches language setting**
    - 使用 fast-check 生成随机语言设置
    - 验证所有文本元素使用正确语言
    - **Validates: Requirements 4.5, 5.3, 5.4**
  
  - [x] 4.4 编写单元测试
    - 测试按钮渲染
    - 测试确认对话框显示/隐藏
    - 测试 DeactivateLicense 调用
    - 测试错误处理
    - _Requirements: 2.1, 3.1, 3.5_

- [x] 5. Final Checkpoint - 完成验证
  - 确保所有测试通过
  - 验证中英文切换正常
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- 本功能主要修改 `AboutModal.tsx` 和 `i18n.ts` 两个文件
- 复用现有的 `ActivationModal` 组件，无需创建新组件
- 使用现有的 `DeactivateLicense()` 和 `GetActivationStatus()` 后端方法
