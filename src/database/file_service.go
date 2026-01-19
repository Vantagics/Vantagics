package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"
)

// FileCategory represents the category of a file
type FileCategory string

const (
	// AllFiles represents all files available for download
	AllFiles FileCategory = "all_files"
	// UserRequestRelated represents files related to user requests
	UserRequestRelated FileCategory = "user_request_related"
)

// FileInfo represents metadata about a downloadable file
type FileInfo struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Size        int64        `json:"size"`
	CreatedAt   int64        `json:"createdAt"`
	Category    FileCategory `json:"category"`
	DownloadURL string       `json:"downloadUrl"`
}

// FileService provides methods for managing downloadable files
type FileService struct {
	db      *sql.DB
	dataDir string
}

// NewFileService creates a new FileService instance
func NewFileService(db *sql.DB, dataDir string) *FileService {
	return &FileService{
		db:      db,
		dataDir: dataDir,
	}
}

// GetFilesByCategory retrieves all files for a specific category
func (s *FileService) GetFilesByCategory(category FileCategory) ([]FileInfo, error) {
	// For now, we'll scan the filesystem in the dataDir
	// In a future implementation, this could be backed by a database table
	
	var files []FileInfo
	
	// Define subdirectories for each category
	var categoryDir string
	switch category {
	case AllFiles:
		categoryDir = "files"
	case UserRequestRelated:
		categoryDir = "user_requests"
	default:
		return files, nil
	}
	
	// Build the full path using filepath.Join for cross-platform compatibility
	fullPath := filepath.Join(s.dataDir, categoryDir)
	
	// Check if directory exists
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		// If directory doesn't exist, return empty list (not an error)
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, err
	}
	
	// Scan files in the directory
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		fileInfo := FileInfo{
			ID:          entry.Name(), // Use filename as ID for now
			Name:        entry.Name(),
			Size:        info.Size(),
			CreatedAt:   info.ModTime().UnixMilli(),
			Category:    category,
			DownloadURL: "/api/files/download/" + entry.Name(),
		}
		
		files = append(files, fileInfo)
	}
	
	return files, nil
}

// HasFiles checks if any files exist in either category
func (s *FileService) HasFiles() (bool, error) {
	// Check AllFiles category
	allFiles, err := s.GetFilesByCategory(AllFiles)
	if err != nil {
		return false, err
	}
	if len(allFiles) > 0 {
		return true, nil
	}
	
	// Check UserRequestRelated category
	userFiles, err := s.GetFilesByCategory(UserRequestRelated)
	if err != nil {
		return false, err
	}
	if len(userFiles) > 0 {
		return true, nil
	}
	
	return false, nil
}

// DownloadFile returns the file path for download given a file ID
func (s *FileService) DownloadFile(fileID string) (string, error) {
	// Try to find the file in both categories
	categories := []FileCategory{AllFiles, UserRequestRelated}
	
	for _, category := range categories {
		var categoryDir string
		switch category {
		case AllFiles:
			categoryDir = "files"
		case UserRequestRelated:
			categoryDir = "user_requests"
		}
		
		filePath := filepath.Join(s.dataDir, categoryDir, fileID)
		
		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
	}
	
	return "", os.ErrNotExist
}
