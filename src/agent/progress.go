package agent

// ProgressUpdate represents a progress update during analysis
type ProgressUpdate struct {
	Stage      string `json:"stage"`       // "initializing", "schema", "query", "analysis", "visualization", "complete"
	Progress   int    `json:"progress"`    // 0-100
	Message    string `json:"message"`     // Human-readable message like "Executing SQL query..."
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

// Pre-defined progress stages with typical progress percentages
const (
	StageInitializing   = "initializing"
	StageSchema         = "schema"
	StageQuery          = "query"
	StageAnalysis       = "analysis"
	StageVisualization  = "visualization"
	StageComplete       = "complete"
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

// NewStreamUpdate creates a new stream update
func NewStreamUpdate(updateType, tool, content string) StreamUpdate {
	return StreamUpdate{
		Type:    updateType,
		Tool:    tool,
		Content: content,
	}
}
