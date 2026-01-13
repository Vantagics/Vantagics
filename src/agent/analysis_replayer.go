package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// AnalysisReplayer replays recorded analysis steps
type AnalysisReplayer struct {
	recording      *AnalysisRecording
	config         *ReplayConfig
	dataSource     *DataSourceService
	sqlExecutor    *SQLExecutorTool
	pythonExecutor *PythonExecutorTool
	llmService     *LLMService
	fieldMappings  []TableMapping
	logger         func(string)
}

// NewAnalysisReplayer creates a new analysis replayer
func NewAnalysisReplayer(
	recording *AnalysisRecording,
	config *ReplayConfig,
	dataSource *DataSourceService,
	sqlExecutor *SQLExecutorTool,
	pythonExecutor *PythonExecutorTool,
	llmService *LLMService,
	logger func(string),
) *AnalysisReplayer {
	if config.MaxFieldDiff == 0 {
		config.MaxFieldDiff = 2 // Default max field difference
	}

	return &AnalysisReplayer{
		recording:      recording,
		config:         config,
		dataSource:     dataSource,
		sqlExecutor:    sqlExecutor,
		pythonExecutor: pythonExecutor,
		llmService:     llmService,
		fieldMappings:  []TableMapping{},
		logger:         logger,
	}
}

// log helper function
func (r *AnalysisReplayer) log(msg string) {
	if r.logger != nil {
		r.logger(msg)
	}
}

// AnalyzeSchemaCompatibility compares source and target schemas
func (r *AnalysisReplayer) AnalyzeSchemaCompatibility() ([]TableMapping, error) {
	r.log("Analyzing schema compatibility...")

	// Get target data source schema
	tables, err := r.dataSource.GetDataSourceTables(r.config.TargetSourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target tables: %w", err)
	}

	targetSchemas := []ReplayTableSchema{}
	for _, tableName := range tables {
		// Get first row to extract columns
		data, err := r.dataSource.GetDataSourceTableData(r.config.TargetSourceID, tableName, 1)
		if err != nil {
			r.log(fmt.Sprintf("Warning: Failed to get columns for table %s: %v", tableName, err))
			continue
		}

		var cols []string
		if len(data) > 0 {
			for k := range data[0] {
				cols = append(cols, k)
			}
		}

		targetSchemas = append(targetSchemas, ReplayTableSchema{
			TableName: tableName,
			Columns:   cols,
		})
	}

	// Match source tables to target tables
	mappings := []TableMapping{}
	for _, sourceTable := range r.recording.SourceSchema {
		// Find matching target table (exact name match or similar)
		targetTable := r.findMatchingTable(sourceTable.TableName, targetSchemas)
		if targetTable == nil {
			r.log(fmt.Sprintf("Warning: No matching table found for source table '%s'", sourceTable.TableName))
			continue
		}

		// Match fields
		fieldMappings, err := r.matchFields(sourceTable.Columns, targetTable.Columns)
		if err != nil {
			r.log(fmt.Sprintf("Warning: Failed to match fields for table %s: %v", sourceTable.TableName, err))
			continue
		}

		mapping := TableMapping{
			SourceTable: sourceTable.TableName,
			TargetTable: targetTable.TableName,
			Mappings:    fieldMappings,
		}

		mappings = append(mappings, mapping)
		r.log(fmt.Sprintf("Mapped table '%s' -> '%s' with %d field mappings",
			sourceTable.TableName, targetTable.TableName, len(fieldMappings)))
	}

	r.fieldMappings = mappings
	return mappings, nil
}

// findMatchingTable finds a matching target table for a source table
func (r *AnalysisReplayer) findMatchingTable(sourceTableName string, targetSchemas []ReplayTableSchema) *ReplayTableSchema {
	// First try exact match
	for i := range targetSchemas {
		if targetSchemas[i].TableName == sourceTableName {
			return &targetSchemas[i]
		}
	}

	// Try case-insensitive match
	sourceLower := strings.ToLower(sourceTableName)
	for i := range targetSchemas {
		if strings.ToLower(targetSchemas[i].TableName) == sourceLower {
			return &targetSchemas[i]
		}
	}

	// If only one table in both source and target, assume they match
	if len(r.recording.SourceSchema) == 1 && len(targetSchemas) == 1 {
		r.log(fmt.Sprintf("Auto-matching single table '%s' -> '%s'", sourceTableName, targetSchemas[0].TableName))
		return &targetSchemas[0]
	}

	return nil
}

// matchFields matches source fields to target fields
func (r *AnalysisReplayer) matchFields(sourceFields, targetFields []string) ([]FieldMapping, error) {
	mappings := []FieldMapping{}
	unmatchedSource := []string{}

	// Build map for quick lookup
	targetMap := make(map[string]bool)
	for _, tf := range targetFields {
		targetMap[tf] = true
	}

	// Step 1: Local matching - exact and case-insensitive
	for _, sf := range sourceFields {
		// Try exact match
		if targetMap[sf] {
			mappings = append(mappings, FieldMapping{
				OldField: sf,
				NewField: sf,
			})
			continue
		}

		// Try case-insensitive match
		found := false
		sfLower := strings.ToLower(sf)
		for _, tf := range targetFields {
			if strings.ToLower(tf) == sfLower {
				mappings = append(mappings, FieldMapping{
					OldField: sf,
					NewField: tf,
				})
				found = true
				break
			}
		}

		if !found {
			unmatchedSource = append(unmatchedSource, sf)
		}
	}

	// Check if difference is within acceptable range
	diffCount := len(unmatchedSource)

	// Step 2: If there are unmatched fields, try LLM-based intelligent matching
	if diffCount > 0 && diffCount <= r.config.MaxFieldDiff && r.config.AutoFixFields {
		r.log(fmt.Sprintf("Local matching found %d unmatched fields, attempting LLM intelligent matching...", diffCount))

		llmMappings, err := r.intelligentFieldMatching(unmatchedSource, targetFields)
		if err != nil {
			r.log(fmt.Sprintf("LLM intelligent matching failed: %v", err))
		} else {
			// Add LLM-suggested mappings
			for _, mapping := range llmMappings {
				r.log(fmt.Sprintf("LLM suggested mapping: '%s' -> '%s'", mapping.OldField, mapping.NewField))
				mappings = append(mappings, mapping)
			}

			// Recalculate unmatched count
			llmMatchedMap := make(map[string]bool)
			for _, m := range llmMappings {
				llmMatchedMap[m.OldField] = true
			}

			newUnmatched := []string{}
			for _, sf := range unmatchedSource {
				if !llmMatchedMap[sf] {
					newUnmatched = append(newUnmatched, sf)
				}
			}
			unmatchedSource = newUnmatched
			diffCount = len(unmatchedSource)
		}
	}

	// Final check
	if diffCount > r.config.MaxFieldDiff {
		return nil, fmt.Errorf("too many unmatched fields (%d): %v (max allowed: %d)",
			diffCount, unmatchedSource, r.config.MaxFieldDiff)
	}

	if diffCount > 0 {
		r.log(fmt.Sprintf("Warning: %d unmatched source fields remain: %v", diffCount, unmatchedSource))
	}

	return mappings, nil
}

// intelligentFieldMatching uses LLM to match unmatched source fields to target fields
func (r *AnalysisReplayer) intelligentFieldMatching(unmatchedSource, targetFields []string) ([]FieldMapping, error) {
	if r.llmService == nil {
		return nil, fmt.Errorf("LLM service not available")
	}

	// Build prompt for LLM
	prompt := fmt.Sprintf(`You are a data schema matching expert. I need to map source field names to target field names.

Source fields (unmatched):
%s

Available target fields:
%s

Please suggest the best matching target field for each source field. Consider:
1. Semantic similarity (e.g., "customer_name" matches "client_name")
2. Common abbreviations (e.g., "qty" matches "quantity")
3. Plural/singular variations (e.g., "products" matches "product")
4. Different naming conventions (camelCase, snake_case, etc.)

Return your answer as a JSON array of mappings in this exact format:
[
  {"source": "source_field_name", "target": "target_field_name"},
  ...
]

Only suggest mappings when you are confident (>80%%) they are semantically equivalent. If no good match exists for a field, omit it from the response.
`,
		strings.Join(unmatchedSource, "\n"),
		strings.Join(targetFields, "\n"),
	)

	r.log("Querying LLM for intelligent field matching...")

	// Call LLM
	response, err := r.llmService.Chat(context.Background(), fmt.Sprintf("%s", prompt))
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse JSON response
	var rawMappings []struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}

	// Extract JSON from response (might be wrapped in markdown code blocks)
	jsonStr := response
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json") + 7
		end := strings.LastIndex(response, "```")
		if end > start {
			jsonStr = response[start:end]
		}
	} else if strings.Contains(response, "```") {
		start := strings.Index(response, "```") + 3
		end := strings.LastIndex(response, "```")
		if end > start {
			jsonStr = response[start:end]
		}
	}

	// Try to parse JSON
	jsonStr = strings.TrimSpace(jsonStr)
	if err := json.Unmarshal([]byte(jsonStr), &rawMappings); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w. Response: %s", err, response)
	}

	// Convert to FieldMapping
	mappings := []FieldMapping{}
	for _, rm := range rawMappings {
		mappings = append(mappings, FieldMapping{
			OldField: rm.Source,
			NewField: rm.Target,
		})
	}

	return mappings, nil
}

// applyFieldMappings applies field mappings to SQL or Python code
func (r *AnalysisReplayer) applyFieldMappings(code string, toolName string) string {
	if len(r.fieldMappings) == 0 {
		return code
	}

	modifiedCode := code

	for _, tableMapping := range r.fieldMappings {
		// Replace table name if different
		if tableMapping.SourceTable != tableMapping.TargetTable {
			// Match table name in SQL (FROM clause, JOIN clause)
			re := regexp.MustCompile(fmt.Sprintf(`\bFROM\s+%s\b`, regexp.QuoteMeta(tableMapping.SourceTable)))
			modifiedCode = re.ReplaceAllString(modifiedCode, fmt.Sprintf("FROM %s", tableMapping.TargetTable))

			re = regexp.MustCompile(fmt.Sprintf(`\bJOIN\s+%s\b`, regexp.QuoteMeta(tableMapping.SourceTable)))
			modifiedCode = re.ReplaceAllString(modifiedCode, fmt.Sprintf("JOIN %s", tableMapping.TargetTable))
		}

		// Replace field names
		for _, fieldMapping := range tableMapping.Mappings {
			if fieldMapping.OldField == fieldMapping.NewField {
				continue
			}

			// For SQL: match column names
			if toolName == "sql_executor" {
				// Match in SELECT clause
				re := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(fieldMapping.OldField)))
				modifiedCode = re.ReplaceAllString(modifiedCode, fieldMapping.NewField)
			}

			// For Python: match column references in pandas operations
			if toolName == "python_tool" {
				// Match df['column']
				oldPattern := fmt.Sprintf(`['"]%s['"]`, regexp.QuoteMeta(fieldMapping.OldField))
				newReplacement := fmt.Sprintf(`'%s'`, fieldMapping.NewField)
				re := regexp.MustCompile(oldPattern)
				modifiedCode = re.ReplaceAllString(modifiedCode, newReplacement)

				// Match df.column
				oldPattern = fmt.Sprintf(`\.%s\b`, regexp.QuoteMeta(fieldMapping.OldField))
				newReplacement = fmt.Sprintf(`.%s`, fieldMapping.NewField)
				re = regexp.MustCompile(oldPattern)
				modifiedCode = re.ReplaceAllString(modifiedCode, newReplacement)
			}
		}
	}

	return modifiedCode
}

// Replay executes the recorded analysis on the target data source
func (r *AnalysisReplayer) Replay() (*ReplayResult, error) {
	r.log("Starting analysis replay...")

	result := &ReplayResult{
		Success:        true,
		StepsExecuted:  0,
		StepsFailed:    0,
		StepResults:    []StepResult{},
		FieldMappings:  r.fieldMappings,
		GeneratedFiles: []string{},
		Charts:         []map[string]interface{}{},
	}

	// Analyze schema compatibility
	mappings, err := r.AnalyzeSchemaCompatibility()
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Schema compatibility check failed: %v", err)
		return result, err
	}

	result.FieldMappings = mappings
	r.log(fmt.Sprintf("Schema analysis complete. Found %d table mappings.", len(mappings)))

	// Replay each step
	for _, step := range r.recording.Steps {
		r.log(fmt.Sprintf("Executing step %d: %s (%s)", step.StepID, step.Description, step.ToolName))

		stepResult := StepResult{
			StepID:   step.StepID,
			Success:  false,
			Modified: false,
		}

		// Apply field mappings to the input
		modifiedInput := r.applyFieldMappings(step.Input, step.ToolName)
		if modifiedInput != step.Input {
			stepResult.Modified = true
			r.log(fmt.Sprintf("Step %d: Applied field mappings", step.StepID))
		}

		// Execute the step based on tool name
		var output string
		var execErr error

		ctx := context.Background()

		switch step.ToolName {
		case "execute_sql":
			// Build SQL executor input
			sqlInput := map[string]interface{}{
				"data_source_id": r.config.TargetSourceID,
				"query":          modifiedInput,
			}
			sqlInputJSON, _ := json.Marshal(sqlInput)
			output, execErr = r.sqlExecutor.InvokableRun(ctx, string(sqlInputJSON))

		case "python_executor":
			// Build Python executor input
			pythonInput := map[string]interface{}{
				"code": modifiedInput,
			}
			pythonInputJSON, _ := json.Marshal(pythonInput)
			output, execErr = r.pythonExecutor.InvokableRun(ctx, string(pythonInputJSON))

		default:
			execErr = fmt.Errorf("unsupported tool: %s", step.ToolName)
		}

		if execErr != nil {
			stepResult.Success = false
			stepResult.ErrorMessage = execErr.Error()
			result.StepsFailed++
			r.log(fmt.Sprintf("Step %d failed: %v", step.StepID, execErr))
		} else {
			stepResult.Success = true
			stepResult.Output = output
			result.StepsExecuted++
			r.log(fmt.Sprintf("Step %d completed successfully", step.StepID))

			// If this step generated a chart, record it
			if step.ChartType != "" {
				chartInfo := map[string]interface{}{
					"step_id":    step.StepID,
					"type":       step.ChartType,
					"data":       output, // The chart data from execution
					"original":   step.ChartData,
				}
				result.Charts = append(result.Charts, chartInfo)
			}
		}

		result.StepResults = append(result.StepResults, stepResult)
	}

	if result.StepsFailed > 0 {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("%d out of %d steps failed", result.StepsFailed, len(r.recording.Steps))
	}

	r.log(fmt.Sprintf("Replay complete. Executed: %d, Failed: %d", result.StepsExecuted, result.StepsFailed))
	return result, nil
}
