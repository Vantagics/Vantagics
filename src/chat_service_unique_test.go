package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"os"
)

// Helper function to create a test ChatService instance
func setupTestChatService(t *testing.T) (*ChatService, string) {
	tmpDir, err := os.MkdirTemp("", "chat-service-unique-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	service := NewChatService(filepath.Join(tmpDir, "sessions"))
	return service, tmpDir
}

// Helper function to create a thread
func createThread(service *ChatService, dataSourceID, title string) (ChatThread, error) {
	return service.CreateThread(dataSourceID, title)
}

func TestChatService_UniqueTitleGeneration(t *testing.T) {
	service, tmpDir := setupTestChatService(t)
	defer os.RemoveAll(tmpDir)

	dsID1 := "datasource-1"
	dsID2 := "datasource-2"
	baseTitle := "My Chat"

	// 1. Create first chat
	thread1, err := createThread(service, dsID1, baseTitle)
	if err != nil {
		t.Fatalf("Failed to create thread 1: %v", err)
	}
	if thread1.Title != baseTitle {
		t.Errorf("Expected title '%s', got '%s'", baseTitle, thread1.Title)
	}

	// 2. Create second chat for same data source, expect increment
	thread2, err := createThread(service, dsID1, baseTitle)
	if err != nil {
		t.Fatalf("Failed to create thread 2: %v", err)
	}
	expectedTitle2 := "My Chat (1)"
	if thread2.Title != expectedTitle2 {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle2, thread2.Title)
	}

	// 3. Create third chat for same data source, expect further increment
	thread3, err := createThread(service, dsID1, baseTitle)
	if err != nil {
		t.Fatalf("Failed to create thread 3: %v", err)
	}
	expectedTitle3 := "My Chat (2)"
	if thread3.Title != expectedTitle3 {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle3, thread3.Title)
	}

	// 4. Create chat for a different data source with the base title, expect no increment
	thread4, err := createThread(service, dsID2, baseTitle)
	if err != nil {
		t.Fatalf("Failed to create thread 4: %v", err)
	}
	if thread4.Title != baseTitle {
		t.Errorf("Expected title '%s' for different DS, got '%s'", baseTitle, thread4.Title)
	}

	// 5. Create chat for different data source with an incremented title, expect no increment
	thread5, err := createThread(service, dsID2, "My Chat (1)")
	if err != nil {
		t.Fatalf("Failed to create thread 5: %v", err)
	}
	expectedTitle5 := "My Chat (1)"
	if thread5.Title != expectedTitle5 {
		t.Errorf("Expected title '%s' for different DS, got '%s'", expectedTitle5, thread5.Title)
	}

	// 6. Test a different base name
	thread6, err := createThread(service, dsID1, "Another Chat")
	if err != nil {
		t.Fatalf("Failed to create thread 6: %v", err)
	}
	expectedTitle6 := "Another Chat"
	if thread6.Title != expectedTitle6 {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle6, thread6.Title)
	}

	// 7. Test an increment of the different base name
	thread7, err := createThread(service, dsID1, "Another Chat")
	if err != nil {
		t.Fatalf("Failed to create thread 7: %v", err)
	}
	expectedTitle7 := "Another Chat (1)"
	if thread7.Title != expectedTitle7 {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle7, thread7.Title)
	}
}

func TestChatService_UpdateThreadTitle_Uniqueness(t *testing.T) {
	service, tmpDir := setupTestChatService(t)
	defer os.RemoveAll(tmpDir)

	dsID1 := "datasource-update-test"
	
	// Create some initial threads
	thread1, _ := createThread(service, dsID1, "Original Title")
	thread2, _ := createThread(service, dsID1, "Target Title")
	createThread(service, dsID1, "Another Title")

	// Test 1: Update to a title that already exists
	newTitle := "Another Title"
	updatedTitle, err := service.UpdateThreadTitle(thread2.ID, newTitle)
	if err != nil {
		t.Fatalf("Failed to update thread title: %v", err)
	}
	expectedUpdatedTitle := "Another Title (1)"
	if updatedTitle != expectedUpdatedTitle {
		t.Errorf("Expected updated title '%s', got '%s'", expectedUpdatedTitle, updatedTitle)
	}

	// Verify the thread in storage
	threads, _ := service.LoadThreads()
	found := false
	for _, thread := range threads {
		if thread.ID == thread2.ID {
			if thread.Title != expectedUpdatedTitle {
				t.Errorf("Thread in storage has incorrect title. Expected '%s', got '%s'", expectedUpdatedTitle, thread.Title)
			}
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Target thread not found after update")
	}

	// Test 2: Update to a title that is currently unique
	newUniqueTitle := "Truly Unique Title"
	updatedTitle, err = service.UpdateThreadTitle(thread1.ID, newUniqueTitle)
	if err != nil {
		t.Fatalf("Failed to update thread title: %v", err)
	}
	if updatedTitle != newUniqueTitle {
		t.Errorf("Expected updated title '%s', got '%s'", newUniqueTitle, updatedTitle)
	}

	// Test 3: Update a thread to its own current title (should not change or increment)
	updatedTitle, err = service.UpdateThreadTitle(thread1.ID, newUniqueTitle)
	if err != nil {
		t.Fatalf("Failed to update thread title to itself: %v", err)
	}
	if updatedTitle != newUniqueTitle {
		t.Errorf("Expected title to remain '%s', got '%s'", newUniqueTitle, updatedTitle)
	}

	// Test 4: Update a thread to a title that exists but belongs to itself (should not increment)
	// First, let's make a thread with "Self Title"
	threadSelf, _ := createThread(service, dsID1, "Self Title")
	// Then, update it to "Self Title"
	updatedTitle, err = service.UpdateThreadTitle(threadSelf.ID, "Self Title")
	if err != nil {
		t.Fatalf("Failed to update thread title to its own existing title: %v", err)
	}
	if updatedTitle != "Self Title" {
		t.Errorf("Expected title to remain 'Self Title', got '%s'", updatedTitle)
	}
}

func TestChatService_GetThreadsByDataSource(t *testing.T) {
	service, tmpDir := setupTestChatService(t)
	defer os.RemoveAll(tmpDir)

	dsID1 := "ds-a"
	dsID2 := "ds-b"

	// Create threads for different data sources
	createThread(service, dsID1, "Chat A1")
	createThread(service, dsID1, "Chat A2")
	createThread(service, dsID2, "Chat B1")

	threadsDS1, err := service.GetThreadsByDataSource(dsID1)
	if err != nil {
		t.Fatalf("GetThreadsByDataSource for %s failed: %v", dsID1, err)
	}
	if len(threadsDS1) != 2 {
		t.Errorf("Expected 2 threads for %s, got %d", dsID1, len(threadsDS1))
	}

	threadsDS2, err := service.GetThreadsByDataSource(dsID2)
	if err != nil {
		t.Fatalf("GetThreadsByDataSource for %s failed: %v", dsID2, err)
	}
	if len(threadsDS2) != 1 {
		t.Errorf("Expected 1 thread for %s, got %d", dsID2, len(threadsDS2))
	}

	threadsDS3, err := service.GetThreadsByDataSource("non-existent-ds")
	if err != nil {
		t.Fatalf("GetThreadsByDataSource for non-existent-ds failed: %v", err)
	}
	if len(threadsDS3) != 0 {
		t.Errorf("Expected 0 threads for non-existent-ds, got %d", len(threadsDS3))
	}
}

func TestChartData_UnmarshalJSON_BackwardCompatibility(t *testing.T) {
	// Test Case 1: Old format (flat structure)
	oldJSON := `{"type": "echarts", "data": "{}"}`
	var chartData1 ChartData
	if err := json.Unmarshal([]byte(oldJSON), &chartData1); err != nil {
		t.Fatalf("Failed to unmarshal old format: %v", err)
	}
	if len(chartData1.Charts) != 1 {
		t.Errorf("Expected 1 chart from old format, got %d", len(chartData1.Charts))
	}
	if len(chartData1.Charts) > 0 {
		if chartData1.Charts[0].Type != "echarts" {
			t.Errorf("Expected type 'echarts', got '%s'", chartData1.Charts[0].Type)
		}
	}

	// Test Case 2: New format (array structure)
	newJSON := `{"charts": [{"type": "image", "data": "base64..."}]}`
	var chartData2 ChartData
	if err := json.Unmarshal([]byte(newJSON), &chartData2); err != nil {
		t.Fatalf("Failed to unmarshal new format: %v", err)
	}
	if len(chartData2.Charts) != 1 {
		t.Errorf("Expected 1 chart from new format, got %d", len(chartData2.Charts))
	}
	if len(chartData2.Charts) > 0 {
		if chartData2.Charts[0].Type != "image" {
			t.Errorf("Expected type 'image', got '%s'", chartData2.Charts[0].Type)
		}
	}

	// Test Case 3: Empty JSON
	emptyJSON := `{}`
	var chartData3 ChartData
	if err := json.Unmarshal([]byte(emptyJSON), &chartData3); err != nil {
		t.Fatalf("Failed to unmarshal empty JSON: %v", err)
	}
	if len(chartData3.Charts) != 0 {
		t.Errorf("Expected 0 charts from empty JSON, got %d", len(chartData3.Charts))
	}
}
