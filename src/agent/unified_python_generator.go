package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// UnifiedPythonGenerator generates complete Python analysis code in a single LLM call
type UnifiedPythonGenerator struct {
	chatModel           model.ChatModel
	dsService           *DataSourceService
	schemaBuilder       *SchemaContextBuilder
	promptBuilder       *AnalysisPromptBuilder
	codeValidator       *CodeValidator
	logger              func(string)
	metrics             *AnalysisMetrics
	classificationHints *ClassificationResult
}

// GeneratedCode represents the output of code generation
type GeneratedCode struct {
	Code          string   `json:"code"`
	SQLQueries    []string `json:"sql_queries"`
	OutputType    string   `json:"output_type"`
	EstimatedTime int      `json:"estimated_time_ms"`
	Assumptions   []string `json:"assumptions"`
	HasChart      bool     `json:"has_chart"`
	HasExport     bool     `json:"has_export"`
}

// NewUnifiedPythonGenerator creates a new unified Python generator
func NewUnifiedPythonGenerator(
	chatModel model.ChatModel,
	dsService *DataSourceService,
	logger func(string),
) *UnifiedPythonGenerator {
	return &UnifiedPythonGenerator{
		chatModel:     chatModel,
		dsService:     dsService,
		schemaBuilder: NewSchemaContextBuilder(dsService, 5*time.Minute, logger),
		promptBuilder: NewAnalysisPromptBuilder(),
		codeValidator: NewCodeValidator(),
		logger:        logger,
	}
}

func (g *UnifiedPythonGenerator) log(msg string) {
	if g.logger != nil {
		g.logger(msg)
	}
}

// SetMetrics sets the metrics collector
func (g *UnifiedPythonGenerator) SetMetrics(metrics *AnalysisMetrics) {
	g.metrics = metrics
}

// SetClassificationHints sets the LLM classification hints for better code generation
func (g *UnifiedPythonGenerator) SetClassificationHints(hints *ClassificationResult) {
	g.classificationHints = hints
}

// GenerateAnalysisCode generates complete Python code for data analysis
func (g *UnifiedPythonGenerator) GenerateAnalysisCode(
	ctx context.Context,
	userRequest string,
	dataSourceID string,
	dbPath string,
	sessionDir string,
) (*GeneratedCode, error) {
	startTime := time.Now()

	// 1. Build schema context
	g.log("[UNIFIED_GEN] Building schema context...")
	schemaStart := time.Now()
	schemaCtx, err := g.schemaBuilder.BuildContext(ctx, dataSourceID, userRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to build schema context: %w", err)
	}
	schemaTime := time.Since(schemaStart)
	if g.metrics != nil {
		g.metrics.RecordSchemaFetch(schemaTime)
	}
	g.log(fmt.Sprintf("[UNIFIED_GEN] Schema context built in %v", schemaTime))

	// 2. Determine output format based on request and classification hints
	outputFormat := g.determineOutputFormat(userRequest)
	
	// Override with classification hints if available
	if g.classificationHints != nil {
		if g.classificationHints.NeedsVisualization {
			outputFormat = "visualization"
			g.log("[UNIFIED_GEN] Output format overridden to visualization by LLM classification")
		}
		if g.classificationHints.NeedsDataExport {
			// Add export hint to prompt
			g.log("[UNIFIED_GEN] Data export requested by LLM classification")
		}
	}
	
	g.log(fmt.Sprintf("[UNIFIED_GEN] Output format: %s", outputFormat))

	// 3. Build prompt with classification hints
	prompt := g.promptBuilder.BuildPromptWithHints(userRequest, schemaCtx, outputFormat, g.classificationHints)

	// 4. Call LLM to generate code
	g.log("[UNIFIED_GEN] Calling LLM for code generation...")
	codeGenStart := time.Now()
	
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	response, err := g.chatModel.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM code generation failed: %w", err)
	}

	codeGenTime := time.Since(codeGenStart)
	if g.metrics != nil {
		g.metrics.RecordCodeGeneration(codeGenTime)
		g.metrics.IncrementLLMCalls()
	}
	g.log(fmt.Sprintf("[UNIFIED_GEN] Code generated in %v", codeGenTime))

	// 5. Extract Python code from response
	code := g.extractPythonCode(response.Content)
	if code == "" {
		return nil, fmt.Errorf("no Python code found in LLM response")
	}

	// 6. Post-process code (inject paths)
	code = g.postProcessCode(code, dbPath, sessionDir)

	// 7. Validate code
	g.log("[UNIFIED_GEN] Validating generated code...")
	validationResult := g.codeValidator.ValidateCode(code)
	if !validationResult.Valid {
		return nil, fmt.Errorf("code validation failed: %v", validationResult.Errors)
	}

	// Log warnings if any
	for _, warning := range validationResult.Warnings {
		g.log(fmt.Sprintf("[UNIFIED_GEN] Warning: %s", warning))
	}

	// 8. Build result
	result := &GeneratedCode{
		Code:          code,
		SQLQueries:    validationResult.SQLQueries,
		OutputType:    outputFormat,
		EstimatedTime: g.estimateExecutionTime(code),
		Assumptions:   g.extractAssumptions(code),
		HasChart:      validationResult.HasChart,
		HasExport:     validationResult.HasExport,
	}

	totalTime := time.Since(startTime)
	g.log(fmt.Sprintf("[UNIFIED_GEN] Total generation time: %v", totalTime))

	return result, nil
}

// determineOutputFormat determines the output format based on user request
// More inclusive: default to visualization for most analysis requests
func (g *UnifiedPythonGenerator) determineOutputFormat(request string) string {
	requestLower := strings.ToLower(request)

	// Check for explicit aggregation-only keywords (no visualization needed)
	aggOnlyKeywords := []string{
		"总数", "总计", "计数", "count", "sum total",
		"有多少", "how many",
	}
	isAggOnly := false
	for _, keyword := range aggOnlyKeywords {
		if strings.Contains(requestLower, keyword) {
			isAggOnly = true
			break
		}
	}
	
	// Check for visualization keywords
	vizKeywords := []string{
		"图", "图表", "可视化", "趋势", "分布", "对比", "排名",
		"chart", "visualization", "trend", "distribution", "comparison",
		"柱状图", "折线图", "饼图", "散点图", "热力图",
		"bar", "line", "pie", "scatter", "heatmap",
	}
	for _, keyword := range vizKeywords {
		if strings.Contains(requestLower, keyword) {
			return "visualization"
		}
	}

	// Check for analysis keywords that typically benefit from visualization
	analysisKeywords := []string{
		"分析", "统计", "销售", "收入", "利润", "增长", "下降",
		"按月", "按年", "按季度", "时间", "周期",
		"top", "前", "最", "排行", "占比", "比例",
		"analysis", "sales", "revenue", "growth", "monthly", "yearly",
		"rfm", "cohort", "漏斗", "funnel", "留存", "retention",
	}
	analysisCount := 0
	for _, keyword := range analysisKeywords {
		if strings.Contains(requestLower, keyword) {
			analysisCount++
		}
	}
	
	// If it's an analysis request (not just aggregation), default to visualization
	if analysisCount >= 1 && !isAggOnly {
		return "visualization"
	}

	// Check for aggregation keywords
	aggKeywords := []string{
		"汇总", "统计", "聚合", "分组", "总计", "平均",
		"aggregate", "summary", "group", "total", "average",
	}
	for _, keyword := range aggKeywords {
		if strings.Contains(requestLower, keyword) {
			return "aggregation"
		}
	}

	return "standard"
}

// extractPythonCode extracts Python code from LLM response
func (g *UnifiedPythonGenerator) extractPythonCode(response string) string {
	// Try to extract code from markdown code blocks
	codeBlockPattern := regexp.MustCompile("```(?:python)?\\s*\\n([\\s\\S]*?)\\n```")
	matches := codeBlockPattern.FindStringSubmatch(response)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// If no code block found, try to find code starting with import
	lines := strings.Split(response, "\n")
	var codeLines []string
	inCode := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
			inCode = true
		}
		if inCode {
			codeLines = append(codeLines, line)
		}
	}

	if len(codeLines) > 0 {
		return strings.Join(codeLines, "\n")
	}

	return ""
}

// postProcessCode injects runtime values into the code
func (g *UnifiedPythonGenerator) postProcessCode(code string, dbPath string, sessionDir string) string {
	// Replace placeholders with actual values
	code = strings.ReplaceAll(code, `"{DB_PATH}"`, fmt.Sprintf(`"%s"`, dbPath))
	code = strings.ReplaceAll(code, `'{DB_PATH}'`, fmt.Sprintf(`'%s'`, dbPath))
	code = strings.ReplaceAll(code, `{DB_PATH}`, dbPath)
	
	code = strings.ReplaceAll(code, `"{SESSION_DIR}"`, fmt.Sprintf(`"%s"`, sessionDir))
	code = strings.ReplaceAll(code, `'{SESSION_DIR}'`, fmt.Sprintf(`'%s'`, sessionDir))
	code = strings.ReplaceAll(code, `{SESSION_DIR}`, sessionDir)

	// Also handle common variations
	code = strings.ReplaceAll(code, `DB_PATH = ""`, fmt.Sprintf(`DB_PATH = "%s"`, dbPath))
	code = strings.ReplaceAll(code, `SESSION_DIR = ""`, fmt.Sprintf(`SESSION_DIR = "%s"`, sessionDir))

	return code
}

// estimateExecutionTime estimates the execution time based on code complexity
func (g *UnifiedPythonGenerator) estimateExecutionTime(code string) int {
	baseTime := 1000 // 1 second base

	// Add time for visualization
	if strings.Contains(code, "plt.") || strings.Contains(code, "sns.") {
		baseTime += 2000
	}

	// Add time for complex operations
	if strings.Contains(code, "GROUP BY") || strings.Contains(code, "group by") {
		baseTime += 500
	}

	// Add time for joins
	joinCount := strings.Count(strings.ToUpper(code), "JOIN")
	baseTime += joinCount * 300

	return baseTime
}

// extractAssumptions extracts assumptions from code comments
func (g *UnifiedPythonGenerator) extractAssumptions(code string) []string {
	var assumptions []string

	// Look for assumption comments
	assumptionPattern := regexp.MustCompile(`#\s*(?:假设|Assumption|Note)[:：]\s*(.+)`)
	matches := assumptionPattern.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			assumptions = append(assumptions, strings.TrimSpace(match[1]))
		}
	}

	return assumptions
}

// InvalidateSchemaCache invalidates the schema cache for a data source
func (g *UnifiedPythonGenerator) InvalidateSchemaCache(dataSourceID string) {
	g.schemaBuilder.InvalidateCache(dataSourceID)
}
