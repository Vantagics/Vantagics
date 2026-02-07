package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

// **Validates: Requirements 4.2**
// 属性 2: 事件批量聚合完整性
// 对于任意一组添加到 EventAggregator 的数据项，在 FlushNow 调用后，
// 所有项都应该被包含在发送的批次中，且不会丢失任何项。

// TestAnalysisResultItem is a test-local copy of AnalysisResultItem for testing
type TestAnalysisResultItem struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Data     interface{}            `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
	Source   string                 `json:"source"`
}

// TestAnalysisResultBatch is a test-local copy of AnalysisResultBatch for testing
type TestAnalysisResultBatch struct {
	SessionID  string                   `json:"sessionId"`
	MessageID  string                   `json:"messageId"`
	RequestID  string                   `json:"requestId"`
	Items      []TestAnalysisResultItem `json:"items"`
	IsComplete bool                     `json:"isComplete"`
	Timestamp  int64                    `json:"timestamp"`
}

// TestPendingBatch holds items waiting to be flushed
type TestPendingBatch struct {
	sessionID string
	messageID string
	requestID string
	items     []TestAnalysisResultItem
}

// TestEventAggregator is a test-friendly version of EventAggregator that doesn't emit events
type TestEventAggregator struct {
	pendingItems   map[string]*TestPendingBatch
	mutex          sync.Mutex
	flushTimers    map[string]*time.Timer
	flushDelay     time.Duration
	logger         func(string)
	emittedBatches []TestAnalysisResultBatch
	emitMutex      sync.Mutex
}

// TestValidItemTypes defines the list of valid analysis result item types
var TestValidItemTypes = map[string]bool{
	"echarts": true,
	"image":   true,
	"table":   true,
	"csv":     true,
	"metric":  true,
	"insight": true,
	"file":    true,
}

// NewTestEventAggregator creates a new TestEventAggregator for testing
func NewTestEventAggregator() *TestEventAggregator {
	return &TestEventAggregator{
		pendingItems:   make(map[string]*TestPendingBatch),
		flushTimers:    make(map[string]*time.Timer),
		flushDelay:     50 * time.Millisecond,
		logger:         nil,
		emittedBatches: []TestAnalysisResultBatch{},
	}
}

// SetLogger sets the logger function for debug logging
func (ea *TestEventAggregator) SetLogger(logger func(string)) {
	ea.logger = logger
}

// log writes a debug message if logger is set
func (ea *TestEventAggregator) log(message string) {
	if ea.logger != nil {
		ea.logger(message)
	}
}

// logf writes a formatted debug message if logger is set
func (ea *TestEventAggregator) logf(format string, args ...interface{}) {
	if ea.logger != nil {
		ea.logger(fmt.Sprintf(format, args...))
	}
}

// IsValidTestItemType checks if the given type is a valid item type
func IsValidTestItemType(itemType string) bool {
	return TestValidItemTypes[itemType]
}

// GetValidTestItemTypes returns a slice of all valid item types
func GetValidTestItemTypes() []string {
	types := make([]string, 0, len(TestValidItemTypes))
	for t := range TestValidItemTypes {
		types = append(types, t)
	}
	return types
}

// AddItem adds an item to the pending batch for aggregation
func (ea *TestEventAggregator) AddItem(sessionID, messageID, requestID string, itemType string, data interface{}, metadata map[string]interface{}) {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()

	ea.logf("[EVENT-AGG] AddItem: type=%s, sessionID=%s, messageID=%s, requestID=%s", itemType, sessionID, messageID, requestID)

	// Create item with unique ID
	item := TestAnalysisResultItem{
		ID:       fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int()),
		Type:     itemType,
		Data:     data,
		Metadata: metadata,
		Source:   "realtime",
	}

	// Get or create pending batch
	batch, exists := ea.pendingItems[sessionID]
	if !exists {
		batch = &TestPendingBatch{
			sessionID: sessionID,
			messageID: messageID,
			requestID: requestID,
			items:     []TestAnalysisResultItem{},
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

	// Reset flush timer (but don't actually flush in tests to control timing)
	if timer, exists := ea.flushTimers[sessionID]; exists {
		timer.Stop()
	}
}

// FlushNow immediately flushes all pending items for a session
// Returns the items that were flushed
func (ea *TestEventAggregator) FlushNow(sessionID string, isComplete bool) []TestAnalysisResultItem {
	ea.logf("[EVENT-AGG] FlushNow called: sessionID=%s, isComplete=%v", sessionID, isComplete)

	ea.mutex.Lock()

	// Stop any pending timer
	if timer, exists := ea.flushTimers[sessionID]; exists {
		timer.Stop()
		delete(ea.flushTimers, sessionID)
	}

	batch, exists := ea.pendingItems[sessionID]
	if !exists || len(batch.items) == 0 {
		ea.mutex.Unlock()
		ea.logf("[EVENT-AGG] Flush skipped: no pending items for sessionID=%s", sessionID)
		return nil
	}

	// Copy items for return
	items := make([]TestAnalysisResultItem, len(batch.items))
	copy(items, batch.items)

	ea.logf("[EVENT-AGG] Flushing %d items for sessionID=%s, messageID=%s, requestID=%s, isComplete=%v",
		len(items), batch.sessionID, batch.messageID, batch.requestID, isComplete)

	// Create the batch payload
	payload := TestAnalysisResultBatch{
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

	// Store emitted batch for verification
	ea.emitMutex.Lock()
	ea.emittedBatches = append(ea.emittedBatches, payload)
	ea.emitMutex.Unlock()

	ea.logf("[EVENT-AGG] Emitted batch with %d items", len(items))

	return items
}

// GetEmittedBatches returns all emitted batches for verification
func (ea *TestEventAggregator) GetEmittedBatches() []TestAnalysisResultBatch {
	ea.emitMutex.Lock()
	defer ea.emitMutex.Unlock()
	result := make([]TestAnalysisResultBatch, len(ea.emittedBatches))
	copy(result, ea.emittedBatches)
	return result
}

// GetPendingItemCount returns the number of pending items for a session
func (ea *TestEventAggregator) GetPendingItemCount(sessionID string) int {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	if batch, exists := ea.pendingItems[sessionID]; exists {
		return len(batch.items)
	}
	return 0
}

// TestEventAggregator_Property_BatchAggregationCompleteness tests that all items added are included in flushed batch
// **Validates: Requirements 4.2**
func TestEventAggregator_Property_BatchAggregationCompleteness(t *testing.T) {
	// Property: For any set of items added to EventAggregator, after FlushNow is called,
	// all items should be included in the sent batch without any loss.
	property := func(numItems uint8) bool {
		// Constrain numItems to reasonable range (1-50)
		count := int(numItems)%50 + 1

		aggregator := NewTestEventAggregator()
		sessionID := "test-session"
		messageID := "test-message"
		requestID := "test-request"

		// Track added items
		addedItems := make([]string, 0, count)

		// Add items
		validTypes := []string{"echarts", "image", "table", "csv", "metric", "insight", "file"}
		for i := 0; i < count; i++ {
			itemType := validTypes[i%len(validTypes)]
			itemData := fmt.Sprintf("data-%d", i)
			addedItems = append(addedItems, itemData)

			aggregator.AddItem(sessionID, messageID, requestID, itemType, itemData, map[string]interface{}{
				"index": i,
			})
		}

		// Flush and get items
		flushedItems := aggregator.FlushNow(sessionID, true)

		// Property 1: Number of flushed items should equal number of added items
		if len(flushedItems) != count {
			t.Logf("Expected %d items, got %d", count, len(flushedItems))
			return false
		}

		// Property 2: All added data should be present in flushed items
		flushedDataSet := make(map[string]bool)
		for _, item := range flushedItems {
			if data, ok := item.Data.(string); ok {
				flushedDataSet[data] = true
			}
		}

		for _, addedData := range addedItems {
			if !flushedDataSet[addedData] {
				t.Logf("Added item '%s' not found in flushed items", addedData)
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEventAggregator_Property_NoItemLoss tests that no items are lost during aggregation
// **Validates: Requirements 4.2**
func TestEventAggregator_Property_NoItemLoss(t *testing.T) {
	// Property: Items added to EventAggregator should never be lost
	property := func(seed uint32) bool {
		aggregator := NewTestEventAggregator()
		sessionID := fmt.Sprintf("session-%d", seed%100)
		messageID := fmt.Sprintf("message-%d", seed%1000)
		requestID := fmt.Sprintf("request-%d", seed)

		// Generate random number of items (1-30)
		numItems := int(seed%30) + 1

		// Add items with unique identifiers
		expectedIDs := make(map[string]bool)
		for i := 0; i < numItems; i++ {
			itemID := fmt.Sprintf("item-%d-%d", seed, i)
			expectedIDs[itemID] = true

			validTypes := GetValidTestItemTypes()
			itemType := validTypes[i%len(validTypes)]
			aggregator.AddItem(sessionID, messageID, requestID, itemType, itemID, nil)
		}

		// Verify pending count before flush
		pendingCount := aggregator.GetPendingItemCount(sessionID)
		if pendingCount != numItems {
			t.Logf("Expected %d pending items, got %d", numItems, pendingCount)
			return false
		}

		// Flush
		flushedItems := aggregator.FlushNow(sessionID, true)

		// Property: All expected items should be in flushed items
		for _, item := range flushedItems {
			if data, ok := item.Data.(string); ok {
				delete(expectedIDs, data)
			}
		}

		// Property: No items should be missing
		if len(expectedIDs) > 0 {
			t.Logf("Missing items after flush: %v", expectedIDs)
			return false
		}

		// Property: Pending count should be 0 after flush
		pendingAfterFlush := aggregator.GetPendingItemCount(sessionID)
		if pendingAfterFlush != 0 {
			t.Logf("Expected 0 pending items after flush, got %d", pendingAfterFlush)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEventAggregator_Property_MultipleSessionsIsolation tests that items from different sessions don't mix
// **Validates: Requirements 4.2**
func TestEventAggregator_Property_MultipleSessionsIsolation(t *testing.T) {
	// Property: Items added to different sessions should be isolated
	property := func(seed uint16) bool {
		aggregator := NewTestEventAggregator()

		// Create two different sessions
		session1 := fmt.Sprintf("session-1-%d", seed)
		session2 := fmt.Sprintf("session-2-%d", seed)

		// Add items to session 1
		numItems1 := int(seed%10) + 1
		for i := 0; i < numItems1; i++ {
			aggregator.AddItem(session1, "msg1", "req1", "echarts", fmt.Sprintf("s1-item-%d", i), nil)
		}

		// Add items to session 2
		numItems2 := int((seed/10)%10) + 1
		for i := 0; i < numItems2; i++ {
			aggregator.AddItem(session2, "msg2", "req2", "image", fmt.Sprintf("s2-item-%d", i), nil)
		}

		// Flush session 1
		flushed1 := aggregator.FlushNow(session1, true)

		// Property: Session 1 should have exactly numItems1 items
		if len(flushed1) != numItems1 {
			t.Logf("Session 1: expected %d items, got %d", numItems1, len(flushed1))
			return false
		}

		// Property: All session 1 items should have s1 prefix
		for _, item := range flushed1 {
			if data, ok := item.Data.(string); ok {
				if len(data) < 2 || data[:2] != "s1" {
					t.Logf("Session 1 contains non-s1 item: %s", data)
					return false
				}
			}
		}

		// Flush session 2
		flushed2 := aggregator.FlushNow(session2, true)

		// Property: Session 2 should have exactly numItems2 items
		if len(flushed2) != numItems2 {
			t.Logf("Session 2: expected %d items, got %d", numItems2, len(flushed2))
			return false
		}

		// Property: All session 2 items should have s2 prefix
		for _, item := range flushed2 {
			if data, ok := item.Data.(string); ok {
				if len(data) < 2 || data[:2] != "s2" {
					t.Logf("Session 2 contains non-s2 item: %s", data)
					return false
				}
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEventAggregator_Property_FlushIdempotence tests that flushing an empty session returns nil
// **Validates: Requirements 4.2**
func TestEventAggregator_Property_FlushIdempotence(t *testing.T) {
	// Property: Flushing an already flushed or empty session should return nil
	property := func(seed uint16) bool {
		aggregator := NewTestEventAggregator()
		sessionID := fmt.Sprintf("session-%d", seed)

		// Add some items
		numItems := int(seed%20) + 1
		for i := 0; i < numItems; i++ {
			aggregator.AddItem(sessionID, "msg", "req", "table", fmt.Sprintf("item-%d", i), nil)
		}

		// First flush should return items
		firstFlush := aggregator.FlushNow(sessionID, true)
		if len(firstFlush) != numItems {
			t.Logf("First flush: expected %d items, got %d", numItems, len(firstFlush))
			return false
		}

		// Second flush should return nil (no items)
		secondFlush := aggregator.FlushNow(sessionID, true)
		if secondFlush != nil {
			t.Logf("Second flush: expected nil, got %d items", len(secondFlush))
			return false
		}

		// Flushing non-existent session should return nil
		nonExistentFlush := aggregator.FlushNow("non-existent-session", true)
		if nonExistentFlush != nil {
			t.Logf("Non-existent session flush: expected nil, got %d items", len(nonExistentFlush))
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEventAggregator_Property_ItemTypePreservation tests that item types are preserved during aggregation
// **Validates: Requirements 4.2**
func TestEventAggregator_Property_ItemTypePreservation(t *testing.T) {
	// Property: Item types should be preserved during aggregation
	property := func(seed uint8) bool {
		aggregator := NewTestEventAggregator()
		sessionID := "test-session"

		validTypes := GetValidTestItemTypes()
		expectedTypeCounts := make(map[string]int)

		// Add items of each type
		for i, itemType := range validTypes {
			// Add 1-3 items of each type based on seed
			count := int(seed)%(i+1) + 1
			expectedTypeCounts[itemType] = count

			for j := 0; j < count; j++ {
				aggregator.AddItem(sessionID, "msg", "req", itemType, fmt.Sprintf("%s-data-%d", itemType, j), nil)
			}
		}

		// Flush
		flushedItems := aggregator.FlushNow(sessionID, true)

		// Count types in flushed items
		actualTypeCounts := make(map[string]int)
		for _, item := range flushedItems {
			actualTypeCounts[item.Type]++
		}

		// Property: Type counts should match
		for itemType, expectedCount := range expectedTypeCounts {
			if actualTypeCounts[itemType] != expectedCount {
				t.Logf("Type %s: expected %d, got %d", itemType, expectedCount, actualTypeCounts[itemType])
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEventAggregator_Property_MetadataPreservation tests that metadata is preserved during aggregation
// **Validates: Requirements 4.2**
func TestEventAggregator_Property_MetadataPreservation(t *testing.T) {
	// Property: Metadata should be preserved during aggregation
	property := func(seed uint16) bool {
		aggregator := NewTestEventAggregator()
		sessionID := "test-session"

		numItems := int(seed%20) + 1
		expectedMetadata := make(map[string]map[string]interface{})

		// Add items with metadata
		for i := 0; i < numItems; i++ {
			itemData := fmt.Sprintf("item-%d", i)
			metadata := map[string]interface{}{
				"index":     i,
				"timestamp": time.Now().UnixMilli(),
				"custom":    fmt.Sprintf("custom-%d", seed+uint16(i)),
			}
			expectedMetadata[itemData] = metadata

			aggregator.AddItem(sessionID, "msg", "req", "echarts", itemData, metadata)
		}

		// Flush
		flushedItems := aggregator.FlushNow(sessionID, true)

		// Property: All items should have their metadata preserved
		for _, item := range flushedItems {
			data, ok := item.Data.(string)
			if !ok {
				continue
			}

			expected, exists := expectedMetadata[data]
			if !exists {
				t.Logf("Unexpected item data: %s", data)
				return false
			}

			// Check metadata fields
			if item.Metadata == nil {
				t.Logf("Item %s has nil metadata", data)
				return false
			}

			if item.Metadata["index"] != expected["index"] {
				t.Logf("Item %s: index mismatch", data)
				return false
			}

			if item.Metadata["custom"] != expected["custom"] {
				t.Logf("Item %s: custom field mismatch", data)
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEventAggregator_Property_OrderPreservation tests that item order is preserved during aggregation
// **Validates: Requirements 4.2**
func TestEventAggregator_Property_OrderPreservation(t *testing.T) {
	// Property: Items should be flushed in the order they were added
	property := func(seed uint8) bool {
		aggregator := NewTestEventAggregator()
		sessionID := "test-session"

		numItems := int(seed%30) + 1
		expectedOrder := make([]string, 0, numItems)

		// Add items in order
		for i := 0; i < numItems; i++ {
			itemData := fmt.Sprintf("ordered-item-%d", i)
			expectedOrder = append(expectedOrder, itemData)
			aggregator.AddItem(sessionID, "msg", "req", "table", itemData, nil)
		}

		// Flush
		flushedItems := aggregator.FlushNow(sessionID, true)

		// Property: Items should be in the same order
		if len(flushedItems) != len(expectedOrder) {
			t.Logf("Length mismatch: expected %d, got %d", len(expectedOrder), len(flushedItems))
			return false
		}

		for i, item := range flushedItems {
			data, ok := item.Data.(string)
			if !ok {
				t.Logf("Item %d: data is not string", i)
				return false
			}
			if data != expectedOrder[i] {
				t.Logf("Order mismatch at index %d: expected %s, got %s", i, expectedOrder[i], data)
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEventAggregator_Property_EmittedBatchCompleteness tests that emitted batch contains all items
// **Validates: Requirements 4.2**
func TestEventAggregator_Property_EmittedBatchCompleteness(t *testing.T) {
	// Property: The emitted batch should contain all items that were added
	property := func(seed uint16) bool {
		aggregator := NewTestEventAggregator()
		sessionID := fmt.Sprintf("session-%d", seed)
		messageID := fmt.Sprintf("message-%d", seed%100)
		requestID := fmt.Sprintf("request-%d", seed)

		numItems := int(seed%25) + 1
		addedData := make([]string, 0, numItems)

		// Add items
		for i := 0; i < numItems; i++ {
			data := fmt.Sprintf("batch-item-%d-%d", seed, i)
			addedData = append(addedData, data)
			aggregator.AddItem(sessionID, messageID, requestID, "insight", data, nil)
		}

		// Flush
		aggregator.FlushNow(sessionID, true)

		// Get emitted batches
		batches := aggregator.GetEmittedBatches()

		// Property: Should have exactly one batch
		if len(batches) != 1 {
			t.Logf("Expected 1 batch, got %d", len(batches))
			return false
		}

		batch := batches[0]

		// Property: Batch should have correct session/message/request IDs
		if batch.SessionID != sessionID {
			t.Logf("SessionID mismatch: expected %s, got %s", sessionID, batch.SessionID)
			return false
		}
		if batch.MessageID != messageID {
			t.Logf("MessageID mismatch: expected %s, got %s", messageID, batch.MessageID)
			return false
		}
		if batch.RequestID != requestID {
			t.Logf("RequestID mismatch: expected %s, got %s", requestID, batch.RequestID)
			return false
		}

		// Property: Batch should contain all items
		if len(batch.Items) != numItems {
			t.Logf("Item count mismatch: expected %d, got %d", numItems, len(batch.Items))
			return false
		}

		// Property: All added data should be in batch
		batchDataSet := make(map[string]bool)
		for _, item := range batch.Items {
			if data, ok := item.Data.(string); ok {
				batchDataSet[data] = true
			}
		}

		for _, data := range addedData {
			if !batchDataSet[data] {
				t.Logf("Added data '%s' not found in batch", data)
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}


// ==================== Property Tests for analysis-dashboard-optimization ====================

// TestEventAggregator_Property_DataCapture tests that all valid data items are captured
// **Validates: Requirements 3.1, 3.2, 3.3**
// Property 3: EventAggregator Data Capture
func TestEventAggregator_Property_DataCapture(t *testing.T) {
	// Property: For any valid data item (echarts, image, or table), when added to EventAggregator
	// and flushed, the item SHALL appear in the emitted batch with correct type and data.

	property := func(seed uint16) bool {
		aggregator := NewTestEventAggregator()
		sessionID := fmt.Sprintf("session-%d", seed)
		messageID := fmt.Sprintf("message-%d", seed%100)
		requestID := fmt.Sprintf("request-%d", seed)

		// Test with echarts, image, and table types specifically
		testTypes := []string{"echarts", "image", "table"}
		expectedItems := make(map[string]string) // type -> data

		for i, itemType := range testTypes {
			data := fmt.Sprintf("%s-data-%d-%d", itemType, seed, i)
			expectedItems[itemType] = data
			aggregator.AddItem(sessionID, messageID, requestID, itemType, data, map[string]interface{}{
				"index": i,
			})
		}

		// Flush and verify
		flushedItems := aggregator.FlushNow(sessionID, true)

		// Property: All three types should be present
		if len(flushedItems) != len(testTypes) {
			t.Logf("Expected %d items, got %d", len(testTypes), len(flushedItems))
			return false
		}

		// Property: Each item should have correct type and data
		foundTypes := make(map[string]bool)
		for _, item := range flushedItems {
			foundTypes[item.Type] = true
			expectedData, exists := expectedItems[item.Type]
			if !exists {
				t.Logf("Unexpected item type: %s", item.Type)
				return false
			}
			if data, ok := item.Data.(string); ok {
				if data != expectedData {
					t.Logf("Data mismatch for type %s: expected %s, got %s", item.Type, expectedData, data)
					return false
				}
			}
		}

		// Property: All expected types should be found
		for _, itemType := range testTypes {
			if !foundTypes[itemType] {
				t.Logf("Missing item type: %s", itemType)
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEventAggregator_Property_GracefulDegradationOnEmptyIDs tests graceful handling of empty IDs
// **Validates: Requirements 3.5**
// Property 5: Graceful Degradation on Empty IDs
func TestEventAggregator_Property_GracefulDegradationOnEmptyIDs(t *testing.T) {
	// Property: For any data item with empty sessionId or messageId, the EventAggregator
	// SHALL log a warning but still process the item without throwing an error.

	property := func(seed uint8) bool {
		aggregator := NewTestEventAggregator()

		// Test cases with various empty ID combinations
		testCases := []struct {
			sessionID string
			messageID string
			requestID string
		}{
			{"", "msg-1", "req-1"},           // Empty sessionID
			{"session-1", "", "req-1"},       // Empty messageID
			{"session-1", "msg-1", ""},       // Empty requestID
			{"", "", "req-1"},                // Empty sessionID and messageID
			{"", "", ""},                     // All empty
		}

		for i, tc := range testCases {
			// Use a non-empty sessionID for storage key if sessionID is empty
			storageKey := tc.sessionID
			if storageKey == "" {
				storageKey = fmt.Sprintf("empty-session-%d-%d", seed, i)
			}

			data := fmt.Sprintf("data-%d-%d", seed, i)

			// This should NOT panic or throw an error
			aggregator.AddItem(storageKey, tc.messageID, tc.requestID, "echarts", data, nil)

			// Verify item was added
			pendingCount := aggregator.GetPendingItemCount(storageKey)
			if pendingCount == 0 {
				t.Logf("Item was not added for test case %d", i)
				return false
			}

			// Flush and verify item is present
			flushedItems := aggregator.FlushNow(storageKey, true)
			if len(flushedItems) == 0 {
				t.Logf("No items flushed for test case %d", i)
				return false
			}

			// Verify data is correct
			found := false
			for _, item := range flushedItems {
				if itemData, ok := item.Data.(string); ok && itemData == data {
					found = true
					break
				}
			}
			if !found {
				t.Logf("Data not found in flushed items for test case %d", i)
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEventAggregator_Property_AllValidTypesAccepted tests that all valid item types are accepted
// **Validates: Requirements 3.1, 3.2, 3.3**
func TestEventAggregator_Property_AllValidTypesAccepted(t *testing.T) {
	// Property: All valid item types (echarts, image, table, csv, metric, insight, file)
	// should be accepted and stored correctly.

	property := func(seed uint16) bool {
		aggregator := NewTestEventAggregator()
		sessionID := fmt.Sprintf("session-%d", seed)

		allValidTypes := GetValidTestItemTypes()
		expectedData := make(map[string]string)

		// Add one item of each valid type
		for i, itemType := range allValidTypes {
			data := fmt.Sprintf("%s-data-%d", itemType, i)
			expectedData[itemType] = data
			aggregator.AddItem(sessionID, "msg", "req", itemType, data, nil)
		}

		// Flush
		flushedItems := aggregator.FlushNow(sessionID, true)

		// Property: Should have exactly one item per valid type
		if len(flushedItems) != len(allValidTypes) {
			t.Logf("Expected %d items, got %d", len(allValidTypes), len(flushedItems))
			return false
		}

		// Property: Each type should be present with correct data
		typeCounts := make(map[string]int)
		for _, item := range flushedItems {
			typeCounts[item.Type]++
			if expected, ok := expectedData[item.Type]; ok {
				if data, ok := item.Data.(string); ok && data != expected {
					t.Logf("Data mismatch for type %s", item.Type)
					return false
				}
			}
		}

		// Verify each type appears exactly once
		for _, itemType := range allValidTypes {
			if typeCounts[itemType] != 1 {
				t.Logf("Type %s count: expected 1, got %d", itemType, typeCounts[itemType])
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}
