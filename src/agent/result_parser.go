package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	Path     string `json:"path"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
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
	}

	// Extract tables from output
	result.Tables = p.extractTables(output)

	// Detect generated files in session directory
	if sessionDir != "" {
		chartFiles, exportFiles := p.detectGeneratedFiles(sessionDir)
		result.ChartFiles = chartFiles
		result.ExportFiles = exportFiles
	}

	// Extract warnings
	result.Warnings = p.extractWarnings(output)

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

	// Try to parse JSON tables
	jsonTables := p.extractJSONTables(output)
	tables = append(tables, jsonTables...)

	// Try to parse markdown/text tables
	textTables := p.extractTextTables(output)
	tables = append(tables, textTables...)

	return tables
}

// extractJSONTables extracts JSON formatted tables
func (p *ResultParser) extractJSONTables(output string) []ParsedTable {
	var tables []ParsedTable

	// Look for JSON arrays in output
	jsonPattern := regexp.MustCompile(`\[[\s\S]*?\{[\s\S]*?\}[\s\S]*?\]`)
	matches := jsonPattern.FindAllString(output, -1)

	for _, match := range matches {
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
func (p *ResultParser) detectGeneratedFiles(sessionDir string) ([]FileInfo, []FileInfo) {
	var chartFiles []FileInfo
	var exportFiles []FileInfo

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		p.log("[PARSER] Failed to read session directory: " + err.Error())
		return chartFiles, exportFiles
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		fullPath := filepath.Join(sessionDir, name)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileInfo := FileInfo{
			Path: fullPath,
			Name: name,
			Size: info.Size(),
		}

		// Categorize by extension
		switch ext {
		case ".png", ".jpg", ".jpeg", ".svg", ".pdf":
			fileInfo.Type = "chart"
			chartFiles = append(chartFiles, fileInfo)
		case ".csv", ".xlsx", ".xls", ".json":
			fileInfo.Type = "export"
			exportFiles = append(exportFiles, fileInfo)
		}
	}

	return chartFiles, exportFiles
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
