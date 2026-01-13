package templates

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// ServiceExecutor adapts existing services to the DataExecutor interface
type ServiceExecutor struct {
	SQLExecutor    func(ctx context.Context, dataSourceID, query string) ([]map[string]interface{}, error)
	PythonExecutor func(code, workDir string) (string, error)
	SchemaGetter   func(dataSourceID string) ([]TableInfo, error)
	WorkDir        string
}

// ExecuteSQL runs a SQL query
func (e *ServiceExecutor) ExecuteSQL(ctx context.Context, dataSourceID, query string) ([]map[string]interface{}, error) {
	if e.SQLExecutor == nil {
		return nil, fmt.Errorf("SQL executor not configured")
	}
	return e.SQLExecutor(ctx, dataSourceID, query)
}

// ExecutePython runs Python code
func (e *ServiceExecutor) ExecutePython(ctx context.Context, code, workDir string) (string, error) {
	if e.PythonExecutor == nil {
		return "", fmt.Errorf("Python executor not configured")
	}

	// Use provided workDir or create temp one
	if workDir == "" {
		var err error
		workDir, err = os.MkdirTemp("", "rapidbi_template_*")
		if err != nil {
			return "", fmt.Errorf("failed to create work dir: %v", err)
		}
		defer os.RemoveAll(workDir)
	}
	e.WorkDir = workDir

	output, err := e.PythonExecutor(code, workDir)

	// Check for generated files and append to output
	output = e.appendGeneratedFiles(output, workDir)

	return output, err
}

// GetSchema returns table information
func (e *ServiceExecutor) GetSchema(ctx context.Context, dataSourceID string) ([]TableInfo, error) {
	if e.SchemaGetter == nil {
		return nil, fmt.Errorf("schema getter not configured")
	}
	return e.SchemaGetter(dataSourceID)
}

// appendGeneratedFiles checks for chart.png and CSV files in workDir
func (e *ServiceExecutor) appendGeneratedFiles(output, workDir string) string {
	// Check for chart.png
	chartPath := filepath.Join(workDir, "chart.png")
	if _, err := os.Stat(chartPath); err == nil {
		chartData, readErr := os.ReadFile(chartPath)
		if readErr == nil {
			encoded := base64.StdEncoding.EncodeToString(chartData)
			output += fmt.Sprintf("\n\n![Chart](data:image/png;base64,%s)", encoded)
		}
	}

	// Check for CSV files
	csvFiles, _ := filepath.Glob(filepath.Join(workDir, "*.csv"))
	if len(csvFiles) > 0 {
		output += "\n\n**📊 Generated Data Files:**\n"
		for _, csvPath := range csvFiles {
			csvData, readErr := os.ReadFile(csvPath)
			if readErr == nil {
				encoded := base64.StdEncoding.EncodeToString(csvData)
				fileName := filepath.Base(csvPath)
				output += fmt.Sprintf("- [📥 Download %s](data:text/csv;base64,%s)\n", fileName, encoded)
			}
		}
	}

	return output
}
