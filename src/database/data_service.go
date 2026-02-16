package database

import (
	"fmt"
	"os"
	"path/filepath"
)

// DataService provides methods for checking component data availability
type DataService struct {
	dataCacheDir   string
	fileService    *FileService
	dataSourceSvc  interface{} // Will be set to agent.DataSourceService to avoid circular dependency
}

// NewDataService creates a new DataService instance
func NewDataService(dataCacheDir string, fileService *FileService) *DataService {
	return &DataService{
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
	
	// Use a shared datasources.json check to avoid repeated file reads
	var datasourcesHasData *bool
	checkDatasources := func() (bool, error) {
		if datasourcesHasData != nil {
			return *datasourcesHasData, nil
		}
		metadataPath := filepath.Join(s.dataCacheDir, "datasources.json")
		info, err := os.Stat(metadataPath)
		if err != nil {
			if os.IsNotExist(err) {
				val := false
				datasourcesHasData = &val
				return false, nil
			}
			return false, err
		}
		val := info.Size() > 2 // more than just "[]"
		datasourcesHasData = &val
		return val, nil
	}
	
	for instanceID, componentType := range components {
		var hasData bool
		var err error
		
		switch componentType {
		case "metrics", "table":
			hasData, err = checkDatasources()
		default:
			hasData, err = s.CheckComponentHasData(componentType, instanceID)
		}
		
		if err != nil {
			results[instanceID] = false
			continue
		}
		results[instanceID] = hasData
	}
	
	return results, nil
}

// checkMetricsHasData checks if metrics component has data
func (s *DataService) checkMetricsHasData(instanceID string) (bool, error) {
	metadataPath := filepath.Join(s.dataCacheDir, "datasources.json")
	info, err := os.Stat(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	// File with only "[]" or "[\n]" is at most 3 bytes
	return info.Size() > 3, nil
}

// checkTableHasData checks if table component has data
func (s *DataService) checkTableHasData(instanceID string) (bool, error) {
	metadataPath := filepath.Join(s.dataCacheDir, "datasources.json")
	info, err := os.Stat(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Size() > 3, nil
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
	insightsPath := filepath.Join(s.dataCacheDir, "insights.json")
	info, err := os.Stat(insightsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Size() > 3, nil
}

// checkFileDownloadHasData checks if file download component has data
// Uses FileService.HasFiles() to check both categories
func (s *DataService) checkFileDownloadHasData(instanceID string) (bool, error) {
	if s.fileService == nil {
		return false, fmt.Errorf("file service not initialized")
	}
	
	return s.fileService.HasFiles()
}
