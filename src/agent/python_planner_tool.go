package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// PythonPlannerTool exposes the Python planner as an Eino tool
type PythonPlannerTool struct {
	planner *PythonPlanner
}

// NewPythonPlannerTool creates a new Python planner tool
func NewPythonPlannerTool(planner *PythonPlanner) *PythonPlannerTool {
	return &PythonPlannerTool{
		planner: planner,
	}
}

type pythonPlannerInput struct {
	UserQuery         string   `json:"user_query"`
	DataDescription   string   `json:"data_description,omitempty"`
	SQLResult         string   `json:"sql_result,omitempty"`
	AvailableColumns  []string `json:"available_columns,omitempty"`
	DataSample        string   `json:"data_sample,omitempty"`
}

func (t *PythonPlannerTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "python_planner",
		Desc: `Plan Python code generation with three-phase approach: 1) Library & Data Linking, 2) Logic Planning, 3) Code Generation.

USE THIS TOOL BEFORE python_executor for complex analysis tasks (>20 lines of code).

The tool returns a detailed plan including:
- Required libraries
- Step-by-step logic
- Generated Python code (validated and ≤80 lines)

After receiving the plan, review it and then use python_executor with the generated code.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"user_query": {
				Type:     schema.String,
				Desc:     "The user's analysis request in natural language",
				Required: true,
			},
			"data_description": {
				Type:     schema.String,
				Desc:     "Description of available data (optional)",
				Required: false,
			},
			"sql_result": {
				Type:     schema.String,
				Desc:     "Previous SQL query result in JSON format (optional). If provided, the planner will generate code to load this data.",
				Required: false,
			},
			"available_columns": {
				Type:     schema.Array,
				Desc:     "List of available column names in the data (optional)",
				Required: false,
			},
			"data_sample": {
				Type:     schema.String,
				Desc:     "Sample data rows for reference (optional)",
				Required: false,
			},
		}),
	}, nil
}

func (t *PythonPlannerTool) InvokableRun(ctx context.Context, input string, opts ...tool.Option) (string, error) {
	var planInput pythonPlannerInput
	if err := json.Unmarshal([]byte(input), &planInput); err != nil {
		return "", fmt.Errorf("invalid input format: %v", err)
	}

	if planInput.UserQuery == "" {
		return "", fmt.Errorf("user_query is required")
	}

	// Build available context
	availableContext := &AvailableContext{
		SQLResult:         planInput.SQLResult,
		DataDescription:   planInput.DataDescription,
		AvailableColumns:  planInput.AvailableColumns,
		DataSample:        planInput.DataSample,
	}

	// Run the three-phase planning
	plan, err := t.planner.PlanAndGenerateCode(ctx, planInput.UserQuery, availableContext)
	if err != nil {
		return "", fmt.Errorf("Python planning failed: %v", err)
	}

	// Format the result for LLM
	result := formatPlanForLLM(plan)
	return result, nil
}

// formatPlanForLLM formats the Python plan for LLM consumption
func formatPlanForLLM(plan *PythonPlan) string {
	var result strings.Builder

	result.WriteString("## Python Code Plan (3-Phase Generation)\n\n")

	// Phase 1 Results
	result.WriteString("### Phase 1: Library & Data Linking\n")
	result.WriteString(fmt.Sprintf("**Required Libraries:** %s\n", strings.Join(plan.RequiredLibraries, ", ")))
	result.WriteString(fmt.Sprintf("**Data Sources:** %s\n", strings.Join(plan.DataSources, ", ")))
	result.WriteString(fmt.Sprintf("**Input Format:** %s\n", plan.InputFormat))
	result.WriteString(fmt.Sprintf("**Expected Output:** %s\n\n", plan.ExpectedOutput))

	// Phase 2 Results
	result.WriteString("### Phase 2: Logic Planning\n")
	result.WriteString(fmt.Sprintf("**Overall Logic:** %s\n\n", plan.CodeLogic))

	if plan.DataProcessing != "" {
		result.WriteString(fmt.Sprintf("**Data Processing:** %s\n", plan.DataProcessing))
	}
	if plan.CalculationLogic != "" {
		result.WriteString(fmt.Sprintf("**Calculations:** %s\n", plan.CalculationLogic))
	}
	if plan.VisualizationPlan != "" {
		result.WriteString(fmt.Sprintf("**Visualization:** %s\n", plan.VisualizationPlan))
	}

	if len(plan.Steps) > 0 {
		result.WriteString("\n**Implementation Steps:**\n")
		for i, step := range plan.Steps {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	}
	result.WriteString("\n")

	// Phase 3 Results
	result.WriteString("### Phase 3: Generated Code\n")
	result.WriteString(fmt.Sprintf("**Code Length:** %d lines\n", plan.CodeLength))
	result.WriteString(fmt.Sprintf("**Complexity:** %s\n", plan.Complexity))
	result.WriteString(fmt.Sprintf("**Syntax Valid:** %v\n\n", plan.SyntaxValid))

	if plan.CodeLength > 80 {
		result.WriteString("⚠️ **WARNING:** Generated code exceeds 80-line limit. Consider breaking into multiple parts.\n\n")
	}

	result.WriteString("**Generated Code:**\n```python\n")
	result.WriteString(plan.GeneratedCode)
	result.WriteString("\n```\n\n")

	// Next steps
	result.WriteString("## Next Steps\n")
	result.WriteString("1. Review the generated code above\n")
	result.WriteString("2. If the code looks correct, use python_executor with this exact code\n")
	result.WriteString("3. If errors occur, the error will be analyzed and corrected automatically\n\n")

	result.WriteString("**Ready to execute?** Call python_executor with the code above.")

	return result.String()
}
