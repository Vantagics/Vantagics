package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"vantagedata/i18n"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ValidItemTypes defines the list of valid analysis result item types
var ValidItemTypes = map[string]bool{
	"echarts": true,
	"image":   true,
	"table":   true,
	"csv":     true,
	"metric":  true,
	"insight": true,
	"file":    true,
}

// Error code constants for common error types
// These codes help categorize errors and provide appropriate recovery suggestions
const (
	// Analysis errors
	ErrorCodeAnalysisError     = "ANALYSIS_ERROR"      // General analysis error
	ErrorCodeAnalysisTimeout   = "ANALYSIS_TIMEOUT"    // Analysis timed out
	ErrorCodeAnalysisCancelled = "ANALYSIS_CANCELLED"  // Analysis was cancelled
	
	// Python execution errors
	ErrorCodePythonExecution   = "PYTHON_EXECUTION"    // Python code execution failed
	ErrorCodePythonSyntax      = "PYTHON_SYNTAX"       // Python syntax error
	ErrorCodePythonImport      = "PYTHON_IMPORT"       // Python import error
	ErrorCodePythonMemory      = "PYTHON_MEMORY"       // Python memory error
	
	// Data errors
	ErrorCodeDataNotFound      = "DATA_NOT_FOUND"      // Requested data not found
	ErrorCodeDataInvalid       = "DATA_INVALID"        // Data format is invalid
	ErrorCodeDataEmpty         = "DATA_EMPTY"          // Data is empty
	ErrorCodeDataTooLarge      = "DATA_TOO_LARGE"      // Data exceeds size limit
	
	// Connection errors
	ErrorCodeConnectionFailed  = "CONNECTION_FAILED"   // Connection to service failed
	ErrorCodeConnectionTimeout = "CONNECTION_TIMEOUT"  // Connection timed out
	
	// Permission errors
	ErrorCodePermissionDenied  = "PERMISSION_DENIED"   // Permission denied
	
	// Resource errors
	ErrorCodeResourceBusy      = "RESOURCE_BUSY"       // Resource is busy
	ErrorCodeResourceNotFound  = "RESOURCE_NOT_FOUND"  // Resource not found
)

// ErrorInfo contains detailed error information with recovery suggestions
type ErrorInfo struct {
	Code             string   `json:"code"`             // Error code
	Message          string   `json:"message"`          // User-friendly error message
	Details          string   `json:"details"`          // Technical details (optional)
	RecoverySuggestions []string `json:"recoverySuggestions"` // List of recovery suggestions
	Timestamp        int64    `json:"timestamp"`        // Error timestamp
}

// getRecoverySuggestions returns recovery suggestions based on error code
func getRecoverySuggestions(errorCode string) []string {
	suggestions := make([]string, 0)
	
	switch errorCode {
	case ErrorCodeAnalysisError:
		suggestions = append(suggestions, 
			i18n.T("error.recovery.check_query"),
			i18n.T("error.recovery.simplify_query"),
			i18n.T("error.recovery.refresh_retry"))
	
	case ErrorCodeAnalysisTimeout:
		suggestions = append(suggestions,
			i18n.T("error.recovery.reduce_data_range"),
			i18n.T("error.recovery.check_network"),
			i18n.T("error.recovery.retry_later"))
	
	case ErrorCodeAnalysisCancelled:
		suggestions = append(suggestions,
			i18n.T("error.recovery.resubmit"),
			i18n.T("error.recovery.refresh_retry"))
	
	case ErrorCodePythonExecution:
		suggestions = append(suggestions,
			i18n.T("error.recovery.check_data_format"),
			i18n.T("error.recovery.try_different_method"),
			i18n.T("error.recovery.contact_support"))
	
	case ErrorCodePythonSyntax:
		suggestions = append(suggestions,
			i18n.T("error.recovery.rephrase_query"),
			i18n.T("error.recovery.use_simpler_query"))
	
	case ErrorCodePythonImport:
		suggestions = append(suggestions,
			i18n.T("error.recovery.check_libraries"),
			i18n.T("error.recovery.check_admin"),
			i18n.T("error.recovery.try_different_method"))
	
	case ErrorCodePythonMemory:
		suggestions = append(suggestions,
			i18n.T("error.recovery.reduce_data_range"),
			i18n.T("error.recovery.reduce_batch"),
			i18n.T("error.recovery.retry_later"))
	
	case ErrorCodeDataNotFound:
		suggestions = append(suggestions,
			i18n.T("error.recovery.check_datasource"),
			i18n.T("error.recovery.check_table_field"),
			i18n.T("error.recovery.check_deleted"))
	
	case ErrorCodeDataInvalid:
		suggestions = append(suggestions,
			i18n.T("error.recovery.check_data_format"),
			i18n.T("error.recovery.check_data_type"),
			i18n.T("error.recovery.clean_reimport"))
	
	case ErrorCodeDataEmpty:
		suggestions = append(suggestions,
			i18n.T("error.recovery.adjust_filters"),
			i18n.T("error.recovery.check_data_exists"))
	
	case ErrorCodeDataTooLarge:
		suggestions = append(suggestions,
			i18n.T("error.recovery.reduce_data_range"),
			i18n.T("error.recovery.add_filters"),
			i18n.T("error.recovery.consider_pagination"))
	
	case ErrorCodeConnectionFailed:
		suggestions = append(suggestions,
			i18n.T("error.recovery.check_network"),
			i18n.T("error.recovery.check_service"),
			i18n.T("error.recovery.retry_later"))
	
	case ErrorCodeConnectionTimeout:
		suggestions = append(suggestions,
			i18n.T("error.recovery.check_network"),
			i18n.T("error.recovery.retry_later"),
			i18n.T("error.recovery.contact_support"))
	
	case ErrorCodePermissionDenied:
		suggestions = append(suggestions,
			i18n.T("error.recovery.check_permissions"),
			i18n.T("error.recovery.contact_admin"),
			i18n.T("error.recovery.check_account"))
	
	case ErrorCodeResourceBusy:
		suggestions = append(suggestions,
			i18n.T("error.recovery.resource_busy"),
			i18n.T("error.recovery.retry_later"),
			i18n.T("error.recovery.contact_support"))
	
	case ErrorCodeResourceNotFound:
		suggestions = append(suggestions,
			i18n.T("error.recovery.check_path"),
			i18n.T("error.recovery.check_deleted"),
			i18n.T("error.recovery.confirm_resource"))
	
	default:
		suggestions = append(suggestions,
			i18n.T("error.recovery.retry_later"),
			i18n.T("error.recovery.contact_support"))
	}
	
	return suggestions
}

// getUserFriendlyMessage returns a user-friendly message based on error code
func getUserFriendlyMessage(errorCode, originalMessage string) string {
	// If original message is already user-friendly, use it
	if originalMessage != "" && len([]rune(originalMessage)) > 0 {
		// Check if it's already a localized message (contains non-ASCII)
		for _, r := range originalMessage {
			if r > 127 {
				return originalMessage
			}
		}
	}
	
	// Generate user-friendly message based on error code
	switch errorCode {
	case ErrorCodeAnalysisError:
		return i18n.T("error.analysis_error")
	case ErrorCodeAnalysisTimeout:
		return i18n.T("error.analysis_timeout")
	case ErrorCodeAnalysisCancelled:
		return i18n.T("error.analysis_cancelled")
	case ErrorCodePythonExecution:
		return i18n.T("error.python_execution")
	case ErrorCodePythonSyntax:
		return i18n.T("error.python_syntax")
	case ErrorCodePythonImport:
		return i18n.T("error.python_import")
	case ErrorCodePythonMemory:
		return i18n.T("error.python_memory")
	case ErrorCodeDataNotFound:
		return i18n.T("error.data_not_found")
	case ErrorCodeDataInvalid:
		return i18n.T("error.data_invalid")
	case ErrorCodeDataEmpty:
		return i18n.T("error.data_empty")
	case ErrorCodeDataTooLarge:
		return i18n.T("error.data_too_large")
	case ErrorCodeConnectionFailed:
		return i18n.T("error.connection_failed")
	case ErrorCodeConnectionTimeout:
		return i18n.T("error.connection_timeout")
	case ErrorCodePermissionDenied:
		return i18n.T("error.permission_denied")
	case ErrorCodeResourceBusy:
		return i18n.T("error.resource_busy")
	case ErrorCodeResourceNotFound:
		return i18n.T("error.resource_not_found")
	default:
		if originalMessage != "" {
			return originalMessage
		}
		return i18n.T("error.unknown")
	}
}

// createErrorInfo creates a detailed ErrorInfo with recovery suggestions
func createErrorInfo(errorCode, errorMessage, details string) ErrorInfo {
	return ErrorInfo{
		Code:                errorCode,
		Message:             getUserFriendlyMessage(errorCode, errorMessage),
		Details:             details,
		RecoverySuggestions: getRecoverySuggestions(errorCode),
		Timestamp:           time.Now().UnixMilli(),
	}
}

// ItemValidationResult represents the result of analysis item validation
type ItemValidationResult struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors"`
}

// AnalysisResultItem represents a single analysis result item
type AnalysisResultItem struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`     // echarts, image, table, csv, metric, insight, file
	Data     interface{}            `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
	Source   string                 `json:"source"`   // realtime, completed, cached, restored
}

// AnalysisResultBatch represents a batch of analysis results
type AnalysisResultBatch struct {
	SessionID  string               `json:"sessionId"`
	MessageID  string               `json:"messageId"`
	RequestID  string               `json:"requestId"`
	Items      []AnalysisResultItem `json:"items"`
	IsComplete bool                 `json:"isComplete"`
	Timestamp  int64                `json:"timestamp"`
}

// EventAggregator aggregates multiple events into batched updates
type EventAggregator struct {
	ctx          context.Context
	pendingItems map[string]*pendingBatch    // sessionId -> pending batch
	flushedItems map[string][]AnalysisResultItem // sessionId -> all items flushed so far (for persistence)
	mutex        sync.Mutex
	flushTimers  map[string]*time.Timer   // sessionId -> flush timer
	flushDelay   time.Duration            // delay before flushing (default 50ms)
	logger       func(string)             // optional logger function for debug logging
}

// pendingBatch holds items waiting to be flushed
type pendingBatch struct {
	sessionID string
	messageID string
	requestID string
	items     []AnalysisResultItem
}

// NewEventAggregator creates a new EventAggregator
func NewEventAggregator(ctx context.Context) *EventAggregator {
	return &EventAggregator{
		ctx:          ctx,
		pendingItems: make(map[string]*pendingBatch),
		flushedItems: make(map[string][]AnalysisResultItem),
		flushTimers:  make(map[string]*time.Timer),
		flushDelay:   50 * time.Millisecond,
		logger:       nil,
	}
}

// SetLogger sets the logger function for debug logging
func (ea *EventAggregator) SetLogger(logger func(string)) {
	ea.logger = logger
}

// log writes a debug message if logger is set
func (ea *EventAggregator) log(message string) {
	if ea.logger != nil {
		ea.logger(message)
	}
}

// logf writes a formatted debug message if logger is set
func (ea *EventAggregator) logf(format string, args ...interface{}) {
	if ea.logger != nil {
		ea.logger(fmt.Sprintf(format, args...))
	}
}

// generateItemID generates a unique ID for an item
func generateItemID() string {
	return fmt.Sprintf("%s_%d", time.Now().Format("20060102150405.000000"), generateItemSeq())
}

// itemSeqCounter is an atomic counter for generating unique item IDs
var itemSeqCounter uint64

// generateItemSeq returns a monotonically increasing sequence number
func generateItemSeq() uint64 {
	return atomic.AddUint64(&itemSeqCounter, 1)
}

// IsValidItemType checks if the given type is a valid item type
func IsValidItemType(itemType string) bool {
	return ValidItemTypes[itemType]
}

// GetValidItemTypes returns a slice of all valid item types
func GetValidItemTypes() []string {
	types := make([]string, 0, len(ValidItemTypes))
	for t := range ValidItemTypes {
		types = append(types, t)
	}
	return types
}

// ValidateItem validates an analysis result item and returns validation result
// This method can be used to check items before adding them
func (ea *EventAggregator) ValidateItem(sessionID, messageID, requestID string, itemType string, data interface{}) ItemValidationResult {
	result := ItemValidationResult{
		Valid:    true,
		Warnings: []string{},
		Errors:   []string{},
	}

	// Validate sessionID (required)
	if sessionID == "" {
		result.Warnings = append(result.Warnings, "sessionID is empty")
		ea.log("[EVENT-AGG] Warning: sessionID is empty for item validation")
	}

	// Validate item type
	if itemType == "" {
		result.Warnings = append(result.Warnings, "item type is empty")
		ea.log("[EVENT-AGG] Warning: item type is empty")
	} else if !IsValidItemType(itemType) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("invalid item type: %s (valid types: echarts, image, table, csv, metric, insight, file)", itemType))
		ea.logf("[EVENT-AGG] Warning: invalid item type '%s'", itemType)
	}

	// Validate data (required)
	if data == nil {
		result.Warnings = append(result.Warnings, "data is nil")
		ea.log("[EVENT-AGG] Warning: data is nil for item validation")
	} else {
		// Check if data is an empty string
		if strData, ok := data.(string); ok && strData == "" {
			result.Warnings = append(result.Warnings, "data is an empty string")
			ea.log("[EVENT-AGG] Warning: data is an empty string")
		}
	}

	// Optional field warnings (not blocking)
	if messageID == "" {
		ea.log("[EVENT-AGG] Info: messageID is empty (optional field)")
	}
	if requestID == "" {
		ea.log("[EVENT-AGG] Info: requestID is empty (optional field)")
	}

	return result
}

// validateAndLog validates item data and logs warnings, returns true if validation passes (graceful degradation)
func (ea *EventAggregator) validateAndLog(sessionID, messageID, requestID string, itemType string, data interface{}) bool {
	result := ea.ValidateItem(sessionID, messageID, requestID, itemType, data)
	
	// Log all warnings
	for _, warning := range result.Warnings {
		ea.logf("[EVENT-AGG] Validation warning: %s", warning)
	}
	
	// Always return true for graceful degradation - we log warnings but don't block
	return true
}

// AddItem adds an item to the pending batch for aggregation
func (ea *EventAggregator) AddItem(sessionID, messageID, requestID string, itemType string, data interface{}, metadata map[string]interface{}) {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()

	// Log the AddItem call
	ea.logf("[EVENT-AGG] AddItem: type=%s, sessionID=%s, messageID=%s, requestID=%s", itemType, sessionID, messageID, requestID)

	// Validate the item data (graceful degradation - log warnings but don't block)
	ea.validateAndLog(sessionID, messageID, requestID, itemType, data)

	// Create item
	item := AnalysisResultItem{
		ID:       generateItemID(),
		Type:     itemType,
		Data:     data,
		Metadata: metadata,
		Source:   "realtime",
	}

	// Get or create pending batch
	batch, exists := ea.pendingItems[sessionID]
	if !exists {
		batch = &pendingBatch{
			sessionID: sessionID,
			messageID: messageID,
			requestID: requestID,
			items:     []AnalysisResultItem{},
		}
		ea.pendingItems[sessionID] = batch
	}

	// Update messageID and requestID if provided
	if messageID != "" {
		batch.messageID = messageID
	}
	if requestID != "" {
		batch.requestID = requestID
	}

	// Add item to batch
	batch.items = append(batch.items, item)

	// Reset flush timer
	if timer, exists := ea.flushTimers[sessionID]; exists {
		timer.Stop()
	}
	ea.flushTimers[sessionID] = time.AfterFunc(ea.flushDelay, func() {
		ea.logf("[EVENT-AGG] Timer flush triggered for sessionID=%s", sessionID)
		ea.flush(sessionID, false)
	})
}

// AddECharts adds an ECharts item
func (ea *EventAggregator) AddECharts(sessionID, messageID, requestID string, chartData string) {
	ea.AddItem(sessionID, messageID, requestID, "echarts", chartData, map[string]interface{}{
		"sessionId": sessionID,
		"messageId": messageID,
		"timestamp": time.Now().UnixMilli(),
	})
}

// AddImage adds an image item
func (ea *EventAggregator) AddImage(sessionID, messageID, requestID string, imageData string, fileName string) {
	metadata := map[string]interface{}{
		"sessionId": sessionID,
		"messageId": messageID,
		"timestamp": time.Now().UnixMilli(),
	}
	if fileName != "" {
		metadata["fileName"] = fileName
	}
	ea.AddItem(sessionID, messageID, requestID, "image", imageData, metadata)
}

// AddTable adds a table item
func (ea *EventAggregator) AddTable(sessionID, messageID, requestID string, tableData interface{}) {
	ea.AddItem(sessionID, messageID, requestID, "table", tableData, map[string]interface{}{
		"sessionId": sessionID,
		"messageId": messageID,
		"timestamp": time.Now().UnixMilli(),
	})
}

// AddCSV adds a CSV item
func (ea *EventAggregator) AddCSV(sessionID, messageID, requestID string, csvData string, fileName string) {
	metadata := map[string]interface{}{
		"sessionId": sessionID,
		"messageId": messageID,
		"timestamp": time.Now().UnixMilli(),
	}
	if fileName != "" {
		metadata["fileName"] = fileName
	}
	ea.AddItem(sessionID, messageID, requestID, "csv", csvData, metadata)
}

// AddMetric adds a metric item
func (ea *EventAggregator) AddMetric(sessionID, messageID, requestID string, metric Metric) {
	ea.AddItem(sessionID, messageID, requestID, "metric", metric, map[string]interface{}{
		"sessionId": sessionID,
		"messageId": messageID,
		"timestamp": time.Now().UnixMilli(),
	})
}

// AddInsight adds an insight item
func (ea *EventAggregator) AddInsight(sessionID, messageID, requestID string, insight Insight) {
	ea.AddItem(sessionID, messageID, requestID, "insight", insight, map[string]interface{}{
		"sessionId": sessionID,
		"messageId": messageID,
		"timestamp": time.Now().UnixMilli(),
	})
}

// AddFile adds a file item
func (ea *EventAggregator) AddFile(sessionID, messageID, requestID string, fileName, filePath, fileType string) {
	ea.AddItem(sessionID, messageID, requestID, "file", map[string]interface{}{
		"fileName": fileName,
		"filePath": filePath,
		"fileType": fileType,
	}, map[string]interface{}{
		"sessionId": sessionID,
		"messageId": messageID,
		"timestamp": time.Now().UnixMilli(),
		"fileName":  fileName,
	})
}

// FlushNow immediately flushes all pending items for a session
// Returns the items that were flushed (for persistence)
func (ea *EventAggregator) FlushNow(sessionID string, isComplete bool) []AnalysisResultItem {
	ea.logf("[EVENT-AGG] FlushNow called: sessionID=%s, isComplete=%v", sessionID, isComplete)
	
	ea.mutex.Lock()
	
	// Stop any pending timer
	if timer, exists := ea.flushTimers[sessionID]; exists {
		timer.Stop()
		delete(ea.flushTimers, sessionID)
	}
	
	ea.mutex.Unlock()
	
	// Flush with complete flag and return items
	return ea.flushAndReturn(sessionID, isComplete)
}

// GetAllFlushedItems returns all items that have been flushed for a session
// (including items flushed by timer before FlushNow was called).
// This is used for persistence to ensure all analysis results are saved.
func (ea *EventAggregator) GetAllFlushedItems(sessionID string) []AnalysisResultItem {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	
	items := ea.flushedItems[sessionID]
	if len(items) == 0 {
		ea.logf("[EVENT-AGG] GetAllFlushedItems: no flushed items for sessionID=%s", sessionID)
		return nil
	}
	
	// Return a copy
	result := make([]AnalysisResultItem, len(items))
	copy(result, items)
	
	ea.logf("[EVENT-AGG] GetAllFlushedItems: returning %d items for sessionID=%s", len(result), sessionID)
	return result
}

// ClearFlushedItems clears the tracked flushed items for a session.
// Should be called after items have been persisted.
func (ea *EventAggregator) ClearFlushedItems(sessionID string) {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	
	delete(ea.flushedItems, sessionID)
	ea.logf("[EVENT-AGG] ClearFlushedItems: cleared flushed items for sessionID=%s", sessionID)
}

// flushAndReturn sends the pending batch as an event and returns the items
func (ea *EventAggregator) flushAndReturn(sessionID string, isComplete bool) []AnalysisResultItem {
	ea.mutex.Lock()
	
	batch, exists := ea.pendingItems[sessionID]
	if !exists || len(batch.items) == 0 {
		ea.mutex.Unlock()
		ea.logf("[EVENT-AGG] Flush skipped: no pending items for sessionID=%s", sessionID)
		return nil
	}
	
	// Copy items for return
	items := make([]AnalysisResultItem, len(batch.items))
	copy(items, batch.items)
	
	// Track all flushed items for this session (used by GetAllFlushedItems for persistence)
	ea.flushedItems[sessionID] = append(ea.flushedItems[sessionID], items...)
	
	// Log the flush operation with item count
	ea.logf("[EVENT-AGG] Flushing %d items for sessionID=%s, messageID=%s, requestID=%s, isComplete=%v (total flushed: %d)", 
		len(items), batch.sessionID, batch.messageID, batch.requestID, isComplete, len(ea.flushedItems[sessionID]))
	
	// Create the event payload
	payload := AnalysisResultBatch{
		SessionID:  batch.sessionID,
		MessageID:  batch.messageID,
		RequestID:  batch.requestID,
		Items:      batch.items,
		IsComplete: isComplete,
		Timestamp:  time.Now().UnixMilli(),
	}
	
	// Clear the pending batch
	delete(ea.pendingItems, sessionID)
	delete(ea.flushTimers, sessionID)
	
	ea.mutex.Unlock()
	
	// Emit the event
	runtime.EventsEmit(ea.ctx, "analysis-result-update", payload)
	
	ea.logf("[EVENT-AGG] Emitted 'analysis-result-update' event with %d items", len(items))
	
	return items
}

// flush sends the pending batch as an event (used by timer)
func (ea *EventAggregator) flush(sessionID string, isComplete bool) {
	ea.flushAndReturn(sessionID, isComplete)
}

// Clear clears all pending items for a session
func (ea *EventAggregator) Clear(sessionID string) {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	
	if timer, exists := ea.flushTimers[sessionID]; exists {
		timer.Stop()
		delete(ea.flushTimers, sessionID)
	}
	delete(ea.pendingItems, sessionID)
	delete(ea.flushedItems, sessionID)
	
	// Emit clear event
	runtime.EventsEmit(ea.ctx, "analysis-result-clear", map[string]interface{}{
		"sessionId": sessionID,
	})
}

// SetLoading emits a loading state event
func (ea *EventAggregator) SetLoading(sessionID string, loading bool, requestID string) {
	runtime.EventsEmit(ea.ctx, "analysis-result-loading", map[string]interface{}{
		"sessionId": sessionID,
		"loading":   loading,
		"requestId": requestID,
	})
}

// EmitError emits an error event with detailed error information and recovery suggestions
func (ea *EventAggregator) EmitError(sessionID, requestID, errorMessage string) {
	ea.EmitErrorWithCode(sessionID, requestID, ErrorCodeAnalysisError, errorMessage)
}

// EmitErrorWithCode emits an error event with a specific error code and recovery suggestions
func (ea *EventAggregator) EmitErrorWithCode(sessionID, requestID, errorCode, errorMessage string) {
	// Create detailed error info with recovery suggestions
	errorInfo := createErrorInfo(errorCode, errorMessage, "")
	
	ea.logf("[EVENT-AGG] Emitting error: code=%s, message=%s, suggestions=%d", 
		errorCode, errorInfo.Message, len(errorInfo.RecoverySuggestions))
	
	runtime.EventsEmit(ea.ctx, "analysis-error", map[string]interface{}{
		"sessionId":           sessionID,
		"threadId":            sessionID, // Also include threadId for compatibility
		"requestId":           requestID,
		"code":                errorInfo.Code,
		"error":               errorInfo.Message,
		"message":             errorInfo.Message, // Also include message for compatibility
		"details":             errorInfo.Details,
		"recoverySuggestions": errorInfo.RecoverySuggestions,
		"timestamp":           errorInfo.Timestamp,
	})
}

// EmitErrorWithDetails emits an error event with detailed information including technical details
func (ea *EventAggregator) EmitErrorWithDetails(sessionID, requestID, errorCode, errorMessage, details string) {
	// Create detailed error info with recovery suggestions
	errorInfo := createErrorInfo(errorCode, errorMessage, details)
	
	ea.logf("[EVENT-AGG] Emitting error with details: code=%s, message=%s, details=%s, suggestions=%d", 
		errorCode, errorInfo.Message, details, len(errorInfo.RecoverySuggestions))
	
	runtime.EventsEmit(ea.ctx, "analysis-error", map[string]interface{}{
		"sessionId":           sessionID,
		"threadId":            sessionID,
		"requestId":           requestID,
		"code":                errorInfo.Code,
		"error":               errorInfo.Message,
		"message":             errorInfo.Message,
		"details":             errorInfo.Details,
		"recoverySuggestions": errorInfo.RecoverySuggestions,
		"timestamp":           errorInfo.Timestamp,
	})
}

// EmitTimeout emits a timeout error event with recovery suggestions
func (ea *EventAggregator) EmitTimeout(sessionID, requestID string, duration time.Duration) {
	ea.EmitErrorWithDetails(sessionID, requestID, ErrorCodeAnalysisTimeout, 
		i18n.T("error.analysis_timeout_duration", duration.Round(time.Second)),
		fmt.Sprintf("Analysis timed out after %v", duration.Round(time.Second)))
}

// EmitCancelled emits a cancellation event with recovery suggestions
func (ea *EventAggregator) EmitCancelled(sessionID, requestID string) {
	// Create error info for cancellation
	errorInfo := createErrorInfo(ErrorCodeAnalysisCancelled, i18n.T("error.analysis_cancelled"), "")
	
	runtime.EventsEmit(ea.ctx, "analysis-cancelled", map[string]interface{}{
		"sessionId":           sessionID,
		"threadId":            sessionID,
		"requestId":           requestID,
		"code":                errorInfo.Code,
		"message":             errorInfo.Message,
		"recoverySuggestions": errorInfo.RecoverySuggestions,
		"timestamp":           errorInfo.Timestamp,
	})
}

// EmitDashboardUpdate adds items to the aggregator (new unified event system only)
func (ea *EventAggregator) EmitDashboardUpdate(sessionID, messageID, requestID string, itemType string, data interface{}) {
	// Add to aggregator for new event system
	ea.AddItem(sessionID, messageID, requestID, itemType, data, map[string]interface{}{
		"sessionId": sessionID,
		"messageId": messageID,
		"timestamp": time.Now().UnixMilli(),
	})
}

// EmitDashboardDataUpdate adds metrics and insights to the aggregator (new unified event system only)
func (ea *EventAggregator) EmitDashboardDataUpdate(sessionID, messageID, requestID string, dashboardData DashboardData) {
	// Add metrics and insights to aggregator
	for _, metric := range dashboardData.Metrics {
		ea.AddMetric(sessionID, messageID, requestID, metric)
	}
	for _, insight := range dashboardData.Insights {
		ea.AddInsight(sessionID, messageID, requestID, insight)
	}
}
