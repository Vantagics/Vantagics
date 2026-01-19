package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// DataService provides methods for checking component data availability
type DataService struct {
	db             *sql.DB
	dataCacheDir   string
	fileService    *FileService
	dataSourceSvc  interface{} // Will be set to agent.DataSourceService to avoid circular dependency
}

// NewDataService creates a new DataService instance
func NewDataService(db *sql.DB, dataCacheDir string, fileService *FileService) *DataService {
	return &DataService{
		db:           db,
		dataCacheDir: dataCacheDir,
		fileService:  fileService,
	}
}

// SetDataSourceService sets the data source service (to avoid circular dependency)
func (s *DataService) SetDataSourceService(dataSourceSvc interface{}) {
	s.dataSourceSvc = dataSourceSvc
}

// CheckComponentHasData checks if a specific component instance has data
// componentType: "metrics", "table", "image", "insights", "file_download"
// instanceID: the component instance identifier (e.g., "metrics-0", "table-1")
func (s *DataService) CheckComponentHasData(componentType string, instanceID string) (bool, error) {
	switch componentType {
	case "metrics":
		return s.checkMetricsHasData(instanceID)
	case "table":
		return s.checkTableHasData(instanceID)
	case "image":
		return s.checkImageHasData(instanceID)
	case "insights":
		return s.checkInsightsHasData(instanceID)
	case "file_download":
		return s.checkFileDownloadHasData(instanceID)
	default:
		return false, fmt.Errorf("unsupported component type: %s", componentType)
	}
}

// BatchCheckHasData checks multiple components at once for efficiency
// components: map of component instance ID to component type
// Returns: map of component instance ID to hasData boolean
func (s *DataService) BatchCheckHasData(components map[string]string) (map[string]bool, error) {
	results := make(map[string]bool)
	
	for instanceID, componentType := range components {
		hasData, err := s.CheckComponentHasData(componentType, instanceID)
		if err != nil {
			// Log error but continue with other components
			// Assume no data on error (fail closed for visibility)
			results[instanceID] = false
			continue
		}
		results[instanceID] = hasData
	}
	
	return results, nil
}

// checkMetricsHasData checks if metrics component has data
func (s *DataService) checkMetricsHasData(instanceID string) (bool, error) {
	// Check if there are any data sources with tables
	// Metrics are typically derived from data source queries
	
	// For now, check if any data sources exist with data
	// This is a simplified implementation - in production, you might want to
	// check specific metric queries or cached metric values
	
	// Check if datasources.json exists and has entries
	metadataPath := filepath.Join(s.dataCacheDir, "datasources.json")
	if _, err := os.Stat(metadataPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	
	// Read and check if there are any data sources
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return false, err
	}
	
	// Simple check: if file is empty or just "[]", no data
	if len(data) == 0 || string(data) == "[]" || string(data) == "[\n]" {
		return false, nil
	}
	
	return true, nil
}

// checkTableHasData checks if table component has data
func (s *DataService) checkTableHasData(instanceID string) (bool, error) {
	// Check if there are any data sources with tables
	// Similar to metrics, check if data sources exist
	
	metadataPath := filepath.Join(s.dataCacheDir, "datasources.json")
	if _, err := os.Stat(metadataPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return false, err
	}
	
	if len(data) == 0 || string(data) == "[]" || string(data) == "[\n]" {
		return false, nil
	}
	
	return true, nil
}

// checkImageHasData checks if image component has data
func (s *DataService) checkImageHasData(instanceID string) (bool, error) {
	// Check if there are any image files in the data directory
	// Images are typically stored in a specific directory
	
	// Check for common image directories
	imageDirs := []string{
		filepath.Join(s.dataCacheDir, "images"),
		filepath.Join(s.dataCacheDir, "charts"),
		filepath.Join(s.dataCacheDir, "visualizations"),
	}
	
	for _, dir := range imageDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return false, err
		}
		
		// Check if there are any image files
		for _, entry := range entries {
			if !entry.IsDir() {
				// Found at least one file
				return true, nil
			}
		}
	}
	
	return false, nil
}

// checkInsightsHasData checks if insights component has data
func (s *DataService) checkInsightsHasData(instanceID string) (bool, error) {
	// Check if there are any insights stored
	// Insights might be stored in a database table or file
	
	// Check for insights file or database table
	insightsPath := filepath.Join(s.dataCacheDir, "insights.json")
	if _, err := os.Stat(insightsPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	
	data, err := os.ReadFile(insightsPath)
	if err != nil {
		return false, err
	}
	
	if len(data) == 0 || string(data) == "[]" || string(data) == "[\n]" {
		return false, nil
	}
	
	return true, nil
}

// checkFileDownloadHasData checks if file download component has data
// Uses FileService.HasFiles() to check both categories
func (s *DataService) checkFileDownloadHasData(instanceID string) (bool, error) {
	if s.fileService == nil {
		return false, fmt.Errorf("file service not initialized")
	}
	
	return s.fileService.HasFiles()
}
