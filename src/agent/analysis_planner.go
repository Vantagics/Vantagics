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
	RequestTypeSimple           RequestType = "simple"             // 1次工具调用
	RequestTypeDataQuery        RequestType = "data_query"         // 数据查询
	RequestTypeVisualization    RequestType = "visualization"      // 可视化
	RequestTypeCalculation      RequestType = "calculation"        // 计算
	RequestTypeWebSearch        RequestType = "web_search"         // 网络搜索
	RequestTypeConsultation     RequestType = "consultation"       // 咨询建议
	RequestTypeMultiStepAnalysis RequestType = "multi_step_analysis" // 多步骤分析
)

// SchemaLevel represents the detail level of schema information
type SchemaLevel string

const (
	SchemaLevelBasic    SchemaLevel = "basic"    // 只有表名和描述
	SchemaLevelDetailed SchemaLevel = "detailed" // 完整字段信息
)

// ConsultationPatterns defines patterns for consultation requests
var ConsultationPatterns = []string{
	"建议",
	"分析方向",
	"可以做什么分析",
	"分析思路",
	"怎么分析",
	"分析维度",
	"有什么洞察",
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
	"多维度分析",
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
	prompt := fmt.Sprintf(`你是一个数据分析任务规划专家。分析用户请求，创建最优执行计划。

## 用户请求
"%s"

## 数据源信息
%s

## 任务分类规则
1. **trivial** (无需工具): 问候、闲聊、简单问答
2. **simple** (1次工具调用): 时间查询、简单计算、单位换算
3. **moderate** (2-3次工具调用): 单表查询+可视化
4. **complex** (4+次工具调用): 多表关联、复杂分析

## 快速路径检测
以下情况直接用python_executor，不需要数据源:
- 时间/日期查询 → datetime模块
- 数学计算 → 直接计算
- 单位换算 → 直接换算
- 随机数/UUID → random/uuid模块

## 输出格式 (JSON)
{
  "task_type": "simple|data_query|visualization|calculation|web_search",
  "complexity": "trivial|simple|moderate|complex",
  "is_quick_path": true/false,
  "quick_path_code": "如果is_quick_path为true，提供完整Python代码",
  "needs_schema": true/false,
  "needs_sql": true/false,
  "needs_python": true/false,
  "needs_web_search": true/false,
  "output_format": "text|table|chart|file",
  "estimated_calls": 1-8,
  "steps": [
    {
      "step_num": 1,
      "tool": "工具名称",
      "purpose": "这一步的目的",
      "input": "预期输入",
      "depends_on": []
    }
  ]
}

## 规划原则
1. 最小化工具调用次数
2. 避免重复获取schema
3. 尽可能用一条SQL完成查询
4. 只在需要可视化时才用python_executor

只输出JSON，不要其他内容。`, userQuery, dataSourceInfo)

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: "你是数据分析任务规划专家。只输出有效JSON。"},
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
		!containsAny(queryLower, []string{"订单", "销售", "数据", "查询", "统计", "分析"}) {
		return &AnalysisPlan{
			TaskType:     "calculation",
			Complexity:   "simple",
			IsQuickPath:  true,
			QuickPathCode: `import datetime
print(datetime.datetime.now().strftime("%Y年%m月%d日 %H:%M:%S"))`,
			NeedsPython:  true,
			OutputFormat: "text",
			EstimatedCalls: 1,
			Steps: []PlanStep{
				{StepNum: 1, Tool: "python_executor", Purpose: "获取系统时间", Input: "datetime代码"},
			},
		}
	}

	// Simple math calculations
	if containsAny(queryLower, []string{"计算", "等于多少", "加", "减", "乘", "除", "平方", "开方", "calculate", "compute"}) &&
		!containsAny(queryLower, []string{"订单", "销售", "数据", "查询", "统计"}) {
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
	if containsAny(queryLower, []string{"uuid", "随机数", "random"}) {
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
				{StepNum: 1, Tool: "python_executor", Purpose: "生成随机值", Input: "random/uuid代码"},
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
				Purpose:     "获取数据源基本信息",
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
				Purpose:     "生成可视化结果",
				Input:       "Python code",
				DependsOn:   []int{3},
			},
		},
	}
}

// createFallbackPlan creates a default plan when LLM planning fails
func (p *AnalysisPlanner) createFallbackPlan(query string) *AnalysisPlan {
	queryLower := strings.ToLower(query)

	// Detect if visualization is likely needed
	needsChart := containsAny(queryLower, []string{"图", "chart", "可视化", "趋势", "分布", "对比", "visualization"})

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
			Purpose:   "生成可视化",
			Input:     "matplotlib代码",
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
	sb.WriteString("\n\n📋 执行计划:\n")
	sb.WriteString(fmt.Sprintf("类型: %s | 复杂度: %s | 预计调用: %d次\n", plan.TaskType, plan.Complexity, plan.EstimatedCalls))

	if plan.IsQuickPath && plan.QuickPathCode != "" {
		sb.WriteString("⚡ 快速路径: 直接执行以下代码\n")
		sb.WriteString("```python\n")
		sb.WriteString(plan.QuickPathCode)
		sb.WriteString("\n```\n")
		return sb.String()
	}

	sb.WriteString("步骤:\n")
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
