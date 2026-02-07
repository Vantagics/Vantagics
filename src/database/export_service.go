package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DataServiceInterface defines the interface for data service operations
type DataServiceInterface interface {
	CheckComponentHasData(componentType, instanceID string) (bool, error)
	BatchCheckHasData(components map[string]string) (map[string]bool, error)
}

// LayoutServiceInterface defines the interface for layout service operations
type LayoutServiceInterface interface {
	SaveLayout(config LayoutConfiguration) error
	LoadLayout(userID string) (*LayoutConfiguration, error)
	GetDefaultLayout() LayoutConfiguration
}

// ExportService provides methods for exporting dashboard data with component filtering
type ExportService struct {
	dataService   DataServiceInterface
	layoutService LayoutServiceInterface
}

// NewExportService creates a new ExportService instance
func NewExportService(dataService DataServiceInterface, layoutService LayoutServiceInterface) *ExportService {
	return &ExportService{
		dataService:   dataService,
		layoutService: layoutService,
	}
}

// ExportRequest represents the request structure for dashboard export
type ExportRequest struct {
	LayoutConfig LayoutConfiguration `json:"layoutConfig"`
	Format       string              `json:"format"` // "json", "xlsx", "csv"
	UserID       string              `json:"userId"`
}

// ExportResult represents the result of a dashboard export operation
type ExportResult struct {
	FilePath           string   `json:"filePath"`
	IncludedComponents []string `json:"includedComponents"`
	ExcludedComponents []string `json:"excludedComponents"`
	TotalComponents    int      `json:"totalComponents"`
	ExportedAt         string   `json:"exportedAt"`
	Format             string   `json:"format"`
}

// ComponentData represents the data for a single component in the export
type ComponentData struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	InstanceIdx int         `json:"instanceIdx"`
	Position    Position    `json:"position"`
	Data        interface{} `json:"data"`
}

// Position represents the position and size of a component
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// ExportDashboard exports dashboard data, filtering out empty components
// This is the main export method that coordinates filtering and export generation
func (s *ExportService) ExportDashboard(req ExportRequest) (*ExportResult, error) {
	if s.dataService == nil {
		return nil, fmt.Errorf("data service not initialized")
	}

	if s.layoutService == nil {
		return nil, fmt.Errorf("layout service not initialized")
	}

	// Validate request
	if req.UserID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	if len(req.LayoutConfig.Items) == 0 {
		return nil, fmt.Errorf("layout configuration must contain at least one item")
	}

	// Validate format
	validFormats := map[string]bool{
		"json": true,
		"xlsx": true,
		"csv":  true,
	}
	if !validFormats[req.Format] {
		return nil, fmt.Errorf("unsupported export format: %s", req.Format)
	}

	// Filter empty components
	filteredItems, err := s.FilterEmptyComponents(req.LayoutConfig.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to filter empty components: %w", err)
	}

	// Check if any components have data
	if len(filteredItems) == 0 {
		return nil, fmt.Errorf("no components with data found - cannot generate empty export")
	}

	// Collect component data for export
	componentData, err := s.collectComponentData(filteredItems)
	if err != nil {
		return nil, fmt.Errorf("failed to collect component data: %w", err)
	}

	// Generate export file
	filePath, err := s.generateExportFile(componentData, req.Format, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate export file: %w", err)
	}

	// Calculate included and excluded components
	includedComponents := make([]string, len(filteredItems))
	for i, item := range filteredItems {
		includedComponents[i] = item.I
	}

	excludedComponents := s.calculateExcludedComponents(req.LayoutConfig.Items, filteredItems)

	// Create export result
	result := &ExportResult{
		FilePath:           filePath,
		IncludedComponents: includedComponents,
		ExcludedComponents: excludedComponents,
		TotalComponents:    len(req.LayoutConfig.Items),
		ExportedAt:         time.Now().Format(time.RFC3339),
		Format:             req.Format,
	}

	return result, nil
}

// FilterEmptyComponents filters out layout items that don't have data
// Returns only components that have data available
func (s *ExportService) FilterEmptyComponents(items []LayoutItem) ([]LayoutItem, error) {
	if s.dataService == nil {
		return nil, fmt.Errorf("data service not initialized")
	}

	// Build component map for batch checking
	componentMap := make(map[string]string)
	for _, item := range items {
		componentMap[item.I] = item.Type
	}

	// Batch check data availability
	hasDataMap, err := s.dataService.BatchCheckHasData(componentMap)
	if err != nil {
		return nil, fmt.Errorf("failed to check component data availability: %w", err)
	}

	// Filter items based on data availability
	var filteredItems []LayoutItem
	for _, item := range items {
		if hasData, exists := hasDataMap[item.I]; exists && hasData {
			filteredItems = append(filteredItems, item)
		}
	}

	return filteredItems, nil
}

// collectComponentData collects actual data for each component
func (s *ExportService) collectComponentData(items []LayoutItem) ([]ComponentData, error) {
	var componentData []ComponentData

	for _, item := range items {
		// Get component data based on type
		data, err := s.getComponentDataByType(item.Type, item.I)
		if err != nil {
			// Log error but continue with other components
			// This ensures partial exports can still succeed
			continue
		}

		componentData = append(componentData, ComponentData{
			ID:          item.I,
			Type:        item.Type,
			InstanceIdx: item.InstanceIdx,
			Position: Position{
				X: item.X,
				Y: item.Y,
				W: item.W,
				H: item.H,
			},
			Data: data,
		})
	}

	return componentData, nil
}

// getComponentDataByType retrieves data for a specific component type
func (s *ExportService) getComponentDataByType(componentType, instanceID string) (interface{}, error) {
	// This is a simplified implementation
	// In a real system, you would call specific data retrieval methods
	// based on the component type and instance ID

	switch componentType {
	case "metrics":
		return s.getMetricsData(instanceID)
	case "table":
		return s.getTableData(instanceID)
	case "image":
		return s.getImageData(instanceID)
	case "insights":
		return s.getInsightsData(instanceID)
	case "file_download":
		return s.getFileDownloadData(instanceID)
	default:
		return nil, fmt.Errorf("unsupported component type: %s", componentType)
	}
}

// getMetricsData retrieves metrics data for export
func (s *ExportService) getMetricsData(instanceID string) (interface{}, error) {
	// Placeholder implementation - in real system, this would query actual metrics
	return map[string]interface{}{
		"type":        "metrics",
		"instanceId":  instanceID,
		"placeholder": "Metrics data would be retrieved here",
	}, nil
}

// getTableData retrieves table data for export
func (s *ExportService) getTableData(instanceID string) (interface{}, error) {
	// Placeholder implementation - in real system, this would query actual table data
	return map[string]interface{}{
		"type":        "table",
		"instanceId":  instanceID,
		"placeholder": "Table data would be retrieved here",
	}, nil
}

// getImageData retrieves image data for export
func (s *ExportService) getImageData(instanceID string) (interface{}, error) {
	// Placeholder implementation - in real system, this would retrieve image metadata/paths
	return map[string]interface{}{
		"type":        "image",
		"instanceId":  instanceID,
		"placeholder": "Image data would be retrieved here",
	}, nil
}

// getInsightsData retrieves insights data for export
func (s *ExportService) getInsightsData(instanceID string) (interface{}, error) {
	// Placeholder implementation - in real system, this would query actual insights
	return map[string]interface{}{
		"type":        "insights",
		"instanceId":  instanceID,
		"placeholder": "Insights data would be retrieved here",
	}, nil
}

// getFileDownloadData retrieves file download data for export
func (s *ExportService) getFileDownloadData(instanceID string) (interface{}, error) {
	// Placeholder implementation - in real system, this would retrieve file metadata
	return map[string]interface{}{
		"type":        "file_download",
		"instanceId":  instanceID,
		"placeholder": "File download data would be retrieved here",
	}, nil
}

// generateExportFile creates the actual export file in the specified format
func (s *ExportService) generateExportFile(componentData []ComponentData, format, userID string) (string, error) {
	// Create export directory if it doesn't exist
	exportDir := filepath.Join(os.TempDir(), "dashboard_exports")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("dashboard_export_%s_%s.%s", userID, timestamp, format)
	filePath := filepath.Join(exportDir, filename)

	switch format {
	case "json":
		return s.generateJSONExport(componentData, filePath)
	case "xlsx":
		return s.generateXLSXExport(componentData, filePath)
	case "csv":
		return s.generateCSVExport(componentData, filePath)
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
}

// generateJSONExport creates a JSON export file
func (s *ExportService) generateJSONExport(componentData []ComponentData, filePath string) (string, error) {
	exportData := map[string]interface{}{
		"exportedAt":  time.Now().Format(time.RFC3339),
		"components":  componentData,
		"totalCount":  len(componentData),
		"format":      "json",
	}

	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write JSON file: %w", err)
	}

	return filePath, nil
}

// generateXLSXExport creates an Excel export file
func (s *ExportService) generateXLSXExport(componentData []ComponentData, filePath string) (string, error) {
	// For now, create a simple text file as placeholder
	// In a real implementation, you would use GoExcel library
	content := "XLSX Export Placeholder\n"
	content += fmt.Sprintf("Exported at: %s\n", time.Now().Format(time.RFC3339))
	content += fmt.Sprintf("Components: %d\n\n", len(componentData))

	for _, comp := range componentData {
		content += fmt.Sprintf("Component: %s (Type: %s)\n", comp.ID, comp.Type)
		content += fmt.Sprintf("Position: x=%d, y=%d, w=%d, h=%d\n", comp.Position.X, comp.Position.Y, comp.Position.W, comp.Position.H)
		content += "---\n"
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write XLSX file: %w", err)
	}

	return filePath, nil
}

// generateCSVExport creates a CSV export file
func (s *ExportService) generateCSVExport(componentData []ComponentData, filePath string) (string, error) {
	content := "Component ID,Type,Instance Index,X,Y,Width,Height\n"

	for _, comp := range componentData {
		content += fmt.Sprintf("%s,%s,%d,%d,%d,%d,%d\n",
			comp.ID, comp.Type, comp.InstanceIdx,
			comp.Position.X, comp.Position.Y, comp.Position.W, comp.Position.H)
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write CSV file: %w", err)
	}

	return filePath, nil
}

// calculateExcludedComponents determines which components were excluded from export
func (s *ExportService) calculateExcludedComponents(allItems, includedItems []LayoutItem) []string {
	// Create a map of included component IDs for fast lookup
	includedMap := make(map[string]bool)
	for _, item := range includedItems {
		includedMap[item.I] = true
	}

	// Find excluded components
	var excludedComponents []string
	for _, item := range allItems {
		if !includedMap[item.I] {
			excludedComponents = append(excludedComponents, item.I)
		}
	}

	return excludedComponents
}