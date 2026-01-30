package logger

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Logger handles application logging with automatic compression
type Logger struct {
	file            *os.File
	mu              sync.Mutex
	logDir          string
	filename        string
	maxSizeMB       int64 // Maximum log file size in MB before compression
	maxArchiveCount int   // Maximum number of archived zip files to keep
}

// NewLogger creates a new Logger instance
func NewLogger() *Logger {
	return &Logger{
		maxSizeMB:       100, // Default 100MB
		maxArchiveCount: 10,  // Keep last 10 archives by default
	}
}

// SetMaxSizeMB sets the maximum log file size in MB
func (l *Logger) SetMaxSizeMB(sizeMB int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if sizeMB > 0 {
		l.maxSizeMB = int64(sizeMB)
	}
}

// SetMaxArchiveCount sets the maximum number of archived zip files to keep
func (l *Logger) SetMaxArchiveCount(count int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if count > 0 {
		l.maxArchiveCount = count
	}
}

// GetLogDir returns the current log directory
func (l *Logger) GetLogDir() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logDir
}

// SetLogDir sets the log directory without enabling logging
// This is used for log management (compression, cleanup) when detailed logging is disabled
func (l *Logger) SetLogDir(baseDir string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	actualLogDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(actualLogDir, 0755); err != nil {
		fmt.Printf("[Logger] Failed to create logs directory: %v\n", err)
		return
	}
	
	l.logDir = actualLogDir
	
	// Compress any existing large log files
	l.compressExistingLogs()
	
	// Clean up old archives
	l.cleanupOldArchives()
}

// Init initializes the logging to a file in the specified directory
func (l *Logger) Init(logDir string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
	}

	actualLogDir := filepath.Join(logDir, "logs")
	if err := os.MkdirAll(actualLogDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %v", err)
	}

	l.logDir = actualLogDir

	// Compress any existing large log files on startup
	l.compressExistingLogs()

	// Clean up old archives
	l.cleanupOldArchives()

	dateStr := time.Now().Format("2006-01-02")
	pattern := filepath.Join(actualLogDir, fmt.Sprintf("vantagedata_%s_*.log", dateStr))
	matches, _ := filepath.Glob(pattern)
	runCount := len(matches) + 1
	filename := filepath.Join(actualLogDir, fmt.Sprintf("vantagedata_%s_%d.log", dateStr, runCount))

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	l.file = f
	l.filename = filename
	l.logInternal("App Started")
	return nil
}

// Log writes a message to the log file
func (l *Logger) Log(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logInternal(message)
	l.checkAndRotate()
}

// Logf writes a formatted message to the log file
func (l *Logger) Logf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logInternal(fmt.Sprintf(format, args...))
	l.checkAndRotate()
}

func (l *Logger) logInternal(message string) {
	if l.file == nil {
		return
	}
	timestamp := time.Now().Format("15:04:05.000")
	fmt.Fprintf(l.file, "[%s] %s\n", timestamp, message)
}

// checkAndRotate checks if the log file exceeds the size limit and rotates if needed
func (l *Logger) checkAndRotate() {
	if l.file == nil || l.maxSizeMB <= 0 {
		return
	}

	info, err := l.file.Stat()
	if err != nil {
		return
	}

	maxBytes := l.maxSizeMB * 1024 * 1024
	if info.Size() < maxBytes {
		return
	}

	// Close current file
	l.logInternal("Log file size limit reached, compressing...")
	l.file.Close()

	// Compress the log file
	if err := l.compressLogFile(l.filename); err != nil {
		fmt.Printf("Failed to compress log file: %v\n", err)
	}

	// Create new log file
	dateStr := time.Now().Format("2006-01-02")
	timeStr := time.Now().Format("150405")
	newFilename := filepath.Join(l.logDir, fmt.Sprintf("vantagedata_%s_%s.log", dateStr, timeStr))

	f, err := os.OpenFile(newFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to create new log file: %v\n", err)
		return
	}

	l.file = f
	l.filename = newFilename
	l.logInternal("New log file created after compression")
}

// compressLogFile compresses the log file to a zip archive
func (l *Logger) compressLogFile(logPath string) error {
	// Create zip filename with date-time
	dateTimeStr := time.Now().Format("2006-01-02_150405")
	baseName := filepath.Base(logPath)
	zipPath := filepath.Join(l.logDir, fmt.Sprintf("%s_%s.zip", baseName[:len(baseName)-4], dateTimeStr))

	// Create zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Open log file for reading
	logFile, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open log file for compression: %v", err)
	}
	defer logFile.Close()

	// Get file info for the zip header
	info, err := logFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get log file info: %v", err)
	}

	// Create zip header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create zip header: %v", err)
	}
	header.Method = zip.Deflate

	// Create writer for the file in the zip
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %v", err)
	}

	// Copy log content to zip
	if _, err := io.Copy(writer, logFile); err != nil {
		return fmt.Errorf("failed to write log to zip: %v", err)
	}

	// Close the log file before removing
	logFile.Close()

	// Remove original log file
	if err := os.Remove(logPath); err != nil {
		return fmt.Errorf("failed to remove original log file: %v", err)
	}

	return nil
}

// Close closes the log file
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		l.logInternal("Logging disabled or App stopped.")
		l.file.Close()
		l.file = nil
	}
}

// compressExistingLogs compresses any existing log files that exceed the size limit
func (l *Logger) compressExistingLogs() {
	if l.logDir == "" {
		return
	}

	maxBytes := l.maxSizeMB * 1024 * 1024
	pattern := filepath.Join(l.logDir, "*.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	for _, logPath := range matches {
		info, err := os.Stat(logPath)
		if err != nil {
			continue
		}

		if info.Size() >= maxBytes {
			fmt.Printf("[Logger] Compressing oversized log file: %s (%d MB)\n", logPath, info.Size()/(1024*1024))
			if err := l.compressLogFile(logPath); err != nil {
				fmt.Printf("[Logger] Failed to compress %s: %v\n", logPath, err)
			}
		}
	}
}

// cleanupOldArchives removes old archive files, keeping only the most recent ones
func (l *Logger) cleanupOldArchives() {
	if l.logDir == "" || l.maxArchiveCount <= 0 {
		return
	}

	pattern := filepath.Join(l.logDir, "*.zip")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) <= l.maxArchiveCount {
		return
	}

	// Sort by modification time (oldest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}
	files := make([]fileInfo, 0, len(matches))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: path, modTime: info.ModTime()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	// Remove oldest files
	toRemove := len(files) - l.maxArchiveCount
	for i := 0; i < toRemove; i++ {
		fmt.Printf("[Logger] Removing old archive: %s\n", files[i].path)
		os.Remove(files[i].path)
	}
}

// GetLogStats returns statistics about log files
func (l *Logger) GetLogStats() (totalSizeMB float64, logCount int, archiveCount int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logDir == "" {
		return 0, 0, 0, fmt.Errorf("log directory not initialized")
	}

	// Count log files
	logPattern := filepath.Join(l.logDir, "*.log")
	logMatches, _ := filepath.Glob(logPattern)
	logCount = len(logMatches)

	// Count archive files
	zipPattern := filepath.Join(l.logDir, "*.zip")
	zipMatches, _ := filepath.Glob(zipPattern)
	archiveCount = len(zipMatches)

	// Calculate total size
	var totalSize int64
	allFiles := append(logMatches, zipMatches...)
	for _, path := range allFiles {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		totalSize += info.Size()
	}

	totalSizeMB = float64(totalSize) / (1024 * 1024)
	return totalSizeMB, logCount, archiveCount, nil
}

// CleanupAllLogs compresses all log files and removes old archives
func (l *Logger) CleanupAllLogs() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logDir == "" {
		return fmt.Errorf("log directory not initialized")
	}

	// Close current file temporarily
	currentFile := l.file
	currentFilename := l.filename
	if currentFile != nil {
		l.logInternal("Starting log cleanup...")
		currentFile.Close()
		l.file = nil
	}

	// Compress all log files
	pattern := filepath.Join(l.logDir, "*.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list log files: %v", err)
	}

	for _, logPath := range matches {
		info, err := os.Stat(logPath)
		if err != nil {
			continue
		}
		// Compress files larger than 1MB
		if info.Size() > 1024*1024 {
			if err := l.compressLogFile(logPath); err != nil {
				fmt.Printf("[Logger] Failed to compress %s: %v\n", logPath, err)
			}
		}
	}

	// Clean up old archives
	l.cleanupOldArchives()

	// Reopen log file
	if currentFilename != "" {
		// Create new log file
		dateStr := time.Now().Format("2006-01-02")
		timeStr := time.Now().Format("150405")
		newFilename := filepath.Join(l.logDir, fmt.Sprintf("vantagedata_%s_%s.log", dateStr, timeStr))

		f, err := os.OpenFile(newFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to reopen log file: %v", err)
		}
		l.file = f
		l.filename = newFilename
		l.logInternal("Log cleanup completed, new log file created")
	}

	return nil
}
