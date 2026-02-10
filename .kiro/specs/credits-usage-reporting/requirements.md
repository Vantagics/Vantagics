# 需求文档：Credits 使用量上报与追踪

## 简介

本功能实现试用版客户端定时上报 Credits 使用量到授权服务器，服务器记录使用日志并在激活/刷新时下发已用量，客户端与服务器合并时取较大值。管理后台增加查看使用记录功能。

## 术语表

- **Client（客户端）**: VantageData 桌面应用程序，通过 `LicenseClient` 管理授权
- **Auth_Server（授权服务器）**: 运行在端口 6699 的授权服务，处理激活、SN 申请等请求
- **Management_Server（管理服务器）**: 运行在端口 8899 的管理后台，需要管理员认证
- **Credits_Usage_Log（使用日志表）**: 新增的数据库表，记录每次上报的 Credits 使用量
- **Trial_Client（试用版客户端）**: trust_level 为 "low" 的客户端实例
- **Used_Credits（已用量）**: 客户端或服务器记录的 Credits 累计消耗值
- **Upload_Interval（上报间隔）**: 两次上报之间的最小时间间隔，固定为 1 小时

## 需求

### 需求 1：客户端定时上报 Credits 使用量

**用户故事：** 作为系统管理员，我希望试用版客户端定时上报 Credits 使用量，以便服务器能追踪试用用户的资源消耗。

#### 验收标准

1. WHILE Client 处于 Credits 模式且 trust_level 为 "low"，THE Client SHALL 每隔 1 小时向 Auth_Server 上报当前 Used_Credits 值
2. WHEN Client 本次会话运行时间不足 1 小时即退出，THE Client SHALL 在下次启动时检查距上次上报是否已满 1 小时，若满足则立即上报
3. WHEN Client 上报 Credits 使用量，THE Client SHALL 发送 POST 请求到 Auth_Server 的 `/report-usage` 端点，包含 SN 和 Used_Credits 字段
4. IF Client 上报请求失败（网络错误或服务器返回非成功状态），THEN THE Client SHALL 记录错误日志并在下一个上报周期重试
5. WHEN Client 的 trust_level 为 "high"（正式版），THE Client SHALL 跳过 Credits 使用量上报逻辑
6. WHEN Client 上报成功，THE Client SHALL 将上报时间持久化到本地存储，作为下次上报间隔的计算基准

### 需求 2：服务器接收并记录 Credits 使用量

**用户故事：** 作为系统管理员，我希望授权服务器能接收并记录客户端上报的 Credits 使用量，以便追踪和审计。

#### 验收标准

1. THE Auth_Server SHALL 在端口 6699 上提供 `/report-usage` POST 端点，接收 JSON 格式的 `{sn, used_credits}` 请求体
2. WHEN Auth_Server 收到有效的上报请求，THE Auth_Server SHALL 在 Credits_Usage_Log 表中插入一条记录，包含 sn、used_credits、reported_at 和 client_ip 字段
3. WHEN Auth_Server 收到有效的上报请求，THE Auth_Server SHALL 更新 licenses 表中对应 SN 的 used_credits 列为上报值（若上报值大于当前值）
4. IF Auth_Server 收到的 SN 在 licenses 表中不存在，THEN THE Auth_Server SHALL 返回错误响应 `{success: false, code: "INVALID_SN"}`
5. IF Auth_Server 收到的 used_credits 值为负数，THEN THE Auth_Server SHALL 返回错误响应 `{success: false, code: "INVALID_VALUE"}`
6. WHEN Auth_Server 成功处理上报请求，THE Auth_Server SHALL 返回 `{success: true}` 响应

### 需求 3：服务器在激活/刷新时下发已用量

**用户故事：** 作为客户端用户，我希望在激活或刷新授权时获取服务器记录的已用量，以便本地数据与服务器保持同步。

#### 验收标准

1. WHEN Auth_Server 构建 ActivationData 响应，THE Auth_Server SHALL 从 licenses 表读取 used_credits 值并包含在 ActivationData 中
2. THE Auth_Server 的 ActivationData 结构体 SHALL 包含 `UsedCredits float64` 字段，JSON 标签为 `used_credits`

### 需求 4：客户端合并已用量逻辑

**用户故事：** 作为客户端用户，我希望在刷新授权时本地已用量与服务器已用量正确合并，以防止数据丢失或回退。

#### 验收标准

1. WHEN Client 从 Auth_Server 收到 ActivationData，THE Client SHALL 取 max(服务器 Used_Credits, 本地 Used_Credits) 作为当前 Used_Credits 值
2. WHEN Client 首次激活（本地无已有数据），THE Client SHALL 直接使用服务器下发的 Used_Credits 值

### 需求 5：管理后台查看使用记录

**用户故事：** 作为系统管理员，我希望在邮箱申请记录面板中查看每个序列号的 Credits 使用记录，以便监控和审计。

#### 验收标准

1. WHEN 管理员在邮箱申请记录面板中点击某个 SN 的"使用记录"按钮，THE Management_Server SHALL 显示一个弹窗，列出该 SN 的所有 Credits 使用日志
2. THE Management_Server SHALL 提供 `/api/credits-usage-log` GET 端点，接受 `sn` 查询参数，返回该 SN 的使用日志列表
3. WHEN 使用日志弹窗显示时，THE Management_Server SHALL 按 reported_at 降序排列记录，每条记录显示上报时间、已用量和客户端 IP
4. THE `/api/credits-usage-log` 端点 SHALL 需要管理员认证（通过 authMiddleware）

### 需求 6：数据库 Schema 变更

**用户故事：** 作为开发者，我希望数据库 Schema 支持 Credits 使用量的存储和查询。

#### 验收标准

1. THE Auth_Server SHALL 在 initDB 中通过 migration 为 licenses 表添加 `used_credits FLOAT DEFAULT 0` 列
2. THE Auth_Server SHALL 在 initDB 中创建 `credits_usage_log` 表，包含 id（自增主键）、sn（TEXT）、used_credits（FLOAT）、reported_at（DATETIME）、client_ip（TEXT）字段
