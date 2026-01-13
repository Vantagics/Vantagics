# 分析重放功能使用指南

## 功能概述

分析重放功能允许您将有经验分析人员的数据分析过程固化为可重用的步骤文件，并在其他相似的数据源上重放这些分析步骤。这使得知识可以被团队成员共享和重复使用。

## 核心特性

### 1. 分析过程记录
- **自动记录工具调用**：记录分析过程中执行的所有SQL查询和Python代码
- **保存数据源结构**：记录原始数据源的表结构和字段信息
- **LLM对话保存**：保存完整的AI对话上下文

### 2. 智能字段匹配
分析重放时，系统会智能匹配源数据源和目标数据源的字段：

#### 第一阶段：本地匹配
- **精确匹配**：字段名完全相同
- **大小写不敏感匹配**：忽略大小写差异（如 `CustomerID` vs `customerid`）

#### 第二阶段：LLM智能匹配（当本地匹配失败时）
当有1-2个字段无法本地匹配时，系统会自动调用LLM进行智能匹配：
- **语义相似性**：识别语义相同的字段（如 `customer_name` vs `client_name`）
- **常见缩写**：处理缩写（如 `qty` vs `quantity`）
- **单复数变化**：处理单复数差异（如 `product` vs `products`）
- **命名规范差异**：处理不同命名规范（camelCase vs snake_case）

### 3. 代码自动适配
- **SQL查询适配**：自动替换表名和字段名
- **Python代码适配**：自动替换DataFrame中的列引用
- **正则表达式替换**：精确匹配和替换代码中的标识符

## 使用流程

### 步骤1：保存分析会话

在完成一次分析后，保存分析过程：

```javascript
// 前端调用示例
import { SaveSessionRecording } from '../../wailsjs/go/main/App';

// 保存当前会话的分析记录
const filePath = await SaveSessionRecording(
    threadId,           // 会话ID
    "销售趋势分析模板",   // 记录标题
    "分析月度销售趋势和客户行为"  // 记录描述
);

console.log("Recording saved:", filePath);
```

### 步骤2：查看可用的分析记录

```javascript
import { GetSessionRecordings } from '../../wailsjs/go/main/App';

const recordings = await GetSessionRecordings();
// 返回所有可用的分析记录列表
```

### 步骤3：重放分析

在目标数据源上重放已保存的分析：

```javascript
import { ReplayAnalysisRecording } from '../../wailsjs/go/main/App';

const result = await ReplayAnalysisRecording(
    recordingId,        // 记录ID
    targetSourceId,     // 目标数据源ID
    true,               // 是否自动修复字段差异
    2                   // 最大允许的字段差异数量
);

// 检查结果
if (result.success) {
    console.log(`成功执行 ${result.steps_executed} 个步骤`);
    console.log("生成的图表:", result.charts);
} else {
    console.log(`失败 ${result.steps_failed} 个步骤`);
    console.log("错误:", result.error_message);
}
```

## 数据结构

### AnalysisRecording（分析记录）
```typescript
{
    recording_id: string;           // 记录唯一ID
    title: string;                  // 记录标题
    description: string;            // 记录描述
    created_at: Date;              // 创建时间
    source_id: string;             // 原始数据源ID
    source_name: string;           // 原始数据源名称
    source_schema: TableSchema[];  // 原始表结构
    steps: AnalysisStep[];         // 分析步骤列表
    llm_conversation: ConversationTurn[]; // LLM对话历史
}
```

### AnalysisStep（分析步骤）
```typescript
{
    step_id: number;       // 步骤ID
    timestamp: Date;       // 时间戳
    tool_name: string;     // 工具名称（execute_sql 或 python_executor）
    description: string;   // 步骤描述
    input: string;         // SQL查询或Python代码
    output: string;        // 工具输出
    chart_type: string;    // 图表类型（如果生成了图表）
    chart_data: string;    // 图表数据
}
```

### ReplayResult（重放结果）
```typescript
{
    success: boolean;                  // 是否成功
    steps_executed: number;            // 成功执行的步骤数
    steps_failed: number;              // 失败的步骤数
    step_results: StepResult[];        // 每个步骤的详细结果
    field_mappings: TableMapping[];    // 应用的字段映射
    generated_files: string[];         // 生成的文件列表
    error_message: string;             // 错误信息
    charts: object[];                  // 生成的图表
}
```

## 最佳实践

### 1. 记录命名
- 使用清晰描述性的标题
- 在描述中说明分析的目的和适用场景
- 包含数据源的行业背景信息

### 2. 字段匹配配置
- **允许2个字段差异**：适用于大多数情况
- **允许1个字段差异**：用于要求精确匹配的场景
- **允许0个字段差异**：仅在表结构完全相同时使用

### 3. 重放前检查
在重放前，建议先检查：
1. 目标数据源的表结构
2. 字段名称和类型是否兼容
3. 数据量级是否相似

### 4. 错误处理
- 检查 `result.step_results` 以了解每个步骤的执行情况
- 对于失败的步骤，查看 `error_message` 了解原因
- 可以根据需要手动调整代码后重新保存记录

## 应用场景

### 场景1：每月例行报告
1. 第一个月手动创建完整的销售分析
2. 保存分析记录为"月度销售报告模板"
3. 之后每月只需在新数据上重放即可

### 场景2：跨区域分析
1. 在北京区域数据上进行深度分析
2. 保存分析记录
3. 在上海、深圳等其他区域数据上重放相同分析

### 场景3：新人培训
1. 资深分析师创建最佳实践分析流程
2. 保存为培训模板
3. 新人可以学习和重用这些模板

### 场景4：客户定制分析
1. 为A客户创建定制分析流程
2. 保存分析记录
3. 快速应用到其他相似客户

## 技术细节

### 代码替换机制

#### SQL代码替换
```sql
-- 原始查询
SELECT CustomerName, SUM(TotalAmount) as Revenue
FROM sales_data
GROUP BY CustomerName

-- 字段映射: CustomerName -> client_name, TotalAmount -> amount
-- 替换后
SELECT client_name, SUM(amount) as Revenue
FROM sales_data
GROUP BY client_name
```

#### Python代码替换
```python
# 原始代码
df['CustomerName'].value_counts()
df.groupby('CustomerName')['TotalAmount'].sum()

# 字段映射: CustomerName -> client_name, TotalAmount -> amount
# 替换后
df['client_name'].value_counts()
df.groupby('client_name')['amount'].sum()
```

### LLM智能匹配提示词
系统使用以下提示词请求LLM进行智能字段匹配：

```
You are a data schema matching expert. I need to map source field names to target field names.

Source fields (unmatched):
[未匹配的源字段列表]

Available target fields:
[可用的目标字段列表]

Please suggest the best matching target field for each source field. Consider:
1. Semantic similarity (e.g., "customer_name" matches "client_name")
2. Common abbreviations (e.g., "qty" matches "quantity")
3. Plural/singular variations (e.g., "products" matches "product")
4. Different naming conventions (camelCase, snake_case, etc.)

Return your answer as a JSON array of mappings...
```

## 限制和注意事项

1. **字段数量限制**：默认最多允许2个字段不匹配，超过则重放失败
2. **数据类型**：系统不检查字段数据类型，需要用户自行确保兼容性
3. **业务逻辑**：重放只能适配字段名，无法适配业务逻辑的差异
4. **性能考虑**：大量数据的重放可能需要较长时间
5. **LLM调用成本**：智能字段匹配会消耗LLM API调用额度

## 未来改进方向

1. **可视化编辑器**：提供UI界面编辑和调整记录的步骤
2. **参数化**：支持为记录添加可配置参数
3. **版本管理**：支持记录的版本控制和变更追踪
4. **协作功能**：支持记录的分享和团队协作
5. **自动测试**：在重放前自动测试兼容性
6. **增量更新**：支持只重放新数据的增量分析

## 故障排除

### 问题1：字段匹配失败
**症状**：重放时报告"too many unmatched fields"

**解决方案**：
1. 增加 `maxFieldDiff` 参数值
2. 检查目标数据源的表结构
3. 手动创建字段映射配置

### 问题2：SQL执行失败
**症状**：某些SQL步骤执行失败

**解决方案**：
1. 检查SQL语法是否与目标数据库兼容
2. 验证引用的表和字段是否存在
3. 查看 `step_results` 中的详细错误信息

### 问题3：Python代码执行失败
**症状**：Python步骤执行失败

**解决方案**：
1. 检查数据格式是否匹配
2. 验证所需的Python库是否已安装
3. 查看错误日志了解具体问题

## 总结

分析重放功能是一个强大的知识固化和复用工具，通过智能的字段匹配和代码适配，可以大大提高数据分析的效率和一致性。合理使用这个功能，可以：

- **节省时间**：避免重复编写相同的分析代码
- **确保质量**：使用经过验证的分析流程
- **知识传承**：将专家经验固化为可重用的模板
- **提高一致性**：在不同数据源上应用相同的分析标准
