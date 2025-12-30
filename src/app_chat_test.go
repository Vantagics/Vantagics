package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestApp_ChatManagement(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "rapidbi-test-app")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize App with ChatService pointing to temp dir
	chatPath := filepath.Join(tmpDir, "chat_history.json")
	app := &App{
		ctx: context.Background(),
		chatService: NewChatService(chatPath),
	}
	
	// Test DeleteThread (should succeed even if empty)
	err = app.DeleteThread("non-existent")
	if err != nil {
		t.Errorf("DeleteThread failed: %v", err)
	}

	// Test ClearHistory
	err = app.ClearHistory()
	if err != nil {
		t.Errorf("ClearHistory failed: %v", err)
	}
	
	// Test GetChatHistory
	history, err := app.GetChatHistory()
	if err != nil {
		t.Errorf("GetChatHistory failed: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d", len(history))
	}
	
	// Test SaveChatHistory
	dummy := []ChatThread{{ID: "1"}}
	err = app.SaveChatHistory(dummy)
	if err != nil {
		t.Errorf("SaveChatHistory failed: %v", err)
	}
	
	history, err = app.GetChatHistory()
	if err != nil || len(history) != 1 {
		t.Errorf("GetChatHistory failed after save or length mismatch")
	}
}