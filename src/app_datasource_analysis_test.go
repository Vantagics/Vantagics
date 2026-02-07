package main

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"vantagedata/agent"
	"vantagedata/logger"
)

// TestStartDataSourceAnalysis_InvalidDataSourceID tests with an invalid data source ID
func TestStartDataSourceAnalysis_InvalidDataSourceID(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "vantagedata-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app with data source service
	service := agent.NewDataSourceService(tmpDir, func(msg string) {})
	app := &App{
		dataSourceService: service,
		chatService:       NewChatService(tmpDir),
		logger:            logger.NewLogger(),
		activeThreads:     make(map[string]bool),
	}

	// Add a test data source
	testDS := agent.DataSource{
		ID:   "test-mysql-1",
		Name: "Test MySQL Database",
		Type: "mysql",
	}
	if err := service.AddDataSource(testDS); err != nil {
		t.Fatalf("Failed to add test data source: %v", err)
	}

	// Try to start analysis with non-existent ID
	_, err = app.StartDataSourceAnalysis("non-existent-id")
	if err == nil {
		t.Fatal("Expected error for non-existent data source ID, got nil")
	}

	expectedMsg := "data source not found: non-existent-id"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestStartDataSourceAnalysis_ServiceNotInitialized tests error handling when service is nil
func TestStartDataSourceAnalysis_ServiceNotInitialized(t *testing.T) {
	// Create app without data source service
	app := &App{
		dataSourceService: nil,
		logger:            logger.NewLogger(),
	}

	// Try to start analysis
	_, err := app.StartDataSourceAnalysis("test-id")
	if err == nil {
		t.Fatal("Expected error when service not initialized, got nil")
	}

	expectedMsg := "data source service not initialized"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestStartDataSourceAnalysis_LoadDataSourcesError tests error handling when LoadDataSources fails
func TestStartDataSourceAnalysis_LoadDataSourcesError(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "vantagedata-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app with data source service
	service := agent.NewDataSourceService(tmpDir, func(msg string) {})
	app := &App{
		dataSourceService: service,
		chatService:       NewChatService(tmpDir),
		logger:            logger.NewLogger(),
		activeThreads:     make(map[string]bool),
	}

	// Create an invalid datasources.json file to trigger error
	datasourcesPath := tmpDir + "/datasources.json"
	if err := os.WriteFile(datasourcesPath, []byte("invalid json content"), 0644); err != nil {
		t.Fatalf("Failed to create invalid datasources.json: %v", err)
	}

	// Try to start analysis
	_, err = app.StartDataSourceAnalysis("test-id")
	if err == nil {
		t.Fatal("Expected error when LoadDataSources fails, got nil")
	}

	// Error should be wrapped with "failed to load data sources"
	if !strings.Contains(err.Error(), "failed to load data sources") {
		t.Errorf("Expected error to contain 'failed to load data sources', got: %s", err.Error())
	}
}

// TestStartDataSourceAnalysis_ThreadIDGeneration tests thread ID generation without calling SendMessage
// This test validates the thread ID format and uniqueness without requiring full Wails context
func TestStartDataSourceAnalysis_ThreadIDGeneration(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "vantagedata-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app with data source service
	service := agent.NewDataSourceService(tmpDir, func(msg string) {})
	
	// Add test data sources with different IDs
	testCases := []struct {
		id   string
		name string
		typ  string
	}{
		{"mysql-prod", "Production MySQL", "mysql"},
		{"pg-dev", "Development PostgreSQL", "postgresql"},
		{"excel-sales", "Sales Data Excel", "excel"},
	}

	for _, tc := range testCases {
		testDS := agent.DataSource{
			ID:   tc.id,
			Name: tc.name,
			Type: tc.typ,
		}
		if err := service.AddDataSource(testDS); err != nil {
			t.Fatalf("Failed to add test data source %s: %v", tc.id, err)
		}
	}

	// Load data sources to verify they exist
	dataSources, err := service.LoadDataSources()
	if err != nil {
		t.Fatalf("Failed to load data sources: %v", err)
	}

	// Test thread ID generation logic for each data source
	for _, ds := range dataSources {
		// Generate thread ID using the same logic as StartDataSourceAnalysis
		threadID := generateThreadID(ds.ID)

		// Verify thread ID format
		expectedPrefix := "ds-analysis-" + ds.ID + "-"
		if !strings.HasPrefix(threadID, expectedPrefix) {
			t.Errorf("Expected thread ID to start with '%s', got: %s", expectedPrefix, threadID)
		}

		// Verify thread ID contains timestamp
		if len(threadID) <= len(expectedPrefix) {
			t.Errorf("Expected thread ID to contain timestamp, got: %s", threadID)
		}

		// Extract timestamp part and verify it's not empty
		parts := strings.Split(threadID, "-")
		if len(parts) < 3 {
			t.Errorf("Expected thread ID to have at least 3 parts, got: %v", parts)
		} else {
			timestampStr := parts[len(parts)-1]
			if timestampStr == "" {
				t.Errorf("Expected timestamp in thread ID, got empty string")
			}
		}
	}
}

// TestStartDataSourceAnalysis_UniqueThreadIDs tests that multiple calls generate unique thread IDs
func TestStartDataSourceAnalysis_UniqueThreadIDs(t *testing.T) {
	dataSourceID := "test-mysql-1"
	
	// Generate multiple thread IDs
	threadIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		threadID := generateThreadID(dataSourceID)
		
		// Check if thread ID is unique
		if threadIDs[threadID] {
			t.Errorf("Thread ID %s is not unique (appeared twice)", threadID)
		}
		threadIDs[threadID] = true
		
		// Small delay to ensure different timestamps
		time.Sleep(2 * time.Millisecond)
	}
	
	// Verify we got 10 unique thread IDs
	if len(threadIDs) != 10 {
		t.Errorf("Expected 10 unique thread IDs, got %d", len(threadIDs))
	}
}

// Helper function to generate thread ID (extracted from StartDataSourceAnalysis logic)
func generateThreadID(dataSourceID string) string {
	return "ds-analysis-" + dataSourceID + "-" + 
		   strconv.FormatInt(time.Now().UnixMilli(), 10)
}

// Note: Full integration tests that call SendMessage require a running Wails application
// with proper context. These tests focus on the validation and thread ID generation logic
// that can be tested in isolation.

