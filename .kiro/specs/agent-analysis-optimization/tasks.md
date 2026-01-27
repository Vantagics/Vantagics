# Implementation Plan: Agent Analysis Optimization

## Overview

本实现计划将Agent分析流程优化功能分解为可执行的编码任务。实现将基于现有的 `analysis_planner.go`、`eino.go` 和 `datasource_tool.go` 文件进行扩展和优化。

## Tasks

- [x] 1. 实现增强的请求分类器
  - [x] 1.1 定义RequestType常量和ConsultationPatterns
    - 在 `src/agent/analysis_planner.go` 中添加新的RequestType常量
    - 定义咨询类和多步骤分析的关键词模式
    - _Requirements: 1.1, 1.5_
  
  - [x] 1.2 实现RequestClassifier结构体和ClassifyRequest方法
    - 创建 `src/agent/request_classifier.go` 文件
    - 实现基于关键词的请求分类逻辑
    - 集成到现有的detectQuickPath逻辑中
    - _Requirements: 1.1, 1.4, 1.5_
  
  - [ ]* 1.3 编写请求分类器的属性测试
    - **Property 1: Request Classification Validity**
    - **Property 4: Quick Path Detection Consistency**
    - **Property 5: Consultation Keyword Detection**
    - **Validates: Requirements 1.1, 1.4, 1.5**

- [x] 2. 实现Schema管理器
  - [x] 2.1 定义SchemaLevel和SchemaCache结构体
    - 在 `src/agent/schema_manager.go` 中定义SchemaLevel常量
    - 定义SchemaCache结构体包含TTL和缓存内容
    - _Requirements: 2.1, 2.4, 7.1_
  
  - [x] 2.2 实现SchemaManager核心方法
    - 实现GetSchema方法支持分级获取
    - 实现GetSchemaLevel方法返回请求类型对应的Schema级别
    - 实现缓存逻辑（30分钟TTL）
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 7.1, 7.2_
  
  - [x] 2.3 实现缓存失效和日志记录
    - 实现InvalidateCache方法
    - 添加缓存命中日志记录
    - _Requirements: 7.3, 7.5_
  
  - [ ]* 2.4 编写Schema管理器的属性测试
    - **Property 6: Schema Level Mapping Correctness**
    - **Property 7: Schema Cache Round-Trip**
    - **Property 8: Single-Call Detailed Schema Fetch**
    - **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 7.1, 7.2, 7.4**

- [x] 3. Checkpoint - 确保请求分类和Schema管理测试通过
  - 运行所有测试，确保通过
  - 如有问题，请询问用户

- [-] 4. 实现分步执行器
  - [x] 4.1 定义StepResult和StepAction结构体
    - 在 `src/agent/step_executor.go` 中定义相关结构体
    - 定义StepAction常量（proceed, modify, skip, abort）
    - _Requirements: 3.1, 3.2_
  
  - [x] 4.2 实现StepExecutor核心方法
    - 实现ExecuteSteps方法支持顺序执行
    - 实现EvaluateStepResult方法评估步骤结果
    - 实现重试逻辑（最多3次）
    - _Requirements: 3.1, 3.2, 3.4, 3.5_
  
  - [x] 4.3 实现进度更新回调
    - 在每个步骤完成后调用onProgress回调
    - _Requirements: 3.3_
  
  - [ ]* 4.4 编写分步执行器的属性测试
    - **Property 9: Sequential Step Execution Order**
    - **Property 10: Step Evaluation Completeness**
    - **Property 11: Progress Update Emission**
    - **Property 12: Step Retry Limit Enforcement**
    - **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5**

- [x] 5. 实现执行验证器
  - [x] 5.1 定义ExecutionMetrics结构体
    - 在 `src/agent/execution_validator.go` 中定义结构体
    - 包含PlannedCalls, ActualCalls, DeviationScore等字段
    - _Requirements: 4.3, 4.4_
  
  - [x] 5.2 实现ExecutionValidator核心方法
    - 实现ValidatePlan方法验证计划一致性
    - 实现TrackExecution方法跟踪实际工具调用
    - 实现GetMetrics方法返回执行指标
    - _Requirements: 4.1, 4.2, 4.3, 4.4_
  
  - [x] 5.3 实现偏差警告逻辑
    - 实现LogDeviations方法
    - 当偏差超过50%时记录警告
    - _Requirements: 4.5_
  
  - [ ]* 5.4 编写执行验证器的属性测试
    - **Property 13: Plan Validation and Correction**
    - **Property 14: Execution Tracking Completeness**
    - **Property 15: Deviation Warning Threshold**
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5**

- [x] 6. Checkpoint - 确保分步执行和验证测试通过
  - 运行所有测试，确保通过
  - 如有问题，请询问用户

- [x] 7. 增强AnalysisPlanner
  - [x] 7.1 扩展AnalysisPlan结构体
    - 添加RequestType, SchemaLevel, IsMultiStep, Checkpoints字段
    - 扩展PlanStep添加EstimatedDuration, SchemaLevel, QueryType字段
    - _Requirements: 6.1, 6.2, 6.3, 6.4_
  
  - [x] 7.2 更新PlanAnalysis方法
    - 集成RequestClassifier进行请求分类
    - 根据请求类型生成适当的执行计划
    - 为咨询类请求生成只包含基础Schema获取的计划
    - _Requirements: 1.2, 5.1, 5.2_
  
  - [x] 7.3 实现咨询建议生成逻辑
    - 基于表名和数据源摘要生成分析建议
    - 包含分析维度、可视化建议和示例查询
    - _Requirements: 5.2, 5.3_
  
  - [ ]* 7.4 编写增强AnalysisPlanner的属性测试
    - **Property 2: Consultation Requests Exclude SQL**
    - **Property 3: Multi-Step Requests Have Checkpoints**
    - **Property 16: Consultation Suggestion Content**
    - **Property 17: Consultation Tool Call Limit**
    - **Property 18: Exact Tool Names in Plans**
    - **Property 19: Plan Step Completeness**
    - **Validates: Requirements 1.2, 1.3, 5.1, 5.2, 5.3, 5.4, 6.1, 6.2, 6.3, 6.4**

- [x] 8. 集成到主执行流程
  - [x] 8.1 更新EinoService集成新组件
    - 在 `src/agent/eino.go` 中集成SchemaManager
    - 集成ExecutionValidator进行执行跟踪
    - _Requirements: 2.3, 4.3_
  
  - [x] 8.2 更新DataSourceContextTool
    - 修改 `src/agent/datasource_tool.go` 使用SchemaManager
    - 支持基础和详细Schema级别
    - _Requirements: 2.1, 2.2, 2.5_
  
  - [x] 8.3 更新RunAnalysisWithProgress方法
    - 集成StepExecutor用于多步骤分析
    - 添加执行验证和偏差记录
    - _Requirements: 3.1, 4.3, 4.4_

- [x] 9. Checkpoint - 确保集成测试通过
  - 运行所有测试，确保通过
  - 如有问题，请询问用户

- [ ] 10. 编写集成测试
  - [ ]* 10.1 编写咨询请求流程集成测试
    - 测试咨询请求只获取基础Schema
    - 测试不执行SQL
    - 测试生成建议内容
    - _Requirements: 5.1, 5.2, 5.3, 5.4_
  
  - [ ]* 10.2 编写多步骤分析流程集成测试
    - 测试分步执行
    - 测试进度更新
    - 测试检查点处理
    - _Requirements: 3.1, 3.2, 3.3_
  
  - [ ]* 10.3 编写缓存集成测试
    - 测试跨请求缓存
    - 测试缓存失效
    - 测试TTL行为
    - **Property 20: Cache Invalidation on Structure Change**
    - **Property 21: Cache Hit Logging**
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 11. Final Checkpoint - 确保所有测试通过
  - 运行完整测试套件
  - 验证所有属性测试通过
  - 如有问题，请询问用户

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- 实现语言为Go，使用现有的项目结构和依赖
