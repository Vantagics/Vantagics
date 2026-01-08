package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"rapidbi/config"
)

// MemoryType enum
type MemoryType string

const (
	LongTerm   MemoryType = "long_term"
	MediumTerm MemoryType = "medium_term"
	ShortTerm  MemoryType = "short_term"
)

// SessionMemory holds memory specific to a chat session
type SessionMemory struct {
	LongTerm   []string `json:"long_term"`
	MediumTerm []string `json:"medium_term"`
}

// AgentMemory holds the structured memory
type AgentMemory struct {
	Global   []string                 `json:"global"`
	Sessions map[string]SessionMemory `json:"sessions"`
}

// MemoryService manages agent memory
type MemoryService struct {
	configPath string
	memory     AgentMemory
	mu         sync.Mutex
}

// NewMemoryService creates a new memory service
func NewMemoryService(cfg config.Config) *MemoryService {
	// Use DataCacheDir for storage
	dir := cfg.DataCacheDir
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "RapidBI")
	}
	
	path := filepath.Join(dir, "agent_memory.json")
	
	service := &MemoryService{
		configPath: path,
		memory: AgentMemory{
			Global:   []string{},
			Sessions: make(map[string]SessionMemory),
		},
	}
	
	service.load()
	return service
}

func (s *MemoryService) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return // File might not exist yet
	}

	json.Unmarshal(data, &s.memory)
	if s.memory.Sessions == nil {
		s.memory.Sessions = make(map[string]SessionMemory)
	}
}

func (s *MemoryService) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s.memory, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.configPath, data, 0644)
}

// AddGlobalMemory adds a global fact
func (s *MemoryService) AddGlobalMemory(fact string) error {
	s.mu.Lock()
	s.memory.Global = append(s.memory.Global, fact)
	s.mu.Unlock()
	return s.save()
}

// AddSessionLongTermMemory adds a fact to a session's long term memory
func (s *MemoryService) AddSessionLongTermMemory(threadID string, fact string) error {
	s.mu.Lock()
	session := s.memory.Sessions[threadID]
	session.LongTerm = append(session.LongTerm, fact)
	s.memory.Sessions[threadID] = session
	s.mu.Unlock()
	return s.save()
}

// AddSessionMediumTermMemory adds a fact to a session's medium term memory
func (s *MemoryService) AddSessionMediumTermMemory(threadID string, fact string) error {
	s.mu.Lock()
	session := s.memory.Sessions[threadID]
	session.MediumTerm = append(session.MediumTerm, fact)
	s.memory.Sessions[threadID] = session
	s.mu.Unlock()
	return s.save()
}

// GetMemories returns all memory context for a specific thread
// Returns: global, sessionLong, sessionMedium
func (s *MemoryService) GetMemories(threadID string) (global []string, sessionLong []string, sessionMedium []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Return copies
	global = make([]string, len(s.memory.Global))
	copy(global, s.memory.Global)
	
	if session, ok := s.memory.Sessions[threadID]; ok {
		sessionLong = make([]string, len(session.LongTerm))
		copy(sessionLong, session.LongTerm)
		
		sessionMedium = make([]string, len(session.MediumTerm))
		copy(sessionMedium, session.MediumTerm)
	} else {
		sessionLong = []string{}
		sessionMedium = []string{}
	}
	
	return
}
