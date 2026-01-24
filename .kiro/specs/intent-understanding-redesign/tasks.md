# 实现计划：意图理解系统重新设计

## 概述

本实现计划将现有的8组件意图理解系统简化为4组件架构，提高代码可维护性和系统性能。实现采用Go语言（后端）和TypeScript/React（前端）。

## 任务

- [ ] 1. 创建核心配置和数据结构
  - [x] 1.1 创建 IntentUnderstandingConfig 配置结构
    - 在 `src/agent/intent_understanding_config.go` 中定义简化的配置结构
    - 包含5个核心配置项：Enabled、MaxSuggestions、MaxHistoryRecords、PreferenceThreshold、MaxExclusionSummary
    - 提供默认配置和验证方法
    - _Requirements: 7.3, 11.2_
  
  - [x] 1.2 确保 IntentSuggestion 数据结构兼容
    - 验证现有 IntentSuggestion 结构保持不变
    - 添加必要的辅助方法
    - _Requirements: 7.4_

- [ ] 2. 实现 ContextProvider 组件
  - [x] 2.1 创建 ContextProvider 基础结构
    - 在 `src/agent/context_provider.go` 中实现
    - 整合现有 ContextEnhancer 和 DimensionAnalyzer 的功能
    - 实现 GetContext 方法获取数据源上下文
    - _Requirements: 2.1, 2.7_
  
  - [x] 2.2 实现列类型识别和分析提示生成
    - 复用现有的列类型识别逻辑（dateKeywords、geographicKeywords等）
    - 实现 generateHints 方法生成分析提示
    - 支持日期、地理、数值、分类四种类型
    - _Requirements: 2.2, 2.3, 2.4, 2.5_
  
  - [x] 2.3 实现历史记录获取
    - 复用现有的 AnalysisHistoryStore
    - 实现最近N条记录获取（默认5条）
    - 按时间倒序排列
    - _Requirements: 2.6_
  
  - [ ]* 2.4 编写 ContextProvider 属性测试
    - **Property 3: 上下文包含数据源信息**
    - **Property 4: 列类型识别与分析提示**
    - **Property 5: 历史记录数量限制**
    - **Validates: Requirements 2.1-2.7**

- [ ] 3. 实现 ExclusionManager 组件
  - [x] 3.1 创建 ExclusionManager 基础结构
    - 在 `src/agent/exclusion_manager.go` 中实现
    - 简化现有 ExclusionSummarizer 的逻辑
    - 实现 GenerateSummary 方法
    - _Requirements: 3.2, 3.3_
  
  - [x] 3.2 实现排除项分类合并
    - 实现 CategorizeExclusions 方法
    - 当排除项超过10个时进行分类合并
    - 控制摘要长度不超过300字符
    - _Requirements: 3.4_
  
  - [ ]* 3.3 编写 ExclusionManager 属性测试
    - **Property 6: 排除摘要长度限制**
    - **Property 7: 排除项分类合并**
    - **Validates: Requirements 3.3, 3.4**

- [ ] 4. 实现 IntentRanker 组件
  - [x] 4.1 创建 IntentRanker 基础结构
    - 在 `src/agent/intent_ranker.go` 中实现
    - 简化现有 PreferenceLearner 的意图选择功能
    - 实现 PreferencesStore 存储结构
    - _Requirements: 5.1, 5.2_
  
  - [x] 4.2 实现偏好排序逻辑
    - 实现 RankSuggestions 方法
    - 当选择次数少于阈值（默认3次）时保持原始排序
    - 达到阈值后按频率排序
    - _Requirements: 5.3, 5.4_
  
  - [x] 4.3 实现选择记录功能
    - 实现 RecordSelection 方法
    - 按数据源分别统计
    - 持久化到 JSON 文件
    - _Requirements: 5.1, 5.2_
  
  - [ ]* 4.4 编写 IntentRanker 属性测试
    - **Property 8: 选择记录与数据源隔离**
    - **Property 9: 偏好排序正确性**
    - **Validates: Requirements 5.1-5.4**

- [x] 5. Checkpoint - 确保所有组件测试通过
  - 运行所有单元测试和属性测试
  - 确保所有测试通过，如有问题请询问用户

- [ ] 6. 实现 IntentGenerator 组件
  - [x] 6.1 创建 IntentGenerator 基础结构
    - 在 `src/agent/intent_generator.go` 中实现
    - 整合 ContextProvider 和 ExclusionManager
    - 实现 BuildPrompt 方法构建提示词
    - _Requirements: 1.3_
  
  - [x] 6.2 实现意图生成逻辑
    - 实现 Generate 方法
    - 调用 LLM 生成 3-5 个意图建议
    - 解析 LLM 响应为 IntentSuggestion 列表
    - _Requirements: 1.1, 1.2_
  
  - [ ]* 6.3 编写 IntentGenerator 单元测试
    - 测试提示词构建
    - 测试响应解析
    - 测试错误处理
    - _Requirements: 1.1, 1.2, 1.6_

- [ ] 7. 实现 IntentUnderstandingService 主服务
  - [x] 7.1 创建 IntentUnderstandingService 基础结构
    - 在 `src/agent/intent_understanding_service.go` 中实现
    - 协调所有核心组件
    - 实现配置管理
    - _Requirements: 7.1, 7.3_
  
  - [x] 7.2 实现 GenerateSuggestions 主方法
    - 整合上下文获取、意图生成、偏好排序
    - 处理排除项
    - 返回排序后的意图建议
    - _Requirements: 1.1, 5.3_
  
  - [x] 7.3 实现错误处理和降级逻辑
    - 组件初始化失败时降级
    - LLM 调用失败时返回错误
    - 记录错误日志
    - _Requirements: 7.2, 10.1, 10.2, 10.3, 10.4_
  
  - [ ]* 7.4 编写 IntentUnderstandingService 集成测试
    - 测试完整的意图生成流程
    - 测试配置切换
    - 测试错误处理
    - _Requirements: 7.1, 7.2, 7.3_

- [x] 8. Checkpoint - 确保后端服务测试通过
  - 运行所有后端测试
  - 确保所有测试通过，如有问题请询问用户

- [ ] 9. 集成到现有 API
  - [x] 9.1 更新 app.go 中的意图生成 API
    - 替换现有的 IntentEnhancementService 调用
    - 使用新的 IntentUnderstandingService
    - 保持 API 签名不变
    - _Requirements: 7.1_
  
  - [x] 9.2 更新 GenerateIntentSuggestionsWithExclusions 函数
    - 使用新的服务生成意图建议
    - 传递排除项给 ExclusionManager
    - 返回排序后的建议
    - _Requirements: 3.2, 5.3_
  
  - [x] 9.3 添加选择记录调用
    - 在用户选择意图时调用 RecordSelection
    - 更新偏好数据
    - _Requirements: 5.1_

- [ ] 10. 前端更新
  - [x] 10.1 更新 ChatSidebar.tsx 中的意图显示逻辑
    - 确保意图选项正确显示
    - 显示"已排除N个选项"提示
    - _Requirements: 6.1, 6.4_
  
  - [x] 10.2 更新重新理解按钮逻辑
    - 累积排除项
    - 调用后端生成新建议
    - _Requirements: 3.1_
  
  - [x] 10.3 更新坚持原始请求按钮逻辑
    - 显示原始请求预览（截断至30字符）
    - 清空意图相关状态
    - _Requirements: 4.2, 4.3, 4.4_
  
  - [ ]* 10.4 编写前端单元测试
    - 测试意图显示
    - 测试按钮交互
    - 测试状态管理
    - _Requirements: 6.1-6.6_

- [ ] 11. 清理旧代码
  - [x] 11.1 移除 IntentCache 相关代码
    - 删除 `src/agent/intent_cache.go`
    - 删除 `src/agent/intent_cache_test.go`
    - 移除相关引用
    - _Requirements: 11.3_
  
  - [x] 11.2 移除 SemanticSimilarityCalculator 相关代码
    - 删除 `src/agent/semantic_similarity.go`
    - 删除 `src/agent/semantic_similarity_test.go`
    - 移除相关引用
    - _Requirements: 11.4_
  
  - [x] 11.3 简化 IntentEnhancementService
    - 将其重构为 IntentUnderstandingService 的包装器
    - 或完全移除并更新所有引用
    - _Requirements: 11.1_
  
  - [x] 11.4 更新配置文件
    - 移除旧的配置项
    - 添加新的简化配置
    - _Requirements: 11.2_

- [x] 12. Final Checkpoint - 确保所有测试通过
  - 运行所有后端测试
  - 运行所有前端测试
  - 确保编译通过
  - 确保所有测试通过，如有问题请询问用户

## 注意事项

- 任务标记 `*` 的为可选任务，可以跳过以加快 MVP 开发
- 每个任务都引用了具体的需求，确保可追溯性
- Checkpoint 任务用于验证阶段性成果
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况
