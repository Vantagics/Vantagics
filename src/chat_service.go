package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
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
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	CreatedAt int64         `json:"created_at"`
	Messages  []ChatMessage `json:"messages"`
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
