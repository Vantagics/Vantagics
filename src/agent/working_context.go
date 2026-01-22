package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WorkingContext captures the current UI state and analysis context
// This enables zero-redundancy prompts like "删除这些异常点" where "这些" 
// refers to the currently visible outliers
type WorkingContext struct {
	SessionID        string                 `json:"session_id"`
	ActiveChart      *ChartContext          `json:"active_chart,omitempty"`
	ActiveFilters    map[string]string      `json:"active_filters,omitempty"`
	RecentOperations []Operation            `json:"recent_operations"`
	Highlights       []DataHighlight        `json:"highlights,omitempty"`
	LastUpdate       int64                  `json:"last_update"`
}

// ChartContext describes the currently visible chart
type ChartContext struct {
	Type        string                 `json:"type"`         // "echarts", "image", "table", "csv"
	ChartConfig map[string]interface{} `json:"config,omitempty"` // Chart configuration
	DataSummary DataStatistics         `json:"data_summary"` // Statistical summary
}

// Operation records a user interaction
type Operation struct {
	Action    string `json:"action"`    // "filter", "drilldown", "sort", "select", "highlight"
	Target    string `json:"target"`    // e.g., "city", "sales_amount", "chart_type"
	Value     string `json:"value"`     // e.g., "北京", "DESC", "bar"
	Timestamp int64  `json:"timestamp"`
}

// DataHighlight marks important data points in the current view
type DataHighlight struct {
	Type        string   `json:"type"`        // "outlier", "anomaly", "trend", "selection"
	Description string   `json:"description"` // Human-readable description
	DataPoints  []string `json:"data_points,omitempty"` // Coordinates or IDs
}

// DataStatistics provides statistical summary of current view
type DataStatistics struct {
	RowCount   int                `json:"row_count"`
	Aggregates map[string]float64 `json:"aggregates,omitempty"` // avg, max, min, sum
	Outliers   []OutlierPoint     `json:"outliers,omitempty"`
}

// OutlierPoint represents an anomalous data point
type OutlierPoint struct {
	Label      string                 `json:"label"`
	Coordinates map[string]interface{} `json:"coordinates"` // x, y, or dimension values
	Value      float64                `json:"value"`
}

// WorkingContextManager manages working context per session
type WorkingContextManager struct {
	dataDir      string
	contexts     map[string]*WorkingContext // sessionID -> context
	schemaCache  map[string]string          // dataSourceID -> cached schema
	cacheExpiry  map[string]int64           // dataSourceID -> expiry timestamp
	mu           sync.RWMutex
}

// NewWorkingContextManager creates a new manager
func NewWorkingContextManager(dataDir string) *WorkingContextManager {
	contextDir := filepath.Join(dataDir, "working_contexts")
	_ = os.MkdirAll(contextDir, 0755)

	return &WorkingContextManager{
		dataDir:     dataDir,
		contexts:    make(map[string]*WorkingContext),
		schemaCache: make(map[string]string),
		cacheExpiry: make(map[string]int64),
	}
}

// UpdateContext updates the working context for a session
func (m *WorkingContextManager) UpdateContext(sessionID string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create context
	ctx, exists := m.contexts[sessionID]
	if !exists {
		ctx = &WorkingContext{
			SessionID:        sessionID,
			ActiveFilters:    make(map[string]string),
			RecentOperations: []Operation{},
			Highlights:       []DataHighlight{},
			LastUpdate:       time.Now().Unix(),
		}
		m.contexts[sessionID] = ctx
	}

	// Apply updates
	if activeChart, ok := updates["active_chart"].(map[string]interface{}); ok {
		ctx.ActiveChart = m.parseChartContext(activeChart)
	}

	if filters, ok := updates["active_filters"].(map[string]string); ok {
		ctx.ActiveFilters = filters
	}

	if operation, ok := updates["operation"].(map[string]interface{}); ok {
		op := Operation{
			Action:    getStringField(operation, "action"),
			Target:    getStringField(operation, "target"),
			Value:     getStringField(operation, "value"),
			Timestamp: time.Now().Unix(),
		}
		ctx.RecentOperations = append(ctx.RecentOperations, op)
		
		// Keep only last 5 operations
		if len(ctx.RecentOperations) > 5 {
			ctx.RecentOperations = ctx.RecentOperations[len(ctx.RecentOperations)-5:]
		}
	}

	if highlights, ok := updates["highlights"].([]interface{}); ok {
		ctx.Highlights = m.parseHighlights(highlights)
	}

	ctx.LastUpdate = time.Now().Unix()

	// Persist to disk
	return m.saveContext(sessionID, ctx)
}

// GetContext retrieves the working context for a session
func (m *WorkingContextManager) GetContext(sessionID string) *WorkingContext {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, exists := m.contexts[sessionID]
	if !exists {
		// Try loading from disk
		if loaded := m.loadContext(sessionID); loaded != nil {
			m.mu.RUnlock()
			m.mu.Lock()
			m.contexts[sessionID] = loaded
			m.mu.Unlock()
			m.mu.RLock()
			return loaded
		}
		return nil
	}

	return ctx
}

// FormatForPrompt converts working context to LLM-readable text
func (ctx *WorkingContext) FormatForPrompt() string {
	if ctx == nil {
		return ""
	}

	var result string
	result += "\n=== Current Analysis Context ===\n"

	// Active chart info
	if ctx.ActiveChart != nil {
		result += fmt.Sprintf("Active Chart: %s\n", ctx.ActiveChart.Type)
		if ctx.ActiveChart.DataSummary.RowCount > 0 {
			result += fmt.Sprintf("  - Data Points: %d rows\n", ctx.ActiveChart.DataSummary.RowCount)
		}
		if len(ctx.ActiveChart.DataSummary.Aggregates) > 0 {
			result += "  - Statistics:\n"
			for metric, value := range ctx.ActiveChart.DataSummary.Aggregates {
				result += fmt.Sprintf("    • %s: %.2f\n", metric, value)
			}
		}
		if len(ctx.ActiveChart.DataSummary.Outliers) > 0 {
			result += fmt.Sprintf("  - Outliers Detected: %d points\n", len(ctx.ActiveChart.DataSummary.Outliers))
			for i, outlier := range ctx.ActiveChart.DataSummary.Outliers {
				if i >= 3 {
					result += fmt.Sprintf("    ... and %d more\n", len(ctx.ActiveChart.DataSummary.Outliers)-3)
					break
				}
				result += fmt.Sprintf("    • %s: %.2f\n", outlier.Label, outlier.Value)
			}
		}
	}

	// Active filters
	if len(ctx.ActiveFilters) > 0 {
		result += "Active Filters:\n"
		for dim, value := range ctx.ActiveFilters {
			result += fmt.Sprintf("  - %s = %s\n", dim, value)
		}
	}

	// Recent operations
	if len(ctx.RecentOperations) > 0 {
		result += "Recent User Actions:\n"
		for i := len(ctx.RecentOperations) - 1; i >= 0; i-- {
			op := ctx.RecentOperations[i]
			result += fmt.Sprintf("  %d. %s on '%s'", len(ctx.RecentOperations)-i, op.Action, op.Target)
			if op.Value != "" {
				result += fmt.Sprintf(": %s", op.Value)
			}
			result += "\n"
		}
	}

	// Highlights
	if len(ctx.Highlights) > 0 {
		result += "Highlighted Data:\n"
		for _, highlight := range ctx.Highlights {
			result += fmt.Sprintf("  - %s: %s\n", highlight.Type, highlight.Description)
			if len(highlight.DataPoints) > 0 && len(highlight.DataPoints) <= 3 {
				result += fmt.Sprintf("    Points: %v\n", highlight.DataPoints)
			}
		}
	}

	result += "=== End Context ===\n"

	return result
}

// ClearContext removes the working context for a session
func (m *WorkingContextManager) ClearContext(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.contexts, sessionID)
	
	// Delete from disk
	path := m.getContextPath(sessionID)
	_ = os.Remove(path)
}

// CacheSchema stores schema information for a data source
func (m *WorkingContextManager) CacheSchema(dataSourceID string, schema string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.schemaCache[dataSourceID] = schema
	// Cache expires after 5 minutes
	m.cacheExpiry[dataSourceID] = time.Now().Add(5 * time.Minute).Unix()
}

// GetCachedSchema retrieves cached schema if available and not expired
func (m *WorkingContextManager) GetCachedSchema(dataSourceID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	schema, exists := m.schemaCache[dataSourceID]
	if !exists {
		return "", false
	}

	// Check expiry
	expiry, hasExpiry := m.cacheExpiry[dataSourceID]
	if hasExpiry && time.Now().Unix() > expiry {
		// Expired, remove from cache
		m.mu.RUnlock()
		m.mu.Lock()
		delete(m.schemaCache, dataSourceID)
		delete(m.cacheExpiry, dataSourceID)
		m.mu.Unlock()
		m.mu.RLock()
		return "", false
	}

	return schema, true
}

// ClearSchemaCache clears all cached schemas
func (m *WorkingContextManager) ClearSchemaCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.schemaCache = make(map[string]string)
	m.cacheExpiry = make(map[string]int64)
}

// Internal helper methods

func (m *WorkingContextManager) parseChartContext(data map[string]interface{}) *ChartContext {
	ctx := &ChartContext{
		Type:        getStringField(data, "type"),
		ChartConfig: make(map[string]interface{}),
		DataSummary: DataStatistics{
			Aggregates: make(map[string]float64),
			Outliers:   []OutlierPoint{},
		},
	}

	if config, ok := data["config"].(map[string]interface{}); ok {
		ctx.ChartConfig = config
	}

	if summary, ok := data["data_summary"].(map[string]interface{}); ok {
		if rowCount, ok := summary["row_count"].(float64); ok {
			ctx.DataSummary.RowCount = int(rowCount)
		}
		if agg, ok := summary["aggregates"].(map[string]interface{}); ok {
			for k, v := range agg {
				if fv, ok := v.(float64); ok {
					ctx.DataSummary.Aggregates[k] = fv
				}
			}
		}
		if outliers, ok := summary["outliers"].([]interface{}); ok {
			for _, o := range outliers {
				if om, ok := o.(map[string]interface{}); ok {
					outlier := OutlierPoint{
						Label:       getStringField(om, "label"),
						Coordinates: om,
					}
					if val, ok := om["value"].(float64); ok {
						outlier.Value = val
					}
					ctx.DataSummary.Outliers = append(ctx.DataSummary.Outliers, outlier)
				}
			}
		}
	}

	return ctx
}

func (m *WorkingContextManager) parseHighlights(data []interface{}) []DataHighlight {
	highlights := []DataHighlight{}
	
	for _, item := range data {
		if h, ok := item.(map[string]interface{}); ok {
			highlight := DataHighlight{
				Type:        getStringField(h, "type"),
				Description: getStringField(h, "description"),
			}
			
			if points, ok := h["data_points"].([]interface{}); ok {
				for _, p := range points {
					if ps, ok := p.(string); ok {
						highlight.DataPoints = append(highlight.DataPoints, ps)
					}
				}
			}
			
			highlights = append(highlights, highlight)
		}
	}
	
	return highlights
}

func (m *WorkingContextManager) getContextPath(sessionID string) string {
	return filepath.Join(m.dataDir, "working_contexts", sessionID+".json")
}

func (m *WorkingContextManager) saveContext(sessionID string, ctx *WorkingContext) error {
	path := m.getContextPath(sessionID)
	
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (m *WorkingContextManager) loadContext(sessionID string) *WorkingContext {
	path := m.getContextPath(sessionID)
	
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var ctx WorkingContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil
	}

	return &ctx
}

func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
