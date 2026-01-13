package agent

import "time"

// AnalysisStep represents a single step in an analysis workflow
type AnalysisStep struct {
	StepID      int       `json:"step_id"`
	Timestamp   time.Time `json:"timestamp"`
	ToolName    string    `json:"tool_name"`    // "sql_executor", "python_tool", etc.
	Description string    `json:"description"`  // Human-readable description
	Input       string    `json:"input"`        // SQL query or Python code
	Output      string    `json:"output"`       // Tool output (for reference)
	ChartType   string    `json:"chart_type"`   // "echarts", "image", "table", "csv"
	ChartData   string    `json:"chart_data"`   // Chart data if generated
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
	CreatedAt       time.Time           `json:"created_at"`
	SourceID        string              `json:"source_id"`        // Original data source ID
	SourceName      string              `json:"source_name"`      // Original data source name
	SourceSchema    []ReplayTableSchema `json:"source_schema"`    // Original table schemas
	Steps           []AnalysisStep      `json:"steps"`            // Sequence of analysis steps
	LLMConversation []ConversationTurn  `json:"llm_conversation"` // LLM chat history for context
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
