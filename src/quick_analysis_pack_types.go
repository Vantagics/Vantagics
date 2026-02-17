package main

// ---------------------------------------------------------------------------
// QAP module-wide constants
// ---------------------------------------------------------------------------

const (
	// File format identifiers
	qapFileType      = "VantageData_QuickAnalysisPack"
	qapFormatVersion = "1.0"
	qapSubDir        = "qap" // sub-directory under DataCacheDir for .qap files

	// Execution record types (as stored in executions.json)
	execTypeSQL    = "sql"
	execTypePython = "python"

	// Pack step types
	stepTypeSQL    = "sql_query"
	stepTypePython = "python_code"

	// Log tag prefixes
	logTagPreview = "[QAP-PREVIEW]"
	logTagExport  = "[QAP-EXPORT]"
	logTagImport  = "[QAP-IMPORT]"
	logTagExecute = "[QAP-EXECUTE]"
	logTagReexec  = "[QAP-REEXECUTE]"
)

// ---------------------------------------------------------------------------
// Pack data types
// ---------------------------------------------------------------------------

// QuickAnalysisPack represents a complete quick analysis pack with metadata, schema requirements, and executable steps.
// The pack is serialized as JSON inside a ZIP container (.qap file).
type QuickAnalysisPack struct {
	FileType           string            `json:"file_type"`           // "VantageData_QuickAnalysisPack"
	FormatVersion      string            `json:"format_version"`      // "1.0"
	Metadata           PackMetadata      `json:"metadata"`
	SchemaRequirements []PackTableSchema `json:"schema_requirements"`
	ExecutableSteps    []PackStep        `json:"executable_steps"`
}

// PackMetadata contains descriptive information about the quick analysis pack.
type PackMetadata struct {
	PackName    string `json:"pack_name"`                 // Analysis scenario name (user input)
	Author      string `json:"author"`                    // Creator name (user input)
	CreatedAt   string `json:"created_at"`                // RFC3339 formatted timestamp
	SourceName  string `json:"source_name"`               // Original data source name
	Description string `json:"description"`               // Optional description
	ListingID   int64  `json:"listing_id,omitempty"`      // Marketplace listing ID (0 for local packs)
}


// PackTableSchema represents the schema of a single table in the data source.
type PackTableSchema struct {
	TableName string           `json:"table_name"`
	Columns   []PackColumnInfo `json:"columns"`
}

// PackColumnInfo represents a single column's name and type within a table.
type PackColumnInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// PackStep represents a single executable step (SQL query or Python code) in the analysis pack.
type PackStep struct {
	StepID      int    `json:"step_id"`
	StepType    string `json:"step_type"`                       // "sql_query" or "python_code"
	Code        string `json:"code"`
	Description string `json:"description"`
	UserRequest string `json:"user_request,omitempty"`          // Original user request that triggered this step (for display as "分析请求")
	DependsOn   []int  `json:"depends_on,omitempty"`
	SourceTool  string `json:"source_tool,omitempty"`           // "query_and_chart" if chart code needs DataFrame injection
	PairedSQLStepID int `json:"paired_sql_step_id,omitempty"`  // StepID of the paired SQL step (for query_and_chart chart code)
	EChartsConfigs []string `json:"echarts_configs,omitempty"`  // ECharts JSON configs from original LLM response (for replay)
}

// SchemaValidationResult contains the result of comparing pack schema requirements against a target data source.
type SchemaValidationResult struct {
	Compatible       bool              `json:"compatible"`
	TableCountMatch  bool              `json:"table_count_match"`
	SourceTableCount int               `json:"source_table_count"`
	TargetTableCount int               `json:"target_table_count"`
	MissingTables    []string          `json:"missing_tables"`
	MissingColumns   []MissingColumnInfo `json:"missing_columns"`
	ExtraTables      []string          `json:"extra_tables"`
}

// MissingColumnInfo identifies a column that is required by the pack but missing in the target data source.
type MissingColumnInfo struct {
	TableName  string `json:"table_name"`
	ColumnName string `json:"column_name"`
}

// PackLoadResult contains the result of loading a quick analysis pack file, including the parsed pack,
// schema validation result, and encryption status.
type PackLoadResult struct {
	Pack             *QuickAnalysisPack      `json:"pack"`
	Validation       *SchemaValidationResult  `json:"validation"`
	IsEncrypted      bool                     `json:"is_encrypted"`
	NeedsPassword    bool                     `json:"needs_password"`
	FilePath         string                   `json:"file_path"`
	HasPythonSteps   bool                     `json:"has_python_steps"`
	PythonConfigured bool                     `json:"python_configured"`
}

// ---------------------------------------------------------------------------
// Parsed execution types (shared structure of executions.json)
// ---------------------------------------------------------------------------

// parsedExecutionsMap is the deserialized form of executions.json.
// Key = messageID (request identifier), Value = the entry with user request and execution records.
type parsedExecutionsMap map[string]parsedExecutionEntry

type parsedExecutionEntry struct {
	UserRequest string                  `json:"user_request"`
	Executions  []parsedExecutionRecord `json:"executions"`
	Timestamp   int64                   `json:"timestamp"`
}

type parsedExecutionRecord struct {
	Code            string `json:"code"`
	Type            string `json:"type"`
	StepDescription string `json:"step_description"`
	UserRequest     string `json:"user_request"`
	Timestamp       int64  `json:"timestamp"`
	Success         bool   `json:"success"`
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// execTypeToStepType maps execution record type to pack step type.
// Returns empty string for unrecognized types.
func execTypeToStepType(execType string) string {
	switch execType {
	case execTypeSQL:
		return stepTypeSQL
	case execTypePython:
		return stepTypePython
	default:
		return ""
	}
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

// truncStr truncates a string to maxLen characters, appending "..." if truncated.
func truncStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// renumberSteps assigns sequential StepIDs and rebuilds dependency chains,
// pairing query_and_chart chart steps with their preceding SQL step.
func renumberSteps(steps []PackStep) {
	lastSQLStepID := 0
	for i := range steps {
		steps[i].StepID = i + 1
		switch {
		case steps[i].StepType == stepTypeSQL:
			lastSQLStepID = i + 1
			steps[i].DependsOn = buildDependsOn(i + 1)
		case steps[i].SourceTool == "query_and_chart" && lastSQLStepID > 0:
			steps[i].DependsOn = []int{lastSQLStepID}
			steps[i].PairedSQLStepID = lastSQLStepID
		default:
			steps[i].DependsOn = buildDependsOn(i + 1)
		}
	}
}
