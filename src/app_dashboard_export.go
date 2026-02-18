package main

import (
	"fmt"
	"strings"
	"time"
)

// DashboardExportData represents the data structure for dashboard export
type DashboardExportData struct {
	UserRequest    string            `json:"userRequest"`
	DataSourceName string            `json:"dataSourceName"` // 数据源名称
	MessageID      string            `json:"messageId"`      // 分析请求ID
	Metrics        []DashboardMetric `json:"metrics"`
	Insights       []string          `json:"insights"`
	ChartImage     string            `json:"chartImage"`  // base64 image data (single chart, for backward compatibility)
	ChartImages    []string          `json:"chartImages"` // base64 image data (multiple charts)
	TableData      *TableData        `json:"tableData"`   // table data if present (single table, backward compatibility)
	AllTableData   []NamedTableData  `json:"allTableData"` // all tables with names for multi-sheet export
}

// NamedTableData represents a table with a name, used for multi-sheet Excel export
type NamedTableData struct {
	Name  string    `json:"name"`
	Table TableData `json:"table"`
}

type DashboardMetric struct {
	Title  string `json:"title"`
	Value  string `json:"value"`
	Change string `json:"change"`
}

type TableData struct {
	Columns []TableColumn `json:"columns"`
	Data    [][]any       `json:"data"`
}

type TableColumn struct {
	Title    string `json:"title"`
	DataType string `json:"dataType"`
}

// FilePreviewData represents structured preview data for a file
type FilePreviewData struct {
	Type      string         `json:"type"`      // "table" | "slides" | "text"
	Title     string         `json:"title"`     // File name or title
	Headers   []string       `json:"headers"`   // Table headers (for table type)
	Rows      [][]string     `json:"rows"`      // Table rows (for table type)
	Slides    []SlidePreview `json:"slides"`    // Slide previews (for slides type)
	TotalRows int            `json:"totalRows"` // Total row count (for table type)
	TotalCols int            `json:"totalCols"` // Total column count (for table type)
}

// SlidePreview represents a single slide's preview data
type SlidePreview struct {
	Title string   `json:"title"`
	Texts []string `json:"texts"`
}

// generateExportFilename generates a filename for export with datasource name and message ID
func generateExportFilename(dataSourceName string, messageID string, extension string) string {
	timestamp := time.Now().Format("20060102_150405")

	// 清理数据源名称，移除不合法的文件名字符
	cleanDataSourceName := strings.Map(func(r rune) rune {
		if r == ':' || r == '\\' || r == '/' || r == '?' || r == '*' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, dataSourceName)

	// 限制数据源名称长度
	if len([]rune(cleanDataSourceName)) > 30 {
		cleanDataSourceName = string([]rune(cleanDataSourceName)[:30])
	}

	// 截取messageID的前8位
	shortMessageID := messageID
	if len(messageID) > 8 {
		shortMessageID = messageID[:8]
	}

	// 构建文件名
	var filename string
	if cleanDataSourceName != "" && shortMessageID != "" {
		filename = fmt.Sprintf("%s_%s_%s.%s", cleanDataSourceName, shortMessageID, timestamp, extension)
	} else if cleanDataSourceName != "" {
		filename = fmt.Sprintf("%s_%s.%s", cleanDataSourceName, timestamp, extension)
	} else if shortMessageID != "" {
		filename = fmt.Sprintf("analysis_%s_%s.%s", shortMessageID, timestamp, extension)
	} else {
		filename = fmt.Sprintf("dashboard_%s.%s", timestamp, extension)
	}

	return filename
}

// sanitizeSheetName sanitizes a string for use as an Excel sheet name
func sanitizeSheetName(name string) string {
	sanitized := name
	for _, ch := range []string{":", "\\", "/", "?", "*", "[", "]"} {
		sanitized = strings.ReplaceAll(sanitized, ch, " ")
	}
	sanitized = strings.TrimSpace(sanitized)
	runes := []rune(sanitized)
	if len(runes) > 28 {
		runes = runes[:28]
	}
	if len(runes) > 0 {
		return string(runes)
	}
	return ""
}

// ExportDashboardToPDF exports dashboard data to PDF
func (a *App) ExportDashboardToPDF(data DashboardExportData) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportDashboardToPDF", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportDashboardToPDF(data)
}

// ExportSessionFilesToZip exports all session files to a ZIP archive
func (a *App) ExportSessionFilesToZip(threadID string, messageID string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportSessionFilesToZip", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportSessionFilesToZip(threadID, messageID)
}

// DownloadSessionFile downloads a single session file with save dialog
func (a *App) DownloadSessionFile(threadID string, fileName string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "DownloadSessionFile", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.DownloadSessionFile(threadID, fileName)
}

// GetSessionFileAsBase64 reads a session file and returns it as base64 encoded string
func (a *App) GetSessionFileAsBase64(threadID string, fileName string) (string, error) {
	if a.exportFacadeService == nil {
		return "", WrapError("App", "GetSessionFileAsBase64", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.GetSessionFileAsBase64(threadID, fileName)
}

// GenerateCSVThumbnail generates a text preview for CSV file
func (a *App) GenerateCSVThumbnail(threadID string, fileName string) (string, error) {
	if a.exportFacadeService == nil {
		return "", WrapError("App", "GenerateCSVThumbnail", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.GenerateCSVThumbnail(threadID, fileName)
}

// GenerateFilePreview generates structured preview data for Excel, PPT, and CSV files
func (a *App) GenerateFilePreview(threadID string, fileName string) (string, error) {
	if a.exportFacadeService == nil {
		return "", WrapError("App", "GenerateFilePreview", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.GenerateFilePreview(threadID, fileName)
}

// ExportTableToExcel exports table data to Excel format
func (a *App) ExportTableToExcel(tableData *TableData, sheetName string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportTableToExcel", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportTableToExcel(tableData, sheetName)
}

// ExportDashboardToExcel exports dashboard table data to Excel
func (a *App) ExportDashboardToExcel(data DashboardExportData) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportDashboardToExcel", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportDashboardToExcel(data)
}

// ExportMessageToPDF exports a chat message (LLM output) to PDF
func (a *App) ExportMessageToPDF(content string, messageID string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportMessageToPDF", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportMessageToPDF(content, messageID)
}

// ExportDashboardToPPT exports dashboard data to PowerPoint format
func (a *App) ExportDashboardToPPT(data DashboardExportData) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportDashboardToPPT", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportDashboardToPPT(data)
}

// ExportDashboardToWord exports dashboard data to Word format
func (a *App) ExportDashboardToWord(data DashboardExportData) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportDashboardToWord", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportDashboardToWord(data)
}
