package database

import (
	"database/sql"
	"testing"
	"time"
)

// setupTestDB creates a temporary test database
func setupTestDB(t *testing.T) (*sql.DB, string) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Initialize database
	db, err := InitDB(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return db, tempDir
}

// TestSaveLayout_Insert tests inserting a new layout configuration
func TestSaveLayout_Insert(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	config := LayoutConfiguration{
		UserID:   "user123",
		IsLocked: false,
		Items: []LayoutItem{
			{
				I:           "metrics-0",
				X:           0,
				Y:           0,
				W:           6,
				H:           4,
				MinW:        3,
				MinH:        2,
				Type:        "metrics",
				InstanceIdx: 0,
				Static:      false,
			},
		},
	}

	// Save layout
	err := service.SaveLayout(config)
	if err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}

	// Verify layout was saved
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query layout: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 layout, got %d", count)
	}

	// Verify data
	var isLocked bool
	var layoutData string
	err = db.QueryRow("SELECT is_locked, layout_data FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&isLocked, &layoutData)
	if err != nil {
		t.Fatalf("Failed to query layout data: %v", err)
	}

	if isLocked != config.IsLocked {
		t.Errorf("Expected isLocked=%v, got %v", config.IsLocked, isLocked)
	}

	if layoutData == "" {
		t.Error("Expected layout_data to be non-empty")
	}
}

// TestSaveLayout_Update tests updating an existing layout configuration
func TestSaveLayout_Update(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	// Insert initial layout
	config := LayoutConfiguration{
		UserID:   "user456",
		IsLocked: false,
		Items: []LayoutItem{
			{
				I:           "metrics-0",
				X:           0,
				Y:           0,
				W:           6,
				H:           4,
				Type:        "metrics",
				InstanceIdx: 0,
			},
		},
	}

	err := service.SaveLayout(config)
	if err != nil {
		t.Fatalf("Initial SaveLayout failed: %v", err)
	}

	// Update layout
	config.IsLocked = true
	config.Items = append(config.Items, LayoutItem{
		I:           "table-0",
		X:           6,
		Y:           0,
		W:           12,
		H:           8,
		Type:        "table",
		InstanceIdx: 0,
	})

	err = service.SaveLayout(config)
	if err != nil {
		t.Fatalf("Update SaveLayout failed: %v", err)
	}

	// Verify only one layout exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query layout: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 layout after update, got %d", count)
	}

	// Verify updated data
	var isLocked bool
	err = db.QueryRow("SELECT is_locked FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&isLocked)
	if err != nil {
		t.Fatalf("Failed to query layout data: %v", err)
	}

	if isLocked != true {
		t.Errorf("Expected isLocked=true after update, got %v", isLocked)
	}
}

// TestSaveLayout_Transaction tests that SaveLayout uses transactions correctly
func TestSaveLayout_Transaction(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	config := LayoutConfiguration{
		UserID:   "user789",
		IsLocked: false,
		Items: []LayoutItem{
			{
				I:           "metrics-0",
				X:           0,
				Y:           0,
				W:           6,
				H:           4,
				Type:        "metrics",
				InstanceIdx: 0,
			},
		},
	}

	// Save layout
	err := service.SaveLayout(config)
	if err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}

	// Verify layout was committed
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query layout: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 layout (transaction committed), got %d", count)
	}
}

// TestSaveLayout_ValidationErrors tests validation error handling
func TestSaveLayout_ValidationErrors(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	tests := []struct {
		name        string
		config      LayoutConfiguration
		expectError bool
		errorMsg    string
	}{
		{
			name: "Missing UserID",
			config: LayoutConfiguration{
				UserID: "",
				Items: []LayoutItem{
					{I: "test", X: 0, Y: 0, W: 1, H: 1, Type: "metrics"},
				},
			},
			expectError: true,
			errorMsg:    "userID is required",
		},
		{
			name: "Empty Items",
			config: LayoutConfiguration{
				UserID: "user123",
				Items:  []LayoutItem{},
			},
			expectError: true,
			errorMsg:    "layout must contain at least one item",
		},
		{
			name: "Valid Config",
			config: LayoutConfiguration{
				UserID: "user123",
				Items: []LayoutItem{
					{I: "test", X: 0, Y: 0, W: 1, H: 1, Type: "metrics"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.SaveLayout(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestSaveLayout_NilDatabase tests error handling when database is nil
func TestSaveLayout_NilDatabase(t *testing.T) {
	service := NewLayoutService(nil)

	config := LayoutConfiguration{
		UserID: "user123",
		Items: []LayoutItem{
			{I: "test", X: 0, Y: 0, W: 1, H: 1, Type: "metrics"},
		},
	}

	err := service.SaveLayout(config)
	if err == nil {
		t.Error("Expected error when database is nil, got nil")
	}

	expectedMsg := "database connection is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestSaveLayout_IDGeneration tests that ID is generated if not provided
func TestSaveLayout_IDGeneration(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	config := LayoutConfiguration{
		UserID: "user_id_gen",
		Items: []LayoutItem{
			{I: "test", X: 0, Y: 0, W: 1, H: 1, Type: "metrics"},
		},
		// ID not provided
	}

	err := service.SaveLayout(config)
	if err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}

	// Verify ID was generated
	var id string
	err = db.QueryRow("SELECT id FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to query layout: %v", err)
	}

	if id == "" {
		t.Error("Expected ID to be generated, got empty string")
	}
}

// TestSaveLayout_Timestamps tests that timestamps are set correctly
func TestSaveLayout_Timestamps(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	config := LayoutConfiguration{
		UserID: "user_timestamps",
		Items: []LayoutItem{
			{I: "test", X: 0, Y: 0, W: 1, H: 1, Type: "metrics"},
		},
	}

	beforeSave := time.Now()
	err := service.SaveLayout(config)
	if err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}
	afterSave := time.Now()

	// Verify timestamps
	var createdAt, updatedAt int64
	err = db.QueryRow("SELECT created_at, updated_at FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&createdAt, &updatedAt)
	if err != nil {
		t.Fatalf("Failed to query timestamps: %v", err)
	}

	beforeSaveMs := beforeSave.UnixMilli()
	afterSaveMs := afterSave.UnixMilli()

	if createdAt < beforeSaveMs || createdAt > afterSaveMs {
		t.Errorf("created_at timestamp out of expected range: %v", createdAt)
	}

	if updatedAt < beforeSaveMs || updatedAt > afterSaveMs {
		t.Errorf("updated_at timestamp out of expected range: %v", updatedAt)
	}

	// Update and verify updated_at changes
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	config.IsLocked = true
	beforeUpdate := time.Now()
	err = service.SaveLayout(config)
	if err != nil {
		t.Fatalf("Update SaveLayout failed: %v", err)
	}
	afterUpdate := time.Now()

	var newUpdatedAt int64
	err = db.QueryRow("SELECT updated_at FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&newUpdatedAt)
	if err != nil {
		t.Fatalf("Failed to query updated timestamp: %v", err)
	}

	if newUpdatedAt <= updatedAt {
		t.Errorf("updated_at should be newer after update: old=%v, new=%v", updatedAt, newUpdatedAt)
	}

	beforeUpdateMs := beforeUpdate.UnixMilli()
	afterUpdateMs := afterUpdate.UnixMilli()

	if newUpdatedAt < beforeUpdateMs || newUpdatedAt > afterUpdateMs {
		t.Errorf("updated_at timestamp out of expected range: %v", newUpdatedAt)
	}
}

// TestSaveLayout_ComplexLayout tests saving a complex layout with multiple items
func TestSaveLayout_ComplexLayout(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	config := LayoutConfiguration{
		UserID:   "user_complex",
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
	}

	err := service.SaveLayout(config)
	if err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}

	// Verify layout was saved
	var layoutData string
	err = db.QueryRow("SELECT layout_data FROM layout_configs WHERE user_id = ?", config.UserID).Scan(&layoutData)
	if err != nil {
		t.Fatalf("Failed to query layout: %v", err)
	}

	if layoutData == "" {
		t.Error("Expected layout_data to be non-empty")
	}

	// Verify JSON contains all items
	// This is a basic check - more detailed JSON parsing could be added
	for _, item := range config.Items {
		if !contains(layoutData, item.I) {
			t.Errorf("Expected layout_data to contain item ID '%s'", item.I)
		}
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSaveLayout_UniqueUserConstraint tests that the UNIQUE constraint on user_id is respected
func TestSaveLayout_UniqueUserConstraint(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	userID := "unique_user"

	// Save first layout
	config1 := LayoutConfiguration{
		UserID: userID,
		Items: []LayoutItem{
			{I: "test1", X: 0, Y: 0, W: 1, H: 1, Type: "metrics"},
		},
	}

	err := service.SaveLayout(config1)
	if err != nil {
		t.Fatalf("First SaveLayout failed: %v", err)
	}

	// Save second layout with same user_id (should update, not insert)
	config2 := LayoutConfiguration{
		UserID:   userID,
		IsLocked: true,
		Items: []LayoutItem{
			{I: "test2", X: 1, Y: 1, W: 2, H: 2, Type: "table"},
		},
	}

	err = service.SaveLayout(config2)
	if err != nil {
		t.Fatalf("Second SaveLayout failed: %v", err)
	}

	// Verify only one layout exists for this user
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM layout_configs WHERE user_id = ?", userID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query layout count: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 layout for user, got %d", count)
	}

	// Verify the layout was updated (is_locked should be true)
	var isLocked bool
	err = db.QueryRow("SELECT is_locked FROM layout_configs WHERE user_id = ?", userID).Scan(&isLocked)
	if err != nil {
		t.Fatalf("Failed to query is_locked: %v", err)
	}

	if !isLocked {
		t.Error("Expected is_locked to be true after update")
	}
}

// TestLoadLayout_Success tests successfully loading a layout configuration
func TestLoadLayout_Success(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	// First, save a layout
	config := LayoutConfiguration{
		UserID:   "user_load_success",
		IsLocked: true,
		Items: []LayoutItem{
			{
				I:           "metrics-0",
				X:           0,
				Y:           0,
				W:           6,
				H:           4,
				MinW:        3,
				MinH:        2,
				Type:        "metrics",
				InstanceIdx: 0,
				Static:      false,
			},
			{
				I:           "table-0",
				X:           6,
				Y:           0,
				W:           12,
				H:           8,
				MinW:        8,
				MinH:        6,
				Type:        "table",
				InstanceIdx: 0,
				Static:      false,
			},
		},
	}

	err := service.SaveLayout(config)
	if err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}

	// Now load the layout
	loadedConfig, err := service.LoadLayout(config.UserID)
	if err != nil {
		t.Fatalf("LoadLayout failed: %v", err)
	}

	// Verify loaded configuration
	if loadedConfig == nil {
		t.Fatal("Expected non-nil configuration")
	}

	if loadedConfig.UserID != config.UserID {
		t.Errorf("Expected UserID=%s, got %s", config.UserID, loadedConfig.UserID)
	}

	if loadedConfig.IsLocked != config.IsLocked {
		t.Errorf("Expected IsLocked=%v, got %v", config.IsLocked, loadedConfig.IsLocked)
	}

	if len(loadedConfig.Items) != len(config.Items) {
		t.Errorf("Expected %d items, got %d", len(config.Items), len(loadedConfig.Items))
	}

	// Verify first item details
	if len(loadedConfig.Items) > 0 {
		item := loadedConfig.Items[0]
		expectedItem := config.Items[0]

		if item.I != expectedItem.I {
			t.Errorf("Expected item ID=%s, got %s", expectedItem.I, item.I)
		}
		if item.X != expectedItem.X {
			t.Errorf("Expected X=%d, got %d", expectedItem.X, item.X)
		}
		if item.Y != expectedItem.Y {
			t.Errorf("Expected Y=%d, got %d", expectedItem.Y, item.Y)
		}
		if item.W != expectedItem.W {
			t.Errorf("Expected W=%d, got %d", expectedItem.W, item.W)
		}
		if item.H != expectedItem.H {
			t.Errorf("Expected H=%d, got %d", expectedItem.H, item.H)
		}
		if item.Type != expectedItem.Type {
			t.Errorf("Expected Type=%s, got %s", expectedItem.Type, item.Type)
		}
	}
}

// TestLoadLayout_NotFound tests loading a layout that doesn't exist
func TestLoadLayout_NotFound(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	// Try to load a layout for a user that doesn't exist
	loadedConfig, err := service.LoadLayout("nonexistent_user")

	if err == nil {
		t.Error("Expected error when loading non-existent layout, got nil")
	}

	if loadedConfig != nil {
		t.Error("Expected nil configuration when layout not found")
	}

	expectedMsg := "no layout found for user: nonexistent_user"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestLoadLayout_EmptyUserID tests error handling for empty userID
func TestLoadLayout_EmptyUserID(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	loadedConfig, err := service.LoadLayout("")

	if err == nil {
		t.Error("Expected error when userID is empty, got nil")
	}

	if loadedConfig != nil {
		t.Error("Expected nil configuration when userID is empty")
	}

	expectedMsg := "userID is required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestLoadLayout_NilDatabase tests error handling when database is nil
func TestLoadLayout_NilDatabase(t *testing.T) {
	service := NewLayoutService(nil)

	loadedConfig, err := service.LoadLayout("user123")

	if err == nil {
		t.Error("Expected error when database is nil, got nil")
	}

	if loadedConfig != nil {
		t.Error("Expected nil configuration when database is nil")
	}

	expectedMsg := "database connection is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestLoadLayout_ComplexLayout tests loading a complex layout with multiple items
func TestLoadLayout_ComplexLayout(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	// Save a complex layout
	config := LayoutConfiguration{
		UserID:   "user_complex_load",
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
				MaxW:        12,
				MaxH:        8,
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
				Static:      true,
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
	}

	err := service.SaveLayout(config)
	if err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}

	// Load the layout
	loadedConfig, err := service.LoadLayout(config.UserID)
	if err != nil {
		t.Fatalf("LoadLayout failed: %v", err)
	}

	// Verify all items were loaded
	if len(loadedConfig.Items) != len(config.Items) {
		t.Fatalf("Expected %d items, got %d", len(config.Items), len(loadedConfig.Items))
	}

	// Verify each item
	for i, expectedItem := range config.Items {
		item := loadedConfig.Items[i]

		if item.I != expectedItem.I {
			t.Errorf("Item %d: Expected ID=%s, got %s", i, expectedItem.I, item.I)
		}
		if item.X != expectedItem.X {
			t.Errorf("Item %d: Expected X=%d, got %d", i, expectedItem.X, item.X)
		}
		if item.Y != expectedItem.Y {
			t.Errorf("Item %d: Expected Y=%d, got %d", i, expectedItem.Y, item.Y)
		}
		if item.W != expectedItem.W {
			t.Errorf("Item %d: Expected W=%d, got %d", i, expectedItem.W, item.W)
		}
		if item.H != expectedItem.H {
			t.Errorf("Item %d: Expected H=%d, got %d", i, expectedItem.H, item.H)
		}
		if item.MinW != expectedItem.MinW {
			t.Errorf("Item %d: Expected MinW=%d, got %d", i, expectedItem.MinW, item.MinW)
		}
		if item.MinH != expectedItem.MinH {
			t.Errorf("Item %d: Expected MinH=%d, got %d", i, expectedItem.MinH, item.MinH)
		}
		if item.MaxW != expectedItem.MaxW {
			t.Errorf("Item %d: Expected MaxW=%d, got %d", i, expectedItem.MaxW, item.MaxW)
		}
		if item.MaxH != expectedItem.MaxH {
			t.Errorf("Item %d: Expected MaxH=%d, got %d", i, expectedItem.MaxH, item.MaxH)
		}
		if item.Type != expectedItem.Type {
			t.Errorf("Item %d: Expected Type=%s, got %s", i, expectedItem.Type, item.Type)
		}
		if item.InstanceIdx != expectedItem.InstanceIdx {
			t.Errorf("Item %d: Expected InstanceIdx=%d, got %d", i, expectedItem.InstanceIdx, item.InstanceIdx)
		}
		if item.Static != expectedItem.Static {
			t.Errorf("Item %d: Expected Static=%v, got %v", i, expectedItem.Static, item.Static)
		}
	}
}

// TestLoadLayout_RoundTrip tests that save and load produce equivalent configurations
func TestLoadLayout_RoundTrip(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	// Create original configuration
	originalConfig := LayoutConfiguration{
		UserID:   "user_roundtrip",
		IsLocked: true,
		Items: []LayoutItem{
			{
				I:           "metrics-0",
				X:           5,
				Y:           10,
				W:           6,
				H:           4,
				MinW:        3,
				MinH:        2,
				MaxW:        12,
				MaxH:        8,
				Type:        "metrics",
				InstanceIdx: 0,
				Static:      false,
			},
		},
	}

	// Save the configuration
	err := service.SaveLayout(originalConfig)
	if err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}

	// Load the configuration
	loadedConfig, err := service.LoadLayout(originalConfig.UserID)
	if err != nil {
		t.Fatalf("LoadLayout failed: %v", err)
	}

	// Verify round-trip equivalence
	if loadedConfig.UserID != originalConfig.UserID {
		t.Errorf("UserID mismatch: expected %s, got %s", originalConfig.UserID, loadedConfig.UserID)
	}

	if loadedConfig.IsLocked != originalConfig.IsLocked {
		t.Errorf("IsLocked mismatch: expected %v, got %v", originalConfig.IsLocked, loadedConfig.IsLocked)
	}

	if len(loadedConfig.Items) != len(originalConfig.Items) {
		t.Fatalf("Items count mismatch: expected %d, got %d", len(originalConfig.Items), len(loadedConfig.Items))
	}

	// Verify item details match
	for i := range originalConfig.Items {
		original := originalConfig.Items[i]
		loaded := loadedConfig.Items[i]

		if loaded.I != original.I || loaded.X != original.X || loaded.Y != original.Y ||
			loaded.W != original.W || loaded.H != original.H || loaded.Type != original.Type {
			t.Errorf("Item %d mismatch: original=%+v, loaded=%+v", i, original, loaded)
		}
	}
}

// TestLoadLayout_Timestamps tests that timestamps are preserved correctly
func TestLoadLayout_Timestamps(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	config := LayoutConfiguration{
		UserID: "user_timestamps_load",
		Items: []LayoutItem{
			{I: "test", X: 0, Y: 0, W: 1, H: 1, Type: "metrics"},
		},
	}

	// Save the layout
	beforeSave := time.Now()
	err := service.SaveLayout(config)
	if err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}
	afterSave := time.Now()

	// Load the layout
	loadedConfig, err := service.LoadLayout(config.UserID)
	if err != nil {
		t.Fatalf("LoadLayout failed: %v", err)
	}

	// Verify timestamps are within expected range
	if loadedConfig.CreatedAt.Before(beforeSave) || loadedConfig.CreatedAt.After(afterSave) {
		t.Errorf("CreatedAt timestamp out of expected range: %v", loadedConfig.CreatedAt)
	}

	if loadedConfig.UpdatedAt.Before(beforeSave) || loadedConfig.UpdatedAt.After(afterSave) {
		t.Errorf("UpdatedAt timestamp out of expected range: %v", loadedConfig.UpdatedAt)
	}

	// Verify ID is not empty
	if loadedConfig.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

// TestLoadLayout_MultipleUsers tests that layouts are correctly isolated by user
func TestLoadLayout_MultipleUsers(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	// Save layouts for two different users
	config1 := LayoutConfiguration{
		UserID:   "user1",
		IsLocked: true,
		Items: []LayoutItem{
			{I: "user1-item", X: 0, Y: 0, W: 6, H: 4, Type: "metrics"},
		},
	}

	config2 := LayoutConfiguration{
		UserID:   "user2",
		IsLocked: false,
		Items: []LayoutItem{
			{I: "user2-item", X: 10, Y: 10, W: 8, H: 6, Type: "table"},
		},
	}

	err := service.SaveLayout(config1)
	if err != nil {
		t.Fatalf("SaveLayout for user1 failed: %v", err)
	}

	err = service.SaveLayout(config2)
	if err != nil {
		t.Fatalf("SaveLayout for user2 failed: %v", err)
	}

	// Load user1's layout
	loaded1, err := service.LoadLayout("user1")
	if err != nil {
		t.Fatalf("LoadLayout for user1 failed: %v", err)
	}

	// Load user2's layout
	loaded2, err := service.LoadLayout("user2")
	if err != nil {
		t.Fatalf("LoadLayout for user2 failed: %v", err)
	}

	// Verify user1's layout
	if loaded1.UserID != "user1" {
		t.Errorf("Expected user1, got %s", loaded1.UserID)
	}
	if !loaded1.IsLocked {
		t.Error("Expected user1's layout to be locked")
	}
	if len(loaded1.Items) != 1 || loaded1.Items[0].I != "user1-item" {
		t.Error("User1's items don't match")
	}

	// Verify user2's layout
	if loaded2.UserID != "user2" {
		t.Errorf("Expected user2, got %s", loaded2.UserID)
	}
	if loaded2.IsLocked {
		t.Error("Expected user2's layout to be unlocked")
	}
	if len(loaded2.Items) != 1 || loaded2.Items[0].I != "user2-item" {
		t.Error("User2's items don't match")
	}
}

// TestGetDefaultLayout tests that GetDefaultLayout returns a valid default configuration
func TestGetDefaultLayout(t *testing.T) {
	service := NewLayoutService(nil) // Database not needed for GetDefaultLayout

	defaultConfig := service.GetDefaultLayout()

	// Verify basic structure
	if defaultConfig.ID != "default" {
		t.Errorf("Expected ID='default', got '%s'", defaultConfig.ID)
	}

	if defaultConfig.UserID != "" {
		t.Errorf("Expected empty UserID, got '%s'", defaultConfig.UserID)
	}

	if defaultConfig.IsLocked {
		t.Error("Expected IsLocked=false for default layout")
	}

	// Verify timestamps are set
	if defaultConfig.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if defaultConfig.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify all 5 component types are present
	if len(defaultConfig.Items) != 5 {
		t.Fatalf("Expected 5 items in default layout, got %d", len(defaultConfig.Items))
	}

	// Track which component types we've seen
	componentTypes := make(map[string]bool)
	for _, item := range defaultConfig.Items {
		componentTypes[item.Type] = true
	}

	// Verify all required component types are present
	requiredTypes := []string{"metrics", "table", "image", "insights", "file_download"}
	for _, requiredType := range requiredTypes {
		if !componentTypes[requiredType] {
			t.Errorf("Missing required component type: %s", requiredType)
		}
	}
}

// TestGetDefaultLayout_ComponentDetails tests the specific details of each default component
func TestGetDefaultLayout_ComponentDetails(t *testing.T) {
	service := NewLayoutService(nil)
	defaultConfig := service.GetDefaultLayout()

	// Define expected components
	expectedComponents := map[string]LayoutItem{
		"metrics-0": {
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
		"table-0": {
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
		"image-0": {
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
		"insights-0": {
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
		"file_download-0": {
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
	}

	// Verify each component matches expected values
	for _, item := range defaultConfig.Items {
		expected, exists := expectedComponents[item.I]
		if !exists {
			t.Errorf("Unexpected component ID: %s", item.I)
			continue
		}

		if item.X != expected.X {
			t.Errorf("Component %s: Expected X=%d, got %d", item.I, expected.X, item.X)
		}
		if item.Y != expected.Y {
			t.Errorf("Component %s: Expected Y=%d, got %d", item.I, expected.Y, item.Y)
		}
		if item.W != expected.W {
			t.Errorf("Component %s: Expected W=%d, got %d", item.I, expected.W, item.W)
		}
		if item.H != expected.H {
			t.Errorf("Component %s: Expected H=%d, got %d", item.I, expected.H, item.H)
		}
		if item.MinW != expected.MinW {
			t.Errorf("Component %s: Expected MinW=%d, got %d", item.I, expected.MinW, item.MinW)
		}
		if item.MinH != expected.MinH {
			t.Errorf("Component %s: Expected MinH=%d, got %d", item.I, expected.MinH, item.MinH)
		}
		if item.Type != expected.Type {
			t.Errorf("Component %s: Expected Type=%s, got %s", item.I, expected.Type, item.Type)
		}
		if item.InstanceIdx != expected.InstanceIdx {
			t.Errorf("Component %s: Expected InstanceIdx=%d, got %d", item.I, expected.InstanceIdx, item.InstanceIdx)
		}
		if item.Static != expected.Static {
			t.Errorf("Component %s: Expected Static=%v, got %v", item.I, expected.Static, item.Static)
		}
	}
}

// TestGetDefaultLayout_GridConstraints tests that default layout respects grid constraints
func TestGetDefaultLayout_GridConstraints(t *testing.T) {
	service := NewLayoutService(nil)
	defaultConfig := service.GetDefaultLayout()

	// Grid is 24 columns wide
	maxColumns := 24

	for _, item := range defaultConfig.Items {
		// Check X position is within bounds
		if item.X < 0 || item.X >= maxColumns {
			t.Errorf("Component %s: X position %d is out of bounds (0-%d)", item.I, item.X, maxColumns-1)
		}

		// Check that X + W doesn't exceed grid width
		if item.X+item.W > maxColumns {
			t.Errorf("Component %s: X(%d) + W(%d) = %d exceeds grid width %d", item.I, item.X, item.W, item.X+item.W, maxColumns)
		}

		// Check Y position is non-negative
		if item.Y < 0 {
			t.Errorf("Component %s: Y position %d is negative", item.I, item.Y)
		}

		// Check dimensions are positive
		if item.W <= 0 {
			t.Errorf("Component %s: Width %d must be positive", item.I, item.W)
		}
		if item.H <= 0 {
			t.Errorf("Component %s: Height %d must be positive", item.I, item.H)
		}

		// Check min constraints are respected
		if item.MinW > 0 && item.W < item.MinW {
			t.Errorf("Component %s: Width %d is less than MinW %d", item.I, item.W, item.MinW)
		}
		if item.MinH > 0 && item.H < item.MinH {
			t.Errorf("Component %s: Height %d is less than MinH %d", item.I, item.H, item.MinH)
		}

		// Check max constraints are respected (if set)
		if item.MaxW > 0 && item.W > item.MaxW {
			t.Errorf("Component %s: Width %d exceeds MaxW %d", item.I, item.W, item.MaxW)
		}
		if item.MaxH > 0 && item.H > item.MaxH {
			t.Errorf("Component %s: Height %d exceeds MaxH %d", item.I, item.H, item.MaxH)
		}
	}
}

// TestGetDefaultLayout_NoOverlaps tests that default layout has no overlapping components
func TestGetDefaultLayout_NoOverlaps(t *testing.T) {
	service := NewLayoutService(nil)
	defaultConfig := service.GetDefaultLayout()

	// Check each pair of components for overlap
	for i := 0; i < len(defaultConfig.Items); i++ {
		for j := i + 1; j < len(defaultConfig.Items); j++ {
			item1 := defaultConfig.Items[i]
			item2 := defaultConfig.Items[j]

			// Check if rectangles overlap
			// Two rectangles overlap if:
			// - item1.X < item2.X + item2.W AND
			// - item1.X + item1.W > item2.X AND
			// - item1.Y < item2.Y + item2.H AND
			// - item1.Y + item1.H > item2.Y

			overlapsX := item1.X < item2.X+item2.W && item1.X+item1.W > item2.X
			overlapsY := item1.Y < item2.Y+item2.H && item1.Y+item1.H > item2.Y

			if overlapsX && overlapsY {
				t.Errorf("Components %s and %s overlap: %s[%d,%d,%d,%d] vs %s[%d,%d,%d,%d]",
					item1.I, item2.I,
					item1.I, item1.X, item1.Y, item1.W, item1.H,
					item2.I, item2.X, item2.Y, item2.W, item2.H)
			}
		}
	}
}

// TestGetDefaultLayout_Consistency tests that multiple calls return consistent results
func TestGetDefaultLayout_Consistency(t *testing.T) {
	service := NewLayoutService(nil)

	// Call GetDefaultLayout multiple times
	config1 := service.GetDefaultLayout()
	config2 := service.GetDefaultLayout()

	// Verify same number of items
	if len(config1.Items) != len(config2.Items) {
		t.Errorf("Inconsistent item count: %d vs %d", len(config1.Items), len(config2.Items))
	}

	// Verify items are the same (excluding timestamps which may differ slightly)
	for i := range config1.Items {
		item1 := config1.Items[i]
		item2 := config2.Items[i]

		if item1.I != item2.I || item1.X != item2.X || item1.Y != item2.Y ||
			item1.W != item2.W || item1.H != item2.H || item1.Type != item2.Type {
			t.Errorf("Item %d differs between calls: %+v vs %+v", i, item1, item2)
		}
	}
}

// TestGetDefaultLayout_CanBeSaved tests that the default layout can be saved to database
func TestGetDefaultLayout_CanBeSaved(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	service := NewLayoutService(db)

	// Get default layout
	defaultConfig := service.GetDefaultLayout()

	// Set a user ID (required for saving)
	defaultConfig.UserID = "test_user_default"

	// Try to save it
	err := service.SaveLayout(defaultConfig)
	if err != nil {
		t.Fatalf("Failed to save default layout: %v", err)
	}

	// Verify it was saved
	loadedConfig, err := service.LoadLayout(defaultConfig.UserID)
	if err != nil {
		t.Fatalf("Failed to load saved default layout: %v", err)
	}

	// Verify loaded config matches
	if len(loadedConfig.Items) != len(defaultConfig.Items) {
		t.Errorf("Item count mismatch: expected %d, got %d", len(defaultConfig.Items), len(loadedConfig.Items))
	}
}
