package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestChatService_Persistence(t *testing.T) {
	// Setup temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "rapidbi-test-chat")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create ChatService with mock storage path
	service := &ChatService{
		storagePath: filepath.Join(tmpDir, "chat_history.json"),
	}

	// Test data
	msg1 := ChatMessage{
		ID:        "msg1",
		Role:      "user",
		Content:   "Hello",
		Timestamp: time.Now().Unix(),
	}
	msg2 := ChatMessage{
		ID:        "msg2",
		Role:      "assistant",
		Content:   "Hi there!",
		Timestamp: time.Now().Unix(),
	}

	thread := ChatThread{
		ID:        "thread1",
		Title:     "Test Thread",
		CreatedAt: time.Now().Unix(),
		Messages:  []ChatMessage{msg1, msg2},
	}

	// Test SaveHistory (implied save via SaveThread)
	// We'll simulate saving a thread
	threads := []ChatThread{thread}
	err = service.SaveThreads(threads)
	if err != nil {
		t.Fatalf("SaveThreads failed: %v", err)
	}

	// Verify file existence
	if _, err := os.Stat(service.storagePath); os.IsNotExist(err) {
		t.Fatal("chat_history.json was not created")
	}

	// Test LoadHistory (LoadThreads)
	loadedThreads, err := service.LoadThreads()
	if err != nil {
		t.Fatalf("LoadThreads failed: %v", err)
	}

	if len(loadedThreads) != 1 {
		t.Fatalf("Expected 1 thread, got %d", len(loadedThreads))
	}

	if loadedThreads[0].ID != thread.ID {
		t.Errorf("Expected thread ID %s, got %s", thread.ID, loadedThreads[0].ID)
	}

	if len(loadedThreads[0].Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(loadedThreads[0].Messages))
	}

	// Test DeleteThread
	err = service.DeleteThread("thread1")
	if err != nil {
		t.Fatalf("DeleteThread failed: %v", err)
	}

	loadedThreads, err = service.LoadThreads()
	if err != nil {
		t.Fatalf("LoadThreads failed after delete: %v", err)
	}

	if len(loadedThreads) != 0 {
		t.Errorf("Expected 0 threads after delete, got %d", len(loadedThreads))
	}
}

func TestChatService_ClearHistory(t *testing.T) {
	// Setup temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "rapidbi-test-chat-clear")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	service := &ChatService{
		storagePath: filepath.Join(tmpDir, "chat_history.json"),
	}

	// Create a dummy file with some content
	dummyData := []ChatThread{{ID: "1", Title: "Test"}}
	data, _ := json.Marshal(dummyData)
	_ = os.WriteFile(service.storagePath, data, 0644)

	// Test ClearHistory
	err = service.ClearHistory()
	if err != nil {
		t.Fatalf("ClearHistory failed: %v", err)
	}

	// Verify file is empty list or gone/re-initialized
	loadedThreads, err := service.LoadThreads()
	if err != nil {
		t.Fatalf("LoadThreads failed after clear: %v", err)
	}

	if len(loadedThreads) != 0 {
		t.Errorf("Expected 0 threads after clear, got %d", len(loadedThreads))
	}
}

func TestChatService_LoadThreads_MissingFile(t *testing.T) {
	service := NewChatService("non-existent-file.json")
	threads, err := service.LoadThreads()
	if err != nil {
		t.Fatalf("LoadThreads should not fail for missing file: %v", err)
	}
	if len(threads) != 0 {
		t.Errorf("Expected 0 threads for missing file, got %d", len(threads))
	}
}

func TestChatService_LoadThreads_MalformedFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-malformed")
	defer os.RemoveAll(tmpDir)
	
	path := filepath.Join(tmpDir, "malformed.json")
	os.WriteFile(path, []byte("invalid json"), 0644)
	
	service := NewChatService(path)
	_, err := service.LoadThreads()
	if err == nil {
		t.Error("Expected error for malformed JSON file")
	}
}

func TestChatService_SaveThreads_MkdirError(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-mkdir-err")
	defer os.RemoveAll(tmpDir)
	
	// Create a file where the directory should be
	path := filepath.Join(tmpDir, "blocked-dir")
	os.WriteFile(path, []byte("i am a file"), 0644)
	
	// Try to save a file inside that "file"
	service := NewChatService(filepath.Join(path, "history.json"))
	err := service.SaveThreads([]ChatThread{{ID: "1"}})
	if err == nil {
		t.Error("Expected error when MkdirAll fails")
	}
}

func TestChatService_DeleteThread_MissingFile(t *testing.T) {
	service := NewChatService("missing-file-for-delete.json")
	err := service.DeleteThread("any-id")
	if err != nil {
		t.Errorf("DeleteThread should not fail if file is missing: %v", err)
	}
}
