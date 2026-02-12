package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportQuickAnalysisPack exports a quick analysis pack from a chat session.
// It extracts all SQL/Python steps from the session trajectory, collects the full
// data source schema, and packages everything into a .qap ZIP file.
//
// Parameters:
//   - threadID: the chat thread/session ID
//   - author: creator name (user input)
//   - password: optional encryption password (empty string = no encryption)
func (a *App) ExportQuickAnalysisPack(threadID string, author string, password string) error {
	a.Log(fmt.Sprintf("[QAP-EXPORT] Starting export for thread: %s, author: %s", threadID, author))

	// 1. Load the thread to get data source ID
	thread, err := a.chatService.LoadThread(threadID)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Error loading thread: %v", err))
		return fmt.Errorf("failed to load thread: %w", err)
	}

	// 2. Extract all SQL/Python steps from trajectory files
	steps, err := a.collectSessionSteps(threadID)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Error collecting steps: %v", err))
		return err
	}

	if len(steps) == 0 {
		a.Log("[QAP-EXPORT] No executable steps found")
		return fmt.Errorf("该会话没有可导出的分析操作")
	}

	a.Log(fmt.Sprintf("[QAP-EXPORT] Collected %d steps", len(steps)))

	// 3. Collect full schema from the data source
	schema, err := a.collectFullSchema(thread.DataSourceID)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Error collecting schema: %v", err))
		return fmt.Errorf("failed to collect schema: %w", err)
	}

	a.Log(fmt.Sprintf("[QAP-EXPORT] Collected schema with %d tables", len(schema)))

	// 4. Get data source name for metadata
	dsInfo, err := a.getDataSourceInfoForThread(thread.DataSourceID)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Error getting data source info: %v", err))
		return fmt.Errorf("failed to get data source info: %w", err)
	}

	// 5. Build QuickAnalysisPack object
	pack := QuickAnalysisPack{
		FileType:      "VantageData_QuickAnalysisPack",
		FormatVersion: "1.0",
		Metadata: PackMetadata{
			Author:     author,
			CreatedAt:  time.Now().Format(time.RFC3339),
			SourceName: dsInfo.Name,
		},
		SchemaRequirements: schema,
		ExecutableSteps:    steps,
	}

	// 6. Marshal to JSON
	jsonData, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Error marshaling JSON: %v", err))
		return fmt.Errorf("failed to marshal pack: %w", err)
	}

	// 7. Show save dialog with .qap extension
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出快捷分析包",
		DefaultFilename: fmt.Sprintf("analysis_%s.qap", time.Now().Format("20060102_150405")),
		Filters: []runtime.FileFilter{
			{DisplayName: "Quick Analysis Pack (*.qap)", Pattern: "*.qap"},
		},
	})

	if err != nil || savePath == "" {
		a.Log("[QAP-EXPORT] User cancelled save dialog")
		return nil // User cancelled
	}

	// 8. Pack to ZIP (with optional encryption)
	if err := PackToZip(jsonData, savePath, password); err != nil {
		a.Log(fmt.Sprintf("[QAP-EXPORT] Error creating ZIP: %v", err))
		return fmt.Errorf("failed to create pack file: %w", err)
	}

	a.Log(fmt.Sprintf("[QAP-EXPORT] Successfully exported to: %s", savePath))
	return nil
}

// collectSessionSteps extracts all SQL and Python steps from a session's trajectory files.
// It reads all trajectory JSON files in chronological order and extracts tool calls
// for execute_sql and execute_python, converting them to PackStep format.
func (a *App) collectSessionSteps(threadID string) ([]PackStep, error) {
	cfg, _ := a.GetConfig()
	trajectoryDir := filepath.Join(cfg.DataCacheDir, "sessions", threadID, "trajectory")

	a.Log(fmt.Sprintf("[QAP-EXPORT] Looking for trajectory files in: %s", trajectoryDir))

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
		// Parse timestamps from filenames for proper ordering
		nameI := strings.TrimSuffix(jsonFiles[i].Name(), ".json")
		nameJ := strings.TrimSuffix(jsonFiles[j].Name(), ".json")
		tsI, errI := strconv.ParseInt(nameI, 10, 64)
		tsJ, errJ := strconv.ParseInt(nameJ, 10, 64)
		if errI != nil || errJ != nil {
			return jsonFiles[i].Name() < jsonFiles[j].Name()
		}
		return tsI < tsJ
	})

	a.Log(fmt.Sprintf("[QAP-EXPORT] Found %d trajectory files", len(jsonFiles)))

	var steps []PackStep
	stepID := 1

	for _, file := range jsonFiles {
		trajectoryPath := filepath.Join(trajectoryDir, file.Name())
		data, err := os.ReadFile(trajectoryPath)
		if err != nil {
			a.Log(fmt.Sprintf("[QAP-EXPORT] Error reading %s: %v", file.Name(), err))
			continue
		}

		var trace []map[string]interface{}
		if err := json.Unmarshal(data, &trace); err != nil {
			a.Log(fmt.Sprintf("[QAP-EXPORT] Error parsing %s: %v", file.Name(), err))
			continue
		}

		for _, entry := range trace {
			role, _ := entry["role"].(string)
			if role != "assistant" {
				continue
			}

			toolCalls, ok := entry["tool_calls"].([]interface{})
			if !ok {
				continue
			}

			for _, tc := range toolCalls {
				toolCall, ok := tc.(map[string]interface{})
				if !ok {
					continue
				}

				function, ok := toolCall["function"].(map[string]interface{})
				if !ok {
					continue
				}

				toolName, _ := function["name"].(string)
				argsStr, _ := function["arguments"].(string)

				var args map[string]interface{}
				if argsStr != "" {
					json.Unmarshal([]byte(argsStr), &args)
				}

				switch toolName {
				case "execute_sql":
					query, _ := args["query"].(string)
					if query != "" {
						desc, _ := args["description"].(string)
						if desc == "" {
							desc = "SQL Query"
						}
						steps = append(steps, PackStep{
							StepID:      stepID,
							StepType:    "sql_query",
							Code:        query,
							Description: desc,
							DependsOn:   buildDependsOn(stepID),
						})
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
							StepType:    "python_code",
							Code:        code,
							Description: desc,
							DependsOn:   buildDependsOn(stepID),
						})
						stepID++
					}
				}
			}
		}
	}

	a.Log(fmt.Sprintf("[QAP-EXPORT] Extraction complete: found %d steps", len(steps)))
	return steps, nil
}

// buildDependsOn returns the dependency list for a step.
// Each step depends on the previous step (sequential execution order).
// The first step has no dependencies.
func buildDependsOn(currentStepID int) []int {
	if currentStepID <= 1 {
		return nil
	}
	return []int{currentStepID - 1}
}
