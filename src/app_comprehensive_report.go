package main

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"vantagedata/export"
	"vantagedata/i18n"

	"github.com/cloudwego/eino/schema"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// regex to extract base64 images from markdown content
var base64ImageRegex = regexp.MustCompile(`!\[.*?\]\((data:image/[^;]+;base64,[A-Za-z0-9+/=]+)\)`)

// ComprehensiveReportRequest represents the request for generating a comprehensive report
type ComprehensiveReportRequest struct {
	ThreadID       string   `json:"threadId"`
	DataSourceName string   `json:"dataSourceName"`
	SessionName    string   `json:"sessionName"`
	ChartImages    []string `json:"chartImages"` // base64 encoded chart images from frontend ECharts rendering
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
	chartImageSet := make(map[string]bool) // deduplicate chart images

	// Known suggestion request patterns that should be skipped
	suggestionPatterns := []string{
		"请给出一些本数据源的分析建议",
		"Give me some analysis suggestions for this data source",
	}

	for i, msg := range thread.Messages {
		if msg.Role == "user" {
			// Skip suggestion request messages (auto-generated first message)
			isSuggestionRequest := false
			trimmedContent := strings.TrimSpace(msg.Content)
			for _, pattern := range suggestionPatterns {
				if strings.Contains(trimmedContent, pattern) {
					isSuggestionRequest = true
					break
				}
			}
			if isSuggestionRequest {
				a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Skipping suggestion request message: %s", msg.ID))
				continue
			}

			// Get analysis data using App method (resolves file:// references)
			analysisData, err := a.GetMessageAnalysisData(req.ThreadID, msg.ID)
			if err != nil {
				a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Failed to get analysis data for message %s: %v", msg.ID, err))
			}

			// Add user's analysis request
			analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.analysis_request"), msg.Content))

			// Get the corresponding assistant response and extract inline base64 images
			if i+1 < len(thread.Messages) && thread.Messages[i+1].Role == "assistant" {
				assistantMsg := thread.Messages[i+1]
				if assistantMsg.Content != "" {
					// Extract base64 images from assistant message content
					matches := base64ImageRegex.FindAllStringSubmatch(assistantMsg.Content, -1)
					for _, match := range matches {
						if len(match) > 1 && !chartImageSet[match[1]] {
							chartImages = append(chartImages, match[1])
							chartImageSet[match[1]] = true
						}
					}

					// Add text content (strip inline images for LLM summary)
					cleanContent := base64ImageRegex.ReplaceAllString(assistantMsg.Content, "[图表]")
					analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.analysis_result"), cleanContent))

					// Extract json:table blocks from assistant message content
					reJsonTable := regexp.MustCompile("(?s)```\\s*json:table\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
					for _, jtMatch := range reJsonTable.FindAllStringSubmatchIndex(assistantMsg.Content, -1) {
						if len(jtMatch) >= 4 {
							fullMatchStart := jtMatch[0]
							jsonContent := strings.TrimSpace(assistantMsg.Content[jtMatch[2]:jtMatch[3]])

							// Extract table title from the line before the code block
							tableTitle := ""
							if fullMatchStart > 0 {
								textBefore := assistantMsg.Content[:fullMatchStart]
								lastNewline := strings.LastIndex(textBefore, "\n")
								if lastNewline >= 0 {
									lineBeforeCodeBlock := strings.TrimSpace(textBefore[lastNewline+1:])
									lineBeforeCodeBlock = strings.TrimLeft(lineBeforeCodeBlock, "#*- ")
									lineBeforeCodeBlock = strings.TrimRight(lineBeforeCodeBlock, ":：")
									tableTitle = strings.TrimSpace(lineBeforeCodeBlock)
									if strings.HasPrefix(tableTitle, "{") || strings.HasPrefix(tableTitle, "[") || strings.HasPrefix(tableTitle, "```") {
										tableTitle = ""
									}
								}
							}
							if tableTitle == "" {
								tableTitle = fmt.Sprintf("%s %d", i18n.T("comprehensive_report.table"), len(allTableData)+1)
							}

							// Try to parse as object array
							var tableData []map[string]interface{}
							if err := json.Unmarshal([]byte(jsonContent), &tableData); err != nil {
								// Try 2D array format
								var arrayData [][]interface{}
								if err2 := json.Unmarshal([]byte(jsonContent), &arrayData); err2 == nil && len(arrayData) > 1 {
									headers := make([]string, len(arrayData[0]))
									for i, h := range arrayData[0] {
										headers[i] = fmt.Sprintf("%v", h)
									}
									tableData = make([]map[string]interface{}, 0, len(arrayData)-1)
									for _, row := range arrayData[1:] {
										rowMap := make(map[string]interface{})
										for ri, val := range row {
											if ri < len(headers) {
												rowMap[headers[ri]] = val
											}
										}
										tableData = append(tableData, rowMap)
									}
								}
							}

							if len(tableData) > 0 {
								// Convert rows format to columns/data format
								firstRow := tableData[0]
								var tableCols []TableColumn
								var colOrder []string
								for key := range firstRow {
									tableCols = append(tableCols, TableColumn{Title: key, DataType: "string"})
									colOrder = append(colOrder, key)
								}
								var tableRows [][]any
								for _, row := range tableData {
									var rowData []any
									for _, col := range colOrder {
										rowData = append(rowData, row[col])
									}
									tableRows = append(tableRows, rowData)
								}
								if len(tableCols) > 0 && len(tableRows) > 0 {
									allTableData = append(allTableData, NamedTableData{
										Name: tableTitle,
										Table: TableData{
											Columns: tableCols,
											Data:    tableRows,
										},
									})
									a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Extracted json:table from content: %s (%d rows)", tableTitle, len(tableRows)))
								}
							}
						}
					}

					// Extract markdown tables from assistant message content
					mdTables := extractMarkdownTablesFromContent(assistantMsg.Content)
					for _, mdTable := range mdTables {
						if len(mdTable.Rows) > 0 {
							mdTitle := mdTable.Title
							if mdTitle == "" {
								mdTitle = fmt.Sprintf("%s %d", i18n.T("comprehensive_report.table"), len(allTableData)+1)
							}
							// Convert rows format to columns/data format
							firstRow := mdTable.Rows[0]
							var tableCols []TableColumn
							var colOrder []string
							for key := range firstRow {
								tableCols = append(tableCols, TableColumn{Title: key, DataType: "string"})
								colOrder = append(colOrder, key)
							}
							var tableRows [][]any
							for _, row := range mdTable.Rows {
								var rowData []any
								for _, col := range colOrder {
									rowData = append(rowData, row[col])
								}
								tableRows = append(tableRows, rowData)
							}
							if len(tableCols) > 0 && len(tableRows) > 0 {
								allTableData = append(allTableData, NamedTableData{
									Name: mdTitle,
									Table: TableData{
										Columns: tableCols,
										Data:    tableRows,
									},
								})
								a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Extracted markdown table from content: %s (%d rows)", mdTitle, len(tableRows)))
							}
						}
					}
				}
			}

			// Process analysis results for tables and insights
			if analysisData != nil {
				if items, ok := analysisData["analysisResults"]; items != nil && ok {
					if resultItems, ok := items.([]AnalysisResultItem); ok {
						a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Message %s has %d analysis result items", msg.ID, len(resultItems)))
						for _, item := range resultItems {
							a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT]   Item %s: type=%s, dataType=%T", item.ID, item.Type, item.Data))
							switch item.Type {
							case "echarts":
								// ECharts data is JSON config, not an image - cannot render server-side
								// Frontend is responsible for rendering ECharts to images and passing via req.ChartImages
								a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Found echarts item %s (data len: %d) - will be rendered by frontend", item.ID, len(fmt.Sprintf("%v", item.Data))))
							case "table":
								// Table data may be a JSON string (resolved from file://) or a map
								tableMap := make(map[string]interface{})
								switch td := item.Data.(type) {
								case string:
									// Resolved from file:// - parse JSON string
									if err := json.Unmarshal([]byte(td), &tableMap); err != nil {
										a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Failed to parse table JSON: %v", err))
										continue
									}
								case map[string]interface{}:
									tableMap = td
								default:
									continue
								}

								// Try columns/data format first
								columns, colOk := tableMap["columns"].([]interface{})
								data, dataOk := tableMap["data"].([]interface{})
								if colOk && dataOk {
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
											tableRows = append(tableRows, rowSlice)
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
										if item.Metadata != nil {
											if name, ok := item.Metadata["name"].(string); ok && name != "" {
												tableName = name
											}
										}
										allTableData = append(allTableData, NamedTableData{
											Name: tableName,
											Table: TableData{
												Columns: tableCols,
												Data:    tableRows,
											},
										})
										// Also add table summary to analysis contents for LLM
										var colNames []string
										for _, c := range tableCols {
											colNames = append(colNames, c.Title)
										}
										analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s: %s (%d %s)",
											i18n.T("comprehensive_report.table"), tableName,
											strings.Join(colNames, ", "), len(tableRows), "rows"))
									}
								} else if rows, rowsOk := tableMap["rows"].([]interface{}); rowsOk && len(rows) > 0 {
									// Fallback: rows format from restored data {title: "...", rows: [{col: val, ...}, ...]}
									tableTitle, _ := tableMap["title"].(string)
									if tableTitle == "" {
										tableTitle = fmt.Sprintf("%s %d", i18n.T("comprehensive_report.table"), len(allTableData)+1)
									}
									// Extract columns from first row
									if firstRow, ok := rows[0].(map[string]interface{}); ok {
										var tableCols []TableColumn
										var colOrder []string
										for key := range firstRow {
											tableCols = append(tableCols, TableColumn{Title: key, DataType: "string"})
											colOrder = append(colOrder, key)
										}
										var tableRows [][]any
										for _, row := range rows {
											if rowMap, ok := row.(map[string]interface{}); ok {
												var rowData []any
												for _, col := range colOrder {
													rowData = append(rowData, rowMap[col])
												}
												tableRows = append(tableRows, rowData)
											}
										}
										if len(tableCols) > 0 && len(tableRows) > 0 {
											allTableData = append(allTableData, NamedTableData{
												Name: tableTitle,
												Table: TableData{
													Columns: tableCols,
													Data:    tableRows,
												},
											})
											analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s: %s (%d %s)",
												i18n.T("comprehensive_report.table"), tableTitle,
												strings.Join(colOrder, ", "), len(tableRows), "rows"))
										}
									}
								}
							case "insight":
								if strData, ok := item.Data.(string); ok && strData != "" {
									analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.insight"), strData))
								} else if mapData, ok := item.Data.(map[string]interface{}); ok {
									// Insight may be stored as {text: "...", icon: "..."} object
									if text, ok := mapData["text"].(string); ok && text != "" {
										analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.insight"), text))
									}
								} else if insightObj, ok := item.Data.(Insight); ok && insightObj.Text != "" {
									analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.insight"), insightObj.Text))
								}
							case "metric":
								// Metric may be stored as {title: "...", value: "...", change: "..."} or Metric struct
								if mapData, ok := item.Data.(map[string]interface{}); ok {
									title, _ := mapData["title"].(string)
									value, _ := mapData["value"].(string)
									if title != "" && value != "" {
										change, _ := mapData["change"].(string)
										metricText := fmt.Sprintf("%s: %s", title, value)
										if change != "" {
											metricText += fmt.Sprintf(" (%s)", change)
										}
										analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.key_metric"), metricText))
									}
								} else if strData, ok := item.Data.(string); ok && strData != "" {
									analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.key_metric"), strData))
								}
							case "csv":
								// CSV data as string - include as analysis content
								if strData, ok := item.Data.(string); ok && strData != "" {
									analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.table"), strData))
								}
							}
						}
					}
				}
			}
		}
	}

	// Also collect chart images from session files (Python-generated charts)
	sessionFiles, err := a.chatService.GetSessionFiles(req.ThreadID)
	if err == nil {
		sessionDir := a.chatService.GetSessionFilesDirectory(req.ThreadID)
		for _, file := range sessionFiles {
			if file.Type == "image" && (strings.HasSuffix(file.Name, ".png") || strings.HasSuffix(file.Name, ".jpg") || strings.HasSuffix(file.Name, ".jpeg")) {
				filePath := filepath.Join(sessionDir, file.Name)
				imageData, readErr := os.ReadFile(filePath)
				if readErr == nil {
					base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)
					if !chartImageSet[base64Data] {
						chartImages = append(chartImages, base64Data)
						chartImageSet[base64Data] = true
					}
				}
			}
		}
	}

	// Merge chart images from frontend (ECharts rendered to base64 by frontend)
	if len(req.ChartImages) > 0 {
		a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Received %d chart images from frontend", len(req.ChartImages)))
		for _, img := range req.ChartImages {
			if !chartImageSet[img] {
				chartImages = append(chartImages, img)
				chartImageSet[img] = true
			}
		}
	}

	if len(analysisContents) == 0 {
		a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] No valid analysis contents found. Thread has %d messages.", len(thread.Messages)))
		for idx, msg := range thread.Messages {
			a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT]   Message[%d]: role=%s, id=%s, contentLen=%d", idx, msg.Role, msg.ID, len(msg.Content)))
		}
		return nil, fmt.Errorf("%s", i18n.T("comprehensive_report.no_valid_analysis"))
	}

	// Compute content hash to detect changes
	contentHash := computeAnalysisHash(analysisContents, len(allTableData))
	reportID := fmt.Sprintf("comprehensive_%s", req.ThreadID)

	// Check if we have a cached report with the same content
	comprehensiveReportCacheMu.Lock()
	if cached, ok := comprehensiveReportCache[reportID]; ok {
		if cached.ContentHash == contentHash {
			// Update chart images from frontend even when using cache
			// (frontend may have rendered new ECharts since last cache)
			if len(req.ChartImages) > 0 || len(chartImages) > 0 {
				allCharts := chartImages
				// Deduplicate
				seen := make(map[string]bool)
				for _, img := range allCharts {
					seen[img] = true
				}
				for _, img := range req.ChartImages {
					if !seen[img] {
						allCharts = append(allCharts, img)
					}
				}
				cached.ExportData.ChartImages = allCharts
			}
			comprehensiveReportCacheMu.Unlock()
			a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Using cached report: %s (updated %d chart images)", reportID, len(cached.ExportData.ChartImages)))
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

	a.Log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Export data: insights=%d, insightsLen=%d, charts=%d, tables=%d",
		len(exportData.Insights),
		func() int { total := 0; for _, s := range exportData.Insights { total += len(s) }; return total }(),
		len(exportData.ChartImages),
		len(exportData.AllTableData)))

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
