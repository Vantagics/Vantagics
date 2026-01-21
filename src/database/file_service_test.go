package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFileServiceStructure verifies the FileService struct and interfaces are properly defined
func TestFileServiceStructure(t *testing.T) {
	// Test FileCategory constants
	if AllFiles != "all_files" {
		t.Errorf("AllFiles constant should be 'all_files', got '%s'", AllFiles)
	}
	if UserRequestRelated != "user_request_related" {
		t.Errorf("UserRequestRelated constant should be 'user_request_related', got '%s'", UserRequestRelated)
	}

	// Test FileService creation
	service := NewFileService(nil, "/test/data")
	if service == nil {
		t.Fatal("NewFileService should return a non-nil service")
	}

	if service.dataDir != "/test/data" {
		t.Errorf("Expected dataDir to be '/test/data', got '%s'", service.dataDir)
	}
}

// TestFileInfoStructure verifies the FileInfo struct has all required fields
func TestFileInfoStructure(t *testing.T) {
	fileInfo := FileInfo{
		ID:          "test-id",
		Name:        "test-file.txt",
		Size:        1024,
		Category:    AllFiles,
		DownloadURL: "/download/test-id",
	}

	if fileInfo.ID != "test-id" {
		t.Errorf("Expected ID to be 'test-id', got '%s'", fileInfo.ID)
	}
	if fileInfo.Name != "test-file.txt" {
		t.Errorf("Expected Name to be 'test-file.txt', got '%s'", fileInfo.Name)
	}
	if fileInfo.Size != 1024 {
		t.Errorf("Expected Size to be 1024, got %d", fileInfo.Size)
	}
	if fileInfo.Category != AllFiles {
		t.Errorf("Expected Category to be AllFiles, got '%s'", fileInfo.Category)
	}
	if fileInfo.DownloadURL != "/download/test-id" {
		t.Errorf("Expected DownloadURL to be '/download/test-id', got '%s'", fileInfo.DownloadURL)
	}
}

// TestFileCategoryType verifies FileCategory is a string type
func TestFileCategoryType(t *testing.T) {
	var category FileCategory = "test_category"
	if string(category) != "test_category" {
		t.Errorf("FileCategory should be convertible to string")
	}
}

// setupTestDir creates a temporary directory with test files
func setupTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "fileservice_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create subdirectories for categories
	filesDir := filepath.Join(tmpDir, "files")
	userRequestsDir := filepath.Join(tmpDir, "user_requests")

	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files dir: %v", err)
	}
	if err := os.MkdirAll(userRequestsDir, 0755); err != nil {
		t.Fatalf("Failed to create user_requests dir: %v", err)
	}

	// Create test files in AllFiles category
	testFiles := []string{"file1.txt", "file2.csv", "file3.json"}
	for _, filename := range testFiles {
		path := filepath.Join(filesDir, filename)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create test files in UserRequestRelated category
	userFiles := []string{"request1.txt", "request2.pdf"}
	for _, filename := range userFiles {
		path := filepath.Join(userRequestsDir, filename)
		if err := os.WriteFile(path, []byte("user request content"), 0644); err != nil {
			t.Fatalf("Failed to create user request file %s: %v", filename, err)
		}
	}

	return tmpDir
}

// TestGetFilesByCategory_AllFiles tests retrieving files from AllFiles category
func TestGetFilesByCategory_AllFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	service := NewFileService(nil, tmpDir)
	files, err := service.GetFilesByCategory(AllFiles)

	if err != nil {
		t.Fatalf("GetFilesByCategory returned error: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	// Verify file properties
	for _, file := range files {
		if file.ID == "" {
			t.Error("File ID should not be empty")
		}
		if file.Name == "" {
			t.Error("File Name should not be empty")
		}
		if file.Size <= 0 {
			t.Error("File Size should be greater than 0")
		}
		if file.Category != AllFiles {
			t.Errorf("Expected category AllFiles, got %s", file.Category)
		}
		if file.DownloadURL == "" {
			t.Error("DownloadURL should not be empty")
		}
		if file.CreatedAt == 0 {
			t.Error("CreatedAt should not be zero")
		}
	}
}

// TestGetFilesByCategory_UserRequestRelated tests retrieving files from UserRequestRelated category
func TestGetFilesByCategory_UserRequestRelated(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	service := NewFileService(nil, tmpDir)
	files, err := service.GetFilesByCategory(UserRequestRelated)

	if err != nil {
		t.Fatalf("GetFilesByCategory returned error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Verify category is correct
	for _, file := range files {
		if file.Category != UserRequestRelated {
			t.Errorf("Expected category UserRequestRelated, got %s", file.Category)
		}
	}
}

// TestGetFilesByCategory_EmptyDirectory tests behavior with empty directory
func TestGetFilesByCategory_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileservice_empty_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create empty subdirectories
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files dir: %v", err)
	}

	service := NewFileService(nil, tmpDir)
	files, err := service.GetFilesByCategory(AllFiles)

	if err != nil {
		t.Fatalf("GetFilesByCategory returned error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d", len(files))
	}
}

// TestGetFilesByCategory_NonExistentDirectory tests behavior when directory doesn't exist
func TestGetFilesByCategory_NonExistentDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileservice_nonexistent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Don't create subdirectories
	service := NewFileService(nil, tmpDir)
	files, err := service.GetFilesByCategory(AllFiles)

	// Should return empty list, not an error
	if err != nil {
		t.Fatalf("GetFilesByCategory should not return error for non-existent directory: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files for non-existent directory, got %d", len(files))
	}
}

// TestGetFilesByCategory_InvalidCategory tests behavior with invalid category
func TestGetFilesByCategory_InvalidCategory(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	service := NewFileService(nil, tmpDir)
	files, err := service.GetFilesByCategory(FileCategory("invalid_category"))

	if err != nil {
		t.Fatalf("GetFilesByCategory returned error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files for invalid category, got %d", len(files))
	}
}

// TestHasFiles_WithFiles tests HasFiles when files exist
func TestHasFiles_WithFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	service := NewFileService(nil, tmpDir)
	hasFiles, err := service.HasFiles()

	if err != nil {
		t.Fatalf("HasFiles returned error: %v", err)
	}

	if !hasFiles {
		t.Error("HasFiles should return true when files exist")
	}
}

// TestHasFiles_WithoutFiles tests HasFiles when no files exist
func TestHasFiles_WithoutFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileservice_nofiles_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create empty subdirectories
	filesDir := filepath.Join(tmpDir, "files")
	userRequestsDir := filepath.Join(tmpDir, "user_requests")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files dir: %v", err)
	}
	if err := os.MkdirAll(userRequestsDir, 0755); err != nil {
		t.Fatalf("Failed to create user_requests dir: %v", err)
	}

	service := NewFileService(nil, tmpDir)
	hasFiles, err := service.HasFiles()

	if err != nil {
		t.Fatalf("HasFiles returned error: %v", err)
	}

	if hasFiles {
		t.Error("HasFiles should return false when no files exist")
	}
}

// TestHasFiles_OnlyAllFiles tests HasFiles when only AllFiles category has files
func TestHasFiles_OnlyAllFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileservice_allfiles_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories
	filesDir := filepath.Join(tmpDir, "files")
	userRequestsDir := filepath.Join(tmpDir, "user_requests")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files dir: %v", err)
	}
	if err := os.MkdirAll(userRequestsDir, 0755); err != nil {
		t.Fatalf("Failed to create user_requests dir: %v", err)
	}

	// Create file only in AllFiles category
	path := filepath.Join(filesDir, "test.txt")
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := NewFileService(nil, tmpDir)
	hasFiles, err := service.HasFiles()

	if err != nil {
		t.Fatalf("HasFiles returned error: %v", err)
	}

	if !hasFiles {
		t.Error("HasFiles should return true when AllFiles category has files")
	}
}

// TestHasFiles_OnlyUserRequestRelated tests HasFiles when only UserRequestRelated category has files
func TestHasFiles_OnlyUserRequestRelated(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileservice_userrequest_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories
	filesDir := filepath.Join(tmpDir, "files")
	userRequestsDir := filepath.Join(tmpDir, "user_requests")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files dir: %v", err)
	}
	if err := os.MkdirAll(userRequestsDir, 0755); err != nil {
		t.Fatalf("Failed to create user_requests dir: %v", err)
	}

	// Create file only in UserRequestRelated category
	path := filepath.Join(userRequestsDir, "request.txt")
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := NewFileService(nil, tmpDir)
	hasFiles, err := service.HasFiles()

	if err != nil {
		t.Fatalf("HasFiles returned error: %v", err)
	}

	if !hasFiles {
		t.Error("HasFiles should return true when UserRequestRelated category has files")
	}
}

// TestDownloadFile_Success tests successful file download
func TestDownloadFile_Success(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	service := NewFileService(nil, tmpDir)

	// Test downloading from AllFiles category
	filePath, err := service.DownloadFile("file1.txt")
	if err != nil {
		t.Fatalf("DownloadFile returned error: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "files", "file1.txt")
	if filePath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, filePath)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Downloaded file path does not exist")
	}
}

// TestDownloadFile_UserRequestCategory tests downloading from UserRequestRelated category
func TestDownloadFile_UserRequestCategory(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	service := NewFileService(nil, tmpDir)

	// Test downloading from UserRequestRelated category
	filePath, err := service.DownloadFile("request1.txt")
	if err != nil {
		t.Fatalf("DownloadFile returned error: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "user_requests", "request1.txt")
	if filePath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, filePath)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Downloaded file path does not exist")
	}
}

// TestDownloadFile_NotFound tests behavior when file doesn't exist
func TestDownloadFile_NotFound(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	service := NewFileService(nil, tmpDir)

	// Try to download non-existent file
	filePath, err := service.DownloadFile("nonexistent.txt")

	if err != os.ErrNotExist {
		t.Errorf("Expected os.ErrNotExist, got %v", err)
	}

	if filePath != "" {
		t.Errorf("Expected empty path for non-existent file, got %s", filePath)
	}
}

// TestFileMetadataTracking tests that file metadata is properly tracked
func TestFileMetadataTracking(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	service := NewFileService(nil, tmpDir)
	files, err := service.GetFilesByCategory(AllFiles)

	if err != nil {
		t.Fatalf("GetFilesByCategory returned error: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("Expected at least one file")
	}

	// Verify metadata is tracked for each file
	for _, file := range files {
		// Check ID is set
		if file.ID == "" {
			t.Error("File ID should be tracked")
		}

		// Check Name is set
		if file.Name == "" {
			t.Error("File Name should be tracked")
		}

		// Check Size is tracked
		if file.Size <= 0 {
			t.Error("File Size should be tracked and greater than 0")
		}

		// Check CreatedAt is tracked
		if file.CreatedAt == 0 {
			t.Error("File CreatedAt should be tracked")
		}

		// Check CreatedAt is reasonable (not in the future)
		if file.CreatedAt > time.Now().Add(time.Minute).Unix() {
			t.Error("File CreatedAt should not be in the future")
		}

		// Check Category is tracked
		if file.Category == "" {
			t.Error("File Category should be tracked")
		}

		// Check DownloadURL is tracked
		if file.DownloadURL == "" {
			t.Error("File DownloadURL should be tracked")
		}
	}
}

// TestGetFilesByCategory_IgnoresSubdirectories tests that subdirectories are ignored
func TestGetFilesByCategory_IgnoresSubdirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileservice_subdir_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files directory with a file and a subdirectory
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files dir: %v", err)
	}

	// Create a file
	filePath := filepath.Join(filesDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(filesDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	service := NewFileService(nil, tmpDir)
	files, err := service.GetFilesByCategory(AllFiles)

	if err != nil {
		t.Fatalf("GetFilesByCategory returned error: %v", err)
	}

	// Should only return the file, not the subdirectory
	if len(files) != 1 {
		t.Errorf("Expected 1 file (subdirectories should be ignored), got %d", len(files))
	}

	if len(files) > 0 && files[0].Name != "test.txt" {
		t.Errorf("Expected file name 'test.txt', got '%s'", files[0].Name)
	}
}
