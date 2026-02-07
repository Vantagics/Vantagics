package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// **Validates: Requirements 1.1, 2.1**
// 测试 ResultParser 的核心功能：
// - JSON 表格解析
// - 文件检测
// - ECharts 配置识别

// =============================================================================
// JSON 表格解析测试
// =============================================================================

// TestResultParser_ExtractJSONTables_ValidArray tests parsing valid JSON array tables
// **Validates: Requirements 2.1**
func TestResultParser_ExtractJSONTables_ValidArray(t *testing.T) {
	parser := NewResultParser(nil)

	testCases := []struct {
		name           string
		input          string
		expectedTables int
		expectedRows   int
	}{
		{
			name: "Simple JSON array",
			input: `[{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}]`,
			expectedTables: 1,
			expectedRows:   2,
		},
		{
			name: "JSON array with surrounding text",
			input: `Analysis result:
[{"product": "A", "sales": 100}, {"product": "B", "sales": 200}]
End of analysis.`,
			expectedTables: 1,
			expectedRows:   2,
		},
		{
			name: "Multiple JSON arrays",
			input: `First table: [{"x": 1}]
Second table: [{"y": 2}, {"y": 3}]`,
			expectedTables: 2,
			expectedRows:   1, // First table has 1 row
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.ParseOutput(tc.input, "")
			
			if len(result.Tables) < tc.expectedTables {
				t.Errorf("Expected at least %d tables, got %d", tc.expectedTables, len(result.Tables))
				return
			}
			
			if len(result.Tables) > 0 && len(result.Tables[0].Rows) != tc.expectedRows {
				t.Errorf("Expected %d rows in first table, got %d", tc.expectedRows, len(result.Tables[0].Rows))
			}
		})
	}
}

// TestResultParser_ExtractJSONTables_InvalidJSON tests handling of invalid JSON
// **Validates: Requirements 2.1**
func TestResultParser_ExtractJSONTables_InvalidJSON(t *testing.T) {
	parser := NewResultParser(nil)

	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "Malformed JSON",
			input: `[{"name": "Alice", "age": }]`,
		},
		{
			name:  "Plain text without JSON",
			input: `This is just plain text without any JSON data.`,
		},
		{
			name:  "Empty array",
			input: `[]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.ParseOutput(tc.input, "")
			// Should not crash and should return empty or minimal tables
			if result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

// TestResultParser_ExtractJSONTables_ComplexData tests parsing complex JSON structures
// **Validates: Requirements 2.1**
func TestResultParser_ExtractJSONTables_ComplexData(t *testing.T) {
	parser := NewResultParser(nil)

	// Test with nested data
	input := `[
		{"id": 1, "name": "Product A", "details": {"price": 99.99, "stock": 50}},
		{"id": 2, "name": "Product B", "details": {"price": 149.99, "stock": 30}}
	]`

	result := parser.ParseOutput(input, "")

	if len(result.Tables) == 0 {
		t.Error("Expected at least one table from complex JSON")
		return
	}

	table := result.Tables[0]
	if len(table.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.Rows))
	}

	// Verify headers exist
	if len(table.Headers) == 0 {
		t.Error("Expected headers to be extracted")
	}
}

// =============================================================================
// 文件检测测试
// =============================================================================

// TestResultParser_DetectGeneratedFiles_ImageFiles tests detection of image files
// **Validates: Requirements 1.1, 2.1**
func TestResultParser_DetectGeneratedFiles_ImageFiles(t *testing.T) {
	parser := NewResultParser(nil)

	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "result_parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test image files
	imageFiles := []string{"chart.png", "graph.jpg", "diagram.svg", "photo.webp", "image.bmp"}
	for _, name := range imageFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	chartFiles, _ := parser.detectGeneratedFiles(tempDir)

	if len(chartFiles) != len(imageFiles) {
		t.Errorf("Expected %d chart files, got %d", len(imageFiles), len(chartFiles))
	}

	// Verify all expected extensions are detected
	foundExtensions := make(map[string]bool)
	for _, f := range chartFiles {
		ext := filepath.Ext(f.Name)
		foundExtensions[ext] = true
	}

	expectedExtensions := []string{".png", ".jpg", ".svg", ".webp", ".bmp"}
	for _, ext := range expectedExtensions {
		if !foundExtensions[ext] {
			t.Errorf("Expected to find file with extension %s", ext)
		}
	}
}

// TestResultParser_DetectGeneratedFiles_ExportFiles tests detection of export files
// **Validates: Requirements 2.1**
func TestResultParser_DetectGeneratedFiles_ExportFiles(t *testing.T) {
	parser := NewResultParser(nil)

	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "result_parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test export files
	exportFiles := []string{"data.csv", "report.xlsx", "output.json", "results.txt"}
	for _, name := range exportFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	_, exports := parser.detectGeneratedFiles(tempDir)

	if len(exports) != len(exportFiles) {
		t.Errorf("Expected %d export files, got %d", len(exportFiles), len(exports))
	}
}

// TestResultParser_DetectGeneratedFiles_RecursiveScan tests recursive directory scanning
// **Validates: Requirements 1.1**
func TestResultParser_DetectGeneratedFiles_RecursiveScan(t *testing.T) {
	parser := NewResultParser(nil)

	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "result_parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectory
	subDir := filepath.Join(tempDir, "files")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create files in root and subdirectory
	rootFile := filepath.Join(tempDir, "root_chart.png")
	subFile := filepath.Join(subDir, "sub_chart.png")

	if err := os.WriteFile(rootFile, []byte("root content"), 0644); err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}
	if err := os.WriteFile(subFile, []byte("sub content"), 0644); err != nil {
		t.Fatalf("Failed to create sub file: %v", err)
	}

	chartFiles, _ := parser.detectGeneratedFiles(tempDir)

	// Should find both files
	if len(chartFiles) != 2 {
		t.Errorf("Expected 2 chart files (recursive scan), got %d", len(chartFiles))
	}
}

// TestResultParser_DetectGeneratedFiles_EmptyDirectory tests handling of empty directory
// **Validates: Requirements 1.1**
func TestResultParser_DetectGeneratedFiles_EmptyDirectory(t *testing.T) {
	parser := NewResultParser(nil)

	// Create an empty temporary directory
	tempDir, err := os.MkdirTemp("", "result_parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	chartFiles, exportFiles := parser.detectGeneratedFiles(tempDir)

	if len(chartFiles) != 0 {
		t.Errorf("Expected 0 chart files in empty dir, got %d", len(chartFiles))
	}
	if len(exportFiles) != 0 {
		t.Errorf("Expected 0 export files in empty dir, got %d", len(exportFiles))
	}
}

// TestResultParser_DetectGeneratedFiles_NonExistentDirectory tests handling of non-existent directory
// **Validates: Requirements 1.1**
func TestResultParser_DetectGeneratedFiles_NonExistentDirectory(t *testing.T) {
	parser := NewResultParser(nil)

	chartFiles, exportFiles := parser.detectGeneratedFiles("/non/existent/path")

	if len(chartFiles) != 0 {
		t.Errorf("Expected 0 chart files for non-existent dir, got %d", len(chartFiles))
	}
	if len(exportFiles) != 0 {
		t.Errorf("Expected 0 export files for non-existent dir, got %d", len(exportFiles))
	}
}

// TestResultParser_DetectGeneratedFiles_FileSizeFilter tests filtering by file size
// **Validates: Requirements 1.1**
func TestResultParser_DetectGeneratedFiles_FileSizeFilter(t *testing.T) {
	parser := NewResultParser(nil)

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "result_parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an empty file (should be filtered out)
	emptyFile := filepath.Join(tempDir, "empty.png")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// Create a file with content (should be detected)
	validFile := filepath.Join(tempDir, "valid.png")
	if err := os.WriteFile(validFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create valid file: %v", err)
	}

	chartFiles, _ := parser.detectGeneratedFiles(tempDir)

	// Should only find the valid file (empty file filtered out)
	if len(chartFiles) != 1 {
		t.Errorf("Expected 1 chart file (empty filtered), got %d", len(chartFiles))
	}
}

// TestResultParser_DetectGeneratedFilesWithConfig tests custom configuration
// **Validates: Requirements 1.1**
func TestResultParser_DetectGeneratedFilesWithConfig(t *testing.T) {
	parser := NewResultParser(nil)

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "result_parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFile := filepath.Join(tempDir, "test.png")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with custom config that has larger min file size
	config := &FileDetectionConfig{
		MinFileSize:              100, // Larger than our test file
		MaxFileAge:               24 * time.Hour,
		SupportedImageExtensions: []string{".png"},
		SupportedExportExtensions: []string{".csv"},
	}

	chartFiles, _ := parser.detectGeneratedFilesWithConfig(tempDir, config)

	// Should not find the file (too small)
	if len(chartFiles) != 0 {
		t.Errorf("Expected 0 chart files (min size filter), got %d", len(chartFiles))
	}
}

// =============================================================================
// ECharts 配置识别测试
// =============================================================================

// TestResultParser_IsEChartsConfig_ValidConfig tests detection of valid ECharts configurations
// **Validates: Requirements 1.1**
func TestResultParser_IsEChartsConfig_ValidConfig(t *testing.T) {
	parser := NewResultParser(nil)

	testCases := []struct {
		name     string
		config   string
		expected bool
	}{
		{
			name: "Bar chart config",
			config: `{
				"title": {"text": "Sales Data"},
				"xAxis": {"type": "category", "data": ["Mon", "Tue", "Wed"]},
				"yAxis": {"type": "value"},
				"series": [{"type": "bar", "data": [120, 200, 150]}]
			}`,
			expected: true,
		},
		{
			name: "Line chart config",
			config: `{
				"xAxis": {"type": "category"},
				"yAxis": {"type": "value"},
				"series": [{"type": "line", "data": [1, 2, 3, 4, 5]}],
				"tooltip": {"trigger": "axis"}
			}`,
			expected: true,
		},
		{
			name: "Pie chart config",
			config: `{
				"title": {"text": "Distribution"},
				"legend": {"orient": "vertical"},
				"series": [{"type": "pie", "data": [{"value": 335, "name": "A"}]}]
			}`,
			expected: true,
		},
		{
			name: "Complex chart with multiple series",
			config: `{
				"title": {"text": "Multi-series"},
				"tooltip": {},
				"legend": {"data": ["Series 1", "Series 2"]},
				"xAxis": {"data": ["A", "B", "C"]},
				"yAxis": {},
				"series": [
					{"name": "Series 1", "type": "bar", "data": [5, 20, 36]},
					{"name": "Series 2", "type": "line", "data": [10, 15, 25]}
				]
			}`,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.IsEChartsConfig(tc.config)
			if result.IsECharts != tc.expected {
				t.Errorf("Expected IsECharts=%v, got %v (score: %d, reason: %s)",
					tc.expected, result.IsECharts, result.Score, result.Reason)
			}
		})
	}
}

// TestResultParser_IsEChartsConfig_InvalidConfig tests rejection of non-ECharts JSON
// **Validates: Requirements 1.1**
func TestResultParser_IsEChartsConfig_InvalidConfig(t *testing.T) {
	parser := NewResultParser(nil)

	testCases := []struct {
		name     string
		config   string
		expected bool
	}{
		{
			name:     "Simple object",
			config:   `{"name": "John", "age": 30}`,
			expected: false,
		},
		{
			name:     "Array data",
			config:   `[{"id": 1}, {"id": 2}]`,
			expected: false,
		},
		{
			name:     "Empty object",
			config:   `{}`,
			expected: false,
		},
		{
			name:     "Config with only title",
			config:   `{"title": {"text": "Just a title"}}`,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.IsEChartsConfig(tc.config)
			if result.IsECharts != tc.expected {
				t.Errorf("Expected IsECharts=%v, got %v (score: %d, reason: %s)",
					tc.expected, result.IsECharts, result.Score, result.Reason)
			}
		})
	}
}

// TestResultParser_IsEChartsConfig_InvalidJSON tests handling of invalid JSON
// **Validates: Requirements 1.1**
func TestResultParser_IsEChartsConfig_InvalidJSON(t *testing.T) {
	parser := NewResultParser(nil)

	testCases := []struct {
		name   string
		config string
	}{
		{
			name:   "Malformed JSON",
			config: `{"title": "test"`,
		},
		{
			name:   "Empty string",
			config: ``,
		},
		{
			name:   "Plain text",
			config: `This is not JSON`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.IsEChartsConfig(tc.config)
			if result.IsECharts {
				t.Errorf("Expected IsECharts=false for invalid JSON, got true")
			}
		})
	}
}

// TestResultParser_IsEChartsConfigFromMap tests detection from parsed map
// **Validates: Requirements 1.1**
func TestResultParser_IsEChartsConfigFromMap(t *testing.T) {
	parser := NewResultParser(nil)

	// Valid ECharts config as map
	validConfig := map[string]interface{}{
		"xAxis":  map[string]interface{}{"type": "category"},
		"yAxis":  map[string]interface{}{"type": "value"},
		"series": []interface{}{map[string]interface{}{"type": "bar", "data": []int{1, 2, 3}}},
	}

	result := parser.IsEChartsConfigFromMap(validConfig)
	if !result.IsECharts {
		t.Errorf("Expected valid ECharts config to be detected, score: %d", result.Score)
	}

	// Invalid config (regular data)
	invalidConfig := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{"name": "Alice"},
			map[string]interface{}{"name": "Bob"},
		},
	}

	result = parser.IsEChartsConfigFromMap(invalidConfig)
	if result.IsECharts {
		t.Errorf("Expected regular data to not be detected as ECharts")
	}

	// Nil input
	result = parser.IsEChartsConfigFromMap(nil)
	if result.IsECharts {
		t.Errorf("Expected nil input to not be detected as ECharts")
	}
}

// TestResultParser_DetectAndClassifyJSON tests JSON classification
// **Validates: Requirements 1.1, 2.1**
func TestResultParser_DetectAndClassifyJSON(t *testing.T) {
	parser := NewResultParser(nil)

	testCases := []struct {
		name         string
		input        string
		expectedType string
		expectError  bool
	}{
		{
			name: "ECharts config",
			input: `{
				"xAxis": {"type": "category"},
				"yAxis": {"type": "value"},
				"series": [{"type": "bar", "data": [1, 2, 3]}]
			}`,
			expectedType: "echarts",
			expectError:  false,
		},
		{
			name:         "JSON array (table data)",
			input:        `[{"name": "A", "value": 1}, {"name": "B", "value": 2}]`,
			expectedType: "json_array",
			expectError:  false,
		},
		{
			name:         "Regular JSON object",
			input:        `{"name": "test", "count": 42}`,
			expectedType: "json_object",
			expectError:  false,
		},
		{
			name:         "JSON primitive (string)",
			input:        `"hello world"`,
			expectedType: "json_primitive",
			expectError:  false,
		},
		{
			name:         "JSON primitive (number)",
			input:        `42`,
			expectedType: "json_primitive",
			expectError:  false,
		},
		{
			name:         "Invalid JSON",
			input:        `{invalid}`,
			expectedType: "invalid",
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonType, _, err := parser.DetectAndClassifyJSON(tc.input)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if jsonType != tc.expectedType {
				t.Errorf("Expected type %s, got %s", tc.expectedType, jsonType)
			}
		})
	}
}

// TestResultParser_EChartsDetectionResult_Score tests the scoring mechanism
// **Validates: Requirements 1.1**
func TestResultParser_EChartsDetectionResult_Score(t *testing.T) {
	parser := NewResultParser(nil)

	// Test that more ECharts fields result in higher scores
	minimalConfig := `{"series": [{"type": "bar", "data": [1]}]}`
	fullConfig := `{
		"title": {"text": "Full Chart"},
		"tooltip": {},
		"legend": {},
		"xAxis": {"type": "category"},
		"yAxis": {"type": "value"},
		"series": [{"type": "bar", "data": [1, 2, 3]}],
		"grid": {}
	}`

	minResult := parser.IsEChartsConfig(minimalConfig)
	fullResult := parser.IsEChartsConfig(fullConfig)

	if fullResult.Score <= minResult.Score {
		t.Errorf("Expected full config score (%d) > minimal config score (%d)",
			fullResult.Score, minResult.Score)
	}

	// Verify matched fields are tracked
	if len(fullResult.MatchedFields) <= len(minResult.MatchedFields) {
		t.Errorf("Expected more matched fields in full config")
	}
}

// =============================================================================
// 错误检测测试
// =============================================================================

// TestResultParser_HasError tests error detection in output
// **Validates: Requirements 1.1**
func TestResultParser_HasError(t *testing.T) {
	parser := NewResultParser(nil)

	testCases := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "Python error",
			input:       "Error: Division by zero",
			expectError: true,
		},
		{
			name:        "Chinese error",
			input:       "数据库错误: 连接失败",
			expectError: true,
		},
		{
			name:        "Exception traceback",
			input:       "Traceback (most recent call last):\n  File...",
			expectError: true,
		},
		{
			name:        "Normal output",
			input:       "Analysis completed successfully.\nResults: 42",
			expectError: false,
		},
		{
			name:        "Empty output",
			input:       "",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.ParseOutput(tc.input, "")
			if result.Success == tc.expectError {
				t.Errorf("Expected Success=%v, got %v", !tc.expectError, result.Success)
			}
		})
	}
}

// =============================================================================
// 警告提取测试
// =============================================================================

// TestResultParser_ExtractWarnings tests warning extraction from output
// **Validates: Requirements 1.1**
func TestResultParser_ExtractWarnings(t *testing.T) {
	parser := NewResultParser(nil)

	input := `Processing data...
Warning: Deprecated function used
UserWarning: This feature will be removed
FutureWarning: Use new API instead
Analysis complete.`

	result := parser.ParseOutput(input, "")

	if len(result.Warnings) != 3 {
		t.Errorf("Expected 3 warnings, got %d", len(result.Warnings))
	}
}

// =============================================================================
// 辅助方法测试
// =============================================================================

// TestResultParser_IsImageExtension tests image extension detection
// **Validates: Requirements 1.1**
func TestResultParser_IsImageExtension(t *testing.T) {
	parser := NewResultParser(nil)

	imageExtensions := []string{".png", ".jpg", ".jpeg", ".svg", ".gif", ".webp", ".bmp", ".tiff", ".tif"}
	nonImageExtensions := []string{".csv", ".xlsx", ".json", ".txt", ".pdf", ".doc"}

	for _, ext := range imageExtensions {
		if !parser.isImageExtension(ext) {
			t.Errorf("Expected %s to be recognized as image extension", ext)
		}
	}

	for _, ext := range nonImageExtensions {
		if parser.isImageExtension(ext) {
			t.Errorf("Expected %s to NOT be recognized as image extension", ext)
		}
	}
}

// TestResultParser_IsExportExtension tests export extension detection
// **Validates: Requirements 2.1**
func TestResultParser_IsExportExtension(t *testing.T) {
	parser := NewResultParser(nil)

	exportExtensions := []string{".csv", ".xlsx", ".xls", ".json", ".txt", ".html", ".xml"}
	nonExportExtensions := []string{".png", ".jpg", ".gif", ".exe", ".dll"}

	for _, ext := range exportExtensions {
		if !parser.isExportExtension(ext) {
			t.Errorf("Expected %s to be recognized as export extension", ext)
		}
	}

	for _, ext := range nonExportExtensions {
		if parser.isExportExtension(ext) {
			t.Errorf("Expected %s to NOT be recognized as export extension", ext)
		}
	}
}

// =============================================================================
// 集成测试
// =============================================================================

// TestResultParser_ParseOutput_Integration tests the full parsing flow
// **Validates: Requirements 1.1, 2.1**
func TestResultParser_ParseOutput_Integration(t *testing.T) {
	parser := NewResultParser(nil)

	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "result_parser_integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	chartFile := filepath.Join(tempDir, "chart.png")
	exportFile := filepath.Join(tempDir, "data.csv")
	if err := os.WriteFile(chartFile, []byte("chart content"), 0644); err != nil {
		t.Fatalf("Failed to create chart file: %v", err)
	}
	if err := os.WriteFile(exportFile, []byte("csv content"), 0644); err != nil {
		t.Fatalf("Failed to create export file: %v", err)
	}

	// Test output with JSON table
	output := `Analysis Results:
[{"product": "A", "sales": 100}, {"product": "B", "sales": 200}]
Chart saved to chart.png
Data exported to data.csv`

	result := parser.ParseOutput(output, tempDir)

	// Verify success
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.ErrorMsg)
	}

	// Verify tables extracted
	if len(result.Tables) == 0 {
		t.Error("Expected at least one table to be extracted")
	}

	// Verify files detected
	if len(result.ChartFiles) == 0 {
		t.Error("Expected chart files to be detected")
	}
	if len(result.ExportFiles) == 0 {
		t.Error("Expected export files to be detected")
	}
}

// TestResultParser_FormatAsText tests text formatting of results
// **Validates: Requirements 1.1**
func TestResultParser_FormatAsText(t *testing.T) {
	parser := NewResultParser(nil)

	// Test successful result
	successResult := &ExecutionResult{
		Success:    true,
		TextOutput: "Analysis complete",
		ChartFiles: []FileInfo{{Name: "chart.png"}},
		ExportFiles: []FileInfo{{Name: "data.csv"}},
	}

	formatted := parser.FormatAsText(successResult)
	if formatted == "" {
		t.Error("Expected non-empty formatted text")
	}

	// Test failed result
	failedResult := &ExecutionResult{
		Success:  false,
		ErrorMsg: "Database connection failed",
	}

	formatted = parser.FormatAsText(failedResult)
	if formatted == "" {
		t.Error("Expected non-empty formatted text for error")
	}
}

// =============================================================================
// 边界情况测试
// =============================================================================

// TestResultParser_EdgeCases tests various edge cases
// **Validates: Requirements 1.1, 2.1**
func TestResultParser_EdgeCases(t *testing.T) {
	parser := NewResultParser(nil)

	t.Run("Nil logger", func(t *testing.T) {
		p := NewResultParser(nil)
		// Should not panic
		result := p.ParseOutput("test", "")
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})

	t.Run("Empty output", func(t *testing.T) {
		result := parser.ParseOutput("", "")
		if result == nil {
			t.Error("Expected non-nil result for empty output")
		}
		if !result.Success {
			t.Error("Expected success for empty output")
		}
	})

	t.Run("Very long output", func(t *testing.T) {
		// Generate a long output
		longOutput := ""
		for i := 0; i < 1000; i++ {
			longOutput += "Line of output data\n"
		}
		result := parser.ParseOutput(longOutput, "")
		if result == nil {
			t.Error("Expected non-nil result for long output")
		}
	})

	t.Run("Unicode content", func(t *testing.T) {
		unicodeOutput := `分析结果：
[{"名称": "产品A", "销量": 100}, {"名称": "产品B", "销量": 200}]
图表已保存`
		result := parser.ParseOutput(unicodeOutput, "")
		if result == nil {
			t.Error("Expected non-nil result for unicode output")
		}
	})

	t.Run("Mixed content", func(t *testing.T) {
		mixedOutput := `English text
中文文本
[{"key": "value"}]
More text`
		result := parser.ParseOutput(mixedOutput, "")
		if result == nil {
			t.Error("Expected non-nil result for mixed content")
		}
	})
}

// TestResultParser_DefaultFileDetectionConfig tests default configuration
// **Validates: Requirements 1.1**
func TestResultParser_DefaultFileDetectionConfig(t *testing.T) {
	config := DefaultFileDetectionConfig()

	if config == nil {
		t.Fatal("Expected non-nil default config")
	}

	if config.MinFileSize <= 0 {
		t.Error("Expected positive MinFileSize")
	}

	if config.MaxFileAge <= 0 {
		t.Error("Expected positive MaxFileAge")
	}

	if len(config.SupportedImageExtensions) == 0 {
		t.Error("Expected non-empty SupportedImageExtensions")
	}

	if len(config.SupportedExportExtensions) == 0 {
		t.Error("Expected non-empty SupportedExportExtensions")
	}
}

// TestResultParser_EChartsFieldWeights tests that field weights are properly defined
// **Validates: Requirements 1.1**
func TestResultParser_EChartsFieldWeights(t *testing.T) {
	// Verify core fields have high weights
	coreFields := []string{"series", "xAxis", "yAxis"}
	for _, field := range coreFields {
		weight, exists := echartsFieldWeights[field]
		if !exists {
			t.Errorf("Expected core field %s to have weight defined", field)
		}
		if weight < 3 {
			t.Errorf("Expected core field %s to have weight >= 3, got %d", field, weight)
		}
	}

	// Verify threshold is reasonable
	if echartsScoreThreshold <= 0 {
		t.Error("Expected positive score threshold")
	}
}
