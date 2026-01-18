package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ExecutionRecord represents a single execution (SQL or Python)
type ExecutionRecord struct {
	Index            int                    `json:"index"`              // Execution index (1, 2, 3...)
	Type             string                 `json:"type"`               // "sql" or "python"
	Timestamp        int64                  `json:"timestamp"`          // Unix timestamp in milliseconds
	UserRequest      string                 `json:"user_request"`       // User's original request/question
	ExecutionOrder   int                    `json:"execution_order"`    // Order within the analysis (1, 2, 3...)
	StepDescription  string                 `json:"step_description"`   // What this step does (e.g., "Query sales data", "Calculate RFM scores")
	DataSourceID     string                 `json:"data_source_id"`     // Data source ID
	DataSourceName   string                 `json:"data_source_name"`   // Data source name
	Tables           []TableMetadata        `json:"tables"`             // Tables involved
	Code             string                 `json:"code"`               // SQL query or Python code
	CodeFile         string                 `json:"code_file"`          // Relative path to code file
	Success          bool                   `json:"success"`            // Whether execution succeeded
	Error            string                 `json:"error,omitempty"`    // Error message if failed
	Output           string                 `json:"output,omitempty"`   // Execution output (truncated)
	DependsOn        []int                  `json:"depends_on,omitempty"` // Indices of executions this depends on
}

// TableMetadata represents metadata for a table
type TableMetadata struct {
	Name    string   `json:"name"`    // Table name
	Columns []string `json:"columns"` // Column names
}

// ExecutionRecorder records SQL and Python executions
type ExecutionRecorder struct {
	sessionDir     string
	executionDir   string
	records        []ExecutionRecord
	dataSourceID   string
	dataSourceName string
	userRequest    string // User's original request
	messageID      string // Message ID to group executions
	executionOrder int    // Current execution order counter
	logger         func(string)
}

// NewExecutionRecorder creates a new execution recorder
func NewExecutionRecorder(sessionDir, dataSourceID, dataSourceName, userRequest, messageID string, logger func(string)) *ExecutionRecorder {
	executionDir := filepath.Join(sessionDir, "execution")
	
	// Create execution directory
	if err := os.MkdirAll(executionDir, 0755); err != nil {
		if logger != nil {
			logger(fmt.Sprintf("[EXECUTION-RECORDER] Failed to create execution directory: %v", err))
		}
	}
	
	return &ExecutionRecorder{
		sessionDir:     sessionDir,
		executionDir:   executionDir,
		records:        []ExecutionRecord{},
		dataSourceID:   dataSourceID,
		dataSourceName: dataSourceName,
		userRequest:    userRequest,
		messageID:      messageID,
		executionOrder: 0,
		logger:         logger,
	}
}

// SetUserRequest updates the user request (useful if it changes during analysis)
func (r *ExecutionRecorder) SetUserRequest(request string) {
	r.userRequest = request
}

// RecordSQL records a SQL execution
func (r *ExecutionRecorder) RecordSQL(query string, tables []TableMetadata, success bool, errorMsg string, output string, stepDescription string) error {
	index := len(r.records) + 1
	r.executionOrder++
	timestamp := time.Now().UnixMilli()
	
	// Save SQL to file
	filename := fmt.Sprintf("%03d_sql.sql", index)
	filepath := filepath.Join(r.executionDir, filename)
	
	// Add comment header to SQL file with metadata
	header := fmt.Sprintf("-- Execution Order: %d\n-- User Request: %s\n-- Step: %s\n-- Timestamp: %s\n-- Data Source: %s\n\n",
		r.executionOrder,
		r.userRequest,
		stepDescription,
		time.Unix(timestamp/1000, 0).Format("2006-01-02 15:04:05"),
		r.dataSourceName,
	)
	
	fullContent := header + query
	
	if err := os.WriteFile(filepath, []byte(fullContent), 0644); err != nil {
		if r.logger != nil {
			r.logger(fmt.Sprintf("[EXECUTION-RECORDER] Failed to save SQL file: %v", err))
		}
		return err
	}
	
	// Truncate output if too long
	truncatedOutput := output
	if len(output) > 1000 {
		truncatedOutput = output[:1000] + "... (truncated)"
	}
	
	// Determine dependencies (previous SQL executions that this might depend on)
	var dependsOn []int
	for i := len(r.records) - 1; i >= 0; i-- {
		if r.records[i].Type == "sql" && r.records[i].Success {
			// Simple heuristic: if this SQL references tables from previous SQL, it depends on it
			// For now, we'll just mark the immediate previous SQL as a dependency
			dependsOn = append(dependsOn, r.records[i].Index)
			break
		}
	}
	
	record := ExecutionRecord{
		Index:           index,
		Type:            "sql",
		Timestamp:       timestamp,
		UserRequest:     r.userRequest,
		ExecutionOrder:  r.executionOrder,
		StepDescription: stepDescription,
		DataSourceID:    r.dataSourceID,
		DataSourceName:  r.dataSourceName,
		Tables:          tables,
		Code:            query,
		CodeFile:        filename,
		Success:         success,
		Error:           errorMsg,
		Output:          truncatedOutput,
		DependsOn:       dependsOn,
	}
	
	r.records = append(r.records, record)
	
	if r.logger != nil {
		r.logger(fmt.Sprintf("[EXECUTION-RECORDER] Recorded SQL execution #%d (order %d): %s", index, r.executionOrder, filename))
	}
	
	return nil
}

// RecordPython records a Python execution
func (r *ExecutionRecorder) RecordPython(code string, success bool, errorMsg string, output string, stepDescription string) error {
	index := len(r.records) + 1
	r.executionOrder++
	timestamp := time.Now().UnixMilli()
	
	// Save Python code to file
	filename := fmt.Sprintf("%03d_python.py", index)
	filepath := filepath.Join(r.executionDir, filename)
	
	// Add comment header to Python file with metadata
	header := fmt.Sprintf("# Execution Order: %d\n# User Request: %s\n# Step: %s\n# Timestamp: %s\n# Data Source: %s\n\n",
		r.executionOrder,
		r.userRequest,
		stepDescription,
		time.Unix(timestamp/1000, 0).Format("2006-01-02 15:04:05"),
		r.dataSourceName,
	)
	
	fullContent := header + code
	
	if err := os.WriteFile(filepath, []byte(fullContent), 0644); err != nil {
		if r.logger != nil {
			r.logger(fmt.Sprintf("[EXECUTION-RECORDER] Failed to save Python file: %v", err))
		}
		return err
	}
	
	// Truncate output if too long
	truncatedOutput := output
	if len(output) > 1000 {
		truncatedOutput = output[:1000] + "... (truncated)"
	}
	
	// Determine dependencies (usually depends on the most recent SQL execution)
	var dependsOn []int
	for i := len(r.records) - 1; i >= 0; i-- {
		if r.records[i].Type == "sql" && r.records[i].Success {
			dependsOn = append(dependsOn, r.records[i].Index)
			break
		}
	}
	
	record := ExecutionRecord{
		Index:           index,
		Type:            "python",
		Timestamp:       timestamp,
		UserRequest:     r.userRequest,
		ExecutionOrder:  r.executionOrder,
		StepDescription: stepDescription,
		DataSourceID:    r.dataSourceID,
		DataSourceName:  r.dataSourceName,
		Tables:          []TableMetadata{}, // Python doesn't directly reference tables
		Code:            code,
		CodeFile:        filename,
		Success:         success,
		Error:           errorMsg,
		Output:          truncatedOutput,
		DependsOn:       dependsOn,
	}
	
	r.records = append(r.records, record)
	
	if r.logger != nil {
		r.logger(fmt.Sprintf("[EXECUTION-RECORDER] Recorded Python execution #%d (order %d): %s", index, r.executionOrder, filename))
	}
	
	return nil
}

// SaveToFile saves all execution records to a JSON file grouped by message
func (r *ExecutionRecorder) SaveToFile() error {
	if len(r.records) == 0 {
		if r.logger != nil {
			r.logger("[EXECUTION-RECORDER] No executions to save")
		}
		return nil
	}
	
	jsonPath := filepath.Join(r.executionDir, "executions.json")
	
	// Load existing data if file exists
	existingData := make(map[string]interface{})
	if fileData, err := os.ReadFile(jsonPath); err == nil {
		// File exists, load it
		if err := json.Unmarshal(fileData, &existingData); err != nil {
			if r.logger != nil {
				r.logger(fmt.Sprintf("[EXECUTION-RECORDER] Warning: Failed to parse existing JSON, will overwrite: %v", err))
			}
			existingData = make(map[string]interface{})
		}
	}
	
	// Add or update this message's executions
	existingData[r.messageID] = map[string]interface{}{
		"user_request":     r.userRequest,
		"data_source_id":   r.dataSourceID,
		"data_source_name": r.dataSourceName,
		"timestamp":        time.Now().UnixMilli(),
		"executions":       r.records,
	}
	
	// Marshal with indentation for readability
	data, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		if r.logger != nil {
			r.logger(fmt.Sprintf("[EXECUTION-RECORDER] Failed to marshal JSON: %v", err))
		}
		return err
	}
	
	// Write to file
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		if r.logger != nil {
			r.logger(fmt.Sprintf("[EXECUTION-RECORDER] Failed to save JSON file: %v", err))
		}
		return err
	}
	
	if r.logger != nil {
		r.logger(fmt.Sprintf("[EXECUTION-RECORDER] Saved %d execution records for message %s to %s", len(r.records), r.messageID, jsonPath))
	}
	
	return nil
}

// GetRecordCount returns the number of recorded executions
func (r *ExecutionRecorder) GetRecordCount() int {
	return len(r.records)
}
