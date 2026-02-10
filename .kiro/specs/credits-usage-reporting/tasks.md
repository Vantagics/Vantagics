# 实现计划：Credits 使用量上报与追踪

## 概述

按照服务器端 → 客户端 → 管理后台的顺序实现，确保每一步都可增量验证。Go 语言实现，属性测试使用 `testing/quick`。

## Tasks

- [x] 1. 服务器端数据库 Schema 变更与 ActivationData 扩展
  - [x] 1.1 在 `tools/license_server/main.go` 的 `initDB()` 中添加 migration
    - 添加 `db.Exec("ALTER TABLE licenses ADD COLUMN used_credits FLOAT DEFAULT 0")` migration（与现有 migration 风格一致）
    - 添加 `db.Exec("CREATE TABLE IF NOT EXISTS credits_usage_log (...)")` 建表语句，包含 id（自增主键）、sn（TEXT）、used_credits（FLOAT）、reported_at（DATETIME）、client_ip（TEXT）
    - _Requirements: 6.1, 6.2_

  - [x] 1.2 在服务器端 `ActivationData` 结构体中添加 `UsedCredits float64` 字段
    - 在 `tools/license_server/main.go` 第 ~203 行的 `ActivationData` 结构体中添加 `UsedCredits float64 \`json:"used_credits"\``
    - 在 `handleActivate` 的 SELECT 查询中增加 `COALESCE(used_credits, 0)` 列
    - 在 Scan 中增加对应变量，并赋值给 `activationData.UsedCredits`
    - _Requirements: 3.1, 3.2_

  - [x] 1.3 实现 `handleReportUsage` 处理函数并注册端点
    - 在 `tools/license_server/main.go` 中新增 `handleReportUsage` 函数
    - 设置 CORS 头（Access-Control-Allow-Origin: *, Allow-Methods: POST, OPTIONS, Allow-Headers: Content-Type）
    - OPTIONS 请求返回 200，非 POST 返回 405
    - 解析 `{sn, used_credits}` JSON 请求体，解析失败返回 `{success: false, code: "INVALID_REQUEST"}`
    - 验证 SN 存在于 licenses 表，不存在返回 `{success: false, code: "INVALID_SN"}`
    - 验证 used_credits >= 0，否则返回 `{success: false, code: "INVALID_VALUE"}`
    - `INSERT INTO credits_usage_log (sn, used_credits, reported_at, client_ip) VALUES (?, ?, datetime('now'), ?)`
    - `UPDATE licenses SET used_credits = MAX(used_credits, ?) WHERE sn = ?`
    - 返回 `{success: true}`
    - 在 `startAuthServer()` 中注册 `mux.HandleFunc("/report-usage", handleReportUsage)`
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

  - [ ]* 1.4 为 handleReportUsage 编写属性测试和单元测试
    - **Property 4: 服务器 max-update 语义**
    - **Property 5: 服务器日志插入完整性**
    - 单元测试：无效 SN、负数 credits、JSON 解析失败等边界情况
    - **Validates: Requirements 2.2, 2.3, 2.4, 2.5**

- [x] 2. Checkpoint - 确保服务器端编译通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 3. 客户端合并逻辑与上报功能
  - [x] 3.1 修改 `src/agent/license_client.go` 中 `Activate()` 的合并逻辑
    - 将现有代码 `if c.data != nil && c.data.UsedCredits > 0 { data.UsedCredits = c.data.UsedCredits }` 改为 `if c.data != nil { data.UsedCredits = math.Max(data.UsedCredits, c.data.UsedCredits) }`
    - 添加 `"math"` 包导入
    - _Requirements: 4.1, 4.2_

  - [x] 3.2 在 `LicenseClient` 中添加上报相关字段和方法
    - 在 `LicenseClient` 结构体中添加 `lastReportAt time.Time`、`reportTicker *time.Ticker`、`reportStopCh chan struct{}` 字段
    - 实现 `ReportUsage() error`：构建 POST 请求到 `c.serverURL + "/report-usage"`，发送 `{sn, used_credits}` JSON，成功后更新 `c.lastReportAt` 并调用 `SaveActivationData()`
    - 实现 `ShouldReportOnStartup() bool`：返回 `!c.lastReportAt.IsZero() && time.Since(c.lastReportAt) >= time.Hour`
    - 实现 `StartUsageReporting()`：创建 1 小时 ticker goroutine，每次 tick 检查 `IsCreditsMode() && GetTrustLevel() == "low"` 后调用 `ReportUsage()`
    - 实现 `StopUsageReporting()`：关闭 ticker 和 stopCh
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [x] 3.3 扩展 `SaveActivationData` / `LoadActivationData` 持久化 lastReportAt
    - 在 `SaveActivationData` 的 saveData 结构体中添加 `LastReportAt string \`json:"last_report_at"\``
    - Save 时写入 `c.lastReportAt.Format(time.RFC3339)`
    - 在 `LoadActivationData` 的 saveData 结构体中添加同样字段
    - Load 时解析并恢复 `c.lastReportAt`
    - _Requirements: 1.6_

  - [x] 3.4 在 `src/app.go` 中集成上报逻辑
    - 在 `startup()` 中 license 加载/激活成功后，检查 `ShouldReportOnStartup()` 并在满足上报条件时调用 `ReportUsage()`
    - 调用 `StartUsageReporting()` 启动定时上报
    - 在 `shutdown()` 中调用 `StopUsageReporting()`
    - _Requirements: 1.1, 1.2_

  - [ ]* 3.5 为客户端逻辑编写属性测试
    - **Property 1: 上报条件判断** — 生成随机 (credits_mode, trust_level) 组合验证
    - **Property 2: 启动补报时间判断** — 生成随机时间戳验证 ShouldReportOnStartup
    - **Property 3: 上报结果与持久化一致性** — 验证 lastReportAt 仅在成功时更新
    - **Property 7: 客户端合并取最大值** — 生成随机非负值对验证 math.Max 语义
    - **Validates: Requirements 1.1, 1.2, 1.4, 1.5, 1.6, 4.1**

- [x] 4. Checkpoint - 确保客户端编译通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 5. 管理后台使用记录查看
  - [x] 5.1 实现 `handleCreditsUsageLog` 处理函数并注册端点
    - 在 `tools/license_server/main.go` 中新增 `handleCreditsUsageLog` 函数
    - GET 请求，从查询参数获取 `sn`
    - 查询 `SELECT sn, used_credits, reported_at, client_ip FROM credits_usage_log WHERE sn=? ORDER BY reported_at DESC`
    - 返回 JSON 数组 `[{sn, used_credits, reported_at, client_ip}]`，无记录返回 `[]`
    - 在 `startManageServer()` 中注册 `mux.HandleFunc("/api/credits-usage-log", authMiddleware(handleCreditsUsageLog))`
    - _Requirements: 5.2, 5.3, 5.4_

  - [x] 5.2 在 `tools/license_server/templates/email_records.go` 中添加"使用记录"按钮和弹窗
    - 在操作按钮区（修改、分组、展期等按钮之后）添加"使用记录"按钮
    - 使用 `data-action="usage-log"` 和 `data-key` 属性
    - 在事件委托 switch 中添加 `case 'usage-log'` 处理
    - 实现 `showUsageLog(sn)` 函数：fetch `/api/credits-usage-log?sn=XXX`，用 `showModal()` 显示表格
    - 表格列：上报时间、已用量、客户端 IP
    - _Requirements: 5.1, 5.3_

  - [ ]* 5.3 为使用日志排序编写属性测试
    - **Property 8: 使用日志降序排列**
    - **Validates: Requirements 5.3**

- [x] 6. 最终 Checkpoint - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

## 备注

- 标记 `*` 的任务为可选测试任务，可跳过以加快 MVP 进度
- 每个任务引用了具体的需求编号以便追溯
- Checkpoint 确保增量验证
- 属性测试使用 Go `testing/quick`，每个属性至少 100 次迭代
- 单元测试使用 `httptest.NewServer` mock HTTP，数据库测试使用内存 SQLite
