package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vantagics/i18n"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExecuteQuickAnalysisPack loads a .qap file, creates a Replay_Session, and executes
// all steps sequentially. Results are sent to the EventAggregator for dashboard display.
// On step failure, the error is logged and emitted, but execution continues to the next step.
// If a replay session already exists for the same datasource + qap file, it will be reused
// (re-executed) instead of creating a duplicate session.
func (a *App) ExecuteQuickAnalysisPack(filePath string, dataSourceID string, password string) error {
	a.Log(fmt.Sprintf("%s Starting execution: file=%s, dataSource=%s", logTagExecute, filePath, dataSourceID))

	// 1. Load and validate the pack
	result, err := a.loadAndValidatePack(filePath, dataSourceID, password)
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("failed to load pack: result is nil")
	}
	if result.Pack == nil {
		return fmt.Errorf("failed to load pack: pack data is nil")
	}
	if !result.Validation.Compatible {
		return fmt.Errorf("%s", i18n.T("qap.missing_required_tables", strings.Join(result.Validation.MissingTables, ", ")))
	}

	pack := result.Pack

	// 2. Pre-checks: Python environment + marketplace license
	if err := a.preCheckPackExecution(pack, logTagExecute); err != nil {
		return err
	}

	// 3. Check if a replay session already exists for this datasource + qap file
	existingThread, _ := a.chatService.FindReplaySessionByQapFile(dataSourceID, filePath)
	if existingThread != nil {
		a.Log(fmt.Sprintf("%s Found existing replay session %s for datasource=%s, file=%s �re-executing", logTagExecute, existingThread.ID, dataSourceID, filePath))
		return a.ReExecuteQuickAnalysisPack(existingThread.ID)
	}

	// 4. Create a new Replay_Session thread
	threadTitle := fmt.Sprintf("�%s (by %s)", pack.Metadata.PackName, pack.Metadata.Author)
	thread, err := a.CreateChatThread(dataSourceID, threadTitle)
	if err != nil {
		a.Log(fmt.Sprintf("%s Error creating thread: %v", logTagExecute, err))
		return fmt.Errorf("failed to create replay session: %w", err)
	}

	// Mark as replay session with pack metadata
	thread.IsReplaySession = true
	thread.PackMetadata = &pack.Metadata
	thread.QapFilePath = filePath
	if err := a.chatService.saveThreadInternal(thread); err != nil {
		a.Log(fmt.Sprintf("%s Error saving thread metadata: %v", logTagExecute, err))
		// Clean up the created thread on save failure
		_ = a.chatService.DeleteThread(thread.ID)
		return fmt.Errorf("failed to save replay session: %w", err)
	}

	a.Log(fmt.Sprintf("%s Created replay session: %s", logTagExecute, thread.ID))

	// 5. Notify frontend to switch to this thread
	runtime.EventsEmit(a.ctx, "qap-session-created", map[string]string{
		"threadId":     thread.ID,
		"dataSourceId": dataSourceID,
		"title":        thread.Title,
	})

	// 6. Execute all steps
	summary := a.executeStepLoop(thread.ID, dataSourceID, pack.ExecutableSteps, logTagExecute)

	// 7. Emit completion with execution summary
	completionText := a.buildCompletionText("qap.execution_complete", summary)
	a.emitCompletion(thread.ID, len(pack.ExecutableSteps), completionText)

	a.Log(fmt.Sprintf("%s Execution completed successfully", logTagExecute))

	// 8. Billing
	a.handleBilling(pack.Metadata.ListingID, logTagExecute)

	return nil
}

// ExecuteQuickAnalysisPackDirect executes a quick analysis pack directly with a known datasource,
// skipping the datasource selection step. This is called when the user triggers execution
// from a datasource item's execute button (the datasource context is already known).
func (a *App) ExecuteQuickAnalysisPackDirect(filePath string, dataSourceID string, password string) error {
	a.Log(fmt.Sprintf("%s Direct execution (datasource pre-selected): file=%s, dataSource=%s", logTagExecute, filePath, dataSourceID))
	return a.ExecuteQuickAnalysisPack(filePath, dataSourceID, password)
}

// ReExecuteQuickAnalysisPack re-executes a quick analysis pack in the current session.
// Clears all messages in the thread, then re-executes all steps in the same session.
func (a *App) ReExecuteQuickAnalysisPack(threadID string) error {
	a.Log(fmt.Sprintf("%s Starting re-execution in thread: %s", logTagReexec, threadID))

	// 1. Load the thread to get pack metadata
	thread, err := a.chatService.LoadThread(threadID)
	if err != nil {
		return fmt.Errorf("failed to load thread: %w", err)
	}
	if thread == nil {
		return fmt.Errorf("thread not found: %s", threadID)
	}
	if !thread.IsReplaySession || thread.QapFilePath == "" {
		return fmt.Errorf("%s", i18n.T("qap.not_replay_session"))
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
		return fmt.Errorf("%s", i18n.T("qap.missing_required_tables", strings.Join(result.Validation.MissingTables, ", ")))
	}

	pack := result.Pack

	// 3. Pre-checks: Python environment + marketplace license
	if err := a.preCheckPackExecution(pack, logTagReexec); err != nil {
		return err
	}

	// 4. Clear existing messages in the thread
	if err := a.chatService.ClearThreadMessages(threadID); err != nil {
		a.Log(fmt.Sprintf("%s Error clearing messages: %v", logTagReexec, err))
		return fmt.Errorf("failed to clear messages: %w", err)
	}

	// Clear EventAggregator cache and flushed items to avoid mixing old and new results
	if a.eventAggregator != nil {
		a.eventAggregator.Clear(threadID)
		a.eventAggregator.ClearFlushedItems(threadID)
	}

	runtime.EventsEmit(a.ctx, "thread-updated", threadID)

	// 5. Execute all steps
	summary := a.executeStepLoop(threadID, thread.DataSourceID, pack.ExecutableSteps, logTagReexec)

	// 6. Emit completion with execution summary
	completionText := a.buildCompletionText("qap.reexecution_complete", summary)
	a.emitCompletion(threadID, len(pack.ExecutableSteps), completionText)

	a.Log(fmt.Sprintf("%s Re-execution completed successfully", logTagReexec))

	// 7. Billing
	a.handleBilling(pack.Metadata.ListingID, logTagReexec)

	return nil
}

// ---------------------------------------------------------------------------
// Shared execution helpers
// ---------------------------------------------------------------------------

// preCheckPackExecution validates Python availability and marketplace license before execution.
func (a *App) preCheckPackExecution(pack *QuickAnalysisPack, logTag string) error {
	if a.packHasPythonSteps(pack) && !a.isPythonConfigured() {
		a.Log(fmt.Sprintf("%s Pack contains Python steps but Python environment is not configured", logTag))
		return fmt.Errorf("%s", i18n.T("qap.python_not_configured"))
	}

	listingID := pack.Metadata.ListingID
	if listingID > 0 && a.usageLicenseStore != nil {
		allowed, reason := a.usageLicenseStore.CheckPermission(listingID)
		if !allowed {
			a.Log(fmt.Sprintf("%s Permission denied for listing %d: %s", logTag, listingID, reason))
			return fmt.Errorf("%s", i18n.T("qap.permission_denied", reason))
		}
	}
	return nil
}

// executeStepLoop runs all pack steps sequentially, handling dependency checks,
// progress events, and per-step dispatch.
// StepExecutionSummary holds the execution statistics for a pack run.
type StepExecutionSummary struct {
	Total     int
	Succeeded int
	Failed    int
	Skipped   int
}

func (a *App) executeStepLoop(threadID, dataSourceID string, steps []PackStep, logTag string) StepExecutionSummary {
	totalSteps := len(steps)
	stepResults := make(map[int]interface{}, totalSteps) // Pre-allocate with capacity

	cfg, err := a.GetConfig()
	if err != nil {
		a.Log(fmt.Sprintf("%s Failed to get config: %v", logTag, err))
		return StepExecutionSummary{Total: totalSteps, Failed: totalSteps}
	}
	sessionDir := filepath.Join(cfg.DataCacheDir, "sessions", threadID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		a.Log(fmt.Sprintf("%s Failed to create session directory: %v", logTag, err))
	}

	// Scan existing image files in sessionDir before executing any steps,
	// so that detectAndSendPythonChartFiles only sends newly generated files.
	existingFiles := make(map[string]bool)
	if entries, err := os.ReadDir(sessionDir); err == nil {
		imageExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true}
		for _, entry := range entries {
			if !entry.IsDir() {
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				if imageExts[ext] {
					existingFiles[entry.Name()] = true
				}
			}
		}
		a.Log(fmt.Sprintf("%s Found %d existing image files in session directory", logTag, len(existingFiles)))
	}

	summary := StepExecutionSummary{Total: totalSteps}

	for i, step := range steps {
		a.Log(fmt.Sprintf("%s Executing step %d/%d: %s (%s)", logTag, i+1, totalSteps, step.Description, step.StepType))

		// Check if any dependency step failed
		if a.hasDependencyFailure(step, stepResults, logTag) {
			a.emitSkipMessage(threadID, step)
			summary.Skipped++
			continue
		}

		// Emit progress event
		runtime.EventsEmit(a.ctx, "qap-progress", map[string]interface{}{
			"threadId":    threadID,
			"currentStep": i + 1,
			"totalSteps":  totalSteps,
			"stepType":    step.StepType,
			"description": step.Description,
		})

		messageID := strconv.FormatInt(time.Now().UnixNano(), 10)

		switch step.StepType {
		case stepTypeSQL:
			a.executePackSQLStep(threadID, messageID, step, stepResults, dataSourceID)
		case stepTypePython:
			a.executePackPythonStep(threadID, messageID, step, stepResults, sessionDir, existingFiles)
		default:
			a.Log(fmt.Sprintf("%s Unknown step type: %s, skipping", logTag, step.StepType))
			summary.Skipped++
			continue
		}

		// A step is successful if its result was stored in stepResults
		if _, ok := stepResults[step.StepID]; ok {
			summary.Succeeded++
		} else {
			summary.Failed++
		}

		runtime.EventsEmit(a.ctx, "thread-updated", threadID)
		time.Sleep(100 * time.Millisecond)
	}

	return summary
}

// hasDependencyFailure checks if any dependency step failed (result not in stepResults).
func (a *App) hasDependencyFailure(step PackStep, stepResults map[int]interface{}, logTag string) bool {
	for _, depID := range step.DependsOn {
		if _, ok := stepResults[depID]; !ok {
			a.Log(fmt.Sprintf("%s Skipping step %d: dependency step %d failed or was skipped", logTag, step.StepID, depID))
			return true
		}
	}
	return false
}

// emitSkipMessage sends a skip notification to the chat thread.
func (a *App) emitSkipMessage(threadID string, step PackStep) {
	skipMsg := ChatMessage{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Role:      "assistant",
		Content:   i18n.T("qap.step_skipped", step.StepID, step.Description),
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(threadID, skipMsg)
	runtime.EventsEmit(a.ctx, "thread-updated", threadID)
	time.Sleep(100 * time.Millisecond)
}

// buildCompletionText generates a completion message that includes an execution summary.
func (a *App) buildCompletionText(i18nKey string, summary StepExecutionSummary) string {
	base := i18n.T(i18nKey, summary.Total)
	if summary.Failed == 0 && summary.Skipped == 0 {
		return base
	}
	detail := i18n.T("qap.execution_summary", summary.Succeeded, summary.Failed, summary.Skipped)
	return base + "\n\n" + detail
}

// emitCompletion sends the completion message, flushes results, and emits the qap-complete event.
func (a *App) emitCompletion(threadID string, totalSteps int, completionText string) {
	completionMsg := ChatMessage{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Role:      "assistant",
		Content:   completionText,
		Timestamp: time.Now().Unix(),
	}
	a.chatService.AddMessage(threadID, completionMsg)

	if a.eventAggregator != nil {
		a.eventAggregator.FlushNow(threadID, true)
	}

	runtime.EventsEmit(a.ctx, "qap-complete", map[string]interface{}{
		"threadId":   threadID,
		"totalSteps": totalSteps,
	})
	runtime.EventsEmit(a.ctx, "thread-updated", threadID)
}

// handleBilling consumes usage or validates subscription for marketplace packs after execution.
func (a *App) handleBilling(listingID int64, logTag string) {
	if listingID <= 0 || a.usageLicenseStore == nil {
		return
	}
	lic := a.usageLicenseStore.GetLicense(listingID)
	if lic == nil {
		a.Log(fmt.Sprintf("%s No license found for listing %d, skipping billing", logTag, listingID))
		return
	}

	a.Log(fmt.Sprintf("%s Billing: listing_id=%d, model=%s, remaining=%d, total=%d",
		logTag, listingID, lic.PricingModel, lic.RemainingUses, lic.TotalUses))

	switch lic.PricingModel {
	case "per_use":
		if err := a.usageLicenseStore.ConsumeUse(listingID); err != nil {
			a.Log(fmt.Sprintf("%s Failed to consume use for listing %d: %v", logTag, listingID, err))
		} else {
			_ = a.usageLicenseStore.Save()
			go a.ReportPackUsage(listingID, time.Now().Format(time.RFC3339))
		}
	case "subscription", "time_limited":
		a.ValidateSubscriptionLicenseAsync(listingID)
	}
}

// packHasPythonSteps checks if the pack contains any python_code steps.
func (a *App) packHasPythonSteps(pack *QuickAnalysisPack) bool {
	for _, step := range pack.ExecutableSteps {
		if step.StepType == stepTypePython {
			return true
		}
	}
	return false
}

// isPythonConfigured checks if the Python environment is configured.
func (a *App) isPythonConfigured() bool {
	if a.einoService != nil && a.einoService.HasPython() {
		return true
	}
	cfg, err := a.GetConfig()
	if err == nil && cfg.PythonPath != "" {
		return true
	}
	return a.findPythonPath() != ""
}

// findPythonPath returns a usable Python path, checking config first, then auto-detecting.
func (a *App) findPythonPath() string {
	cfg, _ := a.GetConfig()
	if cfg.PythonPath != "" {
		return cfg.PythonPath
	}
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
