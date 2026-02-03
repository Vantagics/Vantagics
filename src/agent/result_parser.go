package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ResultParser parses Python execution output
type ResultParser struct {
	logger func(string)
}

// ExecutionResult represents the parsed execution result
type ExecutionResult struct {
	Success     bool                     `json:"success"`
	TextOutput  string                   `json:"text_output"`
	Tables      []ParsedTable            `json:"tables"`
	ChartFiles  []FileInfo               `json:"chart_files"`
	ExportFiles []FileInfo               `json:"export_files"`
	ErrorMsg    string                   `json:"error_msg"`
	Warnings    []string                 `json:"warnings"`
}

// ParsedTable represents a parsed data table
type ParsedTable struct {
	Name    string                   `json:"name"`
	Headers []string                 `json:"headers"`
	Rows    []map[string]interface{} `json:"rows"`
}

// FileInfo represents information about a generated file
type FileInfo struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time,omitempty"`
}

// FileDetectionConfig holds configuration for file detection
type FileDetectionConfig struct {
	// MinFileSize is the minimum file size in bytes (files smaller than this are ignored)
	MinFileSize int64
	// MaxFileAge is the maximum age of files to consider (files older than this are ignored)
	MaxFileAge time.Duration
	// SupportedImageExtensions lists all supported image file extensions
	SupportedImageExtensions []string
	// SupportedExportExtensions lists all supported export file extensions
	SupportedExportExtensions []string
}

// DefaultFileDetectionConfig returns the default file detection configuration
func DefaultFileDetectionConfig() *FileDetectionConfig {
	return &FileDetectionConfig{
		MinFileSize: 1, // Minimum 1 byte (filter out empty files)
		MaxFileAge:  24 * time.Hour, // Files modified within the last 24 hours
		SupportedImageExtensions: []string{
			".png", ".jpg", ".jpeg", ".svg", ".gif",
			".webp", ".bmp", ".tiff", ".tif",
		},
		SupportedExportExtensions: []string{
			".csv", ".xlsx", ".xls", ".json",
			".txt", ".html", ".xml",
		},
	}
}

// NewResultParser creates a new result parser
func NewResultParser(logger func(string)) *ResultParser {
	return &ResultParser{
		logger: logger,
	}
}

func (p *ResultParser) log(msg string) {
	if p.logger != nil {
		p.logger(msg)
	}
}

// ParseOutput parses Python stdout to extract structured results
func (p *ResultParser) ParseOutput(output string, sessionDir string) *ExecutionResult {
	p.log(fmt.Sprintf("[PARSER] ParseOutput started - output length: %d, sessionDir: %s", len(output), sessionDir))
	
	result := &ExecutionResult{
		Success:    true,
		TextOutput: output,
		Tables:     []ParsedTable{},
		ChartFiles: []FileInfo{},
		ExportFiles: []FileInfo{},
		Warnings:   []string{},
	}

	// Check for error indicators
	if p.hasError(output) {
		result.Success = false
		result.ErrorMsg = p.extractErrorMessage(output)
		p.log(fmt.Sprintf("[PARSER] Error detected in output: %s", result.ErrorMsg))
	} else {
		p.log("[PARSER] No errors detected in output")
	}

	// Extract tables from output
	result.Tables = p.extractTables(output)
	p.log(fmt.Sprintf("[PARSER] Extracted %d tables from output", len(result.Tables)))

	// Detect generated files in session directory
	if sessionDir != "" {
		chartFiles, exportFiles := p.detectGeneratedFiles(sessionDir)
		result.ChartFiles = chartFiles
		result.ExportFiles = exportFiles
		p.log(fmt.Sprintf("[PARSER] Detected files - charts: %d, exports: %d", len(chartFiles), len(exportFiles)))
	} else {
		p.log("[PARSER] No session directory provided, skipping file detection")
	}

	// Extract warnings
	result.Warnings = p.extractWarnings(output)
	if len(result.Warnings) > 0 {
		p.log(fmt.Sprintf("[PARSER] Extracted %d warnings from output", len(result.Warnings)))
	}

	p.log(fmt.Sprintf("[PARSER] ParseOutput completed - success: %v, tables: %d, charts: %d, exports: %d, warnings: %d",
		result.Success, len(result.Tables), len(result.ChartFiles), len(result.ExportFiles), len(result.Warnings)))

	return result
}

// hasError checks if the output contains error indicators
func (p *ResultParser) hasError(output string) bool {
	errorPatterns := []string{
		"Error:", "错误:", "Exception:", "Traceback",
		"数据库错误", "分析错误", "查询结果为空",
		"sqlite3.Error", "pandas.errors",
	}
	outputLower := strings.ToLower(output)
	for _, pattern := range errorPatterns {
		if strings.Contains(outputLower, strings.ToLower(pattern)) {
			p.log(fmt.Sprintf("[PARSER] Error pattern matched: '%s'", pattern))
			return true
		}
	}
	return false
}

// extractErrorMessage extracts the error message from output
func (p *ResultParser) extractErrorMessage(output string) string {
	lines := strings.Split(output, "\n")
	
	// Look for error lines
	for i, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "error") || 
		   strings.Contains(lineLower, "错误") ||
		   strings.Contains(lineLower, "exception") {
			// Return this line and a few following lines
			endIdx := i + 3
			if endIdx > len(lines) {
				endIdx = len(lines)
			}
			return strings.Join(lines[i:endIdx], "\n")
		}
	}
	
	// If no specific error found, return last few lines
	if len(lines) > 5 {
		return strings.Join(lines[len(lines)-5:], "\n")
	}
	return output
}

// extractTables extracts table data from output
func (p *ResultParser) extractTables(output string) []ParsedTable {
	var tables []ParsedTable

	p.log("[PARSER] Starting table extraction from output")

	// Try to parse JSON tables
	jsonTables := p.extractJSONTables(output)
	tables = append(tables, jsonTables...)
	if len(jsonTables) > 0 {
		p.log(fmt.Sprintf("[PARSER] Found %d JSON tables", len(jsonTables)))
	}

	// Try to parse markdown/text tables
	textTables := p.extractTextTables(output)
	tables = append(tables, textTables...)
	if len(textTables) > 0 {
		p.log(fmt.Sprintf("[PARSER] Found %d text/markdown tables", len(textTables)))
	}

	return tables
}

// extractJSONTables extracts JSON formatted tables
func (p *ResultParser) extractJSONTables(output string) []ParsedTable {
	var tables []ParsedTable

	// Look for JSON arrays in output
	jsonPattern := regexp.MustCompile(`\[[\s\S]*?\{[\s\S]*?\}[\s\S]*?\]`)
	matches := jsonPattern.FindAllString(output, -1)

	p.log(fmt.Sprintf("[PARSER] Found %d potential JSON array patterns", len(matches)))

	for i, match := range matches {
		var data []map[string]interface{}
		if err := json.Unmarshal([]byte(match), &data); err == nil && len(data) > 0 {
			// Extract headers from first row
			var headers []string
			for key := range data[0] {
				headers = append(headers, key)
			}

			tables = append(tables, ParsedTable{
				Name:    "data",
				Headers: headers,
				Rows:    data,
			})
			p.log(fmt.Sprintf("[PARSER] Successfully parsed JSON table %d with %d rows and %d columns", i+1, len(data), len(headers)))
		} else if err != nil {
			p.log(fmt.Sprintf("[PARSER] Failed to parse JSON pattern %d: %v", i+1, err))
		}
	}

	return tables
}

// extractTextTables extracts text/markdown formatted tables
func (p *ResultParser) extractTextTables(output string) []ParsedTable {
	var tables []ParsedTable
	lines := strings.Split(output, "\n")
	
	// Look for pandas DataFrame output pattern
	// Typically has header row followed by data rows with consistent spacing
	var tableLines []string
	inTable := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Detect table start (multiple columns separated by spaces)
		if !inTable && len(trimmed) > 0 {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 && !strings.HasPrefix(trimmed, "===") {
				inTable = true
				tableLines = []string{line}
				continue
			}
		}
		
		if inTable {
			if trimmed == "" || strings.HasPrefix(trimmed, "===") {
				// End of table
				if len(tableLines) > 1 {
					table := p.parseTextTable(tableLines)
					if table != nil {
						tables = append(tables, *table)
					}
				}
				inTable = false
				tableLines = nil
			} else {
				tableLines = append(tableLines, line)
			}
		}
	}
	
	// Handle table at end of output
	if inTable && len(tableLines) > 1 {
		table := p.parseTextTable(tableLines)
		if table != nil {
			tables = append(tables, *table)
		}
	}
	
	return tables
}

// parseTextTable parses a text table into structured format
func (p *ResultParser) parseTextTable(lines []string) *ParsedTable {
	if len(lines) < 2 {
		return nil
	}
	
	// First line is headers
	headers := strings.Fields(lines[0])
	if len(headers) == 0 {
		return nil
	}
	
	var rows []map[string]interface{}
	for i := 1; i < len(lines); i++ {
		values := strings.Fields(lines[i])
		if len(values) == 0 {
			continue
		}
		
		row := make(map[string]interface{})
		for j, header := range headers {
			if j < len(values) {
				row[header] = values[j]
			} else {
				row[header] = ""
			}
		}
		rows = append(rows, row)
	}
	
	return &ParsedTable{
		Name:    "result",
		Headers: headers,
		Rows:    rows,
	}
}

// detectGeneratedFiles detects files generated in the session directory
// Recursively scans the session directory and its subdirectories (e.g., "files")
// Enhanced to support more file types and filter by size/modification time
// Validates: Requirements 1.2 (PNG images), 2.2 (multiple table formats)
func (p *ResultParser) detectGeneratedFiles(sessionDir string) ([]FileInfo, []FileInfo) {
	return p.detectGeneratedFilesWithConfig(sessionDir, DefaultFileDetectionConfig())
}

// detectGeneratedFilesWithConfig detects files with custom configuration
func (p *ResultParser) detectGeneratedFilesWithConfig(sessionDir string, config *FileDetectionConfig) ([]FileInfo, []FileInfo) {
	var chartFiles []FileInfo
	var exportFiles []FileInfo

	if config == nil {
		config = DefaultFileDetectionConfig()
	}

	p.log(fmt.Sprintf("[PARSER] Starting recursive scan of session directory: %s", sessionDir))
	p.log(fmt.Sprintf("[PARSER] File detection config - MinSize: %d bytes, MaxAge: %v", config.MinFileSize, config.MaxFileAge))
	p.log(fmt.Sprintf("[PARSER] Supported image extensions: %v", config.SupportedImageExtensions))
	p.log(fmt.Sprintf("[PARSER] Supported export extensions: %v", config.SupportedExportExtensions))

	// Check if session directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		p.log(fmt.Sprintf("[PARSER] Session directory does not exist: %s", sessionDir))
		return chartFiles, exportFiles
	}

	// Build extension lookup maps for faster checking
	imageExtMap := make(map[string]bool)
	for _, ext := range config.SupportedImageExtensions {
		imageExtMap[strings.ToLower(ext)] = true
	}
	exportExtMap := make(map[string]bool)
	for _, ext := range config.SupportedExportExtensions {
		exportExtMap[strings.ToLower(ext)] = true
	}

	// Track statistics for debugging
	var totalFilesScanned int
	var filesSkippedEmpty int
	var filesSkippedOld int
	var filesSkippedUnknown int

	now := time.Now()

	// Scan the session directory recursively
	err := filepath.Walk(sessionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			p.log(fmt.Sprintf("[PARSER] Error accessing path: %s - %v", path, err))
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			p.log(fmt.Sprintf("[PARSER] Entering directory: %s", path))
			return nil
		}

		totalFilesScanned++
		name := info.Name()
		ext := strings.ToLower(filepath.Ext(name))
		fileSize := info.Size()
		modTime := info.ModTime()

		p.log(fmt.Sprintf("[PARSER] Checking file: %s (ext: %s, size: %d bytes, modified: %v)", 
			path, ext, fileSize, modTime.Format(time.RFC3339)))

		// Filter by file size - skip empty files
		if fileSize < config.MinFileSize {
			p.log(fmt.Sprintf("[PARSER] Skipping file (too small): %s (%d bytes < %d bytes)", 
				name, fileSize, config.MinFileSize))
			filesSkippedEmpty++
			return nil
		}

		// Filter by modification time - skip old files
		fileAge := now.Sub(modTime)
		if config.MaxFileAge > 0 && fileAge > config.MaxFileAge {
			p.log(fmt.Sprintf("[PARSER] Skipping file (too old): %s (age: %v > max: %v)", 
				name, fileAge, config.MaxFileAge))
			filesSkippedOld++
			return nil
		}

		fileInfo := FileInfo{
			Path:         path,
			Name:         name,
			Size:         fileSize,
			ModifiedTime: modTime,
		}

		// Check if it's an image file
		if imageExtMap[ext] {
			fileInfo.Type = "chart"
			chartFiles = append(chartFiles, fileInfo)
			p.log(fmt.Sprintf("[PARSER] Found chart/image file: %s (type: %s, size: %d bytes)", 
				path, ext, fileSize))
			return nil
		}

		// Check if it's an export file
		if exportExtMap[ext] {
			fileInfo.Type = "export"
			exportFiles = append(exportFiles, fileInfo)
			p.log(fmt.Sprintf("[PARSER] Found export file: %s (type: %s, size: %d bytes)", 
				path, ext, fileSize))
			return nil
		}

		// Handle PDF files specially - could be chart or export
		if ext == ".pdf" {
			// PDF in "files" subdirectory is treated as export
			// PDF in root or other directories is treated as chart
			if strings.Contains(path, string(filepath.Separator)+"files"+string(filepath.Separator)) {
				fileInfo.Type = "export"
				exportFiles = append(exportFiles, fileInfo)
				p.log(fmt.Sprintf("[PARSER] Found export PDF: %s (%d bytes)", path, fileSize))
			} else {
				fileInfo.Type = "chart"
				chartFiles = append(chartFiles, fileInfo)
				p.log(fmt.Sprintf("[PARSER] Found chart PDF: %s (%d bytes)", path, fileSize))
			}
			return nil
		}

		// Unknown file type
		p.log(fmt.Sprintf("[PARSER] Skipping unknown file type: %s (ext: %s)", name, ext))
		filesSkippedUnknown++

		return nil
	})

	if err != nil {
		p.log(fmt.Sprintf("[PARSER] Failed to walk session directory: %v", err))
	}

	// Log summary statistics
	p.log(fmt.Sprintf("[PARSER] Scan complete - Total files: %d, Charts: %d, Exports: %d", 
		totalFilesScanned, len(chartFiles), len(exportFiles)))
	p.log(fmt.Sprintf("[PARSER] Skipped - Empty: %d, Old: %d, Unknown type: %d", 
		filesSkippedEmpty, filesSkippedOld, filesSkippedUnknown))

	return chartFiles, exportFiles
}

// isImageExtension checks if the extension is a supported image format
func (p *ResultParser) isImageExtension(ext string) bool {
	config := DefaultFileDetectionConfig()
	for _, imgExt := range config.SupportedImageExtensions {
		if strings.EqualFold(ext, imgExt) {
			return true
		}
	}
	return false
}

// isExportExtension checks if the extension is a supported export format
func (p *ResultParser) isExportExtension(ext string) bool {
	config := DefaultFileDetectionConfig()
	for _, expExt := range config.SupportedExportExtensions {
		if strings.EqualFold(ext, expExt) {
			return true
		}
	}
	return false
}

// EChartsDetectionResult represents the result of ECharts configuration detection
type EChartsDetectionResult struct {
	IsECharts     bool              `json:"is_echarts"`
	Score         int               `json:"score"`
	Threshold     int               `json:"threshold"`
	MatchedFields []string          `json:"matched_fields"`
	Reason        string            `json:"reason"`
}

// ECharts characteristic fields with their weights
// Higher weight means more indicative of ECharts configuration
var echartsFieldWeights = map[string]int{
	// Core chart fields (high weight)
	"series":     3, // Most important - defines chart data series
	"xAxis":      3, // X-axis configuration
	"yAxis":      3, // Y-axis configuration
	
	// Common configuration fields (medium weight)
	"title":      2, // Chart title
	"legend":     2, // Legend configuration
	"tooltip":    2, // Tooltip configuration
	"grid":       2, // Grid layout
	"dataZoom":   2, // Data zoom component
	"visualMap":  2, // Visual mapping
	
	// Additional ECharts fields (lower weight)
	"toolbox":    1, // Toolbox component
	"polar":      1, // Polar coordinate system
	"radar":      1, // Radar chart
	"geo":        1, // Geographic coordinate system
	"parallel":   1, // Parallel coordinate system
	"timeline":   1, // Timeline component
	"graphic":    1, // Graphic elements
	"calendar":   1, // Calendar coordinate system
	"dataset":    1, // Dataset component
	"aria":       1, // Accessibility
	"axisPointer": 1, // Axis pointer
	"brush":      1, // Brush component
	"color":      1, // Color palette
	"backgroundColor": 1, // Background color
	"textStyle":  1, // Global text style
	"animation":  1, // Animation settings
	"animationDuration": 1, // Animation duration
	"animationEasing": 1, // Animation easing
}

// Minimum score threshold to consider as ECharts config
const echartsScoreThreshold = 4

// IsEChartsConfig checks if a JSON string represents an ECharts configuration
// Returns detailed detection result including score and matched fields
// Validates: Requirement 1.1 (ECharts chart configuration can be correctly parsed and rendered)
func (p *ResultParser) IsEChartsConfig(jsonStr string) *EChartsDetectionResult {
	result := &EChartsDetectionResult{
		IsECharts:     false,
		Score:         0,
		Threshold:     echartsScoreThreshold,
		MatchedFields: []string{},
		Reason:        "",
	}

	p.log(fmt.Sprintf("[PARSER] Starting ECharts detection for JSON string (length: %d)", len(jsonStr)))

	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		result.Reason = fmt.Sprintf("Invalid JSON: %v", err)
		p.log(fmt.Sprintf("[PARSER] ECharts detection failed - %s", result.Reason))
		return result
	}

	return p.IsEChartsConfigFromMap(data)
}

// IsEChartsConfigFromMap checks if a map represents an ECharts configuration
// This is useful when you already have parsed JSON data
// Validates: Requirement 1.1 (ECharts chart configuration can be correctly parsed and rendered)
func (p *ResultParser) IsEChartsConfigFromMap(data map[string]interface{}) *EChartsDetectionResult {
	result := &EChartsDetectionResult{
		IsECharts:     false,
		Score:         0,
		Threshold:     echartsScoreThreshold,
		MatchedFields: []string{},
		Reason:        "",
	}

	if data == nil {
		result.Reason = "Input data is nil"
		p.log("[PARSER] ECharts detection failed - input data is nil")
		return result
	}

	p.log(fmt.Sprintf("[PARSER] Checking map with %d fields for ECharts characteristics", len(data)))

	// Check for ECharts characteristic fields and calculate score
	for field, weight := range echartsFieldWeights {
		if _, exists := data[field]; exists {
			result.Score += weight
			result.MatchedFields = append(result.MatchedFields, field)
			p.log(fmt.Sprintf("[PARSER] Found ECharts field '%s' (weight: %d, cumulative score: %d)", 
				field, weight, result.Score))
		}
	}

	// Additional validation for series field (most important indicator)
	if seriesData, exists := data["series"]; exists {
		if p.isValidEChartsSeries(seriesData) {
			result.Score += 2 // Bonus for valid series structure
			p.log("[PARSER] Series field has valid ECharts structure (+2 bonus)")
		}
	}

	// Determine if it's an ECharts config based on score
	result.IsECharts = result.Score >= result.Threshold

	if result.IsECharts {
		result.Reason = fmt.Sprintf("Score %d >= threshold %d, matched fields: %v", 
			result.Score, result.Threshold, result.MatchedFields)
		p.log(fmt.Sprintf("[PARSER] ECharts detection POSITIVE - %s", result.Reason))
	} else {
		result.Reason = fmt.Sprintf("Score %d < threshold %d, matched fields: %v", 
			result.Score, result.Threshold, result.MatchedFields)
		p.log(fmt.Sprintf("[PARSER] ECharts detection NEGATIVE - %s", result.Reason))
	}

	return result
}

// isValidEChartsSeries validates if the series data has a valid ECharts structure
func (p *ResultParser) isValidEChartsSeries(seriesData interface{}) bool {
	// Series can be an array or a single object
	switch series := seriesData.(type) {
	case []interface{}:
		// Array of series - check if at least one has valid structure
		for _, s := range series {
			if seriesMap, ok := s.(map[string]interface{}); ok {
				if p.hasValidSeriesFields(seriesMap) {
					return true
				}
			}
		}
	case map[string]interface{}:
		// Single series object
		return p.hasValidSeriesFields(series)
	}
	return false
}

// hasValidSeriesFields checks if a series object has valid ECharts fields
func (p *ResultParser) hasValidSeriesFields(series map[string]interface{}) bool {
	// Common series fields in ECharts
	validSeriesFields := []string{"type", "data", "name", "stack", "areaStyle", "itemStyle", "label", "emphasis"}
	
	matchCount := 0
	for _, field := range validSeriesFields {
		if _, exists := series[field]; exists {
			matchCount++
		}
	}
	
	// Consider valid if at least 2 common fields are present
	// Most ECharts series have at least "type" and "data"
	return matchCount >= 2
}

// DetectAndClassifyJSON attempts to detect if a JSON string is ECharts config or regular JSON
// Returns the type classification and the parsed data
func (p *ResultParser) DetectAndClassifyJSON(jsonStr string) (string, interface{}, error) {
	p.log(fmt.Sprintf("[PARSER] Classifying JSON content (length: %d)", len(jsonStr)))

	// First, try to parse as JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		p.log(fmt.Sprintf("[PARSER] JSON parse error: %v", err))
		return "invalid", nil, err
	}

	// Check if it's an object (map)
	if dataMap, ok := data.(map[string]interface{}); ok {
		// Check if it's an ECharts configuration
		echartsResult := p.IsEChartsConfigFromMap(dataMap)
		if echartsResult.IsECharts {
			p.log(fmt.Sprintf("[PARSER] Classified as ECharts config (score: %d)", echartsResult.Score))
			return "echarts", data, nil
		}
		p.log("[PARSER] Classified as regular JSON object")
		return "json_object", data, nil
	}

	// Check if it's an array
	if _, ok := data.([]interface{}); ok {
		p.log("[PARSER] Classified as JSON array (likely table data)")
		return "json_array", data, nil
	}

	// Other JSON types (string, number, boolean, null)
	p.log("[PARSER] Classified as JSON primitive")
	return "json_primitive", data, nil
}

// extractWarnings extracts warning messages from output
func (p *ResultParser) extractWarnings(output string) []string {
	var warnings []string

	warningPatterns := []string{
		"Warning:", "警告:", "UserWarning:", "FutureWarning:",
		"DeprecationWarning:", "注意:",
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		for _, pattern := range warningPatterns {
			if strings.Contains(line, pattern) {
				warnings = append(warnings, strings.TrimSpace(line))
				p.log(fmt.Sprintf("[PARSER] Found warning: %s", strings.TrimSpace(line)))
				break
			}
		}
	}

	return warnings
}

// FormatAsText formats the result as plain text
func (p *ResultParser) FormatAsText(result *ExecutionResult) string {
	var sb strings.Builder

	if !result.Success {
		sb.WriteString("执行失败: ")
		sb.WriteString(result.ErrorMsg)
		sb.WriteString("\n")
		return sb.String()
	}

	sb.WriteString(result.TextOutput)

	if len(result.ChartFiles) > 0 {
		sb.WriteString("\n\n生成的图表:\n")
		for _, f := range result.ChartFiles {
			sb.WriteString("- " + f.Name + "\n")
		}
	}

	if len(result.ExportFiles) > 0 {
		sb.WriteString("\n导出的文件:\n")
		for _, f := range result.ExportFiles {
			sb.WriteString("- " + f.Name + "\n")
		}
	}

	return sb.String()
}
