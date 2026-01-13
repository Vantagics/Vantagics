# Error Knowledge System - 错误知识管理系统

## 概述

参考 Manus 的 "拥抱失败" 方法，RapidBI 现在实现了完整的错误知识管理系统，用于记录、学习和避免重复错误。

## 核心理念

> "错误恢复是真正代理行为最清晰的信号之一。" - Manus Team

我们的系统通过以下方式实现这一理念：

1. **记录所有错误** - 将每次失败都视为学习机会
2. **提取解决方案** - 保存成功的修复方法
3. **智能匹配** - 基于错误特征自动查找相似历史错误
4. **提供提示** - 在遇到类似错误时，自动建议已验证的解决方案

## 系统架构

### 1. ErrorKnowledge 核心组件 (`src/agent/error_knowledge.go`)

```go
type ErrorRecord struct {
    ID           string    // 唯一标识符
    Timestamp    time.Time // 发生时间
    ErrorType    string    // 错误类型: "sql", "python", "schema", "timeout"
    ErrorMessage string    // 原始错误信息
    Context      string    // 执行上下文
    Solution     string    // 解决方案
    Successful   bool      // 是否成功解决
    Tags         []string  // 用于相似性匹配的标签
}
```

### 2. 智能标签提取

系统自动从错误信息中提取标签，用于相似性匹配：

**SQL 错误模式：**
- `column_not_found` - 列不存在
- `table_not_found` - 表不存在
- `syntax_error` - 语法错误
- `ambiguous_column` - 列名歧义
- `division_zero` - 除零错误
- `date_function` - 日期函数错误
- `aggregation` - 聚合查询错误

**Python 错误模式：**
- `key_error` - 键错误
- `type_error` - 类型错误
- `value_error` - 值错误
- `index_error` - 索引错误
- `import_error` - 导入错误
- `module_not_found` - 模块未找到

### 3. 集成点

#### SQL Executor Tool
- **执行前**: 查询相似历史错误，提供解决方案提示
- **执行后**:
  - 失败时：记录错误和上下文
  - 重试成功：记录成功的修正 SQL

```go
// 示例：SQL 执行流程
1. 尝试执行 SQL
2. 如果失败：查询 error_knowledge.FindSimilarErrors("sql", errorMsg)
3. 显示历史解决方案提示给 LLM
4. 使用 SQL Planner 尝试自动修正
5. 如果修正成功：记录成功的解决方案
6. 如果最终失败：记录失败案例
```

#### Python Executor Tool
- **执行失败时**:
  - 查询历史相似错误
  - 添加解决方案提示到输出
  - 记录新的错误案例

```go
// 示例：Python 执行流程
1. 执行 Python 代码
2. 如果失败：查询相似错误
3. 在错误输出中附加历史解决方案
4. 记录错误和上下文
```

## 使用示例

### 场景 1：SQL 列名错误

**第一次遇到：**
```
Error: no such column: customer_name
→ 系统记录：错误类型=sql, 标签=[column_not_found], 成功=false
```

**LLM 自我修正：**
```
Error: no such column: customer_name
→ 系统记录：修正后的 SQL 使用了正确的列名 CustomerName
           错误类型=sql, 标签=[column_not_found], 成功=true
```

**第二次遇到相似错误：**
```
Error: no such column: product_name
→ 系统提示：
   💡 Historical Solutions (from past errors):
   1. Error: no such column: customer_name
      Solution: Corrected SQL: SELECT CustomerName FROM ...

   ⚠️ Consider these proven solutions before attempting a fix.
```

### 场景 2：Python KeyError

**第一次：**
```python
df['revenue_share']  # KeyError
→ 记录错误和上下文
```

**第二次遇到类似问题：**
```
💡 Historical Solutions:
1. Error: KeyError: 'revenue_share'
   Solution: Calculate column first:
   total = df['total_revenue'].sum()
   df['revenue_share'] = df['total_revenue'] / total * 100
```

## 存储位置

错误知识库存储在：
```
{DATA_CACHE_DIR}/error_knowledge.json
```

默认保留最近 100 条记录，自动清理旧记录。

## 查看错误知识库

系统提供 `GetErrorSummary()` 方法来查看统计信息：

```go
summary := errorKnowledge.GetErrorSummary()
// 输出：
// 📊 Error Knowledge Base: 42 records
// ✅ Successfully resolved: 35 (83%)
// By type:
//   - sql: 28
//   - python: 14
```

## 优势

### 1. 避免重复错误
通过查询历史记录，系统能快速识别并应用已验证的解决方案。

### 2. 加速调试
LLM 收到历史解决方案提示后，可以更快地定位和修复问题。

### 3. 知识积累
随着时间推移，系统积累了越来越多的有效解决方案，形成知识库。

### 4. 提升透明度
用户可以看到系统遇到了什么问题，如何解决的，建立信任。

### 5. 自我改进
系统通过记录成功率（Successful字段），可以评估不同解决方案的有效性。

## 未来改进方向

1. **更智能的相似度匹配**
   - 使用向量嵌入来匹配语义相似的错误
   - 考虑上下文相似性（如：相同的数据源、相同的查询模式）

2. **解决方案排名**
   - 根据历史成功率对解决方案进行排序
   - 优先推荐最有效的解决方案

3. **跨会话学习**
   - 在不同用户、不同会话间共享知识
   - 构建全局错误解决方案库

4. **自动应用修复**
   - 对于高置信度的匹配，自动应用历史解决方案
   - 减少人工干预

5. **可视化界面**
   - 添加前端界面展示错误趋势
   - 提供错误知识库的搜索和浏览功能

## 参考

本实现参考了 Manus 团队的 "拥抱失败" 方法论：
- 记录所有失败到 task_plan.md 风格的日志
- 将错误视为学习机会而非问题
- 通过历史经验避免重复犯错
