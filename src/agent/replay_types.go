package agent

import (
	"encoding/json"
	"time"
)

// AnalysisStep represents a single step in an analysis workflow
type AnalysisStep struct {
	StepID      int    `json:"step_id"`
	Timestamp   int64  `json:"timestamp"`    // Unix timestamp in milliseconds
	ToolName    string `json:"tool_name"`    // "sql_executor", "python_tool", etc.
	Description string `json:"description"`  // Human-readable description
	Input       string `json:"input"`        // SQL query or Python code
	Output      string `json:"output"`       // Tool output (for reference)
	ChartType   string `json:"chart_type"`   // "echarts", "image", "table", "csv"
	ChartData   string `json:"chart_data"`   // Chart data if generated
}

// UnmarshalJSON implements custom unmarshaling to handle both new (int64) and old (time.Time string) formats
func (as *AnalysisStep) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with Timestamp as interface{} to handle both formats
	type Alias AnalysisStep
	aux := &struct {
		Timestamp interface{} `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(as),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle Timestamp field conversion
	switch v := aux.Timestamp.(type) {
	case float64:
		// Already int64 (JSON numbers are float64)
		as.Timestamp = int64(v)
	case string:
		// Old time.Time format - parse and convert to Unix milliseconds
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			as.Timestamp = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", v); err == nil {
			as.Timestamp = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05Z07:00", v); err == nil {
			as.Timestamp = t.UnixMilli()
		} else {
			// If parsing fails, use current time
			as.Timestamp = time.Now().UnixMilli()
		}
	case nil:
		// No Timestamp field - use current time
		as.Timestamp = time.Now().UnixMilli()
	default:
		as.Timestamp = time.Now().UnixMilli()
	}

	return nil
}

// TableSchema represents the schema of a table
type ReplayTableSchema struct {
	TableName string   `json:"table_name"`
	Columns   []string `json:"columns"`
}

// AnalysisRecording represents a complete recorded analysis workflow
type AnalysisRecording struct {
	RecordingID     string              `json:"recording_id"`
	Title           string              `json:"title"`
	Description     string              `json:"description"`
	CreatedAt       int64               `json:"created_at"`        // Unix timestamp in milliseconds
	SourceID        string              `json:"source_id"`        // Original data source ID
	SourceName      string              `json:"source_name"`      // Original data source name
	SourceSchema    []ReplayTableSchema `json:"source_schema"`    // Original table schemas
	Steps           []AnalysisStep      `json:"steps"`            // Sequence of analysis steps
	LLMConversation []ConversationTurn  `json:"llm_conversation"` // LLM chat history for context
}

// UnmarshalJSON implements custom unmarshaling to handle both new (int64) and old (time.Time string) formats
func (ar *AnalysisRecording) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with CreatedAt as interface{} to handle both formats
	type Alias AnalysisRecording
	aux := &struct {
		CreatedAt interface{} `json:"created_at"`
		*Alias
	}{
		Alias: (*Alias)(ar),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle CreatedAt field conversion
	switch v := aux.CreatedAt.(type) {
	case float64:
		// Already int64 (JSON numbers are float64)
		ar.CreatedAt = int64(v)
	case string:
		// Old time.Time format - parse and convert to Unix milliseconds
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			ar.CreatedAt = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", v); err == nil {
			ar.CreatedAt = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05Z07:00", v); err == nil {
			ar.CreatedAt = t.UnixMilli()
		} else {
			// If parsing fails, use current time
			ar.CreatedAt = time.Now().UnixMilli()
		}
	case nil:
		// No CreatedAt field - use current time
		ar.CreatedAt = time.Now().UnixMilli()
	default:
		ar.CreatedAt = time.Now().UnixMilli()
	}

	return nil
}

// ConversationTurn represents a single LLM conversation turn
type ConversationTurn struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"` // Message content
}

// FieldMapping represents a mapping between old and new field names
type FieldMapping struct {
	OldField string `json:"old_field"`
	NewField string `json:"new_field"`
}

// TableMapping represents field mappings for a table
type TableMapping struct {
	SourceTable string         `json:"source_table"`
	TargetTable string         `json:"target_table"`
	Mappings    []FieldMapping `json:"mappings"`
}

// ReplayConfig contains configuration for replaying an analysis
type ReplayConfig struct {
	RecordingID      string         `json:"recording_id"`
	TargetSourceID   string         `json:"target_source_id"`
	TargetSourceName string         `json:"target_source_name"`
	TableMappings    []TableMapping `json:"table_mappings"`
	AutoFixFields    bool           `json:"auto_fix_fields"` // Whether to auto-fix field mismatches
	MaxFieldDiff     int            `json:"max_field_diff"`  // Max allowed field differences (default: 2)
}

// ReplayResult represents the result of a replay operation
type ReplayResult struct {
	Success        bool                   `json:"success"`
	StepsExecuted  int                    `json:"steps_executed"`
	StepsFailed    int                    `json:"steps_failed"`
	StepResults    []StepResult           `json:"step_results"`
	FieldMappings  []TableMapping         `json:"field_mappings"` // Applied field mappings
	GeneratedFiles []string               `json:"generated_files"`
	ErrorMessage   string                 `json:"error_message"`
	Charts         []map[string]interface{} `json:"charts"` // Generated charts
}

// StepResult represents the result of executing a single step
type StepResult struct {
	StepID       int    `json:"step_id"`
	Success      bool   `json:"success"`
	Output       string `json:"output"`
	ErrorMessage string `json:"error_message"`
	ChartData    string `json:"chart_data"`
	Modified     bool   `json:"modified"` // Whether the step was modified during replay
}
