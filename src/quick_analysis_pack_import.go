package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"vantagedata/agent"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// LoadQuickAnalysisPack opens a file dialog for the user to select a .qap file,
// checks if it is encrypted, and if not, parses and validates it against the target data source.
// If the file is encrypted, it returns a PackLoadResult with NeedsPassword=true so the frontend
// can prompt for a password and call LoadQuickAnalysisPackWithPassword.
func (a *App) LoadQuickAnalysisPack(dataSourceID string) (*PackLoadResult, error) {
	a.Log("[QAP-IMPORT] Starting quick analysis pack import")

	// 1. Show file picker for .qap files
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "加载快捷分析包",
		Filters: []runtime.FileFilter{
			{DisplayName: "Quick Analysis Pack (*.qap)", Pattern: "*.qap"},
		},
	})
	if err != nil || filePath == "" {
		a.Log("[QAP-IMPORT] User cancelled file selection")
		return nil, nil // User cancelled
	}

	a.Log(fmt.Sprintf("[QAP-IMPORT] Selected file: %s", filePath))

	// 2. Check if the file is encrypted
	encrypted, err := IsEncrypted(filePath)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-IMPORT] Error checking encryption: %v", err))
		return nil, fmt.Errorf("文件格式无效，无法解析快捷分析包: %w", err)
	}

	if encrypted {
		a.Log("[QAP-IMPORT] File is encrypted, requesting password")
		return &PackLoadResult{
			IsEncrypted:   true,
			NeedsPassword: true,
			FilePath:      filePath,
		}, nil
	}

	// 3. Not encrypted — unpack, parse, and validate
	return a.loadAndValidatePack(filePath, dataSourceID, "")
}

// LoadQuickAnalysisPackWithPassword loads an encrypted .qap file using the provided password,
// parses the JSON content, and validates the schema against the target data source.
func (a *App) LoadQuickAnalysisPackWithPassword(filePath string, dataSourceID string, password string) (*PackLoadResult, error) {
	a.Log(fmt.Sprintf("[QAP-IMPORT] Loading encrypted pack with password: %s", filePath))
	return a.loadAndValidatePack(filePath, dataSourceID, password)
}

// loadAndValidatePack is the shared logic for loading, parsing, and validating a .qap file.
func (a *App) loadAndValidatePack(filePath string, dataSourceID string, password string) (*PackLoadResult, error) {
	// 1. Unpack from ZIP
	jsonData, err := UnpackFromZip(filePath, password)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-IMPORT] Error unpacking: %v", err))
		if err == ErrWrongPassword {
			return nil, fmt.Errorf("口令不正确")
		}
		return nil, fmt.Errorf("文件格式无效，无法解析快捷分析包: %w", err)
	}

	// 2. Parse JSON into QuickAnalysisPack
	var pack QuickAnalysisPack
	if err := json.Unmarshal(jsonData, &pack); err != nil {
		a.Log(fmt.Sprintf("[QAP-IMPORT] Error parsing JSON: %v", err))
		return nil, fmt.Errorf("文件格式无效，无法解析快捷分析包: %w", err)
	}

	// 3. Validate file type
	if pack.FileType != "VantageData_QuickAnalysisPack" {
		a.Log("[QAP-IMPORT] Invalid file type")
		return nil, fmt.Errorf("文件格式无效: 不是有效的快捷分析包文件")
	}

	a.Log(fmt.Sprintf("[QAP-IMPORT] Parsed pack: %s by %s, %d steps",
		pack.Metadata.SourceName, pack.Metadata.Author, len(pack.ExecutableSteps)))

	// 4. Collect target data source schema and validate
	targetSchema, err := a.collectFullSchema(dataSourceID)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-IMPORT] Error collecting target schema: %v", err))
		return nil, fmt.Errorf("无法获取目标数据源的 schema: %w", err)
	}

	validation := ValidateSchema(pack.SchemaRequirements, targetSchema)
	a.Log(fmt.Sprintf("[QAP-IMPORT] Schema validation: compatible=%v, missing_tables=%d, missing_columns=%d",
		validation.Compatible, len(validation.MissingTables), len(validation.MissingColumns)))

	// 5. If there are missing tables, block import
	if !validation.Compatible {
		a.Log(fmt.Sprintf("[QAP-IMPORT] Schema incompatible: missing tables: %v", validation.MissingTables))
	}

	return &PackLoadResult{
		Pack:        &pack,
		Validation:  validation,
		IsEncrypted: password != "",
		FilePath:    filePath,
	}, nil
}

// ExecuteQuickAnalysisPack loads a .qap file, creates a Replay_Session, and executes
// all steps sequentially. Results are sent to the EventAggregator for dashboard display.
// On step failure, the error is logged and emitted, but execution continues to the next step.
func (a *App) ExecuteQuickAnalysisPack(filePath string, dataSourceID string, password string) error {
	a.Log(fmt.Sprintf("[QAP-EXECUTE] Starting execution: file=%s, dataSource=%s", filePath, dataSourceID))

	// 1. Load and parse the pack
	result, err := a.loadAndValidatePack(filePath, dataSourceID, password)
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("failed to load pack")
	}
	if !result.Validation.Compatible {
		return fmt.Errorf("目标数据源缺少必需的表: %s", strings.Join(result.Validation.MissingTables, ", "))
	}

	pack := result.Pack

	// 2. Create a new Replay_Session thread
	threadTitle := fmt.Sprintf("⚡ %s (by %s)", pack.Metadata.SourceName, pack.Metadata.Author)
	thread, err := a.CreateChatThread(dataSourceID, threadTitle)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Error creating thread: %v", err))
		return fmt.Errorf("failed to create replay session: %w", err)
	}

	// Mark as replay session with pack metadata
	thread.IsReplaySession = true
	thread.PackMetadata = &pack.Metadata
	thread.QapFilePath = filePath
	if err := a.chatService.saveThreadInternal(thread); err != nil {
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Error saving thread metadata: %v", err))
		return fmt.Errorf("failed to save replay session: %w", err)
	}

	a.Log(fmt.Sprintf("[QAP-EXECUTE] Created replay session: %s", thread.ID))

	// 3. Notify frontend to switch to this thread
	runtime.EventsEmit(a.ctx, "start-new-chat", map[string]string{
		"dataSourceId": dataSourceID,
		"sessionName":  thread.Title,
	})

	// 4. Execute steps sequentially
	totalSteps := len(pack.ExecutableSteps)
	stepResults := make(map[int]interface{})

	for i, step := range pack.ExecutableSteps {
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Executing step %d/%d: %s (%s)", i+1, totalSteps, step.Description, step.StepType))

		// Emit progress event
		runtime.EventsEmit(a.ctx, "qap-progress", map[string]interface{}{
			"threadId":    thread.ID,
			"currentStep": i + 1,
			"totalSteps":  totalSteps,
			"stepType":    step.StepType,
			"description": step.Description,
		})

		messageID := strconv.FormatInt(time.Now().UnixNano(), 10)

		switch step.StepType {
		case "sql_query":
			a.executePackSQLStep(thread.ID, messageID, step, stepResults, dataSourceID)

		case "python_code":
			a.executePackPythonStep(thread.ID, messageID, step, stepResults)
		}

		// Update UI
		runtime.EventsEmit(a.ctx, "thread-updated", thread.ID)
	}

	// 5. Emit completion event
	completionMsg := ChatMessage{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Role:      "assistant",
		Content:   fmt.Sprintf("✅ 快捷分析包执行完成！共执行了 %d 个步骤。", totalSteps),
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(thread.ID, completionMsg)

	runtime.EventsEmit(a.ctx, "qap-complete", map[string]interface{}{
		"threadId":   thread.ID,
		"totalSteps": totalSteps,
	})
	runtime.EventsEmit(a.ctx, "thread-updated", thread.ID)

	// Flush all pending analysis results
	if a.eventAggregator != nil {
		a.eventAggregator.FlushNow(thread.ID, true)
	}

	a.Log("[QAP-EXECUTE] Execution completed successfully")
	return nil
}

// executePackSQLStep executes a single SQL step from the pack.
// On success, the result is stored in stepResults and sent to the EventAggregator.
// On failure, an error message is added to the chat and emitted, but execution does not stop.
func (a *App) executePackSQLStep(threadID string, messageID string, step PackStep, stepResults map[int]interface{}, dataSourceID string) {
	result, err := a.dataSourceService.ExecuteSQL(dataSourceID, step.Code)
	if err != nil {
		errMsg := fmt.Sprintf("步骤 %d 执行失败: %v", step.StepID, err)
		a.Log(fmt.Sprintf("[QAP-EXECUTE] SQL Error: %s", errMsg))

		// Add error message to chat
		errorChatMsg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   fmt.Sprintf("❌ 步骤 %d (%s) 执行失败：%v\n\n```sql\n%s\n```", step.StepID, step.Description, err, step.Code),
			Timestamp: time.Now().Unix(),
		}
		a.chatService.AddMessage(threadID, errorChatMsg)

		// Emit error to EventAggregator
		if a.eventAggregator != nil {
			a.eventAggregator.EmitError(threadID, "", errMsg)
		}
		return
	}

	stepResults[step.StepID] = result
	a.Log(fmt.Sprintf("[QAP-EXECUTE] SQL step %d executed successfully, %d rows", step.StepID, len(result)))

	// Add success message with results
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	successMsg := ChatMessage{
		ID:        messageID,
		Role:      "assistant",
		Content:   fmt.Sprintf("✅ 步骤 %d (%s):\n\n```json:table\n%s\n```", step.StepID, step.Description, string(resultJSON)),
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(threadID, successMsg)

	// Send table data to EventAggregator for dashboard display
	if a.eventAggregator != nil {
		a.eventAggregator.AddTable(threadID, messageID, "", result)
		a.eventAggregator.FlushNow(threadID, false)
	}
}

// executePackPythonStep executes a single Python step from the pack.
// On success, the output is added to the chat. On failure, an error message is logged and emitted.
func (a *App) executePackPythonStep(threadID string, messageID string, step PackStep, stepResults map[int]interface{}) {
	cfg, err := a.GetConfig()
	if err != nil || cfg.PythonPath == "" {
		errMsg := fmt.Sprintf("步骤 %d 执行失败: Python 环境未配置", step.StepID)
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Python Error: %s", errMsg))

		errorChatMsg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   fmt.Sprintf("❌ 步骤 %d (%s) 执行失败：Python 环境未配置\n\n```python\n%s\n```", step.StepID, step.Description, step.Code),
			Timestamp: time.Now().Unix(),
		}
		a.chatService.AddMessage(threadID, errorChatMsg)

		if a.eventAggregator != nil {
			a.eventAggregator.EmitError(threadID, "", errMsg)
		}
		return
	}

	// Substitute dependency placeholders ${STEP_N_RESULT} with actual results
	code := step.Code
	for _, depID := range step.DependsOn {
		depResult := stepResults[depID]
		placeholder := fmt.Sprintf("${STEP_%d_RESULT}", depID)
		resultJSON, _ := json.Marshal(depResult)
		code = strings.ReplaceAll(code, placeholder, string(resultJSON))
	}

	// Execute Python code
	ps := &agent.PythonService{}
	output, err := ps.ExecuteScript(cfg.PythonPath, code)
	if err != nil {
		errMsg := fmt.Sprintf("步骤 %d 执行失败: %v", step.StepID, err)
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Python Error: %s", errMsg))

		errorChatMsg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   fmt.Sprintf("❌ 步骤 %d (%s) 执行失败：%v\n\n```python\n%s\n```", step.StepID, step.Description, err, step.Code),
			Timestamp: time.Now().Unix(),
		}
		a.chatService.AddMessage(threadID, errorChatMsg)

		if a.eventAggregator != nil {
			a.eventAggregator.EmitError(threadID, "", errMsg)
		}
		return
	}

	stepResults[step.StepID] = output
	a.Log(fmt.Sprintf("[QAP-EXECUTE] Python step %d executed successfully", step.StepID))

	// Add success message with output
	successMsg := ChatMessage{
		ID:        messageID,
		Role:      "assistant",
		Content:   fmt.Sprintf("✅ 步骤 %d (%s):\n\n```\n%s\n```", step.StepID, step.Description, output),
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(threadID, successMsg)
}
