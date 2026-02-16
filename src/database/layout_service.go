package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LayoutConfiguration represents the complete layout configuration for a user
type LayoutConfiguration struct {
	ID        string       `json:"id"`
	UserID    string       `json:"userId"`
	IsLocked  bool         `json:"isLocked"`
	Items     []LayoutItem `json:"items"`
	CreatedAt int64        `json:"createdAt"`
	UpdatedAt int64        `json:"updatedAt"`
}

// LayoutItem represents a single component in the layout
type LayoutItem struct {
	I           string `json:"i"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	W           int    `json:"w"`
	H           int    `json:"h"`
	MinW        int    `json:"minW,omitempty"`
	MinH        int    `json:"minH,omitempty"`
	MaxW        int    `json:"maxW,omitempty"`
	MaxH        int    `json:"maxH,omitempty"`
	Static      bool   `json:"static"`
	Type        string `json:"type"`
	InstanceIdx int    `json:"instanceIdx"`
}

// LayoutService provides methods for managing layout configurations using JSON storage
type LayoutService struct {
	filePath string
	mu       sync.RWMutex
}

// NewLayoutService creates a new LayoutService instance
func NewLayoutService(dataDir string) *LayoutService {
	return &LayoutService{
		filePath: filepath.Join(dataDir, "layout_configs.json"),
	}
}

// loadAllConfigs reads all layout configurations from the JSON file
func (s *LayoutService) loadAllConfigs() (map[string]LayoutConfiguration, error) {
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return make(map[string]LayoutConfiguration), nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read layout file: %w", err)
	}

	var configs map[string]LayoutConfiguration
	if err := json.Unmarshal(data, &configs); err != nil {
		// If corrupted, return empty map to allow recovery
		return make(map[string]LayoutConfiguration), nil
	}

	return configs, nil
}

// saveAllConfigs writes all layout configurations to the JSON file
func (s *LayoutService) saveAllConfigs(configs map[string]LayoutConfiguration) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal layout configs: %w", err)
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// SaveLayout saves a layout configuration to the JSON file
func (s *LayoutService) SaveLayout(config LayoutConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate required fields
	if config.UserID == "" {
		return fmt.Errorf("userID is required")
	}

	if len(config.Items) == 0 {
		return fmt.Errorf("layout must contain at least one item")
	}

	// Load existing configs
	configs, err := s.loadAllConfigs()
	if err != nil {
		return err
	}

	// Generate ID if not provided
	if config.ID == "" {
		if existing, ok := configs[config.UserID]; ok {
			config.ID = existing.ID
		} else {
			config.ID = uuid.New().String()
		}
	}

	// Set timestamps
	now := time.Now().UnixMilli()
	if config.CreatedAt == 0 {
		if existing, ok := configs[config.UserID]; ok {
			config.CreatedAt = existing.CreatedAt
		} else {
			config.CreatedAt = now
		}
	}
	config.UpdatedAt = now

	// Update the map
	configs[config.UserID] = config

	// Save back to file
	return s.saveAllConfigs(configs)
}

// LoadLayout retrieves a layout configuration from the JSON file for a specific user
func (s *LayoutService) LoadLayout(userID string) (*LayoutConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Validate required fields
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	// Load all configs
	configs, err := s.loadAllConfigs()
	if err != nil {
		return nil, err
	}

	// Find the config for this user
	config, ok := configs[userID]
	if !ok {
		return nil, fmt.Errorf("no layout found for user: %s", userID)
	}

	return &config, nil
}

// GetDefaultLayout returns a default layout configuration
func (s *LayoutService) GetDefaultLayout() LayoutConfiguration {
	now := time.Now().UnixMilli()

	return LayoutConfiguration{
		ID:       "default",
		UserID:   "",
		IsLocked: false,
		Items: []LayoutItem{
			{
				I:           "metrics-0",
				X:           0,
				Y:           0,
				W:           8,
				H:           4,
				MinW:        4,
				MinH:        2,
				Type:        "metrics",
				InstanceIdx: 0,
				Static:      false,
			},
			{
				I:           "table-0",
				X:           0,
				Y:           4,
				W:           16,
				H:           8,
				MinW:        8,
				MinH:        6,
				Type:        "table",
				InstanceIdx: 0,
				Static:      false,
			},
			{
				I:           "image-0",
				X:           16,
				Y:           0,
				W:           8,
				H:           6,
				MinW:        4,
				MinH:        4,
				Type:        "image",
				InstanceIdx: 0,
				Static:      false,
			},
			{
				I:           "insights-0",
				X:           16,
				Y:           6,
				W:           8,
				H:           6,
				MinW:        4,
				MinH:        4,
				Type:        "insights",
				InstanceIdx: 0,
				Static:      false,
			},
			{
				I:           "file_download-0",
				X:           0,
				Y:           12,
				W:           24,
				H:           6,
				MinW:        8,
				MinH:        4,
				Type:        "file_download",
				InstanceIdx: 0,
				Static:      false,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
