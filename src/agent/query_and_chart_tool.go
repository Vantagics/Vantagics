package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// QueryAndChartTool combines SQL execution + Python chart generation in a single tool call.
// This eliminates the most common 2-step pattern: execute_sql �python_executor,
// saving 1-2 agent iterations per visualization request.
type QueryAndChartTool struct {
	sqlTool    *SQLExecutorTool
	pythonTool *PythonExecutorTool
	logger     func(string)
}

func NewQueryAndChartTool(sqlTool *SQLExecutorTool, pythonTool *PythonExecutorTool, logger func(string)) *QueryAndChartTool {
	return &QueryAndChartTool{
		sqlTool:    sqlTool,
		pythonTool: pythonTool,
		logger:     logger,
	}
}

type queryAndChartInput struct {
	DataSourceID string `json:"data_source_id"`
	Query        string `json:"query"`
	ChartCode    string `json:"chart_code"`
	ChartTitle   string `json:"chart_title,omitempty"`
}

func (t *QueryAndChartTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "query_and_chart",
		Desc: `Execute a SQL query AND generate a chart from the results in ONE step.

**Use this instead of calling execute_sql + python_executor separately!**
This saves a round-trip and is the preferred way to create visualizations.

The tool will:
1. Execute the SQL query against the data source
2. Pass the query results as a pandas DataFrame variable named 'df' to the Python code
3. Execute the Python chart code with matplotlib/seaborn

**Python code requirements:**
- The query results are pre-loaded as a pandas DataFrame named 'df'
- Use matplotlib (plt) for charts �it's pre-imported
- Call plt.savefig() at the end to save the chart
- Use plt.tight_layout() before saving
- Set Chinese font if labels contain Chinese: plt.rcParams['font.sans-serif'] = ['SimHei', 'Arial Unicode MS', 'DejaVu Sans']

**Example:**
{
  "data_source_id": "abc123",
  "query": "SELECT category, SUM(sales) as total FROM orders GROUP BY category ORDER BY total DESC LIMIT 10",
  "chart_code": "import matplotlib.pyplot as plt\nplt.rcParams['font.sans-serif'] = ['SimHei', 'Arial Unicode MS']\nplt.figure(figsize=(10,6))\nplt.bar(df['category'], df['total'])\nplt.title('Sales by Category')\nplt.xticks(rotation=45)\nplt.tight_layout()\nplt.savefig('chart.png', dpi=150)",
  "chart_title": "Sales by Category"
}

**When to use:**
- User asks for a chart/visualization with data from the database
- Any request that needs both SQL data retrieval AND a chart
- Replaces the pattern: execute_sql �python_executor`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"data_source_id": {
				Type:     schema.String,
				Desc:     "The ID of the data source to query.",
				Required: true,
			},
			"query": {
				Type:     schema.String,
				Desc:     "SQL SELECT query to execute. Results will be available as 'df' DataFrame in chart_code.",
				Required: true,
			},
			"chart_code": {
				Type:     schema.String,
				Desc:     "Python code to generate the chart. Query results are pre-loaded as pandas DataFrame 'df'. Use matplotlib for plotting.",
				Required: true,
			},
			"chart_title": {
				Type:     schema.String,
				Desc:     "Optional title for the chart (used in logging).",
				Required: false,
			},
		}),
	}, nil
}

func (t *QueryAndChartTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input queryAndChartInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("invalid input: %v", err)
	}

	if t.logger != nil {
		title := input.ChartTitle
		if title == "" {
			title = "(untitled)"
		}
		t.logger(fmt.Sprintf("[QUERY-AND-CHART] Starting composite tool: chart=%s", title))
	}

	// Step 1: Execute SQL
	sqlInput, _ := json.Marshal(map[string]string{
		"data_source_id": input.DataSourceID,
		"query":          input.Query,
	})

	sqlResult, err := t.sqlTool.InvokableRun(ctx, string(sqlInput), opts...)
	if err != nil {
		return fmt.Sprintf(`{"error": "SQL execution failed: %s", "stage": "sql"}`, err.Error()), nil
	}

	// Check if SQL result contains an error
	if strings.Contains(sqlResult, `"error"`) {
		var sqlErr map[string]interface{}
		if json.Unmarshal([]byte(sqlResult), &sqlErr) == nil {
			if errMsg, ok := sqlErr["error"]; ok {
				return fmt.Sprintf(`{"error": "SQL execution failed: %v", "stage": "sql", "sql_result": %s}`, errMsg, sqlResult), nil
			}
		}
	}

	// Parse SQL result to get row count for logging
	var sqlData map[string]interface{}
	rowCount := 0
	if json.Unmarshal([]byte(sqlResult), &sqlData) == nil {
		if rows, ok := sqlData["rows"]; ok {
			if rowArr, ok := rows.([]interface{}); ok {
				rowCount = len(rowArr)
			}
		}
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[QUERY-AND-CHART] SQL returned %d rows, generating chart...", rowCount))
	}

	// Step 2: Build Python code that loads SQL results into DataFrame and runs chart code
	// We inject the SQL result as JSON and parse it into a DataFrame
	pythonCode := buildChartPythonCode(sqlResult, input.ChartCode)

	pyInput, _ := json.Marshal(map[string]string{
		"code": pythonCode,
	})

	pyResult, err := t.pythonTool.InvokableRun(ctx, string(pyInput), opts...)
	if err != nil {
		// Return SQL results even if chart fails �the data is still useful
		return fmt.Sprintf(`{"sql_result": %s, "chart_error": "%s", "stage": "chart"}`, sqlResult, err.Error()), nil
	}

	if t.logger != nil {
		t.logger("[QUERY-AND-CHART] Composite tool completed successfully")
	}

	// Combine results
	result := map[string]interface{}{
		"sql_rows":     rowCount,
		"chart_result": pyResult,
		"success":      true,
	}

	// Include a summary of the SQL data (first few rows) for the LLM to reference
	if rowCount > 0 && rowCount <= 5 {
		result["sql_data_preview"] = sqlResult
	} else if rowCount > 5 {
		result["sql_data_note"] = fmt.Sprintf("Query returned %d rows. Chart has been generated from the full dataset.", rowCount)
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// buildChartPythonCode wraps the user's chart code with DataFrame loading from SQL results
func buildChartPythonCode(sqlResultJSON string, chartCode string) string {
	// Use base64 encoding to safely embed JSON in Python code, avoiding string escaping issues
	encoded := base64.StdEncoding.EncodeToString([]byte(sqlResultJSON))

	return fmt.Sprintf(`import pandas as pd
import json
import base64
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

# Load SQL query results into DataFrame
_sql_result = json.loads(base64.b64decode("%s").decode("utf-8"))
if isinstance(_sql_result, list):
    df = pd.DataFrame(_sql_result)
elif "rows" in _sql_result and _sql_result["rows"]:
    df = pd.DataFrame(_sql_result["rows"])
elif "data" in _sql_result:
    df = pd.DataFrame(_sql_result["data"])
else:
    df = pd.DataFrame()

print(f"DataFrame loaded: {len(df)} rows, {len(df.columns)} columns")
if len(df) > 0:
    print(f"Columns: {list(df.columns)}")
    print(df.head(3).to_string())

# User chart code
%s
`, encoded, chartCode)
}
