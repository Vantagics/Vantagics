package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

// LayoutService provides methods for managing layout configurations
type LayoutService struct {
	db *sql.DB
}

// NewLayoutService creates a new LayoutService instance
func NewLayoutService(db *sql.DB) *LayoutService {
	return &LayoutService{
		db: db,
	}
}

// SaveLayout saves a layout configuration to the database
// It performs an INSERT or UPDATE based on whether a layout exists for the user
func (s *LayoutService) SaveLayout(config LayoutConfiguration) error {
	if s.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Validate required fields
	if config.UserID == "" {
		return fmt.Errorf("userID is required")
	}

	if len(config.Items) == 0 {
		return fmt.Errorf("layout must contain at least one item")
	}

	// Serialize layout items to JSON
	layoutData, err := json.Marshal(map[string]interface{}{
		"items": config.Items,
	})
	if err != nil {
		return fmt.Errorf("failed to serialize layout data: %w", err)
	}

	// Generate ID if not provided
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now().UnixMilli()
	if config.CreatedAt == 0 {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	// Begin transaction for data integrity
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Check if a layout already exists for this user
	var existingID string
	err = tx.QueryRow("SELECT id FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// No existing layout, perform INSERT
		query := `
			INSERT INTO layout_configs (id, user_id, is_locked, layout_data, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`
		_, err = tx.Exec(query, config.ID, config.UserID, config.IsLocked, string(layoutData), config.CreatedAt, config.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert layout configuration: %w", err)
		}
	} else if err != nil {
		// Database error
		return fmt.Errorf("failed to check existing layout: %w", err)
	} else {
		// Existing layout found, perform UPDATE
		query := `
			UPDATE layout_configs
			SET is_locked = ?, layout_data = ?, updated_at = ?
			WHERE user_id = ?
		`
		_, err = tx.Exec(query, config.IsLocked, string(layoutData), config.UpdatedAt, config.UserID)
		if err != nil {
			return fmt.Errorf("failed to update layout configuration: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadLayout retrieves a layout configuration from the database for a specific user
// Returns the layout configuration if found, or an error if not found or on database error
func (s *LayoutService) LoadLayout(userID string) (*LayoutConfiguration, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	// Validate required fields
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	// Query the database for the user's layout
	var id string
	var isLocked bool
	var layoutData string
	var createdAt, updatedAt int64

	query := `
		SELECT id, is_locked, layout_data, created_at, updated_at
		FROM layout_configs
		WHERE user_id = ?
	`

	err := s.db.QueryRow(query, userID).Scan(&id, &isLocked, &layoutData, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no layout found for user: %s", userID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to query layout: %w", err)
	}

	// Deserialize the JSON layout data
	var layoutDataMap map[string]interface{}
	err = json.Unmarshal([]byte(layoutData), &layoutDataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize layout data: %w", err)
	}

	// Extract items array
	itemsInterface, ok := layoutDataMap["items"]
	if !ok {
		return nil, fmt.Errorf("layout data missing 'items' field")
	}

	// Convert items to JSON and then to []LayoutItem
	itemsJSON, err := json.Marshal(itemsInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal items: %w", err)
	}

	var items []LayoutItem
	err = json.Unmarshal(itemsJSON, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal items: %w", err)
	}

	// Construct and return the LayoutConfiguration
	config := &LayoutConfiguration{
		ID:        id,
		UserID:    userID,
		IsLocked:  isLocked,
		Items:     items,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	return config, nil
}

// GetDefaultLayout returns a default layout configuration
// This is used when no saved layout exists for a user
// The default layout includes all 5 component types: metrics, table, image, insights, file_download
func (s *LayoutService) GetDefaultLayout() LayoutConfiguration {
	now := time.Now().UnixMilli()

	return LayoutConfiguration{
		ID:       "default",
		UserID:   "",
		IsLocked: false,
		Items: []LayoutItem{
			// Key metrics at top-left
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
			// Data table in center
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
			// Images on right side
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
			// Insights below images
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
			// File download area at bottom
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
