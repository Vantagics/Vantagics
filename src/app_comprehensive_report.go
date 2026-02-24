package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"vantagics/export"
	"vantagics/i18n"
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
func (a *App) PrepareComprehensiveReport(req ComprehensiveReportRequest) (*ComprehensiveReportResult, error) {
	if a.exportFacadeService == nil {
		return nil, WrapError("App", "PrepareComprehensiveReport", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.PrepareComprehensiveReport(req)
}

// ExportComprehensiveReport exports a previously prepared comprehensive report
func (a *App) ExportComprehensiveReport(reportID string, format string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportComprehensiveReport", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportComprehensiveReport(reportID, format)
}

// GenerateComprehensiveReport is kept for backward compatibility
// It prepares and immediately exports in Word format
func (a *App) GenerateComprehensiveReport(req ComprehensiveReportRequest) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "GenerateComprehensiveReport", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.GenerateComprehensiveReport(req)
}

func buildComprehensiveSummary(dataSourceName, sessionName string, analysisContents []string, packMeta *PackMetadata) string {
	var sb strings.Builder

	// If pack metadata is available, add analysis pack info section at the beginning
	if packMeta != nil {
		sb.WriteString(i18n.T("comprehensive_report.pack_info_header"))
		sb.WriteString("\n")
		if packMeta.Author != "" {
			sb.WriteString(i18n.T("comprehensive_report.pack_author"))
			sb.WriteString(packMeta.Author)
			sb.WriteString("\n")
		}
		if packMeta.Description != "" {
			sb.WriteString(i18n.T("comprehensive_report.pack_description"))
			sb.WriteString(packMeta.Description)
			sb.WriteString("\n")
		}
		if packMeta.SourceName != "" {
			sb.WriteString(i18n.T("comprehensive_report.pack_source_name"))
			sb.WriteString(packMeta.SourceName)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

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

// sanitizeFileName removes or replaces characters that are invalid in file names
func sanitizeFileName(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	result = strings.TrimSpace(result)
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}
