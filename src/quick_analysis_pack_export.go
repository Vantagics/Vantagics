package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"vantagedata/i18n"
)

// ExportQuickAnalysisPack exports a quick analysis pack from a chat session.
// It extracts all SQL/Python steps from the session trajectory, collects the full
// data source schema, and packages everything into a .qap ZIP file.
// The file is automatically saved to {DataCacheDir}/qap/ directory.
//
// Parameters:
//   - threadID: the chat thread/session ID
//   - packName: analysis scenario name (user input)
//   - author: creator name (user input)
//   - password: optional encryption password (empty string = no encryption)
//
// Returns the saved file path and any error.
func (a *App) ExportQuickAnalysisPack(threadID string, packName string, author string, password string) (string, error) {
	a.Log(fmt.Sprintf("[QAP-EXPORT] Starting export for thread: %s, packName: %s, author: %s", threadID, packName, author))

	// 1. Load the thread to get data source ID
	thread, err := a.chatService.LoadThread(threadID)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Error loading thread: %v", err))
		return "", fmt.Errorf("failed to load thread: %w", err)
	}

	// 2. Extract all SQL/Python steps from trajectory files
	steps, err := a.collectSessionSteps(threadID)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Error collecting steps: %v", err))
		return "", err
	}

	if len(steps) == 0 {
		a.Log("[QAP-EXPORT] No executable steps found")
		return "", fmt.Errorf("%s", i18n.T("qap.no_exportable_operations"))
	}

	a.Log(fmt.Sprintf("[QAP-EXPORT] Collected %d steps", len(steps)))

	// Log step type breakdown
	sqlCount, pyCount := 0, 0
	for _, s := range steps {
		switch s.StepType {
		case stepTypeSQL:
			sqlCount++
		case stepTypePython:
			pyCount++
		}
	}
	a.Log(fmt.Sprintf("[QAP-EXPORT] Step breakdown: %d SQL, %d Python", sqlCount, pyCount))

	// 3. Build and save the pack (shared logic handles ECharts, schema, metadata, ZIP)
	return a.buildAndSavePack(thread, steps, packName, author, password)
}


// collectSessionSteps extracts all SQL and Python steps from a session.
// It uses executions.json for SQL steps (most reliable — only records successful executions),
// and supplements with Python/chart code from trajectory files (with error filtering).
func (a *App) collectSessionSteps(threadID string) ([]PackStep, error) {
	// Strategy 1: Get SQL steps from executions.json (only successful executions)
	sqlSteps := a.collectStepsFromExecutions(threadID)
	a.Log(fmt.Sprintf("[QAP-EXPORT] Found %d steps from executions.json", len(sqlSteps)))

	// Strategy 2: Get Python/chart steps from trajectory (with error filtering)
	pythonSteps, _ := a.collectPythonStepsFromTrajectory(threadID)
	a.Log(fmt.Sprintf("[QAP-EXPORT] Found %d Python/chart steps from trajectory", len(pythonSteps)))

	// Merge: SQL steps from executions.json + Python steps from trajectory
	if len(sqlSteps) > 0 || len(pythonSteps) > 0 {
		// Deduplicate Python steps that might appear in both sources
		seenPythonCode := make(map[string]bool)
		for _, s := range sqlSteps {
			if s.StepType == "python_code" {
				seenPythonCode[s.Code] = true
			}
		}
		var dedupedPythonSteps []PackStep
		for _, s := range pythonSteps {
			if !seenPythonCode[s.Code] {
				dedupedPythonSteps = append(dedupedPythonSteps, s)
				seenPythonCode[s.Code] = true
			}
		}

		allSteps := append(sqlSteps, dedupedPythonSteps...)
		// Re-number step IDs and rebuild dependencies
		renumberSteps(allSteps)
		a.Log(fmt.Sprintf("[QAP-EXPORT] Total merged steps: %d", len(allSteps)))
		return allSteps, nil
	}

	// Strategy 3: Fall back to full trajectory parsing if both above yield nothing
	a.Log("[QAP-EXPORT] No steps from executions.json or trajectory Python, trying full trajectory")
	return a.collectStepsFromTrajectory(threadID)
}

// collectPythonStepsFromTrajectory extracts only Python/chart steps from trajectory files,
// filtering out failed tool calls.
func (a *App) collectPythonStepsFromTrajectory(threadID string) ([]PackStep, error) {
	cfg, _ := a.GetConfig()
	trajectoryDir := filepath.Join(cfg.DataCacheDir, "sessions", threadID, "trajectory")

	files, err := os.ReadDir(trajectoryDir)
	if err != nil {
		return nil, nil
	}

	var jsonFiles []os.DirEntry
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			jsonFiles = append(jsonFiles, file)
		}
	}
	sort.Slice(jsonFiles, func(i, j int) bool {
		nameI := strings.TrimSuffix(jsonFiles[i].Name(), ".json")
		nameJ := strings.TrimSuffix(jsonFiles[j].Name(), ".json")
		tsI, errI := strconv.ParseInt(nameI, 10, 64)
		tsJ, errJ := strconv.ParseInt(nameJ, 10, 64)
		if errI != nil || errJ != nil {
			return jsonFiles[i].Name() < jsonFiles[j].Name()
		}
		return tsI < tsJ
	})

	var steps []PackStep
	stepID := 1
	seenCode := make(map[string]bool) // Deduplicate identical Python code

	for _, file := range jsonFiles {
		data, err := os.ReadFile(filepath.Join(trajectoryDir, file.Name()))
		if err != nil {
			continue
		}

		var trajectory struct {
			Steps []struct {
				Type       string `json:"type"`
				ToolName   string `json:"tool_name"`
				ToolInput  string `json:"tool_input"`
				ToolOutput string `json:"tool_output"`
				Error      string `json:"error"`
			} `json:"steps"`
		}
		if err := json.Unmarshal(data, &trajectory); err != nil {
			continue
		}

		for _, step := range trajectory.Steps {
			if step.Type != "tool_call" {
				continue
			}
			if step.Error != "" || isToolOutputFailed(step.ToolOutput) {
				continue
			}

			// Only extract Python/chart code (SQL comes from executions.json)
			if step.ToolName != "execute_python" && step.ToolName != "query_and_chart" {
				continue
			}

			var args map[string]interface{}
			if step.ToolInput == "" {
				continue
			}
			if err := json.Unmarshal([]byte(step.ToolInput), &args); err != nil {
				unescaped := unescapeToolInput(step.ToolInput)
				if err2 := json.Unmarshal([]byte(unescaped), &args); err2 != nil {
					continue
				}
			}

			switch step.ToolName {
			case "execute_python":
				code, _ := args["code"].(string)
				if code != "" {
					if seenCode["python:"+code] {
						continue
					}
					seenCode["python:"+code] = true
					desc, _ := args["description"].(string)
					if desc == "" {
						desc = "Python Code"
					}
					steps = append(steps, PackStep{
						StepID:      stepID,
						StepType:    stepTypePython,
						Code:        code,
						Description: desc,
					})
					stepID++
				}

			case "query_and_chart":
				// Only extract chart_code (SQL is from executions.json)
				chartCode, _ := args["chart_code"].(string)
				chartTitle, _ := args["chart_title"].(string)
				if chartCode != "" {
					if seenCode["chart:"+chartCode] {
						continue
					}
					seenCode["chart:"+chartCode] = true
					desc := chartTitle
					if desc == "" {
						desc = "Chart Generation"
					} else {
						desc = desc + " (Chart)"
					}
					steps = append(steps, PackStep{
						StepID:      stepID,
						StepType:    stepTypePython,
						Code:        chartCode,
						Description: desc,
						SourceTool:  "query_and_chart",
					})
					stepID++
				}
			}
		}
	}

	return steps, nil
}

// collectStepsFromTrajectory extracts steps from trajectory JSON files.
// It filters out failed tool calls by checking tool_output for error indicators.
func (a *App) collectStepsFromTrajectory(threadID string) ([]PackStep, error) {
	cfg, _ := a.GetConfig()
	trajectoryDir := filepath.Join(cfg.DataCacheDir, "sessions", threadID, "trajectory")

	a.Log(fmt.Sprintf("[QAP-EXPORT] Looking for trajectory files in: %s", trajectoryDir))
	a.Log(fmt.Sprintf("[QAP-EXPORT] DataCacheDir=%s, threadID=%s", cfg.DataCacheDir, threadID))

	files, err := os.ReadDir(trajectoryDir)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Trajectory directory not found: %v", err))
		return nil, fmt.Errorf("trajectory directory not found: %v", err)
	}

	// Collect and sort JSON files by name (timestamp-based) for chronological order
	var jsonFiles []os.DirEntry
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			jsonFiles = append(jsonFiles, file)
		}
	}

	sort.Slice(jsonFiles, func(i, j int) bool {
		nameI := strings.TrimSuffix(jsonFiles[i].Name(), ".json")
		nameJ := strings.TrimSuffix(jsonFiles[j].Name(), ".json")
		tsI, errI := strconv.ParseInt(nameI, 10, 64)
		tsJ, errJ := strconv.ParseInt(nameJ, 10, 64)
		if errI != nil || errJ != nil {
			return jsonFiles[i].Name() < jsonFiles[j].Name()
		}
		return tsI < tsJ
	})

	a.Log(fmt.Sprintf("[QAP-EXPORT] Found %d trajectory JSON files", len(jsonFiles)))

	var steps []PackStep
	stepID := 1

	for _, file := range jsonFiles {
		trajectoryPath := filepath.Join(trajectoryDir, file.Name())
		data, err := os.ReadFile(trajectoryPath)
		if err != nil {
			a.Log(fmt.Sprintf("[QAP-EXPORT] Error reading %s: %v", file.Name(), err))
			continue
		}

		a.Log(fmt.Sprintf("[QAP-EXPORT] Processing file %s (%d bytes)", file.Name(), len(data)))

		// Parse as AgentTrajectory object (the actual format saved by saveTrajectory)
		var trajectory struct {
			Steps []struct {
				Type       string `json:"type"`
				ToolName   string `json:"tool_name"`
				ToolInput  string `json:"tool_input"`
				ToolOutput string `json:"tool_output"`
				Error      string `json:"error"`
			} `json:"steps"`
		}
		if err := json.Unmarshal(data, &trajectory); err != nil {
			a.Log(fmt.Sprintf("[QAP-EXPORT] Error parsing trajectory JSON %s: %v", file.Name(), err))
			continue
		}

		a.Log(fmt.Sprintf("[QAP-EXPORT] File %s has %d steps", file.Name(), len(trajectory.Steps)))

		for si, step := range trajectory.Steps {
			a.Log(fmt.Sprintf("[QAP-EXPORT]   Step[%d]: type=%s, tool_name=%s, input_len=%d", si, step.Type, step.ToolName, len(step.ToolInput)))

			if step.Type != "tool_call" {
				continue
			}

			// Skip steps with explicit errors
			if step.Error != "" {
				a.Log(fmt.Sprintf("[QAP-EXPORT]   Skipping step with error: %s", truncStr(step.Error, 100)))
				continue
			}

			// Skip steps whose output indicates failure
			if isToolOutputFailed(step.ToolOutput) {
				a.Log(fmt.Sprintf("[QAP-EXPORT]   Skipping step with failed output: %s", truncStr(step.ToolOutput, 100)))
				continue
			}

			var args map[string]interface{}
			if step.ToolInput != "" {
				// Try direct JSON parse first (works for correctly escaped or unescaped inputs)
				if err := json.Unmarshal([]byte(step.ToolInput), &args); err != nil {
					a.Log(fmt.Sprintf("[QAP-EXPORT]   Direct parse failed: %v, trying unescape", err))
					// ToolInput was escaped by escapeForTraining before saving.
					// After JSON decoding from the trajectory file, we have the escaped string.
					// We need to reverse the escaping to get valid JSON.
					unescaped := unescapeToolInput(step.ToolInput)
					if err2 := json.Unmarshal([]byte(unescaped), &args); err2 != nil {
						a.Log(fmt.Sprintf("[QAP-EXPORT]   Unescape parse also failed: %v (first 200: %s)", err2, truncStr(unescaped, 200)))
						continue
					}
					a.Log(fmt.Sprintf("[QAP-EXPORT]   Unescape parse succeeded"))
				} else {
					a.Log(fmt.Sprintf("[QAP-EXPORT]   Direct parse succeeded"))
				}
			} else {
				a.Log(fmt.Sprintf("[QAP-EXPORT]   Empty tool_input, skipping"))
				continue
			}

			switch step.ToolName {
			case "execute_sql":
				query, _ := args["query"].(string)
				if query != "" {
					desc, _ := args["description"].(string)
					if desc == "" {
						desc = "SQL Query"
					}
					steps = append(steps, PackStep{
						StepID:      stepID,
						StepType:    stepTypeSQL,
						Code:        query,
						Description: desc,
						DependsOn:   buildDependsOn(stepID),
					})
					a.Log(fmt.Sprintf("[QAP-EXPORT]   Added SQL step #%d: %s", stepID, truncStr(query, 80)))
					stepID++
				}

			case "execute_python":
				code, _ := args["code"].(string)
				if code != "" {
					desc, _ := args["description"].(string)
					if desc == "" {
						desc = "Python Code"
					}
					steps = append(steps, PackStep{
						StepID:      stepID,
						StepType:    stepTypePython,
						Code:        code,
						Description: desc,
						DependsOn:   buildDependsOn(stepID),
					})
					a.Log(fmt.Sprintf("[QAP-EXPORT]   Added Python step #%d: %s", stepID, truncStr(code, 80)))
					stepID++
				}

			case "query_and_chart":
				query, _ := args["query"].(string)
				chartCode, _ := args["chart_code"].(string)
				chartTitle, _ := args["chart_title"].(string)
				a.Log(fmt.Sprintf("[QAP-EXPORT]   query_and_chart: query_len=%d, chart_len=%d, title=%s", len(query), len(chartCode), chartTitle))

				if query != "" {
					desc := chartTitle
					if desc == "" {
						desc = "SQL Query"
					}
					sqlStepID := stepID
					steps = append(steps, PackStep{
						StepID:      stepID,
						StepType:    stepTypeSQL,
						Code:        query,
						Description: desc,
						DependsOn:   buildDependsOn(stepID),
					})
					a.Log(fmt.Sprintf("[QAP-EXPORT]   Added SQL step #%d from query_and_chart", stepID))
					stepID++

					if chartCode != "" {
						desc := chartTitle
						if desc == "" {
							desc = "Chart Generation"
						} else {
							desc = desc + " (Chart)"
						}
						steps = append(steps, PackStep{
							StepID:          stepID,
							StepType:        stepTypePython,
							Code:            chartCode,
							Description:     desc,
							DependsOn:       []int{sqlStepID},
							SourceTool:      "query_and_chart",
							PairedSQLStepID: sqlStepID,
						})
						a.Log(fmt.Sprintf("[QAP-EXPORT]   Added Chart step #%d from query_and_chart (paired with SQL #%d)", stepID, sqlStepID))
						stepID++
					}
				} else if chartCode != "" {
					desc := chartTitle
					if desc == "" {
						desc = "Chart Generation"
					} else {
						desc = desc + " (Chart)"
					}
					steps = append(steps, PackStep{
						StepID:      stepID,
						StepType:    stepTypePython,
						Code:        chartCode,
						Description: desc,
						DependsOn:   buildDependsOn(stepID),
						SourceTool:  "query_and_chart",
					})
					a.Log(fmt.Sprintf("[QAP-EXPORT]   Added Chart step #%d from query_and_chart (no paired SQL)", stepID))
					stepID++
				}

			default:
				a.Log(fmt.Sprintf("[QAP-EXPORT]   Unhandled tool: %s", step.ToolName))
			}
		}
	}

	a.Log(fmt.Sprintf("[QAP-EXPORT] Extraction from trajectory complete: found %d steps total", len(steps)))

	return steps, nil
}

// collectStepsFromExecutions extracts steps from the executions.json file.
// This is the primary source for SQL steps as it only records successful executions.
// It also captures Python steps if they were recorded in executions.json.
func (a *App) collectStepsFromExecutions(threadID string) []PackStep {
	cfg, _ := a.GetConfig()
	executionsPath := filepath.Join(cfg.DataCacheDir, "sessions", threadID, "execution", "executions.json")

	data, err := os.ReadFile(executionsPath)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] executions.json not found: %v", err))
		return nil
	}

	var executionsMap parsedExecutionsMap
	if err := json.Unmarshal(data, &executionsMap); err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Error parsing executions.json: %v", err))
		return nil
	}

	// Collect all request entries and sort by timestamp
	type requestEntry struct {
		key   string
		entry parsedExecutionEntry
	}
	var entries []requestEntry
	for k, v := range executionsMap {
		entries = append(entries, requestEntry{key: k, entry: v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].entry.Timestamp < entries[j].entry.Timestamp
	})

	var steps []PackStep
	stepID := 1
	seenCode := make(map[string]bool)

	for _, entry := range entries {
		for _, exec := range entry.entry.Executions {
			if exec.Code == "" || !exec.Success {
				continue
			}

			codeKey := exec.Type + ":" + exec.Code
			if seenCode[codeKey] {
				continue
			}
			seenCode[codeKey] = true

			stepType := execTypeToStepType(exec.Type)
			if stepType == "" {
				continue
			}

			desc := exec.StepDescription
			if desc == "" {
				desc = exec.UserRequest
			}
			if desc == "" {
				if stepType == stepTypeSQL {
					desc = "SQL Query"
				} else {
					desc = "Python Code"
				}
			}

			steps = append(steps, PackStep{
				StepID:      stepID,
				StepType:    stepType,
				Code:        exec.Code,
				Description: desc,
				UserRequest: exec.UserRequest,
				DependsOn:   buildDependsOn(stepID),
			})
			a.Log(fmt.Sprintf("[QAP-EXPORT] Fallback step #%d: type=%s, desc=%s", stepID, stepType, truncStr(desc, 60)))
			stepID++
		}
	}

	return steps
}



// isToolOutputFailed checks if a tool output indicates the tool call failed.
// This is used to filter out failed attempts from trajectory when exporting.
func isToolOutputFailed(output string) bool {
	if output == "" {
		return false
	}
	lower := strings.ToLower(output)
	// Check for SQL error indicators
	sqlErrorIndicators := []string{
		"sql logic error",
		"no such column",
		"no such table",
		"syntax error",
		"query failed",
		"ambiguous column",
		"near \"",
		"table already exists",
		"unique constraint failed",
		"not null constraint failed",
		"foreign key constraint failed",
	}
	for _, indicator := range sqlErrorIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	// Check for Python/chart error indicators
	pythonErrorIndicators := []string{
		"python path is not configured",
		"traceback (most recent call last)",
		"modulenotfounderror",
		"nameerror",
		"syntaxerror",
		"typeerror",
		"valueerror",
		"keyerror",
		"indexerror",
		"filenotfounderror",
		"importerror",
		"attributeerror",
		"zerodivisionerror",
		"runtimeerror",
	}
	for _, indicator := range pythonErrorIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	// Check for non-zero exit status (exit status 0 is success)
	if strings.Contains(lower, "exit status") && !strings.Contains(lower, "exit status 0") {
		return true
	}
	// Check for explicit failure markers
	if strings.Contains(lower, "\"success\":false") || strings.Contains(lower, "\"success\\\":false") ||
		strings.Contains(lower, "\\\\\\\"success\\\\\\\":false") {
		return true
	}
	return false
}

// unescapeToolInput reverses the escapeForTraining transformation applied to ToolInput.
// After JSON decoding the trajectory file, ToolInput contains the escapeForTraining output.
// escapeForTraining does: \ → \\, \n → \\n, \r → \\r, \t → \\t, " → \"
// The result should be valid JSON (the original tool call arguments).
func unescapeToolInput(input string) string {
	input = strings.TrimSpace(input)

	// Try multiple unescape strategies and pick the first that produces valid JSON.
	var test map[string]interface{}

	// Strategy 1: Reverse escapeForTraining in correct order.
	// escapeForTraining output has: \" for quotes, \\n for JSON newline escapes, \\\\ for literal backslashes.
	// We reverse: \" → ", \\n → \n (JSON escape, not actual newline), \\\\ → \\, etc.
	s1 := input
	// Use a unique sentinel for literal backslash pairs
	const sentinel = "\x00\x01\x02BSLASH\x02\x01\x00"
	s1 = strings.ReplaceAll(s1, `\\\\`, sentinel)   // protect literal backslash pairs
	s1 = strings.ReplaceAll(s1, `\\"`, `"`)          // \" after \\ protection → just "
	s1 = strings.ReplaceAll(s1, `\"`, `"`)            // remaining \" → "
	s1 = strings.ReplaceAll(s1, `\\n`, `\n`)          // \\n → \n (JSON escape)
	s1 = strings.ReplaceAll(s1, `\\r`, `\r`)          // \\r → \r (JSON escape)
	s1 = strings.ReplaceAll(s1, `\\t`, `\t`)          // \\t → \t (JSON escape)
	s1 = strings.ReplaceAll(s1, `\\`, `\`)            // remaining \\ → \
	s1 = strings.ReplaceAll(s1, sentinel, `\\`)       // restore literal backslash pairs
	if json.Unmarshal([]byte(s1), &test) == nil {
		return s1
	}

	// Strategy 2: Simple single-level unescape — quotes first, then backslashes.
	s2 := input
	s2 = strings.ReplaceAll(s2, `\"`, `"`)
	s2 = strings.ReplaceAll(s2, `\\`, `\`)
	if json.Unmarshal([]byte(s2), &test) == nil {
		return s2
	}

	// Strategy 3: Reverse order — backslashes first, then quotes.
	s3 := input
	s3 = strings.ReplaceAll(s3, `\\`, `\`)
	s3 = strings.ReplaceAll(s3, `\"`, `"`)
	if json.Unmarshal([]byte(s3), &test) == nil {
		return s3
	}

	// Return strategy 1 result as best effort
	return s1
}

// attachEChartsFromMessages 从会话的 assistant 消息中提取 ECharts JSON 配置，
// 并附加到对应的步骤上。原始分析中 LLM 会在文本响应中直接生成 json:echarts 块，
// 这些图表配置需要在导出时保存，以便快捷分析包重放时能重新显示图表。
func (a *App) attachEChartsFromMessages(thread *ChatThread, steps []PackStep) {
	if thread == nil || len(thread.Messages) == 0 || len(steps) == 0 {
		return
	}

	reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	reEChartsNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:echarts\\s*\\n(\\{[\\s\\S]+?\\n\\})(?:\\s*\\n(?:---|###)|\\s*$)")

	// 收集所有 assistant 消息中的 ECharts 配置，按消息顺序
	type echartsGroup struct {
		configs []string
	}
	var groups []echartsGroup

	for _, msg := range thread.Messages {
		if msg.Role != "assistant" {
			continue
		}

		var configs []string
		// 提取 backtick 格式的 ECharts
		for _, match := range reECharts.FindAllStringSubmatch(msg.Content, -1) {
			if len(match) > 1 {
				chartJSON := strings.TrimSpace(match[1])
				if chartJSON != "" && json.Valid([]byte(chartJSON)) {
					configs = append(configs, chartJSON)
				}
			}
		}
		// 提取无 backtick 格式的 ECharts
		for _, match := range reEChartsNoBT.FindAllStringSubmatch(msg.Content, -1) {
			if len(match) > 1 {
				chartJSON := strings.TrimSpace(match[1])
				if chartJSON != "" && json.Valid([]byte(chartJSON)) {
					configs = append(configs, chartJSON)
				}
			}
		}

		if len(configs) > 0 {
			groups = append(groups, echartsGroup{configs: configs})
		}
	}

	if len(groups) == 0 {
		return
	}

	// 将 ECharts 配置分配到步骤上：按顺序一一对应
	// 如果 ECharts 组数多于步骤数，多余的附加到最后一个步骤
	for i, group := range groups {
		stepIdx := i
		if stepIdx >= len(steps) {
			stepIdx = len(steps) - 1
		}
		steps[stepIdx].EChartsConfigs = append(steps[stepIdx].EChartsConfigs, group.configs...)
	}

	totalCharts := 0
	for _, g := range groups {
		totalCharts += len(g.configs)
	}
	a.Log(fmt.Sprintf("[QAP-EXPORT] Attached %d ECharts configs from %d message groups to steps", totalCharts, len(groups)))
}
