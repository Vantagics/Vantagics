# 报告生成系统国际化 - 快速开始指南

## 概述

本指南帮助你快速开始报告生成系统的多语言国际化改造。完整的规范文档位于 `.kiro/specs/report-i18n/`。

## 快速链接

- **需求文档**: `.kiro/specs/report-i18n/requirements.md`
- **设计文档**: `.kiro/specs/report-i18n/design.md`
- **任务列表**: `.kiro/specs/report-i18n/tasks.md`
- **问题分析**: `doc/REPORT_I18N_ANALYSIS.md`

## 第一步：创建国际化提示词模块

### 1. 创建文件

创建 `src/i18n/prompts.go`:

```go
package i18n

var reportSystemPrompts = map[Language]string{
	English: `You are a professional data analysis report formatting expert. Your sole task is to organize the analysis results provided below into a well-formatted formal report.

【CRITICAL RULES】
- All required data is provided below, no additional information needed
- Output the report body directly, starting with # title on the first line
- Strictly forbidden to output any transitional phrases, explanations, or preparatory statements
- If you output any non-report content, the task is considered failed

Core Principles:
- Report body must strictly use the original text from "Analysis Insights (AI Results)"
- Forbidden to rewrite, condense, or speculate on content not in the provided data
- Your job is formatting and organization, not rewriting
- Distribute original analysis text reasonably across report sections
- If original analysis text already contains section titles (##, ###), preserve the structure
- Key metrics and data table information can be referenced as supplementary content

Report Format Requirements:
1. First line must be report title using level-1 heading format (# Title), title should concisely summarize analysis topic (max 20 words)
2. Use Markdown level-2 headings (## Title) for sections
3. Report structure:
   - ## Analysis Background and Objectives: Briefly explain user's analysis request and goals (1-2 sentences)
   - ## Data Overview: Briefly describe data source information (1-2 sentences)
   - Then organize original "Analysis Insights" content completely into subsequent sections. If original content has structure, preserve it; if not, logically divide into:
     - ## Key Metrics Analysis
     - ## In-depth Data Analysis
     - ## Key Findings and Insights
     - ## Conclusions and Recommendations
4. Do not generate Markdown tables (| col1 | col2 | format) in the report, data tables will be automatically appended at the end by the system
5. You may reference key data points from data tables in text, but do not attempt to copy entire tables
6. Do not add data or conclusions not in the original analysis
7. Do not create data tables or use placeholders (like "Data", "Metric1") to replace real content`,

	Chinese: `你是一位专业的数据分析报告排版专家。你的唯一任务是将下方提供的分析结果直接整理成一份格式规范的正式报告。

【最重要的规则】
- 所有需要的数据已经完整提供在下方，不需要获取任何额外信息
- 必须直接输出报告正文，第一行就是 # 标题
- 严禁输出任何过渡语、解释、准备说明（如"我将..."、"首先让我..."、"让我来..."等）
- 如果你输出了任何非报告正文的内容，视为任务失败

核心原则：
- 报告正文必须严格使用提供的"分析洞察（AI分析结果）"中的原始文字内容
- 严禁改写、缩减或臆测任何不在提供数据中的内容
- 你的工作是排版和组织，不是重新撰写
- 将原始分析文字按照报告结构合理分配到各章节中
- 如果原始分析文字已经包含了章节标题（如 ##、###），应保留其结构
- 关键指标数据和数据表信息可以作为补充内容引用

报告格式要求：
1. 第一行必须是报告标题，使用一级标题格式（# 标题），标题应简洁概括分析主题（不超过20个字）
2. 使用 Markdown 二级标题（## 标题）分节
3. 报告结构：
   - ## 分析背景与目的：简要说明用户的分析请求和目标（1-2句话即可）
   - ## 数据概况：简要描述数据源信息（1-2句话即可）
   - 然后将"分析洞察"中的原始分析内容完整地组织到后续章节中。如果原始内容已有结构，保留其结构；如果没有，按逻辑分为：
     - ## 关键指标分析
     - ## 深度数据分析
     - ## 关键发现与洞察
     - ## 结论与建议
4. 不要在报告中生成 Markdown 表格（| 列1 | 列2 | 格式），数据表格会由系统自动附加在报告末尾
5. 可以在文字中引用数据表中的关键数据点，但不要试图复制整个表格
6. 不要添加原始分析中没有的数据或结论
7. 不要自行创建数据表格或用占位符（如"数据"、"指标1"）替代真实内容`,
}

var reportUserPromptTemplates = map[Language]string{
	English: "Here is the complete analysis data, please output the report body directly (first line must be # title, do not output any other content):\n\n%s",
	Chinese: "以下是完整的分析数据，请直接输出报告正文（第一行必须是 # 标题，不要输出任何其他内容）：\n\n%s",
}

var dataSummaryTemplates = map[Language]map[string]string{
	English: {
		"user_request":     "## Analysis Request",
		"data_source":      "## Data Source",
		"key_metrics":      "## Key Metrics Data",
		"insights":         "## Analysis Insights (AI Results)",
		"data_table":       "## Data Table",
		"multiple_tables":  "## Multiple Data Tables",
		"charts":           "## Charts",
	},
	Chinese: {
		"user_request":     "## 用户分析请求",
		"data_source":      "## 数据源",
		"key_metrics":      "## 关键指标数据",
		"insights":         "## 分析洞察（AI分析结果）",
		"data_table":       "## 数据表",
		"multiple_tables":  "## 多个数据表",
		"charts":           "## 图表",
	},
}

// GetReportSystemPrompt 获取报告生成的系统提示词
func GetReportSystemPrompt() string {
	lang := GetLanguage()
	if prompt, ok := reportSystemPrompts[lang]; ok {
		return prompt
	}
	return reportSystemPrompts[English]
}

// GetReportUserPromptTemplate 获取用户提示词模板
func GetReportUserPromptTemplate() string {
	lang := GetLanguage()
	if template, ok := reportUserPromptTemplates[lang]; ok {
		return template
	}
	return reportUserPromptTemplates[English]
}

// GetDataSummaryTemplate 获取数据摘要模板
func GetDataSummaryTemplate(section string) string {
	lang := GetLanguage()
	if templates, ok := dataSummaryTemplates[lang]; ok {
		if template, ok := templates[section]; ok {
			return template
		}
	}
	// Fallback to English
	if templates, ok := dataSummaryTemplates[English]; ok {
		if template, ok := templates[section]; ok {
			return template
		}
	}
	return "## " + section
}
```

### 2. 添加翻译键

在 `src/i18n/translations_en.go` 中添加：

```go
// 报告生成
"report.save_dialog_title":        "Save Analysis Report",
"report.filename_prefix":          "Analysis_Report",
"report.llm_not_initialized":      "LLM service not initialized, please configure API Key first",
"report.generation_failed":        "Report generation failed: %s",
"report.data_expired":             "Report data has expired, please regenerate",
"report.word_generation_failed":   "Word document generation failed: %s",
"report.pdf_generation_failed":    "PDF generation failed: %s",
"report.no_content":               "No content to export",

// 报告章节
"report.section.key_indicators":      "Key Indicators",
"report.section.data_visualization":  "Data Visualization",
"report.section.data_tables":         "Data Tables",

// 报告元素
"report.data_source_label":       "Data Source: ",
"report.analysis_request_label":  "Analysis Request:",
"report.generated_time_label":    "Generated Time: ",
"report.footer_text":             "Generated by VantageData Intelligent Analysis System",
"report.chart_label":             "Chart %d / %d",
"report.total_rows":              "Total %d rows",
"report.showing_columns":         "(Showing first %d columns)",
"report.category_label":          "Category",
```

在 `src/i18n/translations_zh.go` 中添加对应的中文翻译。

## 第二步：修改报告生成控制器

在 `src/app_report_generate.go` 中：

```go
import "vantagedata/i18n"

// 修改 callLLMForReport 函数
func (a *App) callLLMForReport(dataSummary string, userRequest string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 使用国际化提示词
	systemPrompt := i18n.GetReportSystemPrompt()
	userPromptTemplate := i18n.GetReportUserPromptTemplate()
	userPrompt := fmt.Sprintf(userPromptTemplate, dataSummary)

	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: userPrompt},
	}

	resp, err := a.einoService.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf(i18n.T("report.generation_failed", err.Error()))
	}

	// ... 后续处理
}

// 修改 buildDataSummary 函数
func buildDataSummary(req ReportGenerateRequest) string {
	var sb strings.Builder

	sb.WriteString(i18n.GetDataSummaryTemplate("user_request") + "\n")
	sb.WriteString(req.UserRequest)
	sb.WriteString("\n\n")

	sb.WriteString(i18n.GetDataSummaryTemplate("data_source") + "\n")
	if req.DataSourceName != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", 
			i18n.T("report.data_source_label"), req.DataSourceName))
	}
	sb.WriteString("\n")

	if len(req.Metrics) > 0 {
		sb.WriteString(i18n.GetDataSummaryTemplate("key_metrics") + "\n")
		// ... 其他代码
	}

	// ... 其他章节类似处理
}
```

## 第三步：修改PDF导出服务

在 `src/export/pdf_gopdf_service.go` 中：

```go
import "vantagedata/i18n"

// 修改章节标题
func (s *GopdfService) addMetricsSection(pdf *gopdf.GoPdf, metrics []MetricData, fontName string) {
	y := s.addSectionTitle(pdf, i18n.T("report.section.key_indicators"), fontName)
	// ... 其他代码
}

// 修改页面元素
func (s *GopdfService) addCoverPage(pdf *gopdf.GoPdf, title string, dataSourceName string, userRequest string, fontName string) {
	// ... 其他代码
	
	if dataSourceName != "" {
		dsText := i18n.T("report.data_source_label") + dataSourceName
		// ... 渲染代码
	}
	
	if userRequest != "" {
		labelText := i18n.T("report.analysis_request_label")
		// ... 渲染代码
	}
	
	// ... 其他代码
}
```

## 测试

### 快速测试步骤

1. **设置语言为English**
   ```go
   // 在配置中设置
   config.Language = "English"
   ```

2. **生成报告**
   ```go
   reportID, err := app.PrepareReport(req)
   if err != nil {
       log.Fatal(err)
   }
   
   err = app.ExportReport(reportID, "pdf")
   if err != nil {
       log.Fatal(err)
   }
   ```

3. **验证**
   - 打开生成的PDF
   - 检查所有文本是否为英文
   - 检查文件对话框标题
   - 检查默认文件名

4. **切换到中文测试**
   ```go
   config.Language = "简体中文"
   ```
   重复步骤2-3

## 常见问题

### Q: 如何添加新的翻译键？
A: 在 `translations_en.go` 和 `translations_zh.go` 中添加相同的键，然后在代码中使用 `i18n.T("key")`。

### Q: 如何修改LLM提示词？
A: 编辑 `src/i18n/prompts.go` 中的 `reportSystemPrompts` 映射。

### Q: 如何测试特定语言？
A: 使用 `i18n.SetLanguage(i18n.English)` 或 `i18n.SetLanguage(i18n.Chinese)` 设置语言。

### Q: 报告质量下降怎么办？
A: 检查提示词翻译质量，必要时请母语者审查并调整措辞。

## 下一步

1. 完成核心功能（P0任务）
2. 运行集成测试
3. 修复发现的问题
4. 完成其他导出格式（P1-P2任务）
5. 更新文档

## 获取帮助

- 查看完整规范：`.kiro/specs/report-i18n/`
- 查看问题分析：`doc/REPORT_I18N_ANALYSIS.md`
- 查看国际化文档：`doc/I18N_OVERVIEW.md`

---

**开始时间**: 现在  
**预计完成**: 5-7天  
**优先级**: P0（紧急）
