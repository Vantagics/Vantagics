# 需求文档

## 简介

本功能旨在增强现有的意图理解系统，通过引入上下文增强、用户偏好学习、动态维度调整、Few-shot示例和缓存机制，提升意图建议的准确性和响应速度。该功能将与现有的 `GenerateIntentSuggestionsWithExclusions` 和 `buildIntentUnderstandingPrompt` 函数集成，同时保持向后兼容性。

## 术语表

- **Intent_Understanding_System**: 意图理解系统，负责解析用户模糊请求并生成多个可能的分析意图建议
- **Context_Enhancer**: 上下文增强器，负责收集和整合历史分析记录作为意图生成的参考
- **Preference_Learner**: 偏好学习器，负责记录用户意图选择并优化建议排序
- **Dimension_Analyzer**: 维度分析器，负责根据数据特征动态调整分析维度
- **Example_Provider**: 示例提供器，负责根据数据类型提供领域特定的Few-shot示例
- **Intent_Cache**: 意图缓存，负责缓存相似请求的意图建议以减少LLM调用
- **Semantic_Similarity_Calculator**: 语义相似度计算器，负责计算请求之间的语义相似度以确定缓存命中
- **Analysis_History**: 分析历史记录，包含用户在特定数据源上执行过的分析类型和结果
- **Intent_Selection_Record**: 意图选择记录，记录用户选择的意图及其频率

## 需求

### 需求 1: 上下文增强

**用户故事:** 作为数据分析师，我希望系统在生成意图建议时考虑我之前在该数据源上的分析历史，以便获得更相关的建议。

#### 验收标准

1. WHEN 用户请求意图建议 THEN Intent_Understanding_System SHALL 从 Analysis_History 中检索该数据源的历史分析记录
2. WHEN 历史分析记录存在 THEN Context_Enhancer SHALL 将最近10条分析记录作为上下文包含在提示词中
3. WHEN 历史分析记录为空 THEN Intent_Understanding_System SHALL 正常生成建议而不包含历史上下文
4. THE Context_Enhancer SHALL 按时间倒序排列历史记录，最新的记录优先
5. WHEN 构建提示词 THEN Context_Enhancer SHALL 包含历史分析的类型、目标维度和关键发现

### 需求 2: 用户偏好学习

**用户故事:** 作为频繁使用系统的用户，我希望系统能学习我的分析偏好，将我常用的分析类型排在建议列表前面。

#### 验收标准

1. WHEN 用户选择一个意图建议 THEN Preference_Learner SHALL 记录该选择到 Intent_Selection_Record
2. THE Preference_Learner SHALL 维护每种意图类型的选择频率计数
3. WHEN 生成意图建议 THEN Intent_Understanding_System SHALL 根据用户历史选择频率对建议进行重新排序
4. WHEN 用户选择频率相同 THEN Intent_Understanding_System SHALL 保持LLM返回的原始排序
5. THE Preference_Learner SHALL 支持按数据源分别统计偏好，以适应不同数据集的分析需求
6. WHEN 偏好数据不足（少于5次选择）THEN Intent_Understanding_System SHALL 使用默认排序

### 需求 3: 动态维度调整

**用户故事:** 作为数据分析师，我希望系统能根据数据的特征自动调整分析维度建议，例如有日期列时强调时间分析。

#### 验收标准

1. WHEN 数据源包含日期类型列 THEN Dimension_Analyzer SHALL 在提示词中强调时间序列分析维度
2. WHEN 数据源包含地理位置列（如省份、城市、区域）THEN Dimension_Analyzer SHALL 在提示词中强调区域分析维度
3. WHEN 数据源包含数值类型列 THEN Dimension_Analyzer SHALL 在提示词中强调统计分析维度
4. WHEN 数据源包含分类列 THEN Dimension_Analyzer SHALL 在提示词中强调分组对比分析维度
5. THE Dimension_Analyzer SHALL 根据列的语义信息（列名和数据类型）自动识别列的分析潜力
6. WHEN 多种维度类型同时存在 THEN Dimension_Analyzer SHALL 按相关性权重排序维度建议

### 需求 4: Few-shot 示例

**用户故事:** 作为系统用户，我希望意图建议的质量更高，通过提供具体示例来帮助LLM更好地理解期望的输出格式。

#### 验收标准

1. WHEN 构建意图理解提示词 THEN Example_Provider SHALL 包含2-3个高质量的Few-shot示例
2. THE Example_Provider SHALL 根据数据类型（销售、财务、用户行为等）选择相关的领域示例
3. WHEN 数据源有语义标签 THEN Example_Provider SHALL 优先使用匹配该领域的示例
4. THE Few-shot 示例 SHALL 展示完整的输入-输出格式，包括title、description、icon和query字段
5. WHEN 无法确定数据领域 THEN Example_Provider SHALL 使用通用的数据分析示例

### 需求 5: 缓存机制

**用户故事:** 作为系统用户，我希望对相似请求的响应更快，通过缓存减少不必要的LLM调用。

#### 验收标准

1. WHEN 生成意图建议前 THEN Intent_Cache SHALL 检查是否存在语义相似的缓存请求
2. WHEN 语义相似度超过阈值（0.85）THEN Intent_Cache SHALL 返回缓存的建议而不调用LLM
3. WHEN 缓存未命中 THEN Intent_Understanding_System SHALL 调用LLM并将结果存入缓存
4. THE Intent_Cache SHALL 使用请求文本和数据源ID作为缓存键的组成部分
5. THE Intent_Cache SHALL 设置缓存过期时间为24小时，以确保建议的时效性
6. WHEN 缓存条目超过1000条 THEN Intent_Cache SHALL 使用LRU策略清理最少使用的条目
7. THE Semantic_Similarity_Calculator SHALL 使用文本嵌入向量计算语义相似度
8. IF 缓存服务不可用 THEN Intent_Understanding_System SHALL 降级为直接调用LLM

### 需求 6: 向后兼容性

**用户故事:** 作为现有用户，我希望新功能不会破坏现有的意图理解流程。

#### 验收标准

1. THE Intent_Understanding_System SHALL 保持现有 `GenerateIntentSuggestions` 和 `GenerateIntentSuggestionsWithExclusions` API的签名不变
2. WHEN 新功能组件初始化失败 THEN Intent_Understanding_System SHALL 降级为原有行为
3. THE Intent_Understanding_System SHALL 支持通过配置开关独立启用或禁用每个增强功能
4. WHEN 所有增强功能禁用 THEN Intent_Understanding_System SHALL 表现与当前版本完全一致

### 需求 7: 性能要求

**用户故事:** 作为用户，我希望意图建议的响应时间不会因为新功能而显著增加。

#### 验收标准

1. WHEN 缓存命中 THEN Intent_Understanding_System SHALL 在100毫秒内返回建议
2. WHEN 缓存未命中 THEN Intent_Understanding_System SHALL 在原有响应时间基础上增加不超过200毫秒
3. THE Context_Enhancer SHALL 异步预加载历史记录以减少请求时延迟
4. THE Dimension_Analyzer SHALL 在数据源加载时预计算维度特征

### 需求 8: 多语言支持

**用户故事:** 作为中文用户，我希望所有增强功能都能正确支持中文。

#### 验收标准

1. THE Intent_Understanding_System SHALL 根据用户语言设置生成对应语言的建议
2. THE Example_Provider SHALL 提供中英文两套Few-shot示例
3. THE Semantic_Similarity_Calculator SHALL 支持中英文混合文本的相似度计算
4. WHEN 用户语言为简体中文 THEN 所有系统生成的文本 SHALL 使用简体中文
