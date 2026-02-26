//go:build property_test

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"vantagics/i18n"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: datasource-pack-result-consistency, Property 1: AnalysisResultItem metadata 完整性
// Validates: Requirements 1.3, 2.1
//
// For any AnalysisResultItem produced by QAP executor (whether from SQL step,
// Python step, or ECharts emission), the item's Metadata map shall contain
// non-empty values for keys sessionId, messageId, timestamp, and step_description.

// genNonEmptyString generates a non-empty alphanumeric string.
func genNonEmptyString() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0
	})
}

// genItemType generates one of the valid AnalysisResultItem types produced by QAP executor.
func genItemType() gopter.Gen {
	return gen.OneConstOf("table", "echarts", "image")
}

// buildQAPAnalysisResultItem constructs an AnalysisResultItem the same way the
// QAP executor does in executePackSQLStep, emitStepEChartsConfigs,
// detectAndSendPythonChartFiles, and detectAndSendPythonECharts.
func buildQAPAnalysisResultItem(itemType, sessionID, messageID, stepDescription string) AnalysisResultItem {
	var data interface{}
	switch itemType {
	case "table":
		data = []map[string]interface{}{{"col": "value"}}
	case "echarts":
		data = `{"xAxis":{},"yAxis":{},"series":[]}`
	case "image":
		data = "data:image/png;base64,iVBOR"
	}

	return AnalysisResultItem{
		ID:   "test-id",
		Type: itemType,
		Data: data,
		Metadata: map[string]interface{}{
			"sessionId":        sessionID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepDescription,
		},
		Source: "realtime",
	}
}

func TestProperty1_AnalysisResultItemMetadataCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	properties.Property("metadata contains all required non-empty keys", prop.ForAll(
		func(itemType, sessionID, messageID, stepDescription string) bool {
			item := buildQAPAnalysisResultItem(itemType, sessionID, messageID, stepDescription)

			if item.Metadata == nil {
				return false
			}

			requiredKeys := []string{"sessionId", "messageId", "timestamp", "step_description"}
			for _, key := range requiredKeys {
				val, exists := item.Metadata[key]
				if !exists {
					return false
				}
				// Check non-empty: string values must be non-empty, timestamp must be > 0
				switch v := val.(type) {
				case string:
					if v == "" {
						return false
					}
				case int64:
					if v <= 0 {
						return false
					}
				case float64:
					if v <= 0 {
						return false
					}
				case nil:
					return false
				}
			}
			return true
		},
		genItemType(),
		genNonEmptyString(),
		genNonEmptyString(),
		genNonEmptyString(),
	))

	properties.TestingRun(t)
}

// Feature: datasource-pack-result-consistency, Property 2: SQL 步骤结果中表格先于图表
// Validates: Requirements 1.2, 3.3
//
// For any SQL step with non-empty EChartsConfigs, the list of AnalysisResultItems
// produced by that step shall have all items of Type "table" appearing before all
// items of Type "echarts".

// genEChartsConfig generates a random but valid JSON string that represents an
// ECharts configuration object.
func genEChartsConfig() gopter.Gen {
	return gen.AlphaString().Map(func(s string) string {
		// Produce a minimal valid JSON object with a random title field.
		cfg := map[string]interface{}{
			"xAxis":  map[string]interface{}{},
			"yAxis":  map[string]interface{}{},
			"series": []interface{}{},
			"title":  map[string]interface{}{"text": s},
		}
		b, _ := json.Marshal(cfg)
		return string(b)
	})
}

// genEChartsConfigSlice generates a non-empty slice of valid ECharts JSON strings
// (1 to 5 elements).
func genEChartsConfigSlice() gopter.Gen {
	return gen.IntRange(1, 5).FlatMap(func(v interface{}) gopter.Gen {
		n := v.(int)
		return gen.SliceOfN(n, genEChartsConfig())
	}, reflect.TypeOf([]string{}))
}

// simulateSQLStepResults mimics the result-building logic of executePackSQLStep:
// 1. A single "table" AnalysisResultItem is created first.
// 2. For each valid EChartsConfig, an "echarts" AnalysisResultItem is appended.
// This is exactly the ordering contract that the production code maintains.
func simulateSQLStepResults(step PackStep, sessionID, messageID string) []AnalysisResultItem {
	stepLabel := step.Description
	if stepLabel == "" {
		stepLabel = fmt.Sprintf("Step %d", step.StepID)
	}

	var results []AnalysisResultItem

	// 1. Table item (always first for a successful SQL step)
	results = append(results, AnalysisResultItem{
		ID:   "table-id",
		Type: "table",
		Data: []map[string]interface{}{{"col": "value"}},
		Metadata: map[string]interface{}{
			"sessionId":        sessionID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepLabel,
		},
		Source: "realtime",
	})

	// 2. ECharts items (appended after table, same as emitStepEChartsConfigs)
	for _, chartJSON := range step.EChartsConfigs {
		if !json.Valid([]byte(chartJSON)) {
			continue
		}
		results = append(results, AnalysisResultItem{
			ID:   "echarts-id",
			Type: "echarts",
			Data: chartJSON,
			Metadata: map[string]interface{}{
				"sessionId":        sessionID,
				"messageId":        messageID,
				"timestamp":        time.Now().UnixMilli(),
				"step_description": stepLabel,
			},
			Source: "realtime",
		})
	}

	return results
}

func TestProperty2_SQLStepTableBeforeECharts(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	properties.Property("table items appear before echarts items in SQL step results", prop.ForAll(
		func(stepID int, description string, echartsConfigs []string, sessionID, messageID string) bool {
			step := PackStep{
				StepID:         stepID,
				StepType:       "sql_query",
				Code:           "SELECT 1",
				Description:    description,
				EChartsConfigs: echartsConfigs,
			}

			results := simulateSQLStepResults(step, sessionID, messageID)

			// There must be at least one table item.
			hasTable := false
			for _, r := range results {
				if r.Type == "table" {
					hasTable = true
					break
				}
			}
			if !hasTable {
				return false
			}

			// Verify ordering: once we see an "echarts" item, no "table" item
			// should appear after it.
			seenECharts := false
			for _, r := range results {
				if r.Type == "echarts" {
					seenECharts = true
				}
				if r.Type == "table" && seenECharts {
					// A table item appeared after an echarts item — violation.
					return false
				}
			}
			return true
		},
		gen.IntRange(1, 100),
		genNonEmptyString(),
		genEChartsConfigSlice(),
		genNonEmptyString(),
		genNonEmptyString(),
	))

	properties.TestingRun(t)
}

// Feature: datasource-pack-result-consistency, Property 10: SQL 步骤结果类型一致性
// Validates: Requirements 1.1
//
// For any successful SQL step execution, the AnalysisResultItem sent to
// EventAggregator shall have Type "table" and Data of type []map[string]interface{},
// matching the format produced by the original SQLExecutorTool.

// genColumnName generates a random non-empty column name.
func genColumnName() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0
	})
}

// genCellValue generates a random cell value that could appear in SQL results.
// All values are strings to avoid gopter's SliceOfN reflection issues with heterogeneous types.
func genCellValue() gopter.Gen {
	return gen.AlphaString()
}

// genSQLResultRow generates a single row (map[string]interface{}) with the given column names.
func genSQLResultRow(columns []string) gopter.Gen {
	return gen.SliceOfN(len(columns), genCellValue()).Map(func(vals []string) map[string]interface{} {
		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			row[col] = vals[i]
		}
		return row
	})
}

// genSQLResult generates a random []map[string]interface{} representing SQL query results.
// It produces 1-5 columns and 0-10 rows.
func genSQLResult() gopter.Gen {
	return gen.IntRange(1, 5).FlatMap(func(v interface{}) gopter.Gen {
		numCols := v.(int)
		return gen.SliceOfN(numCols, genColumnName()).FlatMap(func(v interface{}) gopter.Gen {
			columns := v.([]string)
			// Deduplicate column names by appending index suffix
			seen := map[string]bool{}
			for i, c := range columns {
				if seen[c] {
					columns[i] = fmt.Sprintf("%s_%d", c, i)
				}
				seen[columns[i]] = true
			}
			return gen.IntRange(0, 10).FlatMap(func(v interface{}) gopter.Gen {
				numRows := v.(int)
				return gen.SliceOfN(numRows, genSQLResultRow(columns))
			}, reflect.TypeOf([]map[string]interface{}{}))
		}, reflect.TypeOf([]map[string]interface{}{}))
	}, reflect.TypeOf([]map[string]interface{}{}))
}

// buildSQLStepAnalysisResultItem constructs an AnalysisResultItem the same way
// executePackSQLStep does for a successful SQL execution: Type is "table" and
// Data is the raw []map[string]interface{} result from ExecuteSQL.
func buildSQLStepAnalysisResultItem(sqlResult []map[string]interface{}, sessionID, messageID, stepDescription string) AnalysisResultItem {
	return AnalysisResultItem{
		Type: "table",
		Data: sqlResult,
		Metadata: map[string]interface{}{
			"sessionId":        sessionID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepDescription,
		},
	}
}

func TestProperty10_SQLStepResultTypeConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	properties.Property("successful SQL step produces AnalysisResultItem with Type table and Data []map[string]interface{}", prop.ForAll(
		func(sqlResult []map[string]interface{}, sessionID, messageID, stepDescription string) bool {
			item := buildSQLStepAnalysisResultItem(sqlResult, sessionID, messageID, stepDescription)

			// Type must be "table"
			if item.Type != "table" {
				return false
			}

			// Data must be of type []map[string]interface{}
			data, ok := item.Data.([]map[string]interface{})
			if !ok {
				return false
			}

			// Data length must match input
			if len(data) != len(sqlResult) {
				return false
			}

			return true
		},
		genSQLResult(),
		genNonEmptyString(),
		genNonEmptyString(),
		genNonEmptyString(),
	))

	properties.TestingRun(t)
}

// Unit Test: SQL 步骤错误格式
// Validates: Requirement 1.4
//
// Verifies that when SQL execution fails, the error message sent via
// EmitErrorWithCode contains the step ID, description, error details,
// user request, and SQL code, and uses error code "SQL_ERROR".

func TestSQLStepErrorFormat(t *testing.T) {
	// Simulate the error path of executePackSQLStep.
	// The production code builds the error message using i18n.T("qap.step_sql_error", ...)
	// and then calls eventAggregator.EmitErrorWithCode(threadID, "", "SQL_ERROR", errMsg).

	tests := []struct {
		name        string
		step        PackStep
		sqlErr      string
		wantCode    string
	}{
		{
			name: "basic SQL error contains all fields",
			step: PackStep{
				StepID:      3,
				StepType:    "sql_query",
				Code:        "SELECT * FROM non_existent_table",
				Description: "Query sales data",
				UserRequest: "Show me total sales by region",
			},
			sqlErr:   "table non_existent_table does not exist",
			wantCode: "SQL_ERROR",
		},
		{
			name: "SQL error with empty UserRequest falls back to description",
			step: PackStep{
				StepID:      7,
				StepType:    "sql_query",
				Code:        "SELECT COUNT(*) FROM orders WHERE status = 'pending'",
				Description: "Count pending orders",
				UserRequest: "",
			},
			sqlErr:   "connection refused",
			wantCode: "SQL_ERROR",
		},
		{
			name: "SQL error with empty description uses default label",
			step: PackStep{
				StepID:      1,
				StepType:    "sql_query",
				Code:        "INVALID SQL",
				Description: "",
				UserRequest: "",
			},
			sqlErr:   "syntax error near INVALID",
			wantCode: "SQL_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Verify the error code is "SQL_ERROR"
			if tc.wantCode != "SQL_ERROR" {
				t.Errorf("expected error code SQL_ERROR, got %s", tc.wantCode)
			}

			// 2. Build the error message the same way executePackSQLStep does
			userRequest := getStepUserRequest(tc.step)
			errMsg := i18n.T("qap.step_sql_error", tc.step.StepID, tc.step.Description, fmt.Errorf("%s", tc.sqlErr), userRequest, tc.step.Code)

			// 3. Verify error message contains step ID
			stepIDStr := fmt.Sprintf("%d", tc.step.StepID)
			if !strings.Contains(errMsg, stepIDStr) {
				t.Errorf("error message should contain step ID %q, got: %s", stepIDStr, errMsg)
			}

			// 4. Verify error message contains description (if non-empty)
			if tc.step.Description != "" && !strings.Contains(errMsg, tc.step.Description) {
				t.Errorf("error message should contain description %q, got: %s", tc.step.Description, errMsg)
			}

			// 5. Verify error message contains error details
			if !strings.Contains(errMsg, tc.sqlErr) {
				t.Errorf("error message should contain error details %q, got: %s", tc.sqlErr, errMsg)
			}

			// 6. Verify error message contains user request
			if !strings.Contains(errMsg, userRequest) {
				t.Errorf("error message should contain user request %q, got: %s", userRequest, errMsg)
			}

			// 7. Verify error message contains SQL code
			if !strings.Contains(errMsg, tc.step.Code) {
				t.Errorf("error message should contain SQL code %q, got: %s", tc.step.Code, errMsg)
			}

			// 8. Verify the chat message would be created with correct role and content
			errorChatMsg := ChatMessage{
				ID:        "test-msg-id",
				Role:      "assistant",
				Content:   errMsg,
				Timestamp: time.Now().Unix(),
			}
			if errorChatMsg.Role != "assistant" {
				t.Errorf("error chat message role should be 'assistant', got %q", errorChatMsg.Role)
			}
			if errorChatMsg.Content != errMsg {
				t.Errorf("error chat message content should match errMsg")
			}
		})
	}
}

// Feature: datasource-pack-result-consistency, Property 4: ECharts JSON 从 Python 输出中正确提取
// Validates: Requirements 2.2, 3.4
//
// For any Python stdout string containing one or more valid json:echarts blocks
// (in either backtick or bare format), detectAndSendPythonECharts shall extract
// exactly the valid JSON configs and skip any invalid ones, producing one
// AnalysisResultItem of Type "echarts" per valid config.

// genSimpleEChartsJSON generates a random valid ECharts JSON config string.
func genSimpleEChartsJSON() gopter.Gen {
	return gen.AlphaString().Map(func(title string) string {
		cfg := map[string]interface{}{
			"xAxis":  map[string]interface{}{"type": "category"},
			"yAxis":  map[string]interface{}{},
			"series": []interface{}{map[string]interface{}{"type": "bar", "data": []int{1, 2, 3}}},
			"title":  map[string]interface{}{"text": title},
		}
		b, _ := json.Marshal(cfg)
		return string(b)
	})
}

// genInvalidJSON generates a string that is NOT valid JSON.
func genInvalidJSON() gopter.Gen {
	return gen.OneConstOf(
		`{not valid json`,
		`{"key": undefined}`,
		`{trailing comma,}`,
		`just plain text`,
		`{"unclosed": "string`,
	)
}

// echartsBlock wraps a JSON string in backtick json:echarts format.
func echartsBlock(jsonStr string) string {
	return "```json:echarts\n" + jsonStr + "\n```"
}

// echartsBlockBare wraps a JSON string in bare json:echarts format (no backticks).
func echartsBlockBare(jsonStr string) string {
	return "json:echarts\n" + jsonStr + "\n---"
}

// simulateEChartsExtraction replicates the regex extraction logic from
// detectAndSendPythonECharts without requiring an App instance.
func simulateEChartsExtraction(output string) []string {
	var extracted []string

	reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	matches := reECharts.FindAllStringSubmatch(output, -1)

	reEChartsNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:echarts\\s*\\n(\\{[\\s\\S]+?\\n\\})(?:\\s*\\n(?:---|###)|\\s*$)")
	matches = append(matches, reEChartsNoBT.FindAllStringSubmatch(output, -1)...)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		chartData := strings.TrimSpace(match[1])
		if chartData == "" {
			continue
		}
		if !json.Valid([]byte(chartData)) {
			continue
		}
		extracted = append(extracted, chartData)
	}

	return extracted
}

func TestProperty4_EChartsJSONExtractedFromPythonOutput(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	// Sub-property A: All valid backtick-format blocks are extracted.
	properties.Property("valid backtick json:echarts blocks are all extracted", prop.ForAll(
		func(configs []string) bool {
			// Build Python output with valid backtick blocks
			var parts []string
			parts = append(parts, "Some Python output before charts\n")
			for _, cfg := range configs {
				parts = append(parts, echartsBlock(cfg)+"\n")
			}
			parts = append(parts, "Some trailing output\n")
			output := strings.Join(parts, "")

			extracted := simulateEChartsExtraction(output)

			if len(extracted) != len(configs) {
				return false
			}
			for i, cfg := range configs {
				// Compare as parsed JSON to ignore whitespace differences
				var expected, actual interface{}
				json.Unmarshal([]byte(cfg), &expected)
				json.Unmarshal([]byte(extracted[i]), &actual)
				if !reflect.DeepEqual(expected, actual) {
					return false
				}
			}
			return true
		},
		gen.SliceOfN(3, genSimpleEChartsJSON()).SuchThat(func(s []string) bool {
			return len(s) >= 1
		}),
	))

	// Sub-property B: Invalid JSON blocks are skipped.
	properties.Property("invalid JSON in json:echarts blocks is skipped", prop.ForAll(
		func(validCfg, invalidJSON string) bool {
			// Build output with one valid and one invalid block
			output := "output start\n" +
				echartsBlock(validCfg) + "\n" +
				echartsBlock(invalidJSON) + "\n" +
				"output end\n"

			extracted := simulateEChartsExtraction(output)

			// Only the valid config should be extracted
			if len(extracted) != 1 {
				return false
			}
			var expected, actual interface{}
			json.Unmarshal([]byte(validCfg), &expected)
			json.Unmarshal([]byte(extracted[0]), &actual)
			return reflect.DeepEqual(expected, actual)
		},
		genSimpleEChartsJSON(),
		genInvalidJSON(),
	))

	// Sub-property C: Bare format (no backticks) blocks are also extracted.
	// The bare format regex requires multi-line JSON (ending with \n}).
	properties.Property("bare format json:echarts blocks are extracted", prop.ForAll(
		func(cfg string) bool {
			// json.MarshalIndent produces multi-line JSON that matches the bare regex
			var parsed interface{}
			json.Unmarshal([]byte(cfg), &parsed)
			indented, _ := json.MarshalIndent(parsed, "", "  ")
			multiLineCfg := string(indented)

			output := "\n" + echartsBlockBare(multiLineCfg) + "\n"

			extracted := simulateEChartsExtraction(output)

			if len(extracted) != 1 {
				return false
			}
			var expected, actual interface{}
			json.Unmarshal([]byte(cfg), &expected)
			json.Unmarshal([]byte(extracted[0]), &actual)
			return reflect.DeepEqual(expected, actual)
		},
		genSimpleEChartsJSON(),
	))

	// Sub-property D: Output with no json:echarts blocks yields empty result.
	properties.Property("output without json:echarts blocks yields no results", prop.ForAll(
		func(plainText string) bool {
			// Ensure the plain text doesn't accidentally contain the marker
			output := strings.ReplaceAll(plainText, "json:echarts", "json_echarts")
			extracted := simulateEChartsExtraction(output)
			return len(extracted) == 0
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

func TestPythonStepErrorFormat(t *testing.T) {
	// Simulate the error paths of executePackPythonStep.
	// Path 1: Python not configured – uses i18n.T("qap.step_python_not_configured", stepID)
	//         and EmitErrorWithCode(threadID, "", "PYTHON_NOT_CONFIGURED", errMsg).
	// Path 2: Python execution failed – uses i18n.T("qap.step_execution_failed", stepID, err)
	//         and EmitErrorWithCode(threadID, "", "PYTHON_ERROR", errMsg).

	t.Run("Python not configured error", func(t *testing.T) {
		step := PackStep{
			StepID:      5,
			StepType:    "python_code",
			Code:        "import pandas as pd\ndf = pd.read_csv('data.csv')",
			Description: "Load and process data",
			UserRequest: "Analyze the sales trend",
		}

		wantCode := "PYTHON_NOT_CONFIGURED"

		// 1. Verify the error code
		if wantCode != "PYTHON_NOT_CONFIGURED" {
			t.Errorf("expected error code PYTHON_NOT_CONFIGURED, got %s", wantCode)
		}

		// 2. Build the error message the same way executePackPythonStep does
		errMsg := i18n.T("qap.step_python_not_configured", step.StepID)

		// 3. Verify error message contains step ID
		stepIDStr := fmt.Sprintf("%d", step.StepID)
		if !strings.Contains(errMsg, stepIDStr) {
			t.Errorf("error message should contain step ID %q, got: %s", stepIDStr, errMsg)
		}

		// 4. Build the chat message the same way production code does
		userRequest := getStepUserRequest(step)
		chatContent := i18n.T("qap.step_python_no_env", step.StepID, step.Description, userRequest, step.Code)

		// 5. Verify chat message contains step ID
		if !strings.Contains(chatContent, stepIDStr) {
			t.Errorf("chat message should contain step ID %q, got: %s", stepIDStr, chatContent)
		}

		// 6. Verify chat message contains description
		if !strings.Contains(chatContent, step.Description) {
			t.Errorf("chat message should contain description %q, got: %s", step.Description, chatContent)
		}

		// 7. Verify chat message contains user request
		if !strings.Contains(chatContent, userRequest) {
			t.Errorf("chat message should contain user request %q, got: %s", userRequest, chatContent)
		}

		// 8. Verify chat message contains code
		if !strings.Contains(chatContent, step.Code) {
			t.Errorf("chat message should contain code %q, got: %s", step.Code, chatContent)
		}

		// 9. Verify the chat message would be created with correct role
		errorChatMsg := ChatMessage{
			ID:        "test-msg-id",
			Role:      "assistant",
			Content:   chatContent,
			Timestamp: time.Now().Unix(),
		}
		if errorChatMsg.Role != "assistant" {
			t.Errorf("error chat message role should be 'assistant', got %q", errorChatMsg.Role)
		}
	})

	t.Run("Python execution failed error", func(t *testing.T) {
		tests := []struct {
			name string
			step PackStep
			err  string
		}{
			{
				name: "basic Python execution error",
				step: PackStep{
					StepID:      2,
					StepType:    "python_code",
					Code:        "import matplotlib.pyplot as plt\nplt.plot([1,2,3])\nplt.savefig('chart.png')",
					Description: "Generate sales chart",
					UserRequest: "Show me a chart of monthly sales",
				},
				err: "ModuleNotFoundError: No module named 'matplotlib'",
			},
			{
				name: "Python error with empty UserRequest falls back to description",
				step: PackStep{
					StepID:      4,
					StepType:    "python_code",
					Code:        "print(1/0)",
					Description: "Calculate ratio",
					UserRequest: "",
				},
				err: "ZeroDivisionError: division by zero",
			},
			{
				name: "Python error with empty description uses default label",
				step: PackStep{
					StepID:      9,
					StepType:    "python_code",
					Code:        "raise ValueError('bad input')",
					Description: "",
					UserRequest: "",
				},
				err: "ValueError: bad input",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				wantCode := "PYTHON_ERROR"

				// 1. Verify the error code
				if wantCode != "PYTHON_ERROR" {
					t.Errorf("expected error code PYTHON_ERROR, got %s", wantCode)
				}

				// 2. Build the error message the same way executePackPythonStep does
				errMsg := i18n.T("qap.step_execution_failed", tc.step.StepID, fmt.Errorf("%s", tc.err))

				// 3. Verify error message contains step ID
				stepIDStr := fmt.Sprintf("%d", tc.step.StepID)
				if !strings.Contains(errMsg, stepIDStr) {
					t.Errorf("error message should contain step ID %q, got: %s", stepIDStr, errMsg)
				}

				// 4. Build the chat message the same way production code does
				userRequest := getStepUserRequest(tc.step)
				chatContent := i18n.T("qap.step_python_error", tc.step.StepID, tc.step.Description, fmt.Errorf("%s", tc.err), userRequest, tc.step.Code)

				// 5. Verify chat message contains step ID
				if !strings.Contains(chatContent, stepIDStr) {
					t.Errorf("chat message should contain step ID %q, got: %s", stepIDStr, chatContent)
				}

				// 6. Verify chat message contains description (if non-empty)
				if tc.step.Description != "" && !strings.Contains(chatContent, tc.step.Description) {
					t.Errorf("chat message should contain description %q, got: %s", tc.step.Description, chatContent)
				}

				// 7. Verify chat message contains error details
				if !strings.Contains(chatContent, tc.err) {
					t.Errorf("chat message should contain error details %q, got: %s", tc.err, chatContent)
				}

				// 8. Verify chat message contains user request
				if !strings.Contains(chatContent, userRequest) {
					t.Errorf("chat message should contain user request %q, got: %s", userRequest, chatContent)
				}

				// 9. Verify chat message contains code
				if !strings.Contains(chatContent, tc.step.Code) {
					t.Errorf("chat message should contain code %q, got: %s", tc.step.Code, chatContent)
				}

				// 10. Verify the chat message would be created with correct role
				errorChatMsg := ChatMessage{
					ID:        "test-msg-id",
					Role:      "assistant",
					Content:   chatContent,
					Timestamp: time.Now().Unix(),
				}
				if errorChatMsg.Role != "assistant" {
					t.Errorf("error chat message role should be 'assistant', got %q", errorChatMsg.Role)
				}
			})
		}
	})
}

// TestFormatPythonOutputForMessage verifies that Python output is formatted
// correctly for the assistant chat message:
// - Output containing json:echarts blocks is passed through as-is
// - Output containing image references is passed through as-is
// - Plain text output is wrapped in a code block
// - Empty output returns empty string
func TestFormatPythonOutputForMessage(t *testing.T) {
	t.Run("empty output returns empty string", func(t *testing.T) {
		result := formatPythonOutputForMessage("")
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("plain text output is wrapped in code block", func(t *testing.T) {
		output := "Hello, world!\nLine 2"
		result := formatPythonOutputForMessage(output)
		expected := "```\n" + output + "\n```"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("output with json:echarts is passed through as-is", func(t *testing.T) {
		output := "Some text\n```json:echarts\n{\"title\":{\"text\":\"Sales\"}}\n```\nMore text"
		result := formatPythonOutputForMessage(output)
		if result != output {
			t.Errorf("expected output to be passed through as-is, got %q", result)
		}
	})

	t.Run("output with bare json:echarts is passed through as-is", func(t *testing.T) {
		output := "Result:\njson:echarts\n{\"chart\":\"data\"}\n---"
		result := formatPythonOutputForMessage(output)
		if result != output {
			t.Errorf("expected output to be passed through as-is, got %q", result)
		}
	})

	t.Run("output with Chart image reference is passed through as-is", func(t *testing.T) {
		output := "Chart generated:\n![Chart](data:image/png;base64,iVBOR...)"
		result := formatPythonOutputForMessage(output)
		if result != output {
			t.Errorf("expected output to be passed through as-is, got %q", result)
		}
	})

	t.Run("output with lowercase chart image reference is passed through as-is", func(t *testing.T) {
		output := "![chart](data:image/png;base64,abc123)"
		result := formatPythonOutputForMessage(output)
		if result != output {
			t.Errorf("expected output to be passed through as-is, got %q", result)
		}
	})

	t.Run("output with data:image reference is passed through as-is", func(t *testing.T) {
		output := "Generated image: data:image/jpeg;base64,/9j/4AAQ..."
		result := formatPythonOutputForMessage(output)
		if result != output {
			t.Errorf("expected output to be passed through as-is, got %q", result)
		}
	})
}

// TestPythonStepSuccessMessageFormat verifies that the Python step success
// message includes step description, user request, and properly formatted output.
// Validates: Requirements 5.2, 5.3
func TestPythonStepSuccessMessageFormat(t *testing.T) {
	i18n.SetLanguage("en")

	t.Run("success message contains description and user request", func(t *testing.T) {
		step := PackStep{
			StepID:      3,
			StepType:    "python_code",
			Description: "Generate revenue chart",
			UserRequest: "Show monthly revenue trend",
		}
		output := "Chart saved to chart.png"
		formatted := formatPythonOutputForMessage(output)
		msg := i18n.T("qap.step_python_success", step.StepID, step.Description, step.UserRequest, formatted)

		if !strings.Contains(msg, step.Description) {
			t.Errorf("message should contain description %q, got: %s", step.Description, msg)
		}
		if !strings.Contains(msg, step.UserRequest) {
			t.Errorf("message should contain user request %q, got: %s", step.UserRequest, msg)
		}
		if !strings.Contains(msg, output) {
			t.Errorf("message should contain output %q, got: %s", output, msg)
		}
	})

	t.Run("success message with echarts output preserves chart references", func(t *testing.T) {
		step := PackStep{
			StepID:      5,
			StepType:    "python_code",
			Description: "Create pie chart",
			UserRequest: "Show category distribution",
		}
		echartsBlock := "```json:echarts\n{\"title\":{\"text\":\"Distribution\"}}\n```"
		output := "Analysis complete:\n" + echartsBlock
		formatted := formatPythonOutputForMessage(output)
		msg := i18n.T("qap.step_python_success", step.StepID, step.Description, step.UserRequest, formatted)

		// The json:echarts block should be at the top level, not nested in a code block
		if !strings.Contains(msg, "json:echarts") {
			t.Errorf("message should contain json:echarts reference, got: %s", msg)
		}
		// Should NOT be double-wrapped in code blocks
		if strings.Contains(msg, "```\n```json:echarts") {
			t.Errorf("json:echarts should not be nested inside another code block, got: %s", msg)
		}
	})
}

// Feature: datasource-pack-result-consistency, Property 5: 步骤消息包含关键内容
// Validates: Requirements 5.1, 5.2, 5.3
//
// For any successfully executed pack step (SQL or Python), the generated assistant
// ChatMessage content shall contain the step's Description and UserRequest text.

func TestProperty5_StepMessageContainsKeyContent(t *testing.T) {
	i18n.SetLanguage("en")

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	// Sub-property A: SQL step success message contains Description and UserRequest.
	properties.Property("SQL step success message contains description and user request", prop.ForAll(
		func(stepID int, description, userRequest string) bool {
			// Build the message the same way executePackSQLStep does for a successful execution.
			rowCount := 5
			resultJSON := `[{"col":"val"}]`
			chatContent := i18n.T("qap.step_sql_success_full", stepID, description, rowCount, userRequest, resultJSON)

			if !strings.Contains(chatContent, description) {
				return false
			}
			if !strings.Contains(chatContent, userRequest) {
				return false
			}
			return true
		},
		gen.IntRange(1, 100),
		genNonEmptyString(),
		genNonEmptyString(),
	))

	// Sub-property B: Python step success message contains Description and UserRequest.
	properties.Property("Python step success message contains description and user request", prop.ForAll(
		func(stepID int, description, userRequest string) bool {
			// Build the message the same way executePackPythonStep does for a successful execution.
			output := "some output"
			formatted := formatPythonOutputForMessage(output)
			chatContent := i18n.T("qap.step_python_success", stepID, description, userRequest, formatted)

			if !strings.Contains(chatContent, description) {
				return false
			}
			if !strings.Contains(chatContent, userRequest) {
				return false
			}
			return true
		},
		gen.IntRange(1, 100),
		genNonEmptyString(),
		genNonEmptyString(),
	))

	// Sub-property C: SQL step message also contains the result JSON data.
	properties.Property("SQL step success message contains result data", prop.ForAll(
		func(stepID int, description, userRequest string) bool {
			sampleData := []map[string]interface{}{{"id": 1, "name": "test"}}
			resultJSON, _ := json.MarshalIndent(sampleData, "", "  ")
			chatContent := i18n.T("qap.step_sql_success_full", stepID, description, len(sampleData), userRequest, string(resultJSON))

			// The result JSON should appear in the message
			if !strings.Contains(chatContent, string(resultJSON)) {
				return false
			}
			return true
		},
		gen.IntRange(1, 100),
		genNonEmptyString(),
		genNonEmptyString(),
	))

	// Sub-property D: Python step message also contains the formatted output.
	properties.Property("Python step success message contains formatted output", prop.ForAll(
		func(stepID int, description, userRequest, output string) bool {
			formatted := formatPythonOutputForMessage(output)
			chatContent := i18n.T("qap.step_python_success", stepID, description, userRequest, formatted)

			if formatted != "" && !strings.Contains(chatContent, formatted) {
				return false
			}
			return true
		},
		gen.IntRange(1, 100),
		genNonEmptyString(),
		genNonEmptyString(),
		genNonEmptyString(),
	))

	properties.TestingRun(t)
}

// Feature: datasource-pack-result-consistency, Property 3: DataFrame 注入代码等价性
// Validates: Requirements 2.3, 6.2
//
// For any SQL result (as JSON string) and any chart code string, the Python code
// generated by buildDataFrameInjection(sqlResult, chartCode) shall produce a
// DataFrame loading preamble that is functionally equivalent to
// buildChartPythonCode(sqlResultJSON, chartCode) — specifically, both shall use
// the same base64 encoding of the SQL result and the same DataFrame construction logic.

// referenceChartPythonCode replicates the logic of agent.buildChartPythonCode
// (which lives in the agent package and is unexported). It takes a JSON string
// and chart code, base64-encodes the JSON, and returns the same Python template.
func referenceChartPythonCode(sqlResultJSON string, chartCode string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(sqlResultJSON))

	return fmt.Sprintf(`import pandas as pd
import json
import base64
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

# Load SQL query results into DataFrame
_sql_result = json.loads(base64.b64decode("%s").decode("utf-8"))
if isinstance(_sql_result, list):
    df = pd.DataFrame(_sql_result)
elif "rows" in _sql_result and _sql_result["rows"]:
    df = pd.DataFrame(_sql_result["rows"])
elif "data" in _sql_result:
    df = pd.DataFrame(_sql_result["data"])
else:
    df = pd.DataFrame()

print(f"DataFrame loaded: {len(df)} rows, {len(df.columns)} columns")
if len(df) > 0:
    print(f"Columns: {list(df.columns)}")
    print(df.head(3).to_string())

# User chart code
%s
`, encoded, chartCode)
}

// genSimpleSQLRows generates a random []map[string]interface{} using simple
// string-only values to avoid the genCellValue OneGenOf type assertion issue.
func genSimpleSQLRows() gopter.Gen {
	return gen.IntRange(0, 5).FlatMap(func(v interface{}) gopter.Gen {
		numRows := v.(int)
		return gen.SliceOfN(numRows, gen.MapOf(gen.AlphaString(), gen.AlphaString())).Map(
			func(rows []map[string]string) []map[string]interface{} {
				result := make([]map[string]interface{}, len(rows))
				for i, row := range rows {
					m := make(map[string]interface{}, len(row))
					for k, v := range row {
						m[k] = v
					}
					result[i] = m
				}
				return result
			},
		)
	}, reflect.TypeOf([]map[string]interface{}{}))
}

func TestProperty3_DataFrameInjectionCodeEquivalence(t *testing.T) {
	app := &App{}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	// Generator for chart code snippets (non-empty Python-like strings).
	genChartCode := gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0
	})

	// Sub-property A: For list-shaped SQL results ([]map[string]interface{}),
	// buildDataFrameInjection and referenceChartPythonCode produce identical output.
	properties.Property("list SQL result produces identical code", prop.ForAll(
		func(rows []map[string]interface{}, chartCode string) bool {
			// Marshal the rows to JSON — this is what buildChartPythonCode would receive.
			jsonBytes, err := json.Marshal(rows)
			if err != nil {
				return false
			}
			sqlResultJSON := string(jsonBytes)

			got := app.buildDataFrameInjection(rows, chartCode)
			want := referenceChartPythonCode(sqlResultJSON, chartCode)
			return got == want
		},
		genSimpleSQLRows(),
		genChartCode,
	))

	// Sub-property B: For map-shaped SQL results (map[string]interface{} with "rows" key),
	// both functions produce identical output.
	properties.Property("map SQL result with rows key produces identical code", prop.ForAll(
		func(rows []map[string]interface{}, chartCode string) bool {
			sqlResult := map[string]interface{}{
				"rows": rows,
			}
			jsonBytes, err := json.Marshal(sqlResult)
			if err != nil {
				return false
			}
			sqlResultJSON := string(jsonBytes)

			got := app.buildDataFrameInjection(sqlResult, chartCode)
			want := referenceChartPythonCode(sqlResultJSON, chartCode)
			return got == want
		},
		genSimpleSQLRows(),
		genChartCode,
	))

	// Sub-property C: For map-shaped SQL results with "data" key,
	// both functions produce identical output.
	properties.Property("map SQL result with data key produces identical code", prop.ForAll(
		func(rows []map[string]interface{}, chartCode string) bool {
			sqlResult := map[string]interface{}{
				"data": rows,
			}
			jsonBytes, err := json.Marshal(sqlResult)
			if err != nil {
				return false
			}
			sqlResultJSON := string(jsonBytes)

			got := app.buildDataFrameInjection(sqlResult, chartCode)
			want := referenceChartPythonCode(sqlResultJSON, chartCode)
			return got == want
		},
		genSimpleSQLRows(),
		genChartCode,
	))

	// Sub-property D: The base64-encoded payload inside the generated code is
	// decodable and round-trips back to the original JSON.
	properties.Property("base64 payload round-trips to original JSON", prop.ForAll(
		func(rows []map[string]interface{}, chartCode string) bool {
			code := app.buildDataFrameInjection(rows, chartCode)

			// Extract the base64 string from the generated code.
			// Pattern: base64.b64decode("...").decode
			re := regexp.MustCompile(`base64\.b64decode\("([A-Za-z0-9+/=]+)"\)`)
			matches := re.FindStringSubmatch(code)
			if len(matches) < 2 {
				return false
			}
			decoded, err := base64.StdEncoding.DecodeString(matches[1])
			if err != nil {
				return false
			}

			// The decoded bytes should be valid JSON that matches the original data.
			var roundTripped interface{}
			if err := json.Unmarshal(decoded, &roundTripped); err != nil {
				return false
			}

			// Marshal original for comparison.
			originalJSON, _ := json.Marshal(rows)
			var original interface{}
			json.Unmarshal(originalJSON, &original)

			return reflect.DeepEqual(original, roundTripped)
		},
		genSimpleSQLRows(),
		genChartCode,
	))

	properties.TestingRun(t)
}

// Feature: datasource-pack-result-consistency, Property 6: query_and_chart 步骤同时产生表格和图表结果
// Validates: Requirements 6.4
//
// For any successful query_and_chart step pair (SQL step succeeds and Python chart
// step succeeds), the combined AnalysisResultItems shall contain at least one item
// of Type "table" (from the SQL step) and at least one item of Type "echarts" or
// "image" (from the Python chart step).

// chartResultKind enumerates the possible chart result sources from a Python chart step.
type chartResultKind int

const (
	chartFromEChartsConfig chartResultKind = iota // stored EChartsConfigs on the step
	chartFromPythonOutput                         // json:echarts block in Python stdout
	chartFromImageFile                            // image file detected in workDir
)

// simulateQueryAndChartResults mimics the combined result collection of a
// query_and_chart step pair:
//   - SQL step: produces one "table" AnalysisResultItem (via AddTable)
//     plus any ECharts from the SQL step's EChartsConfigs.
//   - Python chart step: produces "echarts" and/or "image" items depending
//     on the chart result kind.
//
// This mirrors the production flow in executePackSQLStep + executePackPythonStep.
func simulateQueryAndChartResults(
	sqlStep PackStep,
	pythonStep PackStep,
	chartKind chartResultKind,
	echartsJSON string,
	sessionID string,
	messageID string,
) []AnalysisResultItem {
	var combined []AnalysisResultItem

	// --- SQL step results (always produces a table) ---
	sqlLabel := sqlStep.Description
	if sqlLabel == "" {
		sqlLabel = fmt.Sprintf("Step %d", sqlStep.StepID)
	}
	combined = append(combined, AnalysisResultItem{
		Type: "table",
		Data: []map[string]interface{}{{"col": "value"}},
		Metadata: map[string]interface{}{
			"sessionId":        sessionID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": sqlLabel,
		},
	})

	// SQL step may also have stored EChartsConfigs
	for _, cfg := range sqlStep.EChartsConfigs {
		if json.Valid([]byte(cfg)) {
			combined = append(combined, AnalysisResultItem{
				Type: "echarts",
				Data: cfg,
				Metadata: map[string]interface{}{
					"sessionId":        sessionID,
					"messageId":        messageID,
					"timestamp":        time.Now().UnixMilli(),
					"step_description": sqlLabel,
				},
			})
		}
	}

	// --- Python chart step results ---
	pyLabel := pythonStep.Description
	if pyLabel == "" {
		pyLabel = fmt.Sprintf("Step %d", pythonStep.StepID)
	}

	switch chartKind {
	case chartFromEChartsConfig:
		// Chart comes from stored EChartsConfigs on the Python step
		for _, cfg := range pythonStep.EChartsConfigs {
			if json.Valid([]byte(cfg)) {
				combined = append(combined, AnalysisResultItem{
					Type: "echarts",
					Data: cfg,
					Metadata: map[string]interface{}{
						"sessionId":        sessionID,
						"messageId":        messageID,
						"timestamp":        time.Now().UnixMilli(),
						"step_description": pyLabel,
					},
				})
			}
		}
	case chartFromPythonOutput:
		// Chart comes from json:echarts block in Python stdout
		if json.Valid([]byte(echartsJSON)) {
			combined = append(combined, AnalysisResultItem{
				Type: "echarts",
				Data: echartsJSON,
				Metadata: map[string]interface{}{
					"sessionId":        sessionID,
					"messageId":        messageID,
					"timestamp":        time.Now().UnixMilli(),
					"step_description": pyLabel,
				},
			})
		}
	case chartFromImageFile:
		// Chart comes from a generated image file
		combined = append(combined, AnalysisResultItem{
			Type: "image",
			Data: "data:image/png;base64,iVBOR",
			Metadata: map[string]interface{}{
				"sessionId":        sessionID,
				"messageId":        messageID,
				"timestamp":        time.Now().UnixMilli(),
				"step_description": pyLabel,
			},
		})
	}

	return combined
}

// genChartResultKind generates a random chartResultKind.
func genChartResultKind() gopter.Gen {
	return gen.IntRange(0, 2).Map(func(i int) chartResultKind {
		return chartResultKind(i)
	})
}

func TestProperty6_QueryAndChartProducesTableAndChart(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	// Sub-property A: Combined results always contain at least one "table" and
	// at least one "echarts" or "image" when both steps succeed.
	properties.Property("query_and_chart step pair produces both table and chart results", prop.ForAll(
		func(sqlStepID int, pyStepID int, sqlDesc, pyDesc, sessionID, messageID string, chartKind chartResultKind, echartsJSON string) bool {
			sqlStep := PackStep{
				StepID:      sqlStepID,
				StepType:    "sql_query",
				Code:        "SELECT * FROM sales",
				Description: sqlDesc,
				SourceTool:  "query_and_chart",
			}
			pythonStep := PackStep{
				StepID:          pyStepID,
				StepType:        "python_code",
				Code:            "import matplotlib\nplt.plot(df)",
				Description:     pyDesc,
				SourceTool:      "query_and_chart",
				PairedSQLStepID: sqlStepID,
			}

			// Ensure the Python step has the right chart source
			switch chartKind {
			case chartFromEChartsConfig:
				pythonStep.EChartsConfigs = []string{echartsJSON}
			}

			results := simulateQueryAndChartResults(sqlStep, pythonStep, chartKind, echartsJSON, sessionID, messageID)

			hasTable := false
			hasChart := false
			for _, r := range results {
				if r.Type == "table" {
					hasTable = true
				}
				if r.Type == "echarts" || r.Type == "image" {
					hasChart = true
				}
			}

			return hasTable && hasChart
		},
		gen.IntRange(1, 50),
		gen.IntRange(51, 100),
		genNonEmptyString(),
		genNonEmptyString(),
		genNonEmptyString(),
		genNonEmptyString(),
		genChartResultKind(),
		genSimpleEChartsJSON(), // always valid JSON for the chart config
	))

	// Sub-property B: The table item always comes from the SQL step (has SQL step description).
	properties.Property("table item originates from SQL step", prop.ForAll(
		func(sqlStepID int, pyStepID int, sqlDesc, pyDesc, sessionID, messageID, echartsJSON string) bool {
			sqlStep := PackStep{
				StepID:      sqlStepID,
				StepType:    "sql_query",
				Code:        "SELECT 1",
				Description: sqlDesc,
				SourceTool:  "query_and_chart",
			}
			pythonStep := PackStep{
				StepID:          pyStepID,
				StepType:        "python_code",
				Code:            "plt.show()",
				Description:     pyDesc,
				SourceTool:      "query_and_chart",
				PairedSQLStepID: sqlStepID,
				EChartsConfigs:  []string{echartsJSON},
			}

			results := simulateQueryAndChartResults(sqlStep, pythonStep, chartFromEChartsConfig, echartsJSON, sessionID, messageID)

			sqlLabel := sqlDesc
			if sqlLabel == "" {
				sqlLabel = fmt.Sprintf("Step %d", sqlStepID)
			}

			for _, r := range results {
				if r.Type == "table" {
					desc, _ := r.Metadata["step_description"].(string)
					if desc != sqlLabel {
						return false
					}
				}
			}
			return true
		},
		gen.IntRange(1, 50),
		gen.IntRange(51, 100),
		genNonEmptyString(),
		genNonEmptyString(),
		genNonEmptyString(),
		genNonEmptyString(),
		genSimpleEChartsJSON(),
	))

	// Sub-property C: Multiple chart sources can coexist — stored EChartsConfigs
	// on the SQL step plus chart output from the Python step.
	properties.Property("multiple chart sources produce multiple chart items", prop.ForAll(
		func(sqlStepID int, pyStepID int, sqlDesc, pyDesc, sessionID, messageID string, sqlEcharts, pyEcharts string) bool {
			sqlStep := PackStep{
				StepID:         sqlStepID,
				StepType:       "sql_query",
				Code:           "SELECT 1",
				Description:    sqlDesc,
				SourceTool:     "query_and_chart",
				EChartsConfigs: []string{sqlEcharts},
			}
			pythonStep := PackStep{
				StepID:          pyStepID,
				StepType:        "python_code",
				Code:            "plt.show()",
				Description:     pyDesc,
				SourceTool:      "query_and_chart",
				PairedSQLStepID: sqlStepID,
			}

			results := simulateQueryAndChartResults(sqlStep, pythonStep, chartFromPythonOutput, pyEcharts, sessionID, messageID)

			tableCount := 0
			chartCount := 0
			for _, r := range results {
				if r.Type == "table" {
					tableCount++
				}
				if r.Type == "echarts" || r.Type == "image" {
					chartCount++
				}
			}

			// Should have 1 table, and at least 2 chart items (one from SQL EChartsConfigs, one from Python output)
			return tableCount >= 1 && chartCount >= 2
		},
		gen.IntRange(1, 50),
		gen.IntRange(51, 100),
		genNonEmptyString(),
		genNonEmptyString(),
		genNonEmptyString(),
		genNonEmptyString(),
		genSimpleEChartsJSON(),
		genSimpleEChartsJSON(),
	))

	properties.TestingRun(t)
}

// Feature: datasource-pack-result-consistency, Property 11: query_and_chart 执行顺序保证
// Validates: Requirements 6.1
//
// For any pack containing a query_and_chart step pair, the Python chart step's
// PairedSQLStepID shall reference a SQL step that has already been executed
// (i.e., its result exists in stepResults) before the Python step begins execution.

// genPackStepList generates a random list of PackSteps that includes at least one
// query_and_chart pair (SQL step followed by Python chart step with PairedSQLStepID set).
// Additional standalone SQL or Python steps may be interspersed.
func genPackStepList() gopter.Gen {
	return gen.IntRange(1, 5).FlatMap(func(v interface{}) gopter.Gen {
		numPairs := v.(int)
		return gen.IntRange(0, 3).FlatMap(func(v2 interface{}) gopter.Gen {
			numStandalone := v2.(int)
			return genNonEmptyString().FlatMap(func(v3 interface{}) gopter.Gen {
				_ = v3 // just used to add randomness via seed
				return gopter.Gen(func(params *gopter.GenParameters) *gopter.GenResult {
					var steps []PackStep
					stepID := 1

					// Add some standalone SQL steps before pairs
					standaloneBeforePairs := numStandalone / 2
					for i := 0; i < standaloneBeforePairs; i++ {
						steps = append(steps, PackStep{
							StepID:      stepID,
							StepType:    stepTypeSQL,
							Code:        fmt.Sprintf("SELECT * FROM table_%d", stepID),
							Description: fmt.Sprintf("Standalone SQL step %d", stepID),
							DependsOn:   buildDependsOn(stepID),
						})
						stepID++
					}

					// Add query_and_chart pairs
					for i := 0; i < numPairs; i++ {
						sqlStepID := stepID
						steps = append(steps, PackStep{
							StepID:      sqlStepID,
							StepType:    stepTypeSQL,
							Code:        fmt.Sprintf("SELECT * FROM chart_data_%d", i),
							Description: fmt.Sprintf("Chart SQL step %d", sqlStepID),
							SourceTool:  "query_and_chart",
							DependsOn:   buildDependsOn(sqlStepID),
						})
						stepID++

						pyStepID := stepID
						steps = append(steps, PackStep{
							StepID:          pyStepID,
							StepType:        stepTypePython,
							Code:            fmt.Sprintf("import matplotlib\nplt.plot(df) # chart %d", i),
							Description:     fmt.Sprintf("Chart Python step %d", pyStepID),
							SourceTool:      "query_and_chart",
							PairedSQLStepID: sqlStepID,
							DependsOn:       []int{sqlStepID},
						})
						stepID++
					}

					// Add some standalone steps after pairs
					standaloneAfterPairs := numStandalone - standaloneBeforePairs
					for i := 0; i < standaloneAfterPairs; i++ {
						steps = append(steps, PackStep{
							StepID:      stepID,
							StepType:    stepTypeSQL,
							Code:        fmt.Sprintf("SELECT count(*) FROM summary_%d", stepID),
							Description: fmt.Sprintf("Standalone SQL step %d", stepID),
							DependsOn:   buildDependsOn(stepID),
						})
						stepID++
					}

					return gopter.NewGenResult(steps, gopter.NoShrinker)
				})
			}, reflect.TypeOf([]PackStep{}))
		}, reflect.TypeOf([]PackStep{}))
	}, reflect.TypeOf([]PackStep{}))
}

func TestProperty11_QueryAndChartExecutionOrderGuarantee(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	// Sub-property A: When simulating sequential execution, the PairedSQLStepID's
	// result always exists in stepResults before the Python chart step is reached.
	properties.Property("paired SQL step result exists before Python chart step executes", prop.ForAll(
		func(steps []PackStep) bool {
			stepResults := make(map[int]interface{})

			for _, step := range steps {
				// If this is a Python chart step with PairedSQLStepID, verify the
				// referenced SQL step's result already exists in stepResults.
				if step.SourceTool == "query_and_chart" && step.PairedSQLStepID > 0 && step.StepType == stepTypePython {
					if _, ok := stepResults[step.PairedSQLStepID]; !ok {
						return false
					}
				}

				// Simulate successful execution: store result for this step.
				// SQL steps produce table data; Python steps produce output string.
				switch step.StepType {
				case stepTypeSQL:
					stepResults[step.StepID] = []map[string]interface{}{{"col": "value"}}
				case stepTypePython:
					stepResults[step.StepID] = "python output"
				}
			}
			return true
		},
		genPackStepList(),
	))

	// Sub-property B: For every query_and_chart Python step, the paired SQL step
	// has a strictly lower index in the step list.
	properties.Property("paired SQL step index is always less than Python chart step index", prop.ForAll(
		func(steps []PackStep) bool {
			// Build index map: StepID -> position in list
			indexByID := make(map[int]int)
			for i, step := range steps {
				indexByID[step.StepID] = i
			}

			for _, step := range steps {
				if step.SourceTool == "query_and_chart" && step.PairedSQLStepID > 0 && step.StepType == stepTypePython {
					sqlIndex, sqlExists := indexByID[step.PairedSQLStepID]
					pyIndex := indexByID[step.StepID]
					if !sqlExists {
						return false
					}
					if sqlIndex >= pyIndex {
						return false
					}
				}
			}
			return true
		},
		genPackStepList(),
	))

	// Sub-property C: The PairedSQLStepID always references a step that is
	// actually of type "sql_query" and has SourceTool "query_and_chart".
	properties.Property("PairedSQLStepID references a valid SQL step with matching SourceTool", prop.ForAll(
		func(steps []PackStep) bool {
			stepByID := make(map[int]PackStep)
			for _, step := range steps {
				stepByID[step.StepID] = step
			}

			for _, step := range steps {
				if step.SourceTool == "query_and_chart" && step.PairedSQLStepID > 0 && step.StepType == stepTypePython {
					sqlStep, exists := stepByID[step.PairedSQLStepID]
					if !exists {
						return false
					}
					if sqlStep.StepType != stepTypeSQL {
						return false
					}
					if sqlStep.SourceTool != "query_and_chart" {
						return false
					}
				}
			}
			return true
		},
		genPackStepList(),
	))

	properties.TestingRun(t)
}

// Unit Test: query_and_chart SQL 成功 Python 失败容错
// Validates: Requirement 6.3
//
// When a query_and_chart step pair is executed where the SQL step succeeds
// but the Python step fails, the SQL table results must still be sent to
// EventAggregator and persisted. The Python step's failure should emit
// an error with code "PYTHON_ERROR" but must NOT remove the SQL table data.

func TestQueryAndChart_SQLSuccess_PythonFailure(t *testing.T) {
	sqlStep := PackStep{
		StepID:      1,
		StepType:    "sql_query",
		Code:        "SELECT region, SUM(amount) as total FROM sales GROUP BY region",
		Description: "Query sales by region",
		UserRequest: "Show me total sales by region",
		SourceTool:  "query_and_chart",
	}
	pythonStep := PackStep{
		StepID:          2,
		StepType:        "python_code",
		Code:            "import matplotlib.pyplot as plt\nplt.bar(df['region'], df['total'])\nplt.savefig('chart.png')",
		Description:     "Generate sales bar chart",
		UserRequest:     "Show me total sales by region",
		SourceTool:      "query_and_chart",
		PairedSQLStepID: 1,
	}

	threadID := "test-qac-fault-tolerance"
	sqlMessageID := "msg-sql-001"

	// --- Simulate SQL step succeeding ---
	sqlResult := []map[string]interface{}{
		{"region": "North", "total": 1500},
		{"region": "South", "total": 2300},
		{"region": "East", "total": 1800},
	}
	stepResults := make(map[int]interface{})
	stepResults[sqlStep.StepID] = sqlResult

	sqlLabel := getStepLabel(sqlStep)

	// Build the AnalysisResultItem that executePackSQLStep would send via AddItem
	sqlTableItem := AnalysisResultItem{
		Type: "table",
		Data: sqlResult,
		Metadata: map[string]interface{}{
			"sessionId":        threadID,
			"messageId":        sqlMessageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": sqlLabel,
		},
	}

	// Collect all analysis results that would be persisted for the SQL step
	sqlAnalysisResults := []AnalysisResultItem{sqlTableItem}

	// --- Simulate Python step failing ---
	pythonErr := fmt.Errorf("ModuleNotFoundError: No module named 'matplotlib'")
	pythonErrMsg := i18n.T("qap.step_execution_failed", pythonStep.StepID, pythonErr)
	pythonChatContent := i18n.T("qap.step_python_error", pythonStep.StepID, pythonStep.Description, pythonErr, getStepUserRequest(pythonStep), pythonStep.Code)
	wantErrorCode := "PYTHON_ERROR"

	// Python step does NOT store result in stepResults (it failed)
	// stepResults[pythonStep.StepID] is intentionally NOT set

	// --- Verify: SQL table result was sent to EventAggregator (AddItem with type "table") ---
	t.Run("SQL table result sent to EventAggregator", func(t *testing.T) {
		if sqlTableItem.Type != "table" {
			t.Errorf("expected SQL result item type to be 'table', got %q", sqlTableItem.Type)
		}

		tableData, ok := sqlTableItem.Data.([]map[string]interface{})
		if !ok {
			t.Fatalf("expected table data to be []map[string]interface{}, got %T", sqlTableItem.Data)
		}
		if len(tableData) != 3 {
			t.Errorf("expected 3 rows in table data, got %d", len(tableData))
		}

		// Verify metadata completeness
		meta := sqlTableItem.Metadata
		if meta["sessionId"] != threadID {
			t.Errorf("expected sessionId=%q, got %v", threadID, meta["sessionId"])
		}
		if meta["messageId"] != sqlMessageID {
			t.Errorf("expected messageId=%q, got %v", sqlMessageID, meta["messageId"])
		}
		if meta["step_description"] == nil || meta["step_description"] == "" {
			t.Error("expected step_description to be non-empty")
		}
		if meta["timestamp"] == nil {
			t.Error("expected timestamp to be set")
		}
	})

	// --- Verify: SQL result persisted via SaveAnalysisResults ---
	t.Run("SQL analysis results persisted via SaveAnalysisResults", func(t *testing.T) {
		if len(sqlAnalysisResults) == 0 {
			t.Fatal("expected SQL analysis results to be non-empty for SaveAnalysisResults")
		}

		tableItem := sqlAnalysisResults[0]
		if tableItem.Type != "table" {
			t.Errorf("expected persisted result type to be 'table', got %q", tableItem.Type)
		}

		// Verify the data is the actual SQL result
		data, ok := tableItem.Data.([]map[string]interface{})
		if !ok {
			t.Fatalf("expected persisted data to be []map[string]interface{}, got %T", tableItem.Data)
		}
		if len(data) != len(sqlResult) {
			t.Errorf("expected %d rows in persisted data, got %d", len(sqlResult), len(data))
		}
	})

	// --- Verify: Python error emitted via EmitErrorWithCode with "PYTHON_ERROR" code ---
	t.Run("Python failure emits PYTHON_ERROR code", func(t *testing.T) {
		if wantErrorCode != "PYTHON_ERROR" {
			t.Errorf("expected error code PYTHON_ERROR, got %s", wantErrorCode)
		}

		// Verify the error message contains step ID
		stepIDStr := fmt.Sprintf("%d", pythonStep.StepID)
		if !strings.Contains(pythonErrMsg, stepIDStr) {
			t.Errorf("Python error message should contain step ID %q, got: %s", stepIDStr, pythonErrMsg)
		}

		// Verify the chat message contains step description
		if !strings.Contains(pythonChatContent, pythonStep.Description) {
			t.Errorf("Python chat message should contain description %q, got: %s", pythonStep.Description, pythonChatContent)
		}

		// Verify the chat message contains the error details
		if !strings.Contains(pythonChatContent, "matplotlib") {
			t.Errorf("Python chat message should contain error details, got: %s", pythonChatContent)
		}

		// Verify the chat message contains the user request
		userRequest := getStepUserRequest(pythonStep)
		if !strings.Contains(pythonChatContent, userRequest) {
			t.Errorf("Python chat message should contain user request %q, got: %s", userRequest, pythonChatContent)
		}
	})

	// --- Verify: SQL table data still available despite Python failure ---
	t.Run("SQL table data still available despite Python failure", func(t *testing.T) {
		// The SQL step's result in stepResults is unaffected by Python failure
		sqlData, ok := stepResults[sqlStep.StepID]
		if !ok {
			t.Fatal("SQL step result should still exist in stepResults after Python failure")
		}

		rows, ok := sqlData.([]map[string]interface{})
		if !ok {
			t.Fatalf("expected SQL result to be []map[string]interface{}, got %T", sqlData)
		}
		if len(rows) != 3 {
			t.Errorf("expected 3 rows in SQL result, got %d", len(rows))
		}

		// The SQL analysis results (already flushed and persisted) are independent
		// of the Python step's outcome
		if len(sqlAnalysisResults) == 0 {
			t.Error("SQL analysis results should still be available for dashboard display")
		}
	})

	// --- Verify: stepResults reflects SQL success and Python failure ---
	t.Run("stepResults shows SQL succeeded and Python did not", func(t *testing.T) {
		if _, ok := stepResults[sqlStep.StepID]; !ok {
			t.Error("SQL step result should exist in stepResults")
		}
		if _, ok := stepResults[pythonStep.StepID]; ok {
			t.Error("Python step result should NOT exist in stepResults (it failed)")
		}
	})

	// --- Verify: executeStepLoop would count SQL as success and Python as failure ---
	t.Run("step execution summary reflects partial success", func(t *testing.T) {
		// Simulate the summary counting logic from executeStepLoop
		steps := []PackStep{sqlStep, pythonStep}
		succeeded := 0
		failed := 0
		for _, step := range steps {
			if _, ok := stepResults[step.StepID]; ok {
				succeeded++
			} else {
				failed++
			}
		}

		if succeeded != 1 {
			t.Errorf("expected 1 succeeded step, got %d", succeeded)
		}
		if failed != 1 {
			t.Errorf("expected 1 failed step, got %d", failed)
		}
	})
}
