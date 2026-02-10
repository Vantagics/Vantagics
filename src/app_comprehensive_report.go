package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
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

// ComprehensiveReportRequest represents the request for generating a comprehensive report
type ComprehensiveReportRequest struct {
	ThreadID       string `json:"threadId"`
	DataSourceName string `json:"dataSourceName"`
	SessionName    string `json:"sessionName"`
}

// ComprehensiveReportResult represents the result of preparing a comprehensive report
type ComprehensiveReportResult struct {
	ReportID string `json:"reportId"`
	Cached   bool   `json:"cached"`
}

// cachedComprehensiveReport holds a prepared comprehensive report
type cachedComprehensiveReport struct {
	ExportData     export.DashboardData
	CreatedAt      time.Time
	ContentHash    string // Hash of analysis content to detect changes
	DataSourceName string
	SessionName    string
}

var (
	comprehensiveReportCache   = make(map[string]*cachedComprehensiveReport)
	comprehensiveReportCacheMu sync.Mutex
)

// computeAnalysisHash computes a hash of all analysis content to detect changes
func computeAnalysisHash(contents []string, tableCount int) string {
	hasher := md5.New()
	for _, content := range contents {
		hasher.Write([]byte(content))
	}
	hasher.Write([]byte(fmt.Sprintf("tables:%d", tableCount)))
	return hex.EncodeToString(hasher.Sum(nil))
}

// PrepareComprehensiveReport prepares a comprehensive report and caches it
// Returns a report ID that can be used with ExportComprehensiveReport
func (a *App) PrepareComprehensiveReport(req ComprehensiveReportRequest) (*ComprehensiveReportResult, error) {
	if a.einoService == nil {
		return nil, fmt.Errorf("%s", i18n.T("report.llm_not_initialized"))
	}

	if a.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}

	a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Starting report preparation for thread: %s", req.ThreadID))

	// Load the thread to get all messages
	thread, err := a.chatService.LoadThread(req.ThreadID)
	if err != nil {
		return nil, fmt.Errorf("failed to load thread: %v", err)
	}

	if thread == nil {
		return nil, fmt.Errorf("thread not found: %s", req.ThreadID)
	}

	// Collect all valid analysis results (excluding the first suggestion message)
	var analysisContents []string
	var chartImages []string
	var allTableData []NamedTableData
	isFirstUserMessage := true

	for i, msg := range thread.Messages {
		if msg.Role == "user" {
			if isFirstUserMessage {
				// Skip the first user message (analysis suggestions)
				isFirstUserMessage = false
				continue
			}

			// Get analysis data for this message
			analysisData, err := a.chatService.GetMessageAnalysisData(req.ThreadID, msg.ID)
			if err != nil {
				a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Failed to get analysis data for message %s: %v", msg.ID, err))
				continue
			}

			// Add user's analysis request
			analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.analysis_request"), msg.Content))

			// Get the corresponding assistant response
			if i+1 < len(thread.Messages) && thread.Messages[i+1].Role == "assistant" {
				assistantMsg := thread.Messages[i+1]
				if assistantMsg.Content != "" {
					analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.analysis_result"), assistantMsg.Content))
				}
			}

			// Process analysis results
			if items, ok := analysisData["analysisResults"]; items != nil && ok {
				if resultItems, ok := items.([]AnalysisResultItem); ok {
					for _, item := range resultItems {
						switch item.Type {
						case "echarts":
							// For charts, we need to note them for the report
							if strData, ok := item.Data.(string); ok && strData != "" {
								chartImages = append(chartImages, strData)
							}
						case "table":
							// Extract table data
							if tableData, ok := item.Data.(map[string]interface{}); ok {
								if columns, ok := tableData["columns"].([]interface{}); ok {
									if data, ok := tableData["data"].([]interface{}); ok {
										var tableCols []TableColumn
										for _, col := range columns {
											if colMap, ok := col.(map[string]interface{}); ok {
												tableCols = append(tableCols, TableColumn{
													Title:    fmt.Sprintf("%v", colMap["title"]),
													DataType: fmt.Sprintf("%v", colMap["dataType"]),
												})
											}
										}
										var tableRows [][]any
										for _, row := range data {
											if rowSlice, ok := row.([]interface{}); ok {
												var rowData []any
												for _, cell := range rowSlice {
													rowData = append(rowData, cell)
												}
												tableRows = append(tableRows, rowData)
											} else if rowMap, ok := row.(map[string]interface{}); ok {
												var rowData []any
												for _, col := range tableCols {
													if val, exists := rowMap[col.Title]; exists {
														rowData = append(rowData, val)
													} else {
														rowData = append(rowData, nil)
													}
												}
												tableRows = append(tableRows, rowData)
											}
										}
										if len(tableCols) > 0 && len(tableRows) > 0 {
											tableName := fmt.Sprintf("%s %d", i18n.T("comprehensive_report.table"), len(allTableData)+1)
											if name, ok := item.Metadata["name"].(string); ok && name != "" {
												tableName = name
											}
											allTableData = append(allTableData, NamedTableData{
												Name: tableName,
												Table: TableData{
													Columns: tableCols,
													Data:    tableRows,
												},
											})
										}
									}
								}
							}
						case "insight":
							if strData, ok := item.Data.(string); ok && strData != "" {
								analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.insight"), strData))
							}
						}
					}
				}
			}
		}
	}

	if len(analysisContents) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("comprehensive_report.no_valid_analysis"))
	}

	// Compute content hash to detect changes
	contentHash := computeAnalysisHash(analysisContents, len(allTableData))
	reportID := fmt.Sprintf("comprehensive_%s", req.ThreadID)

	// Check if we have a cached report with the same content
	comprehensiveReportCacheMu.Lock()
	if cached, ok := comprehensiveReportCache[reportID]; ok {
		if cached.ContentHash == contentHash {
			comprehensiveReportCacheMu.Unlock()
			a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Using cached report: %s", reportID))
			return &ComprehensiveReportResult{
				ReportID: reportID,
				Cached:   true,
			}, nil
		}
	}
	comprehensiveReportCacheMu.Unlock()

	a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Generating new report, collected %d analysis contents, %d tables, %d charts", 
		len(analysisContents), len(allTableData), len(chartImages)))

	// Build comprehensive summary for LLM
	comprehensiveSummary := buildComprehensiveSummary(req.DataSourceName, req.SessionName, analysisContents)

	// Call LLM to generate comprehensive report
	reportText, err := a.callLLMForComprehensiveReport(comprehensiveSummary)
	if err != nil {
		a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] LLM report generation failed: %v", err))
		return nil, fmt.Errorf("%s: %v", i18n.T("report.generation_failed"), err)
	}

	a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] LLM generated report text: %d chars", len(reportText)))

	// Parse into sections
	parsed := parseReportSections(reportText)
	a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Parsed report: title=%q, sections=%d", parsed.ReportTitle, len(parsed.Sections)))

	// Build export data
	exportData := buildComprehensiveReportExportData(req, parsed, chartImages, allTableData)

	// Cache the report
	comprehensiveReportCacheMu.Lock()
	// Clean old entries (keep max 10)
	if len(comprehensiveReportCache) >= 10 {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range comprehensiveReportCache {
			if oldestKey == "" || v.CreatedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.CreatedAt
			}
		}
		if oldestKey != "" {
			delete(comprehensiveReportCache, oldestKey)
		}
	}
	comprehensiveReportCache[reportID] = &cachedComprehensiveReport{
		ExportData:     exportData,
		CreatedAt:      time.Now(),
		ContentHash:    contentHash,
		DataSourceName: req.DataSourceName,
		SessionName:    req.SessionName,
	}
	comprehensiveReportCacheMu.Unlock()

	a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Report prepared and cached: %s", reportID))
	return &ComprehensiveReportResult{
		ReportID: reportID,
		Cached:   false,
	}, nil
}

// ExportComprehensiveReport exports a previously prepared comprehensive report
func (a *App) ExportComprehensiveReport(reportID string, format string) error {
	comprehensiveReportCacheMu.Lock()
	cached, ok := comprehensiveReportCache[reportID]
	comprehensiveReportCacheMu.Unlock()

	if !ok {
		return fmt.Errorf("%s", i18n.T("report.data_expired"))
	}

	timestamp := time.Now().Format("20060102_150405")

	switch format {
	case "pdf":
		return a.doExportComprehensivePDF(cached.ExportData, cached.DataSourceName, cached.SessionName, timestamp)
	default:
		return a.doExportComprehensiveWord(cached.ExportData, cached.DataSourceName, cached.SessionName, timestamp)
	}
}

// GenerateComprehensiveReport is kept for backward compatibility
// It prepares and immediately exports in Word format
func (a *App) GenerateComprehensiveReport(req ComprehensiveReportRequest) error {
	result, err := a.PrepareComprehensiveReport(req)
	if err != nil {
		return err
	}
	return a.ExportComprehensiveReport(result.ReportID, "word")
}

func buildComprehensiveSummary(dataSourceName, sessionName string, analysisContents []string) string {
	var sb strings.Builder

	sb.WriteString(i18n.T("comprehensive_report.data_source"))
	sb.WriteString(dataSourceName)
	sb.WriteString("\n\n")

	sb.WriteString(i18n.T("comprehensive_report.session_name"))
	sb.WriteString(sessionName)
	sb.WriteString("\n\n")

	sb.WriteString(i18n.T("comprehensive_report.all_analysis_results"))
	sb.WriteString("\n\n")

	for _, content := range analysisContents {
		sb.WriteString(content)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String()
}

func (a *App) callLLMForComprehensiveReport(summary string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	systemPrompt := i18n.GetComprehensiveReportSystemPrompt()
	userPrompt := fmt.Sprintf(i18n.GetComprehensiveReportUserPromptTemplate(), summary)

	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: userPrompt},
	}

	resp, err := a.einoService.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	content := resp.Content
	content = strings.TrimSpace(content)

	// Strip any preamble before the actual report
	if !strings.HasPrefix(content, "#") {
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

func buildComprehensiveReportExportData(req ComprehensiveReportRequest, parsed reportParseResult, chartImages []string, allTableData []NamedTableData) export.DashboardData {
	exportData := export.DashboardData{
		UserRequest:    fmt.Sprintf("%s - %s", req.DataSourceName, req.SessionName),
		DataSourceName: req.DataSourceName,
		ReportTitle:    parsed.ReportTitle,
		ChartImages:    chartImages,
	}

	// Convert table data
	if len(allTableData) > 0 {
		exportData.AllTableData = make([]export.NamedTableExportData, len(allTableData))
		for i, nt := range allTableData {
			cols := make([]export.TableColumn, len(nt.Table.Columns))
			for j, col := range nt.Table.Columns {
				cols[j] = export.TableColumn{
					Title:    col.Title,
					DataType: col.DataType,
				}
			}
			// Convert [][]any to [][]interface{} for export
			var exportRows [][]interface{}
			for _, row := range nt.Table.Data {
				var rowData []interface{}
				for _, cell := range row {
					rowData = append(rowData, cell)
				}
				exportRows = append(exportRows, rowData)
			}
			exportData.AllTableData[i] = export.NamedTableExportData{
				Name:  nt.Name,
				Table: export.TableData{Columns: cols, Data: exportRows},
			}
		}
	}

	// Reconstruct the full report text from parsed sections
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

func (a *App) doExportComprehensiveWord(exportData export.DashboardData, dataSourceName, sessionName, timestamp string) error {
	// Sanitize file name components
	safeDSName := sanitizeFileName(dataSourceName)
	safeSessionName := sanitizeFileName(sessionName)
	
	defaultFilename := fmt.Sprintf("%s_%s_%s.docx", i18n.T("comprehensive_report.filename_prefix"), safeDSName, safeSessionName)

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("comprehensive_report.save_dialog_title"),
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

	a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Word report saved: %s", savePath))
	return nil
}

func (a *App) doExportComprehensivePDF(exportData export.DashboardData, dataSourceName, sessionName, timestamp string) error {
	// Sanitize file name components
	safeDSName := sanitizeFileName(dataSourceName)
	safeSessionName := sanitizeFileName(sessionName)
	
	defaultFilename := fmt.Sprintf("%s_%s_%s.pdf", i18n.T("comprehensive_report.filename_prefix"), safeDSName, safeSessionName)

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("comprehensive_report.save_dialog_title"),
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

	a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] PDF report saved: %s", savePath))
	return nil
}

// sanitizeFileName removes or replaces characters that are invalid in file names
func sanitizeFileName(name string) string {
	// Replace invalid characters with underscore
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Trim spaces and limit length
	result = strings.TrimSpace(result)
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}
