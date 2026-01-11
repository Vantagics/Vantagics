package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"rapidbi/config"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type PythonExecutorTool struct {
	pythonService PythonExecutor
	cfg           config.Config
}

func NewPythonExecutorTool(cfg config.Config) *PythonExecutorTool {
	return &PythonExecutorTool{
		pythonService: &PythonService{},
		cfg:           cfg,
	}
}

type pythonInput struct {
	Code string `json:"code"`
}

func (t *PythonExecutorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "python_executor",
		Desc: "Execute Python code for data analysis or visualization. Use pandas for data and matplotlib/seaborn for charts. Always save any generated plots as 'chart.png' in the current working directory. The tool returns stdout/stderr and will automatically detect 'chart.png'.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"code": {
				Type:     schema.String,
				Desc:     "The Python code to execute. Must be valid Python 3 code.",
				Required: true,
			},
		}),
	}, nil
}

func (t *PythonExecutorTool) InvokableRun(ctx context.Context, input string, opts ...tool.Option) (string, error) {
	// Check if input looks truncated (doesn't end with })
	trimmed := strings.TrimSpace(input)
	if !strings.HasSuffix(trimmed, "}") && !strings.HasSuffix(trimmed, "}\"") {
		return "", fmt.Errorf("Tool input truncated - Python code too long.\n\n" +
			"Please break this analysis into smaller steps:\n" +
			"1. Data Preparation: Load and clean data (1 tool call)\n" +
			"2. Calculation: Compute metrics/scores (1 tool call)\n" +
			"3. Visualization: Create charts (1 tool call)\n" +
			"4. Summary: Print key findings (1 tool call)\n\n" +
			"Each step should be under 80 lines of code. Use simple, focused operations.")
	}

	var in pythonInput
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		// Enhanced error reporting for debugging
		// Write full input to temp file for inspection
		tmpFile, tmpErr := os.CreateTemp("", "rapidbi_python_input_*.json")
		if tmpErr == nil {
			tmpFile.WriteString(input)
			tmpFile.Close()
			return "", fmt.Errorf("invalid input: %v. Full input saved to: %s", err, tmpFile.Name())
		}

		truncated := input
		if len(input) > 500 {
			truncated = input[:500] + "... (truncated)"
		}
		return "", fmt.Errorf("invalid input: %v. Input received (first 500 chars): %s", err, truncated)
	}

	if t.cfg.PythonPath == "" {
		return "", fmt.Errorf("python path is not configured. Please set it in Settings -> Python Environment")
	}

	// Create temp working directory
	workDir, err := os.MkdirTemp("", "rapidbi_py_*")
	if err != nil {
		return "", fmt.Errorf("failed to create work dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	// Wrap script to change directory so chart.png goes to workDir
	// Also force matplotlib to use Agg backend to prevent GUI popups
	// Pre-import common libraries to avoid NameError
	wrappedScript := fmt.Sprintf(`import os
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import pandas as pd
import numpy as np
os.chdir(r'%s')
%s`, workDir, in.Code)

	output, err := t.pythonService.ExecuteScript(t.cfg.PythonPath, wrappedScript)

	// Check for common data loading errors
	if err != nil && strings.Contains(output, "NameError") &&
	   (strings.Contains(output, "df") || strings.Contains(output, "data")) {
		output += "\n\n💡 HINT: It looks like you're using a DataFrame variable that doesn't exist. "
		output += "Remember to FIRST use execute_sql to query data, then LOAD it in Python:\n"
		output += "   data = json.loads('''<SQL result JSON>''')\n"
		output += "   df = pd.DataFrame(data)\n"
	}

	// Check for chart.png
	chartPath := filepath.Join(workDir, "chart.png")
	if _, statErr := os.Stat(chartPath); statErr == nil {
		chartData, readErr := os.ReadFile(chartPath)
		if readErr == nil {
			encoded := base64.StdEncoding.EncodeToString(chartData)
			output += fmt.Sprintf("\n\n![Chart](data:image/png;base64,%s)", encoded)
		}
	}

	if err != nil {
		return output, fmt.Errorf("python execution error: %v\nOutput:\n%s", err, output)
	}

	return output, nil
}
