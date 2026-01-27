# Agent Analysis Optimization - Quick Reference

## 核心改进

### 1. 咨询类请求优化
```go
// 之前: 2次工具调用
// 现在: 1次工具调用

// 自动检测咨询关键词
classifier := NewRequestClassifier(logger)
requestType := classifier.ClassifyRequest("对本数据源提出一些分析建议", "")
// 返回: RequestTypeConsultation

// 自动生成简化计划
plan := planner.createConsultationPlan()
// 只包含1个步骤: get_data_source_context (basic level)
```

### 2. Schema缓存机制
```go
// 创建Schema管理器
schemaManager := NewSchemaManager(dsService, 30*time.Minute, logger)

// 第一次调用: 获取并缓存
content, cached, err := schemaManager.GetSchema(ctx, dataSourceID, SchemaLevelBasic)
// cached = false, 执行了获取

// 第二次调用: 使用缓存
content, cached, err := schemaManager.GetSchema(ctx, dataSourceID, SchemaLevelBasic)
// cached = true, 直接返回缓存

// 失效缓存
schemaManager.InvalidateCache(dataSourceID)
```

### 3. 执行计划验证
```go
// 创建验证器
validator := NewExecutionValidator(logger)

// 验证计划
plan, warnings := validator.ValidatePlan(plan, RequestTypeConsultation)
// 自动移除SQL步骤（如果有的话）

// 跟踪执行
validator.TrackExecution("get_data_source_context", 1)
validator.TrackExecution("execute_sql", 2)

// 获取指标
metrics := validator.GetMetrics()
// metrics.DeviationScore: 0.0 = 完美匹配, 1.0 = 完全不同

// 记录偏差
validator.LogDeviations()
// 如果偏差 > 0.5，记录警告
```

### 4. 分步执行
```go
// 创建执行器
executor := NewStepExecutor(planner, validator, 3, logger)

// 执行多步骤分析
result, err := executor.ExecuteSteps(ctx, plan, func(step, total int, result *StepResult) {
    fmt.Printf("Step %d/%d: %v\n", step, total, result.Success)
})

// 每个步骤最多重试3次
// 支持进度回调
// 支持检查点验证
```

## 使用示例

### 示例1: 咨询类请求
```go
// 用户请求: "对本数据源提出一些分析建议"

// 1. 分类请求
classifier := NewRequestClassifier(logger)
requestType := classifier.ClassifyRequest(userQuery, dataSourceInfo)
// 返回: RequestTypeConsultation

// 2. 创建计划
plan := planner.createConsultationPlan()
// 步骤: [get_data_source_context (basic)]

// 3. 验证计划
validator := NewExecutionValidator(logger)
plan, warnings := validator.ValidatePlan(plan, requestType)

// 4. 执行计划
// 只需1次工具调用，获取基础Schema
// 然后基于Schema生成分析建议
```

### 示例2: 多步骤分析
```go
// 用户请求: "对本数据源进行全面分析"

// 1. 分类请求
requestType := classifier.ClassifyRequest(userQuery, dataSourceInfo)
// 返回: RequestTypeMultiStepAnalysis

// 2. 创建计划
plan := planner.createMultiStepPlan()
// 步骤: [
//   1. get_data_source_context (detailed)
//   2. execute_sql (aggregation) [checkpoint]
//   3. execute_sql (join)
//   4. python_executor (visualization)
// ]

// 3. 验证计划
plan, warnings := validator.ValidatePlan(plan, requestType)

// 4. 执行计划
executor := NewStepExecutor(planner, validator, 3, logger)
result, err := executor.ExecuteSteps(ctx, plan, onProgress)
// 分步执行，支持中间结果反馈
```

### 示例3: Schema缓存
```go
// 创建Schema管理器
schemaManager := NewSchemaManager(dsService, 30*time.Minute, logger)

// 第一个请求
schema1, cached1, _ := schemaManager.GetSchema(ctx, dsID, SchemaLevelBasic)
// cached1 = false (新获取)

// 第二个请求（同一会话）
schema2, cached2, _ := schemaManager.GetSchema(ctx, dsID, SchemaLevelBasic)
// cached2 = true (使用缓存)

// 数据源结构变化
schemaManager.InvalidateCache(dsID)

// 第三个请求
schema3, cached3, _ := schemaManager.GetSchema(ctx, dsID, SchemaLevelBasic)
// cached3 = false (重新获取)
```

## 关键常量

### RequestType
```go
RequestTypeTrivial          // 无需工具调用
RequestTypeSimple           // 1次工具调用
RequestTypeDataQuery        // 数据查询
RequestTypeVisualization    // 可视化
RequestTypeCalculation      // 计算
RequestTypeWebSearch        // 网络搜索
RequestTypeConsultation     // ✨ 咨询建议
RequestTypeMultiStepAnalysis // ✨ 多步骤分析
```

### SchemaLevel
```go
SchemaLevelBasic    // 只有表名和描述
SchemaLevelDetailed // 完整字段信息
```

### StepAction
```go
StepActionProceed // 继续下一步
StepActionModify  // 修改并重试
StepActionSkip    // 跳过此步
StepActionAbort   // 中止执行
```

## 性能指标

### 咨询类请求
- **之前**: 2次工具调用 (get_data_source_context × 2)
- **现在**: 1次工具调用 (get_data_source_context × 1)
- **改进**: 50% 减少

### Schema缓存
- **缓存命中率**: 取决于会话中的重复请求
- **TTL**: 30分钟
- **缓存大小**: 基础Schema < 详细Schema

### 执行计划精确性
- **偏差评分**: 0.0 (完美) ~ 1.0 (完全不同)
- **警告阈值**: > 0.5

## 日志示例

```
[CLASSIFIER] Request classified as: consultation
[SCHEMA-MANAGER] Cache hit for ds-123 (level: basic)
[VALIDATOR] Plan validated: 1 steps, 0 warnings
[STEP-EXECUTOR] Executing step 1/4: 获取完整数据结构
[STEP-EXECUTOR] Step 1 result: success=true, action=proceed
[VALIDATOR] Execution metrics: planned=4, actual=4, deviation=0.00
```

## 集成检查清单

- [x] RequestClassifier 实现
- [x] SchemaManager 实现
- [x] StepExecutor 实现
- [x] ExecutionValidator 实现
- [x] AnalysisPlanner 增强
- [x] EinoService 集成
- [x] 代码编译通过
- [ ] 单元测试 (可选)
- [ ] 属性测试 (可选)
- [ ] 集成测试 (可选)
- [ ] 性能测试 (可选)

## 故障排除

### 问题: 咨询请求仍然执行SQL
**原因**: RequestClassifier未正确识别咨询关键词
**解决**: 检查ConsultationPatterns中是否包含用户请求中的关键词

### 问题: Schema缓存未生效
**原因**: 缓存已过期（TTL > 30分钟）或被手动失效
**解决**: 检查缓存统计 `schemaManager.GetCacheStats()`

### 问题: 执行偏差过大
**原因**: 实际执行步骤与计划不匹配
**解决**: 检查执行日志，使用 `validator.LogDeviations()` 查看详细信息

## 相关文件

- 需求文档: `.kiro/specs/agent-analysis-optimization/requirements.md`
- 设计文档: `.kiro/specs/agent-analysis-optimization/design.md`
- 任务列表: `.kiro/specs/agent-analysis-optimization/tasks.md`
- 实现总结: `.kiro/specs/agent-analysis-optimization/IMPLEMENTATION_SUMMARY.md`

## 下一步

1. 编写单元测试验证各组件功能
2. 编写属性测试验证21个正确性属性
3. 编写集成测试验证端到端流程
4. 性能测试验证缓存和执行效率
5. 更新API文档和使用指南
