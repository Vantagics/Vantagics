package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
// Note: AnalysisResultItem is defined in event_aggregator.go
type ChatMessage struct {
	ID              string                 `json:"id"`
	Role            string                 `json:"role"` // "user" or "assistant"
	Content         string                 `json:"content"`
	Timestamp       int64                  `json:"timestamp"`
	ChartData       *ChartData             `json:"chart_data,omitempty"`       // Legacy: Associated chart/visualization data
	TimingData      map[string]interface{} `json:"timing_data,omitempty"`      // Detailed timing information for analysis stages
	AnalysisResults []AnalysisResultItem   `json:"analysis_results,omitempty"` // New unified analysis results
	HasAnalysisData bool                   `json:"has_analysis_data,omitempty"` // Lightweight flag: true if message has analysis_results or chart_data (set by LoadThreads after stripping heavy data)
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
	ID              string        `json:"id"`
	Title           string        `json:"title"`
	DataSourceID    string        `json:"data_source_id"` // Associated Data Source ID
	CreatedAt       int64         `json:"created_at"`
	Messages        []ChatMessage `json:"messages"`
	Files           []SessionFile `json:"files,omitempty"`           // Generated files during session
	IsReplaySession bool          `json:"is_replay_session,omitempty"` // Quick analysis replay session flag
	PackMetadata    *PackMetadata `json:"pack_metadata,omitempty"`     // Quick analysis pack metadata
	QapFilePath     string        `json:"qap_file_path,omitempty"`     // File path of the imported .qap file (for re-execution)
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
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create sessions directory %s: %v\n", sessionsDir, err)
	}
	return &ChatService{
		sessionsDir: sessionsDir,
	}
}

// getThreadPath returns the path to the history file for a given thread
// Validates threadID to prevent path traversal attacks
func (s *ChatService) getThreadPath(threadID string) string {
	return filepath.Join(s.sessionsDir, s.sanitizeThreadID(threadID), "history.json")
}

// sanitizeThreadID sanitizes a threadID to prevent path traversal attacks
// Only allows alphanumeric, hyphens, and underscores
func (s *ChatService) sanitizeThreadID(threadID string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, threadID)
	if safe == "" {
		safe = "invalid"
	}
	return safe
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
						// Set lightweight flag before stripping heavy data,
						// so frontend knows which messages have results without carrying the actual data.
						for i := range t.Messages {
							if len(t.Messages[i].AnalysisResults) > 0 || t.Messages[i].ChartData != nil {
								t.Messages[i].HasAnalysisData = true
							}
							t.Messages[i].AnalysisResults = nil
							t.Messages[i].ChartData = nil
						}
						threads = append(threads, t)
					}
				}
			}
		}
	}

	// Sort by CreatedAt descending (newest first)
	sort.Slice(threads, func(i, j int) bool {
		return threads[i].CreatedAt > threads[j].CreatedAt
	})

	return threads, nil
}

// GetThreadsByDataSource loads chat threads filtered by data source ID
func (s *ChatService) GetThreadsByDataSource(dataSourceID string) ([]ChatThread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	threads, err := s.loadThreadsInternal()
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

// FindReplaySessionByQapFile finds an existing replay session for a given datasource and qap file path.
// Returns the thread if found, nil otherwise.
func (s *ChatService) FindReplaySessionByQapFile(dataSourceID, qapFilePath string) (*ChatThread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	threads, err := s.loadThreadsInternal()
	if err != nil {
		return nil, err
	}

	for _, t := range threads {
		if t.DataSourceID == dataSourceID && t.IsReplaySession && t.QapFilePath == qapFilePath {
			return &t, nil
		}
	}
	return nil, nil
}


// CheckSessionNameExists checks if a thread with the same title already exists for a specific data source
func (s *ChatService) CheckSessionNameExists(dataSourceID string, title string, excludeThreadID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	threads, err := s.loadThreadsInternal()
	if err != nil {
		return false, err
	}

	for _, t := range threads {
		if t.DataSourceID == dataSourceID && strings.EqualFold(t.Title, title) && t.ID != excludeThreadID {
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

// loadThreadsInternal loads threads without locking (strips heavy data like LoadThreads)
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
						// Strip heavy data — this is used internally for title checks etc.
						for i := range t.Messages {
							t.Messages[i].AnalysisResults = nil
							t.Messages[i].ChartData = nil
						}
						threads = append(threads, t)
					}
				}
			}
		}
	}
	return threads, nil
}

// LoadThread loads a single thread by ID with full data (including analysis results)
func (s *ChatService) LoadThread(threadID string) (*ChatThread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.getThreadPath(threadID)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("thread not found: %s", threadID)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read thread: %v", err)
	}

	var t ChatThread
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("failed to parse thread: %v", err)
	}

	return &t, nil
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

	dir := filepath.Join(s.sessionsDir, s.sanitizeThreadID(threadID))
	return os.RemoveAll(dir)
}

// ClearThreadMessages clears all messages from a thread but keeps the thread itself
func (s *ChatService) ClearThreadMessages(threadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.getThreadPath(threadID)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read thread: %w", err)
	}

	var thread ChatThread
	if err := json.Unmarshal(data, &thread); err != nil {
		return fmt.Errorf("failed to parse thread: %w", err)
	}

	thread.Messages = []ChatMessage{}
	thread.Files = []SessionFile{}

	newData, err := json.MarshalIndent(thread, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal thread: %w", err)
	}

	if err := os.WriteFile(path, newData, 0644); err != nil {
		return fmt.Errorf("failed to write thread: %w", err)
	}

	// Clean up analysis results directory
	resultsDir := filepath.Join(s.sessionsDir, s.sanitizeThreadID(threadID), "analysis_results")
	os.RemoveAll(resultsDir)

	// Clean up files directory
	filesDir := filepath.Join(s.sessionsDir, s.sanitizeThreadID(threadID), "files")
	os.RemoveAll(filesDir)

	return nil
}

// ClearHistory deletes all chat history
func (s *ChatService) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Safety check: only delete if the path looks like a sessions directory
	// to prevent accidental deletion of unrelated directories
	if s.sessionsDir == "" || s.sessionsDir == "/" || s.sessionsDir == "\\" {
		return fmt.Errorf("refusing to clear history: sessions directory path is unsafe: %q", s.sessionsDir)
	}
	if !strings.Contains(s.sessionsDir, "sessions") {
		return fmt.Errorf("refusing to clear history: sessions directory path does not contain 'sessions': %q", s.sessionsDir)
	}

	return os.RemoveAll(s.sessionsDir)
}

// GetSessionDirectory returns the directory path for a specific session
func (s *ChatService) GetSessionDirectory(threadID string) string {
	return filepath.Join(s.sessionsDir, s.sanitizeThreadID(threadID))
}

// GetSessionFilesDirectory returns the files directory path for a specific session
func (s *ChatService) GetSessionFilesDirectory(threadID string) string {
	return filepath.Join(s.sessionsDir, s.sanitizeThreadID(threadID), "files")
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



// SaveAnalysisResults saves analysis results for a specific message
func (s *ChatService) SaveAnalysisResults(threadID, messageID string, results []AnalysisResultItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.getThreadPath(threadID)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var t ChatThread
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	// Find the message and update its analysis results
	for i := range t.Messages {
		if t.Messages[i].ID == messageID {
			t.Messages[i].AnalysisResults = results
			return s.saveThreadInternal(t)
		}
	}

	return fmt.Errorf("message not found: %s", messageID)
}

// AppendAnalysisResults appends additional analysis results to a specific message
// without overwriting existing results. Used for late-arriving items like extracted
// metrics and suggestions that are processed after the initial save.
func (s *ChatService) AppendAnalysisResults(threadID, messageID string, newResults []AnalysisResultItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.getThreadPath(threadID)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var t ChatThread
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	// Find the message and append to its analysis results
	for i := range t.Messages {
		if t.Messages[i].ID == messageID {
			t.Messages[i].AnalysisResults = append(t.Messages[i].AnalysisResults, newResults...)
			return s.saveThreadInternal(t)
		}
	}

	return fmt.Errorf("message not found: %s", messageID)
}

// GetAnalysisResults retrieves analysis results for a specific message
func (s *ChatService) GetAnalysisResults(threadID, messageID string) ([]AnalysisResultItem, error) {
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

	// Find the message
	for _, msg := range t.Messages {
		if msg.ID == messageID {
			return msg.AnalysisResults, nil
		}
	}

	return nil, fmt.Errorf("message not found: %s", messageID)
}

// GetMessageAnalysisData retrieves all analysis data for a message (for dashboard restoration)
func (s *ChatService) GetMessageAnalysisData(threadID, messageID string) (map[string]interface{}, error) {
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

	// Find the message
	for i, msg := range t.Messages {
		if msg.ID == messageID {
			analysisResults := msg.AnalysisResults

			// Check which item types are missing from analysis_results.
			// This can happen with historical data where items weren't fully persisted.
			// In that case, parse the next assistant message's content as a fallback.
			hasECharts := false
			hasTable := false
			hasInsight := false
			for _, item := range analysisResults {
				switch item.Type {
				case "echarts":
					hasECharts = true
				case "table":
					hasTable = true
				case "insight":
					hasInsight = true
				}
			}

			if !hasECharts || !hasTable || !hasInsight {
				// Find the next assistant message
				var assistantContent string
				if i+1 < len(t.Messages) && t.Messages[i+1].Role == "assistant" {
					assistantContent = t.Messages[i+1].Content
				}

				if assistantContent != "" {
					extracted := s.extractAnalysisItemsFromContent(assistantContent, threadID, messageID)
					for _, item := range extracted {
						// Only add types that are missing from existing results
						if item.Type == "echarts" && hasECharts {
							continue
						}
						if item.Type == "table" && hasTable {
							continue
						}
						if item.Type == "insight" && hasInsight {
							continue
						}
						analysisResults = append(analysisResults, item)
					}
				}
			}

			result := map[string]interface{}{
				"messageId":       msg.ID,
				"threadId":        threadID,
				"analysisResults": analysisResults,
			}
			// Include legacy chart data if present
			if msg.ChartData != nil {
				result["chartData"] = msg.ChartData
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("message not found: %s", messageID)
}

// extractAnalysisItemsFromContent parses assistant message content for echarts/table blocks
// and returns them as AnalysisResultItem entries. This is used as a fallback when
// analysis_results on disk doesn't contain these items (e.g., historical data).
func (s *ChatService) extractAnalysisItemsFromContent(content, threadID, messageID string) []AnalysisResultItem {
	var items []AnalysisResultItem
	now := time.Now().UnixMilli()
	seq := 0

	// Extract ECharts blocks
	reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	reEChartsNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:echarts\\s*\\n(\\{[\\s\\S]+?\\n\\})(?:\\s*\\n(?:---|###)|\\s*$)")
	allEChartsMatches := reECharts.FindAllStringSubmatch(content, -1)
	allEChartsMatches = append(allEChartsMatches, reEChartsNoBT.FindAllStringSubmatch(content, -1)...)
	for _, match := range allEChartsMatches {
		if len(match) > 1 {
			jsonStr := strings.TrimSpace(match[1])
			// Validate JSON
			var testJSON map[string]interface{}
			if json.Unmarshal([]byte(jsonStr), &testJSON) != nil {
				// Try cleaning JS functions
				cleaned := cleanEChartsJSONSimple(jsonStr)
				if json.Unmarshal([]byte(cleaned), &testJSON) == nil {
					jsonStr = cleaned
				} else {
					continue // Skip unparseable echarts
				}
			}
			seq++
			items = append(items, AnalysisResultItem{
				ID:   fmt.Sprintf("restored_echarts_%s_%d", messageID, seq),
				Type: "echarts",
				Data: jsonStr,
				Metadata: map[string]interface{}{
					"sessionId": threadID,
					"messageId": messageID,
					"timestamp": now,
				},
				Source: "restored",
			})
		}
	}

	// Extract table blocks (json:table format)
	reTable := regexp.MustCompile("(?s)```\\s*json:table\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	// Also match json:table without backticks
	reTableNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:table\\s*\\n((?:\\{[\\s\\S]+?\\n\\}|\\[[\\s\\S]+?\\n\\]))(?:\\s*\\n(?:---|###)|\\s*$)")
	allTableMatchIndices := reTable.FindAllStringSubmatchIndex(content, -1)
	allTableMatchIndices = append(allTableMatchIndices, reTableNoBT.FindAllStringSubmatchIndex(content, -1)...)
	for idx, match := range allTableMatchIndices {
		if len(match) >= 4 {
			fullMatchStart := match[0]
			jsonContent := strings.TrimSpace(content[match[2]:match[3]])

			// Extract table title from the line before the code block
			tableTitle := ""
			if fullMatchStart > 0 {
				textBefore := content[:fullMatchStart]
				lastNewline := strings.LastIndex(textBefore, "\n")
				if lastNewline >= 0 {
					lineBeforeCodeBlock := strings.TrimSpace(textBefore[lastNewline+1:])
					tableTitle = strings.TrimLeft(lineBeforeCodeBlock, "#*- ")
					tableTitle = strings.TrimRight(tableTitle, ":：")
					tableTitle = strings.TrimSpace(tableTitle)
					if strings.HasPrefix(tableTitle, "{") || strings.HasPrefix(tableTitle, "[") || strings.HasPrefix(tableTitle, "```") {
						tableTitle = ""
					}
				}
			}

			// Try to parse as object array
			var tableData []map[string]interface{}
			var columnsOrder []string
			if err := json.Unmarshal([]byte(jsonContent), &tableData); err != nil {
				// Try {columns: [...], data: [[...], ...]} format
				var colDataFormat struct {
					Columns []string        `json:"columns"`
					Data    [][]interface{} `json:"data"`
				}
				if err2 := json.Unmarshal([]byte(jsonContent), &colDataFormat); err2 == nil && len(colDataFormat.Columns) > 0 && len(colDataFormat.Data) > 0 {
					columnsOrder = colDataFormat.Columns
					tableData = make([]map[string]interface{}, 0, len(colDataFormat.Data))
					for _, row := range colDataFormat.Data {
						rowMap := make(map[string]interface{})
						for i, val := range row {
							if i < len(colDataFormat.Columns) {
								rowMap[colDataFormat.Columns[i]] = val
							}
						}
						tableData = append(tableData, rowMap)
					}
				} else {
					// Try 2D array format
					var arrayData [][]interface{}
					if err3 := json.Unmarshal([]byte(jsonContent), &arrayData); err3 == nil && len(arrayData) > 1 {
						headers := make([]string, len(arrayData[0]))
						for i, h := range arrayData[0] {
							headers[i] = fmt.Sprintf("%v", h)
						}
						columnsOrder = headers
						tableData = make([]map[string]interface{}, 0, len(arrayData)-1)
						for _, row := range arrayData[1:] {
							rowMap := make(map[string]interface{})
							for i, val := range row {
								if i < len(headers) {
									rowMap[headers[i]] = val
								}
							}
							tableData = append(tableData, rowMap)
						}
					}
				}
			} else {
				columnsOrder = extractJSONObjectKeysOrdered(jsonContent)
			}

			if len(tableData) > 0 {
				tableDataWithTitle := map[string]interface{}{
					"title":   tableTitle,
					"columns": columnsOrder,
					"rows":    tableData,
				}
				seq++
				items = append(items, AnalysisResultItem{
					ID:   fmt.Sprintf("restored_table_%s_%d_%d", messageID, idx, seq),
					Type: "table",
					Data: tableDataWithTitle,
					Metadata: map[string]interface{}{
						"sessionId": threadID,
						"messageId": messageID,
						"timestamp": now,
					},
					Source: "restored",
				})
			}
		}
	}

	// Extract standard Markdown tables (| col1 | col2 | format)
	mdTables := extractMarkdownTablesFromContent(content)
	for mdIdx, mdTable := range mdTables {
		if len(mdTable.Rows) > 0 {
			tableDataWithTitle := map[string]interface{}{
				"title":   mdTable.Title,
				"columns": mdTable.Columns,
				"rows":    mdTable.Rows,
			}
			seq++
			items = append(items, AnalysisResultItem{
				ID:   fmt.Sprintf("restored_mdtable_%s_%d_%d", messageID, mdIdx, seq),
				Type: "table",
				Data: tableDataWithTitle,
				Metadata: map[string]interface{}{
					"sessionId": threadID,
					"messageId": messageID,
					"timestamp": now,
				},
				Source: "restored",
			})
		}
	}

	// Extract insights from text (suggestions, recommendations, etc.)
	insights := extractSuggestionInsightsFromContent(content)
	for _, insight := range insights {
		seq++
		items = append(items, AnalysisResultItem{
			ID:   fmt.Sprintf("restored_insight_%s_%d", messageID, seq),
			Type: "insight",
			Data: insight,
			Metadata: map[string]interface{}{
				"sessionId": threadID,
				"messageId": messageID,
				"timestamp": now,
			},
			Source: "restored",
		})
	}

	return items
}

// extractSuggestionInsightsFromContent extracts suggestion/insight items from assistant message text.
// This replicates the logic from App.extractSuggestionInsights for use in ChatService.
func extractSuggestionInsightsFromContent(content string) []Insight {
	if content == "" {
		return nil
	}

	var insights []Insight
	lines := strings.Split(content, "\n")

	numberPattern := regexp.MustCompile(`^\s*\*{0,2}(\d+)[.、)]\*{0,2}\s*(.+)`)
	listPattern := regexp.MustCompile(`^\s*[-•]\s+(.+)`)
	boldTitlePattern := regexp.MustCompile(`^\s*\*\*(.+?)\*\*\s*[：:\-–—]\s*(.+)`)
	suggestionPattern := regexp.MustCompile(`(?i)(建议|suggest|recommend|next|further|深入|可以进一步|后续|下一步|洞察|insight|分析方向|可以从|希望从哪)`)

	inCodeBlock := false
	foundSuggestionSection := false
	consecutiveBoldItems := 0

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock || trimmedLine == "" {
			continue
		}

		if suggestionPattern.MatchString(trimmedLine) {
			foundSuggestionSection = true
		}

		var suggestionText string

		// Numbered items (only in suggestion section)
		if matches := numberPattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
			if foundSuggestionSection {
				suggestionText = strings.TrimSpace(matches[2])
			}
		}

		// Bold title with colon/dash
		if suggestionText == "" {
			if matches := boldTitlePattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
				title := strings.TrimSpace(matches[1])
				desc := strings.TrimSpace(matches[2])
				if desc != "" {
					suggestionText = title + "：" + desc
				} else {
					suggestionText = title
				}
				consecutiveBoldItems++
				if consecutiveBoldItems >= 3 {
					foundSuggestionSection = true
				}
			} else {
				consecutiveBoldItems = 0
			}
		}

		// Markdown list items (only in suggestion section)
		if suggestionText == "" && foundSuggestionSection {
			if matches := listPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
				suggestionText = strings.TrimSpace(matches[1])
			}
		}

		// Clean up markdown formatting - remove all ** markers (including incomplete ones in the middle)
		if suggestionText != "" {
			suggestionText = strings.ReplaceAll(suggestionText, "**", "")
			suggestionText = strings.TrimSpace(suggestionText)
		}

		if suggestionText != "" && len([]rune(suggestionText)) > 5 {
			insights = append(insights, Insight{
				Text: suggestionText,
				Icon: "lightbulb",
			})
		}
	}

	if len(insights) > 9 {
		insights = insights[:9]
	}

	return insights
}

// cleanEChartsJSONSimple is a simplified version of cleanEChartsJSON for use in chat_service.
// It removes JavaScript function definitions from ECharts JSON to make it parseable.
func cleanEChartsJSONSimple(jsonStr string) string {
	result := jsonStr

	for {
		idx := strings.Index(result, "function")
		if idx < 0 {
			break
		}

		// Check if preceded by colon (JSON value context)
		prefixStart := idx - 1
		for prefixStart >= 0 && (result[prefixStart] == ' ' || result[prefixStart] == '\t' || result[prefixStart] == '\n' || result[prefixStart] == '\r') {
			prefixStart--
		}
		if prefixStart < 0 || result[prefixStart] != ':' {
			result = result[:idx] + "FUNC_SKIP" + result[idx+8:]
			continue
		}

		// Find opening brace
		braceStart := strings.Index(result[idx:], "{")
		if braceStart < 0 {
			break
		}
		braceStart += idx

		// Find matching closing brace
		depth := 0
		braceEnd := -1
		for i := braceStart; i < len(result); i++ {
			if result[i] == '{' {
				depth++
			} else if result[i] == '}' {
				depth--
				if depth == 0 {
					braceEnd = i
					break
				}
			}
		}
		if braceEnd < 0 {
			break
		}

		// Walk back to find key start
		removeStart := prefixStart
		keyStart := removeStart - 1
		for keyStart >= 0 && (result[keyStart] == ' ' || result[keyStart] == '\t' || result[keyStart] == '\n' || result[keyStart] == '\r') {
			keyStart--
		}
		if keyStart >= 0 && result[keyStart] == '"' {
			keyStart--
			for keyStart >= 0 && result[keyStart] != '"' {
				keyStart--
			}
			if keyStart > 0 {
				keyStart--
				for keyStart >= 0 && (result[keyStart] == ' ' || result[keyStart] == '\t' || result[keyStart] == '\n' || result[keyStart] == '\r') {
					keyStart--
				}
				if keyStart >= 0 && result[keyStart] == ',' {
					removeStart = keyStart
				} else {
					removeStart = keyStart + 1
				}
			}
		}

		after := result[braceEnd+1:]
		trimmedAfter := strings.TrimLeft(after, " \t\n\r")
		if len(trimmedAfter) > 0 && trimmedAfter[0] == ',' && removeStart > 0 && result[removeStart] != ',' {
			after = trimmedAfter[1:]
		}
		result = result[:removeStart] + after
	}

	result = strings.ReplaceAll(result, "FUNC_SKIP", "function")

	reTrailingComma := regexp.MustCompile(`,(\s*[}\]])`)
	result = reTrailingComma.ReplaceAllString(result, "$1")

	return result
}


// MarkdownTableDataCS represents a parsed markdown table (ChatService local copy)
type MarkdownTableDataCS struct {
	Title   string                   `json:"title"`
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

// extractMarkdownTablesFromContent extracts standard markdown tables from content
// This is used for restoring historical analysis results that contain markdown tables
func extractMarkdownTablesFromContent(text string) []MarkdownTableDataCS {
	var tables []MarkdownTableDataCS

	lines := strings.Split(text, "\n")
	i := 0

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Check if this line looks like a markdown table header (starts and ends with |)
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			// Check if next line is a separator (|---|---|)
			if i+1 < len(lines) {
				sepLine := strings.TrimSpace(lines[i+1])
				if isMarkdownTableSeparatorCS(sepLine) {
					// Look for table title in preceding lines
					title := extractTableTitleCS(lines, i)

					// Found a table, parse it
					table := parseMarkdownTableFromLinesCS(lines, i)
					table.Title = title
					if len(table.Rows) > 0 {
						tables = append(tables, table)
					}
					// Skip past the table
					for i < len(lines) {
						l := strings.TrimSpace(lines[i])
						if !strings.HasPrefix(l, "|") || !strings.HasSuffix(l, "|") {
							break
						}
						i++
					}
					continue
				}
			}
		}
		i++
	}

	return tables
}

// extractTableTitleCS looks for a table title in the lines preceding the table
func extractTableTitleCS(lines []string, tableStartIdx int) string {
	// Search up to 3 lines before the table for a title
	for j := tableStartIdx - 1; j >= 0 && j >= tableStartIdx-3; j-- {
		line := strings.TrimSpace(lines[j])
		if line == "" {
			continue
		}

		// Skip if it's a table line
		if strings.HasPrefix(line, "|") {
			continue
		}

		// Check for markdown headers: ### Title, ## Title, # Title
		if strings.HasPrefix(line, "#") {
			title := strings.TrimLeft(line, "#")
			title = strings.TrimSpace(title)
			if title != "" {
				return title
			}
		}

		// Check for bold text: **Title** or **Title**：description
		if strings.HasPrefix(line, "**") {
			endIdx := strings.Index(line[2:], "**")
			if endIdx > 0 {
				title := line[2 : 2+endIdx]
				title = strings.TrimSpace(title)
				if title != "" {
					return title
				}
			}
		}

		// Check for numbered list with bold: 1. **Title**
		boldPattern := regexp.MustCompile(`^\d*[.、)]\s*\*\*(.+?)\*\*`)
		if matches := boldPattern.FindStringSubmatch(line); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}

		// Stop searching if we found a non-title line
		break
	}

	return ""
}

// isMarkdownTableSeparatorCS checks if a line is a markdown table separator
func isMarkdownTableSeparatorCS(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return false
	}
	inner := strings.Trim(line, "|")
	parts := strings.Split(inner, "|")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		cleaned := strings.Trim(part, ":-")
		if cleaned != "" {
			return false
		}
	}
	return true
}

// parseMarkdownTableFromLinesCS parses a markdown table starting at the given line index
func parseMarkdownTableFromLinesCS(lines []string, startIdx int) MarkdownTableDataCS {
	table := MarkdownTableDataCS{
		Title:   "",
		Columns: []string{},
		Rows:    []map[string]interface{}{},
	}

	if startIdx >= len(lines) {
		return table
	}

	// Parse header row
	headerLine := strings.TrimSpace(lines[startIdx])
	headers := parseMarkdownTableRowCellsCS(headerLine)
	if len(headers) == 0 {
		return table
	}

	table.Columns = headers

	// Skip separator line
	dataStartIdx := startIdx + 2

	// Parse data rows
	for i := dataStartIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			break
		}
		// Skip if it's another separator line
		if isMarkdownTableSeparatorCS(line) {
			continue
		}

		cells := parseMarkdownTableRowCellsCS(line)
		row := make(map[string]interface{})
		for j, header := range headers {
			if j < len(cells) {
				row[header] = cells[j]
			} else {
				row[header] = ""
			}
		}
		table.Rows = append(table.Rows, row)
	}

	return table
}

// parseMarkdownTableRowCellsCS splits a markdown table row into cells
func parseMarkdownTableRowCellsCS(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}
