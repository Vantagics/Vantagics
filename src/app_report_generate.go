package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"vantagedata/export"

	"github.com/cloudwego/eino/schema"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ReportGenerateRequest represents the request data for report generation
type ReportGenerateRequest struct {
	UserRequest    string            `json:"userRequest"`
	DataSourceName string            `json:"dataSourceName"` // 数据源名称
	Metrics        []DashboardMetric `json:"metrics"`
	Insights       []string          `json:"insights"`
	ChartImages    []string          `json:"chartImages"` // base64 encoded chart images
	TableData      *TableData        `json:"tableData"`
	AllTableData   []NamedTableData  `json:"allTableData"`
	Format         string            `json:"format"` // "word" or "pdf"
}

// GenerateReport uses LLM to generate a formal report from analysis results,
// then exports it as Word or PDF.
func (a *App) GenerateReport(req ReportGenerateRequest) error {
	if a.einoService == nil {
		return fmt.Errorf("LLM 服务未初始化，请先配置 API Key")
	}

	a.Log("[REPORT] Starting report generation...")

	// 1. Build a summary of all analysis data for the LLM
	dataSummary := buildDataSummary(req)

	// 2. Call LLM to generate report narrative
	reportText, err := a.callLLMForReport(dataSummary, req.UserRequest)
	if err != nil {
		a.Log(fmt.Sprintf("[REPORT] LLM report generation failed: %v", err))
		return fmt.Errorf("报告生成失败: %v", err)
	}

	a.Log(fmt.Sprintf("[REPORT] LLM generated report text: %d chars", len(reportText)))

	// 3. Parse LLM output into report title and sections
	parsed := parseReportSections(reportText)

	a.Log(fmt.Sprintf("[REPORT] Parsed report: title=%q, sections=%d", parsed.ReportTitle, len(parsed.Sections)))

	// 4. Determine format and export
	format := req.Format
	if format == "" {
		format = "word"
	}

	timestamp := time.Now().Format("20060102_150405")

	switch format {
	case "pdf":
		return a.exportReportAsPDF(req, parsed, timestamp)
	default:
		return a.exportReportAsWord(req, parsed, timestamp)
	}
}

// buildDataSummary creates a text summary of all dashboard data for the LLM prompt
func buildDataSummary(req ReportGenerateRequest) string {
	var sb strings.Builder

	sb.WriteString("## 用户分析请求\n")
	sb.WriteString(req.UserRequest)
	sb.WriteString("\n\n")

	sb.WriteString("## 数据源\n")
	if req.DataSourceName != "" {
		sb.WriteString(fmt.Sprintf("数据源名称: %s\n", req.DataSourceName))
	}
	sb.WriteString("\n")

	// Metrics
	if len(req.Metrics) > 0 {
		sb.WriteString("## 关键指标数据\n")
		for _, m := range req.Metrics {
			sb.WriteString(fmt.Sprintf("- %s: %s", m.Title, m.Value))
			if m.Change != "" {
				sb.WriteString(fmt.Sprintf(" (变化: %s)", m.Change))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Insights
	if len(req.Insights) > 0 {
		sb.WriteString("## 分析洞察（AI分析结果）\n")
		for _, insight := range req.Insights {
			sb.WriteString(insight)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Table data summary - provide more detail
	if req.TableData != nil && len(req.TableData.Columns) > 0 {
		sb.WriteString("## 数据表\n")
		sb.WriteString("列: ")
		colNames := make([]string, len(req.TableData.Columns))
		for i, col := range req.TableData.Columns {
			colNames[i] = col.Title
		}
		sb.WriteString(strings.Join(colNames, ", "))
		sb.WriteString(fmt.Sprintf("\n总行数: %d\n", len(req.TableData.Data)))

		// Include more sample rows for richer analysis
		maxSampleRows := 20
		if len(req.TableData.Data) < maxSampleRows {
			maxSampleRows = len(req.TableData.Data)
		}
		if maxSampleRows > 0 {
			sb.WriteString("数据样本:\n")
			for i := 0; i < maxSampleRows; i++ {
				row := req.TableData.Data[i]
				vals := make([]string, 0, len(row))
				for j := 0; j < len(req.TableData.Columns) && j < len(row); j++ {
					vals = append(vals, fmt.Sprintf("%s=%v", req.TableData.Columns[j].Title, row[j]))
				}
				sb.WriteString(fmt.Sprintf("  %s\n", strings.Join(vals, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	// Multiple tables - provide more detail
	if len(req.AllTableData) > 0 {
		sb.WriteString("## 多个数据表\n")
		for _, nt := range req.AllTableData {
			sb.WriteString(fmt.Sprintf("### 表: %s\n", nt.Name))
			colNames := make([]string, len(nt.Table.Columns))
			for i, col := range nt.Table.Columns {
				colNames[i] = col.Title
			}
			sb.WriteString(fmt.Sprintf("列: %s, 总行数: %d\n", strings.Join(colNames, ", "), len(nt.Table.Data)))

			// Include sample data from each table
			maxSample := 10
			if len(nt.Table.Data) < maxSample {
				maxSample = len(nt.Table.Data)
			}
			if maxSample > 0 {
				sb.WriteString("数据样本:\n")
				for i := 0; i < maxSample; i++ {
					row := nt.Table.Data[i]
					vals := make([]string, 0, len(row))
					for j := 0; j < len(nt.Table.Columns) && j < len(row); j++ {
						vals = append(vals, fmt.Sprintf("%s=%v", nt.Table.Columns[j].Title, row[j]))
					}
					sb.WriteString(fmt.Sprintf("  %s\n", strings.Join(vals, ", ")))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Chart info
	if len(req.ChartImages) > 0 {
		sb.WriteString(fmt.Sprintf("## 图表\n共有 %d 个图表/可视化，请在报告中描述这些图表可能展示的内容\n\n", len(req.ChartImages)))
	}

	return sb.String()
}

// reportSection represents a section of the generated report
type reportSection struct {
	Title   string
	Content string
}

// reportParseResult holds the parsed report title and sections
type reportParseResult struct {
	ReportTitle string
	Sections    []reportSection
}

// parseReportSections parses the LLM output into a title and structured sections.
// The first "# " heading is treated as the report title.
func parseReportSections(text string) reportParseResult {
	result := reportParseResult{}
	var sections []reportSection
	lines := strings.Split(text, "\n")

	var currentSection *reportSection

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect section headers (## or #)
		if strings.HasPrefix(trimmed, "## ") {
			if currentSection != nil {
				sections = append(sections, *currentSection)
			}
			currentSection = &reportSection{
				Title:   strings.TrimPrefix(trimmed, "## "),
				Content: "",
			}
		} else if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			// First "# " heading becomes the report title
			if result.ReportTitle == "" {
				result.ReportTitle = strings.TrimPrefix(trimmed, "# ")
				// Don't create a section for the report title itself
				continue
			}
			if currentSection != nil {
				sections = append(sections, *currentSection)
			}
			currentSection = &reportSection{
				Title:   strings.TrimPrefix(trimmed, "# "),
				Content: "",
			}
		} else {
			if currentSection == nil {
				// Content before any section header goes into an intro section
				if strings.TrimSpace(trimmed) == "" && len(sections) == 0 {
					continue // skip leading blank lines
				}
				currentSection = &reportSection{
					Title:   "",
					Content: "",
				}
			}
			if currentSection.Content != "" {
				currentSection.Content += "\n"
			}
			currentSection.Content += line
		}
	}

	if currentSection != nil {
		sections = append(sections, *currentSection)
	}

	result.Sections = sections
	return result
}

// callLLMForReport calls the LLM to generate a formal report narrative
func (a *App) callLLMForReport(dataSummary string, userRequest string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	systemPrompt := `你是一位专业的数据分析报告撰写专家。请严格根据提供的分析数据和洞察结果，撰写一份正式、详尽的数据分析报告。

核心原则：
- 报告内容必须完全基于提供的实际分析数据、指标、洞察文字和图表信息
- 严禁编造或臆测任何不在提供数据中的内容
- 所有结论和建议必须有提供的数据作为依据
- 如果提供了"分析洞察（AI分析结果）"，应将其中的分析内容、发现和结论作为报告的核心素材进行整理和扩展

报告格式要求：
1. 使用正式、专业的语言风格，内容要详实充分
2. 第一行必须是报告标题，使用一级标题格式（# 标题），标题应简洁概括分析主题（不超过20个字）
3. 报告结构清晰，使用 Markdown 二级标题（## 标题）分节
4. 包含以下部分：
   - ## 分析背景与目的：引用用户的原始分析请求，说明分析背景和目标
   - ## 数据概况：基于提供的数据源名称、字段、数据量等信息描述数据范围
   - ## 关键指标概览：逐一解读提供的每个关键指标的当前值及变化趋势，分析其业务意义
   - ## 深度数据分析：基于提供的洞察结果和数据表内容进行深入解读。如果有图表，描述图表展示的内容
   - ## 关键发现与洞察：整理和总结提供的分析洞察中的重要发现，每个发现引用具体数据
   - ## 结论与建议：基于以上分析总结核心结论，提出具体建议
5. 每个章节内容要充实，深入分析而非简单罗列
6. 大量引用提供的具体数据和指标来支撑分析结论
7. 语言专业严谨，逻辑清晰，有理有据
8. 直接输出报告正文，不要输出任何额外说明`

	userPrompt := fmt.Sprintf("请根据以下分析数据撰写正式报告：\n\n%s", dataSummary)

	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: userPrompt},
	}

	resp, err := a.einoService.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	return resp.Content, nil
}

// exportReportAsWord generates a Word document with LLM-generated report content
func (a *App) exportReportAsWord(req ReportGenerateRequest, parsed reportParseResult, timestamp string) error {
	defaultFilename := fmt.Sprintf("分析报告_%s.docx", timestamp)

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存分析报告",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Word文档", Pattern: "*.docx"},
		},
	})
	if err != nil || savePath == "" {
		return nil
	}

	// Build export data with LLM-generated insights replacing raw insights
	exportData := export.DashboardData{
		UserRequest:    req.UserRequest,
		DataSourceName: req.DataSourceName,
		ReportTitle:    parsed.ReportTitle,
		Metrics:        make([]export.MetricData, len(req.Metrics)),
		ChartImages:    req.ChartImages,
	}

	for i, m := range req.Metrics {
		exportData.Metrics[i] = export.MetricData{
			Title:  m.Title,
			Value:  m.Value,
			Change: m.Change,
		}
	}

	if req.TableData != nil {
		exportData.TableData = &export.TableData{
			Columns: make([]export.TableColumn, len(req.TableData.Columns)),
			Data:    req.TableData.Data,
		}
		for i, col := range req.TableData.Columns {
			exportData.TableData.Columns[i] = export.TableColumn{
				Title:    col.Title,
				DataType: col.DataType,
			}
		}
	}

	// Pass all tables for multi-table report
	if len(req.AllTableData) > 0 {
		exportData.AllTableData = make([]export.NamedTableExportData, len(req.AllTableData))
		for i, nt := range req.AllTableData {
			cols := make([]export.TableColumn, len(nt.Table.Columns))
			for j, col := range nt.Table.Columns {
				cols[j] = export.TableColumn{
					Title:    col.Title,
					DataType: col.DataType,
				}
			}
			exportData.AllTableData[i] = export.NamedTableExportData{
				Name:  nt.Name,
				Table: export.TableData{Columns: cols, Data: nt.Table.Data},
			}
		}
	}

	// Convert LLM sections into insights for the Word export
	var reportInsights []string
	for _, sec := range parsed.Sections {
		if sec.Title != "" {
			reportInsights = append(reportInsights, "## "+sec.Title)
		}
		if sec.Content != "" {
			reportInsights = append(reportInsights, sec.Content)
		}
	}
	exportData.Insights = reportInsights

	wordService := export.NewWordExportService()
	wordBytes, err := wordService.ExportDashboardToWord(exportData)
	if err != nil {
		return fmt.Errorf("Word文档生成失败: %v", err)
	}

	err = os.WriteFile(savePath, wordBytes, 0644)
	if err != nil {
		return fmt.Errorf("写入Word文件失败: %v", err)
	}

	a.Log(fmt.Sprintf("[REPORT] Report exported as Word: %s", savePath))
	return nil
}

// exportReportAsPDF generates a PDF document with LLM-generated report content
func (a *App) exportReportAsPDF(req ReportGenerateRequest, parsed reportParseResult, timestamp string) error {
	defaultFilename := fmt.Sprintf("分析报告_%s.pdf", timestamp)

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存分析报告",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF文件", Pattern: "*.pdf"},
		},
	})
	if err != nil || savePath == "" {
		return nil
	}

	exportData := export.DashboardData{
		UserRequest:    req.UserRequest,
		DataSourceName: req.DataSourceName,
		ReportTitle:    parsed.ReportTitle,
		Metrics:        make([]export.MetricData, len(req.Metrics)),
		ChartImages:    req.ChartImages,
	}

	for i, m := range req.Metrics {
		exportData.Metrics[i] = export.MetricData{
			Title:  m.Title,
			Value:  m.Value,
			Change: m.Change,
		}
	}

	if req.TableData != nil {
		exportData.TableData = &export.TableData{
			Columns: make([]export.TableColumn, len(req.TableData.Columns)),
			Data:    req.TableData.Data,
		}
		for i, col := range req.TableData.Columns {
			exportData.TableData.Columns[i] = export.TableColumn{
				Title:    col.Title,
				DataType: col.DataType,
			}
		}
	}

	// Pass all tables for multi-table report
	if len(req.AllTableData) > 0 {
		exportData.AllTableData = make([]export.NamedTableExportData, len(req.AllTableData))
		for i, nt := range req.AllTableData {
			cols := make([]export.TableColumn, len(nt.Table.Columns))
			for j, col := range nt.Table.Columns {
				cols[j] = export.TableColumn{
					Title:    col.Title,
					DataType: col.DataType,
				}
			}
			exportData.AllTableData[i] = export.NamedTableExportData{
				Name:  nt.Name,
				Table: export.TableData{Columns: cols, Data: nt.Table.Data},
			}
		}
	}

	// Convert LLM sections into insights
	var reportInsights []string
	for _, sec := range parsed.Sections {
		if sec.Title != "" {
			reportInsights = append(reportInsights, "## "+sec.Title)
		}
		if sec.Content != "" {
			reportInsights = append(reportInsights, sec.Content)
		}
	}
	exportData.Insights = reportInsights

	pdfService := export.NewPDFExportService()
	pdfBytes, err := pdfService.ExportDashboardToPDF(exportData)
	if err != nil {
		return fmt.Errorf("PDF生成失败: %v", err)
	}

	err = os.WriteFile(savePath, pdfBytes, 0644)
	if err != nil {
		return fmt.Errorf("写入PDF文件失败: %v", err)
	}

	a.Log(fmt.Sprintf("[REPORT] Report exported as PDF: %s", savePath))
	return nil
}
