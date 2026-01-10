package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	var in pythonInput
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return "", fmt.Errorf("invalid input: %v", err)
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
	wrappedScript := fmt.Sprintf("import os\nos.chdir(r'%s')\n%s", workDir, in.Code)

	output, err := t.pythonService.ExecuteScript(t.cfg.PythonPath, wrappedScript)
	
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
		return output, fmt.Errorf("python execution error: %v", err)
	}

	return output, nil
}
