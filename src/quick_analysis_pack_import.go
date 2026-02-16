package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// LoadQuickAnalysisPack opens a file dialog for the user to select a .qap file,
// checks if it is encrypted, and if not, parses and validates it against the target data source.
// If the file is encrypted, it returns a PackLoadResult with NeedsPassword=true so the frontend
// can prompt for a password and call LoadQuickAnalysisPackWithPassword.

// LoadQuickAnalysisPackByPath loads a .qap file from a given path (no file picker),
// checks encryption, and validates against the target data source.
func (a *App) LoadQuickAnalysisPackByPath(filePath string, dataSourceID string) (*PackLoadResult, error) {
	a.Log(fmt.Sprintf("[QAP-IMPORT] Loading pack by path: %s for datasource: %s", filePath, dataSourceID))

	encrypted, err := IsEncrypted(filePath)
	if err != nil {
		return nil, fmt.Errorf("文件格式无效，无法解析快捷分析包: %w", err)
	}

	if encrypted {
		// Check if we have a stored password from marketplace download
		if storedPwd, ok := a.packPasswords[filePath]; ok && storedPwd != "" {
			a.Log("[QAP-IMPORT] Found stored password for encrypted pack, attempting auto-decrypt")
			result, err := a.loadAndValidatePack(filePath, dataSourceID, storedPwd)
			if err == nil {
				return result, nil
			}
			// Auto-decrypt failed, fall back to manual password input
			a.Log(fmt.Sprintf("[QAP-IMPORT] Auto-decrypt with stored password failed: %v, falling back to manual input", err))
		}
		return &PackLoadResult{
			IsEncrypted:   true,
			NeedsPassword: true,
			FilePath:      filePath,
		}, nil
	}

	return a.loadAndValidatePack(filePath, dataSourceID, "")
}
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
		// Check if we have a stored password from marketplace download
		if storedPwd, ok := a.packPasswords[filePath]; ok && storedPwd != "" {
			a.Log("[QAP-IMPORT] Found stored password for encrypted pack, attempting auto-decrypt")
			result, err := a.loadAndValidatePack(filePath, dataSourceID, storedPwd)
			if err == nil {
				return result, nil
			}
			// Auto-decrypt failed, fall back to manual password input
			a.Log(fmt.Sprintf("[QAP-IMPORT] Auto-decrypt with stored password failed: %v, falling back to manual input", err))
		}
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
	// If no password provided, check if we have a stored password from marketplace download
	if password == "" {
		if storedPwd, ok := a.packPasswords[filePath]; ok && storedPwd != "" {
			a.Log("[QAP-IMPORT] Using stored marketplace password for auto-decrypt")
			password = storedPwd
		}
	}

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

	// 3. Validate file type and format version
	if pack.FileType != "VantageData_QuickAnalysisPack" {
		a.Log("[QAP-IMPORT] Invalid file type")
		return nil, fmt.Errorf("文件格式无效: 不是有效的快捷分析包文件")
	}
	if pack.FormatVersion != "" && pack.FormatVersion != "1.0" {
		a.Log(fmt.Sprintf("[QAP-IMPORT] Unsupported format version: %s", pack.FormatVersion))
		return nil, fmt.Errorf("不支持的分析包版本: %s，请升级软件后重试", pack.FormatVersion)
	}

	a.Log(fmt.Sprintf("[QAP-IMPORT] Parsed pack: %s by %s, %d steps",
		pack.Metadata.SourceName, pack.Metadata.Author, len(pack.ExecutableSteps)))

	if len(pack.ExecutableSteps) == 0 {
		a.Log("[QAP-IMPORT] Pack has no executable steps")
		return nil, fmt.Errorf("分析包中没有可执行的步骤")
	}

	// 4. Collect target data source schema (only for tables the pack requires) and validate
	targetSchema, err := a.collectTargetSchemaForPack(dataSourceID, pack.SchemaRequirements)
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
		HasPythonSteps:   a.packHasPythonSteps(&pack),
		PythonConfigured: a.isPythonConfigured(),
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

	// 2. Pre-check: if pack contains python_code steps, verify Python environment is configured
	if a.packHasPythonSteps(pack) && !a.isPythonConfigured() {
		a.Log("[QAP-EXECUTE] Pack contains Python steps but Python environment is not configured")
		return fmt.Errorf("此分析包包含 Python 脚本，但尚未配置 Python 环境。请在设置中配置 Python 路径后重试。")
	}

	// 2.5. Check usage license permission for marketplace packs
	listingID := pack.Metadata.ListingID
	if listingID > 0 && a.usageLicenseStore != nil {
		allowed, reason := a.usageLicenseStore.CheckPermission(listingID)
		if !allowed {
			a.Log(fmt.Sprintf("[QAP-EXECUTE] Permission denied for listing %d: %s", listingID, reason))
			return fmt.Errorf("权限不足: %s", reason)
		}
	}

	// 3. Create a new Replay_Session thread
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

	// 4. Notify frontend to switch to this thread (without creating a new one)
	runtime.EventsEmit(a.ctx, "qap-session-created", map[string]string{
		"threadId":     thread.ID,
		"dataSourceId": dataSourceID,
		"title":        thread.Title,
	})

	// 5. Execute steps sequentially
	totalSteps := len(pack.ExecutableSteps)
	stepResults := make(map[int]interface{})

	// Compute session directory for Python chart file output
	cfg, _ := a.GetConfig()
	sessionDir := filepath.Join(cfg.DataCacheDir, "sessions", thread.ID)
	// Ensure session directory exists for chart file output
	os.MkdirAll(sessionDir, 0755)

	for i, step := range pack.ExecutableSteps {
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Executing step %d/%d: %s (%s)", i+1, totalSteps, step.Description, step.StepType))

		// Check if any dependency step failed (result not in stepResults)
		dependencyFailed := false
		for _, depID := range step.DependsOn {
			if _, ok := stepResults[depID]; !ok {
				dependencyFailed = true
				a.Log(fmt.Sprintf("[QAP-EXECUTE] Skipping step %d: dependency step %d failed or was skipped", step.StepID, depID))
				break
			}
		}
		if dependencyFailed {
			skipMsg := ChatMessage{
				ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
				Role:      "assistant",
				Content:   fmt.Sprintf("⏭️ 步骤 %d (%s) 已跳过：依赖的前置步骤执行失败", step.StepID, step.Description),
				Timestamp: time.Now().Unix(),
			}
			a.chatService.AddMessage(thread.ID, skipMsg)
			runtime.EventsEmit(a.ctx, "thread-updated", thread.ID)
			time.Sleep(100 * time.Millisecond)
			continue
		}

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
			a.executePackPythonStep(thread.ID, messageID, step, stepResults, sessionDir)
		}

		// Update UI and add small delay for UI rendering and unique messageID generation
		runtime.EventsEmit(a.ctx, "thread-updated", thread.ID)
		time.Sleep(100 * time.Millisecond)
	}

	// 6. Emit completion message and push all results to dashboard
	completionMsg := ChatMessage{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Role:      "assistant",
		Content:   fmt.Sprintf("✅ 快捷分析包执行完成！共执行了 %d 个步骤。", totalSteps),
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(thread.ID, completionMsg)

	// Flush all pending analysis results to ensure they are saved to disk
	if a.eventAggregator != nil {
		a.eventAggregator.FlushNow(thread.ID, true)
	}

	// Emit completion event — the frontend qap-complete handler will call
	// ShowAllSessionResults from the frontend context, which works reliably.
	// Calling ShowAllSessionResults from within this RPC doesn't work because
	// Wails events emitted during a long-running RPC are not processed until
	// the RPC returns, by which time the ImportPackDialog has already closed.
	runtime.EventsEmit(a.ctx, "qap-complete", map[string]interface{}{
		"threadId":   thread.ID,
		"totalSteps": totalSteps,
	})
	runtime.EventsEmit(a.ctx, "thread-updated", thread.ID)

	a.Log("[QAP-EXECUTE] Execution completed successfully")

	// 7. Consume one use for per_use marketplace packs after successful execution
	if listingID > 0 && a.usageLicenseStore != nil {
		if err := a.usageLicenseStore.ConsumeUse(listingID); err != nil {
			a.Log(fmt.Sprintf("[QAP-EXECUTE] Failed to consume use for listing %d: %v", listingID, err))
		} else {
			_ = a.usageLicenseStore.Save()
		}
	}

	return nil
}

// ReExecuteQuickAnalysisPack 在当前会话中重新执行快捷分析包。
// 清除当前会话的所有消息，然后在同一会话中重新执行所有步骤。
func (a *App) ReExecuteQuickAnalysisPack(threadID string) error {
	a.Log(fmt.Sprintf("[QAP-REEXECUTE] Starting re-execution in thread: %s", threadID))

	// 1. Load the thread to get pack metadata
	thread, err := a.chatService.LoadThread(threadID)
	if err != nil {
		return fmt.Errorf("failed to load thread: %w", err)
	}
	if thread == nil {
		return fmt.Errorf("thread not found: %s", threadID)
	}
	if !thread.IsReplaySession || thread.QapFilePath == "" {
		return fmt.Errorf("该会话不是快捷分析会话")
	}

	// 2. Load and validate the pack
	result, err := a.loadAndValidatePack(thread.QapFilePath, thread.DataSourceID, "")
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

	// 3. Pre-check Python
	if a.packHasPythonSteps(pack) && !a.isPythonConfigured() {
		return fmt.Errorf("此分析包包含 Python 脚本，但尚未配置 Python 环境。请在设置中配置 Python 路径后重试。")
	}

	// 3.5. Check usage license permission for marketplace packs
	listingID := pack.Metadata.ListingID
	if listingID > 0 && a.usageLicenseStore != nil {
		allowed, reason := a.usageLicenseStore.CheckPermission(listingID)
		if !allowed {
			a.Log(fmt.Sprintf("[QAP-REEXECUTE] Permission denied for listing %d: %s", listingID, reason))
			return fmt.Errorf("权限不足: %s", reason)
		}
	}

	// 4. Clear existing messages in the thread
	if err := a.chatService.ClearThreadMessages(threadID); err != nil {
		a.Log(fmt.Sprintf("[QAP-REEXECUTE] Error clearing messages: %v", err))
		return fmt.Errorf("failed to clear messages: %w", err)
	}

	// Notify frontend that messages were cleared
	runtime.EventsEmit(a.ctx, "thread-updated", threadID)

	// 5. Execute steps sequentially (same logic as ExecuteQuickAnalysisPack)
	totalSteps := len(pack.ExecutableSteps)
	stepResults := make(map[int]interface{})

	cfg, _ := a.GetConfig()
	sessionDir := filepath.Join(cfg.DataCacheDir, "sessions", threadID)
	os.MkdirAll(sessionDir, 0755)

	for i, step := range pack.ExecutableSteps {
		a.Log(fmt.Sprintf("[QAP-REEXECUTE] Executing step %d/%d: %s (%s)", i+1, totalSteps, step.Description, step.StepType))

		dependencyFailed := false
		for _, depID := range step.DependsOn {
			if _, ok := stepResults[depID]; !ok {
				dependencyFailed = true
				break
			}
		}
		if dependencyFailed {
			skipMsg := ChatMessage{
				ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
				Role:      "assistant",
				Content:   fmt.Sprintf("⏭️ 步骤 %d (%s) 已跳过：依赖的前置步骤执行失败", step.StepID, step.Description),
				Timestamp: time.Now().Unix(),
			}
			a.chatService.AddMessage(threadID, skipMsg)
			runtime.EventsEmit(a.ctx, "thread-updated", threadID)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		runtime.EventsEmit(a.ctx, "qap-progress", map[string]interface{}{
			"threadId":    threadID,
			"currentStep": i + 1,
			"totalSteps":  totalSteps,
			"stepType":    step.StepType,
			"description": step.Description,
		})

		messageID := strconv.FormatInt(time.Now().UnixNano(), 10)

		switch step.StepType {
		case "sql_query":
			a.executePackSQLStep(threadID, messageID, step, stepResults, thread.DataSourceID)
		case "python_code":
			a.executePackPythonStep(threadID, messageID, step, stepResults, sessionDir)
		}

		runtime.EventsEmit(a.ctx, "thread-updated", threadID)
		time.Sleep(100 * time.Millisecond)
	}

	// 6. Completion - push all results then emit completion event
	completionMsg := ChatMessage{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Role:      "assistant",
		Content:   fmt.Sprintf("✅ 快捷分析包重新执行完成！共执行了 %d 个步骤。", totalSteps),
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(threadID, completionMsg)

	// Flush all pending analysis results to ensure they are saved to disk
	if a.eventAggregator != nil {
		a.eventAggregator.FlushNow(threadID, true)
	}

	// Emit completion event — frontend qap-complete handler calls ShowAllSessionResults
	runtime.EventsEmit(a.ctx, "qap-complete", map[string]interface{}{
		"threadId":   threadID,
		"totalSteps": totalSteps,
	})
	runtime.EventsEmit(a.ctx, "thread-updated", threadID)

	a.Log("[QAP-REEXECUTE] Re-execution completed successfully")

	// 7. Consume one use for per_use marketplace packs after successful re-execution
	if listingID > 0 && a.usageLicenseStore != nil {
		if err := a.usageLicenseStore.ConsumeUse(listingID); err != nil {
			a.Log(fmt.Sprintf("[QAP-REEXECUTE] Failed to consume use for listing %d: %v", listingID, err))
		} else {
			_ = a.usageLicenseStore.Save()
		}
	}

	return nil
}

// packHasPythonSteps checks if the pack contains any python_code steps.
func (a *App) packHasPythonSteps(pack *QuickAnalysisPack) bool {
	for _, step := range pack.ExecutableSteps {
		if step.StepType == "python_code" {
			return true
		}
	}
	return false
}

// isPythonConfigured checks if the Python environment is configured.
func (a *App) isPythonConfigured() bool {
	// Check EinoService first (same pool used by normal analysis)
	if a.einoService != nil && a.einoService.HasPython() {
		return true
	}
	// Check config
	cfg, err := a.GetConfig()
	if err == nil && cfg.PythonPath != "" {
		return true
	}
	// Also check if we can find Python automatically
	return a.findPythonPath() != ""
}

// findPythonPath returns a usable Python path, checking config first, then auto-detecting.
func (a *App) findPythonPath() string {
	cfg, _ := a.GetConfig()
	if cfg.PythonPath != "" {
		return cfg.PythonPath
	}
	// Auto-detect Python from system
	if a.pythonService != nil {
		envs := a.pythonService.ProbePythonEnvironments()
		for _, env := range envs {
			if env.Path != "" {
				a.Log(fmt.Sprintf("[QAP] Auto-detected Python: %s", env.Path))
				return env.Path
			}
		}
	}
	return ""
}

// getStepLabel returns a human-readable label for a pack step.
// If the step has a non-empty Description, it is returned as-is.
// Otherwise, a default label is generated based on StepType and StepID.
func getStepLabel(step PackStep) string {
	if step.Description != "" {
		return step.Description
	}
	switch step.StepType {
	case "sql_query":
		return fmt.Sprintf("SQL 查询 #%d", step.StepID)
	case "python_code":
		return fmt.Sprintf("Python 脚本 #%d", step.StepID)
	default:
		return fmt.Sprintf("步骤 #%d", step.StepID)
	}
}

// getStepUserRequest returns the original user request for display as "分析请求".
// Prefers UserRequest (the original user question), falls back to Description, then default label.
func getStepUserRequest(step PackStep) string {
	if step.UserRequest != "" {
		return step.UserRequest
	}
	return getStepLabel(step)
}

// executePackSQLStep executes a single SQL step from the pack.
// On success, the result is stored in stepResults and sent to the EventAggregator.
// On failure, an error message is added to the chat and emitted, but execution does not stop.
func (a *App) executePackSQLStep(threadID string, messageID string, step PackStep, stepResults map[int]interface{}, dataSourceID string) {
	stepLabel := getStepLabel(step)
	userRequest := getStepUserRequest(step)

	result, err := a.dataSourceService.ExecuteSQL(dataSourceID, step.Code)
	if err != nil {
		errMsg := fmt.Sprintf("步骤 %d 执行失败: %v", step.StepID, err)
		a.Log(fmt.Sprintf("[QAP-EXECUTE] SQL Error: %s", errMsg))

		// Add error message to chat
		errorChatMsg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   fmt.Sprintf("❌ 步骤 %d (%s) 执行失败：%v\n\n> 📋 分析请求：%s\n\n```sql\n%s\n```", step.StepID, step.Description, err, userRequest, step.Code),
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

	// Add success message with results (truncate large result sets for chat display)
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	chatContent := fmt.Sprintf("✅ 步骤 %d (%s):\n\n> 📋 分析请求：%s\n\n```json:table\n%s\n```", step.StepID, step.Description, userRequest, string(resultJSON))
	if len(chatContent) > 50000 {
		// Truncate for chat display but keep full result in stepResults for downstream steps
		previewJSON, _ := json.MarshalIndent(result[:min(len(result), 20)], "", "  ")
		chatContent = fmt.Sprintf("✅ 步骤 %d (%s) (共 %d 行，显示前 20 行):\n\n> 📋 分析请求：%s\n\n```json:table\n%s\n```", step.StepID, step.Description, len(result), userRequest, string(previewJSON))
	}
	successMsg := ChatMessage{
		ID:        messageID,
		Role:      "assistant",
		Content:   chatContent,
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(threadID, successMsg)

	// Send table data to EventAggregator for dashboard display
	if a.eventAggregator != nil {
		var analysisResults []AnalysisResultItem

		a.eventAggregator.AddItem(threadID, messageID, "", "table", result, map[string]interface{}{
			"sessionId":        threadID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepLabel,
		})
		analysisResults = append(analysisResults, AnalysisResultItem{
			Type: "table",
			Data: result,
			Metadata: map[string]interface{}{
				"sessionId":        threadID,
				"messageId":        messageID,
				"step_description": stepLabel,
			},
		})

		// 发送步骤中保存的 ECharts 配置到仪表盘
		echartsResults := a.emitStepEChartsConfigs(threadID, messageID, step)
		analysisResults = append(analysisResults, echartsResults...)

		a.eventAggregator.FlushNow(threadID, false)

		// Save analysis results for later re-push via ShowStepResultOnDashboard
		if err := a.chatService.SaveAnalysisResults(threadID, messageID, analysisResults); err != nil {
			a.Log(fmt.Sprintf("[QAP-EXECUTE] Failed to save SQL analysis results for step %d: %v", step.StepID, err))
		}
	}
}

// executePackPythonStep executes a single Python step from the pack.
// It tries the EinoService Python pool first (same as normal analysis), then falls back to auto-detected Python.
// On failure, an error message is logged and emitted.
func (a *App) executePackPythonStep(threadID string, messageID string, step PackStep, stepResults map[int]interface{}, workDir string) {
	stepLabel := getStepLabel(step)
	userRequest := getStepUserRequest(step)

	// Substitute dependency placeholders ${STEP_N_RESULT} with actual results
	code := step.Code
	for _, depID := range step.DependsOn {
		depResult := stepResults[depID]
		placeholder := fmt.Sprintf("${STEP_%d_RESULT}", depID)
		resultJSON, _ := json.Marshal(depResult)
		code = strings.ReplaceAll(code, placeholder, string(resultJSON))
	}

	// If this is a chart step from query_and_chart, inject SQL results as DataFrame 'df'
	if step.SourceTool == "query_and_chart" && step.PairedSQLStepID > 0 {
		if sqlResult, ok := stepResults[step.PairedSQLStepID]; ok {
			code = a.buildDataFrameInjection(sqlResult, code)
			a.Log(fmt.Sprintf("[QAP-EXECUTE] Injected DataFrame from SQL step %d for chart code", step.PairedSQLStepID))
		}
	}

	var output string
	var execErr error

	// Strategy 1: Use EinoService's Python pool (same path as normal analysis sessions)
	if a.einoService != nil && a.einoService.HasPython() {
		a.Log("[QAP-EXECUTE] Executing Python via EinoService pool")
		output, execErr = a.einoService.ExecutePython(code, workDir)
	} else {
		// Strategy 2: Find Python path via config or auto-detection
		pythonPath := a.findPythonPath()
		if pythonPath == "" {
			errMsg := fmt.Sprintf("步骤 %d 执行失败: Python 环境未配置", step.StepID)
			a.Log(fmt.Sprintf("[QAP-EXECUTE] Python Error: %s", errMsg))

			errorChatMsg := ChatMessage{
				ID:        messageID,
				Role:      "assistant",
				Content:   fmt.Sprintf("❌ 步骤 %d (%s) 执行失败：Python 环境未配置\n\n> 📋 分析请求：%s\n\n```python\n%s\n```", step.StepID, step.Description, userRequest, step.Code),
				Timestamp: time.Now().Unix(),
			}
			a.chatService.AddMessage(threadID, errorChatMsg)

			if a.eventAggregator != nil {
				a.eventAggregator.EmitError(threadID, "", errMsg)
			}
			return
		}

		a.Log(fmt.Sprintf("[QAP-EXECUTE] Executing Python via PythonService with path: %s", pythonPath))
		ps := a.pythonService
		output, execErr = ps.ExecuteScript(pythonPath, code)
	}

	if execErr != nil {
		errMsg := fmt.Sprintf("步骤 %d 执行失败: %v", step.StepID, execErr)
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Python Error: %s", errMsg))

		errorChatMsg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   fmt.Sprintf("❌ 步骤 %d (%s) 执行失败：%v\n\n> 📋 分析请求：%s\n\n```python\n%s\n```", step.StepID, step.Description, execErr, userRequest, step.Code),
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
		Content:   fmt.Sprintf("✅ 步骤 %d (%s):\n\n> 📋 分析请求：%s\n\n```\n%s\n```", step.StepID, step.Description, userRequest, output),
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(threadID, successMsg)

	// Collect analysis results from Python output and chart files
	if a.eventAggregator != nil {
		var analysisResults []AnalysisResultItem

		// 1. Scan workDir for newly generated image files (.png, .jpg, .jpeg)
		imageResults := a.detectAndSendPythonChartFiles(threadID, messageID, workDir, stepLabel)
		analysisResults = append(analysisResults, imageResults...)

		// 2. Detect ECharts JSON blocks in Python stdout (marked with json:echarts)
		echartsResults := a.detectAndSendPythonECharts(threadID, messageID, output, stepLabel)
		analysisResults = append(analysisResults, echartsResults...)

		// 3. Emit stored ECharts configs from pack export
		storedEchartsResults := a.emitStepEChartsConfigs(threadID, messageID, step)
		analysisResults = append(analysisResults, storedEchartsResults...)

		// 4. Flush to ensure data is immediately pushed to frontend
		if len(analysisResults) > 0 {
			a.eventAggregator.FlushNow(threadID, false)
		}

		// 5. Save analysis results to the message for later re-push via ShowStepResultOnDashboard
		if len(analysisResults) > 0 {
			if err := a.chatService.SaveAnalysisResults(threadID, messageID, analysisResults); err != nil {
				a.Log(fmt.Sprintf("[QAP-EXECUTE] Failed to save analysis results for step %d: %v", step.StepID, err))
			}
		}
	}
}

// detectAndSendPythonChartFiles scans the workDir for newly generated image files
// and sends them to the EventAggregator for dashboard display.
// Returns the analysis result items for persistence.
func (a *App) detectAndSendPythonChartFiles(threadID, messageID, workDir string, stepDescription string) []AnalysisResultItem {
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return nil
	}

	imageExtensions := map[string]bool{".png": true, ".jpg": true, ".jpeg": true}
	var results []AnalysisResultItem

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !imageExtensions[ext] {
			continue
		}

		filePath := filepath.Join(workDir, entry.Name())
		imageData, err := os.ReadFile(filePath)
		if err != nil {
			a.Log(fmt.Sprintf("[QAP-EXECUTE] Failed to read chart file %s: %v", entry.Name(), err))
			continue
		}

		base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)
		a.eventAggregator.AddItem(threadID, messageID, "", "image", base64Data, map[string]interface{}{
			"sessionId":        threadID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"fileName":         entry.Name(),
			"step_description": stepDescription,
		})
		results = append(results, AnalysisResultItem{
			Type: "image",
			Data: base64Data,
			Metadata: map[string]interface{}{
				"fileName":         entry.Name(),
				"step_description": stepDescription,
			},
		})
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Sent chart image to dashboard: %s", entry.Name()))
	}

	return results
}

// detectAndSendPythonECharts parses Python stdout for ECharts JSON blocks
// marked with ```json:echarts ... ``` or bare json:echarts markers,
// sends them via EventAggregator, and returns analysis result items.
func (a *App) detectAndSendPythonECharts(threadID, messageID, output string, stepDescription string) []AnalysisResultItem {
	var results []AnalysisResultItem

	// Match ECharts blocks with backtick fences: ```json:echarts\n{...}\n```
	reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	matches := reECharts.FindAllStringSubmatch(output, -1)

	// Also match bare json:echarts markers (no backticks): json:echarts\n{...}
	reEChartsNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:echarts\\s*\\n(\\{[\\s\\S]+?\\n\\})(?:\\s*\\n(?:---|###)|\\s*$)")
	matches = append(matches, reEChartsNoBT.FindAllStringSubmatch(output, -1)...)

	for i, match := range matches {
		if len(match) < 2 {
			continue
		}
		chartData := strings.TrimSpace(match[1])
		if chartData == "" {
			continue
		}

		// Validate it's valid JSON
		if !json.Valid([]byte(chartData)) {
			a.Log(fmt.Sprintf("[QAP-EXECUTE] Skipping invalid ECharts JSON in Python output (match #%d)", i+1))
			continue
		}

		a.eventAggregator.AddItem(threadID, messageID, "", "echarts", chartData, map[string]interface{}{
			"sessionId":        threadID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepDescription,
		})
		results = append(results, AnalysisResultItem{
			Type: "echarts",
			Data: chartData,
			Metadata: map[string]interface{}{
				"step_description": stepDescription,
			},
		})
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Sent ECharts config to dashboard from Python output (match #%d)", i+1))
	}

	return results
}

// buildDataFrameInjection wraps chart code with DataFrame loading from SQL results,
// similar to how query_and_chart tool's buildChartPythonCode works during normal analysis.
// The SQL result ([]map[string]interface{}) is serialized to JSON and loaded as pandas DataFrame 'df'.
func (a *App) buildDataFrameInjection(sqlResult interface{}, chartCode string) string {
	resultJSON, err := json.Marshal(sqlResult)
	if err != nil {
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Failed to marshal SQL result for DataFrame injection: %v", err))
		return chartCode
	}

	// Use base64 encoding to safely embed JSON in Python code, avoiding string escaping issues
	encoded := base64.StdEncoding.EncodeToString(resultJSON)

	return fmt.Sprintf(`import pandas as pd
import json
import base64
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

# Load SQL query results into DataFrame (injected by QAP executor)
_sql_result = json.loads(base64.b64decode("%s").decode("utf-8"))
if isinstance(_sql_result, list):
    df = pd.DataFrame(_sql_result)
elif isinstance(_sql_result, dict):
    if "rows" in _sql_result and _sql_result["rows"]:
        df = pd.DataFrame(_sql_result["rows"])
    elif "data" in _sql_result:
        df = pd.DataFrame(_sql_result["data"])
    else:
        df = pd.DataFrame()
else:
    df = pd.DataFrame()

print(f"DataFrame loaded: {len(df)} rows, {len(df.columns)} columns")

# Chart code from analysis pack
%s
`, encoded, chartCode)
}

// emitStepEChartsConfigs 将步骤中保存的 ECharts 配置发送到 EventAggregator。
// 这些配置是在导出时从原始 LLM 响应中提取的，用于在重放时重现图表。
func (a *App) emitStepEChartsConfigs(threadID, messageID string, step PackStep) []AnalysisResultItem {
	if len(step.EChartsConfigs) == 0 || a.eventAggregator == nil {
		return nil
	}

	stepLabel := getStepLabel(step)
	var results []AnalysisResultItem
	for i, chartJSON := range step.EChartsConfigs {
		if !json.Valid([]byte(chartJSON)) {
			a.Log(fmt.Sprintf("[QAP-EXECUTE] Skipping invalid stored ECharts config #%d for step %d", i+1, step.StepID))
			continue
		}
		a.eventAggregator.AddItem(threadID, messageID, "", "echarts", chartJSON, map[string]interface{}{
			"sessionId":        threadID,
			"messageId":        messageID,
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepLabel,
		})
		results = append(results, AnalysisResultItem{
			Type: "echarts",
			Data: chartJSON,
			Metadata: map[string]interface{}{
				"step_description": stepLabel,
			},
		})
		a.Log(fmt.Sprintf("[QAP-EXECUTE] Sent stored ECharts config #%d to dashboard for step %d", i+1, step.StepID))
	}
	return results
}
