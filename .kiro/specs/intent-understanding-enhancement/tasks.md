# 实现计划: 意图理解增强

## 概述

本实现计划将意图理解增强功能分解为可执行的编码任务。实现采用渐进式方法，每个组件独立开发和测试，最后集成到现有系统中。

## 任务

- [x] 1. 创建基础架构和配置
  - [x] 1.1 创建 IntentEnhancementConfig 配置结构
    - 在 `src/agent/intent_enhancement_config.go` 中定义配置结构
    - 包含所有增强功能的开关和参数
    - 添加默认配置值
    - _Requirements: 6.3_
  
  - [x] 1.2 扩展 config.Config 支持意图增强配置
    - 在 `src/config/config.go` 中添加 IntentEnhancement 字段
    - 更新配置加载和保存逻辑
    - _Requirements: 6.3_
  
  - [x] 1.3 创建 IntentEnhancementService 主服务框架
    - 在 `src/agent/intent_enhancement_service.go` 中创建服务
    - 实现初始化和降级逻辑
    - _Requirements: 6.2, 6.4_

- [x] 2. 实现上下文增强器 (Context Enhancer)
  - [x] 2.1 创建 AnalysisRecord 数据结构和存储
    - 在 `src/agent/context_enhancer.go` 中定义结构
    - 实现 JSON 文件存储和加载
    - _Requirements: 1.1_
  
  - [x] 2.2 实现 ContextEnhancer 核心功能
    - 实现 GetHistoryContext 方法
    - 实现 AddAnalysisRecord 方法
    - 实现按时间排序和数量限制
    - _Requirements: 1.2, 1.4_
  
  - [x] 2.3 实现 BuildContextSection 提示词构建
    - 构建包含历史分析信息的提示词片段
    - 支持中英文两种语言
    - _Requirements: 1.5, 8.1_
  
  - [x]* 2.4 编写 ContextEnhancer 属性测试
    - **Property 1: 历史上下文构建正确性**
    - **Validates: Requirements 1.1, 1.2, 1.4, 1.5**

- [x] 3. 实现维度分析器 (Dimension Analyzer)
  - [x] 3.1 创建 ColumnCharacteristics 和 DimensionRecommendation 结构
    - 在 `src/agent/dimension_analyzer.go` 中定义结构
    - _Requirements: 3.5_
  
  - [x] 3.2 实现列类型识别逻辑
    - 识别日期列（date, time, datetime, 日期, 时间等）
    - 识别地理列（province, city, region, 省份, 城市等）
    - 识别数值列（amount, count, price, 金额, 数量等）
    - 识别分类列（category, type, status, 类型, 状态等）
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_
  
  - [x] 3.3 实现 GetDimensionRecommendations 方法
    - 根据列特征生成维度推荐
    - 实现优先级排序
    - _Requirements: 3.6_
  
  - [x] 3.4 实现 BuildDimensionSection 提示词构建
    - 构建维度推荐的提示词片段
    - 支持中英文两种语言
    - _Requirements: 8.1_
  
  - [x]* 3.5 编写 DimensionAnalyzer 属性测试
    - **Property 3: 维度分析正确性**
    - **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6**

- [x] 4. 检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 5. 实现示例提供器 (Example Provider)
  - [x] 5.1 创建 FewShotExample 结构和内置示例库
    - 在 `src/agent/example_provider.go` 中定义结构
    - 创建销售、财务、用户行为、通用四个领域的示例
    - 每个领域提供中英文两套示例
    - _Requirements: 4.1, 4.4, 8.2_
  
  - [x] 5.2 实现 DetectDomain 领域检测
    - 根据表名和列名检测数据领域
    - 支持关键词匹配
    - _Requirements: 4.2, 4.3_
  
  - [x] 5.3 实现 GetExamples 和 BuildExampleSection
    - 根据领域和语言选择示例
    - 构建 Few-shot 示例的提示词片段
    - _Requirements: 4.1, 4.2, 4.5_
  
  - [x]* 5.4 编写 ExampleProvider 属性测试
    - **Property 4: Few-shot 示例正确性**
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.4**

- [x] 6. 实现语义相似度计算器 (Semantic Similarity Calculator)
  - [x] 6.1 创建 SemanticSimilarityCalculator 结构
    - 在 `src/agent/semantic_similarity.go` 中定义结构
    - 实现基于词袋模型的简单相似度计算
    - _Requirements: 5.7_
  
  - [x] 6.2 实现 CalculateSimilarity 方法
    - 使用 Jaccard 相似度或余弦相似度
    - 支持中英文分词
    - _Requirements: 5.7, 8.3_
  
  - [x] 6.3 实现 IsSimilar 阈值判断
    - 根据配置的阈值判断是否相似
    - _Requirements: 5.2_

- [x] 7. 实现意图缓存 (Intent Cache)
  - [x] 7.1 创建 IntentCache 和 CacheEntry 结构
    - 在 `src/agent/intent_cache.go` 中定义结构
    - 实现内存缓存和 JSON 持久化
    - _Requirements: 5.4_
  
  - [x] 7.2 实现 Get 和 Set 方法
    - 实现缓存查找（使用语义相似度）
    - 实现缓存存储
    - _Requirements: 5.1, 5.3_
  
  - [x] 7.3 实现 LRU 淘汰策略
    - 使用双向链表实现 LRU
    - 当条目超过限制时淘汰最少使用的
    - _Requirements: 5.6_
  
  - [x] 7.4 实现缓存过期清理
    - 在访问时检查过期
    - 定期清理过期条目
    - _Requirements: 5.5_
  
  - [x]* 7.5 编写 IntentCache 属性测试
    - **Property 5: 缓存键唯一性**
    - **Property 6: 缓存语义相似度命中**
    - **Property 7: 缓存LRU淘汰**
    - **Property 8: 缓存过期清理**
    - **Validates: Requirements 5.1, 5.2, 5.4, 5.5, 5.6**

- [x] 8. 检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 9. 扩展偏好学习器 (Preference Learner)
  - [x] 9.1 添加 IntentSelectionRecord 结构
    - 在 `src/agent/preference_learner.go` 中添加结构
    - 添加意图选择存储
    - _Requirements: 2.1_
  
  - [x] 9.2 实现 TrackIntentSelection 方法
    - 记录用户的意图选择
    - 按数据源分别统计
    - _Requirements: 2.1, 2.2, 2.5_
  
  - [x] 9.3 实现 GetIntentRankingBoost 方法
    - 根据选择频率计算排序提升值
    - 处理偏好数据不足的情况
    - _Requirements: 2.3, 2.6_
  
  - [x]* 9.4 编写 PreferenceLearner 扩展属性测试
    - **Property 2: 偏好学习和排序正确性**
    - **Validates: Requirements 2.1, 2.2, 2.3, 2.5**

- [x] 10. 集成到主服务
  - [x] 10.1 实现 EnhancePrompt 方法
    - 整合所有增强组件的输出
    - 构建增强后的完整提示词
    - _Requirements: 1.5, 3.6, 4.1_
  
  - [x] 10.2 实现 RankSuggestions 方法
    - 根据用户偏好重新排序建议
    - _Requirements: 2.3_
  
  - [x] 10.3 实现 RecordSelection 方法
    - 记录用户选择并更新偏好
    - _Requirements: 2.1_

- [x] 11. 修改现有意图生成函数
  - [x] 11.1 修改 GenerateIntentSuggestionsWithExclusions
    - 在 `src/app.go` 中集成 IntentEnhancementService
    - 添加缓存检查逻辑
    - 添加偏好排序逻辑
    - _Requirements: 5.1, 5.2, 5.3, 2.3_
  
  - [x] 11.2 修改 buildIntentUnderstandingPrompt
    - 调用 EnhancePrompt 增强提示词
    - 保持向后兼容性
    - _Requirements: 6.1, 6.4_
  
  - [x] 11.3 添加分析完成后的历史记录
    - 在分析完成时调用 AddAnalysisRecord
    - _Requirements: 1.1_

- [x] 12. 前端集成
  - [x] 12.1 修改 ChatSidebar.tsx 记录意图选择
    - 在用户选择意图时调用后端记录接口
    - _Requirements: 2.1_
  
  - [x] 12.2 添加意图增强配置UI（可选）
    - 在设置页面添加增强功能开关
    - _Requirements: 6.3_

- [x] 13. 检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 14. 编写集成测试和性能测试
  - [x]* 14.1 编写端到端集成测试
    - 测试完整的意图生成流程
    - 验证所有增强功能协同工作
    - _Requirements: 6.4_
  
  - [x]* 14.2 编写配置开关测试
    - **Property 9: 配置开关独立性**
    - **Property 10: 向后兼容性**
    - **Validates: Requirements 6.3, 6.4**
  
  - [x]* 14.3 编写性能测试
    - **Property 12: 缓存命中响应时间**
    - **Validates: Requirements 7.1, 7.2**
  
  - [x]* 14.4 编写多语言测试
    - **Property 11: 多语言输出一致性**
    - **Validates: Requirements 8.1, 8.4**

- [x] 15. 最终检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

## 注意事项

- 标记为 `*` 的任务为可选任务，可以跳过以加快MVP开发
- 每个任务都引用了具体的需求以确保可追溯性
- 检查点确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况
