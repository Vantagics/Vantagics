package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDashboardDataSerialization(t *testing.T) {
	// This test expects DashboardData, Metric, and Insight to be defined in app.go
	// Since we haven't defined them yet, this should fail to compile or run.
	
	data := DashboardData{
		Metrics: []Metric{
			{Title: "Total Sales", Value: "$12,345", Change: "+15%"},
			{Title: "Active Users", Value: "1,234", Change: "+5%"},
		},
		Insights: []Insight{
			{Text: "Sales increased by 15% this week!", Icon: "trending-up"},
			{Text: "User engagement is at an all-time high.", Icon: "star"},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal DashboardData: %v", err)
	}

	var decoded DashboardData
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal DashboardData: %v", err)
	}

	if len(decoded.Metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(decoded.Metrics))
	}

	if decoded.Metrics[0].Title != "Total Sales" {
		t.Errorf("Expected 'Total Sales', got '%s'", decoded.Metrics[0].Title)
	}
}

func TestGetDashboardData(t *testing.T) {
	app := NewApp()
	data := app.GetDashboardData()

	if len(data.Metrics) == 0 {
		t.Error("Expected metrics to be populated")
	}

	if len(data.Insights) == 0 {
		t.Error("Expected insights to be populated")
	}
}

func TestSendMessage(t *testing.T) {
	// Setup temp dir for config
	tmpDir, err := os.MkdirTemp("", "rapidbi-test-config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	app := NewApp()
	app.ctx = context.Background()

	// We need to override getStorageDir but since it's a method we can't easily.
	// However, we can just ensure no config file exists in the default location or 
	// better, we can mock the behavior if we had an interface.
	// For now, let's just use a config that we know has no API key.
	
	// Actually, let's just test that it returns the expected message when API key is empty
	llm := NewLLMService(Config{LLMProvider: "OpenAI", APIKey: ""})
	resp, err := llm.Chat(app.ctx, "Hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	
	if resp != "Please set your API key in settings." {
		t.Errorf("Unexpected response: %s", resp)
	}
}

func TestApp_GetConfig_Default(t *testing.T) {
	app := NewApp()
	config, err := app.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if config.LLMProvider == "" {
		t.Error("Expected default LLMProvider to be set")
	}
}

func TestApp_Greet(t *testing.T) {
	app := NewApp()
	resp := app.Greet("Test")
	if resp != "Hello Test, It's show time!" {
		t.Errorf("Unexpected greeting: %s", resp)
	}
}

func TestApp_getConfigPath(t *testing.T) {
	app := NewApp()
	app.storageDir = "/tmp/test"
	path, err := app.getConfigPath()
	if err != nil {
		t.Fatalf("getConfigPath failed: %v", err)
	}
	if !contains(path, "config.json") {
		t.Errorf("Expected path to contain config.json, got %s", path)
	}
}

func TestApp_TestLLMConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "ok"}},
			},
		})
	}))
	defer server.Close()

	app := NewApp()
	app.ctx = context.Background()
	
	config := Config{
		LLMProvider: "OpenAI",
		APIKey:      "test",
		BaseURL:     server.URL,
	}
	
	result := app.TestLLMConnection(config)
	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}
}

func TestApp_TestLLMConnection_Fail(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	
	config := Config{
		LLMProvider: "OpenAI",
		APIKey:      "", // Missing key
	}
	
	result := app.TestLLMConnection(config)
	if result.Success {
		t.Error("Expected failure for missing API key")
	}
}

func TestApp_GetStorageDir_Default(t *testing.T) {
	app := NewApp()
	app.storageDir = ""
	path, err := app.getStorageDir()
	if err != nil {
		t.Fatalf("getStorageDir failed: %v", err)
	}
	if path == "" {
		t.Error("Expected default storage path")
	}
}

func TestApp_SaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "rapidbi-test-storage-config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	app := NewApp()
	app.storageDir = tmpDir

	config := Config{
		LLMProvider: "TestProvider",
		APIKey:      "test-key",
		DarkMode:    true,
	}

	err = app.SaveConfig(config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loadedConfig, err := app.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if loadedConfig.LLMProvider != config.LLMProvider {
		t.Errorf("Expected provider %s, got %s", config.LLMProvider, loadedConfig.LLMProvider)
	}
	if loadedConfig.DarkMode != config.DarkMode {
		t.Error("DarkMode mismatch")
	}
}

func TestApp_SendMessage_Success(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-send")
	defer os.RemoveAll(tmpDir)

	app := NewApp()
	app.ctx = context.Background()
	app.storageDir = tmpDir

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "App says hi"}},
			},
		})
	}))
	defer server.Close()

	config := Config{
		LLMProvider: "OpenAI",
		APIKey:      "test-key",
		BaseURL:     server.URL,
	}
	app.SaveConfig(config)

	resp, err := app.SendMessage("Hello")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if resp != "App says hi" {
		t.Errorf("Expected 'App says hi', got '%s'", resp)
	}
}

func TestLLMServiceChat_UnsupportedProvider(t *testing.T) {
	service := NewLLMService(Config{LLMProvider: "Unknown", APIKey: "dummy"})
	resp, err := service.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp != "Unsupported LLM provider." {
		t.Errorf("Expected 'Unsupported LLM provider.', got '%s'", resp)
	}
}

func TestApp_Startup(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-startup")
	defer os.RemoveAll(tmpDir)

	app := NewApp()
	app.storageDir = tmpDir
	app.startup(context.Background())

	if app.chatService == nil {
		t.Error("Expected chatService to be initialized after startup")
	}
}

func TestApp_ChatMethods_NoService(t *testing.T) {
	app := NewApp()
	// app.chatService is nil

	_, err := app.GetChatHistory()
	if err == nil {
		t.Error("Expected error for nil chatService in GetChatHistory")
	}

	err = app.SaveChatHistory(nil)
	if err == nil {
		t.Error("Expected error for nil chatService in SaveChatHistory")
	}

	err = app.DeleteThread("1")
	if err == nil {
		t.Error("Expected error for nil chatService in DeleteThread")
	}

	err = app.ClearHistory()
	if err == nil {
		t.Error("Expected error for nil chatService in ClearHistory")
	}
}

func TestApp_GetConfig_ReadError(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-read-err")
	defer os.RemoveAll(tmpDir)
	
	app := NewApp()
	app.storageDir = tmpDir
	
	// Create a directory where the config file should be
	os.MkdirAll(filepath.Join(tmpDir, "config.json"), 0755)
	
	_, err := app.GetConfig()
	if err == nil {
		t.Error("Expected error when reading directory as file")
	}
}

func TestApp_GetConfig_UnmarshalError(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-unmarshal-err")
	defer os.RemoveAll(tmpDir)
	
	app := NewApp()
	app.storageDir = tmpDir
	os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("invalid json"), 0644)
	
	_, err := app.GetConfig()
	if err == nil {
		t.Error("Expected error for malformed config JSON")
	}
}

func TestApp_SaveConfig_WriteError(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-write-err")
	defer os.RemoveAll(tmpDir)
	
	app := NewApp()
	app.storageDir = tmpDir
	
	// Create a directory where the config file should be to cause write error
	os.MkdirAll(filepath.Join(tmpDir, "config.json"), 0755)
	
	err := app.SaveConfig(Config{})
	if err == nil {
		t.Error("Expected error when writing to a directory path")
	}
}
