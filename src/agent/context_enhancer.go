package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// AnalysisRecord 分析记录
// Represents a single analysis record for historical context
// Validates: Requirements 1.1
type AnalysisRecord struct {
	ID            string    `json:"id"`              // Unique identifier (format: ah_timestamp)
	DataSourceID  string    `json:"data_source_id"`  // ID of the data source analyzed
	AnalysisType  string    `json:"analysis_type"`   // Type: trend, comparison, distribution, etc.
	TargetColumns []string  `json:"target_columns"`  // Columns involved in the analysis
	KeyFindings   string    `json:"key_findings"`    // Summary of key findings
	Timestamp     time.Time `json:"timestamp"`       // When the analysis was performed
}

// AnalysisHistoryFile represents the JSON file structure for analysis history
type AnalysisHistoryFile struct {
	Records []AnalysisRecord `json:"records"`
}

// AnalysisHistoryStore manages the storage and retrieval of analysis records
// Thread-safe implementation with JSON file persistence
type AnalysisHistoryStore struct {
	dataDir  string           // Directory for storing the JSON file
	filePath string           // Full path to analysis_history.json
	records  []AnalysisRecord // In-memory cache of records
	mu       sync.RWMutex     // Mutex for thread-safe operations
	loaded   bool             // Whether records have been loaded from file
}

// NewAnalysisHistoryStore creates a new AnalysisHistoryStore
// Parameters:
//   - dataDir: the directory where analysis_history.json will be stored
//
// Returns a new AnalysisHistoryStore instance
func NewAnalysisHistoryStore(dataDir string) *AnalysisHistoryStore {
	return &AnalysisHistoryStore{
		dataDir:  dataDir,
		filePath: filepath.Join(dataDir, "analysis_history.json"),
		records:  make([]AnalysisRecord, 0),
		loaded:   false,
	}
}

// Load loads analysis records from the JSON file
// Thread-safe operation that reads from disk and caches in memory
// Returns error if file exists but cannot be read or parsed
// If file doesn't exist, initializes with empty records (not an error)
func (s *AnalysisHistoryStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.loadUnsafe()
}

// loadUnsafe loads records without acquiring the lock (caller must hold lock)
func (s *AnalysisHistoryStore) loadUnsafe() error {
	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// File doesn't exist, initialize with empty records
		s.records = make([]AnalysisRecord, 0)
		s.loaded = true
		return nil
	}

	// Read file content
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read analysis history file: %w", err)
	}

	// Handle empty file
	if len(data) == 0 {
		s.records = make([]AnalysisRecord, 0)
		s.loaded = true
		return nil
	}

	// Parse JSON
	var historyFile AnalysisHistoryFile
	if err := json.Unmarshal(data, &historyFile); err != nil {
		return fmt.Errorf("failed to parse analysis history file: %w", err)
	}

	s.records = historyFile.Records
	if s.records == nil {
		s.records = make([]AnalysisRecord, 0)
	}
	s.loaded = true

	return nil
}

// Save saves all analysis records to the JSON file
// Thread-safe operation that writes the in-memory cache to disk
// Creates the data directory if it doesn't exist
func (s *AnalysisHistoryStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveUnsafe()
}

// saveUnsafe saves records without acquiring the lock (caller must hold lock)
func (s *AnalysisHistoryStore) saveUnsafe() error {
	// Ensure data directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create history file structure
	historyFile := AnalysisHistoryFile{
		Records: s.records,
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(historyFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analysis history: %w", err)
	}

	// Write to file
	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write analysis history file: %w", err)
	}

	return nil
}

// AddRecord adds a new analysis record to the store
// Thread-safe operation that adds to memory and persists to disk
// Generates a unique ID if not provided
// Validates: Requirements 1.1
func (s *AnalysisHistoryStore) AddRecord(record AnalysisRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure records are loaded
	if !s.loaded {
		if err := s.loadUnsafe(); err != nil {
			return fmt.Errorf("failed to load records before adding: %w", err)
		}
	}

	// Generate ID if not provided
	if record.ID == "" {
		record.ID = fmt.Sprintf("ah_%d", time.Now().UnixNano())
	}

	// Set timestamp if not provided
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	// Add record to the beginning (newest first)
	s.records = append([]AnalysisRecord{record}, s.records...)

	// Save to file
	return s.saveUnsafe()
}

// GetRecordsByDataSource retrieves analysis records for a specific data source
// Thread-safe operation that returns records sorted by timestamp (newest first)
// Parameters:
//   - dataSourceID: the ID of the data source to filter by
//   - maxRecords: maximum number of records to return (0 for all)
//
// Returns records sorted by timestamp in descending order (newest first)
// Validates: Requirements 1.2, 1.4
func (s *AnalysisHistoryStore) GetRecordsByDataSource(dataSourceID string, maxRecords int) ([]AnalysisRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure records are loaded (need write lock for this)
	if !s.loaded {
		s.mu.RUnlock()
		s.mu.Lock()
		if !s.loaded {
			if err := s.loadUnsafe(); err != nil {
				s.mu.Unlock()
				return nil, fmt.Errorf("failed to load records: %w", err)
			}
		}
		s.mu.Unlock()
		s.mu.RLock()
	}

	// Filter records by data source ID
	// Initialize with empty slice to ensure we never return nil
	filtered := make([]AnalysisRecord, 0)
	for _, record := range s.records {
		if record.DataSourceID == dataSourceID {
			filtered = append(filtered, record)
		}
	}

	// Sort by timestamp descending (newest first)
	// Validates: Requirements 1.4
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// Limit to maxRecords if specified
	// Validates: Requirements 1.2 (max 10 records)
	if maxRecords > 0 && len(filtered) > maxRecords {
		filtered = filtered[:maxRecords]
	}

	return filtered, nil
}

// GetAllRecords retrieves all analysis records
// Thread-safe operation that returns all records sorted by timestamp (newest first)
func (s *AnalysisHistoryStore) GetAllRecords() ([]AnalysisRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure records are loaded
	if !s.loaded {
		s.mu.RUnlock()
		s.mu.Lock()
		if !s.loaded {
			if err := s.loadUnsafe(); err != nil {
				s.mu.Unlock()
				return nil, fmt.Errorf("failed to load records: %w", err)
			}
		}
		s.mu.Unlock()
		s.mu.RLock()
	}

	// Return a copy to prevent external modification
	result := make([]AnalysisRecord, len(s.records))
	copy(result, s.records)

	// Sort by timestamp descending (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result, nil
}

// DeleteRecord deletes an analysis record by ID
// Thread-safe operation that removes from memory and persists to disk
func (s *AnalysisHistoryStore) DeleteRecord(recordID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure records are loaded
	if !s.loaded {
		if err := s.loadUnsafe(); err != nil {
			return fmt.Errorf("failed to load records before deleting: %w", err)
		}
	}

	// Find and remove the record
	found := false
	newRecords := make([]AnalysisRecord, 0, len(s.records))
	for _, record := range s.records {
		if record.ID != recordID {
			newRecords = append(newRecords, record)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("record with ID %s not found", recordID)
	}

	s.records = newRecords

	// Save to file
	return s.saveUnsafe()
}

// Clear removes all analysis records
// Thread-safe operation that clears memory and persists empty state to disk
func (s *AnalysisHistoryStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = make([]AnalysisRecord, 0)
	s.loaded = true

	return s.saveUnsafe()
}

// Count returns the total number of analysis records
// Thread-safe operation
func (s *AnalysisHistoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.records)
}

// CountByDataSource returns the number of records for a specific data source
// Thread-safe operation
func (s *AnalysisHistoryStore) CountByDataSource(dataSourceID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, record := range s.records {
		if record.DataSourceID == dataSourceID {
			count++
		}
	}
	return count
}

// GetFilePath returns the path to the JSON storage file
func (s *AnalysisHistoryStore) GetFilePath() string {
	return s.filePath
}

// IsLoaded returns whether the records have been loaded from file
func (s *AnalysisHistoryStore) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}
