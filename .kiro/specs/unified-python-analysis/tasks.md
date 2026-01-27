# Implementation Plan: Unified Python Analysis

## Overview

本实现计划将统一Python分析流程优化分解为可执行的编码任务。核心目标是将多次LLM调用合并为单次调用，直接生成完整的Python分析代码。

## Tasks

- [x] 1. 实现Schema上下文构建器
  - [x] 1.1 创建SchemaContextBuilder结构体和接口
    - 在 `src/agent/schema_context_builder.go` 中实现
    - 定义 SchemaContext, TableSchema, ColumnInfo, TableRelationship 数据结构
    - 实现 BuildContext 方法获取完整Schema信息
    - 实现 PrioritizeTables 方法根据用户请求选择相关表
    - _Requirements: 3.1, 3.3, 3.5_
  
  - [x] 1.2 实现Schema缓存机制
    - 添加 SchemaCache 结构体支持TTL缓存
    - 实现缓存命中/未命中逻辑
    - 添加缓存失效机制
    - _Requirements: 3.4_
  
  - [ ]* 1.3 编写Schema上下文构建器的属性测试
    - **Property 5: Schema Caching Round-Trip**
    - **Validates: Requirements 3.4**

- [x] 2. 实现代码验证器
  - [x] 2.1 创建CodeValidator结构体
    - 在 `src/agent/code_validator.go` 中实现
    - 定义 ValidationResult 数据结构
    - 实现 ValidateCode 方法检查代码安全性
    - 实现 ExtractSQLQueries 方法从Python代码中提取SQL
    - _Requirements: 6.1, 6.2, 6.3_
  
  - [x] 2.2 实现SQL只读验证
    - 验证所有SQL语句为SELECT语句
    - 检测并拒绝INSERT/UPDATE/DELETE/DROP等操作
    - _Requirements: 6.3_
  
  - [x] 2.3 实现代码安全检查
    - 检测系统命令执行 (os.system, subprocess)
    - 检测文件删除操作 (os.remove, shutil.rmtree)
    - 检测未授权网络操作 (requests, urllib)
    - _Requirements: 6.1, 6.2_
  
  - [ ]* 2.4 编写代码验证器的属性测试
    - **Property 8: Code Safety Validation**
    - **Validates: Requirements 6.1, 6.2, 6.3**

- [x] 3. Checkpoint - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 4. 实现提示词构建器
  - [x] 4.1 创建AnalysisPromptBuilder结构体
    - 在 `src/agent/analysis_prompt_builder.go` 中实现
    - 定义 CodeTemplate 数据结构
    - 实现 BuildPrompt 方法构建完整提示词
    - 实现 GetTemplate 方法获取代码模板
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_
  
  - [x] 4.2 创建代码模板
    - 定义标准代码结构模板（imports, connection, query, processing, visualization, output）
    - 包含错误处理模板（try-except-finally）
    - 包含中文输出模板
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_
  
  - [ ]* 4.3 编写代码模板结构的属性测试
    - **Property 4: Code Template Structure Validation**
    - **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5**

- [x] 5. 实现统一Python代码生成器
  - [x] 5.1 创建UnifiedPythonGenerator结构体
    - 在 `src/agent/unified_python_generator.go` 中实现
    - 定义 GeneratedCode 数据结构
    - 实现 GenerateAnalysisCode 方法
    - 集成 SchemaContextBuilder, AnalysisPromptBuilder, CodeValidator
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_
  
  - [x] 5.2 实现LLM代码生成调用
    - 构建完整提示词
    - 调用LLM生成代码
    - 解析LLM响应提取Python代码
    - _Requirements: 1.1_
  
  - [x] 5.3 实现代码后处理
    - 注入数据库路径和会话目录
    - 验证生成的代码
    - 提取SQL查询进行验证
    - _Requirements: 1.3, 2.3_
  
  - [ ]* 5.4 编写统一生成器的属性测试
    - **Property 1: Single LLM Call for Unified Analysis**
    - **Property 2: Generated Code Completeness**
    - **Property 3: Schema Context Accuracy**
    - **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 5.5, 8.2**

- [x] 6. Checkpoint - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 7. 实现请求路由器
  - [x] 7.1 创建RequestRouter结构体
    - 在 `src/agent/request_router.go` 中实现
    - 定义 ExecutionPath 常量
    - 实现 RouteRequest 方法
    - 实现 ShouldUseUnifiedPath 方法
    - _Requirements: 5.1, 5.2, 5.3, 5.4_
  
  - [x] 7.2 实现请求分类逻辑
    - 检测可视化关键词
    - 检测简单查询关键词
    - 检测咨询建议关键词
    - _Requirements: 5.1, 5.2, 5.3_
  
  - [ ]* 7.3 编写请求路由的属性测试
    - **Property 7: Request Routing Correctness**
    - **Validates: Requirements 5.1, 5.2, 5.3, 5.4**

- [x] 8. 实现性能指标收集器
  - [x] 8.1 创建AnalysisMetrics结构体
    - 在 `src/agent/analysis_metrics.go` 中实现
    - 定义 MetricsSummary 数据结构
    - 实现各阶段计时方法
    - 实现 GetSummary 方法
    - _Requirements: 8.1, 8.2, 8.3_
  
  - [x] 8.2 实现性能对比和日志
    - 计算与基线方法的对比
    - 记录性能改进百分比
    - 添加超时警告日志
    - _Requirements: 8.3, 8.5_
  
  - [ ]* 8.3 编写性能指标的属性测试
    - **Property 10: Performance Metrics Logging**
    - **Validates: Requirements 8.1, 8.3, 8.4, 8.5**

- [x] 9. 实现结果解析器
  - [x] 9.1 创建ResultParser结构体
    - 在 `src/agent/result_parser.go` 中实现
    - 定义 ExecutionResult 数据结构
    - 实现 ParseOutput 方法解析Python输出
    - 支持多种输出格式（text, table, JSON, image）
    - _Requirements: 4.1, 4.2, 4.5_
  
  - [x] 9.2 实现文件事件发射
    - 检测生成的图表文件
    - 发射文件保存事件
    - 记录文件名、类型、大小
    - _Requirements: 4.3_
  
  - [ ]* 9.3 编写结果解析的属性测试
    - **Property 6: Result Structure Completeness**
    - **Validates: Requirements 4.1, 4.2, 4.5**

- [x] 10. Checkpoint - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 11. 集成到主Agent流程
  - [x] 11.1 修改EinoService集成统一分析路径
    - 在 `src/agent/eino.go` 中添加统一分析路径
    - 在请求处理流程中集成RequestRouter
    - 根据路由结果选择执行路径
    - _Requirements: 5.1, 5.4, 5.5_
  
  - [x] 11.2 实现统一分析执行流程
    - 获取Schema上下文
    - 调用UnifiedPythonGenerator生成代码
    - 执行Python代码
    - 解析并返回结果
    - _Requirements: 1.1, 1.4, 1.5_
  
  - [x] 11.3 添加进度更新和超时处理
    - 在长时间运行时发射进度更新
    - 实现执行超时机制
    - 添加超时警告日志
    - _Requirements: 6.4, 8.4, 8.5_

- [x] 12. 实现执行超时和安全机制
  - [x] 12.1 增强Python执行器的超时控制
    - 在 `src/agent/execution_safety.go` 中实现安全执行包装器
    - 实现进程超时终止
    - 返回超时错误信息
    - _Requirements: 6.4_
  
  - [x] 12.2 实现不安全操作阻断
    - 在执行前进行代码安全检查
    - 阻断不安全代码执行
    - 返回安全错误信息
    - _Requirements: 6.5_
  
  - [ ]* 12.3 编写超时和安全机制的属性测试
    - **Property 9: Execution Timeout Enforcement**
    - **Validates: Requirements 6.4, 6.5**

- [x] 13. Final Checkpoint - 确保所有测试通过
  - 构建成功，所有核心功能已实现。

## Notes

- 标记为 `*` 的任务是可选的测试任务，可以跳过以加快MVP开发
- 每个任务都引用了具体的需求以确保可追溯性
- Checkpoint任务用于确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况

## Implementation Summary

### Files Created:
- `src/agent/schema_context_builder.go` - Schema上下文构建器，支持TTL缓存
- `src/agent/code_validator.go` - 代码安全验证器
- `src/agent/analysis_prompt_builder.go` - 提示词构建器
- `src/agent/unified_python_generator.go` - 统一Python代码生成器
- `src/agent/request_router.go` - 请求路由器
- `src/agent/analysis_metrics.go` - 性能指标收集器
- `src/agent/result_parser.go` - 结果解析器
- `src/agent/example_provider.go` - 示例提供器
- `src/agent/execution_safety.go` - 执行安全包装器

### Files Modified:
- `src/agent/eino.go` - 集成统一分析路径
- `src/agent/datasource_service.go` - 添加表列类型和行数获取方法

### Key Features:
1. 单次LLM调用生成完整Python分析代码
2. Schema上下文缓存（5分钟TTL）
3. 代码安全验证（检测危险操作）
4. SQL只读验证
5. 执行超时控制（120秒）
6. 性能指标收集和对比
7. 智能请求路由（快速路径/统一路径/多步骤路径）
