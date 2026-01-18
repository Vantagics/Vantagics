package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================================================================
// Import and Validation Implementation
// ============================================================================

// ValidationResult holds the results of schema validation
type ValidationResult struct {
	Compatible bool              `json:"compatible"`
	Issues     []ValidationIssue `json:"issues"`
}

// ValidationIssue represents a single validation problem
type ValidationIssue struct {
	Type     string `json:"type"`     // "missing_table", "missing_column", "type_mismatch"
	Table    string `json:"table,omitempty"`
	Column   string `json:"column,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning"
}

// PrepareImportAnalysis shows file picker and returns parsed export data
func (a *App) PrepareImportAnalysis() (*AnalysisExport, error) {
	a.Log("[IMPORT] Preparing import analysis process")
	
	// Show file picker
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "导入分析过程",
		Filters: []runtime.FileFilter{
			{DisplayName: "RapidBI Analysis", Pattern: "*.rbi"},
		},
	})
	
	if err != nil || filePath == "" {
		a.Log("[IMPORT] User cancelled file selection")
		return nil, nil // User cancelled
	}
	
	a.Log(fmt.Sprintf("[IMPORT] Selected file: %s", filePath))
	
	// Parse import file
	exportData, err := a.parseImportFile(filePath)
	if err != nil {
		a.Log(fmt.Sprintf("[IMPORT] Error parsing file: %v", err))
		return nil, err
	}
	
	// Validate RBI format
	if exportData.FileType != "RapidBI_Analysis_Export" {
		a.Log("[IMPORT] Invalid file format - missing RBI identifier")
		return nil, fmt.Errorf("invalid file format: not a RapidBI analysis export file")
	}
	
	a.Log(fmt.Sprintf("[IMPORT] Successfully parsed RBI file (format version %s)", exportData.FormatVersion))
	return exportData, nil
}

// ImportAnalysisProcess handles the complete import workflow (simple version)
// This is a convenience function that automatically selects the first data source
func (a *App) ImportAnalysisProcess() error {
	a.Log("[IMPORT] Starting simple import analysis process")
	
	// 1. Prepare (file picker + parse)
	exportData, err := a.PrepareImportAnalysis()
	if err != nil {
		return err
	}
	
	if exportData == nil {
		return nil // User cancelled
	}
	
	// 2. Get first available data source
	sources, err := a.dataSourceService.LoadDataSources()
	if err != nil {
		a.Log(fmt.Sprintf("[IMPORT] Error loading data sources: %v", err))
		return err
	}
	
	if len(sources) == 0 {
		return fmt.Errorf("no data sources available for import")
	}
	
	targetDataSourceID := sources[0].ID
	a.Log(fmt.Sprintf("[IMPORT] Using data source: %s (%s)", sources[0].Name, targetDataSourceID))
	
	// 3. Validate
	validation, err := a.ValidateImportAnalysis(exportData, targetDataSourceID)
	if err != nil {
		a.Log(fmt.Sprintf("[IMPORT] Validation error: %v", err))
		return err
	}
	
	if !validation.Compatible {
		a.Log("[IMPORT] Incompatible schema - cannot import")
		var issueMessages []string
		for _, issue := range validation.Issues {
			if issue.Severity == "error" {
				issueMessages = append(issueMessages, issue.Message)
			}
		}
		return fmt.Errorf("schema incompatible:\n%s", strings.Join(issueMessages, "\n"))
	}
	
	// 4. Execute
	a.Log("[IMPORT] Starting execution of imported analysis")
	err = a.ExecuteImportAnalysis(exportData, targetDataSourceID)
	if err != nil {
		a.Log(fmt.Sprintf("[IMPORT] Execution error: %v", err))
		return err
	}
	
	a.Log("[IMPORT] Import completed successfully")
	return nil
}

// parseImportFile reads and parses the import JSON file
func (a *App) parseImportFile(filePath string) (*AnalysisExport, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	
	var export AnalysisExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("invalid export file format: %v", err)
	}
	
	return &export, nil
}

// ValidateImportAnalysis checks if the analysis can run on the target data source
func (a *App) ValidateImportAnalysis(exportData *AnalysisExport, targetDataSourceID string) (*ValidationResult, error) {
	a.Log(fmt.Sprintf("[VALIDATE] Validating against data source: %s", targetDataSourceID))
	
	// Get target schema
	tables, err := a.dataSourceService.GetDataSourceTables(targetDataSourceID)
	if err != nil {
		return nil, err
	}
	
	a.Log(fmt.Sprintf("[VALIDATE] Target data source has %d tables", len(tables)))
	
	// Build schema map
	schemaMap := make(map[string][]string) // table -> columns
	for _, tableName := range tables {
		data, err := a.dataSourceService.GetDataSourceTableData(targetDataSourceID, tableName, 1)
		if err == nil && len(data) > 0 {
			var cols []string
			for col := range data[0] {
				cols = append(cols, col)
			}
			schemaMap[tableName] = cols
		}
	}
	
	// Validate
	result := &ValidationResult{
		Compatible: true,
		Issues:     []ValidationIssue{},
	}
	
	for _, reqTable := range exportData.SchemaRequirements.Tables {
		cols, exists := schemaMap[reqTable.Name]
		if !exists {
			result.Compatible = false
			result.Issues = append(result.Issues, ValidationIssue{
				Type:     "missing_table",
				Table:    reqTable.Name,
				Message:  fmt.Sprintf("表 '%s' 不存在", reqTable.Name),
				Severity: "error",
			})
			a.Log(fmt.Sprintf("[VALIDATE] Missing table: %s", reqTable.Name))
			continue
		}
		
		// Check columns
		colSet := make(map[string]bool)
		for _, col := range cols {
			colSet[col] = true
		}
		
		for _, reqCol := range reqTable.Columns {
			if !colSet[reqCol] {
				result.Issues = append(result.Issues, ValidationIssue{
					Type:     "missing_column",
					Table:    reqTable.Name,
					Column:   reqCol,
					Message:  fmt.Sprintf("字段 '%s.%s' 不存在", reqTable.Name, reqCol),
					Severity: "warning",
				})
				a.Log(fmt.Sprintf("[VALIDATE] Missing column: %s.%s", reqTable.Name, reqCol))
			}
		}
	}
	
	a.Log(fmt.Sprintf("[VALIDATE] Validation complete - %d issues found", len(result.Issues)))
	return result, nil
}

// ExecuteImportAnalysis replays the analysis without AI
func (a *App) ExecuteImportAnalysis(exportData *AnalysisExport, targetDataSourceID string) error {
	a.Log("[EXECUTE] Starting analysis replay")
	
	// Create new thread
	threadTitle := fmt.Sprintf("Imported: %s", exportData.DataSource.Name)
	thread, err := a.CreateChatThread(targetDataSourceID, threadTitle)
	if err != nil {
		return fmt.Errorf("failed to create chat thread: %v", err)
	}
	
	a.Log(fmt.Sprintf("[EXECUTE] Created new thread: %s", thread.ID))
	
	// Notify frontend to switch to this thread
	runtime.EventsEmit(a.ctx, "start-new-chat", map[string]string{
		"dataSourceId": targetDataSourceID,
		"sessionName":  thread.Title,
	})
	
	// Store step results
	stepResults := make(map[int]interface{})
	
	// Execute steps in order
	for _, step := range exportData.ExecutableSteps {
		a.Log(fmt.Sprintf("[EXECUTE] Executing step %d: %s", step.StepID, step.StepType))
		
		switch step.StepType {
		case "sql_query":
			result, err := a.dataSourceService.ExecuteSQL(targetDataSourceID, step.SQL)
			if err != nil {
				errMsg := fmt.Sprintf("步骤 %d 执行失败: %v", step.StepID, err)
				a.Log(fmt.Sprintf("[EXECUTE] SQL Error: %s", errMsg))
				
				// Add error message to chat
				errorMsg := ChatMessage{
					ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
					Role:      "assistant",
					Content:   fmt.Sprintf("执行SQL失败：%v\n\nSQL:\n```sql\n%s\n```", err, step.SQL),
					Timestamp: time.Now().Unix(),
				}
				a.chatService.AddMessage(thread.ID, errorMsg)
				runtime.EventsEmit(a.ctx, "thread-updated", thread.ID)
				
				return fmt.Errorf(errMsg)
			}
			
			stepResults[step.StepID] = result
			a.Log(fmt.Sprintf("[EXECUTE] SQL executed successfully, %d rows", len(result)))
			
			// Add success message with results
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			successMsg := ChatMessage{
				ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
				Role:      "assistant",
				Content:   fmt.Sprintf("执行SQL成功 (步骤 %d):\n\n```json:table\n%s\n```", step.StepID, string(resultJSON)),
				Timestamp: time.Now().Unix(),
			}
			a.chatService.AddMessage(thread.ID, successMsg)
			
		case "python_code":
			// For Python, we need to substitute ${STEP_N_RESULT} with actual results
			code := step.PythonCode
			for _, depID := range step.DependsOn {
				depResult := stepResults[depID]
				placeholder := fmt.Sprintf("${STEP_%d_RESULT}", depID)
				resultJSON, _ := json.Marshal(depResult)
				code = strings.ReplaceAll(code, placeholder, string(resultJSON))
			}
			
			a.Log(fmt.Sprintf("[EXECUTE] Python code prepared (depends on %d steps)", len(step.DependsOn)))
			
			// TODO: Execute Python via Python service
			// For now, just add a placeholder message
			pythonMsg := ChatMessage{
				ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
				Role:      "assistant",
				Content:   fmt.Sprintf("Python执行 (步骤 %d):\n\n```python\n%s\n```\n\n*注：Python执行功能待实现*", step.StepID, code),
				Timestamp: time.Now().Unix(),
			}
			a.chatService.AddMessage(thread.ID, pythonMsg)
		}
		
		// Emit progress
		runtime.EventsEmit(a.ctx, "import-progress", map[string]interface{}{
			"step":    step.StepID,
			"total":   len(exportData.ExecutableSteps),
			"message": step.Description,
		})
		
		// Update UI
		runtime.EventsEmit(a.ctx, "thread-updated", thread.ID)
	}
	
	// Add completion message
	completionMsg := ChatMessage{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Role:      "assistant",
		Content:   fmt.Sprintf("✅ 分析导入完成！成功执行了 %d 个步骤。", len(exportData.ExecutableSteps)),
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(thread.ID, completionMsg)
	runtime.EventsEmit(a.ctx, "thread-updated", thread.ID)
	
	a.Log("[EXECUTE] Analysis replay completed successfully")
	return nil
}
