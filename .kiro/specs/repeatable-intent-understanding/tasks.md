# 实现计划：可重复意图理解

## 概述

本实现计划将"可重复意图理解"功能分解为可执行的编码任务。实现采用增量方式，先清理旧代码，再实现后端功能，然后实现前端功能，最后进行集成测试。

## 任务

- [x] 1. 清理旧代码并准备基础设施
  - [x] 1.1 清理后端 app.go 中的旧意图生成代码
    - 简化 `GenerateIntentSuggestionsWithExclusions` 函数
    - 移除 `buildIntentUnderstandingPrompt` 中复杂的增强逻辑
    - 保留基本的排除项处理逻辑
    - _Requirements: 10.1_
  
  - [x] 1.2 准备前端代码清理
    - 确认 `handleRetryIntentUnderstanding` 和 `formatIntentSuggestions` 已被移除
    - 保留意图相关状态和 refs 定义
    - _Requirements: 10.2, 10.3, 10.4_

- [x] 2. 实现后端排除项摘要器
  - [x] 2.1 创建 ExclusionSummarizer 结构体
    - 在 `src/agent/exclusion_summarizer.go` 中创建新文件
    - 实现 `NewExclusionSummarizer` 构造函数
    - 实现 `NeedsSummarization` 方法（阈值判断，默认 6 个）
    - _Requirements: 6.1_
  
  - [x] 2.2 实现摘要生成逻辑
    - 实现 `SummarizeExclusions` 方法
    - 提取分析类型和主题
    - 合并相似排除项
    - 控制摘要长度（≤500字符）
    - _Requirements: 6.2, 6.3, 6.4_
  
  - [x]* 2.3 编写 ExclusionSummarizer 单元测试
    - 测试阈值判断
    - 测试摘要生成
    - 测试长度限制
    - **Property 4: 摘要机制正确性**
    - **Validates: Requirements 6.1, 6.3, 6.4, 6.5**

- [x] 3. 更新后端意图生成 API
  - [x] 3.1 集成 ExclusionSummarizer 到 GenerateIntentSuggestionsWithExclusions
    - 当排除项超过阈值时使用摘要
    - 更新 `buildIntentUnderstandingPrompt` 支持摘要
    - _Requirements: 6.5, 2.2_
  
  - [x] 3.2 更新提示词模板
    - 添加"坚持我的请求"相关指导
    - 支持中英文
    - _Requirements: 1.4, 9.1_
  
  - [x]* 3.3 编写后端 API 单元测试
    - 测试正常生成流程
    - 测试带排除项的生成
    - 测试摘要集成
    - **Property 5: 意图建议数量正确性**
    - **Validates: Requirements 1.1**

- [x] 4. Checkpoint - 确保后端测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 5. 实现前端 formatIntentSuggestions 函数
  - [x] 5.1 重新实现 formatIntentSuggestions
    - 格式化意图建议为 Markdown
    - 显示已排除选项数量
    - 添加"重新理解"按钮标记
    - 添加"坚持我的请求"按钮标记（显示原始请求预览）
    - _Requirements: 1.2, 3.1, 3.4, 5.1, 5.2_
  
  - [x]* 5.2 编写 formatIntentSuggestions 单元测试
    - 测试正常格式化
    - 测试空列表处理
    - 测试排除数量显示
    - 测试按钮顺序
    - **Property 3: 意图显示格式正确性**
    - **Validates: Requirements 1.2, 3.1, 5.1**

- [x] 6. 实现前端 handleRetryIntentUnderstanding 函数
  - [x] 6.1 重新实现 handleRetryIntentUnderstanding
    - 将当前意图建议添加到排除列表
    - 调用后端 API (GenerateIntentSuggestionsWithExclusions) 生成新建议
    - 更新 UI 状态
    - 处理加载状态
    - _Requirements: 2.1, 2.4, 2.5_
  
  - [x]* 6.2 编写 handleRetryIntentUnderstanding 单元测试
    - 测试排除项累积
    - 测试 API 调用
    - 测试状态更新
    - **Property 1: 排除项累积正确性**
    - **Validates: Requirements 2.1, 2.2, 2.3**

- [x] 7. 实现前端 handleStickToOriginal 函数
  - [x] 7.1 实现 handleStickToOriginal
    - 使用原始请求进入分析流程
    - 调用 clearIntentState 清理状态
    - _Requirements: 3.2, 3.3_
  
  - [x] 7.2 实现 clearIntentState
    - 清空 intentSuggestions
    - 清空 excludedIntentSuggestions
    - 清空 pendingMessage、pendingThreadId、intentMessageId
    - 同步更新 refs
    - _Requirements: 8.2_
  
  - [x]* 7.3 编写状态管理单元测试
    - 测试状态清理
    - 测试 refs 同步
    - **Property 2: 状态清理正确性**
    - **Property 6: 原始请求保留正确性**
    - **Validates: Requirements 3.2, 3.3, 4.2, 8.2**

- [x] 8. 更新前端意图选择处理
  - [x] 8.1 更新 handleIntentSelect 函数
    - 处理"重新理解"按钮点击（调用 handleRetryIntentUnderstanding）
    - 处理"坚持我的请求"按钮点击（调用 handleStickToOriginal）
    - 处理正常意图选择
    - _Requirements: 4.1, 4.2, 4.3_
  
  - [x] 8.2 更新 MessageBubble 中的按钮点击处理
    - 识别"重新理解"按钮标记 ([INTENT_RETRY_BUTTON])
    - 识别"坚持我的请求"按钮标记 ([INTENT_STICK_ORIGINAL])
    - 正确路由到对应处理函数
    - _Requirements: 5.3, 5.4_

- [x] 9. Checkpoint - 确保前端测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 10. 实现边界情况处理
  - [x] 10.1 处理 LLM 空响应
    - 显示友好提示
    - 自动显示"坚持我的请求"选项
    - _Requirements: 7.1_
  
  - [x] 10.2 处理 LLM 调用失败和超时
    - 显示错误信息
    - 允许重试或使用原始请求
    - _Requirements: 7.2_
  
  - [x] 10.3 处理排除项过多
    - 当排除项 > 15 时显示警告提示
    - 建议用户使用原始请求或重新表述
    - _Requirements: 7.3_

- [x] 11. 添加国际化支持
  - [x] 11.1 添加/更新国际化文本
    - 添加 `stick_to_original` 文本（中英文）
    - 添加 `excluded_count` 文本（中英文）
    - 添加 `no_more_suggestions` 文本（中英文）
    - 添加 `too_many_exclusions_warning` 文本（中英文）
    - 添加 `intent_generation_failed` 文本（中英文）
    - _Requirements: 9.1, 9.2, 9.3, 9.4_
  
  - [x]* 11.2 编写国际化完整性测试
    - 验证所有新增文本键存在于两种语言中
    - **Property 7: 国际化文本完整性**
    - **Validates: Requirements 9.1**

- [x] 12. Checkpoint - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 13. 集成测试和优化
  - [x] 13.1 进行端到端集成测试
    - 测试完整的意图理解流程
    - 测试多次"重新理解"循环
    - 测试"坚持我的请求"流程
    - 测试边界情况
    - _Requirements: 2.4_
  
  - [x] 13.2 性能优化
    - 确保 UI 响应流畅
    - 优化摘要生成性能
    - _Requirements: 1.1_

- [x] 14. Final Checkpoint - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

## 注意事项

- 标记为 `*` 的任务是可选的测试任务，可以跳过以加快 MVP 开发
- 每个任务都引用了具体的需求编号以确保可追溯性
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况
- 实现时需要保持前后端状态同步
- 注意使用 refs 避免 React 闭包问题

## 实现状态总结

所有核心功能任务已完成：
- ✅ 后端 ExclusionSummarizer 已实现（src/agent/exclusion_summarizer.go）
- ✅ 后端 GenerateIntentSuggestionsWithExclusions 已集成摘要功能
- ✅ 后端提示词模板已更新，支持中英文"坚持我的请求"指导
- ✅ 前端 formatIntentSuggestions 已实现（ChatSidebar.tsx）
- ✅ 前端 handleRetryIntentUnderstanding 已实现
- ✅ 前端 handleStickToOriginal 已实现
- ✅ 前端 clearIntentState 已实现
- ✅ 前端 handleIntentSelect 已更新
- ✅ MessageBubble 按钮标记识别已实现
- ✅ 边界情况处理已实现（空响应、失败、排除项过多）
- ✅ 国际化文本已添加（中英文）

可选测试任务（标记为 `*`）已全部实现：
- ✅ 2.3 ExclusionSummarizer 单元测试（src/agent/exclusion_summarizer_test.go）
- ✅ 3.3 后端 API 单元测试（src/app_intent_suggestions_test.go）
- ✅ 5.2 formatIntentSuggestions 单元测试（src/frontend/src/components/ChatSidebar.formatIntentSuggestions.test.tsx）
- ✅ 6.2 handleRetryIntentUnderstanding 单元测试（src/frontend/src/components/ChatSidebar.handleRetryIntent.test.tsx）
- ✅ 7.3 状态管理单元测试（src/frontend/src/components/ChatSidebar.stateManagement.test.tsx）
- ✅ 11.2 国际化完整性测试（src/frontend/src/i18n.test.ts）

