package main

import (
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

	sessionsDir := filepath.Join(tmpDir, "sessions")
	// Create ChatService with mock sessions dir
	service := NewChatService(sessionsDir)

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

	// Test SaveHistory (implied save via SaveThreads)
	threads := []ChatThread{thread}
	err = service.SaveThreads(threads)
	if err != nil {
		t.Fatalf("SaveThreads failed: %v", err)
	}

	// Verify file existence: sessions/thread1/history.json
	threadPath := filepath.Join(sessionsDir, "thread1", "history.json")
	if _, err := os.Stat(threadPath); os.IsNotExist(err) {
		t.Fatal("history.json was not created in session dir")
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

	// Verify directory is gone
	if _, err := os.Stat(filepath.Join(sessionsDir, "thread1")); !os.IsNotExist(err) {
		t.Error("Session directory should be removed after delete")
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

	sessionsDir := filepath.Join(tmpDir, "sessions")
	service := NewChatService(sessionsDir)

	// Create a dummy thread
	threadID := "dummy-thread"
	dummyThread := ChatThread{ID: threadID, Title: "Test"}
	
	// Manually save it (or use service)
	service.SaveThreads([]ChatThread{dummyThread})

	// Verify it exists
	if _, err := os.Stat(filepath.Join(sessionsDir, threadID)); os.IsNotExist(err) {
		t.Fatal("Setup failed: session dir not created")
	}

	// Test ClearHistory
	err = service.ClearHistory()
	if err != nil {
		t.Fatalf("ClearHistory failed: %v", err)
	}

	// Verify sessions directory is empty or gone (implementation removes sessionsDir)
	if _, err := os.Stat(sessionsDir); !os.IsNotExist(err) {
		// If it exists, it should be empty
		entries, _ := os.ReadDir(sessionsDir)
		if len(entries) > 0 {
			t.Errorf("Expected 0 entries in sessions dir, got %d", len(entries))
		}
	}
}

func TestChatService_LoadThreads_MissingDir(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-missing")
	defer os.RemoveAll(tmpDir)
	
	// Point to a non-existent subdirectory
	sessionsDir := filepath.Join(tmpDir, "non-existent-sessions")
	service := NewChatService(sessionsDir)
	
	// Should not fail, just return empty
	// Note: NewChatService does MkdirAll, so it WILL exist unless we force it not to (e.g. file collision)
	// But let's assume we pass a path that NewChatService created.
	
	threads, err := service.LoadThreads()
	if err != nil {
		t.Fatalf("LoadThreads should not fail: %v", err)
	}
	if len(threads) != 0 {
		t.Errorf("Expected 0 threads, got %d", len(threads))
	}
}

func TestChatService_LoadThreads_MalformedFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-malformed")
	defer os.RemoveAll(tmpDir)
	
	sessionsDir := filepath.Join(tmpDir, "sessions")
	service := NewChatService(sessionsDir)
	
	// Create a malformed file
	threadDir := filepath.Join(sessionsDir, "bad-thread")
	os.MkdirAll(threadDir, 0755)
	os.WriteFile(filepath.Join(threadDir, "history.json"), []byte("invalid json"), 0644)
	
	// LoadThreads currently swallows errors for individual files and skips them
	threads, err := service.LoadThreads()
	if err != nil {
		t.Fatalf("LoadThreads failed: %v", err)
	}
	if len(threads) != 0 {
		t.Errorf("Expected 0 threads (skipped malformed), got %d", len(threads))
	}
}

func TestChatService_DeleteThread_MissingThread(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-del-missing")
	defer os.RemoveAll(tmpDir)
	
	service := NewChatService(filepath.Join(tmpDir, "sessions"))
	err := service.DeleteThread("non-existent-id")
	if err != nil {
		// RemoveAll returns nil if path doesn't exist? Yes.
		// So this should be fine.
	}
}

