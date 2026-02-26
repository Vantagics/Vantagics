package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"vantagics/i18n"
)

// ---------------------------------------------------------------------------
// Step label helpers
// ---------------------------------------------------------------------------

// getStepLabel returns a human-readable label for a pack step.
// If the step has a non-empty Description, it is returned as-is.
// Otherwise, a default label is generated based on StepType and StepID.
func getStepLabel(step PackStep) string {
	if step.Description != "" {
		return step.Description
	}
	switch step.StepType {
	case stepTypeSQL:
		return i18n.T("qap.step_sql_query", step.StepID)
	case stepTypePython:
		return i18n.T("qap.step_python_script", step.StepID)
	default:
		return i18n.T("qap.step_generic", step.StepID)
	}
}

// getStepUserRequest returns the original user request for display as "分析请求".
// Prefers UserRequest (the original user question), falls back to Description, then default label.
func getStepUserRequest(step PackStep) string {
	if step.UserRequest != "" {
		return step.UserRequest
	}
	return getStepLabel(step)
}

// ---------------------------------------------------------------------------
// SQL step execution
// ---------------------------------------------------------------------------

// executePackSQLStep executes a single SQL step from the pack.
// On success, the result is stored in stepResults and sent to the EventAggregator.
// On failure, an error message is added to the chat and emitted, but execution does not stop.
func (a *App) executePackSQLStep(threadID string, messageID string, step PackStep, stepResults map[int]interface{}, dataSourceID string) {
	stepLabel := getStepLabel(step)
	userRequest := getStepUserRequest(step)

	a.Log(fmt.Sprintf("%s SQL step %d code:\n%s", logTagExecute, step.StepID, step.Code))
	result, err := a.dataSourceService.ExecuteSQL(dataSourceID, step.Code)
	if err != nil {
		errMsg := i18n.T("qap.step_sql_error", step.StepID, step.Description, err, userRequest, step.Code)
		a.Log(fmt.Sprintf("%s SQL Error: %s", logTagExecute, errMsg))

		errorChatMsg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   errMsg,
			Timestamp: time.Now().Unix(),
		}
		a.chatService.AddMessage(threadID, errorChatMsg)

		if a.eventAggregator != nil {
			a.eventAggregator.EmitErrorWithCode(threadID, "", "SQL_ERROR", errMsg)
		}
		return
	}

	stepResults[step.StepID] = result
	a.Log(fmt.Sprintf("%s SQL step %d executed successfully, %d rows", logTagExecute, step.StepID, len(result)))

	// Log column info and null value statistics for debugging
	a.logSQLResultDiagnostics(step.StepID, result)

	// Add success message with results (truncate large result sets for chat display)
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	chatContent := i18n.T("qap.step_sql_success_full", step.StepID, step.Description, len(result), userRequest, string(resultJSON))
	if len(chatContent) > 50000 {
		previewJSON, _ := json.MarshalIndent(result[:min(len(result), 20)], "", "  ")
		chatContent = i18n.T("qap.step_sql_success_truncated", step.StepID, step.Description, len(result), userRequest, string(previewJSON))
	}
	successMsg := ChatMessage{
		ID:        messageID,
		Role:      "assistant",
		Content:   chatContent,
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(threadID, successMsg)

	// Send table data to EventAggregator for dashboard display
	if a.eventAggregator != nil {
		var analysisResults []AnalysisResultItem

		// Add table result to EventAggregator
		a.eventAggregator.AddItem(threadID, messageID, "", "table", result, map[string]interface{}{
			"sessionId":        threadID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepLabel,
		})
		
		// Store table result for persistence
		analysisResults = append(analysisResults, AnalysisResultItem{
			Type: "table",
			Data: result,
			Metadata: map[string]interface{}{
				"sessionId":        threadID,
				"messageId":        messageID,
				"timestamp":        time.Now().UnixMilli(),
				"step_description": stepLabel,
			},
		})

		// Add ECharts configs from step (must be after table for correct ordering)
		echartsResults := a.emitStepEChartsConfigs(threadID, messageID, step)
		analysisResults = append(analysisResults, echartsResults...)

		// Flush results to dashboard
		a.eventAggregator.FlushNow(threadID, false)

		// Persist analysis results to database
		if err := a.chatService.SaveAnalysisResults(threadID, messageID, analysisResults); err != nil {
			a.Log(fmt.Sprintf("%s Failed to save SQL analysis results for step %d: %v", logTagExecute, step.StepID, err))
		}
	}
}

// logSQLResultDiagnostics logs column info and null value statistics for debugging.
func (a *App) logSQLResultDiagnostics(stepID int, result []map[string]interface{}) {
	if len(result) == 0 {
		return
	}

	cols := make([]string, 0, len(result[0]))
	for col := range result[0] {
		cols = append(cols, col)
	}
	a.Log(fmt.Sprintf("%s SQL step %d columns (%d): %s", logTagExecute, stepID, len(cols), strings.Join(cols, ", ")))

	nullCounts := make(map[string]int)
	for _, row := range result {
		for col, val := range row {
			if val == nil {
				nullCounts[col]++
			}
		}
	}
	if len(nullCounts) > 0 {
		var nullInfo []string
		var allNullCols []string
		for col, count := range nullCounts {
			nullInfo = append(nullInfo, fmt.Sprintf("%s=%d/%d", col, count, len(result)))
			if count == len(result) {
				allNullCols = append(allNullCols, col)
			}
		}
		a.Log(fmt.Sprintf("%s SQL step %d null values: %s", logTagExecute, stepID, strings.Join(nullInfo, ", ")))
		if len(allNullCols) > 0 {
			a.Log(fmt.Sprintf("%s WARNING: SQL step %d has %d columns with ALL null values: %s. This may indicate a data import type inference issue. Consider re-importing the data source.",
				logTagExecute, stepID, len(allNullCols), strings.Join(allNullCols, ", ")))
		}
	}
}

// ---------------------------------------------------------------------------
// Python step execution
// ---------------------------------------------------------------------------

// executePackPythonStep executes a single Python step from the pack.
// It tries the EinoService Python pool first (same as normal analysis), then falls back to auto-detected Python.
// On failure, an error message is logged and emitted.
func (a *App) executePackPythonStep(threadID string, messageID string, step PackStep, stepResults map[int]interface{}, workDir string, existingFiles map[string]bool) {
	stepLabel := getStepLabel(step)
	userRequest := getStepUserRequest(step)

	// Substitute dependency placeholders ${STEP_N_RESULT} with actual results
	code := step.Code
	for _, depID := range step.DependsOn {
		depResult := stepResults[depID]
		placeholder := fmt.Sprintf("${STEP_%d_RESULT}", depID)
		resultJSON, _ := json.Marshal(depResult)
		code = strings.ReplaceAll(code, placeholder, string(resultJSON))
	}

	// If this is a chart step from query_and_chart, inject SQL results as DataFrame 'df'
	if step.SourceTool == "query_and_chart" && step.PairedSQLStepID > 0 {
		if sqlResult, ok := stepResults[step.PairedSQLStepID]; ok {
			code = a.buildDataFrameInjection(sqlResult, code)
			a.Log(fmt.Sprintf("%s Injected DataFrame from SQL step %d for chart code", logTagExecute, step.PairedSQLStepID))
		}
	}

	var output string
	var execErr error

	// Strategy 1: Use EinoService's Python pool (same path as normal analysis sessions)
	if a.einoService != nil && a.einoService.HasPython() {
		a.Log(fmt.Sprintf("%s Executing Python via EinoService pool", logTagExecute))
		output, execErr = a.einoService.ExecutePython(code, workDir)
	} else {
		// Strategy 2: Find Python path via config or auto-detection
		pythonPath := a.findPythonPath()
		if pythonPath == "" {
			errMsg := i18n.T("qap.step_python_not_configured", step.StepID)
			a.Log(fmt.Sprintf("%s Python Error: %s", logTagExecute, errMsg))

			errorChatMsg := ChatMessage{
				ID:        messageID,
				Role:      "assistant",
				Content:   i18n.T("qap.step_python_no_env", step.StepID, step.Description, userRequest, step.Code),
				Timestamp: time.Now().Unix(),
			}
			a.chatService.AddMessage(threadID, errorChatMsg)

			if a.eventAggregator != nil {
				a.eventAggregator.EmitErrorWithCode(threadID, "", "PYTHON_NOT_CONFIGURED", errMsg)
			}
			return
		}

		a.Log(fmt.Sprintf("%s Executing Python via PythonService with path: %s", logTagExecute, pythonPath))
		ps := a.pythonService
		output, execErr = ps.ExecuteScript(pythonPath, code)
	}

	if execErr != nil {
		errMsg := i18n.T("qap.step_execution_failed", step.StepID, execErr)
		a.Log(fmt.Sprintf("%s Python Error: %s", logTagExecute, errMsg))

		errorChatMsg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   i18n.T("qap.step_python_error", step.StepID, step.Description, execErr, userRequest, step.Code),
			Timestamp: time.Now().Unix(),
		}
		a.chatService.AddMessage(threadID, errorChatMsg)

		if a.eventAggregator != nil {
			a.eventAggregator.EmitErrorWithCode(threadID, "", "PYTHON_ERROR", errMsg)
		}
		return
	}

	stepResults[step.StepID] = output
	a.Log(fmt.Sprintf("%s Python step %d executed successfully", logTagExecute, step.StepID))

	successMsg := ChatMessage{
		ID:        messageID,
		Role:      "assistant",
		Content:   i18n.T("qap.step_python_success", step.StepID, step.Description, userRequest, formatPythonOutputForMessage(output)),
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(threadID, successMsg)

	// Collect analysis results from Python output and chart files
	if a.eventAggregator != nil {
		var analysisResults []AnalysisResultItem

		// Detect and send image files (must check existingFiles to avoid duplicates)
		imageResults := a.detectAndSendPythonChartFiles(threadID, messageID, workDir, stepLabel, existingFiles)
		analysisResults = append(analysisResults, imageResults...)

		// Detect and send ECharts from Python stdout
		echartsResults := a.detectAndSendPythonECharts(threadID, messageID, output, stepLabel)
		analysisResults = append(analysisResults, echartsResults...)

		// Send stored ECharts configs from step metadata
		storedEchartsResults := a.emitStepEChartsConfigs(threadID, messageID, step)
		analysisResults = append(analysisResults, storedEchartsResults...)

		// Flush results to dashboard if any were collected
		if len(analysisResults) > 0 {
			a.eventAggregator.FlushNow(threadID, false)
			
			// Persist analysis results to database
			if err := a.chatService.SaveAnalysisResults(threadID, messageID, analysisResults); err != nil {
				a.Log(fmt.Sprintf("%s Failed to save analysis results for step %d: %v", logTagExecute, step.StepID, err))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Result detection helpers
// ---------------------------------------------------------------------------

// detectAndSendPythonChartFiles scans the workDir for newly generated image files
// and sends them to the EventAggregator for dashboard display.
// The existingFiles map is used to filter out files that existed before this step,
// ensuring only newly generated charts are sent.
func (a *App) detectAndSendPythonChartFiles(threadID, messageID, workDir string, stepDescription string, existingFiles map[string]bool) []AnalysisResultItem {
	entries, err := os.ReadDir(workDir)
	if err != nil {
		a.Log(fmt.Sprintf("%s Failed to read workDir %s: %v", logTagExecute, workDir, err))
		return nil
	}

	imageExtensions := map[string]bool{".png": true, ".jpg": true, ".jpeg": true}
	var results []AnalysisResultItem

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !imageExtensions[ext] {
			continue
		}

		// Skip files that existed before this step executed
		if existingFiles != nil && existingFiles[entry.Name()] {
			a.Log(fmt.Sprintf("%s Skipping existing file: %s", logTagExecute, entry.Name()))
			continue
		}

		filePath := filepath.Join(workDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			a.Log(fmt.Sprintf("%s Failed to stat chart file %s: %v", logTagExecute, entry.Name(), err))
			continue
		}
		// Skip files larger than 10MB to prevent memory exhaustion
		const maxChartFileSize = 10 * 1024 * 1024
		if info.Size() > maxChartFileSize {
			a.Log(fmt.Sprintf("%s Skipping oversized chart file %s (%d bytes)", logTagExecute, entry.Name(), info.Size()))
			continue
		}
		imageData, err := os.ReadFile(filePath)
		if err != nil {
			a.Log(fmt.Sprintf("%s Failed to read chart file %s: %v", logTagExecute, entry.Name(), err))
			continue
		}

		base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)
		
		// Add to EventAggregator
		a.eventAggregator.AddItem(threadID, messageID, "", "image", base64Data, map[string]interface{}{
			"sessionId":        threadID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"fileName":         entry.Name(),
			"step_description": stepDescription,
		})
		
		// Store for persistence
		results = append(results, AnalysisResultItem{
			Type: "image",
			Data: base64Data,
			Metadata: map[string]interface{}{
				"sessionId":        threadID,
				"messageId":        messageID,
				"timestamp":        time.Now().UnixMilli(),
				"fileName":         entry.Name(),
				"step_description": stepDescription,
			},
		})
		a.Log(fmt.Sprintf("%s Sent chart image to dashboard: %s", logTagExecute, entry.Name()))
	}

	return results
}


// formatPythonOutputForMessage formats Python execution output for the assistant
// chat message. If the output contains chart references (json:echarts blocks or
// inline images), it is included as-is so the frontend can render them properly.
// Plain text output is wrapped in a code block for readability. This ensures
// chart reference formatting is consistent with the original analysis flow.
func formatPythonOutputForMessage(output string) string {
	if output == "" {
		return ""
	}
	hasECharts := strings.Contains(output, "json:echarts")
	hasImage := strings.Contains(output, "![Chart](") || strings.Contains(output, "![chart](") || strings.Contains(output, "data:image/")
	if hasECharts || hasImage {
		return output
	}
	return "```\n" + output + "\n```"
}

// detectAndSendPythonECharts parses Python stdout for ECharts JSON blocks
// marked with ```json:echarts ... ``` or bare json:echarts markers,
// sends them via EventAggregator, and returns analysis result items.
// Invalid JSON configs are skipped with a warning log.
func (a *App) detectAndSendPythonECharts(threadID, messageID, output string, stepDescription string) []AnalysisResultItem {
	if output == "" {
		return nil
	}

	var results []AnalysisResultItem

	// Pattern 1: Backtick-wrapped ECharts blocks
	reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	matches := reECharts.FindAllStringSubmatch(output, -1)

	// Pattern 2: Bare json:echarts markers (no backticks)
	reEChartsNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:echarts\\s*\\n(\\{[\\s\\S]+?\\n\\})(?:\\s*\\n(?:---|###)|\\s*$)")
	matches = append(matches, reEChartsNoBT.FindAllStringSubmatch(output, -1)...)

	for i, match := range matches {
		if len(match) < 2 {
			continue
		}
		chartData := strings.TrimSpace(match[1])
		if chartData == "" {
			continue
		}

		// Skip oversized ECharts JSON configs (max 1MB) to prevent DoS
		const maxEChartsSize = 1024 * 1024
		if len(chartData) > maxEChartsSize {
			a.Log(fmt.Sprintf("%s Skipping oversized ECharts JSON in Python output (match #%d, %d bytes)", logTagExecute, i+1, len(chartData)))
			continue
		}

		// Validate JSON before sending
		if !json.Valid([]byte(chartData)) {
			a.Log(fmt.Sprintf("%s Skipping invalid ECharts JSON in Python output (match #%d)", logTagExecute, i+1))
			continue
		}

		// Add to EventAggregator
		a.eventAggregator.AddItem(threadID, messageID, "", "echarts", chartData, map[string]interface{}{
			"sessionId":        threadID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepDescription,
		})
		
		// Store for persistence
		results = append(results, AnalysisResultItem{
			Type: "echarts",
			Data: chartData,
			Metadata: map[string]interface{}{
				"sessionId":        threadID,
				"messageId":        messageID,
				"timestamp":        time.Now().UnixMilli(),
				"step_description": stepDescription,
			},
		})
		a.Log(fmt.Sprintf("%s Sent ECharts config to dashboard from Python output (match #%d)", logTagExecute, i+1))
	}

	return results
}

// buildDataFrameInjection wraps chart code with DataFrame loading from SQL results,
// similar to how query_and_chart tool's buildChartPythonCode works during normal analysis.
func (a *App) buildDataFrameInjection(sqlResult interface{}, chartCode string) string {
	if sqlResult == nil {
		a.Log(fmt.Sprintf("%s SQL result is nil, skipping DataFrame injection", logTagExecute))
		return chartCode
	}

	resultJSON, err := json.Marshal(sqlResult)
	if err != nil {
		a.Log(fmt.Sprintf("%s Failed to marshal SQL result for DataFrame injection: %v", logTagExecute, err))
		return chartCode
	}

	encoded := base64.StdEncoding.EncodeToString(resultJSON)

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


// emitStepEChartsConfigs sends stored ECharts configs from a step to the EventAggregator.
// These configs were extracted from the original LLM response during export.
func (a *App) emitStepEChartsConfigs(threadID, messageID string, step PackStep) []AnalysisResultItem {
	if len(step.EChartsConfigs) == 0 || a.eventAggregator == nil {
		return nil
	}

	stepLabel := getStepLabel(step)
	var results []AnalysisResultItem
	for i, chartJSON := range step.EChartsConfigs {
		if !json.Valid([]byte(chartJSON)) {
			a.Log(fmt.Sprintf("%s Skipping invalid stored ECharts config #%d for step %d", logTagExecute, i+1, step.StepID))
			continue
		}
		a.eventAggregator.AddItem(threadID, messageID, "", "echarts", chartJSON, map[string]interface{}{
			"sessionId":        threadID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepLabel,
		})
		results = append(results, AnalysisResultItem{
			Type: "echarts",
			Data: chartJSON,
			Metadata: map[string]interface{}{
				"sessionId":        threadID,
				"messageId":        messageID,
				"timestamp":        time.Now().UnixMilli(),
				"step_description": stepLabel,
			},
		})
		a.Log(fmt.Sprintf("%s Sent stored ECharts config #%d to dashboard for step %d", logTagExecute, i+1, step.StepID))
	}
	return results
}
