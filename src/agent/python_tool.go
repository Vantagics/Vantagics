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
	pool          *PythonPool
}

func NewPythonExecutorTool(cfg config.Config) *PythonExecutorTool {
	return &PythonExecutorTool{
		pythonService: &PythonService{},
		cfg:           cfg,
	}
}

// NewPythonExecutorToolWithPool creates a tool with a shared pool for better performance
func NewPythonExecutorToolWithPool(cfg config.Config, pool *PythonPool) *PythonExecutorTool {
	return &PythonExecutorTool{
		pythonService: &PythonService{},
		cfg:           cfg,
		pool:          pool,
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
		// Return as SUCCESS with guidance so LLM continues
		guidance := "⚠️ CODE TOO LONG - Automatically proceeding with Step 1 only.\n\n" +
			"I detected you tried to write >80 lines. For RFM/clustering, I'll do Step 1 first:\n\n" +
			"📋 NEXT ACTION: Execute python_executor with this EXACT code:\n" +
			"```python\n" +
			"import json\n" +
			"import pandas as pd\n\n" +
			"# Load the SQL result from previous execute_sql call\n" +
			"data = json.loads('''PASTE_THE_JSON_FROM_SQL_RESULT_HERE''')\n" +
			"df = pd.DataFrame(data)\n\n" +
			"# Calculate RFM scores\n" +
			"ref_date = df['OrderDate'].max()\n" +
			"rfm = df.groupby('CustomerID').agg({\n" +
			"    'OrderDate': lambda x: (ref_date - x.max()).days,  # Recency in days\n" +
			"    'OrderID': 'count',  # Frequency of orders\n" +
			"    'TotalAmount': 'sum'  # Monetary value\n" +
			"}).rename(columns={'OrderDate': 'R', 'OrderID': 'F', 'TotalAmount': 'M'})\n\n" +
			"print('RFM Scores calculated:')\n" +
			"print(rfm.describe())\n" +
			"print(f'\\nTotal customers analyzed: {len(rfm)}')\n" +
			"```\n\n" +
			"🔄 After this succeeds, we'll do Step 2 (segmentation), Step 3 (visualization), and Step 4 (summary) separately."

		// Return this as a successful tool result so the LLM sees it and continues
		return guidance, nil
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
	// Use base64 encoding to safely pass user code (avoids quote conflicts)

	// Encode user code to base64 to avoid any string delimiter conflicts
	encodedCode := base64.StdEncoding.EncodeToString([]byte(in.Code))

	wrappedScript := fmt.Sprintf(`import os
import sys
import traceback
import base64
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import pandas as pd
import numpy as np
os.chdir(r'%s')

# Helper function to safely access DataFrame columns with debugging
def safe_column_access(df, col_name, row_context=""):
    if col_name not in df.columns:
        print(f"\n❌ ERROR: Column '{col_name}' not found in DataFrame!")
        print(f"📋 Available columns: {list(df.columns)}")
        if row_context:
            print(f"🔍 Context: {row_context}")
        print(f"\n💡 HINT: You may need to create this column before accessing it.")
        print(f"   Example: df['{col_name}'] = <your calculation here>")
        raise KeyError(f"Column '{col_name}' not found. Available: {list(df.columns)}")
    return df[col_name]

# Decode user code from base64 to avoid quote conflicts
user_code = base64.b64decode('%s').decode('utf-8')

try:
    exec(user_code)
except KeyError as e:
    # Enhanced KeyError handling for DataFrame column issues
    if "'revenue_share'" in str(e) or "'revenue_share'" in traceback.format_exc():
        print(f"\n❌ KeyError: {e}")
        print("\n💡 FIX REQUIRED: The 'revenue_share' column was not calculated.")
        print("   Add this BEFORE trying to access it:")
        print("   total_revenue = df_cat['total_revenue'].sum()")
        print("   df_cat['revenue_share'] = (df_cat['total_revenue'] / total_revenue * 100)")
        sys.exit(1)
    else:
        print(f"\n❌ KeyError: {e}")
        print(traceback.format_exc())
        sys.exit(1)
except Exception as e:
    print(f"\n❌ Error: {e}")
    print(traceback.format_exc())
    sys.exit(1)
`, workDir, encodedCode)

	var output string

	// Use pool if available for faster execution
	if t.pool != nil {
		output, err = t.pool.Execute(in.Code, workDir)
	} else {
		output, err = t.pythonService.ExecuteScript(t.cfg.PythonPath, wrappedScript)
	}

	// Check for common data loading errors
	if err != nil && strings.Contains(output, "NameError") &&
	   (strings.Contains(output, "df") || strings.Contains(output, "data")) {
		output += "\n\n💡 HINT: It looks like you're using a DataFrame variable that doesn't exist. "
		output += "Remember to FIRST use execute_sql to query data, then LOAD it in Python:\n"
		output += "   data = json.loads('''<SQL result JSON>''')\n"
		output += "   df = pd.DataFrame(data)\n"
	}

	// Check for revenue_share KeyError (common in market basket analysis)
	if err != nil && strings.Contains(output, "revenue_share") && strings.Contains(output, "KeyError") {
		output += "\n\n🔧 AUTOMATIC FIX SUGGESTION:\n"
		output += "The AI agent should regenerate the Python code with proper column calculation.\n"
		output += "Before accessing df_cat['revenue_share'], add:\n"
		output += "   total_revenue = df_cat['total_revenue'].sum()\n"
		output += "   df_cat['revenue_share'] = (df_cat['total_revenue'] / total_revenue * 100)\n\n"
		output += "⚠️ Agent should retry with corrected code."
	}

	// Check for general KeyError with helpful context
	if err != nil && strings.Contains(output, "KeyError") && !strings.Contains(output, "revenue_share") {
		output += "\n\n💡 DEBUGGING TIP: A column or key was accessed that doesn't exist.\n"
		output += "Common fixes:\n"
		output += "   1. Check DataFrame columns: print(df.columns)\n"
		output += "   2. Create missing column before accessing it\n"
		output += "   3. Verify the SQL query returned expected columns\n"
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

	// Check for CSV files (common patterns: *.csv, rfm*.csv, result*.csv, etc.)
	csvFiles, _ := filepath.Glob(filepath.Join(workDir, "*.csv"))
	if len(csvFiles) > 0 {
		output += "\n\n**📊 Generated Data Files:**\n"
		for _, csvPath := range csvFiles {
			csvData, readErr := os.ReadFile(csvPath)
			if readErr == nil {
				// Convert CSV to base64 for download
				encoded := base64.StdEncoding.EncodeToString(csvData)
				fileName := filepath.Base(csvPath)
				// Create a markdown link with data URI for download
				output += fmt.Sprintf("- [📥 Download %s](data:text/csv;base64,%s)\n", fileName, encoded)

				// Also show a preview of first few lines if CSV is not too large
				if len(csvData) < 5000 {
					lines := strings.Split(string(csvData), "\n")
					if len(lines) > 10 {
						preview := strings.Join(lines[:10], "\n")
						output += fmt.Sprintf("\nPreview (first 10 rows):\n```csv\n%s\n...\n```\n", preview)
					} else {
						output += fmt.Sprintf("\nContent:\n```csv\n%s\n```\n", string(csvData))
					}
				}
			}
		}
	}

	if err != nil {
		return output, fmt.Errorf("python execution error: %v\nOutput:\n%s", err, output)
	}

	return output, nil
}
