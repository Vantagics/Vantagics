package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestCheckComponentHasData_Metrics tests the CheckComponentHasData method for metrics component
func TestCheckComponentHasData_Metrics(t *testing.T) {
	// Create temporary directory for test data
	tempDir := t.TempDir()
	
	// Create DataService
	fileService := NewFileService(nil, tempDir)
	dataService := NewDataService(nil, tempDir, fileService)
	
	t.Run("no data sources", func(t *testing.T) {
		hasData, err := dataService.CheckComponentHasData("metrics", "metrics-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasData {
			t.Error("expected hasData to be false when no data sources exist")
		}
	})
	
	t.Run("empty data sources array", func(t *testing.T) {
		// Create empty datasources.json
		metadataPath := filepath.Join(tempDir, "datasources.json")
		err := os.WriteFile(metadataPath, []byte("[]"), 0644)
		if err != nil {
			t.Fatalf("failed to create datasources.json: %v", err)
		}
		
		hasData, err := dataService.CheckComponentHasData("metrics", "metrics-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasData {
			t.Error("expected hasData to be false when data sources array is empty")
		}
	})
	
	t.Run("with data sources", func(t *testing.T) {
		// Create datasources.json with data
		metadataPath := filepath.Join(tempDir, "datasources.json")
		dataSources := []map[string]interface{}{
			{
				"id":   "test-1",
				"name": "Test Data Source",
				"type": "csv",
			},
		}
		data, _ := json.Marshal(dataSources)
		err := os.WriteFile(metadataPath, data, 0644)
		if err != nil {
			t.Fatalf("failed to create datasources.json: %v", err)
		}
		
		hasData, err := dataService.CheckComponentHasData("metrics", "metrics-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasData {
			t.Error("expected hasData to be true when data sources exist")
		}
	})
}

// TestCheckComponentHasData_Table tests the CheckComponentHasData method for table component
func TestCheckComponentHasData_Table(t *testing.T) {
	tempDir := t.TempDir()
	fileService := NewFileService(nil, tempDir)
	dataService := NewDataService(nil, tempDir, fileService)
	
	t.Run("no data sources", func(t *testing.T) {
		hasData, err := dataService.CheckComponentHasData("table", "table-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasData {
			t.Error("expected hasData to be false when no data sources exist")
		}
	})
	
	t.Run("with data sources", func(t *testing.T) {
		metadataPath := filepath.Join(tempDir, "datasources.json")
		dataSources := []map[string]interface{}{
			{
				"id":   "test-1",
				"name": "Test Data Source",
				"type": "excel",
			},
		}
		data, _ := json.Marshal(dataSources)
		err := os.WriteFile(metadataPath, data, 0644)
		if err != nil {
			t.Fatalf("failed to create datasources.json: %v", err)
		}
		
		hasData, err := dataService.CheckComponentHasData("table", "table-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasData {
			t.Error("expected hasData to be true when data sources exist")
		}
	})
}

// TestCheckComponentHasData_Image tests the CheckComponentHasData method for image component
func TestCheckComponentHasData_Image(t *testing.T) {
	tempDir := t.TempDir()
	fileService := NewFileService(nil, tempDir)
	dataService := NewDataService(nil, tempDir, fileService)
	
	t.Run("no images", func(t *testing.T) {
		hasData, err := dataService.CheckComponentHasData("image", "image-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasData {
			t.Error("expected hasData to be false when no images exist")
		}
	})
	
	t.Run("with images in images directory", func(t *testing.T) {
		// Create images directory with a file
		imagesDir := filepath.Join(tempDir, "images")
		err := os.MkdirAll(imagesDir, 0755)
		if err != nil {
			t.Fatalf("failed to create images directory: %v", err)
		}
		
		imagePath := filepath.Join(imagesDir, "test.png")
		err = os.WriteFile(imagePath, []byte("fake image data"), 0644)
		if err != nil {
			t.Fatalf("failed to create image file: %v", err)
		}
		
		hasData, err := dataService.CheckComponentHasData("image", "image-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasData {
			t.Error("expected hasData to be true when images exist")
		}
	})
	
	t.Run("with images in charts directory", func(t *testing.T) {
		// Clean up previous test
		tempDir2 := t.TempDir()
		fileService2 := NewFileService(nil, tempDir2)
		dataService2 := NewDataService(nil, tempDir2, fileService2)
		
		// Create charts directory with a file
		chartsDir := filepath.Join(tempDir2, "charts")
		err := os.MkdirAll(chartsDir, 0755)
		if err != nil {
			t.Fatalf("failed to create charts directory: %v", err)
		}
		
		chartPath := filepath.Join(chartsDir, "chart.png")
		err = os.WriteFile(chartPath, []byte("fake chart data"), 0644)
		if err != nil {
			t.Fatalf("failed to create chart file: %v", err)
		}
		
		hasData, err := dataService2.CheckComponentHasData("image", "image-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasData {
			t.Error("expected hasData to be true when charts exist")
		}
	})
}

// TestCheckComponentHasData_Insights tests the CheckComponentHasData method for insights component
func TestCheckComponentHasData_Insights(t *testing.T) {
	tempDir := t.TempDir()
	fileService := NewFileService(nil, tempDir)
	dataService := NewDataService(nil, tempDir, fileService)
	
	t.Run("no insights", func(t *testing.T) {
		hasData, err := dataService.CheckComponentHasData("insights", "insights-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasData {
			t.Error("expected hasData to be false when no insights exist")
		}
	})
	
	t.Run("empty insights array", func(t *testing.T) {
		insightsPath := filepath.Join(tempDir, "insights.json")
		err := os.WriteFile(insightsPath, []byte("[]"), 0644)
		if err != nil {
			t.Fatalf("failed to create insights.json: %v", err)
		}
		
		hasData, err := dataService.CheckComponentHasData("insights", "insights-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasData {
			t.Error("expected hasData to be false when insights array is empty")
		}
	})
	
	t.Run("with insights", func(t *testing.T) {
		insightsPath := filepath.Join(tempDir, "insights.json")
		insights := []map[string]interface{}{
			{
				"id":      "insight-1",
				"title":   "Test Insight",
				"content": "This is a test insight",
			},
		}
		data, _ := json.Marshal(insights)
		err := os.WriteFile(insightsPath, data, 0644)
		if err != nil {
			t.Fatalf("failed to create insights.json: %v", err)
		}
		
		hasData, err := dataService.CheckComponentHasData("insights", "insights-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasData {
			t.Error("expected hasData to be true when insights exist")
		}
	})
}

// TestCheckComponentHasData_FileDownload tests the CheckComponentHasData method for file_download component
func TestCheckComponentHasData_FileDownload(t *testing.T) {
	tempDir := t.TempDir()
	fileService := NewFileService(nil, tempDir)
	dataService := NewDataService(nil, tempDir, fileService)
	
	t.Run("no files", func(t *testing.T) {
		hasData, err := dataService.CheckComponentHasData("file_download", "file_download-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasData {
			t.Error("expected hasData to be false when no files exist")
		}
	})
	
	t.Run("with files in all_files category", func(t *testing.T) {
		// Create files directory with a file
		filesDir := filepath.Join(tempDir, "files")
		err := os.MkdirAll(filesDir, 0755)
		if err != nil {
			t.Fatalf("failed to create files directory: %v", err)
		}
		
		filePath := filepath.Join(filesDir, "test.txt")
		err = os.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		
		hasData, err := dataService.CheckComponentHasData("file_download", "file_download-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasData {
			t.Error("expected hasData to be true when files exist")
		}
	})
	
	t.Run("with files in user_request_related category", func(t *testing.T) {
		// Clean up previous test
		tempDir2 := t.TempDir()
		fileService2 := NewFileService(nil, tempDir2)
		dataService2 := NewDataService(nil, tempDir2, fileService2)
		
		// Create user_requests directory with a file
		userRequestsDir := filepath.Join(tempDir2, "user_requests")
		err := os.MkdirAll(userRequestsDir, 0755)
		if err != nil {
			t.Fatalf("failed to create user_requests directory: %v", err)
		}
		
		filePath := filepath.Join(userRequestsDir, "request.pdf")
		err = os.WriteFile(filePath, []byte("pdf content"), 0644)
		if err != nil {
			t.Fatalf("failed to create request file: %v", err)
		}
		
		hasData, err := dataService2.CheckComponentHasData("file_download", "file_download-0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasData {
			t.Error("expected hasData to be true when user request files exist")
		}
	})
}

// TestCheckComponentHasData_UnsupportedType tests error handling for unsupported component types
func TestCheckComponentHasData_UnsupportedType(t *testing.T) {
	tempDir := t.TempDir()
	fileService := NewFileService(nil, tempDir)
	dataService := NewDataService(nil, tempDir, fileService)
	
	_, err := dataService.CheckComponentHasData("unsupported", "unsupported-0")
	if err == nil {
		t.Error("expected error for unsupported component type")
	}
	
	expectedError := "unsupported component type: unsupported"
	if err.Error() != expectedError {
		t.Errorf("expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestBatchCheckHasData tests the BatchCheckHasData method
func TestBatchCheckHasData(t *testing.T) {
	tempDir := t.TempDir()
	fileService := NewFileService(nil, tempDir)
	dataService := NewDataService(nil, tempDir, fileService)
	
	t.Run("empty batch", func(t *testing.T) {
		components := make(map[string]string)
		results, err := dataService.BatchCheckHasData(components)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected empty results, got %d results", len(results))
		}
	})
	
	t.Run("mixed components with no data", func(t *testing.T) {
		components := map[string]string{
			"metrics-0":       "metrics",
			"table-0":         "table",
			"image-0":         "image",
			"insights-0":      "insights",
			"file_download-0": "file_download",
		}
		
		results, err := dataService.BatchCheckHasData(components)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if len(results) != 5 {
			t.Errorf("expected 5 results, got %d", len(results))
		}
		
		for instanceID, hasData := range results {
			if hasData {
				t.Errorf("expected %s to have no data", instanceID)
			}
		}
	})
	
	t.Run("mixed components with some data", func(t *testing.T) {
		// Create data for metrics and table
		metadataPath := filepath.Join(tempDir, "datasources.json")
		dataSources := []map[string]interface{}{
			{
				"id":   "test-1",
				"name": "Test Data Source",
				"type": "csv",
			},
		}
		data, _ := json.Marshal(dataSources)
		err := os.WriteFile(metadataPath, data, 0644)
		if err != nil {
			t.Fatalf("failed to create datasources.json: %v", err)
		}
		
		// Create data for file_download
		filesDir := filepath.Join(tempDir, "files")
		err = os.MkdirAll(filesDir, 0755)
		if err != nil {
			t.Fatalf("failed to create files directory: %v", err)
		}
		filePath := filepath.Join(filesDir, "test.txt")
		err = os.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		
		components := map[string]string{
			"metrics-0":       "metrics",
			"table-0":         "table",
			"image-0":         "image",
			"insights-0":      "insights",
			"file_download-0": "file_download",
		}
		
		results, err := dataService.BatchCheckHasData(components)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if len(results) != 5 {
			t.Errorf("expected 5 results, got %d", len(results))
		}
		
		// Check expected results
		expectedResults := map[string]bool{
			"metrics-0":       true,  // has data sources
			"table-0":         true,  // has data sources
			"image-0":         false, // no images
			"insights-0":      false, // no insights
			"file_download-0": true,  // has files
		}
		
		for instanceID, expectedHasData := range expectedResults {
			if results[instanceID] != expectedHasData {
				t.Errorf("expected %s hasData to be %v, got %v", instanceID, expectedHasData, results[instanceID])
			}
		}
	})
	
	t.Run("batch with unsupported type", func(t *testing.T) {
		components := map[string]string{
			"metrics-0":     "metrics",
			"unsupported-0": "unsupported",
		}
		
		results, err := dataService.BatchCheckHasData(components)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		// Should have results for both, with unsupported returning false
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
		
		if results["unsupported-0"] != false {
			t.Error("expected unsupported component to return false")
		}
	})
}

// TestBatchCheckHasData_Performance tests that batch checking is efficient
func TestBatchCheckHasData_Performance(t *testing.T) {
	tempDir := t.TempDir()
	fileService := NewFileService(nil, tempDir)
	dataService := NewDataService(nil, tempDir, fileService)
	
	// Create test data
	metadataPath := filepath.Join(tempDir, "datasources.json")
	dataSources := []map[string]interface{}{
		{"id": "test-1", "name": "Test 1", "type": "csv"},
	}
	data, _ := json.Marshal(dataSources)
	os.WriteFile(metadataPath, data, 0644)
	
	// Create a large batch
	components := make(map[string]string)
	for i := 0; i < 100; i++ {
		components[fmt.Sprintf("metrics-%d", i)] = "metrics"
		components[fmt.Sprintf("table-%d", i)] = "table"
	}
	
	// This should complete quickly
	results, err := dataService.BatchCheckHasData(components)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(results) != 200 {
		t.Errorf("expected 200 results, got %d", len(results))
	}
}
