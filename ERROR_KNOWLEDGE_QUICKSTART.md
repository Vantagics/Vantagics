# 错误知识系统 - 快速开始指南

## 系统已自动启用 ✅

错误知识系统已经完全集成到 RapidBI 中，无需任何配置即可使用！

## 它是如何工作的？

### 1. 自动错误记录

当 AI 执行 SQL 或 Python 代码时遇到错误，系统会自动：
- 📝 记录错误类型、消息和上下文
- 🏷️ 提取错误特征标签
- 💾 保存到错误知识库 (`{DATA_DIR}/error_knowledge.json`)

### 2. 智能提示

下次遇到相似错误时，系统会自动：
- 🔍 查找历史相似错误
- 💡 显示已验证的解决方案
- ⚡ 加速问题解决

### 3. 自我修正记录

当 AI 成功修正错误后：
- ✅ 记录成功的修正方案
- 📊 更新成功率统计
- 🎯 为未来提供参考

## 实际案例

### 案例 1: SQL 列名错误

**第一次执行：**
```sql
SELECT customer_name FROM customers
```
```
❌ SQL Error: no such column: customer_name
```

**系统记录：**
```json
{
  "error_type": "sql",
  "tags": ["column_not_found"],
  "successful": false
}
```

**AI 自我修正：**
```sql
SELECT CustomerName FROM customers  -- 正确的列名
```

**系统更新记录：**
```json
{
  "error_type": "sql",
  "tags": ["column_not_found"],
  "solution": "使用正确的列名: CustomerName",
  "successful": true
}
```

**第二次遇到类似错误：**
```sql
SELECT product_name FROM products
```
```
❌ SQL Error: no such column: product_name

💡 Historical Solutions (from past errors):
1. Error: no such column: customer_name
   Solution: 使用正确的列名: CustomerName

⚠️ Consider these proven solutions before attempting a fix.
```

AI 看到这个提示后，会立即：
1. 调用 `get_data_source_context` 查看正确的列名
2. 使用正确的列名 `ProductName` 重写查询
3. 成功执行！

### 案例 2: Python KeyError

**第一次：**
```python
revenue_share = df['revenue_share']  # KeyError
```
```
❌ KeyError: 'revenue_share'
```

**系统记录并建议：**
```
💡 HINT: Column not calculated before access
Solution:
  total = df['total_revenue'].sum()
  df['revenue_share'] = df['total_revenue'] / total * 100
```

**下次遇到：**
```python
profit_share = df['profit_share']  # KeyError
```
```
❌ KeyError: 'profit_share'

💡 Historical Solutions (from past errors):
1. Error: KeyError: 'revenue_share'
   Solution: Calculate column before accessing:
     total = df['total_revenue'].sum()
     df['revenue_share'] = df['total_revenue'] / total * 100
```

AI 理解模式，自动修正为：
```python
# Calculate profit_share first
total_profit = df['profit'].sum()
df['profit_share'] = df['profit'] / total_profit * 100
profit_share = df['profit_share']  # Now it works!
```

## 查看错误知识库统计

### 通过前端 API（即将支持）

```javascript
// 调用 App 的 GetErrorKnowledgeSummary 方法
const summary = await window.go.main.App.GetErrorKnowledgeSummary();

console.log(summary);
/*
{
  "total_records": 42,
  "successful_count": 35,
  "success_rate": 83.3,
  "by_type": {
    "sql": 28,
    "python": 14
  },
  "recent_errors": [...]
}
*/
```

### 通过日志

错误知识系统会在日志中输出关键信息：

```
[INFO] Error knowledge system initialized
[ERROR-KNOWLEDGE] Recorded: sql → Corrected SQL (success: true)
[ERROR-KNOWLEDGE] Found similar past errors:
💡 Historical Solutions (from past errors):
1. Error: no such column: customer_name
   Solution: ...
```

## 数据存储

错误知识库存储在：
```
Windows: C:\Users\{用户名}\AppData\Roaming\rapidbi\error_knowledge.json
macOS: ~/Library/Application Support/rapidbi/error_knowledge.json
Linux: ~/.config/rapidbi/error_knowledge.json
```

文件格式：
```json
[
  {
    "id": "err_1705123456789000000",
    "timestamp": "2025-01-12T10:30:45Z",
    "error_type": "sql",
    "error_message": "no such column: customer_name",
    "context": "Executing SQL query (attempt 1/3): SELECT customer_name FROM...",
    "solution": "Corrected SQL:\nSELECT CustomerName FROM customers",
    "successful": true,
    "tags": ["column_not_found"]
  }
]
```

## 性能优化

### 自动清理
- 系统自动保留最近 100 条记录
- 旧记录会被自动清理，避免文件过大

### 字符截断
- 错误消息：最多 500 字符
- 上下文：最多 200 字符
- 解决方案：最多 500 字符

## 高级功能

### 1. 相似度匹配算法

系统使用基于标签的相似度匹配：

```go
// SQL 错误标签示例
"no such column: xxx" → tags: ["column_not_found"]
"syntax error near YEAR" → tags: ["syntax_error", "date_function"]
"GROUP BY error" → tags: ["aggregation"]

// Python 错误标签示例
"KeyError: 'xxx'" → tags: ["key_error"]
"TypeError: xxx" → tags: ["type_error"]
"ModuleNotFoundError: xxx" → tags: ["module_not_found"]
```

匹配规则：
1. 必须是相同的错误类型（sql/python）
2. 必须成功解决过（successful = true）
3. 至少有一个相同的标签

### 2. 提示格式化

系统将历史解决方案格式化为易读的提示：

```
💡 Historical Solutions (from past errors):
1. **Error:** no such column: customer_name
   **Solution:** Corrected SQL: SELECT CustomerName FROM customers

2. **Error:** syntax error near YEAR
   **Solution:** Use strftime('%Y', date_col) for SQLite

⚠️ Consider these proven solutions before attempting a fix.
```

## 最佳实践

### ✅ DO：
1. 让系统自动运行 - 无需手动干预
2. 查看日志了解系统学习进度
3. 信任历史解决方案提示

### ❌ DON'T：
1. 不要手动编辑 error_knowledge.json（系统自动管理）
2. 不要禁用错误记录功能
3. 不要忽略历史解决方案提示

## 故障排查

### Q: 系统没有记录错误？
A: 检查：
- EinoService 是否正确初始化
- 工具是否成功注入了 errorKnowledge
- 日志中是否有 "[ERROR-KNOWLEDGE]" 相关信息

### Q: 没有看到历史解决方案提示？
A: 可能原因：
- 这是首次遇到该类型错误（知识库为空）
- 错误标签不匹配（需要积累更多案例）
- 之前的错误都失败了（只显示成功的解决方案）

### Q: 如何清空错误知识库？
A: 删除文件：
```bash
# Windows
del %APPDATA%\rapidbi\error_knowledge.json

# macOS/Linux
rm ~/Library/Application\ Support/rapidbi/error_knowledge.json
# or
rm ~/.config/rapidbi/error_knowledge.json
```

## 未来功能预览

即将推出：
- 📊 **错误趋势分析** - 可视化错误类型和频率
- 🔄 **跨会话学习** - 在不同用户间共享知识
- 🎯 **自动应用修复** - 高置信度自动修正
- 🌐 **云端知识库** - 全局错误解决方案库
- 📈 **成功率排名** - 显示最有效的解决方案

## 支持

遇到问题或有建议？
- 查看日志文件获取详细信息
- 检查 ERROR_KNOWLEDGE_SYSTEM.md 了解系统架构
- 提交 Issue 或反馈

---

**🎉 享受智能错误处理，让 AI 从错误中学习！**
