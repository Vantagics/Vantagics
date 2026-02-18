package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"vantagedata/i18n"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// ExportableRequest represents a single user analysis request that can be selected for export.
type ExportableRequest struct {
	RequestID        string `json:"request_id"`         // messageID key from executions.json
	UserRequest      string `json:"user_request"`       // The user's original request text
	StepCount        int    `json:"step_count"`         // Number of executable steps in this request
	Timestamp        int64  `json:"timestamp"`          // Timestamp for ordering
	IsAutoSuggestion bool   `json:"is_auto_suggestion"` // True if this is the system auto-generated first analysis suggestion
}

// suggestionPatterns are the known auto-generated first analysis suggestion patterns.
var suggestionPatterns = []string{
	"请给出一些本数据源的分析建议",
	"Give me some analysis suggestions for this data source",
	"请给出一些本数据源的分析建议。",
	"Give me some analysis suggestions for this data source.",
}

// isAutoSuggestionRequest checks if a user request matches the auto-generated suggestion patterns.
func isAutoSuggestionRequest(userRequest string) bool {
	trimmed := strings.TrimSpace(userRequest)
	for _, pattern := range suggestionPatterns {
		if trimmed == pattern {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Loading executions
// ---------------------------------------------------------------------------

// loadExecutionsMap reads and parses executions.json once.
func (a *App) loadExecutionsMap(threadID string) (parsedExecutionsMap, error) {
	cfg, _ := a.GetConfig()
	executionsPath := filepath.Join(cfg.DataCacheDir, "sessions", threadID, "execution", "executions.json")

	data, err := os.ReadFile(executionsPath)
	if err != nil {
		return nil, err
	}

	var m parsedExecutionsMap
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Preview: list exportable requests
// ---------------------------------------------------------------------------

// GetThreadExportableRequests returns the list of analysis requests in a thread
// that have executable steps, suitable for display in the export selection UI.
// The first auto-generated suggestion is flagged with IsAutoSuggestion=true.
func (a *App) GetThreadExportableRequests(threadID string) ([]ExportableRequest, error) {
	a.Log(fmt.Sprintf("%s Getting exportable requests for thread: %s", logTagPreview, threadID))

	executionsMap, err := a.loadExecutionsMap(threadID)
	if err != nil {
		a.Log(fmt.Sprintf("%s executions.json load error: %v", logTagPreview, err))
		return nil, fmt.Errorf("%s", i18n.T("qap.no_exportable_records"))
	}

	requests := buildExportableRequests(executionsMap)

	// Sort by timestamp (chronological order)
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].Timestamp < requests[j].Timestamp
	})

	// Fallback: if the earliest request wasn't detected as auto-suggestion by its own text,
	// check the first user message in the thread to see if the session started with one.
	a.tryMarkFirstAutoSuggestion(requests, threadID)

	a.Log(fmt.Sprintf("%s Found %d exportable requests", logTagPreview, len(requests)))
	return requests, nil
}

// buildExportableRequests converts a parsedExecutionsMap into a flat list of ExportableRequest,
// keeping only entries that have at least one successful step with code.
func buildExportableRequests(executionsMap parsedExecutionsMap) []ExportableRequest {
	var requests []ExportableRequest
	for msgID, entry := range executionsMap {
		stepCount := countSuccessfulSteps(entry.Executions)
		if stepCount == 0 {
			continue
		}

		userReq := entry.UserRequest
		if userReq == "" {
			userReq = i18n.T("qap.unknown_request")
		}

		requests = append(requests, ExportableRequest{
			RequestID:        msgID,
			UserRequest:      userReq,
			StepCount:        stepCount,
			Timestamp:        entry.Timestamp,
			IsAutoSuggestion: isAutoSuggestionRequest(userReq),
		})
	}
	return requests
}

// countSuccessfulSteps returns the number of executions that have non-empty code and succeeded.
func countSuccessfulSteps(execs []parsedExecutionRecord) int {
	n := 0
	for _, exec := range execs {
		if exec.Code != "" && exec.Success {
			n++
		}
	}
	return n
}

// tryMarkFirstAutoSuggestion checks the thread's first user message and, if it matches
// a suggestion pattern, marks the earliest request as auto-suggestion.
func (a *App) tryMarkFirstAutoSuggestion(requests []ExportableRequest, threadID string) {
	if len(requests) == 0 || requests[0].IsAutoSuggestion || a.chatService == nil {
		return
	}
	thread, err := a.chatService.LoadThread(threadID)
	if err != nil || thread == nil {
		return
	}
	for _, msg := range thread.Messages {
		if msg.Role == "user" {
			if isAutoSuggestionRequest(msg.Content) {
				requests[0].IsAutoSuggestion = true
			}
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Export: selective pack export
// ---------------------------------------------------------------------------

// ExportQuickAnalysisPackSelected exports a quick analysis pack with only the selected requests.
// selectedRequestIDs is a list of messageIDs from executions.json to include.
// If empty, all requests are included (backward compatible).
func (a *App) ExportQuickAnalysisPackSelected(threadID string, packName string, author string, password string, selectedRequestIDs []string) (string, error) {
	if len(selectedRequestIDs) == 0 {
		return a.ExportQuickAnalysisPack(threadID, packName, author, password)
	}

	a.Log(fmt.Sprintf("%s Starting selective export for thread: %s, selected %d requests", logTagExport, threadID, len(selectedRequestIDs)))

	// 1. Load the thread
	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}
	thread, err := a.chatService.LoadThread(threadID)
	if err != nil {
		return "", fmt.Errorf("failed to load thread: %w", err)
	}

	// 2. Parse executions.json once, then extract steps for selected requests only
	executionsMap, err := a.loadExecutionsMap(threadID)
	if err != nil {
		return "", fmt.Errorf("failed to load executions: %w", err)
	}

	steps := extractStepsFromEntries(executionsMap, selectedRequestIDs)
	if len(steps) == 0 {
		return "", fmt.Errorf("%s", i18n.T("qap.no_exportable_steps"))
	}

	// 3. Merge trajectory-based Python steps that belong to selected requests
	a.mergePythonTrajectorySteps(&steps, executionsMap, selectedRequestIDs, threadID)

	// 4. Re-number step IDs and rebuild dependencies
	renumberSteps(steps)

	a.Log(fmt.Sprintf("%s Collected %d steps from selected requests", logTagExport, len(steps)))

	// 5. Build and save the pack (shared logic)
	return a.buildAndSavePack(thread, steps, packName, author, password)
}

// mergePythonTrajectorySteps collects Python steps from trajectory data and merges
// them into the existing steps slice, filtering by selected request IDs.
func (a *App) mergePythonTrajectorySteps(steps *[]PackStep, executionsMap parsedExecutionsMap, selectedRequestIDs []string, threadID string) {
	pythonSteps, _ := a.collectPythonStepsFromTrajectory(threadID)
	if len(pythonSteps) == 0 {
		return
	}

	// Build lookup sets
	seenCode := make(map[string]bool, len(*steps))
	for _, s := range *steps {
		if s.StepType == stepTypePython {
			seenCode[s.Code] = true
		}
	}

	selectedUserRequests := make(map[string]bool, len(selectedRequestIDs))
	for _, id := range selectedRequestIDs {
		if entry, ok := executionsMap[id]; ok && entry.UserRequest != "" {
			selectedUserRequests[entry.UserRequest] = true
		}
	}

	for _, s := range pythonSteps {
		if seenCode[s.Code] {
			continue
		}
		// If the Python step has a UserRequest, only include it if it matches a selected request.
		// If it has no UserRequest (trajectory-only), include it as a best-effort.
		if s.UserRequest != "" && !selectedUserRequests[s.UserRequest] {
			continue
		}
		*steps = append(*steps, s)
		seenCode[s.Code] = true
	}
}

// ---------------------------------------------------------------------------
// Build & save pack (shared export logic)
// ---------------------------------------------------------------------------

// buildAndSavePack handles the common export logic: ECharts attachment, schema collection,
// metadata assembly, JSON marshaling, and ZIP packaging.
func (a *App) buildAndSavePack(thread *ChatThread, steps []PackStep, packName, author, password string) (string, error) {
	// Attach ECharts configs from assistant messages
	a.attachEChartsFromMessages(thread, steps)

	// Collect and filter schema to only referenced tables
	fullSchema, err := a.collectFullSchema(thread.DataSourceID)
	if err != nil {
		return "", fmt.Errorf("failed to collect schema: %w", err)
	}
	schema := filterSchemaByReferencedTables(fullSchema, steps)

	// Get data source display name
	dsInfo, err := a.getDataSourceInfoForThread(thread.DataSourceID)
	if err != nil {
		return "", fmt.Errorf("failed to get data source info: %w", err)
	}

	// Assemble pack
	pack := QuickAnalysisPack{
		FileType:      qapFileType,
		FormatVersion: qapFormatVersion,
		Metadata: PackMetadata{
			PackName:   packName,
			Author:     author,
			CreatedAt:  time.Now().Format(time.RFC3339),
			SourceName: dsInfo.Name,
		},
		SchemaRequirements: schema,
		ExecutableSteps:    steps,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal pack: %w", err)
	}

	// Write ZIP file
	savePath, err := a.writePackZip(jsonData, password)
	if err != nil {
		return "", err
	}

	a.Log(fmt.Sprintf("%s Successfully exported to: %s", logTagExport, savePath))
	return savePath, nil
}

// writePackZip creates the qap directory if needed and writes the ZIP file.
func (a *App) writePackZip(jsonData []byte, password string) (string, error) {
	cfg, err := a.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	qapDir := filepath.Join(cfg.DataCacheDir, qapSubDir)
	if err := os.MkdirAll(qapDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create qap directory: %w", err)
	}
	savePath := filepath.Join(qapDir, fmt.Sprintf("analysis_%s.qap", time.Now().Format("20060102_150405")))

	if err := PackToZip(jsonData, savePath, password); err != nil {
		return "", fmt.Errorf("failed to create pack file: %w", err)
	}
	return savePath, nil
}

// ---------------------------------------------------------------------------
// Step extraction from parsed executions
// ---------------------------------------------------------------------------

// extractStepsFromEntries extracts PackSteps from a pre-parsed executionsMap for the given request IDs.
func extractStepsFromEntries(executionsMap parsedExecutionsMap, selectedRequestIDs []string) []PackStep {
	selectedSet := make(map[string]bool, len(selectedRequestIDs))
	for _, id := range selectedRequestIDs {
		selectedSet[id] = true
	}

	// Collect and sort selected entries by timestamp
	type requestEntry struct {
		key   string
		entry parsedExecutionEntry
	}
	var entries []requestEntry
	for k, v := range executionsMap {
		if selectedSet[k] {
			entries = append(entries, requestEntry{key: k, entry: v})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].entry.Timestamp < entries[j].entry.Timestamp
	})

	var steps []PackStep
	stepID := 1
	seenCode := make(map[string]bool)

	for _, re := range entries {
		for _, exec := range re.entry.Executions {
			if exec.Code == "" || !exec.Success {
				continue
			}

			codeKey := exec.Type + ":" + exec.Code
			if seenCode[codeKey] {
				continue
			}
			seenCode[codeKey] = true

			stepType := execTypeToStepType(exec.Type)
			if stepType == "" {
				continue
			}

			steps = append(steps, PackStep{
				StepID:      stepID,
				StepType:    stepType,
				Code:        exec.Code,
				Description: stepDescription(exec, stepType),
				UserRequest: exec.UserRequest,
				DependsOn:   buildDependsOn(stepID),
			})
			stepID++
		}
	}

	return steps
}


// stepDescription returns a human-readable description for a step,
// falling back through StepDescription → UserRequest → default label.
func stepDescription(exec parsedExecutionRecord, stepType string) string {
	if exec.StepDescription != "" {
		return exec.StepDescription
	}
	if exec.UserRequest != "" {
		return exec.UserRequest
	}
	if stepType == stepTypeSQL {
		return "SQL Query"
	}
	return "Python Code"
}
