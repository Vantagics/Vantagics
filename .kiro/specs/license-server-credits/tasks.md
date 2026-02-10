# Implementation Plan: 授权服务器 Credits 支持

## Overview

在 `tools/license_server/main.go` 和 `tools/license_server/templates/licenses.go` 中添加 credits 模式支持。所有改动集中在这两个文件中，按照数据层→API层→UI层的顺序递增实现。

## Tasks

- [x] 1. 数据层：数据库迁移和结构体扩展
  - [x] 1.1 在 `License` 结构体中添加 `TotalCredits float64` 字段（JSON 标签 `total_credits`）
    - 文件：`tools/license_server/main.go`，约 125 行 License 结构体
    - _Requirements: 1.3_
  - [x] 1.2 在 `ActivationData` 结构体中添加 `TotalCredits float64` 字段（JSON 标签 `total_credits`）
    - 文件：`tools/license_server/main.go`，约 180 行 ActivationData 结构体
    - _Requirements: 2.1_
  - [x] 1.3 在数据库迁移代码中添加 `ALTER TABLE licenses ADD COLUMN total_credits FLOAT DEFAULT 0`
    - 文件：`tools/license_server/main.go`，约 370 行迁移代码块
    - _Requirements: 1.1, 1.2_
  - [ ]* 1.4 编写属性测试：结构体 JSON 序列化往返一致性
    - **Property 1: 结构体 JSON 序列化往返一致性**
    - **Validates: Requirements 1.3, 2.1**

- [x] 2. API层：创建和激活 API 支持 credits
  - [x] 2.1 修改 `handleCreateLicense`：请求结构体添加 `TotalCredits` 字段，负值校验，INSERT 语句增加 `total_credits` 列
    - 文件：`tools/license_server/main.go`，约 1270 行
    - _Requirements: 3.1, 3.2, 3.3_
  - [x] 2.2 修改 `handleBatchCreateLicense`：请求结构体添加 `TotalCredits` 字段，负值校验，INSERT 语句增加 `total_credits` 列
    - 文件：`tools/license_server/main.go`，约 1340 行
    - _Requirements: 4.1, 4.2, 4.3_
  - [x] 2.3 修改 `handleSearchLicenses`：SELECT 查询增加 `COALESCE(total_credits, 0)` 列，Scan 增加 `&l.TotalCredits`
    - 文件：`tools/license_server/main.go`，约 1748 行
    - _Requirements: 6.3_
  - [x] 2.4 修改 `handleActivate`：SELECT 查询增加 `COALESCE(total_credits, 0)` 列，Scan 增加 `&license.TotalCredits`，构建 ActivationData 时设置 `TotalCredits`
    - 文件：`tools/license_server/main.go`，约 3805 行
    - _Requirements: 2.2, 2.3_
  - [ ]* 2.5 编写属性测试：创建序列号后 total_credits 存储一致性
    - **Property 2: 创建序列号后 total_credits 存储一致性**
    - **Validates: Requirements 3.1, 4.1, 6.3**
  - [ ]* 2.6 编写属性测试：激活数据包含正确的 total_credits
    - **Property 3: 激活数据包含正确的 total_credits**
    - **Validates: Requirements 2.2**

- [x] 3. Checkpoint - 确保数据层和 API 层改动正确
  - 确保所有测试通过，如有问题请向用户确认。

- [x] 4. API层：新增 set-credits 端点
  - [x] 4.1 实现 `handleSetCredits` 函数：接受 `sn` 和 `total_credits` 参数，负值校验，UPDATE 数据库
    - 遵循 `handleSetDailyAnalysis` 的模式
    - 文件：`tools/license_server/main.go`
    - _Requirements: 7.3, 7.4, 7.5_
  - [x] 4.2 注册路由 `mux.HandleFunc("/api/licenses/set-credits", authMiddleware(handleSetCredits))`
    - 文件：`tools/license_server/main.go`，约 853 行路由注册区域
    - _Requirements: 7.3_
  - [ ]* 4.3 编写属性测试：设置 Credits 更新往返一致性
    - **Property 4: 设置 Credits 更新往返一致性**
    - **Validates: Requirements 7.3**

- [x] 5. UI层：批量生成对话框添加模式选择
  - [x] 5.1 修改 `showBatchCreate` 函数：在每日分析次数输入框之前添加模式选择单选按钮（"每日限制" / "Credits"），添加 credits 输入框（默认隐藏）
    - 文件：`tools/license_server/templates/licenses.go`
    - _Requirements: 5.1, 5.2, 5.3, 5.6_
  - [x] 5.2 添加 `toggleBatchMode` JS 函数：根据选中的模式显示/隐藏对应输入框
    - 文件：`tools/license_server/templates/licenses.go`
    - _Requirements: 5.2, 5.3_
  - [x] 5.3 修改 `doBatchCreate` 函数：根据选中模式设置 `total_credits` 和 `daily_analysis` 参数（互斥）
    - 文件：`tools/license_server/templates/licenses.go`
    - _Requirements: 5.4, 5.5_
  - [ ]* 5.4 编写属性测试：批量创建模式互斥性
    - **Property 5: 批量创建模式互斥性**
    - **Validates: Requirements 5.4, 5.5**

- [x] 6. UI层：序列号列表展示 credits 信息和操作按钮
  - [x] 6.1 修改 `loadLicenses` 函数中的序列号显示逻辑：当 `total_credits > 0` 时显示 `Credits: X`，否则显示现有的每日分析信息
    - 文件：`tools/license_server/templates/licenses.go`
    - _Requirements: 6.1, 6.2_
  - [x] 6.2 在序列号操作按钮区域添加"Credits"按钮，调用 `setCredits(sn, total_credits)`
    - 文件：`tools/license_server/templates/licenses.go`
    - _Requirements: 7.1_
  - [x] 6.3 添加 `setCredits` 和 `doSetCredits` JS 函数：显示设置 Credits 对话框，提交到 `/api/licenses/set-credits`
    - 文件：`tools/license_server/templates/licenses.go`
    - _Requirements: 7.1, 7.2, 7.3_
  - [ ]* 6.4 编写属性测试：序列号列表显示逻辑一致性
    - **Property 6: 序列号列表显示逻辑一致性**
    - **Validates: Requirements 6.1, 6.2**

- [x] 7. Final checkpoint - 确保所有改动正确
  - 确保所有测试通过，如有问题请向用户确认。

## Notes

- 所有改动集中在 `tools/license_server/main.go` 和 `tools/license_server/templates/licenses.go` 两个文件
- Tasks marked with `*` are optional and can be skipped for faster MVP
- 数据库迁移遵循现有的 `ALTER TABLE ADD COLUMN` 模式
- `handleSetCredits` 遵循现有的 `handleSetDailyAnalysis` 模式
- UI 改动遵循现有的 Go 模板字符串拼接模式
- Property tests 使用 Go `testing/quick` 包，每个属性至少 100 次迭代
