package agent

// ProgressUpdate represents a progress update during analysis
type ProgressUpdate struct {
	Stage      string `json:"stage"`       // "initializing", "schema", "query", "analysis", "visualization", "complete"
	Progress   int    `json:"progress"`    // 0-100
	Message    string `json:"message"`     // Human-readable message or i18n key like "progress.loading_schema"
	Step       int    `json:"step"`        // Current step number
	Total      int    `json:"total"`       // Total steps
	ToolName   string `json:"tool_name"`   // Tool being executed (optional)
	ToolOutput string `json:"tool_output"` // Streaming tool output (optional)
}

// StreamUpdate represents a streaming content update
type StreamUpdate struct {
	Type    string `json:"type"`    // "tool_start", "tool_output", "content"
	Tool    string `json:"tool"`    // Tool name (if applicable)
	Content string `json:"content"` // Partial content
}

// ProgressCallback is a function that receives progress updates
type ProgressCallback func(update ProgressUpdate)

// StreamCallback is a function that receives streaming content
type StreamCallback func(update StreamUpdate)

// Pre-defined progress stages
const (
	StageInitializing  = "initializing"
	StageSchema        = "schema"
	StageQuery         = "query"
	StageAnalysis      = "analysis"
	StageVisualization = "visualization"
	StageExporting     = "exporting"
	StageSearching     = "searching"
	StageComplete      = "complete"
)

// NewProgressUpdate creates a new progress update
func NewProgressUpdate(stage string, progress int, message string, step, total int) ProgressUpdate {
	return ProgressUpdate{
		Stage:    stage,
		Progress: progress,
		Message:  message,
		Step:     step,
		Total:    total,
	}
}

// NewToolProgressUpdate creates a progress update for a specific tool execution
func NewToolProgressUpdate(stage string, progress int, message string, step, total int, toolName string) ProgressUpdate {
	return ProgressUpdate{
		Stage:    stage,
		Progress: progress,
		Message:  message,
		Step:     step,
		Total:    total,
		ToolName: toolName,
	}
}

// NewStreamUpdate creates a new stream update
func NewStreamUpdate(updateType, tool, content string) StreamUpdate {
	return StreamUpdate{
		Type:    updateType,
		Tool:    tool,
		Content: content,
	}
}

// ToolProgressMapping maps tool names to their stage, progress, and i18n message key
var ToolProgressMapping = map[string]struct {
	Stage    string
	Progress int
	Message  string
}{
	"get_data_source_context": {StageSchema, 25, "progress.loading_schema"},
	"execute_sql":             {StageQuery, 40, "progress.executing_sql"},
	"python_executor":         {StageAnalysis, 60, "progress.running_python"},
	"web_search":              {StageSearching, 50, "progress.web_searching"},
	"web_fetch":               {StageSearching, 55, "progress.fetching_page"},
	"export_data":             {StageExporting, 70, "progress.exporting_data"},
	"mcp_service":             {StageAnalysis, 50, "progress.calling_mcp"},
	"get_local_time":          {StageAnalysis, 30, "progress.ai_processing"},
	"get_device_location":     {StageAnalysis, 30, "progress.ai_processing"},
	"uapi_search":             {StageSearching, 50, "progress.web_searching"},
}
