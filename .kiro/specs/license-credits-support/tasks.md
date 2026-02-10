# 实现计划：License Credits 支持

## 概述

基于已有的授权系统，新增 credits 模式支持。改动涉及后端 `LicenseClient`、`App.GetActivationStatus`，以及前端 `AboutModal`。采用增量实现方式，先改数据结构，再改逻辑，最后改前端。

## 任务

- [x] 1. 扩展 ActivationData 数据结构和常量定义
  - [x] 1.1 在 `src/agent/license_client.go` 中为 `ActivationData` 添加 `TotalCredits float64` 和 `UsedCredits float64` 字段（JSON tag 分别为 `total_credits` 和 `used_credits`），并定义 `CreditsPerAnalysis = 1.5` 常量
    - _Requirements: 2.1, 4.3_
  - [x] 1.2 在 `SaveActivationData` 的 `saveData` 结构体中添加 `UsedCredits` 字段，保存时写入 `c.data.UsedCredits`
    - _Requirements: 2.2_
  - [x] 1.3 在 `LoadActivationData` 的 `saveData` 结构体中添加 `UsedCredits` 字段，加载后将值恢复到 `c.data.UsedCredits`
    - _Requirements: 2.3_

- [x] 2. 实现 credits 模式判定和查询方法
  - [x] 2.1 在 `LicenseClient` 中实现 `IsCreditsMode() bool` 方法：当 `c.data != nil && c.data.TotalCredits > 0` 时返回 `true`
    - _Requirements: 1.1, 1.2, 1.3_
  - [x] 2.2 在 `LicenseClient` 中实现 `GetCreditsStatus() (float64, float64, bool)` 方法：返回 `totalCredits, usedCredits, isCreditsMode`
    - _Requirements: 5.1_
  - [x] 2.3 编写属性测试 `TestPropertyModeDetection`
    - **Property 1: 模式判定一致性**
    - **Validates: Requirements 1.1, 1.2, 1.3**

- [x] 3. 修改 CanAnalyze 支持 credits 模式
  - [x] 3.1 在 `CanAnalyze()` 方法开头添加 credits 模式分支：当 `IsCreditsMode()` 为 true 时，检查 `TotalCredits - UsedCredits >= CreditsPerAnalysis`，不足时返回包含剩余 credits 的中文提示
    - _Requirements: 3.1, 3.2, 3.3_
  - [x] 3.2 编写属性测试 `TestPropertyCanAnalyzeCreditsThreshold`
    - **Property 3: CanAnalyze credits 阈值检查**
    - **Validates: Requirements 3.1, 3.2**
  - [x] 3.3 编写属性测试 `TestPropertyCreditsModeBypasses DailyLimit`
    - **Property 4: Credits 模式绕过每日限制**
    - **Validates: Requirements 3.3**

- [x] 4. 修改 IncrementAnalysis 支持 credits 扣减
  - [x] 4.1 在 `IncrementAnalysis()` 方法中添加 credits 模式分支：当 `IsCreditsMode()` 为 true 时，`c.data.UsedCredits += CreditsPerAnalysis`，并记录日志
    - _Requirements: 4.1, 4.2_
  - [x] 4.2 编写属性测试 `TestPropertyIncrementAnalysisCreditsDeduction`
    - **Property 5: IncrementAnalysis credits 扣减精度**
    - **Validates: Requirements 4.1**

- [x] 5. Checkpoint - 确保后端逻辑测试通过
  - 确保所有测试通过，如有问题请向用户确认。

- [x] 6. 修改 GetActivationStatus 返回 credits 信息
  - [x] 6.1 在 `src/app.go` 的 `GetActivationStatus()` 中调用 `GetCreditsStatus()` 获取 credits 信息，将 `total_credits`、`used_credits`、`credits_mode` 添加到返回的 map 中
    - _Requirements: 5.1, 5.2, 5.3_
  - [x] 6.2 编写属性测试 `TestPropertyGetActivationStatusCreditsConsistency`
    - **Property 6: GetActivationStatus credits 字段一致性**
    - **Validates: Requirements 5.1, 5.2, 5.3**

- [x] 7. 编写 credits 持久化往返测试
  - [x] 7.1 编写属性测试 `TestPropertyCreditsPersistenceRoundTrip`
    - **Property 2: Credits 持久化往返一致性**
    - **Validates: Requirements 2.2, 2.3**

- [x] 8. 修改前端 AboutModal 支持 credits 展示
  - [x] 8.1 在 `AboutModal.tsx` 的 `activationStatus` 状态类型中添加 `total_credits`、`used_credits`、`credits_mode` 字段，并在 `GetActivationStatus` 回调中读取这些值
    - _Requirements: 5.1_
  - [x] 8.2 在 AboutModal 中添加 credits 用量展示区域：当 `credits_mode === true` 时显示已用/总量数值和进度条，credits 用完时进度条显示红色；同时隐藏每日分析用量区域
    - _Requirements: 6.1, 6.2, 6.3, 6.5_
  - [x] 8.3 确保当 `credits_mode` 为 false 或未设置时，保持现有每日分析用量展示不变
    - _Requirements: 6.4_
  - [x] 8.4 编写前端单元测试：验证 credits 模式和每日限制模式下的条件渲染
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 9. Final checkpoint - 确保所有测试通过
  - 确保所有测试通过，如有问题请向用户确认。

## 备注

- 标记 `*` 的任务为可选任务，可跳过以加快 MVP 进度
- 每个任务引用了具体的需求编号以便追溯
- 属性测试验证通用正确性属性，单元测试验证具体示例和边界情况
- 服务端 `total_credits` 字段的下发由服务端团队负责，本实现仅处理客户端接收和使用
