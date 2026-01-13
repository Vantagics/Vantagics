package templates

import (
	"context"
	"strings"
)

// ProgressCallback for reporting progress during template execution
type ProgressCallback func(stage string, progress int, message string, step, total int)

// TemplateResult holds the result of template execution
type TemplateResult struct {
	Success    bool   `json:"success"`
	Output     string `json:"output"`
	ChartData  string `json:"chart_data,omitempty"`  // Base64 image or ECharts JSON
	ChartType  string `json:"chart_type,omitempty"`  // "image" or "echarts"
	CSVData    string `json:"csv_data,omitempty"`    // Base64 CSV data
	Error      string `json:"error,omitempty"`
}

// DataExecutor interface for executing SQL and Python
type DataExecutor interface {
	ExecuteSQL(ctx context.Context, dataSourceID, query string) ([]map[string]interface{}, error)
	ExecutePython(ctx context.Context, code, workDir string) (string, error)
	GetSchema(ctx context.Context, dataSourceID string) ([]TableInfo, error)
}

// TableInfo holds table metadata
type TableInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

// AnalysisTemplate defines the interface for pre-built analysis templates
type AnalysisTemplate interface {
	// Name returns the template identifier
	Name() string

	// Description returns a human-readable description
	Description() string

	// Keywords returns trigger keywords that detect this template
	Keywords() []string

	// RequiredColumns returns the column types needed (e.g., "date", "customer_id", "amount")
	RequiredColumns() []string

	// CanExecute checks if the template can run with the given schema
	CanExecute(tables []TableInfo) bool

	// Execute runs the template and returns results
	Execute(ctx context.Context, executor DataExecutor, dataSourceID string, onProgress ProgressCallback) (*TemplateResult, error)
}

// Registry holds all available templates
var Registry = make(map[string]AnalysisTemplate)

// Register adds a template to the registry
func Register(template AnalysisTemplate) {
	Registry[template.Name()] = template
}

// DetectTemplate tries to match user input to a template
func DetectTemplate(userMessage string) AnalysisTemplate {
	lower := strings.ToLower(userMessage)

	for _, template := range Registry {
		for _, keyword := range template.Keywords() {
			if strings.Contains(lower, strings.ToLower(keyword)) {
				return template
			}
		}
	}

	return nil
}

// GetTemplate returns a template by name
func GetTemplate(name string) AnalysisTemplate {
	return Registry[name]
}

// ListTemplates returns all registered templates
func ListTemplates() []AnalysisTemplate {
	templates := make([]AnalysisTemplate, 0, len(Registry))
	for _, t := range Registry {
		templates = append(templates, t)
	}
	return templates
}
