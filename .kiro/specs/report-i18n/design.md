# 报告生成系统多语言国际化 - 设计文档

## 1. 架构设计

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        用户界面                              │
│                    (语言设置: EN/ZH)                         │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   报告生成控制器                             │
│                 (app_report_generate.go)                     │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  PrepareReport()                                      │  │
│  │  - 获取当前语言设置                                   │  │
│  │  - 构建国际化数据摘要                                 │  │
│  │  - 调用LLM生成报告                                    │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    国际化层 (i18n)                           │
│                                                              │
│  ┌──────────────────┐  ┌──────────────────┐               │
│  │  translations    │  │  prompts.go      │               │
│  │  - en/zh keys    │  │  - LLM prompts   │               │
│  └──────────────────┘  └──────────────────┘               │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      LLM服务                                 │
│                  (根据语言选择提示词)                         │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    导出服务层                                │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │   PDF    │  │   Word   │  │  Excel   │  │   PPT    │  │
│  │ Service  │  │ Service  │  │ Service  │  │ Service  │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
│       │              │              │              │        │
│       └──────────────┴──────────────┴──────────────┘        │
│                      使用 i18n.T()                           │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 数据流

```
用户请求生成报告
    │
    ▼
获取当前语言设置 (i18n.GetLanguage())
    │
    ▼
构建国际化数据摘要
    ├─ 章节标题: i18n.T("report.section.xxx")
    ├─ 标签文本: i18n.T("report.xxx_label")
    └─ 用户数据: 保持原样
    │
    ▼
选择对应语言的LLM提示词
    ├─ English: i18n.GetReportSystemPrompt()
    └─ Chinese: i18n.GetReportSystemPrompt()
    │
    ▼
LLM生成报告内容
    │
    ▼
解析报告结构
    │
    ▼
导出服务渲染
    ├─ 页面元素: i18n.T("report.xxx")
    ├─ 章节标题: i18n.T("report.section.xxx")
    └─ 格式化文本: 根据语言格式化
    │
    ▼
保存文件对话框
    ├─ 标题: i18n.T("report.save_dialog_title")
    └─ 文件名: i18n.T("report.filename_prefix") + timestamp
```

## 2. 模块设计

### 2.1 国际化提示词模块 (i18n/prompts.go)

**职责**: 管理LLM提示词的多语言版本

**接口设计**:
```go
package i18n

// GetReportSystemPrompt 获取报告生成的系统提示词
func GetReportSystemPrompt() string

// GetReportUserPromptTemplate 获取用户提示词模板
func GetReportUserPromptTemplate() string

// GetDataSummaryTemplate 获取数据摘要模板
func GetDataSummaryTemplate(section string) string
```

**数据结构**:
```go
var reportSystemPrompts = map[Language]string{
    English: "...",
    Chinese: "...",
}

var reportUserPromptTemplates = map[Language]string{
    English: "...",
    Chinese: "...",
}

var dataSummaryTemplates = map[Language]map[string]string{
    English: {
        "user_request": "## Analysis Request",
        "data_source": "## Data Source",
        // ...
    },
    Chinese: {
        "user_request": "## 用户分析请求",
        "data_source": "## 数据源",
        // ...
    },
}
```

### 2.2 报告生成控制器 (app_report_generate.go)

**修改点**:
1. `PrepareReport()`: 添加语言检测
2. `callLLMForReport()`: 使用国际化提示词
3. `buildDataSummary()`: 使用国际化章节标题
4. `doExportPDF/Word()`: 使用国际化对话框标题

**关键函数修改**:
```go
func (a *App) callLLMForReport(dataSummary string, userRequest string) (string, error) {
    // 使用国际化提示词
    systemPrompt := i18n.GetReportSystemPrompt()
    userPromptTemplate := i18n.GetReportUserPromptTemplate()
    userPrompt := fmt.Sprintf(userPromptTemplate, dataSummary)
    
    // ... LLM调用
}

func buildDataSummary(req ReportGenerateRequest) string {
    var sb strings.Builder
    
    // 使用国际化章节标题
    sb.WriteString(i18n.GetDataSummaryTemplate("user_request") + "\n")
    sb.WriteString(req.UserRequest)
    sb.WriteString("\n\n")
    
    sb.WriteString(i18n.GetDataSummaryTemplate("data_source") + "\n")
    // ...
}
```

### 2.3 PDF导出服务 (pdf_gopdf_service.go)

**修改点**:
1. `addSectionTitle()`: 接受国际化标题
2. `addCoverPage()`: 使用国际化标签
3. `addMetricsSection()`: 使用国际化章节标题
4. `addChartsSection()`: 使用国际化图表标签
5. `addTableSection()`: 使用国际化表格信息
6. `addPageFooters()`: 使用国际化页脚

**关键函数修改**:
```go
func (s *GopdfService) addMetricsSection(pdf *gopdf.GoPdf, metrics []MetricData, fontName string) {
    // 使用国际化章节标题
    y := s.addSectionTitle(pdf, i18n.T("report.section.key_indicators"), fontName)
    // ...
}

func (s *GopdfService) addCoverPage(pdf *gopdf.GoPdf, title string, dataSourceName string, userRequest string, fontName string) {
    // 使用国际化标签
    if dataSourceName != "" {
        dsText := i18n.T("report.data_source_label") + dataSourceName
        // ...
    }
    
    if userRequest != "" {
        labelText := i18n.T("report.analysis_request_label")
        // ...
    }
    
    // 使用国际化时间格式
    timestamp := formatTimestamp(time.Now())
    timeText := i18n.T("report.generated_time_label") + timestamp
    // ...
}
```

## 3. 翻译键设计

### 3.1 翻译键命名规范

**格式**: `report.{category}.{item}`

**分类**:
- `report.section.*`: 报告章节标题
- `report.*_label`: 标签文本
- `report.*_message`: 消息文本
- `report.error.*`: 错误消息

### 3.2 完整翻译键列表

```go
// 对话框和文件名
"report.save_dialog_title"
"report.filename_prefix"

// 错误消息
"report.llm_not_initialized"
"report.generation_failed"
"report.data_expired"
"report.word_generation_failed"
"report.pdf_generation_failed"
"report.excel_generation_failed"
"report.ppt_generation_failed"
"report.no_content"

// 报告章节
"report.section.user_request"
"report.section.data_source"
"report.section.key_metrics"
"report.section.insights"
"report.section.data_table"
"report.section.multiple_tables"
"report.section.charts"
"report.section.key_indicators"
"report.section.data_visualization"
"report.section.data_tables"

// 标签文本
"report.data_source_label"
"report.analysis_request_label"
"report.generated_time_label"
"report.chart_label"
"report.total_rows"
"report.showing_columns"
"report.category_label"

// 页脚
"report.footer_text"

// 提示词模板键（内部使用）
"report.prompt.system"
"report.prompt.user_template"
```

## 4. LLM提示词设计

### 4.1 英文提示词结构

```
You are a professional data analysis report formatting expert.

【CRITICAL RULES】
- All required data is provided below
- Output report body directly, starting with # title
- Forbidden to output transitional phrases
- Task fails if non-report content is output

Core Principles:
- Use original text from "Analysis Insights"
- Forbidden to rewrite or speculate
- Job is formatting, not rewriting
- Distribute content across sections
- Preserve existing structure if present

Report Format:
1. First line: # Title (max 20 words)
2. Use ## for sections
3. Structure:
   - ## Analysis Background and Objectives
   - ## Data Overview
   - ## Key Metrics Analysis
   - ## In-depth Data Analysis
   - ## Key Findings and Insights
   - ## Conclusions and Recommendations
4. No Markdown tables (system appends them)
5. Reference data points, don't copy tables
6. No added data or conclusions
7. No placeholders
```

### 4.2 中文提示词结构

```
你是一位专业的数据分析报告排版专家。

【最重要的规则】
- 所有需要的数据已经完整提供在下方
- 必须直接输出报告正文，第一行就是 # 标题
- 严禁输出任何过渡语、解释、准备说明
- 如果你输出了任何非报告正文的内容，视为任务失败

核心原则：
- 报告正文必须严格使用提供的"分析洞察（AI分析结果）"中的原始文字内容
- 严禁改写、缩减或臆测任何不在提供数据中的内容
- 你的工作是排版和组织，不是重新撰写
- 将原始分析文字按照报告结构合理分配到各章节中
- 如果原始分析文字已经包含了章节标题，应保留其结构

报告格式要求：
1. 第一行必须是报告标题，使用一级标题格式（# 标题），标题应简洁概括分析主题（不超过20个字）
2. 使用 Markdown 二级标题（## 标题）分节
3. 报告结构：
   - ## 分析背景与目的
   - ## 数据概况
   - ## 关键指标分析
   - ## 深度数据分析
   - ## 关键发现与洞察
   - ## 结论与建议
4. 不要在报告中生成 Markdown 表格，数据表格会由系统自动附加在报告末尾
5. 可以在文字中引用数据表中的关键数据点，但不要试图复制整个表格
6. 不要添加原始分析中没有的数据或结论
7. 不要自行创建数据表格或用占位符替代真实内容
```

## 5. 日期时间格式化

### 5.1 格式化函数设计

```go
// formatTimestamp 根据当前语言格式化时间戳
func formatTimestamp(t time.Time) string {
    switch i18n.GetLanguage() {
    case i18n.English:
        return t.Format("January 02, 2006 15:04:05")
    case i18n.Chinese:
        return t.Format("2006年01月02日 15:04:05")
    default:
        return t.Format("2006-01-02 15:04:05")
    }
}

// formatDate 根据当前语言格式化日期
func formatDate(t time.Time) string {
    switch i18n.GetLanguage() {
    case i18n.English:
        return t.Format("January 02, 2006")
    case i18n.Chinese:
        return t.Format("2006年01月02日")
    default:
        return t.Format("2006-01-02")
    }
}
```

## 6. 错误处理设计

### 6.1 错误消息国际化

**原则**:
- 所有用户可见的错误消息必须国际化
- 保留技术细节（如堆栈跟踪）为英文
- 使用参数化错误消息

**示例**:
```go
// 修改前
return fmt.Errorf("报告生成失败: %v", err)

// 修改后
return fmt.Errorf(i18n.T("report.generation_failed", err.Error()))
```

### 6.2 错误恢复策略

- LLM调用失败：返回友好的错误消息，建议用户检查API配置
- 字体加载失败：尝试备用字体，最后使用系统默认字体
- 文件保存失败：提示用户检查权限和磁盘空间

## 7. 测试设计

### 7.1 单元测试

**测试用例**:
1. `TestGetReportSystemPrompt`: 测试获取不同语言的提示词
2. `TestBuildDataSummary`: 测试数据摘要构建
3. `TestFormatTimestamp`: 测试时间格式化
4. `TestTranslationKeys`: 测试所有翻译键存在

### 7.2 集成测试

**测试场景**:
1. 英文环境生成PDF报告
2. 中文环境生成PDF报告
3. 语言切换后生成报告
4. 多格式导出测试
5. 错误场景测试

### 7.3 端到端测试

**测试流程**:
1. 用户设置语言为English
2. 创建分析请求
3. 生成报告
4. 验证报告内容为英文
5. 切换语言为简体中文
6. 生成报告
7. 验证报告内容为中文

## 8. 性能优化

### 8.1 翻译缓存

- 翻译在启动时加载到内存
- 使用map实现O(1)查找
- 避免重复翻译查找

### 8.2 提示词优化

- 控制提示词长度，避免token浪费
- 使用简洁明确的指令
- 避免冗余信息

### 8.3 报告生成优化

- 保持现有的报告缓存机制
- 避免重复的LLM调用
- 优化PDF渲染性能

## 9. 安全考虑

### 9.1 输入验证

- 验证用户请求内容
- 过滤特殊字符
- 限制输入长度

### 9.2 输出清理

- 清理LLM输出中的敏感信息
- 验证报告结构
- 防止注入攻击

## 10. 扩展性设计

### 10.1 添加新语言

**步骤**:
1. 在 `i18n.go` 中添加新语言常量
2. 在 `translations_xx.go` 中添加翻译
3. 在 `prompts.go` 中添加提示词
4. 更新日期格式化函数
5. 测试新语言

### 10.2 添加新导出格式

**步骤**:
1. 创建新的导出服务
2. 使用 `i18n.T()` 国际化所有文本
3. 实现统一的接口
4. 添加测试

## 11. 部署考虑

### 11.1 向后兼容

- 保持现有API接口不变
- 默认语言为English（或根据系统语言）
- 现有报告缓存仍然有效

### 11.2 配置管理

- 语言设置存储在用户配置中
- 支持环境变量覆盖
- 提供命令行参数

### 11.3 监控和日志

- 记录语言切换事件
- 记录报告生成成功/失败
- 记录LLM调用统计

---

**文档版本**: 1.0  
**创建日期**: 2024-02-08  
**最后更新**: 2024-02-08  
**状态**: 待审批
