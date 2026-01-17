package main

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DashboardExportData represents the data structure for dashboard export
type DashboardExportData struct {
	UserRequest string            `json:"userRequest"`
	Metrics     []DashboardMetric `json:"metrics"`
	Insights    []string          `json:"insights"`
	ChartImage  string            `json:"chartImage"`  // base64 image data (single chart, for backward compatibility)
	ChartImages []string          `json:"chartImages"` // base64 image data (multiple charts)
	TableData   *TableData        `json:"tableData"`   // table data if present
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

// ExportDashboardToPDF exports dashboard data to PDF using chromedp
func (a *App) ExportDashboardToPDF(data DashboardExportData) error {
	// Generate HTML for dashboard
	html := generateDashboardHTML(data)
	
	// Create temp HTML file
	tmpDir := os.TempDir()
	htmlPath := filepath.Join(tmpDir, "dashboard_export.html")
	err := os.WriteFile(htmlPath, []byte(html), 0644)
	if err != nil {
		return fmt.Errorf("创建临时HTML文件失败: %v", err)
	}
	defer os.Remove(htmlPath)
	
	// Save dialog
	timestamp := time.Now().Format("20060102_150405")
	defaultFilename := fmt.Sprintf("dashboard_%s.pdf", timestamp)

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出仪表盘为PDF",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF文件", Pattern: "*.pdf"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}
	
	// Use chromedp to render HTML to PDF
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	
	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	var pdfBuf []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate("file://"+htmlPath),
		chromedp.Sleep(1*time.Second), // Wait for rendering
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(false).
				WithPaperWidth(8.27).  // A4 width in inches
				WithPaperHeight(11.69). // A4 height in inches
				Do(ctx)
			return err
		}),
	)
	
	if err != nil {
		return fmt.Errorf("PDF生成失败: %v\n提示：需要安装Chrome浏览器", err)
	}
	
	// Write PDF file
	return os.WriteFile(savePath, pdfBuf, 0644)
}

// generateDashboardHTML creates beautiful HTML for dashboard export
func generateDashboardHTML(data DashboardExportData) string {
	var html strings.Builder
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	html.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>智能仪表盘报告</title>
    <style>
        @page {
            margin: 15mm;
            size: A4;
        }
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Microsoft YaHei', sans-serif;
            line-height: 1.6;
            color: #1e293b;
            background: #ffffff;
            padding: 20px;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            padding: 30px 0;
            border-bottom: 3px solid #3b82f6;
            margin-bottom: 40px;
        }
        .header h1 {
            font-size: 32px;
            color: #3b82f6;
            margin-bottom: 10px;
            font-weight: 700;
        }
        .header .timestamp {
            color: #64748b;
            font-size: 14px;
        }
        .user-request {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 25px;
            border-radius: 12px;
            margin-bottom: 35px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .user-request h2 {
            font-size: 16px;
            margin-bottom: 10px;
            opacity: 0.9;
        }
        .user-request .text {
            font-size: 18px;
            font-weight: 500;
        }
        .section {
            margin-bottom: 35px;
            page-break-inside: avoid;
        }
        .section-title {
            font-size: 22px;
            color: #1e293b;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 2px solid #e2e8f0;
            font-weight: 600;
        }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 20px;
            margin-bottom: 20px;
        }
        .metric-card {
            background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%);
            border: 2px solid #cbd5e1;
            border-radius: 10px;
            padding: 20px;
            text-align: center;
            transition: transform 0.2s;
        }
        .metric-title {
            font-size: 13px;
            color: #64748b;
            margin-bottom: 8px;
            font-weight: 500;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .metric-value {
            font-size: 28px;
            font-weight: bold;
            color: #0f172a;
            margin-bottom: 5px;
        }
        .metric-change {
            font-size: 13px;
            color: #059669;
            font-weight: 600;
        }
        .insights-grid {
            display: grid;
            gap: 15px;
        }
        .insight-item {
            background: #f8fafc;
            border-left: 4px solid #3b82f6;
            padding: 18px 20px;
            border-radius: 6px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
        }
        .insight-item::before {
            content: "💡 ";
            font-size: 18px;
            margin-right: 8px;
        }
        .insight-text {
            color: #475569;
            font-size: 15px;
            line-height: 1.7;
        }
        .chart-section {
            background: #f8fafc;
            border: 2px solid #e2e8f0;
            border-radius: 12px;
            padding: 25px;
            text-align: center;
        }
        .chart-image {
            max-width: 100%;
            height: auto;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            margin-top: 15px;
        }
        .footer {
            text-align: center;
            margin-top: 50px;
            padding-top: 20px;
            border-top: 1px solid #e2e8f0;
            color: #94a3b8;
            font-size: 13px;
        }
        .empty-state {
            text-align: center;
            padding: 40px;
            color: #94a3b8;
            font-style: italic;
        }
        .table-container {
            overflow-x: auto;
            margin: 20px 0;
            border: 1px solid #e2e8f0;
            border-Radius: 8px;
        }
        .data-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 13px;
        }
        .data-table thead {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .data-table th {
            padding: 12px 10px;
            text-align: left;
            font-weight: 600;
            border-bottom: 2px solid #cbd5e1;
        }
        .data-table td {
            padding: 10px;
            border-bottom: 1px solid #e2e8f0;
        }
        .data-table tbody tr:nth-child(even) {
            background-color: #f8fafc;
        }
        .data-table tbody tr:hover {
            background-color: #f1f5f9;
        }
        .table-note {
            text-align: center;
            color: #64748b;
            font-size: 12px;
            font-style: italic;
            margin-top: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 智能仪表盘报告</h1>
            <div class="timestamp">生成时间: ` + timestamp + `</div>
        </div>
`)

	// User Request Section
	if data.UserRequest != "" {
		html.WriteString(`
        <div class="user-request">
            <h2>用户请求</h2>
            <div class="text">` + data.UserRequest + `</div>
        </div>
`)
	}

	// Metrics Section
	if len(data.Metrics) > 0 {
		html.WriteString(`
        <div class="section">
            <h2 class="section-title">关键指标</h2>
            <div class="metrics-grid">
`)
		for _, metric := range data.Metrics {
			html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-title">%s</div>
                    <div class="metric-value">%s</div>
                    <div class="metric-change">%s</div>
                </div>
`, metric.Title, metric.Value, metric.Change))
		}
		html.WriteString(`
            </div>
        </div>
`)
	}

	// Insights Section
	if len(data.Insights) > 0 {
		html.WriteString(`
        <div class="section">
            <h2 class="section-title">智能洞察</h2>
            <div class="insights-grid">
`)
		for _, insight := range data.Insights {
			html.WriteString(fmt.Sprintf(`
                <div class="insight-item">
                    <div class="insight-text">%s</div>
                </div>
`, insight))
		}
		html.WriteString(`
            </div>
        </div>
`)
	}

	// Chart Section - Support multiple charts
	if len(data.ChartImages) > 0 {
		html.WriteString(`
        <div class="section">
            <h2 class="section-title">数据可视化</h2>
`)
		for i, chartImage := range data.ChartImages {
			html.WriteString(fmt.Sprintf(`
            <div class="chart-section" style="margin-bottom: %s;">
                <img src="%s" class="chart-image" alt="图表 %d" />
            </div>
`, func() string {
				if i < len(data.ChartImages)-1 {
					return "20px"
				}
				return "0"
			}(), chartImage, i+1))
		}
		html.WriteString(`
        </div>
`)
	} else if data.ChartImage != "" {
		// Fallback to single chart for backward compatibility
		html.WriteString(`
        <div class="section">
            <h2 class="section-title">数据可视化</h2>
            <div class="chart-section">
                <img src="` + data.ChartImage + `" class="chart-image" alt="图表" />
            </div>
        </div>
`)
	}

	// Table Section
	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		html.WriteString(`
        <div class="section">
            <h2 class="section-title">数据表格</h2>
            <div class="table-container">
                <table class="data-table">
                    <thead>
                        <tr>
`)
		for _, col := range data.TableData.Columns {
			html.WriteString(fmt.Sprintf(`
                            <th>%s</th>
`, col.Title))
		}
		html.WriteString(`
                        </tr>
                    </thead>
                    <tbody>
`)
		// Limit to first 100 rows for PDF
		maxRows := len(data.TableData.Data)
		if maxRows > 100 {
			maxRows = 100
		}
		for i := 0; i < maxRows; i++ {
			row := data.TableData.Data[i]
			html.WriteString(`
                        <tr>
`)
			for _, cell := range row {
				html.WriteString(fmt.Sprintf(`
                            <td>%v</td>
`, cell))
			}
			html.WriteString(`
                        </tr>
`)
		}
		html.WriteString(`
                    </tbody>
                </table>
            </div>
`)
		if len(data.TableData.Data) > 100 {
			html.WriteString(fmt.Sprintf(`
            <p class="table-note">注：仅显示前100行，共%d行数据</p>
`, len(data.TableData.Data)))
		}
		html.WriteString(`
        </div>
`)
	}

	html.WriteString(`
        <div class="footer">
            由 RapidBI 智能分析系统生成
        </div>
    </div>
</body>
</html>`)

	return html.String()
}


// ExportSessionFilesToZip exports all session files to a ZIP archive
// If messageID is provided, only exports files associated with that message
func (a *App) ExportSessionFilesToZip(threadID string, messageID string) error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}

	// Get session files directory
	filesDir := a.chatService.GetSessionFilesDirectory(threadID)
	
	// Check if directory exists
	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		return fmt.Errorf("no files found for this session")
	}

	// Get list of files
	allFiles, err := a.chatService.GetSessionFiles(threadID)
	if err != nil {
		return fmt.Errorf("failed to get session files: %w", err)
	}

	// Filter files by messageID if provided
	var files []SessionFile
	if messageID != "" {
		for _, file := range allFiles {
			if file.MessageID == messageID {
				files = append(files, file)
			}
		}
		a.Log(fmt.Sprintf("Filtered %d files for message %s (from %d total files)", len(files), messageID, len(allFiles)))
	} else {
		files = allFiles
		a.Log(fmt.Sprintf("Exporting all %d files (no message filter)", len(files)))
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to export for this request")
	}

	// Create output filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	var outputFilename string
	if messageID != "" {
		outputFilename = fmt.Sprintf("request_files_%s.zip", timestamp)
	} else {
		outputFilename = fmt.Sprintf("session_files_%s.zip", timestamp)
	}

	// Show save dialog
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
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
		// User cancelled
		return nil
	}

	// Create ZIP file
	zipFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create ZIP file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add each file to ZIP
	successCount := 0
	for _, file := range files {
		filePath := filepath.Join(filesDir, file.Name)
		
		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			a.Log(fmt.Sprintf("Warning: File not found: %s", filePath))
			continue
		}

		// Open source file
		sourceFile, err := os.Open(filePath)
		if err != nil {
			a.Log(fmt.Sprintf("Warning: Failed to open file %s: %v", filePath, err))
			continue
		}

		// Create file in ZIP
		zipFileWriter, err := zipWriter.Create(file.Name)
		if err != nil {
			sourceFile.Close()
			a.Log(fmt.Sprintf("Warning: Failed to create ZIP entry for %s: %v", file.Name, err))
			continue
		}

		// Copy file content to ZIP
		_, err = io.Copy(zipFileWriter, sourceFile)
		sourceFile.Close()
		
		if err != nil {
			a.Log(fmt.Sprintf("Warning: Failed to write file %s to ZIP: %v", file.Name, err))
			continue
		}

		successCount++
		a.Log(fmt.Sprintf("Added file to ZIP: %s", file.Name))
	}

	if successCount == 0 {
		return fmt.Errorf("failed to add any files to ZIP")
	}

	a.Log(fmt.Sprintf("Successfully exported %d files to: %s", successCount, savePath))
	
	return nil
}


// DownloadSessionFile downloads a single session file with save dialog
func (a *App) DownloadSessionFile(threadID string, fileName string) error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}

	// Get source file path
	filesDir := a.chatService.GetSessionFilesDirectory(threadID)
	sourceFilePath := filepath.Join(filesDir, fileName)

	// Check if file exists
	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", fileName)
	}

	// Show save dialog
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: fileName,
		Title:           "Save File",
	})

	if err != nil {
		return fmt.Errorf("failed to show save dialog: %w", err)
	}

	if savePath == "" {
		// User cancelled
		return nil
	}

	// Copy file to selected location
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

	a.Log(fmt.Sprintf("File downloaded successfully: %s -> %s", fileName, savePath))
	
	return nil
}


// GetSessionFileAsBase64 reads a session file and returns it as base64 encoded string
// This is used for displaying image thumbnails in the frontend
func (a *App) GetSessionFileAsBase64(threadID string, fileName string) (string, error) {
	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	// Get file path
	filesDir := a.chatService.GetSessionFilesDirectory(threadID)
	filePath := filepath.Join(filesDir, fileName)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fileName)
	}

	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect MIME type based on file extension
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

	// Encode to base64 with data URI scheme
	base64Data := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(fileData))
	
	return base64Data, nil
}


// GenerateCSVThumbnail generates a thumbnail image for CSV file preview
func (a *App) GenerateCSVThumbnail(threadID string, fileName string) (string, error) {
	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	// Get file path
	filesDir := a.chatService.GetSessionFilesDirectory(threadID)
	filePath := filepath.Join(filesDir, fileName)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fileName)
	}

	// Read CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("CSV file is empty")
	}

	// Limit to first 5 rows (header + 4 data rows)
	maxRows := 5
	if len(records) > maxRows {
		records = records[:maxRows]
	}

	// Limit to first 4 columns
	maxCols := 4
	for i := range records {
		if len(records[i]) > maxCols {
			records[i] = records[i][:maxCols]
		}
	}

	// Generate thumbnail using HTML/CSS approach
	// Create HTML table
	var htmlBuilder strings.Builder
	htmlBuilder.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>
body {
    margin: 0;
    padding: 20px;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    background: white;
}
table {
    border-collapse: collapse;
    width: 100%;
    font-size: 12px;
}
th {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 8px 12px;
    text-align: left;
    font-weight: 600;
    border: 1px solid #5a67d8;
}
td {
    padding: 6px 12px;
    border: 1px solid #e2e8f0;
    color: #1e293b;
}
tr:nth-child(even) {
    background-color: #f8fafc;
}
.truncate {
    max-width: 150px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
</style>
</head>
<body>
<table>
`)

	// Add table rows
	for i, record := range records {
		htmlBuilder.WriteString("<tr>")
		for _, cell := range record {
			// Truncate long text
			cellText := cell
			if len(cellText) > 30 {
				cellText = cellText[:27] + "..."
			}
			// Escape HTML
			cellText = strings.ReplaceAll(cellText, "&", "&amp;")
			cellText = strings.ReplaceAll(cellText, "<", "&lt;")
			cellText = strings.ReplaceAll(cellText, ">", "&gt;")
			
			if i == 0 {
				htmlBuilder.WriteString(fmt.Sprintf("<th class='truncate'>%s</th>", cellText))
			} else {
				htmlBuilder.WriteString(fmt.Sprintf("<td class='truncate'>%s</td>", cellText))
			}
		}
		htmlBuilder.WriteString("</tr>\n")
	}

	htmlBuilder.WriteString(`
</table>
</body>
</html>`)

	// Create temp HTML file
	tmpDir := os.TempDir()
	htmlPath := filepath.Join(tmpDir, fmt.Sprintf("csv_preview_%d.html", time.Now().UnixNano()))
	err = os.WriteFile(htmlPath, []byte(htmlBuilder.String()), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create temp HTML: %w", err)
	}
	defer os.Remove(htmlPath)

	// Use chromedp to render HTML to image
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var buf []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate("file://"+htmlPath),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Screenshot(`table`, &buf, chromedp.NodeVisible, chromedp.ByQuery),
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate screenshot: %w", err)
	}

	// Encode to base64
	base64Data := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf))
	
	return base64Data, nil
}
