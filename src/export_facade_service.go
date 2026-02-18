package main

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"html"
	"vantagedata/agent"
	"vantagedata/export"
	"vantagedata/i18n"

	ppt "github.com/VantageDataChat/GoPPT"
	gospreadsheet "github.com/VantageDataChat/GoExcel"
	"github.com/cloudwego/eino/schema"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportManager 定义导出管理接口
type ExportManager interface {
	ExportToCSV(id string, tableNames []string, outputPath string) error
	ExportToJSON(id string, tableNames []string, outputPath string) error
	ExportToSQL(id string, tableNames []string, outputPath string) error
	ExportToExcel(id string, tableNames []string, outputPath string) error
	ExportToMySQL(id string, tableNames []string, host, port, user, password, database string) error
	TestMySQLConnection(host, port, user, password string) error
	GetMySQLDatabases(host, port, user, password string) ([]string, error)
	ExportSessionHTML(threadID string) error
	ExportDashboardToPDF(data DashboardExportData) error
	ExportDashboardToExcel(data DashboardExportData) error
	ExportDashboardToPPT(data DashboardExportData) error
	ExportDashboardToWord(data DashboardExportData) error
	ExportSessionFilesToZip(threadID string, messageID string) error
	DownloadSessionFile(threadID string, fileName string) error
	GetSessionFileAsBase64(threadID string, fileName string) (string, error)
	GenerateCSVThumbnail(threadID string, fileName string) (string, error)
	GenerateFilePreview(threadID string, fileName string) (string, error)
	ExportTableToExcel(tableData *TableData, sheetName string) error
	ExportMessageToPDF(content string, messageID string) error
	PrepareComprehensiveReport(req ComprehensiveReportRequest) (*ComprehensiveReportResult, error)
	ExportComprehensiveReport(reportID string, format string) error
	GenerateComprehensiveReport(req ComprehensiveReportRequest) error
}

// ExportFacadeService 导出服务门面，封装所有导出相关的业务逻辑
type ExportFacadeService struct {
	ctx               context.Context
	dataSourceService *agent.DataSourceService
	chatService       *ChatService
	einoService       *agent.EinoService
	logger            func(string)

	// getMessageAnalysisDataFn is injected to resolve analysis data for comprehensive reports
	getMessageAnalysisDataFn func(threadID, messageID string) (map[string]interface{}, error)

	// Comprehensive report cache
	comprehensiveReportCache   map[string]*cachedComprehensiveReport
	comprehensiveReportCacheMu sync.Mutex
}

// NewExportFacadeService 创建新的 ExportFacadeService 实例
func NewExportFacadeService(
	dataSourceService *agent.DataSourceService,
	chatService *ChatService,
	einoService *agent.EinoService,
	logger func(string),
) *ExportFacadeService {
	return &ExportFacadeService{
		dataSourceService:        dataSourceService,
		chatService:              chatService,
		einoService:              einoService,
		logger:                   logger,
		comprehensiveReportCache: make(map[string]*cachedComprehensiveReport),
	}
}

// Name 返回服务名称
func (e *ExportFacadeService) Name() string {
	return "export"
}

// Initialize 初始化导出门面服务
func (e *ExportFacadeService) Initialize(ctx context.Context) error {
	e.ctx = ctx
	return nil
}

// Shutdown 关闭导出门面服务
func (e *ExportFacadeService) Shutdown() error {
	return nil
}

// SetContext 设置 Wails 上下文
func (e *ExportFacadeService) SetContext(ctx context.Context) {
	e.ctx = ctx
}

// SetGetMessageAnalysisDataFn 注入获取分析数据的函数
func (e *ExportFacadeService) SetGetMessageAnalysisDataFn(fn func(threadID, messageID string) (map[string]interface{}, error)) {
	e.getMessageAnalysisDataFn = fn
}

// SetEinoService updates the EinoService reference (used during reinitializeServices)
func (e *ExportFacadeService) SetEinoService(es *agent.EinoService) {
	e.einoService = es
}

// log 记录日志
func (e *ExportFacadeService) log(msg string) {
	if e.logger != nil {
		e.logger(msg)
	}
}

// --- Data Source Export Methods ---

// ExportToCSV exports one or more data source tables to CSV
func (e *ExportFacadeService) ExportToCSV(id string, tableNames []string, outputPath string) error {
	if e.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return e.dataSourceService.ExportToCSV(id, tableNames, outputPath)
}

// ExportToJSON exports one or more data source tables to JSON
func (e *ExportFacadeService) ExportToJSON(id string, tableNames []string, outputPath string) error {
	if e.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return e.dataSourceService.ExportToJSON(id, tableNames, outputPath)
}

// ExportToSQL exports one or more data source tables to SQL
func (e *ExportFacadeService) ExportToSQL(id string, tableNames []string, outputPath string) error {
	if e.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return e.dataSourceService.ExportToSQL(id, tableNames, outputPath)
}

// ExportToExcel exports one or more data source tables to Excel (.xlsx)
func (e *ExportFacadeService) ExportToExcel(id string, tableNames []string, outputPath string) error {
	if e.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return e.dataSourceService.ExportToExcel(id, tableNames, outputPath)
}

// ExportToMySQL exports one or more data source tables to MySQL
func (e *ExportFacadeService) ExportToMySQL(id string, tableNames []string, host, port, user, password, database string) error {
	if e.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	config := agent.DataSourceConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
	}
	return e.dataSourceService.ExportToMySQL(id, tableNames, config)
}

// TestMySQLConnection tests the connection to a MySQL server
func (e *ExportFacadeService) TestMySQLConnection(host, port, user, password string) error {
	if e.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return e.dataSourceService.TestMySQLConnection(host, port, user, password)
}

// GetMySQLDatabases returns a list of databases from the MySQL server
func (e *ExportFacadeService) GetMySQLDatabases(host, port, user, password string) ([]string, error) {
	if e.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}
	return e.dataSourceService.GetMySQLDatabases(host, port, user, password)
}

// --- Session HTML Export ---

// ExportSessionHTML exports the session trace as an HTML file
func (e *ExportFacadeService) ExportSessionHTML(threadID string) error {
	// Load thread from chat service to get complete message data including charts
	threads, err := e.chatService.LoadThreads()
	if err != nil {
		return fmt.Errorf("failed to load threads: %v", err)
	}

	var targetThread *ChatThread
	for i := range threads {
		if threads[i].ID == threadID {
			targetThread = &threads[i]
			break
		}
	}

	if targetThread == nil {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	if len(targetThread.Messages) == 0 {
		return fmt.Errorf("no messages found in thread")
	}

	// Generate HTML
	var htmlBuf strings.Builder
	safeTitle := html.EscapeString(targetThread.Title)
	htmlBuf.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Analysis Session Export - ` + safeTitle + `</title>
<style>
body { 
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; 
    max-width: 1200px; 
    margin: 0 auto; 
    padding: 20px; 
    line-height: 1.6; 
    background-color: #f8fafc;
}
.header {
    background: linear-gradient(135deg, #3b82f6, #6366f1);
    color: white;
    padding: 30px;
    border-radius: 12px;
    margin-bottom: 30px;
    text-align: center;
}
.header h1 {
    margin: 0 0 10px 0;
    font-size: 2em;
}
.header p {
    margin: 0;
    opacity: 0.9;
}
.message { 
    margin-bottom: 25px; 
    padding: 20px; 
    border-radius: 12px; 
    background: white;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
.user { 
    border-left: 4px solid #3b82f6; 
}
.assistant { 
    border-left: 4px solid #10b981; 
}
.role { 
    font-weight: 600; 
    margin-bottom: 10px; 
    color: #1e293b; 
    font-size: 0.9em;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}
.content {
    color: #334155;
    font-size: 15px;
}
.content h1, .content h2, .content h3, .content h4 {
    color: #0f172a;
    margin: 16px 0 8px 0;
    font-weight: 600;
}
.content ul, .content ol {
    margin: 12px 0;
    padding-left: 24px;
}
.content li {
    margin: 6px 0;
}
.content hr {
    border: none;
    border-top: 1px solid #e2e8f0;
    margin: 20px 0;
}
.chart-container {
    margin: 20px 0;
    padding: 20px;
    background: #f8fafc;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
}
.chart-title {
    font-weight: 600;
    color: #475569;
    margin-bottom: 15px;
    font-size: 0.9em;
}
img { 
    max-width: 100%; 
    height: auto; 
    border: 1px solid #e2e8f0; 
    border-radius: 8px; 
    margin: 10px 0;
    display: block;
}
pre { 
    background: #1e293b; 
    color: #e2e8f0; 
    padding: 15px; 
    border-radius: 8px; 
    overflow-x: auto; 
    font-family: "Consolas", "Monaco", monospace;
    font-size: 13px;
}
code {
    background: #f1f5f9;
    color: #0f172a;
    padding: 2px 6px;
    border-radius: 4px;
    font-family: "Consolas", "Monaco", monospace;
    font-size: 13px;
}
pre code {
    background: transparent;
    color: inherit;
    padding: 0;
}
.table-container {
    overflow-x: auto;
    margin: 15px 0;
}
table {
    width: 100%;
    border-collapse: collapse;
    font-size: 14px;
}
th, td {
    padding: 10px;
    text-align: left;
    border: 1px solid #e2e8f0;
}
th {
    background: #f1f5f9;
    font-weight: 600;
    color: #475569;
}
tr:nth-child(even) {
    background: #f8fafc;
}
.footer {
    margin-top: 40px;
    padding: 20px;
    text-align: center;
    color: #64748b;
    font-size: 0.9em;
    border-top: 1px solid #e2e8f0;
}
@media print {
    body { background: white; }
    .message { box-shadow: none; border: 1px solid #e2e8f0; page-break-inside: avoid; }
    .chart-container { page-break-inside: avoid; }
}
</style>
</head>
<body>
<div class="header">
<h1>📊 ` + safeTitle + `</h1>
<p>导出时间: ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
</div>
`)

	for _, msg := range targetThread.Messages {
		if msg.Role == "system" {
			continue
		}

		// Filter out technical code blocks
		content := msg.Content
		content = regexp.MustCompile("```[ \t]*json:echarts[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("(?:^|\\n)json:echarts\\s*\\n\\{[\\s\\S]+?\\n\\}").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*json:table[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("(?:^|\\n)json:table\\s*\\n(?:\\{[\\s\\S]+?\\n\\}|\\[[\\s\\S]+?\\n\\])").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*json:metrics[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*json:dashboard[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*(sql|SQL)[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*(python|Python|py)[\\s\\S]*?```").ReplaceAllString(content, "")

		// Convert Markdown to HTML using the shared function
		content = convertMarkdownToHTML(content)

		divClass := "message " + strings.ToLower(msg.Role)
		roleLabel := strings.ToUpper(msg.Role)
		if msg.Role == "user" {
			roleLabel = "👤 " + roleLabel
		} else {
			roleLabel = "🤖 " + roleLabel
		}

		htmlBuf.WriteString(fmt.Sprintf(`<div class="%s">
<div class="role">%s</div>
<div class="content">%s</div>
`, divClass, roleLabel, content))

		// Add chart data if present
		if msg.ChartData != nil && len(msg.ChartData.Charts) > 0 {
			for idx, chart := range msg.ChartData.Charts {
				htmlBuf.WriteString(`<div class="chart-container">`)
				htmlBuf.WriteString(fmt.Sprintf(`<div class="chart-title">📊 Chart %d - Type: %s</div>`, idx+1, strings.ToUpper(chart.Type)))

				switch chart.Type {
				case "image":
					if strings.HasPrefix(chart.Data, "data:image") {
						htmlBuf.WriteString(fmt.Sprintf(`<img src="%s" alt="Chart Image" />`, chart.Data))
					} else {
						htmlBuf.WriteString(`<p style="color: #64748b; font-style: italic;">Image data not available</p>`)
					}
				case "echarts":
					htmlBuf.WriteString(`<div style="padding: 20px; background: #f1f5f9; border-radius: 8px; border: 2px dashed #cbd5e1;">`)
					htmlBuf.WriteString(`<p style="color: #64748b; text-align: center; margin: 0;">`)
					htmlBuf.WriteString(`📊 ECharts Interactive Chart<br>`)
					htmlBuf.WriteString(`<small>This chart requires JavaScript to render. Please view in the original application for full interactivity.</small>`)
					htmlBuf.WriteString(`</p></div>`)
					htmlBuf.WriteString(`<details style="margin-top: 10px;">`)
					htmlBuf.WriteString(`<summary style="cursor: pointer; color: #64748b; font-size: 0.9em;">View Chart Configuration</summary>`)
					htmlBuf.WriteString(fmt.Sprintf(`<pre><code>%s</code></pre>`, html.EscapeString(chart.Data)))
					htmlBuf.WriteString(`</details>`)
				case "table", "csv":
					var tableData [][]interface{}
					if err := json.Unmarshal([]byte(chart.Data), &tableData); err == nil && len(tableData) > 0 {
						htmlBuf.WriteString(`<div class="table-container"><table>`)
						if len(tableData) > 0 {
							htmlBuf.WriteString(`<thead><tr>`)
							for _, cell := range tableData[0] {
								htmlBuf.WriteString(fmt.Sprintf(`<th>%s</th>`, html.EscapeString(fmt.Sprintf("%v", cell))))
							}
							htmlBuf.WriteString(`</tr></thead>`)
						}
						if len(tableData) > 1 {
							htmlBuf.WriteString(`<tbody>`)
							for _, row := range tableData[1:] {
								htmlBuf.WriteString(`<tr>`)
								for _, cell := range row {
									htmlBuf.WriteString(fmt.Sprintf(`<td>%s</td>`, html.EscapeString(fmt.Sprintf("%v", cell))))
								}
								htmlBuf.WriteString(`</tr>`)
							}
							htmlBuf.WriteString(`</tbody>`)
						}
						htmlBuf.WriteString(`</table></div>`)
					} else {
						htmlBuf.WriteString(`<p style="color: #64748b; font-style: italic;">Table data not available</p>`)
					}
				default:
					htmlBuf.WriteString(fmt.Sprintf(`<p style="color: #64748b; font-style: italic;">Unsupported chart type: %s</p>`, html.EscapeString(chart.Type)))
				}

				htmlBuf.WriteString(`</div>`)
			}
		}

		htmlBuf.WriteString(`</div>`)
	}

	htmlBuf.WriteString(`<div class="footer">
<p>Generated by VantageData - Intelligent Business Intelligence Platform</p>
<p>` + time.Now().Format("2006-01-02 15:04:05") + `</p>
</div>
</body></html>`)

	// Save File Dialog
	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		Title:           "Export Analysis to HTML",
		DefaultFilename: fmt.Sprintf("analysis_%s_%s.html", targetThread.Title, time.Now().Format("20060102_150405")),
		Filters:         []runtime.FileFilter{{DisplayName: "HTML Files", Pattern: "*.html"}},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	return os.WriteFile(savePath, []byte(htmlBuf.String()), 0644)
}



// --- Dashboard Export Methods ---

// ExportDashboardToPDF exports dashboard data to PDF using gopdf library
func (e *ExportFacadeService) ExportDashboardToPDF(data DashboardExportData) error {
	defaultFilename := generateExportFilename(data.DataSourceName, data.MessageID, "pdf")

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("dashboard.export_pdf_title"),
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: i18n.T("dashboard.filter_pdf"), Pattern: "*.pdf"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	pdfService := export.NewPDFExportService()

	exportData := export.DashboardData{
		UserRequest:    data.UserRequest,
		DataSourceName: data.DataSourceName,
		Metrics:        make([]export.MetricData, len(data.Metrics)),
		Insights:       data.Insights,
		ChartImages:    data.ChartImages,
	}

	for i, m := range data.Metrics {
		exportData.Metrics[i] = export.MetricData{
			Title:  m.Title,
			Value:  m.Value,
			Change: m.Change,
		}
	}

	if data.TableData != nil {
		exportData.TableData = &export.TableData{
			Columns: make([]export.TableColumn, len(data.TableData.Columns)),
			Data:    data.TableData.Data,
		}
		for i, col := range data.TableData.Columns {
			exportData.TableData.Columns[i] = export.TableColumn{
				Title:    col.Title,
				DataType: col.DataType,
			}
		}
	}

	pdfBytes, err := pdfService.ExportDashboardToPDF(exportData)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("export.pdf_failed", err))
	}

	err = os.WriteFile(savePath, pdfBytes, 0644)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.write_pdf_failed", err))
	}

	e.log(fmt.Sprintf("Dashboard exported to PDF successfully: %s", savePath))
	return nil
}

// ExportDashboardToExcel exports dashboard table data to Excel
func (e *ExportFacadeService) ExportDashboardToExcel(data DashboardExportData) error {
	hasMultipleTables := len(data.AllTableData) > 0
	hasSingleTable := data.TableData != nil && len(data.TableData.Columns) > 0

	if !hasMultipleTables && !hasSingleTable {
		return fmt.Errorf("no table data to export")
	}

	defaultFilename := generateExportFilename(data.DataSourceName, data.MessageID, "xlsx")

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("dashboard.export_excel_title"),
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: i18n.T("dashboard.filter_excel"), Pattern: "*.xlsx"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	excelService := export.NewExcelExportService()

	var excelBytes []byte

	if hasMultipleTables {
		var orderedTables []export.NamedTable
		usedNames := make(map[string]int)

		for i, namedTable := range data.AllTableData {
			sheetName := sanitizeSheetName(namedTable.Name)
			if sheetName == "" {
				sheetName = i18n.T("dashboard.sheet_fallback", i+1)
			}

			if count, exists := usedNames[sheetName]; exists {
				usedNames[sheetName] = count + 1
				sheetName = fmt.Sprintf("%s(%d)", sheetName, count+1)
			} else {
				usedNames[sheetName] = 1
			}

			exportTableData := &export.TableData{
				Columns: make([]export.TableColumn, len(namedTable.Table.Columns)),
				Data:    namedTable.Table.Data,
			}
			for j, col := range namedTable.Table.Columns {
				exportTableData.Columns[j] = export.TableColumn{
					Title:    col.Title,
					DataType: col.DataType,
				}
			}
			orderedTables = append(orderedTables, export.NamedTable{
				Name:  sheetName,
				Table: exportTableData,
			})
		}

		excelBytes, err = excelService.ExportOrderedTablesToExcel(orderedTables)
		if err != nil {
			return fmt.Errorf("%s", i18n.T("dashboard.generate_excel_failed", err))
		}
	} else {
		exportTableData := &export.TableData{
			Columns: make([]export.TableColumn, len(data.TableData.Columns)),
			Data:    data.TableData.Data,
		}
		for i, col := range data.TableData.Columns {
			exportTableData.Columns[i] = export.TableColumn{
				Title:    col.Title,
				DataType: col.DataType,
			}
		}

		sheetName := i18n.T("dashboard.sheet_default")
		if data.UserRequest != "" {
			if s := sanitizeSheetName(data.UserRequest); s != "" {
				sheetName = s
			}
		}

		excelBytes, err = excelService.ExportTableToExcel(exportTableData, sheetName)
		if err != nil {
			return fmt.Errorf("%s", i18n.T("dashboard.generate_excel_failed", err))
		}
	}

	err = os.WriteFile(savePath, excelBytes, 0644)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.write_excel_failed", err))
	}

	e.log(fmt.Sprintf("Dashboard data exported to Excel successfully: %s (%d tables)", savePath, max(len(data.AllTableData), 1)))
	return nil
}

// ExportDashboardToPPT exports dashboard data to PowerPoint format
func (e *ExportFacadeService) ExportDashboardToPPT(data DashboardExportData) error {
	defaultFilename := generateExportFilename(data.DataSourceName, data.MessageID, "pptx")

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("dashboard.export_ppt_title"),
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: i18n.T("dashboard.filter_ppt"), Pattern: "*.pptx"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	pptService := export.NewGoPPTService()

	exportData := export.DashboardData{
		UserRequest:    data.UserRequest,
		DataSourceName: data.DataSourceName,
		Metrics:        make([]export.MetricData, len(data.Metrics)),
		Insights:       data.Insights,
		ChartImages:    data.ChartImages,
	}

	for i, metric := range data.Metrics {
		exportData.Metrics[i] = export.MetricData{
			Title:  metric.Title,
			Value:  metric.Value,
			Change: metric.Change,
		}
	}

	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		exportData.TableData = &export.TableData{
			Columns: make([]export.TableColumn, len(data.TableData.Columns)),
			Data:    data.TableData.Data,
		}
		for i, col := range data.TableData.Columns {
			exportData.TableData.Columns[i] = export.TableColumn{
				Title:    col.Title,
				DataType: col.DataType,
			}
		}
	}

	if len(exportData.ChartImages) == 0 && data.ChartImage != "" {
		exportData.ChartImages = []string{data.ChartImage}
	}

	pptBytes, err := pptService.ExportDashboardToPPT(exportData)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.generate_ppt_failed", err))
	}

	err = os.WriteFile(savePath, pptBytes, 0644)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.write_ppt_failed", err))
	}

	e.log(fmt.Sprintf("PPT exported successfully to: %s", savePath))
	return nil
}

// ExportDashboardToWord exports dashboard data to Word format
func (e *ExportFacadeService) ExportDashboardToWord(data DashboardExportData) error {
	defaultFilename := generateExportFilename(data.DataSourceName, data.MessageID, "docx")

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("dashboard.export_word_title"),
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: i18n.T("dashboard.filter_word"), Pattern: "*.docx"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	wordService := export.NewWordExportService()

	exportData := export.DashboardData{
		UserRequest:    data.UserRequest,
		DataSourceName: data.DataSourceName,
		Metrics:        make([]export.MetricData, len(data.Metrics)),
		Insights:       data.Insights,
		ChartImages:    data.ChartImages,
	}

	for i, metric := range data.Metrics {
		exportData.Metrics[i] = export.MetricData{
			Title:  metric.Title,
			Value:  metric.Value,
			Change: metric.Change,
		}
	}

	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		exportData.TableData = &export.TableData{
			Columns: make([]export.TableColumn, len(data.TableData.Columns)),
			Data:    data.TableData.Data,
		}
		for i, col := range data.TableData.Columns {
			exportData.TableData.Columns[i] = export.TableColumn{
				Title:    col.Title,
				DataType: col.DataType,
			}
		}
	}

	wordBytes, err := wordService.ExportDashboardToWord(exportData)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.generate_word_failed", err))
	}

	err = os.WriteFile(savePath, wordBytes, 0644)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.write_word_failed", err))
	}

	e.log(fmt.Sprintf("Word exported successfully to: %s", savePath))
	return nil
}

// --- Session File Export Methods ---

// ExportSessionFilesToZip exports all session files to a ZIP archive
func (e *ExportFacadeService) ExportSessionFilesToZip(threadID string, messageID string) error {
	if e.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}

	filesDir := e.chatService.GetSessionFilesDirectory(threadID)

	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		return fmt.Errorf("no files found for this session")
	}

	allFiles, err := e.chatService.GetSessionFiles(threadID)
	if err != nil {
		return fmt.Errorf("failed to get session files: %w", err)
	}

	var files []SessionFile
	if messageID != "" {
		for _, file := range allFiles {
			if file.MessageID == messageID {
				files = append(files, file)
			}
		}
		e.log(fmt.Sprintf("Filtered %d files for message %s (from %d total files)", len(files), messageID, len(allFiles)))
	} else {
		files = allFiles
		e.log(fmt.Sprintf("Exporting all %d files (no message filter)", len(files)))
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to export for this request")
	}

	timestamp := time.Now().Format("20060102_150405")
	var outputFilename string
	if messageID != "" {
		outputFilename = fmt.Sprintf("request_files_%s.zip", timestamp)
	} else {
		outputFilename = fmt.Sprintf("session_files_%s.zip", timestamp)
	}

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		DefaultFilename: outputFilename,
		Title:           "Export Data Files",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "ZIP Archive (*.zip)",
				Pattern:     "*.zip",
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to show save dialog: %w", err)
	}

	if savePath == "" {
		return nil
	}

	zipFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create ZIP file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	successCount := 0
	for _, file := range files {
		filePath := filepath.Join(filesDir, file.Name)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			e.log(fmt.Sprintf("Warning: File not found: %s", filePath))
			continue
		}

		sourceFile, err := os.Open(filePath)
		if err != nil {
			e.log(fmt.Sprintf("Warning: Failed to open file %s: %v", filePath, err))
			continue
		}

		zipFileWriter, err := zipWriter.Create(file.Name)
		if err != nil {
			sourceFile.Close()
			e.log(fmt.Sprintf("Warning: Failed to create ZIP entry for %s: %v", file.Name, err))
			continue
		}

		_, err = io.Copy(zipFileWriter, sourceFile)
		sourceFile.Close()

		if err != nil {
			e.log(fmt.Sprintf("Warning: Failed to write file %s to ZIP: %v", file.Name, err))
			continue
		}

		successCount++
		e.log(fmt.Sprintf("Added file to ZIP: %s", file.Name))
	}

	if successCount == 0 {
		return fmt.Errorf("failed to add any files to ZIP")
	}

	e.log(fmt.Sprintf("Successfully exported %d files to: %s", successCount, savePath))
	return nil
}

// DownloadSessionFile downloads a single session file with save dialog
func (e *ExportFacadeService) DownloadSessionFile(threadID string, fileName string) error {
	if e.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}

	filesDir := e.chatService.GetSessionFilesDirectory(threadID)
	sourceFilePath := filepath.Join(filesDir, fileName)

	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", fileName)
	}

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		DefaultFilename: fileName,
		Title:           "Save File",
	})

	if err != nil {
		return fmt.Errorf("failed to show save dialog: %w", err)
	}

	if savePath == "" {
		return nil
	}

	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	e.log(fmt.Sprintf("File downloaded successfully: %s -> %s", fileName, savePath))
	return nil
}

// GetSessionFileAsBase64 reads a session file and returns it as base64 encoded string
func (e *ExportFacadeService) GetSessionFileAsBase64(threadID string, fileName string) (string, error) {
	if e.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	filesDir := e.chatService.GetSessionFilesDirectory(threadID)
	filePath := filepath.Join(filesDir, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		files, listErr := os.ReadDir(filesDir)
		if listErr != nil {
			return "", fmt.Errorf("file not found: %s (directory read error: %v)", fileName, listErr)
		}

		var matchedFile string
		var latestModTime int64
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if strings.HasSuffix(name, "_"+fileName) {
				info, infoErr := f.Info()
				if infoErr == nil {
					modTime := info.ModTime().UnixNano()
					if modTime > latestModTime {
						latestModTime = modTime
						matchedFile = name
					}
				} else if matchedFile == "" {
					matchedFile = name
				}
			}
		}

		if matchedFile != "" {
			filePath = filepath.Join(filesDir, matchedFile)
			e.log(fmt.Sprintf("[GetSessionFileAsBase64] Resolved '%s' to '%s'", fileName, matchedFile))
		} else {
			return "", fmt.Errorf("file not found: %s", fileName)
		}
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	var mimeType string
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".svg":
		mimeType = "image/svg+xml"
	case ".webp":
		mimeType = "image/webp"
	default:
		mimeType = "application/octet-stream"
	}

	base64Data := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(fileData))
	return base64Data, nil
}

// --- File Preview Methods ---

// GenerateCSVThumbnail generates a text preview for CSV file
func (e *ExportFacadeService) GenerateCSVThumbnail(threadID string, fileName string) (string, error) {
	if e.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	filesDir := e.chatService.GetSessionFilesDirectory(threadID)
	filePath := filepath.Join(filesDir, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fileName)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("CSV file is empty")
	}

	maxRows := 5
	if len(records) > maxRows {
		records = records[:maxRows]
	}

	maxCols := 4
	for i := range records {
		if len(records[i]) > maxCols {
			records[i] = records[i][:maxCols]
		}
	}

	return "", nil
}

// GenerateFilePreview generates structured preview data for Excel, PPT, and CSV files
func (e *ExportFacadeService) GenerateFilePreview(threadID string, fileName string) (string, error) {
	if e.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	filesDir := e.chatService.GetSessionFilesDirectory(threadID)
	filePath := filepath.Join(filesDir, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		files, listErr := os.ReadDir(filesDir)
		if listErr != nil {
			return "", fmt.Errorf("file not found: %s", fileName)
		}
		var matchedFile string
		var latestModTime int64
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if strings.HasSuffix(f.Name(), "_"+fileName) {
				info, infoErr := f.Info()
				if infoErr == nil {
					modTime := info.ModTime().UnixNano()
					if modTime > latestModTime {
						latestModTime = modTime
						matchedFile = f.Name()
					}
				} else if matchedFile == "" {
					matchedFile = f.Name()
				}
			}
		}
		if matchedFile != "" {
			filePath = filepath.Join(filesDir, matchedFile)
		} else {
			return "", fmt.Errorf("file not found: %s", fileName)
		}
	}

	ext := strings.ToLower(filepath.Ext(fileName))

	switch ext {
	case ".xlsx", ".xls":
		return e.generateExcelPreview(filePath, fileName)
	case ".pptx":
		return e.generatePPTPreview(filePath, fileName)
	case ".csv":
		return e.generateCSVPreview(filePath, fileName)
	default:
		return "", fmt.Errorf("preview not supported for file type: %s", ext)
	}
}

// generateExcelPreview reads an Excel file and returns table preview data
func (e *ExportFacadeService) generateExcelPreview(filePath string, fileName string) (string, error) {
	wb, err := gospreadsheet.OpenFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open Excel file: %w", err)
	}

	ws := wb.GetActiveSheet()
	if ws == nil {
		return "", fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := ws.RowIterator()
	if err != nil {
		return "", fmt.Errorf("failed to read Excel rows: %w", err)
	}

	if len(rows) == 0 {
		return "", fmt.Errorf("Excel file is empty")
	}

	var allRows [][]string
	for _, row := range rows {
		var strRow []string
		for _, cell := range row {
			if cell != nil {
				strRow = append(strRow, cell.GetStringValue())
			} else {
				strRow = append(strRow, "")
			}
		}
		allRows = append(allRows, strRow)
	}

	if len(allRows) == 0 {
		return "", fmt.Errorf("Excel file is empty")
	}

	preview := FilePreviewData{
		Type:      "table",
		Title:     fileName,
		TotalRows: len(allRows) - 1,
		TotalCols: len(allRows[0]),
	}

	maxCols := 6
	headers := allRows[0]
	if len(headers) > maxCols {
		headers = headers[:maxCols]
	}
	preview.Headers = headers

	maxRows := 8
	dataRows := allRows[1:]
	if len(dataRows) > maxRows {
		dataRows = dataRows[:maxRows]
	}

	for _, row := range dataRows {
		displayRow := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				val := row[i]
				if len([]rune(val)) > 20 {
					val = string([]rune(val)[:18]) + ".."
				}
				displayRow[i] = val
			}
		}
		preview.Rows = append(preview.Rows, displayRow)
	}

	jsonBytes, err := json.Marshal(preview)
	if err != nil {
		return "", fmt.Errorf("failed to marshal preview: %w", err)
	}
	return string(jsonBytes), nil
}

// generatePPTPreview reads a PPTX file and returns slide preview data
func (e *ExportFacadeService) generatePPTPreview(filePath string, fileName string) (string, error) {
	reader := &ppt.PPTXReader{}
	pres, err := reader.Read(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PPT file: %w", err)
	}

	slides := pres.GetAllSlides()
	if len(slides) == 0 {
		return "", fmt.Errorf("PPT file has no slides")
	}

	preview := FilePreviewData{
		Type:  "slides",
		Title: fileName,
	}

	maxSlides := 3
	if len(slides) < maxSlides {
		maxSlides = len(slides)
	}

	for i := 0; i < maxSlides; i++ {
		slide := slides[i]
		sp := SlidePreview{}

		for _, shape := range slide.GetShapes() {
			if rts, ok := shape.(*ppt.RichTextShape); ok {
				for _, para := range rts.GetParagraphs() {
					var text string
					for _, elem := range para.GetElements() {
						if run, ok := elem.(*ppt.TextRun); ok {
							text += run.GetText()
						}
					}
					text = strings.TrimSpace(text)
					if text == "" {
						continue
					}
					if sp.Title == "" {
						sp.Title = text
					} else {
						if len([]rune(text)) > 60 {
							text = string([]rune(text)[:58]) + ".."
						}
						sp.Texts = append(sp.Texts, text)
					}
				}
			}
		}

		if sp.Title != "" || len(sp.Texts) > 0 {
			preview.Slides = append(preview.Slides, sp)
		}
	}

	jsonBytes, err := json.Marshal(preview)
	if err != nil {
		return "", fmt.Errorf("failed to marshal preview: %w", err)
	}
	return string(jsonBytes), nil
}

// generateCSVPreview reads a CSV file and returns table preview data
func (e *ExportFacadeService) generateCSVPreview(filePath string, fileName string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("CSV file is empty")
	}

	preview := FilePreviewData{
		Type:      "table",
		Title:     fileName,
		TotalRows: len(records) - 1,
		TotalCols: len(records[0]),
	}

	maxCols := 6
	headers := records[0]
	if len(headers) > maxCols {
		headers = headers[:maxCols]
	}
	preview.Headers = headers

	maxRows := 8
	dataRows := records[1:]
	if len(dataRows) > maxRows {
		dataRows = dataRows[:maxRows]
	}

	for _, row := range dataRows {
		displayRow := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				val := row[i]
				if len([]rune(val)) > 20 {
					val = string([]rune(val)[:18]) + ".."
				}
				displayRow[i] = val
			}
		}
		preview.Rows = append(preview.Rows, displayRow)
	}

	jsonBytes, err := json.Marshal(preview)
	if err != nil {
		return "", fmt.Errorf("failed to marshal preview: %w", err)
	}
	return string(jsonBytes), nil
}

// --- Table and Message Export ---

// ExportTableToExcel exports table data to Excel format
func (e *ExportFacadeService) ExportTableToExcel(tableData *TableData, sheetName string) error {
	if tableData == nil || len(tableData.Columns) == 0 {
		return fmt.Errorf("no table data to export")
	}

	timestamp := time.Now().Format("20060102_150405")
	defaultFilename := fmt.Sprintf("table_%s.xlsx", timestamp)
	if sheetName != "" {
		defaultFilename = fmt.Sprintf("%s_%s.xlsx", sheetName, timestamp)
	}

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("dashboard.export_table_title"),
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: i18n.T("dashboard.filter_excel"), Pattern: "*.xlsx"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	excelService := export.NewExcelExportService()

	exportTableData := &export.TableData{
		Columns: make([]export.TableColumn, len(tableData.Columns)),
		Data:    tableData.Data,
	}
	for i, col := range tableData.Columns {
		exportTableData.Columns[i] = export.TableColumn{
			Title:    col.Title,
			DataType: col.DataType,
		}
	}

	excelBytes, err := excelService.ExportTableToExcel(exportTableData, sheetName)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.generate_excel_failed", err))
	}

	err = os.WriteFile(savePath, excelBytes, 0644)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.write_excel_failed", err))
	}

	e.log(fmt.Sprintf("Table exported to Excel successfully: %s", savePath))
	return nil
}

// ExportMessageToPDF exports a chat message (LLM output) to PDF
func (e *ExportFacadeService) ExportMessageToPDF(content string, messageID string) error {
	if content == "" {
		return fmt.Errorf("%s", i18n.T("dashboard.no_exportable_content"))
	}

	timestamp := time.Now().Format("20060102_150405")
	defaultFilename := fmt.Sprintf("analysis_%s.pdf", timestamp)
	if messageID != "" {
		shortID := messageID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		defaultFilename = fmt.Sprintf("analysis_%s_%s.pdf", shortID, timestamp)
	}

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		Title:           i18n.T("dashboard.export_message_pdf_title"),
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: i18n.T("dashboard.filter_pdf"), Pattern: "*.pdf"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	pdfService := export.NewPDFExportService()

	lines := strings.Split(content, "\n")
	var insights []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			insights = append(insights, trimmed)
		}
	}

	exportData := export.DashboardData{
		UserRequest: i18n.T("dashboard.export_result_label"),
		Insights:    insights,
	}

	pdfBytes, err := pdfService.ExportDashboardToPDF(exportData)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.generate_pdf_failed", err))
	}

	err = os.WriteFile(savePath, pdfBytes, 0644)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.write_pdf_failed", err))
	}

	e.log(fmt.Sprintf("Message exported to PDF successfully: %s", savePath))
	return nil
}

// --- Comprehensive Report Methods ---

// exportComputeAnalysisHash computes a hash of all analysis content to detect changes
func exportComputeAnalysisHash(contents []string, tableCount int) string {
	hasher := md5.New()
	for _, content := range contents {
		hasher.Write([]byte(content))
	}
	hasher.Write([]byte(fmt.Sprintf("tables:%d", tableCount)))
	return hex.EncodeToString(hasher.Sum(nil))
}

// PrepareComprehensiveReport prepares a comprehensive report and caches it
func (e *ExportFacadeService) PrepareComprehensiveReport(req ComprehensiveReportRequest) (*ComprehensiveReportResult, error) {
	if e.einoService == nil {
		return nil, fmt.Errorf("%s", i18n.T("report.llm_not_initialized"))
	}

	if e.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}

	e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Starting report preparation for thread: %s", req.ThreadID))

	thread, err := e.chatService.LoadThread(req.ThreadID)
	if err != nil {
		return nil, fmt.Errorf("failed to load thread: %v", err)
	}

	if thread == nil {
		return nil, fmt.Errorf("thread not found: %s", req.ThreadID)
	}

	var analysisContents []string
	var chartImages []string
	var allTableData []NamedTableData
	chartImageSet := make(map[string]bool)

	suggestionPatterns := []string{
		"请给出一些本数据源的分析建议",
		"Give me some analysis suggestions for this data source",
	}

	if thread.IsReplaySession {
		e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Replay session detected, collecting from assistant messages"))
		for _, msg := range thread.Messages {
			if msg.Role != "assistant" {
				continue
			}
			trimmed := strings.TrimSpace(msg.Content)
			if strings.HasPrefix(trimmed, "✅ 快捷分析包执行完成") || strings.HasPrefix(trimmed, "⏭️") {
				continue
			}
			if strings.HasPrefix(trimmed, "❌") {
				continue
			}

			if idx := strings.Index(msg.Content, "📋 分析请求："); idx >= 0 {
				reqStart := idx + len("📋 分析请求：")
				reqEnd := strings.Index(msg.Content[reqStart:], "\n")
				if reqEnd > 0 {
					userReq := strings.TrimSpace(msg.Content[reqStart : reqStart+reqEnd])
					if userReq != "" {
						analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.analysis_request"), userReq))
					}
				}
			}

			if msg.Content != "" {
				analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.analysis_result"), msg.Content))
			}

			if e.getMessageAnalysisDataFn != nil {
				analysisData, err := e.getMessageAnalysisDataFn(req.ThreadID, msg.ID)
				if err != nil {
					e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Failed to get analysis data for replay message %s: %v", msg.ID, err))
				}
				if analysisData != nil {
					e.collectAnalysisResultItems(analysisData, msg.ID, &analysisContents, &allTableData, true)
				}
			}
		}
	} else {
		for i, msg := range thread.Messages {
			if msg.Role == "user" {
				isSuggestionRequest := false
				trimmedContent := strings.TrimSpace(msg.Content)
				for _, pattern := range suggestionPatterns {
					if strings.Contains(trimmedContent, pattern) {
						isSuggestionRequest = true
						break
					}
				}
				if isSuggestionRequest {
					e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Skipping suggestion request message: %s", msg.ID))
					continue
				}

				var analysisData map[string]interface{}
				if e.getMessageAnalysisDataFn != nil {
					analysisData, err = e.getMessageAnalysisDataFn(req.ThreadID, msg.ID)
					if err != nil {
						e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Failed to get analysis data for message %s: %v", msg.ID, err))
					}
				}

				analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.analysis_request"), msg.Content))

				if i+1 < len(thread.Messages) && thread.Messages[i+1].Role == "assistant" {
					assistantMsg := thread.Messages[i+1]
					if assistantMsg.Content != "" {
						matches := base64ImageRegex.FindAllStringSubmatch(assistantMsg.Content, -1)
						for _, match := range matches {
							if len(match) > 1 && !chartImageSet[match[1]] {
								chartImages = append(chartImages, match[1])
								chartImageSet[match[1]] = true
							}
						}

						cleanContent := base64ImageRegex.ReplaceAllString(assistantMsg.Content, "[图表]")
						analysisContents = append(analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.analysis_result"), cleanContent))

						e.extractJsonTablesFromContent(assistantMsg.Content, &allTableData, &analysisContents)
						e.extractMarkdownTablesFromAssistantContent(assistantMsg.Content, &allTableData, &analysisContents)
					}
				}

				if analysisData != nil {
					e.collectAnalysisResultItems(analysisData, msg.ID, &analysisContents, &allTableData, false)
				}
			}
		}
	}

	// Collect chart images from session files
	sessionFiles, err := e.chatService.GetSessionFiles(req.ThreadID)
	if err == nil {
		sessionDir := e.chatService.GetSessionFilesDirectory(req.ThreadID)
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

	if len(req.ChartImages) > 0 {
		e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Received %d chart images from frontend", len(req.ChartImages)))
		for _, img := range req.ChartImages {
			if !chartImageSet[img] {
				chartImages = append(chartImages, img)
				chartImageSet[img] = true
			}
		}
	}

	if len(analysisContents) == 0 {
		e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] No valid analysis contents found. Thread has %d messages.", len(thread.Messages)))
		for idx, msg := range thread.Messages {
			e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT]   Message[%d]: role=%s, id=%s, contentLen=%d", idx, msg.Role, msg.ID, len(msg.Content)))
		}
		return nil, fmt.Errorf("%s", i18n.T("comprehensive_report.no_valid_analysis"))
	}

	contentHash := exportComputeAnalysisHash(analysisContents, len(allTableData))
	reportID := fmt.Sprintf("comprehensive_%s", req.ThreadID)

	e.comprehensiveReportCacheMu.Lock()
	if cached, ok := e.comprehensiveReportCache[reportID]; ok {
		if cached.ContentHash == contentHash {
			if len(req.ChartImages) > 0 || len(chartImages) > 0 {
				allCharts := chartImages
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
			e.comprehensiveReportCacheMu.Unlock()
			e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Using cached report: %s (updated %d chart images)", reportID, len(cached.ExportData.ChartImages)))
			return &ComprehensiveReportResult{
				ReportID: reportID,
				Cached:   true,
			}, nil
		}
	}
	e.comprehensiveReportCacheMu.Unlock()

	e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Generating new report, collected %d analysis contents, %d tables, %d charts",
		len(analysisContents), len(allTableData), len(chartImages)))

	var packMeta *PackMetadata
	if thread.IsReplaySession && thread.PackMetadata != nil {
		packMeta = thread.PackMetadata
		if req.DataSourceName == "" && packMeta.SourceName != "" {
			req.DataSourceName = packMeta.SourceName
			e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Using pack source name as data source: %s", req.DataSourceName))
		}
	}

	comprehensiveSummary := buildComprehensiveSummary(req.DataSourceName, req.SessionName, analysisContents, packMeta)

	reportText, err := e.callLLMForComprehensiveReport(comprehensiveSummary)
	if err != nil {
		e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] LLM report generation failed: %v", err))
		return nil, fmt.Errorf("%s: %v", i18n.T("report.generation_failed"), err)
	}

	e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] LLM generated report text: %d chars", len(reportText)))

	parsed := parseReportSections(reportText)
	e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Parsed report: title=%q, sections=%d", parsed.ReportTitle, len(parsed.Sections)))

	exportData := buildComprehensiveReportExportData(req, parsed, chartImages, allTableData)

	e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Export data: insights=%d, insightsLen=%d, charts=%d, tables=%d",
		len(exportData.Insights),
		func() int {
			total := 0
			for _, s := range exportData.Insights {
				total += len(s)
			}
			return total
		}(),
		len(exportData.ChartImages),
		len(exportData.AllTableData)))

	e.comprehensiveReportCacheMu.Lock()
	if len(e.comprehensiveReportCache) >= 10 {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range e.comprehensiveReportCache {
			if oldestKey == "" || v.CreatedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.CreatedAt
			}
		}
		if oldestKey != "" {
			delete(e.comprehensiveReportCache, oldestKey)
		}
	}
	e.comprehensiveReportCache[reportID] = &cachedComprehensiveReport{
		ExportData:     exportData,
		CreatedAt:      time.Now(),
		ContentHash:    contentHash,
		DataSourceName: req.DataSourceName,
		SessionName:    req.SessionName,
	}
	e.comprehensiveReportCacheMu.Unlock()

	e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Report prepared and cached: %s", reportID))
	return &ComprehensiveReportResult{
		ReportID: reportID,
		Cached:   false,
	}, nil
}

// ExportComprehensiveReport exports a previously prepared comprehensive report
func (e *ExportFacadeService) ExportComprehensiveReport(reportID string, format string) error {
	e.comprehensiveReportCacheMu.Lock()
	cached, ok := e.comprehensiveReportCache[reportID]
	e.comprehensiveReportCacheMu.Unlock()

	if !ok {
		return fmt.Errorf("%s", i18n.T("report.data_expired"))
	}

	timestamp := time.Now().Format("20060102_150405")
	_ = timestamp

	switch format {
	case "pdf":
		return e.doExportComprehensivePDF(cached.ExportData, cached.DataSourceName, cached.SessionName)
	default:
		return e.doExportComprehensiveWord(cached.ExportData, cached.DataSourceName, cached.SessionName)
	}
}

// GenerateComprehensiveReport is kept for backward compatibility
func (e *ExportFacadeService) GenerateComprehensiveReport(req ComprehensiveReportRequest) error {
	result, err := e.PrepareComprehensiveReport(req)
	if err != nil {
		return err
	}
	return e.ExportComprehensiveReport(result.ReportID, "word")
}

// callLLMForComprehensiveReport calls the LLM to generate a comprehensive report
func (e *ExportFacadeService) callLLMForComprehensiveReport(summary string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	systemPrompt := i18n.GetComprehensiveReportSystemPrompt()
	userPrompt := fmt.Sprintf(i18n.GetComprehensiveReportUserPromptTemplate(), summary)

	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: userPrompt},
	}

	resp, err := e.einoService.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	content := resp.Content
	content = strings.TrimSpace(content)

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

// doExportComprehensiveWord exports comprehensive report as Word document
func (e *ExportFacadeService) doExportComprehensiveWord(exportData export.DashboardData, dataSourceName, sessionName string) error {
	safeDSName := sanitizeFileName(dataSourceName)
	safeSessionName := sanitizeFileName(sessionName)

	defaultFilename := fmt.Sprintf("%s_%s_%s.docx", i18n.T("comprehensive_report.filename_prefix"), safeDSName, safeSessionName)

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
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

	e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Word report saved: %s", savePath))
	return nil
}

// doExportComprehensivePDF exports comprehensive report as PDF document
func (e *ExportFacadeService) doExportComprehensivePDF(exportData export.DashboardData, dataSourceName, sessionName string) error {
	safeDSName := sanitizeFileName(dataSourceName)
	safeSessionName := sanitizeFileName(sessionName)

	defaultFilename := fmt.Sprintf("%s_%s_%s.pdf", i18n.T("comprehensive_report.filename_prefix"), safeDSName, safeSessionName)

	savePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
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

	e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] PDF report saved: %s", savePath))
	return nil
}

// --- Helper Methods for Comprehensive Report ---

// collectAnalysisResultItems processes analysis data items and adds them to the report
func (e *ExportFacadeService) collectAnalysisResultItems(analysisData map[string]interface{}, msgID string, analysisContents *[]string, allTableData *[]NamedTableData, isReplay bool) {
	items, ok := analysisData["analysisResults"]
	if items == nil || !ok {
		return
	}
	resultItems, ok := items.([]AnalysisResultItem)
	if !ok {
		return
	}

	e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Message %s has %d analysis result items", msgID, len(resultItems)))

	for _, item := range resultItems {
		if !isReplay {
			e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT]   Item %s: type=%s, dataType=%T", item.ID, item.Type, item.Data))
		}
		switch item.Type {
		case "echarts":
			e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Found echarts item %s - will be rendered by frontend", item.ID))
		case "table":
			tableMap := make(map[string]interface{})
			switch td := item.Data.(type) {
			case string:
				if err := json.Unmarshal([]byte(td), &tableMap); err != nil {
					if !isReplay {
						e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Failed to parse table JSON: %v", err))
					}
					continue
				}
			case map[string]interface{}:
				tableMap = td
			default:
				continue
			}

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
					tableName := fmt.Sprintf("%s %d", i18n.T("comprehensive_report.table"), len(*allTableData)+1)
					if item.Metadata != nil {
						if name, ok := item.Metadata["name"].(string); ok && name != "" {
							tableName = name
						}
					}
					*allTableData = append(*allTableData, NamedTableData{
						Name:  tableName,
						Table: TableData{Columns: tableCols, Data: tableRows},
					})
					var colNames []string
					for _, c := range tableCols {
						colNames = append(colNames, c.Title)
					}
					*analysisContents = append(*analysisContents, fmt.Sprintf("### %s\n%s: %s (%d %s)",
						i18n.T("comprehensive_report.table"), tableName,
						strings.Join(colNames, ", "), len(tableRows), "rows"))
				}
			} else if rows, rowsOk := tableMap["rows"].([]interface{}); rowsOk && len(rows) > 0 {
				tableTitle, _ := tableMap["title"].(string)
				if tableTitle == "" {
					tableTitle = fmt.Sprintf("%s %d", i18n.T("comprehensive_report.table"), len(*allTableData)+1)
				}
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
						*allTableData = append(*allTableData, NamedTableData{
							Name:  tableTitle,
							Table: TableData{Columns: tableCols, Data: tableRows},
						})
						*analysisContents = append(*analysisContents, fmt.Sprintf("### %s\n%s: %s (%d %s)",
							i18n.T("comprehensive_report.table"), tableTitle,
							strings.Join(colOrder, ", "), len(tableRows), "rows"))
					}
				}
			}
		case "insight":
			if strData, ok := item.Data.(string); ok && strData != "" {
				*analysisContents = append(*analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.insight"), strData))
			} else if mapData, ok := item.Data.(map[string]interface{}); ok {
				if text, ok := mapData["text"].(string); ok && text != "" {
					*analysisContents = append(*analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.insight"), text))
				}
			} else if insightObj, ok := item.Data.(Insight); ok && insightObj.Text != "" {
				*analysisContents = append(*analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.insight"), insightObj.Text))
			}
		case "metric":
			if mapData, ok := item.Data.(map[string]interface{}); ok {
				title, _ := mapData["title"].(string)
				value, _ := mapData["value"].(string)
				if title != "" && value != "" {
					change, _ := mapData["change"].(string)
					metricText := fmt.Sprintf("%s: %s", title, value)
					if change != "" {
						metricText += fmt.Sprintf(" (%s)", change)
					}
					*analysisContents = append(*analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.key_metric"), metricText))
				}
			} else if strData, ok := item.Data.(string); ok && strData != "" {
				*analysisContents = append(*analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.key_metric"), strData))
			}
		case "csv":
			if strData, ok := item.Data.(string); ok && strData != "" {
				*analysisContents = append(*analysisContents, fmt.Sprintf("### %s\n%s", i18n.T("comprehensive_report.table"), strData))
			}
		}
	}
}

// extractJsonTablesFromContent extracts json:table blocks from assistant message content
func (e *ExportFacadeService) extractJsonTablesFromContent(content string, allTableData *[]NamedTableData, analysisContents *[]string) {
	reJsonTable := regexp.MustCompile("(?s)```\\s*json:table\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	reJsonTableNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:table\\s*\\n((?:\\{[\\s\\S]+?\\n\\}|\\[[\\s\\S]+?\\n\\]))(?:\\s*\\n(?:---|###)|\\s*$)")
	allJTMatches := reJsonTable.FindAllStringSubmatchIndex(content, -1)
	allJTMatches = append(allJTMatches, reJsonTableNoBT.FindAllStringSubmatchIndex(content, -1)...)

	for _, jtMatch := range allJTMatches {
		if len(jtMatch) >= 4 {
			fullMatchStart := jtMatch[0]
			jsonContent := strings.TrimSpace(content[jtMatch[2]:jtMatch[3]])

			tableTitle := ""
			if fullMatchStart > 0 {
				textBefore := content[:fullMatchStart]
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
				tableTitle = fmt.Sprintf("%s %d", i18n.T("comprehensive_report.table"), len(*allTableData)+1)
			}

			var tableData []map[string]interface{}
			if err := json.Unmarshal([]byte(jsonContent), &tableData); err != nil {
				var colDataFormat struct {
					Columns []string        `json:"columns"`
					Data    [][]interface{} `json:"data"`
				}
				if err2 := json.Unmarshal([]byte(jsonContent), &colDataFormat); err2 == nil && len(colDataFormat.Columns) > 0 && len(colDataFormat.Data) > 0 {
					tableData = make([]map[string]interface{}, 0, len(colDataFormat.Data))
					for _, row := range colDataFormat.Data {
						rowMap := make(map[string]interface{})
						for i, val := range row {
							if i < len(colDataFormat.Columns) {
								rowMap[colDataFormat.Columns[i]] = val
							}
						}
						tableData = append(tableData, rowMap)
					}
				} else {
					var arrayData [][]interface{}
					if err3 := json.Unmarshal([]byte(jsonContent), &arrayData); err3 == nil && len(arrayData) > 1 {
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
			}

			if len(tableData) > 0 {
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
					*allTableData = append(*allTableData, NamedTableData{
						Name: tableTitle,
						Table: TableData{
							Columns: tableCols,
							Data:    tableRows,
						},
					})
					e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Extracted json:table from content: %s (%d rows)", tableTitle, len(tableRows)))
				}
			}
		}
	}
}

// extractMarkdownTablesFromAssistantContent extracts markdown tables from assistant message content
func (e *ExportFacadeService) extractMarkdownTablesFromAssistantContent(content string, allTableData *[]NamedTableData, analysisContents *[]string) {
	mdTables := extractMarkdownTablesFromContent(content)
	for _, mdTable := range mdTables {
		if len(mdTable.Rows) > 0 {
			mdTitle := mdTable.Title
			if mdTitle == "" {
				mdTitle = fmt.Sprintf("%s %d", i18n.T("comprehensive_report.table"), len(*allTableData)+1)
			}
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
				*allTableData = append(*allTableData, NamedTableData{
					Name: mdTitle,
					Table: TableData{
						Columns: tableCols,
						Data:    tableRows,
					},
				})
				e.log(fmt.Sprintf("[COMPREHENSIVE-REPORT] Extracted markdown table from content: %s (%d rows)", mdTitle, len(tableRows)))
			}
		}
	}
}
