package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ChatMessage represents a single message in a chat thread
type ChatMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"` // "user" or "assistant"
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// ChatThread represents a conversation thread
type ChatThread struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	DataSourceID string        `json:"data_source_id"` // Associated Data Source ID
	CreatedAt    int64         `json:"created_at"`
	Messages     []ChatMessage `json:"messages"`
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

