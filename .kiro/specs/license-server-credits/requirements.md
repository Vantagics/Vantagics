# 需求文档：授权服务器 Credits 支持

## 简介

为授权服务器（管理端）添加 credits 模式支持。当前授权服务器仅支持按"每日分析次数"限制的方式管理序列号，本功能将新增 credits（积分）模式。管理员在生成序列号时可以选择使用 credits 模式还是每日限制模式，并在管理界面中查看和修改 credits 配置。客户端的 credits 支持已在之前的 spec（`license-credits-support`）中实现，本 spec 聚焦于服务端数据存储、API 下发和管理界面的改动。

## 术语表

- **License_Server**：授权服务器，Go 应用程序，负责序列号管理、激活数据下发和管理界面
- **Admin_UI**：管理界面，运行在 8899 端口的 Web 管理后台，用于序列号的增删改查
- **License**：序列号记录，数据库 `licenses` 表中的一行数据
- **ActivationData**：激活数据结构体，服务端构建并加密下发给客户端的配置数据
- **Credits_Mode**：积分模式，`total_credits > 0` 时启用，客户端按 credits 消耗而非每日次数限制
- **Daily_Limit_Mode**：每日限制模式，现有的按日限制分析次数的授权方式
- **Batch_Create_Modal**：批量生成对话框，Admin_UI 中用于批量创建序列号的弹窗
- **Total_Credits**：总积分，分配给序列号的 credits 总量，float64 类型

## 需求

### 需求 1：数据库 Credits 字段

**用户故事：** 作为系统管理员，我希望数据库能存储序列号的 credits 配置，以便支持 credits 模式的序列号管理。

#### 验收标准

1. THE License_Server SHALL 在 `licenses` 表中包含 `total_credits` 列（FLOAT 类型，默认值 0）
2. WHEN License_Server 启动且 `licenses` 表缺少 `total_credits` 列时，THE License_Server SHALL 通过数据库迁移自动添加该列
3. THE License 结构体 SHALL 包含 `TotalCredits float64` 字段，JSON 标签为 `total_credits`

### 需求 2：ActivationData Credits 下发

**用户故事：** 作为系统管理员，我希望客户端激活时能收到 credits 配置，以便客户端根据 credits 模式运行。

#### 验收标准

1. THE ActivationData 结构体 SHALL 包含 `TotalCredits float64` 字段，JSON 标签为 `total_credits`
2. WHEN 客户端激活序列号时，THE License_Server SHALL 从数据库读取该序列号的 `total_credits` 值并填入 ActivationData
3. WHEN 序列号的 `total_credits` 为 0 时，THE License_Server SHALL 在 ActivationData 中将 `total_credits` 设为 0（客户端据此判定为 Daily_Limit_Mode）

### 需求 3：单个序列号创建支持 Credits

**用户故事：** 作为系统管理员，我希望创建单个序列号时能指定 credits 数量，以便为特定用户分配 credits 模式的授权。

#### 验收标准

1. WHEN 创建单个序列号的请求包含 `total_credits` 参数时，THE License_Server SHALL 将该值存入数据库的 `total_credits` 列
2. WHEN 创建单个序列号的请求未包含 `total_credits` 参数时，THE License_Server SHALL 将 `total_credits` 默认设为 0
3. IF `total_credits` 参数为负数，THEN THE License_Server SHALL 将其视为 0

### 需求 4：批量创建序列号支持 Credits

**用户故事：** 作为系统管理员，我希望批量生成序列号时能指定 credits 数量，以便高效地创建 credits 模式的授权。

#### 验收标准

1. WHEN 批量创建序列号的请求包含 `total_credits` 参数时，THE License_Server SHALL 将该值存入每个新建序列号的 `total_credits` 列
2. WHEN 批量创建序列号的请求未包含 `total_credits` 参数时，THE License_Server SHALL 将 `total_credits` 默认设为 0
3. IF `total_credits` 参数为负数，THEN THE License_Server SHALL 将其视为 0

### 需求 5：管理界面批量生成对话框 Credits 模式选择

**用户故事：** 作为系统管理员，我希望在批量生成对话框中选择 credits 模式或每日限制模式，以便根据需要创建不同类型的序列号。

#### 验收标准

1. THE Batch_Create_Modal SHALL 包含一个模式选择控件（单选按钮），选项为"每日限制"和"Credits"
2. WHEN 管理员选择"每日限制"模式时，THE Batch_Create_Modal SHALL 显示每日分析次数输入框并隐藏 credits 输入框
3. WHEN 管理员选择"Credits"模式时，THE Batch_Create_Modal SHALL 显示 credits 数量输入框并隐藏每日分析次数输入框
4. WHEN 管理员选择"Credits"模式并提交时，THE Batch_Create_Modal SHALL 将 `total_credits` 参数发送到批量创建 API，并将 `daily_analysis` 设为 0
5. WHEN 管理员选择"每日限制"模式并提交时，THE Batch_Create_Modal SHALL 将 `total_credits` 设为 0，并发送 `daily_analysis` 参数
6. THE Batch_Create_Modal SHALL 默认选中"每日限制"模式

### 需求 6：管理界面序列号列表 Credits 展示

**用户故事：** 作为系统管理员，我希望在序列号列表中看到 credits 信息，以便快速了解每个序列号的授权模式和配额。

#### 验收标准

1. WHEN 序列号的 `total_credits` 大于 0 时，THE Admin_UI SHALL 在序列号详情中显示 `Credits: X`（X 为 total_credits 值）替代每日分析次数的显示
2. WHEN 序列号的 `total_credits` 等于 0 时，THE Admin_UI SHALL 继续显示现有的每日分析次数信息（`每日分析: X次` 或 `无限`）
3. THE License_Server 的序列号搜索 API SHALL 在返回数据中包含 `total_credits` 字段

### 需求 7：管理界面设置 Credits

**用户故事：** 作为系统管理员，我希望能修改已有序列号的 credits 配置，以便灵活调整用户的 credits 配额。

#### 验收标准

1. THE Admin_UI SHALL 为每个序列号提供"设置 Credits"操作按钮
2. WHEN 管理员点击"设置 Credits"按钮时，THE Admin_UI SHALL 显示一个对话框，包含 credits 数量输入框和当前值
3. WHEN 管理员提交新的 credits 值时，THE License_Server SHALL 更新数据库中该序列号的 `total_credits` 列
4. IF 提交的 credits 值为负数，THEN THE License_Server SHALL 将其视为 0
5. WHEN credits 值设为 0 时，THE License_Server SHALL 将该序列号切换回 Daily_Limit_Mode（由客户端根据 `total_credits == 0` 判定）
