package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// StartAnalysisTool allows the free chat agent to list data sources and start analysis sessions
type StartAnalysisTool struct {
	logFunc           func(string)
	dsService         *DataSourceService
	emitStartChatFunc func(dataSourceID, dataSourceName string) // callback to emit start-new-chat event

	// Cache to avoid redundant LoadDataSources calls between list→start sequence
	mu          sync.Mutex
	cachedDS    []DataSource
	cachedAt    time.Time
	cacheTTL    time.Duration
}

// NewStartAnalysisTool creates a new start analysis tool
func NewStartAnalysisTool(logFunc func(string), dsService *DataSourceService, emitStartChatFunc func(string, string)) *StartAnalysisTool {
	return &StartAnalysisTool{
		logFunc:           logFunc,
		dsService:         dsService,
		emitStartChatFunc: emitStartChatFunc,
		cacheTTL:          30 * time.Second, // Cache valid for 30s (covers list→start sequence)
	}
}

type startAnalysisInput struct {
	Action       string `json:"action"`
	DataSourceID string `json:"data_source_id,omitempty"`
}

func (t *StartAnalysisTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "start_datasource_analysis",
		Desc: `List available data sources or start a data analysis session.

Use this tool when the user wants to analyze data, mentions analyzing a specific dataset/data source, or asks about available data sources.

Actions:
- "list": List all available data sources (name, type, id). Use this first to find the data source ID.
- "start": Start a new analysis session for a specific data source. Requires data_source_id.

Examples of user intent that should trigger this tool:
- "分析销售数据" / "帮我分析一下销售数据" / "analyze sales data"
- "我想看看用户行为数据" / "look at user behavior data"
- "有哪些数据源" / "what data sources are available"
- "分析一下xxx" where xxx is a data source name`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "Action to perform: 'list' to list available data sources, 'start' to start analysis on a specific data source",
				Required: true,
			},
			"data_source_id": {
				Type:     schema.String,
				Desc:     "The ID of the data source to analyze (required when action is 'start'). Get this from the 'list' action first.",
				Required: false,
			},
		}),
	}, nil
}

func (t *StartAnalysisTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	t.log("[START-ANALYSIS-TOOL] Invoked with args: %s", argumentsInJSON)

	var input startAnalysisInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse input: %v", err)
	}

	switch strings.ToLower(input.Action) {
	case "list":
		return t.listDataSources()
	case "start":
		return t.startAnalysis(input.DataSourceID)
	default:
		return "", fmt.Errorf("unknown action: %s (use 'list' or 'start')", input.Action)
	}
}

// loadDataSourcesCached returns cached data sources if still valid, otherwise reloads
func (t *StartAnalysisTool) loadDataSourcesCached() ([]DataSource, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cachedDS != nil && time.Since(t.cachedAt) < t.cacheTTL {
		return t.cachedDS, nil
	}

	sources, err := t.dsService.LoadDataSources()
	if err != nil {
		return nil, err
	}
	t.cachedDS = sources
	t.cachedAt = time.Now()
	return sources, nil
}

func (t *StartAnalysisTool) listDataSources() (string, error) {
	if t.dsService == nil {
		return `{"error": "data source service not available"}`, nil
	}

	sources, err := t.loadDataSourcesCached()
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to load data sources: %s"}`, err.Error()), nil
	}

	if len(sources) == 0 {
		return `{"data_sources": [], "message": "No data sources available. Please add a data source first."}`, nil
	}

	type dsSummary struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	summaries := make([]dsSummary, 0, len(sources))
	for _, ds := range sources {
		summaries = append(summaries, dsSummary{
			ID:   ds.ID,
			Name: ds.Name,
			Type: ds.Type,
		})
	}

	result, _ := json.Marshal(map[string]interface{}{
		"data_sources": summaries,
		"count":        len(summaries),
	})

	t.log("[START-ANALYSIS-TOOL] Listed %d data sources", len(summaries))
	return string(result), nil
}

func (t *StartAnalysisTool) startAnalysis(dataSourceID string) (string, error) {
	if dataSourceID == "" {
		return `{"error": "data_source_id is required. Use action='list' first to get available data source IDs."}`, nil
	}

	if t.emitStartChatFunc == nil {
		return `{"error": "start chat function not available"}`, nil
	}

	// Look up the data source name (uses cache from prior list call)
	sources, err := t.loadDataSourcesCached()
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to load data sources: %s"}`, err.Error()), nil
	}

	var dsName string
	for _, ds := range sources {
		if ds.ID == dataSourceID {
			dsName = ds.Name
			break
		}
	}
	if dsName == "" {
		return fmt.Sprintf(`{"error": "data source not found: %s"}`, dataSourceID), nil
	}

	// Emit start-new-chat event to trigger the existing frontend analysis flow
	t.emitStartChatFunc(dataSourceID, dsName)

	t.log("[START-ANALYSIS-TOOL] Emitted start-new-chat for data source %s (%s)", dataSourceID, dsName)

	result, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Analysis session for '%s' is being started. The user will see the analysis in a new chat thread.", dsName),
	})
	return string(result), nil
}

func (t *StartAnalysisTool) log(format string, args ...interface{}) {
	if t.logFunc != nil {
		t.logFunc(fmt.Sprintf(format, args...))
	}
}
