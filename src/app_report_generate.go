package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"vantagedata/export"
	"vantagedata/i18n"

	"github.com/cloudwego/eino/schema"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ReportGenerateRequest represents the request data for report generation
type ReportGenerateRequest struct {
	UserRequest    string            `json:"userRequest"`
	DataSourceName string            `json:"dataSourceName"`
	Metrics        []DashboardMetric `json:"metrics"`
	Insights       []string          `json:"insights"`
	ChartImages    []string          `json:"chartImages"`
	TableData      *TableData        `json:"tableData"`
	AllTableData   []NamedTableData  `json:"allTableData"`
	Format         string            `json:"format"` // kept for compatibility
}

// cachedReport holds a prepared report that can be exported in any format
type cachedReport struct {
	ExportData export.DashboardData
	CreatedAt  time.Time
}

var (
	reportCache   = make(map[string]*cachedReport)
	reportCacheMu sync.Mutex
)

// PrepareReport generates the LLM report once and caches it.
// Returns a report ID that can be used with ExportReport.
func (a *App) PrepareReport(req ReportGenerateRequest) (string, error) {
	if a.einoService == nil {
		return "", fmt.Errorf("%s", i18n.T("report.llm_not_initialized"))
	}

	a.Log("[REPORT] Starting report generation...")

	// 1. Build data summary for LLM
	dataSummary := buildDataSummary(req)

	// 2. Call LLM once
	reportText, err := a.callLLMForReport(dataSummary, req.UserRequest)
	if err != nil {
		a.Log(fmt.Sprintf("[REPORT] LLM report generation failed: %v", err))
		return "", fmt.Errorf("%s: %v", i18n.T("report.generation_failed"), err)
	}

	a.Log(fmt.Sprintf("[REPORT] LLM generated report text: %d chars", len(reportText)))

	// 3. Parse into sections
	parsed := parseReportSections(reportText)
	a.Log(fmt.Sprintf("[REPORT] Parsed report: title=%q, sections=%d", parsed.ReportTitle, len(parsed.Sections)))

	// 4. Build export data
	exportData := buildReportExportData(req, parsed)

	// 5. Cache with a unique ID
	reportID := fmt.Sprintf("report_%d", time.Now().UnixMilli())

	reportCacheMu.Lock()
	// Clean old entries (keep max 5)
	if len(reportCache) >= 5 {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range reportCache {
			if oldestKey == "" || v.CreatedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.CreatedAt
			}
		}
		if oldestKey != "" {
			delete(reportCache, oldestKey)
		}
	}
	reportCache[reportID] = &cachedReport{
		ExportData: exportData,
		CreatedAt:  time.Now(),
	}
	reportCacheMu.Unlock()

	a.Log(fmt.Sprintf("[REPORT] Report prepared and cached: %s", reportID))
	return reportID, nil
}

// ExportReport exports a previously prepared report in the specified format.
func (a *App) ExportReport(reportID string, format string) error {
	reportCacheMu.Lock()
	cached, ok := reportCache[reportID]
	reportCacheMu.Unlock()

	if !ok {
		return fmt.Errorf("%s", i18n.T("report.data_expired"))
	}

	timestamp := time.Now().Format("20060102_150405")

	switch format {
	case "pdf":
		return a.doExportPDF(cached.ExportData, timestamp)
	default:
		return a.doExportWord(cached.ExportData, timestamp)
	}
}

// GenerateReport is kept for backward compatibility.
// It prepares and immediately exports in the requested format.
func (a *App) GenerateReport(req ReportGenerateRequest) error {
	reportID, err := a.PrepareReport(req)
	if err != nil {
		return err
	}
	format := req.Format
	if format == "" {
		format = "word"
	}
	return a.ExportReport(reportID, format)
}

func (a *App) doExportWord(exportData export.DashboardData, timestamp string) error {
	defaultFilename := fmt.Sprintf("%s_%s.docx", i18n.T("report.filename_prefix"), timestamp)

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("report.save_dialog_title"),
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Word文档", Pattern: "*.docx"},
		},
	})
	if err != nil || savePath == "" {
		return nil
	}

	wordService := export.NewWordExportService()
	wordBytes, err := wordService.ExportDashboardToWord(exportData)
	if err != nil {
		return fmt.Errorf(i18n.T("report.word_generation_failed"), err)
	}

	err = os.WriteFile(savePath, wordBytes, 0644)
	if err != nil {
		return fmt.Errorf(i18n.T("report.write_file_failed"), err)
	}

	a.Log(fmt.Sprintf("[REPORT] Word report saved: %s", savePath))
	return nil
}

func (a *App) doExportPDF(exportData export.DashboardData, timestamp string) error {
	defaultFilename := fmt.Sprintf("%s_%s.pdf", i18n.T("report.filename_prefix"), timestamp)

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("report.save_dialog_title"),
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF文件", Pattern: "*.pdf"},
		},
	})
	if err != nil || savePath == "" {
		return nil
	}

	pdfService := export.NewPDFExportService()
	pdfBytes, err := pdfService.ExportDashboardToPDF(exportData)
	if err != nil {
		return fmt.Errorf(i18n.T("report.pdf_generation_failed"), err)
	}

	err = os.WriteFile(savePath, pdfBytes, 0644)
	if err != nil {
		return fmt.Errorf(i18n.T("report.write_file_failed"), err)
	}

	a.Log(fmt.Sprintf("[REPORT] PDF report saved: %s", savePath))
	return nil
}

// buildReportExportData builds the shared export data from request and parsed LLM output
func buildReportExportData(req ReportGenerateRequest, parsed reportParseResult) export.DashboardData {
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

	// Pass table data for direct rendering by the export engine.
	// Tables are rendered separately after the LLM report text — the LLM is
	// instructed NOT to include markdown tables in its output.
	if req.TableData != nil && len(req.TableData.Columns) > 0 {
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

	// Reconstruct the full report text from parsed sections as a single insight.
	var sb strings.Builder
	for idx, sec := range parsed.Sections {
		if idx > 0 {
			sb.WriteString("\n\n")
		}
		if sec.Title != "" {
			sb.WriteString("## ")
			sb.WriteString(sec.Title)
			sb.WriteString("\n")
		}
		if sec.Content != "" {
			sb.WriteString(sec.Content)
		}
	}
	if sb.Len() > 0 {
		exportData.Insights = []string{sb.String()}
	}

	return exportData
}

// buildDataSummary creates a text summary of all dashboard data for the LLM prompt
func buildDataSummary(req ReportGenerateRequest) string {
	var sb strings.Builder

	sb.WriteString(i18n.GetDataSummaryTemplate("user_request"))
	sb.WriteString(req.UserRequest)
	sb.WriteString("\n\n")

	sb.WriteString(i18n.GetDataSummaryTemplate("data_source"))
	if req.DataSourceName != "" {
		sb.WriteString(i18n.FormatDataSummaryTemplate("data_source_name", req.DataSourceName))
	}
	sb.WriteString("\n")

	if len(req.Metrics) > 0 {
		sb.WriteString(i18n.GetDataSummaryTemplate("key_metrics"))
		for _, m := range req.Metrics {
			sb.WriteString(fmt.Sprintf("- %s: %s", m.Title, m.Value))
			if m.Change != "" {
				sb.WriteString(i18n.FormatDataSummaryTemplate("metric_change", m.Change))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(req.Insights) > 0 {
		sb.WriteString(i18n.GetDataSummaryTemplate("insights"))
		for _, insight := range req.Insights {
			// Strip embedded key=value data lines and json:table blocks from insights
			// These will be rendered directly by the PDF engine as tables
			cleaned := stripTableDataFromInsight(insight)
			sb.WriteString(cleaned)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if req.TableData != nil && len(req.TableData.Columns) > 0 {
		// Table data is rendered directly by the PDF engine, not passed to LLM
		colNames := make([]string, len(req.TableData.Columns))
		for i, col := range req.TableData.Columns {
			colNames[i] = col.Title
		}
		sb.WriteString(i18n.FormatDataSummaryTemplate("data_table",
			len(req.TableData.Data), strings.Join(colNames, ", ")))
	}

	if len(req.AllTableData) > 0 {
		// Table data is rendered directly by the PDF engine, not passed to LLM
		sb.WriteString(i18n.GetDataSummaryTemplate("multiple_tables"))
		for _, nt := range req.AllTableData {
			colNames := make([]string, len(nt.Table.Columns))
			for i, col := range nt.Table.Columns {
				colNames[i] = col.Title
			}
			sb.WriteString(i18n.FormatDataSummaryTemplate("table_info",
				nt.Name, len(nt.Table.Data), strings.Join(colNames, ", ")))
		}
		sb.WriteString("\n")
	}

	if len(req.ChartImages) > 0 {
		sb.WriteString(i18n.FormatDataSummaryTemplate("charts", len(req.ChartImages)))
	}

	return sb.String()
}

type reportSection struct {
	Title   string
	Content string
}

type reportParseResult struct {
	ReportTitle string
	Sections    []reportSection
}

func parseReportSections(text string) reportParseResult {
	result := reportParseResult{}
	var sections []reportSection
	lines := strings.Split(text, "\n")

	var currentSection *reportSection

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			if currentSection != nil {
				sections = append(sections, *currentSection)
			}
			currentSection = &reportSection{
				Title:   strings.TrimPrefix(trimmed, "## "),
				Content: "",
			}
		} else if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			if result.ReportTitle == "" {
				result.ReportTitle = strings.TrimPrefix(trimmed, "# ")
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
				if strings.TrimSpace(trimmed) == "" && len(sections) == 0 {
					continue
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

// stripTableDataFromInsight removes embedded structured data from insight text
// so the LLM only sees narrative content. The structured data (key=value lines,
// json:table blocks, markdown tables) will be rendered directly by the PDF engine.
func stripTableDataFromInsight(insight string) string {
	lines := strings.Split(insight, "\n")
	var result []string
	inCodeBlock := false

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Track code blocks — remove json:table blocks entirely
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				// Starting a code block
				if strings.Contains(trimmed, "json:table") {
					// Skip the entire json:table block
					i++
					for i < len(lines) {
						if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
							i++
							break
						}
						i++
					}
					continue
				}
				inCodeBlock = true
			} else {
				inCodeBlock = false
			}
			result = append(result, line)
			i++
			continue
		}

		// Handle json:table without backticks
		if trimmed == "json:table" {
			// Skip the json:table marker and the JSON block that follows
			i++
			depth := 0
			started := false
			for i < len(lines) {
				lt := strings.TrimSpace(lines[i])
				if !started && (strings.HasPrefix(lt, "{") || strings.HasPrefix(lt, "[")) {
					started = true
				}
				if started {
					depth += strings.Count(lt, "{") + strings.Count(lt, "[")
					depth -= strings.Count(lt, "}") + strings.Count(lt, "]")
					i++
					if depth <= 0 {
						break
					}
				} else {
					i++
				}
			}
			continue
		}

		if inCodeBlock {
			result = append(result, line)
			i++
			continue
		}

		// Skip markdown table blocks (| col1 | col2 |)
		if strings.HasPrefix(trimmed, "|") && strings.Count(trimmed, "|") >= 3 {
			// Skip consecutive | lines
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
				i++
			}
			continue
		}

		// Skip consecutive key=value structured lines (2+ lines with same keys)
		if strings.Contains(trimmed, "=") && trimmed != "" {
			keys := extractKVKeys(trimmed)
			if len(keys) >= 2 {
				// Look ahead to see if next lines have the same pattern
				j := i + 1
				matchCount := 1
				for j < len(lines) {
					nextTrimmed := strings.TrimSpace(lines[j])
					if nextTrimmed == "" {
						break
					}
					nextKeys := extractKVKeys(nextTrimmed)
					if len(nextKeys) == len(keys) && sameKeys(keys, nextKeys) {
						matchCount++
						j++
					} else {
						break
					}
				}
				if matchCount >= 2 {
					// Skip all these key=value lines
					i = j
					continue
				}
			}
		}

		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}

// extractKVKeys extracts key names from a key=value line
func extractKVKeys(line string) []string {
	content := line

	// Strip leading list markers
	content = strings.TrimLeft(content, " \t")
	if strings.HasPrefix(content, "- ") || strings.HasPrefix(content, "* ") {
		content = content[2:]
	}

	// Strip label prefix (text before ： or :)
	for _, sep := range []string{"：", ": "} {
		idx := strings.Index(content, sep)
		if idx > 0 && idx < 60 && strings.Contains(content[idx+len(sep):], "=") {
			content = content[idx+len(sep):]
			break
		}
	}

	normalized := strings.ReplaceAll(content, "，", ",")
	parts := strings.Split(normalized, ",")

	var keys []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		eqIdx := strings.Index(part, "=")
		if eqIdx <= 0 {
			return nil
		}
		keys = append(keys, strings.TrimSpace(part[:eqIdx]))
	}
	return keys
}

// sameKeys checks if two key slices contain the same keys
func sameKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (a *App) callLLMForReport(dataSummary string, userRequest string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	systemPrompt := i18n.GetReportSystemPrompt()
	userPromptTemplate := i18n.GetReportUserPromptTemplate()
	userPrompt := fmt.Sprintf(userPromptTemplate, dataSummary)

	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: userPrompt},
	}

	resp, err := a.einoService.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	// Post-process: strip any preamble before the actual report (# title)
	content := resp.Content
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "#") {
		// LLM may have output preamble text before the report title.
		// Find the first line starting with "# " and discard everything before it.
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "# ") {
				content = strings.Join(lines[i:], "\n")
				break
			}
		}
	}

	return content, nil
}
