package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/model"
	einoSchema "github.com/cloudwego/eino/schema"
)

// PythonPlanner implements a three-phase Python code generation workflow:
// Phase 1: Library & Data Linking - Identify required libraries and data sources
// Phase 2: Logic Planning - Generate code logic before implementation
// Phase 3: Code Generation - Generate and validate Python code
type PythonPlanner struct {
	chatModel model.ChatModel
	logger    func(string)
}

// PythonPlan represents the result of Python planning
type PythonPlan struct {
	// Phase 1: Library & Data Linking
	RequiredLibraries []string          `json:"required_libraries"` // pandas, matplotlib, numpy, etc.
	DataSources       []string          `json:"data_sources"`       // SQL results, CSV files, etc.
	InputFormat       string            `json:"input_format"`       // JSON, DataFrame, etc.
	ExpectedOutput    string            `json:"expected_output"`    // Chart, Table, CSV, etc.

	// Phase 2: Logic Planning
	CodeLogic         string   `json:"code_logic"`         // Natural language description
	DataProcessing    string   `json:"data_processing"`    // How data will be processed
	CalculationLogic  string   `json:"calculation_logic"`  // Calculations and transformations
	VisualizationPlan string   `json:"visualization_plan"` // Visualization approach
	Steps             []string `json:"steps"`              // Step-by-step breakdown

	// Phase 3: Code Generation
	GeneratedCode string `json:"generated_code"`
	CodeLength    int    `json:"code_length"`   // Number of lines
	Complexity    string `json:"complexity"`    // simple, moderate, complex
	SyntaxValid   bool   `json:"syntax_valid"`  // Basic syntax check
}

// AvailableContext holds information available for Python code generation
type AvailableContext struct {
	SQLResult         string   `json:"sql_result,omitempty"`          // Previous SQL query result
	DataDescription   string   `json:"data_description,omitempty"`    // Description of available data
	AvailableColumns  []string `json:"available_columns,omitempty"`   // Column names
	DataSample        string   `json:"data_sample,omitempty"`         // Sample data rows
}

// NewPythonPlanner creates a new Python planner
func NewPythonPlanner(chatModel model.ChatModel, logger func(string)) *PythonPlanner {
	return &PythonPlanner{
		chatModel: chatModel,
		logger:    logger,
	}
}

// Phase1LibraryAndDataLinking identifies required libraries and data sources
func (p *PythonPlanner) Phase1LibraryAndDataLinking(ctx context.Context, userQuery string, availableContext *AvailableContext) (*PythonPlan, error) {
	if p.logger != nil {
		p.logger("[PYTHON-PLANNER] Phase 1: Library & Data Linking")
	}

	contextDesc := "No previous data available"
	if availableContext != nil {
		if availableContext.DataDescription != "" {
			contextDesc = availableContext.DataDescription
		}
		if availableContext.SQLResult != "" {
			contextDesc += fmt.Sprintf("\nPrevious SQL result available (JSON format)")
		}
		if len(availableContext.AvailableColumns) > 0 {
			contextDesc += fmt.Sprintf("\nAvailable columns: %s", strings.Join(availableContext.AvailableColumns, ", "))
		}
		if availableContext.DataSample != "" {
			contextDesc += fmt.Sprintf("\nData sample:\n%s", availableContext.DataSample)
		}
	}

	prompt := fmt.Sprintf(`You are a Python expert performing LIBRARY & DATA LINKING.

## Task
Analyze the user's request and identify what libraries and data sources are needed.

## User Request
"%s"

## Available Data
%s

## Instructions
1. List ONLY the libraries needed for this task (e.g., pandas, matplotlib, numpy, seaborn, scipy)
2. Identify data sources needed (SQL results, files, etc.)
3. Determine input format (JSON, CSV, DataFrame, etc.)
4. Determine expected output (Chart image, Table, CSV file, Summary statistics, etc.)
5. Do NOT write code yet - just identify the requirements

## Output Format (JSON)
{
  "required_libraries": ["pandas", "matplotlib"],
  "data_sources": ["SQL query result"],
  "input_format": "JSON array from SQL result",
  "expected_output": "matplotlib chart saved as chart.png",
  "reasoning": "Brief explanation of why these libraries are needed"
}

Output ONLY valid JSON, no other text.`, userQuery, contextDesc)

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: "You are a Python expert specializing in data analysis. Output only valid JSON."},
		{Role: einoSchema.User, Content: prompt},
	}

	resp, err := p.chatModel.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("library linking failed: %v", err)
	}

	// Parse response
	plan := &PythonPlan{}
	content := strings.TrimSpace(resp.Content)
	// Extract JSON from markdown code blocks if present
	content = extractJSON(content)

	var linkResult struct {
		RequiredLibraries []string `json:"required_libraries"`
		DataSources       []string `json:"data_sources"`
		InputFormat       string   `json:"input_format"`
		ExpectedOutput    string   `json:"expected_output"`
		Reasoning         string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(content), &linkResult); err != nil {
		if p.logger != nil {
			p.logger(fmt.Sprintf("[PYTHON-PLANNER] Failed to parse library linking result: %v", err))
		}
		// Fallback: assume pandas and matplotlib
		plan.RequiredLibraries = []string{"pandas", "matplotlib"}
		plan.DataSources = []string{"SQL result"}
		plan.InputFormat = "JSON"
		plan.ExpectedOutput = "Chart or Table"
	} else {
		plan.RequiredLibraries = linkResult.RequiredLibraries
		plan.DataSources = linkResult.DataSources
		plan.InputFormat = linkResult.InputFormat
		plan.ExpectedOutput = linkResult.ExpectedOutput
	}

	return plan, nil
}

// Phase2LogicPlanning generates code logic before implementation
func (p *PythonPlanner) Phase2LogicPlanning(ctx context.Context, userQuery string, plan *PythonPlan, availableContext *AvailableContext) error {
	if p.logger != nil {
		p.logger("[PYTHON-PLANNER] Phase 2: Logic Planning")
	}

	contextDesc := ""
	if availableContext != nil {
		if availableContext.DataSample != "" {
			contextDesc += fmt.Sprintf("Data sample:\n%s\n\n", availableContext.DataSample)
		}
		if len(availableContext.AvailableColumns) > 0 {
			contextDesc += fmt.Sprintf("Available columns: %s\n", strings.Join(availableContext.AvailableColumns, ", "))
		}
	}

	prompt := fmt.Sprintf(`You are a Python expert creating a CODE PLAN.

## User Request
"%s"

## Selected Libraries
%s

## Data Context
%s

## Instructions
Before writing code, describe your implementation logic step by step:
1. How will you load and prepare the data?
2. What data processing is needed? (filtering, grouping, calculations)
3. What calculations or transformations?
4. What visualization or output format?
5. Break down into clear steps (max 5 steps)

## Critical Constraints
- Code must be ≤80 lines (Python executor limit)
- If task is complex, break into multiple parts
- Each part must be independently executable

## Output Format (JSON)
{
  "code_logic": "Overall description of what the code will do",
  "data_processing": "How data will be loaded and processed (e.g., 'Load JSON, convert to DataFrame, filter rows')",
  "calculation_logic": "Calculations and transformations (e.g., 'Group by category, calculate sum and average')",
  "visualization_plan": "Visualization approach (e.g., 'Bar chart with matplotlib, save as chart.png')",
  "steps": [
    "Step 1: Load JSON data into pandas DataFrame",
    "Step 2: Process data (filter, group, aggregate)",
    "Step 3: Create visualization with matplotlib",
    "Step 4: Save chart.png and print summary"
  ],
  "complexity": "simple|moderate|complex",
  "estimated_lines": 45
}

Output ONLY valid JSON, no other text.`,
		userQuery,
		strings.Join(plan.RequiredLibraries, ", "),
		contextDesc)

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: "You are a Python expert specializing in code planning. Output only valid JSON."},
		{Role: einoSchema.User, Content: prompt},
	}

	resp, err := p.chatModel.Generate(ctx, msgs)
	if err != nil {
		return fmt.Errorf("logic planning failed: %v", err)
	}

	// Parse response
	content := strings.TrimSpace(resp.Content)
	content = extractJSON(content)

	var logicResult struct {
		CodeLogic         string   `json:"code_logic"`
		DataProcessing    string   `json:"data_processing"`
		CalculationLogic  string   `json:"calculation_logic"`
		VisualizationPlan string   `json:"visualization_plan"`
		Steps             []string `json:"steps"`
		Complexity        string   `json:"complexity"`
		EstimatedLines    int      `json:"estimated_lines"`
	}

	if err := json.Unmarshal([]byte(content), &logicResult); err != nil {
		if p.logger != nil {
			p.logger(fmt.Sprintf("[PYTHON-PLANNER] Failed to parse logic planning result: %v", err))
		}
		plan.CodeLogic = "Direct code generation"
		plan.Complexity = "simple"
	} else {
		plan.CodeLogic = logicResult.CodeLogic
		plan.DataProcessing = logicResult.DataProcessing
		plan.CalculationLogic = logicResult.CalculationLogic
		plan.VisualizationPlan = logicResult.VisualizationPlan
		plan.Steps = logicResult.Steps
		plan.Complexity = logicResult.Complexity
		plan.CodeLength = logicResult.EstimatedLines
	}

	return nil
}

// Phase3CodeGeneration generates the final Python code
func (p *PythonPlanner) Phase3CodeGeneration(ctx context.Context, userQuery string, plan *PythonPlan, availableContext *AvailableContext) error {
	if p.logger != nil {
		p.logger("[PYTHON-PLANNER] Phase 3: Code Generation")
	}

	contextDesc := ""
	dataLoadCode := ""
	if availableContext != nil {
		if availableContext.SQLResult != "" {
			contextDesc += "Previous SQL result available (JSON format)\n"
			dataLoadCode = `# Load data from SQL result
import json
import pandas as pd

# Parse the JSON result from previous SQL query
data = json.loads('''SQL_RESULT_PLACEHOLDER''')
df = pd.DataFrame(data)
`
		}
		if len(availableContext.AvailableColumns) > 0 {
			contextDesc += fmt.Sprintf("Available columns: %s\n", strings.Join(availableContext.AvailableColumns, ", "))
		}
		if availableContext.DataSample != "" {
			contextDesc += fmt.Sprintf("\nData sample (first 3 rows):\n%s\n", availableContext.DataSample)
		}
	}

	// Build step-by-step guidance
	stepsText := ""
	if len(plan.Steps) > 0 {
		stepsText = "Follow these steps:\n"
		for i, step := range plan.Steps {
			stepsText += fmt.Sprintf("%d. %s\n", i+1, step)
		}
	}

	prompt := fmt.Sprintf(`You are a Python expert writing data analysis code.

## User Request
"%s"

## Code Plan
Logic: %s
Data Processing: %s
Calculations: %s
Visualization: %s

## Data Context
%s

## Required Libraries
%s

## Steps to Implement
%s

## Critical Rules
1. Code must be ≤80 lines (STRICT LIMIT)
2. ONLY use libraries from: %s
3. If loading data from SQL result, use this template:
%s
4. For charts:
   - Use matplotlib.pyplot as plt
   - Save as: plt.savefig('chart.png')
   - Close after saving: plt.close()
5. Print meaningful summaries (not raw DataFrames)
6. Handle errors gracefully
7. Use appropriate chart types (bar, line, scatter, pie, etc.)
8. Include proper labels, titles, and legends

## Output Format
Output ONLY the Python code, wrapped in python code block:
` + "```python\nYOUR CODE HERE\n```",
		userQuery,
		plan.CodeLogic,
		plan.DataProcessing,
		plan.CalculationLogic,
		plan.VisualizationPlan,
		contextDesc,
		strings.Join(plan.RequiredLibraries, ", "),
		stepsText,
		strings.Join(plan.RequiredLibraries, ", "),
		dataLoadCode)

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: getPythonExpertSystemPrompt()},
		{Role: einoSchema.User, Content: prompt},
	}

	resp, err := p.chatModel.Generate(ctx, msgs)
	if err != nil {
		return fmt.Errorf("code generation failed: %v", err)
	}

	// Extract Python code from response
	content := resp.Content
	code := extractPythonCode(content)

	plan.GeneratedCode = code
	plan.CodeLength = len(strings.Split(code, "\n"))
	plan.SyntaxValid = basicPythonSyntaxCheck(code)

	return nil
}

// PlanAndGenerateCode performs the complete three-phase Python code generation
func (p *PythonPlanner) PlanAndGenerateCode(ctx context.Context, userQuery string, availableContext *AvailableContext) (*PythonPlan, error) {
	// Phase 1: Library & Data Linking
	plan, err := p.Phase1LibraryAndDataLinking(ctx, userQuery, availableContext)
	if err != nil {
		return nil, err
	}

	// Phase 2: Logic Planning
	if err := p.Phase2LogicPlanning(ctx, userQuery, plan, availableContext); err != nil {
		// Non-fatal, continue with code generation
		if p.logger != nil {
			p.logger(fmt.Sprintf("[PYTHON-PLANNER] Logic planning warning: %v", err))
		}
	}

	// Phase 3: Code Generation
	if err := p.Phase3CodeGeneration(ctx, userQuery, plan, availableContext); err != nil {
		return nil, err
	}

	return plan, nil
}

// ValidateAndCorrectCode validates Python code and attempts to correct errors
func (p *PythonPlanner) ValidateAndCorrectCode(ctx context.Context, code string, errorMsg string, availableContext *AvailableContext) (string, error) {
	if p.logger != nil {
		p.logger("[PYTHON-PLANNER] Self-correction: Fixing Python error")
	}

	contextDesc := ""
	if availableContext != nil {
		if len(availableContext.AvailableColumns) > 0 {
			contextDesc += fmt.Sprintf("Available columns: %s\n", strings.Join(availableContext.AvailableColumns, ", "))
		}
		if availableContext.DataSample != "" {
			contextDesc += fmt.Sprintf("\nData sample:\n%s\n", availableContext.DataSample)
		}
	}

	prompt := fmt.Sprintf(`You are a Python expert fixing code errors.

## Original Code
` + "```python\n%s\n```" + `

## Error Message
%s

## Data Context
%s

## Instructions
1. Analyze the error message carefully
2. Common issues to check:
   - Column names (case-sensitive, check against available columns)
   - Library imports (pandas, matplotlib, numpy, etc.)
   - DataFrame operations (proper syntax)
   - File paths (use 'chart.png' for charts)
   - Data types and conversions
   - Syntax errors (indentation, parentheses, quotes)
3. Output the CORRECTED code only
4. Code must still be ≤80 lines

## Output Format
Output ONLY the corrected Python code, wrapped in python code block:
` + "```python\nCORRECTED CODE HERE\n```",
		code, errorMsg, contextDesc)

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: getPythonExpertSystemPrompt()},
		{Role: einoSchema.User, Content: prompt},
	}

	resp, err := p.chatModel.Generate(ctx, msgs)
	if err != nil {
		return "", err
	}

	// Extract Python code from response
	correctedCode := extractPythonCode(resp.Content)
	return correctedCode, nil
}

// Helper functions

// getPythonExpertSystemPrompt returns the system prompt for Python code generation
func getPythonExpertSystemPrompt() string {
	return `## Role
You are a senior Python data analyst, expert in pandas, matplotlib, numpy, and data visualization.

## Constraints
1. NO HALLUCINATION: Only use columns that exist in the data
2. CODE LENGTH: Maximum 80 lines (STRICT)
3. LIBRARIES: Only use pandas, matplotlib, numpy, seaborn, scipy
4. SAFETY: Handle errors, missing values, and edge cases
5. PERFORMANCE: Efficient code, avoid unnecessary loops

## Chart Generation Rules
- Always save charts as 'chart.png' in current directory
- Use plt.savefig('chart.png', dpi=100, bbox_inches='tight')
- Close plot after saving: plt.close()
- Include proper labels, title, legend
- Choose appropriate chart type for data

## Output
- Clean, executable Python code
- Include comments for complex logic
- Print meaningful summaries (not raw data dumps)
- Proper error handling`
}

// extractJSON extracts JSON from markdown code blocks
func extractJSON(content string) string {
	content = strings.TrimSpace(content)

	// Try json code block
	if idx := strings.Index(content, "```json"); idx >= 0 {
		content = content[idx+7:]
		if endIdx := strings.Index(content, "```"); endIdx >= 0 {
			content = content[:endIdx]
		}
	} else if idx := strings.Index(content, "```"); idx >= 0 {
		// Try generic code block
		content = content[idx+3:]
		if endIdx := strings.Index(content, "```"); endIdx >= 0 {
			content = content[:endIdx]
		}
	}

	return strings.TrimSpace(content)
}

// extractPythonCode extracts Python code from markdown code blocks
func extractPythonCode(content string) string {
	pythonRegex := regexp.MustCompile("(?s)```python\\s*(.+?)\\s*```")
	matches := pythonRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try without language tag
	codeRegex := regexp.MustCompile("(?s)```\\s*(.+?)\\s*```")
	matches = codeRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		code := strings.TrimSpace(matches[1])
		// Check if it looks like Python
		if strings.Contains(code, "import") || strings.Contains(code, "def ") || strings.Contains(code, "print(") {
			return code
		}
	}

	// Return entire content as fallback
	return strings.TrimSpace(content)
}

// basicPythonSyntaxCheck performs basic syntax validation
func basicPythonSyntaxCheck(code string) bool {
	// Very basic checks
	lines := strings.Split(code, "\n")

	// Check for basic Python patterns
	hasImport := false
	hasPrint := false

	openParens := 0
	openBrackets := 0
	openBraces := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
			hasImport = true
		}
		if strings.Contains(trimmed, "print(") {
			hasPrint = true
		}

		// Count brackets
		openParens += strings.Count(line, "(") - strings.Count(line, ")")
		openBrackets += strings.Count(line, "[") - strings.Count(line, "]")
		openBraces += strings.Count(line, "{") - strings.Count(line, "}")
	}

	// Basic validation
	if openParens != 0 || openBrackets != 0 || openBraces != 0 {
		return false // Unbalanced brackets
	}

	// Should have at least imports or print
	return hasImport || hasPrint || len(lines) > 5
}
