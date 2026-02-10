# 需求文档：License Credits 支持

## 简介

为现有的授权系统添加 credits（积分）模式支持。当前系统仅支持按日限制分析次数的方式，本功能将新增一种一次性发放 credits 的授权模式。用户在 credits 模式下，每次分析消耗 1.5 个 credits，直到 credits 用完为止。前端"关于"对话框需要根据授权模式显示对应的用量信息。

## 术语表

- **License_System**：授权系统，负责管理用户的授权状态、分析限制和 credits 余额
- **LicenseClient**：后端授权客户端，`LicenseClient` 结构体，负责授权验证、分析计数和 credits 扣减
- **ActivationData**：激活数据结构体，包含授权配置信息（LLM 配置、分析限制、credits 等）
- **AboutModal**：前端"关于"对话框组件，显示授权状态和用量信息
- **Credits**：积分，一次性发放的分析配额，每次分析消耗固定数量
- **Daily_Limit_Mode**：每日限制模式，现有的按日限制分析次数的授权方式
- **Credits_Mode**：积分模式，新增的一次性发放 credits 的授权方式
- **Analysis**：分析操作，用户发起的一次数据分析请求

## 需求

### 需求 1：授权模式判定

**用户故事：** 作为系统管理员，我希望授权系统能区分每日限制模式和积分模式，以便对不同授权类型的用户采用不同的限制策略。

#### 验收标准

1. WHEN ActivationData 中 `total_credits` 大于 0，THE License_System SHALL 将该授权识别为 Credits_Mode
2. WHEN ActivationData 中 `total_credits` 等于 0 且 `daily_analysis` 大于 0，THE License_System SHALL 将该授权识别为 Daily_Limit_Mode
3. WHEN ActivationData 中 `total_credits` 等于 0 且 `daily_analysis` 等于 0，THE License_System SHALL 允许无限制分析

### 需求 2：Credits 数据持久化

**用户故事：** 作为用户，我希望我的 credits 使用记录在应用重启后仍然保留，以便不会因为重启而丢失已用 credits 的记录。

#### 验收标准

1. THE License_System SHALL 在 ActivationData 中存储 `total_credits`（总 credits，float64 类型）和 `used_credits`（已用 credits，float64 类型）字段
2. WHEN credits 发生变化时，THE LicenseClient SHALL 将更新后的数据持久化到本地加密文件
3. WHEN 应用启动并加载已保存的激活数据时，THE LicenseClient SHALL 恢复 `used_credits` 的值

### 需求 3：Credits 模式下的分析权限检查

**用户故事：** 作为用户，我希望在 credits 不足时系统阻止分析，以便我能及时了解 credits 已用完。

#### 验收标准

1. WHILE 授权处于 Credits_Mode，WHEN 用户发起分析请求，THE LicenseClient SHALL 检查剩余 credits 是否大于等于 1.5
2. IF 剩余 credits 不足 1.5，THEN THE LicenseClient SHALL 拒绝分析请求并返回包含当前剩余 credits 数量的提示消息
3. WHILE 授权处于 Credits_Mode，THE LicenseClient SHALL 跳过每日分析次数限制的检查

### 需求 4：Credits 扣减

**用户故事：** 作为用户，我希望每次成功分析后自动扣减 credits，以便 credits 余额准确反映实际使用情况。

#### 验收标准

1. WHEN 一次分析成功完成且授权处于 Credits_Mode，THE LicenseClient SHALL 从 `used_credits` 中增加 1.5
2. WHEN credits 扣减完成后，THE LicenseClient SHALL 立即将更新后的数据持久化到本地文件
3. THE LicenseClient SHALL 使用常量定义每次分析消耗的 credits 数量（1.5）

### 需求 5：Credits 状态查询

**用户故事：** 作为前端组件，我希望能获取 credits 相关的状态信息，以便在界面上展示给用户。

#### 验收标准

1. WHEN 前端调用 `GetActivationStatus` 时，THE License_System SHALL 在返回数据中包含 `total_credits`、`used_credits` 和 `credits_mode` 字段
2. WHEN 授权处于 Credits_Mode，THE License_System SHALL 将 `credits_mode` 设为 `true`
3. WHEN 授权不处于 Credits_Mode，THE License_System SHALL 将 `credits_mode` 设为 `false`

### 需求 6：前端 Credits 用量展示

**用户故事：** 作为用户，我希望在"关于"对话框中看到我的 credits 使用情况，以便了解还剩多少 credits 可用。

#### 验收标准

1. WHILE 授权处于 Credits_Mode，THE AboutModal SHALL 显示 credits 用量区域，包含已用 credits 和总 credits 数值
2. WHILE 授权处于 Credits_Mode，THE AboutModal SHALL 显示 credits 用量进度条
3. WHILE 授权处于 Credits_Mode，THE AboutModal SHALL 隐藏每日分析次数的用量展示
4. WHILE 授权处于 Daily_Limit_Mode，THE AboutModal SHALL 隐藏 credits 用量展示并继续显示每日分析次数
5. WHEN credits 已全部用完，THE AboutModal SHALL 将进度条显示为警告色（红色）

### 需求 7：Credits 数据从服务端获取

**用户故事：** 作为系统管理员，我希望 credits 的总量由服务端在激活或刷新时下发，以便集中管理不同用户的 credits 配额。

#### 验收标准

1. WHEN 授权激活成功且服务端返回的数据中包含 `total_credits` 字段，THE LicenseClient SHALL 将该值存入 ActivationData
2. WHEN 授权刷新成功且服务端返回新的 `total_credits` 值，THE LicenseClient SHALL 更新 ActivationData 中的 `total_credits`
3. WHEN 服务端返回的数据中不包含 `total_credits` 字段，THE LicenseClient SHALL 将 `total_credits` 默认设为 0
