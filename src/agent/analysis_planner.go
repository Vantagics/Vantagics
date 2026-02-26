package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	einoSchema "github.com/cloudwego/eino/schema"
)

// RequestType represents the classification of a user request
type RequestType string

const (
	RequestTypeTrivial          RequestType = "trivial"            // 无需工具调用
	RequestTypeSimple           RequestType = "simple"             // 1次工具调�
	RequestTypeDataQuery        RequestType = "data_query"         // 数据查询
	RequestTypeVisualization    RequestType = "visualization"      // 可视�
	RequestTypeCalculation      RequestType = "calculation"        // 计算
	RequestTypeWebSearch        RequestType = "web_search"         // 网络搜索
	RequestTypeConsultation     RequestType = "consultation"       // 咨询建议
	RequestTypeMultiStepAnalysis RequestType = "multi_step_analysis" // 多步骤分�
)

// SchemaLevel represents the detail level of schema information
type SchemaLevel string

const (
	SchemaLevelBasic    SchemaLevel = "basic"    // 只有表名和描�
	SchemaLevelDetailed SchemaLevel = "detailed" // 完整字段信息
)

// ConsultationPatterns defines patterns for consultation requests
var ConsultationPatterns = []string{
	"建议",
	"分析方向",
	"可以做什么分�",
	"分析思路",
	"怎么分析",
	"分析维度",
	"有什么洞�",
	"suggest",
	"recommendation",
	"what analysis",
	"how to analyze",
}

// MultiStepPatterns defines patterns for multi-step analysis
var MultiStepPatterns = []string{
	"全面分析",
	"深入分析",
	"综合分析",
	"多维度分�",
	"详细分析",
	"complete analysis",
	"comprehensive analysis",
	"in-depth analysis",
}

// SchemaLevelMapping maps request types to schema levels
var SchemaLevelMapping = map[RequestType]SchemaLevel{
	RequestTypeTrivial:           SchemaLevelBasic,
	RequestTypeSimple:            SchemaLevelBasic,
	RequestTypeConsultation:      SchemaLevelBasic,
	RequestTypeDataQuery:         SchemaLevelDetailed,
	RequestTypeVisualization:     SchemaLevelDetailed,
	RequestTypeCalculation:       SchemaLevelBasic,
	RequestTypeWebSearch:         SchemaLevelBasic,
	RequestTypeMultiStepAnalysis: SchemaLevelDetailed,
}

// AnalysisPlanner performs task decomposition before execution
// to avoid redundant steps and improve efficiency
type AnalysisPlanner struct {
	chatModel model.ChatModel
	logger    func(string)
}

// AnalysisPlan represents the execution plan for a user request
type AnalysisPlan struct {
	// Task classification
	TaskType     string `json:"task_type"`     // simple, data_query, visualization, calculation, web_search
	Complexity   string `json:"complexity"`    // trivial, simple, moderate, complex
	
	// Execution plan
	Steps        []PlanStep `json:"steps"`
	EstimatedCalls int     `json:"estimated_calls"` // Expected tool calls
	
	// Quick path detection
	IsQuickPath  bool   `json:"is_quick_path"`  // Can be done without data source
	QuickPathCode string `json:"quick_path_code,omitempty"` // Direct Python code for quick path
	
	// Data requirements
	NeedsSchema  bool     `json:"needs_schema"`   // Needs get_data_source_context
	NeedsSQL     bool     `json:"needs_sql"`      // Needs execute_sql
	NeedsPython  bool     `json:"needs_python"`   // Needs python_executor
	NeedsWebSearch bool   `json:"needs_web_search"` // Needs web_search
	
	// Output format
	OutputFormat string `json:"output_format"` // text, table, chart, file
	
	// New fields for enhanced planning
	RequestType    RequestType `json:"request_type"`
	SchemaLevel    SchemaLevel `json:"schema_level"`
	IsMultiStep    bool        `json:"is_multi_step"`
	Checkpoints    []int       `json:"checkpoints,omitempty"` // Step numbers that are checkpoints
}

// PlanStep represents a single step in the execution plan
type PlanStep struct {
	StepNum          int    `json:"step_num"`
	Tool             string `json:"tool"`        // python_executor, get_data_source_context, execute_sql, web_search
	Purpose          string `json:"purpose"`     // What this step accomplishes
	Input            string `json:"input"`       // Expected input/parameters
	DependsOn        []int  `json:"depends_on"`  // Step numbers this depends on
	EstimatedDuration int   `json:"estimated_duration_ms,omitempty"`
	SchemaLevel      string `json:"schema_level,omitempty"`      // For get_data_source_context
	QueryType        string `json:"query_type,omitempty"`        // For execute_sql
	IsCheckpoint     bool   `json:"is_checkpoint,omitempty"`
}

// NewAnalysisPlanner creates a new analysis planner
func NewAnalysisPlanner(chatModel model.ChatModel, logger func(string)) *AnalysisPlanner {
	return &AnalysisPlanner{
		chatModel: chatModel,
		logger:    logger,
	}
}

// PlanAnalysis analyzes the user request and creates an execution plan
func (p *AnalysisPlanner) PlanAnalysis(ctx context.Context, userQuery string, dataSourceInfo string) (*AnalysisPlan, error) {
	if p.logger != nil {
		p.logger("[PLANNER] Analyzing request and creating execution plan")
	}

	// First, check for quick path (no LLM call needed)
	quickPlan := p.detectQuickPath(userQuery)
	if quickPlan != nil {
		if p.logger != nil {
			p.logger(fmt.Sprintf("[PLANNER] Quick path detected: %s", quickPlan.TaskType))
		}
		return quickPlan, nil
	}

	// Classify the request
	classifier := NewRequestClassifier(p.logger)
	requestType := classifier.ClassifyRequest(userQuery, dataSourceInfo)

	// For consultation requests, create a simple plan without SQL
	if requestType == RequestTypeConsultation {
		if p.logger != nil {
			p.logger("[PLANNER] Consultation request detected - creating simple plan")
		}
		return p.createConsultationPlan(), nil
	}

	// For multi-step analysis, create a plan with checkpoints
	if requestType == RequestTypeMultiStepAnalysis {
		if p.logger != nil {
			p.logger("[PLANNER] Multi-step analysis detected - creating checkpoint plan")
		}
		return p.createMultiStepPlan(), nil
	}

	// For complex requests, use LLM to plan
	prompt := fmt.Sprintf(`You are a data analysis task planning expert. Analyze the user request and create an optimal execution plan.

## User Request
"%s"

## Data Source Info
%s

## Task Classification Rules
1. **trivial** (no tools): Greetings, small talk, simple Q&A
2. **simple** (1 tool call): Time queries, simple calculations, unit conversions
3. **moderate** (2-3 tool calls): Single table query + visualization
4. **complex** (4+ tool calls): Multi-table joins, complex analysis

## Quick Path Detection
These can use python_executor directly without data source:
- Time/date queries �datetime module
- Math calculations �direct computation
- Unit conversions �direct conversion
- Random numbers/UUID �random/uuid module

## Output Format (JSON)
{
  "task_type": "simple|data_query|visualization|calculation|web_search",
  "complexity": "trivial|simple|moderate|complex",
  "is_quick_path": true/false,
  "quick_path_code": "if is_quick_path is true, provide complete Python code",
  "needs_schema": true/false,
  "needs_sql": true/false,
  "needs_python": true/false,
  "needs_web_search": true/false,
  "output_format": "text|table|chart|file",
  "estimated_calls": 1-8,
  "steps": [
    {
      "step_num": 1,
      "tool": "tool_name",
      "purpose": "purpose of this step",
      "input": "expected input",
      "depends_on": []
    }
  ]
}

## Planning Principles
1. Minimize tool call count
2. Avoid redundant schema fetches
3. Use a single SQL query when possible
4. Only use python_executor when visualization is needed

Output only JSON.`, userQuery, dataSourceInfo)

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: "You are a data analysis task planning expert. Output only valid JSON."},
		{Role: einoSchema.User, Content: prompt},
	}

	resp, err := p.chatModel.Generate(ctx, msgs)
	if err != nil {
		if p.logger != nil {
			p.logger(fmt.Sprintf("[PLANNER] LLM planning failed: %v, using fallback", err))
		}
		return p.createFallbackPlan(userQuery), nil
	}

	// Parse response
	plan := &AnalysisPlan{}
	content := strings.TrimSpace(resp.Content)
	content = extractJSON(content)

	if err := json.Unmarshal([]byte(content), plan); err != nil {
		if p.logger != nil {
			p.logger(fmt.Sprintf("[PLANNER] Failed to parse plan: %v, using fallback", err))
		}
		return p.createFallbackPlan(userQuery), nil
	}

	// Set request type and schema level
	plan.RequestType = requestType
	plan.SchemaLevel = classifier.GetSchemaLevel(requestType)

	if p.logger != nil {
		p.logger(fmt.Sprintf("[PLANNER] Plan created: type=%s, complexity=%s, steps=%d, estimated_calls=%d",
			plan.TaskType, plan.Complexity, len(plan.Steps), plan.EstimatedCalls))
	}

	return plan, nil
}

// detectQuickPath checks if the request can be handled without LLM planning
func (p *AnalysisPlanner) detectQuickPath(query string) *AnalysisPlan {
	queryLower := strings.ToLower(query)

	// Time/Date queries
	if containsAny(queryLower, []string{"时间", "日期", "几点", "今天", "现在", "当前时间", "what time", "current time", "today", "date"}) &&
		!containsAny(queryLower, []string{"订单", "销�", "数据", "查询", "统计", "分析"}) {
		return &AnalysisPlan{
			TaskType:     "calculation",
			Complexity:   "simple",
			IsQuickPath:  true,
			QuickPathCode: `import datetime
print(datetime.datetime.now().strftime("%Y�m�d�%H:%M:%S"))`,
			NeedsPython:  true,
			OutputFormat: "text",
			EstimatedCalls: 1,
			Steps: []PlanStep{
				{StepNum: 1, Tool: "python_executor", Purpose: "获取系统时间", Input: "datetime代码"},
			},
		}
	}

	// Simple math calculations
	if containsAny(queryLower, []string{"计算", "等于多少", "�", "�", "�", "�", "平方", "开�", "calculate", "compute"}) &&
		!containsAny(queryLower, []string{"订单", "销�", "数据", "查询", "统计"}) {
		return &AnalysisPlan{
			TaskType:     "calculation",
			Complexity:   "simple",
			IsQuickPath:  true,
			NeedsPython:  true,
			OutputFormat: "text",
			EstimatedCalls: 1,
			Steps: []PlanStep{
				{StepNum: 1, Tool: "python_executor", Purpose: "执行计算", Input: "数学表达式"},
			},
		}
	}

	// UUID generation
	if containsAny(queryLower, []string{"uuid", "随机�", "random"}) {
		code := ""
		if strings.Contains(queryLower, "uuid") {
			code = `import uuid
print(str(uuid.uuid4()))`
		} else {
			code = `import random
print(random.randint(1, 100))`
		}
		return &AnalysisPlan{
			TaskType:     "calculation",
			Complexity:   "simple",
			IsQuickPath:  true,
			QuickPathCode: code,
			NeedsPython:  true,
			OutputFormat: "text",
			EstimatedCalls: 1,
			Steps: []PlanStep{
				{StepNum: 1, Tool: "python_executor", Purpose: "生成随机�", Input: "random/uuid代码"},
			},
		}
	}

	return nil
}

// createConsultationPlan creates a plan for consultation requests
func (p *AnalysisPlanner) createConsultationPlan() *AnalysisPlan {
	return &AnalysisPlan{
		TaskType:       "consultation",
		Complexity:     "simple",
		IsQuickPath:    false,
		NeedsSchema:    true,
		NeedsSQL:       false,
		NeedsPython:    false,
		NeedsWebSearch: false,
		OutputFormat:   "text",
		RequestType:    RequestTypeConsultation,
		SchemaLevel:    SchemaLevelBasic,
		IsMultiStep:    false,
		EstimatedCalls: 1,
		Steps: []PlanStep{
			{
				StepNum:     1,
				Tool:        "get_data_source_context",
				Purpose:     "获取数据源基本信�",
				Input:       "data_source_id",
				SchemaLevel: string(SchemaLevelBasic),
			},
		},
	}
}

// createMultiStepPlan creates a plan for multi-step analysis
func (p *AnalysisPlanner) createMultiStepPlan() *AnalysisPlan {
	return &AnalysisPlan{
		TaskType:       "multi_step_analysis",
		Complexity:     "complex",
		IsQuickPath:    false,
		NeedsSchema:    true,
		NeedsSQL:       true,
		NeedsPython:    true,
		NeedsWebSearch: false,
		OutputFormat:   "chart",
		RequestType:    RequestTypeMultiStepAnalysis,
		SchemaLevel:    SchemaLevelDetailed,
		IsMultiStep:    true,
		EstimatedCalls: 4,
		Checkpoints:    []int{2}, // Checkpoint after step 2
		Steps: []PlanStep{
			{
				StepNum:     1,
				Tool:        "get_data_source_context",
				Purpose:     "获取完整数据结构",
				Input:       "data_source_id",
				SchemaLevel: string(SchemaLevelDetailed),
			},
			{
				StepNum:     2,
				Tool:        "execute_sql",
				Purpose:     "执行初步分析查询",
				Input:       "SQL query",
				DependsOn:   []int{1},
				QueryType:   "aggregation",
				IsCheckpoint: true,
			},
			{
				StepNum:     3,
				Tool:        "execute_sql",
				Purpose:     "执行深入分析查询",
				Input:       "SQL query",
				DependsOn:   []int{2},
				QueryType:   "join",
			},
			{
				StepNum:     4,
				Tool:        "python_executor",
				Purpose:     "生成可视化结�",
				Input:       "Python code",
				DependsOn:   []int{3},
			},
		},
	}
}

// createFallbackPlan creates a default plan when LLM planning fails
// Default to including visualization for most analysis requests
func (p *AnalysisPlanner) createFallbackPlan(query string) *AnalysisPlan {
	queryLower := strings.ToLower(query)

	// Check if this is a simple count/list query (no visualization needed)
	simpleQueryPatterns := []string{"有多少", "总数", "计数", "列出所有", "显示所有"}
	isSimpleQuery := false
	for _, pattern := range simpleQueryPatterns {
		if strings.Contains(queryLower, pattern) {
			isSimpleQuery = true
			break
		}
	}

	// Detect if visualization is likely needed (more inclusive)
	// Most analysis requests benefit from visualization
	vizKeywords := []string{
		"�", "chart", "可视�", "趋势", "分布", "对比", "visualization",
		"分析", "统计", "销�", "收入", "利润", "增长",
		"按月", "按年", "时间", "周期", "排名", "top", "�",
		"analysis", "sales", "revenue", "growth", "monthly", "yearly",
	}
	needsChart := false
	for _, keyword := range vizKeywords {
		if strings.Contains(queryLower, keyword) {
			needsChart = true
			break
		}
	}
	
	// Default to chart for analysis requests unless it's a simple query
	if !isSimpleQuery && !needsChart {
		// Check if it mentions data-related terms
		dataTerms := []string{"数据", "订单", "客户", "产品", "data", "order", "customer", "product"}
		for _, term := range dataTerms {
			if strings.Contains(queryLower, term) {
				needsChart = true
				break
			}
		}
	}

	plan := &AnalysisPlan{
		TaskType:       "data_query",
		Complexity:     "moderate",
		IsQuickPath:    false,
		NeedsSchema:    true,
		NeedsSQL:       true,
		NeedsPython:    needsChart,
		OutputFormat:   "table",
		RequestType:    RequestTypeDataQuery,
		SchemaLevel:    SchemaLevelDetailed,
		IsMultiStep:    false,
		EstimatedCalls: 2,
		Steps: []PlanStep{
			{
				StepNum:     1,
				Tool:        "get_data_source_context",
				Purpose:     "获取数据结构",
				Input:       "相关表名",
				SchemaLevel: string(SchemaLevelDetailed),
			},
			{
				StepNum:     2,
				Tool:        "execute_sql",
				Purpose:     "查询数据",
				Input:       "SQL查询",
				DependsOn:   []int{1},
				QueryType:   "general",
			},
		},
	}

	if needsChart {
		plan.OutputFormat = "chart"
		plan.EstimatedCalls = 3
		plan.Steps = append(plan.Steps, PlanStep{
			StepNum:   3,
			Tool:      "python_executor",
			Purpose:   "生成可视化图�",
			Input:     "matplotlib/seaborn代码",
			DependsOn: []int{2},
		})
	}

	return plan
}

// FormatPlanForPrompt formats the plan as guidance for the main agent
func (p *AnalysisPlanner) FormatPlanForPrompt(plan *AnalysisPlan) string {
	if plan == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n📋 Execution Plan:\n")
	sb.WriteString(fmt.Sprintf("Type: %s | Complexity: %s | Estimated calls: %d\n", plan.TaskType, plan.Complexity, plan.EstimatedCalls))

	if plan.IsQuickPath && plan.QuickPathCode != "" {
		sb.WriteString("�Quick path: execute the following code directly\n")
		sb.WriteString("```python\n")
		sb.WriteString(plan.QuickPathCode)
		sb.WriteString("\n```\n")
		return sb.String()
	}

	sb.WriteString("Steps:\n")
	for _, step := range plan.Steps {
		sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", step.StepNum, step.Tool, step.Purpose))
	}

	return sb.String()
}


// containsAny checks if s contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
