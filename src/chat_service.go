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
	storagePath string
	mu          sync.Mutex
}

// NewChatService creates a new instance of ChatService
func NewChatService(storagePath string) *ChatService {
	return &ChatService{
		storagePath: storagePath,
	}
}

// LoadThreads loads all chat threads from storage
func (s *ChatService) LoadThreads() ([]ChatThread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.storagePath); os.IsNotExist(err) {
		return []ChatThread{}, nil
	}

	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		return nil, err
	}

	var threads []ChatThread
	if len(data) == 0 {
		return []ChatThread{}, nil
	}

	if err := json.Unmarshal(data, &threads); err != nil {
		return nil, err
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

	data, err := json.MarshalIndent(threads, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.storagePath, data, 0644)
}

// AddMessage adds a message to a specific thread
func (s *ChatService) AddMessage(threadID string, msg ChatMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	threads, err := s.loadThreadsInternal()
	if err != nil {
		return err
	}

	found := false
	for i := range threads {
		if threads[i].ID == threadID {
			threads[i].Messages = append(threads[i].Messages, msg)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("thread not found")
	}

	return s.saveThreadsInternal(threads)
}

// generateUniqueTitle ensures a title is unique within a data source
func (s *ChatService) generateUniqueTitle(threads []ChatThread, dataSourceID, title, excludeThreadID string) string {
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
	return newTitle
}

// CreateThread creates a new chat thread with a unique title
func (s *ChatService) CreateThread(dataSourceID, title string) (ChatThread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	threads, err := s.loadThreadsInternal()
	if err != nil {
		return ChatThread{}, err
	}

	uniqueTitle := s.generateUniqueTitle(threads, dataSourceID, title, "")

	newThread := ChatThread{
		ID:           strconv.FormatInt(time.Now().UnixNano(), 10), // Simple ID generation
		Title:        uniqueTitle,
		DataSourceID: dataSourceID,
		CreatedAt:    time.Now().Unix(),
		Messages:     []ChatMessage{},
	}

	threads = append([]ChatThread{newThread}, threads...) // Prepend

	if err := s.saveThreadsInternal(threads); err != nil {
		return ChatThread{}, err
	}

	return newThread, nil
}

// UpdateThreadTitle updates a thread's title, ensuring uniqueness
func (s *ChatService) UpdateThreadTitle(threadID, newTitle string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	threads, err := s.loadThreadsInternal()
	if err != nil {
		return "", err
	}

	var targetThread *ChatThread
	for i := range threads {
		if threads[i].ID == threadID {
			targetThread = &threads[i]
			break
		}
	}

	if targetThread == nil {
		return "", fmt.Errorf("thread not found")
	}

	uniqueTitle := s.generateUniqueTitle(threads, targetThread.DataSourceID, newTitle, threadID)
	targetThread.Title = uniqueTitle

	if err := s.saveThreadsInternal(threads); err != nil {
		return "", err
	}

	return uniqueTitle, nil
}

// loadThreadsInternal loads threads without locking (assumes lock held)
func (s *ChatService) loadThreadsInternal() ([]ChatThread, error) {
	if _, err := os.Stat(s.storagePath); os.IsNotExist(err) {
		return []ChatThread{}, nil
	}

	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return []ChatThread{}, nil
	}

	var threads []ChatThread
	if err := json.Unmarshal(data, &threads); err != nil {
		return nil, err
	}

	return threads, nil
}

// saveThreadsInternal saves threads without locking (assumes lock held)
func (s *ChatService) saveThreadsInternal(threads []ChatThread) error {
	data, err := json.MarshalIndent(threads, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.storagePath, data, 0644)
}

// DeleteThread deletes a thread by ID
func (s *ChatService) DeleteThread(threadID string) error {
	// Note: We need to load, filter, and save. 
	// To avoid deadlock, we shouldn't call LoadThreads/SaveThreads directly if they also lock.
	// But here they do lock. So we will implement logic carefully or use internal methods.
	// Since Load/Save lock the whole operation, we can't wrap them in another lock easily without re-entrant lock or splitting logic.
	// For simplicity in this MVP, we'll acquire lock here and duplicate the file IO or use un-exported methods.
	// BETTER APPROACH: Let's reuse Load/Save but we have to be careful about race conditions if another caller calls in between.
	// However, ChatService is likely a singleton used by App. 
	// Let's implement atomic-like operation by holding lock across the read-modify-write.

	s.mu.Lock()
	defer s.mu.Unlock()

	// Read (internal implementation to verify lock safety)
	var threads []ChatThread
	if _, err := os.Stat(s.storagePath); !os.IsNotExist(err) {
		data, err := os.ReadFile(s.storagePath)
		if err == nil && len(data) > 0 {
			_ = json.Unmarshal(data, &threads)
		}
	}

	// Filter
	newThreads := []ChatThread{}
	for _, t := range threads {
		if t.ID != threadID {
			newThreads = append(newThreads, t)
		}
	}

	// Write
	data, err := json.MarshalIndent(newThreads, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(s.storagePath, data, 0644)
}

// ClearHistory deletes all chat history
func (s *ChatService) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	empty := []ChatThread{}
	data, err := json.MarshalIndent(empty, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.storagePath, data, 0644)
}
