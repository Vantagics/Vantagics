package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// saveChartDataToFile saves large chart data to a separate file to avoid truncation issues
// Returns the file reference string (e.g., "file://echarts_1234567890.json") or error
func (a *App) saveChartDataToFile(threadID, chartType, data string) (string, error) {
	// Only save to file if data is large enough to potentially cause issues
	const sizeThreshold = 10 * 1024 // 10KB
	if len(data) < sizeThreshold {
		// Return empty string to indicate inline storage should be used
		return "", nil
	}

	// Get session files directory
	sessionDir := a.chatService.GetSessionFilesDirectory(threadID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create session files directory: %w", err)
	}

	// Create unique filename with timestamp
	timestamp := time.Now().UnixMilli()
	filename := fmt.Sprintf("%s_%d.json", chartType, timestamp)
	filepath := filepath.Join(sessionDir, filename)

	// Save data to file
	if err := os.WriteFile(filepath, []byte(data), 0644); err != nil {
		return "", fmt.Errorf("failed to write chart data to file: %w", err)
	}

	// Log the file save
	a.Log(fmt.Sprintf("[CHART-FILE] Saved %s data to file: %s (%d bytes)", chartType, filename, len(data)))

	// Return file reference
	fileRef := fmt.Sprintf("file://%s", filename)
	return fileRef, nil
}

// ReadChartDataFile reads chart data from a file reference
// Input: threadID and file reference string (e.g., "file://echarts_1234567890.json")
// Returns: the chart data as a string
func (a *App) ReadChartDataFile(threadID, fileRef string) (string, error) {
	// Parse the file reference to extract filename
	const prefix = "file://"
	if len(fileRef) < len(prefix) || fileRef[:len(prefix)] != prefix {
		return "", fmt.Errorf("invalid file reference format: %s", fileRef)
	}
	
	filename := fileRef[len(prefix):]
	
	// Validate filename to prevent directory traversal attacks
	if filepath.Base(filename) != filename {
		return "", fmt.Errorf("invalid filename: contains path separators")
	}
	
	// Get session files directory
	sessionDir := a.chatService.GetSessionFilesDirectory(threadID)
	
	// Build full file path
	fullPath := filepath.Join(sessionDir, filename)
	
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("chart data file not found: %s", filename)
	}
	
	// Read file content
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read chart data file: %w", err)
	}
	
	a.Log(fmt.Sprintf("[CHART-FILE] Read %s from file: %s (%d bytes)", filename, fullPath, len(data)))
	
	return string(data), nil
}
