package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"testing/quick"
)

// **Validates: Requirements 4.1, 4.2, 4.3, 4.5, 4.6**
// Property 6: Response Parsing Completeness
// Property 7: JSON Error Logging

// ResponseParseResult holds the parsed results from an LLM response
type ResponseParseResult struct {
	EChartsBlocks []string
	Images        []string
	Tables        [][]map[string]interface{}
	CSVBlocks     []string
	Errors        []string
}

// parseResponse extracts all visualization data from an LLM response
// This is a test-friendly version of the parsing logic in app.go
func parseResponse(resp string) ResponseParseResult {
	result := ResponseParseResult{
		EChartsBlocks: []string{},
		Images:        []string{},
		Tables:        [][]map[string]interface{}{},
		CSVBlocks:     []string{},
		Errors:        []string{},
	}

	// 1. ECharts JSON - Extract ALL matches
	reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	allEChartsMatches := reECharts.FindAllStringSubmatch(resp, -1)
	for _, match := range allEChartsMatches {
		if len(match) > 1 {
			jsonStr := strings.TrimSpace(match[1])
			var testJSON map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &testJSON); err == nil {
				result.EChartsBlocks = append(result.EChartsBlocks, jsonStr)
			} else {
				maxLen := 500
				if len(jsonStr) < maxLen {
					maxLen = len(jsonStr)
				}
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse echarts JSON: %s", jsonStr[:maxLen]))
			}
		}
	}

	// 2. Markdown Image (Base64) - Extract ALL matches
	reImage := regexp.MustCompile(`!\[.*?\]\((data:image\/.*?;base64,.*?)\)`)
	allImageMatches := reImage.FindAllStringSubmatch(resp, -1)
	for _, match := range allImageMatches {
		if len(match) > 1 {
			result.Images = append(result.Images, match[1])
		}
	}

	// 3. Table Data - Extract ALL matches
	reTable := regexp.MustCompile("(?s)```\\s*json:table\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	allTableMatches := reTable.FindAllStringSubmatch(resp, -1)
	for _, match := range allTableMatches {
		if len(match) > 1 {
			jsonStr := strings.TrimSpace(match[1])
			var tableData []map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &tableData); err == nil {
				result.Tables = append(result.Tables, tableData)
			} else {
				maxLen := 500
				if len(jsonStr) < maxLen {
					maxLen = len(jsonStr)
				}
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse table JSON: %s", jsonStr[:maxLen]))
			}
		}
	}

	// 4. CSV Download Link - Extract ALL matches
	reCSV := regexp.MustCompile(`\[.*?\]\((data:text/csv;base64,[A-Za-z0-9+/=]+)\)`)
	allCSVMatches := reCSV.FindAllStringSubmatch(resp, -1)
	for _, match := range allCSVMatches {
		if len(match) > 1 {
			result.CSVBlocks = append(result.CSVBlocks, match[1])
		}
	}

	return result
}

// generateEChartsBlock generates a valid ECharts JSON block
func generateEChartsBlock(index int) string {
	return fmt.Sprintf(`{
  "title": {"text": "Chart %d"},
  "xAxis": {"type": "category", "data": ["A", "B", "C"]},
  "yAxis": {"type": "value"},
  "series": [{"data": [%d, %d, %d], "type": "bar"}]
}`, index, index*10, index*20, index*30)
}

// generateImageBlock generates a valid base64 image markdown
func generateImageBlock(index int) string {
	// Simple 1x1 PNG base64
	return fmt.Sprintf("![Chart %d](data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==)", index)
}

// generateTableBlock generates a valid table JSON block
func generateTableBlock(index int) string {
	return fmt.Sprintf(`[
  {"id": %d, "name": "Item %d", "value": %d},
  {"id": %d, "name": "Item %d", "value": %d}
]`, index, index, index*100, index+1, index+1, (index+1)*100)
}

// generateCSVBlock generates a valid CSV data URL
func generateCSVBlock(index int) string {
	return fmt.Sprintf("[Download CSV %d](data:text/csv;base64,aWQsbmFtZSx2YWx1ZQoxLEl0ZW0xLDEwMAoyLEl0ZW0yLDIwMA==)", index)
}

// TestResponseParser_Property_ParsingCompleteness tests that all blocks are extracted
// **Validates: Requirements 4.1, 4.2, 4.3, 4.6**
// Property 6: Response Parsing Completeness
func TestResponseParser_Property_ParsingCompleteness(t *testing.T) {
	// Property: For any LLM response containing N json:echarts blocks, M base64 images,
	// and K json:table blocks, the parser SHALL extract exactly N + M + K items.

	property := func(seed uint8) bool {
		// Generate random counts (1-5 of each type)
		numECharts := int(seed%5) + 1
		numImages := int((seed/5)%5) + 1
		numTables := int((seed/25)%5) + 1

		// Build response with multiple blocks
		var sb strings.Builder
		sb.WriteString("Here is the analysis result:\n\n")

		// Add ECharts blocks
		for i := 0; i < numECharts; i++ {
			sb.WriteString("```json:echarts\n")
			sb.WriteString(generateEChartsBlock(i + 1))
			sb.WriteString("\n```\n\n")
		}

		// Add image blocks
		for i := 0; i < numImages; i++ {
			sb.WriteString(generateImageBlock(i + 1))
			sb.WriteString("\n\n")
		}

		// Add table blocks
		for i := 0; i < numTables; i++ {
			sb.WriteString("```json:table\n")
			sb.WriteString(generateTableBlock(i + 1))
			sb.WriteString("\n```\n\n")
		}

		resp := sb.String()
		result := parseResponse(resp)

		// Property: Should extract exactly numECharts ECharts blocks
		if len(result.EChartsBlocks) != numECharts {
			t.Logf("ECharts: expected %d, got %d", numECharts, len(result.EChartsBlocks))
			return false
		}

		// Property: Should extract exactly numImages images
		if len(result.Images) != numImages {
			t.Logf("Images: expected %d, got %d", numImages, len(result.Images))
			return false
		}

		// Property: Should extract exactly numTables tables
		if len(result.Tables) != numTables {
			t.Logf("Tables: expected %d, got %d", numTables, len(result.Tables))
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestResponseParser_Property_JSONErrorLogging tests that JSON errors are logged with truncated content
// **Validates: Requirements 4.5**
// Property 7: JSON Error Logging
func TestResponseParser_Property_JSONErrorLogging(t *testing.T) {
	// Property: For any invalid JSON content in a response, the parser SHALL log an error
	// message containing at most the first 500 characters of the problematic content.

	property := func(seed uint16) bool {
		// Generate invalid JSON of varying lengths
		invalidJSONLength := int(seed%1000) + 100 // 100-1099 characters
		invalidJSON := strings.Repeat("x", invalidJSONLength)

		resp := fmt.Sprintf("```json:echarts\n%s\n```", invalidJSON)
		result := parseResponse(resp)

		// Property: Should have at least one error
		if len(result.Errors) == 0 {
			t.Logf("Expected error for invalid JSON, got none")
			return false
		}

		// Property: Error message should contain at most 500 characters of the invalid content
		for _, errMsg := range result.Errors {
			// The error message format is "Failed to parse echarts JSON: <content>"
			// Check that the content portion is at most 500 characters
			if strings.Contains(errMsg, invalidJSON) && len(invalidJSON) > 500 {
				t.Logf("Error message contains more than 500 characters of invalid content")
				return false
			}
		}

		// Property: No ECharts blocks should be extracted from invalid JSON
		if len(result.EChartsBlocks) != 0 {
			t.Logf("Expected 0 ECharts blocks for invalid JSON, got %d", len(result.EChartsBlocks))
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestResponseParser_Property_MixedContentExtraction tests extraction from mixed content
// **Validates: Requirements 4.1, 4.2, 4.3, 4.6**
func TestResponseParser_Property_MixedContentExtraction(t *testing.T) {
	// Property: Parser should correctly extract all types even when mixed with other content

	property := func(seed uint8) bool {
		// Build response with mixed content
		var sb strings.Builder
		sb.WriteString("# Analysis Report\n\n")
		sb.WriteString("Here are the results of the analysis:\n\n")

		// Add some text before first chart
		sb.WriteString("## Sales Overview\n\n")
		sb.WriteString("The following chart shows the sales trend:\n\n")

		// Add ECharts
		sb.WriteString("```json:echarts\n")
		sb.WriteString(generateEChartsBlock(1))
		sb.WriteString("\n```\n\n")

		// Add more text
		sb.WriteString("As we can see from the chart above, sales are increasing.\n\n")

		// Add image
		sb.WriteString("Here is a visualization:\n\n")
		sb.WriteString(generateImageBlock(1))
		sb.WriteString("\n\n")

		// Add table
		sb.WriteString("## Detailed Data\n\n")
		sb.WriteString("```json:table\n")
		sb.WriteString(generateTableBlock(1))
		sb.WriteString("\n```\n\n")

		// Add CSV
		sb.WriteString("Download the data: ")
		sb.WriteString(generateCSVBlock(1))
		sb.WriteString("\n\n")

		// Add another ECharts at the end
		sb.WriteString("## Additional Chart\n\n")
		sb.WriteString("```json:echarts\n")
		sb.WriteString(generateEChartsBlock(2))
		sb.WriteString("\n```\n\n")

		resp := sb.String()
		result := parseResponse(resp)

		// Property: Should extract 2 ECharts blocks
		if len(result.EChartsBlocks) != 2 {
			t.Logf("ECharts: expected 2, got %d", len(result.EChartsBlocks))
			return false
		}

		// Property: Should extract 1 image
		if len(result.Images) != 1 {
			t.Logf("Images: expected 1, got %d", len(result.Images))
			return false
		}

		// Property: Should extract 1 table
		if len(result.Tables) != 1 {
			t.Logf("Tables: expected 1, got %d", len(result.Tables))
			return false
		}

		// Property: Should extract 1 CSV
		if len(result.CSVBlocks) != 1 {
			t.Logf("CSV: expected 1, got %d", len(result.CSVBlocks))
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestResponseParser_Property_EmptyResponse tests handling of empty responses
// **Validates: Requirements 4.6**
func TestResponseParser_Property_EmptyResponse(t *testing.T) {
	// Property: Empty or text-only responses should return empty results without errors

	property := func(seed uint8) bool {
		testCases := []string{
			"",
			"Just some text without any charts or tables.",
			"# Heading\n\nSome paragraph text.\n\n## Another heading",
			"```python\nprint('hello')\n```", // Code block but not json:echarts
		}

		for _, resp := range testCases {
			result := parseResponse(resp)

			// Property: Should have no extracted items
			if len(result.EChartsBlocks) != 0 {
				t.Logf("Expected 0 ECharts, got %d for response: %s", len(result.EChartsBlocks), resp[:min(50, len(resp))])
				return false
			}
			if len(result.Images) != 0 {
				t.Logf("Expected 0 Images, got %d", len(result.Images))
				return false
			}
			if len(result.Tables) != 0 {
				t.Logf("Expected 0 Tables, got %d", len(result.Tables))
				return false
			}
			if len(result.CSVBlocks) != 0 {
				t.Logf("Expected 0 CSV, got %d", len(result.CSVBlocks))
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}
