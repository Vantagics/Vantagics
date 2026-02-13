package main

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"testing/quick"
	"time"
)

// Feature: quick-analysis-dashboard-display, Property 2: Step result metadata contains step_description with save/retrieve consistency
// **Validates: Requirements 5.1, 5.4**

// generateRandomStepDescription generates a random step_description string.
// It may return an empty string to test the empty-description case.
func generateRandomStepDescription(r *rand.Rand) string {
	if r.Intn(5) == 0 {
		return "" // ~20% chance of empty description
	}
	return generateRandomString(r, 60)
}

// generateAnalysisResultsWithStepDescription creates random AnalysisResultItems
// where each item's metadata includes a step_description field.
func generateAnalysisResultsWithStepDescription(r *rand.Rand, stepDesc string) []AnalysisResultItem {
	n := r.Intn(5) + 1 // 1-5 items
	items := make([]AnalysisResultItem, n)
	for i := range items {
		itemType := randomAnalysisResultType(r)
		var data interface{}
		switch itemType {
		case "table":
			data = randomTableData(r)
		case "echarts":
			data = randomEChartsData(r)
		case "image":
			data = randomImageData(r)
		default:
			data = randomTableData(r)
		}
		meta := map[string]interface{}{
			"sessionId":        fmt.Sprintf("session_%d", r.Intn(10000)),
			"messageId":        fmt.Sprintf("msg_%d", r.Intn(10000)),
			"timestamp":        time.Now().UnixMilli(),
			"step_description": stepDesc,
		}
		items[i] = AnalysisResultItem{
			ID:       fmt.Sprintf("item_%d_%d", i, r.Intn(10000)),
			Type:     itemType,
			Data:     data,
			Metadata: meta,
		}
	}
	return items
}

// TestProperty2_StepDescriptionSaveRetrieveConsistency verifies that for any
// random step_description string, when saved as part of AnalysisResultItem metadata
// via SaveAnalysisResults and then retrieved via GetMessageAnalysisData, the
// step_description value is identical to the original.
// **Validates: Requirements 5.1, 5.4**
func TestProperty2_StepDescriptionSaveRetrieveConsistency(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		tmpDir := t.TempDir()
		chatService := NewChatService(filepath.Join(tmpDir, "sessions"))

		// Create a thread and add a message
		thread, err := chatService.CreateThread("ds1", "test-thread")
		if err != nil {
			t.Logf("seed=%d: failed to create thread: %v", seed, err)
			return false
		}

		msgID := fmt.Sprintf("msg_%d", r.Intn(100000))
		msg := ChatMessage{
			ID:        msgID,
			Role:      "assistant",
			Content:   "✅ 步骤 1 (test):\n\n> 📋 分析请求：test\n\n```json:table\n[{\"col\":1}]\n```",
			Timestamp: time.Now().Unix(),
		}
		if err := chatService.AddMessage(thread.ID, msg); err != nil {
			t.Logf("seed=%d: failed to add message: %v", seed, err)
			return false
		}

		// Generate a random step_description and analysis results with it
		stepDesc := generateRandomStepDescription(r)
		savedResults := generateAnalysisResultsWithStepDescription(r, stepDesc)

		// Save analysis results
		if err := chatService.SaveAnalysisResults(thread.ID, msgID, savedResults); err != nil {
			t.Logf("seed=%d: failed to save analysis results: %v", seed, err)
			return false
		}

		// Retrieve via GetMessageAnalysisData
		analysisData, err := chatService.GetMessageAnalysisData(thread.ID, msgID)
		if err != nil {
			t.Logf("seed=%d: GetMessageAnalysisData returned error: %v", seed, err)
			return false
		}

		rawItems, ok := analysisData["analysisResults"]
		if !ok || rawItems == nil {
			t.Logf("seed=%d: analysisResults not found in returned data", seed)
			return false
		}

		retrievedItems, ok := rawItems.([]AnalysisResultItem)
		if !ok {
			t.Logf("seed=%d: analysisResults is not []AnalysisResultItem", seed)
			return false
		}

		// Property: count must match
		if len(retrievedItems) != len(savedResults) {
			t.Logf("seed=%d: count mismatch: saved=%d, retrieved=%d",
				seed, len(savedResults), len(retrievedItems))
			return false
		}

		// Property: step_description in metadata must be preserved for every item
		for i, retrieved := range retrievedItems {
			retrievedDesc, ok := retrieved.Metadata["step_description"]
			if !ok {
				t.Logf("seed=%d: item %d missing step_description in metadata", seed, i)
				return false
			}

			// After JSON round-trip, the value comes back as a string
			retrievedDescStr, ok := retrievedDesc.(string)
			if !ok {
				t.Logf("seed=%d: item %d step_description is not a string, got %T", seed, i, retrievedDesc)
				return false
			}

			if retrievedDescStr != stepDesc {
				t.Logf("seed=%d: item %d step_description mismatch: saved=%q, retrieved=%q",
					seed, i, stepDesc, retrievedDescStr)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 2 (step_description save/retrieve consistency) failed: %v", err)
	}
}
