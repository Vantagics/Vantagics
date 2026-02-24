package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// PackPasswordStore persists marketplace pack encryption passwords to disk
// so they survive application restarts. Passwords are keyed by file path.
type PackPasswordStore struct {
	mu       sync.RWMutex
	filePath string
	// passwords maps local .qap file path -> encryption password
	passwords map[string]string
}

// NewPackPasswordStore creates a new store with the default file path
// (~/.vantagedata/pack_passwords.json).
func NewPackPasswordStore() (*PackPasswordStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	fp := filepath.Join(home, ".vantagedata", "pack_passwords.json")
	return &PackPasswordStore{
		filePath:  fp,
		passwords: make(map[string]string),
	}, nil
}

// Load reads the password store from disk. If the file does not exist, the store
// remains empty (no error). Corrupted files are silently reset to empty.
func (s *PackPasswordStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.passwords = make(map[string]string)
			return nil
		}
		return fmt.Errorf("failed to read pack password file: %w", err)
	}

	var stored map[string]string
	if err := json.Unmarshal(data, &stored); err != nil {
		s.passwords = make(map[string]string)
		return nil
	}
	if stored == nil {
		stored = make(map[string]string)
	}
	s.passwords = stored
	return nil
}

// Save writes the password store to disk, creating the directory if needed.
func (s *PackPasswordStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	data, err := json.Marshal(s.passwords)
	if err != nil {
		return fmt.Errorf("failed to marshal pack passwords: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0600)
}

// SetPassword stores a password for the given file path.
func (s *PackPasswordStore) SetPassword(filePath, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.passwords[filePath] = password
}

// GetPassword returns the stored password for the given file path.
// Returns ("", false) if no password is stored.
func (s *PackPasswordStore) GetPassword(filePath string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pwd, ok := s.passwords[filePath]
	return pwd, ok
}

// DeletePassword removes the password for the given file path.
func (s *PackPasswordStore) DeletePassword(filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.passwords, filePath)
}

// LoadIntoMap copies all stored passwords into the provided map (for backward compatibility
// with the in-memory packPasswords map).
func (s *PackPasswordStore) LoadIntoMap(m map[string]string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.passwords {
		m[k] = v
	}
}
