package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"vantagedata/config"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type PythonExecutorTool struct {
	pythonService     PythonExecutor
	cfg               config.Config
	pool              *PythonPool
	errorKnowledge    *ErrorKnowledge
	sessionDir        string // Directory to save session files
	requestID         string // Request ID for unique file naming
	onFileSaved       func(fileName, fileType string, fileSize int64) // Callback when file is saved
	executionRecorder *ExecutionRecorder // Records Python executions for replay
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

// SetErrorKnowledge injects the error knowledge system
func (t *PythonExecutorTool) SetErrorKnowledge(ek *ErrorKnowledge) {
	t.errorKnowledge = ek
}

// SetExecutionRecorder injects the execution recorder
func (t *PythonExecutorTool) SetExecutionRecorder(recorder *ExecutionRecorder) {
	t.executionRecorder = recorder
}

// SetSessionDirectory sets the directory where session files should be saved
func (t *PythonExecutorTool) SetSessionDirectory(dir string) {
	t.sessionDir = dir
}

// SetRequestID sets the request ID for unique file naming
func (t *PythonExecutorTool) SetRequestID(requestID string) {
	t.requestID = requestID
}

// SetFileSavedCallback sets the callback for when files are saved
func (t *PythonExecutorTool) SetFileSavedCallback(callback func(fileName, fileType string, fileSize int64)) {
	t.onFileSaved = callback
}

type pythonInput struct {
	Code string `json:"code"`
}

func (t *PythonExecutorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "python_executor",
		Desc: `Execute Python code for data analysis or visualization.

**IMPORTANT - File Saving:**
- Use FILES_DIR variable for saving files (it will be automatically injected)
- Charts: plt.savefig(os.path.join(FILES_DIR, 'chart.png'), dpi=150)
- Excel: df.to_excel(os.path.join(FILES_DIR, 'data.xlsx'), index=False)
- CSV: df.to_csv(os.path.join(FILES_DIR, 'data.csv'), index=False)

**Code Template:**
` + "```python" + `
import os
import pandas as pd
import matplotlib.pyplot as plt
matplotlib.use('Agg')

FILES_DIR = ""  # Will be injected automatically

# Ensure directory exists
os.makedirs(FILES_DIR, exist_ok=True)

# Your analysis code here...

# Save chart
chart_path = os.path.join(FILES_DIR, 'chart.png')
plt.savefig(chart_path, dpi=150, bbox_inches='tight')
print(f"✅ Chart saved: {chart_path}")
` + "```" + `

The tool returns stdout/stderr and will automatically detect generated files.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"code": {
				Type:     schema.String,
				Desc:     "The Python code to execute. Must be valid Python 3 code. Use FILES_DIR for file paths.",
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
		// Return error as content so LLM can handle it
		truncated := input
		if len(input) > 500 {
			truncated = input[:500] + "... (truncated)"
		}
		return fmt.Sprintf("❌ Error: Invalid input format: %v\n\nInput received (first 500 chars):\n%s\n\n💡 Please provide valid JSON with a 'code' field containing Python code.", err, truncated), nil
	}

	// Inject FILES_DIR and SESSION_DIR into the code if it contains placeholders
	code := in.Code
	if t.sessionDir != "" {
		sessionDirNormalized := strings.ReplaceAll(t.sessionDir, "\\", "/")
		filesDir := sessionDirNormalized + "/files"
		
		// Replace FILES_DIR placeholders
		code = strings.ReplaceAll(code, `"{FILES_DIR}"`, fmt.Sprintf(`"%s"`, filesDir))
		code = strings.ReplaceAll(code, `'{FILES_DIR}'`, fmt.Sprintf(`'%s'`, filesDir))
		code = strings.ReplaceAll(code, `{FILES_DIR}`, fmt.Sprintf(`"%s"`, filesDir))
		code = strings.ReplaceAll(code, `FILES_DIR = ""`, fmt.Sprintf(`FILES_DIR = "%s"`, filesDir))
		code = strings.ReplaceAll(code, `FILES_DIR = ''`, fmt.Sprintf(`FILES_DIR = '%s'`, filesDir))
		code = strings.ReplaceAll(code, `FILES_DIR = None`, fmt.Sprintf(`FILES_DIR = "%s"`, filesDir))
		code = strings.ReplaceAll(code, `files_dir = ""`, fmt.Sprintf(`files_dir = "%s"`, filesDir))
		code = strings.ReplaceAll(code, `files_dir = ''`, fmt.Sprintf(`files_dir = '%s'`, filesDir))
		
		// Replace SESSION_DIR placeholders (also map to files dir for backward compatibility)
		code = strings.ReplaceAll(code, `"{SESSION_DIR}"`, fmt.Sprintf(`"%s"`, filesDir))
		code = strings.ReplaceAll(code, `'{SESSION_DIR}'`, fmt.Sprintf(`'%s'`, filesDir))
		code = strings.ReplaceAll(code, `{SESSION_DIR}`, fmt.Sprintf(`"%s"`, filesDir))
		code = strings.ReplaceAll(code, `SESSION_DIR = ""`, fmt.Sprintf(`SESSION_DIR = "%s"`, filesDir))
		code = strings.ReplaceAll(code, `SESSION_DIR = ''`, fmt.Sprintf(`SESSION_DIR = '%s'`, filesDir))
		code = strings.ReplaceAll(code, `SESSION_DIR = None`, fmt.Sprintf(`SESSION_DIR = "%s"`, filesDir))
		code = strings.ReplaceAll(code, `session_dir = ""`, fmt.Sprintf(`session_dir = "%s"`, filesDir))
		code = strings.ReplaceAll(code, `session_dir = ''`, fmt.Sprintf(`session_dir = '%s'`, filesDir))
	}
	in.Code = code

	if t.cfg.PythonPath == "" {
		return "❌ Error: Python path is not configured.\n\n💡 Please set it in Settings -> Python Environment.", nil
	}

	// Don't clear old chart files - keep all files from all user requests
	// Each file will have a unique name with timestamp or message ID
	// This allows users to download all generated files from the session

	// Create temp working directory
	workDir, err := os.MkdirTemp("", "vantagedata_py_*")
	if err != nil {
		return fmt.Sprintf("❌ Error: Failed to create work dir: %v", err), nil
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
# Configure Chinese font support with multiple fallbacks
plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei', 'SimSun', 'KaiTi', 'FangSong', 'STSong', 'STKaiti', 'STFangsong', 'DejaVu Sans', 'Arial Unicode MS']
plt.rcParams['axes.unicode_minus'] = False  # Fix minus sign display
plt.rcParams['font.family'] = 'sans-serif'  # Ensure sans-serif is used
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
	executionContext := fmt.Sprintf("Executing Python code: %s", truncateString(in.Code, 200))

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

	// If there's an error, check error knowledge for similar issues
	if err != nil && t.errorKnowledge != nil {
		// Extract error type from output
		errorType := "python"
		errorMsg := output
		if len(errorMsg) > 500 {
			errorMsg = errorMsg[:500]
		}

		// Query for similar errors
		hints := t.errorKnowledge.FormatHintsForLLM(errorType, errorMsg)
		if hints != "" {
			output += hints
		}

		// Record the error
		t.errorKnowledge.RecordError(errorType, errorMsg, executionContext, "Execution failed", false)
	}

	// Helper function to save file to session directory
	saveToSession := func(srcPath, fileName, fileType string) (string, error) {
		if t.sessionDir == "" {
			return "", nil // No session directory configured, skip
		}

		// Ensure session files directory exists
		filesDir := filepath.Join(t.sessionDir, "files")
		if err := os.MkdirAll(filesDir, 0755); err != nil {
			return "", err
		}

		// Read file
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return "", err
		}

		// Generate unique filename with request ID to prevent overwriting
		// Format: requestId_originalName.ext (e.g., msg_123_chart.png)
		// If no request ID, fall back to timestamp
		ext := filepath.Ext(fileName)
		baseName := strings.TrimSuffix(fileName, ext)
		var uniqueFileName string
		if t.requestID != "" {
			uniqueFileName = fmt.Sprintf("%s_%s%s", t.requestID, baseName, ext)
		} else {
			// Fallback to timestamp if no request ID
			timestamp := time.Now().UnixNano() / 1000000 // milliseconds
			uniqueFileName = fmt.Sprintf("%s_%d%s", baseName, timestamp, ext)
		}
		
		destPath := filepath.Join(filesDir, uniqueFileName)

		// Write file to session directory
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return "", err
		}

		// Notify callback
		if t.onFileSaved != nil {
			fileInfo, _ := os.Stat(destPath)
			finalName := filepath.Base(destPath)
			t.onFileSaved(finalName, fileType, fileInfo.Size())
		}

		return filepath.Base(destPath), nil
	}

	// Check for chart.png
	chartPath := filepath.Join(workDir, "chart.png")
	if _, statErr := os.Stat(chartPath); statErr == nil {
		chartData, readErr := os.ReadFile(chartPath)
		if readErr == nil {
			// Save to session directory if configured
			savedName, saveErr := saveToSession(chartPath, "chart.png", "image")
			if saveErr == nil && savedName != "" {
				// Reference the saved file
				output += fmt.Sprintf("\n\n📊 **Chart saved:** `files/%s`\n", savedName)
			}

			// Always include base64 for inline display
			encoded := base64.StdEncoding.EncodeToString(chartData)
			output += fmt.Sprintf("![Chart](data:image/png;base64,%s)", encoded)
		}
	}

	// Check for CSV files (common patterns: *.csv, rfm*.csv, result*.csv, etc.)
	csvFiles, _ := filepath.Glob(filepath.Join(workDir, "*.csv"))
	if len(csvFiles) > 0 {
		output += "\n\n**📊 Generated Data Files:**\n"
		for _, csvPath := range csvFiles {
			csvData, readErr := os.ReadFile(csvPath)
			if readErr == nil {
				fileName := filepath.Base(csvPath)

				// Save to session directory if configured
				savedName, saveErr := saveToSession(csvPath, fileName, "csv")
				if saveErr == nil && savedName != "" {
					output += fmt.Sprintf("- 📁 **%s** (saved to session)\n", savedName)
				}

				// Convert CSV to base64 for download
				encoded := base64.StdEncoding.EncodeToString(csvData)
				// Create a markdown link with data URI for download
				output += fmt.Sprintf("  [📥 Download](data:text/csv;base64,%s)\n", encoded)

				// Also show a preview of first few lines if CSV is not too large
				if len(csvData) < 5000 {
					lines := strings.Split(string(csvData), "\n")
					if len(lines) > 10 {
						preview := strings.Join(lines[:10], "\n")
						output += fmt.Sprintf("\n  Preview (first 10 rows):\n```csv\n%s\n...\n```\n", preview)
					} else {
						output += fmt.Sprintf("\n  Content:\n```csv\n%s\n```\n", string(csvData))
					}
				}
			}
		}
	}

	// If execution succeeded, record it as a successful attempt (useful for learning patterns)
	if err == nil && t.errorKnowledge != nil {
		// We could record successful patterns here, but for now just log
		// This would be useful for building a knowledge base of working code patterns
	}
	
	// Record execution for replay
	if t.executionRecorder != nil {
		success := err == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		// Generate step description from code
		stepDescription := t.generateStepDescription(in.Code)
		t.executionRecorder.RecordPython(in.Code, success, errorMsg, output, stepDescription)
	}

	if err != nil {
		// Return error as content so LLM can retry
		return fmt.Sprintf("❌ Python execution failed: %v\n\n%s\n\n💡 Please fix the code and try again.", err, output), nil
	}

	return output, nil
}


// generateStepDescription generates a human-readable description of what the Python code does
func (t *PythonExecutorTool) generateStepDescription(code string) string {
	codeLower := strings.ToLower(code)
	
	// Check for common patterns
	if strings.Contains(codeLower, "plt.") || strings.Contains(codeLower, "matplotlib") {
		if strings.Contains(codeLower, "savefig") {
			return "Generate and save chart"
		}
		return "Generate chart"
	} else if strings.Contains(codeLower, "to_csv") {
		return "Export data to CSV"
	} else if strings.Contains(codeLower, "groupby") || strings.Contains(codeLower, "agg(") {
		return "Aggregate and analyze data"
	} else if strings.Contains(codeLower, "merge") || strings.Contains(codeLower, "join") {
		return "Merge datasets"
	} else if strings.Contains(codeLower, "sort") {
		return "Sort data"
	} else if strings.Contains(codeLower, "filter") || strings.Contains(codeLower, "[") && strings.Contains(codeLower, "]") {
		return "Filter data"
	} else if strings.Contains(codeLower, "describe()") || strings.Contains(codeLower, "info()") {
		return "Analyze data statistics"
	} else if strings.Contains(codeLower, "pd.dataframe") {
		return "Process data"
	}
	
	return "Execute Python code"
}
