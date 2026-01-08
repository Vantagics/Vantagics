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

// GlobalMemory holds the global memory
type GlobalMemory struct {
	Global []string `json:"global"`
}

// MemoryService manages agent memory
type MemoryService struct {
	dataDir    string
	globalPath string
	globalMem  GlobalMemory
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
	
	sessionsDir := filepath.Join(dir, "sessions")
	_ = os.MkdirAll(sessionsDir, 0755)

	path := filepath.Join(dir, "agent_memory.json") // Keep legacy name for global or migrate? Let's use it for global.
	
	service := &MemoryService{
		dataDir:    dir,
		globalPath: path,
		globalMem: GlobalMemory{
			Global: []string{},
		},
	}
	
	service.loadGlobal()
	return service
}

func (s *MemoryService) loadGlobal() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.globalPath)
	if err != nil {
		return // File might not exist yet
	}

	// Try unmarshal into GlobalMemory
	if err := json.Unmarshal(data, &s.globalMem); err != nil {
		// Fallback: It might be the old format (AgentMemory)
		// We can try to recover global memory from it
		var oldMem struct {
			Global []string `json:"global"`
		}
		if err2 := json.Unmarshal(data, &oldMem); err2 == nil {
			s.globalMem.Global = oldMem.Global
		}
	}
}

func (s *MemoryService) saveGlobal() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s.globalMem, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.globalPath, data, 0644)
}

func (s *MemoryService) getSessionPath(threadID string) string {
	return filepath.Join(s.dataDir, "sessions", threadID, "memory.json")
}

func (s *MemoryService) loadSession(threadID string) (SessionMemory, error) {
	path := s.getSessionPath(threadID)
	mem := SessionMemory{
		LongTerm:   []string{},
		MediumTerm: []string{},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return mem, nil
		}
		return mem, err
	}

	err = json.Unmarshal(data, &mem)
	return mem, err
}

func (s *MemoryService) saveSession(threadID string, mem SessionMemory) error {
	path := s.getSessionPath(threadID)
	
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// AddGlobalMemory adds a global fact
func (s *MemoryService) AddGlobalMemory(fact string) error {
	s.mu.Lock()
	s.globalMem.Global = append(s.globalMem.Global, fact)
	s.mu.Unlock()
	return s.saveGlobal()
}

// AddSessionLongTermMemory adds a fact to a session's long term memory
func (s *MemoryService) AddSessionLongTermMemory(threadID string, fact string) error {
	// Lock entire service or just file IO? 
	// To prevent concurrent writes to same file, we should lock.
	// Since we don't have per-session locks, we use global lock for simplicity or file locking.
	// Using global lock for safety.
	s.mu.Lock()
	defer s.mu.Unlock()

	mem, err := s.loadSession(threadID)
	if err != nil {
		return err
	}

	mem.LongTerm = append(mem.LongTerm, fact)
	return s.saveSession(threadID, mem)
}

// AddSessionMediumTermMemory adds a fact to a session's medium term memory
func (s *MemoryService) AddSessionMediumTermMemory(threadID string, fact string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mem, err := s.loadSession(threadID)
	if err != nil {
		return err
	}

	mem.MediumTerm = append(mem.MediumTerm, fact)
	return s.saveSession(threadID, mem)
}

// GetMemories returns all memory context for a specific thread
// Returns: global, sessionLong, sessionMedium
func (s *MemoryService) GetMemories(threadID string) (global []string, sessionLong []string, sessionMedium []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Return copies of global
	global = make([]string, len(s.globalMem.Global))
	copy(global, s.globalMem.Global)
	
	// Load session memory
	mem, _ := s.loadSession(threadID)
	
	sessionLong = mem.LongTerm
	sessionMedium = mem.MediumTerm
	
	return
}

