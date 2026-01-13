# 会话文件管理系统

## 概述

RapidBI 现在支持将分析过程中生成的所有数据文件（图片、CSV等）自动保存到会话目录中，方便用户查看、管理和复用。

## 核心特性

### 1. 自动文件保存

在分析过程中生成的所有文件会自动保存到会话目录：
- **图片文件** - Python 生成的图表（chart.png）
- **CSV 文件** - 导出的数据文件
- **其他数据文件** - 任何由分析产生的文件

### 2. 文件组织结构

```
{DATA_DIR}/sessions/
├── {thread_id_1}/
│   ├── history.json          # 会话历史
│   └── files/                # 生成的文件目录
│       ├── chart.png         # 图表
│       ├── chart_1.png       # 重复文件会自动编号
│       ├── result.csv        # CSV 数据
│       └── rfm_analysis.csv  # 分析结果
├── {thread_id_2}/
│   ├── history.json
│   └── files/
│       └── ...
```

### 3. 文件跟踪

每个会话会跟踪所有生成的文件：

```json
{
  "id": "1705123456789000000",
  "title": "销售数据分析",
  "data_source_id": "ds_001",
  "created_at": 1705123456,
  "messages": [...],
  "files": [
    {
      "name": "chart.png",
      "path": "files/chart.png",
      "type": "image",
      "size": 45678,
      "created_at": 1705123789
    },
    {
      "name": "result.csv",
      "path": "files/result.csv",
      "type": "csv",
      "size": 12345,
      "created_at": 1705123790
    }
  ]
}
```

## 使用示例

### Python 代码生成图表

当用户要求生成图表时：

```python
import matplotlib.pyplot as plt
import pandas as pd

# 分析代码...
data = {'Category': ['A', 'B', 'C'], 'Value': [10, 20, 15]}
df = pd.DataFrame(data)

# 生成图表
plt.figure(figsize=(10, 6))
plt.bar(df['Category'], df['Value'])
plt.title('Category Analysis')
plt.savefig('chart.png')  # 会自动保存到会话目录
```

系统会自动：
1. 保存 `chart.png` 到 `{session_dir}/files/chart.png`
2. 注册文件到会话记录
3. 在聊天响应中显示图表

输出示例：
```
📊 **Chart saved:** `files/chart.png`
![Chart](data:image/png;base64,...)
```

### Python 代码生成 CSV

```python
import pandas as pd

# RFM 分析
rfm = df.groupby('CustomerID').agg({...})

# 保存结果
rfm.to_csv('rfm_analysis.csv', index=True)
```

输出示例：
```
📊 Generated Data Files:
- 📁 **rfm_analysis.csv** (saved to session)
  [📥 Download](data:text/csv;base64,...)

  Preview (first 10 rows):
  ```csv
  CustomerID,R,F,M
  C001,15,5,1250.50
  C002,30,3,890.20
  ...
  ```
```

## API 接口

### 1. 获取会话文件列表

```javascript
// 前端调用
const files = await window.go.main.App.GetSessionFiles(threadID);

// 返回
[
  {
    "name": "chart.png",
    "path": "files/chart.png",
    "type": "image",
    "size": 45678,
    "created_at": 1705123789
  },
  {
    "name": "result.csv",
    "path": "files/result.csv",
    "type": "csv",
    "size": 12345,
    "created_at": 1705123790
  }
]
```

### 2. 获取文件路径

```javascript
const filePath = await window.go.main.App.GetSessionFilePath(threadID, "chart.png");
// 返回: "C:/Users/.../rapidbi/sessions/1705123456789000000/files/chart.png"
```

### 3. 打开文件

```javascript
// 在默认应用中打开文件
await window.go.main.App.OpenSessionFile(threadID, "chart.png");
```

### 4. 删除文件

```javascript
// 删除会话文件
await window.go.main.App.DeleteSessionFile(threadID, "chart.png");
```

## 实现细节

### 1. ChatThread 结构

添加了 `Files` 字段来跟踪生成的文件：

```go
type ChatThread struct {
    ID           string        `json:"id"`
    Title        string        `json:"title"`
    DataSourceID string        `json:"data_source_id"`
    CreatedAt    int64         `json:"created_at"`
    Messages     []ChatMessage `json:"messages"`
    Files        []SessionFile `json:"files,omitempty"` // 新增
}

type SessionFile struct {
    Name      string `json:"name"`       // 文件名
    Path      string `json:"path"`       // 相对路径
    Type      string `json:"type"`       // 文件类型
    Size      int64  `json:"size"`       // 文件大小
    CreatedAt int64  `json:"created_at"` // 创建时间
}
```

### 2. ChatService 新方法

```go
// 获取会话目录
func (s *ChatService) GetSessionDirectory(threadID string) string

// 获取会话文件目录
func (s *ChatService) GetSessionFilesDirectory(threadID string) string

// 添加会话文件记录
func (s *ChatService) AddSessionFile(threadID string, file SessionFile) error

// 获取会话文件列表
func (s *ChatService) GetSessionFiles(threadID string) ([]SessionFile, error)
```

### 3. PythonExecutorTool 增强

添加了会话目录支持：

```go
type PythonExecutorTool struct {
    pythonService   PythonExecutor
    cfg             config.Config
    pool            *PythonPool
    errorKnowledge  *ErrorKnowledge
    sessionDir      string                                          // 新增
    onFileSaved     func(fileName, fileType string, fileSize int64) // 新增
}

// 设置会话目录
func (t *PythonExecutorTool) SetSessionDirectory(dir string)

// 设置文件保存回调
func (t *PythonExecutorTool) SetFileSavedCallback(callback func(...))
```

### 4. 文件保存逻辑

当 Python 代码执行完成后：

1. 扫描临时工作目录，查找生成的文件
2. 将文件复制到会话目录（`{session_dir}/files/`）
3. 如果文件已存在，自动编号（`chart_1.png`, `chart_2.png`）
4. 触发回调，注册文件到 ChatThread
5. 在响应中显示文件信息和预览

## 文件类型识别

系统自动识别以下文件类型：

| 扩展名 | 类型标识 | 说明 |
|--------|----------|------|
| .png, .jpg, .jpeg, .gif | image | 图片文件 |
| .csv | csv | CSV 数据文件 |
| .json | data | JSON 数据 |
| .txt | text | 文本文件 |
| .xlsx | excel | Excel 文件 |
| 其他 | file | 通用文件 |

## 前端集成建议

### 1. 会话文件列表面板

在聊天界面旁边显示文件列表：

```jsx
function SessionFilesPanel({ threadID }) {
  const [files, setFiles] = useState([]);

  useEffect(() => {
    loadFiles();
  }, [threadID]);

  async function loadFiles() {
    const fileList = await window.go.main.App.GetSessionFiles(threadID);
    setFiles(fileList);
  }

  return (
    <div className="session-files-panel">
      <h3>Session Files</h3>
      {files.map(file => (
        <div key={file.name} className="file-item">
          <span className="file-icon">{getFileIcon(file.type)}</span>
          <span className="file-name">{file.name}</span>
          <span className="file-size">{formatSize(file.size)}</span>
          <button onClick={() => openFile(file.name)}>Open</button>
          <button onClick={() => deleteFile(file.name)}>Delete</button>
        </div>
      ))}
    </div>
  );
}
```

### 2. 文件链接

在消息中显示文件链接：

```jsx
// 解析消息内容，将文件引用转换为可点击链接
function renderMessage(content) {
  // 匹配: 📊 **Chart saved:** `files/chart.png`
  const filePattern = /📊 \*\*Chart saved:\*\* `files\/([^`]+)`/g;

  return content.replace(filePattern, (match, fileName) => {
    return `<a href="#" onclick="openSessionFile('${fileName}')">${fileName}</a>`;
  });
}
```

### 3. 文件预览

对于图片文件，可以直接在界面中预览：

```jsx
function FilePreview({ file }) {
  if (file.type === 'image') {
    return <img src={`file://${file.path}`} alt={file.name} />;
  } else if (file.type === 'csv') {
    return <CSVPreview filePath={file.path} />;
  }
  return <FileDownload file={file} />;
}
```

## 优势

### 1. 持久化存储
- 会话关闭后文件仍然保留
- 可以随时回顾之前的分析结果
- 方便导出和分享

### 2. 文件管理
- 所有文件集中在会话目录
- 易于查找和管理
- 自动处理文件名冲突

### 3. 数据复用
- 可以在后续分析中引用之前生成的文件
- 支持跨会话数据比较
- 方便构建分析报告

### 4. 用户体验
- 无需手动保存文件
- 文件自动关联到会话
- 一键打开和删除

## 最佳实践

### 1. 文件命名

在 Python 代码中使用描述性的文件名：

```python
# ✅ Good
plt.savefig('sales_trend_2024.png')
rfm.to_csv('customer_rfm_segments.csv')

# ❌ Bad (不清楚是什么)
plt.savefig('chart.png')
rfm.to_csv('data.csv')
```

### 2. 文件大小控制

避免生成过大的文件：

```python
# 限制 CSV 行数
df_sample = df.head(1000)
df_sample.to_csv('sample_data.csv')

# 压缩图片
plt.savefig('chart.png', dpi=100, bbox_inches='tight')
```

### 3. 清理临时文件

在分析完成后，可以建议用户删除不需要的文件：

```
✅ Analysis complete!
Generated files:
- chart.png (45 KB)
- intermediate_data.csv (2.3 MB) - You can delete this if not needed
- final_results.csv (120 KB)
```

## 未来改进

计划中的功能：

1. **文件标签** - 为文件添加标签，便于分类
2. **文件搜索** - 在所有会话中搜索特定类型的文件
3. **批量导出** - 一次性导出会话的所有文件
4. **文件版本** - 保留文件的多个版本
5. **文件预览** - 内置预览器，无需打开外部应用
6. **云存储集成** - 同步到云端存储
7. **分享功能** - 生成分享链接

## 故障排查

### Q: 文件没有保存到会话目录？

A: 检查：
- 会话目录是否正确传递到工具
- Python 代码是否正确保存了文件
- 文件保存回调是否触发
- 查看日志中的 `[SESSION]` 相关信息

### Q: 文件名有冲突怎么办？

A: 系统会自动处理：
- `chart.png` → `chart_1.png` → `chart_2.png`
- 不会覆盖已有文件

### Q: 如何清理旧文件？

A: 方法：
1. 使用 API 删除单个文件
2. 删除整个会话（包含所有文件）
3. 手动删除会话目录

### Q: 文件太大怎么办？

A: 建议：
- 在 Python 代码中限制文件大小
- 使用采样或压缩
- 只保存必要的数据

---

**🎉 享受便捷的会话文件管理！所有分析产物自动保存，随时可查！**
