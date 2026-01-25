package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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
	pendingItems map[string]*pendingBatch // sessionId -> pending batch
	mutex        sync.Mutex
	flushTimers  map[string]*time.Timer   // sessionId -> flush timer
	flushDelay   time.Duration            // delay before flushing (default 50ms)
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
		flushTimers:  make(map[string]*time.Timer),
		flushDelay:   50 * time.Millisecond,
	}
}

// generateItemID generates a unique ID for an item
func generateItemID() string {
	return time.Now().Format("20060102150405.000000")
}

// AddItem adds an item to the pending batch for aggregation
func (ea *EventAggregator) AddItem(sessionID, messageID, requestID string, itemType string, data interface{}, metadata map[string]interface{}) {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()

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

// flushAndReturn sends the pending batch as an event and returns the items
func (ea *EventAggregator) flushAndReturn(sessionID string, isComplete bool) []AnalysisResultItem {
	ea.mutex.Lock()
	
	batch, exists := ea.pendingItems[sessionID]
	if !exists || len(batch.items) == 0 {
		ea.mutex.Unlock()
		return nil
	}
	
	// Copy items for return
	items := make([]AnalysisResultItem, len(batch.items))
	copy(items, batch.items)
	
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

// EmitError emits an error event with detailed error information
func (ea *EventAggregator) EmitError(sessionID, requestID, errorMessage string) {
	ea.EmitErrorWithCode(sessionID, requestID, "ANALYSIS_ERROR", errorMessage)
}

// EmitErrorWithCode emits an error event with a specific error code
func (ea *EventAggregator) EmitErrorWithCode(sessionID, requestID, errorCode, errorMessage string) {
	runtime.EventsEmit(ea.ctx, "analysis-error", map[string]interface{}{
		"sessionId": sessionID,
		"threadId":  sessionID, // Also include threadId for compatibility
		"requestId": requestID,
		"code":      errorCode,
		"error":     errorMessage,
		"message":   errorMessage, // Also include message for compatibility
		"timestamp": time.Now().UnixMilli(),
	})
}

// EmitTimeout emits a timeout error event
func (ea *EventAggregator) EmitTimeout(sessionID, requestID string, duration time.Duration) {
	ea.EmitErrorWithCode(sessionID, requestID, "ANALYSIS_TIMEOUT", 
		fmt.Sprintf("分析超时（已运行 %v）。请尝试简化查询或稍后重试。", duration.Round(time.Second)))
}

// EmitCancelled emits a cancellation event
func (ea *EventAggregator) EmitCancelled(sessionID, requestID string) {
	runtime.EventsEmit(ea.ctx, "analysis-cancelled", map[string]interface{}{
		"sessionId": sessionID,
		"threadId":  sessionID,
		"requestId": requestID,
		"message":   "分析已取消",
		"timestamp": time.Now().UnixMilli(),
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
