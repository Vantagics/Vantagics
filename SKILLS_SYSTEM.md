# RapidBI Skills 插件系统

## 概述

RapidBI Skills 是一个基于 Anthropic Skills 理念的本地插件系统，用于将固定的分析步骤固化为可复用的知识和工具。

## 架构设计

### 核心组件

1. **Skill 定义** (`src/agent/templates/skill.go`)
   - `SkillManifest`: Skill元数据结构
   - `ConfigurableSkill`: 基于配置的Skill实现
   - 支持从JSON配置加载Skills

2. **Skill Manager** (`src/agent/templates/skill_manager.go`)
   - 负责Skills的加载、注册、管理
   - 支持启用/禁用Skills
   - 支持按分类和关键词搜索Skills

3. **Eino集成** (`src/agent/eino.go`)
   - SkillManager集成到EinoService
   - Skills自动注册到Template Registry
   - 通过关键词自动匹配和执行

4. **后端API** (`src/app.go`)
   - `GetSkills()`: 获取所有Skills
   - `GetEnabledSkills()`: 获取启用的Skills
   - `EnableSkill(id)`: 启用Skill
   - `DisableSkill(id)`: 禁用Skill
   - `ReloadSkills()`: 重新加载Skills

## Skill 结构

### 目录结构

```
RapidBI/
├── skills/                         # Skills目录
│   ├── cohort_analysis/           # 同期群分析
│   │   ├── skill.json            # Skill配置文件
│   │   ├── cohort_analysis.py    # Python代码
│   │   └── README.md             # Skill文档
│   ├── sales_funnel/              # 销售漏斗分析
│   │   ├── skill.json
│   │   ├── sales_funnel.py
│   │   └── README.md
│   └── rfm_analysis/              # RFM分析 (已有硬编码实现)
│       └── ...
```

### Skill配置文件 (skill.json)

```json
{
  "id": "cohort_analysis",
  "name": "同期群分析",
  "description": "分析不同时间段用户的留存和行为模式",
  "version": "1.0.0",
  "author": "RapidBI System",
  "category": "user_analytics",
  "keywords": [
    "cohort",
    "cohort analysis",
    "同期群",
    "留存分析"
  ],
  "required_columns": [
    "user_id",
    "date",
    "event"
  ],
  "tools": ["sql", "python"],
  "language": "python",
  "code_template": "cohort_analysis.py",
  "parameters": {
    "cohort_period": "month"
  },
  "enabled": true,
  "icon": "users",
  "tags": ["retention", "user_behavior"]
}
```

### 配置字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | Skill唯一标识符 |
| `name` | string | Skill名称（显示用） |
| `description` | string | Skill描述 |
| `version` | string | 版本号 |
| `author` | string | 作者 |
| `category` | string | 分类（user_analytics, sales_analytics等） |
| `keywords` | []string | 触发关键词（用于自动匹配） |
| `required_columns` | []string | 需要的数据列类型 |
| `tools` | []string | 使用的工具（sql, python） |
| `language` | string | 执行语言（python, sql, hybrid） |
| `code_template` | string | 代码文件名 |
| `parameters` | map | 可配置参数 |
| `enabled` | bool | 是否启用 |
| `icon` | string | 图标名称 |
| `tags` | []string | 标签 |

## 已实现的Skills

### 1. 同期群分析 (Cohort Analysis)
- **ID**: `cohort_analysis`
- **功能**: 分析用户留存和生命周期
- **输出**:
  - 留存率热力图
  - 留存曲线图
  - 详细数据CSV

### 2. 销售漏斗分析 (Sales Funnel)
- **ID**: `sales_funnel`
- **功能**: 分析销售转化漏斗
- **输出**:
  - 漏斗可视化
  - 各阶段转化率
  - 流失分析

### 3. RFM分析 (已有)
- **ID**: `rfm`
- **功能**: 客户价值分层
- **实现**: 硬编码实现 (`src/agent/templates/rfm_template.go`)

## 使用方式

### 1. 自动触发

用户在对话中输入包含Skill关键词的查询，系统会自动检测并执行对应的Skill：

```
用户: "帮我做一下同期群分析"
系统: [自动检测到 cohort_analysis Skill]
     [执行分析并返回结果]
```

### 2. 系统工作流程

1. 用户输入查询
2. `DetectTemplate()` 检测关键词匹配
3. 调用 `CanExecute()` 检查数据是否满足要求
4. 执行 `Execute()` 运行分析
5. 返回结果（文本+图表+数据文件）

## 创建新的Skill

### 步骤1: 创建Skill目录

```bash
mkdir -p skills/my_analysis
```

### 步骤2: 创建skill.json

```json
{
  "id": "my_analysis",
  "name": "我的分析",
  "description": "自定义分析描述",
  "version": "1.0.0",
  "author": "Your Name",
  "category": "custom",
  "keywords": ["关键词1", "keyword2"],
  "required_columns": ["column_type1", "column_type2"],
  "tools": ["python"],
  "language": "python",
  "code_template": "my_analysis.py",
  "enabled": true,
  "icon": "chart",
  "tags": ["custom"]
}
```

### 步骤3: 创建Python代码

在 `skills/my_analysis/my_analysis.py` 中编写分析代码：

```python
import json
import pandas as pd
import matplotlib.pyplot as plt

# 数据会以 {{data}} 占位符注入
data_json = '''{{data}}'''

# 模板模式检测
if data_json == '{{data}}':
    print("This is a template")
    exit(0)

# 解析数据
df = pd.DataFrame(json.loads(data_json))

# 执行分析
print("分析结果...")

# 生成可视化
plt.figure()
# ... 绘图代码 ...
plt.savefig('result.png')
print("图表已保存: result.png")
```

### 步骤4: 重新加载Skills

系统启动时会自动加载，或调用 `ReloadSkills()` API手动重新加载。

## 占位符系统

Skills支持在代码中使用占位符，系统会自动替换：

### 数据占位符
- `{{data}}`: JSON格式的数据（从SQL查询获取）
- `{{table}}`: 表名
- `{{column_name}}`: 匹配到的列名

### 参数占位符
- `{{parameter_name}}`: 来自skill.json的parameters配置

示例：
```python
# skill.json中定义: "parameters": {"threshold": "0.7"}
# 代码中使用:
threshold = float('{{threshold}}')  # 会被替换为 0.7
```

## 技术特性

### 1. 自动列匹配
系统会自动找到最匹配required_columns的实际列名，无需硬编码列名。

### 2. 多种执行模式
- **Python**: 执行Python代码分析
- **SQL**: 执行SQL查询
- **Hybrid**: SQL+Python组合

### 3. 进度回调
Skills执行过程中可以报告进度，前端可以显示进度条。

### 4. 错误处理
集成了ErrorKnowledge系统，可以从历史错误中学习。

## 前端集成 (待实现)

### 预期功能
1. **Skills列表页面**
   - 显示所有可用Skills
   - 按分类筛选
   - 启用/禁用开关

2. **Skill详情**
   - 显示Skill元数据
   - 查看需求列
   - 查看示例

3. **快速触发**
   - 在聊天界面提供Skill快捷入口
   - 智能推荐适合当前数据源的Skills

## API参考

### GetSkills()
```go
skills, err := app.GetSkills()
// 返回所有Skills（包括禁用的）
```

### GetEnabledSkills()
```go
skills, err := app.GetEnabledSkills()
// 只返回启用的Skills
```

### EnableSkill(id)
```go
err := app.EnableSkill("cohort_analysis")
// 启用指定Skill
```

### DisableSkill(id)
```go
err := app.DisableSkill("cohort_analysis")
// 禁用指定Skill
```

### ReloadSkills()
```go
err := app.ReloadSkills()
// 从磁盘重新加载所有Skills
```

## 扩展方向

### 1. Skill Marketplace
- 共享和下载社区Skills
- 导入/导出Skills

### 2. 可视化配置
- 前端UI配置Skill
- 无需编写JSON

### 3. Skill组合
- 将多个Skills组成工作流
- Pipeline式执行

### 4. 参数化执行
- 前端提供参数输入界面
- 动态调整Skill行为

## 总结

RapidBI Skills插件系统提供了一个灵活、可扩展的框架，用于将常见的数据分析流程固化为可复用的知识。通过简单的JSON配置和Python代码，即可创建新的分析能力，并自动集成到系统的对话式AI中。

系统的核心优势：
- ✅ **低门槛**: JSON配置 + Python代码即可
- ✅ **自动化**: 关键词自动匹配，无需手动选择
- ✅ **灵活性**: 支持纯SQL、纯Python或混合模式
- ✅ **智能化**: 自动列匹配，适应不同数据结构
- ✅ **可维护**: 统一的Skill管理和版本控制
