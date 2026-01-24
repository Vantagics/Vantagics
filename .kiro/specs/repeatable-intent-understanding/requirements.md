# 需求文档

## 简介

本功能实现"可重复意图理解"机制。当用户发出分析请求且自动意图理解选项开启时，系统由 LLM 根据数据源的背景信息理解用户的意图，将分析请求改写为更具体的描述。系统将所有可能的理解结果、"重新理解"选项以及"坚持我的请求"选项展示在对话区。用户可以无限次点击"重新理解"获取新的意图选项（每次将之前的选项作为排除项），也可以选择"坚持我的请求"直接使用原始输入进行分析。

## 术语表

- **Intent_Understanding_System**: 意图理解系统，负责解析用户的分析请求并生成多个可能的意图解释
- **Intent_Suggestion**: 意图建议，包含标题、描述、图标和具体查询内容的意图解释对象
- **Exclusion_List**: 排除列表，累积保存用户已拒绝的所有意图建议
- **Retry_Button**: 重新理解按钮，用户点击后触发新一轮意图理解
- **Stick_To_Original_Button**: 坚持我的请求按钮，用户点击后直接使用原始输入进行分析
- **Chat_Area**: 对话区，显示意图选项和交互按钮的界面区域
- **Data_Source_Context**: 数据源上下文，包含表名、列信息等用于意图理解的背景信息
- **Original_Request**: 用户的原始分析请求文本

## 需求

### 需求 1：初始意图理解

**用户故事：** 作为用户，我希望发送分析请求后，系统能自动理解我的意图并提供多个可能的解释供我选择。

#### 验收标准

1. WHEN 用户发送分析请求且自动意图理解选项开启 THEN Intent_Understanding_System SHALL 调用 LLM 生成 3-5 个意图建议
2. WHEN 意图建议生成完成 THEN Chat_Area SHALL 显示所有意图选项、"重新理解"按钮和"坚持我的请求"按钮
3. THE 意图选项 SHALL 以可点击的形式展示，包含图标、标题和描述
4. THE LLM 生成的意图建议 SHALL 使用与系统语言设置一致的语言

### 需求 2：重新理解循环

**用户故事：** 作为用户，我希望能够无限次点击"重新理解"按钮，直到找到符合我意图的选项。

#### 验收标准

1. WHEN 用户点击"重新理解"按钮 THEN Intent_Understanding_System SHALL 将当前显示的所有意图建议添加到 Exclusion_List 中
2. WHEN 调用 LLM 生成新意图建议时 THEN Intent_Understanding_System SHALL 将完整的 Exclusion_List 传递给 LLM 作为排除项
3. THE Exclusion_List SHALL 在整个意图理解会话期间持续累积，直到用户选择某个意图或点击"坚持我的请求"
4. THE Intent_Understanding_System SHALL 支持用户无限次重复点击"重新理解"按钮
5. WHILE 正在生成新意图建议时 THE "重新理解"按钮 SHALL 显示加载状态并禁用点击
6. FOR ALL 排除项 THE LLM 生成的新建议 SHALL NOT 与排除项在语义上重复

### 需求 3：坚持原始请求

**用户故事：** 作为用户，我希望能够跳过意图理解，直接使用我的原始请求进行分析。

#### 验收标准

1. THE "坚持我的请求"按钮 SHALL 在意图选项显示时始终可见
2. WHEN 用户点击"坚持我的请求"按钮 THEN Intent_Understanding_System SHALL 直接使用 Original_Request 进入分析流程
3. WHEN 用户点击"坚持我的请求"按钮 THEN Intent_Understanding_System SHALL 清空所有意图相关状态（Exclusion_List、pendingMessage、intentSuggestions 等）
4. THE "坚持我的请求"按钮 SHALL 显示用户原始请求的预览（截断显示）

### 需求 4：选择意图

**用户故事：** 作为用户，我希望选择某个意图后，系统能使用该意图的具体查询进行分析。

#### 验收标准

1. WHEN 用户选择某个意图建议 THEN Intent_Understanding_System SHALL 使用该意图的 query 字段进入分析流程
2. WHEN 用户选择某个意图 THEN Intent_Understanding_System SHALL 清空所有意图相关状态
3. WHEN 用户选择某个意图 THEN Intent_Understanding_System SHALL 记录该选择用于偏好学习

### 需求 5：用户界面显示

**用户故事：** 作为用户，我希望在对话区清晰地看到意图选项和操作按钮。

#### 验收标准

1. WHEN 意图建议生成完成 THEN Chat_Area SHALL 按以下顺序显示：意图选项列表、"重新理解"按钮、"坚持我的请求"按钮
2. WHEN Exclusion_List 不为空时 THEN Chat_Area SHALL 显示已排除选项的数量（如"已排除 N 个选项"）
3. THE "重新理解"按钮 SHALL 带有明显的视觉区分（如刷新图标）
4. THE "坚持我的请求"按钮 SHALL 带有明显的视觉区分（如原始请求图标）

### 需求 6：排除项摘要机制

**用户故事：** 作为用户，我希望即使多次重新理解，系统也能高效处理排除项，不会因为上下文过长而影响性能或准确性。

#### 验收标准

1. WHEN Exclusion_List 中的排除项数量超过阈值（如 6 个） THEN Intent_Understanding_System SHALL 对排除项进行摘要压缩
2. THE 摘要 SHALL 保留排除项的核心语义特征，包括分析类型、目标维度和关键主题
3. WHEN 生成摘要时 THEN Intent_Understanding_System SHALL 将多个相似的排除项合并为一个摘要描述
4. THE 摘要后的排除项描述 SHALL 控制在合理长度内（如不超过 500 字符）
5. THE LLM 提示词 SHALL 使用摘要后的排除描述而非完整的排除项列表，以防止上下文超载

### 需求 7：边界情况处理

**用户故事：** 作为用户，我希望系统能优雅地处理各种边界情况。

#### 验收标准

1. IF LLM 无法生成新的意图建议（返回空列表） THEN Intent_Understanding_System SHALL 显示友好提示并自动提供"坚持我的请求"选项
2. IF LLM 调用失败或超时 THEN Intent_Understanding_System SHALL 显示错误信息并允许用户重试或使用原始请求
3. WHEN 排除项数量超过合理阈值（如 15 个） THEN Intent_Understanding_System SHALL 提示用户考虑使用原始请求或重新表述

### 需求 8：状态管理

**用户故事：** 作为开发者，我希望意图理解的状态能够正确管理，避免状态混乱。

#### 验收标准

1. THE Intent_Understanding_System SHALL 使用 React refs 跟踪意图相关状态以避免闭包问题
2. WHEN 意图理解会话结束 THEN Intent_Understanding_System SHALL 清理所有相关状态
3. THE 状态更新 SHALL 保持同步，确保 refs 和 state 的一致性

### 需求 9：国际化支持

**用户故事：** 作为用户，我希望意图理解界面支持中英文显示。

#### 验收标准

1. THE 所有用户界面文本 SHALL 支持中英文切换
2. THE "重新理解"按钮文本 SHALL 根据语言设置显示对应文本
3. THE "坚持我的请求"按钮文本 SHALL 根据语言设置显示对应文本
4. THE 已排除选项数量提示 SHALL 根据语言设置显示对应文本

### 需求 10：代码清理

**用户故事：** 作为开发者，我希望清理旧的意图理解代码，保持代码整洁。

#### 验收标准

1. THE 后端 GenerateIntentSuggestionsWithExclusions 函数 SHALL 简化，移除复杂的增强逻辑
2. THE 前端 handleRetryIntentUnderstanding 函数 SHALL 重新实现以支持新的流程
3. THE 前端 formatIntentSuggestions 函数 SHALL 重新实现以支持新的显示格式
4. THE 旧代码清理 SHALL 保留相关的 UI 组件和类型定义
