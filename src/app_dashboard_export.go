package main

import (
	"archive/zip"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"vantagedata/export"
	"strings"
	"time"

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

// ExportDashboardToPDF exports dashboard data to PDF using gopdf library
func (a *App) ExportDashboardToPDF(data DashboardExportData) error {
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

	// Try using maroto first (faster, no Chrome dependency)
	pdfService := export.NewPDFExportService()
	
	// Convert DashboardExportData to export.DashboardData
	exportData := export.DashboardData{
		UserRequest: data.UserRequest,
		Metrics:     make([]export.MetricData, len(data.Metrics)),
		Insights:    data.Insights,
		ChartImages: data.ChartImages,
	}

	// Convert metrics
	for i, m := range data.Metrics {
		exportData.Metrics[i] = export.MetricData{
			Title:  m.Title,
			Value:  m.Value,
			Change: m.Change,
		}
	}

	// Convert table data if present
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

	// Generate PDF using maroto
	pdfBytes, err := pdfService.ExportDashboardToPDF(exportData)
	if err != nil {
		return fmt.Errorf("PDF生成失败: %v", err)
	}

	// Write PDF file
	err = os.WriteFile(savePath, pdfBytes, 0644)
	if err != nil {
		return fmt.Errorf("写入PDF文件失败: %v", err)
	}

	a.Log(fmt.Sprintf("Dashboard exported to PDF successfully: %s", savePath))
	return nil
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

	// Check if file exists with exact name
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File not found with exact name, try to find a file with unique prefix
		// Files are saved with format: requestId_originalName.ext (e.g., msg_123_chart.png)
		// So if looking for "chart.png", we should find "*_chart.png"
		
		// List all files in the directory
		files, listErr := os.ReadDir(filesDir)
		if listErr != nil {
			return "", fmt.Errorf("file not found: %s (directory read error: %v)", fileName, listErr)
		}
		
		// Look for files that end with _<fileName>
		var matchedFile string
		var latestModTime int64
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			// Check if file ends with _<fileName> (e.g., "msg_123_chart.png" matches "chart.png")
			if strings.HasSuffix(name, "_"+fileName) {
				// Get file info to find the most recent one
				info, infoErr := f.Info()
				if infoErr == nil {
					modTime := info.ModTime().UnixNano()
					if modTime > latestModTime {
						latestModTime = modTime
						matchedFile = name
					}
				} else if matchedFile == "" {
					// If we can't get info, just use the first match
					matchedFile = name
				}
			}
		}
		
		if matchedFile != "" {
			filePath = filepath.Join(filesDir, matchedFile)
			a.Log(fmt.Sprintf("[GetSessionFileAsBase64] Resolved '%s' to '%s'", fileName, matchedFile))
		} else {
			return "", fmt.Errorf("file not found: %s", fileName)
		}
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


// GenerateCSVThumbnail generates a text preview for CSV file
// Note: This returns a JSON string with CSV preview data instead of an image
// since chromedp has been removed from the project
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

	// Return empty string to indicate CSV preview is not available
	// Frontend should handle this by showing a generic CSV icon or text preview
	return "", nil
}


// ExportTableToExcel exports table data to Excel format
func (a *App) ExportTableToExcel(tableData *TableData, sheetName string) error {
	if tableData == nil || len(tableData.Columns) == 0 {
		return fmt.Errorf("no table data to export")
	}

	// Save dialog
	timestamp := time.Now().Format("20060102_150405")
	defaultFilename := fmt.Sprintf("table_%s.xlsx", timestamp)
	if sheetName != "" {
		defaultFilename = fmt.Sprintf("%s_%s.xlsx", sheetName, timestamp)
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出表格为Excel",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Excel文件", Pattern: "*.xlsx"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	// Create Excel export service
	excelService := export.NewExcelExportService()

	// Convert TableData to export.TableData
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

	// Generate Excel file
	excelBytes, err := excelService.ExportTableToExcel(exportTableData, sheetName)
	if err != nil {
		return fmt.Errorf("Excel生成失败: %v", err)
	}

	// Write Excel file
	err = os.WriteFile(savePath, excelBytes, 0644)
	if err != nil {
		return fmt.Errorf("写入Excel文件失败: %v", err)
	}

	a.Log(fmt.Sprintf("Table exported to Excel successfully: %s", savePath))
	return nil
}


// ExportDashboardToExcel exports dashboard table data to Excel
func (a *App) ExportDashboardToExcel(data DashboardExportData) error {
	if data.TableData == nil || len(data.TableData.Columns) == 0 {
		return fmt.Errorf("no table data to export")
	}

	// Save dialog
	timestamp := time.Now().Format("20060102_150405")
	defaultFilename := fmt.Sprintf("dashboard_data_%s.xlsx", timestamp)

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出仪表盘数据为Excel",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Excel文件", Pattern: "*.xlsx"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	// Create Excel export service
	excelService := export.NewExcelExportService()

	// Convert TableData to export.TableData
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

	// Generate Excel file with dashboard info
	sheetName := "数据分析"
	if data.UserRequest != "" {
		// Use first 20 chars of user request as sheet name
		sheetName = data.UserRequest
		if len(sheetName) > 20 {
			sheetName = sheetName[:20]
		}
	}

	excelBytes, err := excelService.ExportTableToExcel(exportTableData, sheetName)
	if err != nil {
		return fmt.Errorf("Excel生成失败: %v", err)
	}

	// Write Excel file
	err = os.WriteFile(savePath, excelBytes, 0644)
	if err != nil {
		return fmt.Errorf("写入Excel文件失败: %v", err)
	}

	a.Log(fmt.Sprintf("Dashboard data exported to Excel successfully: %s", savePath))
	return nil
}


// ExportMessageToPDF exports a chat message (LLM output) to PDF
// This is used for exporting analysis results from the chat area
func (a *App) ExportMessageToPDF(content string, messageID string) error {
	if content == "" {
		return fmt.Errorf("没有可导出的内容")
	}

	// Save dialog
	timestamp := time.Now().Format("20060102_150405")
	defaultFilename := fmt.Sprintf("analysis_%s.pdf", timestamp)
	if messageID != "" {
		// Use shorter message ID for filename
		shortID := messageID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		defaultFilename = fmt.Sprintf("analysis_%s_%s.pdf", shortID, timestamp)
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出分析结果为PDF",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF文件", Pattern: "*.pdf"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	// Create PDF export service
	pdfService := export.NewPDFExportService()

	// Parse content to extract insights (split by newlines, filter empty lines)
	lines := strings.Split(content, "\n")
	var insights []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			insights = append(insights, trimmed)
		}
	}

	// Create export data structure
	exportData := export.DashboardData{
		UserRequest: "分析结果导出",
		Insights:    insights,
	}

	// Generate PDF
	pdfBytes, err := pdfService.ExportDashboardToPDF(exportData)
	if err != nil {
		return fmt.Errorf("PDF生成失败: %v", err)
	}

	// Write PDF file
	err = os.WriteFile(savePath, pdfBytes, 0644)
	if err != nil {
		return fmt.Errorf("写入PDF文件失败: %v", err)
	}

	a.Log(fmt.Sprintf("Message exported to PDF successfully: %s", savePath))
	return nil
}

// ExportDashboardToPPT exports dashboard data to PowerPoint format
func (a *App) ExportDashboardToPPT(data DashboardExportData) error {
	// Save dialog
	timestamp := time.Now().Format("20060102_150405")
	defaultFilename := fmt.Sprintf("dashboard_%s.pptx", timestamp)

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出仪表盘为PPT",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "PowerPoint文件", Pattern: "*.pptx"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	// Use gooxml PPT service (better compatibility)
	pptService := export.NewGooxmlPPTService()

	// Convert DashboardExportData to export.DashboardData
	exportData := export.DashboardData{
		UserRequest: data.UserRequest,
		Metrics:     make([]export.MetricData, len(data.Metrics)),
		Insights:    data.Insights,
		ChartImages: data.ChartImages,
	}

	// Convert metrics
	for i, metric := range data.Metrics {
		exportData.Metrics[i] = export.MetricData{
			Title:  metric.Title,
			Value:  metric.Value,
			Change: metric.Change,
		}
	}

	// Convert table data if present
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

	// Fallback to single chart image if ChartImages is empty
	if len(exportData.ChartImages) == 0 && data.ChartImage != "" {
		exportData.ChartImages = []string{data.ChartImage}
	}

	// Generate PPT using gooxml
	pptBytes, err := pptService.ExportDashboardToPPT(exportData)
	if err != nil {
		return fmt.Errorf("PPT生成失败: %v", err)
	}

	// Write PPT file
	err = os.WriteFile(savePath, pptBytes, 0644)
	if err != nil {
		return fmt.Errorf("写入PPT文件失败: %v", err)
	}

	a.Log(fmt.Sprintf("PPT exported successfully to: %s", savePath))
	return nil
}
