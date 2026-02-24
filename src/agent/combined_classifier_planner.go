package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Cache configuration constants
const (
	combinedCacheTTL     = 5 * time.Minute // Cache entries expire after 5 minutes
	combinedCacheMaxSize = 100             // Maximum number of cached entries
)

// CombinedClassifierPlanner merges request classification and analysis planning
// into a single LLM call, eliminating the overhead of two sequential calls.
type CombinedClassifierPlanner struct {
	chatModel model.ChatModel
	logger    func(string)
	cache     map[string]*CombinedResult
	cacheMu   sync.RWMutex
}

// CombinedResult holds both classification and planning results from a single LLM call
type CombinedResult struct {
	// Classification fields
	RequestType        string   `json:"request_type"`
	NeedsVisualization bool     `json:"needs_visualization"`
	NeedsDataExport    bool     `json:"needs_data_export"`
	Confidence         float64  `json:"confidence"`
	Reasoning          string   `json:"reasoning"`
	SuggestedChartType string   `json:"suggested_chart_type"`

	// Planning fields
	TaskType       string     `json:"task_type"`
	Complexity     string     `json:"complexity"`
	IsQuickPath    bool       `json:"is_quick_path"`
	QuickPathCode  string     `json:"quick_path_code,omitempty"`
	NeedsSchema    bool       `json:"needs_schema"`
	NeedsSQL       bool       `json:"needs_sql"`
	NeedsPython    bool       `json:"needs_python"`
	NeedsWebSearch bool       `json:"needs_web_search"`
	OutputFormat   string     `json:"output_format"`
	EstimatedCalls int        `json:"estimated_calls"`
	Steps          []PlanStep `json:"steps"`

	// Cache metadata
	CachedAt time.Time `json:"-"`
}

// ToClassificationResult converts to the existing ClassificationResult type
func (r *CombinedResult) ToClassificationResult() *ClassificationResult {
	return &ClassificationResult{
		RequestType:        r.RequestType,
		NeedsVisualization: r.NeedsVisualization,
		NeedsDataExport:    r.NeedsDataExport,
		Confidence:         r.Confidence,
		Reasoning:          r.Reasoning,
		SuggestedChartType: r.SuggestedChartType,
		SuggestedOutputs:   r.suggestedOutputs(),
		CachedAt:           r.CachedAt,
	}
}

func (r *CombinedResult) suggestedOutputs() []string {
	var outputs []string
	if r.NeedsVisualization {
		outputs = append(outputs, "chart")
	}
	if r.NeedsDataExport {
		outputs = append(outputs, "excel")
	}
	outputs = append(outputs, "table", "insight")
	return outputs
}

// ToAnalysisPlan converts to the existing AnalysisPlan type
func (r *CombinedResult) ToAnalysisPlan() *AnalysisPlan {
	plan := &AnalysisPlan{
		TaskType:       r.TaskType,
		Complexity:     r.Complexity,
		IsQuickPath:    r.IsQuickPath,
		QuickPathCode:  r.QuickPathCode,
		NeedsSchema:    r.NeedsSchema,
		NeedsSQL:       r.NeedsSQL,
		NeedsPython:    r.NeedsPython,
		NeedsWebSearch: r.NeedsWebSearch,
		OutputFormat:   r.OutputFormat,
		EstimatedCalls: r.EstimatedCalls,
		Steps:          r.Steps,
	}

	// Map request type
	switch r.RequestType {
	case "consultation":
		plan.RequestType = RequestTypeConsultation
		plan.SchemaLevel = SchemaLevelBasic
	case "web_search":
		plan.RequestType = RequestTypeWebSearch
		plan.SchemaLevel = SchemaLevelBasic
	case "calculation":
		plan.RequestType = RequestTypeCalculation
		plan.SchemaLevel = SchemaLevelBasic
	case "visualization":
		plan.RequestType = RequestTypeVisualization
		plan.SchemaLevel = SchemaLevelDetailed
	default:
		plan.RequestType = RequestTypeDataQuery
		plan.SchemaLevel = SchemaLevelDetailed
	}

	return plan
}

// NewCombinedClassifierPlanner creates a new combined classifier+planner
func NewCombinedClassifierPlanner(chatModel model.ChatModel, logger func(string)) *CombinedClassifierPlanner {
	return &CombinedClassifierPlanner{
		chatModel: chatModel,
		logger:    logger,
		cache:     make(map[string]*CombinedResult),
	}
}

func (c *CombinedClassifierPlanner) log(msg string) {
	if c.logger != nil {
		c.logger(msg)
	}
}

// ClassifyAndPlan performs classification and planning in a single LLM call.
// This replaces the previous two-call pattern (RequestTypeClassifier + AnalysisPlanner).
func (c *CombinedClassifierPlanner) ClassifyAndPlan(ctx context.Context, userQuery string, dataSourceInfo string) (*CombinedResult, error) {
	if userQuery == "" {
		return c.quickResult("calculation", "trivial"), nil
	}

	// 1. Check cache using the new TTL-aware method
	cacheKey := fmt.Sprintf("%s|%s", userQuery, dataSourceInfo)
	if cached, ok := c.getCachedResult(cacheKey); ok {
		c.log("[COMBINED] Using cached classify+plan result")
		return cached, nil
	}

	// 2. Try quick path detection without LLM
	queryLower := strings.ToLower(userQuery)
	if quick := c.detectQuickPath(queryLower); quick != nil {
		c.log(fmt.Sprintf("[COMBINED] Quick path detected: %s", quick.TaskType))
		return quick, nil
	}

	// 3. Try keyword-based classification for obvious cases (skip LLM)
	if result := c.tryKeywordClassification(queryLower, dataSourceInfo); result != nil {
		c.log(fmt.Sprintf("[COMBINED] Keyword classification: type=%s, skipping LLM", result.RequestType))
		c.cacheResult(cacheKey, result)
		return result, nil
	}

	// 4. Single LLM call for combined classification + planning
	startTime := time.Now()
	c.log("[COMBINED] Calling LLM for combined classify+plan...")

	prompt := c.buildCombinedPrompt(userQuery, dataSourceInfo)

	messages := []*schema.Message{
		{Role: schema.System, Content: "You are a request classification and task planning expert. Output only valid JSON."},
		{Role: schema.User, Content: prompt},
	}

	resp, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		c.log(fmt.Sprintf("[COMBINED] LLM call failed: %v, using fallback", err))
		return c.fallbackResult(queryLower, dataSourceInfo), nil
	}

	// Parse response
	result := &CombinedResult{}
	content := strings.TrimSpace(resp.Content)
	content = extractJSON(content)

	if err := json.Unmarshal([]byte(content), result); err != nil {
		c.log(fmt.Sprintf("[COMBINED] Failed to parse response: %v, using fallback", err))
		return c.fallbackResult(queryLower, dataSourceInfo), nil
	}

	result.CachedAt = time.Now()
	c.cacheResult(cacheKey, result)

	c.log(fmt.Sprintf("[COMBINED] Classify+plan complete in %v: type=%s, complexity=%s, viz=%v, steps=%d",
		time.Since(startTime), result.RequestType, result.Complexity, result.NeedsVisualization, len(result.Steps)))

	return result, nil
}

func (c *CombinedClassifierPlanner) buildCombinedPrompt(userQuery, dataSourceInfo string) string {
	return fmt.Sprintf(`Analyze the user request, performing both classification and execution planning.

## User Request
"%s"

## Data Source Info
%s

## Classification Rules
- consultation: Pure consultation/advice (no actual analysis needed)
- data_analysis: Data analysis (query + analysis)
- visualization: Explicitly needs charts
- data_export: Data export
- calculation: Simple calculation (no database needed)
- web_search: Needs web search

## Visualization Decision
Default to generating charts, unless it's consultation/calculation/user only wants text results.
Involves analysis, statistics, trends, distribution, comparison, ranking �?needs_visualization=true

## Chart Types
Time trends→line, Category comparison→bar, Proportions→pie, Multi-dimensional→grouped_bar

## Complexity
trivial (no tools), simple (1 call), moderate (2-3 calls), complex (4+ calls)

## Output JSON
{
  "request_type": "string",
  "needs_visualization": bool,
  "needs_data_export": bool,
  "confidence": 0.0-1.0,
  "reasoning": "brief reason",
  "suggested_chart_type": "line|bar|pie|grouped_bar|scatter|heatmap",
  "task_type": "simple|data_query|visualization|calculation|web_search",
  "complexity": "trivial|simple|moderate|complex",
  "is_quick_path": bool,
  "needs_schema": bool,
  "needs_sql": bool,
  "needs_python": bool,
  "needs_web_search": bool,
  "output_format": "text|table|chart|file",
  "estimated_calls": 1-8,
  "steps": [{"step_num":1,"tool":"tool_name","purpose":"purpose","input":"input","depends_on":[]}]
}

Output only JSON.`, userQuery, dataSourceInfo)
}

func (c *CombinedClassifierPlanner) cacheResult(key string, result *CombinedResult) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	// Set cache timestamp
	result.CachedAt = time.Now()

	// Evict expired entries and enforce size limit
	if len(c.cache) >= combinedCacheMaxSize {
		// Remove expired entries first
		now := time.Now()
		for k, v := range c.cache {
			if now.Sub(v.CachedAt) > combinedCacheTTL {
				delete(c.cache, k)
			}
		}
		// If still over limit, remove oldest entries
		for len(c.cache) >= combinedCacheMaxSize {
			var oldestKey string
			var oldestTime time.Time
			for k, v := range c.cache {
				if oldestKey == "" || v.CachedAt.Before(oldestTime) {
					oldestKey = k
					oldestTime = v.CachedAt
				}
			}
			if oldestKey != "" {
				delete(c.cache, oldestKey)
			} else {
				break
			}
		}
	}

	c.cache[key] = result
}

// getCachedResult retrieves a cached result if it exists and is not expired
func (c *CombinedClassifierPlanner) getCachedResult(key string) (*CombinedResult, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	result, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// Check TTL
	if time.Since(result.CachedAt) > combinedCacheTTL {
		return nil, false
	}

	return result, true
}

func (c *CombinedClassifierPlanner) detectQuickPath(queryLower string) *CombinedResult {
	// Time/Date queries
	if containsAny(queryLower, []string{"时间", "日期", "几点", "今天", "现在", "当前时间", "what time", "current time"}) &&
		!containsAny(queryLower, []string{"订单", "销�?, "数据", "查询", "统计", "分析"}) {
		return &CombinedResult{
			RequestType: "calculation", TaskType: "calculation", Complexity: "simple",
			IsQuickPath: true, NeedsPython: true, OutputFormat: "text", EstimatedCalls: 1,
			Confidence: 1.0,
			QuickPathCode: `import datetime
print(datetime.datetime.now().strftime("%Y�?m�?d�?%H:%M:%S"))`,
			Steps: []PlanStep{{StepNum: 1, Tool: "python_executor", Purpose: "获取系统时间"}},
		}
	}

	// Greetings
	if containsAny(queryLower, []string{"你好", "hello", "hi", "帮助", "help"}) &&
		!containsAny(queryLower, []string{"分析", "查询", "数据"}) {
		return c.quickResult("consultation", "trivial")
	}

	return nil
}

func (c *CombinedClassifierPlanner) tryKeywordClassification(queryLower, dataSourceInfo string) *CombinedResult {
	// Pure consultation (no analysis keywords)
	analysisIndicators := []string{"分析", "统计", "查询", "计算", "对比", "趋势", "分布", "排名",
		"销�?, "订单", "客户", "产品", "收入", "利润", "数量", "�?, "�?}
	hasAnalysis := false
	for _, ind := range analysisIndicators {
		if strings.Contains(queryLower, ind) {
			hasAnalysis = true
			break
		}
	}

	if !hasAnalysis && containsAny(queryLower, []string{"可以做什么分�?, "分析方向", "怎么分析", "能做什�?, "建议"}) {
		return &CombinedResult{
			RequestType: "consultation", TaskType: "consultation", Complexity: "simple",
			NeedsSchema: true, OutputFormat: "text", EstimatedCalls: 1, Confidence: 0.9,
			Steps: []PlanStep{{StepNum: 1, Tool: "get_data_source_context", Purpose: "获取数据源信�?}},
		}
	}

	// Web search patterns
	if containsAny(queryLower, []string{"天气", "新闻", "股价", "汇率", "搜索最�?}) &&
		!containsAny(queryLower, []string{"数据", "订单", "分析"}) {
		return &CombinedResult{
			RequestType: "web_search", TaskType: "web_search", Complexity: "simple",
			NeedsWebSearch: true, OutputFormat: "text", EstimatedCalls: 1, Confidence: 0.85,
			Steps: []PlanStep{{StepNum: 1, Tool: "web_search", Purpose: "搜索信息"}},
		}
	}

	// Data export patterns - detect when user wants to export/download data
	if containsAny(queryLower, []string{"导出", "下载", "export", "download"}) &&
		containsAny(queryLower, []string{"数据", "表格", "excel", "csv", "data", "table"}) {
		return &CombinedResult{
			RequestType: "data_export", TaskType: "data_query", Complexity: "moderate",
			NeedsSchema: true, NeedsSQL: true, NeedsDataExport: true,
			OutputFormat: "file", EstimatedCalls: 3, Confidence: 0.9,
			Steps: []PlanStep{
				{StepNum: 1, Tool: "get_data_source_context", Purpose: "获取数据结构"},
				{StepNum: 2, Tool: "execute_sql", Purpose: "查询数据", DependsOn: []int{1}},
				{StepNum: 3, Tool: "export_data", Purpose: "导出为Excel", DependsOn: []int{2}},
			},
		}
	}

	// Don't use keyword classification for ambiguous cases - let LLM decide
	return nil
}

func (c *CombinedClassifierPlanner) quickResult(reqType, complexity string) *CombinedResult {
	return &CombinedResult{
		RequestType: reqType, TaskType: reqType, Complexity: complexity,
		IsQuickPath: true, OutputFormat: "text", EstimatedCalls: 0, Confidence: 1.0,
	}
}

func (c *CombinedClassifierPlanner) fallbackResult(queryLower, dataSourceInfo string) *CombinedResult {
	result := &CombinedResult{
		RequestType:        "data_analysis",
		NeedsVisualization: true,
		Confidence:         0.5,
		Reasoning:          "Fallback - defaulting to visualization",
		SuggestedChartType: "bar",
		TaskType:           "data_query",
		Complexity:         "moderate",
		NeedsSchema:        true,
		NeedsSQL:           true,
		NeedsPython:        true,
		OutputFormat:       "chart",
		EstimatedCalls:     3,
		CachedAt:           time.Now(),
		Steps: []PlanStep{
			{StepNum: 1, Tool: "get_data_source_context", Purpose: "获取数据结构"},
			{StepNum: 2, Tool: "execute_sql", Purpose: "查询数据", DependsOn: []int{1}},
			{StepNum: 3, Tool: "python_executor", Purpose: "生成图表", DependsOn: []int{2}},
		},
	}

	// Adjust for non-analysis requests
	if containsAny(queryLower, []string{"建议", "可以做什�?}) && !containsAny(queryLower, []string{"分析", "统计"}) {
		result.RequestType = "consultation"
		result.NeedsVisualization = false
		result.OutputFormat = "text"
		result.NeedsPython = false
		result.EstimatedCalls = 1
		result.Steps = result.Steps[:1]
	}

	// Chart type hints
	if result.NeedsVisualization {
		if containsAny(queryLower, []string{"趋势", "变化", "时间", "�?, "�?}) {
			result.SuggestedChartType = "line"
		} else if containsAny(queryLower, []string{"占比", "比例", "分布"}) {
			result.SuggestedChartType = "pie"
		}
	}

	return result
}
