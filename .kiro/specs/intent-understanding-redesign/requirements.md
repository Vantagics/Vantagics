# 需求文档

## 简介

本功能旨在重新设计意图理解系统，基于现有的复杂实现进行简化和优化。现有系统包含多个增强组件（上下文增强、维度分析、示例提供、缓存、偏好学习、排除项摘要），但存在以下问题：

1. **复杂度过高**：8个独立组件，配置项繁多，维护困难
2. **准确性不足**：语义相似度计算基于简单的词袋模型，效果有限
3. **用户体验欠佳**：重新理解循环可能导致用户困惑
4. **性能开销**：多组件协调增加延迟

重新设计的目标是：简化架构、提高准确性、优化体验、保持兼容。

## 术语表

- **Intent_Understanding_System**: 意图理解系统，负责解析用户分析请求并生成意图建议
- **Intent_Suggestion**: 意图建议，包含标题、描述、图标和具体查询的意图解释对象
- **Intent_Generator**: 意图生成器，核心组件，负责调用LLM生成意图建议
- **Context_Provider**: 上下文提供器，整合数据源信息和历史记录为LLM提供上下文
- **Exclusion_Manager**: 排除项管理器，管理用户拒绝的意图并生成排除提示
- **Intent_Ranker**: 意图排序器，根据用户偏好对建议进行排序
- **Data_Source_Context**: 数据源上下文，包含表名、列信息、数据特征等
- **Analysis_History**: 分析历史，用户在数据源上的历史分析记录
- **User_Preference**: 用户偏好，基于历史选择学习的意图偏好

## 需求

### 需求 1：核心意图生成

**用户故事：** 作为数据分析师，我希望系统能准确理解我的分析请求，并提供3-5个高质量的意图建议供我选择。

#### 验收标准

1. WHEN 用户发送分析请求且意图理解功能开启 THEN Intent_Generator SHALL 调用LLM生成3-5个意图建议
2. THE 每个意图建议 SHALL 包含完整的title、description、icon和query字段
3. WHEN 生成意图建议时 THEN Intent_Generator SHALL 将数据源的列信息和数据特征作为上下文传递给LLM
4. THE 意图建议的query字段 SHALL 是可直接执行的具体分析请求
5. THE 意图建议 SHALL 使用与系统语言设置一致的语言（中文或英文）
6. WHEN LLM调用失败 THEN Intent_Understanding_System SHALL 返回错误信息并允许用户使用原始请求

### 需求 2：上下文整合

**用户故事：** 作为数据分析师，我希望系统在生成意图建议时能考虑数据源的特征和我的历史分析，提供更相关的建议。

#### 验收标准

1. WHEN 生成意图建议时 THEN Context_Provider SHALL 收集数据源的表名、列名、列类型信息
2. WHEN 数据源包含日期类型列 THEN Context_Provider SHALL 在上下文中标注"适合时间序列分析"
3. WHEN 数据源包含地理位置列（省、市、区等）THEN Context_Provider SHALL 在上下文中标注"适合区域分析"
4. WHEN 数据源包含数值类型列 THEN Context_Provider SHALL 在上下文中标注"适合统计分析"
5. WHEN 数据源包含分类列 THEN Context_Provider SHALL 在上下文中标注"适合分组对比分析"
6. WHEN 用户有历史分析记录 THEN Context_Provider SHALL 包含最近5条分析记录作为参考
7. THE 上下文信息 SHALL 以结构化格式传递给LLM，便于理解和使用

### 需求 3：重新理解机制

**用户故事：** 作为用户，我希望当提供的意图建议不符合我的需求时，能够请求系统重新理解，并排除之前不满意的选项。

#### 验收标准

1. WHEN 用户点击"重新理解"按钮 THEN Exclusion_Manager SHALL 将当前显示的所有意图建议添加到排除列表
2. WHEN 调用LLM生成新意图建议时 THEN Exclusion_Manager SHALL 将排除列表的摘要传递给LLM
3. THE 排除列表摘要 SHALL 简洁描述已排除的分析方向，不超过300字符
4. WHEN 排除项超过10个 THEN Exclusion_Manager SHALL 对排除项进行分类合并
5. THE Intent_Understanding_System SHALL 支持用户无限次重新理解
6. FOR ALL 新生成的意图建议 SHALL NOT 与排除项在语义上重复

### 需求 4：坚持原始请求

**用户故事：** 作为用户，我希望能够跳过意图理解，直接使用我的原始请求进行分析。

#### 验收标准

1. THE "坚持我的请求"按钮 SHALL 在意图选项显示时始终可见
2. WHEN 用户点击"坚持我的请求"按钮 THEN Intent_Understanding_System SHALL 直接使用原始请求进入分析流程
3. WHEN 用户点击"坚持我的请求"按钮 THEN Intent_Understanding_System SHALL 清空所有意图相关状态
4. THE "坚持我的请求"按钮 SHALL 显示用户原始请求的预览（截断至30字符）

### 需求 5：意图选择与偏好学习

**用户故事：** 作为频繁使用系统的用户，我希望系统能学习我的分析偏好，将我常用的分析类型排在建议列表前面。

#### 验收标准

1. WHEN 用户选择某个意图建议 THEN Intent_Ranker SHALL 记录该选择
2. THE Intent_Ranker SHALL 按数据源分别统计用户的意图选择频率
3. WHEN 生成意图建议后 THEN Intent_Ranker SHALL 根据用户历史选择频率对建议进行重新排序
4. WHEN 用户选择次数少于3次 THEN Intent_Ranker SHALL 保持LLM返回的原始排序
5. WHEN 用户选择某个意图 THEN Intent_Understanding_System SHALL 使用该意图的query字段进入分析流程
6. WHEN 用户选择某个意图 THEN Intent_Understanding_System SHALL 清空所有意图相关状态

### 需求 6：用户界面

**用户故事：** 作为用户，我希望在对话区清晰地看到意图选项和操作按钮，界面简洁易用。

#### 验收标准

1. WHEN 意图建议生成完成 THEN 界面 SHALL 按以下顺序显示：意图选项列表、"重新理解"按钮、"坚持我的请求"按钮
2. THE 每个意图选项 SHALL 以卡片形式展示，包含图标、标题和描述
3. WHEN 正在生成意图建议时 THEN 界面 SHALL 显示加载状态
4. WHEN 排除列表不为空时 THEN 界面 SHALL 显示"已排除N个选项"的提示
5. THE "重新理解"按钮 SHALL 带有刷新图标
6. THE "坚持我的请求"按钮 SHALL 带有原始请求图标

### 需求 7：向后兼容性

**用户故事：** 作为现有用户，我希望新功能不会破坏现有的意图理解流程。

#### 验收标准

1. THE Intent_Understanding_System SHALL 保持现有API签名不变
2. WHEN 新功能组件初始化失败 THEN Intent_Understanding_System SHALL 降级为原有行为
3. THE Intent_Understanding_System SHALL 支持通过配置开关启用或禁用意图理解功能
4. THE 现有的IntentSuggestion数据结构 SHALL 保持不变

### 需求 8：性能要求

**用户故事：** 作为用户，我希望意图建议的响应时间合理，不会因为新功能而显著增加等待时间。

#### 验收标准

1. THE 意图生成请求 SHALL 在5秒内返回结果（不含LLM响应时间）
2. THE 上下文整合过程 SHALL 在100毫秒内完成
3. THE 排除项摘要生成 SHALL 在50毫秒内完成
4. THE 偏好排序 SHALL 在10毫秒内完成

### 需求 9：国际化支持

**用户故事：** 作为中文用户，我希望所有界面文本和意图建议都能正确支持中文。

#### 验收标准

1. THE Intent_Understanding_System SHALL 根据用户语言设置生成对应语言的建议
2. THE 所有用户界面文本 SHALL 支持中英文切换
3. THE 排除项摘要 SHALL 使用与用户语言设置一致的语言
4. WHEN 用户语言为简体中文 THEN 所有系统生成的文本 SHALL 使用简体中文

### 需求 10：错误处理

**用户故事：** 作为用户，我希望系统能优雅地处理各种错误情况，并提供清晰的反馈。

#### 验收标准

1. IF LLM无法生成意图建议（返回空列表）THEN Intent_Understanding_System SHALL 显示友好提示并提供"坚持我的请求"选项
2. IF LLM调用超时 THEN Intent_Understanding_System SHALL 显示超时提示并允许用户重试
3. IF 数据源信息获取失败 THEN Intent_Understanding_System SHALL 使用基本上下文继续生成建议
4. WHEN 发生错误时 THEN Intent_Understanding_System SHALL 记录错误日志便于排查

### 需求 11：代码简化

**用户故事：** 作为开发者，我希望意图理解系统的代码结构清晰，易于维护和扩展。

#### 验收标准

1. THE 新设计 SHALL 将现有8个组件简化为4个核心组件：Intent_Generator、Context_Provider、Exclusion_Manager、Intent_Ranker
2. THE 配置项 SHALL 从现有的10+项简化为5项以内
3. THE 新代码 SHALL 移除未使用的缓存机制（IntentCache）
4. THE 新代码 SHALL 移除复杂的语义相似度计算（SemanticSimilarityCalculator）
5. THE 新代码 SHALL 保留并简化排除项摘要功能（ExclusionSummarizer）
6. THE 新代码 SHALL 保留并简化偏好学习功能（PreferenceLearner）
