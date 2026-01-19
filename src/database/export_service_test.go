package database

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// MockDataService for testing
type MockDataService struct {
	hasDataMap map[string]bool
	batchError error
}

func (m *MockDataService) CheckComponentHasData(componentType, instanceID string) (bool, error) {
	if m.hasDataMap == nil {
		return false, nil
	}
	return m.hasDataMap[instanceID], nil
}

func (m *MockDataService) BatchCheckHasData(components map[string]string) (map[string]bool, error) {
	if m.batchError != nil {
		return nil, m.batchError
	}

	result := make(map[string]bool)
	for instanceID := range components {
		if m.hasDataMap != nil {
			result[instanceID] = m.hasDataMap[instanceID]
		} else {
			result[instanceID] = false
		}
	}
	return result, nil
}

// MockLayoutService for testing
type MockLayoutService struct{}

func (m *MockLayoutService) SaveLayout(config LayoutConfiguration) error {
	return nil
}

func (m *MockLayoutService) LoadLayout(userID string) (*LayoutConfiguration, error) {
	return &LayoutConfiguration{}, nil
}

func (m *MockLayoutService) GetDefaultLayout() LayoutConfiguration {
	return LayoutConfiguration{}
}

func TestNewExportService(t *testing.T) {
	dataService := &MockDataService{}
	layoutService := &MockLayoutService{}

	exportService := NewExportService(dataService, layoutService)

	if exportService == nil {
		t.Fatal("NewExportService returned nil")
	}

	if exportService.dataService != dataService {
		t.Error("DataService not set correctly")
	}

	if exportService.layoutService != layoutService {
		t.Error("LayoutService not set correctly")
	}
}

func TestFilterEmptyComponents(t *testing.T) {
	tests := []struct {
		name           string
		items          []LayoutItem
		hasDataMap     map[string]bool
		batchError     error
		expectedCount  int
		expectedError  bool
		expectedItems  []string
	}{
		{
			name: "Filter out empty components",
			items: []LayoutItem{
				{I: "metrics-0", Type: "metrics"},
				{I: "table-0", Type: "table"},
				{I: "image-0", Type: "image"},
			},
			hasDataMap: map[string]bool{
				"metrics-0": true,
				"table-0":   false,
				"image-0":   true,
			},
			expectedCount: 2,
			expectedItems: []string{"metrics-0", "image-0"},
		},
		{
			name: "All components have data",
			items: []LayoutItem{
				{I: "metrics-0", Type: "metrics"},
				{I: "table-0", Type: "table"},
			},
			hasDataMap: map[string]bool{
				"metrics-0": true,
				"table-0":   true,
			},
			expectedCount: 2,
			expectedItems: []string{"metrics-0", "table-0"},
		},
		{
			name: "No components have data",
			items: []LayoutItem{
				{I: "metrics-0", Type: "metrics"},
				{I: "table-0", Type: "table"},
			},
			hasDataMap: map[string]bool{
				"metrics-0": false,
				"table-0":   false,
			},
			expectedCount: 0,
			expectedItems: []string{},
		},
		{
			name: "Empty items list",
			items: []LayoutItem{},
			hasDataMap: map[string]bool{},
			expectedCount: 0,
			expectedItems: []string{},
		},
		{
			name: "Batch check error",
			items: []LayoutItem{
				{I: "metrics-0", Type: "metrics"},
			},
			batchError:    &MockError{"batch check failed"},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataService := &MockDataService{
				hasDataMap: tt.hasDataMap,
				batchError: tt.batchError,
			}
			layoutService := &MockLayoutService{}
			exportService := NewExportService(dataService, layoutService)

			result, err := exportService.FilterEmptyComponents(tt.items)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d items, got %d", tt.expectedCount, len(result))
			}

			// Check that the correct items are included
			resultIDs := make([]string, len(result))
			for i, item := range result {
				resultIDs[i] = item.I
			}

			for _, expectedID := range tt.expectedItems {
				found := false
				for _, resultID := range resultIDs {
					if resultID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected item %s not found in result", expectedID)
				}
			}
		})
	}
}

func TestFilterEmptyComponents_NilDataService(t *testing.T) {
	exportService := &ExportService{
		dataService:   nil,
		layoutService: &MockLayoutService{},
	}

	items := []LayoutItem{
		{I: "metrics-0", Type: "metrics"},
	}

	_, err := exportService.FilterEmptyComponents(items)
	if err == nil {
		t.Error("Expected error for nil data service")
	}

	if !strings.Contains(err.Error(), "data service not initialized") {
		t.Errorf("Expected 'data service not initialized' error, got: %v", err)
	}
}

func TestExportDashboard(t *testing.T) {
	tests := []struct {
		name           string
		request        ExportRequest
		hasDataMap     map[string]bool
		expectedError  bool
		errorContains  string
	}{
		{
			name: "Successful export with filtered components",
			request: ExportRequest{
				LayoutConfig: LayoutConfiguration{
					Items: []LayoutItem{
						{I: "metrics-0", Type: "metrics", X: 0, Y: 0, W: 4, H: 2},
						{I: "table-0", Type: "table", X: 4, Y: 0, W: 8, H: 4},
						{I: "image-0", Type: "image", X: 12, Y: 0, W: 4, H: 4},
					},
				},
				Format: "json",
				UserID: "test-user",
			},
			hasDataMap: map[string]bool{
				"metrics-0": true,
				"table-0":   false,
				"image-0":   true,
			},
		},
		{
			name: "Error - empty user ID",
			request: ExportRequest{
				LayoutConfig: LayoutConfiguration{
					Items: []LayoutItem{
						{I: "metrics-0", Type: "metrics"},
					},
				},
				Format: "json",
				UserID: "",
			},
			expectedError: true,
			errorContains: "userID is required",
		},
		{
			name: "Error - empty layout items",
			request: ExportRequest{
				LayoutConfig: LayoutConfiguration{
					Items: []LayoutItem{},
				},
				Format: "json",
				UserID: "test-user",
			},
			expectedError: true,
			errorContains: "layout configuration must contain at least one item",
		},
		{
			name: "Error - invalid format",
			request: ExportRequest{
				LayoutConfig: LayoutConfiguration{
					Items: []LayoutItem{
						{I: "metrics-0", Type: "metrics"},
					},
				},
				Format: "invalid",
				UserID: "test-user",
			},
			expectedError: true,
			errorContains: "unsupported export format",
		},
		{
			name: "Error - no components with data",
			request: ExportRequest{
				LayoutConfig: LayoutConfiguration{
					Items: []LayoutItem{
						{I: "metrics-0", Type: "metrics"},
						{I: "table-0", Type: "table"},
					},
				},
				Format: "json",
				UserID: "test-user",
			},
			hasDataMap: map[string]bool{
				"metrics-0": false,
				"table-0":   false,
			},
			expectedError: true,
			errorContains: "no components with data found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataService := &MockDataService{
				hasDataMap: tt.hasDataMap,
			}
			layoutService := &MockLayoutService{}
			exportService := NewExportService(dataService, layoutService)

			result, err := exportService.ExportDashboard(tt.request)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result but got nil")
			}

			// Verify result structure
			if result.FilePath == "" {
				t.Error("Expected file path to be set")
			}

			if result.TotalComponents != len(tt.request.LayoutConfig.Items) {
				t.Errorf("Expected total components %d, got %d", len(tt.request.LayoutConfig.Items), result.TotalComponents)
			}

			if result.Format != tt.request.Format {
				t.Errorf("Expected format %s, got %s", tt.request.Format, result.Format)
			}

			// Verify included/excluded components
			expectedIncluded := 0
			for instanceID, hasData := range tt.hasDataMap {
				if hasData {
					expectedIncluded++
					found := false
					for _, included := range result.IncludedComponents {
						if included == instanceID {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected component %s to be included", instanceID)
					}
				}
			}

			if len(result.IncludedComponents) != expectedIncluded {
				t.Errorf("Expected %d included components, got %d", expectedIncluded, len(result.IncludedComponents))
			}

			// Clean up generated file
			if result.FilePath != "" {
				os.Remove(result.FilePath)
			}
		})
	}
}

func TestExportDashboard_NilServices(t *testing.T) {
	tests := []struct {
		name          string
		dataService   DataServiceInterface
		layoutService LayoutServiceInterface
		errorContains string
	}{
		{
			name:          "Nil data service",
			dataService:   nil,
			layoutService: &MockLayoutService{},
			errorContains: "data service not initialized",
		},
		{
			name:          "Nil layout service",
			dataService:   &MockDataService{},
			layoutService: nil,
			errorContains: "layout service not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exportService := &ExportService{
				dataService:   tt.dataService,
				layoutService: tt.layoutService,
			}

			request := ExportRequest{
				LayoutConfig: LayoutConfiguration{
					Items: []LayoutItem{
						{I: "metrics-0", Type: "metrics"},
					},
				},
				Format: "json",
				UserID: "test-user",
			}

			_, err := exportService.ExportDashboard(request)
			if err == nil {
				t.Error("Expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
			}
		})
	}
}

func TestGenerateJSONExport(t *testing.T) {
	exportService := NewExportService(&MockDataService{}, &MockLayoutService{})

	componentData := []ComponentData{
		{
			ID:          "metrics-0",
			Type:        "metrics",
			InstanceIdx: 0,
			Position:    Position{X: 0, Y: 0, W: 4, H: 2},
			Data:        map[string]interface{}{"test": "data"},
		},
	}

	// Create temp directory for test
	tempDir := os.TempDir()
	filePath := filepath.Join(tempDir, "test_export.json")
	defer os.Remove(filePath)

	resultPath, err := exportService.generateJSONExport(componentData, filePath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resultPath != filePath {
		t.Errorf("Expected path %s, got %s", filePath, resultPath)
	}

	// Verify file was created and has correct content
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Export file was not created")
	}

	// Read and verify JSON content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportData map[string]interface{}
	err = json.Unmarshal(content, &exportData)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify structure
	if exportData["format"] != "json" {
		t.Error("Expected format to be 'json'")
	}

	if exportData["totalCount"] != float64(1) {
		t.Errorf("Expected totalCount to be 1, got %v", exportData["totalCount"])
	}

	components, ok := exportData["components"].([]interface{})
	if !ok {
		t.Error("Expected components to be an array")
	}

	if len(components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(components))
	}
}

func TestGenerateCSVExport(t *testing.T) {
	exportService := NewExportService(&MockDataService{}, &MockLayoutService{})

	componentData := []ComponentData{
		{
			ID:          "metrics-0",
			Type:        "metrics",
			InstanceIdx: 0,
			Position:    Position{X: 0, Y: 0, W: 4, H: 2},
		},
		{
			ID:          "table-0",
			Type:        "table",
			InstanceIdx: 1,
			Position:    Position{X: 4, Y: 0, W: 8, H: 4},
		},
	}

	// Create temp directory for test
	tempDir := os.TempDir()
	filePath := filepath.Join(tempDir, "test_export.csv")
	defer os.Remove(filePath)

	resultPath, err := exportService.generateCSVExport(componentData, filePath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resultPath != filePath {
		t.Errorf("Expected path %s, got %s", filePath, resultPath)
	}

	// Verify file was created and has correct content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	contentStr := string(content)
	lines := strings.Split(strings.TrimSpace(contentStr), "\n")

	// Verify header
	expectedHeader := "Component ID,Type,Instance Index,X,Y,Width,Height"
	if lines[0] != expectedHeader {
		t.Errorf("Expected header '%s', got '%s'", expectedHeader, lines[0])
	}

	// Verify data rows
	if len(lines) != 3 { // header + 2 data rows
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	expectedRow1 := "metrics-0,metrics,0,0,0,4,2"
	if lines[1] != expectedRow1 {
		t.Errorf("Expected row 1 '%s', got '%s'", expectedRow1, lines[1])
	}

	expectedRow2 := "table-0,table,1,4,0,8,4"
	if lines[2] != expectedRow2 {
		t.Errorf("Expected row 2 '%s', got '%s'", expectedRow2, lines[2])
	}
}

func TestCalculateExcludedComponents(t *testing.T) {
	exportService := NewExportService(&MockDataService{}, &MockLayoutService{})

	allItems := []LayoutItem{
		{I: "metrics-0"},
		{I: "table-0"},
		{I: "image-0"},
		{I: "insights-0"},
	}

	includedItems := []LayoutItem{
		{I: "metrics-0"},
		{I: "image-0"},
	}

	excluded := exportService.calculateExcludedComponents(allItems, includedItems)

	expectedExcluded := []string{"table-0", "insights-0"}
	if len(excluded) != len(expectedExcluded) {
		t.Errorf("Expected %d excluded components, got %d", len(expectedExcluded), len(excluded))
	}

	for _, expected := range expectedExcluded {
		found := false
		for _, actual := range excluded {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected excluded component %s not found", expected)
		}
	}
}

func TestGetComponentDataByType(t *testing.T) {
	exportService := NewExportService(&MockDataService{}, &MockLayoutService{})

	tests := []struct {
		name          string
		componentType string
		instanceID    string
		expectError   bool
	}{
		{"metrics", "metrics", "metrics-0", false},
		{"table", "table", "table-0", false},
		{"image", "image", "image-0", false},
		{"insights", "insights", "insights-0", false},
		{"file_download", "file_download", "file_download-0", false},
		{"unsupported", "unsupported", "unsupported-0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := exportService.getComponentDataByType(tt.componentType, tt.instanceID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if data == nil {
				t.Error("Expected data but got nil")
			}

			// Verify data structure
			dataMap, ok := data.(map[string]interface{})
			if !ok {
				t.Error("Expected data to be a map")
			}

			if dataMap["type"] != tt.componentType {
				t.Errorf("Expected type %s, got %v", tt.componentType, dataMap["type"])
			}

			if dataMap["instanceId"] != tt.instanceID {
				t.Errorf("Expected instanceId %s, got %v", tt.instanceID, dataMap["instanceId"])
			}
		})
	}
}

// MockError for testing error conditions
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// Test property: Export result metadata includes correct component lists
func TestExportResultMetadata(t *testing.T) {
	dataService := &MockDataService{
		hasDataMap: map[string]bool{
			"metrics-0":      true,
			"table-0":        false,
			"image-0":        true,
			"insights-0":     false,
			"file_download-0": true,
		},
	}
	layoutService := &MockLayoutService{}
	exportService := NewExportService(dataService, layoutService)

	request := ExportRequest{
		LayoutConfig: LayoutConfiguration{
			Items: []LayoutItem{
				{I: "metrics-0", Type: "metrics"},
				{I: "table-0", Type: "table"},
				{I: "image-0", Type: "image"},
				{I: "insights-0", Type: "insights"},
				{I: "file_download-0", Type: "file_download"},
			},
		},
		Format: "json",
		UserID: "test-user",
	}

	result, err := exportService.ExportDashboard(request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Clean up
	defer os.Remove(result.FilePath)

	// Verify metadata
	if result.TotalComponents != 5 {
		t.Errorf("Expected total components 5, got %d", result.TotalComponents)
	}

	expectedIncluded := []string{"metrics-0", "image-0", "file_download-0"}
	if len(result.IncludedComponents) != len(expectedIncluded) {
		t.Errorf("Expected %d included components, got %d", len(expectedIncluded), len(result.IncludedComponents))
	}

	expectedExcluded := []string{"table-0", "insights-0"}
	if len(result.ExcludedComponents) != len(expectedExcluded) {
		t.Errorf("Expected %d excluded components, got %d", len(expectedExcluded), len(result.ExcludedComponents))
	}

	// Verify export timestamp is recent
	exportTime, err := time.Parse(time.RFC3339, result.ExportedAt)
	if err != nil {
		t.Errorf("Failed to parse export timestamp: %v", err)
	}

	if time.Since(exportTime) > time.Minute {
		t.Error("Export timestamp is not recent")
	}
}