package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================================================================
// Data Structures for Analysis Export/Import
// ============================================================================

// AnalysisExport represents the complete export format
type AnalysisExport struct {
	FileType           string             `json:"file_type"`            // "RapidBI_Analysis_Export"
	FormatVersion      string             `json:"format_version"`       // "2.0"
	Description        string             `json:"description"`          // Human-readable description
	ExportedAt         string             `json:"exported_at"`          // RFC3339 format
	DataSource         DataSourceInfo     `json:"data_source"`
	SchemaRequirements SchemaRequirements `json:"schema_requirements"`
	ExecutableSteps    []ExecutableStep   `json:"executable_steps"`
	OriginalResults    *OriginalResults   `json:"original_results,omitempty"`
}

// DataSourceInfo contains metadata about the original data source
type DataSourceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// SchemaRequirements defines the database schema needed to run this analysis
type SchemaRequirements struct {
	Tables []TableRequirement `json:"tables"`
}

// TableRequirement specifies a required table and its columns
type TableRequirement struct {
	Name    string            `json:"name"`
	Columns []string          `json:"columns"`
	Types   map[string]string `json:"types,omitempty"`
}

// ExecutableStep represents a single step in the analysis (SQL or Python)
type ExecutableStep struct {
	StepID               int                    `json:"step_id"`
	StepType             string                 `json:"step_type"` // "sql_query", "python_code"
	Description          string                 `json:"description"`
	SQL                  string                 `json:"sql,omitempty"`
	PythonCode           string                 `json:"code,omitempty"`
	DependsOn            []int                  `json:"depends_on,omitempty"`
	ExpectedResultSchema *ResultSchema          `json:"expected_result_schema,omitempty"`
	Produces             *ProducesInfo          `json:"produces,omitempty"`
}

// ResultSchema defines the expected structure of query results
type ResultSchema struct {
	Columns []string `json:"columns"`
	Types   []string `json:"types"`
}

// ProducesInfo describes what this step generates
type ProducesInfo struct {
	Type     string `json:"type"`     // "image", "table", "chart"
	Filename string `json:"filename,omitempty"`
}

// OriginalResults stores the results from the original execution
type OriginalResults struct {
	Summary        string                   `json:"summary"`
	Visualizations []map[string]interface{} `json:"visualizations,omitempty"`
}

// ============================================================================
// Export Implementation
// ============================================================================

// ExportAnalysisProcess exports the analysis results for a user message
func (a *App) ExportAnalysisProcess(messageID string) error {
	a.Log(fmt.Sprintf("[EXPORT] Starting export for message ID: %s", messageID))
	
	// 1. Find the message and thread
	thread, message, err := a.findMessageByID(messageID)
	if err != nil {
		a.Log(fmt.Sprintf("[EXPORT] Error finding message: %v", err))
		return err
	}
	
	if thread == nil || message == nil {
		return fmt.Errorf("message not found")
	}
	
	a.Log(fmt.Sprintf("[EXPORT] Found message in thread: %s", thread.ID))
	
	// 2. Extract executable steps from session
	steps, err := a.extractExecutableSteps(thread.ID, message)
	if err != nil {
		a.Log(fmt.Sprintf("[EXPORT] Error extracting steps: %v", err))
		return err
	}
	
	if len(steps) == 0 {
		a.Log("[EXPORT] No executable steps found")
		return fmt.Errorf("no executable steps found in this analysis")
	}
	
	a.Log(fmt.Sprintf("[EXPORT] Extracted %d executable steps", len(steps)))
	
	// 3. Extract schema requirements
	schemaReq := a.extractSchemaFromSteps(steps)
	a.Log(fmt.Sprintf("[EXPORT] Extracted schema requirements for %d tables", len(schemaReq.Tables)))
	
	// 4. Get data source info
	dataSource, err := a.getDataSourceInfoForThread(thread.DataSourceID)
	if err != nil {
		a.Log(fmt.Sprintf("[EXPORT] Error getting data source info: %v", err))
		return err
	}
	
	// 5. Build export structure
	export := AnalysisExport{
		FileType:           "RapidBI_Analysis_Export",
		FormatVersion:      "2.0",
		Description:        "RapidBI 分析过程导出文件 - 包含可执行的 SQL/Python 步骤",
		ExportedAt:         time.Now().Format(time.RFC3339),
		DataSource:         dataSource,
		SchemaRequirements: schemaReq,
		ExecutableSteps:    steps,
	}
	
	// 6. Marshal to JSON
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		a.Log(fmt.Sprintf("[EXPORT] Error marshaling JSON: %v", err))
		return err
	}
	
	// 7. Show save dialog with .rbi extension
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出分析过程",
		DefaultFilename: fmt.Sprintf("analysis_%s.rbi", time.Now().Format("20060102_150405")),
		Filters: []runtime.FileFilter{
			{DisplayName: "RapidBI Analysis", Pattern: "*.rbi"},
		},
	})
	
	if err != nil || savePath == "" {
		a.Log("[EXPORT] User cancelled save dialog")
		return nil // User cancelled
	}
	
	// 8. Write file
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		a.Log(fmt.Sprintf("[EXPORT] Error writing file: %v", err))
		return err
	}
	
	a.Log(fmt.Sprintf("[EXPORT] Successfully exported to: %s", savePath))
	return nil
}

// findMessageByID finds a message by ID across all threads
func (a *App) findMessageByID(messageID string) (*ChatThread, *ChatMessage, error) {
	threads, err := a.chatService.LoadThreads()
	if err != nil {
		return nil, nil, err
	}
	
	for i := range threads {
		for j := range threads[i].Messages {
			if threads[i].Messages[j].ID == messageID {
				return &threads[i], &threads[i].Messages[j], nil
			}
		}
	}
	
	return nil, nil, fmt.Errorf("message not found")
}

// extractExecutableSteps parses trajectory files to extract SQL and Python code
func (a *App) extractExecutableSteps(threadID string, userMsg *ChatMessage) ([]ExecutableStep, error) {
	cfg, _ := a.GetConfig()
	trajectoryDir := filepath.Join(cfg.DataCacheDir, "sessions", threadID, "trajectory")
	
	a.Log(fmt.Sprintf("[EXPORT] Looking for trajectory files in: %s", trajectoryDir))
	
	// List all trajectory files
	files, err := os.ReadDir(trajectoryDir)
	if err != nil {
		a.Log(fmt.Sprintf("[EXPORT] Trajectory directory not found: %v", err))
		return nil, fmt.Errorf("trajectory directory not found: %v", err)
	}
	
	a.Log(fmt.Sprintf("[EXPORT] Found %d files in trajectory directory", len(files)))
	
	var steps []ExecutableStep
	stepID := 1
	
	// Process each trajectory file (they are named by request ID)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		
		trajectoryPath := filepath.Join(trajectoryDir, file.Name())
		a.Log(fmt.Sprintf("[EXPORT] Reading trajectory file: %s", file.Name()))
		
		data, err := os.ReadFile(trajectoryPath)
		if err != nil {
			a.Log(fmt.Sprintf("[EXPORT] Error reading file %s: %v", file.Name(), err))
			continue
		}
		
		// Parse trace to find tool calls
		var trace []map[string]interface{}
		if err := json.Unmarshal(data, &trace); err != nil {
			a.Log(fmt.Sprintf("[EXPORT] Error parsing JSON in %s: %v", file.Name(), err))
			continue
		}
		
		a.Log(fmt.Sprintf("[EXPORT] Parsed trajectory %s with %d entries", file.Name(), len(trace)))
		
		// Check if this trajectory contains our user message
		foundUserMsg := false
		for idx, entry := range trace {
			role, _ := entry["role"].(string)
			
			// Look for the user message
			if role == "user" && !foundUserMsg {
				content, _ := entry["content"].(string)
				// Try exact match first
				if content == userMsg.Content {
					a.Log(fmt.Sprintf("[EXPORT] Found exact matching user message in %s at index %d", file.Name(), idx))
					foundUserMsg = true
				} else if strings.Contains(content, userMsg.Content) || strings.Contains(userMsg.Content, content) {
					// Fallback: partial match (in case message was truncated or modified)
					a.Log(fmt.Sprintf("[EXPORT] Found partial matching user message in %s at index %d", file.Name(), idx))
					foundUserMsg = true
				}
				continue
			}
			
			if !foundUserMsg {
				continue
			}
			
			// Extract tool calls from assistant messages
			if role == "assistant" {
				a.Log(fmt.Sprintf("[EXPORT] Processing assistant message at index %d", idx))
				toolCalls, ok := entry["tool_calls"].([]interface{})
				if !ok {
					a.Log("[EXPORT] No tool_calls found in assistant message")
					continue
				}
				
				a.Log(fmt.Sprintf("[EXPORT] Found %d tool calls", len(toolCalls)))
				
				for tcIdx, tc := range toolCalls {
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
					
					a.Log(fmt.Sprintf("[EXPORT] Tool call %d: %s", tcIdx, toolName))
					
					var args map[string]interface{}
					if argsStr != "" {
						json.Unmarshal([]byte(argsStr), &args)
					}
					
					switch toolName {
					case "execute_sql":
						query, _ := args["query"].(string)
						if query != "" {
							a.Log(fmt.Sprintf("[EXPORT] Adding SQL step: %.100s...", query))
							steps = append(steps, ExecutableStep{
								StepID:      stepID,
								StepType:    "sql_query",
								Description: "SQL Query Execution",
								SQL:         query,
							})
							stepID++
						}
						
					case "execute_python":
						code, _ := args["code"].(string)
						if code != "" {
							a.Log(fmt.Sprintf("[EXPORT] Adding Python step: %.100s...", code))
							steps = append(steps, ExecutableStep{
								StepID:      stepID,
								StepType:    "python_code",
								Description: "Python Execution",
								PythonCode:  code,
							})
							stepID++
						}
					}
				}
				
				// Found the message and processed its response, we're done with this file
				break
			}
		}
		
		// If we found the message in this file, we're done
		if foundUserMsg {
			a.Log(fmt.Sprintf("[EXPORT] Successfully extracted steps from %s", file.Name()))
			break
		} else {
			a.Log(fmt.Sprintf("[EXPORT] User message not found in %s, trying next file", file.Name()))
		}
	}
	
	// Fallback: if no user message matched, extract ALL tool calls from latest trajectory
	if len(steps) == 0 && len(files) > 0 {
		a.Log("[EXPORT] No matching user message found, extracting all steps from latest trajectory as fallback")
		
		// Find the latest trajectory file
		var latestFile os.DirEntry
		var latestTime int64
		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".json") {
				continue
			}
			// Extract timestamp from filename
			name := strings.TrimSuffix(file.Name(), ".json")
			if timestamp, err := strconv.ParseInt(name, 10, 64); err == nil {
				if timestamp > latestTime {
					latestTime = timestamp
					latestFile = file
				}
			}
		}
		
		if latestFile != nil {
			trajectoryPath := filepath.Join(trajectoryDir, latestFile.Name())
			a.Log(fmt.Sprintf("[EXPORT] Using latest trajectory file: %s", latestFile.Name()))
			
			data, err := os.ReadFile(trajectoryPath)
			if err == nil {
				var trace []map[string]interface{}
				if err := json.Unmarshal(data, &trace); err == nil {
					// Extract all tool calls without matching user message
					for _, entry := range trace {
						role, _ := entry["role"].(string)
						if role == "assistant" {
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
										a.Log(fmt.Sprintf("[EXPORT] Adding SQL step from fallback: %.100s...", query))
										steps = append(steps, ExecutableStep{
											StepID:      stepID,
											StepType:    "sql_query",
											Description: "SQL Query Execution",
											SQL:         query,
										})
										stepID++
									}
									
								case "execute_python":
									code, _ := args["code"].(string)
									if code != "" {
										a.Log(fmt.Sprintf("[EXPORT] Adding Python step from fallback: %.100s...", code))
										steps = append(steps, ExecutableStep{
											StepID:      stepID,
											StepType:    "python_code",
											Description: "Python Execution",
											PythonCode:  code,
										})
										stepID++
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	a.Log(fmt.Sprintf("[EXPORT] Extraction complete: found %d steps", len(steps)))
	return steps, nil
}

// extractSchemaFromSteps analyzes SQL queries to extract table and column requirements
func (a *App) extractSchemaFromSteps(steps []ExecutableStep) SchemaRequirements {
	tableMap := make(map[string]map[string]bool) // table -> set of columns
	
	for _, step := range steps {
		if step.StepType != "sql_query" {
			continue
		}
		
		// Extract table names (simple regex for FROM and JOIN clauses)
		tablePattern := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
		tables := tablePattern.FindAllStringSubmatch(step.SQL, -1)
		
		for _, match := range tables {
			tableName := match[1]
			if tableMap[tableName] == nil {
				tableMap[tableName] = make(map[string]bool)
			}
		}
		
		// Extract column names (simplified - doesn't handle all SQL features)
		// This regex looks for identifiers that might be column names
		colPattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\b`)
		cols := colPattern.FindAllString(step.SQL, -1)
		
		// Filter out SQL keywords
		sqlKeywords := map[string]bool{
			"SELECT": true, "FROM": true, "WHERE": true, "AND": true, "OR": true,
			"JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
			"ON": true, "AS": true, "GROUP": true, "BY": true, "ORDER": true,
			"LIMIT": true, "OFFSET": true, "HAVING": true, "COUNT": true, "SUM": true,
			"AVG": true, "MAX": true, "MIN": true, "DISTINCT": true, "NULL": true,
			"IS": true, "NOT": true, "IN": true, "LIKE": true, "BETWEEN": true,
		}
		
		for _, col := range cols {
			upperCol := strings.ToUpper(col)
			if !sqlKeywords[upperCol] {
				// Add to all tables (imprecise, but good enough for validation)
				for tableName := range tableMap {
					tableMap[tableName][col] = true
				}
			}
		}
	}
	
	// Convert map to structured format
	var requirements SchemaRequirements
	for tableName, columns := range tableMap {
		var colList []string
		for col := range columns {
			colList = append(colList, col)
		}
		requirements.Tables = append(requirements.Tables, TableRequirement{
			Name:    tableName,
			Columns: colList,
		})
	}
	
	return requirements
}

// getDataSourceInfoForThread retrieves data source metadata
func (a *App) getDataSourceInfoForThread(dataSourceID string) (DataSourceInfo, error) {
	if a.dataSourceService == nil {
		return DataSourceInfo{}, fmt.Errorf("data source service not initialized")
	}
	
	sources, err := a.dataSourceService.LoadDataSources()
	if err != nil {
		return DataSourceInfo{}, err
	}
	
	for _, ds := range sources {
		if ds.ID == dataSourceID {
			return DataSourceInfo{
				ID:   ds.ID,
				Name: ds.Name,
				Type: ds.Type,
			}, nil
		}
	}
	
	return DataSourceInfo{}, fmt.Errorf("data source not found")
}
