package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger handles application logging
type Logger struct {
	file *os.File
	mu   sync.Mutex
}

// NewLogger creates a new Logger instance
func NewLogger() *Logger {
	return &Logger{}
}

// Init initializes the logging to a file in the specified directory
func (l *Logger) Init(logDir string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
	}

	dateStr := time.Now().Format("2006-01-02")
	pattern := filepath.Join(logDir, fmt.Sprintf("rapidbi_%s_*.log", dateStr))
	matches, _ := filepath.Glob(pattern)
	runCount := len(matches) + 1
	filename := filepath.Join(logDir, fmt.Sprintf("rapidbi_%s_%d.log", dateStr, runCount))

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	l.file = f
	l.logInternal("App Started")
	return nil
}

// Log writes a message to the log file
func (l *Logger) Log(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logInternal(message)
}

// Logf writes a formatted message to the log file
func (l *Logger) Logf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logInternal(fmt.Sprintf(format, args...))
}

func (l *Logger) logInternal(message string) {
	if l.file == nil {
		return
	}
	timestamp := time.Now().Format("15:04:05.000")
	fmt.Fprintf(l.file, "[%s] %s\n", timestamp, message)
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
