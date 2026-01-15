package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ChartItem represents a single chart/visualization item
type ChartItem struct {
	Type string `json:"type"` // "echarts", "image", "table", "csv"
	Data string `json:"data"` // JSON string or base64/data URL
}

// ChartData represents chart/visualization data associated with a message (supports multiple charts)
type ChartData struct {
	Charts []ChartItem `json:"charts"` // Array of chart items
}

// UnmarshalJSON implements custom unmarshaling to handle both new (Charts array) and old (flat Type/Data) formats
func (c *ChartData) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as new format
	type Alias ChartData
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err == nil && len(c.Charts) > 0 {
		// Clean any function definitions from chart data
		for i := range c.Charts {
			c.Charts[i].Data = cleanChartData(c.Charts[i].Data)
		}
		return nil
	}

	// Try to unmarshal as old format (flat)
	var old struct {
		Type string `json:"type"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(data, &old); err == nil && old.Type != "" {
		// Clean any function definitions from chart data
		cleanedData := cleanChartData(old.Data)
		c.Charts = []ChartItem{{Type: old.Type, Data: cleanedData}}
		return nil
	}
	
	return nil
}

// cleanChartData removes JavaScript functions from chart data to prevent JSON parsing errors
func cleanChartData(data string) string {
	if data == "" {
		return data
	}
	
	// Check if data contains function definitions
	if !strings.Contains(data, "function(") && !strings.Contains(data, "function ") {
		return data
	}
	
	// Remove common function patterns that might appear in ECharts configs
	cleanedData := data
	
	// Remove formatter functions
	cleanedData = regexp.MustCompile(`(?s),?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}`).ReplaceAllString(cleanedData, "")
	
	// Remove matter functions
	cleanedData = regexp.MustCompile(`(?s),?\s*"?matter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}`).ReplaceAllString(cleanedData, "")
	
	// Remove any other function properties
	cleanedData = regexp.MustCompile(`(?s),?\s*[a-zA-Z_$][a-zA-Z0-9_$]*\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}`).ReplaceAllString(cleanedData, "")
	
	// Clean up any trailing commas that might be left
	cleanedData = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(cleanedData, "$1")
	
	// Clean up any leading commas
	cleanedData = regexp.MustCompile(`(\{\s*),`).ReplaceAllString(cleanedData, "$1")
	
	return cleanedData
}

// ChatMessage represents a single message in a chat thread
type ChatMessage struct {
	ID           string                 `json:"id"`
	Role         string                 `json:"role"` // "user" or "assistant"
	Content      string                 `json:"content"`
	Timestamp    int64                  `json:"timestamp"`
	ChartData    *ChartData             `json:"chart_data,omitempty"`    // Associated chart/visualization data (can contain multiple charts)
	TimingData   map[string]interface{} `json:"timing_data,omitempty"`   // Detailed timing information for analysis stages
}

// UnmarshalJSON implements custom unmarshaling to handle both new (int64) and old (time.Time string) formats
func (cm *ChatMessage) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with Timestamp as interface{} to handle both formats
	type Alias ChatMessage
	aux := &struct {
		Timestamp interface{} `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(cm),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle Timestamp field conversion
	switch v := aux.Timestamp.(type) {
	case float64:
		// Already int64 (JSON numbers are float64)
		cm.Timestamp = int64(v)
	case int64:
		// Direct int64
		cm.Timestamp = v
	case int:
		// Convert int to int64
		cm.Timestamp = int64(v)
	case string:
		// Old time.Time format - parse and convert to Unix milliseconds
		if v == "" {
			cm.Timestamp = time.Now().UnixMilli()
		} else if t, err := time.Parse(time.RFC3339, v); err == nil {
			cm.Timestamp = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", v); err == nil {
			cm.Timestamp = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05Z07:00", v); err == nil {
			cm.Timestamp = t.UnixMilli()
		} else {
			// If parsing fails, use current time
			cm.Timestamp = time.Now().UnixMilli()
		}
	case nil:
		// No Timestamp field - use current time
		cm.Timestamp = time.Now().UnixMilli()
	default:
		// Safe fallback
		cm.Timestamp = time.Now().UnixMilli()
	}

	return nil
}

// SessionFile represents a file generated during the session
type SessionFile struct {
	Name      string `json:"name"`        // File name (e.g., "chart.png", "result.csv")
	Path      string `json:"path"`        // Relative path within session directory
	Type      string `json:"type"`        // "image", "csv", "data", etc.
	Size      int64  `json:"size"`        // File size in bytes
	CreatedAt int64  `json:"created_at"`  // Unix timestamp
	MessageID string `json:"message_id,omitempty"` // Associated message ID (optional)
}

// UnmarshalJSON implements custom unmarshaling to handle both new (int64) and old (time.Time string) formats
func (sf *SessionFile) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with CreatedAt as interface{} to handle both formats
	type Alias SessionFile
	aux := &struct {
		CreatedAt interface{} `json:"created_at"`
		*Alias
	}{
		Alias: (*Alias)(sf),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle CreatedAt field conversion
	switch v := aux.CreatedAt.(type) {
	case float64:
		// Already int64 (JSON numbers are float64)
		sf.CreatedAt = int64(v)
	case int64:
		// Direct int64
		sf.CreatedAt = v
	case int:
		// Convert int to int64
		sf.CreatedAt = int64(v)
	case string:
		// Old time.Time format - parse and convert to Unix milliseconds
		if v == "" {
			sf.CreatedAt = time.Now().UnixMilli()
		} else if t, err := time.Parse(time.RFC3339, v); err == nil {
			sf.CreatedAt = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", v); err == nil {
			sf.CreatedAt = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05Z07:00", v); err == nil {
			sf.CreatedAt = t.UnixMilli()
		} else {
			// If parsing fails, use current time
			sf.CreatedAt = time.Now().UnixMilli()
		}
	case nil:
		// No CreatedAt field - use current time
		sf.CreatedAt = time.Now().UnixMilli()
	default:
		// Safe fallback
		sf.CreatedAt = time.Now().UnixMilli()
	}

	return nil
}

// ChatThread represents a conversation thread
type ChatThread struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	DataSourceID string        `json:"data_source_id"` // Associated Data Source ID
	CreatedAt    int64         `json:"created_at"`
	Messages     []ChatMessage `json:"messages"`
	Files        []SessionFile `json:"files,omitempty"` // Generated files during session
}

// UnmarshalJSON implements custom unmarshaling to handle both new (int64) and old (time.Time string) formats
func (ct *ChatThread) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with CreatedAt as interface{} to handle both formats
	type Alias ChatThread
	aux := &struct {
		CreatedAt interface{} `json:"created_at"`
		*Alias
	}{
		Alias: (*Alias)(ct),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle CreatedAt field conversion
	switch v := aux.CreatedAt.(type) {
	case float64:
		// Already int64 (JSON numbers are float64)
		ct.CreatedAt = int64(v)
	case int64:
		// Direct int64
		ct.CreatedAt = v
	case int:
		// Convert int to int64
		ct.CreatedAt = int64(v)
	case string:
		// Old time.Time format - parse and convert to Unix milliseconds
		if v == "" {
			ct.CreatedAt = time.Now().UnixMilli()
		} else if t, err := time.Parse(time.RFC3339, v); err == nil {
			ct.CreatedAt = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", v); err == nil {
			ct.CreatedAt = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05Z07:00", v); err == nil {
			ct.CreatedAt = t.UnixMilli()
		} else {
			// If parsing fails, use current time
			ct.CreatedAt = time.Now().UnixMilli()
		}
	case nil:
		// No CreatedAt field - use current time
		ct.CreatedAt = time.Now().UnixMilli()
	default:
		// Safe fallback
		ct.CreatedAt = time.Now().UnixMilli()
	}

	return nil
}

// ChatService handles the persistence of chat history
type ChatService struct {
	sessionsDir string
	mu          sync.Mutex
}

// NewChatService creates a new instance of ChatService
func NewChatService(sessionsDir string) *ChatService {
	_ = os.MkdirAll(sessionsDir, 0755)
	return &ChatService{
		sessionsDir: sessionsDir,
	}
}

// getThreadPath returns the path to the history file for a given thread
func (s *ChatService) getThreadPath(threadID string) string {
	return filepath.Join(s.sessionsDir, threadID, "history.json")
}

// LoadThreads loads all chat threads from storage
func (s *ChatService) LoadThreads() ([]ChatThread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.sessionsDir)
	if err != nil {
		return []ChatThread{}, nil
	}

	var threads []ChatThread
	for _, entry := range entries {
		if entry.IsDir() {
			threadID := entry.Name()
			path := s.getThreadPath(threadID)
			
			if _, err := os.Stat(path); err == nil {
				data, err := os.ReadFile(path)
				if err == nil {
					var t ChatThread
					if err := json.Unmarshal(data, &t); err == nil {
						threads = append(threads, t)
					}
				}
			}
		}
	}

	// Sort by CreatedAt descending (newest first)
	// Or preserve order if important? Usually UI sorts.
	// Let's sort to be consistent with previous behavior (append prepend).
	// Previous behavior was prepend new threads.
	// We can sort by CreatedAt desc here.
	for i := 0; i < len(threads); i++ {
		for j := i + 1; j < len(threads); j++ {
			if threads[i].CreatedAt < threads[j].CreatedAt {
				threads[i], threads[j] = threads[j], threads[i]
			}
		}
	}

	return threads, nil
}

// GetThreadsByDataSource loads chat threads filtered by data source ID
func (s *ChatService) GetThreadsByDataSource(dataSourceID string) ([]ChatThread, error) {
	threads, err := s.LoadThreads()
	if err != nil {
		return nil, err
	}

	filtered := []ChatThread{}
	for _, t := range threads {
		if t.DataSourceID == dataSourceID {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

// CheckSessionNameExists checks if a thread with the same title already exists for a specific data source
func (s *ChatService) CheckSessionNameExists(dataSourceID string, title string) (bool, error) {
	threads, err := s.LoadThreads()
	if err != nil {
		return false, err
	}

	for _, t := range threads {
		if t.DataSourceID == dataSourceID && strings.EqualFold(t.Title, title) {
			return true, nil
		}
	}
	return false, nil
}

// SaveThreads saves the given list of threads to storage
func (s *ChatService) SaveThreads(threads []ChatThread) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range threads {
		if err := s.saveThreadInternal(t); err != nil {
			return err
		}
	}
	return nil
}

// saveThreadInternal saves a single thread (assumes lock held)
func (s *ChatService) saveThreadInternal(t ChatThread) error {
	path := s.getThreadPath(t.ID)
	dir := filepath.Dir(path)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// AddMessage adds a message to a specific thread
func (s *ChatService) AddMessage(threadID string, msg ChatMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load specific thread
	path := s.getThreadPath(threadID)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var t ChatThread
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	t.Messages = append(t.Messages, msg)
	return s.saveThreadInternal(t)
}

// generateUniqueTitle ensures a title is unique within a data source
func (s *ChatService) generateUniqueTitle(dataSourceID, title, excludeThreadID string) (string, error) {
	// We need to check against ALL threads to ensure uniqueness
	// This calls LoadThreads which acquires lock, so we must be careful if we already hold lock?
	// generateUniqueTitle is internal helper, usually we hold lock.
	// But LoadThreads acquires lock. Deadlock hazard.
	// Let's make LoadThreadsInternal or just duplicate logic.
	
	// Since we are refactoring, let's just ReadDir here manually or assume LoadThreads is safe if we don't hold lock 
	// before calling generateUniqueTitle.
	// However, usually generateUniqueTitle is called inside CreateThread which holds lock.
	// So we need `loadThreadsInternal`.
	
	threads, err := s.loadThreadsInternal()
	if err != nil {
		return "", err
	}

	existingTitles := make(map[string]bool)
	for _, t := range threads {
		if t.DataSourceID == dataSourceID && t.ID != excludeThreadID {
			existingTitles[t.Title] = true
		}
	}

	newTitle := title
	counter := 1
	for existingTitles[newTitle] {
		newTitle = fmt.Sprintf("%s (%d)", title, counter)
		counter++
	}
	return newTitle, nil
}

// loadThreadsInternal loads threads without locking
func (s *ChatService) loadThreadsInternal() ([]ChatThread, error) {
	entries, err := os.ReadDir(s.sessionsDir)
	if err != nil {
		return []ChatThread{}, nil
	}

	var threads []ChatThread
	for _, entry := range entries {
		if entry.IsDir() {
			threadID := entry.Name()
			path := s.getThreadPath(threadID)
			
			if _, err := os.Stat(path); err == nil {
				data, err := os.ReadFile(path)
				if err == nil {
					var t ChatThread
					if err := json.Unmarshal(data, &t); err == nil {
						threads = append(threads, t)
					}
				}
			}
		}
	}
	return threads, nil
}

// CreateThread creates a new chat thread with a unique title
func (s *ChatService) CreateThread(dataSourceID, title string) (ChatThread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	uniqueTitle, err := s.generateUniqueTitle(dataSourceID, title, "")
	if err != nil {
		return ChatThread{}, err
	}

	newThread := ChatThread{
		ID:           strconv.FormatInt(time.Now().UnixNano(), 10),
		Title:        uniqueTitle,
		DataSourceID: dataSourceID,
		CreatedAt:    time.Now().Unix(),
		Messages:     []ChatMessage{},
	}

	if err := s.saveThreadInternal(newThread); err != nil {
		return ChatThread{}, err
	}

	return newThread, nil
}

// UpdateThreadTitle updates a thread's title, ensuring uniqueness
func (s *ChatService) UpdateThreadTitle(threadID, newTitle string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load specific thread to get DataSourceID
	path := s.getThreadPath(threadID)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var t ChatThread
	if err := json.Unmarshal(data, &t); err != nil {
		return "", err
	}

	uniqueTitle, err := s.generateUniqueTitle(t.DataSourceID, newTitle, threadID)
	if err != nil {
		return "", err
	}

	t.Title = uniqueTitle
	if err := s.saveThreadInternal(t); err != nil {
		return "", err
	}

	return uniqueTitle, nil
}

// DeleteThread deletes a thread by ID
func (s *ChatService) DeleteThread(threadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.sessionsDir, threadID)
	return os.RemoveAll(dir)
}

// ClearHistory deletes all chat history
func (s *ChatService) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.RemoveAll(s.sessionsDir)
}

// GetSessionDirectory returns the directory path for a specific session
func (s *ChatService) GetSessionDirectory(threadID string) string {
	return filepath.Join(s.sessionsDir, threadID)
}

// GetSessionFilesDirectory returns the files directory path for a specific session
func (s *ChatService) GetSessionFilesDirectory(threadID string) string {
	return filepath.Join(s.sessionsDir, threadID, "files")
}

// AddSessionFile registers a file generated during the session
func (s *ChatService) AddSessionFile(threadID string, file SessionFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load specific thread
	path := s.getThreadPath(threadID)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var t ChatThread
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	// Add file to the list
	t.Files = append(t.Files, file)
	return s.saveThreadInternal(t)
}

// GetSessionFiles returns all files for a specific session
func (s *ChatService) GetSessionFiles(threadID string) ([]SessionFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.getThreadPath(threadID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var t ChatThread
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}

	return t.Files, nil
}


