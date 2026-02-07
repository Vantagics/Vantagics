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

	// 1. Check cache (valid for 5 minutes)
	cacheKey := fmt.Sprintf("%s|%s", userQuery, dataSourceInfo)
	c.cacheMu.RLock()
	if cached, ok := c.cache[cacheKey]; ok {
		if time.Since(cached.CachedAt) < 5*time.Minute {
			c.cacheMu.RUnlock()
			c.log("[COMBINED] Using cached classify+plan result")
			return cached, nil
		}
	}
	c.cacheMu.RUnlock()

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
		{Role: schema.System, Content: "你是请求分类和任务规划专家。只输出有效JSON。"},
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
	return fmt.Sprintf(`分析用户请求，同时完成分类和执行规划。

## 用户请求
"%s"

## 数据源信息
%s

## 分类规则
- consultation: 纯咨询/建议（不需要实际分析）
- data_analysis: 数据分析（查询+分析）
- visualization: 明确需要图表
- data_export: 数据导出
- calculation: 简单计算（不需要数据库）
- web_search: 需要网络搜索

## 可视化判断
默认生成图表，除非是咨询/计算/用户只要文字结果。
涉及分析、统计、趋势、分布、对比、排名 → needs_visualization=true

## 图表类型
时间趋势→line, 分类对比→bar, 占比→pie, 多维→grouped_bar

## 复杂度
trivial(无需工具), simple(1次调用), moderate(2-3次), complex(4+次)

## 输出JSON
{
  "request_type": "string",
  "needs_visualization": bool,
  "needs_data_export": bool,
  "confidence": 0.0-1.0,
  "reasoning": "简短原因",
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
  "steps": [{"step_num":1,"tool":"工具名","purpose":"目的","input":"输入","depends_on":[]}]
}

只输出JSON。`, userQuery, dataSourceInfo)
}

func (c *CombinedClassifierPlanner) cacheResult(key string, result *CombinedResult) {
	c.cacheMu.Lock()
	c.cache[key] = result
	c.cacheMu.Unlock()
}

func (c *CombinedClassifierPlanner) detectQuickPath(queryLower string) *CombinedResult {
	// Time/Date queries
	if containsAny(queryLower, []string{"时间", "日期", "几点", "今天", "现在", "当前时间", "what time", "current time"}) &&
		!containsAny(queryLower, []string{"订单", "销售", "数据", "查询", "统计", "分析"}) {
		return &CombinedResult{
			RequestType: "calculation", TaskType: "calculation", Complexity: "simple",
			IsQuickPath: true, NeedsPython: true, OutputFormat: "text", EstimatedCalls: 1,
			Confidence: 1.0,
			QuickPathCode: `import datetime
print(datetime.datetime.now().strftime("%Y年%m月%d日 %H:%M:%S"))`,
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
		"销售", "订单", "客户", "产品", "收入", "利润", "数量", "图", "表"}
	hasAnalysis := false
	for _, ind := range analysisIndicators {
		if strings.Contains(queryLower, ind) {
			hasAnalysis = true
			break
		}
	}

	if !hasAnalysis && containsAny(queryLower, []string{"可以做什么分析", "分析方向", "怎么分析", "能做什么", "建议"}) {
		return &CombinedResult{
			RequestType: "consultation", TaskType: "consultation", Complexity: "simple",
			NeedsSchema: true, OutputFormat: "text", EstimatedCalls: 1, Confidence: 0.9,
			Steps: []PlanStep{{StepNum: 1, Tool: "get_data_source_context", Purpose: "获取数据源信息"}},
		}
	}

	// Web search patterns
	if containsAny(queryLower, []string{"天气", "新闻", "股价", "汇率", "搜索最新"}) &&
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
	if containsAny(queryLower, []string{"建议", "可以做什么"}) && !containsAny(queryLower, []string{"分析", "统计"}) {
		result.RequestType = "consultation"
		result.NeedsVisualization = false
		result.OutputFormat = "text"
		result.NeedsPython = false
		result.EstimatedCalls = 1
		result.Steps = result.Steps[:1]
	}

	// Chart type hints
	if result.NeedsVisualization {
		if containsAny(queryLower, []string{"趋势", "变化", "时间", "月", "年"}) {
			result.SuggestedChartType = "line"
		} else if containsAny(queryLower, []string{"占比", "比例", "分布"}) {
			result.SuggestedChartType = "pie"
		}
	}

	return result
}
