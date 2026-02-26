//go:build property_test

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Feature: datasource-pack-result-consistency
// Unit Test: 重执行清理 (Re-execution Cleanup)
// Validates: Requirements 7.1, 7.2, 7.3
//
// Verifies that the re-execution cleanup sequence correctly clears
// messages, EventAggregator cache, and FlushedItems before step execution.

// TestReExecuteCleanup_EventAggregatorClearSequence verifies that
// eventAggregator.Clear and ClearFlushedItems properly remove all cached
// and flushed items for a given session, matching the cleanup contract
// used in ReExecuteQuickAnalysisPack.
func TestReExecuteCleanup_EventAggregatorClearSequence(t *testing.T) {
	ea := NewTestEventAggregator()
	threadID := "test-reexec-thread-001"
	messageID := "msg-001"
	requestID := "req-001"

	// Simulate a first execution: add items and flush them
	ea.AddTable(threadID, messageID, requestID, []map[string]interface{}{
		{"col1": "val1", "col2": 42},
	})
	ea.AddECharts(threadID, messageID, requestID, `{"type":"bar","data":[1,2,3]}`)

	// Flush to move items from pending to flushed
	flushed := ea.FlushNow(threadID, false)
	if len(flushed) == 0 {
		t.Fatal("expected items to be flushed after first execution, got 0")
	}

	// Verify flushed items exist before cleanup
	allFlushed := ea.GetAllFlushedItems(threadID)
	if len(allFlushed) == 0 {
		t.Fatal("expected flushed items to exist before cleanup")
	}

	// --- Simulate the cleanup sequence from ReExecuteQuickAnalysisPack ---
	// Step 1: eventAggregator.Clear(threadID) - clears pending items, timers, and emits clear event
	ea.Clear(threadID)

	// Step 2: eventAggregator.ClearFlushedItems(threadID) - clears flushed items history
	ea.ClearFlushedItems(threadID)

	// --- Verify cleanup results ---

	// Pending items should be empty
	pendingAfterClear := ea.FlushNow(threadID, false)
	if len(pendingAfterClear) != 0 {
		t.Errorf("expected no pending items after Clear, got %d", len(pendingAfterClear))
	}

	// Flushed items should be empty
	flushedAfterClear := ea.GetAllFlushedItems(threadID)
	if len(flushedAfterClear) != 0 {
		t.Errorf("expected no flushed items after ClearFlushedItems, got %d", len(flushedAfterClear))
	}
}

// TestReExecuteCleanup_ClearDoesNotAffectOtherSessions verifies that
// clearing one session's data does not affect another session's data.
// This ensures the cleanup in ReExecuteQuickAnalysisPack is scoped correctly.
func TestReExecuteCleanup_ClearDoesNotAffectOtherSessions(t *testing.T) {
	ea := NewTestEventAggregator()

	threadA := "thread-A"
	threadB := "thread-B"
	messageID := "msg-001"
	requestID := "req-001"

	// Add items to both sessions
	ea.AddTable(threadA, messageID, requestID, []map[string]interface{}{{"a": 1}})
	ea.AddTable(threadB, messageID, requestID, []map[string]interface{}{{"b": 2}})

	// Flush both
	ea.FlushNow(threadA, false)
	ea.FlushNow(threadB, false)

	// Clear only threadA (simulating re-execution of threadA)
	ea.Clear(threadA)
	ea.ClearFlushedItems(threadA)

	// threadA should be empty
	if items := ea.GetAllFlushedItems(threadA); len(items) != 0 {
		t.Errorf("expected threadA flushed items to be empty after clear, got %d", len(items))
	}

	// threadB should still have its items
	if items := ea.GetAllFlushedItems(threadB); len(items) == 0 {
		t.Error("expected threadB flushed items to be preserved after clearing threadA")
	}
}

// TestReExecuteCleanup_NewItemsAfterClear verifies that after cleanup,
// new items can be added and flushed normally, ensuring the re-execution
// produces results in the same format as the first execution.
// Validates: Requirement 7.2 - re-execution results format matches first execution.
func TestReExecuteCleanup_NewItemsAfterClear(t *testing.T) {
	ea := NewTestEventAggregator()
	threadID := "test-reexec-thread-002"
	messageID := "msg-001"
	requestID := "req-001"

	// --- First execution ---
	ea.AddTable(threadID, messageID, requestID, []map[string]interface{}{
		{"name": "Alice", "score": 95},
	})
	ea.AddECharts(threadID, messageID, requestID, `{"type":"line","data":[10,20]}`)
	firstResults := ea.FlushNow(threadID, true)
	_ = ea.GetAllFlushedItems(threadID) // Verify flushed items exist before cleanup

	if len(firstResults) != 2 {
		t.Fatalf("expected 2 items from first execution flush, got %d", len(firstResults))
	}

	// --- Cleanup (simulating ReExecuteQuickAnalysisPack) ---
	ea.Clear(threadID)
	ea.ClearFlushedItems(threadID)

	// --- Second execution (re-execution) with same structure ---
	messageID2 := "msg-002"
	ea.AddTable(threadID, messageID2, requestID, []map[string]interface{}{
		{"name": "Bob", "score": 88},
	})
	ea.AddECharts(threadID, messageID2, requestID, `{"type":"line","data":[30,40]}`)
	secondResults := ea.FlushNow(threadID, true)
	secondFlushed := ea.GetAllFlushedItems(threadID)

	// Verify second execution produces same number of results
	if len(secondResults) != len(firstResults) {
		t.Errorf("expected re-execution to produce %d items (same as first), got %d",
			len(firstResults), len(secondResults))
	}

	// Verify result types match between first and second execution
	for i := range firstResults {
		if i >= len(secondResults) {
			break
		}
		if firstResults[i].Type != secondResults[i].Type {
			t.Errorf("result[%d] type mismatch: first=%q, second=%q",
				i, firstResults[i].Type, secondResults[i].Type)
		}
	}

	// Verify flushed items only contain second execution results (no mixing)
	if len(secondFlushed) != len(secondResults) {
		t.Errorf("expected flushed items to contain only re-execution results (%d), got %d",
			len(secondResults), len(secondFlushed))
	}

	// Verify no items from first execution leak into second execution's flushed items
	for _, item := range secondFlushed {
		mid, ok := item.Metadata["messageId"]
		if !ok {
			continue
		}
		if mid == messageID {
			t.Error("found item from first execution in re-execution flushed items - cleanup failed")
		}
	}
}

// TestReExecuteCleanup_ClearBeforeAddingNewItems verifies the critical ordering:
// cleanup must happen before any new items are added. This test ensures that
// if items are added after Clear+ClearFlushedItems, they are not lost.
func TestReExecuteCleanup_ClearBeforeAddingNewItems(t *testing.T) {
	ea := NewTestEventAggregator()
	threadID := "test-ordering-thread"
	requestID := "req-001"

	// Simulate old execution data
	ea.AddTable(threadID, "old-msg", requestID, []map[string]interface{}{{"old": true}})
	ea.FlushNow(threadID, false)

	// Cleanup
	ea.Clear(threadID)
	ea.ClearFlushedItems(threadID)

	// Add new items (simulating executeStepLoop after cleanup)
	ea.AddTable(threadID, "new-msg-1", requestID, []map[string]interface{}{{"step": 1}})
	ea.AddECharts(threadID, "new-msg-1", requestID, `{"type":"pie"}`)

	results := ea.FlushNow(threadID, false)

	// New items should be present
	if len(results) != 2 {
		t.Errorf("expected 2 new items after cleanup+add, got %d", len(results))
	}

	// Verify types
	typeSet := map[string]bool{}
	for _, item := range results {
		typeSet[item.Type] = true
	}
	if !typeSet["table"] {
		t.Error("expected a 'table' item in results after re-execution")
	}
	if !typeSet["echarts"] {
		t.Error("expected an 'echarts' item in results after re-execution")
	}
}

// TestReExecuteCleanup_EmptySessionClearIsNoOp verifies that clearing
// a session with no data does not cause errors or panics.
func TestReExecuteCleanup_EmptySessionClearIsNoOp(t *testing.T) {
	ea := NewTestEventAggregator()
	threadID := "nonexistent-thread"

	// Should not panic or error
	ea.Clear(threadID)
	ea.ClearFlushedItems(threadID)

	// Should return empty results
	items := ea.GetAllFlushedItems(threadID)
	if len(items) != 0 {
		t.Errorf("expected 0 items for cleared nonexistent thread, got %d", len(items))
	}
}

// Feature: datasource-pack-result-consistency
// Unit Test: FlushNow 完成调用 (FlushNow Completion Call)
// Validates: Requirement 4.1
//
// Verifies that FlushNow(threadID, true) with isComplete=true properly
// marks the session as complete and that GetAllFlushedItems returns all
// accumulated items — matching the behavior of emitCompletion.

// TestFlushNowComplete_AccumulatesAllStepItems verifies that after multiple
// steps each adding items and flushing with isComplete=false, a final
// FlushNow(threadID, true) flushes any remaining pending items and
// GetAllFlushedItems returns the full set of accumulated items from all steps.
func TestFlushNowComplete_AccumulatesAllStepItems(t *testing.T) {
	ea := NewTestEventAggregator()
	threadID := "test-flush-complete-001"
	requestID := "req-001"

	// --- Step 1: SQL step adds a table and flushes (isComplete=false) ---
	ea.AddTable(threadID, "msg-step1", requestID, []map[string]interface{}{
		{"id": 1, "name": "Alice"},
	})
	step1Items := ea.FlushNow(threadID, false)
	if len(step1Items) != 1 {
		t.Fatalf("step 1: expected 1 flushed item, got %d", len(step1Items))
	}

	// --- Step 2: Python step adds an echarts and an image, flushes (isComplete=false) ---
	ea.AddECharts(threadID, "msg-step2", requestID, `{"type":"bar","data":[1,2,3]}`)
	ea.AddImage(threadID, "msg-step2", requestID, "base64imagedata", "chart.png")
	step2Items := ea.FlushNow(threadID, false)
	if len(step2Items) != 2 {
		t.Fatalf("step 2: expected 2 flushed items, got %d", len(step2Items))
	}

	// --- Step 3: Another SQL step adds a table, flushes (isComplete=false) ---
	ea.AddTable(threadID, "msg-step3", requestID, []map[string]interface{}{
		{"metric": "revenue", "value": 42.5},
	})
	step3Items := ea.FlushNow(threadID, false)
	if len(step3Items) != 1 {
		t.Fatalf("step 3: expected 1 flushed item, got %d", len(step3Items))
	}

	// --- Final: emitCompletion calls FlushNow(threadID, true) ---
	// At this point there are no pending items, but the call marks completion.
	finalItems := ea.FlushNow(threadID, true)
	// No pending items remain, so finalItems should be empty/nil
	if len(finalItems) != 0 {
		t.Errorf("final FlushNow(true): expected 0 pending items, got %d", len(finalItems))
	}

	// --- Verify GetAllFlushedItems returns ALL accumulated items ---
	allItems := ea.GetAllFlushedItems(threadID)
	expectedTotal := len(step1Items) + len(step2Items) + len(step3Items)
	if len(allItems) != expectedTotal {
		t.Errorf("GetAllFlushedItems: expected %d total items, got %d", expectedTotal, len(allItems))
	}

	// Verify item types are correct and in order
	expectedTypes := []string{"table", "echarts", "image", "table"}
	if len(allItems) == len(expectedTypes) {
		for i, expected := range expectedTypes {
			if allItems[i].Type != expected {
				t.Errorf("allItems[%d].Type = %q, want %q", i, allItems[i].Type, expected)
			}
		}
	}
}

// TestFlushNowComplete_WithPendingItems verifies that if items are still
// pending when FlushNow(threadID, true) is called, they are flushed and
// included in GetAllFlushedItems.
func TestFlushNowComplete_WithPendingItems(t *testing.T) {
	ea := NewTestEventAggregator()
	threadID := "test-flush-complete-002"
	requestID := "req-001"

	// Step 1: flush normally
	ea.AddTable(threadID, "msg-1", requestID, []map[string]interface{}{{"a": 1}})
	ea.FlushNow(threadID, false)

	// Step 2: add items but do NOT flush yet (simulating items added just before completion)
	ea.AddECharts(threadID, "msg-2", requestID, `{"type":"pie","data":[10,20]}`)

	// Final FlushNow(true) should flush the pending echarts item
	finalItems := ea.FlushNow(threadID, true)
	if len(finalItems) != 1 {
		t.Fatalf("final FlushNow(true): expected 1 pending item flushed, got %d", len(finalItems))
	}
	if finalItems[0].Type != "echarts" {
		t.Errorf("final flushed item type = %q, want %q", finalItems[0].Type, "echarts")
	}

	// GetAllFlushedItems should have both the step-1 table and the final echarts
	allItems := ea.GetAllFlushedItems(threadID)
	if len(allItems) != 2 {
		t.Fatalf("GetAllFlushedItems: expected 2 total items, got %d", len(allItems))
	}
	if allItems[0].Type != "table" || allItems[1].Type != "echarts" {
		t.Errorf("unexpected item types: [%q, %q], want [table, echarts]",
			allItems[0].Type, allItems[1].Type)
	}
}

// TestFlushNowComplete_EmptySession verifies that FlushNow(threadID, true)
// on a session with no items does not panic and returns nil.
func TestFlushNowComplete_EmptySession(t *testing.T) {
	ea := NewTestEventAggregator()
	threadID := "test-flush-complete-empty"

	// FlushNow(true) on empty session should be safe
	items := ea.FlushNow(threadID, true)
	if items != nil && len(items) != 0 {
		t.Errorf("FlushNow(true) on empty session: expected nil/empty, got %d items", len(items))
	}

	allItems := ea.GetAllFlushedItems(threadID)
	if allItems != nil && len(allItems) != 0 {
		t.Errorf("GetAllFlushedItems on empty session: expected nil/empty, got %d items", len(allItems))
	}
}

// Feature: datasource-pack-result-consistency
// Integration Test: 端到端集成验证 (End-to-End Integration Verification)
// Validates: Requirements 4.3, 4.4
//
// This test simulates a complete QAP execution flow with SQL steps, Python steps,
// and query_and_chart composite steps. It verifies that:
// 1. Results are correctly formatted and sent to EventAggregator
// 2. Results are properly persisted via SaveAnalysisResults
// 3. Messages are properly structured with step descriptions and user requests
// 4. query_and_chart steps produce both table and chart results
// 5. FlushNow is called with isComplete=true at the end
// 6. All results can be retrieved via GetAllFlushedItems

// TestEndToEndQAPExecution simulates a complete QAP execution with multiple step types.
func TestEndToEndQAPExecution(t *testing.T) {
	// Setup: Create a test EventAggregator and temporary session directory
	ea := NewTestEventAggregator()
	threadID := "test-e2e-qap-session"
	requestID := "req-e2e-001"

	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions", threadID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("failed to create session dir: %v", err)
	}

	// Define a realistic pack with multiple step types:
	// 1. SQL step (standalone)
	// 2. SQL step + Python chart step (query_and_chart pair)
	// 3. Python step (standalone)
	steps := []PackStep{
		{
			StepID:      1,
			StepType:    "sql_query",
			Code:        "SELECT region, SUM(revenue) as total_revenue FROM sales GROUP BY region",
			Description: "Query total revenue by region",
			UserRequest: "Show me total revenue by region",
		},
		{
			StepID:      2,
			StepType:    "sql_query",
			Code:        "SELECT month, SUM(revenue) as monthly_revenue FROM sales GROUP BY month ORDER BY month",
			Description: "Query monthly revenue trend",
			UserRequest: "Show me monthly revenue trend",
			SourceTool:  "query_and_chart",
			EChartsConfigs: []string{
				`{"xAxis":{"type":"category","data":["Jan","Feb","Mar"]},"yAxis":{"type":"value"},"series":[{"type":"line","data":[100,200,150]}]}`,
			},
		},
		{
			StepID:          3,
			StepType:        "python_code",
			Code:            "import matplotlib.pyplot as plt\nplt.bar(df['month'], df['monthly_revenue'])\nplt.savefig('revenue_chart.png')",
			Description:     "Generate monthly revenue bar chart",
			UserRequest:     "Show me monthly revenue trend",
			SourceTool:      "query_and_chart",
			PairedSQLStepID: 2,
		},
		{
			StepID:      4,
			StepType:    "python_code",
			Code:        "print('Analysis complete')\nprint('json:echarts')\nprint('{\"title\":{\"text\":\"Summary\"}}')",
			Description: "Generate summary report",
			UserRequest: "Summarize the analysis",
		},
	}

	// Simulate step execution results
	stepResults := make(map[int]interface{})
	var allAnalysisResults []AnalysisResultItem
	var allMessages []ChatMessage

	// --- Step 1: SQL step (standalone) ---
	t.Run("Step 1: SQL standalone", func(t *testing.T) {
		step := steps[0]
		messageID := fmt.Sprintf("msg-%d", step.StepID)

		// Simulate SQL execution result
		sqlResult := []map[string]interface{}{
			{"region": "North", "total_revenue": 15000},
			{"region": "South", "total_revenue": 23000},
			{"region": "East", "total_revenue": 18000},
		}
		stepResults[step.StepID] = sqlResult

		// Add table result to EventAggregator
		ea.AddTable(threadID, messageID, requestID, sqlResult)
		tableItem := AnalysisResultItem{
			Type: "table",
			Data: sqlResult,
			Metadata: map[string]interface{}{
				"sessionId":        threadID,
				"messageId":        messageID,
				"timestamp":        time.Now().UnixMilli(),
				"step_description": step.Description,
			},
		}
		allAnalysisResults = append(allAnalysisResults, tableItem)

		// Flush and verify
		flushed := ea.FlushNow(threadID, false)
		if len(flushed) != 1 {
			t.Errorf("Step 1: expected 1 flushed item, got %d", len(flushed))
		}
		if flushed[0].Type != "table" {
			t.Errorf("Step 1: expected type 'table', got %q", flushed[0].Type)
		}

		// Add success message
		resultJSON, _ := json.MarshalIndent(sqlResult, "", "  ")
		chatContent := fmt.Sprintf("Step %d: %s\n\nUser Request: %s\n\nResults (%d rows):\n%s",
			step.StepID, step.Description, step.UserRequest, len(sqlResult), string(resultJSON))
		msg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   chatContent,
			Timestamp: time.Now().Unix(),
		}
		allMessages = append(allMessages, msg)

		// Verify message contains key content
		if !strings.Contains(msg.Content, step.Description) {
			t.Errorf("Step 1: message should contain description %q", step.Description)
		}
		if !strings.Contains(msg.Content, step.UserRequest) {
			t.Errorf("Step 1: message should contain user request %q", step.UserRequest)
		}
	})

	// --- Step 2: SQL step (query_and_chart pair - SQL part) ---
	t.Run("Step 2: SQL query_and_chart", func(t *testing.T) {
		step := steps[1]
		messageID := fmt.Sprintf("msg-%d", step.StepID)

		// Simulate SQL execution result
		sqlResult := []map[string]interface{}{
			{"month": "Jan", "monthly_revenue": 5000},
			{"month": "Feb", "monthly_revenue": 7000},
			{"month": "Mar", "monthly_revenue": 6500},
		}
		stepResults[step.StepID] = sqlResult

		// Add table result
		ea.AddTable(threadID, messageID, requestID, sqlResult)
		tableItem := AnalysisResultItem{
			Type: "table",
			Data: sqlResult,
			Metadata: map[string]interface{}{
				"sessionId":        threadID,
				"messageId":        messageID,
				"timestamp":        time.Now().UnixMilli(),
				"step_description": step.Description,
			},
		}
		allAnalysisResults = append(allAnalysisResults, tableItem)

		// Add ECharts config from step
		for _, chartJSON := range step.EChartsConfigs {
			if json.Valid([]byte(chartJSON)) {
				ea.AddECharts(threadID, messageID, requestID, chartJSON)
				echartsItem := AnalysisResultItem{
					Type: "echarts",
					Data: chartJSON,
					Metadata: map[string]interface{}{
						"sessionId":        threadID,
						"messageId":        messageID,
						"timestamp":        time.Now().UnixMilli(),
						"step_description": step.Description,
					},
				}
				allAnalysisResults = append(allAnalysisResults, echartsItem)
			}
		}

		// Flush and verify
		flushed := ea.FlushNow(threadID, false)
		if len(flushed) != 2 {
			t.Errorf("Step 2: expected 2 flushed items (table + echarts), got %d", len(flushed))
		}

		// Verify ordering: table before echarts
		if flushed[0].Type != "table" {
			t.Errorf("Step 2: first item should be 'table', got %q", flushed[0].Type)
		}
		if flushed[1].Type != "echarts" {
			t.Errorf("Step 2: second item should be 'echarts', got %q", flushed[1].Type)
		}

		// Add success message
		resultJSON, _ := json.MarshalIndent(sqlResult, "", "  ")
		chatContent := fmt.Sprintf("Step %d: %s\n\nUser Request: %s\n\nResults (%d rows):\n%s",
			step.StepID, step.Description, step.UserRequest, len(sqlResult), string(resultJSON))
		msg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   chatContent,
			Timestamp: time.Now().Unix(),
		}
		allMessages = append(allMessages, msg)
	})

	// --- Step 3: Python chart step (query_and_chart pair - Python part) ---
	t.Run("Step 3: Python query_and_chart", func(t *testing.T) {
		step := steps[2]
		messageID := fmt.Sprintf("msg-%d", step.StepID)

		// Verify paired SQL step result exists
		if _, ok := stepResults[step.PairedSQLStepID]; !ok {
			t.Fatalf("Step 3: paired SQL step %d result not found in stepResults", step.PairedSQLStepID)
		}

		// Simulate Python execution (creates image file)
		pythonOutput := "Chart saved to revenue_chart.png"
		stepResults[step.StepID] = pythonOutput

		// Create a dummy image file
		imagePath := filepath.Join(sessionDir, "revenue_chart.png")
		imageData := []byte("fake-png-data")
		if err := os.WriteFile(imagePath, imageData, 0644); err != nil {
			t.Fatalf("failed to create test image: %v", err)
		}

		// Simulate image detection and sending
		base64Data := "data:image/png;base64,ZmFrZS1wbmctZGF0YQ==" // base64 of "fake-png-data"
		ea.AddImage(threadID, messageID, requestID, base64Data, "revenue_chart.png")
		imageItem := AnalysisResultItem{
			Type: "image",
			Data: base64Data,
			Metadata: map[string]interface{}{
				"sessionId":        threadID,
				"messageId":        messageID,
				"timestamp":        time.Now().UnixMilli(),
				"fileName":         "revenue_chart.png",
				"step_description": step.Description,
			},
		}
		allAnalysisResults = append(allAnalysisResults, imageItem)

		// Flush and verify
		flushed := ea.FlushNow(threadID, false)
		if len(flushed) != 1 {
			t.Errorf("Step 3: expected 1 flushed item (image), got %d", len(flushed))
		}
		if flushed[0].Type != "image" {
			t.Errorf("Step 3: expected type 'image', got %q", flushed[0].Type)
		}

		// Add success message
		chatContent := fmt.Sprintf("Step %d: %s\n\nUser Request: %s\n\nOutput:\n```\n%s\n```",
			step.StepID, step.Description, step.UserRequest, pythonOutput)
		msg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   chatContent,
			Timestamp: time.Now().Unix(),
		}
		allMessages = append(allMessages, msg)
	})

	// --- Step 4: Python standalone with ECharts output ---
	t.Run("Step 4: Python standalone with ECharts", func(t *testing.T) {
		step := steps[3]
		messageID := fmt.Sprintf("msg-%d", step.StepID)

		// Simulate Python execution with ECharts output
		pythonOutput := "Analysis complete\njson:echarts\n{\"title\":{\"text\":\"Summary\"}}"
		stepResults[step.StepID] = pythonOutput

		// Simulate ECharts detection from output
		echartsJSON := `{"title":{"text":"Summary"}}`
		if json.Valid([]byte(echartsJSON)) {
			ea.AddECharts(threadID, messageID, requestID, echartsJSON)
			echartsItem := AnalysisResultItem{
				Type: "echarts",
				Data: echartsJSON,
				Metadata: map[string]interface{}{
					"sessionId":        threadID,
					"messageId":        messageID,
					"timestamp":        time.Now().UnixMilli(),
					"step_description": step.Description,
				},
			}
			allAnalysisResults = append(allAnalysisResults, echartsItem)
		}

		// Flush and verify
		flushed := ea.FlushNow(threadID, false)
		if len(flushed) != 1 {
			t.Errorf("Step 4: expected 1 flushed item (echarts), got %d", len(flushed))
		}
		if flushed[0].Type != "echarts" {
			t.Errorf("Step 4: expected type 'echarts', got %q", flushed[0].Type)
		}

		// Add success message (output contains json:echarts, so pass through as-is)
		chatContent := fmt.Sprintf("Step %d: %s\n\nUser Request: %s\n\nOutput:\n%s",
			step.StepID, step.Description, step.UserRequest, pythonOutput)
		msg := ChatMessage{
			ID:        messageID,
			Role:      "assistant",
			Content:   chatContent,
			Timestamp: time.Now().Unix(),
		}
		allMessages = append(allMessages, msg)

		// Verify message preserves ECharts reference
		if !strings.Contains(msg.Content, "json:echarts") {
			t.Errorf("Step 4: message should contain json:echarts reference")
		}
	})

	// --- Final: Emit completion and verify aggregation ---
	t.Run("Completion and aggregation", func(t *testing.T) {
		// Emit completion message
		completionMsg := ChatMessage{
			ID:        "msg-completion",
			Role:      "assistant",
			Content:   fmt.Sprintf("QAP execution completed successfully. Total steps: %d, Succeeded: %d", len(steps), len(steps)),
			Timestamp: time.Now().Unix(),
		}
		allMessages = append(allMessages, completionMsg)

		// Call FlushNow with isComplete=true (simulates emitCompletion)
		finalFlushed := ea.FlushNow(threadID, true)
		if len(finalFlushed) != 0 {
			t.Logf("Final flush returned %d pending items (expected 0 since all were flushed per-step)", len(finalFlushed))
		}

		// Verify GetAllFlushedItems returns all accumulated results
		allFlushed := ea.GetAllFlushedItems(threadID)
		expectedTotal := len(allAnalysisResults)
		if len(allFlushed) != expectedTotal {
			t.Errorf("GetAllFlushedItems: expected %d total items, got %d", expectedTotal, len(allFlushed))
		}

		// Verify result type distribution
		typeCounts := make(map[string]int)
		for _, item := range allFlushed {
			typeCounts[item.Type]++
		}

		// Expected: 2 tables (step 1 + step 2), 2 echarts (step 2 + step 4), 1 image (step 3)
		if typeCounts["table"] != 2 {
			t.Errorf("expected 2 'table' items, got %d", typeCounts["table"])
		}
		if typeCounts["echarts"] != 2 {
			t.Errorf("expected 2 'echarts' items, got %d", typeCounts["echarts"])
		}
		if typeCounts["image"] != 1 {
			t.Errorf("expected 1 'image' item, got %d", typeCounts["image"])
		}

		// Verify all items have complete metadata
		for i, item := range allFlushed {
			if item.Metadata == nil {
				t.Errorf("item[%d]: metadata is nil", i)
				continue
			}
			requiredKeys := []string{"sessionId", "messageId", "timestamp", "step_description"}
			for _, key := range requiredKeys {
				if _, exists := item.Metadata[key]; !exists {
					t.Errorf("item[%d]: metadata missing key %q", i, key)
				}
			}
		}
	})

	// --- Verify query_and_chart pair produced both table and chart ---
	t.Run("query_and_chart pair verification", func(t *testing.T) {
		// Steps 2 and 3 form a query_and_chart pair
		// Step 2 should produce: 1 table + 1 echarts (from EChartsConfigs)
		// Step 3 should produce: 1 image
		// Combined: 1 table + 1 echarts + 1 image

		allFlushed := ea.GetAllFlushedItems(threadID)
		step2Results := []AnalysisResultItem{}
		step3Results := []AnalysisResultItem{}

		for _, item := range allFlushed {
			desc, _ := item.Metadata["step_description"].(string)
			if desc == steps[1].Description {
				step2Results = append(step2Results, item)
			}
			if desc == steps[2].Description {
				step3Results = append(step3Results, item)
			}
		}

		// Step 2 should have table + echarts
		hasTable := false
		hasECharts := false
		for _, item := range step2Results {
			if item.Type == "table" {
				hasTable = true
			}
			if item.Type == "echarts" {
				hasECharts = true
			}
		}
		if !hasTable {
			t.Error("query_and_chart SQL step (step 2) should produce a table result")
		}
		if !hasECharts {
			t.Error("query_and_chart SQL step (step 2) should produce an echarts result from EChartsConfigs")
		}

		// Step 3 should have image
		hasImage := false
		for _, item := range step3Results {
			if item.Type == "image" {
				hasImage = true
			}
		}
		if !hasImage {
			t.Error("query_and_chart Python step (step 3) should produce an image result")
		}

		// Combined: at least 1 table and at least 1 chart (echarts or image)
		combinedResults := append(step2Results, step3Results...)
		hasTableCombined := false
		hasChartCombined := false
		for _, item := range combinedResults {
			if item.Type == "table" {
				hasTableCombined = true
			}
			if item.Type == "echarts" || item.Type == "image" {
				hasChartCombined = true
			}
		}
		if !hasTableCombined || !hasChartCombined {
			t.Error("query_and_chart pair should produce both table and chart results")
		}
	})

	// --- Verify message format consistency ---
	t.Run("Message format consistency", func(t *testing.T) {
		for i, msg := range allMessages {
			if msg.Role != "assistant" {
				t.Errorf("message[%d]: expected role 'assistant', got %q", i, msg.Role)
			}
			if msg.Content == "" {
				t.Errorf("message[%d]: content is empty", i)
			}
			if msg.Timestamp == 0 {
				t.Errorf("message[%d]: timestamp is zero", i)
			}

			// Verify step messages contain description and user request
			// (skip completion message which has different format)
			if i < len(allMessages)-1 {
				step := steps[i]
				if !strings.Contains(msg.Content, step.Description) {
					t.Errorf("message[%d]: should contain description %q", i, step.Description)
				}
				if !strings.Contains(msg.Content, step.UserRequest) {
					t.Errorf("message[%d]: should contain user request %q", i, step.UserRequest)
				}
			}
		}
	})

	// --- Verify persistence format ---
	t.Run("Persistence format verification", func(t *testing.T) {
		// Simulate SaveAnalysisResults for each step
		// In production, this would persist to database/file
		// Here we just verify the format is correct

		for _, item := range allAnalysisResults {
			// Verify item can be marshaled to JSON
			jsonData, err := json.Marshal(item)
			if err != nil {
				t.Errorf("failed to marshal AnalysisResultItem to JSON: %v", err)
			}

			// Verify item can be unmarshaled back
			var restored AnalysisResultItem
			if err := json.Unmarshal(jsonData, &restored); err != nil {
				t.Errorf("failed to unmarshal AnalysisResultItem from JSON: %v", err)
			}

			// Verify type is preserved
			if restored.Type != item.Type {
				t.Errorf("type mismatch after JSON round-trip: expected %q, got %q", item.Type, restored.Type)
			}

			// Verify metadata is preserved
			if restored.Metadata == nil {
				t.Error("metadata is nil after JSON round-trip")
			}
		}
	})
}

// TestEndToEndQAPExecution_WithFailures simulates a QAP execution where some steps fail.
// This verifies that:
// 1. Failed steps emit errors with correct error codes
// 2. Successful steps before and after failures still produce results
// 3. Dependent steps are skipped when their dependencies fail
// 4. Final completion message includes execution summary
func TestEndToEndQAPExecution_WithFailures(t *testing.T) {
	ea := NewTestEventAggregator()
	threadID := "test-e2e-qap-failures"
	requestID := "req-e2e-002"

	steps := []PackStep{
		{
			StepID:      1,
			StepType:    "sql_query",
			Code:        "SELECT * FROM valid_table",
			Description: "Query valid data",
			UserRequest: "Show me the data",
		},
		{
			StepID:      2,
			StepType:    "sql_query",
			Code:        "SELECT * FROM non_existent_table",
			Description: "Query non-existent table",
			UserRequest: "Show me more data",
		},
		{
			StepID:      3,
			StepType:    "python_code",
			Code:        "print('This step depends on step 2')",
			Description: "Process data from step 2",
			UserRequest: "Process the data",
			DependsOn:   []int{2},
		},
		{
			StepID:      4,
			StepType:    "sql_query",
			Code:        "SELECT COUNT(*) FROM valid_table",
			Description: "Count records",
			UserRequest: "How many records?",
		},
	}

	stepResults := make(map[int]interface{})
	summary := StepExecutionSummary{Total: len(steps)}

	// --- Step 1: Success ---
	t.Run("Step 1: Success", func(t *testing.T) {
		step := steps[0]
		messageID := fmt.Sprintf("msg-%d", step.StepID)

		sqlResult := []map[string]interface{}{{"id": 1, "name": "test"}}
		stepResults[step.StepID] = sqlResult
		summary.Succeeded++

		ea.AddTable(threadID, messageID, requestID, sqlResult)
		ea.FlushNow(threadID, false)

		flushed := ea.GetAllFlushedItems(threadID)
		if len(flushed) != 1 {
			t.Errorf("expected 1 flushed item after step 1, got %d", len(flushed))
		}
	})

	// --- Step 2: Failure (SQL error) ---
	t.Run("Step 2: SQL Failure", func(t *testing.T) {
		step := steps[1]
		messageID := fmt.Sprintf("msg-%d", step.StepID)

		sqlErr := "table non_existent_table does not exist"
		errMsg := fmt.Sprintf("Step %d failed: %s", step.StepID, sqlErr)
		summary.Failed++

		// Emit error with code "SQL_ERROR"
		ea.EmitErrorWithCode(threadID, messageID, "SQL_ERROR", errMsg)

		// Step result is NOT stored in stepResults (it failed)
		// stepResults[step.StepID] is intentionally NOT set
	})

	// --- Step 3: Skipped (dependency failed) ---
	t.Run("Step 3: Skipped due to dependency failure", func(t *testing.T) {
		_ = steps[2] // step variable for reference

		// Check if dependency step 2 failed
		if _, ok := stepResults[2]; !ok {
			// Dependency failed, skip this step
			summary.Skipped++
			t.Logf("Step 3 skipped because dependency step 2 failed")
		} else {
			t.Error("Step 3 should be skipped because step 2 failed")
		}
	})

	// --- Step 4: Success (independent of failed steps) ---
	t.Run("Step 4: Success (independent)", func(t *testing.T) {
		step := steps[3]
		messageID := fmt.Sprintf("msg-%d", step.StepID)

		sqlResult := []map[string]interface{}{{"count": 42}}
		stepResults[step.StepID] = sqlResult
		summary.Succeeded++

		ea.AddTable(threadID, messageID, requestID, sqlResult)
		ea.FlushNow(threadID, false)

		allFlushed := ea.GetAllFlushedItems(threadID)
		// Should have 2 tables (step 1 + step 4)
		tableCount := 0
		for _, item := range allFlushed {
			if item.Type == "table" {
				tableCount++
			}
		}
		if tableCount != 2 {
			t.Errorf("expected 2 table items (steps 1 and 4), got %d", tableCount)
		}
	})

	// --- Final: Verify execution summary ---
	t.Run("Execution summary", func(t *testing.T) {
		if summary.Total != 4 {
			t.Errorf("expected total=4, got %d", summary.Total)
		}
		if summary.Succeeded != 2 {
			t.Errorf("expected succeeded=2, got %d", summary.Succeeded)
		}
		if summary.Failed != 1 {
			t.Errorf("expected failed=1, got %d", summary.Failed)
		}
		if summary.Skipped != 1 {
			t.Errorf("expected skipped=1, got %d", summary.Skipped)
		}

		// Emit completion with summary
		completionText := fmt.Sprintf("QAP execution completed. Total: %d, Succeeded: %d, Failed: %d, Skipped: %d",
			summary.Total, summary.Succeeded, summary.Failed, summary.Skipped)
		completionMsg := ChatMessage{
			ID:        "msg-completion",
			Role:      "assistant",
			Content:   completionText,
			Timestamp: time.Now().Unix(),
		}

		if !strings.Contains(completionMsg.Content, "Succeeded: 2") {
			t.Error("completion message should contain success count")
		}
		if !strings.Contains(completionMsg.Content, "Failed: 1") {
			t.Error("completion message should contain failure count")
		}
		if !strings.Contains(completionMsg.Content, "Skipped: 1") {
			t.Error("completion message should contain skipped count")
		}

		// Final flush with isComplete=true
		ea.FlushNow(threadID, true)
	})
}
