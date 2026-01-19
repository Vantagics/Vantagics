package database

import (
	"reflect"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

/**
 * Property-Based Tests for Dashboard Drag-Drop Layout Backend
 * 
 * These tests verify universal properties that should hold true across
 * all valid executions of the backend services.
 */

// Test data generators for property-based testing
func genLayoutItem() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		gen.IntRange(0, 23),
		gen.IntRange(0, 100),
		gen.IntRange(1, 24),
		gen.IntRange(1, 20),
		gen.OneConstOf("metrics", "table", "image", "insights", "file_download"),
		gen.IntRange(0, 10),
		gen.Bool(),
	).Map(func(vals []interface{}) LayoutItem {
		return LayoutItem{
			I:           vals[0].(string),
			X:           vals[1].(int),
			Y:           vals[2].(int),
			W:           vals[3].(int),
			H:           vals[4].(int),
			Type:        vals[5].(string),
			InstanceIdx: vals[6].(int),
			Static:      vals[7].(bool),
		}
	})
}

func genLayoutConfiguration() gopter.Gen {
	return gopter.CombineGens(
		gen.UUIDVersion4(),
		gen.UUIDVersion4(),
		gen.Bool(),
		gen.SliceOf(genLayoutItem(), reflect.TypeOf([]LayoutItem{})).SuchThat(func(v interface{}) bool {
			items := v.([]LayoutItem)
			return len(items) >= 1 && len(items) <= 20
		}),
	).Map(func(vals []interface{}) LayoutConfiguration {
		return LayoutConfiguration{
			ID:        vals[0].(string),
			UserID:    vals[1].(string),
			IsLocked:  vals[2].(bool),
			Items:     vals[3].([]LayoutItem),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	})
}

func genComponentType() gopter.Gen {
	return gen.OneConstOf("metrics", "table", "image", "insights", "file_download")
}

func genFileInfo() gopter.Gen {
	return gopter.CombineGens(
		gen.UUIDVersion4(),
		gen.AlphaString(),
		gen.IntRange(0, 10000000),
	).Map(func(vals []interface{}) FileInfo {
		return FileInfo{
			ID:        vals[0].(string),
			Name:      vals[1].(string) + ".pdf",
			Size:      int64(vals[2].(int)),
			CreatedAt: time.Now(),
			Category:  AllFiles,
		}
	})
}

func TestProperty13_LayoutConfigurationRoundTrip(t *testing.T) {
	// Feature: dashboard-drag-drop-layout, Property 13: Layout Configuration Round-Trip
	properties := gopter.NewProperties(nil)

	properties.Property("save then load produces equivalent config",
		prop.ForAll(
			func(config LayoutConfiguration) bool {
				// Arrange: Create layout service with in-memory database
				service, cleanup := setupTestLayoutService(t)
				defer cleanup()

				// Act: Save configuration
				err := service.SaveLayout(config)
				if err != nil {
					t.Logf("Failed to save layout: %v", err)
					return false
				}

				// Load configuration
				loaded, err := service.LoadLayout(config.UserID)
				if err != nil {
					t.Logf("Failed to load layout: %v", err)
					return false
				}

				// Assert: Loaded configuration should match saved configuration
				if loaded == nil {
					t.Log("Loaded configuration is nil")
					return false
				}

				// Compare essential fields (ignoring timestamps which may differ slightly)
				if loaded.ID != config.ID {
					t.Logf("ID mismatch: expected %s, got %s", config.ID, loaded.ID)
					return false
				}

				if loaded.UserID != config.UserID {
					t.Logf("UserID mismatch: expected %s, got %s", config.UserID, loaded.UserID)
					return false
				}

				if loaded.IsLocked != config.IsLocked {
					t.Logf("IsLocked mismatch: expected %t, got %t", config.IsLocked, loaded.IsLocked)
					return false
				}

				if len(loaded.Items) != len(config.Items) {
					t.Logf("Items length mismatch: expected %d, got %d", len(config.Items), len(loaded.Items))
					return false
				}

				// Compare layout items
				for i, expectedItem := range config.Items {
					if i >= len(loaded.Items) {
						t.Logf("Missing item at index %d", i)
						return false
					}

					actualItem := loaded.Items[i]
					if !layoutItemsEqual(expectedItem, actualItem) {
						t.Logf("Item mismatch at index %d: expected %+v, got %+v", i, expectedItem, actualItem)
						return false
					}
				}

				return true
			},
			genLayoutConfiguration(),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty14_ExportFiltersEmptyComponents(t *testing.T) {
	// Feature: dashboard-drag-drop-layout, Property 14: Export Filters Empty Components
	properties := gopter.NewProperties(nil)

	properties.Property("export includes only components with data",
		prop.ForAll(
			func(layoutItems []LayoutItem) bool {
				// Arrange: Create services
				layoutService, layoutCleanup := setupTestLayoutService(t)
				defer layoutCleanup()

				dataService, dataCleanup := setupTestDataService(t)
				defer dataCleanup()

				exportService := NewExportService(dataService, layoutService)

				// Create a mix of components with and without data
				config := LayoutConfiguration{
					ID:       "test-config",
					UserID:   "test-user",
					IsLocked: false,
					Items:    layoutItems,
				}

				// Act: Filter empty components
				filteredItems, err := exportService.FilterEmptyComponents(config.Items)
				if err != nil {
					t.Logf("Failed to filter components: %v", err)
					return false
				}

				// Assert: All filtered items should have data
				for _, item := range filteredItems {
					hasData, err := dataService.CheckComponentHasData(item.Type, item.I)
					if err != nil {
						t.Logf("Failed to check data for component %s: %v", item.I, err)
						return false
					}

					if !hasData {
						t.Logf("Filtered component %s should have data but doesn't", item.I)
						return false
					}
				}

				// Filtered items should be a subset of original items
				if len(filteredItems) > len(config.Items) {
					t.Logf("Filtered items count (%d) exceeds original count (%d)", len(filteredItems), len(config.Items))
					return false
				}

				return true
			},
			gen.SliceOf(genLayoutItem(), reflect.TypeOf([]LayoutItem{})).SuchThat(func(v interface{}) bool {
				items := v.([]LayoutItem)
				return len(items) >= 1 && len(items) <= 10
			}),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

func TestProperty15_ComponentTypeConsistency(t *testing.T) {
	// Feature: dashboard-drag-drop-layout, Property 15: Component Type Consistency
	properties := gopter.NewProperties(nil)

	properties.Property("all instances of a component type support same operations",
		prop.ForAll(
			func(componentType string, instanceCount int) bool {
				// Arrange: Create data service
				dataService, cleanup := setupTestDataService(t)
				defer cleanup()

				// Create multiple instances of the same component type
				instances := make([]string, instanceCount)
				for i := 0; i < instanceCount; i++ {
					instances[i] = componentType + "-" + string(rune('0'+i))
				}

				// Act & Assert: All instances should support the same operations
				for _, instanceID := range instances {
					// Check data availability (should not error)
					_, err := dataService.CheckComponentHasData(componentType, instanceID)
					if err != nil {
						t.Logf("CheckComponentHasData failed for %s: %v", instanceID, err)
						return false
					}

					// Get component data (should not error for valid types)
					_, err = dataService.GetComponentData(componentType, instanceID)
					if err != nil && componentType != "invalid_type" {
						t.Logf("GetComponentData failed for %s: %v", instanceID, err)
						return false
					}
				}

				// Batch operations should work for all instances
				hasDataMap, err := dataService.BatchCheckHasData(instances)
				if err != nil {
					t.Logf("BatchCheckHasData failed: %v", err)
					return false
				}

				// Should have results for all instances
				if len(hasDataMap) != len(instances) {
					t.Logf("BatchCheckHasData returned %d results for %d instances", len(hasDataMap), len(instances))
					return false
				}

				return true
			},
			genComponentType(),
			gen.IntRange(1, 5),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property tests for file service
func TestFileServiceProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("file service maintains category consistency",
		prop.ForAll(
			func(category FileCategory) bool {
				// Arrange: Create file service
				fileService, cleanup := setupTestFileService(t)
				defer cleanup()

				// Act: Get files for category
				files, err := fileService.GetFilesByCategory(category)
				if err != nil {
					t.Logf("GetFilesByCategory failed for %s: %v", category, err)
					return false
				}

				// Assert: All returned files should belong to the requested category
				for _, file := range files {
					if file.Category != category {
						t.Logf("File %s has category %s but was returned for category %s", file.ID, file.Category, category)
						return false
					}
				}

				return true
			},
			gen.OneConstOf(AllFiles, UserRequestRelated),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper functions for property tests
func layoutItemsEqual(a, b LayoutItem) bool {
	return a.I == b.I &&
		a.X == b.X &&
		a.Y == b.Y &&
		a.W == b.W &&
		a.H == b.H &&
		a.Type == b.Type &&
		a.InstanceIdx == b.InstanceIdx &&
		a.Static == b.Static
}

// Test setup helpers (reuse existing test setup functions)
func setupTestLayoutService(t *testing.T) (*LayoutService, func()) {
	// Reuse the existing test setup from layout_service_test.go
	service, cleanup, err := setupInMemoryLayoutService()
	if err != nil {
		t.Fatalf("Failed to setup test layout service: %v", err)
	}
	return service, cleanup
}

func setupTestDataService(t *testing.T) (*DataService, func()) {
	// Create a mock data service for testing
	service := &DataService{}
	return service, func() {}
}

func setupTestFileService(t *testing.T) (*FileService, func()) {
	// Create a mock file service for testing
	service := &FileService{
		dataDir: t.TempDir(),
	}
	return service, func() {}
}